# Backend

这个目录是 `caipu-miniapp` 的 Go 后端项目起点。

当前初始化结果：

- 路由框架：`chi`
- 配置加载：环境变量 + `configs/local.env`
- 日志：标准库 `slog`
- 数据库：`SQLite`
- 迁移：启动时自动执行 `migrations/*.sql`
- 健康检查：`GET /livez`（进程存活）、`GET /readyz`（依赖就绪）；
  `/healthz`、`/api/healthz` 是 readiness 兼容别名
- 已实现首批业务闭环：`auth + kitchens + invite + recipe + upload`
- 已新增后台能力：`/api/admin/* + AI 审计日志 + 运行时配置中心`

技术评估结论：

- 当前阶段前后端放在一个仓库、后端独立放到 `backend/` 目录最合适
- `Go + chi + SQLite` 足够覆盖共享空间第一版需求
- `SQLite` 采用 `WAL` 和 `busy_timeout`，并将 `MaxOpenConns` 设为 `1`，优先保证稳定性
- 先把配置、迁移、路由、健康检查和数据库底座跑通，再逐步加登录、空间、邀请、菜谱模块

本地启动：

```bash
cd backend
cp configs/example.env configs/local.env
chmod 600 configs/local.env
APP_ENV_FILE=configs/local.env go run ./cmd/server
```

- 进程环境变量优先于 env file；`APP_ENV_FILE` 指向的文件缺失、格式错误或权限宽于
  `0600` 时拒绝启动。
- 只有进程明确设置 `APP_ENV=local` 且未设置 `APP_ENV_FILE` 时，才自动补读
  `configs/local.env` / `.env`；非 local 环境不会自动读取本地 dotenv。
- typed 环境变量若格式非法会在监听端口前聚合报错，错误只包含变量名，不回显值。
- 可用 `APP_ENV_FILE=... go run ./cmd/server -check-config` 只校验配置，不打开数据库。
- 可用 `go run ./cmd/server -version` 输出 JSON 构建身份；开发构建回退为
  `releaseId=dev`、`gitCommit/buildTime=unknown`，正式 release 由 ldflags 注入。

开发与重构自验：

```bash
cd backend
go test ./...
go vet ./...
go test -race ./...
go mod verify
govulncheck ./...
```

- 核心包按“业务服务 / 外部协议 / 持久化 / 规则”组织，应用装配集中在 `internal/app/`。
- 涉及迁移、配置、鉴权或跨模块接线时，除定向测试外必须执行后端全量测试。
- `.github/workflows/backend.yml` 固定使用 Go `1.26.5`，执行格式、测试、覆盖率、
  `vet`、竞态、模块校验和可达漏洞扫描。
- 请求 body 默认限制为 1 MiB；饮食管家流式请求限制为 2 MiB；图片上传限制为
  `UPLOAD_MAX_IMAGE_MB + 1 MiB`，超限统一返回 413。
- 所有 HTTP 响应带 `X-Request-ID`；请求完成日志使用路由模板并记录状态、耗时、
  error code/type chain 和 allowlist 业务 ID，不记录原始 path/query。客户端 500 保持
  通用文案，可用 request ID 关联后端脱敏错误链。
- 全局 `slog` handler 会集中遮蔽 Bearer/JWT、Cookie、API Key、SESSDATA、URL 敏感部分
  和 AI request body；新增日志字段必须继续复用该 logger，不直接输出请求体。
- 所有外部成功响应都有场景化字节上限：文本 AI 为 2 MiB、饮食管家/B 站/Sidecar
  为 4 MiB、图片 base64 JSON 为 16 MiB、微信登录为 64 KiB；超限统一按 502 处理，
  原始 Provider 响应只进入脱敏日志/审计，不作为公开错误文案。
- 管理员/用户登录和邀请码预览/接受启用 IP + 凭据目标双维度短期限流；只有本机
  反向代理提供的 `X-Real-IP` 会被信任。
