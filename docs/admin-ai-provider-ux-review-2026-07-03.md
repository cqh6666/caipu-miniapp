# AI Provider 管理页 UX 优化文档（第三轮 · 做减法）

> 时间：2026-07-03
> 范围：`admin-web/src/pages/AIProvidersPage.vue`
> 前序：`docs/admin-ai-provider-ux-review.md`（2026-04 两轮评估）、`docs/admin-ai-provider-ui-design-report-2026-04-24.md`
> 定位：前两轮以「补能力、补概念解释」为主，建议已基本落地进当前代码；本轮问题已从「功能缺失」转向「信息密度过载、行动焦点缺失」，主题是 **做减法、立主线**。

---

## 1. 本轮结论（TL;DR）

当前页面工程完成度很高，不缺功能：

- 保存有 diff 确认弹窗（`buildSaveConfirmVNode` + `save-confirm-message-box`），删除节点 / 清空密钥 / 放弃草稿都有二次确认；
- 熔断有时序可视化（`routing-timeline-hint`），审计有 diff 时间线，页面有锚点目录与键盘快捷键；
- 场景卡带三项健康快照（最近测试 / 配置风险 / 告警状态）。

问题不在能力，而在 **密度和焦点**：

1. 心智模型偏重——「草稿/正式 × 新路由开关 × 兼容模式 × 线上链路」四个正交概念叠加，需要一张矩阵表让用户自己推理"线上到底生效什么"。
2. 状态信息重复——同一类状态（线上链路 / 告警 / 场景状态）在 5+ 处重复呈现，视觉无主次。
3. 核心动作（保存）被拆散且弱化——保存/测试是底部一枚小圆形图标，放弃在顶部，单节点测试在卡内。

**一句话：把「线上到底生效什么 + 我有没有未保存改动 + 保存动作」做成全页最突出、结论明确的主线，其余状态做减法。**

> **阅读指引（含给 AI 的导航）：** §3–§4 是结构性问题与方向；§7 是截图实证；§8 是 `ui-ux-pro-max` 规范背书。**若只想拿可执行清单，直接看 §9 统一问题索引（每条带稳定 ID / 严重度 / 影响面 / 是否待验证）+ §10 术语表 + §11 落地路线图。** 前文各节均以 `I##` 回指索引。
>
> ⚠️ **重要前提：** §7 部分问题（告警作用域、maxAttempts 语义、测试 ok 判定）由截图**推断**得出，落地前须按 §9「待验证」列先在代码/数据核实，不要当既定 bug 直接改。

---

## 2. 现状还原

单页承载 3 个 AI 场景（正文总结 / 标题清洗 / 步骤图生成）的路由编辑。从上到下：

| 区块 | 关键元素 | 作用 |
| --- | --- | --- |
| 场景卡区 `#ai-provider-scene-cards` | 3 张 tab 卡：策略 / 节点数 / 来源 / 三项健康 / 最近修改 / 线上链路，支持快捷键切换 | 选场景 + 概览 |
| 兼容模式告警 `routing-alert` | 兼容模式提示 | 风险提示 |
| 面包屑 `routing-breadcrumb` | "场景策略 › 正在编辑：X" + 线上链路 popover（`channelMatrix` 三列矩阵） | 定位 + 生效解释 |
| 状态条 `routing-status-strip` | 再次重复：场景标题 / 链路 / reason + 告警状态 + 告警配置 popover | 状态 |
| 风险条 `routing-risk-strip` | 阻断性风险（`blockingRiskItems`） | 风险 |
| 双栏编辑 `routing-editor-grid` | 左：场景策略（开关/策略/尝试/熔断/冷却/retryOn/请求参数 + 时序图）；右：Provider 节点（折叠卡 + 拖拽 + 启停 + 单测 + 更多菜单） | 主编辑 |
| 最近测试结果 `routing-test-card` | attempts 表格 | 反馈 |
| 最近审计 `audit-section` | diff 时间线 + 完整审计抽屉 | 追溯 |
| 底部悬浮栏 `routing-bottom-bar` | 状态 + 草稿摘要 popover + 刷新/测试/保存（圆形图标） | 行动 |

---

## 3. 问题梳理（本轮新增，按优先级）

### P0-1 · 心智模型过重：需要用矩阵表回答"线上生效什么"

