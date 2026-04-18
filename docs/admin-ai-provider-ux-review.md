# AI Provider 管理页 UX 评估与优化建议

> 评估对象：`admin-web/src/pages/AIProvidersPage.vue`
> 依赖组件：`AppShell.vue` / `StatusTag.vue` / `FilterToolbar.vue` / `PageState.vue`
> Design tokens：`admin-web/src/style.css`（`--color-primary` / `--shadow-card` / `--radius-xl`…）
> 评估时间：2026-04-18

本文定位：为后台运维人员日常维护 AI 多 Provider 路由（summary / title / flowchart）的编辑器提供一份可落地的 UX 优化清单。目标是：**降低高频运维操作的认知负荷、提升失误可恢复性、增强决策信号可见性**，同时让界面更具高级感。

---

## 1. 页面角色与使用场景还原

- **主要用户**：后端/运维在配置中心切换模型密钥、调试兼容链路、排查上游异常。
- **高频动作**：切换场景卡 → 开关 Provider / 调整顺序 → 测试 → 保存 → 翻审计定位变更。
- **关键任务**（代码中对应）：
  1. 判断当前场景是否已切到新路由（`compatibilityMode` / `enabled`）。
  2. 对比草稿与远端（`isDirty`）并决定「测试当前草稿」还是「保存场景」。
  3. 针对单节点换密钥 / 定向测试（`handleTestSingleProvider`）。
  4. 复盘最近变更与测试结果（`路由审计`）。

理解这些后，可以把现状问题按「**信息架构 / 操作链路 / 反馈可视化 / 视觉层级 / 风险与无障碍 / 审计区**」六块梳理。

---

## 2. 现状问题梳理

### 2.1 信息架构：场景卡信息密度低且与下方面板冗余

- 三张场景卡（正文总结 / 标题清洗 / 步骤图生成）当前主要承载静态文本：策略 / 节点数 / 来源 / 最近修改，**没有任何运行时信号**（近 N 小时成功率、熔断状态、最近一次失败原因）。运维打开页面最想知道的「哪个场景现在是不是健康的」无法一眼得出。
- 场景卡与下方「场景策略」卡片存在**三处重复信息**：策略、最近修改、启用/兼容状态，占据了可观视觉空间却不承担新的信号。
- 选中态只依靠 1px 浅蓝描边，**在第一眼识别「我在哪个场景」上不够强**；场景卡本身是 `<button>`，语义对，但未提供 `role=tab` / `aria-selected`，屏幕阅读器无法感知 Tab 切换语义。

### 2.2 操作链路：保存 / 草稿 / 测试三者边界模糊

- 顶部「保存场景」按钮仅判断 `!draftScene`（模板 `:disabled="!draftScene"`），**不区分是否有改动**，未改动也能点。一次无意义的 save 会产生一条审计记录，污染复盘数据。
- 「测试当前草稿」的含义对于新同事不够明确：它测的是内存里的草稿还是服务端最新配置？当前只有「路由审计加载失败」等文案能侧面说明。
- **缺少「放弃草稿」**：当前要撤销草稿只能手动反改或刷新页面，刷新会丢失所有未保存修改；同时没有「离开页面前未保存确认」（`isDirty` 已算出却没有 `beforeRouteLeave` 保护）。
- 破坏性操作分布不均：删除节点有 `ElMessageBox.confirm`，而「清空旧密钥」只是一个 `text` 按钮，单击立刻触发 `clearApiKey=true`；「上移 / 下移」也没有轻量的撤销路径。

### 2.3 反馈可视化：测试结果远离触发点

- 测试结果卡 (`routing-test-card`) 被放在**两大编辑面板之下**，1440px 宽度下需要滚动才能看到，触发「测试当前草稿」后看不到即时反馈的用户体验断裂。
- `testResult` 同时承载「整条草稿测试」和「单节点测试」的结果，**没有二级 tab 或并列对比**，用户切换测试类型时结果被直接覆盖，易混淆。
- 路由审计的「旧值 / 新值」列只显示 `key=value` 扁平字符串且超出宽度被 `show-overflow-tooltip` 截断。当 `ai.routing.summary.scene` 这种行出现「`enabled=true strategy=round_robin_failover maxAttem...`」时，完全看不到差异，**无法作为复盘工具**使用。

