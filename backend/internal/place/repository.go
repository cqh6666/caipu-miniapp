package place

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByKitchenID(ctx context.Context, kitchenID int64, filter ListFilter) ([]Place, error) {
	args := []any{kitchenID}
	clauses := []string{"kitchen_id = ?", "deleted_at IS NULL"}

	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}

	if filter.Keyword != "" {
		keyword := "%" + strings.ToLower(filter.Keyword) + "%"
		clauses = append(clauses, `(LOWER(name) LIKE ? OR LOWER(address) LIKE ? OR LOWER(note) LIKE ? OR LOWER(tags_json) LIKE ?)`)
		args = append(args, keyword, keyword, keyword, keyword)
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, kitchen_id, name, type, address, latitude, longitude, price, source, source_url,
       image_urls_json, status, tags_json, note, visited_at, created_by, updated_by, created_at, updated_at
  FROM places
 WHERE `+strings.Join(clauses, " AND ")+`
 ORDER BY
       CASE status WHEN 'want' THEN 0 ELSE 1 END,
       updated_at DESC,
       created_at DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list places by kitchen: %w", err)
	}
	defer rows.Close()

	var items []Place
	for rows.Next() {
		item, err := scanPlace(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate places: %w", err)
	}

	return items, nil
}

func (r *Repository) FindByID(ctx context.Context, placeID string) (Place, error) {
	return scanPlace(r.db.QueryRowContext(ctx, `
SELECT id, kitchen_id, name, type, address, latitude, longitude, price, source, source_url,
       image_urls_json, status, tags_json, note, visited_at, created_by, updated_by, created_at, updated_at
  FROM places
 WHERE id = ? AND deleted_at IS NULL`, placeID))
}

func (r *Repository) Create(ctx context.Context, item Place) (Place, error) {
	imageURLsJSON, err := marshalStringList(item.ImageURLs)
	if err != nil {
		return Place{}, fmt.Errorf("marshal place image urls: %w", err)
	}
	tagsJSON, err := marshalStringList(item.Tags)
	if err != nil {
		return Place{}, fmt.Errorf("marshal place tags: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO places (
  id, kitchen_id, name, type, address, latitude, longitude, price, source, source_url,
  image_urls_json, status, tags_json, note, visited_at, created_by, updated_by, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.KitchenID, item.Name, item.Type, item.Address, item.Latitude, item.Longitude,
		item.Price, item.Source, item.SourceURL, imageURLsJSON, item.Status, tagsJSON, item.Note,
		item.VisitedAt, item.CreatedBy, item.UpdatedBy, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return Place{}, fmt.Errorf("insert place: %w", err)
	}

	return r.FindByID(ctx, item.ID)
}

func (r *Repository) Update(ctx context.Context, item Place) (Place, error) {
	imageURLsJSON, err := marshalStringList(item.ImageURLs)
	if err != nil {
		return Place{}, fmt.Errorf("marshal place image urls: %w", err)
	}
	tagsJSON, err := marshalStringList(item.Tags)
	if err != nil {
		return Place{}, fmt.Errorf("marshal place tags: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE places
   SET name = ?,
       type = ?,
       address = ?,
       latitude = ?,
       longitude = ?,
       price = ?,
       source = ?,
       source_url = ?,
       image_urls_json = ?,
       status = ?,
       tags_json = ?,
       note = ?,
       visited_at = ?,
       updated_by = ?,
       updated_at = ?
 WHERE id = ? AND deleted_at IS NULL`,
		item.Name, item.Type, item.Address, item.Latitude, item.Longitude, item.Price, item.Source,
		item.SourceURL, imageURLsJSON, item.Status, tagsJSON, item.Note, item.VisitedAt,
		item.UpdatedBy, item.UpdatedAt, item.ID,
	)
	if err != nil {
		return Place{}, fmt.Errorf("update place: %w", err)
	}

	if count, _ := result.RowsAffected(); count == 0 {
		return Place{}, sql.ErrNoRows
	}

	return r.FindByID(ctx, item.ID)
}

func (r *Repository) Delete(ctx context.Context, placeID string, userID int64, deletedAt string) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE places
   SET deleted_at = ?,
       updated_by = ?,
       updated_at = ?
 WHERE id = ? AND deleted_at IS NULL`, deletedAt, userID, deletedAt, placeID)
	if err != nil {
		return fmt.Errorf("delete place: %w", err)
	}

	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}

	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPlace(s scanner) (Place, error) {
	var item Place
	var imageURLsJSON string
	var tagsJSON string

	if err := s.Scan(
		&item.ID,
		&item.KitchenID,
		&item.Name,
		&item.Type,
		&item.Address,
		&item.Latitude,
		&item.Longitude,
		&item.Price,
		&item.Source,
		&item.SourceURL,
		&imageURLsJSON,
		&item.Status,
		&tagsJSON,
		&item.Note,
		&item.VisitedAt,
		&item.CreatedBy,
		&item.UpdatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Place{}, err
	}

	item.ImageURLs = unmarshalStringList(imageURLsJSON)
	item.Tags = unmarshalStringList(tagsJSON)
	return item, nil
}

func marshalStringList(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalStringList(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return []string{}
	}
	if values == nil {
		return []string{}
	}
	return values
}