- 每个 owner 只有一个显式默认空间；菜谱和地点响应包含从 1 开始的 `version`，对应
  PUT/PATCH 必须回传该值。过期版本返回 409，客户端应刷新后由用户确认重试，禁止自动
  重放旧表单。菜谱、地点、菜单、邀请和空间关键写入会在提交事务内再次检查 membership。
- AI 路由 scene 与运行时配置 group（含 B 站配置）同样返回整数 `version`；管理端保存
  必须携带 `expectedVersion/version`，过期写返回 409。B 站配置、旧值审计和版本递增在
  同一事务提交，审计失败整体回滚。
- AI Provider 邮件告警以 failure streak + SQLite outbox 持久化；跨阈值记录和 enqueue
  同事务，原子 claim 防止并发重复认领，失败退避并由后台 worker 重试。SMTP 不支持
  幂等键，若邮件已发送而 sent 状态尚未落库时进程崩溃，仍存在极小重复投递窗口。
- `schema_migrations` 保存 migration 文件 SHA-256；历史表首次启动会回填，之后文件字节
  改变即拒绝启动/readiness。新增 migration 序号必须唯一，两个既有 `019_*` 是唯一例外。

生产运维约定：

- `GET /livez` 只证明进程仍在；`GET /readyz` 会在 2 秒上限内检查 DB ping、当前
  release 的迁移是否全部应用、SQLite 父目录和 uploads 是否可写。就绪响应头
  `X-Release-ID` 与 JSON 的 release/commit/build time/Go toolchain 由构建时 ldflags 注入。
- 服务器源码发布统一使用仓库根目录的
  `bash scripts/deploy-backend-on-server.sh`。脚本先测试和校验配置，再做 SQLite Online
  Backup、在备份副本预演迁移、构建 `releases/<release-id>`、原子切换 `current`，重启后
  要求连续 readiness 成功且 release ID 相符；失败恢复上一二进制，不自动反向执行 SQL。
  每个 release 的 `manifest.env` 同时记录二进制 SHA-256、Git commit、构建时间、Go
  toolchain、migration 数量和集合摘要，`migrations.sha256` 保存逐文件校验和。
- `backend/scripts/deploy.sh` 已废弃，禁止再覆盖运行中的固定二进制。
- `scripts/backup.sh` 使用 `sqlite3 .backup` 打包 `app.db + uploads.tar.gz + metadata +
  SHA256SUMS`；`scripts/verify-backup.sh` 会校验哈希、`PRAGMA quick_check` 并在临时目录
  解包 uploads。初始化脚本安装每日备份与每周恢复校验 timer；生产应通过
  `/etc/caipu-backend-backup.env` 配置 `OFFSITE_BACKUP_TARGET`，并建议设置
  `REQUIRE_OFFSITE_BACKUP=1`。
- bootstrap 默认将 journald 全局上限设为 `512M`、最长保留 `14day`（可用
  `JOURNAL_SYSTEM_MAX_USE` / `JOURNAL_MAX_RETENTION_SEC` 覆盖），并安装
  `caipu-backend-ops-health.timer`。巡检覆盖 5xx、worker error、磁盘、备份年龄和恢复
  演练年龄；生产仍需让监控平台订阅失败 unit，详见
  [后端运维告警策略](../docs/backend-operations-alert-policy.md)。
- 当前后端可维护性重构范围、优先级、进度与验证记录见
  [Go 后端可维护性重构路线图](../docs/backend-refactor-roadmap-2026-07-14.md)。
- R0～R13 已完成：核心契约/CI、Admin HTTP 域边界、airouter/appsettings HTTP 注入和
  recipe worker deadline 生命周期均已落地；DATA-005、DB-001、OPS-003 也已按并发和
  运维证据完成。R14/R15 的统计仓储与配置分组继续保持条件
  观察，只有出现查询计划或模块参数漏接证据时才启动，避免机械拆分。