### 2.4 视觉层级：操作堆叠 + 质感平庸

- Provider 节点卡右上角并排了「开关 / 上移 / 下移 / 测试 / 删除(红)」五个控件，**色彩对比高的红色「删除」紧挨主功能「测试」**，且间距仅 6–8px，手抖误触概率高。
- 卡片整体以 `rgba(255,255,255,0.98) → rgba(247,250,252,0.94)` 线性渐变 + 浅灰描边 + 淡阴影构成，偏「通用中后台模板」；对高级感的缺失主要在：
  - 缺少**全局运行状态条**（当前所有场景的生产/兼容比例、启用节点总数、最近 1h 路由成功率）。
  - 所有卡片视觉权重相同（字号 20–22px、Radius 18px、同款阴影），**没有主次节奏**。
  - 高频控件仍是文字按钮，缺少 Icon-only + Tooltip 的收敛方案。
- 顶部工具栏从左到右是「刷新 / 测试当前草稿 / 保存场景 / 账号」，**主 CTA「保存场景」和破坏性操作「刷新」同级**，没有视觉断点。

### 2.5 风险与无障碍

- **API Key 输入器的 placeholder 是 masked 值**（如 `sk-Y...CLHz`）。这与 Element Plus `show-password` 的行为叠加会让用户误以为脱敏值就是当前密码，而「留空 = 保留旧值」「点击清空才真正删除」的语义藏在一行灰色小字里，误操作成本极高（密钥丢失 / 上线事故）。
- 删除节点后 `maxAttempts` 仅在保存时做 `clamp`，UI 层短暂出现「启用节点 1 个 / 最大尝试次数 5」的矛盾态，对排查造成干扰。
- 「允许切换到下一个节点的错误类型」六项勾选（timeout / network / rate_limit / auth / upstream / invalid_response）**没有解释文案**，`upstream`、`invalid_response` 对偏运维的同事门槛高。
- 错误类型、调度策略、请求参数默认值都没有「恢复默认」按钮，改坏了只能看 `docs/ai-multi-provider-routing-design.md` 翻出合适值。
- 无深色模式；对长时间驻守后台的运维而言，暗色是「高级感 + 眼睛友好」的低成本加分项。

### 2.6 路由审计区

- 默认分组锁死在当前场景 `ai.routing.${scene}`，**无法并列对比多场景变更**；运维经常需要看「全局路由最近一次调整」。
- 筛选器只有「动作」下拉，**缺少操作人、时间范围、settingKey 精确搜索**；且 `FilterToolbar` 已经支持 `activeFilters` chips，这里没有用到。
- 表格无 `pageSize` 切换，31 条默认 20，体感上必须翻 2 次；且审计行无法点击展开完整 JSON diff。
- 审计缺少「回滚」或至少「复制为当前草稿」的快捷操作，定位到一次坏配置后，用户只能手抄。

---

## 3. 优化建议（按优先级）

### P0 · 一周内可落地，立竿见影

1. **保存按钮基于 `isDirty` 禁用**
   - `保存场景` 改为 `:disabled="!isDirty || savingScene"`；在 title 或 Tooltip 上提示 `当前没有未保存改动`。
   - 同步新增 `放弃草稿` 按钮，`draftScene = hydrateScene(remoteScene)`，并用 `ElMessageBox` 做二次确认。
2. **未保存草稿离开保护**
   - 在组件内 `onBeforeRouteLeave` + `window.beforeunload` 双层拦截；与 `handleSceneChange` 复用一套 prompt。
3. **Provider 节点卡操作区收敛**
   - 将「开关 / 上移 / 下移 / 测试 / 更多」改为 **图标按钮 + Tooltip**；删除挪入「更多」下拉菜单（`el-dropdown`），与「复制节点 / 重置为默认」并列。
   - 新增拖拽排序（`vuedraggable` 或 HTML5 drag），`moveProvider` 继续作为键盘可达的 fallback。
4. **API Key 输入器改造**
   - 输入框默认始终空，`masked` 值以只读 chip 形式展示在上方：`当前密钥 · sk-Y...CLHz`，旁边两个按钮：`更换密钥`（展开输入）/`清空密钥`（红色、二次确认）。
   - 避免任何 placeholder 载带 masked 文本。
