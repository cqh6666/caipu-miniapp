# Linkparse Sidecar 重构迁移说明

这份文档面向已经在线上部署过旧 sidecar 方案的环境，目标是把“小红书专用 sidecar”升级为现在的通用解析 sidecar，并给后续接入 B 站解析能力预留清晰边界。

适用场景：

- 你已经在云服务器上跑过 `xhs-sidecar` 或同类命名的服务
- 你准备把服务目录、systemd、环境变量统一收口到 `linkparse-sidecar`
- 你希望后面继续把 B 站解析也逐步迁到 sidecar

## 1. 这次重构改了什么

核心变化只有两层：

1. 对外命名从“小红书专用 sidecar”改成“通用链接解析 sidecar”
2. backend 和 sidecar 的交互改成统一的 `linkparse sidecar` 协议

当前实现落点：

- sidecar 目录从 `sidecars/xhs-sidecar` 改为 `sidecars/linkparse-sidecar`
- backend 连接 sidecar 的配置统一为 `LINKPARSE_SIDECAR_*`
- sidecar 现在同时暴露：
  - `POST /v1/parse/xiaohongshu`
  - `POST /v1/parse/bilibili`
- backend 开启 sidecar 后：
  - 小红书解析走 sidecar
  - B 站预览和解析也优先走 sidecar
- sidecar 内部仍然保留平台级环境变量：
  - 小红书相关还是 `XHS_*`
  - B 站相关是 `BILI_*`

这意味着：

- “跨服务边界”的命名已经泛化为 `linkparse`
- “平台内部实现”的命名仍然按平台拆开保留

这样后续继续扩展 B 站、抖音、快手时，backend 不需要再次改一轮总线命名。

## 2. 旧名和新名对照

| 旧概念 | 新概念 | 说明 |
| --- | --- | --- |
| `xhs-sidecar` | `linkparse-sidecar` | sidecar 服务整体改名 |
| `sidecars/xhs-sidecar` | `sidecars/linkparse-sidecar` | 仓库目录改名 |
| `caipu-xhs-sidecar` 或自定义旧服务名 | `caipu-linkparse-sidecar` | 建议统一成新 service 名；如果你线上不是这个名字，按实际替换 |
| `runtime/xhs-sidecar/...` | `runtime/linkparse-sidecar/...` | 推荐同步改运行时目录，便于长期维护 |
| `XHS_SIDECAR_*` | `LINKPARSE_SIDECAR_*` | backend 连接 sidecar 的环境变量统一改名 |

注意：

- 当前 backend 代码只认 `LINKPARSE_SIDECAR_*`，不再兼容旧的 `XHS_SIDECAR_*` 命名。
- `XHS_PROVIDER_*`、`XHS_REDNOTE_*`、`XHS_TRANSCRIPT_*` 这些 sidecar 内部变量没有改名，因为它们描述的是“小红书 provider 内部行为”。

## 3. backend 和 sidecar 现在如何交互

### 小红书

链路：

`backend -> POST /v1/parse/xiaohongshu -> linkparse-sidecar -> xiaohongshu provider`

说明：

- sidecar 负责抓取、归一化、可选的视频转字幕
- backend 负责后续 AI 总结、规则整理、生成菜谱草稿

### B 站

链路：

`backend -> POST /v1/parse/bilibili -> linkparse-sidecar -> bilibili provider`

说明：

- backend 会把当前拿到的 `SESSDATA` 通过请求头 `X-Bilibili-SESSDATA` 透传给 sidecar
- sidecar 再去请求 B 站视频信息和字幕接口
- 如果 `LINKPARSE_SIDECAR_ENABLED=false`，backend 仍然保留 B 站本地解析 fallback

注意：

- 小红书目前没有 backend 本地 fallback，sidecar 关掉后小红书解析也会一起失效。

## 4. 云服务器迁移步骤

以下命令假设你的部署目录是：

- 项目根目录：`/srv/caipu-miniapp`
- backend：`/srv/caipu-miniapp/backend`
- sidecar：`/srv/caipu-miniapp/sidecars/linkparse-sidecar`

### 4.1 拉最新代码

```bash
cd /srv/caipu-miniapp
git pull origin main
```

如果你线上不是直接 `git pull`，而是走打包/同步发布，也要确认最终目录名已经变成：

```bash
/srv/caipu-miniapp/sidecars/linkparse-sidecar
```

### 4.2 安装 sidecar 依赖

