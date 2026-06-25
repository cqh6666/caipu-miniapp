# 空间统计能力设计

修改时间：2026-06-25 19:30:00 +0800 CST

适用范围：微信小程序首页、空间页、美食库、打卡点、菜单安排、Go 后端空间相关接口。

## 1. 结论

统计能力建议定位为”空间概览 / 空间洞察”，以当前空间为唯一主维度，而不是分别做
“美食库统计”和”打卡库统计”两个割裂入口。

推荐第一阶段落地：

1. 在底部 `空间` 页的当前空间卡片下方新增”空间概览”卡片。
2. 卡片展示 4 个核心数字：菜品数、打卡点数、已安排菜单天数、本周可选 + 周末可去。
3. 支持**下拉刷新**，主动触发数据同步 + 统计重新聚合，并显示明确的同步状态。
4. 点击”查看洞察”打开半屏详情，分为”总览 / 美食库 / 打卡库 / 菜单安排”四组。
5. **总览 Tab 直接露出”高分复访推荐 Top 3”**，让用户一眼看到”下次可以去哪”。
6. V1 先基于前端已同步的 `recipes`、`places`、`mealOrderStore`、`kitchenMembers`
   本地聚合，不新增后端接口。
7. V2 再补 `GET /api/kitchens/{kitchenID}/stats?window=30d` 后端聚合接口，用于趋势、
   成员贡献、历史行为和大数据量场景。

不建议新增底部第四个”统计”Tab。统计属于低频回看和空间经营感增强，放在 `空间` 页
最符合现有信息架构；美食库和打卡点列表页只保留轻量计数，避免干扰”找菜 / 找店”的
主任务。

## 2. 当前依据

### 2.1 前端结构

当前首页主容器通过 `activeSection` 在 `library` 和 `kitchen` 间切换；`library`
内部再通过 `appMode` 在 `cook` 和 `explore` 间切换：

- `pages/index/index.vue`
  - `activeSection === 'library'`：承载美食库和打卡点。
  - `appMode === 'cook'`：美食库。
  - `appMode === 'explore'`：打卡点。
  - `activeSection === 'kitchen'`：空间页。
- `pages/index/components/kitchen-section.vue`
  - 已展示当前空间、空间身份、成员、邀请成员、加入邀请码和退出空间。
  - 适合承接空间级统计入口。

### 2.2 后端接口与数据归属

现有后端业务接口都围绕 `kitchenID` 组织：

| 能力 | 当前接口 | 说明 |
| --- | --- | --- |
| 空间成员 | `GET /api/kitchens/{kitchenID}/members` | 成员数与成员身份 |
| 美食库 | `GET /api/kitchens/{kitchenID}/recipes` | 菜谱列表 |
| 打卡库 | `GET /api/kitchens/{kitchenID}/places` | 打卡点列表 |
| 菜单安排 | `GET /api/kitchens/{kitchenID}/meal-plans` | 草稿和已提交菜单 |
| 智能添加 | `POST /api/kitchens/{kitchenID}/add-link-previews` | 菜谱 / 打卡地统一识别 |

因此统计口径应统一为“当前空间内的数据快照”。空间切换后，统计也随 `currentKitchenId`
切换。

### 2.3 当前可用字段

美食库可用字段：

| 字段 | 用途 |
| --- | --- |
| `mealType` | 早餐 / 正餐结构 |
| `status` | 想吃 / 吃过结构 |
| `imageUrls` | 图片完整度 |
| `parseStatus` | 菜谱自动解析状态 |
| `flowchartStatus` | 一图看懂生成状态 |
| `createdAt` / `updatedAt` | 近期新增与更新 |

打卡库可用字段：

| 字段 | 用途 |
| --- | --- |
| `status` | 想去 / 去过结构 |
| `visitedAt` | 已打卡时间统计 |
| `revisitRating` | 重访意愿与推荐强度 |
| `recommendedItems` | 推荐菜 / 推荐项沉淀 |
| `latitude` / `longitude` | 地图定位完整度 |
| `scenes` / `tags` | 场景与标签分布 |
| `externalProvider` / `externalPoiId` | POI 匹配覆盖 |
| `createdAt` / `updatedAt` | 近期新增与更新 |

菜单安排可用字段：

| 字段 | 用途 |
| --- | --- |
| `drafts` | 草稿中菜单 |
| `submitted` | 已提交菜单 |
| `planDate` | 已安排日期、未来菜单 |
| `items` | 菜单菜品数量与快照 |