最强信号：面包屑里用一张三列表格 `channelMatrix`（草稿/正式 × 新路由 × 实际生效链路）去解释生效逻辑。对配置类后台，用户的核心焦虑是 **"我改的东西到底生不生效"**，而这里要靠 popover 里的矩阵去推理，信任成本极高。

- 概念数量：`draftScene.enabled`（新路由开关）× `compatibilityMode`（兼容模式）× 草稿 vs 正式（`isDirty`）× `currentChannel`（线上链路）——四维正交。
- 症状：面包屑、状态条、场景卡 footer 都在讲"线上链路"，说明这个概念本身难懂到需要反复解释。

### P0-2 · 状态信息严重重复，视觉无焦点

"线上链路 / 告警状态 / 场景状态"在 **顶部工具栏 chip、场景卡、面包屑、`routing-status-strip`、`routing-risk-strip`、底部栏** 反复出现。整页是 `page-card` + 多条 strip 堆叠，缺一个明确视觉焦点；真正的行动焦点（保存）反而最不显眼。

### P0-3 · 保存是全页最重要的动作，却是底部一枚小圆图标

- 保存 / 测试在 `routing-bottom-bar`（circle icon）；放弃草稿在顶部 `toolbar`；单节点测试在每张 provider 卡内。核心动作被拆散在三处。
- 用图标承载"保存/发布线上 AI 路由"这种高风险操作，可发现性与确认感偏弱。
- "共 N 项改动"藏在"草稿摘要" popover 里，需主动点开才可见（`sceneDiff`）。

### P1-1 · 测试对象（草稿 vs 线上）不够显性

测试用的是草稿（tooltip `测试当前草稿`），但这句话埋在 hover 里。用户容易困惑"我测的是刚改的还是线上的"。测试结果卡（`routing-test-card`）也未标注对象。

### P1-2 · 首次 / 空状态缺引导

新增节点（`handleAddProvider`）是空白表单，首次进入直接面对满屏概念，没有"推荐配置 / 一键最小可用"。对低频后台，几个月后回来需重新理解。

### P2-1 · 响应式存疑

双栏 `routing-editor-grid` + 多张表格 + 写死宽度的 popover（680 / 340 / 360）+ 快捷键提示仅 `viewportWidth >= 1024` 显示。若存在平板 / 小窗使用场景，需实测挤压情况。

---

## 4. 优化建议

### P0-1 → 用「一句人话的结论」替代矩阵推理

- 在场景卡与主编辑区顶部，常驻一行 **陈述句结论**，而非让用户算：
  - 无草稿：`线上正在使用：主节点 gpt-4.1-mini（priority failover）`
  - 有草稿：`线上仍在使用：主节点 gpt-4.1-mini · 你的草稿保存后才会切换`
  - 兼容模式：`线上仍走旧单 Provider 配置 · 启用并保存本场景后才切换到新路由`
- 把 `channelMatrix` 矩阵降级为"为什么？"的次级解释（结论旁一个 `?` 展开），不作为理解生效的主路径。
- 原则：**状态陈述结论，不让用户做组合推理。**

### P0-2 → 状态去重，合并为一处权威区

- 建立单一"当前场景状态"权威区（建议合并 `routing-status-strip` 承担），只讲三件事：**线上生效什么 / 是否有未保存草稿 / 告警状态**。
- `routing-breadcrumb` 的链路 popover、场景卡 footer 的链路重复项弱化或移除。
- `routing-status-strip` / `routing-risk-strip` / 面包屑三条 strip 收敛为一到两条，风险并入状态区高亮而非独立条带。

### P0-3 → 底部保存栏升级为主线 CTA

- `isDirty` 为真时：把保存做成 **带文字的主按钮**「保存并发布」，测试做成次级文字按钮，圆形图标仅留刷新。
- 把 `sceneDiff` 的"共 N 项改动"**平铺**到保存栏（不藏 popover），点击展开明细。目标：让"有未保存改动"变成想忽略都难的状态。
- 顶部"放弃草稿"移到保存栏旁，与保存/测试同处，形成完整动作组。

### P1-1 → 显性标注测试对象

- 测试按钮与 `routing-test-card` 标注 **"测试对象：当前草稿（未保存）"**，结果区补一句"该结果不代表线上当前行为"。

### P1-2 → 空/首次引导

- `handleAddProvider` 提供 **常见 provider 预设**（OpenAI 兼容 / 图片生成等）一键填充。
- 空场景给"快速开始"路径，降低首次认知成本。

### P2-1 → 响应式实测与降级