和 B 站自动解析相关的可选配置：

- `RECIPE_AUTO_PARSE_ENABLED`
- `RECIPE_AUTO_PARSE_INTERVAL_SECONDS`
- `RECIPE_AUTO_PARSE_BATCH_SIZE`
- `RECIPE_AUTO_PARSE_MAX_ATTEMPTS`
- `RECIPE_AUTO_PARSE_RETRY_BASE_SECONDS`
- `RECIPE_AUTO_PARSE_STALE_SECONDS`
- `RECIPE_IMAGE_MIRROR_ENABLED`
- `RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS`
- `RECIPE_IMAGE_MIRROR_BATCH_SIZE`
- `CREDENTIALS_SECRET`
- `CREDENTIALS_KEY_VERSION`
- `CREDENTIALS_PREVIOUS_KEYS`（仅在凭据密钥轮换窗口临时配置旧 keyring）
- `APP_SETTINGS_ACCESS_MODE`
- `APP_ADMIN_OPENIDS`
- `APP_SETTINGS_ALLOWED_OPENIDS`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD_HASH`
- `ADMIN_JWT_SECRET`
- `ADMIN_COOKIE_PATH`（开发/standalone 为 `/api/admin`；当前线上共享前缀为
  `/caipu-api/admin`）
- `HEALTH_NGINX_SERVICE_NAME`
- `HEALTH_BACKEND_SERVICE_NAME`
- `HEALTH_SIDECAR_SERVICE_NAME`
- `HEALTH_BACKEND_BASE_URL`（留空时按 `APP_ADDR` 端口生成本机探测地址）
- `AI_BASE_URL`
- `AI_API_KEY`
- `AI_MODEL`
- `AI_TIMEOUT_SECONDS`
- `AI_FLOWCHART_BASE_URL`
- `AI_FLOWCHART_API_KEY`
- `AI_FLOWCHART_MODEL`
- `AI_FLOWCHART_ENDPOINT_MODE`
- `AI_FLOWCHART_RESPONSE_FORMAT`
- `AI_FLOWCHART_TIMEOUT_SECONDS`
- `RECIPE_FLOWCHART_ENABLED`
- `RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED`
- `RECIPE_FLOWCHART_INTERVAL_SECONDS`
- `RECIPE_FLOWCHART_BATCH_SIZE`
- `AI_TITLE_ENABLED`
- `AI_TITLE_BASE_URL`
- `AI_TITLE_API_KEY`
- `AI_TITLE_MODEL`
- `AI_TITLE_STREAM`
- `AI_TITLE_TEMPERATURE`
- `AI_TITLE_MAX_TOKENS`
- `AI_TITLE_TIMEOUT_SECONDS`
- `AI_ALERT_ENABLED`
- `AI_ALERT_FAILURE_THRESHOLD`
- `AI_ALERT_SMTP_HOST`
- `AI_ALERT_SMTP_PORT`
- `AI_ALERT_SMTP_USERNAME`
- `AI_ALERT_SMTP_PASSWORD`
- `AI_ALERT_FROM_EMAIL`
- `AI_ALERT_TO_EMAILS`
- `DIET_ASSISTANT_AI_BASE_URL`
- `DIET_ASSISTANT_AI_API_KEY`
- `DIET_ASSISTANT_AI_MODEL`
- `DIET_ASSISTANT_AI_THINKING_TYPE`
- `DIET_ASSISTANT_AI_REASONING_EFFORT`
- `DIET_ASSISTANT_AI_TIMEOUT_SECONDS`
- `LINKPARSE_SIDECAR_ENABLED`
- `LINKPARSE_SIDECAR_BASE_URL`
- `LINKPARSE_SIDECAR_TIMEOUT_SECONDS`
- `LINKPARSE_SIDECAR_API_KEY`
- `AMAP_PLACE_PREVIEW_ENABLED`
- `AMAP_WEB_SERVICE_KEY`
- `AMAP_PLACE_PREVIEW_DEFAULT_CITY`
- `AMAP_PLACE_PREVIEW_TIMEOUT_SECONDS`
- `AMAP_PLACE_PREVIEW_MAX_ATTEMPTS`
- `AMAP_PLACE_PREVIEW_QPS_DELAY_MS`
- `INVITE_SHARE_FONT_PATH`
- `INVITE_SHARE_FONT_BOLD_PATH`

饮食管家 AI 配置默认使用 LongCat OpenAI-compatible 接口：

- `DIET_ASSISTANT_AI_BASE_URL=https://api.longcat.chat/openai/v1`
- `DIET_ASSISTANT_AI_MODEL=LongCat-2.0-Preview`
- `DIET_ASSISTANT_AI_API_KEY` 填 LongCat API Key
- `DIET_ASSISTANT_AI_THINKING_TYPE` 可选填 `enabled` 或 `disabled`；留空时不发送
  `thinking` 字段，使用 provider 默认值
