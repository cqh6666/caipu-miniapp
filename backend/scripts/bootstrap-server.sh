#!/usr/bin/env bash
set -euo pipefail

SERVER_HOST="${SERVER_HOST:-}"
DOMAIN="${DOMAIN:-}"
APP_DIR="${APP_DIR:-/srv/caipu-miniapp/backend}"
ADMIN_WEB_DIR="${ADMIN_WEB_DIR:-/srv/caipu-miniapp/admin-web}"
SERVICE_NAME="${SERVICE_NAME:-caipu-backend}"
SERVICE_USER="${SERVICE_USER:-caipu-backend}"
SERVICE_GROUP="${SERVICE_GROUP:-caipu-backend}"
APP_ENV_FILE="${APP_ENV_FILE:-${APP_DIR}/configs/prod.env}"
APP_PORT="${APP_PORT:-8080}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"
ENABLE_UFW="${ENABLE_UFW:-0}"
NGINX_SITE_MODE="${NGINX_SITE_MODE:-shared_prefix}"
API_PREFIX="${API_PREFIX:-/caipu-api}"
UPLOADS_PREFIX="${UPLOADS_PREFIX:-/caipu-uploads}"
HEALTHZ_PATH="${HEALTHZ_PATH:-/caipu-healthz}"
ROOT_PROXY_PASS="${ROOT_PROXY_PASS:-}"
UPLOAD_MAX_IMAGE_MB="${UPLOAD_MAX_IMAGE_MB:-10}"
NGINX_CLIENT_MAX_BODY_SIZE_MB="${NGINX_CLIENT_MAX_BODY_SIZE_MB:-}"
JOURNAL_SYSTEM_MAX_USE="${JOURNAL_SYSTEM_MAX_USE:-512M}"
JOURNAL_MAX_RETENTION_SEC="${JOURNAL_MAX_RETENTION_SEC:-14day}"

