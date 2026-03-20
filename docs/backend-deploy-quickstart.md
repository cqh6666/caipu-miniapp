# 后端快速部署指令

这份文档按 `Ubuntu 22.04/24.04 + systemd + nginx` 编写，目标是把当前 `backend/` 服务快速部署到云服务器，并通过你自己的 `HTTPS` 域名对外提供接口。

说明：

- 这套方案不要求服务器安装 Go
- 推荐在本地先编译 Linux 二进制，再上传到服务器
- 真实 `WECHAT_APP_SECRET` 不要写进 Git，只在服务器 `.env` 中填写
- 如果你担心密钥已经暴露，建议后续去微信公众平台重置一次
- 仓库里已经提供了几份部署相关脚本：
  - `backend/scripts/bootstrap-server.sh` 用于首次初始化服务器
  - `backend/scripts/deploy.sh` 用于后续每次发版
- 如果你当前线上环境是“服务器拉源码并本机编译”，还可以使用：
  - `backend/scripts/deploy-server-build.sh`
- 这些脚本都支持按环境变量覆盖默认值；最常用的是 `SERVER_HOST`，其余变量按脚本场景分别使用

## 0.5 更快的脚本方式

如果你想少敲命令，可以直接用仓库里的脚本。

## 0.6 当前线上实际部署方式

如果当前云服务器还是按源码拉取再本机编译的方式部署，实际命令如下：

```bash
cd /srv/caipu-miniapp
git pull

cd /srv/caipu-miniapp/backend
go build -o bin/server ./cmd/server
systemctl restart caipu-backend
```

说明：

- 这是当前线上环境的“实际生效流程”，和下面那套本地交叉编译 + 上传二进制的方案不同
- 如果下次要快速重发版，优先先按这组命令检查
- 现在仓库里已经补了一份同逻辑脚本：`backend/scripts/deploy-server-build.sh`

可以直接在本地执行：

```bash
cd /path/to/caipu-miniapp/backend
SERVER_HOST=root@你的服务器IP \
./scripts/deploy-server-build.sh
```

默认值对应当前线上约定：

- `REPO_DIR=/srv/caipu-miniapp`
- `BACKEND_DIR=/srv/caipu-miniapp/backend`
- `BINARY_PATH=/srv/caipu-miniapp/backend/bin/server`
- `SERVICE_NAME=caipu-backend`
- `APP_PORT=8080`

如果以后线上目录或服务名改了，可以通过环境变量覆盖：

```bash
cd /path/to/caipu-miniapp/backend
SERVER_HOST=root@你的服务器IP \
REPO_DIR=/srv/caipu-miniapp \
SERVICE_NAME=caipu-backend \
APP_PORT=8080 \
./scripts/deploy-server-build.sh
```

首次初始化服务器：

```bash
cd /path/to/caipu-miniapp/backend
SERVER_HOST=root@你的服务器IP \
DOMAIN=your-domain.example \
ENABLE_UFW=1 \
./scripts/bootstrap-server.sh
```

如果你还想自动申请证书，可以额外带上邮箱：

```bash
cd /path/to/caipu-miniapp/backend
SERVER_HOST=root@你的服务器IP \
DOMAIN=your-domain.example \
CERTBOT_EMAIL=你的邮箱 \
./scripts/bootstrap-server.sh
```

后续每次部署：

```bash
cd /path/to/caipu-miniapp/backend
SERVER_HOST=root@你的服务器IP \
DOMAIN=your-domain.example \
ENV_FILE=configs/local.env \
./scripts/deploy.sh
```

说明：

- `ENV_FILE` 是可选的；如果服务器上已经有正确的 `.env`，后续部署可以不传
- 由于 `configs/local.env` 已被 Git 忽略，用它上传不会把密钥带进仓库
- 如果你的服务不是监听 `127.0.0.1:8080`，可以额外传入 `APP_PORT=你的端口`

## 0. 准备项

先确认以下条件都满足：

- 域名已经把 A 记录指向云服务器公网 IP
- 服务器系统为 Ubuntu
- 服务器已开放 `80` 和 `443`
- 你本地可以 `ssh` 到服务器

下面命令默认使用这些变量，请先替换：

```bash
export SERVER_HOST="root@你的服务器IP"
export APP_DIR="/opt/caipu-miniapp/backend"
export DOMAIN="your-domain.example"
export APP_PORT="8080"
```

## 1. 本地编译并上传

在你本机 `backend/` 目录执行：

```bash
cd /path/to/caipu-miniapp/backend
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/caipu-miniapp-server ./cmd/server
```

创建服务器目录：

```bash
ssh "$SERVER_HOST" "mkdir -p $APP_DIR"
ssh "$SERVER_HOST" "mkdir -p $APP_DIR/data/uploads"
```

上传二进制和迁移文件：

```bash
scp dist/caipu-miniapp-server "$SERVER_HOST:$APP_DIR/"
scp -r migrations "$SERVER_HOST:$APP_DIR/"
```

如果你还想把演示数据脚本一并传上去，可以额外上传源码；如果只是正式运行服务，上面这两步就够了。

## 2. 服务器安装 nginx 和证书工具

在服务器执行：

```bash
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx curl
```

如果你用的是 `ufw`，再执行：

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw --force enable
```

## 3. 写服务器环境变量

在服务器执行下面命令创建运行配置：

```bash
sudo tee "$APP_DIR/.env" >/dev/null <<'EOF'
APP_NAME=caipu-miniapp-backend
APP_ENV=production
APP_ADDR=127.0.0.1:8080
LOG_LEVEL=info

JWT_SECRET=请替换成openssl生成的随机字符串
JWT_EXPIRE_HOURS=720