## 3. 产品目标

### 3.1 用户目标

- 进入空间页后，能快速知道这个空间积累了多少菜、多少店、多少菜单。
- **能从统计中直接获得下一步行动**，例如”本周可选 38 道菜””周末可去 12 家店”。
- 能回看这个空间近期是否在持续使用，而不是只看到静态成员列表。
- **能把”去过的店”和”吃过的菜”沉淀成快速决策**，例如”下次聚餐去哪””这家店点什么”。
- **能主动刷新统计数据**，确保看到最新的空间状态（尤其是多成员协作场景）。

### 3.2 产品目标

- 强化”共享空间”的经营感，而不是只把空间页做成设置页。
- 把美食库、打卡库、菜单安排串成一个统一资产面板。
- **让统计从”数据展示”转向”行动建议”**：优先露出高价值推荐（高分店铺、待尝试菜品），并提供一键跳转到对应操作。
- **增强空间活跃度感知**：通过轻量可视化（迷你图表、时间线）让用户看到”这个空间在持续使用”。

### 3.3 技术目标

- V1 不新增数据库迁移和后端接口，优先基于已同步数据前端聚合。
- 聚合逻辑尽量纯函数化，方便后续迁移到后端接口。
- **统计口径必须随空间切换、数据同步、本地缓存状态保持一致，并提供明确的”数据新鲜度”标识**。
- 后续新增后端统计接口时，接口响应结构与前端 V1 数据模型尽量兼容。

## 4. 信息架构

### 4.1 主入口

位置：`空间` 页，当前空间卡片下方，成员面板上方。

推荐文案：

```text
空间概览
这个空间已经沉淀了 38 道菜和 12 个打卡点

菜品      38
打卡点    12
已安排    5 天
本周可选  16 道   ← 替代”待探索”，直接指向行动
周末可去  8 家

[查看洞察]
```

入口规则：

- 当前空间未连接时，隐藏统计卡或展示同步占位。
- 有缓存但同步失败时，展示”本地缓存”弱提示。
- **支持下拉刷新手势**：触发 `recipe-store.js` / `place-store.js` 的同步方法，并在统计卡显示”同步中”状态；同步完成后自动重新聚合统计数据。
- 空空间时展示引导：**”添加第一道想吃的菜，开启你的美食空间”** + 直接弹出添加面板。
- 不在美食库 / 打卡点列表顶部放完整统计，以免挤压列表主任务。

### 4.2 详情层级

V1 建议使用半屏抽屉，而不是新增独立路由。原因是统计详情第一阶段信息量有限，半屏更
轻，且能保留用户对当前空间页的上下文。

详情结构：

```text
空间洞察
更新时间：刚刚同步 / 2 分钟前   ← 明确数据新鲜度

[总览] [美食库] [打卡库] [菜单安排]

总览：
- 空间资产：菜品、打卡点、菜单、成员
- **高分复访推荐 Top 3**（直接露出，配店铺图标 + 评分 + 推荐菜）
- 待行动：本周可选菜、周末可去店、草稿菜单、缺定位打卡点
- 近期动态：近 30 天新增 / 打卡 / 安排（V1 轻量文字，V2 考虑迷你折线图或时间线）
```

V2 若加入趋势图、成员贡献和历史统计，再考虑新增：

```text
pages/space-stats/index.vue
```

## 5. 指标设计

### 5.1 空间总览指标

| 指标 | 口径 | V1 数据来源 | 展示建议 |
| --- | --- | --- | --- |
| 菜品数 | 当前空间全部未删除菜谱数量 | `recipes.length` | 大数字 |
| 打卡点数 | 当前空间全部未删除打卡点数量 | `places.length` | 大数字 |
| 已安排菜单天数 | 已提交菜单记录数量 | `mealOrderStore.submitted.length` | 大数字 |
| 成员数 | 当前空间成员数量 | `kitchenMembers.length` | 大数字 |
| 本周可选 | 想吃菜数量 | `recipe.status === 'wishlist'` | **替代"待探索"，直接指向行动** |
| 周末可去 | 想去打卡点数量 | `place.status === 'want'` | **替代"待探索"，直接指向行动** |
| 高分复访推荐 Top 3 | `revisitRating >= 4` 的店铺，按评分排序 | `places.filter(...)` | **总览 Tab 直接露出**，配图标 + 评分 + 推荐菜 |

### 5.2 美食库指标

