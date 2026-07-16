# 云服务器配置概览

本文档用于记录当前 `caipu-miniapp` 在线上云服务器上的实际部署形态，
方便后续排障、发版和迁移。文档只记录结构、路径、端口、服务名和配置
入口，不记录任何真实密钥。

仓库文档更新时间：`2026-07-16 14:00 CST`

生产主机最后实机核对为：`2026-07-16 14:00 CST`。后端已完成版本化 release、在线备份、
readiness、Go `1.26.5` 构建身份与最小权限 unit 的生产迁移，并已应用主机全局 journald
`512M/14day` 留存策略；异机备份、故障注入、真实回滚和 ops-health 外部告警仍待执行，
下文不把这些待办提前当成线上事实。

## 1. 服务器基础信息

| 项目 | 当前值 |
| --- | --- |
| 主机名 | `lavm-v3bskiukyp` |
| 系统 | `Ubuntu 22.04.3 LTS` |
| 虚拟化 | `kvm` |
| 架构 | `x86_64` |
| CPU | `2 vCPU` |
| 内存 | `1.9 GiB` |
| Swap | `0 B` |
| 线上域名 | `https://www.gxm1227.top` |
| 项目仓库目录 | `/srv/caipu-miniapp` |

说明：

- 当前项目采用“同一台云服务器承载多个服务”的方式运行。
- `nginx` 负责对外 `80/443` 入口和路径分流。
- Hapi 根站点、小程序后端、linkparse sidecar 目前都在这台机器上。

## 2. 当前流量拓扑

```text
Internet
  -> nginx :80/:443
     -> /                    -> Hapi Hub                -> 127.0.0.1:3006
     -> /admin/              -> admin-web dist          -> /srv/caipu-miniapp/admin-web/dist
     -> /caipu-api/          -> caipu-backend /api/*    -> 127.0.0.1:8080
     -> /caipu-uploads/      -> caipu-backend /uploads/*-> 127.0.0.1:8080
     -> /caipu-healthz       -> caipu-backend readiness -> 127.0.0.1:8080
```

当前监听端口：

| 端口 | 服务 | 说明 |
| --- | --- | --- |
| `80` | `nginx` | HTTP，统一跳转 HTTPS |
| `443` | `nginx` | HTTPS，对外主入口 |
| `3006` | `hapi-hub` | 根站点承载服务 |
| `8080` | `caipu-backend` | Go 后端，仅监听 `127.0.0.1` |
| `8091` | `caipu-linkparse-sidecar` | 解析 sidecar，仅监听 `127.0.0.1` |

## 3. nginx 路由说明

当前线上站点配置文件：

- `/etc/nginx/conf.d/www.gxm1227.top.conf`

当前关键路由：

| 路径 | 去向 | 用途 |
| --- | --- | --- |
| `/` | `http://127.0.0.1:3006` | Hapi 根站点 |
| `/admin/` | `/srv/caipu-miniapp/admin-web/dist/` | 后台管理前端静态资源 |
| `/caipu-api/` | `http://127.0.0.1:8080/api/` | 小程序后端 API 与后台 API，`proxy_read_timeout / proxy_send_timeout = 300s` |
| `/caipu-uploads/` | `http://127.0.0.1:8080/uploads/` | 上传文件访问 |
| `/caipu-healthz` | `http://127.0.0.1:8080/healthz`；后端将其作为 `/readyz` 兼容别名 | 后端 readiness |

关键约束：

- 不要随意改 `location /`，它当前明确承载 Hapi 根站点。
- `admin-web` 当前生产环境默认把 `VITE_API_BASE` 指到
  `/caipu-api`，这是为了兼容现网 nginx 前缀，而不是标准 `/api`。
- `/caipu-api/` 当前显式放宽了 `proxy_read_timeout 300` 与
  `proxy_send_timeout 300`，用于覆盖 `AI Provider` 后台测试等长耗时请求，
  避免在 60 秒附近被代理层提前截断。
- 如果将来想统一改回 `/api`，要同时调整：
  - nginx 路由
  - `admin-web/.env.production`
  - 小程序前端 `utils/app-config.js` 的访问口径

## 4. systemd 服务清单

### 4.1 caipu-backend

用途：

- 承载小程序后端 API
- 承载后台管理 API：`/api/admin/*`
- 承载上传、邀请、AI 审计和运行时配置

服务文件：

- `/etc/systemd/system/caipu-backend.service`

2026-04-23 最后一次实机核对的旧配置（已归档）：