```bash
cd /srv/caipu-miniapp/sidecars/linkparse-sidecar
npm install
npx playwright install chromium
```

如果缺系统依赖：

```bash
cd /srv/caipu-miniapp/sidecars/linkparse-sidecar
npx playwright install-deps chromium
```

### 4.3 调整运行时目录

推荐统一成新目录：

```bash
mkdir -p /srv/caipu-miniapp/runtime/linkparse-sidecar
mkdir -p /srv/caipu-miniapp/runtime/rednote
```

如果你以前把 sidecar 环境文件放在旧目录，比如：

```bash
/srv/caipu-miniapp/runtime/xhs-sidecar/xhs-sidecar.env
```

可以先复制成新文件，再按下文改内容：

```bash
cp /srv/caipu-miniapp/runtime/xhs-sidecar/xhs-sidecar.env \
  /srv/caipu-miniapp/runtime/linkparse-sidecar/linkparse-sidecar.env
```

如果旧文件路径和名字跟这里不同，按你的实际路径处理即可。

### 4.4 更新 sidecar 环境文件

建议统一用：

- `/srv/caipu-miniapp/runtime/linkparse-sidecar/linkparse-sidecar.env`

示例：

```env
PORT=8091
LINKPARSE_INTERNAL_API_KEY=replace-with-random-secret

XHS_PROVIDER_DEFAULT=auto
XHS_PROVIDER_IMPORTER_ENABLED=true
XHS_PROVIDER_REDNOTE_ENABLED=true
XHS_SIDECAR_STUB_MODE=off

XHS_REDNOTE_COOKIE_PATH=/srv/caipu-miniapp/runtime/rednote/cookies.json
XHS_REDNOTE_COOKIE_HEADER=
XHS_REDNOTE_COOKIE_DOMAIN=.xiaohongshu.com
XHS_BROWSER_HEADLESS=true
XHS_REDNOTE_BROWSER_PATH=
XHS_REDNOTE_LOGIN_URL=https://www.xiaohongshu.com/
XHS_REDNOTE_TIMEOUT_MS=15000

XHS_TRANSCRIPT_ENABLED=false
XHS_TRANSCRIPT_PROVIDER=siliconflow
XHS_TRANSCRIPT_API_KEY=
XHS_TRANSCRIPT_MODEL=TeleAI/TeleSpeechASR
XHS_TRANSCRIPT_ENDPOINT=https://api.siliconflow.cn/v1/audio/transcriptions
XHS_TRANSCRIPT_TIMEOUT_MS=120000
XHS_TRANSCRIPT_MAX_VIDEO_MB=80
XHS_TRANSCRIPT_KEEP_TEMP=false
FFMPEG_PATH=ffmpeg

BILI_PROVIDER_DEFAULT=auto
BILI_PROVIDER_OPENAPI_ENABLED=true
```

这里要点只有三个：

- sidecar 对外共享密钥现在统一用 `LINKPARSE_INTERNAL_API_KEY`
- 小红书 provider 相关变量仍然保留 `XHS_*`
- B 站 provider 暂时只需要 `BILI_PROVIDER_*` 开关，不需要单独维护 sidecar 级 Cookie 文件

### 4.5 更新 backend 生产环境变量

编辑 backend 生产配置，比如：

- `/srv/caipu-miniapp/backend/configs/prod.env`

至少确保有：

```env
LINKPARSE_SIDECAR_ENABLED=true
LINKPARSE_SIDECAR_BASE_URL=http://127.0.0.1:8091
LINKPARSE_SIDECAR_TIMEOUT_SECONDS=25
LINKPARSE_SIDECAR_API_KEY=replace-with-random-secret
```

要点：

- `LINKPARSE_SIDECAR_API_KEY` 必须和 sidecar 的 `LINKPARSE_INTERNAL_API_KEY` 一致
- 如果你线上仍然保留旧的 `XHS_SIDECAR_*` 自定义变量名，需要手工替换成 `LINKPARSE_SIDECAR_*`
- backend 开启 sidecar 后，B 站会优先走 sidecar；关闭后仍保留本地 fallback

### 4.6 更新 systemd 服务

建议新服务文件：

- `/etc/systemd/system/caipu-linkparse-sidecar.service`

示例：

```ini
[Unit]
Description=Caipu Linkparse Sidecar
After=network.target

[Service]
Type=simple
WorkingDirectory=/srv/caipu-miniapp/sidecars/linkparse-sidecar
EnvironmentFile=/srv/caipu-miniapp/runtime/linkparse-sidecar/linkparse-sidecar.env
ExecStart=/usr/bin/npm start
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
```

