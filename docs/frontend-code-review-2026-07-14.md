# 前端代码审查报告

- **日期**：2026-07-14
- **范围**：最近两次前端模块化重构（`7353241` / `55e3273`）落地的小程序前端代码 —— `pages/index`、`pages/recipe-detail` 拆出的 `use-*.js` hooks / 组件 / SCSS，以及新增的 `utils/recipe-model.js`、`recipe-repository.js`、`recipe-cache.js`、`count-up.js`、`diet-assistant-*.js`。
- **方法**：复用 / 简化 / 效率 / 抽象层次 4 维度并行审查，逐条源码核实。仅出报告，未改动代码。

---

## 总体评价

**这轮重构的收口层（`utils/`）质量很高**：`recipe-model.js` 持有纯领域归一化、`recipe-repository.js` 是唯一写收口、`recipe-cache.js` 只管本地存储，`count-up.js` / `diet-assistant-sse.js` / stream-decoder 都是依赖注入的纯机制，分层干净。**问题集中在页面层（`pages/index/*`）**：没有把同样的收口思路用到「图片缓存」和「空间切换」上，导致重复实现、热路径浪费和死代码。

---

## 🔴 A. 需先确认的正确性隐患（超出本次优化范围，建议 `/code-review`）

| # | 位置 | 问题 | 核实结论 |
|---|------|------|----------|
| A1 | `pages/index/use-kitchen-space.js:183-191` | `memberRoleLabel/DisplayName/Initial` 方法体调用 `formatMemberRoleLabel/DisplayName/Initial`，全仓库无定义/无 import；同文件 45-57 真正定义的是不带 `format` 前缀的模块函数（且无人用）。这些方法被 `pages/index/components/kitchen-section.vue:98` 模板调用。 | grep 确认 `formatMember*` 零定义；渲染成员卡片时会 `ReferenceError` |
| A2 | `pages/index/use-recipe-library.js:568-600` | `applySession` 用了 `getSessionSnapshot()`（默认参数）和 `getCachedPlaces()`，但本文件未 import 二者（`getSessionSnapshot` 只在 `use-kitchen-space.js` import 了）。该方法被 index.vue / use-kitchen-space.js 多处调用。 | grep 确认本文件无对应 import；执行到即 `ReferenceError` |

> 「重命名/搬运只改了一半」的重构残留。之所以能写出来，是因为这些 hook 实际是**共享同一个 `this` 的 mixin**，作者默认「符号从哪来无所谓」——这也是下面 D1 抽象层次问题的直接佐证。

---

## 🟠 B. 高优先级优化

### B1. 图片本地缓存编排被复制成 3 份 —— 底层机制没做够

*（复用 + 抽象层次 + 效率 三个维度共同指向）*

同一套「按 cacheKey 预热本地缓存 → 命中回填 → 失败降级远端再隐藏 → requestID 防竞态」的状态机写了三遍，仅 map 命名不同：

| 消费方 | 位置 | map key |
|--------|------|---------|
| 首页封面（内联在 hook） | `pages/index/use-recipe-library.js:499-566` + `393-440` | `recipeId` |
| 详情页大图（已抽成 controller） | `pages/recipe-detail/use-recipe-images.js:31-86` | `cacheKey` |
| 详情页流程图（内联在页面） | `pages/recipe-detail/index.vue:852-935` | 单值 |

**代价**：改一次降级策略/并发数/时序守卫要同步 3 处，极易漏。

**方向**：`utils/image-cache.js` 目前只暴露 `getCachedImagePath/warmImageCache/invalidateCachedImage` 原子。把「一组远端 URL → 带 fallback/hidden 的可显示 map + 背景预热 + `handleError`」封装成一个 controller（key 策略参数化），三处各持一个实例。详情页的 `createRecipeImageController` 已是很好的雏形，泛化上移即可。

### B2. `recipeCards` 计算属性在封面预热期被整表重建（O(N²) 热路径）

`buildRecipeCard`（`pages/index/recipe-card.js:134`）把 `cover` 内嵌进卡片对象 → `recipeCards` computed（`pages/index/use-recipe-library.js:721`）依赖 `cachedRecipeCoverMap` → `warmImageCache` 回调每命中一张封面就 `this.cachedRecipeCoverMap = {...}`（`use-recipe-library.js:561`），冷启动 N 张封面触发约 N 次全列表重算，每次对每条 `filteredRecipes` 重跑 `buildRecipeCard`，并引发列表 setData 抖动。