WECHAT_APP_ID=wxafe7c4144c9c063e
WECHAT_APP_SECRET=请填写真实微信小程序密钥

SQLITE_PATH=/opt/caipu-miniapp/backend/data/app.db
SQLITE_BUSY_TIMEOUT_MS=5000
MIGRATION_DIR=/opt/caipu-miniapp/backend/migrations

UPLOAD_DIR=/opt/caipu-miniapp/backend/data/uploads
UPLOAD_PUBLIC_BASE_URL=https://your-domain.example/uploads
UPLOAD_MAX_IMAGE_MB=10

INVITE_DEFAULT_EXPIRE_HOURS=72
INVITE_DEFAULT_MAX_USES=10
EOF
```

其中这两项要按你的实际环境替换：

- `APP_ADDR` 默认写的是 `127.0.0.1:8080`，如果你改了 `APP_PORT`，这里也要一起改
- `UPLOAD_PUBLIC_BASE_URL` 要改成你的正式 `HTTPS` 域名

生成 `JWT_SECRET`：

```bash
openssl rand -hex 32
```

把生成结果填回上面的 `.env` 文件即可。

再把目录权限收紧一点：

```bash
sudo chmod 600 "$APP_DIR/.env"
```

## 4. 先手动跑一次迁移和启动验证

在服务器执行：

```bash
cd "$APP_DIR"
set -a
source .env
set +a
./caipu-miniapp-server -migrate-only
./caipu-miniapp-server
```

保持这个终端不要关，再开一个新终端验证：

```bash
curl "http://127.0.0.1:${APP_PORT}/healthz"
curl "http://127.0.0.1:${APP_PORT}/api/healthz"
```

确认正常后，回到服务终端按 `Ctrl + C` 停掉。

如果你只是想先放一些测试数据，再额外走一次本地种子工具上传源码部署，或者我后面再给你补一版“服务器执行 seed-demo”的方式。

## 5. 配置 systemd 常驻运行

在服务器执行：

```bash
sudo tee /etc/systemd/system/caipu-miniapp-backend.service >/dev/null <<'EOF'
[Unit]
Description=Caipu Miniapp Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/caipu-miniapp/backend
ExecStart=/opt/caipu-miniapp/backend/caipu-miniapp-server
Restart=always
RestartSec=3
EnvironmentFile=/opt/caipu-miniapp/backend/.env

[Install]
WantedBy=multi-user.target
EOF
```

启动并设为开机自启：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now caipu-miniapp-backend
sudo systemctl status caipu-miniapp-backend --no-pager
```

查看日志：

```bash
sudo journalctl -u caipu-miniapp-backend -f
```

## 6. 配置 nginx 反向代理

在服务器执行：

```bash
sudo tee /etc/nginx/sites-available/caipu-miniapp-backend >/dev/null <<'EOF'
server {
    listen 80;
    server_name your-domain.example;

    client_max_body_size 20m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF
```

如果你前面把 `APP_PORT` 改成了别的端口，这里的 `proxy_pass` 也要一起改掉。

启用配置并检查：

```bash
sudo ln -sf /etc/nginx/sites-available/caipu-miniapp-backend /etc/nginx/sites-enabled/caipu-miniapp-backend
sudo nginx -t
sudo systemctl reload nginx
```

## 7. 申请 HTTPS 证书

在服务器执行：

```bash
sudo certbot --nginx -d "$DOMAIN"
```

证书安装完成后，再检查自动续期：

```bash
sudo systemctl status certbot.timer --no-pager
sudo certbot renew --dry-run
```

## 8. 最终验证

在服务器执行：

```bash
curl "https://$DOMAIN/healthz"
curl "https://$DOMAIN/api/healthz"
```

如果返回 `status=ok`，后端就已经对外可用了。

再到微信小程序后台确认：

- `https://你的域名` 已加入“服务器域名”
- 前端已使用 [utils/app-config.js](/Users/chenqh114/uni-app-projects/caipu-miniapp/utils/app-config.js) 里的正式配置
- 服务器 `.env` 中的 `WECHAT_APP_SECRET` 正确

## 9. 后续更新代码

以后你每次发布后端，可以直接重复下面几步：

如果你已经用了上面的脚本，通常直接再次执行下面这条就够了：

```bash
cd /path/to/caipu-miniapp/backend
SERVER_HOST="$SERVER_HOST" \
DOMAIN="$DOMAIN" \
APP_PORT="$APP_PORT" \
ENV_FILE=configs/local.env \
./scripts/deploy.sh
```

如果你想手动发布，也可以按下面步骤走：

本地重新编译上传：

```bash
cd /path/to/caipu-miniapp/backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/caipu-miniapp-server ./cmd/server
scp dist/caipu-miniapp-server "$SERVER_HOST:$APP_DIR/"
scp -r migrations "$SERVER_HOST:$APP_DIR/"
```

服务器重启服务：

```bash
ssh "$SERVER_HOST" "cd $APP_DIR && ./caipu-miniapp-server -migrate-only"
ssh "$SERVER_HOST" "sudo systemctl restart caipu-miniapp-backend"
ssh "$SERVER_HOST" "sudo systemctl status caipu-miniapp-backend --no-pager"
```

## 10. 常用排查命令

```bash
sudo systemctl status caipu-miniapp-backend --no-pager
sudo journalctl -u caipu-miniapp-backend -n 200 --no-pager
sudo nginx -t
curl -I "https://$DOMAIN/healthz"
ls -lah /opt/caipu-miniapp/backend/data
ls -lah /opt/caipu-miniapp/backend/data/uploads
```
