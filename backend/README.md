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

当前可用接口：

- `GET /healthz`
- `GET /api/healthz`
- `POST /api/auth/wechat/login`
- `GET /api/auth/me`
- `GET /api/kitchens`
- `POST /api/kitchens`
- `GET /api/kitchens/{kitchenID}/members`
- `GET /api/invites/{token}`
- `POST /api/kitchens/{kitchenID}/invites`
- `POST /api/invites/{token}/accept`
- `GET /api/kitchens/{kitchenID}/recipes`
- `POST /api/kitchens/{kitchenID}/recipes`
- `GET /api/recipes/{recipeID}`
- `PUT /api/recipes/{recipeID}`
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
  -d '{"title":"番茄滑蛋牛肉","ingredient":"牛肉","mealType":"main","status":"wishlist","parsedContent":{"ingredients":["牛肉 200g"],"steps":["下锅翻炒"]}}'

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

后续第一批建议实现顺序：

1. 增加成员移除和角色调整
2. 增加编辑记录和冲突提示
3. 增加邀请撤销和邀请列表
4. 增加操作日志和基础测试
5. 接入正式微信登录域名和线上部署配置