- `DIET_ASSISTANT_AI_REASONING_EFFORT` 可选填 `high` 或 `max`；仅在 thinking
  未禁用时随请求发送

当前 `POST /api/diet-assistant/chat/stream` 会先用非流式 tools 请求判断是否需要
调用工具，执行工具阶段会通过 SSE 发送 `status` / `tool_start` / `tool_done`
等状态事件；写库类工具成功时，`tool_done` 会附带 `mutation`，例如
`recipe_created`。随后后端会把工具结果交给模型，并以 `delta` 流式返回最终回复。
单 SSE 事件和累计可见内容各限制为 256 KiB，工具块和工具参数各限制为 64 KiB；
流式失败只返回稳定文案与 `requestId`，小程序会把该 ID 附到错误提示中用于排障。
已支持的工具：

- `get_recipe_count`：按当前空间统计菜谱数量，支持餐别和状态过滤
- `search_recipes_by_name`：按菜谱名或食材模糊查询当前空间菜谱，支持餐别和状态过滤，默认返回 5 条，最多返回 10 条
- `get_recipe_by_id`：根据菜谱 ID 获取当前空间内的菜谱详情，返回食材、步骤、备注和来源链接等信息
- `parse_and_add_recipe_from_url`：根据 B 站 / 小红书链接解析菜谱内容，提取食材和步骤，并保存到当前空间美食库

当前版本不暴露单独添加食材的饮食管家 tool；食材会随链接解析出的菜谱一起保存。

聊天记录会在一次用户消息和助手回复都成功完成后，以事务保存到
`diet_assistant_messages`，存储维度为 `user_id + kitchen_id`。历史读取和清空接口
都会校验当前用户仍是该空间成员。

应用级 B 站配置页访问控制：

- 默认使用 `APP_SETTINGS_ACCESS_MODE=admin`。
- `APP_SETTINGS_ACCESS_MODE=all`：所有登录用户都能进入隐藏设置页
- `APP_SETTINGS_ACCESS_MODE=admin`：只有 `APP_ADMIN_OPENIDS` 里的用户能进入
- `APP_SETTINGS_ACCESS_MODE=whitelist`：`APP_ADMIN_OPENIDS` 和 `APP_SETTINGS_ALLOWED_OPENIDS` 里的用户都能进入
- `APP_ADMIN_OPENIDS` / `APP_SETTINGS_ALLOWED_OPENIDS` 都用用户 `openid`，多个值用英文逗号分隔
- 开发登录时，`openid` 形如 `dev:alice`

正式微信登录接线：

