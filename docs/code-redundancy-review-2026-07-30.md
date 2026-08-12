# 项目代码冗余设计审查报告

- **审查时间**：2026-07-30 21:15:48 +0800
- **整改更新时间**：2026-08-12 16:34:19 +0800
- **审查基线**：`46d327da1adc46cf568f15968406de23cf930205`
- **S0 整改落点**：`00efaab71e8fed51be470fbfc2387fd674eb6828`
- **审查范围**：微信小程序、Admin Web、Go 后端、linkparse sidecar、发布脚本、
  根依赖与测试入口
- **协作方式**：共 4 个 Agent 参与（主 Agent 1 个、前端/后端/工程子 Agent
  各 1 个）
- **交付性质**：初始只读审查报告，并追加 S0 整改追踪；未连接或修改生产环境

---

## 整改状态（更新至 2026-08-12）

RD-001 已按独立安全批次完成整改；第一阶段 S1 的 RD-007、RD-010、RD-014 与
RD-012 中的无效 stub 分支也已完成。其余问题仍保持本报告中的待办状态。

| 提交 | 内容 | 状态 |
| --- | --- | --- |
| `83bf5ae` | 新增统一 URL 策略和受控出站请求客户端 | 已完成并通过定向复审 |
| `00efaab` | B 站、小红书、视频转写及 Go sidecar 调用方接入安全边界 | 已完成并通过独立复审 |
| `eeee960` | 删除 S1 死代码、无引用样式、遗留函数、无用依赖与无效 stub 分支 | 已完成并通过独立复审 |

### 事实校正

初始报告对 Go 侧 B 站入口的描述不准确。审查基线中的
`fetchBilibiliViaSidecar` 已在读取 `SESSDATA` 和请求 sidecar 前调用
`extractInputURL`，会拒绝伪后缀域名；因此不能据此断言“已登录小程序用户可从该
HTTP 入口直接把伪后缀 URL 转发给 sidecar”。

基线中的真实缺口是：Go 在得到已校验的 `inputURL` 后，仍把未经规范化的
`rawInput` 发送给 sidecar，使跨进程边界两侧继续独立解析输入；同时 sidecar 自身的
域名匹配、重定向、DNS 目标和凭据发送策略不安全。伪后缀与 Cookie 的 fake fetch
复现证明的是 sidecar 直接调用或其他内部调用方场景下的安全边界漂移，而不是已经证实
的公开 Go 接口绕过。本文后续 RD-001 证据与影响均以这项校正为准。

### 已完成改动

- `lib/url-policy.js` 统一按 DNS label 匹配域名，并拒绝 URL 凭据、非默认端口、
  私网、回环、link-local、CGNAT、ULA、metadata 及可嵌入私网 IPv4 的 IPv6
  地址。兼容的默认端口 HTTP 输入会先规范化为 HTTPS，最终出站请求必须使用 HTTPS。
- `lib/safe-fetch.js` 对每一跳执行 URL 与全部 DNS 答案校验，固定已验证 IP，限制
  重定向、响应体大小和整条请求截止时间，并在跨源时移除 Cookie/Authorization。
- B 站短链与字幕文件不携带 `SESSDATA`；仅固定的
  `api.bilibili.com/x/web-interface/view` 和 `api.bilibili.com/x/player/v2`
  请求允许携带该凭据，且禁止重定向。
- Importer、媒体下载和转写下载接入受控请求层，返回给调用方的图片、视频与封面 URL
  也按小红书媒体域过滤。
- RedNote 禁用 Service Worker，以 `context.route` 限制导航、子资源和跳转，并关闭
  非预期 popup；Go 只把已校验的 `inputURL` 发送给 sidecar。

### 整改验证与残余风险

- 安全请求层定向测试 26 项通过；调用方接入后的 sidecar 全量测试 58 项通过。
- `go test ./internal/linkparse -count=1`、相关 Node 语法检查和
  `git diff --check` 通过；两轮独立 reviewer 最终均未报告阻塞问题。
- 2026-07-31 全量复核中，`npm test`、`npm --prefix admin-web run typecheck`、
  `npm --prefix sidecars/linkparse-sidecar test`、`cd backend && go test ./... -count=1`
  与 `cd backend && go vet ./...` 均通过。
- Playwright 路由只能检查 Chromium 暴露的 URL，不能固定浏览器实际使用的 DNS
  解析结果。生产仍需用容器网络、防火墙或受控代理阻止 loopback、RFC1918、
  link-local、metadata、CGNAT 和 ULA 出站流量。
- 本次未执行部署、生产连通性测试或任何生产配置变更。

