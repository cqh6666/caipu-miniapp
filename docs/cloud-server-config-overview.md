# 云服务器配置概览

本文档用于记录当前 `caipu-miniapp` 在线上云服务器上的实际部署形态，
方便后续排障、发版和迁移。文档只记录结构、路径、端口、服务名和配置
入口，不记录任何真实密钥。

最后核对时间：`2026-04-08 23:50 CST`

## 1. 服务器基础信息

| 项目 | 当前值 |
| --- | --- |
| 主机名 | `lavm-v3bskiukyp` |
| 系统 | `Ubuntu 22.04.3 LTS` |
| 虚拟化 | `kvm` |
| 架构 | `x86_64` |
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
     -> /caipu-healthz       -> caipu-backend /healthz  -> 127.0.0.1:8080
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
| `/caipu-api/` | `http://127.0.0.1:8080/api/` | 小程序后端 API 与后台 API |
| `/caipu-uploads/` | `http://127.0.0.1:8080/uploads/` | 上传文件访问 |
| `/caipu-healthz` | `http://127.0.0.1:8080/healthz` | 后端健康检查 |

关键约束：

- 不要随意改 `location /`，它当前明确承载 Hapi 根站点。
- `admin-web` 当前生产环境默认把 `VITE_API_BASE` 指到
  `/caipu-api`，这是为了兼容现网 nginx 前缀，而不是标准 `/api`。
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

当前关键配置：

- `WorkingDirectory=/srv/caipu-miniapp/backend`
- `Environment=APP_ENV_FILE=/srv/caipu-miniapp/backend/configs/prod.env`
- `ExecStart=/srv/caipu-miniapp/backend/bin/server`

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
| `/srv/caipu-miniapp/backend/bin/server` | 后端编译产物 |
| `/srv/caipu-miniapp/backend/data` | SQLite 和上传目录 |
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

## 6. 环境变量口径

### 6.1 后端 `backend/configs/prod.env`

当前文件包含以下几类键：

- 基础运行：
  `APP_NAME`、`APP_ENV`、`APP_ADDR`、`LOG_LEVEL`
- 鉴权与后台登录：
  `JWT_SECRET`、`JWT_EXPIRE_HOURS`、`ADMIN_USERNAME`、
  `ADMIN_PASSWORD_HASH`、`ADMIN_JWT_SECRET`
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
git pull

cd /srv/caipu-miniapp/backend
go build -o bin/server ./cmd/server
systemctl restart caipu-backend
```

### 7.2 后台管理前端发布

```bash
cd /srv/caipu-miniapp
npm --prefix admin-web install
npm --prefix admin-web run build
```

说明：

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
npm install
systemctl restart caipu-linkparse-sidecar
```

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

## 9. 当前运维注意事项

1. 不要把 `backend/configs/prod.env`、`runtime/*/*.env` 之类的真实配置
   提交进 Git。
2. 修改 `nginx` 时，优先新增精确路径或高优先级前缀路径，不要误改
   `location /`，否则会影响 Hapi 根站点。
3. 当前小程序和后台前端都依赖现网的 `/caipu-api` 前缀，统一改口径前要
   先做全链路评估。
4. 后端代码更新后，只有 `git pull` 不够，必须重新 `go build` 并重启
   `caipu-backend`。
5. 后台前端代码更新后，只有拉代码不够，必须重新构建 `admin-web/dist`。
6. 如果只改了 `admin-web` 页面代码，一般不需要重启 `caipu-backend`。

## 10. 推荐的后续补强

1. 把当前 nginx 站点配置模板同步沉淀到仓库 `docs/` 或部署脚本里，减少
   “服务器真实配置”和“仓库文档”漂移。
2. 考虑为 `admin-web` 补一个专门的发布脚本，例如：
   `scripts/deploy-admin-web.sh`，把安装依赖、构建和必要校验串起来。
3. 如果后续服务继续增多，建议把“路径路由图 + 服务清单 + 配置文件入口”
   维护成固定更新的运维基线文档。
