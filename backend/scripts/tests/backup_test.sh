#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/caipu-backup-test.XXXXXX")"
reader_pid=""

cleanup() {
  if [[ -n "$reader_pid" ]]; then
    exec 3>&- || true
    wait "$reader_pid" 2>/dev/null || true
  fi
  rm -rf -- "$TEST_ROOT"
}
trap cleanup EXIT

mkdir -p "$TEST_ROOT/data/uploads" "$TEST_ROOT/backups"
printf 'upload fixture\n' >"$TEST_ROOT/data/uploads/fixture.txt"
sqlite3 "$TEST_ROOT/data/app.db" \
  "PRAGMA journal_mode=WAL; CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL); INSERT INTO items(name) VALUES ('before-wal');" \
  >/dev/null

# Keep a read transaction open so the second committed row remains in WAL.
mkfifo "$TEST_ROOT/reader.pipe"
sqlite3 "$TEST_ROOT/data/app.db" <"$TEST_ROOT/reader.pipe" >"$TEST_ROOT/reader.out" &
reader_pid=$!
exec 3>"$TEST_ROOT/reader.pipe"
printf '.print reader-ready\nBEGIN; SELECT COUNT(*) FROM items;\n' >&3
for _ in {1..100}; do
  [[ -s "$TEST_ROOT/reader.out" ]] && break
  sleep 0.01
done
[[ -s "$TEST_ROOT/reader.out" ]] || { echo "WAL reader did not become ready" >&2; exit 1; }
sqlite3 "$TEST_ROOT/data/app.db" ".timeout 5000" \
  "INSERT INTO items(name) VALUES ('committed-in-wal');"

SQLITE_PATH="$TEST_ROOT/data/app.db" \
UPLOAD_DIR="$TEST_ROOT/data/uploads" \
BACKUP_ROOT="$TEST_ROOT/backups" \
RELEASE_ID="backup-test" \
RETENTION_DAYS=7 \
  bash "$SCRIPT_DIR/backup.sh" >/dev/null

backup_dir="$(find "$TEST_ROOT/backups" -mindepth 1 -maxdepth 1 -type d -name 'backup-*' -print -quit)"
[[ -n "$backup_dir" ]] || { echo "backup directory was not created" >&2; exit 1; }
bash "$SCRIPT_DIR/verify-backup.sh" "$backup_dir" >/dev/null
BACKUP_ROOT="$TEST_ROOT/backups" bash "$SCRIPT_DIR/verify-latest-backup.sh" >/dev/null

row_count="$(sqlite3 "$backup_dir/app.db" "SELECT COUNT(*) FROM items;")"
[[ "$row_count" == "2" ]] || {
  echo "online backup missed committed WAL data: row_count=$row_count" >&2
  exit 1
}

printf 'tamper' >>"$backup_dir/app.db"
if bash "$SCRIPT_DIR/verify-backup.sh" "$backup_dir" >/dev/null 2>&1; then
  echo "tampered backup unexpectedly passed verification" >&2
  exit 1
fi

echo "backup integration test passed"