### S1 第一阶段整改（2026-08-12）

- RD-007：删除 4 个只写不读状态、3 条末端不消费的 prop 链、随机推荐
  `contextText` 整条死契约、首页/详情页零调用成员、零消费导出、无用 import，及
  Miniapp/Admin 中经全仓检索确认无引用的样式规则；保留动态 class 与跨组件实际消费
  的规则。
- RD-010：删除 8 个无引用 Go 符号。Bilibili 设置仍保留事务内私有 upsert，现行保存
  继续走同事务 CAS 与审计入口；生产装配继续使用带 options 的 worker 构造器。
- RD-014：从根 `package.json` 与 lockfile 删除 `clipboard`、`dayjs`，并清理仅由
  `clipboard` 引入的 4 个传递依赖；未为“用上依赖”改写日期或剪贴板代码。
- RD-012：删除 importer 中与默认失败返回逐字相同的
  `stubMode === "off"` 分支，`demo`、`echo`、`off` 与未知值行为保持不变。
- 本阶段共修改 29 个运行时/配置文件，新增 8 行、删除 796 行；独立 reviewer 最终
  结论为“无阻塞问题”。
- 验证通过：`npm test`、`npm --prefix admin-web run typecheck`、sidecar 58/58、
  `cd backend && go test -p 1 ./... -count=1`、`cd backend && go vet ./...`、
  `npm ci --ignore-scripts --dry-run`、静态引用守卫、SFC 解析、Node 语法检查与
  `git diff --check`。HBuilderX 5.15 微信小程序纯编译成功，未触发自动预览或上传。
- 后端并行全量测试曾触发一次既有 SQLite 并发用例 `SQLITE_BUSY`；目标单测、包测试
  与串行全量复跑均通过。本次未连接生产服务或执行部署。

---

## 1. 结论先行

本轮确认 **15 组可操作的冗余问题**：

| 级别 | 数量 | 含义 |
| --- | ---: | --- |
| P0 | 1 | 冗余实现已经产生安全语义漂移，应立即处理 |
| P1 | 5 | 同一核心能力存在多个事实来源，维护成本或漂移风险高 |
| P2 | 8 | 可分批收口的重复编排、死状态、死代码或工程噪声 |
| P3 | 1 | 收益有限，可在相邻改动时顺手处理 |

最重要的结论不是“文件太大”，而是以下三类问题：

1. **同一安全策略维护多份**：审查基线中的 Go 与 sidecar 平台 URL 校验已经出现
   不同结果；sidecar 被直接调用时会接受伪后缀域名，并把 `SESSDATA` 带到该请求。
   该项已在 2026-07-31 完成 S0 整改。
2. **兼容路径已经有唯一入口，旧协议实现却仍保留**：AI Router 已承接单节点兼容，
   业务包仍维护生产不可达的旧 OpenAI 客户端。
3. **源码复用不等于产物复用**：多个 scoped 小程序组件导入同一整份 SCSS，
   最终 WXSS 仍按组件完整复制。

### 1.1 建议处理顺序

1. RD-001 的 sidecar URL 与凭据发送边界已作为 S0 独立修复。
2. 先做 RD-007、RD-010、RD-014 等低风险删除，缩小后续重构面。
3. 再处理 RD-002～RD-005 的核心事实来源收口。
4. 最后处理 UI/脚本/测试基础设施复用，避免一次大改混入多个风险面。

---

## 2. 审查方法

本轮没有直接复用 2026-07-14 的旧结论，而是先把它们作为排除清单，再对当前
HEAD 重新取证：

- 读取 `CHANGELOG.md`、`README.md`、`backend/README.md`、
  `docs/frontend-code-review-2026-07-14.md`、
  `docs/backend-code-review-todo-2026-07-14.md` 及重构路线图。
- 由三个子 Agent 分别审查小程序前端、Go 后端、Admin/sidecar/脚本。
- 对生产源码执行精确重复块扫描，排除测试、迁移、文档和构建目录。
- 用 `rg` 做全仓符号引用、入口装配和调用链核验。
- 对小程序现有本地开发产物比较 WXSS 行数和字节数。
- 对 sidecar 伪后缀域名做无网络、注入式最小复现。
- 运行现有前端、Admin 与 sidecar 测试作为基线。

---

## 3. P0：立即处理

### RD-001：平台 URL 校验重复且已漂移，sidecar 可把 SESSDATA 发往伪后缀域名（已整改）

