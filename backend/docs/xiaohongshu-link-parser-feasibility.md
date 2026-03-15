# 小红书图文链接解析可行性评估

这份文档面向当前 `caipu-miniapp` 的“添加菜品链接后，后台异步解析并补齐食材和步骤”场景。

目标问题只有一个：

- 当前已经支持 B 站链接异步解析，是否适合继续扩展到“小红书图文链接解析”

结论先说：

- **可行**
- **但不建议走匿名 HTTP 直抓**
- **更适合接一个带登录态的 sidecar，再复用现有 worker + AI 总结链路**

## 当前项目现状

当前仓库的解析链路已经完整跑通，但平台能力基本写死在 B 站：

- `internal/linkparse/service.go`：B 站链接识别、拉视频信息、拉字幕、AI 总结
- `internal/recipe/service.go`：保存菜谱时决定是否自动入队
- `internal/recipe/auto_parse_worker.go`：按固定间隔扫描 `pending` 菜谱并回写结果
- `internal/recipe/repository.go`：读取待处理任务、写入解析状态

也就是说，小红书接入并不需要重做整条链路，重点在于：

1. 新增“小红书链接 -> 图文正文/图片列表”的解析能力
2. 把当前写死的 `bilibili` 逻辑抽成多平台路由

## 调研方案

这次重点看了 4 类方案：

1. `xiaohongshu-importer`
2. `xhs-mcp`
3. `RedNote-MCP`
4. `MediaCrawler`

### 1. xiaohongshu-importer

适合度：

- **适合参考**
- **不适合直接接入生产后端**

原因：

- 它的能力模型和你要的 MVP 很接近：输入分享文本或链接，输出 `title/content/images/videos/tags`
- 但它本质是 Obsidian 插件，不是服务端组件
- 它的提取方式很轻：`requestUrl({ url })` 拉 HTML，然后从页面里的 `window.__INITIAL_STATE__` 提取 `note.desc`、`imageList`、`video`

这套思路的优点是：

- 很轻
- 接入成本低
- 非常适合快速做 demo

但它的问题也很明显：

- 高度依赖小红书当前网页结构
- 不带登录态
- 一旦分享链接落到 404 守卫页、登录页或空壳页，就拿不到正文

### 2. xhs-mcp

适合度：

- **适合做工程化 sidecar**

原因：

- 基于 Puppeteer
- 自带登录、Cookie 管理、浏览器持久化
- 有 `xhs_get_note_detail`
- 还能跑成 MCP / CLI / HTTP 模式

它的缺点是：

- 工程重量明显高于 `xiaohongshu-importer`
- `get note detail` 依赖 `feedId + xsecToken`
- 对你这个“用户只贴一个分享链接”的场景，还需要你自己补一层“分享链接 -> note detail 参数”的包装

### 3. RedNote-MCP

适合度：

- **最接近你当前项目的 sidecar 形态**

原因：

- 明确支持“通过 URL 访问笔记内容”
- 本身就会先登录、加载 Cookie，再用 Playwright 打开实际页面
- 直接从页面 DOM 抽 `title/content/tags/imgs/videos`

对你这个项目来说，它比 `xhs-mcp` 更顺手的地方在于：

- 输入直接就是 URL
- 输出结构更接近“图文摘要原料”
- 更适合做“只服务当前应用”的小型 sidecar

### 4. MediaCrawler

适合度：

- **功能强**
- **不适合作为第一步接入**

原因：

- 它对小红书支持很完整：关键词搜索、指定帖子 ID、评论、创作者主页、登录态缓存
- 但它更像一个多平台爬虫平台，而不是你当前项目里的一个轻量解析组件
- 引入成本、维护成本、法律/风控心智负担都更高

如果你后面想做：

- 批量采集
- 内容搜索
- 评论分析
- 多账号代理池

那它值得重新评估；但就当前“贴单条图文链接 -> 总结食材和步骤”来说，太重了。

## 我做的本地验证

这次没有只看 README，我额外做了两组验证。

### 验证 1：匿名请求分享链接 / explore 链接

我拿了几条公开流传的小红书分享链接和带 `xsec_token` 的 `explore` 链接做匿名请求。

结果比较一致：

- 一部分会跳到 `https://www.xiaohongshu.com/explore`
- 一部分会跳到 404 守卫页
- 返回 HTML 虽然仍有 `window.__INITIAL_STATE__`
- 但 `state.note.noteDetailMap` 里通常只有空壳结构，正文、图片都拿不到

典型现象是：

- 最终页面标题是 `小红书 - 你访问的页面不见了`
- `noteDetailMap` 的 key 是 `null`
- `note.desc` 为空
- `imageList` 为空

### 验证 2：直接套用 xiaohongshu-importer 的提取逻辑

我又把 `xiaohongshu-importer` 的核心提取逻辑按原样跑在这类返回页面上，结果也一致：

- `title` 只会得到 404 页标题
- `images` 是空数组
- `content` 是 `Content not found`

这说明一个关键事实：

- **`xiaohongshu-importer` 这类“匿名抓 HTML + 读 __INITIAL_STATE__”的轻量方案，在当前时点不适合作为你的服务端主方案**

它仍然有参考价值，但更适合作为：

- URL 归一化参考
- note 数据字段映射参考
- 本地离线导入工具

## 可行性结论

### 结论 1：小红书图文解析是可行的

原因：

