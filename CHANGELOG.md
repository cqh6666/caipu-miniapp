# Project Changelog

## 2026-04-23 (P2-D 第三阶段补丁 2：公开只读页隐藏「来源链接 / 备注」空卡)

### Fixed

- **修改时间**：2026-04-23
- **背景**：DTO 收敛后 `link` 和 `note` 已不再吐给公开接口，但前端「来源链接」和「备注」两张卡的外层容器没有 `v-if`，公开模式下仍渲染「暂无链接 / 暂无备注」灰字空卡，体验上像「这里没内容」而不是「该内容不公开」。
- **核心改动**：`pages/recipe-detail/index.vue:355-380` 两张 `detail-card--quiet` 外层容器各加 `v-if="!isPublicView"`，公开模式整块跳过；私有模式下保留「暂无链接 / 暂无备注」作为「这里可以填」的引导，行为不变。
- **影响范围**：单文件、单处模板改动。
- **兼容性·风险**：极低。模板纯 v-if 收敛，不动 JS / 样式 / 数据。
- **验证情况**：✅ esbuild 校验通过。真机联调建议补：① 公开模式：滚动到底部应只看到做法卡片，无空白「来源链接 / 备注」卡；② 私有模式 + 空 link/note：仍显示「暂无链接 / 暂无备注」灰字卡。

## 2026-04-23 (P2-D 第三阶段补丁：公开只读「无图菜谱」首屏空白修复)

### Fixed

- **修改时间**：2026-04-23
- **背景**：第三阶段交付后 review 指出，公开只读页里「无成品图」菜谱会先出现一大块约 380rpx 的空白 Hero 区再显示标题。根因：`.hero-card` 容器始终渲染（`pages/recipe-detail/index.vue:17`），P2-D 阶段把无图占位的「上传成品图」CTA `v-if !isPublicView` 隐藏后（line 83），placeholder 内容空了，但 `.hero-card { min-height: 380rpx }` 还在（line 3121），导致公开分享体验顶部出现裸色块。
- **核心改动**：`pages/recipe-detail/index.vue:16-25` Hero 容器加 `v-if="displayRecipeImages.length || !isPublicView"`——「公开 + 无图」时整块跳过 Hero 渲染，下方 `detail-head` 兜底分支（line 93）无缝接管标题 + meta 显示，无视觉残留。私有模式（含无图）保留原 380rpx placeholder + 上传 CTA，体验不变。
- **影响范围**：`pages/recipe-detail/index.vue` 单文件、单处模板改动；不动 JS/computed/样式。
- **兼容性·风险**：极低。`handleHeroCardTap` 是 Hero 容器的 @tap，容器不渲染则不触发，无副作用。私有模式逻辑完全不变。
- **验证情况**：✅ esbuild 校验通过。真机联调建议补：① 公开模式 + 无图菜谱：标题应紧贴页面顶部 banner，无空白；② 公开模式 + 有图菜谱：Hero 正常显示；③ 私有模式 + 无图菜谱：「上传成品图」placeholder 仍占满 380rpx，可点击触发上传。

## 2026-04-23 (P2-D 第三阶段：share_token Review 修复 — P1×3 / P2×1 / DTO 收敛 / 补测试)

### Fixed

- **修改时间**：2026-04-23
- **背景**：上一轮 share_token 公开只读机制交付后，code review 指出 4 条 P1/P2 与 2 条 Open Question：① 公开模式下二次转发时 `buildRecipeShareConfig` 只读 `this.shareToken`，未把 onLoad 入参 token 同步写回，导致接收者再分享发出去的链接不带 token，第二跳被鉴权墙拦回；② Hero 区「上传成品图」与步骤区「生成一图看懂」CTA 漏加 `v-if !isPublicView`，已登录成员从公开链接进入仍可触发写接口（与「只读」承诺冲突）；③ `EnsureShareToken` 实现是「先查空 → 生成 → 无条件 UPDATE」三步非原子，并发下后写者覆盖先写者，先返回给前端的 token 立刻失效；④ `applyRecipe` 末尾才异步 ensure token，用户秒分享会拿到旧版鉴权链接；⑤ 公开接口直接吐完整 `Recipe`（含 `note` 等私人字段），后续给 Recipe 加字段会默认泄漏；⑥ 测试只覆盖 happy path，缺并发与 DTO 字段防御回归。本次一次性修完，避免分批回归成本。
- **核心改动**：
  - **前端 · P1-1 二次转发兜底**（`pages/recipe-detail/index.vue`）：① `onLoad` 解析到 `shareToken` 时同步写入 `this.shareToken`（不只是 `publicViewToken`）；② `buildRecipeShareConfig` 改为 `effectiveToken = this.shareToken || this.publicViewToken` 双保险，公开模式下任何渠道转发都能正确拼接 token。
  - **前端 · P1-2 写入口封堵**（`pages/recipe-detail/index.vue`）：① 模板：Hero 上传 placeholder（`hero-card__placeholder`）、步骤区「生成『一图看懂』」CTA（`cooking-flowchart-cta`）补加 `v-if !isPublicView`；② 方法层防御性 guard：`chooseHeroImages / handleGenerateFlowchart / openEditSheet / confirmDeleteRecipe / handleParseAction / openCookingMenu / openHeroActionMenu` 入口处统一加 `if (this.isPublicView) return`，避免极端情况下被代码路径触发写接口。`toggleStepCompleted` 仅写本地 storage 不打后端，保留可用。
  - **后端 · P1-3 EnsureShareToken 并发原子化**（`backend/internal/recipe/share_token.go` + `repository.go`）：① `Repository.SetShareToken` 改为「条件 UPDATE：仅当 `share_token IS NULL OR share_token = ''` 时才写」，返回 `(written bool, err error)` 区分「本次成功写入」与「并发竞争失败」；② `Service.EnsureShareToken` 在 `SetShareToken` 返回 `written=false` 时调 `GetShareToken` 回查库里真正生效的 token 返回给调用方，确保所有并发请求最终拿到同一个生效 token。
  - **前端 · P2-1 窗口期修复**（`pages/recipe-detail/index.vue`）：① ensure share_token 时机从「`applyRecipe` 末尾」提前到「`loadRecipe` 拿到 recipeId 后立即 fire」（与缓存读取并行），缩短「打开详情秒分享」窗口；② `ensureShareTokenIfNeeded` 改为返回 `Promise<string|null>` 并用 `_shareTokenEnsurePromise` 字段去重，避免重复请求；③ `onShareAppMessage` / `onShareTimeline` 在 token 未就绪时使用微信 `promise` 字段（基础库 2.12.0+ / 3.12.0+ 兜底）等 ensure 完成后再返回完整 config，老版本会忽略 promise 自动回退到同步配置，行为退化但不报错。
  - **后端 · Open Q1 公开 DTO 白名单收敛**（`backend/internal/recipe/share_token.go`）：新增 `PublicRecipe` struct 作为公开只读视图白名单 DTO，**仅暴露** `id / title / ingredient / summary / imageUrl / imageUrls / flowchartImageUrl / mealType / status / parsedContent / parsedContentEdited`；**剔除** `note`（私人备注）、`link`（原始链接，可能是私域）、`kitchenId / createdBy / updatedBy`（内部 ID）、`createdAt / updatedAt / pinnedAt`（时间戳/排序）、`flowchartProvider / flowchartModel / flowchartStatus / flowchartError / flowchart*RequestedAt/FinishedAt/UpdatedAt / flowchartStale`（流程图过程字段）、`parseStatus / parseSource / parseError / parse*RequestedAt/FinishedAt`（解析过程字段）、`shareToken`（递归暴露无意义）。`PublicRecipeView.Recipe` 类型从 `Recipe` 收窄为 `PublicRecipe`，编译期保证字段白名单。
  - **后端 · Open Q2 补测试**（`backend/internal/recipe/share_token_test.go`）：① `TestEnsureShareTokenConcurrentReturnsSameToken`：16 个 goroutine 并发 ensure 同一 recipeID，断言全部拿到同一 token 且与库内 token 一致（验证 P1-3 修复）。SQLite `:memory:` DB 通过 `db.SetMaxOpenConns(1)` 强制单连接，模拟「条件 UPDATE 串行化」场景；② `TestPublicRecipeViewExcludesPrivateFields`：seed 时 note 写入「私人备注：少放盐」，断言 PublicRecipe JSON 不含 `"note" / "link" / "createdBy" / "shareToken" / "flowchartProvider" / "parseStatus"` 等敏感字段（验证 Open Q1 修复）。
- **影响范围**：
  - 后端：`backend/internal/recipe/share_token.go`（Service 层 + 新 PublicRecipe DTO）、`backend/internal/recipe/repository.go`（SetShareToken 签名变更）、`backend/internal/recipe/share_token_test.go`（新增 2 个用例 + seed 加 note）。无 migration 改动，无路由改动。
  - 前端：`pages/recipe-detail/index.vue`（onLoad / loadRecipe / buildRecipeShareConfig / ensureShareTokenIfNeeded / onShareAppMessage / onShareTimeline / 7 个写入口方法 guard / 2 个模板 v-if / data 加 `_shareTokenEnsurePromise` 字段）。无 utils 文件签名变化。
- **兼容性·风险**：
  - **后端 API 兼容性**：`PublicRecipeView.Recipe` 类型由 `Recipe` 改为 `PublicRecipe`，对应公开接口 `GET /api/public/recipes/by-share-token/{token}` 返回的 JSON 中 recipe 字段缺失 `note / link / createdBy / pinnedAt / flowchart*（除 imageUrl）/ parse* / shareToken`；前端 `normalizeRecipe` 对所有缺失字段都用 `||` 兜底为空值，不会报错；上一轮上线尚未真机联调，无线上回归风险。
  - **后端 Repository 签名变更**：`SetShareToken` 返回值从 `error` 改为 `(bool, error)`，仅 `share_token.go` 一处调用，无外部依赖。
  - **前端 onShareAppMessage promise**：基础库 2.12.0+ 才识别 `promise` 字段，老版本会忽略并使用同步返回值（不带 token 的旧版链接），行为退化但不报错。
  - **公开模式写入口防御**：模板 `v-if` + 方法 guard 双保险，即使未来有新写入口被忘记加 v-if，方法层 guard 也能兜底。新增写方法时仍需主动加 `if (this.isPublicView) return`。
  - **token 永久有效**：本轮未引入 token 主动失效机制，依然是已知遗留风险；除非删除菜谱，token 一旦泄露持有者可永久访问。
- **验证情况**：
  - 后端：`GOCACHE=/tmp/caipu-go-build-cache go test ./internal/recipe ./internal/kitchen` 全过（含新增 2 个用例 + 原 4 个用例 = 6 个 share_token 用例）。
  - 前端：`awk '/^<script>/{flag=1;next} /^<\/script>/{flag=0} flag' pages/recipe-detail/index.vue > /tmp/x.js && npx esbuild /tmp/x.js --log-level=error` 无错。
  - **未做真机联调**（沿袭上轮挂起项），需用户在真机或微信开发者工具补：① 公开模式下二次转发：A→B→C 链路 C 端能否正常打开（验 P1-1）；② 已登录成员从公开链接进入：Hero 区无「上传成品图」按钮、步骤区无「生成一图看懂」CTA（验 P1-2）；③ 同一 recipeID 多端同时首次访问详情页：所有端拿到同一 token（验 P1-3，需后端日志或 DB 抽查辅助）；④ 打开详情后立即点「转发」：链接应带 shareToken（验 P2-1）。

## 2026-04-23 (P2-D 第二阶段：share_token 公开只读访问机制)

### Added

- **修改时间**：2026-04-23
- **背景**：上一轮 P2-D 「开启微信原生分享」交付后，遗留 P1 缺陷「分享接收者无空间权限会打不开详情页」。所有菜谱接口都挂在 `protected.Group(authMiddleware)` 下，前端 `getRecipeById` 又必走 `ensureSession`，导致非空间成员（未登录或登录态属于其他空间）打开分享卡时被鉴权墙挡住，看到的是 toast 而非有意义的内容。本次落地 share_token 方案，让分享出去的链接对接收者「即点即看」，不强求登录或加入空间。
- **核心改动**：
  - **后端 · 数据层（migration 018）**：`backend/migrations/018_add_recipe_share_token.sql` 给 `recipes` 表新增 `share_token TEXT NOT NULL DEFAULT ''` + `share_token_created_at TEXT NOT NULL DEFAULT ''` 两列，并在 `share_token != ''` 上建唯一部分索引。`Recipe` 结构体加 `ShareToken string \`json:"shareToken,omitempty"\`` 字段，但有意**不进 `scanRecipe` 主流程**——主流程涉及 8 处 SELECT 与多个内嵌 schema fixture，引入新字段牵动面太大；share_token 只在「ensure / 公开查」两个独立路径用单独的轻量方法读写。
  - **后端 · Repository 三个独立方法**（`backend/internal/recipe/repository.go` 末尾）：① `GetShareToken` 仅读 `share_token` 列；② `SetShareToken` 仅写 token + 时间戳；③ `FindByShareToken` 走 `scanRecipe` 主流程查菜谱（公开接口走这条）；④ 新增 `FindKitchenAndCreatorMeta`，单条 JOIN `recipes / kitchens / users` 取空间名 + 创建者昵称，给公开接口附加上下文。
  - **后端 · Service（新文件 `share_token.go`）**：① `EnsureShareToken(ctx, userID, recipeID)` 复用 `GetByID` 的成员鉴权链路，幂等返回已有 token 或生成新 token（`crypto/rand` 18 字节 → base64url 截断 22 字符，约 132 位熵）；② `GetByShareToken(ctx, token)` 不做成员鉴权，直接通过 token 反查菜谱 + 元数据，元数据查询失败不致命（降级为空字符串以保证只读体验可用），返回 `PublicRecipeView { Recipe, KitchenName, CreatorName }`。
  - **后端 · 路由 + Handler**：`POST /api/recipes/{recipeID}/share-token` 挂在 `protected.Group`（仅成员可 ensure）；`GET /api/public/recipes/by-share-token/{token}` 挂在公开 `api.Group`（与已有 `/api/invites/{token}` 同层），完全绕过 `authMiddleware`。
  - **后端 · 测试 fixture 补字段**：`status_update_test.go:114` 与 `flowchart_worker_test.go:270` 两处内嵌 `CREATE TABLE recipes` 都补 `share_token` + `share_token_created_at` 两列默认值，避免现有测试因 schema drift 失败。
  - **后端 · 新增单测**（`share_token_test.go`）：4 个用例覆盖 ① EnsureShareToken 幂等性；② 非成员调用 EnsureShareToken 报错；③ GetByShareToken 返回完整空间名 + 创建者昵称；④ token 不存在返回 ErrNotFound。
  - **前端 · 新 API**（`utils/recipe-api.js`）：`ensureRecipeShareToken(recipeId)` 调 POST 接口取 token；`getRecipeByShareToken(token)` 调公开 GET 接口，**显式 `auth: false`** 跳过 Authorization header 注入。
  - **前端 · 新 store 包装**（`utils/recipe-store.js`）：`ensureRecipeShareTokenById` 仍走 `ensureSession`（成员才能 ensure）；`fetchPublicRecipeByShareToken` 不走 `ensureSession`，返回 `{ recipe, kitchenName, creatorName }`，recipe 经 `normalizeRecipe` 处理后与现有页面渲染口径一致。
  - **前端 · 详情页公开模式分支**（`pages/recipe-detail/index.vue`）：① `onLoad` 解析 `shareToken` query 参数，存在则设 `isPublicView = true`；② `loadRecipe` 加公开分支，**优先走 `fetchPublicRecipeByShareToken` 且不进缓存**（避免污染同 id 私有缓存），失败设 `publicViewLoadFailed = true`；③ `applyRecipe` 末尾新增 `ensureShareTokenIfNeeded`，私有模式下后台 fire-and-forget 静默 ensure token，**避免依赖微信 `onShareAppMessage.promise` 字段**（朋友圈基础库 3.12.0+ 才支持）；④ `buildRecipeShareConfig` 拼 path 时，若 `this.shareToken` 已就绪则附加 `&shareToken=xxx`，token 未就绪时退化为旧版链接（功能不退化，仅退化为「需登录成员」体验）。
  - **前端 · 只读 UI 收敛**：① 顶部固定 banner「来自『XX』的菜谱 · 加入空间可参与编辑」+「了解」按钮，加载成功后才显示，公开模式下 `detail-scroll` 顶部留出 76rpx；② Hero 右下 ⋯ 按钮、做法卡片右上 ⋯ 按钮、「生成流程图」按钮入口、底部整条编辑/置顶/删除 footer 全部 `v-if !isPublicView` 隐藏；③ 步骤打勾、复制食材、流程图横屏、图片预览等只读交互全部保留。
  - **前端 · 「了解」按钮 popup**：居中 popup 解释「这道菜由『创建者』整理，分享出来仅供查看。如果想一起编辑、调整步骤或补充心得，可以请对方把你加入空间」+「我知道了」按钮收起，遵循 Apple HIG 信息提示风格。
  - **前端 · 兜底空态**：missing-state 文案按 `isPublicView` 区分——公开失效为「分享链接已失效 / 这道菜谱可能已被删除或分享已收回 / 返回上一页」，私有未找到沿用旧文案；公开模式下 CTA 优先 `navigateBack`（用户期望「关掉这个失效页面」），降级 `reLaunch` 到首页。
  - **前端 · 公开模式选型**：已是该空间成员的人从分享卡进入也**统一走公开只读**，避免「先 ensureSession 再判空间 id 切回鉴权接口」的体验抖动；想编辑可从首页菜谱列表正常进入。
- **影响范围**：
  - 后端：`backend/migrations/018_*.sql`（新建）、`backend/internal/recipe/{model,repository,service,handler,share_token,share_token_test}.go`、`backend/internal/recipe/{status_update_test,flowchart_worker_test}.go`、`backend/internal/app/router.go`；新增公开路由 1 条 + 鉴权路由 1 条。
  - 前端：`utils/recipe-api.js`、`utils/recipe-store.js`、`pages/recipe-detail/index.vue`；无 npm 依赖变化。
- **兼容性·风险**：
  - migration 018 是 ADD COLUMN + 部分唯一索引，无破坏性；老菜谱 `share_token` 默认空字符串，部分唯一索引上 `share_token != ''` 的约束保证不冲突。
  - 老用户首次分享时 `this.shareToken` 还在后台 ensure 中，可能拿到旧版链接（不带 token），接收者仍会被鉴权墙挡住——这是已知的窗口期降级，进页 1-2 秒内 token 就会就绪，下次分享即正常；不阻塞主功能。
  - 公开接口完全不鉴权，**任何持有 token 的人都能看到完整菜谱（含个人备注 / 心得）**——这是产品上的明确取舍：分享出去就视为愿意公开。如果未来需要「指定接收者」级别隐私，需要重新设计 token 携带的接收者绑定信息。
  - 同空间成员从分享卡进入会强制只读，无法直接编辑——按需求决策结果是正向 UX，但需要在首页菜谱列表保留「正常进入即可编辑」入口才不引发用户困惑。
  - `FindKitchenAndCreatorMeta` 新增一次 JOIN 查询，增加一次 round-trip；只在公开接口路径触发，QPS 可控；元数据失败降级为空字符串保证菜谱正文仍可见。
- **验证情况**：
  - 后端：`go build ./...` 通过；`go test ./internal/recipe ./internal/kitchen` 全量通过；`share_token_test.go` 4 个新增用例全部通过（幂等性 / 非成员拒绝 / 元数据完整 / token 失效 NotFound）。
  - 前端：esbuild 对 `utils/recipe-api.js`、`utils/recipe-store.js`、详情页 `<script>` 块全部静态校验通过。
  - **未做真机联调**：建议手测三类场景：① A 用户分享 → A 自己点开（公开只读，看到 banner）；② A 分享 → 未登录新用户点开（不弹起登录，能看完整内容）；③ 删除菜谱后访问失效 token（看到「分享链接已失效」空态）；④ 老链接（不带 shareToken）兜底行为是否退化为旧版鉴权墙，符合预期。
  - **未实现的兜底**：token 主动失效 / 撤回机制未实现，如未来需要「设置过期 / 一键重置 token」需在 service 层加新接口；当前 token 永久有效（除非菜谱被删）。

## 2026-04-23 (P2-D Review 反馈修复：朋友圈开关 + 分享封面用可见首图)

### Fixed

- **修改时间**：2026-04-23
- **背景**：上一批 P2-D 「开启微信原生分享」的 review 中发现两个未爆雷缺陷，需在用户实际触发前修复。第三个 P1 缺陷（分享接收者无空间权限会打不开详情页）涉及鉴权策略与公开化方案权衡，归为下一轮单独迭代。
- **核心改动**：
  - **P1 · 朋友圈入口未生效**（`pages.json`）：上一批只新增了 `onShareTimeline` 生命周期函数，但**微信小程序的硬约束是必须同时在页面配置里开 `enableShareTimeline: true`**，朋友圈菜单项才会真正出现。当前 `pages/recipe-detail/index` 路由项未配置该字段，相当于上一批 CHANGELOG 写的「右上角出现分享到朋友圈」实际未交付。本次补全 `enableShareTimeline: true`；同时显式写出 `enableShareAppMessage: true`（虽默认即 true），便于后续 review 时分享开关在配置里集中可见。
  - **P2 · 分享封面用了已知失效的首图**（`pages/recipe-detail/index.vue`）：上一批 `buildRecipeShareConfig` 直接取 `recipeImages?.[0]` 作为封面候选，但该数组是原始图片列表，不会过滤掉加载失败被 `recipeImageHiddenMap` 标记的坏图。结果是页面本身已经避开了首张坏图，分享卡却继续把已知失效 URL 发给微信，分享出去就是空封面或加载失败提示。本次改为 `visibleRecipeSourceImages?.[0]`，复用页面已经算好的「已过滤坏图的可见原始 URL 列表」computed，与页面渲染口径完全一致。