| 指标 | 口径 | 价值 |
| --- | --- | --- |
| 早餐 / 正餐数量 | 按 `mealType` 聚合 | 看菜谱结构是否偏科 |
| 想吃 / 吃过数量 | 按 `status` 聚合 | 看待尝试池和已沉淀规模 |
| 图片覆盖率 | 有 `imageUrls` 的菜谱 / 全部菜谱 | 列表质感与识别度 |
| 已解析菜谱数 | `parseStatus === 'done'` 或有结构化步骤 | 看菜谱内容完整度 |
| 可生成步骤图数 | 有结构化步骤但未生成流程图 | 引导后续补齐 |
| 最近新增菜谱 | 近 30 天 `createdAt` | 空间活跃度 |

V1 不建议做“吃过趋势”。当前菜谱只有当前 `status`，没有 `doneAt` 或状态历史，无法
准确知道是哪一天从想吃变成吃过。

### 5.3 打卡库指标

| 指标 | 口径 | 价值 |
| --- | --- | --- |
| 想去 / 去过数量 | 按 `status` 聚合 | 看待探索池和已完成规模 |
| 有定位比例 | 有 `latitude` 和 `longitude` 的地点 / 全部地点 | 判断能否直接导航 |
| 已打卡体验完整度 | 去过且有 `revisitRating` 或 `recommendedItems` | 判断记忆是否可复用 |
| 重访推荐数 | `revisitRating >= 4` | 复访候选 |
| 低分避雷数 | `revisitRating <= 2` | 避免重复踩坑 |
| 推荐项 Top | 聚合 `recommendedItems` | 回答“去这家点什么” |
| 场景标签 Top | 聚合 `scenes` 和 `tags` | 回答“聚餐 / 约会 / 顺路去哪” |
| POI 匹配覆盖 | 有 `externalProvider` 和 `externalPoiId` 的地点数量 | 判断外部数据可刷新程度 |

打卡点统计比美食库更适合先做体验洞察，因为 `visitedAt`、`revisitRating`、
`recommendedItems` 等字段已经能表达行为结果和主观体验。

### 5.4 菜单安排指标

| 指标 | 口径 | 价值 |
| --- | --- | --- |
| 已安排天数 | `submitted.length` | 空间计划使用程度 |
| 草稿天数 | `Object.keys(drafts).length` 且有菜品 | 提醒继续完成 |
| 最近安排 | 最近一个 `submitted` | 回看空间行为 |
| 下一次安排 | `planDate >= today` 的最早菜单 | 提醒即将吃什么 |
| 菜单平均菜数 | 已提交菜单 `items.length` 平均值 | 看菜单规模 |
| 菜单菜品来源 | `items.mealTypeSnapshot` 聚合 | 看早餐 / 正餐使用情况 |

## 6. 页面与交互设计

### 6.1 空间概览卡

建议新增组件：

```text
pages/index/components/space-stats-card.vue
```

组件职责：

- 接收 `stats`、`isSyncing`、`hasKitchen`、`isCacheSnapshot`。
- 展示 4 个核心数字和一条行动建议。
- 点击卡片或“查看洞察”向父级发送 `open-stats`。

建议视觉：

- 延续当前空间页暖白、陶土、鼠尾草绿配色。
- 使用 2x2 数字格，数字比标签更突出。
- 避免复杂图表，优先使用进度条、横向条和轻量标签。
- 使用 `uview-plus` 图标或现有静态 SVG，不用 emoji 当 UI 图标。

### 6.2 空间洞察半屏

建议新增组件：

```text
pages/index/components/space-stats-sheet.vue
```

Tab 结构：

| Tab | 内容 |
| --- | --- |
| 总览 | 资产总量、**高分复访推荐 Top 3**、待行动、近期动态 |
| 美食库 | 餐别 / 状态结构、图片覆盖、解析完整度 |
| 打卡库 | 想去 / 去过、有定位、重访评分分布、推荐项、场景标签 |
| 菜单安排 | 草稿、已提交、下一次安排、平均菜数、**快速安排入口** |

交互规则：

- 统计数字支持点击跳转到对应列表筛选。
- “本周可选”跳到美食库并设置 `appMode='cook'`、`activeStatus='wishlist'`。
- “周末可去”跳到打卡点并设置 `appMode='explore'`、`activePlaceStatus='want'`。
- **”高分复访推荐”点击店铺直接跳到打卡点详情**（如果有独立详情页）或筛选定位。
- “草稿菜单”跳到点菜模式或菜单详情入口。
- **”菜单安排” Tab 底部新增”快速安排下一餐”按钮**：从”本周可选”随机推荐 3 道菜，一键加入草稿。
- 缺定位地点先跳到打卡点列表，后续可支持独立筛选。
- **支持下拉刷新**：在半屏内也可触发数据同步，刷新后自动更新各 Tab 数据。