5. **测试结果贴近触发点**
   - 在「测试当前草稿」按钮旁追加一枚 `StatusTag`（tone=success/warning），展示最近一次测试的结论 + 链路节点（`ok via summary-compat`）。
   - 点击 tag 可滚动/展开底部的详情卡。
6. **场景切换防误操作**
   - 切换场景时若 `isDirty`，弹出「保存草稿 / 放弃更改 / 取消」三选一；URL query 的 scene 切换同样走此闸门。

### P1 · 两周内，体感飞跃

7. **场景卡增强为「状态 + 指标」卡片**
   - 顶行保留 eyebrow + 标题 + StatusTag，下方 **加入近 1h 成功率微型 sparkline + 当前启用节点条形**（复用 DashboardPage 的指标源，`adminApi.listAIRoutingScenes` 可以扩展带回 `recentStats`）。
   - 选中态换为 **主色强调条 + 柔和内阴影 + 主色描边**（参考 `--color-primary-soft`），不再仅靠 border-color。
   - 为卡片加 `role=tab` / `aria-selected`，外层包一个 `role=tablist`；键盘 `←/→` 可切换。
8. **保存前「变更摘要」**
   - 点击保存时先弹出 diff dialog（类似 `terraform plan`）列出 `ai.routing.summary.scene` / `.provider.*` 的 `old → new` 关键字段；勾选「我已确认」再继续。
   - 可以复用 `FilterToolbar` 的 chip 风格呈现差异项。
9. **路由审计增强**
   - 表格行改为可点击展开，抽屉内展示完整 JSON diff（左旧右新，差异高亮），复用 `JsonViewerCard`。
   - 筛选器扩展到 `操作人 / settingKey 精确搜索 / 时间范围`，并接入 `FilterToolbar` 的 `activeFilters` chips 和「清除全部」。
   - 分页增加 `sizes` 选项（10/20/50），并把「总数」前置让信息更显眼。
   - 顶部加一枚「查看全部场景审计」切换，配合 `currentAuditGroup`。
10. **错误类型 / 策略的帮助态**
    - 每个 checkbox、`el-select` 选项加 `el-tooltip`：timeout→「读取上游超过场景设定」、invalid_response→「返回非 JSON 或缺字段」。
    - 「恢复默认」按钮放在场景策略卡右上角，调 `hydrateScene` 默认值。

### P2 · 设计沉淀 / 高级感加分

11. **全局运行状态条**
    - 顶部面包屑下增加一条 sticky bar：`3/3 场景 · 2 兼容模式 · 1h 成功率 98.4% · 平均耗时 712ms`，数据来自同一批汇总 API。
    - 点击指标可跳到「AI 任务 / API 调用」对应筛选条件。
12. **暗色模式 + Token 收敛**
    - 基于现有 `--color-*` 补齐 `prefers-color-scheme: dark` 分支；场景卡、provider 卡统一走 `--shadow-card` / `--shadow-pop`，少用行内 `rgba(15,23,42,0.08)`。
    - 主 CTA「保存场景」与次 CTA「测试当前草稿」之间加一条 1px 垂直分隔或 12px 额外间距，拉开节奏。
13. **键盘快捷键**
    - `⌘S` 保存（需 `isDirty`）、`⌘Enter` 测试草稿、`1/2/3` 切场景、`N` 新增节点；在页面右下角 Tooltip 展示提示。
14. **多节点时的折叠 / 分组**
    - 当 providers 数量 > 3 时，默认折叠成 summary 行（名称 + 状态 + 延迟 + Model），点击展开详情；保持大信息量时的浏览效率。
15. **沙盒预演测试**
    - 在「测试当前草稿」旁加「沙盒回放」按钮：挑选最近 N 条真实 trace（接 CallsPage 数据），用当前草稿回跑并出成功率 / P95，避免依赖单次调用撞运气。

---

## 4. 具体改造示例

### 4.1 顶部工具栏重排