- **影响范围**：`pages.json`、`pages/recipe-detail/index.vue`，无新增依赖，不影响其他页面与后端接口。
- **兼容性·风险**：纯配置与前端逻辑修复；朋友圈菜单从原本不出现变为出现，属预期修复；分享封面在大多数无坏图场景下行为不变，仅在首图加载失败时改为用第二张可见图（与页面显示口径一致）。
- **未交付（已记入下一轮）**：分享出去的 path 仍然走鉴权接口 `getRecipeById`，未登录或不在该空间的接收者会被 `ensureSession` + 后端成员校验挡住，看到的是 toast 而非有意义的兜底页。该问题本质是产品定位（私域 vs 公开）的取舍，需要单独评估「公开只读接口 + share_token」「邀请加入空间引导页」「分享时弹出确认框告知接收者范围」三类方案，不在本批次解决。
- **验证情况**：esbuild 静态语法校验通过；`pages.json` 解析通过（去注释后 JSON.parse 验证）；未做真机回归，建议手测：① 真机右上角胶囊确认「分享到朋友圈」菜单项现在能出现；② 故意让首图 URL 失效后再分享，确认拿到的封面是第二张可见图。

## 2026-04-23 (菜品详情页 P2-D：开启微信原生分享 / 朋友圈 / 收藏)

### Added

- **修改时间**：2026-04-23
- **背景**：菜品详情页此前完全没有定义 `onShareAppMessage` / `onShareTimeline` / `onAddToFavorites`，导致微信右上角胶囊菜单里**根本不显示「转发 / 分享到朋友圈 / 收藏」三项**——这是微信小程序的硬约束（必须先定义事件处理函数，菜单项才出现）。用户的分享意图被静默丢弃，是详情页最大的能力空白之一。
- **核心改动**（`pages/recipe-detail/index.vue`）：
  - 新增 3 个生命周期函数：`onShareAppMessage` / `onShareTimeline` / `onAddToFavorites`，分别对应微信好友转发、朋友圈、收藏夹。定义后右上角胶囊菜单自动出现这三项，无需任何 UI 改动。
  - 新增 `buildRecipeShareConfig({ channel })` 统一构造分享配置，按 `message` / `timeline` / `favorite` 三种渠道差异化处理 path / query / imageUrl 字段。
- **文案策略**（简洁派，让封面图说话）：
  - **微信好友转发**：有「完整做法」价值锚点（已生成流程图，或解析后步骤数 ≥ 3）时为 `{菜名} · 完整做法`，否则只用 `{菜名}`。明确告知接收方点开能看到结构化做法，比单纯菜名多一份点击动机。
  - **朋友圈**：只用 `{菜名}`，不加任何动作前缀。朋友圈是炫耀场，最克制的文字最高级，让封面图承担表达。
  - **收藏夹**：只用 `{菜名}`，便于在微信收藏夹中清单式识别。
  - **菜谱无标题（极端兜底）**：菜名退化为「一道值得做的菜」。
- **路径与归因**：转发 path 形如 `/pages/recipe-detail/index?id={recipeId}&from=share`，`from=share` 字段为后续埋点（区分自然访问 vs 分享拉来的）和分享拉新归因留口子；朋友圈不支持自定义 path，仅传 `query=id={recipeId}&from=share`，落地页固定为当前页。
- **图片**：按渠道差异化选封面，由微信端按比例（5:4 / 1:1）自适应裁切；无可用图时不传 `imageUrl`，微信自动截屏兜底。
  - **微信好友转发（5:4）**：优先 `flowchartImageUrl`（流程步骤图），缺则回退 `recipe.images[0]`（成品首图）。理由：转发场景的心智=「教你做菜」，流程图信息密度更高，朋友打开看一眼就能 get 完整做法。
  - **朋友圈（1:1）**：优先 `recipe.images[0]`，缺则回退流程图。理由：朋友圈是炫耀场，成品图远比流程图有传播力，心智=「我做了这个」。
  - **收藏（1:1）**：优先流程图，缺则首图。理由：收藏夹本质是「以后要用」，做法图比成品图实用。
- **影响范围**：菜品详情页 `pages/recipe-detail/index.vue`，无 UI 改动，无新增依赖，不影响其他页面与后端接口。
- **兼容性·风险**：纯前端能力新增，与已有功能正交，零回归风险；分享拉来的用户落到详情页时若未登录会沿用现有未登录态处理流程，本轮不变更。
- **后续可选项**：① 第二期可加自定义分享面板（底部 sheet），覆盖「生成海报 / 复制链接 / 保存图片」等微信原生菜单不支持的场景，需后端动态分享图配合（参考现成的 `backend/internal/invite/share_image.go`）；② 可基于 `from=share` 字段做分享拉新数据看板。
- **验证情况**：esbuild 静态语法校验通过；未做真机回归，建议手测：① 微信开发者工具中点击右上角胶囊菜单，确认「转发 / 分享到朋友圈 / 收藏」三项均出现；② 实际分享后核对标题、封面、跳转 path 是否符合预期；③ 无图菜谱也走一遍确认 imageUrl 不传时的兜底表现。

## 2026-04-23 (菜品详情页 P0 缺陷修复：无图回退 Tab + Hero 操作错位)

### Fixed

- **修改时间**：2026-04-23
- **背景**：在做 Hero 操作菜单与做法 Tab 合并的代码 review 中发现两个未爆雷但已存在的高风险缺陷，需在用户实际触发前先行修复。
- **核心改动**（`pages/recipe-detail/index.vue`）：
  - **P0 · 无图菜谱默认 Tab 落错**：`activeCookingTab` 初始值是 `'flowchart'`，原 watch 只处理「有图变无图」的回退，未覆盖「初次加载就无图」场景，导致进入页面命中流程图空态卡片，把真正的详细步骤和内嵌 CTA 全部挡住，违背 CHANGELOG 写过的「无图时直接展示详细步骤」承诺。
  - 新增 `ensureCookingTabValid` 方法：当 `!hasFlowchart && !isFlowchartActive` 时强制把 Tab 切到 `'steps'`；在 `applyRecipe`（菜谱数据落定）和两个 watcher（`hasFlowchart` / `isFlowchartActive`）里统一调用，覆盖首次加载、流程图被清空、生成任务结束三类时机。
  - **P0 · Hero「设为封面 / 删除」操作错图**：`setCurrentImageAsCover` 与 `deleteCurrentImage` 直接拿 `heroImageIndex`（属于「可见列表」`displayRecipeImages` 的索引）去 splice 原始 `recipeImages` 数组，但中间任意一张图加载失败被 `recipeImageHiddenMap` 隐藏后，两数组顺序错位，会**静默删错图或把错图设为封面**且立刻 `updateRecipeById` 落库。
  - 新增 `resolveOriginalImageIndex(visibleIndex)` 工具方法：通过 `cacheKey` 在 `recipeImages` 中反查真实下标，两个操作前都先做映射；映射失败返回 `-1`，由原有越界判断兜底 return。
- **影响范围**：菜品详情页 `pages/recipe-detail/index.vue`，无新增依赖；不影响其他页面与后端接口。
- **兼容性·风险**：纯前端逻辑修复，数据形态与 API 调用无变化。无图菜谱用户进入后看到的内容会从「空态」变成「详细步骤」，属预期修复；图片加载失败场景下的删/设封面行为从「可能错图」变成「正确目标图」。
- **验证情况**：esbuild 静态语法校验通过；未做真机回归，建议下一次手动覆盖：① 新建无图菜谱进入查看默认 Tab；② 多图菜谱中故意让中间一张图 URL 失效后，对后面的图执行设为封面/删除。

## 2026-04-23 (菜品详情页 Hero 操作菜单：设为封面 / 添加 / 删除)

### Added

- 菜品详情页首图（Hero）右上角新增 ⋯ 操作菜单（`pages/recipe-detail/index.vue`），让用户在浏览图片时就近完成 3 个高频操作，不必跳转编辑页：
  - **设为封面**：把当前查看的图片移到 `images[0]` 位（仅当 `heroImageIndex > 0` 且图片数量 > 1 时显示）
  - **添加更多图片**：复用现有 `chooseHeroImages`，仅当 `images.length < MAX_RECIPE_IMAGES` 时显示
  - **删除这张图**：删除当前查看的图片，需二次确认（`uni.showModal`，文案「删除后无法恢复，仍可重新上传」）
- 解决「拼图当封面」的根本问题之一：用户从小红书爬来的菜谱第一张往往不是最美的成品图，现在他可以滑到自己喜欢的那张直接「设为封面」，不必进编辑页拖拽
- 操作完成后：① `heroImageIndex` 自动校正（设为封面时回到 0 / 删除时防越界）② 触觉反馈（设为封面 medium、删除 light）③ ActionFeedback 顶部提示「已设为封面 / 已删除」

### Changed

- 新增 computed：`canShowHeroActionMenu` / `canSetCurrentAsCover` / `canAddMoreHeroImages` / `canDeleteCurrentImage`，按真实可执行性动态装配菜单项，避免 dead-click
- 新增 methods：`openHeroActionMenu` / `setCurrentImageAsCover` / `confirmDeleteCurrentImage` / `deleteCurrentImage`；统一使用 `updateRecipeById({ images })` 后端契约，无需新增接口
- 新增样式 `.hero-card__action`：56×56rpx 圆形半透明黑底 + `backdrop-filter: blur(10rpx)`，定位 `top: 22rpx; right: 22rpx`（避开底部蒙层的标题/分页器重读区，放在顶部更易够到且不抢戏）；按下态 `transform: scale(0.92)` + 加深背景

### Notes

- 修改时间：2026-04-23 18:10 CST
- 变更背景：用户在 Hero 区重构（蒙层 + 标题压图）完成后追问「设为封面」的实现方案。评估后舍弃「长按图片」方案（隐藏交互发现性差 + 与微信预览长按保存冲突）与「全屏预览页加底栏」方案（成本过高），采用「右上角常驻 ⋯ 菜单」方案，发现性高、与现有「做法卡片 ⋯ 菜单」设计语言一致、复用现有 `updateRecipeById` 后端能力
- 核心改动：模板新增 1 个 `<view>` 按钮（含 v-if 守卫）；script 新增 4 个 computed + 4 个 methods（含 1 个二次确认包装）；样式新增 ~30 行
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：① 上传中（`isUploadingHeroImage`）按钮整体隐藏，避免并发请求；② 设为封面/删除 都会调用 `updateRecipeById` 触发 `applyRecipe`，复用现有图片缓存逻辑；③ 删除有二次确认 + 错误兜底 toast；④ `@tap.stop` 阻止冒泡到 hero 整体的 `previewRecipeImage`，不会误触发大图预览；⑤ 按钮位置在右上而非右下，避开底部蒙层文字密集区，与「做法卡片 ⋯」一致放在卡片顶部右角
- 验证情况：esbuild 静态校验 `<script>` 通过；待真机回归 1) 单张图：⋯ 菜单仅含「添加更多图片」「删除这张图」（无「设为封面」）2) 滑到第 2+ 张：菜单顶部出现「设为封面」3) 点「设为封面」后是否立即切回第 0 位、是否震动反馈、是否显示「已设为封面」4) 点「删除」是否弹二次确认 5) 删除最后一张后是否回到上传占位态 6) 上传中点 ⋯ 是否被隐藏 7) 图片到达 `MAX_RECIPE_IMAGES` 上限时菜单中是否隐藏「添加更多」8) 网络失败时是否 toast 报错且不破坏现有数据

## 2026-04-23 (菜品详情页 Hero 区重构：标题压图 + 渐变蒙层 + 分页器升级)

### Changed

- 菜品详情页首图（Hero）区视觉与交互重构（`pages/recipe-detail/index.vue`），把「图片浏览器 + 标题区」整合为统一的「沉浸式封面」，提升首屏的情感钩子：
  - **H1 关闭 swiper 自动轮播**：从 `:autoplay="length > 1" :interval="3600"` 改为 `:autoplay="false" :duration="280"`。理由：菜谱场景下用户在「想吃」决策阶段需要反复看成品图，自动轮播打断思考；业界菜谱类（Yummly / Tasty / 下厨房）全部不自动轮播
  - **H2 移除「查看大图」chip**：图片本身已经全区域可点 `previewImage`，独立的「📷 查看大图」chip 是冗余功能可见性，造成视觉噪点
  - **H3 分页器升级**：原左下「1/8」灰底数字 chip → 底部居中「圆点指示器 dots」（≤5 张）或「数字 chip」（>5 张）。激活态 dot 从圆形拉伸为短横线（width 10rpx → 24rpx + border-radius 5rpx），符合 iOS Photos / 主流图库 App 的视觉语言
  - **H4 底部渐变蒙层**：新增 `.hero-card__overlay`，280rpx 高，从 0 → 0.85 黑色渐变，为压图标题/分页器提供任意背景色下的稳定读性
  - **H5 标题 + meta chips 压图**：有图时把 `mealLabel / statusLabel / 已置顶` chips + `recipe.title` 从 hero 下方迁移到 hero 内部底部（蒙层之上），节省 ~140rpx 垂直空间，让首屏装下「图 + 标题 + 摘要」三大信息块；无图时回退到原 `.detail-head` 布局
- Hero 高度从 380rpx 增至 520rpx（仅 `hero-card--with-overlay` 模式），为压图标题预留视觉空间

### Added

- 新增样式：
  - `.hero-card--with-overlay`：触发增高 + 压图模式
  - `.hero-card__overlay`：底部渐变蒙层
  - `.hero-card__title-block` / `.hero-card__title`（44rpx / 800 / `text-shadow`）：压图标题块
  - `.hero-card__meta` / `.hero-card__chip` / `.hero-card__chip--meal/--done/--wishlist/--pin` / `.hero-card__chip-text`：压图 chip（半透明深色 + 白字 + `backdrop-filter: blur`，确保任意背景下可读）
  - `.hero-card__pager` / `.hero-card__dot` / `.hero-card__dot--active`：底部居中圆点分页器
  - `.detail-head--summary-only`：有图时只渲染 summary 的紧凑变体（padding 28rpx → 18rpx）

### Removed

- 删除 `.hero-card__preview-tip` / `.hero-card__preview-tip-text` 样式与对应模板（「查看大图」chip）
- swiper 上的 `:interval="3600"` autoplay 配置

### Notes

- 修改时间：2026-04-23 17:30 CST
- 变更背景：用户对详情页首图区截图（番茄肉酱意面，封面是用户拼图含 8 张过程图）发起评审，识别出 5 类问题：① 自动轮播打断思考 ② 「查看大图」chip 冗余 ③ 「1/8」分页器与「查看大图」chip 视觉混淆 ④ 缺少底部蒙层导致压图文字难读 ⑤ 标题与图片分离造成首屏信息密度低。用户决策选择 Quick Win 三项 + Hero 渐变蒙层 + 标题压图共 5 项，深层「拼图当首图」的图片角色分类问题留待后续 P3
- 核心改动：模板层在 hero-card 内新增 overlay + title-block + pager 三个绝对定位层（z-index 2/3/3），并按 `displayRecipeImages.length` 控制压图模式与回退；样式层新增 ~140 行（新 chip 配色系统、dots 分页器、渐变蒙层），删除旧 preview-tip 相关样式
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：① 占位态（`!displayRecipeImages.length`）保留原 detail-head 布局，标题/chips 不会丢失；② Hero 高度从 380rpx 增至 520rpx 仅在有图时生效，可能轻微改变首屏滚动量，需真机回归是否合适；③ 压图 chip 使用 `backdrop-filter: blur(8rpx)`，部分低端安卓机可能不支持 blur，但半透明背景已能保证基本可读；④ swiper 的 `circular` 仍按图片数量条件保留；⑤ 「番茄肉酱意面」这种「拼图当封面」的内容仍会有信息过载问题，但有了渐变蒙层至少压图标题能读清，根本治理需后端引入 `imageRole` 字段（见 P3）；⑥ overlay 与 title-block 都加了 `@tap.stop="handleHeroCardTap"`，避免压图区域吞掉「点图预览大图」的能力
- 验证情况：esbuild 静态校验 `<script>` 通过；待真机回归 1) 单图：是否无分页器、压图标题正常显示 2) 2-5 张图：是否显示圆点 dots，激活态拉伸为短横线 3) >5 张图：是否显示数字 chip 4) 切换图片时分页器是否同步 5) 不再自动轮播（手动滑才动）6) 点击图片任意位置仍能预览大图 7) 占位态（无图）：是否回退到原 detail-head 布局、标题/chips/summary 都在 8) 压图标题/chips 在浅色食物图（白米饭、奶油意面）上的可读性 9) 整体首屏垂直空间感受是否更紧凑

## 2026-04-23 (菜品详情页 P2-B：详细步骤 Tab 体验升级 Batch 1+2)

### Changed

- 「详细步骤」Tab 多项产品/UX 优化（`pages/recipe-detail/index.vue`），从「展示导向」转为「使用导向」：
  - **B1-1 主料紧凑列表**：当主料 < 3 项时，去掉序号胶囊（如「1 多宝鱼 1条」的视觉冗余），改为「· 多宝鱼 1条」点状紧凑列表；≥ 3 项时仍保留序号胶囊
  - **B1-3 已完成态隐藏 banner**：「已自动整理」绿色横条仅在 `pending / processing / failed` 时显示，`done` 状态下隐藏（用户已经看到内容了，banner 是冗余信息）
  - **B1-4 Step 序号简化**：去掉「Step」英文前缀，仅保留数字「1 / 2 / 3」；胶囊从横向 999rpx 圆角改为 48×48rpx 正方形小徽章，让步骤标题真正成为视觉锚点
  - **B2-5 关键参数高亮**：步骤详情中的「数量+单位」「火候」「温度」用正则识别后加粗 + 暖橙色 `#b4664c`，让用户在厨房一手脏的状态下也能瞬间扫到「8分钟」「中火」等关键信息

### Added

- **B1-2 食材清单一键复制**：主料标题行右侧新增「复制清单」按钮（`canCopyIngredientList` / `copyIngredientList`），输出格式化文本：
  ```
  多宝鱼 · 食材清单

  主料：多宝鱼 1条
  配菜：红椒、油、蒸鱼豉油
  调味：葱、姜、盐
  ```
  覆盖「在地铁上看到菜想周末做，要把食材带去超市」的高频离线场景
- **B2-6 步骤打勾完成 + 本地持久化**：
  - 点击 step 卡片任意位置切换完成态；已完成步骤 `opacity: 0.55` + 标题加删除线 + 序号胶囊变绿底承载对勾图标
  - 制作步骤标题行右侧显示进度提示「2 / 4」+ 「重置」按钮（带二次确认）
  - 状态按 `recipeId` 隔离持久化到 `uni.setStorageSync`，key 前缀 `recipe-step-done:`；多菜谱并行做菜互不干扰
- 新增 utils：
  - `STEP_HIGHLIGHT_REGEX`：匹配数量+单位（分钟/秒/克/g/ml/勺/匙/杯/碗/条/个/片/块/颗/粒/根/瓣/只/滴/圈 等）+ 火候（大火/中火/小火/中小火/文火/武火/旺火/微火）+ 温度（°C / 度）
  - `highlightStepDetailText(detail)`：把步骤详情切成 `[{ text, highlight }]` 段落数组，用于模板循环渲染
  - `buildStepCompletedStorageKey(recipeId)`：按菜谱 ID 隔离的 storage key 构造器
- 新增 data：`completedStepIndexMap`
- 新增 computed：`canCopyIngredientList`、`completedStepCount`
- 新增 methods：`copyIngredientList`、`highlightStepDetail`、`isStepCompleted`、`toggleStepCompleted`、`resetCompletedSteps`、`loadCompletedSteps`、`persistCompletedSteps`
- 新增样式：`.parsed-section__head` / `.parsed-section__copy` / `.parsed-section__progress` / `.parsed-main-compact` 等 ~14 个新样式类；`.step-item--done` / `.step-item__index--done` / `.step-item__text--highlight` 等完成态与高亮态样式

### Notes

- 修改时间：2026-04-23 16:10 CST
- 变更背景：用户对 P2-A 完成后的「详细步骤」Tab 截图发起评审，识别出 8 项产品/UX 问题（主料序号冗余、缺购物清单导出、参数淹没、缺进度感、banner 占位过大、Step 胶囊喧宾夺主、缺烹饪模式、来源不可点击），用户决策选择执行 Batch 1+2（前 6 项中的高 ROI 项），P3 烹饪模式与可点击来源留待 P2-C
- 核心改动：模板层重构主料/辅料/制作步骤三个分组的结构与交互（标题行支持「标题 + 右侧操作」布局、步骤卡片支持点击切换完成、详情切片渲染高亮）；script 层新增 3 个 utils + 1 个 data + 2 个 computed + 7 个 methods，并在 `applyRecipe` 中集成本地状态加载；style 层新增 ~150 行
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：① `STEP_HIGHLIGHT_REGEX` 是全局正则，每次调用前已 reset `lastIndex`，不会污染下次调用；② 切片函数保证 0 段时也返回 `[{ text: '', highlight: false }]` 兜底；③ 步骤完成状态按 `recipeId` 隔离，多菜谱并行无干扰；④ `uni.setStorageSync` 失败仅 console.warn 不影响主流程；⑤ 整体改动是「分组级重构」，与上层 P2-A 的 Tab 切换逻辑完全解耦，不影响一图看懂 Tab；⑥ 「复制清单」按钮使用现有 `up-icon` 图标库，跨设备渲染一致
- 验证情况：esbuild 静态校验 `<script>` 通过；待真机回归 1) 主料 1 项时是否显示紧凑点状列表（无「1」胶囊）2) 主料 ≥ 3 项时是否回到序号胶囊样式 3) 「复制清单」点击后剪贴板内容是否包含主料 + 所有辅料分组 4) 「已自动整理」done 态是否隐藏 banner 5) Step 序号是否仅显示数字（无「Step」前缀）6) 步骤详情中「8分钟」「中火」「5g」等关键词是否被加粗高亮 7) 点击 step 卡片是否切换完成态、关页重开后状态是否保留 8) 「重置」是否弹二次确认 9) 切换菜谱后完成进度是否互不干扰

## 2026-04-23 (菜品详情页 P2-A：「一图看懂」与「做法整理」合并为 Tab 卡片)

### Changed

