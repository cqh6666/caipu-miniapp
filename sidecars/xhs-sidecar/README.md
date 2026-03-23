# xhs-sidecar

这是给 `caipu-miniapp` 准备的小红书 sidecar stub。

当前目标不是直接完成真实小红书抓取，而是先把下面这几件事定稳：

- sidecar API 形状
- provider 路由
- `importer / rednote / auto` 三种策略切换
- 视频笔记 `ffmpeg + ASR` 转字幕
- Go 主服务和 sidecar 的联调方式

## 当前状态

当前版本是 **可联调的轻量版**：

- 已实现统一 API
- 已预留两种 provider
- `importer` 已接入一版轻量 HTML 提取逻辑
- `rednote` 已接入一版基于 Playwright + Cookie 文件的真实浏览器 provider

换句话说：

- 它现在已经可以联调 importer 路线
- 也已经可以接 RedNote 登录态路线
- 但整体还不适合直接当生产级小红书解析服务

RedNote 登录态现在支持两种来源：

- `XHS_REDNOTE_COOKIE_PATH`：`cookies.json` 文件
- `XHS_REDNOTE_COOKIE_HEADER`：浏览器里复制出来的整段 `Cookie:` 字符串

## 支持的接口

- `GET /v1/health`
- `GET /v1/providers`
- `GET /v1/auth/rednote/status`
- `POST /v1/parse/xiaohongshu`

另外还提供两个本地辅助命令：

- `npm run rednote:init`
- `npm run rednote:status`

## 本地启动

```bash
cd /Users/alexh/github_proj/caipu-miniapp/sidecars/xhs-sidecar
npm install
npm start
```

如果你要启用视频转字幕，还需要系统里可用的 `ffmpeg`：

```bash
ffmpeg -version
```

如果命令不存在，请先安装 `ffmpeg`，或者通过 `FFMPEG_PATH` 指定可执行文件路径。

默认监听：

- `http://127.0.0.1:8091`

## 和主服务联调

后端需要同时配置：

```env
XHS_SIDECAR_ENABLED=true
XHS_SIDECAR_BASE_URL=http://127.0.0.1:8091
XHS_SIDECAR_TIMEOUT_SECONDS=150
XHS_SIDECAR_PROVIDER=auto
XHS_SIDECAR_API_KEY=
```

如果 sidecar 配了 `XHS_INTERNAL_API_KEY`，记得把同一个值写到后端的 `XHS_SIDECAR_API_KEY`。

主服务启用后，保存带小红书链接的菜谱就会自动进入异步解析队列。

## 推荐环境变量

```env
PORT=8091
XHS_PROVIDER_DEFAULT=auto
XHS_PROVIDER_IMPORTER_ENABLED=true
XHS_PROVIDER_REDNOTE_ENABLED=true
XHS_SIDECAR_STUB_MODE=echo
XHS_INTERNAL_API_KEY=
XHS_REDNOTE_COOKIE_PATH=
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
```

说明：

- `XHS_PROVIDER_DEFAULT`: `auto | importer | rednote`
- `XHS_REDNOTE_COOKIE_HEADER`: 支持直接粘贴整段浏览器 Cookie，适合像 B 站那样维护
- `XHS_REDNOTE_COOKIE_DOMAIN`: 把整段 Cookie 转成 Playwright Cookie 时使用的默认域名
- `XHS_SIDECAR_STUB_MODE`:
  - `off`: importer 只做真实轻量抓取，不做文本回退
  - `echo`: 先做真实轻量抓取，失败后回退到“分享文本里 URL 之外的文字”
  - `demo`: 返回固定演示数据
