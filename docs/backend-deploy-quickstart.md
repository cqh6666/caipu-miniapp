# 后端部署、备份与回滚手册

更新时间：`2026-07-16 10:00 CST`

本文档是 `Ubuntu 22.04/24.04 + systemd + nginx + SQLite WAL` 的仓库内标准口径。
线上仍需在维护窗口执行首次 unit 迁移、异机备份和真实回滚演练，仓库测试不能替代这些
外部验收。

## 1. 标准目录与入口

```text
/srv/caipu-miniapp/backend/
├── configs/prod.env          # 0600，owner=caipu-backend
├── current -> releases/...   # 原子切换的当前 release
├── releases/<release-id>/
│   ├── server
│   ├── migrations/
│   └── manifest.env
├── data/
│   ├── app.db                # 同目录还会有 WAL/SHM
│   └── uploads/
└── backups/backup-.../
    ├── app.db
    ├── uploads.tar.gz
    ├── metadata.txt
    └── SHA256SUMS
```

统一入口：

- 已登录服务器：`bash scripts/deploy-backend-on-server.sh`
- 从本地经 SSH：`bash backend/scripts/deploy-server-build.sh`
- 聚合发布：`bash scripts/deploy-on-server.sh`，只在确实要联动多个模块时使用
- `backend/scripts/deploy.sh` 已废弃，不再允许固定路径覆盖二进制

## 2. 首次初始化

先确保服务器已克隆仓库到 `/srv/caipu-miniapp`，再从本地执行：

```bash
cd backend
SERVER_HOST=root@your-server \
DOMAIN=your-domain.example \
SERVICE_NAME=caipu-backend \
ENABLE_UFW=1 \
bash scripts/bootstrap-server.sh
```

初始化脚本会：

- 安装 nginx、curl、sqlite3、rsync 和 certbot；
- 创建不可登录用户/组 `caipu-backend`；
- 生成 `ExecStart=/srv/caipu-miniapp/backend/current/server` 的 unit；
- 仅授权应用写 `backend/data`，并启用 `NoNewPrivileges`、`PrivateTmp`、
  `ProtectSystem=strict`、`ProtectHome=true`、`UMask=0077` 和 15 秒停止上限；
- 把外部 `/caipu-healthz` 转发到内部 `/readyz`；
- 安装每日一致性备份与每周恢复校验 timer。

初始化不会在 `current` 尚不存在时强行启动后端。创建生产配置后再执行首次发布：

```bash
sudo cp backend/configs/example.env backend/configs/prod.env
sudoedit backend/configs/prod.env
sudo chown caipu-backend:caipu-backend backend/configs/prod.env
sudo chmod 600 backend/configs/prod.env
cd backend
APP_ENV_FILE=configs/prod.env go run ./cmd/server -check-config
```

首次发布前 `current/server` 还不存在，因此先在源码目录校验；填入真实独立密钥后再运行
发布脚本。非 local 环境至少确认：

- `APP_ENV=prod`、`APP_ADDR=127.0.0.1:8080`；
- `JWT_SECRET`、`ADMIN_JWT_SECRET`、`CREDENTIALS_SECRET` 是三个不同的强密钥；
- `ADMIN_COOKIE_PATH=/caipu-api/admin`（共享前缀部署）；
- `UPLOAD_PUBLIC_BASE_URL` 是正式 HTTPS 上传前缀；
- `HEALTH_BACKEND_SERVICE_NAME` 与实际 unit 一致，`HEALTH_BACKEND_BASE_URL` 指向本机
  后端监听地址；
- `WECHAT_APP_ID` 与小程序 manifest 一致，真实密钥不进入 Git。

## 3. 一次后端发布会做什么

```bash
cd /srv/caipu-miniapp
bash scripts/deploy-backend-on-server.sh
```

顺序固定为：

1. `git pull --ff-only` 并运行 `go test ./...`；
2. 构建注入 release ID、commit、build time、Go toolchain 的临时 release，并执行
   `-check-config`；