- 将原本两张独立的「一图看懂」与「做法整理」卡片合并为统一的「做法」卡片（`pages/recipe-detail/index.vue`），通过顶部 Tab 在两种视图之间切换：
  - **顶部 Tab 栏**：分段控制器风格（`.cooking-tabs`），仅在 `hasFlowchart` 为真时渲染；无图时直接展示「详细步骤」内容（无空 Tab）
  - **默认 Tab**：有图优先选中「一图看懂」，无图时仅显示「详细步骤」
  - **统一 ⋯ 菜单**（`openCookingMenu`）：合并原 `openFlowchartMenu` / `openParseMenu`，按可执行性动态装配三类操作 —— ① 重新生成一图看懂（`canRequestFlowchart && !isFlowchartActive`）② 重新整理步骤（`canRequestParse`）③ 查看生成详情（合并 flowchart + parse 来源信息）；全空时降级为 toast
  - **状态条与提示**：`flowchartStatusMeta` / `parseStatusMeta` / `showFlowchartStaleHint` 按当前 Tab 条件渲染，避免无关状态干扰
  - **底部合并 caption**（`.cooking-footer`）：根据当前 Tab 显示 `flowchartCaptionText`（AI 生成 · MM-DD）或 `parseStatusSourceLabel`，把元信息行从 2 行降为 1 行
  - **「详细步骤」Tab 内引导**：当无图但可生成时，在步骤内容上方插入弱主色 CTA `生成「一图看懂」流程图`（`.cooking-flowchart-cta`），把生成入口暴露在用户上下文里
  - **后台进行中状态**：`isCookingActive`（一图生成中 OR 步骤整理中）会把右上 ⋯ 替换为非交互 chip（`cookingActiveLabel`），跨 Tab 也能告知另一任务状态（如「一图生成中…」）

### Added

- `pages/recipe-detail/index.vue` 新增 data `activeCookingTab`（默认 `'flowchart'`）、computed `isCookingActive` / `cookingActiveLabel` / `hasCookingMenuItems` / `cookingFooterText`、watch `hasFlowchart`（图被清空时自动回退到 `steps` Tab）、methods `switchCookingTab(tab)` / `openCookingMenu()`
- 新增样式：`.detail-card--cooking`、`.cooking-tabs` / `.cooking-tabs__item` / `.cooking-tabs__item--active` / `.cooking-tabs__item--hover` / `.cooking-tabs__text`、`.cooking-flowchart-cta` / `.cooking-flowchart-cta__text` / `.cooking-flowchart-cta__arrow`、`.cooking-footer` / `.cooking-footer__text`
- 切 Tab 时调用 `uni.vibrateShort({ type: 'light' })`，与底部「横屏查看」胶囊触觉反馈一致

### Removed

- 移除已被取代的 methods `openFlowchartMenu` 与 `openParseMenu`（原模板中的两个 ⋯ 入口已经被统一的 `openCookingMenu` 替代，无外部调用方）

### Notes

- 修改时间：2026-04-23 14:30 CST
- 变更背景：用户在 P0/P1/P3 完成后选择执行 P2-A，目标是降低详情页的视觉密度（两张卡 → 一张卡 + Tab）、把同一「做法」概念在物理布局上聚拢，并合并冗余的 ⋯ 操作入口
- 核心改动：模板层将原 65~233 行两个 `detail-card` 合并为单个 `detail-card--cooking`，通过 `activeCookingTab` 控制内部三类区域（状态条 / hint / 内容区）的条件渲染；script 层新增 1 个 data、4 个 computed、1 个 watch、2 个 methods，并删除 2 个旧 methods；style 层新增 ~110 行
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：① 用户决策已对齐：未生成流程图时隐藏 Tab、仅展示「详细步骤」；② `watch.hasFlowchart` 在图被清空时把 Tab 回退到 `steps`，避免无图却选中空 Tab；③ 旧 `openFlowchartMenu`/`openParseMenu` 已无模板调用方，删除安全；④ 与现有 `parsedSteps` / `parsedMainIngredients` 等数据契约无变更，仅调整渲染父级
- 验证情况：esbuild 静态校验 `<script>` 通过；待真机回归 1) 有图：默认选「一图看懂」、Tab 切换是否流畅、底部 caption 是否随 Tab 切换 2) 无图：是否仅展示「详细步骤」内容（无 Tab、无空态）3) 无图但 `canRequestFlowchart`：步骤区上方是否出现「生成『一图看懂』」CTA 4) 后台生成中：⋯ 是否被 chip「生成中…」替换 5) 统一 ⋯ 菜单：仅暴露当前可执行项，全空时 toast 6) 流程图被清空（如生成失败）后是否自动切回「详细步骤」

## 2026-04-23 (菜品详情页 P0/P1 回归修复：覆盖确认死代码 + ⋯ 菜单 dead-click)

### Fixed

- 修复 `pages/recipe-detail/index.vue` 中「重新整理」轻确认分支为死代码的回归（CHANGELOG 与实际行为不一致）：
  - 旧逻辑：`needsParseOverwriteConfirm` 包含了 `hasMeaningfulParsedContent`，导致只要有整理结果就走「覆盖警告」弹窗，下方的「轻确认」分支永远不可达
  - 修复：将 `needsParseOverwriteConfirm` 收窄为仅 `hasManualParsedContentEdits`，让覆盖警告只在「真有手动改动」时弹出；纯 AI 结果走轻确认分支「重新整理？将再次调用 AI 整理食材与步骤，消耗 1 次额度。」；无结果直接执行
  - 影响：原本所有重新整理都看到「你手动修改过食材或制作步骤……」的吓人提示（即使没改过），现在恢复为按场景的合适措辞
- 修复一图看懂 ⋯ 菜单在「旧图仍可看 + 后台正在重生成 / 步骤不足」状态下出现 dead-click 的回归：
  - 旧逻辑：只要 `hasFlowchart` 就显示 ⋯ 菜单且固定包含「重新生成步骤图」；点击后 `handleGenerateFlowchart` 直接 `return`，UI 零反馈
  - 修复（多层防御）：
    1. 模板层新增 `isFlowchartActive` 判定，后台 `pending/processing` 时把 ⋯ 替换为非交互的「生成中…」chip，让用户一眼明白当前状态
    2. `openFlowchartMenu` 改为按 `canRequestFlowchart && !isFlowchartActive` 动态装配菜单项，无效动作不再暴露
    3. 兜底：若菜单项全空则降级为 toast `当前无可执行操作` 或 `先补充至少 3 个关键步骤`
- 同步加固「做法整理」`openParseMenu` 的菜单装配逻辑（按 `canRequestParse / parseStatusSourceLabel` 动态组装），保持两个 AI 卡片菜单行为一致

### Added

- `pages/recipe-detail/index.vue` 新增 computed `isFlowchartActive`（基于 `ACTIVE_FLOWCHART_STATUSES`）与样式 `.detail-card__status-chip`，作为「正在生成」状态的视觉锚点

### Notes

- 修改时间：2026-04-23 11:50 CST
- 变更背景：用户在 P0/P1 完成后做了二次评审，指出两个中风险问题：① 做法整理的轻确认分支因 `needsParseOverwriteConfirm` 条件过宽永远不可达；② 一图看懂 ⋯ 菜单在「旧图仍在 + 正在重生成」时仍暴露「重新生成」选项但点击无反馈
- 核心改动：收窄 `needsParseOverwriteConfirm` 语义；菜单按真实可执行性动态装配；模板层用「生成中…」chip 取代不可用的 ⋯ 按钮；菜单空态降级为 toast
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：仅前端逻辑与样式调整，未改后端契约；`hasManualParsedContentEdits` 与 `ACTIVE_FLOWCHART_STATUSES` 都是已存在的依赖；模板新增 `v-if/v-else-if` 条件互斥，不会同时渲染
- 验证情况：esbuild 静态校验 `<script>` 通过；待真机回归 1) 未手动改动时点重新整理是否弹「重新整理？……消耗 1 次额度」轻确认 2) 手动改过后是否弹覆盖警告 3) 后台生成中时是否显示「生成中…」chip 而非 ⋯ 4) 步骤不足时若旧图存在，⋯ 菜单是否只剩「查看生成详情」

## 2026-04-23 (流程图查看交互拆分：轻点预览 / 胶囊横屏)

### Changed

- 拆分菜品详情页一图看懂卡片的「查看流程图」交互为两个独立热区，与用户心智对齐：
  - **图片本体点击** → 调用 `uni.previewImage` 系统原生预览，支持双指缩放、左滑切换、长按保存菜单，零跳转、零等待
  - **右下角胶囊点击** → 跳转 `pages/flowchart-viewer` 横屏沉浸页，文案从「全屏查看 ›」改为更准确的「横屏查看 ›」
  - 胶囊按钮使用 `@tap.stop` 阻止冒泡，确保点击胶囊不会同时触发图片预览
  - `pages/recipe-detail/index.vue` 新增方法 `previewFlowchartImage`，移除外层 `flowchart-panel` 的整体 `@tap`/`hover-class`，由内部图片与胶囊各自承担 hover 反馈
- 微交互打磨：
  - 图片新增 `flowchart-panel__image--active` hover 态（按下时 `opacity: 0.92`），克制不抢戏
  - 胶囊新增 `flowchart-panel__cta--active` hover 态（`scale(0.96)` + 加深背景），明确「这是独立按钮」
  - 轻点图片时附带 `uni.vibrateShort({ type: 'light' })` 触觉反馈，强化「快速预览」的轻交互感

### Notes

- 修改时间：2026-04-23 11:05 CST
- 变更背景：用户反馈希望「轻点 = 快速看大图，主动选择 = 横屏沉浸」，符合移动端的预期；当前实现把整个图片绑定为「跳横屏」入口过于重，且与「全屏查看」浮层重复
- 核心改动：把图片本体与右下胶囊拆成两个独立点击热区，分别承担系统原生预览与横屏沉浸两种使用场景，并对齐文案与微交互反馈
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：仅修改前端交互；`uni.previewImage` 是小程序与 H5 通用 API，无新依赖；`@tap.stop` 在 uni-app 标准事件修饰符中支持；`uni.vibrateShort` 在不支持的端做了 `&&` 兜底
- 验证情况：esbuild 静态校验 `<script>` 通过；待在微信开发者工具与真机回归 1) 轻点图片是否进入系统预览 2) 点击右下胶囊是否仍跳横屏页且不触发预览 3) hover 反馈是否自然 4) 触觉反馈在不支持设备上是否安静降级

## 2026-04-23 (菜品详情页 P1+P3 体验优化：折叠菜单、信息层级、视觉规范)

### Changed

- 菜品详情页两个 AI 卡片右上角的「重新生成 / 重新整理」按钮，在已有产物时折叠为 `⋯` 图标按钮，降低消费场景下的视觉噪声：
  - `pages/recipe-detail/index.vue` 一图看懂卡片：当 `hasFlowchart` 为真时，原次级按钮替换为 `detail-card__icon-action` 图标按钮，点击弹出 `uni.showActionSheet`，含「重新生成步骤图」「查看生成详情」两项，详情通过 `uni.showModal` 展示完整时间与模型名
  - 做法整理卡片：当 `canRequestParse && hasMeaningfulParsedContent` 为真时同样折叠到 `⋯` 菜单，含「重新整理」「查看整理详情」
  - 首次未生成时仍保留主操作按钮，保持初次生成的入口可见性
- 一图看懂卡片底部元信息精简：
  - 移除「已生成：完整时间」「由 xxx-pro 生成」「双指缩放 / 拖动查看细节」共四行冗余文案
  - 改为单行 caption `AI 生成 · MM-DD`（新增 computed `flowchartCaptionText`，仅展示月-日，完整时间收到 `⋯` 菜单的「查看生成详情」）
  - 卡片可点 + 右下角「全屏查看 ›」浮层已经隐含「可放大」语义，无需重复手势提示
- 顶部引言区视觉减负：
  - 移除 `.detail-summary-card::after` 的右上角装饰大引号（截图反馈像孤立 bug）
  - 加深 `::before` 左侧色条颜色（从 `rgba(198,146,99,0.62)` 强化为 `#d1894f`），让「亮点提示」语义靠左色条独立承载
  - `detail-summary` 右内边距从 34rpx 收回 8rpx，文本展示更舒展
- Heading 层级规范化（对齐 ui-ux-pro-max 技能库的 Heading Hierarchy / Font Size Scale 规则）：
  - `.detail-card__title`：30rpx/700 → 32rpx/600，色值从 `#2f2923` 调到现代深色 `#1e293b`
  - `.parsed-section__title`：23rpx/700 → 24rpx/600，色值 `#76695d` → `#475569`，扫读层级更清晰
- 视觉规范化（P3）：
  - 卡片间距 `.detail-card { margin-top }` 由 `20rpx` 统一到 `24rpx`，对齐 8rpx 网格
  - `parsed-section--steps` 顶部间距由 `30rpx` 调到 `32rpx`，与 8rpx 网格一致
  - `.detail-scroll` 底部 padding 由 `188rpx` 调到 `200rpx`，给底部按钮栏留出更明显的呼吸缓冲

### Removed

- 清理一图看懂卡片底部已不再使用的样式类：`.flowchart-panel__meta-group / __meta / __credit / __credit-text / __preview-group / __preview / __preview-tip`，避免 dead code

### Notes

- 修改时间：2026-04-23 10:25 CST
- 变更背景：在 P0 修复转义符、按钮权重、二次确认之后，详情页仍存在 1) 详情页是消费场景但 AI 操作按钮过显眼 2) 一图看懂卡片底部 4 行元信息冗余、模型名对用户无价值 3) 标题层级粒度接近难以扫读 4) 顶部引言右上孤立装饰引号像 bug 等问题，结合 ui-ux-pro-max 的 Heading Hierarchy / Font Size Scale 规则一并优化
- 核心改动：AI 操作折叠为 `⋯` 菜单 + ActionSheet；元信息精简为单行 caption；顶部引言去引号、加深色条；标题字号/字重/字色统一；卡片间距规范化到 8rpx 网格；移除 7 个无用 CSS 类
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：仅涉及详情页前端模板与样式；新增 `openFlowchartMenu / openParseMenu` 两个方法均使用 `uni.showActionSheet + uni.showModal` 原生能力，无新依赖；首次未生成场景仍保留原主操作按钮路径，不影响新用户的初次生成入口
- 验证情况：esbuild 静态校验 `<script>` 语法通过；grep 确认旧字符 `&gt;` / 旧类名全部清除；待在微信开发者工具与真机回归 1) 已有产物时 `⋯` 菜单展开是否正常、二次确认是否串联 2) 顶部引言无右上引号 3) 卡片间距与字号梯度

## 2026-04-23 (菜品详情页 P0 体验修复：转义符、按钮权重、AI 二次确认)

### Fixed

- 修复 `pages/recipe-detail/index.vue` 流程图卡片右下角「全屏查看」浮层中的箭头字符在小程序渲染时被转义显示为 `&gt;` 的问题：
  - 将 `<text>` 内的 ASCII `>` 替换为 Unicode 单字符箭头 `›`，避开 `<text>` 节点对裸 `>` 的转义陷阱

### Changed

- 重排菜品详情页底部操作栏的视觉权重，建立明确的「危险 → 中性 → 主操作」三级梯度：
  - `删除` 由通栏文字按钮收紧为 `96rpx` 宽的图标按钮（`up-icon name="trash"`），并加细微红调描边以保留语义
  - `取消置顶 / 置顶` 维持中等宽度的次按钮风格
  - `编辑` 由 `flex: 1.16` 提升到 `flex: 1.6`，作为主 CTA 占据视觉重心
  - 收益：避免出现「删除最显眼、编辑反而像配角」的语义冲突，同时降低误删概率

### Added

- 为详情页两个 AI 操作补充统一的二次确认弹窗，避免误触消耗 AI 额度：
  - `handleParseAction`：在已有整理结果且非「覆盖手动修改」分支时，新增轻确认弹窗 `重新整理？将再次调用 AI 整理食材与步骤，消耗 1 次额度。`
  - `handleGenerateFlowchart`：在已有流程图（即「重新生成」语义）时，新增 `重新生成步骤图？将再次调用 AI 生成，消耗 1 次额度，约需 15 秒。` 弹窗，并将原生成主体抽出为 `submitFlowchartGeneration` 方法，确认通过后再发起请求

### Notes

- 修改时间：2026-04-23 09:40 CST
- 变更背景：用户反馈截图里出现 `&gt;` 转义字符（bug 级），底部三按钮的颜色/宽度梯度未体现「删除最弱、编辑最强」的语义，且「重新生成 / 重新整理」缺少二次确认易误触，结合 ui-ux-pro-max 技能库的 `Confirmation Dialogs (High)` 与 `Color Only (High)` 规则统一处理
- 核心改动：箭头字符替换为 Unicode `›`；底部按钮重排（删除收紧为图标按钮、编辑加宽为主 CTA）；两个 AI 入口分别补二次确认弹窗
- 影响范围：`pages/recipe-detail/index.vue`、`CHANGELOG.md`
- 兼容性/风险：仅影响详情页前端交互与样式；二次确认沿用 `uni.showModal`，无新依赖；`submitFlowchartGeneration` 是从 `handleGenerateFlowchart` 内联抽出，主流程逻辑不变；底部按钮宽度变化后请在 375px / 414px 真机上各看一次主按钮文字是否舒展
- 验证情况：已用 esbuild 静态校验 `<script>` 块语法通过；待在微信开发者工具与真机回归底部按钮布局、二次确认弹窗交互、流程图卡片箭头显示

## 2026-04-23 (流程图查看入口与横屏引导优化)

### Changed

- 优化菜品详情页流程图卡片的查看入口层级：
  - `pages/recipe-detail/index.vue` 中已有流程图时，顶部操作不再和“查看大图”抢主次，`重新生成` 回落为次级动作
  - 流程图缩略图本体新增悬浮式 `全屏查看` CTA，并把右下角纯提示文案改为 `双指缩放 / 拖动查看细节` 的手势提示，降低“只是说明文字”的误判
- 优化横屏查看页的首次上手反馈：
  - `pages/flowchart-viewer/index.vue` 新增一次性轻提示，首次进入时短暂展示“已进入横屏，双指缩放 · 拖动查看细节”
  - 左上角关闭入口扩为带 `返回详情` 文案的胶囊按钮，短暂显示后自动收成纯图标，兼顾发现性与沉浸感

### Notes

- 修改时间：2026-04-23 00:26 CST
- 变更背景：用户反馈流程图卡片里的 `横屏缩放查看` 更像技术说明，不像主操作；进入横屏查看页后，首次使用也缺少足够自然的手势与返回提示
- 核心改动：把“查看步骤图”从卡片页脚弱文案升级为图片上的显式 CTA，并在横屏查看页补充一次性轻引导和更稳的返回控件，让查看链路更符合“先点开、再放大、看完就退”的直觉
- 影响范围：`pages/recipe-detail/index.vue`、`pages/flowchart-viewer/index.vue`、`CHANGELOG.md`
- 兼容性/风险：本次仅调整前端交互与视觉层级；横屏查看页的提示显隐依赖本地存储与 `setTimeout`，仍建议在微信开发者工具和真机各看一次首开、二次进入和安全区表现
- 验证情况：已执行 `git diff --check`，并完成改动静态自检；待在微信开发者工具和真机补充交互回归

## 2026-04-22 (后台 AI 测试接口长耗时超时修复)

### Fixed

- 修复后台 `admin-web` 在测试 AI 路由 / 运行时配置时，可能被后端全局
  `30s` HTTP 超时提前截断的问题：
  - 新增 `backend/internal/middleware/timeout.go`，支持按请求方法 +
    路由前后缀覆盖请求超时，而不是整站只用单一超时值
  - `backend/internal/app/router.go` 保持大多数接口默认 `30s` 不变，仅将
    `POST /api/admin/ai-routing/scenes/{scene}/test` 与
    `POST /api/admin/runtime-settings/groups/{group}/test` 放宽到 `3m`
  - 补充 `backend/internal/middleware/timeout_test.go`，覆盖匹配测试路由与普通路由
    两种超时分支，避免后续再被全局超时误伤

### Notes

- 修改时间：2026-04-22 23:59 CST
- 变更背景：排查用户在后台配置 `https://api.42w.shop/v1` +
  `gpt-image-2-1536x1024` 时，发现服务器到上游网络连通正常，但后台测试按钮使用的
  真实流程图样例 prompt 实测耗时约 `64s`，被后端入口统一 `30s` 请求超时提前取消
- 核心改动：保留绝大多数 API 的默认超时，仅对后台两个“测试”接口按路由放宽时限，
  让 AI 生图/探活测试能够真正受配置超时控制，而不是先被网关层抢先超时
- 影响范围：`backend/internal/app/router.go`、
  `backend/internal/middleware/timeout.go`、
  `backend/internal/middleware/timeout_test.go`、`CHANGELOG.md`
- 兼容性/风险：后台测试接口允许更长时间占用连接；若后续继续把超长任务塞进同步 HTTP
  请求，仍建议评估异步化或更细粒度的场景超时策略
- 验证情况：已在当前环境复现实测
  `POST https://api.42w.shop/v1/chat/completions`，轻量测试约 `24.8s`，
  后台同款真实流程图样例约 `64.3s`；并执行 `go test ./internal/middleware`

## 2026-04-22 (流程图记录生成 model 与详情页溯源提示)

### Added

- 流程图结果新增生成来源落库能力：
  - 新增迁移 `backend/migrations/017_add_recipe_flowchart_generator.sql`
  - `recipes` 表补充 `flowchart_provider`、`flowchart_model` 字段，用于记录当前这张步骤图生成时实际发出的 provider / model
  - 流程图 worker 在成功写回图片时会同步保存本次请求的 provider / model，后续重新生成会覆盖为最新映射
- 菜品详情页 `pages/recipe-detail/index.vue` 的流程图卡片底部新增轻量 provenance 提示：
  - 在已生成时间下方以小胶囊形式展示 `由 {model} 生成`
  - 视觉上保持为次级信息，不抢主图和“横屏缩放查看”主操作的注意力

### Notes

- 修改时间：2026-04-22 23:41 CST
- 变更背景：用户希望后续每张流程图都能直接追溯到当时请求使用的 model，并在前端详情页以简单提示形式可见，而不是只能去 AI 任务审计页排查
- 核心改动：把流程图生成的 provider / model 从仅存在于 AI 审计记录，扩展为同步保存到菜谱主记录，并在详情页补一个轻量的“由 {model} 生成”溯源提示
- 影响范围：`backend/migrations/017_add_recipe_flowchart_generator.sql`、
  `backend/internal/recipe/*.go`、`pages/recipe-detail/index.vue`、
  `utils/recipe-store.js`、`CHANGELOG.md`
- 兼容性/风险：旧数据默认没有 `flowchart_model`，详情页只会不展示该提示，不影响已有流程图查看；当前按用户要求记录的是“请求时发出的 model”，不是上游网关回显的真实执行模型
- 验证情况：已执行 `go test ./internal/recipe`、`go test ./internal/airouter`、
  `go test ./internal/upload`，并完成详情页样式静态自检

## 2026-04-22 (流程图中转站 `message.images` 兼容修复)

### Fixed