- **位置**：
  - `backend/internal/linkparse/platform.go:34`
  - `backend/internal/securehttp/client.go:108`
  - `backend/internal/linkparse/handler.go:46`
  - `backend/internal/linkparse/bilibili.go:197`
  - `sidecars/linkparse-sidecar/providers/bilibili.js:39`
  - `sidecars/linkparse-sidecar/providers/bilibili.js:96`
  - `sidecars/linkparse-sidecar/providers/bilibili.js:107`
  - `sidecars/linkparse-sidecar/lib/normalize.js:2`
  - `sidecars/linkparse-sidecar/providers/importer.js:186`

#### 证据

Go 侧统一使用 `securehttp.HostMatches`，按 DNS label 边界匹配，并已有伪后缀
回归测试。sidecar 重新实现了两套规则：

- B 站使用 `host.includes("bilibili.com")`。
- 小红书使用无边界正则
  `/(xiaohongshu\.com|xhslink\.com)/`。

最小复现结果：

| 输入 | sidecar 判定 |
| --- | --- |
| `https://bilibili.com.attacker.example/share` | B 站支持 |
| `https://xiaohongshu.com.attacker.example/explore/123` | 小红书支持 |
| `https://xhslink.com.attacker.example/a/123` | 小红书支持 |

进一步给 B 站 provider 注入假的 `SESSDATA=TOP_SECRET` 和 fake fetch 后，捕获到：

```json
{
  "url": "https://bilibili.com.attacker.example/share",
  "cookie": "SESSDATA=TOP_SECRET"
}
```

Go 侧 B 站调用在进入 sidecar 前已执行 `extractInputURL` 严格校验，且只在校验
通过后读取 `SESSDATA`。初始报告关于该调用顺序及公开入口可直接触发的判断有误。
不过，Go 当时仍把 `rawInput` 而非已校验的 `inputURL` 发送到 sidecar；sidecar
若被直接调用或未来增加其他内部调用方，仍会独立接受上述伪后缀并携带凭据请求。

#### 影响

- sidecar 被直接调用或出现其他内部调用方时，可形成 SSRF allowlist 绕过。
- 在该调用前提下，B 站 `SESSDATA` 可能被发送到攻击者控制域名；审查未证明原 Go
  公开接口可以绕过其既有的精确域名校验。
- `redirect: "follow"` 没有逐跳复核 host 与解析后的目标 IP。
- 同一安全策略在 Go、B 站 JS、小红书 JS 三处维护，已经证明会漂移。

#### 最小整改（已完成）

1. sidecar 新增唯一 `lib/url-policy.js`，使用 `URL.hostname` 做 exact/subdomain
   label 匹配，禁止 `includes` 和无边界正则。
2. Go 仅把严格校验后得到的 `inputURL` 发送给 sidecar，不再跨边界发送
   `rawInput`。
3. 只有已确认属于 B 站可信域名的请求才能附加 `SESSDATA`。
4. 禁用自动重定向或手动逐跳校验；同时校验解析 IP，拒绝 loopback、link-local、
   RFC1918、metadata 和其他内部地址。
5. 为 B 站/小红书补伪后缀、用户名混淆、端口、重定向和 DNS 目标测试。

> 本项应先独立修复，不要等待后续大规模“去重重构”一起上线。

---

## 4. P1：核心事实来源收口

### RD-002：B 站抓取在 Go 与 Node sidecar 中维护两套完整实现

- **位置**：
  - `backend/internal/linkparse/bilibili.go:101`
  - `backend/internal/linkparse/bilibili.go:152`
  - `backend/internal/linkparse/bilibili.go:342`
  - `backend/internal/linkparse/bilibili_sidecar.go:16`
  - `sidecars/linkparse-sidecar/providers/bilibili.js:12`
  - `sidecars/linkparse-sidecar/providers/bilibili.js:289`

#### 证据

- Go 直连实现约 655 行。
- Node provider 约 381 行，另有 Go sidecar 适配约 65 行。
- 两边都实现输入提取、短链展开、BV/AV 解析、view API、分页、字幕选择、
  字幕下载和文本拼接。
- sidecar 配置存在时，请求失败会直接返回错误，不会回退到 Go 直连；因此这两套
  实现不是运行时高可用降级，而是配置模式分叉。
- RD-001 的 host 策略差异已经展示双实现的现实维护代价。

#### 建议

在运维约束确认后选一个 I/O owner：

- 若保留 sidecar：Go 只负责鉴权、业务编排、AI 总结和结果映射，删除 Go 直连抓取。
- 若保留 Go：删除 sidecar 的 B 站 provider，让 sidecar 只承载必须使用浏览器的能力。

无论选哪边，都不建议继续以“可能将来做 fallback”为理由维护两份当前不会互相
fallback 的协议栈。

---