- 你的现有架构已经有完整的异步解析链路
- 小红书图文的核心数据本身也适合被 AI 结构化提取
- 图文笔记比纯视频更适合先做 MVP

### 结论 2：匿名直抓不够稳

原因：

- 公开分享链接经常会被导向守卫页 / 404 / 登录态相关页面
- 即使拿到 HTML，也不一定包含有效 note detail
- 单纯依赖页面里的 `__INITIAL_STATE__` 很容易失效

### 结论 3：最适合你的接入方式是“登录态 sidecar”

推荐顺序：

1. **优先做图文笔记**
2. **sidecar 持有 Cookie / 登录态**
3. **Go 后端只负责调用 sidecar + AI 总结 + 写回菜谱**

## 对当前项目的建议接入方案

### 阶段 1：图文 MVP

目标：

- 用户保存一条小红书图文链接
- 后台自动获取标题、正文、图片列表、标签
- 用 AI 总结出食材和步骤

建议实现：

1. 保留现有 worker 模型不变
2. 在 `internal/linkparse` 增加小红书解析 client
3. 新增一个 sidecar 服务，职责只有：
   - 输入：分享文本 / 小红书 URL
   - 输出：`title/content/images/tags/isVideo/sourceUrl`
4. worker 识别到 `xiaohongshu.com / xhslink.com` 时，调用 sidecar
5. sidecar 成功返回后，沿用现有 AI 提取逻辑生成 `ingredient + parsedContent.ingredients + parsedContent.steps`

### 阶段 2：图片 OCR

因为很多小红书菜谱的关键内容不只在正文，还在图片里：

- 配料表
- 调料比例
- 分步图说明

所以如果只吃正文，效果会打折。

建议第二阶段补：

- 抽第一张封面和若干分步图
- OCR 出文本
- 把 OCR 文本和正文一起送给 AI

这样食材和步骤的完整度会明显提升。

### 阶段 3：视频笔记

如果后面要支持小红书视频笔记，再补：

- 音频转写
- OCR
- 视频图文混合总结

这一步明显比图文重，不建议一开始就做。

## 具体到当前仓库，建议怎么改

### 1. 抽平台路由

当前 `recipe/service.go` 和 `recipe/auto_parse_worker.go` 里，平台判断基本还是 B 站写死。

建议先抽出：

- `SupportsAutoParseURL(link string) bool`
- `DetectParsePlatform(link string) string`

让它支持：

- `bilibili`
- `xiaohongshu`

### 2. 抽统一结果模型

现在 `internal/linkparse/model.go` 里只有 `BilibiliParseResult`。

建议补一个更通用的结果模型，例如：

- `LinkParseResult`

统一包含：

- `platform`
- `link`
- `title`
- `content`
- `author`
- `coverURL`
- `images`
- `tags`
- `summaryMode`
- `recipeDraft`
- `warnings`

这样 B 站、小红书后面都能复用同一套 AI 总结入口。

### 3. sidecar 调用方式

推荐不要把浏览器自动化直接塞进 Go 主服务。

更稳的方式是：

- Go 主服务继续保持纯 API + worker
- 单独起一个 Node sidecar
- sidecar 内部用 Playwright / Puppeteer

这样好处是：

- 登录态维护和浏览器依赖独立
- 主服务发布不被浏览器依赖拖累
- 线上故障隔离更清楚

### 4. 配置方式

你已经有全局 B 站 `SESSDATA` 设置页，这套思路可以直接复用到小红书：

- 先支持全局 `XHS_COOKIE`
- 后面如果改成 sidecar 自己维护 Playwright cookies，就只需要在设置页里展示状态，不一定要让用户手贴整段 Cookie

## 推荐路线

如果按“最省事、最适合当前项目”的目标来排，我建议：

1. **不直接接 `xiaohongshu-importer`**
2. **优先考虑 `RedNote-MCP` 这类 URL 直达 + 登录态浏览器方案**
3. **如果后面要做更完整的小红书能力，再升级成 `xhs-mcp` 或独立 sidecar**

我的判断是：

- `xiaohongshu-importer`：适合参考，不适合做主方案
- `RedNote-MCP`：最适合先做 MVP
- `xhs-mcp`：适合做中期工程化版本
- `MediaCrawler`：适合更重的数据采集场景，不适合作为当前第一步

## 最终建议

对 `caipu-miniapp` 来说，小红书图文解析**值得做**，但建议按下面的结论执行：

- **做**
- **先只做图文**
- **不要走匿名直抓**
- **采用带登录态的浏览器 sidecar**
- **现阶段最推荐参考 / 试接的是 `RedNote-MCP` 方案**

如果下一步要真正开做，最值得优先出的不是“全量功能”，而是：

1. 小红书 URL 识别
2. sidecar API 草案
3. worker 路由到 `xiaohongshu`
4. 先用正文 + tags 做 AI 总结
5. 第二期再补 OCR

## 参考项目

- `xiaohongshu-importer`: <https://github.com/bnchiang96/xiaohongshu-importer>
- `xhs-mcp`: <https://github.com/algovate/xhs-mcp>
- `xiaohongshu-mcp`: <https://github.com/xpzouying/xiaohongshu-mcp>
- `RedNote-MCP`: <https://github.com/iFurySt/RedNote-MCP>
- `MediaCrawler`: <https://github.com/NanmiCoder/MediaCrawler>
