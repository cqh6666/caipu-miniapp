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

if [[ -z "$SERVER_HOST" ]]; then
  echo "SERVER_HOST is required, for example: root@1.2.3.4" >&2
  exit 1
fi

if [[ -z "$DOMAIN" ]]; then
  echo "DOMAIN is required, for example: www.gxm1227.top" >&2
  exit 1
fi

ssh "$SERVER_HOST" "APP_DIR='$APP_DIR' SERVICE_NAME='$SERVICE_NAME' DOMAIN='$DOMAIN' BINARY_NAME='$BINARY_NAME' APP_PORT='$APP_PORT' ENABLE_UFW='$ENABLE_UFW' bash -s" <<'REMOTE'
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
3. Run certbot manually:
   ssh $SERVER_HOST "sudo certbot --nginx -d $DOMAIN"
EOF
fi