### RD-003：AI Router 已是生产唯一入口，业务包仍保留旧直连客户端

- **位置**：
  - `backend/internal/app/ai_wiring.go:78`
  - `backend/internal/app/ai_wiring.go:131`
  - `backend/internal/airouter/service.go:412`
  - `backend/internal/linkparse/summary.go:70`
  - `backend/internal/linkparse/summary.go:270`
  - `backend/internal/recipe/flowchart.go:103`
  - `backend/internal/recipe/flowchart.go:136`
  - `backend/internal/recipe/flowchart_client.go:92`

#### 证据

生产组合根始终向 linkparse 和流程图生成器注入 `airouter.Service`；旧环境变量与
运行时单节点配置也已经由 `CompatibilityLoader` 映射成 Router scene。

但业务包仍维护：

- linkparse 的 OpenAI 请求体、鉴权、响应解析、错误映射、审计与大小限制。
- recipe 的 flowchart chat/images 两种 endpoint、响应解码、审计与大小限制。

这些分支只在 Router 为 nil 的手工构造或旧测试中使用。Router 非 nil 时一旦失败，
代码同样不会回退到旧 client。`FlowchartGenerator.IsConfigured` 还会检查本地
client，但 `Generate` 在 Router 非 nil 时不会使用它，形成判定与执行不一致。

#### 建议

- 给业务包注入窄的 Router 接口，而不是具体 `*airouter.Service`。
- 将 Router 作为 AI 功能必需依赖；兼容配置继续只由
  `CompatibilityLoader` 负责。
- 删除 linkparse `aiClient` 和 recipe `flowchartClient` 的协议实现。
- 保留 prompt、业务内容校验、图片 URL 提取等领域纯函数。
- 测试改为 fake Router 或真实 Router + `httptest`。

---

### RD-004：ParsedContent 契约与步骤规范化算法跨包复制

- **位置**：
  - `backend/internal/linkparse/model.go:8`
  - `backend/internal/linkparse/model.go:22`
  - `backend/internal/linkparse/model.go:36`
  - `backend/internal/recipe/model.go:8`
  - `backend/internal/recipe/model.go:22`
  - `backend/internal/recipe/model.go:36`
  - `backend/internal/linkparse/heuristic.go:190`
  - `backend/internal/linkparse/heuristic.go:230`
  - `backend/internal/recipe/parsed_content.go:191`
  - `backend/internal/recipe/parsed_content.go:231`
  - `backend/internal/recipe/auto_parse_worker.go:265`
  - `backend/internal/recipe/auto_parse_worker.go:379`

#### 证据

两个 `model.go` 中的 `ParsedStep`、`ParsedContent`、JSON marshal、兼容旧
`ingredients` 和字符串 `steps` 的 unmarshal 前 90 行除包名外相同。

`splitIngredientLines`、`cleanParsedSteps`、`buildParsedSteps`、
`compactParsedSteps`、`inferParsedStepTitle` 又分别维护。因为类型不同，
worker 还要手写逐字段转换。

两份规则已经轻微漂移：

- linkparse 清洗 detail 时使用 `cleanCandidateLine`。
- recipe 只做 `strings.TrimSpace`。
- “常用配菜/基础调味/常用调味料”的正则词表也不同。

#### 建议

提取中立深模块 `internal/recipecontent`：

1. 统一 `Step`、`Content` 与旧 JSON 兼容解码。
2. 统一食材分组、步骤清洗/压缩/标题推断纯函数。
3. 两业务包先用类型别名或薄适配迁移，避免让 linkparse 反向依赖 recipe。
4. 用相同输入同时断言“解析预览”和“最终持久化”的结果一致。

---

### RD-005：凭据加密包装完全复制，并依赖三次“构造后覆盖”

- **位置**：
  - `backend/internal/airouter/crypto.go:5`
  - `backend/internal/appsettings/crypto.go:5`
  - `backend/internal/airouter/service.go:71`
  - `backend/internal/appsettings/runtime_provider.go:75`
  - `backend/internal/appsettings/service.go:36`
  - `backend/internal/app/app.go:91`
  - `backend/internal/app/app.go:108`
  - `backend/internal/app/app.go:197`

#### 证据

两个 `crypto.go` 除 package 行外实现一致，只是再次包装
`credentialcipher.Box`。Runtime Provider、AI Router、App Settings 又分别：

1. 在构造函数中创建固定 `v1` cipher。
2. 在组合根中调用 `ConfigureCredentialKeys`，用相同 current/previous keyring
   覆盖。

#### 影响

安全敏感配置依赖“构造后必须再调 setter”的时序约定；新增调用方漏调时会静默使用
默认 `v1`，类型系统无法阻止。

