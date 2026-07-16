#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="${1:-${BACKUP_DIR:-}}"
SQLITE3_BIN="${SQLITE3_BIN:-sqlite3}"

fail() {
  echo "backup verification failed: $*" >&2
  exit 1
}

verify_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c SHA256SUMS
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c SHA256SUMS
    return
  fi
  fail "sha256sum or shasum is required"
}

[[ -n "$BACKUP_DIR" ]] || fail "usage: verify-backup.sh <backup-directory>"
[[ -d "$BACKUP_DIR" ]] || fail "backup directory not found: $BACKUP_DIR"
command -v "$SQLITE3_BIN" >/dev/null 2>&1 || fail "sqlite3 is required: $SQLITE3_BIN"
for file in app.db uploads.tar.gz metadata.txt SHA256SUMS; do
  [[ -f "$BACKUP_DIR/$file" ]] || fail "required file is missing: $file"
done

(
  cd "$BACKUP_DIR"
  verify_sha256
) >/dev/null

quick_check="$($SQLITE3_BIN "$BACKUP_DIR/app.db" "PRAGMA quick_check;")"
if [[ "$quick_check" != "ok" ]]; then
  fail "SQLite quick_check did not return ok: $quick_check"
fi

restore_dir="$(mktemp -d "${TMPDIR:-/tmp}/caipu-restore-check.XXXXXX")"
cleanup() {
  rm -rf -- "$restore_dir"
}
trap cleanup EXIT

if tar -tzf "$BACKUP_DIR/uploads.tar.gz" | \
  awk 'BEGIN { bad=0 } /^\// || /(^|\/)\.\.($|\/)/ { bad=1 } END { exit bad ? 0 : 1 }'; then
  fail "uploads archive contains an unsafe path"
fi
mkdir -p "$restore_dir/uploads"
tar -xzf "$BACKUP_DIR/uploads.tar.gz" -C "$restore_dir/uploads"

echo "Backup verified: $BACKUP_DIR"