**方向**：`buildRecipeCard` 不再内嵌 `cover`，`recipeCards` 只依赖 `filteredRecipes`（稳定）；封面渲染时由已存在的 `getRecipeCardDisplayCover(card)`（`use-recipe-library.js:393`）直接读 map。这样预热只更新单元格封面，不重建整表。

### B3. `isStepCompleted(index)` 每次调用都重建整条 key 列表（O(N²) JSON.stringify）

`isStepCompleted`（`pages/recipe-detail/index.vue:1234`）是 method（非 computed），模板每步调用 3 次，每次都 `buildCurrentStepCompletionKeys → buildStepCompletionKeyList(this.parsedSteps)`（`index.vue:1212`），而后者对每步做一次 `JSON.stringify`（`pages/recipe-detail/use-recipe-edit.js:115`）。N 步 → 每次面板重渲染约 3N² 次序列化；`completedStepCount`（`index.vue:641`）又独立重建一次。

**方向**：把 `currentStepCompletionKeys` 提成 computed（随 `parsedSteps` 缓存一次），`isStepCompleted`/`completedStepCount` 只按下标读缓存数组。

### B4. 两个「智能识别」预览面板几乎逐字节复制

`pages/index/components/add-link-preview-panel.vue` 与 `add-recipe-preview-panel.vue`：`data/created/parsingText/methods` 结构一致，`<style>` 约 290 行 SCSS 仅差 `.capability-card__icon` 3 行颜色类，其余逐字节相同。

**方向**：抽一个 `AddPreviewPanelBase`，用 props 传标题/图标文案/`capabilities`/解析函数（`parseShareText` vs `parseRecipeLink`）/stage 文案表；recipe 面板多出的 `hasParseableShareHint` 校验作为可选行为。至少也应把公共 SCSS 抽成 `add-preview-panel.scss` 供两者 `@import`。

---

## 🟡 C. 中优先级：死代码 / 重复实现清理

| # | 位置 | 问题 | 处理 |
|---|------|------|------|
| C1 | `pages/index/index.vue:764` + `use-kitchen-space.js:94/104/109/124` + `use-recipe-library.js:592` | 死状态 `spaceStatsRemoteError`：6 处写、**0 处读**（grep 确认） | 删声明 + 5 处赋值 |
| C2 | `pages/index/index.vue:672` + `use-smart-add.js:147/199` | 死状态 `draftLinkPrefillSource`：只被赋 `''`、从不读、从无非空值 | 删声明 + 2 处赋值 |
| C3 | `pages/index/use-kitchen-space.js:45-57` | 模块函数 `memberRoleLabel/DisplayName/Initial` 无人 import（死代码，与 A1 是同一处半成品重命名） | 与 A1 一并修（让方法直接调这三个模块函数即可，同时消除 A1 的 ReferenceError） |
| C4 | `buildRecipeImageVersion`（`pages/recipe-detail/use-recipe-images.js:4`） | 与 `buildRecipeCoverVersion`（`pages/index/recipe-card.js:90`）逐字重复 | 直接 import 复用，或上移到 `recipe-model.js` |
| C5 | 取图逻辑 4 处各写一份 | `utils/recipe-model.js:292`、`utils/recipe-repository.js:30`、`pages/recipe-detail/use-recipe-images.js:8`、`pages/recipe-detail/index.vue:400` 都在做 `imageUrls?.length ? imageUrls : [image,imageUrl]`，且优先级已开始漂移（详情页漏了 `recipe.images` 分支，与规范实现 `extractRecipeImages`（`recipe-card.js:38`）不一致） | 在 `recipe-model.js` 抽 `getRecipeImageSources(recipe)`，四处统一调用 |
| C6 | 反馈控制器两套实现 | 详情页已抽成 `createActionFeedbackController`（`pages/recipe-detail/use-recipe-detail-state.js:7`），首页却把同逻辑内联成 `showLibraryActionFeedback` + 一堆 `recipeStatusFeedback*` 字段（`use-recipe-library.js:175`） | 首页复用该 controller，`showSparkles` 作可选扩展 |
| C7 | `pages/recipe-detail/use-recipe-edit.js:68` | `buildRecipeEditPayload` 内联的 `normalizeTextList` 与模块级 `comparableTextList`（`use-recipe-edit.js:47`）等价 | 删内联版，复用 `comparableTextList` |

