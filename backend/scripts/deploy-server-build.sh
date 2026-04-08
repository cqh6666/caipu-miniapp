#!/usr/bin/env bash
set -euo pipefail

SERVER_HOST="${SERVER_HOST:-}"
REPO_DIR="${REPO_DIR:-/srv/caipu-miniapp}"
BACKEND_DIR="${BACKEND_DIR:-${REPO_DIR}/backend}"
BINARY_PATH="${BINARY_PATH:-${BACKEND_DIR}/bin/server}"
ADMIN_WEB_DIR="${ADMIN_WEB_DIR:-${REPO_DIR}/admin-web}"
SERVICE_NAME="${SERVICE_NAME:-caipu-backend}"
APP_PORT="${APP_PORT:-8080}"
HEALTHCHECK_PATH="${HEALTHCHECK_PATH:-/healthz}"
SYSTEMCTL_BIN="${SYSTEMCTL_BIN:-systemctl}"
BUILD_ADMIN_WEB="${BUILD_ADMIN_WEB:-1}"

if [[ -z "$SERVER_HOST" ]]; then
  echo "SERVER_HOST is required, for example: root@1.2.3.4" >&2
  exit 1
fi

ssh "$SERVER_HOST" \
  "REPO_DIR='$REPO_DIR' BACKEND_DIR='$BACKEND_DIR' BINARY_PATH='$BINARY_PATH' ADMIN_WEB_DIR='$ADMIN_WEB_DIR' SERVICE_NAME='$SERVICE_NAME' APP_PORT='$APP_PORT' HEALTHCHECK_PATH='$HEALTHCHECK_PATH' SYSTEMCTL_BIN='$SYSTEMCTL_BIN' BUILD_ADMIN_WEB='$BUILD_ADMIN_WEB' bash -s" <<'REMOTE'
set -euo pipefail

echo "==> updating repository"
cd "$REPO_DIR"
git pull --ff-only

echo "==> building backend binary"
cd "$BACKEND_DIR"
mkdir -p "$(dirname "$BINARY_PATH")"
go build -o "$BINARY_PATH" ./cmd/server

if [[ "$BUILD_ADMIN_WEB" == "1" && -d "$ADMIN_WEB_DIR" ]]; then
  echo "==> building admin-web"
  cd "$ADMIN_WEB_DIR"
  npm install
  npm run build
fi

echo "==> restarting service"
"$SYSTEMCTL_BIN" restart "$SERVICE_NAME"
"$SYSTEMCTL_BIN" status "$SERVICE_NAME" --no-pager

echo "==> checking health"
curl --fail --silent "http://127.0.0.1:${APP_PORT}${HEALTHCHECK_PATH}" >/dev/null

cat <<EOF

Deploy completed.

Useful follow-up commands:
- cd $REPO_DIR && git log -1 --oneline
- $SYSTEMCTL_BIN status $SERVICE_NAME --no-pager
- journalctl -u $SERVICE_NAME -f
- curl -I http://127.0.0.1:${APP_PORT}${HEALTHCHECK_PATH}
EOF
REMOTE