- 对 `routing-editor-grid` 在窄屏改单列；popover 宽度改为 `min(90vw, Npx)`；确认平板/小窗下场景卡与表格不溢出。

---

## 5. 验收信号

- [ ] 用户无需打开任何 popover，即可一眼说出"线上此刻在用哪个节点、我的改动是否已生效"。
- [ ] 全页"线上链路/告警/场景状态"重复呈现点从 5+ 降到 ≤2。
- [ ] `isDirty` 时，"保存并发布"是视觉上最突出的元素，且改动条数默认可见。
- [ ] 测试结果能明确回答"这是草稿还是线上"。
- [ ] 首次进入或空节点时有明确的下一步引导。
- [ ] 平板 / 小窗（≤768px）无横向溢出、无 popover 越界。

补充（覆盖 §7 / §8 新增问题）：

- [ ] 同一节点全页只用一个名字（`provider.name`），slug 不作为主展示（I8/I9）。
- [ ] 界面无未本地化的英文技术串（route test succeeded / cooling down 等）（I10）。
- [ ] 任何"成功/失败/告警"状态都同时具备 颜色 + 图标 + 文本，且标注作用域（I1/I11 · Color Only）。
- [ ] 审计区可区分并筛选「配置变更 / 测试执行」，变更不被测试记录淹没（I12）。
- [ ] 图标按钮均有可见文字或稳定 tooltip，且保留 `aria-label`（I15 · Form Control Labels）。
- [ ] 无效配置（如 failover 策略下 maxAttempts=1）在保存前被拦截或纠正（I6）。

---

## 6. 与现有文档的衔接

- 本轮 **不推翻** 前两轮成果；矩阵表、面包屑、告警条、sticky、节点折叠是上一轮"补解释、补能力"的正确产物，只是在能力齐备后需要 **收敛为结论**。
- 落地时优先 P0（纯模板 + CSS + 文案，低风险），P1/P2 视排期推进。
- 若涉及"线上生效结论"的取数逻辑（草稿/正式/兼容三态推导），复用现有 `currentChannel` / `channelMatrix` 计算结果即可，无需改后端契约。

---

## 7. 原型实证（2026-07-03 截图评审）

对当前线上原型截图（正在编辑「正文总结」场景）做逐屏走查，第 3 节的抽象问题在界面上表现为若干**肉眼可见的矛盾**，按"最伤信任 → 可直接删"排列。

### 7.1 状态自相矛盾（最高优先级：信任问题）

**① 告警数字作用域不一致**

- 三张场景卡分别显示「告警中 2 / 2 / 4」个；
- 当前编辑「正文总结」，其下方状态条却显示「**告警中 8 项**」（= 2 + 2 + 4）。

用户在编辑单个场景，状态条却混入**全场景告警合计**。这是信息作用域 bug：状态条应只反映当前场景（2 个），或明确标注"全部场景合计"。现状会让人误判"本场景炸了 8 个"。

> 落地：核对状态条 `alertStatusSummary` / `blockingRiskItems` 的取数范围，确保与 `currentSceneKey` 对齐；跨场景合计需显式文案。

**②「正式模式」与「线上链路」语义冲突**

- 三张卡顶部均为绿色「正式模式」；
- 线上链路却是 `title-compat`、`flowchart-compat`（compat = 兼容模式）。

标签说"正式"、链路名说"compat"，二者直接打脸，用户无法判断线上真实状态。这是「概念过重（§3 P0-1）」的实证——界面自身都未对齐。

**③ 单卡内「绿 + 红」同框**

每张场景卡同时出现：配置风险「✓ 正常」(绿)、告警状态「⊗ 告警中」(红)、顶部「正式模式」(绿)。同一张卡既喊"正常"又喊"告警"，观感割裂，用户不知是否该处置。

> 落地：卡片给一个**聚合结论态**（正常 / 需关注 / 告警），三项明细降为其下的解释，而非三个独立、可互相矛盾的标签。

### 7.2 第一屏没有主 CTA，破坏性动作最显眼

首屏最突出的操作是右上角**红色「放弃草稿」**（破坏性），而"保存 / 发布"在需滚到底的 sticky 栏，首屏不可见。违反"破坏性操作不应比主操作更显眼"。

> 落地（对应 §4 P0-3）：将「保存并发布」提到顶部工具栏做蓝色实心主按钮，「放弃草稿」降为次要文字按钮。