### 6.3 美食库 / 打卡点列表轻量露出

列表页不放完整统计，只做局部增强：

- 美食库筛选条继续展示餐别数量和状态数量。
- 打卡点状态筛选补充 `全部 / 想去 / 去过` 数量。
- 列表标题根据筛选显示当前结果数。

## 7. 前端实现方案

### 7.1 聚合模块

建议新增纯函数模块：

```text
utils/space-stats.js
```

职责：

- `buildSpaceStats({ recipes, places, mealOrderStore, members, now })`
- 输出稳定结构供卡片和半屏复用。
- 不直接依赖 `uni`、页面实例或接口请求。

推荐数据结构：

```js
{
  updatedAt: '',
  source: 'remote' | 'cache',
  isSyncing: false,  // 新增：标识正在同步
  overview: {
    recipeTotal: 0,
    placeTotal: 0,
    submittedMealPlanDays: 0,
    memberTotal: 0,
    weeklyAvailableRecipes: 0,  // 替代 pendingExploreTotal
    weekendAvailablePlaces: 0,
    topRevisitPlaces: []  // 新增：高分复访推荐 Top 3，格式 [{id, name, revisitRating, recommendedItems, imageUrl}]
  },
  recipes: {
    byMealType: { breakfast: 0, main: 0 },
    byStatus: { wishlist: 0, done: 0 },
    imageCoverage: 0,
    parsedTotal: 0,
    recentCreatedTotal: 0
  },
  places: {
    byStatus: { want: 0, visited: 0 },
    locatedTotal: 0,
    highlyRecommendedTotal: 0,
    lowRatingTotal: 0,
    averageRevisitRating: 0,
    topRecommendedItems: [],
    topScenes: []
  },
  mealPlans: {
    draftDays: 0,
    submittedDays: 0,
    nextPlan: null,
    averageDishCount: 0
  },
  actions: []
}
```

### 7.2 页面接入

`pages/index/index.vue`：

- 新增 computed：`spaceStats`。
- 将 `spaceStats` 传给 `KitchenSection` 或直接在空间页模板插入 `space-stats-card`。
- 新增状态：`showSpaceStatsSheet`。
- 新增方法：
  - `openSpaceStatsSheet`
  - `closeSpaceStatsSheet`
  - `handleSpaceStatsAction`
  - **`refreshSpaceStats`**：触发 `recipe-store.js` / `place-store.js` 同步，期间设置 `spaceStats.isSyncing = true`。

`pages/index/components/kitchen-section.vue`：

- 新增 `stats` prop。
- 在当前空间卡片和邀请成员之间插入 `space-stats-card`，或由 `index.vue` 在
  `kitchen-section` 下方插入。

更推荐第二种：`kitchen-section` 保持”空间身份 + 成员管理”职责，统计卡由
`index.vue` 编排，避免组件继续膨胀。

### 7.3 跳转与筛选联动

统计项点击后的页面动作建议都在 `index.vue` 内处理：

| 动作 | 页面状态 |
| --- | --- |
| 查看本周可选 | `activeSection='library'`, `appMode='cook'`, `activeStatus='wishlist'` |
| 查看吃过菜 | `activeSection='library'`, `appMode='cook'`, `activeStatus='done'` |
| 查看周末可去 | `activeSection='library'`, `appMode='explore'`, `activePlaceStatus='want'` |
| 查看去过点 | `activeSection='library'`, `appMode='explore'`, `activePlaceStatus='visited'` |
| 查看高分店铺 | 跳转到打卡点详情或列表筛选（按 `revisitRating >= 4`） |
| 快速安排菜单 | 从 `wishlist` 随机选 3 道菜，打开 `openMealOrderDateSheet` 并预填 |
| 邀请成员 | 复用 `openInviteSheet` |

## 8. 后端演进方案

### 8.1 V1 不新增接口

V1 使用前端已同步数据聚合，原因：

- 当前首页已经拉取 `recipes`、`places`、`meal-plans`、`members`。
- 统计只做当前快照和轻量分布，前端计算成本低。
- 不涉及数据库迁移，风险小。

