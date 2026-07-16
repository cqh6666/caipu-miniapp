#!/usr/bin/env bash
set -euo pipefail

BACKEND_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BACKUP_ROOT="${BACKUP_ROOT:-${BACKEND_DIR}/backups}"
RESTORE_DRILL_STATE_FILE="${RESTORE_DRILL_STATE_FILE:-${BACKUP_ROOT}/.last-restore-drill-ok}"

shopt -s nullglob
backup_dirs=("$BACKUP_ROOT"/backup-*)
if (( ${#backup_dirs[@]} == 0 )); then
  echo "backup verification failed: no backup packages found in $BACKUP_ROOT" >&2
  exit 1
fi

latest="${backup_dirs[0]}"
for backup_dir in "${backup_dirs[@]:1}"; do
  if [[ "$backup_dir" -nt "$latest" ]]; then
    latest="$backup_dir"
  fi
done

bash "$BACKEND_DIR/scripts/verify-backup.sh" "$latest"

state_dir="$(dirname "$RESTORE_DRILL_STATE_FILE")"
mkdir -p "$state_dir"
state_tmp="$(mktemp "${RESTORE_DRILL_STATE_FILE}.tmp.XXXXXX")"
trap 'rm -f -- "$state_tmp"' EXIT
printf 'verified_at_epoch=%s\nbackup_dir=%s\n' "$(date -u +%s)" "$latest" >"$state_tmp"
chmod 600 "$state_tmp"
mv -f -- "$state_tmp" "$RESTORE_DRILL_STATE_FILE"
trap - EXIT
echo "Restore drill state updated: $RESTORE_DRILL_STATE_FILE"
