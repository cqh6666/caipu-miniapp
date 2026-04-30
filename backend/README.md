# Backend

这个目录是 `caipu-miniapp` 的 Go 后端项目起点。

当前初始化结果：

- 路由框架：`chi`
- 配置加载：环境变量 + `configs/local.env`
- 日志：标准库 `slog`
- 数据库：`SQLite`
- 迁移：启动时自动执行 `migrations/*.sql`
- 健康检查：`GET /healthz`、`GET /api/healthz`
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
go run ./cmd/server
```

和 B 站自动解析相关的可选配置：

- `RECIPE_AUTO_PARSE_ENABLED`
- `RECIPE_AUTO_PARSE_INTERVAL_SECONDS`
- `RECIPE_AUTO_PARSE_BATCH_SIZE`
- `RECIPE_IMAGE_MIRROR_ENABLED`
- `RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS`
- `RECIPE_IMAGE_MIRROR_BATCH_SIZE`
- `CREDENTIALS_SECRET`
- `APP_SETTINGS_ACCESS_MODE`
- `APP_ADMIN_OPENIDS`
- `APP_SETTINGS_ALLOWED_OPENIDS`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD_HASH`
- `ADMIN_JWT_SECRET`
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
- `DIET_ASSISTANT_AI_TIMEOUT_SECONDS`
- `LINKPARSE_SIDECAR_ENABLED`
- `LINKPARSE_SIDECAR_BASE_URL`
- `LINKPARSE_SIDECAR_TIMEOUT_SECONDS`
- `LINKPARSE_SIDECAR_API_KEY`
- `INVITE_SHARE_FONT_PATH`
- `INVITE_SHARE_FONT_BOLD_PATH`

饮食管家 AI 配置默认使用 LongCat OpenAI-compatible 接口：

- `DIET_ASSISTANT_AI_BASE_URL=https://api.longcat.chat/openai/v1`
- `DIET_ASSISTANT_AI_MODEL=LongCat-2.0-Preview`
- `DIET_ASSISTANT_AI_API_KEY` 填 LongCat API Key

当前 `POST /api/diet-assistant/chat/stream` 会先用非流式 tools 请求判断是否需要
调用工具，执行工具阶段会通过 SSE 发送 `status` / `tool_start` / `tool_done`
等状态事件，再把工具结果交给模型并以 `delta` 流式返回最终回复。已支持的工具：

- `get_recipe_count`：按当前空间统计菜谱数量，支持餐别和状态过滤
- `search_recipes_by_name`：按菜谱名模糊查询当前空间菜谱，支持餐别和状态过滤
- `parse_and_add_recipe_from_url`：根据 B 站 / 小红书链接解析菜谱内容，提取食材和步骤，并保存到当前空间美食库

当前版本不暴露单独添加食材的饮食管家 tool；食材会随链接解析出的菜谱一起保存。

聊天记录会在一次用户消息和助手回复都成功完成后，以事务保存到
`diet_assistant_messages`，存储维度为 `user_id + kitchen_id`。历史读取和清空接口
都会校验当前用户仍是该空间成员。

应用级 B 站配置页访问控制：

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
- 后台 `配置中心 -> AI Provider 告警` 支持对多 Provider 路由配置连续异常邮件告警，
  默认按同一 Provider 连续异常 `3` 次触发一次邮件；QQ 邮箱推荐使用
  `smtp.qq.com:587` + SMTP 授权码

接口口径说明：

- 下列路径均为后端进程原生暴露的路由口径，例如 `http://127.0.0.1:8080/api/...`
- 当前线上共享域名经 nginx 转发后，后台前端实际访问前缀为 `/caipu-api/*`
- 现网入口与路径分流约定见 `docs/cloud-server-config-overview.md`

当前可用接口：

- `GET /healthz`
- `GET /api/healthz`
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
- `PATCH /api/auth/profile`
- `GET /api/app-settings/bilibili-session`
- `PUT /api/app-settings/bilibili-session`
- `DELETE /api/app-settings/bilibili-session`
- `GET /api/invite-codes/{code}`
- `GET /api/kitchens`
- `POST /api/kitchens`
- `PATCH /api/kitchens/{kitchenID}`
- `GET /api/kitchens/{kitchenID}/members`
- `DELETE /api/kitchens/{kitchenID}/members/me`
- `GET /api/invites/{token}`
- `GET /api/invites/{token}/share-image`
- `POST /api/kitchens/{kitchenID}/invites`
- `POST /api/invites/{token}/accept`
- `POST /api/invite-codes/{code}/accept`
- `POST /api/link-parsers/preview`
- `POST /api/link-parsers/bilibili`
- `POST /api/link-parsers/xiaohongshu`
- `GET /api/diet-assistant/messages`
- `DELETE /api/diet-assistant/messages`
- `POST /api/diet-assistant/chat/stream`
- `GET /api/kitchens/{kitchenID}/meal-plans`
- `PUT /api/kitchens/{kitchenID}/meal-plans/{planDate}/draft`
- `DELETE /api/kitchens/{kitchenID}/meal-plans/{planDate}/draft`
- `POST /api/kitchens/{kitchenID}/meal-plans/{planDate}/submit`
- `POST /api/kitchens/{kitchenID}/meal-plans/{planDate}/draft-from-submitted`
- `DELETE /api/kitchens/{kitchenID}/meal-plans/{planDate}/submitted`
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
go run ./cmd/server -migrate-only
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
- `UPLOAD_PUBLIC_BASE_URL` 为空时，会按当前请求域名自动拼接图片地址
- 上传后的静态资源通过 `/uploads/*` 提供访问
- 小红书/B 站自动解析拿到的第三方图片会先以外链形式写入，随后由后台低频任务异步转存到本地 uploads
- 图片转存频率由 `RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS` 和 `RECIPE_IMAGE_MIRROR_BATCH_SIZE` 控制

当前 B 站自动解析策略：

- 保存菜谱时如果识别到 B 站链接，会自动标记为 `parseStatus=pending`
- 后端定时任务按 `RECIPE_AUTO_PARSE_INTERVAL_SECONDS` 扫描并解析
- 如果配置了全局 `SESSDATA`，解析器会自动带上登录态请求 B 站字幕接口
- 隐藏设置页的访问权限由 `APP_SETTINGS_ACCESS_MODE` 控制，当前默认值是 `all`
- 成功后会自动补齐 `ingredient`、`parsedContent.ingredients`、`parsedContent.steps`
- 失败后会保留 `parseStatus=failed` 和 `parseError`
- 可通过 `POST /api/recipes/{recipeID}/reparse` 手动重新入队

后台管理平台说明：

- 前端后台工程位于仓库根目录 `admin-web/`
- 默认通过同域 `https://你的域名/admin/` 访问
- 后台登录和小程序登录分离，走独立账号：
  - `ADMIN_USERNAME`
  - `ADMIN_PASSWORD_HASH`
  - `ADMIN_JWT_SECRET`（可选）
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
