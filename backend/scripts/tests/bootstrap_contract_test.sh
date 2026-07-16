#!/usr/bin/env bash
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")/.." && pwd)/bootstrap-server.sh"

required_patterns=(
  'User=$SERVICE_USER'
  'Group=$SERVICE_GROUP'
  'ExecStart=$APP_DIR/current/server'
  'NoNewPrivileges=true'
  'PrivateTmp=true'
  'ProtectSystem=strict'
  'ProtectHome=true'
  'ReadWritePaths=$APP_DIR/data'
  'UMask=0077'
  'TimeoutStopSec=15s'
  'SystemMaxUse=$JOURNAL_SYSTEM_MAX_USE'
  'MaxRetentionSec=$JOURNAL_MAX_RETENTION_SEC'
  'LogRateLimitIntervalSec=30s'
  'proxy_pass http://127.0.0.1:$APP_PORT/readyz;'
  '${SERVICE_NAME}-backup.timer'
  '${SERVICE_NAME}-restore-drill.timer'
  '${SERVICE_NAME}-ops-health.timer'
  'ExecStart=$APP_DIR/scripts/check-operational-alerts.sh'
)

for pattern in "${required_patterns[@]}"; do
  if ! grep -Fq "$pattern" "$SCRIPT"; then
    echo "bootstrap contract missing: $pattern" >&2
    exit 1
  fi
done

if grep -Fq 'ExecStart=$APP_DIR/bin/server' "$SCRIPT"; then
  echo "bootstrap must not point systemd at a mutable fixed binary" >&2
  exit 1
fi

echo "bootstrap systemd contract test passed"