### 7.3 冗余，可直接删

- **顶部两个完全相同的绿 chip**「单节点测试通过 · 薄荷中转公益 · 2.4 s」：一个在"放弃草稿"左侧，一个在面包屑右侧——重复，留一。
- **「正式模式」出现 4 次**（三张卡 + 场景策略面板）。
- **「线上链路」重复三处**（场景卡 footer / 面包屑 / 状态条）。

### 7.4 无效配置未被拦截

场景策略「最大尝试次数 = 1」，下方红字「至少 2 次才能触发节点切换」，而当前策略为「轮询起始 + 失败切换」——尝试 1 次永不切换，配置与策略自相矛盾却可存在。

> 落地：failover 类策略下 `maxAttempts` 最小值锁 2，或保存时阻断（`validateBeforeAction`）。

### 7.5 布局效率：左右栏严重不平衡

左「场景策略」在冷却时间后结束、下方大片空白；右「Provider 节点」（3/7 启用、7 节点）无限下滚。双栏导致左侧留白 + 右侧超长。

> 落地（对应 §4 P2-1 延伸）：左栏 `position: sticky` 跟随，或压缩为更紧凑单列，把横向空间让给 Provider 列表。

### 7.6 术语暴露过度

`summary-v2` / `title-compat` / `flowchart-compat` 等内部标识符直接展示给用户，且命名不统一（v2 vs compat）。

> 落地（对应 §4 P0-1）：展示层用人话（"新多节点路由 / 旧单节点兼容"），技术标识符移入 tooltip 或次要位置。

### 7.7 本轮"只改三件事"建议

1. **修状态一致性**：告警作用域（8 项 vs 2 个）+「正式模式 vs compat」矛盾标签——信任问题，最优先。
2. **立主 CTA**：「保存并发布」提为首屏可见主按钮，「放弃草稿」降级。
3. **删冗余**：移除重复的测试 chip 与多余的「正式模式」标签。

### 7.8 第二屏实证（测试结果 / 审计流 / 底部栏）

对页面下半屏（`routing-test-card` + `audit-section` + `routing-bottom-bar`）走查，新增问题如下。

**① 命名双轨（同一节点两个名字）**

同一节点在测试结果区出现两种叫法：Provider 列显示「薄荷中转公益」，"最终节点"与审计却是内部 slug `summary-bohe`（"ok via summary-bohe"）。用户无法对应。

> 落地：展示层统一到 `provider.name`（人类可读名），slug 仅入 tooltip / mono 次要位。

**② 中英混杂的原始日志直出**

中文界面直接透出后端 raw message：`route test succeeded`、`ok via summary-bohe`、`all providers are cooling down`。

> 落地：加一层 message → 文案映射（`所有节点均在冷却中` / `测试通过，经由 薄荷中转公益`）。

**③ 绿标「测试通过」配「所有节点在冷却」——状态矛盾（同 §7.1 病根）**

某条审计打绿色「测试通过」，正文却是 "all providers are cooling down"（全部节点熔断冷却，明显异常态）。核对测试 `ok` 判定逻辑。

**④ 审计流被测试噪音淹没**

最近 5 条里 3 条是「测试通过 · 场景策略」，内容重复（"ok via summary-bohe"），把真正重要的「修改 Model」「保存」稀释。每点一次测试就生成一条同权重审计。

> 落地：视觉/筛选上分离「配置变更」与「测试执行」两类，变更为主线，测试可折叠。

**⑤ diff 展示规则不一致**

「修改 Model」内联展示 `gemini-flash-latest → grok-4.20-fast`；「保存 场景策略」却只提示"点击查看变化"。同为变更，一个内联一个点开。

> 落地：关键字段（1–2 个）统一内联，复杂改动才折叠。

**⑥ 事件标签三套视觉语言**

「修改 Model」蓝色带 icon、「保存」灰色圆点、「测试通过」绿色带勾——同为审计事件却三种样式。统一为一套"动作类型"标签体系。

**⑦ 底部栏：第三次重复 + 保存靠猜**

"单节点测试通过 · 薄荷中转公益 · 2.4s" 第三次出现（工具栏 / 面包屑 / 底部栏）；三个无文字圆形图标（刷新 / 纸飞机 / 蓝勾），测试与保存全靠猜；有未保存草稿时底部未显示"N 项未保存改动"。

---

## 8. 结合 ui-ux-pro-max 的规范化优化方案（供 AI 协作参考）