#### 建议

组合根一次构造不可变 `*credentialcipher.Box`，通过构造参数/Options 注入三个消费者；
删除两份 wrapper 和可变 `ConfigureCredentialKeys`。测试所需便捷构造放在 test
helper，不保留第二套生产初始化协议。

---

### RD-006：scoped 共享 SCSS 在小程序产物中按组件完整复制

- **位置**：
  - `pages/index/components/meal-order-date-sheet.vue:91`
  - `pages/index/components/meal-order-cart-sheet.vue:127`
  - `pages/index/components/meal-order-checkout-sheet.vue:133`
  - `pages/index/components/meal-order-success-sheet.vue:81`
  - `pages/index/components/meal-order-sheet.scss:1`
  - `pages/index/components/add-link-preview-panel.vue:150`
  - `pages/index/components/add-recipe-preview-panel.vue:179`

#### 证据

四个点菜弹层在 scoped style 中导入同一份 500 多行 SCSS。现有本地
`mp-weixin` 开发产物显示：

| 组件 | WXSS |
| --- | ---: |
| date/cart/checkout/success 各自 | 522 行 / 14,261 bytes |
| 四份额外重复量 | 约 42,783 bytes |
| 两个智能识别面板各自 | 289 行 / 6,767 bytes |

四份点菜 WXSS 除 scope hash 外相同；每个组件还带入其他三个组件的专属选择器。
这说明源码 `@import` 只减少了源码重复，没有减少小程序组件产物。

#### 建议

- 将 `meal-order-sheet.scss` 拆成公共壳与 date/cart/checkout/success partial，
  每个 SFC 只导入实际使用部分。
- RD-008 合并智能识别展示骨架后，自然只生成一份对应 WXSS。
- 改后重新执行 HBuilderX 微信编译，对比 WXSS 大小并逐个弹层截图回归。

> 上述字节数来自现有本地开发产物，不等同于生产压缩包的最终节省量。

---

## 5. P2：可分批收口

### RD-007：前端积累了死状态、死契约与无引用样式（已整改）

#### A. 只写不读状态

| 状态 | 声明位置 | 现状 |
| --- | --- | --- |
| `placeSyncErrorMessage` | `pages/index/index.vue:640` | 只赋值，无展示/判断 |
| `isLoadingPlaces` | `pages/index/index.vue:641` | 只赋值，无展示/判断 |
| `mealOrderStoreLoadedKitchenId` | `pages/index/index.vue:698` | 多处写入，无读取 |
| `kitchenMembersKitchenId` | `pages/index/index.vue:725` | 多处写入，无读取 |

对应写入集中在：

- `pages/index/use-place-library.js:100`
- `pages/index/use-meal-order.js:78`
- `pages/index/use-kitchen-space.js:333`

#### B. 完整传递但末端不消费

- 随机推荐生成并维护 `randomPickContextText`，父层传给
  `random-pick-sheet`，但组件模板不读 `contextText`：
  `pages/index/use-recipe-library.js:113`、
  `pages/index/index.vue:491`、
  `pages/index/components/random-pick-sheet.vue:104`。
- `libraryHeaderSummary` 已在首页显示，又经两层 prop 传递到不读取它的叶子组件：
  `pages/index/index.vue:42`、
  `pages/index/index.vue:55`、
  `pages/index/components/library-pane.vue:178`、
  `pages/index/components/library-header-section.vue:82`。
- `placeExtracted` / `placeParseSource` 在父层仍有业务用途，但传给候选弹层的两个 prop
  未被读取：`pages/index/index.vue:441`、
  `pages/index/components/place-candidate-sheet-v2.vue:88`。

#### C. 确认无调用成员与导出

- 首页：`pages/index/use-recipe-library.js:541`、
  `pages/index/use-recipe-library.js:711`、
  `pages/index/use-recipe-library.js:738`。
- 详情页：`pages/recipe-detail/index.vue:581`、
  `pages/recipe-detail/index.vue:851`、
  `pages/recipe-detail/index.vue:1247`。
- 零消费导出：`pages/index/storage.js:27`、
  `pages/index/storage.js:35`、
  `pages/index/recipe-card.js:79`、
  `pages/index/use-place-library.js:49`、
  `utils/auth.js:143`、
  `utils/recipe-model.js:422`。
- 无用 import：`pages/index/use-recipe-library.js:6`、
  `pages/index/use-smart-add.js:9`、
  `pages/recipe-detail/index.vue:216`。

#### D. 无引用 SCSS