- 补齐 OpenAI-compatible 流程图生图对 `message.images` 返回格式的兼容：
  - `backend/internal/airouter/service.go` 在 `flowchart` 场景下优先读取
    `choices[0].message.images[*].image_url.url`，不再只依赖 `message.content`
  - `backend/internal/recipe/flowchart.go` 同步支持从 `images` 数组提取图片引用，
    并允许识别 `data:image/...;base64,...` 形式的图片结果
  - `backend/internal/upload/service.go` 新增对 base64 `data:` 图片地址的落盘支持，
    避免中转站返回内嵌图片时流程卡在“无法下载远端 URL”
- 新增测试覆盖：
  - `backend/internal/airouter/service_test.go` 验证 `flowchart` 场景可消费
    `message.images`
  - `backend/internal/recipe/flowchart_test.go` 验证流程图图片提取支持 `data:` URL
  - `backend/internal/upload/service_test.go` 验证上传服务可直接保存 base64 图片

### Notes

- 修改时间：2026-04-22 23:12 CST
- 变更背景：实测新的流程图中转站在 `chat/completions` 下返回的是
  `message.images[0].image_url.url = data:image/...;base64,...`，而不是仓库原先假设的
  Markdown 图片或公网 URL，导致流程图生成链路实际不兼容
- 核心改动：后端流程图链路统一补齐 `message.images` + `data:` 图片兼容，确保 AI
  返回内嵌图片时也能进入现有上传、存储和前端展示流程
- 影响范围：`backend/internal/airouter/service.go`、
  `backend/internal/recipe/flowchart.go`、
  `backend/internal/upload/service.go`、对应测试文件、`CHANGELOG.md`
- 兼容性/风险：当前仍按配置项记录 provider/model；若中转站内部把 `gpt-image-2`
  映射到别的真实模型，后台观测里展示的仍是配置模型名，和上游最终执行模型可能存在偏差
- 验证情况：已用实际中转站 `https://api.42w.shop/v1` + `gpt-image-2` 做真实流程图生图，
  并执行后端相关单元测试确认 `message.images` / `data:` 图片链路可用

## 2026-04-20 (抖音 Provider POC 设计稿)

### Added

- 新增 `backend/docs/douyin-provider-poc.md`，整理当前仓库接入抖音链接提取能力的
  POC 设计方案：
  - 基于现有 `linkparse-sidecar` 架构评估 `F2 / yt-dlp / you-get` 等开源方案，
    明确 `F2` 更适合作为第一版抖音 provider
  - 明确第一期目标是补 `POST /v1/parse/douyin`、提取视频/原声/文案并复用现有
    `ffmpeg + ASR` 转写链路，而不是重做独立解析总线
  - 约定 Douyin provider 的返回字段、配置项、错误模型、后端接线点和分阶段落地
    路线，并额外指出现有 sidecar 需要补 `audioUrls` 等通用字段扩展

### Notes

- 修改时间：2026-04-20 11:58 CST
- 变更背景：用户希望评估并落一版抖音链接提取能力的 POC 设计文档，用于后续支持
  音频提取、文字版整理和现有菜谱自动解析链路的抖音接入
- 核心改动：补充 Douyin provider 的设计文档，统一记录开源方案取舍、sidecar /
  backend 接口约束、转写复用策略和推荐实现顺序
- 影响范围：`backend/docs/douyin-provider-poc.md`、`CHANGELOG.md`
- 兼容性/风险：当前仅为设计稿，尚未真正接入抖音 provider；文档中推荐把
  `XHS_TRANSCRIPT_*` 逐步抽象为更通用的 `LINKPARSE_TRANSCRIPT_*`，正式实现时需
  保留兼容回退，避免影响现有小红书转写链路
- 验证情况：已执行文档静态自检；待后续真实实现时补充接口测试与联调验证

## 2026-04-18 (概览热点分布改为排行榜主视图)

### Changed

- 后台概览页 `admin-web/src/pages/DashboardPage.vue` 调整三张热点分布卡的默认信息结构：
  - `按场景分布` 继续保留图表默认，用于观察整体任务分布
  - `Provider 热点` 与 `Model 热点` 改为默认显示「排行」视图，不再默认展示成功/失败堆叠条
- `Provider / Model` 排行榜新增更明确的数值锚点：
  - 每行同时展示排名、名称、调用量数字 + 迷你条、成功率进度条 + 百分比
  - 成功率低于阈值时补充告警图标，优先暴露高调用低成功率项
- 长名称从尾部截断改为中间省略，保留模型 / Provider 后缀辨识度；`(empty)` 归并显示为 `未指定`，并在排行榜中固定后置，避免直接参与热点心智排序
- 图表模式同步复用 `未指定` 命名口径，并改为中间省略标签，避免再次出现后缀信息被截断
- 三张分布卡进一步做了一轮布局收口：
  - `按场景分布` 也切到与右侧一致的 `排行 / 图表` 结构，不再单独保留表格样式
  - 右上角视图切换器改为固定单行，计数胶囊独立放置，避免中等宽度下按钮换行导致头部发散
  - 三张卡的副标题文案、头部节奏和排行行高进一步统一，减少“左边像表格、右边像榜单”的拼贴感

### Notes

- 修改时间：2026-04-18 16:07 CST
- 变更背景：用户指出当前热点卡片把“调用量”和“成功率”叠在同一根横向堆叠条里，语义不够直观，且缺少数值锚点、长名称尾缀辨识度不足、`(empty)` 混入排序
- 核心改动：把 Provider / Model 从图表优先切到排行榜优先，并继续统一三张分布卡的头部布局、切换器节奏和排行骨架
- 影响范围：`admin-web/src/pages/DashboardPage.vue`、`CHANGELOG.md`
- 兼容性/风险：仅涉及后台前端概览展示；排行榜行列在窄屏下依赖 CSS grid 自适应，仍建议在真实浏览器里再看一次 3 卡并排和窄宽度换行效果
- 验证情况：已执行 `npm --prefix admin-web run build`

## 2026-04-18 (菜品流程图横屏沉浸查看)

### Added

- 菜品详情页的流程图改为支持独立横屏沉浸查看：
  - `pages/recipe-detail/index.vue` 的流程图卡片入口从原来的单图预览切到专用查看页，入口文案同步改为“横屏缩放查看”
  - 新增 `pages/flowchart-viewer/index.vue`，仅承接步骤图查看，收口为“满屏图片 + 极轻关闭按钮”的横屏预览态，并基于 `movable-view` 打开双指缩放与拖拽
  - 横屏查看页左上角关闭按钮进一步缩小，并同步降低背景、描边和阴影权重，避免在纯图片场景里喧宾夺主
  - `pages.json` 为该查看页单独开启 `pageOrientation: landscape` 与 `disableScroll`，横屏能力只作用于流程图查看链路，不影响原详情页竖屏布局

### Notes

- 修改时间：2026-04-18 16:06 CST
- 变更背景：当前 AI 生成的步骤图默认是横版 `16:9`，直接在竖屏详情卡片里只能看缩略图；用户明确希望只对流程图提供沉浸式横屏查看，而不是把整个菜品详情页改成横屏版
- 核心改动：新增流程图专用横屏查看页，并把详情页流程图点击行为切换为跳转该页；查看页会保持亮屏，并支持双指缩放、拖拽查看，优先保证“打开即横屏、只看图、不打扰”
- 影响范围：`pages/recipe-detail/index.vue`、`pages/flowchart-viewer/index.vue`、`pages.json`、`CHANGELOG.md`
- 兼容性/风险：横屏能力依赖微信小程序 `pageOrientation`；需在微信开发者工具和真机确认自定义导航、刘海屏安全区和返回手势表现是否符合预期
- 验证情况：已执行 `git diff --check`

## 2026-04-18 (概览最近失败支持跳转详情)

### Changed

- 后台概览页 `admin-web/src/pages/DashboardPage.vue` 的「最近失败」卡片改为显式详情入口：
  - 场景列内直接展示「排障详情」链接，不再依赖表格最右侧操作列，避免窄卡片下入口被横向滚动隐藏
  - 开始时间收拢到场景列辅助信息中，并在卡片头部补充「查看 AI 任务」快捷入口
  - 点击记录后会带上当前概览时间窗、场景、状态和 `jobId` 跳转到 `AI 任务` 页
- `admin-web/src/pages/JobsPage.vue` 新增 `jobId` 路由深链能力：
  - 从概览跳入 `AI 任务` 页后，会自动拉起对应任务详情抽屉
  - 关闭详情抽屉时同步清理 URL 中的 `jobId`，保持列表筛选状态和地址栏一致

### Notes

- 修改时间：2026-04-18 15:40 CST
- 变更背景：用户反馈概览页「最近失败」虽然存在详情能力，但入口位于表格最右侧，实际使用时不易发现，也不适合直接从概览进入排障链路
- 核心改动：把概览失败记录改成可见的详情跳转入口，并补齐 `AI 任务` 页的详情深链承接
- 影响范围：`admin-web/src/pages/DashboardPage.vue`、`admin-web/src/pages/JobsPage.vue`、`CHANGELOG.md`
- 兼容性/风险：仅涉及后台前端交互；概览跳转到 `AI 任务` 页时会把当前时间窗换算成绝对时间区间写入 URL，若用户长时间停留后再刷新，列表会按跳转时刻的时间窗回放
- 验证情况：已执行 `npm --prefix admin-web run build`

## 2026-04-18 (AI Provider 管理页二次 UX 迭代落地)

### Changed

- `admin-web/src/pages/AIProvidersPage.vue` 按 `docs/admin-ai-provider-ux-review.md` §8 四阶段落地：
  - **阶段 1**：工具栏分组 + divider + 保存场景 dirty 红点；放弃草稿改 danger link + 动态 tooltip；新增 `.routing-breadcrumb` 编辑上下文（暴露 `--routing-breadcrumb-h`）；三大数值字段补推荐区间 hint 与越界 warn，新增 `.routing-timeline-hint` 重试/熔断时序条 + 预计首轮最长耗时预览。
  - **阶段 2**：新增 `sceneDiff` / `diffCount` 精细化差异计数（scene 字段 + provider id 粒度；排序变化单独记为 `providers.order`），驱动放弃草稿 tooltip 与面包屑「N 项未保存」；`effectiveChannel` 统一三场景「新路由 / 兼容 / 草稿」链路文案（`{scene}-v2|compat|draft`），面包屑右侧加 3×3 状态矩阵 popover。
  - **阶段 3**：`el-alert` 升级为 `show-icon` + 右侧「前往配置」按钮；告警条先回退为静态但真实的入口提示，不再基于当前后端未支持的 `alert.fire/since` mock 数据展示“最近 1h 告警数”；`goAlertConfig` 修正为跳 `/settings?group=ai.provider_alert#ai-provider-alert`。
  - **阶段 4**：策略面板加 `.routing-panel--strategy` + `@media(min-width:1200px)` sticky；`collapsedProviderKeys` 按 localKey 维度管理，`>3` 节点默认仅展开首个，每张节点卡前置折叠按钮，折叠态复用 meta 行展示 Model 等关键信息。

### Notes

- 修改时间：2026-04-18 15:18 CST
- 影响范围：`admin-web/src/pages/AIProvidersPage.vue`、`docs/admin-ai-provider-ux-review.md`、`CHANGELOG.md`
- 影响范围补充：`admin-web/src/pages/SettingsPage.vue`
- 后端依赖：暂未新增接口；真实告警 summary 与 `listAIRoutingScenes` 的 `recentStats` 待单独 issue 跟进
- 兼容性/风险：sticky + 折叠面板在 Safari 下需真机回归 `overflow:auto` 行为；告警条右侧绝对定位按钮在 720px 以下回落为静态布局
- 验证情况：已执行 `npm --prefix admin-web run build`

## 2026-04-18 (文档与初始化脚本口径同步)

### Changed

- 同步后台相关文档到当前已落地实现：
  - `README.md` 的后台页面清单补上 `AI Provider`
  - `backend/README.md` 补齐 `AI Routing`、`invite-codes`、`auth/profile`、
    `recipes/{id}/pin` 等已存在接口，并补充 `/api` 与 `/caipu-api` 的访问口径说明
  - `docs/admin-console-ai-observability-design.md` 从早期设计草案口径更新为当前
    实现口径，修正后台认证、配置中心、服务健康和 `AI Provider` 路由描述
  - `backend/scripts/bootstrap-server.sh` 默认切到共享域名前缀模式，新增
    `NGINX_SITE_MODE`、`API_PREFIX`、`UPLOADS_PREFIX`、`HEALTHZ_PATH` 和
    `ROOT_PROXY_PASS` 参数，并修复 `ADMIN_WEB_DIR` 未传入远端 shell 的问题
  - `docs/backend-deploy-quickstart.md` 改为和脚本保持同一默认口径，同时说明
    `standalone` 兼容模式的使用方式

### Notes

- 修改时间：2026-04-18 13:56 CST
- 变更背景：排查 `admin-web` 与 `backend` 是否同步时，发现接口契约本身已对齐，
  但部分文档仍停留在旧设计稿或早期部署口径，容易误导后续开发和部署操作
- 核心改动：统一后台接口清单、页面说明和部署前缀口径，并让初始化脚本默认对齐
  共享域名现网方案，同时保留显式 `standalone` 兼容模式
- 影响范围：`README.md`、`backend/README.md`、
  `backend/scripts/bootstrap-server.sh`、
  `docs/admin-console-ai-observability-design.md`、
  `docs/backend-deploy-quickstart.md`、`CHANGELOG.md`
- 兼容性/风险：初始化脚本默认生成的 nginx 路由与旧版不同；如果目标域名需要继续
  走独占站点 `/api` 口径，必须显式带 `NGINX_SITE_MODE=standalone`
- 验证情况：已人工对照 `backend/internal/app/router.go`、
  `admin-web/src/api/admin.ts`、`docs/cloud-server-config-overview.md`；
  脚本将通过 `bash -n` 做语法检查

## 2026-04-18 (AI Provider 页面卡顿修复)

### Fixed

- 修复后台 `AI Provider` 页面在已登录状态下可能直接卡住的问题：
  - `admin-web/src/pages/AIProvidersPage.vue` 的场景卡片 `ref` 回调不再写入
    响应式对象，避免渲染阶段反复触发组件更新，导致页面线程卡死
  - 当前场景详情加载完成后改为异步拉取路由审计，不再让审计请求阻塞首屏内容
    展示

### Notes

- 修改时间：2026-04-18 13:52 CST
- 变更背景：用户反馈后台其他页面可正常打开，但 `AI Provider` 页面会单独卡住；
  排查发现接口本身返回正常，问题集中在该页前端渲染链路
- 核心改动：移除模板 `ref` 回调里的响应式写操作，并让审计列表改为后台异步刷新
- 影响范围：`admin-web/src/pages/AIProvidersPage.vue`、`CHANGELOG.md`
- 兼容性/风险：仅影响后台前端页面交互，不改后端 API；审计表会在场景卡片之后异步补齐
- 验证情况：已执行 `npm --prefix admin-web run build`

## 2026-04-18 (AI Provider P0 收口)

### Changed

- `admin-web/src/pages/AIProvidersPage.vue` 按 `docs/admin-ai-provider-ux-review.md`
  继续收口 P0 体验问题：
  - 顶部工具栏新增“放弃草稿”和“最近测试结果”快捷入口，测试结果可一键滚动到详情卡
  - 未保存草稿保护统一覆盖刷新、场景切换、页面离开和浏览器关闭；场景卡补齐
    `tab` 语义与左右方向键切换
  - 放弃草稿时同步清空过期测试结果；方向键切换改为始终基于当前激活场景卡，
    切换后自动把焦点移到新卡片，避免键盘导航停在旧节点
  - Provider 节点操作区改为图标按钮 + Tooltip + 更多菜单，删除移入更多菜单，
    并补充拖拽排序与复制节点
  - API Key 编辑改为“当前密钥 chip + 更换/清空”两段式交互，去掉 masked
    placeholder，清空密钥改为二次确认后仅标记待保存

### Notes

- 修改时间：2026-04-18 10:56 CST
- 变更背景：AI Provider 管理页已经有基础的多 Provider 编辑能力，但保存 / 测试 /
  草稿边界、Provider 卡操作密度和 API Key 误操作风险仍是 P0 级体验硬伤，
  需要按 UX 评审文档继续收口
- 核心改动：统一未保存草稿守卫；把最近测试反馈前置到顶部；收敛 Provider 卡
  高频操作；重做 API Key 替换与清空链路，降低误触风险
- 影响范围：`admin-web/src/pages/AIProvidersPage.vue`、`CHANGELOG.md`
- 兼容性/风险：仅涉及后台前端交互层，不改后端接口；拖拽排序与顶部测试摘要仍建议在
  浏览器内做一次人工回归，确认鼠标拖放和滚动定位手感符合预期
- 验证情况：已执行 `npm --prefix admin-web run build`；当前 `admin-web`
  依赖中未安装 `vue-tsc`，因此本次未做独立的 SFC 类型检查

## 2026-04-18 (概览分布图表修复)

### Fixed

- 修复后台概览页三张分布图在三列卡片布局下串位越界的问题：
  - `admin-web/src/pages/DashboardPage.vue` 的分布图改为固定左侧标签预留宽度，
    不再依赖 `containLabel` 在窄卡片里动态撑开画布
  - 三张分布图卡片新增 `min-width: 0` 与 `overflow: hidden` 约束，避免 ECharts
    canvas 宽度异常时串到相邻卡片
  - `distribution-chart` 容器同步补齐 `max-width` / `overflow` 保护，保证图表只在
    当前卡片内渲染

### Notes

- 修改时间：2026-04-18 04:03 CST
- 变更背景：上一轮概览页新增图表 / 表格切换后，用户反馈三张分布图在桌面端
  三列布局下会互相覆盖，条形图越过卡片边界，导致页面可读性明显下降
- 核心改动：收紧分布图的 ECharts 网格与标签宽度策略，并补足卡片与图表容器的
  宽度/溢出保护，优先保证三列卡片场景下布局稳定
- 影响范围：`admin-web/src/pages/DashboardPage.vue`、`CHANGELOG.md`
- 兼容性/风险：仅涉及后台概览页图表展示；标签宽度改为固定值后，超长名称会更早
  被截断，但 tooltip 仍保留完整数据说明
- 验证情况：已执行 `npm --prefix admin-web run build` 做前端构建验证；建议上线后
  再对 `/admin/` 概览页做一次桌面端人工复核

## 2026-04-18 (概览分布图表视图)

### Added

- 后台概览页「按场景分布 / Provider 热点 / Model 热点」三张卡片新增「图表 / 表格」切换：
  - 默认图表模式，横向堆叠条形图按总数升序排列，绿色为成功调用量、红色为失败调用量，
    一眼可读出「谁调用多 + 谁成功率低」的组合信息
  - Tooltip 同步展示总数、成功率（按阈值着色）、成功 / 失败拆分
  - 切回表格即恢复原有 `TotalCell` / `RateCell` 可视化占比条行为
  - 切换或窗口 resize 时复用已有 ECharts 实例，卸载 / 切表时主动 `dispose()` 释放

### Notes

- 修改时间：2026-04-18 04:45 CST
- 变更背景：用户反馈仅表格承载「总数 + 成功率」两维信息不够直观，希望有图表视图
- 核心改动：`admin-web/src/pages/DashboardPage.vue` 新增 `distViewMode` 状态、三路
  ECharts 实例、`renderDistributionChart()` 堆叠横条渲染与容器变更时的重建逻辑
- 影响范围：`admin-web/src/pages/DashboardPage.vue`
- 兼容性/风险：纯前端可视化变更，不动后端接口；ECharts 已在此页常驻引入，打包体积无
  新增依赖
- 验证情况：`npx vue-tsc --noEmit` 通过；建议在浏览器内切换「图表 / 表格」并触发窗口
  resize 确认图表自适应正常

## 2026-04-18 (仓库清理)

### Changed

- 仓库忽略规则补充本地工具与构建缓存文件：
  - `.gitignore` 新增 `.claude/`、`CLAUDE.md` 与 `*.tsbuildinfo`
  - 避免 Claude 本地权限配置、协作文档草稿和 TypeScript 增量构建缓存持续出现在
    `git status` 中，误混入业务提交

### Notes

- 修改时间：2026-04-18 04:07 CST
- 变更背景：本地工作区中存在 `.claude/settings.local.json`、`CLAUDE.md` 和
  `admin-web/tsconfig.app.tsbuildinfo` 这类本地工具/缓存文件，不适合继续作为
  未跟踪改动长期暴露在仓库状态里
- 核心改动：补齐 `.gitignore`，把仅对本地协作工具或增量构建有效的文件排除出
  版本控制
- 影响范围：`.gitignore`、`CHANGELOG.md`
- 兼容性/风险：仅影响 Git 跟踪规则，不改运行时逻辑；若未来确实需要版本化
  `CLAUDE.md`，可再显式调整忽略规则
- 验证情况：将通过 `git status --short` 确认相关未跟踪文件不再显示

## 2026-04-18 (概览视觉补丁)

### Changed

- 后台概览页的场景 / Provider / Model 热点表，`总数` 列改为带横向占比条的
  可视化展示：
  - `admin-web/src/pages/DashboardPage.vue` 新增 `TotalCell`，按当前表格内
    最大值计算相对宽度
  - 三张热点表的 `总数` 列从纯数字改为“进度条 + 数值”，提升热点分布对比效率
  - `成功率` 列最小宽度同步略微放宽，避免新布局下数值与条形视觉过于拥挤

### Notes

- 修改时间：2026-04-18 04:02 CST
- 变更背景：概览页热点表此前只有纯数字，用户需要在场景、Provider 和 Model
  间快速比较调用量时，需要逐行扫数值，缺少直观的体量对比
- 核心改动：为热点表引入轻量条形对比视觉，不改变后端接口与数据结构，仅优化
  后台概览页的信息密度和扫读效率
- 影响范围：`admin-web/src/pages/DashboardPage.vue`、`CHANGELOG.md`
- 兼容性/风险：仅涉及后台概览页前端展示；条形宽度按当前表格最大 `total`
  相对计算，当所有值都为 `0` 时会自动回退为 `0%`
- 验证情况：已执行 `npm --prefix admin-web run build`

## 2026-04-18 (文档补充)

### Changed