1. 把 `configs/local.env` 或线上环境变量中的 `WECHAT_APP_ID` 设为和 `manifest.json` 里 `mp-weixin.appid` 相同的值
2. 填入对应小程序后台的 `WECHAT_APP_SECRET`
3. 确保前端 `utils/app-config.js` 指向已配置在微信小程序后台的 `HTTPS` 服务域名
4. 前端切到 `wechat` 或 `auto + 非 localhost 域名` 模式
5. 真机登录时，前端会把当前小程序 `appId` 一并带给后端；后端若发现和配置不一致，会直接返回明确错误

填充演示数据：

```bash
cd backend
go run ./cmd/seed-demo
```

这会确保本地数据库里存在两个开发用户 `alice` / `recipe-user`，并生成一个共享空间 `联调试吃空间` 以及多条早餐、正餐、想吃、吃过样例数据，方便直接联调前端。

B 站自动解析 POC 说明见：[docs/bilibili-link-parser-poc.md](./docs/bilibili-link-parser-poc.md)

小红书接入评估与 sidecar 方案见：

- [docs/xiaohongshu-link-parser-feasibility.md](./docs/xiaohongshu-link-parser-feasibility.md)
- [docs/xiaohongshu-sidecar-api-plan.md](./docs/xiaohongshu-sidecar-api-plan.md)
- [docs/xiaohongshu-integration-guide.md](./docs/xiaohongshu-integration-guide.md)
- [docs/xiaohongshu-cloud-deploy.md](./docs/xiaohongshu-cloud-deploy.md)
- [docs/linkparse-sidecar-refactor-migration.md](./docs/linkparse-sidecar-refactor-migration.md)
- AI 多 Provider 配置与轮询 / 降级设计见：[../docs/ai-multi-provider-routing-design.md](../docs/ai-multi-provider-routing-design.md)
- 后台 `AI Provider` 页面是 `summary / title / flowchart` 多节点路由的唯一运维入口；
  `配置中心` 不再展示旧单节点 AI 分组。
- `AI Provider` 文本类 `chat/completions` 节点支持通过 Provider `extra`
  透传 `thinking_type=auto|enabled|disabled` 和 `reasoning_effort=high|max`；
  `auto` 不发送字段，`thinking_type=disabled` 会清空并禁止发送
  `reasoning_effort`。DeepSeek `deepseek-v4-flash` 标题清洗节点建议关闭
  thinking，并保持 `maxTokens=64`、`timeoutSeconds=3-5`。
- `AI Provider -> flowchart` 的 `images/generations` 节点支持配置
  `size / quality / background / output_format / output_compression / n`；
  对 `gpt-image-*` 模型，后端默认解析 `b64_json`，不会随请求发送
  `response_format` 字段。
- 后台 `配置中心 -> AI Provider 告警` 支持对多 Provider 路由配置连续异常邮件告警，
  默认按同一 Provider 连续异常 `3` 次触发一次邮件；QQ 邮箱推荐使用
  `smtp.qq.com:587` + SMTP 授权码
- 后台 `配置中心 -> 小程序功能开关 -> AI 助手入口` 控制小程序首页底部中间按钮：
  开启时打开饮食管家，关闭时打开添加菜谱弹层；小程序通过
  `GET /api/public/app-config` 拉取配置。该开关默认关闭，配置缺失或拉取失败时保持关闭。

接口口径说明：

- 下列路径均为后端进程原生暴露的路由口径，例如 `http://127.0.0.1:8080/api/...`
- 当前线上共享域名经 nginx 转发后，后台前端实际访问前缀为 `/caipu-api/*`
- 现网入口与路径分流约定见 `docs/cloud-server-config-overview.md`

当前可用接口：