```vue
<template #toolbar>
  <StatusTag
    v-if="latestTest"
    :tone="latestTest.ok ? 'success' : 'warning'"
    :text="latestTest.ok ? `测试通过 · ${latestTest.finalProvider}` : `最近测试失败`"
    @click.native="scrollToTestCard"
  />
  <el-button :loading="pageRefreshing" @click="refreshPage">刷新</el-button>
  <el-button :disabled="!isDirty" @click="handleDiscardDraft">放弃草稿</el-button>
  <el-button :loading="testingScene" :disabled="!draftScene" @click="handleTestScene">
    测试草稿 <span class="kbd">⌘↵</span>
  </el-button>
  <el-button
    type="primary"
    :loading="savingScene"
    :disabled="!isDirty"
    @click="handleSaveScene"
  >
    保存场景 <span class="kbd">⌘S</span>
  </el-button>
</template>
```

### 4.2 场景卡状态视觉

```
┌─ 正文总结 ───────────────── [✓ 正式模式] ─┐
│ 策略 轮询起始+失败切换  · 节点 1/1       │
│ ▁▂▃▂▃▇▆▅  成功率 98.2% / 1h            │   ← sparkline + 指标
│ 最近修改 04-18 03:59 · admin            │
└───────────────────────────────────────────┘
   ▲ 选中态：左侧 3px 主色强调条 + 主色 0.08 背景
```

### 4.3 API Key 分离展示

```
┌ API Key ──────────────────────────────────┐
│ 当前密钥 ⦁ sk-Y...CLHz  [更换]  [清空 ⚠] │
│ ───────────────────────────────────────── │
│ (点击「更换」后才展示密码输入框)          │
└───────────────────────────────────────────┘
```

### 4.4 审计行展开 diff

- 行前加 `>` 展开 icon。
- 展开区左右分列：旧值 / 新值；`settingKey` 为 JSON 时调用 `JsonViewerCard` 高亮差异。
- 底部按钮：`复制为当前草稿` / `复制 settingKey`。

---

## 5. 验收信号

改造落地后，围绕以下指标回归：

| 维度 | 指标 | 目标 |
| --- | --- | --- |
| 误操作 | 保存无改动频次 | 降至 0（按钮禁用）|
| 误操作 | 清空 API Key 误触工单 | 零 |
| 可恢复性 | 「放弃草稿」/「离开前确认」日触达 | ≥ 1 次/天（说明保护生效）|
| 信号可见 | 打开页面 3 秒内能说出「哪个场景不健康」 | 运维调研通过率 ≥ 90% |
| 高级感 | 设计评审打分（色阶 / 节奏 / 交互一致性） | ≥ 4/5 |
| 无障碍 | 键盘 Tab 链路可完成保存 / 测试 / 切换场景 | 100% |

---

## 6. 与现有文档的衔接

- 多 Provider 路由本身设计仍以 `docs/ai-multi-provider-routing-design.md` 为准，本文档仅聚焦前端运维体验。
- 顶部状态条 / sparkline 的数据来源，可与 `docs/admin-console-ai-observability-design.md` 对齐，避免再造一套指标口径。
- 后端如需配合（例如审计行的 old/new JSON 结构化、场景汇总接口补充 `recentStats`），后续任务可在 backend 侧开单独 issue 跟进。

---

## 7. 二次 review（2026-04-18 补充）

> 本轮基于实际界面截图复核，补充一版与首次评估并行的快速清单。与前文重合的（场景卡选中态、Provider 节点操作收敛、保存按钮基于 `isDirty`）不再重复，这里只列**首次评估未展开或可再强化**的点。

### 7.1 顶部操作区：主 CTA / 破坏性动作的节奏

- 当前「刷新 / 放弃草稿 / 测试当前草稿 / 保存场景」四个按钮同一体量、同一行，**保存场景作为最强语义动作没有视觉断点**。建议：
  - 刷新降级为 icon-only（与"账号"分区前置）；
  - 「放弃草稿」用 danger-text 次按钮，禁用态 Tooltip 说清「当前无草稿」/「将丢弃 N 项改动」；
  - 「测试当前草稿」和「保存场景」之间加 12px 分隔或 1px 竖线，拉开节奏（与 P2 #12 呼应）。
- 「保存场景」在有未提交改动时补一个小圆点 / `·` badge，比单纯 disabled 切换更能吸引注意。

### 7.2 "正式模式 / 草稿模式 / 新路由 / 兼容链路"概念收敛