- `pages/index/index-page.scss:657` 起的旧统计/菜单/simple list 样式约 139 行。
- `pages/index/components/space-stats-sheet.scss:13` 起的旧 popup header。
- `pages/recipe-detail/recipe-detail-page.scss:527` 等多组旧 modifier。
- `pages/index/components/library-header-section.scss:46` 等旧普通模式元素。
- Admin AI Provider 的孤儿规则见
  `admin-web/src/pages/ai-providers-page.css:276`、
  `admin-web/src/pages/ai-providers-page.css:385`、
  `admin-web/src/pages/ai-providers-page.css:570`、
  `admin-web/src/pages/ai-providers-page.css:1008`、
  `admin-web/src/pages/ai-providers-page.css:1277`、
  `admin-web/src/pages/ai-providers-page.css:2125`。

#### 建议

- 除随机推荐文案外，其余可按文件直接删除并补静态引用守卫。
- 随机推荐需先做产品二选一：恢复“推荐理由”展示，或删除整条 context 链。
- 样式按 selector 簇删除，不顺带改版；前后执行 SFC/微信编译和截图回归。

---

### RD-008：两个智能识别面板仍复制同一组件骨架

- **位置**：
  - `pages/index/components/add-link-preview-panel.vue:1`
  - `pages/index/components/add-recipe-preview-panel.vue:1`
  - `pages/index/components/add-preview-panel.scss:1`

7 月 14 日已经统一请求编排与 SCSS，但两个 151/180 行 SFC 仍复制 popup、解析态、
输入区、关闭、手动录入、props 和 emits。差异主要是文案、平台卡片和菜谱链接预判。

**建议**：抽纯展示 `AddPreviewPanel`，平台卡片和文案使用 props/slot；两个薄 wrapper
保留各自校验，避免把领域判断塞进大量条件分支的万能组件。预计可减少约 80～110 行
SFC，并配合 RD-006 消除一份 WXSS。

---

### RD-009：七个 Handler 完整复制 kitchenID 路径参数解析

- **位置**：
  - `backend/internal/kitchen/handler.go:138`
  - `backend/internal/place/handler.go:191`
  - `backend/internal/recipe/handler.go:308`
  - `backend/internal/mealplan/handler.go:207`
  - `backend/internal/invite/handler.go:183`
  - `backend/internal/spacestats/handler.go:45`
  - `backend/internal/addpreview/handler.go:51`

七处均执行相同的 `chi.URLParam`、trim、必填校验、`ParseInt`、正数校验和同一
`AppError`，约 80 行重复。

**建议**：在 `internal/common` 或窄的 HTTP helper 包增加
`PositiveInt64URLParam(r, "kitchenID")`。不要把 dietassistant 的 query 参数或其他
ID 一并泛化成反射绑定框架。

---

### RD-010：后端存在一组确认无引用的遗留函数（已整改）

- **位置**：
  - `backend/internal/airouter/repository.go:181`
  - `backend/internal/airouter/repository.go:197`
  - `backend/internal/appsettings/repository.go:72`
  - `backend/internal/recipe/repository_codec.go:421`
  - `backend/internal/recipe/auto_parse_worker.go:47`
  - `backend/internal/linkparse/preview_title.go:210`
  - `backend/internal/invite/share_image.go:345`
  - `backend/internal/invite/share_image.go:426`

全仓引用核验确认：

- `listSceneRecordsSnapshot` 只被同样未调用的 `listSceneRecords` 使用。
- `UpsertBilibiliSession` 无调用；现行保存走带 CAS 与同事务审计的入口。
- `mergeRecipeImageURLs`、`NewAutoParseWorker`、
  `isLowConfidencePreviewTitle`、`inviterInitial`、`drawMetricCard` 仅定义。

**建议**：直接删除。尤其不要保留 `UpsertBilibiliSession` 这个可绕过现行事务审计
入口的“捷径”。若未来确需批量 scene 查询，应让现行调用链真正使用并补快照一致性测试，
而不是并存两套 repository 路径。

---

### RD-011：Admin Calls/Jobs 重复列表控制器，任务详情编排重复三次

- **位置**：
  - `admin-web/src/pages/CallsPage.vue:189`
  - `admin-web/src/pages/JobsPage.vue:194`
  - `admin-web/src/pages/CallsPage.vue:317`
  - `admin-web/src/pages/JobsPage.vue:312`
  - `admin-web/src/pages/DashboardPage.vue:771`

Calls/Jobs 两页的路由同步、分页、request query、筛选应用/重置、时间范围、filter chip、
loading/error 高度同构；任务详情加载与 Call/Job drawer 互跳又在三页重复。

**建议**：