- 部署文档补充“Mac 本地发起远端后端重部署”的明确说明：
  - `README.md` 现在显式区分“Mac 本地运行
    `backend/scripts/deploy-server-build.sh`”与“已登录服务器后运行
    `scripts/deploy-backend-on-server.sh`”两种场景
  - `docs/backend-deploy-quickstart.md` 补充脚本角色对照、`SERVER_HOST`
    可写 `user@host` 或 `~/.ssh/config` 主机别名的说明，并把 `my-cloud`
    明确标注为示例别名；同时补充 `PLAN_ONLY=1` 预检查命令
  - 文档同步说明远端默认会进入 `/srv/caipu-miniapp`，并补充 Go 路径兜底说明，
    避免误以为本地直接运行根目录脚本也会自动发起 `ssh`

### Notes

- 修改时间：2026-04-18 03:52 CST
- 变更背景：上一轮已修复远端后端部署脚本在非交互式 `ssh` shell 下找不到
  Go 的问题，但当前 `README.md` 对“本地发起远端部署”和“已登录服务器后在
  机器内执行部署脚本”两类场景区分还不够直白，容易让使用者误判入口脚本
- 核心改动：补齐部署入口说明、预检查示例和默认远端工作目录说明，统一当前
  线上 `/srv/caipu-miniapp` 的实际口径
- 影响范围：`README.md`、`docs/backend-deploy-quickstart.md`、
  `CHANGELOG.md`
- 兼容性/风险：仅文档澄清，不改实际部署逻辑
- 验证情况：已对照 `backend/scripts/deploy-server-build.sh`、
  `scripts/deploy-backend-on-server.sh` 与现网 `/srv/caipu-miniapp` 部署路径
  做代码级复核

## 2026-04-18 (部署补丁)

### Fixed

- 远端 `backend` 发布脚本补齐 Go 可执行文件兜底路径：
  - `scripts/deploy-on-server.sh` 新增 `GO_BIN_DIR`，默认指向
    `/usr/local/go/bin`
  - 当非交互式 `ssh` shell 的 `PATH` 未包含 Go，但 `${GO_BIN_DIR}/go`
    存在时，脚本会自动补上 PATH 后再执行 `go build`
  - 避免从 Mac 本地通过 `backend/scripts/deploy-server-build.sh` 发起远端
    发布时，服务器明明已安装 Go 却报 `env: ‘go’: No such file or directory`

### Notes

- 修改时间：2026-04-18 03:40 CST
- 变更背景：本次从 Mac 本地触发远端 `backend` 重部署时，`ssh` 非交互 shell
  的默认 `PATH` 不包含 `/usr/local/go/bin`，导致线上机虽然已安装 Go，部署脚本
  仍在 `go build` 阶段失败
- 核心改动：为服务器端部署脚本增加 Go 路径自动兜底，保持现有
  `deploy-server-build -> deploy-backend-on-server -> deploy-on-server`
  链路不变，仅修复非交互 shell 找不到 Go 的问题
- 影响范围：`scripts/deploy-on-server.sh`、`CHANGELOG.md`
- 兼容性/风险：默认仍优先使用当前 `PATH` 中的 `go`；只有未找到时才回退到
  `GO_BIN_DIR`，如未来服务器 Go 安装目录不同，可显式覆盖该环境变量
- 验证情况：已通过 `ssh my-cloud 'cd /srv/caipu-miniapp && PATH=/usr/local/go/bin:$PATH bash scripts/deploy-backend-on-server.sh'`
  成功完成一次真实后端重部署，并确认 `caipu-backend` 服务正常拉起

## 2026-04-18 (晚间补丁)

### Added

- `admin-web` 顶栏新增"更新于 hh:mm:ss"时间戳：
  - 新增 `composables/useLastRefreshed.ts`，按 key 打点的全局 reactive map
  - `DashboardPage` / `CallsPage` / `JobsPage` 在各自 `loadXxx` 成功后打点并通过
    `AppShell` 的 `#toolbar` 插槽展示
- `FilterToolbar` 支持 `activeFilters` + `onClearAll`，渲染"已应用筛选"chip 行；
  `CallsPage` / `JobsPage` 接入，单项 chip 可关闭并立刻重新筛选
- 概览页支持时间窗切换：新增 24h / 7d / 30d 单选；后端
  `GET /api/admin/dashboard/overview?windowHours=` 支持 1~720 小时范围

### Changed

- **概览窗口默认从 24h 改为 7d**：`audit.Service.Overview` 签名改为
  `Overview(ctx, windowHours int)`，`windowHours<=0` 走 168（7d）默认值，
  上限 720（30d）；前端 `getDashboardOverview(windowHours?)` 默认传 168
- `AIProvidersPage` 场景卡片 eyebrow 去掉 `text-transform: uppercase` 和
  `letter-spacing`，让中文场景名（"AI 总结 / 标题精修 / 流程图生成"）正常显示
- 四个列表页（Calls / Jobs / AIProviders / Settings）的分页条抽成
  `.pagination-row` 公共类，加 `border-top` 与卡片内容分隔

### Notes

- 修改时间：2026-04-18 晚
- 变更背景：P0 体验硬伤收口第一批，重点补齐刷新时间戳、筛选可见性、分页视觉分隔；
  概览页时间窗口写死 24h 导致数据太稀疏，改为 7d 默认并支持切换
- 接口契约：`GET /api/admin/dashboard/overview` 新增可选 `windowHours` 查询参数，
  旧调用方不传参时行为变化（窗口由 24h 扩大到 7d），响应里 `windowHours` 字段
  已存在、前端卡片注释会随之显示"最近 168 小时"
- 未做：P0-3 侧栏告警红点（待确认是否复用概览接口的失败数）；P1 图表升级

## 2026-04-18

### Changed

- 后台管理前端继续收口为统一壳层工作台：
  - `AppShell` 顶部统一改为“面包屑 + 页面标题 + toolbar 插槽”，
    `AI Provider`、概览页和服务健康页等主操作开始收口到同一顶部动作区
  - 登录页改为品牌介绍 + 表单双栏结构；全局样式 token、表格态、空态和
    路由切页动画同步刷新，后台视觉语言进一步统一
  - 概览页开始显式区分“暂无数据”和“待采样”状态，成功率分布改为条形
    进度展示，服务健康摘要卡也补齐更细的状态文案与跳转提示

### Fixed

- 修复后台壳层侧栏底部把后端状态写死为“后端在线”的误导问题：
  - 新增 `useBackendHealth`，轮询
    `GET /api/admin/server-health/overview` 后按 `online / degraded /
    critical / offline / unknown` 展示真实状态，并可直接跳到服务健康页
- 修复后台登录页亮点列表保留浏览器默认 `ul` 缩进导致文案整体右偏的
  样式回归

### Notes

- 修改时间：2026-04-18 02:48 CST
- 变更背景：上一轮后台视觉重构后，壳层导航、登录页和概览页仍有一批细节
  没有完全收口；其中侧栏底部把后端状态写死为在线，登录页亮点区也因默认
  列表样式出现偏移，需要补齐真实状态表达和基础视觉回归
- 核心改动：后台路由页接入统一转场；`AppShell` 增加 breadcrumb、账号下拉
  与 toolbar 插槽；侧栏/抽屉底部接入真实后端健康探测；登录页改为双栏品牌
  布局并补齐列表重置；`StatusTag`、`PageState` 与概览页指标卡的状态表达
  同步升级
- 影响范围：`admin-web/src/App.vue`、
  `admin-web/src/composables/useBackendHealth.ts`、
  `admin-web/src/components/*`、`admin-web/src/pages/*`、
  `admin-web/src/style.css`、`CHANGELOG.md`
- 兼容性/风险：本次只涉及后台前端与已存在的服务健康接口，不改后端 API
  契约；但后台壳层登录后会新增一次首屏探测并按 `30s` 轮询服务健康概览，
  若会话过期则侧栏状态会回退为 `unknown`，并继续由原有路由守卫接管登录态
- 验证情况：已执行 `cd admin-web && npm run build`；已针对侧栏健康状态不再
  写死、登录页亮点列表缩进修复做代码级复核；本轮未单独补做浏览器人工验收

## 2026-04-15

### Fixed

- 后台“服务健康”里的 `Linkparse Sidecar /v1/health` 探测现在会自动复用
  sidecar 运行时 `API Key`，避免 sidecar 已启用内部鉴权时，后台健康面板
  对 `http://127.0.0.1:8091/v1/health` 误报 `HTTP 401 Unauthorized`

### Notes

- 修改时间：2026-04-15 22:45 CST
- 变更背景：后台服务健康面板会主动探测 sidecar 的 `/v1/health`，但当前
  linkparse-sidecar 在配置 `LINKPARSE_INTERNAL_API_KEY` 后会要求
  `Authorization: Bearer ...`；此前该探测未复用运行时 sidecar `API Key`，
  导致服务本身正常、后台却持续显示 `401 Unauthorized`
- 核心改动：`ServerHealthService` 的 HTTP 探测新增可选 Bearer Token 注入；
  `sidecar-health` 探测改为复用 `sidecar.linkparse.api_key` /
  `LINKPARSE_SIDECAR_API_KEY`；补充定向单测，校验 sidecar 探测会带鉴权头，
  而 backend `/healthz` 不会误带该头
- 影响范围：`backend/internal/admin/server_health.go`、
  `backend/internal/admin/server_health_test.go`、`CHANGELOG.md`
- 兼容性/风险：仅修正后台健康检查探测口径，不改 sidecar 鉴权策略，也不改
  实际业务请求链路
- 验证情况：已执行 `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache
  GOCACHE=/tmp/caipu-go-build-cache go test ./internal/admin`

## 2026-04-15

### Added

- AI 多 Provider 新增连续异常邮件告警能力：
  - 后端新增 `ai_provider_alert_states` 状态表，按 Provider 维度持久化连续失败
    次数、最近错误、最近恢复时间和最近一次已发送告警状态
  - 新增 `backend/internal/aialert/`，支持通过 SMTP 发送告警与测试邮件，
    默认兼容 QQ 邮箱 `smtp.qq.com`
  - `airouter` 现在会在真实 Provider 调用成功/失败后更新连续异常状态；
    同一 Provider 连续异常达到阈值后会自动发邮件，成功一次后自动清零

### Changed

- 后台配置中心新增 `AI Provider 告警` 分组：
  - 可在线配置启停开关、连续异常阈值、SMTP 主机/端口、QQ 邮箱账号、SMTP
    授权码、发件人与收件邮箱
  - 分组“测试连接”改为发送一封测试邮件，便于直接验证 SMTP 与收件链路
- AI Provider 告警邮件模板增强为更适合运维排障的文本格式：
  - 标题开始包含场景中文名与 Provider 展示名
  - 正文补充触发来源、目标对象、最近 3 次失败摘要和静态排查建议
- `AI Provider` 页面补充跳转提示，引导从配置中心配置连续异常告警，避免和
  路由场景编辑入口割裂
- `README.md`、`backend/README.md` 与 `backend/configs/example.env`
  同步补充 AI Provider 告警配置入口与默认环境变量

### Notes

- 修改时间：2026-04-15 22:27 CST
- 变更背景：当前项目已支持同一 AI 场景下配置多个 Provider 并在异常时切换，
  但缺少面向运维的主动告警；用户希望当某个 Provider 连续异常达到阈值时，
  能自动发 QQ 邮箱通知，便于及时排查上游服务或密钥问题
- 核心改动：新增 SMTP 邮件发送与测试能力；新增 Provider 连续失败状态持久化；
  在 `airouter` 的实际调用链路里接入连续异常计数与阈值告警；后台配置中心
  新增 `AI Provider 告警` 分组，默认阈值为 `3`；告警邮件模板补齐场景中文
  名、触发来源、目标对象与最近失败摘要，便于直接排障
- 影响范围：`backend/internal/aialert/`、`backend/internal/airouter/`、
  `backend/internal/appsettings/`、`backend/internal/config/`、
  `backend/internal/app/`、`backend/migrations/016_add_ai_provider_alert_states.sql`、
  `backend/configs/example.env`、`admin-web/src/pages/AIProvidersPage.vue`、
  `README.md`、`backend/README.md`、`CHANGELOG.md`
- 兼容性/风险：告警发送依赖可用的 SMTP 配置；若 SMTP 授权码或收件人配置错
  误，业务主流程不会被阻塞，但阈值触发时后台日志会记录发送失败；当前告警
  只统计真实运行时 Provider 调用，不统计后台“测试当前草稿/单节点测试”
- 验证情况：已执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test
  ./internal/aialert ./internal/appsettings ./internal/airouter ./internal/recipe`；
  已执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/app
  ./cmd/server`；本轮未执行 `admin-web` 构建，前端仅新增静态提示文案

## 2026-04-15

### Fixed

- AI Provider 后台场景测试开始复用真实业务 prompt 模板，并使用内置真实样例
  case 发起测试，避免“最小测试 prompt 通过但真实业务 prompt 表现不同”的偏差
- 菜谱详情页在“做法重新整理”成功但 AI 回退规则整理时，开始展示真实 AI
  失败原因，而不是只提示“规则整理”
- 修复 AI 多 Provider `summary / title / flowchart` 真实运行链路从数据库加载
  Provider 时遗漏密文字段的问题，避免后台“测试当前草稿 / 单节点测试”可成
  功，但实际业务请求因未带 `Authorization` 头而被上游返回 `401 未提供令牌`

### Notes

- 修改时间：2026-04-15 10:25 CST
- 变更背景：排查“做法重新整理”总是直接回退规则整理时，定位到后台 AI
  Provider 页面测试链路使用的是内存草稿配置，而真实业务链路使用的是从
  `ai_route_providers` 回读的运行时配置；后者在组装 `ProviderConfig`
  时只回填了 `HasAPIKey / APIKeyMasked`，遗漏了运行时真正用于解密和注入
  `Authorization` 的密文字段；同时现有详情页在 AI 回退规则整理时只显示泛化
  提示，无法告诉用户真实失败原因；后台 AI Provider 测试此前使用的是最小
  prompt，与真实业务 prompt 存在偏差
- 核心改动：`airouter.buildSceneConfig` 在从数据库恢复 Provider 时同步回填
  `APIKey` 密文，确保真实业务链路与后台测试链路都能在请求前正确解密并注
  入 Bearer Token；`linkparse` 在 AI 总结失败并回退规则整理时会生成真实错
  误摘要，`auto_parse_worker` 会把该提示落到 `recipes.parse_error`，详情页优
  先展示这条回退原因；`AIRouter` 测试链路新增可注入测试输入构造器，当前
  `summary / title / flowchart` 均已切到复用真实业务 prompt 模板和内置样
  例 case；新增定向单测覆盖该回归场景
- 影响范围：`backend/internal/airouter/service.go`、
  `backend/internal/airouter/service_test.go`、
  `backend/internal/linkparse/*`、`backend/internal/recipe/*`、
  `backend/internal/app/app.go`、`pages/recipe-detail/index.vue`、
  `CHANGELOG.md`
- 兼容性/风险：仅修正多 Provider 运行时从数据库恢复配置时的缺失字段，不改
  API 结构、不改调度策略；修复后此前被误判为“AI 不可用”的真实业务请求会
  重新命中已保存的 Provider 凭证
- 验证情况：已通过线上数据库与上游定向复现确认根因；本轮将补充
  `backend/internal/airouter` 定向单测验证

## 2026-04-15

### Fixed

- AI Provider 后台 `summary` 场景的“测试当前草稿 / 单节点测试”把测试请求的
  `maxTokens` 从 `256` 提高到 `1024`，避免部分上游在返回完整菜谱 JSON 前
  被截断，进而误报 `unexpected end of JSON input`

### Notes

- 修改时间：2026-04-15 01:35 CST
- 变更背景：`summary` 场景的结构化测试 prompt 需要模型返回完整菜谱 JSON；
  实际联调中发现 `https://x666.me/v1` 这类上游虽然可正常鉴权，但在测试链
  路即使使用 `maxTokens=512` 仍可能以 `finish_reason=length` 截断输出，导
  致后台误判为 JSON 解析失败
- 核心改动：上调 `summary` 场景路由测试的 token 预算，并补充定向单测覆盖
  该测试上限，避免后续回归
- 影响范围：`backend/internal/airouter/service.go`、
  `backend/internal/airouter/service_test.go`、`CHANGELOG.md`
- 兼容性/风险：仅影响后台 AI Provider 页面里的场景测试，不改真实业务正文总
  结链路；测试请求的输出上限提高后，单次验证的 token 消耗会略有增加
- 验证情况：已执行 `cd backend && go test ./internal/airouter`

## 2026-04-10

### Fixed

- AI 多 Provider 路由补齐首轮实现后的关键闭环修复：
  - `airouter` 现在支持对模型输出内容做场景级校验，`summary / title /
    flowchart` 在上游返回 `200` 但内容结构不合法时，会按
    `invalid_response` 继续切换到下一个 Provider，不再把这类响应误记为
    成功
  - 后台“测试当前草稿 / 单节点测试”改为按场景使用结构化测试 prompt 与输
    出校验，不再只做 `ping + MaxTokens=1` 式联通性探测，避免测试页误报
    可用
  - `flowchart` 的“是否已配置”判定收紧为基于运行时可用路由，而不是只要
    注入 `AIRouter` 或仅存在运行时 loader 就算启用，避免后台接口和
    worker 在实际无可用节点时被误判为可用
  - `AI Provider` 页面切场景和手动刷新时都会先清空旧编辑态，防止目标场景
    加载失败时仍保留上一场景草稿，进而误保存到新的场景 key 上
  - 后台返回的 `compatibilityMode` 改为按真实运行态计算；当场景虽然已保
    存到数据库，但没有可参与调度的 Provider 时，页面会继续明确提示仍在
    走兼容链路
  - 新增 `backend/internal/airouter/service_test.go` 与
    `backend/internal/recipe/flowchart_test.go`，覆盖输出校验切换与
    `flowchart` 配置判定的关键回归场景

### Added

- 新增 AI 多 Provider 配置与调度设计文档：
  - 根目录新增 `docs/ai-multi-provider-routing-design.md`
  - 文档明确了 `summary / title / flowchart` 三个场景的多 Provider
    配置模型、`priority_failover / round_robin_failover` 策略、熔断、
    错误分类、审计口径与 `admin-web` 页面形态
  - `README.md` 与 `backend/README.md` 同步补充该设计文档入口，方便后续
    从项目总览与后端说明中直接查阅

### Notes

- 修改时间：2026-04-10 17:08 CST
- 变更背景：当前 AI 总结、标题精修和流程图生成仍主要依赖单 Provider
  配置，用户希望后台管理端支持维护多个 API，并在运行时进行轮询或异常时
  切换到备用节点；为了避免后续实现时再反复讨论，需要先把存储模型、调度
  策略、审计口径和兼容方案沉淀为项目正式文档
- 变更背景：AI 多 Provider 首轮落地后，代码审查发现仍存在 4 个关键问题：
  场景切换失败时后台页面可能误保存旧草稿、`200` 但结构错误的模型输出无
  法切到备用节点、`flowchart` 可用性判断过宽，以及兼容模式标记与真实运
  行态不一致
- 核心改动：在 `airouter` 引入输出校验回调并统一把不合法内容归类为
  `invalid_response`；`summary / title / flowchart` 三条链路接入该能力；
  后台草稿测试改为按场景做结构化输出校验；`flowchart` 的配置判断改为检
  查真实可用路由与合并后的运行时配置；管理端切场景和刷新时都会重置编辑
  器状态；`compatibilityMode` 与场景摘要节点数改为按运行时是否真的可路
  由来计算
- 影响范围：`backend/internal/airouter/`、`backend/internal/linkparse/`、
  `backend/internal/recipe/`、`admin-web/src/pages/AIProvidersPage.vue`、
  `CHANGELOG.md`
- 兼容性/风险：本次不改数据库结构，也不改现有 API 路径；但路由层现在会
  对模型内容做更严格校验，原先“HTTP 成功但内容格式错误”且被当作成功的
  上游会被识别为失败并触发切换，这属于预期纠偏
- 验证情况：已补充 `airouter` 与 `flowchart` 单测；本轮在云服务器上未再
  执行 `admin-web` 构建；后端定向测试尝试执行 `cd backend && go test
  ./internal/airouter ./internal/recipe`，但当前沙箱环境受 Go 依赖下载网
  络限制，未完成自动化验证

## 2026-04-09

### Changed

- 后台管理前端新增“本地构建产物上传到服务器”的低风险发布链路：
  - 新增 `scripts/upload-admin-web-dist.sh`，支持在本地或 CI 机器上先构建
    `admin-web/dist`，再通过 `scp + ssh + tar` 上传到服务器，并在远端
    原子替换 `/srv/caipu-miniapp/admin-web/dist`
  - 上传脚本现在会优先从本机 `~/.ssh/config` 自动识别
    `one-hub-server / oh-prod / my-cloud`，减少本机 SSH 别名与脚本默认值
    不一致时的手工修改
  - 新脚本支持 `PLAN_ONLY=1`、`BUILD_DIST=0`、`DOMAIN / VERIFY_URL`、
    远端备份保留数量控制等参数，默认适配当前线上目录
  - 远端解压 `dist` 时改为使用 `tar --no-same-owner`，避免从 macOS 打包上传
    后把本地 `uid/gid` 带到服务器静态目录上
  - 仓库 `.gitignore` 新增 `admin-web/.upload-tmp/` 与
    `admin-web/dist.bak-*`，避免服务器上的前端上传临时目录和回滚备份目录
    持续污染 `git status`
  - 根目录 `package.json` 新增 `npm run admin:upload` 入口，便于从 macOS
    本机直接触发上传
  - `README.md` 与 `docs/cloud-server-config-overview.md` 同步改为优先推荐
    “本地构建 -> 上传 dist” 的后台前端发布口径，降低低配线上机参与
    `vite build` 的风险

- 线上部署脚本按服务拆分为独立入口，降低误触发重任务的概率：
  - 新增 `scripts/deploy-backend-on-server.sh`，固定只处理 `backend`
  - 新增 `scripts/deploy-admin-web-on-server.sh`，固定只处理 `admin-web`
  - 新增 `scripts/deploy-linkparse-sidecar-on-server.sh`，固定只处理
    `linkparse-sidecar`，并仅在依赖变更时执行 `npm install`
  - `backend/scripts/deploy-server-build.sh` 改为复用
    `scripts/deploy-backend-on-server.sh`，避免远程 server-build 再把
    `admin-web` 相关变量和逻辑一起带上
  - `scripts/deploy-on-server.sh` 降级为聚合入口，保留给“明确需要同时处理
    backend + admin-web”的场景