截图里同一个场景卡下同时出现三条状态文案：「正式模式」「新路由已启用」「运行时仍走兼容链路」，**语义互相叠加但没有统一的解释入口**，新同事很难判断"我现在到底生效的是哪条链路"。建议：

- 在 StatusTag 右侧增加一枚 `?` icon，打开 popover 给出状态矩阵：

  | 草稿/正式 | 新路由开关 | 实际生效链路 |
  | --- | --- | --- |
  | 正式 · 开 | ✅ | 走新路由，多 Provider 降级 |
  | 正式 · 关 | — | 走兼容单节点 |
  | 草稿 · 开 | 🧪 | 仅测试入口生效，线上仍走旧值 |

- 同时把「运行时仍走兼容链路」改写为**事实性文案**（如 `线上链路：summary-compat`），不要与"新路由已启用"并列造成歧义。

### 7.3 数值输入缺单位 / 建议值 / 语义可视化

「最大尝试次数」「熔断阈值」「冷却时间（秒）」都是裸 stepper：

- 每个字段下加一行 helper text，写明推荐区间与越界风险（例："建议 2–5 次，过大将拖慢失败降级"）。
- 三者语义耦合（连续失败 N 次 → 冷却 T 秒 → 重试），可以在卡内嵌一张迷你时间轴示意图，帮助运维一眼理解触发关系，而不是分别盯三个独立数字。
- 数字变动时做 live-preview：把「预计首轮响应上限」换算成秒级文案（`最大 1 × 超时(s)`）贴在下方。

### 7.4 告警提示条权重过低

"连续异常邮件告警已接入配置中心"目前是浅灰底、无图标、无跳转的灰色面板，**权重像 placeholder**，但内容本身对运维很关键。建议：

- 改为 Element Plus `Alert (type=info, show-icon)`，**右侧直接加「前往配置」跳转按钮**，点击带上目标分组锚点；
- 如近期有 Provider 真正触发过告警，Alert 前置为 `type=warning` 并展示最近一次告警时间 / Provider，避免用户完全忽略。

### 7.5 Sticky 布局 + 节点密度优化（强化 P2 #14）

随着 Provider 节点增多，右侧列会越滚越长，左边「场景策略」却长期保持一屏高度。建议：

- 桌面宽度下（`lg+`）把「场景策略」面板做成 `position: sticky; top: 96px`，滚动节点列表时参数始终可见，便于对照改动；
- 超过 3 个节点时，默认折成**单行 summary**（名称 · 状态 · 延迟 · Model），展开才显示完整表单（与 P2 #14 的折叠策略结合）；
- 折叠行支持拖拽排序，不进编辑就能重排。

### 7.6 场景切换的"当前编辑中"面包屑

即便 P1 #7 把场景卡的选中态加强了，下方面板仍建议补一行"上下文面包屑"：

```
场景策略 · 正在编辑：AI 总结 / 正文总结   [📝 有未保存改动]
```

这一行既做"我在改哪个场景"的 wayfinding，也做 dirty 态的第二处提示，让用户不必回到顶部才能判断是否已脏。

### 7.7 验收信号补充

在 §5 的基础上，本轮新增两个观察项：

| 维度 | 指标 | 目标 |
| --- | --- | --- |
| 概念理解 | 新运维回答「当前场景实际生效哪条链路」的准确率 | ≥ 95% |
| 信息密度 | 1440px 下看完一条场景 + 1 节点编辑无需滚动 | 达成 |

---

## 8. 二次 review 的设计实现方案

> 面向 `admin-web/src/pages/AIProvidersPage.vue` 现状（Vue 3 `<script setup>` + Element Plus + ECharts），逐条落到代码级动作。除特别说明外，所有改动都只在本页组件内，无须后端配合。

### 8.1 顶部操作区节奏（对应 §7.1）

**目标态**：`[测试结果 chip] │ [🔄 icon] │ [放弃草稿] │ [测试草稿] ┃ [保存场景 ●]`

- 新增一层工具栏分组 DOM：
  ```html
  <template #toolbar>
    <div class="toolbar-cluster toolbar-cluster--status">…test chip…</div>
    <div class="toolbar-cluster toolbar-cluster--meta">刷新(icon) · 放弃草稿</div>
    <div class="toolbar-divider" aria-hidden="true" />
    <div class="toolbar-cluster toolbar-cluster--action">测试草稿 · 保存场景</div>
  </template>
  ```
  - `.toolbar-divider`：`width:1px; height:20px; background:var(--color-border-soft); margin:0 4px;`。
  - `toolbar-cluster--action` 与前组之间额外 12px `gap`。