- `GET /healthz`
- `GET /api/healthz`
- `GET /api/public/app-config`
- `POST /api/admin/auth/login`
- `POST /api/admin/auth/logout`
- `GET /api/admin/auth/me`
- `GET /api/admin/dashboard/overview`
- `GET /api/admin/dashboard/failures`
- `GET /api/admin/dashboard/trends`
- `GET /api/admin/server-health/overview`
- `GET /api/admin/ai/jobs`
- `GET /api/admin/ai/jobs/{id}`
- `GET /api/admin/ai/calls`
- `GET /api/admin/ai-routing/scenes`
- `GET /api/admin/ai-routing/scenes/{scene}`
- `PUT /api/admin/ai-routing/scenes/{scene}`
- `POST /api/admin/ai-routing/scenes/{scene}/test`
- `GET /api/admin/runtime-settings`
- `PUT /api/admin/runtime-settings/groups/{group}`
- `POST /api/admin/runtime-settings/groups/{group}/test`
- `GET /api/admin/runtime-settings/audits`
- `POST /api/auth/wechat/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`
- `PATCH /api/auth/profile`
- `GET /api/app-settings/bilibili-session`
- `PUT /api/app-settings/bilibili-session`
- `DELETE /api/app-settings/bilibili-session`
- `GET /api/invite-codes/{code}`
- `GET /api/kitchens`
- `POST /api/kitchens`
- `PATCH /api/kitchens/{kitchenID}`
- `GET /api/kitchens/{kitchenID}/members`
- `GET /api/kitchens/{kitchenID}/stats`
- `DELETE /api/kitchens/{kitchenID}/members/me`
- `GET /api/invites/{token}`
- `GET /api/invites/{token}/share-image`
- `POST /api/kitchens/{kitchenID}/invites`
- `POST /api/invites/{token}/accept`
- `POST /api/invite-codes/{code}/accept`
- `POST /api/link-parsers/preview`
- `POST /api/link-parsers/bilibili`
- `POST /api/link-parsers/xiaohongshu`
- `POST /api/kitchens/{kitchenID}/add-link-previews`
- `GET /api/diet-assistant/messages`
- `DELETE /api/diet-assistant/messages`
- `POST /api/diet-assistant/chat/stream`
- `GET /api/kitchens/{kitchenID}/meal-plans`
- `PUT /api/kitchens/{kitchenID}/meal-plans/{planDate}/draft`
- `DELETE /api/kitchens/{kitchenID}/meal-plans/{planDate}/draft`
- `POST /api/kitchens/{kitchenID}/meal-plans/{planDate}/submit`
- `POST /api/kitchens/{kitchenID}/meal-plans/{planDate}/draft-from-submitted`
- `DELETE /api/kitchens/{kitchenID}/meal-plans/{planDate}/submitted`
- `GET /api/kitchens/{kitchenID}/places`
- `POST /api/kitchens/{kitchenID}/places`
- `GET /api/places/{placeID}`
- `PUT /api/places/{placeID}`
- `PATCH /api/places/{placeID}/status`
- `DELETE /api/places/{placeID}`
- `GET /api/kitchens/{kitchenID}/recipes`
- `POST /api/kitchens/{kitchenID}/recipes`
- `GET /api/recipes/{recipeID}`
- `PUT /api/recipes/{recipeID}`
- `POST /api/recipes/{recipeID}/reparse`
- `POST /api/recipes/{recipeID}/flowchart`
- `PATCH /api/recipes/{recipeID}/pin`
- `PATCH /api/recipes/{recipeID}/status`
- `DELETE /api/recipes/{recipeID}`
- `POST /api/uploads/images`

仅本地环境开放的调试接口：

- `POST /api/auth/dev-login`

本地联调示例：