3. 用 `sqlite3 .backup` 创建发布前一致性备份；
4. 在备份数据库副本上运行全部迁移，预先发现 SQL/数据兼容问题；
5. 对生产数据库执行前向迁移；
6. 完成包含二进制 SHA-256 与逐文件 migration 集合的 release manifest，原子切换
   `current` 后 restart；
7. `/readyz` 连续 3 次成功且 `X-Release-ID` 等于目标 release 才算发布成功；
8. 失败时把 `current` 恢复到上一 release 并再次 restart/readiness。

自动回滚只恢复二进制。SQLite migration 必须保持前向兼容；脚本不会猜测或执行反向
SQL。若新迁移本身破坏旧二进制兼容性，应停止自动发布并按备份做人工数据恢复。

常用覆盖项：

```bash
PLAN_ONLY=1 bash scripts/deploy-backend-on-server.sh
RUN_BACKEND_TESTS=0 bash scripts/deploy-backend-on-server.sh   # 仅紧急窗口，需另有 CI 证据
READY_ATTEMPTS=45 READY_CONSECUTIVE_SUCCESSES=5 \
  bash scripts/deploy-backend-on-server.sh
```

发布脚本会校验 systemd `ExecStart` 必须指向 `current/server`。旧 unit 未迁移时会在备份和
迁移前失败，不会静默覆盖 `bin/server`。

## 4. 健康接口

| 路径 | 语义 | 失败状态 |
| --- | --- | --- |
| `/livez` | 进程仍能处理 HTTP，不访问 DB | 仅进程不可用时失败 |
| `/readyz` | DB ping、当前 release 迁移、SQLite/uploads 可写 | `503` |
| `/healthz` | `/readyz` 兼容别名 | `503` |
| `/api/healthz` | `/readyz` 兼容别名 | `503` |

```bash
curl -i http://127.0.0.1:8080/livez
curl -i http://127.0.0.1:8080/readyz
curl -i https://your-domain.example/caipu-healthz
```

就绪响应头包含 release ID，JSON 同时包含 release ID、commit、build time 和 Go
toolchain。发布判断不能只看进程 PID 或 `/livez`。可用
`current/server -version` 直接核对二进制身份。

## 5. 备份、异机副本与恢复演练

手工创建并验证：

```bash
cd /srv/caipu-miniapp/backend
bash scripts/backup.sh
bash scripts/verify-latest-backup.sh
systemctl list-timers 'caipu-backend-*'
```

`backup.sh` 禁止复制运行中的 `app.db`，使用 SQLite Online Backup 协议捕获已提交 WAL
页面；随后对快照执行 `PRAGMA quick_check`，归档 uploads，并生成 SHA-256 清单。备份目录
在同一文件系统内由隐藏 staging 目录原子 rename 完成。

生产建议创建 `/etc/caipu-backend-backup.env`：

```bash
sudo install -m 600 -o root -g root /dev/null /etc/caipu-backend-backup.env
sudoedit /etc/caipu-backend-backup.env
```

内容示例：

```dotenv
RETENTION_DAYS=14
OFFSITE_BACKUP_TARGET=backup-user@backup-host:/srv/backups/caipu-miniapp
REQUIRE_OFFSITE_BACKUP=1
RSYNC_RSH="ssh -i /etc/caipu-backend-backup-ed25519 -o BatchMode=yes"
```

SSH 私钥需让 `caipu-backend` 用户可读且保持 `0600`，远端账号只授予目标备份目录写权限。
设置 `REQUIRE_OFFSITE_BACKUP=1` 后，缺少异机目标会让备份明确失败并由 systemd 记录。

定时任务：

- `caipu-backend-backup.timer`：每日一致性备份；
- `caipu-backend-restore-drill.timer`：每周校验 SHA-256、SQLite quick check，并在
  `PrivateTmp` 中实际解包 uploads。