如果你线上还保留旧服务名，推荐这样迁移：

```bash
systemctl disable --now caipu-xhs-sidecar || true
systemctl daemon-reload
systemctl enable caipu-linkparse-sidecar
systemctl restart caipu-linkparse-sidecar
systemctl status caipu-linkparse-sidecar --no-pager
```

如果你的旧服务名不是 `caipu-xhs-sidecar`，把命令里的服务名替换成你自己的实际值。

### 4.7 重启 backend

```bash
cd /srv/caipu-miniapp/backend
go build -o bin/server ./cmd/server
systemctl restart caipu-backend
systemctl status caipu-backend --no-pager
```

## 5. 验证清单

假设你启用了 API Key：

```bash
export LINKPARSE_API_KEY=replace-with-random-secret
```

### 5.1 看 sidecar 健康状态

```bash
curl -s http://127.0.0.1:8091/v1/health \
  -H "Authorization: Bearer $LINKPARSE_API_KEY"
```

至少确认：

- `service=linkparse-sidecar`
- `platforms.xiaohongshu` 存在
- `platforms.bilibili` 存在

### 5.2 看 provider 列表

```bash
curl -s http://127.0.0.1:8091/v1/providers \
  -H "Authorization: Bearer $LINKPARSE_API_KEY"
```

至少确认：

- 小红书 provider 列表里有 `importer`、`rednote`
- B 站 provider 列表里有 `openapi`

### 5.3 验证小红书解析

```bash
curl -s -X POST http://127.0.0.1:8091/v1/parse/xiaohongshu \
  -H "Authorization: Bearer $LINKPARSE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "https://www.xiaohongshu.com/explore/68abcd1234",
    "provider": "auto",
    "includeTranscript": false
  }'
```

### 5.4 验证 B 站解析

如果 backend 已经存有 `SESSDATA`，实际生产流量会自动透传给 sidecar。你单独测 sidecar 时，也可以手工带上：

```bash
curl -s -X POST http://127.0.0.1:8091/v1/parse/bilibili \
  -H "Authorization: Bearer $LINKPARSE_API_KEY" \
  -H "Content-Type: application/json" \
  -H "X-Bilibili-SESSDATA: your-sessdata" \
  -d '{
    "input": "https://www.bilibili.com/video/BV1aWCEYHErc",
    "provider": "auto",
    "includeTranscript": true
  }'
```

### 5.5 看 systemd 日志

```bash
journalctl -u caipu-linkparse-sidecar -n 100 --no-pager
journalctl -u caipu-backend -n 100 --no-pager
```

## 6. 常见调整点

最容易漏掉的是下面几类：

- systemd 里的 `WorkingDirectory` 还是旧目录
- `EnvironmentFile` 还是旧的 `xhs-sidecar.env`
- backend 生产环境里还残留旧的 `XHS_SIDECAR_*`
- 只改了 sidecar API Key，没有同步改 backend 的 `LINKPARSE_SIDECAR_API_KEY`
- B 站 provider 已启用，但线上没有可用的 `SESSDATA`，导致拿不到字幕

补充说明：

- sidecar 本身不负责持久化 B 站登录态；它只是消费 backend 透传过来的 `X-Bilibili-SESSDATA`
- 小红书登录态仍由 sidecar 自己维护，来源可以是 `cookies.json` 或整段 `Cookie` 字符串

## 7. 回滚策略

如果新 sidecar 上线后要先止损，最稳妥的回滚顺序是：

1. 把 backend 的 `LINKPARSE_SIDECAR_ENABLED=false`
2. 重启 backend
3. 视情况停止 `caipu-linkparse-sidecar`

回滚效果：

- B 站会回退到 backend 现有的本地解析逻辑
- 小红书 sidecar 解析会停止

如果你要完整回到旧部署结构，再继续做这些事：

1. 恢复旧的 systemd 服务文件和运行时目录命名
2. 切回重构前的代码版本
3. 再次执行 `systemctl daemon-reload` 和服务重启

## 8. 相关文档

- [Linkparse Sidecar 云服务器部署清单](./xiaohongshu-cloud-deploy.md)
- [小红书接入说明](./xiaohongshu-integration-guide.md)
- [Sidecar API 设计说明](./xiaohongshu-sidecar-api-plan.md)