> 本节将前述问题对齐到 `ui-ux-pro-max` skill（`.claude/skills/ui-ux-pro-max`）的设计系统与 UX 规范条目，给出**带规范背书、可被其他 AI 直接执行**的规则。检索命令见每节标注。

### 8.1 设计基线与本地化取舍（重要）

`ui-ux-pro-max --design-system` 对本页（AI 运维配置台）给出的推荐：

- **Pattern：** Data-Dense + Drill-Down（数据密集 + 逐层下钻）
- **Style：** Data-Dense Dashboard —— 多组件、数据表、KPI 卡、最小内边距、网格布局、空间高效、最大化数据可见性
- **推荐配色：** 深色底（`#020617`）+ 绿色正向指示（`#22C55E`）
- **推荐字体：** Fira Code（标题/等宽）+ Fira Sans（正文）
- **关键效果：** hover tooltip、点击放大、行 hover 高亮、筛选平滑动画、加载态
- **反模式：** 过度装饰（ornate）、无筛选（no filtering）

**⚠️ 本地化取舍（请后续 AI 务必遵守）：**

当前 `admin-web` 是**浅色 Element Plus 主题**。**不要**照搬 skill 推荐的深色底配色去重构主题——那会与现有全站风格冲突。本页只采纳 Data-Dense Dashboard 的**信息组织原则**（密度、网格、行 hover、drill-down、必备筛选），配色沿用现有浅色 token。可借鉴之处：

- **等宽字体**用于 `model` / slug / `requestId` / 时间戳等技术标识（提升可扫描性），对应 Fira Code 的定位；
- **"必须有筛选"** 反模式 → 直接支撑 §7.8④ 审计流需要"变更/测试"筛选；
- **"最大化数据可见性 + 最小内边距"** → 支撑 §7.5 双栏留白与 §4 P2-1 布局压缩。

### 8.2 问题 → 规范映射（每条可溯源）

| 本页问题 | ui-ux-pro-max 规范（域·条目·严重度） | Do / Don't | 落地动作 |
| --- | --- | --- | --- |
| 绿+红标同框、告警作用域不一致、正式模式 vs compat（§7.1 / §7.8③） | ux · Accessibility「Color Only」· High | Do: 颜色 **+ 图标 + 文本**；Don't: 仅红/绿传递成功失败 | 状态标签=颜色+icon+文本+**明确作用域**；聚合结论态替代互相打架的多标签 |
| 保存被弱化、放弃草稿最显眼、图标靠猜（§7.2 / §7.8⑦） | ux · Interaction「Confirmation Dialogs」· High；Feedback「Confirmation Messages」· Medium | Do: 破坏性动作二次确认 + 成功 toast | 每屏唯一实心主 CTA「保存并发布」常驻可见；破坏性降为次级/link |
| 中英混杂 raw message（§7.8②） | *skill 库无直接 i18n 条目*；就近可参考 ux · Interaction「Error Feedback」· High（反馈须清晰可理解） | Do: 清晰可理解的本地化反馈 | 后端 message 过映射表，禁止直出英文技术串（归入一致性原则，非 skill 明文条目） |
| 审计流被测试噪音淹没（§7.8④） | design-system 反模式「No filtering」；ux 信息分组 | Do: 提供筛选/分组 | 审计区分「配置变更 / 测试执行」两类 + 筛选 |
| 长列表（7+ 节点、审计流）（§7.5） | web · Performance「Virtualize Lists」· High | Do: **超过 50 项**虚拟化 | Provider/审计列表超阈值时虚拟滚动；先做分页/折叠 |
| 底部无文字图标按钮（§7.8⑦） | web · Accessibility「Semantic HTML」· High；「Form Control Labels」· Critical | Do: 用 `<button>`/`<label>`，控件有可访问名 | 图标按钮补可见文字或稳定 tooltip + 保留 `aria-label` |
| 新增节点空白表单（§3 P1-2） | ux · Feedback「Empty States」· Medium | Do: 空态给提示 + 行动 | 新增节点给 provider 预设；空场景给"快速开始" |
| 测试/保存无过程反馈 | ux · Feedback「Loading Indicators」· High；Forms「Submit Feedback」· High | Do: >300ms 显示 spinner/skeleton，提交后给成功/失败 | 保存/测试按钮 loading 态 + 结果 toast（现已部分具备，补齐一致性） |
| 状态条动态更新播报 | web · Accessibility「Aria Live」· Medium | Do: 动态内容 `aria-live=polite` | 状态条/风险条保留 `aria-live`（现有已具备，勿回退） |

