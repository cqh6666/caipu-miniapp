#!/usr/bin/env bash
set -euo pipefail

SERVER_HOST="${SERVER_HOST:-}"
DOMAIN="${DOMAIN:-}"
APP_DIR="${APP_DIR:-/opt/caipu-miniapp/backend}"
ADMIN_WEB_DIR="${ADMIN_WEB_DIR:-/opt/caipu-miniapp/admin-web}"
SERVICE_NAME="${SERVICE_NAME:-caipu-miniapp-backend}"
BINARY_NAME="${BINARY_NAME:-caipu-miniapp-server}"
APP_PORT="${APP_PORT:-8080}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"
ENABLE_UFW="${ENABLE_UFW:-0}"
NGINX_SITE_MODE="${NGINX_SITE_MODE:-shared_prefix}"
API_PREFIX="${API_PREFIX:-/caipu-api}"
UPLOADS_PREFIX="${UPLOADS_PREFIX:-/caipu-uploads}"
HEALTHZ_PATH="${HEALTHZ_PATH:-/caipu-healthz}"
ROOT_PROXY_PASS="${ROOT_PROXY_PASS:-}"

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

ssh "$SERVER_HOST" "APP_DIR='$APP_DIR' ADMIN_WEB_DIR='$ADMIN_WEB_DIR' SERVICE_NAME='$SERVICE_NAME' DOMAIN='$DOMAIN' BINARY_NAME='$BINARY_NAME' APP_PORT='$APP_PORT' ENABLE_UFW='$ENABLE_UFW' NGINX_SITE_MODE='$NGINX_SITE_MODE' API_PREFIX='$API_PREFIX' UPLOADS_PREFIX='$UPLOADS_PREFIX' HEALTHZ_PATH='$HEALTHZ_PATH' ROOT_PROXY_PASS='$ROOT_PROXY_PASS' bash -s" <<'REMOTE'
set -euo pipefail

sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx curl

if [[ "$ENABLE_UFW" == "1" ]]; then
  sudo apt install -y ufw
  sudo ufw allow OpenSSH
  sudo ufw allow 80/tcp
  sudo ufw allow 443/tcp
  sudo ufw --force enable
fi

sudo mkdir -p "$APP_DIR" "$APP_DIR/data/uploads" "$ADMIN_WEB_DIR"

sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" >/dev/null <<EOF
[Unit]
Description=Caipu Miniapp Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/$BINARY_NAME
Restart=always
RestartSec=3
EnvironmentFile=$APP_DIR/.env

[Install]
WantedBy=multi-user.target
EOF

if [[ "$NGINX_SITE_MODE" == "standalone" ]]; then
  sudo tee "/etc/nginx/sites-available/${SERVICE_NAME}" >/dev/null <<EOF
server {
    listen 80;
    server_name $DOMAIN;

    client_max_body_size 20m;

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

    client_max_body_size 20m;

    location = /admin {
        return 301 /admin/;
    }

    location ^~ /admin/ {
        alias $ADMIN_WEB_DIR/dist/;
        try_files \$uri \$uri/ /admin/index.html;
    }

    location = $HEALTHZ_PATH {
        proxy_pass http://127.0.0.1:$APP_PORT/healthz;
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
sudo systemctl enable nginx
sudo systemctl reload nginx
REMOTE

if [[ -n "$CERTBOT_EMAIL" ]]; then
  ssh "$SERVER_HOST" "sudo certbot --nginx --non-interactive --agree-tos --redirect -m '$CERTBOT_EMAIL' -d '$DOMAIN'"
else
  cat <<EOF
Server bootstrap completed.

Next steps:
1. Upload your backend binary and migrations.
2. Upload $APP_DIR/.env to the server.
3. Verify nginx routes (mode: $NGINX_SITE_MODE).
4. Run certbot manually:
   ssh $SERVER_HOST "sudo certbot --nginx -d $DOMAIN"
EOF
fi