### 8.2 V2 新增统计接口

当出现以下需求时，再新增后端接口：

- 近 7 天 / 近 30 天趋势。
- 成员贡献排行。
- 菜谱状态变化历史。
- 打卡消费统计。
- 数据量增大后前端列表不再全量加载。

建议接口：

```text
GET /api/kitchens/{kitchenID}/stats?window=30d
```

响应结构尽量与 V1 `spaceStats` 对齐。

后端模块建议：

```text
backend/internal/spacestats/
  model.go
  repository.go
  service.go
  handler.go
```

路由放在：

```go
protected.Get("/kitchens/{kitchenID}/stats", spaceStatsHandler.Overview)
```

服务层必须校验当前用户是该空间成员，口径与 `recipe`、`place`、`mealplan` 保持一致。

### 8.3 后续可能需要的数据增强

| 需求 | 当前限制 | 后续增强 |
| --- | --- | --- |
| 吃过趋势 | 菜谱没有 `doneAt` | 增加 `recipe_status_events` 或 `done_at` |
| 成员贡献 | 前端列表只有 ID，缺成员名映射 | 后端聚合时 JOIN `users` |
| 消费统计 | `price` 是文本，如 `¥79/人` | 增加结构化金额字段 |
| 状态历史 | 当前只存最终状态 | 增加事件表 |
| 长期趋势 | 前端只拿当前列表 | 后端按时间窗口聚合 |

## 9. 数据口径与风险

### 9.1 口径约束

- 统计默认只看当前空间，不跨空间汇总。
- V1 统计是“当前快照”，不是完整历史。
- `wishlist` 表示“想吃”，`done` 表示“吃过”。
- `want` 表示“想去”，`visited` 表示“去过”。
- 打卡点外部评分 `rating` 来自高德等第三方，不能与用户重访评分 `revisitRating`
  混合计算。
- `price` 当前为文本，只能展示，不做金额求和或平均。

### 空态策略

| 场景 | 展示策略 |
| --- | --- |
| 没有空间 | 不展示统计，提示创建或加入空间 |
| 空空间 | 展示 0 值统计和**温暖的引导文案**：”添加第一道想吃的菜，开启你的美食空间” + 直接弹出添加面板 |
| 同步中 | 展示骨架态或”正在同步”，统计卡顶部显示加载指示器 |
| 同步失败但有缓存 | 展示缓存统计，标记”本地缓存 · 2 分钟前” + 提供”重新同步”入口 |
| 同步失败且无缓存 | 展示错误提示和重试入口 |

### 9.3 性能风险

V1 前端聚合复杂度为 `O(n)`，当前阶段可接受。需要注意：

- `spaceStats` computed 内避免重复多次遍历大数组，可在纯函数中一次聚合。
- 推荐项和标签 Top 聚合需要限制结果数，例如最多 5 个。
- 日期解析要兼容空字符串和非标准格式，失败时跳过。

## 10. 分期计划

### V1：空间概览卡 + 半屏洞察

范围：

- 新增 `utils/space-stats.js`。
- 新增 `space-stats-card.vue`。
- 新增 `space-stats-sheet.vue`。
- 在空间页接入统计卡和详情半屏。
- **支持下拉刷新**：触发数据同步 + 统计重新聚合。
- **"本周可选" / "周末可去"替代"待探索"**，明确指向行动。
- **总览 Tab 直接露出"高分复访推荐 Top 3"**。
- 支持统计项跳回美食库 / 打卡点筛选。

不做：

- 不新增后端接口。
- 不做成员贡献。
- 不做长期趋势图。
- 不做消费金额统计。

### V1.1：打卡点体验洞察增强

范围：

- 重访评分分布（可用轻量饼图或横向条）。
- 推荐项 Top（可用标签云展示）。
- 场景标签 Top（同样考虑标签云）。
- 缺定位、缺推荐项、缺重访评分的补全提醒。
- **"菜单安排" Tab 新增"快速安排下一餐"按钮**。

依赖：

- 打卡点增强字段在前端新增 / 编辑 / 状态切换链路中稳定落地。

### V2：后端统计接口与趋势

范围：

- 新增 `/api/kitchens/{kitchenID}/stats`。
- 支持 `window=7d|30d|90d|all`。
- 支持新增趋势、打卡趋势、菜单安排趋势（**考虑迷你折线图或时间线卡片**）。
- 支持成员贡献和最近动态。
- **可选：地图热力图**，在"打卡库"中嵌入地图，标注"去过的店"。