- 「刷新」降级：`<el-button circle>` + `<el-icon><Refresh /></el-icon>`，配 `<el-tooltip content="重新拉取远端">`。
- 「放弃草稿」改 `type="danger" link`；禁用态用 `computed(discardTooltip)` 切换文案：`!isDirty ? '当前无未保存改动' : '将丢弃 N 项改动(含 X 个节点)'`；`N` 通过 `diffCount` 计数（见 §8.2）。
- 「保存场景」上加一枚 `dirty badge`：
  ```html
  <el-badge :is-dot="isDirty" class="save-dot" type="primary">
    <el-button type="primary" …>保存场景</el-button>
  </el-badge>
  ```
  CSS 里把红点染成主色：`.save-dot .el-badge__content { background:var(--color-primary); box-shadow:0 0 0 2px var(--color-bg-elevated); }`。

### 8.2 状态矩阵 popover（对应 §7.2）

**新增 computed**：`effectiveChannel(scene)`，返回 `{ label: 'summary-compat' | 'summary-v2', tone, reason }`。规则：

| draft/remote | enabled | compatibilityMode | label |
| --- | --- | --- | --- |
| remote | true | false | `{scene}-v2` |
| remote | * | true | `{scene}-compat` |
| remote | false | false | `{scene}-compat` |
| draft only | — | — | `线上保持不变，仅测试入口生效` |

- 场景卡 `StatusTag` 右侧追加 `<el-popover trigger="hover|click" :width="320">`，内容用一张 3×3 矩阵表（复用 `el-table` size=small 或手写 `<dl>`），当前状态行用 `--color-primary-soft` 高亮。
- 文案改写：把 `运行时仍走兼容链路` / `运行时已走新路由` 两行统一为 `线上链路：<code>{effectiveChannel.label}</code>`，放在原位置；兼容模式 `el-alert` 的 description 也同步引用同一 label，避免两处表述漂移。
- 增加 `diffCount`（`ref`）在 `hydrateScene` / watch(draftScene) 里重算：遍历 `sceneFieldPaths` 对比 `draftScene` vs `remoteScene`，产出 `{ scope: 'scene' | 'provider:{id}', path, from, to }[]`，顶部 discard tooltip / §3 P1#8 的 diff dialog 都复用这份结构。

### 8.3 数值输入语义化（对应 §7.3）

**受影响控件**：模板 L124/128/132 的「最大尝试次数 / 熔断阈值 / 冷却时间」。

- 包一层 `<FieldWithHint>` 局部组件（仅本文件内，`const FieldWithHint = defineComponent({...})` 或纯 template partial）：
  - props: `label`, `unit`, `recommendHint`, `warningWhen(value) => string | null`。
  - slot 放 `el-input-number`；下方渲染 `.field-hint` 小字 + `warningWhen` 命中时切换为 `--color-warning`。
- 推荐区间常量：
  ```js
  const NUMERIC_HINTS = {
    maxAttempts: { range: [2, 5], tip: '建议 2–5 次；过大会把失败降级变慢' },
    circuitThreshold: { range: [3, 10], tip: '连续失败次数达到后触发熔断' },
    cooldownSeconds: { range: [30, 300], tip: '熔断冷却；低于 30s 容易抖动' },
  }
  ```
- 新增一张 `<RoutingTimelineHint>`（纯 SVG，宽 100%，高 48px）嵌在三个数字之下：
  ```
  attempt₁ ──fail──▶ attempt₂ … attemptₙ ──fail×T──▶ [cooldown S]
  ```
  用 `draftScene.strategy.maxAttempts / circuitThreshold / cooldownSeconds` 算段宽；变动时 `watchEffect` 重绘。
- Live preview：`const expectedFirstRoundMs = computed(() => maxAttempts * timeoutMs)`，渲染 `预计首轮最长 {n}s`。

### 8.4 告警条升级（对应 §7.4）

