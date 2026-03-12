# Backend

这个目录是 `caipu-miniapp` 的 Go 后端项目起点。

当前初始化结果：

- 路由框架：`chi`
- 配置加载：环境变量 + `configs/local.env`
- 日志：标准库 `slog`
- 数据库：`SQLite`
- 迁移：启动时自动执行 `migrations/*.sql`
- 健康检查：`GET /healthz`、`GET /api/healthz`
- 已实现首批业务闭环：`auth + kitchens + invite + recipe`

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

当前可用接口：

- `GET /healthz`
- `GET /api/healthz`
- `POST /api/auth/wechat/login`
- `GET /api/auth/me`
- `GET /api/kitchens`
- `POST /api/kitchens`
- `GET /api/invites/{token}`
- `POST /api/kitchens/{kitchenID}/invites`
- `POST /api/invites/{token}/accept`
- `GET /api/kitchens/{kitchenID}/recipes`
- `POST /api/kitchens/{kitchenID}/recipes`
- `GET /api/recipes/{recipeID}`
- `PUT /api/recipes/{recipeID}`
- `PATCH /api/recipes/{recipeID}/status`
- `DELETE /api/recipes/{recipeID}`

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

后续第一批建议实现顺序：

1. `upload`：图片上传
2. 把前端 `recipe-store` 对接到远端 `recipe API`
3. 增加成员管理
4. 增加编辑记录和冲突提示
5. 增加邀请撤销和邀请列表