- 线上小规格云服务器的本机发布链路补齐“低占用、按变更自动收口”能力：
  - 新增 `scripts/deploy-on-server.sh`，支持在服务器本机执行
    `git pull --ff-only` 后自动识别 `backend/` 与 `admin-web/` 的变更范围，
    只构建必要模块，并仅在后端有变更时重启 `caipu-backend`
  - 构建流程默认通过 `nice + ionice` 降低优先级，并将服务器本机构建时的
    `go build` 默认收口到 `GOMAXPROCS=1`，同时给 `admin-web` 提供更保守
    的 `NODE_OPTIONS` 默认值，降低 `2 vCPU / 1.9 GiB RAM / 0 swap`
    机器在部署时被打满的概率
  - `backend/scripts/deploy-server-build.sh` 改为复用上述本机发布脚本，避免
    远程触发发布时仍走“每次都全量构建 + 无条件重启”的旧口径
  - `README.md` 与 `docs/cloud-server-config-overview.md` 同步补充低资源
    服务器发布建议与显式按范围发布命令
- 线上小规格云服务器的本机发布链路进一步收紧为“默认拒绝危险构建”：
  - `scripts/deploy-on-server.sh` 新增 `PLAN_ONLY=1` 预检查模式，可在
    不执行构建与重启的前提下先查看本次 `git pull` 后将会触发哪些动作
  - 脚本现在会检测主机 `CPU / 内存 / swap`，对当前这类
    `2 vCPU / 1.9 GiB RAM / 0 swap` 低配机默认仅允许 `backend` 单独构建，
    但会拒绝 `admin-web` 构建或前后端一起构建；只有显式传入
    `ALLOW_LOW_RESOURCE_BUILD=1` 才允许硬跑前端重任务
  - 相关 README 与云服务器运维文档同步改为“先计划、再决策、必要时强制”
    的口径，避免再次因为脚本默认执行构建而把整机压死

### Notes

- 修改时间：2026-04-09 23:59 CST
- 变更背景：当前线上云服务器仅有 `2 vCPU / 1.9 GiB RAM / 0 swap`，此前
  直接在机器本机执行 `npm install`、`vite build` 与 `go build/go test`
  时容易把 CPU 与内存同时打满，严重时甚至需要重启服务器恢复
- 核心改动：新增低优先级、自动识别变更范围的本机发布脚本，并让远程
  server-build 脚本统一复用该逻辑；随后进一步按 `backend / admin-web /
  linkparse-sidecar` 拆成独立入口；同时新增 `admin-web` 产物上传脚本，
  让后台前端可以从本地或 CI 机器发布而不再依赖线上机构建；相关发布口径
  已正式沉淀到仓库文档
- 影响范围：`scripts/deploy-on-server.sh`、
  `scripts/deploy-backend-on-server.sh`、
  `scripts/deploy-admin-web-on-server.sh`、
  `scripts/deploy-linkparse-sidecar-on-server.sh`、
  `scripts/upload-admin-web-dist.sh`、`backend/scripts/deploy-server-build.sh`、
  `package.json`、`README.md`、`docs/cloud-server-config-overview.md`、
  `CHANGELOG.md`
- 兼容性/风险：默认跳过未变更模块的构建与重启，能显著减轻小机压力，但
  如果遇到“依赖未变更、node_modules 已损坏”的场景，仍需显式使用
  `ADMIN_WEB_INSTALL_MODE=always` 强制重新安装后台依赖
- 验证情况：已执行 `bash -n scripts/deploy-on-server.sh` 与
  `bash -n backend/scripts/deploy-server-build.sh`；已执行
  `RUN_GIT_PULL=0 DEPLOY_SCOPE=none bash scripts/deploy-on-server.sh`
  验证空跑分支；已执行 `bash -n scripts/deploy-backend-on-server.sh`、
  `bash -n scripts/deploy-admin-web-on-server.sh`、
  `bash -n scripts/deploy-linkparse-sidecar-on-server.sh`；已执行
  `RUN_GIT_PULL=0 PLAN_ONLY=1 bash scripts/deploy-backend-on-server.sh`、
  `RUN_GIT_PULL=0 PLAN_ONLY=1 bash scripts/deploy-admin-web-on-server.sh`、
  `RUN_GIT_PULL=0 SIDECAR_INSTALL_MODE=always PLAN_ONLY=1 bash scripts/deploy-linkparse-sidecar-on-server.sh`
  验证拆分入口的计划分支；已执行
  `RUN_GIT_PULL=0 bash scripts/deploy-admin-web-on-server.sh` 与
  `RUN_GIT_PULL=0 SIDECAR_INSTALL_MODE=always bash scripts/deploy-linkparse-sidecar-on-server.sh`
  验证低配机拒绝分支；已执行 `bash -n scripts/upload-admin-web-dist.sh` 与
  `SERVER_HOST=root@example.com DOMAIN=www.example.com PLAN_ONLY=1 bash scripts/upload-admin-web-dist.sh`
  验证 `admin-web` 上传脚本的语法与计划输出；已执行 `git diff --check`；
  本次未在生产机上直接跑前端构建，以避免再次触发高负载

### Added

- 后台管理平台新增“服务健康”标准版能力：
  - 后端新增 `GET /api/admin/server-health/overview`，统一返回主机资源、
    `systemd` 服务状态和内网 HTTP 健康探测结果
  - 前端新增 `服务健康` 独立页面，并在概览页补入同口径的健康摘要卡，
    支持查看 CPU / 内存 / 磁盘、`nginx` / `caipu-backend` /
    `caipu-linkparse-sidecar` 状态以及 `/healthz`、`/api/healthz`、
    sidecar `/v1/health` 探测结果

### Changed

- 后台管理平台补齐“桌面 + 平板优先”的响应式布局收口：
  - `AppShell` 从固定侧栏改为“桌面侧栏 + 平板抽屉导航”双形态，
    统一接入前端断点状态源
  - 概览页、服务健康页、筛选工具条、任务/调用详情抽屉和表格固定操作列
    按 `1440 / 1200 / 992 / 768` 四档重新收口，避免平板和窄屏下出现
    侧栏堆叠、抽屉过宽和固定列遮挡

### Notes

- 修改时间：2026-04-09 18:03 CST
- 变更背景：后台此前已经具备 AI 可观测性与配置中心，但缺少对当前
  云服务器主机资源、核心服务状态和内网健康探测的统一视图；同时现有
  后台虽然有基础断点样式，平板和窄屏下仍存在侧栏、筛选区、抽屉和表格
  体验不一致的问题
- 核心改动：后端新增轻量 `ServerHealthService` 聚合 Linux 主机资源、
  `systemctl is-active` 和内网 HTTP 健康检查；前端新增服务健康页、
  概览页健康摘要卡、`HealthRing` 组件与统一响应式断点源，并重构后台
  壳层为侧栏/抽屉双形态布局
- 影响范围：`backend/internal/admin/*`、`backend/internal/app/*`、
  `admin-web/src/components/*`、`admin-web/src/pages/*`、
  `admin-web/src/router/index.ts`、`admin-web/src/types.ts`、
  `admin-web/src/utils/admin-display.ts`、`admin-web/src/style.css`、
  `README.md`、`backend/README.md`、`CHANGELOG.md`
- 兼容性/风险：标准版仅做手动刷新，不引入 `Prometheus/Grafana`、
  历史时序存储或告警中心；主机资源采集默认依赖 Linux `/proc` 与
  `systemd`，因此本地 macOS 开发环境允许部分检查显示为 `unknown`；
  当前后台首包仍保留 `element-plus` 大 chunk 告警，后续若继续压包，
  仍需进一步做组件和页面级拆分
- 验证情况：已执行
  `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./...`；
  已执行 `cd admin-web && npm run build`；已新增并通过服务健康聚合的
  `healthy / warning / critical / unknown` 回归测试；已确认服务健康页
  构建产物与概览页摘要卡均完成构建级联检查

### Changed

- 后台管理平台首轮从 MVP 升级为“稳重数据台 + 排障优先”的完整工作台：
  - `admin-web` 新增 `StatusTag`、`FilterToolbar`、`PageState`、
    `JsonViewerCard`、`CopyTextButton`、任务/调用详情抽屉等共享组件，
    页面交互从“纯表格 + toast”升级为“可筛选、可回溯、可复制、可空态/
    错误态表达”的工作流
  - 概览页重做为 KPI + 趋势 + 最近失败 + Provider/Model 拆分视图，
    失败任务支持直接打开任务详情并继续下钻到关联调用
  - 任务页、调用页开始使用 URL query 持久化筛选条件，补齐
    `timeFrom/timeTo` 时间范围过滤、重置、显式详情入口与详情抽屉
  - 配置中心补齐脏状态提示、敏感值清空确认、保存前 diff 摘要、
    最近测试结果面板和审计按 `group/action` 过滤
  - 后端运行时配置保存/测试逻辑收紧为“显式非空值优先于清空标记”，
    避免前端同一字段既传新值又带 `clearKeys` 时被误删
  - `vite` 路由切为懒加载，`echarts` 改为按需模块引入，并补
    `manualChunks` 让前端入口包明显收敛；当前仍保留 `element-plus`
    大 chunk 告警，后续若要继续压缩需再推进组件级按需注册
  - 进一步把 `Element Plus` 从全量 `app.use(ElementPlus) + dist/index.css`
    切换为“模板组件按需解析 + 服务组件最小样式引入”，将后台样式产物从
    约 `352 KB` 压到约 `160 KB`，同时保留现有页面功能和样式一致性

### Notes

- 修改时间：2026-04-09 14:32 CST
- 变更背景：后台管理平台虽然已经具备概览、任务、调用、配置中心四个
  基础页面，但此前更偏“能看数据的 MVP”，在配置误操作防护、排障
  下钻路径、筛选持久化、空态/错误态表达和响应式布局上仍明显偏弱，
  日常运维和联调效率不高
- 核心改动：统一后台视觉 token 和交互骨架，新增共享组件与详情抽屉，
  重构概览/任务/调用/配置中心页面，并同步修复运行时配置的清空优先级
  逻辑与回归测试
- 影响范围：`admin-web/src/components/*`、`admin-web/src/pages/*`、
  `admin-web/src/router/index.ts`、`admin-web/src/style.css`、
  `admin-web/src/utils/*`、`admin-web/vite.config.ts`、
  `backend/internal/appsettings/runtime_provider.go`、
  `backend/internal/appsettings/runtime_provider_test.go`、`CHANGELOG.md`
- 兼容性/风险：本次不新增后台公开接口，只开始正式使用已有
  `timeFrom/timeTo` 与审计 `group/action` 查询参数；前端首包已明显拆散，
  但 `element-plus` 仍是当前最大的 vendor chunk，构建时继续有告警；
  现阶段已先完成按需样式与组件注入优化，若后续还要继续压缩 JS 体积，
  需要进一步减少后台对重型表格/抽屉/描述组件的依赖或替换部分 UI 组件
- 验证情况：已执行 `cd admin-web && npm run build`；已执行
  `cd backend && go test ./...`；已重点新增并通过运行时配置
  “显式值覆盖清空标记”的回归测试；已确认概览、任务、调用、配置中心
  页面均完成构建级联检查

### Fixed

- 修复后台 AI 仪表盘概览/趋势接口在产生真实审计耗时数据后返回 `500`：
  - `backend/internal/audit/service.go` 里的概览与趋势统计改为按浮点读取
    SQLite `AVG(duration_ms)` 结果，再安全转换为整数毫秒，避免平均值
    非零时扫描失败
  - 趋势分桶改为直接使用 SQLite 对 RFC3339 时间做日期/整点归一化，
    不再依赖对时间字符串做 `substr + strftime('%s', ...)` 的脆弱组合，
    避免 `24h` 视图出现空 bucket 或异常标签
  - `backend/internal/audit/service_test.go` 新增带真实正耗时样本的回归
    用例，覆盖此前“无数据正常、有数据即 500”的场景

### Notes

- 修改时间：2026-04-09 13:16 CST
- 变更背景：线上后台管理页在 `2026-04-09 09:55 CST` 起连续触发
  `GET /api/admin/dashboard/overview` 与
  `GET /api/admin/dashboard/trends?range=24h` 的 `500`，而
  `GET /api/admin/ai/jobs` 仍保持正常，说明问题集中在审计聚合统计链路
- 核心改动：修正平均耗时聚合的类型处理与时间分桶表达式，让审计概览
  和趋势图在出现真实 AI 调用耗时样本后仍能稳定返回
- 影响范围：`backend/internal/audit/service.go`、
  `backend/internal/audit/service_test.go`、`CHANGELOG.md`
- 兼容性/风险：本次不改接口字段和响应结构，但 `24h` 趋势图的横轴标签
  现在会稳定输出按小时归一化后的时间文本；如果前端后续想展示本地时区，
  仍需单独明确口径
- 验证情况：已执行
  `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/audit`；
  已执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./...`，
  其中 `internal/linkparse` 与 `internal/recipe` 的部分测试因当前沙箱禁止
  `httptest` 监听本地端口而失败，其余包通过；已结合
  `journalctl -u caipu-backend -n 200 --no-pager` 确认线上报错时间点与
  新增审计数据进入 24 小时统计窗口的时间吻合；已执行
  `go build -o bin/server ./cmd/server` 并重启 `caipu-backend`；已通过本机
  Bearer 鉴权直连 `http://127.0.0.1:8080/api/admin/dashboard/overview` 与
  `http://127.0.0.1:8080/api/admin/dashboard/trends?range=24h`，确认两者均
  返回 `200`

## 2026-04-08

### Added

- 新增云服务器配置总览文档：
  - 根目录新增 `docs/cloud-server-config-overview.md`
  - 文档记录当前线上云服务器的实际服务拓扑、`nginx` 路由、`systemd` 服务、端口监听、关键配置文件入口与发布命令
  - 文档明确区分 Hapi 根站点、`caipu-backend`、`admin-web` 静态托管与 `linkparse-sidecar` 的职责边界，便于后续排障和发版

### Notes

- 修改时间：2026-04-08 23:55 CST
- 变更背景：当前线上环境已经同时承载 Hapi 根站点、小程序 Go 后端、后台管理前端静态页和 linkparse sidecar，但这套真实部署关系此前主要散落在服务器配置和对话里，后续回看成本高，也容易在改 nginx 或重启服务时误伤其他链路
- 核心改动：新增一份基于当前云服务器实况整理的运维总览文档，集中记录主机基础信息、域名路径分流、服务名、配置文件位置、环境变量范围、日常发布命令和检查命令，并显式说明哪些文件在 Git 内、哪些只存在服务器本地
- 影响范围：`docs/cloud-server-config-overview.md`、`README.md`、`CHANGELOG.md`
- 兼容性/风险：本次仅新增文档与 README 链接，不改变运行时代码；文档中的服务状态、路径和端口反映的是 2026-04-08 当下线上现状，后续若调整 nginx、systemd 或目录结构，需要同步回写更新，避免文档再次漂移
- 验证情况：已基于当前服务器上的 `/etc/nginx/conf.d/www.gxm1227.top.conf`、`/etc/systemd/system/*.service`、监听端口、运行目录和环境文件键名完成实况核对；已确认文档未写入任何真实密钥或敏感值

### Changed

- 后台管理平台前端补齐“兼容现网 nginx 前缀且不影响 Hapi 根站点”的发布口径：
  - `admin-web` 的后台接口路径改为相对 `VITE_API_BASE` 组装，不再把 `/api` 前缀写死在页面代码里
  - 新增 `admin-web/.env.development` 与 `admin-web/.env.production`，默认分别对接本地 `/api` 与现网 `/caipu-api`
  - 这样线上只需新增 `/admin/` 静态托管即可，不必把现有 `location /` 从 Hapi 服务切走，也不必额外改造现网 `/caipu-api` 约定

### Notes

- 修改时间：2026-04-08 23:31 CST
- 变更背景：现网域名根路径已经由 Hapi 服务承接，微信小程序接口又沿用了 `/caipu-api`、`/caipu-uploads` 这套自定义 nginx 前缀；如果后台管理平台继续写死 `/api/admin/*`，上线时就需要额外改 nginx 的 `/api` 路由，容易误伤现有 Hapi 站点
- 核心改动：将后台前端请求前缀收口为 `VITE_API_BASE + /admin/...` 的组合，并把开发/生产环境默认值分别固化为 `/api` 与 `/caipu-api`；这样本地开发仍走 Vite 代理，线上生产则直接复用现有后端前缀
- 影响范围：`admin-web/src/api/admin.ts`、`admin-web/.env.development`、`admin-web/.env.production`、`README.md`、`CHANGELOG.md`
- 兼容性/风险：本次方案默认现网继续保留 `/caipu-api -> backend /api` 的 nginx 转发；如果后续要把现网统一收口回标准 `/api`，只需同步调整生产环境 `VITE_API_BASE` 或 nginx 映射，不影响 Hapi 根站点
- 验证情况：已完成代码级配置自检；后续将通过本机构建、nginx `/admin/` 静态托管和后台鉴权接口连通性验证确认最终上线链路

### Added

- 新增后台管理平台、AI 可观测性与动态配置中心 MVP 实现：
  - 根目录新增独立后台工程 `admin-web/`，采用 `Vue 3 + Vite + Element Plus + ECharts`
  - 后端新增 `/api/admin/*` 后台认证、仪表盘、AI 任务、AI 调用与运行时配置接口
  - 新增 `ai_job_runs`、`ai_call_logs`、`app_runtime_settings`、`app_setting_audits` 迁移与对应服务模块
  - 新增 `scripts/build-admin-web.sh`，并把根目录 `package.json` 扩展出 `admin:dev / admin:build / admin:preview` 命令

### Changed

- AI / sidecar 链路改为支持运行时配置与统一审计：
  - `linkparse` 的总结、标题精修与 sidecar 调用统一接入任务级 / 调用级埋点
  - 流程图生成器与 worker 改为支持运行时读取 `ai.flowchart.*` 配置，并单独记录 `flowchart` 任务与调用日志
  - 自动解析 worker 会为 `parse_summary` 任务补充 `worker + recipe` 维度的审计上下文
- 应用设置中心扩展为“移动端隐藏设置页 + 后台配置中心”共用底座：
  - 现有 `Bilibili SESSDATA` 仍沿用 `app_bilibili_settings`，但现在会同步写入统一审计表
  - 新增 `RuntimeProvider` 以 15 秒本地缓存承接 `ai.summary / ai.flowchart / ai.title / sidecar.linkparse` 运行时配置读取
- 部署链路升级为支持同域 `/admin`：
  - `backend/scripts/deploy.sh` 现在可本地构建并上传 `admin-web/dist`
  - `backend/scripts/bootstrap-server.sh` 和 `deploy-server-build.sh` 已补齐 `/admin` 静态托管与构建逻辑
  - 部署文档和 README 已同步更新后台账号、环境变量与 nginx 路由说明

### Notes

- 修改时间：2026-04-08 23:58 CST
- 变更背景：设计文档已经明确项目要补一版“后台管理平台 + AI 可观测性 + 动态配置中心”，仓库此前虽有 `appsettings` 和隐藏设置页基础，但仍缺少统一的 AI 成功率统计、失败追踪、后台认证和 PC 端运维入口
- 核心改动：后端新增 `audit / admin / runtime settings` 三层底座，把自动解析、标题精修、流程图生成和 sidecar 调用接入统一埋点；同时新增独立 `admin-web` 工程承接概览、任务、调用和配置中心页面；部署脚本与文档同步收口到同域 `/admin` 路线
- 影响范围：`backend/internal/admin/*`、`backend/internal/audit/*`、`backend/internal/appsettings/*`、`backend/internal/linkparse/*`、`backend/internal/recipe/*`、`backend/internal/app/*`、`backend/migrations/014_add_ai_audit_and_runtime_settings.sql`、`admin-web/*`、`backend/scripts/*`、`scripts/build-admin-web.sh`、`README.md`、`backend/README.md`、`docs/backend-deploy-quickstart.md`、`package.json`
- 兼容性/风险：后台登录依赖新增环境变量 `ADMIN_USERNAME / ADMIN_PASSWORD_HASH`；当前 `admin-web` 构建产物体积较大，`vite build` 会给出大 chunk 警告，后续可再做按页拆包；`/admin` 的 nginx `alias + try_files` 路由已按常见 SPA 方式配置，但上线时仍建议先在目标环境做一次真实刷新验证
- 验证情况：已执行 `cd backend && go test ./...`；已执行 `cd admin-web && npm run build`；已执行 `bash scripts/build-admin-web.sh`；已执行 `bash -n backend/scripts/deploy.sh`、`bash -n backend/scripts/bootstrap-server.sh`、`bash -n backend/scripts/deploy-server-build.sh`、`bash -n scripts/build-admin-web.sh`；已执行 `git diff --check`

### Added

- 新增后台管理平台、AI 可观测性与动态配置中心设计文档：
  - 根目录新增 `docs/admin-console-ai-observability-design.md`
  - 文档明确一期以“应用内埋点 + SQLite + 独立轻量后台”为主路线，不直接以 `Grafana` 作为主后台系统
  - 文档补充了 `ai_job_runs`、`ai_call_logs`、`app_runtime_settings`、`app_setting_audits` 的建议表结构、后台 API、页面信息架构、动态配置边界与分阶段实施方案

### Notes

- 修改时间：2026-04-08 22:09 CST
- 变更背景：当前后端已经具备自动解析、流程图生成、标题精修与隐藏设置页等能力，但仍缺少统一的 AI 调用成功率统计、失败追踪、在线配置与 PC 后台管理方案；为了后续开发时减少反复讨论，需要先把管理后台、AI 可观测性和动态配置中心的整体设计沉淀成项目内正式文档
- 核心改动：新增一份独立设计文档，结合仓库现状给出后台系统推荐落位、模块拆分、数据模型、API 清单、配置热更新边界、前后端技术选型和三阶段排期；方案明确建议在现有 `Go + chi + SQLite` 基础上扩展 `appsettings` 与新增 `audit/admin/admin-web` 模块，一期先实现应用内自管埋点和轻量后台，二期再评估接入 `OpenTelemetry + Grafana`
- 影响范围：`docs/admin-console-ai-observability-design.md`、`CHANGELOG.md`
- 兼容性/风险：本次仅新增设计文档，不涉及运行时代码和接口行为变更；文档里的动态配置、后台认证与埋点口径仍需在正式开发阶段结合实现细节再做一次收口，尤其要避免把“任务成功率”和“API 成功率”混为一谈，以及避免在 SQLite 中无节制存储大体积请求响应内容
- 验证情况：已结合当前仓库中的 `backend/internal/config`、`backend/internal/appsettings`、`backend/internal/app`、`pages/app-settings` 等现有实现做方案对齐；已完成文档内容与项目现状的一致性静态自检；本次未涉及代码执行和接口联调