- `WorkingDirectory=/srv/caipu-miniapp/backend`
- `Environment=APP_ENV_FILE=/srv/caipu-miniapp/backend/configs/prod.env`
- `ExecStart=/srv/caipu-miniapp/backend/bin/server`

2026-07-16 已实机迁移并验证的当前配置：

- `User/Group=caipu-backend`，进程 UID 非 0；
- `WorkingDirectory=/srv/caipu-miniapp/backend/current`；
- `ExecStart=/srv/caipu-miniapp/backend/current/server`；
- `current` 原子指向 `releases/<release-id>`；
- 只允许写 `backend/data`，启用 `NoNewPrivileges`、`PrivateTmp`、
  `ProtectSystem=strict`、`ProtectHome=true` 和 `UMask=0077`；
- 当前 release 为 `20260716T043954Z-c928d193493a`，commit 为 `c928d19`，二进制实际和
  manifest/健康接口均报告 Go `1.26.5`；
- 每日 `caipu-backend-backup.timer` 与每周 `caipu-backend-restore-drill.timer` 已启用；
- 主机全局 journald `512M/14day` 已于 2026-07-16 13:42 应用；用户确认不导出历史日志后，
  journal 从 3.9 GiB 清理至 456 MiB，根分区从 96% 降至 87%、可用空间增至 5.2 GiB；
- 生产 Docker 悬空镜像已于 2026-07-16 13:53 清理，释放 2.277 GB；镜像从 26 个降至
  14 个，根分区进一步降至 81%、可用空间增至 7.3 GiB；
- root/AstrBot npm、AstrBot uv 下载缓存和零容器引用的有标签镜像已于 2026-07-16 14:00
  清理；根分区进一步降至 68%、可用空间增至 13 GiB，Docker 剩余镜像均有容器引用；
- 每五分钟 `caipu-backend-ops-health.timer` 尚未安装或启用，外部告警接收策略仍待确定；
- `HEALTH_BACKEND_SERVICE_NAME` / `HEALTH_BACKEND_BASE_URL` 与实际 unit/端口一致，
  release manifest 可映射 release、commit、build time、Go toolchain 和 migration 集合。

`configs/prod.env` 必须为 `0600`；后端会拒绝读取权限更宽、缺失或格式错误的显式
env file。若 systemd 同时注入同名变量，进程环境值优先，文件只补缺失项。

常用命令：

```bash
systemctl status caipu-backend --no-pager
systemctl restart caipu-backend
journalctl -u caipu-backend -n 200 --no-pager
journalctl -u caipu-backend -f
```

### 4.2 caipu-linkparse-sidecar

用途：

- 承载 B 站 / 小红书相关的 linkparse sidecar
- 当前监听 `127.0.0.1:8091`

服务文件：

- `/etc/systemd/system/caipu-linkparse-sidecar.service`

当前关键配置：

- `WorkingDirectory=/srv/caipu-miniapp/sidecars/linkparse-sidecar`
- `EnvironmentFile=/srv/caipu-miniapp/runtime/linkparse-sidecar/linkparse-sidecar.env`
- `ExecStart=/usr/bin/npm start`

常用命令：

```bash
systemctl status caipu-linkparse-sidecar --no-pager
systemctl restart caipu-linkparse-sidecar
journalctl -u caipu-linkparse-sidecar -n 200 --no-pager
```

### 4.3 hapi-hub

用途：

- 当前承载根站点 `/`
- 监听 `0.0.0.0:3006`

服务文件：

- `/etc/systemd/system/hapi-hub.service`
- `/etc/default/hapi-hub`

当前公开可见的环境键：

- `CLI_API_TOKEN`
- `HAPI_LISTEN_HOST`
- `HAPI_LISTEN_PORT`
- `HAPI_PUBLIC_URL`

常用命令：

```bash
systemctl status hapi-hub --no-pager
systemctl restart hapi-hub
journalctl -u hapi-hub -n 200 --no-pager
```

### 4.4 hapi-runner

用途：

- Hapi 相关 runner，同样以 `systemd` 常驻

服务文件：

- `/etc/systemd/system/hapi-runner.service`
- `/etc/default/hapi-runner`

常用命令：

```bash
systemctl status hapi-runner --no-pager
systemctl restart hapi-runner
journalctl -u hapi-runner -n 200 --no-pager
```

## 5. 项目目录约定

### 5.1 仓库目录

