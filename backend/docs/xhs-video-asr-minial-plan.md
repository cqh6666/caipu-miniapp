# 小红书视频转字幕最小实现方案

本文档用于整理 `caipu-miniapp` 当前仓库里，为 `xhs-sidecar` 增加“小红书视频转字幕”能力的最小落地方案。

目标不是一步做到完整视频理解，而是先把下面这条链路跑通：

1. sidecar 解析小红书视频笔记
2. 拿到视频直链 `videos[0]`
3. 下载视频到临时文件
4. 用 `ffmpeg` 抽出 `mp3`
5. 调 ASR 接口转写字幕
6. 把 `transcript` 一起返回给 Go backend

---

## 一、当前已验证结论

基于当前仓库和真实样例，已经验证：

### 1. sidecar 可以解析出真实小红书视频直链
以这条笔记为例：

- 标题：`都去做这个神仙菜❗️葱姜煎鲳鱼好吃晕了❗️`
- 分享链接：`http://xhslink.com/o/AQp8fxWYBei`

当前 `xhs-sidecar` 已成功解析出：

- `noteType = video`
- `videos[0] = https://sns-video-hw.xhscdn.com/stream/79/110/258/01e8a3ea08bc8e364f03700198c04c7fbe_258.mp4`

### 2. 可以从视频中提取 MP3
已验证以下命令可行：

```bash
ffmpeg -y -i xhs_congjiang_changyu.mp4 -vn -acodec libmp3lame -q:a 2 xhs_congjiang_changyu.mp3
```

说明：

- `-vn`：只保留音频
- `libmp3lame`：导出为 mp3
- `-q:a 2`：较高质量 VBR

### 3. ASR 接口已验证连通
已验证下面接口可正常处理刚提取出的 mp3：

```bash
curl https://api.infiniteai.cc/v1/audio/transcriptions \
  -H "Authorization: Bearer <API_KEY>" \
  -F "file=@./audio.mp3"
```

实际返回 `HTTP 200`，并成功返回整段中文转写文本。

结论：

- 视频直链可拿到
- MP3 可提取
- ASR 可跑通

所以当前最小实现已经具备完整落地条件。

---

## 二、实现目标

在 `xhs-sidecar` 里增加视频字幕能力，使 sidecar 在解析到视频笔记时额外返回：

- `transcript`
- `transcriptStatus`
- `transcriptError`

示例：

```json
{
  "note": {
    "title": "都去做这个神仙菜❗️葱姜煎鲳鱼好吃晕了❗️",
    "content": "#小红书爆款美食# #超级下饭的家常菜#",
    "videos": [
      "https://sns-video-hw.xhscdn.com/stream/...mp4"
    ],
    "noteType": "video",
    "transcript": "这是一道普通的葱姜煎烧鲳鱼...",
    "transcriptStatus": "success",
    "transcriptError": ""
  }
}
```

---

## 三、为什么放在 sidecar

这部分更适合放在 `xhs-sidecar`，而不是放回 Go backend，原因如下：

1. **职责一致**：视频下载、媒体处理、ASR 都属于“外部内容解析能力”
2. **依赖不同**：需要 `ffmpeg`、临时文件、第三方转写接口
3. **风险隔离**：转写失败不应影响主业务服务的登录、厨房、菜谱 CRUD
4. **更易替换**：后续如果替换 ASR 服务，不需要改 Go 主后端核心逻辑

建议职责划分：

- `xhs-sidecar`：负责解析小红书内容 + 视频转字幕
- `backend/`：负责消费 `transcript` 并进一步做 AI 菜谱总结

---

## 四、最小改造范围

建议最小只改这些文件：

- `sidecars/xhs-sidecar/server.js`
- `sidecars/xhs-sidecar/providers/importer.js`
- `sidecars/xhs-sidecar/providers/rednote.js`
- `sidecars/xhs-sidecar/lib/transcript.js`（新增）

如需同步接入主后端，再补：

- `backend/internal/linkparse/service.go`
- 小红书 sidecar 响应结构体定义
- AI 总结 prompt 拼装逻辑

---

## 五、最小架构设计

### 当前链路

```text
前端/小程序 -> Go backend -> xhs-sidecar -> 返回 note 元信息
```

### 增强后链路

```text
前端/小程序
  -> Go backend
  -> xhs-sidecar
      -> 解析 note
      -> 下载视频
      -> ffmpeg 抽音频
      -> ASR 转写
  -> 返回 note + transcript
  -> Go backend 用 content + transcript 生成菜谱
```

---

## 六、统一处理位置建议

不建议分别在 `importer` 和 `rednote` 里都写一份转写流程。

建议做法：

- `provider` 仍然只负责拿到原始 `note`
- `server.js` 在返回前统一判断是否需要转写
- 如果需要，再调用 `lib/transcript.js` 做 enrich

这样好处是：

1. 避免重复实现
2. 降低 provider 耦合
3. 后续扩展 ASR provider 更容易

---

## 七、环境变量建议

推荐给 sidecar 增加这些环境变量：

```env
XHS_TRANSCRIPT_ENABLED=true
XHS_TRANSCRIPT_PROVIDER=infiniteai
XHS_TRANSCRIPT_API_KEY=sk-xxx
XHS_TRANSCRIPT_TIMEOUT_MS=120000
XHS_TRANSCRIPT_MAX_VIDEO_MB=80
XHS_TRANSCRIPT_KEEP_TEMP=false
FFMPEG_PATH=ffmpeg
```