- 把 L70 的 `el-alert` 改造：
  ```html
  <el-alert
    :type="alertTone"         // 有近 1h 触发则 'warning'，否则 'info'
    show-icon
    :closable="false"
    :title="alertTitle"       // '连续异常邮件告警已接入配置中心' / '最近 1h 触发过 N 次告警'
  >
    <template #default>… 描述 …</template>
    <template #append>
      <el-button type="primary" link @click="goAlertConfig">前往配置</el-button>
    </template>
  </el-alert>
  ```
  `el-alert` 无 `#append` 槽 → 用绝对定位 `.routing-alert__action { position:absolute; right:16px; top:50%; transform:translateY(-50%); }`。
- `goAlertConfig`：`router.push({ path: '/settings', query: { group: 'ai.alert' }, hash: '#ai-provider-alert' })`，配置页对应分组加一个 `scrollIntoView` 监听 hash。
- 数据源：新增 `adminApi.fetchAIRoutingAlertSummary({ windowHours: 1 })`（后端若暂无，可先读 `ai_routing_audit` 里 `action=alert.fire` 计数做 mock），`onMounted` 拉一次、`onActivated` 再拉。

### 8.5 Sticky 策略面板 + 节点折叠（对应 §7.5）

- `routing-editor-grid` 目前是双列（L1224）。桌面宽度下给左列加：
  ```css
  @media (min-width: 1200px) {
    .routing-editor-grid > .routing-panel--strategy {
      position: sticky;
      top: calc(var(--app-shell-toolbar-h, 64px) + 24px);
      align-self: start;
      max-height: calc(100vh - 120px);
      overflow: auto;
    }
  }
  ```
  需要给策略面板加 `--strategy` 修饰类。
- Provider 节点折叠：新增 `collapsedProviderIds = ref(new Set())`；`providerRows` length > 3 时默认全部加入。每张 provider 卡顶部新增 `summary-row`：
  ```
  [drag] [▶] [名称] [StatusTag] [延迟 312ms] [model: gpt-4o] [开关] [更多]
  ```
  - `drag` handle 用 `vuedraggable`，`handle: '.provider-card__drag'`；`onEnd` 里改 `draftScene.providers` 顺序，触发 `isDirty`。
  - 折叠行支持拖拽排序，不展开编辑表单；展开后才渲染现有 `el-form`（用 `v-if="!collapsed"` 避免渲染成本）。
- 键盘可达性：`▶` 按钮 `aria-expanded` 同步 collapsed 状态。

### 8.6 编辑上下文面包屑（对应 §7.6）

- `routing-editor-grid` 之上、告警之下插入：
  ```html
  <div class="routing-breadcrumb" aria-live="polite">
    <span class="routing-breadcrumb__crumbs">
      场景策略 <el-icon><ArrowRight /></el-icon>
      正在编辑：{{ currentSceneTitle }}
    </span>
    <StatusTag v-if="isDirty" tone="warning" text="📝 有未保存改动" />
    <span v-else class="routing-breadcrumb__clean">已同步</span>
  </div>
  ```
  CSS：`position: sticky; top: var(--app-shell-toolbar-h); z-index: 9;` + 毛玻璃 `backdrop-filter: blur(10px)` + `background: color-mix(in srgb, var(--color-bg-elevated) 85%, transparent);`。
- 与 §8.5 的 sticky 策略面板叠放时，面包屑 `top` 在前，策略面板 `top` 往下再加 `+40px`；用 CSS 变量 `--routing-breadcrumb-h` 做联动。

### 8.7 新增/调整的状态与工具函数汇总

| 名称 | 类型 | 位置 | 用途 |
| --- | --- | --- | --- |
| `diffCount` / `sceneDiff` | `computed` | script setup 顶层 | 驱动 discard tooltip / save dialog / badge |
| `effectiveChannel(scene)` | `function` | 同上 | §8.2 状态矩阵与文案收敛 |
| `NUMERIC_HINTS` | `const` | 同上 | §8.3 推荐值 / 告警阈值 |
| `collapsedProviderIds` | `ref<Set<string>>` | 同上 | §8.5 折叠状态 |
| `alertSummary` | `ref` + `fetchAIRoutingAlertSummary` | 同上 | §8.4 告警条 |
| `FieldWithHint` | 局部组件 | 同文件 | §8.3 stepper + hint |
| `RoutingTimelineHint` | 局部组件（SVG） | 同文件 | §8.3 时序图 |

