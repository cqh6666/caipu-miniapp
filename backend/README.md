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

技术评估结论：

- 当前阶段前后端放在一个仓库、后端独立放到 `backend/` 目录最合适
- `Go + chi + SQLite` 足够覆盖共享厨房第一版需求
- `SQLite` 采用 `WAL` 和 `busy_timeout`，并将 `MaxOpenConns` 设为 `1`，优先保证稳定性
- 先把配置、迁移、路由、健康检查和数据库底座跑通，再逐步加登录、厨房、邀请、菜谱模块

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
- `AI_BASE_URL`
- `AI_API_KEY`
- `AI_MODEL`
- `AI_TIMEOUT_SECONDS`
- `AI_FLOWCHART_BASE_URL`
- `AI_FLOWCHART_API_KEY`
- `AI_FLOWCHART_MODEL`
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
- `LINKPARSE_SIDECAR_ENABLED`
- `LINKPARSE_SIDECAR_BASE_URL`
- `LINKPARSE_SIDECAR_TIMEOUT_SECONDS`
- `LINKPARSE_SIDECAR_API_KEY`

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

这会确保本地数据库里存在两个开发用户 `alice` / `recipe-user`，并生成一个共享厨房 `联调试吃厨房` 以及多条早餐、正餐、想吃、吃过样例数据，方便直接联调前端。

B 站自动解析 POC 说明见：[docs/bilibili-link-parser-poc.md](./docs/bilibili-link-parser-poc.md)

小红书接入评估与 sidecar 方案见：

- [docs/xiaohongshu-link-parser-feasibility.md](./docs/xiaohongshu-link-parser-feasibility.md)
- [docs/xiaohongshu-sidecar-api-plan.md](./docs/xiaohongshu-sidecar-api-plan.md)
- [docs/xiaohongshu-integration-guide.md](./docs/xiaohongshu-integration-guide.md)
- [docs/xiaohongshu-cloud-deploy.md](./docs/xiaohongshu-cloud-deploy.md)
- [docs/linkparse-sidecar-refactor-migration.md](./docs/linkparse-sidecar-refactor-migration.md)

当前可用接口：

- `GET /healthz`
- `GET /api/healthz`
- `POST /api/auth/wechat/login`
- `GET /api/auth/me`
- `GET /api/app-settings/bilibili-session`
- `PUT /api/app-settings/bilibili-session`
- `DELETE /api/app-settings/bilibili-session`
- `GET /api/kitchens`
- `POST /api/kitchens`
- `GET /api/kitchens/{kitchenID}/members`
- `GET /api/invites/{token}`
- `POST /api/kitchens/{kitchenID}/invites`
- `POST /api/invites/{token}/accept`
- `POST /api/link-parsers/bilibili`
- `POST /api/link-parsers/xiaohongshu`
- `POST /api/link-parsers/preview`
- `GET /api/kitchens/{kitchenID}/recipes`
- `POST /api/kitchens/{kitchenID}/recipes`
- `GET /api/recipes/{recipeID}`
- `PUT /api/recipes/{recipeID}`
- `POST /api/recipes/{recipeID}/reparse`
- `POST /api/recipes/{recipeID}/flowchart`
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

- 厨房成员可以创建邀请链接
- 默认 `72` 小时过期，默认最多使用 `10` 次
- `GET /api/invites/{token}` 可在登录前预览邀请
- 同一用户重复接受同一厨房邀请时会幂等返回 `alreadyMember=true`

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

当前步骤图生成策略：

- 用户仍可通过 `POST /api/recipes/{recipeID}/flowchart` 手动入队生成步骤图
- `RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED=true` 时，后端会在当前没有 `pending / processing` 步骤图任务时，自动补位 1 条候选菜谱入队
- 自动补位只挑选“还没生成步骤图、做法已完整、当前不在自动解析中”的菜谱
- 第一版自动补位不会自动重试 `failed`，也不会自动重生成已有但过期的步骤图
- 步骤图队列状态和生成结果会更新 `flowchart_*` 字段，但不会改 `recipe.updated_at`，避免首页列表被后台任务打乱顺序

当前小红书预留策略：

- 保存菜谱时如果识别到小红书链接，也会自动标记为 `parseStatus=pending`
- 后端 worker 会按平台路由到小红书 sidecar
- 是否真正启用，由 `LINKPARSE_SIDECAR_ENABLED` 控制
- 当前仓库已包含通用解析 sidecar、provider 路由和 RedNote 初始化脚本，目录在 [linkparse-sidecar](../sidecars/linkparse-sidecar)
- 小红书 provider 默认策略由 sidecar 自身的 `XHS_PROVIDER_DEFAULT` 控制，推荐保持 `auto`
- 更完整的使用说明见 [xiaohongshu-integration-guide.md](./docs/xiaohongshu-integration-guide.md)

后续第一批建议实现顺序：

1. 增加成员移除和角色调整
2. 增加编辑记录和冲突提示
3. 增加邀请撤销和邀请列表
4. 增加操作日志和基础测试
5. 接入正式微信登录域名和线上部署配置
