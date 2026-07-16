package bootstrap

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type migrationFile struct {
	name     string
	content  []byte
	checksum string
	applied  bool
}

var migrationFilenamePattern = regexp.MustCompile(`^(\d+)_.*\.sql$`)

var legacyDuplicateMigration019 = map[string]struct{}{
	"019_add_diet_assistant_messages.sql": {},
	"019_add_places.sql":                  {},
}

func RunMigrations(ctx context.Context, db *sql.DB, logger *slog.Logger, dir string) error {
	if err := ensureMigrationTable(ctx, db); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	files, err := loadMigrationFiles(dir)
	if err != nil {
		return err
	}

	for index := range files {
		applied, storedChecksum, err := migrationRecord(ctx, db, files[index].name)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", files[index].name, err)
		}
		files[index].applied = applied
		if !applied {
			continue
		}
		if storedChecksum == "" {
			if _, err := db.ExecContext(
				ctx,
				`UPDATE schema_migrations SET checksum = ? WHERE filename = ? AND checksum = ''`,
				files[index].checksum,
				files[index].name,
			); err != nil {
				return fmt.Errorf("backfill migration checksum %s: %w", files[index].name, err)
			}
			logger.Info("migration checksum backfilled", "file", files[index].name)
			continue
		}
		if storedChecksum != files[index].checksum {
			return fmt.Errorf(
				"migration checksum mismatch for %s: database=%s file=%s",
				files[index].name,
				storedChecksum,
				files[index].checksum,
			)
		}
	}

	appliedCount := 0
	for _, migration := range files {
		if migration.applied {
			continue
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", migration.name, err)
		}

		if _, err := tx.ExecContext(ctx, string(migration.content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", migration.name, err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (filename, checksum, applied_at) VALUES (?, ?, ?)`,
			migration.name,
			migration.checksum,
			time.Now().Format(time.RFC3339),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", migration.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", migration.name, err)
		}

		appliedCount++
		logger.Info("migration applied", "file", migration.name, "checksum", migration.checksum)
	}
	if appliedCount > 0 {
		if err := checkSQLiteIntegrity(ctx, db); err != nil {
			return fmt.Errorf("check database integrity after migrations: %w", err)
		}
	}

	return nil
}

// CheckMigrationsCurrent verifies that every migration shipped with the current
// release has been applied. Extra rows are allowed so a previous, forward-
// compatible binary can still be restored after a newer release migrated the
// database.
func CheckMigrationsCurrent(ctx context.Context, db *sql.DB, dir string) error {
	files, err := loadMigrationFiles(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("migration dir contains no SQL files: %s", dir)
	}

	pending := make([]string, 0)
	for _, migration := range files {
		applied, storedChecksum, err := migrationRecord(ctx, db, migration.name)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", migration.name, err)
		}
		if !applied {
			pending = append(pending, migration.name)
			continue
		}
		if storedChecksum == "" {
			return fmt.Errorf("migration checksum is missing: %s", migration.name)
		}
		if storedChecksum != migration.checksum {
			return fmt.Errorf(
				"migration checksum mismatch for %s: database=%s file=%s",
				migration.name,
				storedChecksum,
				migration.checksum,
			)
		}
	}
	if len(pending) > 0 {
		return fmt.Errorf("pending migrations: %s", strings.Join(pending, ", "))
	}
	return nil
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migration dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	if err := validateMigrationSequences(files); err != nil {
		return nil, err
	}
	return files, nil
}

func loadMigrationFiles(dir string) ([]migrationFile, error) {
	names, err := migrationFiles(dir)
	if err != nil {
		return nil, err
	}
	files := make([]migrationFile, 0, len(names))
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		digest := sha256.Sum256(content)
		files = append(files, migrationFile{
			name:     name,
			content:  content,
			checksum: hex.EncodeToString(digest[:]),
		})
	}
	return files, nil
}

func validateMigrationSequences(files []string) error {
	bySequence := make(map[int][]string, len(files))
	for _, name := range files {
		matches := migrationFilenamePattern.FindStringSubmatch(name)
		if len(matches) != 2 {
			return fmt.Errorf("invalid migration filename %q: expected NNN_description.sql", name)
		}
		sequence, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("parse migration sequence %q: %w", name, err)
		}
		bySequence[sequence] = append(bySequence[sequence], name)
	}
	for sequence, names := range bySequence {
		if len(names) <= 1 {
			continue
		}
		if sequence == 19 && isLegacyDuplicate019(names) {
			continue
		}
		sort.Strings(names)
		return fmt.Errorf("duplicate migration sequence %03d: %s", sequence, strings.Join(names, ", "))
	}
	return nil
}

func isLegacyDuplicate019(names []string) bool {
	if len(names) != len(legacyDuplicateMigration019) {
		return false
	}
	for _, name := range names {
		if _, ok := legacyDuplicateMigration019[name]; !ok {
			return false
		}
	}
	return true
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename TEXT PRIMARY KEY,
	checksum TEXT NOT NULL DEFAULT '',
	applied_at TEXT NOT NULL
);
`)
	if err != nil {
		return err
	}

	hasChecksum := false
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(schema_migrations)`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "checksum" {
			hasChecksum = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if hasChecksum {
		return nil
	}
	_, err = db.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN checksum TEXT NOT NULL DEFAULT ''`)
	return err
}

func migrationRecord(ctx context.Context, db *sql.DB, filename string) (bool, string, error) {
	var checksum string
	err := db.QueryRowContext(
		ctx,
		`SELECT COALESCE(checksum, '') FROM schema_migrations WHERE filename = ? LIMIT 1`,
		filename,
	).Scan(&checksum)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, checksum, nil
}

func checkSQLiteIntegrity(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	problems := make([]string, 0)
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(result), "ok") {
			problems = append(problems, result)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(problems) > 0 {
		return fmt.Errorf("PRAGMA integrity_check failed: %s", strings.Join(problems, "; "))
	}
	return nil
}