### 8.3 可执行硬规则（写给其他 AI 的约束清单）

后续任何 AI 改本页，须满足：

1. **状态三要素**：任何状态呈现 = 颜色 **+** 图标 **+** 文本，且标注**作用域**（当前场景 / 全部场景）。禁止"仅颜色/仅标签"传递成功、失败、告警。（背书：Color Only · High）
2. **单一主 CTA**：每屏有且仅一个实心主按钮；本页为「保存并发布」，须常驻首屏可见并显示未保存改动数；破坏性操作（放弃草稿/删除/清空密钥）一律次级样式 + 二次确认。（背书：Confirmation Dialogs · High）
3. **文案本地化单一出口**：后端 raw message 一律经映射层再展示，禁止英文技术串直出中文界面。
4. **命名单一来源**：展示层只用 `provider.name`；内部 slug（`summary-bohe` 等）仅进 tooltip / 等宽次要位。
5. **列表可伸缩**：列表项 > 50 走虚拟化；审计必须能按「变更 / 测试」筛选。（背书：Virtualize Lists · High + No filtering 反模式）
6. **无障碍不回退**：图标按钮保留 `aria-label` 并补可见文字/tooltip；表单控件保留 `<label>`；动态状态保留 `aria-live`。（背书：Semantic HTML · High、Form Control Labels · Critical、Aria Live · Medium）
7. **主题约束**：沿用现有浅色 Element Plus token，**不得**为套用 Data-Dense 推荐而改深色主题。

### 8.4 建议 Design Tokens（浅色适配，供直接引用）

在现有 Element Plus 变量基础上，为"状态语义"与"技术标识"补齐 token，统一散落的颜色/字号：

```css
/* 状态语义色（浅色主题；颜色恒与 icon+文本同用，见规则 1） */
--ai-state-success: #16A34A;   /* 通过 / 正常 */
--ai-state-warning: #D97706;   /* 需关注 / 配置校验 */
--ai-state-danger:  #DC2626;   /* 告警 / 失败 / 冷却中 */
--ai-state-info:    #2563EB;   /* 新路由 / 变更 */
--ai-state-neutral: #64748B;   /* 停用 / 未测试 */

/* 技术标识等宽（model / slug / requestId / 时间戳），对应 skill 的 Fira Code 定位 */
--ai-mono-font: "Fira Code", "SFMono-Regular", ui-monospace, monospace;

/* 字号阶梯（背书：Font Size Scale · Medium，禁止随意字号） */
--ai-fs-caption: 12px; --ai-fs-body: 14px; --ai-fs-sub: 16px;
--ai-fs-title: 18px;   --ai-fs-h2: 24px;    --ai-fs-h1: 32px;

/* 交互（背书：hover 150–300ms） */
--ai-transition: 200ms;
```

> 说明：以上为浅色适配值，非 skill 原始深色推荐；如需查证原始推荐执行
> `python3 .claude/skills/ui-ux-pro-max/scripts/search.py "AI infrastructure admin dashboard devops config routing observability" --design-system -f markdown`。

---

## 9. 统一问题索引（Issue Register · 落地以此为准）

前文各节的问题去重后汇总为下表，作为**唯一权威清单**。严重度统一为 高/中/低；"影响面"决定改动风险；"待验证"标记由截图推断、需先核实的项（切勿当既定 bug 直接改）。

