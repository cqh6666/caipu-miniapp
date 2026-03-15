# 小红书 Sidecar 云服务器部署清单

这份文档面向 Linux 云服务器，目标是把当前仓库里的小红书 sidecar 部署起来，并接到现有 Go 后端。

假设：

- 项目部署目录：`/srv/caipu-miniapp`
- 后端目录：`/srv/caipu-miniapp/backend`
- sidecar 目录：`/srv/caipu-miniapp/sidecars/xhs-sidecar`
- 后端服务名：`caipu-backend`
- sidecar 服务名：`caipu-xhs-sidecar`
- 当前服务器已安装 `git`、`node`、`npm`

## 1. 拉取最新代码

```bash
cd /srv/caipu-miniapp
git pull origin main
```

## 2. 安装 sidecar 依赖

```bash
cd /srv/caipu-miniapp/sidecars/xhs-sidecar
npm install
```

## 3. 安装 Playwright Chromium

```bash
cd /srv/caipu-miniapp/sidecars/xhs-sidecar
npx playwright install chromium
```

如果服务器缺少系统依赖，可以先执行：

```bash
cd /srv/caipu-miniapp/sidecars/xhs-sidecar
npx playwright install-deps chromium
```

## 4. 准备 sidecar 运行目录

```bash
mkdir -p /srv/caipu-miniapp/runtime/rednote
mkdir -p /srv/caipu-miniapp/runtime/xhs-sidecar
```

说明：

- `runtime/rednote` 用来放 Cookie 文件
- `runtime/xhs-sidecar` 用来放环境文件

## 5. 创建 sidecar 环境文件

创建：

- `/srv/caipu-miniapp/runtime/xhs-sidecar/xhs-sidecar.env`

内容示例：

```env
PORT=8091
XHS_PROVIDER_DEFAULT=auto
XHS_PROVIDER_IMPORTER_ENABLED=true
XHS_PROVIDER_REDNOTE_ENABLED=true
XHS_SIDECAR_STUB_MODE=echo
XHS_INTERNAL_API_KEY=replace-with-random-secret
XHS_REDNOTE_COOKIE_PATH=/srv/caipu-miniapp/runtime/rednote/cookies.json
XHS_BROWSER_HEADLESS=true
XHS_REDNOTE_BROWSER_PATH=
XHS_REDNOTE_LOGIN_URL=https://www.xiaohongshu.com/
XHS_REDNOTE_TIMEOUT_MS=15000
```

推荐先用 `echo` 模式启动，等链路完全跑通后，再考虑切成：

```env
XHS_SIDECAR_STUB_MODE=off
```

## 6. 初始化 RedNote 登录态

这一步需要交互式浏览器，不建议在已经启动 systemd 服务后直接做。推荐先手动执行一次：

```bash
cd /srv/caipu-miniapp/sidecars/xhs-sidecar
export XHS_REDNOTE_COOKIE_PATH=/srv/caipu-miniapp/runtime/rednote/cookies.json
export XHS_REDNOTE_LOGIN_URL=https://www.xiaohongshu.com/
npm run rednote:init
```

执行后：

- 会打开浏览器
- 你手动登录小红书
- 按回车后 Cookie 会保存到 `/srv/caipu-miniapp/runtime/rednote/cookies.json`

然后检查状态：

```bash
cd /srv/caipu-miniapp/sidecars/xhs-sidecar
export XHS_REDNOTE_COOKIE_PATH=/srv/caipu-miniapp/runtime/rednote/cookies.json
npm run rednote:status
```

理想状态至少要看到：

- `playwrightAvailable=true`
- `browserInstalled=true`
- `loggedIn=true`

## 7. 创建 sidecar systemd 服务

创建文件：

- `/etc/systemd/system/caipu-xhs-sidecar.service`

内容示例：

```ini
[Unit]
Description=Caipu Xiaohongshu Sidecar
After=network.target

[Service]
Type=simple
WorkingDirectory=/srv/caipu-miniapp/sidecars/xhs-sidecar
EnvironmentFile=/srv/caipu-miniapp/runtime/xhs-sidecar/xhs-sidecar.env
ExecStart=/usr/bin/npm start
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
```

然后执行：