normalize_prefix() {
  local value="$1"
  if [[ -z "$value" ]]; then
    echo "prefix value is required" >&2
    exit 1
  fi
  if [[ "$value" != /* ]]; then
    value="/$value"
  fi
  value="${value%/}"
  if [[ "$value" == "/" ]]; then
    echo "prefix cannot be '/'" >&2
    exit 1
  fi
  printf '%s' "$value"
}

normalize_path() {
  local value="$1"
  if [[ -z "$value" ]]; then
    echo "path value is required" >&2
    exit 1
  fi
  if [[ "$value" != /* ]]; then
    value="/$value"
  fi
  value="${value%/}"
  if [[ -z "$value" ]]; then
    value="/"
  fi
  printf '%s' "$value"
}

if [[ -z "$SERVER_HOST" ]]; then
  echo "SERVER_HOST is required, for example: root@1.2.3.4" >&2
  exit 1
fi

if [[ -z "$DOMAIN" ]]; then
  echo "DOMAIN is required, for example: www.gxm1227.top" >&2
  exit 1
fi

if ! [[ "$UPLOAD_MAX_IMAGE_MB" =~ ^[1-9][0-9]*$ ]]; then
  echo "UPLOAD_MAX_IMAGE_MB must be a positive integer" >&2
  exit 1
fi

if [[ -z "$NGINX_CLIENT_MAX_BODY_SIZE_MB" ]]; then
  NGINX_CLIENT_MAX_BODY_SIZE_MB=$((UPLOAD_MAX_IMAGE_MB + 1))
fi

if ! [[ "$NGINX_CLIENT_MAX_BODY_SIZE_MB" =~ ^[1-9][0-9]*$ ]]; then
  echo "NGINX_CLIENT_MAX_BODY_SIZE_MB must be a positive integer" >&2
  exit 1
fi

minimum_body_size_mb=$((UPLOAD_MAX_IMAGE_MB + 1))
if (( NGINX_CLIENT_MAX_BODY_SIZE_MB < minimum_body_size_mb )); then
  echo "NGINX_CLIENT_MAX_BODY_SIZE_MB must be at least UPLOAD_MAX_IMAGE_MB + 1" >&2
  exit 1
fi

if ! [[ "$JOURNAL_SYSTEM_MAX_USE" =~ ^[1-9][0-9]*[KMGTP]?$ ]]; then
  echo "JOURNAL_SYSTEM_MAX_USE must be a positive systemd size such as 512M" >&2
  exit 1
fi
if ! [[ "$JOURNAL_MAX_RETENTION_SEC" =~ ^[1-9][0-9]*(s|min|h|day|week|month|year)$ ]]; then
  echo "JOURNAL_MAX_RETENTION_SEC must be a positive systemd duration such as 14day" >&2
  exit 1
fi

case "$NGINX_SITE_MODE" in
  shared_prefix)
    API_PREFIX="$(normalize_prefix "$API_PREFIX")"
    UPLOADS_PREFIX="$(normalize_prefix "$UPLOADS_PREFIX")"
    HEALTHZ_PATH="$(normalize_path "$HEALTHZ_PATH")"
    ;;
  standalone)
    ;;
  *)
    echo "NGINX_SITE_MODE must be 'shared_prefix' or 'standalone'" >&2
    exit 1
    ;;
esac

ssh "$SERVER_HOST" "APP_DIR='$APP_DIR' ADMIN_WEB_DIR='$ADMIN_WEB_DIR' SERVICE_NAME='$SERVICE_NAME' SERVICE_USER='$SERVICE_USER' SERVICE_GROUP='$SERVICE_GROUP' APP_ENV_FILE='$APP_ENV_FILE' DOMAIN='$DOMAIN' APP_PORT='$APP_PORT' ENABLE_UFW='$ENABLE_UFW' NGINX_SITE_MODE='$NGINX_SITE_MODE' API_PREFIX='$API_PREFIX' UPLOADS_PREFIX='$UPLOADS_PREFIX' HEALTHZ_PATH='$HEALTHZ_PATH' ROOT_PROXY_PASS='$ROOT_PROXY_PASS' NGINX_CLIENT_MAX_BODY_SIZE_MB='$NGINX_CLIENT_MAX_BODY_SIZE_MB' JOURNAL_SYSTEM_MAX_USE='$JOURNAL_SYSTEM_MAX_USE' JOURNAL_MAX_RETENTION_SEC='$JOURNAL_MAX_RETENTION_SEC' bash -s" <<'REMOTE'
set -euo pipefail

sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx curl sqlite3 rsync

sudo mkdir -p /etc/systemd/journald.conf.d
sudo tee "/etc/systemd/journald.conf.d/${SERVICE_NAME}.conf" >/dev/null <<EOF
[Journal]
SystemMaxUse=$JOURNAL_SYSTEM_MAX_USE
MaxRetentionSec=$JOURNAL_MAX_RETENTION_SEC
Compress=yes
EOF
sudo systemctl restart systemd-journald

if [[ "$ENABLE_UFW" == "1" ]]; then
  sudo apt install -y ufw
  sudo ufw allow OpenSSH
  sudo ufw allow 80/tcp
  sudo ufw allow 443/tcp
  sudo ufw --force enable
fi

if ! getent group "$SERVICE_GROUP" >/dev/null; then
  sudo groupadd --system "$SERVICE_GROUP"
fi
if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
  sudo useradd --system --gid "$SERVICE_GROUP" --home-dir /nonexistent \
    --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

sudo mkdir -p "$APP_DIR/releases" "$APP_DIR/data/uploads" "$APP_DIR/backups" "$ADMIN_WEB_DIR"
sudo chmod 755 "$APP_DIR" "$APP_DIR/releases"
sudo chown -R "$SERVICE_USER:$SERVICE_GROUP" "$APP_DIR/data"
sudo chmod 750 "$APP_DIR/data" "$APP_DIR/data/uploads"
sudo chown -R "$SERVICE_USER:$SERVICE_GROUP" "$APP_DIR/backups"
sudo chmod 700 "$APP_DIR/backups"
if [[ -f "$APP_ENV_FILE" ]]; then
  sudo chown "$SERVICE_USER:$SERVICE_GROUP" "$APP_ENV_FILE"
  sudo chmod 600 "$APP_ENV_FILE"
fi

sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" >/dev/null <<EOF
[Unit]
Description=Caipu Miniapp Backend
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$APP_DIR/current
Environment=APP_ENV_FILE=$APP_ENV_FILE
Environment=SQLITE_PATH=$APP_DIR/data/app.db
Environment=MIGRATION_DIR=$APP_DIR/current/migrations
Environment=UPLOAD_DIR=$APP_DIR/data/uploads
ExecStart=$APP_DIR/current/server
Restart=on-failure
RestartSec=3
LogRateLimitIntervalSec=30s
LogRateLimitBurst=1000
KillSignal=SIGTERM
TimeoutStopSec=15s
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$APP_DIR/data

[Install]
WantedBy=multi-user.target
EOF

sudo tee "/etc/systemd/system/${SERVICE_NAME}-backup.service" >/dev/null <<EOF
[Unit]
Description=Caipu Miniapp consistent SQLite and uploads backup
After=${SERVICE_NAME}.service

[Service]
Type=oneshot
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$APP_DIR
Environment=SQLITE_PATH=$APP_DIR/data/app.db
Environment=UPLOAD_DIR=$APP_DIR/data/uploads
Environment=BACKUP_ROOT=$APP_DIR/backups
EnvironmentFile=-/etc/${SERVICE_NAME}-backup.env
ExecStart=$APP_DIR/scripts/backup.sh
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$APP_DIR/data $APP_DIR/backups
EOF

sudo tee "/etc/systemd/system/${SERVICE_NAME}-backup.timer" >/dev/null <<EOF
[Unit]
Description=Daily Caipu Miniapp backup

[Timer]
OnCalendar=*-*-* 02:15:00
RandomizedDelaySec=15m
Persistent=true
Unit=${SERVICE_NAME}-backup.service

[Install]
WantedBy=timers.target
EOF

sudo tee "/etc/systemd/system/${SERVICE_NAME}-restore-drill.service" >/dev/null <<EOF
[Unit]
Description=Caipu Miniapp backup restore verification drill

[Service]
Type=oneshot
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$APP_DIR
Environment=BACKUP_ROOT=$APP_DIR/backups
Environment=RESTORE_DRILL_STATE_FILE=$APP_DIR/backups/.last-restore-drill-ok
ExecStart=$APP_DIR/scripts/verify-latest-backup.sh
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$APP_DIR/backups
EOF

sudo tee "/etc/systemd/system/${SERVICE_NAME}-restore-drill.timer" >/dev/null <<EOF
[Unit]
Description=Weekly Caipu Miniapp backup restore verification drill

[Timer]
OnCalendar=Sun *-*-* 04:15:00
RandomizedDelaySec=30m
Persistent=true
Unit=${SERVICE_NAME}-restore-drill.service

[Install]
WantedBy=timers.target
EOF

sudo tee "/etc/systemd/system/${SERVICE_NAME}-ops-health.service" >/dev/null <<EOF
[Unit]
Description=Caipu Miniapp operational alert policy check
After=${SERVICE_NAME}.service

[Service]
Type=oneshot
User=root
Group=root
WorkingDirectory=$APP_DIR
Environment=SERVICE_NAME=$SERVICE_NAME
Environment=BACKUP_ROOT=$APP_DIR/backups
Environment=DISK_PATH=$APP_DIR/data
Environment=RESTORE_DRILL_STATE_FILE=$APP_DIR/backups/.last-restore-drill-ok
ExecStart=$APP_DIR/scripts/check-operational-alerts.sh
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
EOF

sudo tee "/etc/systemd/system/${SERVICE_NAME}-ops-health.timer" >/dev/null <<EOF
[Unit]
Description=Run Caipu Miniapp operational alert checks every five minutes

[Timer]
OnBootSec=5m
OnUnitActiveSec=5m
Persistent=true
Unit=${SERVICE_NAME}-ops-health.service

[Install]
WantedBy=timers.target
EOF

if [[ "$NGINX_SITE_MODE" == "standalone" ]]; then
  sudo tee "/etc/nginx/sites-available/${SERVICE_NAME}" >/dev/null <<EOF
server {
    listen 80;
    server_name $DOMAIN;

    client_max_body_size ${NGINX_CLIENT_MAX_BODY_SIZE_MB}m;

    location = /admin {
        return 301 /admin/;
    }

    location ^~ /admin/ {
        alias $ADMIN_WEB_DIR/dist/;
        try_files \$uri \$uri/ /admin/index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:$APP_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /uploads/ {
        proxy_pass http://127.0.0.1:$APP_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location / {
        proxy_pass http://127.0.0.1:$APP_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
else
  if [[ -n "$ROOT_PROXY_PASS" ]]; then
    root_block=$(cat <<EOF
    location / {
        proxy_pass $ROOT_PROXY_PASS;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
EOF
)
  else
    root_block=$(cat <<'EOF'
    location / {
        return 404;
    }
EOF
)
  fi

  sudo tee "/etc/nginx/sites-available/${SERVICE_NAME}" >/dev/null <<EOF
server {
    listen 80;
    server_name $DOMAIN;

    client_max_body_size ${NGINX_CLIENT_MAX_BODY_SIZE_MB}m;

    location = /admin {
        return 301 /admin/;
    }

    location ^~ /admin/ {
        alias $ADMIN_WEB_DIR/dist/;
        try_files \$uri \$uri/ /admin/index.html;
    }

    location = $HEALTHZ_PATH {
        proxy_pass http://127.0.0.1:$APP_PORT/readyz;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location ^~ $API_PREFIX/ {
        proxy_pass http://127.0.0.1:$APP_PORT/api/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location ^~ $UPLOADS_PREFIX/ {
        proxy_pass http://127.0.0.1:$APP_PORT/uploads/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

$root_block
}
EOF
fi

sudo rm -f /etc/nginx/sites-enabled/default
sudo ln -sf "/etc/nginx/sites-available/${SERVICE_NAME}" "/etc/nginx/sites-enabled/${SERVICE_NAME}"
sudo nginx -t
sudo systemctl daemon-reload
sudo systemctl enable "${SERVICE_NAME}"
sudo systemctl enable --now "${SERVICE_NAME}-backup.timer"
sudo systemctl enable --now "${SERVICE_NAME}-restore-drill.timer"
sudo systemctl enable --now "${SERVICE_NAME}-ops-health.timer"
sudo systemctl enable nginx
sudo systemctl reload nginx
REMOTE

if [[ -n "$CERTBOT_EMAIL" ]]; then
  ssh "$SERVER_HOST" "sudo certbot --nginx --non-interactive --agree-tos --redirect -m '$CERTBOT_EMAIL' -d '$DOMAIN'"
else
  cat <<EOF
Server bootstrap completed.

Next steps:
1. Create $APP_ENV_FILE with mode 0600 and owner $SERVICE_USER:$SERVICE_GROUP.
2. Run the versioned release entry: bash scripts/deploy-backend-on-server.sh
3. Verify nginx routes (mode: $NGINX_SITE_MODE).
4. Confirm the service UID is non-root: systemctl show $SERVICE_NAME -p User -p Group
5. Run certbot manually:
   ssh $SERVER_HOST "sudo certbot --nginx -d $DOMAIN"
EOF
fi
