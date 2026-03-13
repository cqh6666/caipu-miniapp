# B 站自动解析 POC

这个 POC 面向当前 `caipu-miniapp` 的“添加菜品”场景，目标是让用户在保存菜品时只需要粘贴 B 站链接，后端再通过定时任务异步解析并补齐食材和步骤。

当前实现包含两部分：

- `internal/linkparse`：B 站链接解析、字幕读取、AI / 规则总结
- `internal/recipe/auto_parse_worker.go`：按固定间隔扫描待解析菜谱并回写结果

## 用户侧行为

用户保存菜品时：

1. 前端正常调用 `POST /api/kitchens/{kitchenID}/recipes`
2. 如果链接是 B 站链接，后端会把这条菜谱标记成 `parseStatus=pending`
3. 定时任务会自动扫描 `pending` 菜谱
4. 解析成功后，后端会把 `ingredient`、`parsedContent.ingredients`、`parsedContent.steps` 更新回这条菜谱
5. 解析完成后，状态会变成 `parseStatus=done`

如果解析失败：

- 状态会变成 `parseStatus=failed`
- 错误信息会落在 `parseError`

## 菜谱新增字段

菜谱接口现在会多返回以下字段：

- `parseStatus`
- `parseSource`
- `parseError`
- `parseRequestedAt`
- `parseFinishedAt`

目前状态值主要有：

- `pending`：已入队，等待后台任务处理
- `processing`：正在解析
- `done`：解析完成，已回写菜谱
- `failed`：解析失败

## 定时任务配置

`configs/local.env` 里新增了以下配置：

```env
RECIPE_AUTO_PARSE_ENABLED=true
RECIPE_AUTO_PARSE_INTERVAL_SECONDS=30
RECIPE_AUTO_PARSE_BATCH_SIZE=3

APP_SETTINGS_ACCESS_MODE=all
APP_ADMIN_OPENIDS=
APP_SETTINGS_ALLOWED_OPENIDS=

CREDENTIALS_SECRET=

AI_BASE_URL=https://api.openai.com/v1
AI_API_KEY=
AI_MODEL=
AI_TIMEOUT_SECONDS=30
```

说明：

- `RECIPE_AUTO_PARSE_ENABLED=false` 时，后端不会启动自动解析 worker
- `RECIPE_AUTO_PARSE_INTERVAL_SECONDS` 控制轮询间隔
- `RECIPE_AUTO_PARSE_BATCH_SIZE` 控制每轮最多处理多少条菜谱
- `APP_SETTINGS_ACCESS_MODE` 控制隐藏设置页的可见范围，支持 `all`、`admin`、`whitelist`
- `APP_ADMIN_OPENIDS` 用来声明管理员账号
- `APP_SETTINGS_ALLOWED_OPENIDS` 在 `whitelist` 模式下额外放行指定用户
- `AI_MODEL` 为空时，解析仍然会执行，但只走规则总结
- `CREDENTIALS_SECRET` 用来加密存储全局 `SESSDATA`；为空时会回退到 `JWT_SECRET`

## 全局 B 站登录态

当前版本支持在后端维护一份全局 `SESSDATA`，解析器会在请求 B 站字幕接口时自动带上：

- `GET /api/app-settings/bilibili-session`
- `PUT /api/app-settings/bilibili-session`
- `DELETE /api/app-settings/bilibili-session`

保存时支持两种输入：

- 直接填写 `SESSDATA`
- 粘贴整段 Cookie，后端自动提取 `SESSDATA`

只有保存成功后，新的解析任务才会使用这份登录态。

访问控制说明：

- `all`：所有登录用户可见
- `admin`：只有管理员可见
- `whitelist`：管理员和白名单用户可见
- 管理员 / 白名单都按用户 `openid` 配置；开发登录时通常是 `dev:<identity>`

## 后台处理流程

1. 定时任务读取 `parse_status = 'pending'` 的菜谱
2. 把单条菜谱抢占成 `processing`
3. 解析 B 站链接，读取视频信息和字幕
4. 如果配置了模型，就用 AI 提炼食材和步骤
5. 如果没有字幕或模型不可用，就回退到规则总结
6. 成功时写回食材、步骤和解析状态
7. 失败时记录错误信息，状态改成 `failed`

## 启动方式

```bash
cd backend
cp configs/example.env configs/local.env
APP_ENV_FILE=configs/local.env go run ./cmd/server
```

如果只想先验证异步链路，不填 AI 配置也可以。

## 联调示例

先拿开发态 token：

```bash
cd backend
token=$(curl -s -X POST http://127.0.0.1:8080/api/auth/dev-login \
  -H 'Content-Type: application/json' \
  -d '{"identity":"alice"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
```

创建一条带 B 站链接的菜谱：

```bash
curl -s -X POST http://127.0.0.1:8080/api/kitchens/1/recipes \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "B站自动解析测试",
    "ingredient": "",
    "link": "https://www.bilibili.com/video/BV1aWCEYHErc",
    "mealType": "main",
    "status": "wishlist",
    "parsedContent": {
      "ingredients": ["B站自动解析测试 1份", "正餐常用配菜 适量", "基础调味 适量"],
      "steps": ["先整理这道菜的核心做法。", "按自己的口味调整成容易复刻的版本。", "做完以后补充口感和火候记录。"]
    }
  }'
```

刚创建时，返回里会出现：

```json
{
  "parseStatus": "pending",
  "parseSource": "bilibili"
}
```

过几秒再查详情：

```bash
curl -s http://127.0.0.1:8080/api/recipes/<recipeID> \
  -H "Authorization: Bearer $token"
```

成功后会看到类似结果：

```json
{
  "parseStatus": "done",
  "parseSource": "bilibili:heuristic",
  "ingredient": "牛腩、西红柿、土豆",
  "parsedContent": {
    "ingredients": ["牛腩 500克", "西红柿 500克", "土豆 250克"],
    "steps": ["牛腩焯水备用", "番茄炒软后和牛腩一起炖煮"]
  }
}
```

## 调试接口

保留了一个手动调试接口：

- `POST /api/link-parsers/bilibili`
- `POST /api/recipes/{recipeID}/reparse`
- `GET /api/app-settings/bilibili-session`
- `PUT /api/app-settings/bilibili-session`
- `DELETE /api/app-settings/bilibili-session`

其中：

- `POST /api/link-parsers/bilibili` 主要用于开发调试，单独验证“某个链接能不能被解析”
- `POST /api/recipes/{recipeID}/reparse` 用于把失败或需要重试的菜谱重新加入后台解析队列

## 当前限制

- 目前只支持 B 站链接
- 当前 worker 只扫描 `pending`，失败后不会自动无限重试
- 只读取“平台已经提供的字幕”，还没有接音频转写
- 一些视频字幕需要登录态才能拿到，当前通过全局 `SESSDATA` 提升命中率
- 如果用户自己手动填写了明确的食材和步骤，后端不会强制覆盖

## 建议的下一步

1. 前端列表或详情页显示 `parseStatus`
2. 对 `failed` 状态提供“重新解析”入口
3. 接入音频转写，覆盖无字幕视频
4. 解析成功后增加解析时间和来源提示
5. 沿用同一套任务模型扩展到小红书链接