1. 先抽小而深的 `useJobDetailDrawer`。
2. 再抽可配置字段编解码的 `useRoutedAdminList`。
3. 页面继续拥有字段声明、文案、表格列和 active filter 展示，避免泛化整个页面。

---

### RD-012：sidecar 公共层提取不足

- **位置**：
  - `sidecars/linkparse-sidecar/providers/importer.js:6`
  - `sidecars/linkparse-sidecar/providers/importer.js:21`
  - `sidecars/linkparse-sidecar/providers/importer.js:35`
  - `sidecars/linkparse-sidecar/providers/rednote.js:11`
  - `sidecars/linkparse-sidecar/providers/rednote.js:15`
  - `sidecars/linkparse-sidecar/providers/rednote.js:29`
  - `sidecars/linkparse-sidecar/server.js:281`
  - `sidecars/linkparse-sidecar/server.js:349`
  - `sidecars/linkparse-sidecar/providers/importer.js:324`

确认三类重复：

1. importer/rednote 的 `unique`、媒体 URL 归一化、echo/demo note 骨架。
2. XHS/Bilibili handler 的 JSON 解析、空输入、错误 envelope、provider 选择、
   chain 调用和响应收尾。
3. importer 的 `stubMode === "off"` 分支与紧随其后的默认返回完全相同
   （该分支已在 S1 删除）。

**建议**：分别抽 `xhs-provider-common` 与窄的 parse request/error helper；平台抓取与
URL 策略保持显式。先删除无效 `off` 分支。不要一次抽成承载所有平台差异的万能框架。

---

### RD-013：两份服务器发布脚本复制主机资源检测

- **位置**：
  - `scripts/deploy-on-server.sh:50`
  - `scripts/deploy-on-server.sh:76`
  - `scripts/deploy-linkparse-sidecar-on-server.sh:28`
  - `scripts/deploy-linkparse-sidecar-on-server.sh:54`

CPU、内存、Swap、低资源判断、低优先级执行和摘要输出约 60 行同源逻辑维护两份。
两个部署入口本身应继续独立，但主机探测不是领域差异。

**建议**：提取无副作用的 `scripts/lib/host-resources.sh`，两个脚本 source；保留各自
对“后台构建”与“sidecar npm install”的不同决策。改后执行 `bash -n` 与两个
`PLAN_ONLY=1` 契约测试。

---

### RD-014：根工程保留两个完全未使用的直接依赖（已整改）

- **位置**：
  - `package.json:28`
  - `package.json:29`

`clipboard`、`dayjs` 在业务源码中没有 import。剪贴板使用 uni API 或浏览器原生
`navigator.clipboard`。当前本地安装中两者连同 `clipboard` 的传递依赖约占
2.3 MiB；未被 import，因此主要影响安装、lockfile 和供应链面，而不是小程序运行包。

**建议**：从 dependencies 与 lockfile 删除；不要为了“用上 dayjs”反向改写现有日期
代码。执行 clean install、`npm test` 和微信编译。

---

## 6. P3：相邻改动时顺手处理

### RD-015：日期格式化与 Admin 私有测试 runner 仍有小规模重复

- 日期格式化分散在 `pages/app-settings/index.vue:199`、
  `pages/index/use-kitchen-space.js:607`、
  `pages/recipe-detail/use-recipe-async-jobs.js:75`。
- Admin 三条测试脚本在 `admin-web/package.json:10` 重复
  `esbuild -> /tmp -> node`；自定义 `assertEqual` 又分散在
  `admin-web/tests/ai-provider-utils.check.ts:27`、
  `admin-web/tests/dashboard-distribution.check.ts:7`、
  `tests/miniapp-frontend.check.ts:60`。

**建议**：

- 日期工具只抽小函数，以参数明确“是否带年份/空值文案”，不为此保留 `dayjs`。
- 短期合并测试执行器和断言 helper；测试继续增长时再评估 `node:test` 或 Vitest，
  不要为现有少量用例立即引入重框架。

---

## 7. 需要外部使用证据后再处理

以下项目在仓库内看似冗余，但存在仓库外调用或历史资料价值，不能直接删除。

### C-001：废弃部署入口

- `backend/scripts/deploy.sh:5` 明确标记废弃，执行只会报错并退出。
- 删除前需检查服务器 cron、外部 CI、个人命令别名和运维文档。
- 若保留它是为了阻止旧危险发布，应记录明确移除版本/日期，而不是无限期兼容。

### C-002：美团探测 POC

