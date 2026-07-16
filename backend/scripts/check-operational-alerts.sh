#!/usr/bin/env bash
set -euo pipefail

BACKEND_DIR="${BACKEND_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
SERVICE_NAME="${SERVICE_NAME:-caipu-backend}"
BACKUP_ROOT="${BACKUP_ROOT:-${BACKEND_DIR}/backups}"
RESTORE_DRILL_STATE_FILE="${RESTORE_DRILL_STATE_FILE:-${BACKUP_ROOT}/.last-restore-drill-ok}"
DISK_PATH="${DISK_PATH:-${BACKEND_DIR}/data}"
JOURNAL_SINCE="${JOURNAL_SINCE:--5 minutes}"
HTTP_5XX_WARNING_COUNT="${HTTP_5XX_WARNING_COUNT:-1}"
HTTP_5XX_CRITICAL_COUNT="${HTTP_5XX_CRITICAL_COUNT:-5}"
WORKER_ERROR_WARNING_COUNT="${WORKER_ERROR_WARNING_COUNT:-1}"
WORKER_ERROR_CRITICAL_COUNT="${WORKER_ERROR_CRITICAL_COUNT:-3}"
DISK_WARNING_PERCENT="${DISK_WARNING_PERCENT:-80}"
DISK_CRITICAL_PERCENT="${DISK_CRITICAL_PERCENT:-90}"
BACKUP_MAX_AGE_SECONDS="${BACKUP_MAX_AGE_SECONDS:-93600}"
RESTORE_DRILL_MAX_AGE_SECONDS="${RESTORE_DRILL_MAX_AGE_SECONDS:-691200}"
JOURNALCTL_BIN="${JOURNALCTL_BIN:-journalctl}"

status=0

warning() {
  printf 'WARNING %s\n' "$*"
  if (( status < 1 )); then status=1; fi
}

critical() {
  printf 'CRITICAL %s\n' "$*"
  status=2
}

ok() {
  printf 'OK %s\n' "$*"
}

require_non_negative_integer() {
  local name="$1" value="$2"
  if ! [[ "$value" =~ ^[0-9]+$ ]]; then
    critical "$name must be a non-negative integer"
    return 1
  fi
}

for item in \
  "HTTP_5XX_WARNING_COUNT:$HTTP_5XX_WARNING_COUNT" \
  "HTTP_5XX_CRITICAL_COUNT:$HTTP_5XX_CRITICAL_COUNT" \
  "WORKER_ERROR_WARNING_COUNT:$WORKER_ERROR_WARNING_COUNT" \
  "WORKER_ERROR_CRITICAL_COUNT:$WORKER_ERROR_CRITICAL_COUNT" \
  "DISK_WARNING_PERCENT:$DISK_WARNING_PERCENT" \
  "DISK_CRITICAL_PERCENT:$DISK_CRITICAL_PERCENT" \
  "BACKUP_MAX_AGE_SECONDS:$BACKUP_MAX_AGE_SECONDS" \
  "RESTORE_DRILL_MAX_AGE_SECONDS:$RESTORE_DRILL_MAX_AGE_SECONDS"; do
  require_non_negative_integer "${item%%:*}" "${item#*:}" || exit "$status"
done

journal=""
if journal="$($JOURNALCTL_BIN -u "$SERVICE_NAME" --since "$JOURNAL_SINCE" --no-pager -o cat 2>/dev/null)"; then
  http_5xx_count="$(printf '%s\n' "$journal" | grep -Ec 'msg="request completed".*status=5[0-9][0-9]' || true)"
  worker_error_count="$(printf '%s\n' "$journal" | grep -Eci 'level=ERROR.*(worker|job).*(failed|error)' || true)"
  if (( http_5xx_count >= HTTP_5XX_CRITICAL_COUNT )); then
    critical "http_5xx count=$http_5xx_count window=$JOURNAL_SINCE"
  elif (( http_5xx_count >= HTTP_5XX_WARNING_COUNT )); then
    warning "http_5xx count=$http_5xx_count window=$JOURNAL_SINCE"
  else
    ok "http_5xx count=$http_5xx_count window=$JOURNAL_SINCE"
  fi
  if (( worker_error_count >= WORKER_ERROR_CRITICAL_COUNT )); then
    critical "worker_errors count=$worker_error_count window=$JOURNAL_SINCE"
  elif (( worker_error_count >= WORKER_ERROR_WARNING_COUNT )); then
    warning "worker_errors count=$worker_error_count window=$JOURNAL_SINCE"
  else
    ok "worker_errors count=$worker_error_count window=$JOURNAL_SINCE"
  fi
else
  warning "journal query failed service=$SERVICE_NAME"
fi

if disk_percent="$(df -P "$DISK_PATH" 2>/dev/null | awk 'NR == 2 { value=$5; gsub(/%/, "", value); print value }')" && [[ "$disk_percent" =~ ^[0-9]+$ ]]; then
  if (( disk_percent >= DISK_CRITICAL_PERCENT )); then
    critical "disk_usage percent=$disk_percent path=$DISK_PATH"
  elif (( disk_percent >= DISK_WARNING_PERCENT )); then
    warning "disk_usage percent=$disk_percent path=$DISK_PATH"
  else
    ok "disk_usage percent=$disk_percent path=$DISK_PATH"
  fi
else
  warning "disk usage query failed path=$DISK_PATH"
fi

now_epoch="$(date -u +%s)"
shopt -s nullglob
backup_dirs=("$BACKUP_ROOT"/backup-*)
if (( ${#backup_dirs[@]} == 0 )); then
  critical "backup missing root=$BACKUP_ROOT"
else
  latest_backup="${backup_dirs[0]}"
  for backup_dir in "${backup_dirs[@]:1}"; do
    if [[ "$backup_dir" -nt "$latest_backup" ]]; then latest_backup="$backup_dir"; fi
  done
  backup_epoch="$(stat -c %Y "$latest_backup" 2>/dev/null || stat -f %m "$latest_backup" 2>/dev/null || true)"
  if [[ "$backup_epoch" =~ ^[0-9]+$ ]]; then
    backup_age=$((now_epoch - backup_epoch))
    if (( backup_age > BACKUP_MAX_AGE_SECONDS )); then
      critical "backup_age seconds=$backup_age max=$BACKUP_MAX_AGE_SECONDS path=$latest_backup"
    else
      ok "backup_age seconds=$backup_age path=$latest_backup"
    fi
  else
    warning "backup age query failed path=$latest_backup"
  fi
fi

restore_epoch=""
if [[ -f "$RESTORE_DRILL_STATE_FILE" ]]; then
  restore_epoch="$(awk -F= '$1 == "verified_at_epoch" { print $2; exit }' "$RESTORE_DRILL_STATE_FILE")"
fi
if [[ "$restore_epoch" =~ ^[0-9]+$ ]]; then
  restore_age=$((now_epoch - restore_epoch))
  if (( restore_age > RESTORE_DRILL_MAX_AGE_SECONDS )); then
    critical "restore_drill_age seconds=$restore_age max=$RESTORE_DRILL_MAX_AGE_SECONDS"
  else
    ok "restore_drill_age seconds=$restore_age"
  fi
else
  warning "restore drill success marker missing file=$RESTORE_DRILL_STATE_FILE"
fi

exit "$status"