| 路径 | 用途 |
| --- | --- |
| `/srv/caipu-miniapp` | 仓库根目录 |
| `/srv/caipu-miniapp/backend` | Go 后端源码 |
| `/srv/caipu-miniapp/backend/current` | 当前版本原子符号链接，已指向 `releases/20260716T043954Z-c928d193493a` |
| `/srv/caipu-miniapp/backend/releases/<release-id>` | 版本化二进制、迁移和 manifest |
| `/srv/caipu-miniapp/backend/data` | SQLite 和上传目录 |
| `/srv/caipu-miniapp/backend/backups` | 原子备份包与校验清单 |
| `/srv/caipu-miniapp/admin-web` | 后台前端工程 |
| `/srv/caipu-miniapp/admin-web/dist` | 后台前端生产构建产物 |
| `/srv/caipu-miniapp/runtime/linkparse-sidecar` | sidecar 运行时配置 |

### 5.2 关键配置文件

| 路径 | 是否纳入 Git | 说明 |
| --- | --- | --- |
| `backend/configs/example.env` | 是 | 本地/线上环境变量示例 |
| `backend/configs/prod.env` | 否 | 当前线上后端运行配置 |
| `runtime/linkparse-sidecar/linkparse-sidecar.env` | 否 | 当前 sidecar 运行配置 |
| `admin-web/.env.development` | 是 | 后台开发环境 API 前缀 |
| `admin-web/.env.production` | 是 | 后台生产环境 API 前缀 |
| `/etc/nginx/conf.d/www.gxm1227.top.conf` | 否 | 当前 nginx 站点配置 |

后端 env file（包括本地 `configs/local.env` 和线上 `configs/prod.env`）统一使用
`chmod 600`，禁止组用户或其他用户读取。

## 6. 环境变量口径

### 6.1 后端 `backend/configs/prod.env`

当前文件包含以下几类键：

- 基础运行：
  `APP_NAME`、`APP_ENV`、`APP_ADDR`、`LOG_LEVEL`
- 鉴权与后台登录：
  `JWT_SECRET`、`JWT_EXPIRE_HOURS`、`ADMIN_USERNAME`、
  `ADMIN_PASSWORD_HASH`、`ADMIN_JWT_SECRET`、`ADMIN_COOKIE_PATH`
- 微信登录：
  `WECHAT_APP_ID`、`WECHAT_APP_SECRET`
- SQLite 与上传：
  `SQLITE_PATH`、`SQLITE_BUSY_TIMEOUT_MS`、`MIGRATION_DIR`、
  `UPLOAD_DIR`、`UPLOAD_PUBLIC_BASE_URL`、`UPLOAD_MAX_IMAGE_MB`
- 解析与流程图：
  `AI_*`、`AI_FLOWCHART_*`、`AI_TITLE_*`
- sidecar：
  `LINKPARSE_SIDECAR_*`
- 后台/隐藏设置页权限：
  `APP_SETTINGS_ACCESS_MODE`、`APP_ADMIN_OPENIDS`
- worker：
  `RECIPE_AUTO_PARSE_*`、`RECIPE_FLOWCHART_*`、
  `RECIPE_IMAGE_MIRROR_*`

说明：

- 该文件目前位于仓库目录中，但未纳入 Git。
- 修改后需要重启 `caipu-backend` 才会生效。
- 当前共享域名经 `/caipu-api/admin/*` 访问后台 API，故生产
  `ADMIN_COOKIE_PATH` 必须为 `/caipu-api/admin`；standalone `/api/*` 部署改用
  `/api/admin`。Admin Cookie 为 Strict/HttpOnly，写请求另有会话绑定 CSRF 校验。
- 非 local 环境的 `UPLOAD_PUBLIC_BASE_URL` 是必填项，必须为不含凭据、query、fragment
  的正式 HTTPS 地址；上传图片仍为公开资源，但应用禁止目录列表和非随机生成文件访问。

### 6.2 sidecar `runtime/linkparse-sidecar/linkparse-sidecar.env`

当前文件包含以下几类键：

- 服务监听与鉴权：
  `PORT`、`LINKPARSE_INTERNAL_API_KEY`
- B 站 provider：
  `BILI_PROVIDER_*`
- 小红书 provider：
  `XHS_PROVIDER_*`、`XHS_REDNOTE_*`
- 转写能力：
  `XHS_TRANSCRIPT_*`
- 依赖路径：
  `FFMPEG_PATH`

说明：

- 修改后需要重启 `caipu-linkparse-sidecar`。

## 7. 当前发布方式

### 7.1 后端发布

当前线上实际口径是“服务器本机拉代码并编译”：

```bash
cd /srv/caipu-miniapp
bash scripts/deploy-backend-on-server.sh
```