### Added

- 新增微信小程序命令行自动预览能力：
  - 根目录新增 `scripts/wx-auto-preview.sh`，支持在 macOS 上自动查找 HBuilderX 与微信开发者工具 CLI，并串起“编译 -> 打开项目 -> auto-preview”
  - 新增独立说明文档 `docs/wechat-auto-preview.md`，整理前置条件、参数、环境变量和常见排查方式
  - `package.json` 补充 `npm run wx:auto-preview` 与 `npm run wx:auto-preview:skip-compile` 两个快捷命令，减少手动输入成本

### Changed

- 微信好友聊天里的邀请分享卡片继续收口为“更适合聊天缩略图”的精简布局：
  - 动态分享图移除底部三张指标卡，改为“立即加入提示 + 一行关键信息 + 邀请码兜底”的更轻量结构
  - 修复底部邀请码深色条与信息面板纵向重叠的问题，避免聊天卡片缩略图里出现内容压住
  - 图内品牌文案从“我们的数字厨房”收口为“共享厨房邀请”，减少和小程序卡片外层信息的重复
  - 分享标题改为“加入「厨房名」一起维护菜单”，不再重复邀请人昵称，聊天列表里更聚焦“去哪里、为什么点开”
  - 根据真机截图继续把分享图收成“极简邀请函”结构：移除邀请人头像、功能标签、深色底条和卡面邀请码，只保留厨房名、短说明、状态与有效期
  - 分享标题再次收短为“邀请你加入「厨房名」”，减少微信聊天卡片标题折成两行的概率
  - 在极简邀请函结构里补回更克制的特色标签，改为 `共享菜谱 / 同步菜单 / 一起做决定` 三个品牌语义标签，填补留白但不恢复成强功能卡片
  - 标题下方的说明区改为根据厨房名行数动态收放间距，避免单行厨房名时中部留白显得发空
- “邀请成员”弹层里的动作顺序调整为“发送给微信好友”优先：
  - `发送给微信好友` 提升为主按钮，优先承接小程序内最自然的邀请路径
  - `复制邀请码` 下沉为次按钮，但当分享入口关闭时仍保持为唯一主操作，避免降级场景失去重点

### Notes

- 修改时间：2026-04-08 14:52 CST
- 变更背景：极简邀请函版本虽然更克制，但在真机聊天截图里出现了“气质是对的、画面却偏空”的问题，尤其单行厨房名场景下，中部留白较大，缺少一点能传达产品特色的记忆点；同时邀请弹层底部仍把“复制邀请码”放在主操作位，和当前“优先直接发给微信好友”的产品路径不完全一致；另外，当前开发联调仍依赖“手动编译 -> 手动打开微信开发者工具 -> 点击自动预览”，重复成本偏高
- 核心改动：后端动态分享图在保持极简邀请函主结构的基础上，补回三枚更克制的品牌语义标签，承接原先“共享菜谱 / 同步菜单 / 自由切换”那类特色信息，但收口为更统一的视觉语气；同时根据标题实际折行数动态调整说明区和信息面板的纵向位置，让单行标题场景更饱满、双行标题场景也不至于拥挤；前端“邀请成员”弹层同步把 `发送给微信好友` 提升为主按钮，把 `复制邀请码` 调整为次按钮，并保留分享入口关闭时的主操作兜底；前端仓库还新增了微信小程序自动预览脚本和独立文档，支持在 macOS 上复用“编译 -> 打开项目 -> auto-preview”流程
- 影响范围：`backend/internal/invite/share_image.go`、`pages/index/index.vue`、`pages/index/components/invite-sheet.vue`、`scripts/wx-auto-preview.sh`、`docs/wechat-auto-preview.md`、`package.json`、`README.md`
- 兼容性/风险：本次仍依赖微信客户端对 `imageUrl` 的缓存刷新与缩略图裁切策略；同时按钮主次层级已调整，建议在微信真机里重新走一遍“邀请成员 -> 分享 / 复制”链路，确认主按钮样式、`open-type=share` 行为和无分享开关场景都符合预期；自动预览脚本目前只覆盖 macOS，且依赖本机已安装 HBuilderX、微信开发者工具并开启 CLI/HTTP 调用功能
- 验证情况：已执行 `cd backend && go test ./...`；已执行 `git diff --check`；已完成分享图布局代码级静态自检与邀请弹层交互代码自检；新增自动预览脚本已完成 `bash -n` 静态检查与本机 CLI 实跑验证，尚未在另一台全新 Mac 上做跨机器复验，也尚未在微信真机聊天窗口重新发送邀请做最终视觉验收

### Changed

- 首页“厨房”模块的前端展示文案统一调整为“空间”口径：
  - 顶部导航标题、默认分享标题和关于页描述从“数字厨房”改为“数字空间”
  - 首页底部导航、“当前厨房 / 厨房成员 / 厨房名”等模块文案改为“空间”表述
  - 当前名称展示增加前端替换逻辑，已有名称如“海哥的厨房”在该模块里会显示为“海哥的空间”

### Notes

- 修改时间：2026-04-08 21:27 CST
- 变更背景：用户希望首页“厨房”栏目整体改名为“空间”，避免顶部标题、卡片标题、成员区和当前名称展示口径不一致
- 核心改动：仅调整前端展示层文案与显示格式，不修改后端 `kitchen` 实体、接口字段或数据结构；首页模块会把展示名称里的“厨房”替换为“空间”，从而让已有名字在当前视图中同步切到新口径
- 影响范围：`pages.json`、`pages/index/index.vue`、`pages/index/components/kitchen-section.vue`、`pages/about/index.vue`、`README.md`
- 兼容性/风险：本次只统一了首页相关和品牌描述的前端展示文案，技术命名与后端接口仍保留 `kitchen` 口径；其他未纳入本轮的页面或分享图若仍使用旧文案，后续需再做一轮全链路收口
- 验证情况：已执行 `git diff --check`；已通过代码搜索复核首页模块、导航标题、关于页与 README 的相关文案替换范围；当前仓库无可直接执行的前端自动化测试，尚未做微信开发者工具或真机视觉验收

### Changed

- 邀请页与后端邀请链路继续统一为“空间”口径：
  - 邀请页标题、摘要标签、按钮文案、成功提示和说明文案从“厨房”改为“空间”
  - 邀请页展示名称现在会把已有名称里的“厨房”替换成“空间”，避免落地页和首页口径不一致
  - 后端动态分享图与邀请提示语从“共享厨房邀请”改为“共享空间邀请”
  - 后端自动生成的默认名称从“我的厨房 / XX的厨房”改为“我的空间 / XX的空间”

### Notes