最小必需：

- `XHS_TRANSCRIPT_ENABLED`
- `XHS_TRANSCRIPT_API_KEY`

说明：

- `XHS_TRANSCRIPT_ENABLED`：是否启用视频转字幕
- `XHS_TRANSCRIPT_PROVIDER`：当前先支持 `infiniteai`
- `XHS_TRANSCRIPT_API_KEY`：ASR Key
- `XHS_TRANSCRIPT_TIMEOUT_MS`：ASR 请求超时
- `XHS_TRANSCRIPT_MAX_VIDEO_MB`：可选，防止超大视频拖垮服务
- `XHS_TRANSCRIPT_KEEP_TEMP`：调试时保留临时文件
- `FFMPEG_PATH`：系统中 `ffmpeg` 可执行路径

---

## 八、返回字段设计

建议在 `note` 内新增字段：

- `transcript: string`
- `transcriptStatus: string`
- `transcriptError: string`

推荐状态值：

- `disabled`：全局未启用
- `skipped`：不是视频或没有视频链接
- `success`：转写成功
- `failed`：转写失败

建议策略：

- **转写失败不能让整次 note 解析失败**
- 即使 ASR 失败，也要正常返回 `title/content/videos`
- 只在 note 中补充失败状态

示例：

```json
{
  "note": {
    "title": "...",
    "videos": ["https://...mp4"],
    "transcript": "",
    "transcriptStatus": "failed",
    "transcriptError": "asr request failed"
  }
}
```

---

## 九、`lib/transcript.js` 的职责

建议新增文件：

- `sidecars/xhs-sidecar/lib/transcript.js`

对外只暴露一个入口：

```js
async function enrichTranscriptIfNeeded(note, config)
```

内部建议拆分为几个小函数：

- `downloadFile(url, outputPath)`
- `extractMp3(inputMp4, outputMp3, config)`
- `transcribeWithInfiniteAI(mp3Path, config)`
- `cleanupTempFiles(paths, config)`

### 1. 下载视频
输入：

- `note.videos[0]`

输出：

- `/tmp/.../input.mp4`

### 2. 提取 MP3
调用：

```bash
ffmpeg -y -i input.mp4 -vn -acodec libmp3lame -q:a 2 output.mp3
```

### 3. 调用 ASR 接口
接口：

```bash
POST https://api.infiniteai.cc/v1/audio/transcriptions
Authorization: Bearer <API_KEY>
multipart/form-data: file=@audio.mp3
```

取返回里的：

- `text`

### 4. 清理临时文件
默认行为建议：

- 成功后清理 mp4/mp3
- 失败后也清理
- 如果 `XHS_TRANSCRIPT_KEEP_TEMP=true`，则保留供排查

---

## 十、执行条件建议

最小版本只在以下条件满足时转写：

```js
if (!config.transcriptEnabled) skip
if (!note) skip
if (!Array.isArray(note.videos) || note.videos.length === 0) skip
if (note.noteType !== 'video' && note.videos.length === 0) skip
```

也可以后续增加请求参数：

```json
{
  "input": "...",
  "provider": "auto",
  "includeTranscript": true
}
```

但如果追求最小改造，第一期可以不加请求参数，先用全局环境变量控制。

---

## 十一、错误处理原则

推荐原则：

1. note 解析和 transcript 转写分层处理
2. transcript 失败不影响 note 解析结果
3. 返回结构里保留失败状态和错误信息
4. 服务日志里打印：
   - 视频下载耗时
   - ffmpeg 耗时
   - ASR 耗时
   - 失败原因

这样后续调试会方便很多。

---

## 十二、Go backend 的最小接入建议

sidecar 完成后，Go backend 只需小改：

1. 小红书 sidecar 响应结构增加：
   - `transcript`
   - `transcriptStatus`
   - `transcriptError`
2. 在 AI 总结 prompt 中拼入 `transcript`

推荐输入顺序：

1. `title`
2. `content`
3. `tags`
4. `transcript`

这样对视频类小红书内容的菜谱总结质量会明显提升，因为很多关键步骤只存在于口播中。

---

## 十三、实现顺序建议

### P1：打通 sidecar 视频转字幕

1. sidecar 增加 transcript 字段
2. 新增 `lib/transcript.js`
3. `server.js` 成功返回前做 transcript enrich
4. 用现有 API key 跑通真实样本

### P2：主后端消费 transcript

1. backend 接 transcript 字段
2. AI prompt 纳入 transcript
3. 验证菜谱提取效果

### P3：增强稳定性

1. 临时文件大小限制
2. 下载超时控制
3. ASR 超时控制
4. 同视频结果缓存
5. 增加结构化日志

---

## 十四、最小可行结论

基于当前验证结果，`xhs-sidecar` 的“小红书视频转字幕”最小实现方案已经明确：

- 先复用现有 provider 拿视频直链
- 使用临时文件下载 `mp4`
- 用 `ffmpeg` 提取 `mp3`
- 调 `https://api.infiniteai.cc/v1/audio/transcriptions`
- 将返回的 `text` 写入 `note.transcript`
- 即使转写失败，也不影响原有 note 解析成功返回

这是当前改动最小、成功率最高、最适合接入现有仓库的一条路线。