当前发布契约：执行后端测试和配置校验，使用 `sqlite3 .backup` 创建发布前快照，
在快照副本预演 migration，再构建版本化 release、原子切换 `current`；只有 `/readyz`
连续成功且 `X-Release-ID` 与目标一致才成功。失败恢复上一二进制，不自动反向执行 SQL。
生产旧 unit 已于 2026-07-16 迁移完成；后续禁止恢复为直接覆盖 `bin/server` 的发布方式。

说明：

- 当前推荐按服务拆开执行：

```bash
bash scripts/deploy-backend-on-server.sh
bash scripts/deploy-admin-web-on-server.sh
bash scripts/deploy-linkparse-sidecar-on-server.sh
```

- `scripts/deploy-on-server.sh` 仍保留为聚合入口，但只建议在你明确需要一次
  同时处理 `backend + admin-web` 时使用。
- 先看计划而不执行构建，使用：

```bash
PLAN_ONLY=1 bash scripts/deploy-backend-on-server.sh
```

- 当前这台 `2 vCPU / 1.9 GiB RAM / 0 swap` 的线上机，脚本默认允许
  `backend` 单独构建，但会拒绝 `admin-web` 构建或前后端一起构建；这不是
  脚本坏了，而是根据当前机器规格优先拦住更容易把整机打满的
  `npm install` / `vite build` 链路。
- 只有明确在维护窗口、并且接受风险时，才允许显式强制：

```bash
ALLOW_LOW_RESOURCE_BUILD=1 bash scripts/deploy-admin-web-on-server.sh
ALLOW_LOW_RESOURCE_BUILD=1 bash scripts/deploy-linkparse-sidecar-on-server.sh
ALLOW_LOW_RESOURCE_BUILD=1 DEPLOY_SCOPE=all bash scripts/deploy-on-server.sh
```

- 构建默认通过 `nice + ionice` 降低优先级，并把后端 `go build` 限制为
  `GOMAXPROCS=1`；但这只能“稍微减轻”，不能从根本上解决低配机本机构建
  的卡顿风险。
- `admin-web` 默认只有在 `package.json / package-lock.json` 变更或
  `node_modules` 缺失时才执行 `npm install`；其余情况下直接复用现有依赖
  做 `vite build`。
- 如果只是同步源码而不做构建，可用：

```bash
DEPLOY_SCOPE=none bash scripts/deploy-on-server.sh
```

### 7.2 后台管理前端发布

```bash
cd /path/to/caipu-miniapp
DOMAIN=www.gxm1227.top \
  bash scripts/upload-admin-web-dist.sh
```

说明：

- 脚本会优先从你本机 `~/.ssh/config` 自动识别
  `one-hub-server / oh-prod / my-cloud`；如果后续切换服务器，也可以在执行时
  显式覆盖 `SERVER_HOST=...`。
- 当前更推荐“本地或 CI 构建 `dist` -> 上传到服务器 -> 原子替换远端
  `/srv/caipu-miniapp/admin-web/dist`”这条路线，尤其适合当前这台
  `2 vCPU / 1.9 GiB RAM / 0 swap` 的线上机。
- `scripts/upload-admin-web-dist.sh` 会在本地调用
  `scripts/build-admin-web.sh`，随后通过 `scp + ssh + tar` 上传并切换远端
  `dist` 目录；如果设置了 `DOMAIN` 或 `VERIFY_URL`，脚本还会在上传后做
  一次 `curl -I` 验证。
- `scripts/deploy-admin-web-on-server.sh` 是 `admin-web` 独立入口，会固定按
  `admin-web` 范围处理，不会顺手重启 `caipu-backend`，但它仍然属于
  “服务器本机构建”路线，更适合作为维护窗口里的兜底方案。
- `admin-web` 是静态站点，不需要单独的 `systemd` 常驻服务。
- 当前由 `nginx` 直接托管 `/srv/caipu-miniapp/admin-web/dist`。
- 纯静态资源更新通常不需要重启 `nginx`；如果改了 nginx 配置，才需要：

```bash
nginx -t
systemctl reload nginx
```

### 7.3 如果改了 sidecar

```bash
cd /srv/caipu-miniapp/sidecars/linkparse-sidecar
bash ../../scripts/deploy-linkparse-sidecar-on-server.sh
```

说明：

- `scripts/deploy-linkparse-sidecar-on-server.sh` 只处理
  `sidecars/linkparse-sidecar` 与 `caipu-linkparse-sidecar`。