```bash
cd backend
token=$(curl -s -X POST http://127.0.0.1:8080/api/auth/dev-login \
  -H 'Content-Type: application/json' \
  -d '{"identity":"alice"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

curl -s http://127.0.0.1:8080/api/auth/me \
  -H "Authorization: Bearer $token"

curl -s http://127.0.0.1:8080/api/kitchens \
  -H "Authorization: Bearer $token"

invite=$(curl -s -X POST http://127.0.0.1:8080/api/kitchens/1/invites \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  -d '{}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

curl -s http://127.0.0.1:8080/api/invites/$invite

curl -s -X POST http://127.0.0.1:8080/api/kitchens/1/recipes \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  -d '{"title":"番茄滑蛋牛肉","ingredient":"","link":"https://www.bilibili.com/video/BV1aWCEYHErc","mealType":"main","status":"wishlist","parsedContent":{"ingredients":["番茄滑蛋牛肉 1份","正餐常用配菜 适量","基础调味 适量"],"steps":["先整理这道菜的核心做法。","按自己的口味调整成容易复刻的版本。","做完以后补充口感和火候记录。"]}}'

curl -s -X POST http://127.0.0.1:8080/api/uploads/images \
  -H "Authorization: Bearer $token" \
  -F "file=@/path/to/your-image.png"
```

只执行迁移：

```bash
cd backend
APP_ENV_FILE=configs/local.env go run ./cmd/server -migrate-only
```

当前邀请策略：

- 空间成员可以创建邀请链接
- 默认 `72` 小时过期，默认最多使用 `10` 次
- `GET /api/invites/{token}` 可在登录前预览邀请
- `GET /api/invites/{token}/share-image` 会按邀请信息动态生成分享卡封面，适合给小程序 `onShareAppMessage.imageUrl` 直接使用
- 同一用户重复接受同一空间邀请时会幂等返回 `alreadyMember=true`
- 动态分享图默认会尝试读取系统中文字体；若线上环境没有可用中文字体，可显式配置：
  - `INVITE_SHARE_FONT_PATH`
  - `INVITE_SHARE_FONT_BOLD_PATH`

当前上传策略：

- 图片接口为 `POST /api/uploads/images`
- 默认支持 `jpg`、`png`、`webp`、`gif`
- 图片继续是无需登录即可读取的公开资源；文件名包含密码学随机后缀，静态路由不提供
  目录列表，也不会暴露不符合当前生成规则的人工文件或非图片文件
- `APP_ENV=local` 时 `UPLOAD_PUBLIC_BASE_URL` 可留空，并按直连请求域名拼接地址；
  非 local 环境必须显式配置无凭据、query、fragment 的正式 HTTPS 基址，否则拒绝启动
- 上传响应不信任 `X-Forwarded-Host` / `X-Forwarded-Proto`；非 local 环境始终使用
  `UPLOAD_PUBLIC_BASE_URL`，避免 Host Header 污染持久化 URL
- 上传后的静态资源通过 `/uploads/*` 提供访问；成功响应带一年 immutable 缓存和
  `nosniff`、CSP、referrer/CORP 安全头，404/405 使用 `no-store`
- Handler 请求上限之外，上传 Service 会再次执行单文件大小限制；写入采用 `.part`
  临时文件，失败或超限会清理，成功后才原子切换为公开文件名
- 小红书/B 站自动解析拿到的第三方图片会先以外链形式写入，随后由后台低频任务异步转存到本地 uploads
- 图片转存频率由 `RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS` 和 `RECIPE_IMAGE_MIRROR_BATCH_SIZE` 控制

当前 B 站自动解析策略：

- 保存菜谱时如果识别到 B 站链接，会自动标记为 `parseStatus=pending`
- 后端定时任务按 `RECIPE_AUTO_PARSE_INTERVAL_SECONDS` 扫描并解析
- 如果配置了全局 `SESSDATA`，解析器会自动带上登录态请求 B 站字幕接口
- 隐藏设置页的访问权限由 `APP_SETTINGS_ACCESS_MODE` 控制，当前默认值是 `admin`
- 成功后会自动补齐 `ingredient`、`parsedContent.ingredients`、`parsedContent.steps`
- 失败后会保留 `parseStatus=failed` 和 `parseError`
- 可通过 `POST /api/recipes/{recipeID}/reparse` 手动重新入队

后台管理平台说明：