| ID | 问题 | 严重度 | 影响面 | 待验证 | 关键代码符号 | 出处 |
| --- | --- | --- | --- | --- | --- | --- |
| I1 | 告警作用域不一致（当前场景 2 vs 状态条 8） | 高 | 前端取数（疑真 bug） | **是**（8=2+2+4 为推断） | `alertStatusSummary` `blockingRiskItems` `currentSceneKey` | 7.1① |
| I2 | 「正式模式」标签与 `-compat` 链路语义矛盾 | 高 | 前端展示/语义 | **是**（compat 判定口径） | `sceneCards.statusText` `currentChannel` | 7.1②/7.6 |
| I3 | 单卡「绿+红」同框，缺聚合结论态 | 中 | 前端展示 | 否 | `sceneCardHealthMap` | 7.1③ |
| I4 | 首屏无主 CTA、保存被弱化为小图标 | 高 | 前端展示 | 否 | `routing-bottom-bar` `toolbar` | 7.2 / P0-3 |
| I5 | 状态重复呈现（链路/正式模式/测试 chip） | 中 | 前端展示 | 否 | `routing-breadcrumb` `routing-status-strip` | 7.3 / P0-2 |
| I6 | 无效配置未拦截（failover + maxAttempts=1） | 中 | 前端校验 | **是**（1 次是否真不切换） | `validateBeforeAction` `maxAttemptCeiling` | 7.4 |
| I7 | 左右栏布局失衡（左留白 / 右超长） | 低 | CSS | 否 | `routing-editor-grid` | 7.5 / P2-1 |
| I8 | 术语暴露（slug / v2 / compat 直出） | 中 | 前端展示 | 否 | `provider.name` vs `id` | 7.6 / 7.8① |
| I9 | 命名双轨（同节点 name 与 slug 并存） | 中 | 前端展示 | 否 | `providerDisplayName` | 7.8① |
| I10 | 中英混杂 raw message 直出 | 中 | 前端映射（源自后端） | 否 | `testResult.message` / audit message | 7.8② |
| I11 | 绿标「测试通过」配「全节点冷却」 | 高 | 前端或后端判定 | **是**（ok 判定位置） | `testResult.ok` | 7.8③ |
| I12 | 审计流被测试噪音淹没 | 中 | 前端筛选（或需后端字段） | 否 | `recentAudits` `auditBusinessAction` | 7.8④ |
| I13 | diff 展示不一致（内联 vs 点开） | 低 | 前端展示 | 否 | audit diff 渲染 | 7.8⑤ |
| I14 | 审计事件标签三套视觉语言 | 低 | 前端展示 | 否 | `auditBusinessAction` | 7.8⑥ |
| I15 | 底部图标无文字、无未保存计数 | 中 | 前端展示 | 否 | `routing-bottom-bar` `sceneDiff` | 7.8⑦ / P0-3 |
| I16 | 测试对象（草稿 vs 线上）不显性 | 中 | 前端展示 | 否 | `routing-test-card` `testActionTooltip` | P1-1 |
| I17 | 新增节点/空场景缺引导 | 低 | 前端展示 | 否 | `handleAddProvider` | P1-2 |
| I18 | 响应式未验证（popover 定宽/双栏） | 低 | CSS | 否 | `routing-editor-grid` `viewportWidth` | P2-1 |

---

## 10. 术语表（给后续 AI / 新协作者）

本页核心概念多且互相纠缠，落地前须先对齐以下词与代码符号的对应关系：

| 术语 | 含义 | 代码符号 |
| --- | --- | --- |
| 场景 scene | 三条 AI 链路之一：正文总结 / 标题清洗 / 步骤图生成 | `currentSceneKey` `sceneKeys`（summary/title/flowchart） |
| 草稿 draft | 本地未保存的改动；有改动即 dirty | `draftScene` `isDirty` `sceneDiff` |
| 新路由（是否启用） | 该场景是否启用多节点路由 | `draftScene.enabled` |
| 兼容模式 compatibility | 运行时仍优先走**旧单 Provider** 配置；未真正切到多节点 | `draftScene.compatibilityMode` |
| 正式模式 | 与兼容模式相对：已切到新多节点路由 | 由 `compatibilityMode=false` 推导 |
| 线上链路 channel | 当前**实际生效**的配置来源标识（如 `summary-v2` / `xxx-compat`） | `currentChannel` `channelMatrix` `sceneEffectiveChannel` |
| 节点 provider | 单个上游（base URL + key + model + 超时…） | `draftScene.providers[]` |
| 熔断 breaker | 连续失败达阈值后暂停该节点一段冷却时间 | `breaker.failureThreshold` `cooldownSeconds` |

> 关键关系：**「新路由已启用」≠「线上已生效」。** 只有 `enabled=true` 且 `compatibilityMode=false` 且已保存（非 dirty），线上链路才真正走新路由——这正是 I1/I2 混淆的根源，也是 §4 P0-1「用一句人话讲结论」要解决的核心。

---

## 11. 落地路线图与非目标

### 阶段划分（按风险从低到高）

**阶段 1 · 纯展示 / 文案（低风险，可直接合）**
I5 删冗余 · I8/I9 命名统一到 `provider.name` · I10 raw message 映射层 · I13/I14 diff 与标签统一 · I16 标注测试对象 · I15 底部补文字 + 未保存计数。均为模板 + CSS + 文案，不动业务逻辑。