- 如果只是 sidecar JS 代码改动，它会直接重启服务，不会额外跑
  `npm install`。
- 只有在 `package.json / package-lock.json` 变更或 `node_modules` 缺失时，
  才会尝试安装 sidecar 依赖。

## 8. 常用检查命令

### 8.1 服务状态

```bash
systemctl status nginx --no-pager
systemctl status caipu-backend --no-pager
systemctl status caipu-linkparse-sidecar --no-pager
systemctl status hapi-hub --no-pager
systemctl status hapi-runner --no-pager
```

### 8.2 端口检查

```bash
ss -lntp | rg '(:80|:443|:3006|:8080|:8091)'
```

### 8.3 HTTP 检查

```bash
curl -I https://www.gxm1227.top/
curl -I https://www.gxm1227.top/admin/
curl -fsS https://www.gxm1227.top/caipu-healthz
curl -i https://www.gxm1227.top/caipu-api/admin/auth/me
```

说明：

- `/caipu-api/admin/auth/me` 未登录时返回 `401`，这是正常现象。
- 如果这里返回 `404`，通常表示后端二进制还没更新到带后台 API 的版本。
- `/caipu-healthz` 应返回 readiness，并在响应头包含 `X-Release-ID`；进程存活但 DB、迁移
  或数据目录异常时应返回 `503`。内部单独用 `/livez` 判断进程存活。

## 9. 当前运维注意事项

1. 不要把 `backend/configs/prod.env`、`runtime/*/*.env` 之类的真实配置
   提交进 Git。
2. 修改 `nginx` 时，优先新增精确路径或高优先级前缀路径，不要误改
   `location /`，否则会影响 Hapi 根站点。
3. 当前小程序和后台前端都依赖现网的 `/caipu-api` 前缀，统一改口径前要
   先做全链路评估。
4. 后端代码更新后，只有 `git pull` 不够；统一执行
   `bash scripts/deploy-backend-on-server.sh`，禁止手工覆盖 `bin/server`。
5. 后台前端代码更新后，只有拉代码不够，必须重新构建 `admin-web/dist`。
6. 如果只改了 `admin-web` 页面代码，一般不需要重启 `caipu-backend`。
7. 当前线上机器只有 `2 vCPU / 1.9 GiB RAM / 0 swap`，不要在业务高峰期
   运行 `npm install`、`vite build`、`go build`、`go test ./...`
   这类高占用任务。
8. 优先使用拆开的独立脚本：
   `deploy-backend-on-server.sh`、`deploy-admin-web-on-server.sh`、
   `deploy-linkparse-sidecar-on-server.sh`；聚合脚本只在确实需要联动部署时
   再用。
9. `scripts/deploy-admin-web-on-server.sh` 现在默认会拒绝在该机器上做
   `admin-web` 构建；先用 `PLAN_ONLY=1` 看计划，需要仅同步源码时用
   `DEPLOY_SCOPE=none bash scripts/deploy-on-server.sh`。
10. 如果确实要在服务器上强制构建 `admin-web`、执行 sidecar 依赖安装，
    或前后端一起构建，必须显式带 `ALLOW_LOW_RESOURCE_BUILD=1`，并尽量放到
    维护窗口执行。
11. 生产进程 UID、release/`/etc` 不可写、data/WAL 可写、字体读取、服务重启和 readiness
    已验证；仍需配置 `/etc/caipu-backend-backup.env` 的异机 rsync 目标并完成真实恢复，
    以及让告警平台订阅 `caipu-backend-ops-health.service` 的 Warning/Critical。
12. 当前根分区使用率为 68%，可用空间 13 GiB；journald 已限制为 `512M/14day`，Docker
    悬空/未引用镜像和可重建下载缓存已清理。本次未删除备份或业务数据，后续仍禁止为
    腾空间误删 SQLite/WAL/uploads，并需观察 Jdog 审计、Docker 自动更新和本地备份增长。

## 10. 推荐的后续补强

1. 配置异机 rsync，记录远端副本与真实恢复结果；本机 backup/restore-drill 已验证并启用。
2. 继续观察 journald、Jdog 审计、Docker 自动更新和本地备份增长，评估共享主机 68% 的
   容量余量。
3. 安装并启用 `ops-health` timer，接入失败 unit 的外部告警接收端，制造测试
   5xx/worker/备份过期信号并
   记录真实投递结果。
4. 实测健康失败自动回滚和关闭 DB/破坏目录权限时的 readiness/liveness 分离；进程 UID、
   沙箱、字体、WAL、重启和 Go 生产工具链已完成实机核对。