- 修改时间：2026-04-08 21:33 CST
- 变更背景：首页模块已经切成“空间”，但邀请落地页、后端分享图和后端默认命名仍沿用“厨房”，会导致用户在分享链路里看到混合口径
- 核心改动：前端邀请页补充展示层替换逻辑，把邀请名称里的“厨房”统一显示为“空间”；后端分享图标题、兜底文案和默认自动命名同步切到“空间”，让新老数据在邀请链路里都尽量保持一致
- 影响范围：`pages.json`、`pages/invite/index.vue`、`backend/internal/invite/share_image.go`、`backend/internal/invite/service_test.go`、`backend/internal/kitchen/name.go`、`backend/internal/kitchen/name_test.go`、`README.md`
- 兼容性/风险：本次仍然只调整展示文案和默认命名策略，不改动后端 `kitchen` 实体、接口字段和数据库结构；历史自定义名称若包含“厨房”，当前仅在首页和邀请页/分享图展示层替换为“空间”，其他未纳入本轮的页面后续仍可能需要继续收口
- 验证情况：已执行 `gofmt -w backend/internal/kitchen/name.go backend/internal/kitchen/name_test.go backend/internal/invite/share_image.go backend/internal/invite/service_test.go`；已执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/invite ./internal/kitchen`；已执行 `git diff --check`；尝试使用 `@vue/compiler-sfc` 解析 `pages/invite/index.vue`，但当前仓库本地未安装该依赖，暂未完成这一步前端 SFC 自动校验

### Changed

- 补齐“空间”改名后的残留文案：
  - 会话层厨房名兜底从“我的厨房”改为“我的空间”
  - 菜单安排成功反馈与菜单详情页里剩余的“厨房 / 厨房成员”提示统一改为“空间 / 空间成员”

### Notes

- 修改时间：2026-04-08 21:44 CST
- 变更背景：review 发现菜单安排成功反馈、菜单详情页和前端 session fallback 仍会露出“厨房”旧文案，导致“空间”口径没有真正跑通
- 核心改动：补齐前端会话归一化层和点菜后续链路里的残留文案，让用户从首页进入空间、安排菜单、查看详情这一整段路径都保持统一口径
- 影响范围：`utils/auth.js`、`pages/index/components/meal-order-success-sheet.vue`、`pages/meal-plan-detail/index.vue`
- 兼容性/风险：本次只做前端展示文案收口，不涉及接口契约和数据结构；仓库内其他未纳入本轮的历史文档或更远链路仍可能残留“厨房”字样，后续若要全仓统一还需要继续清扫
- 验证情况：已执行 `git diff --check`；已通过代码搜索确认 `utils/auth.js`、`pages/index/components/meal-order-success-sheet.vue`、`pages/meal-plan-detail/index.vue` 中本轮涉及的 `我的厨房 / 厨房 / 厨房成员` 残留文案已清理

### Changed

- 文档与联调样例继续统一为“空间”口径：
  - 主 README、Go 后端启动文档、后端 README、微信登录清单和点菜模式原型文档里的“厨房”概念文案统一改为“空间”
  - `seed-demo` 里的默认空间名与共享样例备注同步改成“空间”口径，避免本地联调时再次出现旧词
  - 文档示例值和说明文字已按当前实现收口，但保留 `kitchenId`、`/api/kitchens`、`kitchen_id` 等真实接口字段和路径不变

### Notes

- 修改时间：2026-04-08 21:46 CST
- 变更背景：运行时代码已经基本统一为“空间”，但仓库内主文档、联调说明和本地种子数据仍大量保留“厨房”，容易让新接手的人、测试同学或本地联调环境继续看到旧口径
- 核心改动：集中收口概念说明、操作清单与示例数据名称，让“空间”成为当前仓库对外的统一产品语言，同时保留后端真实实体名和接口字段名，避免文档统一时误导实现契约
- 影响范围：`README.md`、`README-go.md`、`backend/README.md`、`docs/wechat-login-checklist.md`、`docs/meal-order-mode-prototype-v1.md`、`backend/cmd/seed-demo/main.go`
- 兼容性/风险：本次主要影响文档与本地演示数据；若已有依赖旧样例名的截图、录屏或人工测试脚本，后续需要同步更新；历史 migration 与数据库字段名本轮未动，仍保留 `kitchen` 术语以兼容现有数据结构
- 验证情况：已执行 `gofmt -w backend/cmd/seed-demo/main.go`；已执行 `git diff --check`；已通过代码搜索确认 `README.md`、`README-go.md`、`backend/README.md`、`docs/wechat-login-checklist.md`、`docs/meal-order-mode-prototype-v1.md`、`backend/cmd/seed-demo/main.go` 中已无本轮范围内的“厨房”残留文案

## 2026-04-05

### Changed

- 微信好友聊天里的邀请分享卡片升级为“后端动态生成封面图”：
  - 后端新增 `GET /api/invites/{token}/share-image`，会按邀请信息实时生成暖白纸感的邀请卡封面
  - 分享图现在会动态带出 `厨房名 / 邀请人 / 当前成员数 / 剩余名额 / 有效期 / 邀请码`
  - 前端邀请分享优先使用后端返回的 `shareImageUrl`，并在分享时附带时间戳参数降低微信旧图缓存命中概率
  - 分享标题继续收口为更短的邀请式文案，减少聊天卡片里折成两行的概率
  - 仍保留本地静态封面作为兜底，避免前后端未同时部署时分享卡片直接失效

### Notes

- 修改时间：2026-04-05 23:29 CST
- 变更背景：当前“发送给微信好友”未设置专用封面图，微信会直接截取当前厨房页作为聊天预览，信息噪音偏高，也容易把标题挤成两行
- 核心改动：将“邀请成员 -> 发送给微信好友”从固定静态封面升级为后端实时生成的邀请卡图片，并为邀请接口补充 `shareImageUrl`；分享图视觉收口为暖白纸感、深棕标题、绿色信任状态的简洁高级风格，避免继续把厨房页截图直接暴露在聊天卡片里
- 影响范围：`backend/internal/invite/*`、`backend/internal/kitchen/*`、`backend/internal/app/*`、`backend/internal/config/config.go`、`backend/README.md`、`pages/index/index.vue`、`utils/kitchen-api.js`、`README.md`、`static/invite-share-cover.png`
- 兼容性/风险：新增动态分享图依赖可用中文字体；后端默认会尝试读取系统字体，若线上环境没有可用字体，需要补配 `INVITE_SHARE_FONT_PATH` / `INVITE_SHARE_FONT_BOLD_PATH`；微信聊天卡片仍存在客户端缓存，若旧图未刷新，通常需要重新发送一次邀请消息验证
- 验证情况：已执行 `cd backend && go test ./...`；已完成分享图静态设计自检、前端分享链路代码自检与 `git diff --check`；尚未在微信真机聊天窗口里重新发送邀请做实测

## 2026-04-04

### Changed

- 点菜模式的“菜单详情”补齐删改闭环：
  - 已安排菜单详情新增 `修改菜单 / 删除安排`
  - 修改菜单不再直接改写已提交记录，而是先带出同日期草稿，再回到点菜模式继续编辑
  - 若同日期已经存在一份草稿，则优先继续那份草稿，不覆盖已有修改
  - 菜单草稿详情新增 `删除草稿`，避免只能回到点菜模式里清空
- 点菜模式交互体验继续优化：
  - 日期选择器会直接显示 `草稿中 / 已安排 / 待修改` 状态，减少试探式点击
  - 菜单详情、购物车和确认菜单页补充缩略图、餐别信息和状态说明，回看成本更低
  - 提交菜单后新增成功反馈面板，明确提供 `查看当天菜单 / 继续安排别天` 两个后续动作
  - 首页菜单 spotlight 卡片与底部悬浮条补齐轻量入场动效，状态感更明显
  - 菜单详情从预览弹层升级为独立页面，重操作不再挤在底部弹层里
- 首页动效语言继续收口：
  - 菜单 spotlight 现支持带方向感的左右滑动切换，记录切换不再直接跳字
  - 点菜模式进入 / 退出时，页面内容与底部导航会同步过渡，模式切换更连贯
  - 首页筛选、快捷搜索与点菜模式切换后，菜谱卡片会做轻量交错入场，列表反馈更直接
  - `想吃 / 吃过` 状态切换新增轻震动与页面瞬时完成态提示，当前筛选导致卡片立即消失时也能明确感知切换成功
  - 搜索框、底部 `添加` FAB、点菜悬浮条和卡片状态切换器补齐按压反馈，首页微交互更完整
- 首页“添加菜品”弹层继续收口：
  - 打开弹层时会静默读取系统剪贴板，只要内容里包含 `B 站 / 小红书` 支持链接就会自动带入；若剪贴板里是整段分享文案，也会原样保留在 `菜谱链接` 字段，不再额外展示显式的 `粘贴剪贴板` 操作
  - 首屏继续聚焦 `链接 + 菜名` 主路径，`补充信息（含备注）` 折叠为次级区域，空图片区也收成更紧凑入口
  - 修复 `补充信息` 展开后底部区域无法继续下滑的问题，滚动区现在会基于弹层固定高度正确承接长内容
  - 弹层首屏现进一步改为“自适应紧凑高度 + 极简字段布局”，短内容场景不再被固定高度撑出大块留白；首屏只保留 `菜谱链接`、`菜名` 和 `补充信息（可折叠）` 三块，footer 也会明确提示当前还差什么才能保存
  - 保存前不再把整段分享文案静默裁成裸链接，避免最终落库内容和输入框里看到的不一致
  - 保存中会锁定关闭、遮罩点击和重复提交，底部 footer 会直接展示当前保存状态
  - 保存成功后复用首页统一的轻量完成态反馈，不再额外依赖全局 loading 蒙层
- 后端 `mealplan` 新增草稿/已提交菜单的显式管理接口：
  - 新增从已提交菜单生成编辑草稿的接口
  - 新增删除草稿、删除已提交菜单的接口
- 点菜模式原型文档与后端 README 已同步更新为新的菜单详情交互口径
- 菜品自动解析图片合并策略调整为“保留现有图片在前，链接解析图片补在后”：
  - 新增菜品保存后，后端异步自动解析命中链接图片时，不再因为用户已上传图片就放弃补图
  - 若用户已手动上传图片，现有图片顺序保持不变，解析图片会按去重后追加到后面
  - 图片总数仍受现有上限控制，首张封面继续优先使用当前已有图片
- 后端菜品图片去重升级为“metadata + 内容 hash”：
  - `recipes` 新增 `image_meta_json`，开始记录每张图的来源类型、原始 URL、来源链接和内容 hash
  - 新增/编辑菜品会保留已有图片归因；`重新解析` 时会移除历史 `parsed` 图片，再写入本轮新解析图片，不覆盖用户手动图片
  - 图片转存与上传都会顺手计算 `SHA-256`，转存落地后会按内容 hash 去重，减少“同图不同 URL”导致的重复叠加

### Fixed

- 修复首页菜谱列表里“点击 `想吃 / 吃过` 后卡片会因为 `updated_at` 被刷新而自动前移”的问题：
  - 后端 `PATCH /api/recipes/{recipeID}/status` 现在只更新状态本身，不再改写 `recipe.updated_at`
  - 首页继续沿用现有按 `updated_at` 的排序口径，但单纯切换状态不会再打乱顺序
- 修复首页 `想吃 / 吃过` 状态切换“点下去有点晚半拍”的问题：
  - 前端状态切换改为本地先更新卡片状态，再异步请求后端确认，按钮和筛选结果会立即响应
  - 状态切换成功后不再整页重跑 `applyRecipes()` 和封面缓存同步，只回写当前菜品，减少列表重算带来的顿挫感
- 修复“从首页点进菜谱详情感觉要顿一下”的问题：
  - 首页点击菜谱卡进入详情时，不再先改整列表高亮态再执行跳转，减少跳页前的额外重渲染
  - 菜谱详情页的编辑弹层改为打开时才挂载，避免首屏进入就把整套编辑表单一起初始化
  - 菜谱详情页在首轮数据未到时改为显示加载骨架，不再短暂闪出“没找到这道菜”的误导状态
- 修复首页菜谱卡片里 `置顶` 与来源角标都压在缩略图上导致容易打架的问题：
  - `置顶` 标记改为收成卡片右上角更扁、更克制的小书签式状态标记，不再占用标题和眉标位置
  - 图片区只保留来源角标与张数，更容易扫读，也避免 `小红书 / B站` 与 `置顶` 互相挤压
- 修复首页“帮我选”只有普通 toast、缺少后续操作的问题：
  - 点击 `帮我选` 现在会居中弹出结果卡，展示菜名、缩略图、餐别/状态和一行简短说明，不再只是一句短提示
  - 结果卡去掉了顶部多余的“帮我选了这道”提示和右上角关闭按钮，保留内容本身做主视觉，信息密度更干净
  - 结果卡提供 `了解一下 / 换一个` 两个后续动作，主操作 `换一个` 调整到右侧，并在重抽时补上更明确的卡片切换动效
  - 抽取范围会优先遵循当前可见筛选；在 `全部` 下会优先从当前筛选结果里的 `想吃` 菜品里抽，减少抽到不合语境的结果
  - 结果卡背景层同步收口为更克制的米白纸感底色，外部 glow 和阴影都进一步减弱，减少“奶油感”过重的问题
  - 结果卡动效继续按内容优先原则收口：首次出场和“换一个”的位移、缩放与错峰延迟都再压轻一档，并补上 `prefers-reduced-motion` 兜底，切换体感更利落
  - 结果卡继续升级为暖白毛玻璃表面：主体改为半透明暖白渐层并叠加轻量 `backdrop-filter`、内高光和细边框，遮罩隔离感同步增强，整体更像被托起的推荐卡而不是普通弹层
  - 结果卡毛玻璃表面继续强化：卡壳透明度进一步放开，顶部改成更明显的斜向高光，信息 chip 与次按钮也同步改为半透明玻璃材质，让“玻璃感”不只停留在外壳
  - 结果卡背后补上一层可透出的暖色 ambient backdrop，并下调遮罩压暗强度，让玻璃卡不再只是“半透明白卡”，而是真能透出后方色层
  - 结果卡弹层继续补齐整屏舞台背景：弹层内容区新增全屏暖色 backdrop stage，卡片外的大面积留白不再直接暴露页面白底，空白处点击也仍可直接关闭弹层
  - 结果卡整屏舞台背景透明度继续上调，遮罩和 ambient backdrop 都再放轻一档，让背景更通透、不再压得太实
  - 结果卡整屏舞台背景的底层铺色继续减弱，只保留暖色光斑做氛围焦点，进一步减少“整片奶白蒙层”感
  - 结果卡整屏舞台背景继续降整体透明度，不改光斑分布，只让后方留白再更通透一档
- 修复首页状态切换反馈与列表动效的两个体验问题：
  - `想吃 / 吃过` 完成态提示现在只会显示在“美食库”页，切去“厨房”页时会立即清掉，避免反馈串页
  - 菜谱列表改为通过卡片动画类切换来重播入场动效，不再依赖整段列表重建，筛选和模式切换时的闪动更轻
- 修复首页与菜谱详情里部分成功提示“知道成功了，但感知不够强”的问题：
  - 首页现在会把菜单详情回跳后的 `已带出这天菜单 / 草稿已删除 / 安排已删除` 用统一的轻量完成态提示接住，不再只弹普通 toast
  - 菜谱详情里的 `已加入整理队列 / 已加入生成队列` 也改成同一套提示层，提交成功后更容易感知下一步会在后台继续处理
- 修复点菜模式里“同日期只有备注草稿时，修改菜单会误带出空草稿”的问题：
  - 后端现在只有在同日期草稿里已经有菜品时，才会直接继续那份草稿
  - 若同日期只是 note-only 草稿，点击“修改菜单”会重新带出已安排菜单，避免用户误以为菜丢了
- 修复个人资料页“提示资料已更新，但昵称和头像实际未生效”的回归问题：
  - 后端将“登录态补资料”和“用户主动改资料”拆分为两条更新策略，主动保存资料时允许替换已有的非占位昵称
  - 后端登录补资料不再用微信侧头像覆盖用户已经手动保存过的头像，只会补齐缺失头像
  - 前端登录态自动资料同步同样收窄为“只补缺失资料”，避免保存后立刻被微信侧资料回写覆盖
- 修复步骤图 worker 空闲时只补首次生成、不会继续利用失败任务的问题：
  - 自动补位仍优先挑选从未生成过步骤图的菜谱
  - 若当前没有可用的首次生成候选，会继续回补 `flowchart_status=failed` 且尚未生成步骤图的菜谱
  - 自动回补重新入队时会清空旧的 `flowchart_error`、刷新 `flowchart_requested_at`，但仍不改 `recipe.updated_at`
- 修复“添加菜品”弹层每次打开都会重复自动带入同一条剪贴板分享文案的问题：
  - 恢复记录最近一次自动带入的剪贴板内容，相同分享文案不会在之后每次打开弹层时反复覆盖输入框
- 修复“补充信息（可选）”折叠摘要把默认分类和状态也算成“已填写”的问题：
  - 折叠态计数现改为只统计相对初始草稿真正新增或改动的内容，首屏提示不再出现刚打开就 `已填 2/4`

### Notes

- 修改时间：2026-04-04 19:40 CST
- 变更背景：当前“菜单详情”只能查看和继续安排，已提交菜单缺少修改/删除入口，草稿删除也只能绕回点菜模式，实际体验和用户预期不一致；同时日期选择、提交成功反馈和菜单回看信息密度偏弱，用户需要反复点开确认状态，重操作继续塞在底部弹层里也不利于理解；另外步骤图 worker 在线上偶发失败后会停在 `failed`，空闲时不会继续利用失败任务，导致生图能力有空转窗口；本次又补充了两轮图片口径调整：保存菜品后若后端异步自动解析命中链接图片，不应因用户已手动上传图片而放弃补图；而在继续 `重新解析` 时，也需要避免旧解析图无限叠加、同图不同 URL 反复残留
- 核心改动：在首页菜单详情入口补上 `修改菜单 / 删除安排 / 删除草稿` 闭环；“修改菜单”改为先从已提交菜单复制出同日期草稿，再进入现有点菜模式继续编辑，原已提交菜单会保留到用户重新提交时再覆盖；若同日期已经有草稿，则直接继续草稿，避免覆盖；同时继续优化点菜体验，在日期选择器补上 `草稿中 / 已安排 / 待修改` 状态提示，在菜单详情、购物车和确认菜单页补上缩略图、餐别和状态说明，并在提交后新增成功反馈面板与明确的下一步动作；菜单详情现已独立为 `pages/meal-plan-detail/index.vue`，首页安排卡和成功态会直接进入该页，详情页内再承载删改和继续安排动作，首页旧的菜单详情弹层实现已下线；本轮首页还进一步收口了动效语言，为菜单 spotlight 增加方向感切换动画，为点菜模式切换补齐页面与底部导航的整体过渡，并为筛选后菜谱列表补上轻量交错入场；这次继续补齐了首页交互反馈层，为 `想吃 / 吃过` 状态切换新增轻震动与页面瞬时完成态提示，并为搜索框、底部 `添加` FAB、点菜悬浮条和卡片状态切换器补上更明确的按压反馈；同时把首页 `添加菜品` 弹层继续收口为更明确的主路径交互，打开时会静默读取剪贴板，只要内容里带支持链接就自动带入；若剪贴板里是整段分享文案，也会原样保留在 `菜谱链接` 字段里，保存前不再静默替换成裸链接，避免最终落库内容和输入框不一致；这轮又把弹层从固定大高度进一步收成“短内容自适应、展开后再进入固定滚动”，首屏回到只保留 `菜谱链接`、`菜名` 和 `补充信息（可折叠）` 三块，footer 也会直接解释当前离可保存还差什么；后续又恢复了“相同剪贴板分享文案只自动带一次”的抑制，并把 `补充信息（可选）` 的折叠计数改为基于初始草稿的相对变化，避免新增弹层反复覆盖输入框或一打开就显示默认项已填写；后端新增相应 `mealplan` 管理接口与测试，并补上“note-only 草稿不拦截修改菜单”的边界修复；本次还把首页状态切换收口为纯状态更新，不再改写 `recipe.updated_at`，避免列表因轻操作自动前移；这次继续把前端状态切换改为本地先响应、成功后只回写单条菜品，避开整列表刷新和封面缓存重跑带来的延迟感；同时把首页进入详情的前置高亮更新去掉，并把详情页的编辑弹层改成懒挂载、首屏未加载状态改为骨架屏，减少跳页前后的多余渲染；本次还把首页菜谱卡片的 `置顶` 从图片角标收成卡片右上角的小书签式状态标记，并继续压缩为更扁、更克制的比例，让它和来源角标、标题内容彻底分层，减少信息区的占位感；这次还调整了步骤图 worker 的空闲补位策略，保留“首次生成优先”口径的同时，在没有可用首轮候选时继续回补 `flowchart_status=failed` 且尚未出图的菜谱；本次同步调整后端菜品自动解析的图片合并策略：异步解析命中链接图片时，会保留当前已有图片顺序，并把去重后的解析图片追加到后面，避免用户手动上传图片后链接图被整段跳过；在此基础上，后端菜品图片现在新增 `image_meta_json` 元数据列，上传和转存时会顺手计算 `SHA-256` 内容 hash，`重新解析` 时会移除旧的 `parsed` 图片并用本轮解析结果替换，图片转存落地后再按内容 hash 去重，尽量消除“同图不同 URL”的重复残留；README 和点菜原型文档同步更新
- 影响范围：`components/action-feedback.vue`、`pages/index/index.vue`、`pages/index/meal-order.js`、`pages/index/components/add-recipe-sheet.vue`、`pages/index/components/add-recipe-sheet.scss`、`pages/index/components/library-header-section.vue`、`pages/index/components/library-header-section.scss`、`pages/index/components/recipe-card-item.vue`、`pages/index/components/recipe-card-item.scss`、`pages/index/components/random-pick-sheet.vue`、`pages/index/components/random-pick-sheet.scss`、`pages/index/components/meal-order-cart-sheet.vue`、`pages/index/components/meal-order-checkout-sheet.vue`、`pages/index/components/meal-order-date-sheet.vue`、`pages/index/components/meal-order-sheet.scss`、`pages/index/components/meal-order-success-sheet.vue`、`pages/meal-plan-detail/index.vue`、`pages/recipe-detail/index.vue`、`pages.json`、`README.md`、`utils/meal-plan-api.js`、`backend/internal/app/router.go`、`backend/internal/app/app.go`、`backend/internal/mealplan/*`、`backend/internal/recipe/*`、`backend/internal/upload/*`、`backend/migrations/013_add_recipe_image_meta.sql`、`backend/README.md`、`docs/meal-order-mode-prototype-v1.md`
- 兼容性/风险：本次新增了菜单管理接口，前后端需同时更新后功能才完整；“修改菜单”目前走“复制为草稿再编辑”策略，能避免误覆盖已安排菜单，但共享厨房里仍未实现更细粒度的并发冲突提示；首页新增的动效主要基于 CSS 位移、透明度、轻量延迟与短时反馈层，理论上性能压力有限，但小程序端仍需在真机上确认不同机型下的流畅度、滑动手势与列表滚动是否互相干扰；状态切换的轻震动在不支持 `vibrateShort` 的环境下会静默降级，不影响主流程；状态切换接口现在不再改写 `recipe.updated_at`，若后续需要展示“最近切换状态时间”，需改为单独字段承载而不是继续复用内容更新时间；前端状态切换现已采用乐观更新策略，若接口失败会回滚本地显示并提示错误，但极端弱网下仍建议补一轮真机连点验证；新增的统一反馈层目前已接到首页和菜谱详情，后续若继续扩到复制、保存等高频动作，需要控制触发频率，避免页面被成功提示刷屏；首页 `帮我选` 结果卡当前优先遵循当前可见筛选，在 `全部` 下会进一步偏向当前可见的 `想吃` 菜品，若后续希望支持“只抽早餐 / 只抽主食 / 忽略已吃过”的更多策略，需要把抽取规则显式做成可配置选项；详情页编辑弹层现改为懒挂载，理论上会减轻首屏进入成本，但仍需在真机确认首次打开编辑弹层时的展开流畅度；步骤图 `failed` 任务现在会在队列空闲时自动重试，若存在长期不可恢复错误，当前仍会按空闲节奏持续回补，后续如需更强约束可再补失败次数或退避策略；本次图片合并新口径会让部分菜品在异步解析完成后新增更多图片，若已有图片数量接近上限，则链接解析图片会按顺序截断到现有限额；新 migration 会把历史图片先回填为 `legacy` 来源，历史数据只有在后续编辑、重解析或转存后，才会逐步补齐更精确的来源归因与内容 hash；`添加菜品` 弹层这轮恢复为打开后自动识别剪贴板支持链接，并进一步改成短内容自适应高度，小程序端仍需补一轮真机验证，确认剪贴板授权提示、自动填入时机、弹层拖拽关闭、展开折叠后的滚动边界和上传图片区在不同机型上的表现都稳定；前端当前没有自动化校验脚本，日期状态卡、小图加载、独立菜单详情页跳转、成功面板和首页新动效仍需在微信开发者工具或真机上补一轮完整操作流验证
- 验证情况：已执行 `@vue/compiler-sfc` 对首页、详情页、共享反馈组件与随机结果卡 SFC 做模板解析校验；已执行 `node --check` 对首页与详情页相关组件的 `<script>` 片段做语法校验；已执行 `git diff --check`；本次已补充首页状态切换链路、统一反馈层接入、详情页进入链路与 `帮我选` 结果卡交互的代码级静态自检；已执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/recipe -run 'TestRepositoryRequeueStaleFlowcharts|TestFlowchartWorkerEnqueueAutoCandidates|TestRepositoryQueueFlowchartDoesNotTouchUpdatedAt|TestRepositoryApplyFlowchartResultDoesNotTouchUpdatedAt'`；已执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/recipe/...`，并补充覆盖“现有图片在前、解析图片补后、重复图片去重”的仓储层回归测试；本轮追加执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./...`，确认图片 metadata、上传 hash、迁移与应用装配链路可以一并通过；本轮继续对 `pages/index/index.vue` 与 `pages/index/components/add-recipe-sheet.vue` 再次执行 `@vue/compiler-sfc` 模板解析校验，并补充 `pages/index/storage.js` 的脚本语法检查；前端当前仍无可直接执行的自动化测试，尚未做 HBuilderX / 微信开发者工具实机预览

## 2026-04-03

### Changed

- 首页“美食库”列表卡片升级为更统一的暖色轻立体风格：
  - 菜谱卡片新增按 `想吃 / 吃过` 的轻量状态底色氛围，信息层级调整为“信息眉标 + 菜名 + 摘要”
  - 卡片内状态切换器从通用滑块改为更简洁的图标胶囊切换器，并按反馈移除控件内部额外文案
  - `置顶` 标识从标题行内迁移到图片角标，减少长标题被挤压
- 首页顶部 `想吃 / 吃过` 筛选胶囊同步调整为和卡片状态一致的暖棕 / 灰绿语义风格
- 修复首页状态筛选里 `全部` 标签因缺少对应激活态样式而出现“文字看不见”的回归问题
- 首页列表标题右侧 `帮我选` 按钮升级为更精致的次级动作样式，与当前暖色轻立体语言保持一致
- 详情页视觉层级同步升级为和首页一致的暖色轻立体语言：
  - 顶部标题区改为 `餐别 / 状态 / 置顶` 胶囊标签 + 更清晰的标题摘要层级
  - 摘要区改为带装饰性引号和左侧强调线的引用卡风格，不使用系统 emoji 和额外文字标签
  - `一图看懂 / 做法整理 / 来源链接 / 备注` 重新区分主次卡片强度，步骤与食材条目可读性增强
  - 底部 `删除 / 置顶 / 编辑` 操作栏重新调整按钮体量和语气，突出 `编辑` 主操作
- 新增 / 编辑菜品弹层补齐统一的表单视觉语言：
  - `添加菜品` 弹层重构为“快速入库主路径 + 补充信息”两段式层级，链接识别提示改为更明确的状态反馈
  - `编辑菜品` 弹层保持原有结构，但补齐输入框、分组块、图片区和底部按钮的质感统一
  - 新增 / 编辑弹层里的重复解释和内部实现话术同步收短，降低阅读负担

### Notes

- 修改时间：2026-04-03 00:49 CST
- 变更背景：首页列表和详情页主内容已经升级到更统一的暖色轻立体语言，但“添加菜品 / 编辑菜品”弹层仍偏标准表单观感，快速入库主路径和编辑态的视觉层级都不够明确
- 核心改动：重做首页菜谱卡片的标题区、状态切换器和置顶标识位置，让卡片和顶部“美食库”区域共享同一套暖色、轻立体、带语义状态色的视觉语言；按界面反馈移除切换器内部文字，并在多轮调整后把控件尺寸、图标可见性、`吃过` 绿色对比和 thumb 阴影收敛到更平衡的状态；补齐状态筛选 `all` 分支的默认态和激活态样式；列表标题右侧 `帮我选` 也升级为更有层次的次级动作按钮；详情页则补上状态胶囊标签、去文字标签化的摘要引用卡、主次卡片强度区分、食材步骤条目优化和底部操作栏层级重排；新增菜品弹层改为“主路径 + 补充信息”两段式结构，链接/标题识别提示从灰字 hint 升级为状态反馈卡，编辑菜品弹层也同步收口输入块、分组卡和底部按钮质感，并把重复解释和偏内部实现的文案压缩为更直接的用户语言
- 影响范围：`pages/index/components/recipe-card-item.vue`、`pages/index/components/recipe-card-item.scss`、`pages/index/index.vue`、`pages/index/components/add-recipe-sheet.vue`、`pages/index/components/add-recipe-sheet.scss`、`pages/recipe-detail/index.vue`
- 兼容性/风险：本次主要是前端样式和局部结构调整，不涉及接口契约；由于未在真机上逐机型验证，小屏设备上状态切换器宽度、详情页标题换行、弹层首屏可见高度和底部按钮文案长度仍需实际确认
- 验证情况：已完成代码级静态自检与 `git diff --check`；当前仓库无可直接执行的前端自动化测试脚本，尚未做 HBuilderX / 微信开发者工具实机预览

## 2026-04-02

### Changed

- 来源链接识别策略调整为“规则提取 `platform/url` + 模型清洗 `title`”：
  - 后端 `linkparse` 的平台识别、URL 提取与归一化继续使用现有规则链路
  - 链接标题现为模型优先清洗，若模型不可用、超时或返回空结果，再回退到规则清洗
- 新增菜品弹窗的链接预览会补充标题来源提示：
  - 预览接口新增 `titleSource`
  - 前端会明确展示当前菜名来自 `AI 智能提取` 还是 `规则提取`
- `AI title` 模型配置补充支持独立的 `baseUrl / apiKey / model / timeout`
  - 新增 `AI_TITLE_BASE_URL`、`AI_TITLE_API_KEY`
  - 若标题专用配置为空，会分别回退到全局 `AI_BASE_URL`、`AI_API_KEY`、`AI_MODEL`
- `AI title` 请求参数补充支持独立配置 `stream / temperature / max_tokens`
  - 新增 `AI_TITLE_STREAM`、`AI_TITLE_TEMPERATURE`、`AI_TITLE_MAX_TOKENS`
  - 默认值分别为 `false`、`0`、`64`
- 前端新增菜品弹窗的链接预览不再强依赖本地平台识别命中，疑似分享文案也会继续请求后端预览
- 前端在链接预览阶段保留用户原始粘贴内容不变，只在提交保存前静默规范化为后端返回的标准链接，降低“存进去的是分享文案而不是来源链接”的概率
- 前端来源平台展示文案统一为 `B站 / 小红书`，不再区分 `小红书视频 / 小红书图文`
- 后端步骤图 worker 新增“空闲自动补位”能力：
  - 新增配置 `RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED`
  - 当当前没有 `pending / processing` 任务时，会自动补 1 条“未生成步骤图且已完成做法整理”的菜谱入队
  - 第一版仅处理首次生成，不自动重试 `failed`，也不自动重生成已有但过期的步骤图
- 生产环境已在 `backend/configs/prod.env` 启用 `RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED=true`，`caipu-backend` 重启后按空闲策略自动补位步骤图任务
- 步骤图队列状态与生成结果不再改写 `recipe.updated_at`，避免后台任务打乱首页菜谱排序

### Fixed

- 修复个人资料页选择微信头像后“提示资料已更新但实际头像未生效”的问题：
  - 前端上传逻辑不再把 `chooseAvatar` 返回的微信临时头像路径误判为远程图片
  - 登录态自动资料同步会忽略临时头像路径，避免把无效地址再次写回后端
  - 后端资料更新接口会拒绝临时头像路径，避免旧客户端继续写入无效头像地址
- 修复小红书封面图在首页/详情页不稳定显示的问题：
  - 后端图片转存候选查询会跳过已是 `/uploads/*` 的记录，避免未转存外链被旧数据长期占满扫描窗口
  - 首页封面在本地缓存下载成功后会自动解除 `hidden / fallback` 状态，避免“先报错后缓存成功”仍然一直不显示
  - 详情页主图改为本地缓存优先，缓存失效后回退远程图，远程图再次失败时展示“查看原图”占位而不是白块

### Notes

- 修改时间：2026-04-03 00:07 CST
- 变更背景：现有来源链接提取主要依赖前后端规则，面对不规整分享文案时稳定性不足；同时小红书图片仍存在“外链未及时转存导致首页缺图、详情页白块”的稳定性问题
- 核心改动：保留规则提取来源链接，模型只参与低置信度标题清洗；标题模型现已支持独立配置地址与密钥；前端来源标签统一展示为 `B站 / 小红书`；后端步骤图 worker 新增空闲自动补位策略，并把步骤图状态流转从普通内容更新时间里剥离；个人资料头像上传现会正确识别并上传微信临时头像路径，后端资料更新接口也会拒绝继续写入临时头像地址；后端图片转存候选查询现会优先命中真正仍是外链的菜谱，详情页主图也补齐了缓存回退与失败占位
- 影响范围：`pages/index/index.vue`、`pages/index/recipe-card.js`、`pages/recipe-detail/index.vue`、`utils/auth.js`、`utils/upload-api.js`、`backend/internal/auth/service.go`、`backend/internal/auth/service_test.go`、`backend/internal/linkparse/*`、`backend/internal/recipe/*`、`backend/internal/config/config.go`、`backend/README.md`
- 兼容性/风险：当前仍只支持 `bilibili` / `xiaohongshu` 两个平台；若运行环境未配置 AI 模型，标题会完全沿用规则清洗结果；步骤图自动补位默认关闭，启用后会带来额外图片生成成本；第一版不会自动重试失败任务，也不会自动重生成已有但过期的步骤图；头像临时路径识别当前覆盖 `wxfile://`、`file://`、`blob:` 和 `http(s)://tmp/`；旧前端若仍直接提交临时头像路径，现在会收到明确的 `400` 错误而不是“假成功”；详情页主图在极端情况下会退化为“查看原图”占位，但不会继续渲染成纯白块
- 验证情况：已执行 `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/recipe/...`；已执行 `git diff --check`；已完成详情页主图缓存回退与失败占位静态代码自检；已补充图片转存候选筛选回归测试

## 2026-03-12

### Added

- 初始化 `backend/` Go 项目骨架，包含配置加载、迁移执行、`healthz` 和 SQLite 初始化
- 新增 `go run ./cmd/seed-demo`，可重复填充本地联调用的厨房、成员和菜谱样例数据
- 接入 `auth + kitchens` 闭环：
  - `POST /api/auth/wechat/login`
  - `POST /api/auth/dev-login`（仅本地环境）
  - `GET /api/auth/me`
  - `GET /api/kitchens`
  - `POST /api/kitchens`
  - `GET /api/kitchens/{kitchenID}/members`
- 接入 `recipe` 闭环：
  - `GET /api/kitchens/{kitchenID}/recipes`
  - `POST /api/kitchens/{kitchenID}/recipes`
  - `GET /api/recipes/{recipeID}`
  - `PUT /api/recipes/{recipeID}`
  - `PATCH /api/recipes/{recipeID}/status`
  - `DELETE /api/recipes/{recipeID}`
- 接入 `invite` 闭环：
  - `GET /api/invites/{token}`
  - `POST /api/kitchens/{kitchenID}/invites`
  - `POST /api/invites/{token}/accept`
- 接入 `upload` 闭环：
  - `POST /api/uploads/images`
  - `GET /uploads/*`

### Changed

- `kitchen` 模块新增成员校验能力，供 `recipe` 访问控制复用
- `backend/README.md` 从项目起始说明更新为可直接联调的说明文档
- `backend/README.md` 进一步补充了邀请接口、默认策略和联调示例
- 前端 `utils/recipe-store.js` 已改为“本地缓存 + 远端 API”模式
- 前端新增 `utils/auth.js`、`utils/http.js`、`utils/kitchen-api.js`、`utils/recipe-api.js`、`utils/upload-api.js`
- 前端首页已接入厨房切换器和邀请成员入口，并新增 `pages/invite/index.vue` 处理邀请预览与接受加入
- 前端“厨房”页已接入成员面板，可按当前厨房展示成员列表和自己的角色
- 正式微信登录链路补充了 `appId` 透传与校验，前端也支持显式切换 `dev / wechat / auto` 登录模式
- `.gitignore` 已覆盖 SQLite 运行产物、本地环境文件、备份和覆盖率输出

### Notes

- 第一版 `recipe` 采用软删除
- 第一版 `recipe` 默认按 `updated_at DESC` 排序
- 当 `parsedContent` 为空时，后端会生成兜底的食材和步骤结构，保证前端始终拿到可渲染数据
- 第一版 `invite` 允许任意厨房成员生成邀请，默认 `72` 小时过期、默认最多使用 `10` 次
- 同一用户重复接受同一厨房邀请时会幂等返回，不重复占用邀请次数
- `UPLOAD_PUBLIC_BASE_URL` 为空时，上传接口会按当前请求域名自动返回图片 URL
- 当前前端默认使用 `utils/app-config.js` 里的本地开发地址，并在本地后端环境下自动走 `dev-login`