---

## 🟢 D. 抽象层次 / 低优先级

- **D1 假模块化（根因）**：`pages/index/use-recipe-library.js:568` 的 `applySession` 是「换空间」的跨模块编排器，却挂在菜谱库模块名下，直接伸手重置 place / meal-order / space-stats / kitchen 四个模块的私有 state。每加一处跨模块状态就要回来塞一行 special-case，形成越长越脆的上帝函数。`pages/index/page-module.js` 已有 `lifecycle`（deactivate/dispose）派发机制，**建议同构加一个 `onKitchenChange(prevId, nextId)` 钩子**，让各模块各自实现重置，`applySession` 只广播 kitchenId 变化。这同时能消解 A2 的符号错位。
- **D2 组件越权调 API**：`pages/index/components/add-recipe-preview-panel.vue:100`、`add-link-preview-panel.vue`、`diet-assistant-sheet.vue` 直接 `import previewAddLink` + `getCurrentKitchenId()`，把「取空间→拼请求→stage 编排」塞进叶子组件，违反 CLAUDE.md「改数据逻辑优先走收口层」。方向：组件只 `emit('paste', text)`，请求侧统一回 `use-smart-add`。
- **D3 效率零头**：
  - `refreshSpaceStats`（`pages/index/use-kitchen-space.js:120`）三条互不依赖的网络请求顺序 await → 可 `Promise.all` 并发；
  - `mealOrderCartItems`（`pages/index/use-meal-order.js:535`）为个位数购物车项对全量 recipes 建查找表 → 改为逐项 `find` 或优先用 snapshot；
  - count-up（`pages/index/components/space-stats-card.vue`）6 个统计项各起一个 40ms `setInterval`（~96 次 setData）→ 控制器支持「一批 key 共用一个定时器、每帧合并一次 write」。
- **D4 分散小重复**：
  - 日期 `padStart` 拼接手写于 4+ 文件（`pages/recipe-detail/use-recipe-async-jobs.js:75` 等）→ 可抽 `formatDateTime`；
  - 防抖/一次性定时器骨架（`createSearchBlurController` 等 8+ 处）各写各的 → 可抽 `createDebouncedTask`；
  - `stringifyRecipeParsedContent`（`pages/index/use-smart-add.js:15`）在页面层二次硬编码了 parsedContent schema → 应放到 `recipe-model.js` 旁；
  - `applyPlaces`（`pages/index/use-place-library.js:92`）对已归一化的缓存二次 `normalizePlaceList`。

---

## 建议落地顺序

1. **先修 A1 + A2 + C3**（正确性隐患，建议 `/code-review` 复核后修）——一处半成品重命名同时牵出 ReferenceError 和死代码。
2. **B2 + B3**（两条 O(N²) 热路径，改动局部、风险低、收益直接）。
3. **B1**（图片缓存三合一 controller，收益最大但改动最重，需专门一轮）。
4. **C1/C2/C4/C7**（零风险删除/复用，可顺手清）。
5. B4、C5/C6、D 系列按精力推进。

---

## 已排查、层次合理（无需处理）

- `utils/recipe-store.js` / `recipe-model.js` / `recipe-repository.js` / `recipe-cache.js`：分层干净——store 是薄 barrel，model 持有纯领域逻辑，repository 是唯一写收口，cache 只管本地存储；跨 kitchen 维度由 `kitchenId` 一致贯穿。本次审查中层次最正确的一块。
- `utils/count-up.js`、`diet-assistant-sse.js`、`diet-assistant-stream-decoder.js`（含手写 UTF-8 兜底解码）：纯机制、依赖注入，无领域耦合。
- 网络请求均复用 `http.js`；`streamDietAssistantChat` 用裸 `uni.request` 是为了 `enableChunked`/`onChunkReceived`，属合理绕过。
- `page-module.js` 的 duplicate 守卫 + `requires` 校验 + lifecycle 派发机制本身正确（问题在没被用于 kitchen-switch，见 D1）。
- recipe-detail 下 `components/*.vue` 多为纯展示、无 API/cache/store import；`recipe-edit-sheet.vue` 正确委托 `use-recipe-edit.js`（越权调用集中在首页两个 preview panel 和 diet-assistant-sheet，见 D2）。