- `XHS_REDNOTE_COOKIE_PATH`: RedNote / 小红书登录态 cookie 文件路径，默认是 `~/.mcp/rednote/cookies.json`
- 如果 `XHS_REDNOTE_COOKIE_HEADER` 和 `XHS_REDNOTE_COOKIE_PATH` 同时配置，优先使用 `XHS_REDNOTE_COOKIE_HEADER`
- `XHS_BROWSER_HEADLESS`: RedNote provider 是否无头运行
- `XHS_REDNOTE_BROWSER_PATH`: 可选，自定义浏览器可执行文件路径
- `XHS_REDNOTE_LOGIN_URL`: 初始化登录时打开的页面
- `XHS_REDNOTE_TIMEOUT_MS`: RedNote provider 页面等待超时
- `XHS_TRANSCRIPT_ENABLED`: 是否启用小红书视频转字幕
- `XHS_TRANSCRIPT_PROVIDER`: 默认是 `siliconflow`，当前已兼容 `siliconflow | infiniteai`
- `XHS_TRANSCRIPT_API_KEY`: ASR 服务 API Key
- `XHS_TRANSCRIPT_MODEL`: 硅基流动转写模型，默认 `TeleAI/TeleSpeechASR`
- `XHS_TRANSCRIPT_ENDPOINT`: 可选，自定义转写接口地址
- `XHS_TRANSCRIPT_TIMEOUT_MS`: 整条转字幕链路的总超时预算，不再按“下载 / ffmpeg / ASR”分别重复计时
- `XHS_TRANSCRIPT_MAX_VIDEO_MB`: 视频体积保护阈值，超过后直接跳过转写并返回失败状态
- `XHS_TRANSCRIPT_KEEP_TEMP`: 调试时保留临时 `mp4/mp3`
- `FFMPEG_PATH`: `ffmpeg` 可执行文件路径
- `/v1/auth/rednote/status` 现在会区分：
  - `cookieSource`: `header | file`
  - `playwrightAvailable`: Node 包是否可用
  - `browserInstalled`: Chromium 二进制是否已准备好
  - `ready`: Cookie、Playwright、浏览器三者都已就绪

## 解析请求示例

```bash
curl -s -X POST http://127.0.0.1:8091/v1/parse/xiaohongshu \
  -H 'Content-Type: application/json' \
  -d '{
    "input": "【番茄牛腩】今天炖了一锅超下饭的番茄牛腩，牛腩 500克，番茄 3个。 https://www.xiaohongshu.com/explore/68abcd1234",
    "provider": "auto",
    "includeTranscript": true,
    "includeDebug": true
  }'
```

只有当请求里显式传 `includeTranscript: true` 且 `XHS_TRANSCRIPT_ENABLED=true` 时，返回里的 `note` 才会额外包含：

- `transcript`
- `transcriptStatus`
- `transcriptError`

设计约束是：即使下载视频、抽音频或 ASR 失败，也不影响原始 note 解析成功返回，只会在这些字段里反映状态。

## 下一步接入建议

### 接 `xiaohongshu-importer`

当前已经按它的核心思路接了一版：

1. URL 归一化
2. 拉 HTML
3. 读取 `window.__INITIAL_STATE__`
4. 提取 `title / content / tags / images / videos`

后面如果要继续加强，重点是：

- 处理更多小红书守卫页场景
- 补充更多 note 字段映射
- 提升匿名模式下的命中率判断

### 接 `RedNote-MCP`

当前已经按 RedNote-MCP 的思路接了一版最小可用 provider：

1. 读取 cookie 文件
2. 或读取整段 Cookie 字符串
3. 动态加载 `playwright` / `playwright-core`
4. 打开真实页面
5. 抓取 DOM 中的 title/content/tags/images/videos

推荐先执行一次：

```bash
cd /Users/alexh/github_proj/caipu-miniapp/sidecars/xhs-sidecar
npm run rednote:init
```

登录完成后会把 Cookie 保存到 `~/.mcp/rednote/cookies.json`，跟 `RedNote-MCP` 默认路径保持一致。

之后可以检查状态：

```bash
cd /Users/alexh/github_proj/caipu-miniapp/sidecars/xhs-sidecar
npm run rednote:status
```

如果 `playwrightAvailable=true` 但 `browserInstalled=false`，说明还需要执行：

```bash
cd /Users/alexh/github_proj/caipu-miniapp/sidecars/xhs-sidecar
npx playwright install chromium
```

注意：

- 这一版依赖本地可用的 Playwright 环境
- 还没有做扫码登录流程
- 如果没有 cookie 文件，它会明确返回 `login_required`
- 如果没有安装 Playwright，它会明确返回 `provider_unavailable`

如果你要真正启用它，建议先准备：

```bash
cd /Users/alexh/github_proj/caipu-miniapp/sidecars/xhs-sidecar
npm install playwright
npx playwright install
```

然后把 RedNote-MCP 或其他方式产出的 cookie 文件指到：

```bash
XHS_REDNOTE_COOKIE_PATH=~/.mcp/rednote/cookies.json
```

如果你想像 B 站那样直接维护一段字符串，也可以直接配置：

```bash
XHS_REDNOTE_COOKIE_HEADER='a1=xxx; webId=xxx; gid=xxx'
```

这种方式更适合云服务器，但同样需要注意：

- 它本质上也是登录凭证
- 修改后需要重启 sidecar
- 过期后需要重新替换

## 注意

当前 stub 的 `echo` / `demo` 模式会在返回里写明 warning。

如果要接生产环境，建议：

- 先把 `XHS_SIDECAR_STUB_MODE=off`
- 再逐步替换 provider 的真实实现
