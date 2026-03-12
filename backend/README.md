# Backend

这个目录是 `caipu-miniapp` 的 Go 后端项目起点。

当前初始化结果：

- 路由框架：`chi`
- 配置加载：环境变量 + `configs/local.env`
- 日志：标准库 `slog`
- 数据库：`SQLite`
- 迁移：启动时自动执行 `migrations/*.sql`
- 健康检查：`GET /healthz`、`GET /api/healthz`

技术评估结论：

- 当前阶段前后端放在一个仓库、后端独立放到 `backend/` 目录最合适
- `Go + chi + SQLite` 足够覆盖共享厨房第一版需求
- `SQLite` 采用 `WAL` 和 `busy_timeout`，并将 `MaxOpenConns` 设为 `1`，优先保证稳定性
- 先把配置、迁移、路由、健康检查和数据库底座跑通，再加登录、厨房、邀请、菜谱模块

本地启动：

```bash
cd backend
cp configs/example.env configs/local.env
go run ./cmd/server
```

只执行迁移：

```bash
cd backend
go run ./cmd/server -migrate-only
```

后续第一批建议实现顺序：

1. `auth`：微信登录和 token
2. `kitchen`：厨房列表和创建
3. `invite`：邀请生成、预览、加入
4. `recipe`：菜谱 CRUD 和状态切换
5. `upload`：图片上传