**阶段 2 · 一致性 / 取数（须先按 §9「待验证」核实）**
I1 告警作用域 · I2 正式/compat 口径 · I3 聚合结论态 · I11 测试 ok 判定 · I6 无效配置拦截。**先在代码/数据确认推断成立，再改**；I11/I1 若判定逻辑在后端，需 backend 配合。

**阶段 3 · 结构 / 体验增强（较大改动）**
I4 主 CTA 重排 + 保存栏升级 · I12 审计"变更/测试"分离与筛选 · I7 布局 sticky/单列 · I17 空状态引导 · I18 响应式实测。

### 非目标（本轮明确不做，防止过度施工）

- **不**为套用 Data-Dense 推荐而将浅色主题改深色（见 §8.3 规则 7）。
- **不**改后端接口契约——除非 I11 / I12 经核实确需后端补字段，届时单独评估。
- **不**动既有能力：键盘快捷键、锚点目录、审计抽屉、拖拽排序、diff 确认弹窗——这些是前两轮成果，只做收敛不做移除。
- **不**在本文档内直接改代码；本文档只定义"改什么、为什么、验收标准"。

---

## 12. 第四轮跟进（2026-07-03 · 落地后复盘）

§9 的 I1–I18 大部分已在代码落地。基于对已实现版本的再走查，发现 **上一轮"强化 CTA / 补能力"引入的新冗余** 与几个遗留低优先项，编号 A–E（与 I 系列区分）。

| ID | 问题 | 关联 | 严重度 | 状态 |
| --- | --- | --- | --- | --- |
| A | 顶部工具栏与底部悬浮栏渲染两套**完全相同**的动作组（测试草稿 / 保存并发布 / 放弃草稿）+ 状态标签，页面顶部时同屏重复 | I4/I5/I15 回潮 | 高 | ✅ 已完成 |
| C | 左侧「场景策略」`position: sticky` 因祖先 `.layout-card { overflow: hidden }` 建立滚动容器而失效，长列表滚动时左栏不跟随、首屏留白 | I7 | 中 | ✅ 已完成 |
| B | 草稿改动明细仍藏在底部栏「草稿摘要」popover，未按 P0-3 平铺；状态条只给「N 项未保存」计数 | I15 / P0-3 未尽 | 中 | ⏳ 排期 |
| D | Provider 列表（7+ 节点、均可展开）缺全部折叠/展开与按名称/启用状态的轻量筛选，节点增多后难扫描 | §8.2 No filtering | 中 | ⏳ 排期 |
| E | 编辑区上方仍堆叠 4 条横幅（兼容模式 `el-alert` + 面包屑 + 状态条 + 风险条），P0-2 目标 ≤2；兼容提示已被 `currentEffectDescription` 覆盖，可合并 | I5 / P0-2 未尽 | 低 | ⏳ 排期 |

### 已完成（A / C）

- **A** — 底部悬浮栏改为仅在顶部工具栏 CTA 滚出视口后浮现：顶部加哨兵 `topCtaSentinelRef` + `IntersectionObserver`，收起态 `.routing-bottom-bar--tucked`（下移淡出 + `pointer-events:none`），带 `prefers-reduced-motion` 降级与 `aria-hidden` 同步。彻底消除同屏两套相同 CTA。
- **C** — `admin-web/src/style.css` 的 `.layout-card` 由 `overflow: hidden` 改为 `overflow: clip`（同样裁剪但不建立滚动容器，不困住 sticky）。已用最小复现实测：`hidden` 下滚动 600px 左栏 `top=-508`（失效），`clip` 下 `top=24`（正常）。全站唯一 sticky 元素即该面板，零副作用。

### 排期建议（B / D / E）

- **阶段 1（纯展示，低风险）**：**B** 把 `sceneDiff` 前 1–2 条关键改动（如 `Model: a → b`）内联到底部栏/状态条，其余折叠；**E** 移除独立兼容模式 `el-alert`，把结论并入 `routing-status-strip` 的 `currentEffectDescription`，把编辑区上方横幅收敛到 ≤2 条。
- **阶段 3（体验增强）**：**D** Provider 面板头加「全部折叠/展开」开关与轻量筛选（按名称 / 启用状态），为节点增长做准备；暂不做虚拟化（节点 < 50，见 §8.2 阈值）。