可能需要：

- 新增状态事件表或补充 `doneAt` 字段。
- 为 `recipes(kitchen_id, created_at)`、`places(kitchen_id, created_at)` 等查询补索引。

## 11. 验收标准

### V1 产品验收

- 空间页能看到”空间概览”卡片。
- 切换空间后，统计数字随当前空间变化。
- **支持下拉刷新**，触发数据同步并更新统计，同步期间显示加载状态。
- 同步失败但有缓存时，统计仍可展示并有”本地缓存 · X 分钟前”提示 + “重新同步”入口。
- 点击”本周可选””周末可去”等统计项能跳到对应列表和筛选。
- **总览 Tab 能看到”高分复访推荐 Top 3”**，点击可跳转到店铺详情或筛选。
- **”菜单安排” Tab 有”快速安排下一餐”按钮**，能从想吃池推荐菜品。
- 空空间时有温暖引导：”添加第一道想吃的菜，开启你的美食空间” + 明确动作入口。

### V1 技术验收

- 不新增后端接口和数据库迁移。
- 统计聚合逻辑有独立纯函数，便于单测和迁移。
- `utils/space-stats.js` 覆盖空数据、部分字段缺失、日期异常、标签去重等边界。
- **下拉刷新能正确触发 `recipe-store.js` / `place-store.js` 同步方法**，并在同步完成后自动重新聚合。
- **数据结构包含 `isSyncing`、`updatedAt`、`topRevisitPlaces` 等新增字段**。
- 微信小程序 375px 宽度下数字和文案不溢出。
- 统计卡与现有空间页视觉风格一致，不影响邀请成员和成员列表操作。

## 12. 推荐优先级

第一优先级：

- 空间总览 4 数字（**拆分为"本周可选" / "周末可去"，替代"待探索"**）。
- **下拉刷新**：主动触发数据同步 + 统计重新聚合。
- **总览直接露出"高分复访推荐 Top 3"**，配图标 + 评分 + 推荐菜。
- 待行动：本周可选菜、周末可去店、草稿菜单。
- 打卡库：想去 / 去过、有定位、重访推荐。

第二优先级：

- 美食库图片覆盖、解析完整度。
- 菜单安排下一次安排、平均菜数。
- **"快速安排下一餐"按钮**：从想吃池随机推荐 3 道菜。
- 推荐项 Top、场景标签 Top（**考虑标签云可视化**）。
- 重访评分分布（**轻量饼图或横向条**）。

第三优先级：

- 成员贡献。
- 30 天趋势（**迷你折线图或时间线卡片**）。
- 消费相关统计。
- 独立统计页面。
- 地图热力图（V2）。

---

## 13. 用户体验优化要点总结

基于产品目标和用户需求，以下是关键的体验优化方向：

### 13.1 数据时效性与透明度

- **问题**：前端聚合基于已缓存数据，多成员协作时可能陈旧。
- **方案**：下拉刷新 + 明确的"更新时间"/"同步中"状态 + "本地缓存"弱提示。

### 13.2 从数据展示到行动建议

- **问题**："待探索"概念模糊，用户不知道该去哪个列表。
- **方案**：拆分为"本周可选 X 道"和"周末可去 X 家"，点击分别跳转。

### 13.3 高价值内容前置

- **问题**：`revisitRating`、`recommendedItems` 是最有价值的部分，但藏在详情深处。
- **方案**：总览 Tab 直接露出"高分复访推荐 Top 3"，配图标 + 评分 + 推荐菜。

### 13.4 空间活跃度可视化

- **问题**：静态数字堆砌，缺少"持续使用"的感知。
- **方案**：V1 用近期动态文字，V2 考虑迷你折线图或时间线卡片。

### 13.5 菜单安排联动

- **问题**：看到"已安排 5 天"后，下一步动作不明确。
- **方案**："快速安排下一餐"按钮，从想吃池随机推荐 3 道菜，一键加入草稿。

### 13.6 空态引导更温暖

- **问题**：空空间只展示"添加入口"，缺少温度。
- **方案**："添加第一道想吃的菜，开启你的美食空间" + 直接弹出添加面板。

### 13.7 可视化增强（V1.1 / V2）

- 重访评分分布：轻量饼图或横向条。
- 场景标签：标签云（气泡大小表示频次）。
- 地图热力图（V2）：在"打卡库"中嵌入地图，标注"去过的店"。
