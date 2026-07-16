#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/caipu-ops-health-test.XXXXXX")"
BACKUP_ROOT="$TEST_ROOT/backups"
DISK_PATH="$TEST_ROOT/data"
STATE_FILE="$BACKUP_ROOT/.last-restore-drill-ok"
FAKE_JOURNALCTL="$TEST_ROOT/journalctl"

cleanup() {
  rm -rf -- "$TEST_ROOT"
}
trap cleanup EXIT

mkdir -p "$BACKUP_ROOT/backup-current" "$DISK_PATH"
printf 'verified_at_epoch=%s\nbackup_dir=%s\n' "$(date -u +%s)" "$BACKUP_ROOT/backup-current" >"$STATE_FILE"

cat >"$FAKE_JOURNALCTL" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod 755 "$FAKE_JOURNALCTL"

JOURNALCTL_BIN="$FAKE_JOURNALCTL" \
BACKUP_ROOT="$BACKUP_ROOT" \
DISK_PATH="$DISK_PATH" \
RESTORE_DRILL_STATE_FILE="$STATE_FILE" \
  bash "$SCRIPT_ROOT/check-operational-alerts.sh" >/dev/null

cat >"$FAKE_JOURNALCTL" <<'EOF'
#!/usr/bin/env bash
for _ in 1 2 3 4 5; do
  echo 'level=ERROR msg="request completed" status=500'
done
for _ in 1 2 3; do
  echo 'level=ERROR msg="recipe worker job failed"'
done
EOF
chmod 755 "$FAKE_JOURNALCTL"

set +e
JOURNALCTL_BIN="$FAKE_JOURNALCTL" \
BACKUP_ROOT="$BACKUP_ROOT" \
DISK_PATH="$DISK_PATH" \
RESTORE_DRILL_STATE_FILE="$STATE_FILE" \
  bash "$SCRIPT_ROOT/check-operational-alerts.sh" >/dev/null
status=$?
set -e
if [[ "$status" != "2" ]]; then
  echo "critical operational signals returned status $status, want 2" >&2
  exit 1
fi

echo "operational alert policy integration test passed"