```bash
systemctl daemon-reload
systemctl enable caipu-xhs-sidecar
systemctl restart caipu-xhs-sidecar
systemctl status caipu-xhs-sidecar --no-pager
```

查看日志：

```bash
journalctl -u caipu-xhs-sidecar -n 100 --no-pager
```

## 8. 验证 sidecar 服务

```bash
curl -s http://127.0.0.1:8091/v1/health
curl -s http://127.0.0.1:8091/v1/providers
curl -s http://127.0.0.1:8091/v1/auth/rednote/status
```

如果你配置了 `XHS_INTERNAL_API_KEY`，要改成带请求头：

```bash
curl -s http://127.0.0.1:8091/v1/health \
  -H 'Authorization: Bearer replace-with-random-secret'
```

## 9. 配置后端接入 sidecar

编辑生产环境配置：

- `/srv/caipu-miniapp/backend/configs/prod.env`

至少补上：

```env
XHS_SIDECAR_ENABLED=true
XHS_SIDECAR_BASE_URL=http://127.0.0.1:8091
XHS_SIDECAR_TIMEOUT_SECONDS=25
XHS_SIDECAR_PROVIDER=auto
XHS_SIDECAR_API_KEY=replace-with-random-secret
```

说明：

- `XHS_SIDECAR_API_KEY` 要和 sidecar 的 `XHS_INTERNAL_API_KEY` 一致
- 如果 sidecar 没开 API Key，这里可以留空

## 10. 重新编译并重启后端

```bash
cd /srv/caipu-miniapp/backend
go build -o bin/server ./cmd/server
systemctl restart caipu-backend
systemctl status caipu-backend --no-pager
```

查看日志：

```bash
journalctl -u caipu-backend -n 100 --no-pager
```

## 11. 端到端验证

### 验证 sidecar 解析

```bash
curl -s -X POST http://127.0.0.1:8091/v1/parse/xiaohongshu \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer replace-with-random-secret' \
  -d '{
    "input": "【番茄牛腩】今天炖了一锅超下饭的番茄牛腩，牛腩500克。 https://www.xiaohongshu.com/explore/68b6e4f3000000001f03379f",
    "provider": "auto",
    "includeDebug": true
  }'
```

### 验证后端异步解析

1. 在小程序里新建一条带小红书链接的菜谱
2. 确认它先进入 `parseStatus=pending`
3. 等待 worker 处理
4. 查看详情页中的解析状态

服务端可观察日志：

```bash
journalctl -u caipu-backend -f --no-pager
```

如果 sidecar 已真正接通，常见结果会是：

- `parseSource = xiaohongshu:ai`
- 或 `parseSource = xiaohongshu:heuristic`

## 12. 常见问题

### `playwrightAvailable=true` 但 `browserInstalled=false`

执行：

```bash
cd /srv/caipu-miniapp/sidecars/xhs-sidecar
npx playwright install chromium
```

### `rednote` 返回 `login_required`

说明 Cookie 文件不存在、为空，或者初始化登录态没有完成。重新执行：

```bash
cd /srv/caipu-miniapp/sidecars/xhs-sidecar
export XHS_REDNOTE_COOKIE_PATH=/srv/caipu-miniapp/runtime/rednote/cookies.json
npm run rednote:init
```

### 只想先跑轻量模式

可以暂时关闭 `rednote`：

```env
XHS_PROVIDER_REDNOTE_ENABLED=false
XHS_PROVIDER_DEFAULT=importer
```

### 只想先用 sidecar 回退分享文本

保持：

```env
XHS_SIDECAR_STUB_MODE=echo
```

这样即使匿名抓取失败，只要用户提交的是“分享文本 + 链接”，仍然能回退到可见文字做总结。

## 13. 推荐上线策略

第一阶段建议：

- `XHS_PROVIDER_DEFAULT=auto`
- `XHS_PROVIDER_IMPORTER_ENABLED=true`
- `XHS_PROVIDER_REDNOTE_ENABLED=true`
- `XHS_SIDECAR_STUB_MODE=echo`

等这条链路稳定后，再逐步收紧到：

- `XHS_SIDECAR_STUB_MODE=off`
- 更严格的 API Key
- 更稳定的 Cookie 更新流程