- `scripts/probe-meituan-place-link.mjs:697` 是约 750 行独立 CLI。
- 未接入 package scripts、CI、后端或 sidecar，仅被产品设计文档引用。
- 若已不再用于人工研究，应删除或迁到 `docs/research/tools/` 并标记
  frozen/unsupported；若仍使用，则补正式命令入口和最小纯函数测试。

### C-003：旧 Go 启动设计文档

- `README-go.md:1` 是 1,132 行“从 0 到 1”启动方案。
- 全仓没有入口引用，内容仍写“Go 1.24+ / 第一版”，与当前
  `backend/README.md`、Go 1.26.5、现行安全和发布契约重叠且部分过时。
- 可在确认没有历史审计价值后删除；若要保留，应移入历史设计目录并加醒目的
  archived 标记。

---

## 8. 明确不建议当作冗余处理

为避免“DRY 过度”，以下相似代码本轮主动排除：

1. **各业务包的包内 `HTTPDoer`**：不同上游有独立 SSRF、大小上限和错误语义，
   不应合并为万能 HTTP 客户端。
2. **仓储事务内 membership 复核**：用于关闭 TOCTOU 窗口，不能上移成服务层一次检查。
3. **两个历史 `019_*` migration、旧密文解码、旧 ParsedContent 输入**：属于已发布
   兼容契约，没有版本下线证据前不能删除。
4. **小程序本地/远端空间统计双路径**：本地聚合承担离线和远端失败降级，不是无意义双算。
5. **`utils/recipe-store.js` barrel**：当前仍有多个生产消费者，暂不属于死兼容层。
6. **按文件行数机械拆 `spacestats/repository.go` 或
   `admin/server_health.go`**：缺少查询退化或职责失控证据。
7. **后端/后台独立部署 wrapper**：它们固化不同发布范围，是安全入口，不因只有几行就删。

---

## 9. 建议实施批次

| 批次 | 内容 | 风险 | 验收 |
| --- | --- | --- | --- |
| S0 | RD-001 URL/凭据边界 | 高影响、改动可控 | sidecar + Go 伪后缀/重定向/凭据测试 |
| S1（已完成） | RD-007、RD-010、RD-014 与无效 stub 分支 | 低 | 全仓引用、前端/Go/sidecar 测试 |
| S2 | RD-006、RD-008 前端产物去重 | 中 | HBuilderX 编译、WXSS 体积、弹层截图 |
| S3 | RD-004、RD-005 领域/凭据事实来源 | 中高 | JSON 兼容、密钥轮换、全量竞态 |
| S4 | RD-002、RD-003 删除双协议栈 | 高 | 直连/sidecar 决策、Router 契约、全量回归 |
| S5 | RD-009、RD-011～RD-015 工程收口 | 低到中 | 定向测试、typecheck、脚本 PLAN_ONLY |

每个批次建议单独提交，S0 不与其他重构混合。

---

## 10. 验证记录与限制

### 2026-07-30 审查基线已执行

- `npm run test:miniapp`：通过。
- `npm test`：通过。
- `npm --prefix admin-web run typecheck`：通过。
- `npm --prefix sidecars/linkparse-sidecar test`：17 项通过。
- `cd backend && go test ./... -count=1`：通过。
- `cd backend && go vet ./...`：通过。
- 精确重复块扫描：确认 ParsedContent、cipher wrapper、handler 参数解析、
  provider 公共骨架与脚本资源探测重复。
- sidecar fake fetch 最小复现：确认伪后缀被接受且 B 站 Cookie 被附加。
- `git status --short`：审查阶段无代码改动。

### 2026-08-12 S1 整改已执行

- `npm test`、`npm --prefix admin-web run typecheck`：通过。
- `npm --prefix sidecars/linkparse-sidecar test`：58 项通过。
- `cd backend && go test -p 1 ./... -count=1`、`cd backend && go vet ./...`：通过。
- `npm ci --ignore-scripts --dry-run`、`npm prune --ignore-scripts`：lockfile 一致，
  本地清理 6 个已移除依赖，审计 0 漏洞。
- 目标符号与 selector 全仓引用守卫、SFC 解析、Node 语法检查、
  `git diff --check`：通过。
- HBuilderX 5.15 `launch mp-weixin --compile true`：编译成功，未自动预览或上传。
- 独立 reviewer：无阻塞问题。

### 未执行/限制

- 未连接生产服务、数据库、Nginx 或真实 sidecar。
- 未使用真实 `SESSDATA`，复现仅使用假值和注入式 fake fetch。
- 未重新执行 HBuilderX 微信编译；RD-006 的字节数来自当前工作区已有开发产物，
  实际改造收益需重编译确认。
- 未核查 cron、外部 CI、个人脚本，因此 C-001～C-003 只列为条件清理。
