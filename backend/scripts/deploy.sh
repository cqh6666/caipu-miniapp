#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

SERVER_HOST="${SERVER_HOST:-}"
DOMAIN="${DOMAIN:-}"
APP_DIR="${APP_DIR:-/opt/caipu-miniapp/backend}"
SERVICE_NAME="${SERVICE_NAME:-caipu-miniapp-backend}"
BINARY_NAME="${BINARY_NAME:-caipu-miniapp-server}"
GOOS_TARGET="${GOOS_TARGET:-linux}"
GOARCH_TARGET="${GOARCH_TARGET:-amd64}"
APP_PORT="${APP_PORT:-8080}"
HEALTHCHECK_PATH="${HEALTHCHECK_PATH:-/healthz}"
ENV_FILE="${ENV_FILE:-}"
REMOTE_TMP_DIR="${APP_DIR}/.deploy-tmp"

if [[ -z "$SERVER_HOST" ]]; then
  echo "SERVER_HOST is required, for example: root@1.2.3.4" >&2
  exit 1
fi

mkdir -p dist

echo "==> building ${BINARY_NAME} for ${GOOS_TARGET}/${GOARCH_TARGET}"
CGO_ENABLED=0 GOOS="$GOOS_TARGET" GOARCH="$GOARCH_TARGET" go build -o "dist/${BINARY_NAME}" ./cmd/server

echo "==> preparing remote staging directory"
ssh "$SERVER_HOST" "rm -rf '$REMOTE_TMP_DIR' && mkdir -p '$REMOTE_TMP_DIR' '$APP_DIR/data/uploads'"

echo "==> uploading binary and migrations"
scp "dist/${BINARY_NAME}" "$SERVER_HOST:$REMOTE_TMP_DIR/$BINARY_NAME"
scp -r migrations "$SERVER_HOST:$REMOTE_TMP_DIR/"

if [[ -n "$ENV_FILE" ]]; then
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "ENV_FILE not found: $ENV_FILE" >&2
    exit 1
  fi

  echo "==> uploading environment file"
  scp "$ENV_FILE" "$SERVER_HOST:$REMOTE_TMP_DIR/.env"
fi

echo "==> moving release into place"
ssh "$SERVER_HOST" "mkdir -p '$APP_DIR' '$APP_DIR/data/uploads' && rm -rf '$APP_DIR/migrations' && mv '$REMOTE_TMP_DIR/migrations' '$APP_DIR/migrations' && mv '$REMOTE_TMP_DIR/$BINARY_NAME' '$APP_DIR/$BINARY_NAME' && if [ -f '$REMOTE_TMP_DIR/.env' ]; then mv '$REMOTE_TMP_DIR/.env' '$APP_DIR/.env' && chmod 600 '$APP_DIR/.env'; fi && rm -rf '$REMOTE_TMP_DIR'"

echo "==> running migrations"
ssh "$SERVER_HOST" "cd '$APP_DIR' && ./'$BINARY_NAME' -migrate-only"

echo "==> restarting service"
ssh "$SERVER_HOST" "sudo systemctl enable --now '$SERVICE_NAME'"
ssh "$SERVER_HOST" "sudo systemctl status '$SERVICE_NAME' --no-pager"

echo "==> checking health"
ssh "$SERVER_HOST" "curl --fail --silent http://127.0.0.1:${APP_PORT}${HEALTHCHECK_PATH}"

cat <<EOF

Deploy completed.

Useful follow-up commands:
- ssh $SERVER_HOST "sudo journalctl -u $SERVICE_NAME -f"
- ssh $SERVER_HOST "curl -I https://${DOMAIN:-your-domain.example}${HEALTHCHECK_PATH}"
EOF