### 8.8 落地顺序与风险

1. **第 1 批**（无后端依赖，~0.5d）：§8.1 工具栏分组、§8.6 面包屑、§8.3 hint 文案、§8.4 前端文案升级。风险最低，先合。
2. **第 2 批**（~1d）：§8.2 diff 基础设施 + 状态矩阵 popover，复用到 P1 #8 的 diff dialog。
3. **第 3 批**（~1–1.5d）：§8.5 拖拽 + 折叠，引入 `vuedraggable` 依赖，需在 `admin-web/package.json` 登记。
4. **第 4 批**（依赖后端）：§8.4 告警 summary 接口、§8.3 成功率 sparkline 如果要真数据，需扩展 `listAIRoutingScenes` 返回 `recentStats`。建议单独开 backend issue，与前端 mock 并行。

主要风险：
- sticky + 折叠面板在 Safari 下对 `overflow:auto` 父级敏感，合并时务必真机回归；
- 拖拽改顺序会把 `draftScene` 变脏，需确认和 `handleDiscardDraft` 的 revert 路径一致；
- `el-alert` append 区用绝对定位存在 i18n 长文本重叠风险，长文案走换行 fallback。

---

## 9. 落地 TODO（分阶段）

> 已评估现状与方案的差异：`放弃草稿` / `isDirty` 禁用已在模板 L11/L20 实现；Provider 节点已用原生 HTML5 drag（L210 等），因此 §8.5 不再引入 `vuedraggable`，复用现有 drag handle，仅加折叠态。

### 阶段 1 · 低风险先合（~0.5d，纯模板 + CSS）

- [x] **§8.1 工具栏节奏**：`#toolbar` 切成三组 cluster + 1px divider；刷新改 circle + icon-only + Tooltip；放弃草稿 `type="danger" link` + 动态 tooltip `将丢弃 {diffCount} 项改动`；保存场景外包 `<el-badge :is-dot="isDirty">`。
- [x] **§8.6 编辑面包屑**：`routing-editor-grid` 上方插入 `.routing-breadcrumb`，sticky + 毛玻璃；暴露 CSS 变量 `--routing-breadcrumb-h` 给阶段 4 复用。
- [x] **§8.3 数值字段 hint**：封装 `FieldWithHint`（label / unit / recommendHint / warningWhen），常量 `NUMERIC_HINTS`；三字段下方嵌 `RoutingTimelineHint` SVG + `expectedFirstRoundMs` 文案。

### 阶段 2 · 概念收敛（~1d，为 P1#8 diff dialog 铺路）

- [x] **sceneDiff 基础设施**：`computed sceneDiff`: `{ scope, path, from, to }[]`，`diffCount = sceneDiff.length`；接入阶段 1 的 discardTooltip。
- [x] **§8.2 状态矩阵**：`effectiveChannel(scene)` + 场景卡 `StatusTag` 右侧 `el-popover` 3×3 矩阵；模板 L56 / 兼容 alert 文案统一为「线上链路：`{scene}-v2|compat`」。

### 阶段 3 · 告警升级（~0.5d，可 mock 不阻塞）

- [x] **§8.4 告警条**：L70 `el-alert` 改造 + 绝对定位「前往配置」按钮；`fetchAIRoutingAlertSummary({ windowHours: 1 })` 先读 `ai_routing_audit` 做 mock；后端真 summary 接口单独开 issue。

### 阶段 4 · 布局重构（~1d，需 Safari 真机回归）

- [x] **§8.5 sticky + 折叠**：左列加 `.routing-panel--strategy`，`@media(min-width:1200px)` 下 `position:sticky; top:calc(toolbar-h + breadcrumb-h + 24px)`；`collapsedProviderIds = ref(Set)`，>3 节点默认折叠；折叠行 `summary-row` 复用现有 HTML5 drag handle，展开用 `v-if` 控制现有表单。

### 跟进（非本批次）

- [ ] P1 #8「保存前变更摘要 dialog」→ 复用阶段 2 的 `sceneDiff`。
- [ ] 后端 `listAIRoutingScenes` 扩展 `recentStats` / `fetchAIRoutingAlertSummary` 真实接口（单独 issue）。