- 前端后台工程位于仓库根目录 `admin-web/`
- 默认通过同域 `https://你的域名/admin/` 访问
- 后台登录和小程序登录分离，走独立账号：
  - `ADMIN_USERNAME`
  - `ADMIN_PASSWORD_HASH`
  - `ADMIN_JWT_SECRET`（非 `local` 环境必须配置，且必须与 `JWT_SECRET`、`CREDENTIALS_SECRET` 独立）
  - `ADMIN_COOKIE_PATH`（必须匹配浏览器实际 Admin API 前缀）
- 后台只接受 HttpOnly Cookie；Cookie 使用 `SameSite=Strict` 和窄 Path，所有写接口还需
  携带与会话绑定的 `X-CSRF-Token`，Admin Web 会在登录或恢复会话后自动维护该值
- 用户 Bearer 包含 `jti` 和数据库 `token_version`；`POST /api/auth/logout` 会递增版本，
  立即撤销该用户所有旧 token，重新登录后签发新版本 token
- `app_runtime_settings` 支持在线覆盖 `AI / sidecar` 相关配置，并在更新后自动失效本地缓存
- `app_setting_audits` 记录后台保存、测试，以及移动端 `Bilibili SESSDATA` 更新动作

当前步骤图生成策略：

- 用户仍可通过 `POST /api/recipes/{recipeID}/flowchart` 手动入队生成步骤图
- `RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED=true` 时，后端会在当前没有 `pending / processing` 步骤图任务时，自动补位 1 条候选菜谱入队
- 自动补位会优先挑选“还没生成步骤图、做法已完整、当前不在自动解析中”的菜谱
- 若当前没有可用的首次生成候选，会继续回补 `flowchart_status=failed` 且尚未生成图片的菜谱
- 当前仍不会自动重生成已有但过期的步骤图
- 步骤图队列状态和生成结果会更新 `flowchart_*` 字段，但不会改 `recipe.updated_at`，避免首页列表被后台任务打乱顺序
- `PATCH /api/recipes/{recipeID}/status` 只切换 `想吃 / 吃过` 状态，不会改 `recipe.updated_at`，避免首页列表被轻操作打乱顺序

空间统计接口：

- `GET /api/kitchens/{kitchenID}/stats?window=30d`
- `window` 支持 `7d`、`30d`、`90d`、`all`，默认 `30d`
- 响应格式为 `{ "stats": ... }`，包含 `overview`、`recipes`、`places`、`mealPlans`、
  `members`、`trends`、`actions`
- 统计接口复用空间成员校验；非成员不能读取目标空间统计
- 菜谱“吃过趋势”来自 `recipe_status_events`，并维护内部字段 `recipes.done_at`
- 打卡点“去过趋势”来自 `place_status_events`
- 打卡点消费统计保留原始 `price` 文本，同时写入内部结构化字段
  `price_amount_cents / price_currency / price_type`；现有打卡点 API 不暴露这些内部字段

当前小红书预留策略：

- 保存菜谱时如果识别到小红书链接，也会自动标记为 `parseStatus=pending`
- 后端 worker 会按平台路由到小红书 sidecar
- 是否真正启用，由 `LINKPARSE_SIDECAR_ENABLED` 控制
- 当前仓库已包含通用解析 sidecar、provider 路由和 RedNote 初始化脚本，目录在 [linkparse-sidecar](../sidecars/linkparse-sidecar)
- 小红书 provider 默认策略由 sidecar 自身的 `XHS_PROVIDER_DEFAULT` 控制，推荐保持 `auto`
- 更完整的使用说明见 [xiaohongshu-integration-guide.md](./docs/xiaohongshu-integration-guide.md)

后续第一批建议实现顺序：

1. 增加管理员移除成员和角色调整
2. 增加编辑记录和冲突提示
3. 增加邀请撤销和邀请列表
4. 增加操作日志和基础测试
5. 接入正式微信登录域名和线上部署配置