- `caipu-backend-ops-health.timer`：每五分钟检查 5xx、worker error、磁盘、备份年龄和
  恢复演练年龄；非零退出码需由生产监控平台订阅。

检查日志：

```bash
journalctl -u caipu-backend-backup.service -n 100 --no-pager
journalctl -u caipu-backend-restore-drill.service -n 100 --no-pager
journalctl -u caipu-backend-ops-health.service -n 100 --no-pager
```

bootstrap 默认把主机 journald 全局容量限制为 `512M`、最长保留 `14day`；共享主机可用
`JOURNAL_SYSTEM_MAX_USE`、`JOURNAL_MAX_RETENTION_SEC` 覆盖。完整阈值、退出码和生产
接收端验收见 [后端运维告警策略](backend-operations-alert-policy.md)。

## 6. 人工恢复步骤

恢复会覆盖线上数据，只能在维护窗口、确认目标 backup 后执行。先验证，不要直接操作：

```bash
BACKUP=/srv/caipu-miniapp/backend/backups/backup-YYYYMMDDTHHMMSSZ-release-pid
bash /srv/caipu-miniapp/backend/scripts/verify-backup.sh "$BACKUP"
```

确认后停止服务，把当前数据整体保留为时间戳目录，再从已验证包恢复：

```bash
sudo systemctl stop caipu-backend
cd /srv/caipu-miniapp/backend
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
sudo mkdir -m 700 "data.before-${stamp}"
sudo mv data/app.db data/app.db-wal data/app.db-shm "data.before-${stamp}/" 2>/dev/null || true
sudo mv data/uploads "data.before-${stamp}/uploads"
sudo install -m 600 -o caipu-backend -g caipu-backend "$BACKUP/app.db" data/app.db
sudo install -d -m 750 -o caipu-backend -g caipu-backend data/uploads
sudo tar -xzf "$BACKUP/uploads.tar.gz" -C data/uploads
sudo chown -R caipu-backend:caipu-backend data/uploads
sudo systemctl start caipu-backend
curl -fsS http://127.0.0.1:8080/readyz
```

若当前 release 需要备份中不存在的迁移，应先把 `current` 切到与备份 release 匹配的目录，
或经过评审后重新执行前向迁移。不要在不清楚 schema 的情况下删除 `schema_migrations`。

## 7. 最小权限验收

部署后执行：

```bash
systemctl show caipu-backend -p User -p Group -p ExecStart -p FragmentPath
pid="$(systemctl show caipu-backend -p MainPID --value)"
ps -o user,group,pid,command -p "$pid"
sudo -u caipu-backend test ! -w /etc
sudo -u caipu-backend test ! -w /srv/caipu-miniapp/backend/current
sudo -u caipu-backend test -w /srv/caipu-miniapp/backend/data
systemd-analyze security caipu-backend.service
```

预期进程 UID 非 0，只能写 SQLite/WAL/SHM 与 uploads；release、源码、`/etc` 不可写。
字体路径若配置在 release 外，必须只读可访问。权限或 DB 故障时 `/readyz` 应返回 503，
`/livez` 仍返回 200；真实破坏性演练必须先备份并安排维护窗口。

## 8. 常用排查

```bash
systemctl status caipu-backend --no-pager
journalctl -u caipu-backend -n 200 --no-pager
readlink -f /srv/caipu-miniapp/backend/current
cat /srv/caipu-miniapp/backend/current/manifest.env
cat /srv/caipu-miniapp/backend/current/migrations.sha256
/srv/caipu-miniapp/backend/current/server -version
curl -i http://127.0.0.1:8080/readyz
ls -lah /srv/caipu-miniapp/backend/releases
ls -lah /srv/caipu-miniapp/backend/backups
```

线上首次执行 bootstrap、备份异机复制、最小权限和真实回滚演练后，应把结果与时间回写
到 `docs/cloud-server-config-overview.md`，避免把仓库目标状态误写成已验证生产事实。
