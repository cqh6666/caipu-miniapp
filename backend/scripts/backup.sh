#!/usr/bin/env bash
set -euo pipefail

BACKEND_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SQLITE_PATH="${SQLITE_PATH:-${BACKEND_DIR}/data/app.db}"
UPLOAD_DIR="${UPLOAD_DIR:-${BACKEND_DIR}/data/uploads}"
BACKUP_ROOT="${BACKUP_ROOT:-${BACKEND_DIR}/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
OFFSITE_BACKUP_TARGET="${OFFSITE_BACKUP_TARGET:-}"
REQUIRE_OFFSITE_BACKUP="${REQUIRE_OFFSITE_BACKUP:-0}"
RELEASE_ID="${RELEASE_ID:-}"
SQLITE3_BIN="${SQLITE3_BIN:-sqlite3}"
RSYNC_BIN="${RSYNC_BIN:-rsync}"

fail() {
  echo "backup failed: $*" >&2
  exit 1
}

sha256_files() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$@"
    return
  fi
  fail "sha256sum or shasum is required"
}

if ! [[ "$RETENTION_DAYS" =~ ^[0-9]+$ ]]; then
  fail "RETENTION_DAYS must be a non-negative integer"
fi
if [[ "$REQUIRE_OFFSITE_BACKUP" != "0" && "$REQUIRE_OFFSITE_BACKUP" != "1" ]]; then
  fail "REQUIRE_OFFSITE_BACKUP must be 0 or 1"
fi
if [[ "$REQUIRE_OFFSITE_BACKUP" == "1" && -z "$OFFSITE_BACKUP_TARGET" ]]; then
  fail "OFFSITE_BACKUP_TARGET is required when REQUIRE_OFFSITE_BACKUP=1"
fi
command -v "$SQLITE3_BIN" >/dev/null 2>&1 || fail "sqlite3 is required: $SQLITE3_BIN"
[[ -f "$SQLITE_PATH" ]] || fail "SQLite database not found: $SQLITE_PATH"
[[ -d "$UPLOAD_DIR" ]] || fail "upload directory not found: $UPLOAD_DIR"
if [[ -z "$RELEASE_ID" && -f "${BACKEND_DIR}/current/manifest.env" ]]; then
  RELEASE_ID="$(awk -F= '$1 == "release_id" { print substr($0, index($0, "=") + 1); exit }' \
    "${BACKEND_DIR}/current/manifest.env")"
fi
RELEASE_ID="${RELEASE_ID:-unknown}"

mkdir -p "$BACKUP_ROOT"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
safe_release_id="$(printf '%s' "$RELEASE_ID" | tr -c 'A-Za-z0-9._-' '-')"
backup_name="backup-${timestamp}-${safe_release_id}-$$"
staging_dir="$(mktemp -d "${BACKUP_ROOT}/.${backup_name}.XXXXXX")"
final_dir="${BACKUP_ROOT}/${backup_name}"

cleanup() {
  if [[ -n "${staging_dir:-}" && -d "$staging_dir" ]]; then
    rm -rf -- "$staging_dir"
  fi
}
trap cleanup EXIT

# sqlite3 .backup uses SQLite's online backup protocol and includes committed
# WAL pages; copying only app.db here would produce an inconsistent snapshot.
escaped_snapshot="${staging_dir}/app.db"
if [[ "$escaped_snapshot" == *'"'* || "$escaped_snapshot" == *$'\n'* ]]; then
  fail "backup path contains unsupported characters"
fi
"$SQLITE3_BIN" "$SQLITE_PATH" ".timeout 10000" ".backup \"${escaped_snapshot}\""

quick_check="$($SQLITE3_BIN "$staging_dir/app.db" "PRAGMA quick_check;")"
if [[ "$quick_check" != "ok" ]]; then
  fail "SQLite quick_check did not return ok: $quick_check"
fi

tar -czf "$staging_dir/uploads.tar.gz" -C "$UPLOAD_DIR" .

cat >"$staging_dir/metadata.txt" <<EOF
format_version=1
created_at=${timestamp}
release_id=${RELEASE_ID}
sqlite_source=${SQLITE_PATH}
uploads_source=${UPLOAD_DIR}
EOF

(
  cd "$staging_dir"
  sha256_files app.db uploads.tar.gz metadata.txt >SHA256SUMS
)
chmod 600 "$staging_dir/app.db" "$staging_dir/uploads.tar.gz" \
  "$staging_dir/metadata.txt" "$staging_dir/SHA256SUMS"
chmod 700 "$staging_dir"
mv "$staging_dir" "$final_dir"
staging_dir=""

if [[ -n "$OFFSITE_BACKUP_TARGET" ]]; then
  command -v "$RSYNC_BIN" >/dev/null 2>&1 || fail "rsync is required for offsite backup: $RSYNC_BIN"
  "$RSYNC_BIN" -a --checksum "${final_dir}/" "${OFFSITE_BACKUP_TARGET%/}/${backup_name}/"
fi

find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d \
  -name 'backup-*' -mtime "+${RETENTION_DAYS}" -exec rm -rf -- {} +

echo "Backup created: $final_dir"
