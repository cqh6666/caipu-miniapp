# Project Changelog

## 2026-07-14 (Go 后端可维护性重构 R0～R9 全量收口)

### Changed

- **修改时间**：2026-07-14 07:36:16 +0800
- **变更背景**：后端已按业务域建立包边界，但链接解析、AI 路由、菜谱、饮食管家、运行时配置和应用装配等核心文件
  持续增长到 700～2,300 行，协议、规则、持久化和编排职责混放；同时 `app/config/bootstrap/db/wechat`
  等基础边界缺少自动化回归保护，需要在不改变业务契约的前提下系统降低维护成本。
- **核心改动**：
  - 新增 `docs/backend-refactor-roadmap-2026-07-14.md`，记录 R0～R9 的基线、优先级、验收门禁和逐项结果。
  - `internal/app` 提取 AI/链接解析、饮食管家菜谱工具和 worker 装配；`app.go` 由 734 行降至 293 行。
  - `linkparse`、`dietassistant`、`airouter`、`appsettings` 按服务编排、外部协议、规则、配置更新/缓存和探测拆分。
  - `recipe` 按主实体、自动解析队列、流程图队列、图片、分享、序列化、输入/内容规则拆分 repository/service，
    流程图进一步拆为生成器、客户端和提示词；`audit` 拆分写入、任务/调用查询、Dashboard 和筛选规则。
  - `config` 分离环境解析与校验/回退；新增配置、迁移、SQLite、微信客户端和真实应用路由/鉴权集成测试。
  - 更新 `backend/README.md`，固化 `go test`、`go vet`、全量竞态测试和路线图入口。
- **影响范围**：后端内部代码组织、应用依赖装配、AI/链接解析/菜谱/审计/配置模块和自动化测试；不修改前端、
  HTTP 路由、JSON 字段、数据库迁移、环境变量名称或部署脚本。
- **兼容性或风险**：保留多 Provider 路由/熔断/重试、密钥保留与清空、解析/流程图队列、菜谱状态事件、图片镜像、
  分享令牌、LongCat SSE 工具协议、运行时配置默认值与审计行为。主要风险是跨文件移动遗漏依赖，已由定向测试、
  真实全迁移装配测试和全量竞态门禁覆盖；未引入新运行时依赖。
- **验证情况**：`cd backend && go test ./... -count=1`、`go test -race ./... -count=1`、`go vet ./...`、
  `gofmt -d internal cmd` 与 `git diff --check` 全部通过；整体语句覆盖率 42.8%，`app/config/bootstrap/db/wechat`
  分别为 84.9%/93.1%/81.0%/73.1%/94.7%；最大生产 Go 文件由 2,304 行降至 652 行。

## 2026-07-14 (前端第二阶段重构 R1～R10 全量收口)

### Changed

- **修改时间**：2026-07-14 01:18:25 +0800
- **变更背景**：承接 7 月 11 日的大文件拆分，在页面体量下降后继续处理重复异步流程、数据层职责混杂、
  页面模块隐式依赖、Admin 重复面板和 AI Provider 页面编排过重问题，并建立可持续运行的前端测试入口。
- **核心改动**：
  - 测试入口：根目录新增 Miniapp 规则测试并由 `npm test` 聚合 Miniapp 与 Admin；Admin 测试按规则、Dashboard
    和组件 SSR 分组，移除归属错误的旧测试文件。
  - Miniapp 公共逻辑：添加链接预览收口为共享流程 controller；空间数字动效收口为无 UI count-up controller；
    饮食管家流式响应拆为 UTF-8 decoder、SSE parser 和请求适配层；菜谱数据层拆为 model、cache、repository，
    `recipe-store.js` 保留兼容出口。
  - Miniapp 页面所有权：首页五个业务模块建立显式 contract 与统一 lifecycle；菜谱详情提取加载、步骤完成态和
    动作反馈 controller。首页保持 1,063 行，详情页降至 1,550 行。
  - Admin Dashboard：三组分布区域复用 `DistributionPanel.vue` 和独立转换规则，页面降至 847 行。
  - Admin AI Provider：Provider 编辑和告警动作下沉为 composable；场景策略、保存确认拆为组件；保存差异格式化
    独立为纯规则；`ProviderEditor` 收口为 `selectOptions + editor` 契约；页面降至 2,357 行。
  - 新增 `docs/frontend-refactor-roadmap-2026-07-13.md`，记录 R1～R10 的范围、依赖、验收标准和实施结果。
- **影响范围**：微信小程序首页、菜谱详情、空间统计、添加链接预览、饮食管家流式响应与菜谱本地/远端数据层；
  Admin Dashboard、AI Provider 编辑/告警/场景策略/保存确认；前端依赖、测试脚本和重构文档。后端 API、数据库、
  持久化字段和路由地址均未调整。
- **兼容性或风险**：保留 Options API、uni-app 微信小程序编译方式、旧菜谱导入出口、密钥留空保留/显式清空、
  草稿离页保护、公开菜谱只读和告警旧字段兼容。Admin 登录态下的完整保存、告警处置和审计筛选未做人工点击复验；
  微信自动预览已触发成功但未逐页人工走查。Vite 仍保留既有 Element Plus 大 chunk 警告。
- **验证情况**：`npm test`、`npm --prefix admin-web run typecheck`、`npm --prefix admin-web run build`、
  6 个本次修改的 Miniapp SFC 脚本/模板编译与 `git diff --check` 均通过；HBuilderX 5.07 明确输出
  “项目 caipu-miniapp 编译成功”，微信开发者工具 `auto-preview` 成功完成。最终已使用 `git status --short`
  审计改动范围。

## 2026-07-13 (Admin Web AI Provider 节点编辑区空白修复)

### Fixed

- **修改时间**：2026-07-13 14:04:29 +0800
- **变更背景**：7 月 11 日将 Provider 编辑区拆为 `ProviderEditor.vue` 后，子组件模板继续读取
  `helpTips.providerName` 等说明文案，但父页面未通过组件契约传入 `helpTips`。线上有已配置节点且首节点展开时，
  Vue 渲染抛出 `Cannot read properties of undefined (reading 'providerName')`，导致右侧节点编辑区只剩空白卡片。
- **核心改动**：`AIProvidersPage.vue` 显式向 `ProviderEditor` 传入 `helpTips`；子组件补齐必填
  `ProviderHelpTips` prop，并修正 Provider 预设项 `title` 类型；新增 Vite SSR 组件渲染检查，覆盖“已有节点且
  首节点展开”的真实触发条件，并纳入 `test:frontend-refactor`。
- **影响范围**：仅影响 `admin-web` 的 AI Provider 节点编辑区和前端回归测试；不修改后端 API、数据库、
  已保存的 AI 路由配置或小程序端。
- **兼容性或风险**：组件说明文案仍由父页面统一维护，Provider 配置读写契约不变；生产构建保留既有
  Element Plus 大 chunk 警告。
- **验证情况**：本地 `npm run typecheck`、`npm run test:ai-provider-utils`、
  `npm run test:frontend-refactor` 与 `npm run build` 均通过；本地构建产物已通过
  `scripts/upload-admin-web-dist.sh` 发布到 `my-cloud`，发布版本 `20260713-140308`，公开入口返回 HTTP 200。
  登录态线上复验确认正文总结右侧完整展示 `3/7` 个启用节点、7 个节点摘要与首节点编辑字段，控制台不再出现
  `providerName` TypeError；仍保留既有 Element Plus `ElOnlyChild` 与 checkbox 弃用警告。

## 2026-07-11 (前端大文件拆分 A～E 全量收口)

### Added

- **修改时间**：2026-07-11 16:20:05 +0800
- **变更背景**：小程序首页、菜谱详情页与 Admin Web AI Provider 管理页已增长为数千行页面级单体，多个业务域、
  异步状态和展示规则集中在同一 SFC，后续功能迭代与回归范围难以控制，需要先建立可持续执行的拆分看板，
  再按可独立验证的工作包逐步落地。
- **核心改动**：
  - Admin AI Provider：新增告警、审计、草稿和校验规则模块，新增 `useAIRoutingDraft.ts`，并拆出场景卡、
    告警生命周期、Provider 编辑、测试结果和审计视图组件；`AIProvidersPage.vue` 由 7,991 行降至 3,276 行。
  - 小程序首页：新增菜谱库、地点库、菜单同步、空间成员和智能添加五个业务模块，拆出
    `library-pane.vue` / `place-pane.vue`，统一搜索失焦与菜单同步计时器所有权；二次审计后将五个业务域的
    methods/computed 全部下沉，首页由 5,490 行降至 1,100 行。
  - 菜谱详情：新增编辑、异步任务、图片和分享四个模块，拆出只读提示、Hero、做法、流程图和编辑器组件；
    二次审计后由编辑器拥有本地草稿和关闭确认，由三个 controller 分别拥有轮询、图片和分享流程，详情页由
    5,491 行降至 1,587 行。微信运行时冒烟发现并修复拆分遗漏的 `toPositiveInteger` 引用。
  - 次级收口：Dashboard 新增 `useDashboardCharts.ts` 管理 ECharts 生命周期；Admin 全局样式拆为
    `tokens/base/shell/components/responsive` 五层，`style.css` 仅保留稳定导入；复核三个次级大组件后暂不机械拆分。
  - 新增 AI Provider 与小程序重构规则检查脚本，并将 A～E 状态、文件体量、D3 结论和验收结果回写到
    `docs/frontend-large-file-refactor-todo-2026-07-11.md`。
- **影响范围**：Admin AI Provider、Dashboard 和全局样式组织；小程序首页、地点详情入口、菜谱详情内部组织；
  不修改后端 API、数据库、持久化结构或业务字段契约。
- **兼容性/风险**：保留旧告警字段、未知枚举、密钥保留/清空、多节点失败切换、公开只读和旧菜谱状态回退。
  Admin 登录态下的保存、告警动作与审计筛选尚未人工复验，因为当前 Browser 运行时无可用实例；
  微信开发者工具仍显示既有基础库 timeout、废弃属性和 WXSS 选择器警告。
- **验证情况**：`test:ai-provider-utils`、`test:frontend-refactor`、TypeScript、Vite build、全部新增 JS 语法、
  首页/详情及子组件 SFC 编译、`git diff --check` 均通过；HBuilderX 5.07 微信小程序编译成功，
  微信开发者工具自动预览成功，并完成美食库、打卡点、地点详情和私有菜谱详情关键冒烟。
  Vite 保留既有 `element-plus` 大 chunk warning。
  16:20 二次所有权审计后再次执行上述自动化门禁；同时修复模块对象闭合和 import 位置错误，HBuilderX 明确输出
  “项目 caipu-miniapp 编译成功”，微信开发者工具自动预览再次成功。

## 2026-07-06 (Admin Web AI Provider 归档后场景卡状态文案修正)

### Fixed

- **修改时间**：2026-07-06 11:18:11 +0800
- **变更背景**：用户在线上验证 AI Provider 告警归档后，发现对应场景卡仍显示笼统的“需关注”状态，
  容易误解为归档未生效或仍存在红色当前告警。
- **核心改动**：`admin-web/src/pages/AIProvidersPage.vue` 调整场景卡聚合状态规则：场景卡主状态只由
  当前告警生命周期、发布阻断风险、缺密钥和兼容模式驱动；“最近测试异常”不再单独把场景卡顶成“需关注”。
  黄色告警生命周期状态统一显示为“待复核”，兼容模式显示为“兼容模式”，避免所有非绿色状态都混成“需关注”。
- **影响范围**：仅影响 `admin-web` 的 `AI Provider` 场景卡状态标签与问题摘要展示；
  不修改后端 API、数据库、告警状态计算、AI 路由配置或小程序端。
- **兼容性或风险**：若同场景仍存在未归档的 `stale / pending_verify / muted` 告警，场景卡仍会显示“待复核”；
  归档所有待复核项后，且无配置风险/兼容模式时，才会回到“运行正常”。
- **验证情况**：已执行 `npm --prefix admin-web run build` 与
  `git diff --check -- admin-web/src/pages/AIProvidersPage.vue`，均通过；Vite 仍提示既有大 chunk warning。

## 2026-07-06 (AI Provider 告警状态生命周期：文档收口 + 全链路实现)

### Added

- **修改时间**：2026-07-06 15:55:00 +0800
- **变更背景**：承接同日《告警状态交互优化需求文档》，先完成一轮产品交互评审收口，再落地前后端实现，
  把“当前需要处理的红色告警”和“历史/过期/静默的可追溯记录”彻底拆开。
- **文档收口**：重写 `docs/admin-ai-provider-alert-state-ux-design-2026-07-06.md`：状态与颜色一一映射
  （`muted` 统一黄色 + 静音角标，不再与 `archived` 共用灰色）；补齐状态机缺口（`pending_verify` 入边、
  `muted → archived`、手动解除静默、`recovered → normal`）；`pending_verify` 触发收敛为“仅对已有告警节点”；
  `stale` 展示名改为“待复测（已过期）”避免“已解决”错觉；新增批量处置、复测成本提示、空态、活跃窗口倒计时、
  日志跳转带上下文等交互细则；明确告警状态为 per-Provider 粒度（Provider 归属唯一场景）。
- **后端接口契约**：
  - `GET /api/admin/ai-routing/alerts/overview` 的 `items[]` 新增 `alertStatus`（`normal/active/stale/pending_verify/muted/archived/recovered`）、
    `statusReason`、`activeUntil`、`mutedUntil`、`archivedAt`、`isProviderEnabled`、`isInEffectiveRoute`、
    `canRetest/canArchive/canMute/canUnmute` 等字段；概览新增 `activeWindowHours` 与各状态计数
    （`staleAlertCount/pendingVerifyCount/mutedAlertCount/archivedAlertCount/recoveredCount/reviewAlertCount`）。
    旧字段 `thresholdReached` 过渡期保留。
  - 新增管理接口：`POST /ai-routing/alerts/{providerId}/retest|archive|mute|unmute`，统一返回
    `{ ok, message, overview }`。
  - 新增运行时配置 `ai.provider_alert.active_window_hours`（默认 72h），控制红色→黄色（待复测）的降级窗口。
- **数据库迁移**：`backend/migrations/023_add_ai_provider_alert_lifecycle.sql` 为
  `ai_provider_alert_states` 增加 `archived_*/muted_*/last_config_changed_at` 字段，并新增
  `ai_provider_alert_events` 处置事件表（`retest_succeeded/retest_failed/archived/muted/unmuted/config_changed`）。

### Changed

- **后端状态计算**：`aialert` 在 `Overview` 阶段按活跃窗口 + Provider 运行时存在性（由 `airouter`
  经 `ProviderStatusResolver` 反向注入）推导 `alertStatus`；`airouter` 实现单节点 `RetestProvider`
  （`route_test` 输入跳过自动追踪，由 `aialert` 显式落状态）。`app.go` 用 setter 注入打破包级循环依赖。
  保存场景配置后由 admin handler 调用 `NoteSceneConfigChanged`，对该场景已有告警节点标记配置变更。
- **前端交互**：`admin-web/src/pages/AIProvidersPage.vue` 改用 `alertStatus` 分层（红=active、黄=待复核、
  灰=已归档、绿=正常/已恢复）；场景卡在有红色时并列展示“告警中 N · 待复核 M”；“告警配置”弹层重构为
  三段式（当前告警 / 待复核 / 最近恢复）+ 节点处置动作（复测并恢复 / 静默 24h / 解除静默 / 归档 / 查看日志）
  + 批量复测/归档 + 复测真实调用成本提示 + 空态；`requestId` 可复制；`types.ts`/`api/admin.ts` 同步扩展。
- **行为语义**：新失败自动解除归档并重新计数；真实成功/复测成功清零计数并解除归档与静默；静默期不计入红色告警。

### 影响范围与验证

- **影响范围**：后端 `aialert/airouter/admin/appsettings/common`、迁移、管理路由；`admin-web` 类型/API/AI Provider 页；产品文档。不影响小程序端与部署脚本。
- **兼容性/风险**：无 resolver 时按“仍在路由”兜底以保持旧行为；复测会触发真实上游调用（前端已加成本二次确认）；
  Provider 删除后仅保留“归档 / 查看日志”。多实例部署以数据库为准。
- **验证情况**：`cd backend && go build ./... && go test ./...` 全通过（新增 `aialert` 生命周期/状态计算/动作单测、
  更新 admin/airouter 测试桩）；全量迁移 SQL 对临时 SQLite 库应用通过，确认 023 新列与事件表可创建；
  `admin-web` `vite build` 通过（仍有既有大 chunk warning）。
  未做浏览器端联调（需后端 + 登录态 + 告警数据种子，成本较高）。

## 2026-07-06 (Admin Web AI Provider 告警状态交互优化需求文档)

### Added

- **修改时间**：2026-07-06 10:12:37 +0800
- **变更背景**：后台 `AI Provider` 告警状态当前以连续失败计数作为红色告警依据，
  已触发告警的 Provider 若长期没有真实业务成功调用，页面会持续显示“告警中”，容易把历史失败误判为当前故障。
- **核心改动**：新增
  `docs/admin-ai-provider-alert-state-ux-design-2026-07-06.md`，定义“当前告警 / 历史告警 /
  待复测 / 已恢复 / 已归档 / 已静默”的产品状态模型、场景卡与告警弹层交互、后端 overview
  字段扩展、归档/静默/复测接口建议、数据模型建议与分阶段落地计划。
- **影响范围**：仅新增产品需求文档与项目变更记录；未修改前后端运行时代码、接口实现、数据库迁移或部署脚本。
- **兼容性或风险**：文档中的接口和数据字段为后续实现建议，实际落地前需结合后端调用频率、复测成本、
  Provider 删除后的历史快照保留策略继续细化。
- **验证情况**：已执行 Markdown 文档创建与变更范围检查；无需运行构建。

## 2026-07-06 (Admin Web AI Provider 场景卡告警摘要补强)

### Changed

- **修改时间**：2026-07-06 09:57:02 +0800
- **变更背景**：`admin-web` 的 `AI Provider` 场景卡在第三轮 UX 收敛后仍需要进一步减少
  健康状态明细堆叠，并让“为何告警”在当前场景内更容易定位到具体节点。
- **核心改动**：
  - `admin-web/src/pages/AIProvidersPage.vue` 将场景卡内“最近测试 / 配置风险 / 告警状态”
    三行明细收敛为仅在需处置时展示的一句话问题摘要。
  - 告警摘要优先展示当前场景中连续失败次数最高的节点、失败次数、错误类型和相对失败时间；
    多节点告警时补充告警节点数量。
  - “告警配置”弹层新增当前场景正在告警的节点列表，展示 Provider 名称、模型、连续失败次数、
    错误类型、最后错误和 requestId，便于从页面直接定位问题。
  - 补充问题摘要与告警节点列表的局部样式，并清理已不再使用的场景卡健康快照样式。
- **影响范围**：仅影响 `admin-web` 的 `AI Provider` 管理页展示与前端告警摘要文案；
  不修改后端 API、数据库、AI 路由保存 payload、小程序页面或部署脚本。
- **兼容性或风险**：告警明细依赖现有 `AIRoutingAlertOverview` 返回字段；未知后端错误类型会原样展示，
  避免因前端未覆盖枚举而隐藏真实错误。
- **验证情况**：已执行 `npm --prefix admin-web run build` 与
  `git diff --check -- admin-web/src/pages/AIProvidersPage.vue`，均通过；Vite 仍提示既有
  大 chunk 体积 warning，不影响本次构建产物生成。

## 2026-07-03 (Admin Web AI Provider 管理页 UX 收敛)

### Changed

- **修改时间**：2026-07-03 18:39:45 +0800
- **变更背景**：根据 `docs/admin-ai-provider-ux-review-2026-07-03.md` 第三轮评审，
  `admin-web` 的 `AI Provider` 页面功能已较完整，但存在状态重复、线上生效链路难以一眼判断、
  保存发布动作弱化、测试/审计反馈噪音过高等问题，需要在不改后端契约的前提下做信息收敛。
- **核心改动**：
  - `admin-web/src/pages/AIProvidersPage.vue` 将顶部工具栏升级为草稿测试、保存并发布、放弃草稿和刷新
    的统一动作组；底部悬浮栏改为文字按钮与默认可见草稿摘要，避免保存动作只靠圆形图标猜测。
  - 场景卡从“正式模式/兼容模式”单标签改为聚合结论态（未配置 / 需关注 / 告警中 / 运行正常），
    主编辑区新增“一句话线上生效结论”，把 `summary-v2`、`title-compat` 等内部标识降级到说明弹层。
  - 告警状态按当前场景统计并在文案中标注作用域；测试结果区显式说明“测试对象：当前草稿”，
    并把 `route test succeeded`、`ok via ...`、`all providers are cooling down` 等原始英文消息映射为中文。
  - 最近审计增加“全部 / 配置变更 / 测试执行”筛选，默认多拉取最近 12 条后本地分组，降低测试记录淹没配置变更的概率。
  - 新增空 Provider 场景的常见节点模板入口；多启用节点下 `maxAttempts < 2` 会在风险区提示并在保存/测试前拦截。
  - 弹层宽度改为响应式，页面内部定位滚动从 `scrollIntoView` 改为显式 `window.scrollTo`。
  - （补充 20:35）底部悬浮操作栏改为仅在顶部工具栏 CTA 滚出视口后才浮现（`IntersectionObserver` 观察顶部哨兵，
    `.routing-bottom-bar--tucked` 平滑收起），消除顶部与底部同时出现两套相同「测试草稿 / 保存并发布 / 放弃草稿」的重复。
  - （补充 20:35）修复左侧「场景策略」面板 `position: sticky` 因祖先 `.layout-card { overflow: hidden }` 建立滚动容器
    而失效的问题：`admin-web/src/style.css` 将其改为 `overflow: clip`（同样裁剪但不困住 sticky），滚动长 Provider 列表时左栏跟随。
- **影响范围**：仅影响 `admin-web` 的 `AI Provider` 管理页展示、前端校验、测试/审计文案和本地构建产物；
  不修改后端 API、数据库、AI 路由保存 payload 结构或小程序端页面。
- **兼容性或风险**：真实 AI Provider 页面需要后台登录态才能完整目视复验；本地仅验证未登录重定向路径。
  审计筛选基于现有 `action` 字段做前端分组，不改变服务端审计查询接口。
- **验证情况**：已执行 `npm --prefix admin-web run build` 与
  `git diff --check -- admin-web/src/pages/AIProvidersPage.vue`，均通过；本地启动 Go 后端和 Vite dev server 后，
  访问 `/admin/ai-providers` 未登录态可正常跳转登录页且新标签页无控制台 warn/error；已将本地构建产物上传并激活到
  `my-cloud`，`curl -I https://www.gxm1227.top/admin/` 返回 HTTP 200。

## 2026-07-03 (AI Provider 正文总结兼容默认流式网关)

### Fixed

- **修改时间**：2026-07-03 17:44:28 +0800
- **变更背景**：后台 `AI Provider -> 正文总结` 测试接入 `https://x666.me/v1`
  的 `grok-4.20-fast` 时，连通性正常但测试反馈 `invalid chat completion response`。
  实测该网关在未显式声明 `stream:false` 时会返回 SSE `chat.completion.chunk`
  数据流，而后端当前按非流式 JSON 响应解析。
- **核心改动**：`backend/internal/airouter/service.go` 对文本类
  `chat/completions` 请求始终显式发送 `stream` 字段，默认值为 `false`，
  避免 OpenAI-compatible 网关把省略字段解释为流式输出；新增单元测试覆盖
  `summary` 场景请求体必须包含 `stream:false`。
- **影响范围**：后端 AI 多 Provider 路由的 `summary / title / flowchart`
  文本类 `chat/completions` 请求；不影响 `images/generations` 生图链路或
  admin-web 页面交互。
- **兼容性或风险**：若某节点在后台配置里显式开启 `requestOptions.stream=true`，
  请求仍会发送 `stream:true`，但当前后端仍主要按非流式响应解析；正文总结默认
  测试与生产解析建议保持 `stream:false`。
- **验证情况**：已执行 `go test ./internal/airouter`；并用 `grok-4.20-fast`
  实测确认未带 `stream:false` 时返回 SSE，显式带 `stream:false` 后返回标准
  `chat.completion` JSON。

## 2026-07-03 (空间洞察分组数字动效与小程序渲染修复)

### Fixed

- **修改时间**：2026-07-03 10:22:21 +0800
- **变更背景**：用户反馈空间洞察页 `美食库 / 打卡库` 分组的数据数字没有像首页空间概览卡一样播放动效，随后发现接入数字动效后小程序端数字区域为空，只剩 `/`、标签和进度条。
- **核心改动**：
  - `pages/space-stats/index.vue` 不再把洞察页数字交给自定义数字组件渲染，改为与首页空间概览卡一致的页面内 `animatedNumbers` 状态 + count-up 计时器 + 普通 `<text>{{ ... }}</text>` 输出，规避微信小程序端自定义组件嵌套导致文本为空的风险。
  - `美食库 / 打卡库 / 菜单安排` 分组改为首次切到 Tab 时再挂载对应内容，核心比例卡与完整度数字会从 0 滚动到目标值；后续刷新或远端 stats 更新时，已挂载分组数字重新按当前值过渡到新值。
  - 动效时长沿用 `640ms` 与 cubic ease-out 口径，不调整现有卡片视觉、配色、接口契约或数据来源。
- **影响范围**：仅前端小程序空间洞察独立页的分组 Tab 数字展示与动效；不影响首页空间概览卡、后端 `GET /api/kitchens/{kitchenID}/stats` 接口或统计口径。
- **兼容性或风险**：依赖页面计时器驱动数字过渡；真机 / 微信开发者工具仍需复验 Tab 首次切换和刷新后的体感。
- **验证情况**：已执行 `pages/space-stats/index.vue` 的 `<script>` 语法检查与 `git diff --check -- pages/space-stats/index.vue`，均通过；未执行 HBuilderX / 微信开发者工具构建。

## 2026-07-03 (首页美食库与打卡点支持左右滑切换与动画)

### Added

- **修改时间**：2026-07-03 09:55:54 +0800
- **变更背景**：用户希望首页 `美食库 / 打卡点` 除点击顶部分段按钮外，也能通过触摸左滑 / 右滑切换，并补充滑动方向感动画。
- **核心改动**：
  - `pages/index/index.vue` 在清单页容器上新增 `touchstart / touchend / touchcancel` 手势监听，横向位移达到
    `64px` 且明显大于纵向位移时触发模式切换。
  - 左滑从美食库切到打卡点，右滑从打卡点切回美食库；切换仍复用既有 `switchAppMode`，因此打卡点进入时继续触发
    `refreshPlaces({ silent: true })`。
  - 新增 `appModeMotionDirection` 内容面板进入动画：切到打卡点时从右侧轻滑入，切回美食库时从左侧轻滑入，时长
    `240ms`，沿用首页既有 `cubic-bezier(0.2, 0.8, 0.2, 1)` 动效节奏。
  - 搜索框聚焦时不启动模式手势，降低输入场景误触；`library-header-section.vue` 的菜单 spotlight 左右滑增加
    `.stop`，避免其内部轮播手势冒泡触发模式切换。
  - `kitchen-section.vue` 新增 `overview` slot，首页空间概览卡继续挂在空间页头部信息之后，保持空间页信息层级稳定。
- **影响范围**：仅前端小程序首页清单内的 `美食库 / 打卡点` 模式切换与空间页概览卡挂载位置；不影响底部 `清单 / 空间` 导航、弹层或后端接口契约。
- **兼容性或风险**：依赖微信小程序触摸事件；极端斜向滑动不会触发切换，以优先保护上下滚动体验。手势体感需在微信开发者工具或真机复验。
- **验证情况**：已执行首页与 `library-header-section` 的 `<script>` 语法检查、`git diff --check`；真机 / 微信开发者工具手势体验待复验。

## 2026-07-03 (手动添加菜品表单标题衬线化)

### Changed

- **修改时间**：2026-07-03
- **变更背景**：用户希望手动添加菜品页面的顶部标题更精致，调整为衬线体视觉。
- **核心改动**：`pages/index/components/add-recipe-sheet.scss` 中手动添加菜品抽屉标题
  `.sheet__title` 使用 `Songti SC / STSong / SimSun / Noto Serif SC / serif` 字体栈，
  并将字重从 `700` 调整为 `600`、补充 `line-height`，让中文标题更轻、更接近现有
  美食空间衬线风格。
- **影响范围**：仅前端小程序手动添加菜品表单标题；不影响智能解析入口、表单字段、保存逻辑或后端接口契约。
- **兼容性或风险**：部分安卓设备可能回退系统 serif，但不影响功能。
- **验证情况**：已执行 `git diff --check`；真机 / 微信开发者工具视觉效果待复验。

## 2026-07-01 (空间洞察由半屏弹层改为独立页面)

### Changed

- **修改时间**：2026-07-01
- **变更背景**：用户反馈半屏（`up-popup`）展示空间较小，希望“查看洞察”跳转到独立页面，浏览更从容。
- **核心改动**：
  - 新增页面 `pages/space-stats/index.vue`（`pages.json` 注册 `navigationBarTitleText: 空间洞察`），复用原半屏的四 Tab 内容与白卡样式（`@import` 复用 `pages/index/components/space-stats-sheet.scss`）。
  - 新增 `utils/space-stats-bridge.js` 作为首页 ↔ 洞察页的内存桥：进入时传统计快照 + `kitchenId`（深拷贝，避免跨页响应式引用），页内动作（进店详情 / 看草稿）写回 `pendingAction`，由首页 `onShow` 消费执行页内切换。
  - 洞察页自带刷新：`onLoad` 用快照秒开，随后静默 `getKitchenStats(kitchenId,{window:'all'})` 补齐后端数据（趋势 / 成员贡献 / 评分分布），失败保留快照并提示。
  - 首页 `openSpaceStatsSheet` → `openSpaceStatsPage`：`navigateTo` 洞察页并同步刷新首页侧统计；移除 `<space-stats-sheet>` 使用、组件导入与注册、`showSpaceStatsSheet` 状态、`closeSpaceStatsSheet` 及各处调用。
  - 删除 `pages/index/components/space-stats-sheet.vue`（样式 `.scss` 保留，供新页面复用）。
- **影响范围**：仅前端小程序；点击“查看洞察”由弹半屏改为整页跳转，返回用原生返回；不改后端接口契约。
- **兼容性或风险**：洞察页刷新只拉后端 stats，后端不可用时保留进入时的本地聚合快照（不再在页内做本地重聚合）；页内动作依赖首页 `onShow` 消费，返回首页后生效。
- **验证情况**：`node --check utils/space-stats-bridge.js` 通过；页面跳转 / 刷新 / 动作回跳需在微信开发者工具复验。

## 2026-07-01 (空间概览卡 + 空间洞察半屏 UI 改版：衬线数字 / 白卡风格 / 去时间窗口)

### Changed

- **修改时间**：2026-07-01
- **变更背景**：用户参照原型（`我们的美食空间/` React mock）反馈空间概览与洞察半屏信息偏多、风格不统一，要求精简展示、只留有用模块，并统一为原型的白卡视觉。
- **首页空间概览卡（`space-stats-card`）**：
  - 数字改衬线字体栈（`Georgia/Songti SC`），大数字 52→56rpx、字重 700→600。
  - “查看洞察”从底部整行大按钮移到右上角淡陶土 pill；空态交给 empty-state 次级入口，去掉重复。
  - 移除首页“本地缓存 · X 前 / 重新同步”chip（缓存来源与手动刷新收敛到洞察半屏内），并清理随之无用的 `isCacheSnapshot` prop / `refresh` emit / `updatedAtLabel`。
- **空间洞察半屏（`space-stats-sheet`）**：
  - 四个 Tab 统一为「白卡 + 衬线数值 + 比例(X / Y)/进度条/标签」视觉：新增 `ratio-card` / `insight-card` / `progress-*` 组件类。
  - **美食库**：想吃/吃过、早餐/正餐改比例卡；数据完整度折叠 → 常驻进度条卡（图片覆盖率 % / 已智能解析 道）。
  - **打卡库**：想去/去过、有定位比例改比例卡；重访分布、高频场景标签、常点推荐改白卡承载；去掉 POI 折叠区。
  - **总览**做减法：移除“资产总量”四宫格（与首页重复）与“待行动”列表；保留高分复访推荐、近期动态（升级白卡）+ 趋势柱、成员贡献（改白卡 + 分隔线行）。
  - **菜单安排**：4 宫格 + dual-bar → 比例卡（已安排/草稿天数、平均每餐、早餐/正餐）+ 下一次安排白卡。
- **去掉时间维度窗口**：移除半屏 `7d/30d/90d/all` 切换器及相关逻辑（`change-window` emit / `handleSpaceStatsWindowChange` / `windowLabel` 等）；前端默认口径固定为 `all`（`index.vue` 的 `spaceStatsWindow: 'all'`）。
  - 本地聚合 `buildSpaceStats` 新增 `windowToDays` 映射，`all` 表示不做时间过滤，使“近期动态”本地口径与 `window=all` 一致（不再固定近 30 天）。
- **影响范围**：仅前端小程序（空间概览卡 / 洞察半屏 / 本地聚合口径）；不改后端接口契约，`getKitchenStats` 仍按 `window` 请求（现固定 `all`）。
- **兼容性或风险**：衬线数字在安卓个别机型可能回退系统 serif；窗口切换能力下线，如需按时间筛选需另行恢复。
- **验证情况**：`node --check utils/space-stats.js` / `utils/space-stats-api.js` 通过；四个 Tab 真机观感待用户在微信开发者工具复验。

## 2026-07-01 (空间统计入口点击与数据兜底修复)

### Fixed

- **修改时间**：2026-07-01
- **变更背景**：用户反馈空间页“查看洞察”点击没反应，并且统计拿不到相关数据；排查发现入口热区偏小且靠近固定底部导航，同时后端 stats 请求失败会被静默回退，容易表现为没有任何反馈。
- **核心改动**：
  - 扩大 `space-stats-card` 的“查看洞察”点击热区并阻止 tap 冒泡，降低被底部导航层或空态区域干扰的概率。
  - 打开空间洞察时，如果当前统计仍为空，主动串起菜谱 / 打卡点 / 菜单 / 后端 stats 同步，避免只展示旧的本地空快照。
  - 后端 stats 拉取失败时不再完全静默：半屏显示“后端统计暂不可用，已显示本地聚合”，手动刷新时给 toast，并在控制台输出诊断日志。
  - 本地聚合数据统一标记为 `cache`，只有 `spaceStatsRemote` 成功落地时才标记为 `remote`，避免把列表同步误判为后端统计成功。
  - 空态卡片内部补充“查看洞察”次级入口；同一空间空态自动同步只触发一次，避免用户反复打开半屏时重复打接口。
  - 远端失败提示与“本地聚合”来源 chip 合并展示，减少半屏头部信息噪音。
- **影响范围**：空间页统计卡点击、空间洞察半屏、后端 stats 请求失败兜底展示；不改后端接口契约。
- **兼容性或风险**：后端 stats 不可用时仍展示本地聚合；新增提示只在失败时出现，正常远端统计不受影响。
- **验证情况**：已执行 `node --check utils/space-stats.js`、`node --check utils/space-stats-api.js`、`go test ./internal/spacestats`；真机点击与微信开发者工具弹层仍需用户侧复验。

## 2026-07-01 (空间统计前端 V1.1 + V2：分组 Tab / 后端 stats 对接 / 云端验证)

### Added

- **修改时间**：2026-07-01
- **变更背景**：在 V1 前端闭环基础上，按用户要求一次性补齐设计文档 V1.1 + V2 全部前端编码。
- **V1.1（`space-stats-sheet.vue` / `index.vue`）**：
  - 半屏由单 Tab 扩展为「总览 / 美食库 / 打卡库 / 菜单安排」分组 Tab，按 §5.5 指标分层
    （用户价值指标默认露出，图片覆盖 / 定位 / POI 等数据健康指标折叠进「数据完整度」）。
  - 打卡库新增重访评分分布（横向条）、推荐项 Top / 场景标签 Top（标签云）、缺项补全引导。
  - 打卡点状态筛选补充「全部 / 想去 / 去过」计数（`placeStatusCount`），与美食库计数对齐。
- **V2（`utils/space-stats-api.js` / `utils/space-stats.js` / 组件）**：
  - 新增 `getKitchenStats(kitchenId,{window})` 封装 `GET /caipu-api/kitchens/{id}/stats`。
  - `mapRemoteStatsToViewModel` 把后端响应归一为与 `buildSpaceStats` 同构的视图模型，组件无需感知数据源。
  - `index.vue` 数据源切换：优先后端 stats（趋势 / 成员贡献 / 评分分布），失败静默回退本地聚合，
    保留 `source: remote/cache`；空间切换清空远端快照。
  - 半屏新增 `7d/30d/90d/all` 窗口切换、迷你趋势 bar、成员贡献分组。
  - `buildSpaceStats` 输出扩展 `revisitRatingDistribution` / `itemsByMealType` / `members` / `trends` / `window`，
    使本地聚合也能驱动 V1.1 分组 Tab。
- **接口契约健壮性**：adapter 兼容后端字段改名过渡期——线上二进制（6/29 构建）overview 仍是
  旧字段 `weeklyAvailableRecipes` / `weekendAvailablePlaces`，7/1 的 `wishlistRecipeTotal` /
  `wantPlaceTotal` 改名尚未部署；adapter 对两种字段名都兼容，并忽略后端过时的 action label
  （按稳定的 `target` 重新生成「想吃 N 道菜可选」等文案）。
- **云端后端数据链路验证（`ssh my-cloud`，只读）**：后端 `/api/healthz` 200；migration 021 的
  `recipe_status_events` / `place_status_events` / `recipes.done_at` / 结构化价格字段均已落地；
  §9.3 所需列齐全；用 prod JWT 签测试 token 端到端调用 `/stats`，`7d/30d/90d/all` 正常、
  非法 window 返回 40000；真实响应经本地 adapter 归一正确（想吃 34、成员贡献 3 位、趋势有数据点）。
- **待办**：后端需重新部署以使 7/1 字段改名（`wishlistRecipeTotal` / `wantPlaceTotal`）在线上生效；
  前端已兼容，不阻塞。

## 2026-07-01 (空间统计前端 V1：概览卡 + 半屏总览单 Tab)

### Added

- **修改时间**：2026-07-01
- **变更背景**：按 `docs/space-statistics-capability-design.md` §11 V1 范围落地前端闭环，
  基于首页已同步的 `recipes` / `places` / `mealOrderStore` / `kitchenMembers` 本地聚合，
  不新增后端接口与数据库迁移。
- **新增文件**：
  - `utils/space-stats.js`：`buildSpaceStats(...)` 纯函数聚合（概览 4 数字、想吃/想去、
    高分复访 Top 3、待行动、近 30 天动态）+ `formatRelativeUpdatedAt`；覆盖空数据、
    缺字段、异常日期、标签去重等边界。
  - `styles/space-stats-tokens.scss`：Nature Distilled 语义色 / 圆角 / 阴影 token
    （以 mixin 注入组件 scoped 样式，不污染全局）。
  - `pages/index/components/space-stats-card.{vue,scss}`：2×2 数字格 + 想吃/想去行动条 +
    同步/缓存/空态；数字 count-up 动效；事件 `open-stats / action / refresh`。
  - `pages/index/components/space-stats-sheet.{vue,scss}`：半屏单 Tab 总览（资产总量 →
    高分复访 Top 3 → 待行动 → 近期动态），顶部显式"刷新"按钮 + 更新时间，
    半屏内不做下拉刷新。
- **首页接线**（`pages/index/index.vue`）：新增 `spaceStats` / `isSpaceStatsCacheSnapshot`
  computed，`openSpaceStatsSheet` / `closeSpaceStatsSheet` / `refreshSpaceStats` /
  `handleSpaceStatsAction` 方法；空间页主卡片下拉刷新（`onPullDownRefresh`，
  `pages.json` 开启 `enablePullDownRefresh`）；`refreshRecipes` 成功/失败路径维护
  `spaceStatsSource` / `spaceStatsSyncedAt` 新鲜度标识；统计项跳转美食库/打卡点筛选与
  店铺详情。
- **范围说明**：分组 Tab、列表轻量计数（V1.1）、后端 stats 接口对接（V2）未包含。

## 2026-07-01 (空间统计需求体验收敛 + 接口字段命名对齐)

### Changed

- **修改时间**：2026-07-01
- **变更背景**：对 `docs/space-statistics-capability-design.md` 做交互体验评审后收敛需求，
  其中"去掉本周/周末伪时效命名"导致前端文档字段与已落地后端接口出现命名偏差，需统一。
- **接口契约变更**：`GET /api/kitchens/{kitchenID}/stats` 响应 `overview` 下字段改名，
  前后端统一为"想吃/想去"语义，去除伪时效命名：
  - `weeklyAvailableRecipes` → `wishlistRecipeTotal`
  - `weekendAvailablePlaces` → `wantPlaceTotal`
  - 涉及 `backend/internal/spacestats/model.go`、`repository.go`、`repository_test.go`；
    该接口目前尚无前端消费方（前端 V1 仍本地聚合），改名零回归。
- **设计文档收敛**（`docs/space-statistics-capability-design.md`）：移除"快速安排下一餐"
  （随机推荐非有效行动建议）；新增 §5.5 指标分层原则（用户价值指标 vs 数据健康指标）；
  半屏内改按钮刷新、下拉刷新仅留空间页主卡片；"本周可选/周末可去"统一为"想吃/想去"；
  V1 范围收敛为"统计卡 + 单 Tab 总览 + 筛选跳转"，分组 Tab 与列表计数下沉 V1.1。

## 2026-06-29 (添加菜谱异步自动解析后端可靠性)

### Added

- **修改时间**：2026-06-29
- **变更背景**：承接添加菜谱 preview 超时转后台方案，用户本轮要求只完成后端工作。此前
  `AutoParseWorker` 已有 `pending/processing/done/failed` 队列，但缺少自动重试、退避、
  `processing` 卡死回收和占位标题安全覆盖能力，模型资源繁忙或上游超时时容易直接进入
  `failed`。
- **核心改动**：
  - 新增 `backend/migrations/022_add_recipe_auto_parse_retry.sql`，为 `recipes` 增加
    `title_source`、`parse_attempts`、`parse_next_attempt_at`、`parse_last_error_type`、
    `parse_processing_started_at`，并新增 `idx_recipes_parse_status_next_attempt`。
  - `backend/internal/recipe/auto_parse_worker.go` 支持可重试错误自动退避：`timeout`、
    `network`、`rate_limit`、`upstream` 默认最多尝试 `3` 次；`invalid_response` 最多自动
    重试 `1` 次；每轮执行前回收超过阈值的 `processing` 任务。
  - `backend/internal/recipe/repository.go` / `service.go` / `model.go` 接入新字段：
    pending 查询会跳过未到 `parse_next_attempt_at` 的任务；进入 processing 时递增
    `parse_attempts` 并记录开始时间；手动重试或换链接时重置重试状态；解析成功后清空
    retry/stale 字段。
  - 新增 `titleSource` 接口字段与 `title_source` 写库规则：`manual` 默认保护用户标题；
    `placeholder` 且模型返回标题非空时，worker 可覆盖为模型标题并置为 `parsed`；用户编辑
    标题时回到 `manual`。
  - 新增后端配置项并写入 `backend/configs/example.env` / `backend/README.md`：
    `RECIPE_AUTO_PARSE_MAX_ATTEMPTS`、`RECIPE_AUTO_PARSE_RETRY_BASE_SECONDS`、
    `RECIPE_AUTO_PARSE_STALE_SECONDS`。
  - 更新 `docs/add-recipe-async-timeout-design.md`，将后端 P1 重试、stale 回收和标题覆盖策略
    标记为已落地，并说明前端若希望猜测标题被后台解析结果覆盖，需要传
    `titleSource: 'placeholder'`。
- **影响范围**：后端菜谱创建 / 更新 / 自动解析 worker / 菜谱列表和详情响应字段 / 数据库迁移 /
  自动解析相关运行配置；不修改前端页面与小程序构建链路。
- **兼容性/风险**：
  - 存量菜谱迁移后 `title_source='manual'`，后台不会覆盖已有标题；存量 `failed` 菜谱仍需用户
    手动重试，不会因升级突然批量重跑。
  - 自动重试会增加少量后台调用，但有最大次数和退避控制；线上可通过新增环境变量调低尝试次数
    或拉长退避。
  - 前端当前若不传 `titleSource`，后端按 `manual` 保护标题；后续前端接入占位标题覆盖时需同步
    传入 `placeholder`。
- **验证情况**：
  - 已执行 `go test ./internal/recipe`，通过。
  - 已执行后端全量 `go test ./...`，通过。
  - 已用临时 SQLite 顺序执行 `backend/migrations/001` 到 `022`，验证新列和
    `idx_recipes_parse_status_next_attempt` 可创建。

## 2026-06-29 (添加菜谱 Preview 超时转后台 P0 前端实现)

### Added

- **修改时间**：2026-06-29
- **变更背景**：落地 `docs/add-recipe-async-timeout-design.md` 的 P0 前端兜底：菜谱 preview
  解析过慢（超时）时不再只弹错误，而是用原始链接 + 猜测标题先建占位菜谱，交由后端既有
  `parse_status=pending` 自动解析队列补全。仅实现前端，不动后端接口与数据库迁移。
- **核心改动**：
  - `utils/add-preview-api.js`：`previewAddLink` 超时由 `45s` 收紧到 `22s`（低于后端 `30s`，
    确保超时兜底真正触发）；新增 `isPreviewTimeoutError(error)` 归一化超时判断。
  - `pages/index/components/add-recipe-preview-panel.vue`：新增 `preview-timeout` 事件；解析
    catch 分支识别超时后 emit `preview-timeout`（带原始文案）并关闭面板，不再只弹失败；
    抽出 `stopParsing` 复用清理逻辑。
  - `pages/index/index.vue`：绑定 `@preview-timeout="handleRecipePreviewTimeoutFallback"`，
    新方法用 `extractSupportedDraftLink` + `guessDraftTitleFromShareText` 调
    `createRecipeFromDraft` 建占位菜谱（`parsedContent` 留空、`parsedContentEdited=false`、
    `titleSource=placeholder`），toast `模型繁忙，已转后台，稍后自动补全`，`15s` 后静默
    刷新一次；提取不到可支持链接时 toast `解析超时，请手动填写` 且不建占位；创建失败
    toast `后台保存失败，请稍后重试`。
  - `utils/recipe-store.js`：创建 / 更新菜谱 payload 透传 `titleSource`，让后端能识别超时占位
    标题并在后台解析成功后安全覆盖为模型标题。
  - `pages/index/recipe-card.js` / `recipe-card-item.vue` / `recipe-card-item.scss`：新增
    `buildRecipeParseBadge`，卡片标题上方信息行右侧展示「解析中」(`pending/processing`) /
    「待重试」(`failed`) 徽标，不遮挡来源与图片数量角标。
- **影响范围**：仅前端首页添加菜谱链路与菜谱卡片展示；地点添加链路不受影响（兜底只在菜谱
  入口 `add-recipe-preview-panel` 启用）。
- **兼容性/风险**：依赖 `POST /api/kitchens/{kitchenID}/recipes` 在链接支持且结构化内容为空时
  自动置 `parse_status=pending`；本次后端已支持 `title_source=placeholder` 的标题覆盖保护。
- **验证情况**：本次为前端改动，未执行小程序构建或真机验证；建议后续在微信开发者工具按
  设计文档第 10 节验收（模拟 preview 超时、无链接不误建、徽标四态）。

## 2026-06-29 (添加菜谱 Preview 超时转后台设计)

### Added

- **修改时间**：2026-06-29
- **变更背景**：用户希望优化添加菜谱体验：当小红书 / B 站菜谱 preview 因模型资源或上游
  链路超过约 20s 时，不再只让用户等待或弹错误，而是先保存菜谱，再由后台自动解析补全。
- **核心改动**：
  - 新增 `docs/add-recipe-async-timeout-design.md`，明确 P0 采用前端兜底方案：
    `previewAddLink` 超时后由前端使用原始链接和 `guessDraftTitleFromShareText` 猜测标题，
    调用 `createRecipeFromDraft` 创建菜谱，并 toast 提示“模型繁忙，已转后台，稍后自动补全”。
  - 文档拆分前后端工作：前端补 preview timeout 分支、首页“解析中”徽标和延迟刷新；
    后端 P0 复用现有 `recipes.parse_status=pending` 自动解析队列，不新增接口或迁移。
  - 补充 P1 / P2 后端可靠性演进：自动解析重试退避、`processing` stale 回收、
    `title_source` 标题覆盖保护和管理后台观测扩展。
- **影响范围**：本次仅新增设计文档和变更记录，不修改前端页面、后端接口、数据库迁移或运行配置。
- **兼容性/风险**：
  - P0 方案依赖当前 `POST /api/kitchens/{kitchenID}/recipes` 在链接支持且结构化内容为空时
    自动置为 `parse_status=pending` 的既有行为。
  - 后续实现需避免地点分享误入菜谱兜底，并处理同一链接重复创建、猜测标题不准、
    用户手动编辑被后台结果覆盖等风险。
- **验证情况**：
  - 已基于 `utils/add-preview-api.js`、`pages/index/draft-link.js`、
    `pages/index/components/add-recipe-preview-panel.vue`、
    `pages/index/components/add-link-preview-panel.vue`、`pages/index/recipe-card.js`、
    `pages/index/components/recipe-card-item.vue`、`utils/recipe-store.js`、
    `backend/internal/recipe/service.go` 与 `backend/internal/recipe/auto_parse_worker.go`
    校准设计口径。
  - 本次为文档新增，未执行前后端构建或单测。

## 2026-06-29 (修复小红书菜谱口令被误判为地点分享)

### Fixed

- **修改时间**：2026-06-29
- **变更背景**：用户反馈解析小红书分享口令后，返回的菜谱链接与菜名为空。经排查（含 SSH
  到生产服务器确认 linkparse sidecar、AI 总结、AMAP 均正常运行）定位为前端/sidecar 均无问题，
  而是 `add-link-previews` 接口的**内容分流误判**：口令结尾的来源标记 `【小红书】` 被
  `extractShareName` 当成地点名，叠加口令里的 `xhslink.com` 短链，使 `looksLikePlaceShare`
  判真，把本应走菜谱解析的小红书口令分流到了地点（place）分支，导致 `recipeDraft` 为空。
- **核心改动**：
  - `backend/internal/addpreview/share_parser.go`：新增 `sharePlatformLabels` 平台标记集合与
    `isSharePlatformLabel`，`extractShareName` 跳过 `小红书 / 哔哩哔哩 / B站 / 抖音 / 快手 / 微博`
    等来源标记括号，避免平台名被误当成地点名。
  - `backend/internal/addpreview/service.go`：菜谱解析失败分支补充 `RecipeDraft.Link =
    extractFirstURL(text)` 兜底，失败时至少回传原始链接。
  - `backend/internal/addpreview/service_test.go`：新增
    `TestXiaohongshuRecipeShareNotMisclassifiedAsPlace`，并保留 `TestParseShareText`（真实
    美团/点评地点口令）验证地点识别未受影响。
- **影响范围**：仅后端 `POST /api/kitchens/{kitchenID}/add-link-previews` 的内容类型分流；
  小红书等菜谱口令现可正确进入菜谱解析链路。需重新部署后端方在生产生效。

## 2026-06-29 (剪贴板读取改为仅用户手势触发，合规整改)

### Changed

- **修改时间**：2026-06-29
- **变更背景**：微信《用户隐私保护指引》审核反馈「读取你的剪切板接口」说明内容与使用场景
  不符。排查发现首页"手动录入"表单在打开瞬间会静默调用 `uni.getClipboardData` 自动预填
  分享链接，属于无明确用户手势的主动读取，真机会弹「正在使用你复制的内容」黄条，导致用途
  说明与实际行为不一致。
- **核心改动**：
  - 移除 `pages/index/index.vue` 中 `handleRecipeManualEntry` 打开手动表单时的静默自动
    读取（删除 `tryAutoPrefillDraftLinkFromClipboard` / `readClipboardText` 及
    `draftClipboardPrefillRequestID` 等随之失效的辅助方法与状态）。
  - 现在全程仅在用户主动点击识别面板的"粘贴链接"按钮（`add-recipe-preview-panel.vue` /
    `add-link-preview-panel.vue` 的 `handlePasteLink`）时才读取剪贴板，行为与隐私指引文案
    对齐。
  - 清理由此产生的死引用（`extractSupportedDraftLink` 等导入、`lastDraftLinkPrefill` 状态、
    永不触发的"已带入剪贴板"提示文案分支）。
- **影响范围**：仅前端首页添加菜谱流程；手动表单不再自动带入剪贴板链接，用户可手动粘贴到
  链接输入框或改用识别面板的"粘贴链接"按钮。
- **配套动作**：需在小程序后台《用户隐私保护指引》将"读取剪切板"用途说明改为紧扣"识别菜谱
  分享链接以快速导入"的场景化文案后重新提审。

## 2026-06-26 (空间统计后端能力提前落地)

### Added

- **修改时间**：2026-06-26 19:51:09 +0800 CST
- **变更背景**：空间统计前端 V1 可以先基于本地快照聚合，但用户希望提前完成后端表结构、
  数据源和接口，方便后续切换到服务端统计，同时不能影响现有菜谱、打卡点和菜单接口。
- **核心改动**：
  - 新增 `backend/internal/spacestats/` 模块，提供
    `GET /api/kitchens/{kitchenID}/stats?window=7d|30d|90d|all`，默认窗口为 `30d`。
  - 接口响应按 `overview`、`recipes`、`places`、`mealPlans`、`members`、`trends`、
    `actions` 分组返回，并复用空间成员权限校验。
  - 新增 `backend/migrations/021_add_space_stats_support.sql`，补充 `recipes.done_at`、
    `recipe_status_events`、`place_status_events`、打卡点结构化价格字段和统计索引。
  - 菜谱创建 / 编辑 / 状态切换会维护 `done_at` 和状态事件；状态切换仍不改
    `recipe.updated_at`，保持首页排序行为不变。
  - 打卡点创建 / 编辑 / 状态切换会维护状态事件，并从原始 `price` 文本提取内部
    `price_amount_cents / price_currency / price_type`，不改变现有打卡点 JSON 响应。
  - 更新 `backend/README.md` 和 `docs/space-statistics-capability-design.md`，标记后端
    stats 能力已提前落地，前端后续可选择接入。
- **影响范围**：
  - 影响后端统计接口、菜谱和打卡点写入链路、数据库迁移、后端接口文档和空间统计设计文档。
  - 不改变现有 `recipes`、`places`、`meal-plans` 业务接口响应契约；新增价格结构字段为
    后端内部统计使用。
- **兼容性/风险**：
  - 迁移会为存量菜谱和打卡点回填初始状态事件；历史事件只有迁移时点可推断的信息，
    后续趋势以新写入事件为准。
  - `price` 文本解析只提取首个数字，适配 `¥98/人`、`人均 98` 等常见格式；复杂价格描述
    会保留原文本但不进入金额统计。
  - 状态事件只在状态实际变化时新增，避免重复点击造成趋势虚高。
- **验证情况**：
  - 已执行 `go test ./internal/recipe ./internal/place ./internal/spacestats`，通过。
  - 已执行后端全量 `go test ./...`，通过。
  - 已执行 `git diff --check`，通过。
  - 已用 `sqlite3` 顺序执行 `001` 到 `021` 迁移文件，验证新增状态事件表和索引可创建。

## 2026-06-25 (空间统计落地工作评估补充)

### Changed

- **修改时间**：2026-06-25 21:44:21 +0800 CST
- **变更背景**：空间统计设计已明确产品方向，但后续实施仍需要拆清前端、后端各自要做的
  工作、V1 / V2 边界、接口演进条件和验证标准。
- **核心改动**：
  - 更新 `docs/space-statistics-capability-design.md` 的修改时间。
  - 新增“落地工作评估：前后端工作包”章节，明确 V1 先以前端本地聚合闭环，
    V2 再新增后端统计接口。
  - 拆分 V1 前端必须完成的工作包：`utils/space-stats.js`、空间统计卡、
    空间洞察半屏、首页接线、数据刷新、筛选跳转、快速安排和局部计数。
  - 补充 V1 后端配合项、V2 后端 `spacestats` 模块、V2 前端数据源切换、
    建议实施顺序和暂缓项。
- **影响范围**：
  - 本次仅修改设计文档和变更记录，不修改前端页面、后端接口、数据库迁移或运行配置。
- **兼容性/风险**：
  - 文档明确 V1 不新增接口，避免为了当前快照统计提前引入后端聚合复杂度。
  - V2 接口、状态事件表、索引优化仍为后续演进项，真正实现时需单独补接口契约和测试。
- **验证情况**：
  - 已基于现有 `pages/index/index.vue`、`kitchen-section.vue`、`utils/recipe-store.js`、
    `utils/place-store.js`、`utils/meal-plan-api.js`、`backend/internal/app/router.go`
    和 `backend/internal/mealplan` 结构校准落地工作包。
  - 本次为文档更新，未执行前后端构建或单测。

## 2026-06-25 (打卡点增强字段前端接入)

### Added

- **修改时间**：2026-06-25 21:38:26 +0800 CST
- **变更背景**：后端已提供打卡点增强字段、状态体验采集和高德候选预填能力，前端需要把
  电话、重访意愿、推荐项、更多信息和状态切换体验采集接入到完整用户链路。
- **核心改动**：
  - 新增 `pages/index/components/place-experience-sheet.vue`，在打卡点从“想去”切换为
    “去过”时采集重访意愿、推荐项和打卡日期，并保留“暂时跳过”路径。
  - 扩展 `pages/index/components/place-edit-sheet.vue`，新增电话、去过状态下的体验记录、
    更多信息折叠区，以及就餐提示、适合场景、最佳时段、时长、陪同人推荐、停车 /
    交通等字段输入。
  - 优化 `pages/index/components/place-detail-sheet.vue`，展示多图轮播、电话快捷拨打 /
    复制、重访意愿、推荐项、外部评分、场景标签、出行信息和外部数据来源。
  - 优化 `pages/index/components/place-card-item.vue`，去过卡片展示重访星级、推荐项和
    场景标签，并对高分 / 低分体验做轻量视觉区分。
  - 扩展 `utils/place-store.js` 和 `utils/place-api.js`，归一化新增字段并在状态更新接口中
    携带体验数据。
- **影响范围**：
  - 影响首页打卡点列表、打卡点新增 / 编辑弹层、详情弹层、想去 / 去过状态切换和本地缓存
    字段归一化。
  - 依赖后端已落地的打卡点增强字段接口；不新增后端 API、数据库迁移或第三方 Key 配置。
- **兼容性/风险**：
  - 旧打卡点没有新增字段时会按空值或 `0` 兜底，不强制用户补充。
  - 切回“想去”仍沿用后端清空体验字段策略，前端当前直接执行，后续如用户反馈可加二次确认。
  - 电话拨打依赖微信 `makePhoneCall` 能力，真机需要确认授权 / 失败提示体验。
- **验证情况**：
  - 待执行 SFC 解析检查覆盖 `place-card-item.vue`、`place-detail-sheet.vue`、
    `place-edit-sheet.vue`、`place-experience-sheet.vue` 和 `pages/index/index.vue`。
  - 未运行 HBuilderX 或微信开发者工具真机预览。

## 2026-06-25 (微信位置隐私声明口径优化)

### Changed

- **修改时间**：2026-06-25 20:43:44 +0800 CST
- **变更背景**：微信小程序“奇思妙想的小海”用户隐私保护指引审核提示
  “【收集你的位置信息接口】说明内容不符合接口使用场景”，需要让代码侧授权弹窗文案
  和后台隐私指引用途聚焦到真实的打卡点选点场景。
- **核心改动**：
  - 将 `manifest.json` 中 `scope.userLocation.desc` 从“用于选择和打开打卡点位置”
    调整为“用于选择并保存打卡点位置”。
  - 在 `manifest.json` 的 `mp-weixin` 配置中补充 `requiredPrivateInfos: ["chooseLocation"]`，
    明确当前使用的是微信地图选点接口。
- **影响范围**：
  - 仅影响微信小程序位置授权弹窗说明和小程序地理位置隐私接口声明。
  - 不修改打卡点新增 / 编辑 / 详情页逻辑，也不新增持续定位、后台定位或附近推荐能力。
- **兼容性/风险**：
  - 需在微信公众平台隐私保护指引中同步使用一致口径：
    “用于用户在新增或编辑打卡点时，主动选择餐厅或地点位置，并保存该打卡点的地址和经纬度，
    便于后续查看和打开地图导航。”
  - 若后续新增 `getLocation`、持续定位或附近推荐能力，需要重新补充接口声明和隐私用途说明。
- **验证情况**：
  - 已检查当前代码仅在打卡点编辑链路调用 `uni.chooseLocation`，详情 / 列表仅基于已保存经纬度
    调用 `uni.openLocation`。
  - 本次为配置和文案调整，未运行 HBuilderX 或微信开发者工具真机预览。

## 2026-06-25 (打卡点列表卡片比例优化)

### Changed

- **修改时间**：2026-06-25 20:08:44 +0800 CST
- **变更背景**：打卡点列表中店铺图片使用固定小正方形展示，遇到较高卡片或长店名时，
  图片与卡片比例不协调，视觉重心偏散。
- **核心改动**：
  - 调整 `pages/index/components/place-card-item.vue` 的卡片内边距、圆角和最小高度。
  - 将打卡点封面从固定 `176rpx x 176rpx` 正方形优化为更宽、可随卡片高度拉伸的
    封面区域，并设置合理的最小 / 最大高度。
  - 微调店名、地址和底部操作按钮尺寸，让文字列与图片列比例更均衡。
  - 将地图操作按钮从底部 footer 流式占位改为右下角绝对定位；无标签时不再保留空的
    底部标签行。
- **影响范围**：
  - 仅影响首页打卡点模式的列表卡片视觉展示；不修改打卡点数据、接口、筛选、
    详情页或保存逻辑。
- **兼容性/风险**：
  - 继续使用 `image mode="aspectFill"`，不同来源的横图 / 竖图仍会被裁切到封面容器内；
    后续可在微信开发者工具里结合真实图片进一步微调宽度。
- **验证情况**：
  - 已使用临时 `@vue/compiler-sfc` 对 `place-card-item.vue` 完成 SFC 解析检查，通过。
  - 已执行 `git diff --check -- pages/index/components/place-card-item.vue`，通过。

## 2026-06-25 (空间统计能力设计)

### Added

- **修改时间**：2026-06-25 13:16:40 +0800 CST
- **变更背景**：需要评估当前美食库、打卡库和空间页的信息架构，明确后续统计能力
  是否应以空间为主维度，以及入口、展示指标和前后端实现路径。
- **核心改动**：
  - 新增 `docs/space-statistics-capability-design.md`，提出“空间概览 / 空间洞察”
    作为统计能力定位。
  - 明确主入口建议放在底部 `空间` 页当前空间卡片下方，不新增底部第四个统计 Tab；
    美食库和打卡点列表页只保留轻量计数。
  - 梳理 V1 可基于前端已同步的 `recipes`、`places`、`mealOrderStore`、
    `kitchenMembers` 本地聚合，不新增后端接口；V2 再补
    `GET /api/kitchens/{kitchenID}/stats?window=30d`。
  - 定义空间总览、美食库、打卡库、菜单安排四类指标，以及数据口径、风险、分期计划
    和验收标准。
- **影响范围**：
  - 本次仅新增设计文档和变更记录，不修改小程序页面、后端接口、数据库迁移或运行配置。
- **兼容性/风险**：
  - 设计文档指出 V1 是当前快照统计，菜谱缺少 `doneAt` 或状态历史，暂不适合做
    “吃过趋势”；`price` 当前为文本，暂不做消费金额统计。
- **验证情况**：
  - 已基于现有 `pages/index/index.vue`、`kitchen-section.vue`、`utils/recipe-store.js`、
    `utils/place-store.js`、`utils/meal-plan-api.js` 和后端空间路由完成口径校准。
  - 本次为设计文档新增，未执行前后端构建或单测。

## 2026-06-25 (智能添加剪贴板读取兜底)

### Fixed

- **修改时间**：2026-06-25 10:48:34 +0800 CST
- **变更背景**：用户点击智能添加入口的“点此粘贴分享链接”时，即使系统剪贴板已有分享
  文案，前端仍提示“请使用下方输入框粘贴分享内容”，说明智能添加面板没有稳定读到
  `uni.getClipboardData` 的返回值。
- **核心改动**：
  - 将 `add-link-preview-panel.vue` 和 `add-recipe-preview-panel.vue` 的剪贴板读取从
    `await uni.getClipboardData()` 改为 callback 包装的 `success/fail` 形式，和项目内
    现有自动预填逻辑保持一致。
  - 读取失败时保留 `console.warn` 输出，便于在微信开发者工具中区分 API 失败、权限 /
    隐私配置失败和剪贴板为空。
  - 将兜底提示调整为“未读取到剪贴板，请粘贴到输入框”，避免误导用户以为必须手动粘贴。
- **影响范围**：
  - 影响新增打卡点和新增菜品的智能识别入口；不影响手动填写、后端预览接口或保存接口。
- **兼容性/风险**：
  - 如果微信小程序后台未在用户隐私保护指引中声明剪贴板读取，或运行环境禁止读取剪贴板，
    前端仍会进入输入框兜底，但现在控制台会保留失败原因。
- **验证情况**：
  - 已使用 `@vue/compiler-sfc` 对 `add-link-preview-panel.vue` 和
    `add-recipe-preview-panel.vue` 完成 SFC 解析检查，通过。
  - 已执行 `git diff --check -- pages/index/components/add-link-preview-panel.vue pages/index/components/add-recipe-preview-panel.vue`，通过。

## 2026-06-25 (打卡点增强字段后端接口)

### Added

- **修改时间**：2026-06-25 09:57:51 +0800 CST
- **变更背景**：打卡点编辑字段增强设计已经明确高优先级和中优先级字段，后端需要先提供
  稳定的落库、回显、状态体验采集和高德候选预填能力，前端后续可逐步接入。
- **核心改动**：
  - 新增 `backend/migrations/020_add_place_enhancement_fields.sql`，为 `places` 表补充
    `phone`、`revisit_rating`、`recommended_items_json`、`external_provider`、
    `external_poi_id`、`rating`、`dining_tips`、`scenes_json`、`best_time`、`duration`、
    `companion_tags_json`、`parking_note` 等字段，并增加外部 POI 与重访评分索引。
  - 扩展 `Place` 响应、创建 / 更新请求、仓储读写和搜索字段；普通更新改为合并语义，
    未传入的新字段不会被旧前端误清空。
  - 扩展状态更新接口，支持切换为 `visited` 时同时保存 `visitedAt`、`revisitRating`
    和 `recommendedItems`；切回 `want` 时清空打卡体验字段。
  - 扩展高德 POI 候选解析与排序输出，候选 `placeDraft` 现在可携带电话、评分、
    `externalProvider=amap`、`externalPoiId`、商圈辅助地址和推断标签。
  - 基础分享解析草稿保留提取到的电话，后续前端选择候选或手动补全时不会丢失。
- **影响范围**：
  - 影响打卡点创建、编辑、列表、详情、状态切换和统一智能添加预览接口的后端响应。
  - 本次不修改前端页面；现有前端即使暂未展示新增字段，也可通过后端合并更新避免已保存
    字段被覆盖为空。
- **兼容性/风险**：
  - 新增数据库字段均带默认值，兼容旧数据。
  - `source` 继续表示分享来源；高德来源仅写入 `externalProvider=amap`，避免混淆来源
    语义。
  - 推荐项和重访评分属于已打卡体验字段，地点切回 `want` 时会清空，需要前端在交互上
    做确认或跳过提示。
- **验证情况**：
  - 已执行 `go test ./internal/place ./internal/addpreview`，通过。
  - 已执行 `go test ./...`，通过。

## 2026-06-24 (打卡点编辑字段增强设计)

### Added

- **修改时间**：2026-06-25 09:28:28 +0800 CST
- **变更背景**：高德 POI 已能补全电话、评分、人均、图片和外部 POI ID，现有打卡点
  编辑表单仍主要保存基础字段，需要明确哪些新增信息值得进入编辑页，以及哪些应作为
  详情页或系统追踪信息，避免表单无序膨胀。
- **核心改动**：
  - 新增 `docs/place-edit-fields-enhancement-design.md`，提出打卡点字段增强方案。
  - 将新增信息分为高优先级字段和中优先级字段：电话、推荐菜 / 想点菜、到访日期、
    外部 POI 标识优先进入 V1；外部评分、营业时间、排队 / 预约、适合场景、停车 /
    交通进入 `更多信息` 折叠区。
  - 明确编辑页、详情页和列表卡片各自应展示的信息密度，以及外部 Provider / POI ID
    默认作为隐藏系统字段，不让用户直接编辑。
  - 补充后端 `places` 表、API 请求响应、智能添加候选预填、兼容性风险和 V1 / V1.1 /
    V2 分期建议。
  - **追加修正**（2026-06-25 09:28:28 +0800 CST）：补充前端与后端分别需要完成的
    实施清单、当前仓库涉及文件、前后端联调依赖和验收矩阵；修正接口口径为继续复用
    `add-link-previews` 统一预览接口，不新增独立高德搜索接口；明确 `source` 继续表示
    分享来源，高德 POI 来源写入 `externalProvider=amap`。
- **影响范围**：
  - 本次仅新增设计文档和变更记录，不修改前端页面、后端接口、数据库迁移或高德调用
    逻辑。
- **兼容性/风险**：
  - 文档建议后续新增字段均使用默认值兼容旧数据；真正实现时需要同步更新前端
    normalize、后端 Place 模型、迁移脚本和详情展示。
  - 中优先级字段必须折叠展示，否则会拉长新增 / 编辑流程，降低用户保存意愿。
- **验证情况**：
  - 已基于现有 `place-edit-sheet.vue`、`utils/place-store.js`、`backend/internal/place`
    模型、`backend/internal/addpreview` 候选链路和 `places` 迁移结构校准字段命名、
    展示位置与前后端实施边界。
  - 本次为设计文档新增，未执行前后端构建或单测。

## 2026-06-24 (打卡点详情图片轮播)

### Changed

- **修改时间**：2026-06-24 15:07:58 +0800 CST
- **变更背景**：打卡点保存链路已支持多张图片，但详情页顶部只展示首图，用户无法在详情
  页面内直接感知还有更多图片。
- **核心改动**：
  - 将打卡点详情页顶部 hero 图从单张 `image` 调整为 `swiper`，多图时自动轮播并循环
    切换。
  - 多图时展示页码和圆点进度；点击当前图片仍复用原有 `uni.previewImage` 多图预览。
  - 单图和无图场景保持原有展示逻辑；列表卡片仍只展示首图，避免列表信息密度变化。
- **影响范围**：
  - 仅影响 `pages/index/components/place-detail-sheet.vue` 的打卡点详情展示交互。
- **兼容性/风险**：
  - 微信小程序 `swiper` 支持自动轮播；需要在微信开发者工具或真机确认轮播手势和点击
    预览体验。
- **验证情况**：
  - 已使用 `@vue/compiler-sfc` 对 `place-detail-sheet.vue` 完成 SFC 解析检查，通过。
  - 已执行 `git diff --check`，通过。

## 2026-06-24 (打卡点候选图片保存前镜像)

### Fixed

- **修改时间**：2026-06-24 12:26:03 +0800 CST
- **变更背景**：真实高德 POI 候选会返回图片 URL，用户从候选创建打卡点后数据库中
  虽然保存了 `image_urls_json`，但图片多为高德外链，包含 `http://store.is.autonavi.com`
  和 `https://aos-comment.amap.com` 等域名，微信小程序线上环境可能因域名白名单或
  HTTP 限制无法展示，看起来像“添加成功但没有图片”。
- **核心改动**：
  - `place.Service` 注入现有 `upload.Service`，在创建或更新打卡点前，将外部远程图片
    通过 `SaveRemoteImage` 镜像到自有 `/uploads/` 体系，再把自有图片 URL 入库。
  - 远程图片镜像失败时跳过失败图片，不阻塞打卡点保存。
  - `upload.Service.IsManagedImageURL` 增加对线上 nginx 前缀 `/caipu-uploads/` 的识别，
    避免编辑已有自有图片时重复镜像。
  - 增加单测覆盖远程图片保存前镜像，以及 `/caipu-uploads/` 自有图片识别。
- **影响范围**：
  - 影响打卡点创建和更新时的图片保存链路；菜谱图片镜像、上传接口和数据库结构不变。
- **兼容性/风险**：
  - 第三方图片下载失败时会跳过该图片，因此极端情况下仍可能无图，但不会导致保存失败。
  - 保存耗时会随远程图片下载增加；当前候选图数量最多 6 张，后续可按需要进一步限制为
    前 3 张。
- **验证情况**：
  - 已执行 `go test ./internal/place ./internal/upload`，通过。
  - 已执行 `go test ./...`，通过。

## 2026-06-24 (分享链接智能添加真实接口联调)

### Added

- **修改时间**：2026-06-24 11:33:07 +0800 CST
- **变更背景**：分享链接智能添加前端 POC 已验证交互，需要实现后端统一预览接口，
  并将前端从 Mock 数据改为调用真实接口，支持美团 / 大众点评打卡地解析和小红书 /
  B站菜谱解析分流。
- **核心改动**：
  - 新增 `backend/internal/addpreview/` 服务包，提供
    `POST /api/kitchens/{kitchenID}/add-link-previews`，在鉴权和空间成员校验后识别
    `recipe` / `place` 内容类型，并返回 `recipe_result`、`place_candidates`、
    `partial` 或 `failed`。
  - 打卡地分支支持从分享文案提取名称、地址、电话、来源链接和美团 `poiId` 信息；
    高德 POI 查询启用时，会按店名、清洗店名、店名 + 地址、电话、地址多关键词查询，
    并基于名称、地址、道路 / 商圈 / 地标、类目、图片和评分做可解释排序。
  - 菜谱分支复用现有 `linkparse.Service.ParseRecipeLink`，不修改旧
    `link-parsers/preview`、`link-parsers/bilibili`、`link-parsers/xiaohongshu`
    接口契约。
  - 新增高德预览配置项：`AMAP_PLACE_PREVIEW_ENABLED`、`AMAP_WEB_SERVICE_KEY`、
    `AMAP_PLACE_PREVIEW_DEFAULT_CITY`、`AMAP_PLACE_PREVIEW_TIMEOUT_SECONDS`、
    `AMAP_PLACE_PREVIEW_MAX_ATTEMPTS`、`AMAP_PLACE_PREVIEW_QPS_DELAY_MS`；示例配置只保留
    占位，不写入真实 Key。
  - 新增 `utils/add-preview-api.js`，将打卡点与菜品智能添加面板都改为调用真实
    `add-link-previews` 接口；移除固定 Mock 候选和固定菜谱数据。
  - 更新 `pages/index/index.vue`，兼容后端 `partial` 打卡地结果、候选 `placeDraft`
    图片字段、菜谱 `imageUrls` 和结构化 `parsedContent` 到现有表单字段的转换。
  - 更新 `docs/place-link-preview-poc-test.md` 与 `backend/README.md`，同步当前真实
    接口联调口径、配置项和接口清单。
- **影响范围**：
  - 影响首页新增菜品 / 新增打卡点的智能识别入口；编辑已有记录、旧菜谱链接解析接口、
    菜品保存接口、打卡点保存接口和数据库结构保持不变。
  - 高德 Key 仅通过后端环境变量配置，不下发前端；未新增数据库迁移。
- **兼容性/风险**：
  - 高德未启用或查询失败时，新接口返回 `partial`，前端会用基础字段进入原手动表单。
  - 候选图片仍为高德外链，保存前暂未镜像到 `/uploads/`；后续需补图片镜像和微信域名
    白名单策略。
  - 高德 POI 仍可能返回同商圈或同品牌异地门店，因此前端继续展示候选让用户确认，不会
    自动创建菜品或打卡点。
- **验证情况**：
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check scripts/probe-meituan-place-link.mjs`，通过。
  - 已使用临时 `/tmp/caipu-miniapp-sfc-check` 依赖中的 `@vue/compiler-sfc` 对首页和
    3 个智能添加组件完成 SFC 解析检查，通过。
  - 已执行 `git diff --check`，通过。
  - 已启动临时本地后端并用临时环境变量配置高德 WebService Key，调用
    `POST /api/kitchens/1/add-link-previews` 复测样例美团文案，返回
    `place_candidates`，首个候选为 `旺记碳烤肥牛(多丰喜市园区北滘店)`，包含评分、
    人均和 3 张图片。

## 2026-06-24 (分享链接智能添加前端 POC)

### Added

- **修改时间**：2026-06-24 10:38:53 +0800 CST
- **变更背景**：在分享链接智能添加产品设计确认后，需要先落地一个纯前端 POC，
  验证新增入口、智能解析中状态、菜谱 / 打卡地分流、地点候选确认和表单预填
  是否符合预期。
- **核心改动**：
  - 新增 `pages/index/components/add-link-preview-panel.vue`，作为打卡点新增的智能
    识别第一屏，支持读取剪贴板、手动粘贴分享文案、展示解析中阶段文案，并用
    Mock 数据返回打卡地候选或菜谱结果。
  - 新增 `pages/index/components/add-recipe-preview-panel.vue`，作为菜品新增的智能
    识别第一屏，支持小红书 / B站链接 Mock 解析，并在成功后预填现有添加菜品表单。
  - 新增 `pages/index/components/place-candidate-sheet-v2.vue`，展示 1 到 3 个可能的
    地点候选，包含图片、名称、地址、评分、人均和匹配理由，并支持选择候选或
    回退手动填写。
  - 更新 `pages/index/index.vue`，将底部新增打卡点入口切到智能识别面板；将新增
    菜品入口切到菜谱智能识别面板；识别到打卡地时先展示候选，选择后预填现有
    打卡点表单；识别到菜谱时复用现有添加菜品表单预填逻辑。
  - 新增 `docs/place-link-preview-poc-test.md`，记录 POC 功能范围、Mock 测试文案、
    打卡点与菜品入口测试步骤、下一步真实接口接入项和已知限制。
- **影响范围**：
  - 影响首页底部新增菜品 / 新增打卡点的新增入口交互；编辑已有菜品或打卡点不受
    影响。
  - 当前仍为纯前端 Mock，不新增后端 API、数据库迁移、配置项或第三方接口调用。
- **兼容性/风险**：
  - POC 会改变新增入口第一屏，但保留手动填写兜底；用户仍需在原表单中确认保存，
    不会自动创建菜品或打卡点。
  - 候选图片使用占位图 URL；真实接入后仍需处理图片镜像、微信域名白名单、接口
    超时和错误兜底。
- **验证情况**：
  - 已执行 `node --check scripts/probe-meituan-place-link.mjs`，通过。
  - 已使用临时 `/tmp/caipu-miniapp-sfc-check` 依赖中的 `@vue/compiler-sfc` 对
    `pages/index/index.vue`、`add-link-preview-panel.vue`、
    `add-recipe-preview-panel.vue`、`place-candidate-sheet-v2.vue` 完成 SFC 解析检查，
    通过。
  - 已执行 `git diff --check`，通过。
  - 未启动 HBuilderX / 微信开发者工具做真机联调；提交前仍建议在微信开发者工具中
    按 `docs/place-link-preview-poc-test.md` 验证完整交互。

## 2026-06-23 (分享链接智能添加产品设计)

### Added

- **修改时间**：2026-06-24 09:17:12 +0800 CST
- **变更背景**：在美团 / 大众点评分享文案和高德 POI 探测可行性验证后，需要形成
  可进入排期评审的产品设计文档，明确用户交互、后端接口、数据策略和风险边界。
- **核心改动**：
  - 新增 `docs/place-link-preview-product-design.md`，定义 V1 方案为“从分享链接添加”
    的智能填充能力：用户粘贴分享文案后，先展示智能解析中状态，再根据内容类型分流
    到菜谱或打卡地结果；用户确认后仅预填表单，不自动创建菜品或打卡点。
  - 文档建议新增 `POST /caipu-api/kitchens/{kitchenID}/add-link-previews` 统一智能添加
    预览接口，返回 `contentType`、`status`、`recipeDraft`、`placeDraft`、
    `candidates`、`warnings` 等字段；保存仍复用现有菜品保存接口和
    `POST /caipu-api/kitchens/{kitchenID}/places`。
  - 文档明确不直接调整现有 `link-parsers/preview`、`link-parsers/bilibili`、
    `link-parsers/xiaohongshu` 菜谱解析接口契约；新增 `addpreview` 服务分层作为统一
    分流层，内部复用现有菜谱解析服务，打卡地分支再调用高德 POI。
  - 文档明确高德 Key 仅存后端配置，不下发小程序端；打卡地候选排序需综合名称、
    地址、类目、电话、分店 / 商圈词、门牌、图片和评分，并返回匹配理由。
  - 文档给出 V1 / V1.1 / V2 分期：V1 不改数据库，V1.1 做图片镜像和限流缓存，V2
    再考虑保存外部 POI ID、评分和电话等增强字段。
  - **追加修正**（2026-06-24 09:10:58 +0800 CST）：根据推荐的添加页面设计，补充
    新增打卡点第一屏布局：顶部主入口 `点击粘贴分享链接`，中部展示 `打卡地`
    和 `菜谱灵感` 两类可解析内容及对应平台，底部保留 `手动填写信息` 入口。
  - **追加修正**（2026-06-24 09:13:49 +0800 CST）：补充点击主入口后的智能解析中
    状态与结果分流：先展示 `智能解析中`、加载动画和“正在提取地点 / 菜谱信息”等
    阶段文案；若解析为菜谱则进入现有添加菜品界面并预填菜谱名等字段，若解析为
    打卡地则展示推测地点候选，用户选择后再预填打卡点表单。
  - **追加修正**（2026-06-24 09:17:12 +0800 CST）：评估菜谱解析接口影响后，将后端
    方案调整为新增统一 `add-link-previews` 智能添加预览接口，不破坏现有菜谱解析
    接口；新接口负责内容类型识别，并在菜谱分支复用现有解析服务。
- **影响范围**：
  - 仅新增和更新产品 / 技术设计文档，不修改前端页面、后端接口、数据库迁移或部署
    配置。
- **兼容性/风险**：
  - 文档明确 V1 不做爬虫、不绕过第三方风控、不自动落库；高德 POI 仍可能返回同商圈
    或同品牌异地门店，正式交互必须让用户确认候选。
  - 高德图片外链存在 HTTP、域名白名单和可用性风险，文档建议后续通过后端镜像到
    现有 `/uploads/` 体系解决。
- **验证情况**：
  - 已结合现有 `places` CRUD、`PlaceEditSheet`、`utils/place-store.js` 和高德探测
    脚本输出校准字段映射。
  - 本次为文档新增，未执行前后端构建；后续实现时需补充 Go 单测、前端编译检查和
    微信开发者工具联调验证。

## 2026-06-20 (美团打卡点链接图片探测脚本)

### Added

- **修改时间**：2026-06-23 22:24:53 +0800 CST
- **变更背景**：需要评估美团 / 大众点评分享文案是否能自动解析出打卡点名称、
  地址、电话、门店图片、评分、人均等信息，并参考开源美团 / 点评爬虫项目先做
  最小脚本验证。
- **核心改动**：
  - 新增 `scripts/probe-meituan-place-link.mjs`，支持从命令行参数、STDIN 或内置
    示例文案提取打卡点名称、地址、电话、来源链接。
  - 脚本会展开 `dpurl.cn` 短链，解析美团 App 唤起链接里的 `poiId` 与
    `poiIdEncrypt`，并尝试探测唤起页、移动店铺页、PC 店铺页和旧 PC API 页面中
    的 `og:image`、图片 URL 与 JSON 字段。
  - 输出中区分 `imageURLs` 与过滤后的 `usableImageURLs`，避免把美团 App 唤起页、
    官网合作页、二维码或模板变量图片误判为店铺宣传图。
  - **追加修正**（2026-06-20 16:46:47 +0800 CST）：脚本新增高德 WebService POI
    探测参数，支持通过 `--amap-key` 或 `AMAP_WEB_SERVICE_KEY` 传入 Key，并用
    `--amap-city` / `AMAP_CITY` 指定城市；高德返回的 `biz_ext.rating`、
    `biz_ext.cost` 与 `photos` 会汇总进输出。
  - **追加修正**（2026-06-23 22:24:53 +0800 CST）：高德探测改为多关键词尝试并
    对候选 POI 做可解释排序，综合名称、地址、类目、电话、分店 / 商圈词、门牌、
    图片和评分打分；输出新增 `amap.rankedPois`、`matchScore`、`matchReasons`，
    避免直接把第一条商场、停车场或同品牌异地门店当作最佳店铺。
- **影响范围**：
  - 仅新增独立调研脚本，不接入前端表单、后端接口、数据库结构或正式
    `linkparse-sidecar`。
- **兼容性/风险**：
  - 当前脚本不绕过验证码、不使用 cookie 池、不接代理；探测结果显示样例短链能
    稳定解析 `poiId=1280169388`，但真实移动店铺页会进入美团 `spiderindefence`
    身份核实页，旧 PC 美食接口会重定向到官网合作页。
  - 因此无登录态 / 官方接口 / 地图 POI 兜底时，不能稳定获取真实店铺宣传图、
    美团评分或人均价；后续正式能力建议优先采用“复制文案提取 + 地图 POI 补全”，
    美团字段仅作为实验性增强。
  - 高德 POI 可以补全评分、人均、经纬度和图片，但搜索结果仍可能出现商场、
    停车场、同品牌异地门店或同商圈相邻餐厅；正式接入时应返回 Top 候选让用户
    确认，不宜完全静默自动落库。
- **验证情况**：
  - 已执行 `node --check scripts/probe-meituan-place-link.mjs`，通过。
  - 已执行 `node scripts/probe-meituan-place-link.mjs`，样例文案可提取店名、地址、
    电话、短链、`poiId` 与 `poiIdEncrypt`；`usableImageURLs` 为空，确认未命中真实
    店铺图片。
  - 已确认当前仓库和环境未配置 `AMAP_WEB_SERVICE_KEY` / `AMAP_KEY`，不带 Key 运行时
    脚本会在 `amap.reason` 中提示缺少高德 WebService Key。
  - 已用临时环境变量传入高德 WebService Key 复测样例，不将 Key 写入仓库；脚本
    命中高德 POI `B0JUN7FVJK`：`旺记碳烤肥牛(多丰喜市园区北滘店)`，返回评分
    `4.7`、人均 `79.00`、地址 `人昌路2号(华美达广场旁)` 与 3 张图片 URL。
  - 已对上述 3 张高德图片 URL 执行 HEAD 检查，均返回 `200` 与 `image/jpeg`；
    已执行 `rg` 确认本次临时 Key 未写入仓库文件。

## 2026-06-16 (首页首次进入延后授权登录)

### Changed

- **修改时间**：2026-06-16 01:30:14 +0800 CST
- **变更背景**：微信小程序审核反馈指出，用户一进入首页、尚未浏览体验功能服务时，
  即被要求授权手机号码、头像、昵称进行授权登录；需要调整为用户先浏览功能服务，
  后续再自行选择授权登录。
- **核心改动**：
  - 移除 `App.vue` 启动阶段的 `ensureSession()`，避免小程序 `onLaunch` 立即
    创建登录会话。
  - 首页 `refreshRecipes()` 在本地无 token 时仅展示已有缓存/游客态，不再自动
    调用登录接口；已有 token 的老用户仍按原逻辑刷新空间、菜谱、打卡点和成员数据。
  - 移除登录链路中的 `uni.getUserInfo` 自动头像昵称读取与自动资料同步；微信登录
    仅提交 `wx.login` code 和 `appId`。
  - 移除首页资料不完整时自动弹出“完善资料”弹层；头像昵称授权入口仅保留在用户
    主动打开“修改资料”后触发。
- **影响范围**：
  - 影响首页首次进入、登录会话创建时机、头像昵称资料补全时机。
  - 不新增手机号授权入口；本次代码搜索确认当前前端无 `getPhoneNumber` 调用。
  - 不修改后端登录接口、用户资料保存接口、菜谱/打卡点/空间数据结构。
- **兼容性/风险**：
  - 未登录新用户首次进入首页会处于可浏览/游客态；点击新增菜品、AI 助手、打卡点、
    空间邀请等需要身份的操作时，仍会按现有业务链路触发登录。
  - 已有 token 用户仍会自动刷新数据，历史使用体验基本不变。
  - 由于未启动微信开发者工具，本次未做真机授权弹窗复测，提交审核前建议用清空
    缓存的新微信号验证首次打开首页不出现头像、昵称、手机号授权弹窗。
- **验证情况**：
  - 已用代码搜索确认 `App.vue`、`pages/`、`components/`、`utils/` 中无
    `getPhoneNumber`、`getUserInfo`、`getUserProfile`、`maybePromptProfile`
    与 `hasDismissedProfilePrompt` 残留；仅保留用户主动资料弹层中的
    `open-type="chooseAvatar"`。
  - 已使用临时安装的 `@vue/compiler-sfc` 完成 `App.vue`、
    `pages/index/index.vue` 与 `pages/index/components/profile-sheet.vue` 的
    模板/脚本编译检查，通过。

## 2026-06-16 (底部主入口隐藏文字标签)

### Changed

- **修改时间**：2026-06-16 00:55:02 +0800 CST
- **变更背景**：用户确认底部中间主按钮不需要额外显示“添加菜品 / 添加打卡点”
  等文字，避免底部导航显得拥挤和重复。
- **核心改动**：
  - 移除首页底部中间主按钮下方的动态文字标签。
  - 保留原有点击分流：打卡点模式仍打开“新增打卡点”，美食库模式仍按后台开关
    进入 AI 助手或添加菜品。
- **影响范围**：
  - 仅影响首页底部导航中间主按钮的可见文字，不修改按钮图标、点击逻辑或后端
    配置。
- **兼容性/风险**：
  - 低风险；文字移除后需依赖图标和用户习惯识别主动作。
- **验证情况**：
  - 已确认 `pages/index/index.vue` 中无 `primaryFabLabel` 与 `nav-center__label`
    残留。
  - 已使用 `@vue/compiler-sfc` 完成 `pages/index/index.vue` 的模板/脚本编译检查，
    通过。
  - 已执行 `git diff --check`，通过。

## 2026-06-16 (美食库菜单卡片优化与过期隐藏)

### Changed

- **修改时间**：2026-06-16 00:52:10 +0800 CST
- **变更背景**：首页美食库中的菜单提示卡片仍偏提示条样式，且当菜单日期已早于
  当前日期时，过去菜单仍会作为 fallback 出现在首页 spotlight 中，容易让用户
  误以为还有待查看的当前菜单。
- **核心改动**：
  - 优化美食库菜单 spotlight 卡片为“日期块 + 菜单标题 + 状态/数量 + 摘要 +
    进入按钮”的宽松卡片结构，强化未来菜单预告感。
  - `mealOrderSpotlightRecords` 仅保留 `planDate >= 今天` 的菜单草稿或已安排记录；
    过期菜单不再在首页显式展示，但不删除数据，也不影响菜单详情页历史查看。
- **影响范围**：
  - 影响首页“清单 -> 美食库”模式下的菜单 spotlight 展示和视觉样式；不修改
    菜单保存、提交、详情页路由或后端接口。
- **兼容性/风险**：
  - 日期比较继续使用项目已有本地日期 `toISODate(new Date())`，避免 UTC 跨天
    误差；如果空间中只有过期菜单，首页将不再显示菜单卡片。
  - 视觉改动集中在 `library-header-section`，建议后续在微信开发者工具中检查
    375px 宽度和多菜单轮播时的真实效果。
- **验证情况**：
  - 已使用 `@vue/compiler-sfc` 完成 `pages/index/index.vue` 与
    `pages/index/components/library-header-section.vue` 的模板/脚本编译检查，通过。
  - 已执行 `git diff --check`，通过。

## 2026-06-16 (底部主入口适配打卡点模式)

### Changed

- **修改时间**：2026-06-16 00:43:40 +0800 CST
- **变更背景**：首页切到“打卡点”模式后，底部中间主入口仍容易表现为添加菜品
  或 AI 助手入口，和当前模式下的“添加打卡点”预期不一致。
- **核心改动**：
  - 新增底部主入口动态文案：打卡点模式显示“添加打卡点”，美食库模式按后台
    开关显示“AI 助手”或“添加菜品”。
  - 调整底部主入口点击逻辑：当 `activeSection === 'library'` 且
    `appMode === 'explore'` 时，优先打开“添加打卡点”弹层，不再进入菜品/AI
    分流。
- **影响范围**：
  - 影响首页底部中间主按钮的文案和点击行为；不修改打卡点表单字段、后端接口
    或菜品新增弹层本身。
- **兼容性/风险**：
  - 低风险；仅在清单页打卡点模式覆盖原有主按钮行为，其他页面/模式保持原分流。
- **验证情况**：
  - 已使用 `@vue/compiler-sfc` 完成 `pages/index/index.vue` 的模板/脚本编译检查，
    通过。
  - 已执行 `git diff --check`，通过。

## 2026-06-16 (打卡点列表卡片质感升级)

### Changed

- **修改时间**：2026-06-16 00:35:37 +0800 CST
- **变更背景**：用户提供打卡点列表参考图，要求每个卡片更宽敞、更有设计感，
  并要求“已打卡”状态具备独立特殊样式；参考 `我们的美食空间/` 中的
  `PlaceCard` 设计。
- **核心改动**：
  - 放大打卡点卡片留白、圆角、封面尺寸、标题字号和列表间距，让卡片从紧凑
    列表项升级为更像探店收藏卡的展示形态。
  - 来源徽标回到封面左上角，元信息改为图标 + 费用/价格，标签保留在底部。
  - “已打卡”卡片新增鼠尾草绿主题、封面轻灰化、右上斜向“已打卡”印章和
    绿色完成按钮，呼应参考实现里的完成态视觉。
- **影响范围**：
  - 仅影响首页“清单 -> 打卡点”列表卡片视觉和列表间距，不修改后端接口、
    详情页交互或新增/编辑表单。
- **兼容性/风险**：
  - 继续使用现有 `Place` 字段和 uview-plus 可用图标；无新增依赖。
  - 已打卡印章会占用卡片右上空间，超长标题场景已预留右侧避让空间，但仍建议
    在真机中检查极端长名称表现。
- **验证情况**：
  - 已使用 `@vue/compiler-sfc` 完成 `pages/index/index.vue` 与
    `pages/index/components/place-card-item.vue` 的模板/脚本编译检查，通过。
  - 已执行 `git diff --check`，通过。
  - SCSS 完整预处理检查仍受当前依赖环境缺少 `sass` 包限制，建议后续在
    HBuilderX / 微信开发者工具中预览确认真机效果。

## 2026-06-16 (打卡点详情页沉浸式 UI 复原)

### Changed

- **修改时间**：2026-06-16 00:21:02 +0800 CST
- **变更背景**：用户提供打卡点详情页 UI 参考图，要求打开后的详情页按参考图
  进行 1:1 方向复原，并结合 `$ui-ux-pro-max` 的移动端触控与餐饮详情设计建议。
- **核心改动**：
  - 重构 `place-detail-sheet.vue` 为沉浸式详情弹层：顶部大图、来源胶囊、深色
    圆形关闭按钮、白色圆角内容面板、衬线大标题、标签胶囊、位置卡片、
    来源网页卡片和底部固定 CTA。
  - 保留现有业务能力：打开地图、标记去过/改回想去、编辑、删除；编辑/删除
    移到首屏下方管理区，避免破坏参考图首屏视觉。
  - 新增 `open-source` 事件和首页 `openPlaceSourceURL` 处理；在小程序内暂按
    复制来源链接处理，规避未配置 web-view 业务域名时直开外链失败的问题。
  - **追加修正**（2026-06-16 00:27:10 +0800 CST）：将位置卡片右侧不受
    uview-plus 支持的 `nav-fill` 改为 `arrow-upward` 旋转后的导航箭头，避免
    小程序中直接显示图标名称文本。
- **影响范围**：
  - 影响首页打卡点详情弹层的视觉结构和来源链接交互，不修改后端接口、数据
    字段、列表筛选或新增/编辑表单。
- **兼容性/风险**：
  - 图片、位置、来源链接均继续使用现有 `Place` 字段；外链动作为复制链接，
    后续如接入 web-view 可替换为直跳。
  - 该改动视觉范围较大，需在微信开发者工具或真机中确认不同屏高下的首屏贴合度。
- **验证情况**：
  - 已按 `$ui-ux-pro-max` 执行设计系统检索和移动端触控建议检索。
  - 已使用 `@vue/compiler-sfc` 完成 `pages/index/index.vue`、
    `place-card-item.vue`、`place-detail-sheet.vue` 的模板/脚本编译检查，通过。
  - 已执行 `git diff --check`，通过。
  - 已确认 `pages/`、`components/`、`utils/` 中无 `nav-fill` 残留。
  - SCSS 完整预处理检查仍受当前依赖环境缺少 `sass` 包限制，建议后续在
    HBuilderX / 微信开发者工具中预览确认真机效果。

## 2026-06-16 (打卡点名称字体优化)

### Changed

- **修改时间**：2026-06-16 00:09:37 +0800 CST
- **变更背景**：打卡点名称承担地点标题的视觉角色，列表和详情中继续使用无衬线
  字体会显得偏工具化，和当前卡片设计的轻质感不够匹配。
- **核心改动**：
  - 将打卡点列表卡片名称与详情弹层名称统一切换为宋体系衬线字体栈：
    `"Songti SC", "STSong", "SimSun", "DejaVu Serif", serif`。
- **影响范围**：
  - 仅影响打卡点名称文本的字体呈现，不修改列表数据、地图动作、状态切换或
    后端接口。
- **兼容性/风险**：
  - 低风险；字体按系统字体栈兜底，不新增字体文件或运行依赖。
- **验证情况**：
  - 待本地 UI 预览确认 iOS / Android 微信字体兜底效果。

## 2026-06-15 (打卡点地图按钮语义修正)

### Changed

- **修改时间**：2026-06-15 23:48:49 +0800 CST
- **变更背景**：打卡点卡片右下角动作需要明确表达“打开地图”，避免被误读为
  导航箭头或状态标记。
- **核心改动**：
  - 将打卡点卡片右下角动作统一改回地图图标 `map-fill`，保持圆形快捷按钮
    样式，保留无坐标时的弱化态。
- **影响范围**：
  - 仅影响首页“清单 -> 打卡点”列表卡片的右下角动作展示，不修改数据接口、
    状态切换或详情页行为。
- **兼容性/风险**：
  - 低风险；仍调用现有 `open-location` 逻辑，未新增交互分支。
- **验证情况**：
  - 待本地 UI 预览确认图标识别度；已完成源码修改，后续可结合微信开发者工具
    检查视觉效果。

## 2026-06-15 (打卡点列表 UI 优化)

### Changed

- **修改时间**：2026-06-15 23:09:00 +0800 CST
- **变更背景**：打卡点列表在长店名和地址场景下信息挤压明显，右侧地图动作容易被
  内容挤到卡片边缘，影响首页打卡点模式的可读性和操作稳定性。
- **核心改动**：
  - 优化 `place-card-item.vue` 的卡片信息层级：封面缩小、状态改为封面角标、
    标题支持两行稳定截断，地址单行弱化展示，来源与标签收进底部标签区。
  - 右下角动作统一表达为“打开地图”，无坐标时改为弱化样式，避免与“去过”
    状态含义混淆。
  - 调整首页打卡点列表间距、空态卡片和“关于我们”入口的轻量视觉样式。
- **影响范围**：
  - 仅影响首页“清单 -> 打卡点”列表和空态展示，不修改后端接口、字段结构、
    新增/编辑/删除/状态切换逻辑。
- **兼容性/风险**：
  - 低风险；仍复用现有 `Place` 数据字段和 uview-plus 图标，不新增依赖。
- **验证情况**：
  - 已执行 `git diff --check`，通过。
  - 已使用 `@vue/compiler-sfc` 完成 `pages/index/index.vue` 与
    `pages/index/components/place-card-item.vue` 的模板/脚本编译检查，通过。
  - SCSS 编译检查因当前依赖环境缺少 `sass` 包未完整执行；本次样式为局部
    SCSS/CSS 调整，建议后续在 HBuilderX / 微信开发者工具中预览确认真机效果。

## 2026-06-15 (打卡点数据交互 v1)

### Added

- **修改时间**：2026-06-15 22:40:28 +0800 CST
- **变更背景**：首页已有“打卡点”原型入口和卡片展示，但仍使用本地
  `mockPlaces`，没有空间维度数据同步、增删改、状态切换、地图选点或后端接口，
  无法形成可用闭环。
- **核心改动**：
  - 后端新增 `places` 表迁移、`backend/internal/place` 模块和受保护接口：
    `GET/POST /api/kitchens/{kitchenID}/places`、`GET/PUT/DELETE /api/places/{placeID}`、
    `PATCH /api/places/{placeID}/status`；权限模型复用空间成员校验，删除为软删除。
  - 小程序新增 `utils/place-api.js` 与 `utils/place-store.js`，按 `kitchenId`
    本地缓存打卡点数据，并复用现有上传链路保存图片。
  - 首页打卡点模式移除 `mockPlaces`，接入真实列表、新增/编辑半屏弹层、详情弹层、
    删除确认、想去/去过状态切换、搜索筛选、图片预览、微信地图选点和打开地图。
  - `manifest.json` 补充微信位置权限说明；README 与后端 README 同步新增打卡点能力和接口说明。
- **影响范围**：
  - 影响首页“清单 -> 打卡点”模式、底部主按钮在打卡点模式下的点击目标、
    小程序位置权限声明、后端数据库迁移与受保护 API 路由。
  - 不修改菜谱详情页、菜单详情页、饮食管家协议或第三方地点解析能力。
- **兼容性/风险**：
  - 新表独立新增，不迁移既有菜谱或菜单数据；旧客户端不调用新接口时不受影响。
  - 地图选点和打开地图依赖微信小程序位置授权，真机需补充验证授权弹窗与拒绝授权后的提示。
- **验证情况**：
  - 已执行 `go test ./internal/place`，通过。
  - 已执行 `go test ./internal/app`，通过（无测试文件，仅编译检查）。
  - 已执行 `go test ./...`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析并编译
    `pages/index/index.vue`、`place-card-item.vue`、`place-edit-sheet.vue`、
    `place-detail-sheet.vue`，通过。
  - 已执行 `node --check utils/place-api.js utils/place-store.js`，通过。
  - 未运行 HBuilderX / 微信开发者工具真机预览；建议补测新增、编辑、删除、
    切空间同步、离线缓存兜底、地图选点和打开地图。

## 2026-06-09 (AI Provider 支持 thinking 参数透传)

### Changed

- **修改时间**：2026-06-09 17:51:50 +0800 CST
- **变更背景**：DeepSeek `deepseek-v4-flash` 默认开启 thinking；用于后台标题清洗时，
  默认思考模式会消耗 `max_tokens`，导致 `content` 为空或 JSON 不稳定，需要能在
  `AI Provider` 页面显式关闭 thinking。
- **核心改动**：
  - `admin-web` 的 `AI Provider` 文本类节点新增 `Thinking` 与
    `Reasoning Effort` 配置；非图片节点可选择 `auto / enabled / disabled`，
    摘要行会展示已显式配置的 thinking 状态。
  - `backend/internal/airouter` 新增 Provider `extra` 中
    `thinking_type` 与 `reasoning_effort` 的规范化、校验、持久化和 OpenAI-compatible
    请求体透传；`auto` 不发送字段，`thinking_type=disabled` 时不发送
    `reasoning_effort`。
  - README 与后端 README 同步说明 DeepSeek 标题清洗建议配置。
- **影响范围**：
  - 影响 `AI Provider` 多节点路由中 `summary / title` 等
    `chat/completions` 文本节点的可选请求参数。
  - 不影响 `images/generations` 图片节点；图片节点会自动清理文本 thinking 参数。
- **兼容性/风险**：
  - 默认 `auto` 不发送新字段，既有 Provider 行为保持不变。
  - 非 DeepSeek 兼容方若不支持这些字段，应保持 `auto`，避免上游拒绝未知参数。
- **验证情况**：
  - 已执行 `go test ./internal/airouter`，通过。
  - 已执行 `npm --prefix admin-web run build`，通过（仅有既有 chunk size warning）。

## 2026-05-21 (修复首页模板多余闭合标签)

### Fixed

- **修改时间**：2026-05-21 23:39:53 +0800 CST
- **变更背景**：HBuilderX / Vite 编译首页时报
  `[plugin:vite:vue] Invalid end tag`，定位到
  `pages/index/index.vue:540:2` 的根节点结束标签被误判为非法结束标签。
- **核心改动**：
  - 删除 `pages/index/index.vue` 中“美食库模式”和“打卡点模式”之间多余的
    `</view>`，恢复 `appMode === 'cook'` 与 `v-else` 打卡点分支的正确模板层级。
- **影响范围**：
  - 仅影响首页 Vue 模板编译结构，不修改页面业务逻辑、接口契约或运行配置。
- **兼容性/风险**：
  - 低风险；修复前模板根节点被提前闭合，修复后分支结构恢复为预期的
    `cook / explore` 互斥渲染。
- **验证情况**：
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析并编译
    `pages/index/index.vue`，通过。

## 2026-05-11 (新增小程序 AI 助手入口开关)

### Added

- **修改时间**：2026-05-11 11:28:36 +0800 CST
- **变更背景**：需要在后台控制小程序首页底部中间按钮的点击目标；开启 AI
  助手时继续打开饮食管家，关闭后回到常规添加菜谱入口。
- **核心改动**：
  - 后端运行时配置新增 `miniapp.features.diet_assistant_enabled`，
    默认值为 `true`，保持现有线上底部按钮默认打开饮食管家的行为。
  - 后端新增 `GET /api/public/app-config`，只返回小程序公开功能开关，不暴露
    后台敏感配置。
  - `admin-web` 配置中心新增“小程序功能开关”分组，使用现有开关控件维护
    “AI 助手入口”。
  - 小程序首页会拉取并缓存公开配置；底部中间按钮和搜索无结果空态会按开关
    分流到饮食管家或添加菜谱弹层。
- **影响范围**：
  - 影响小程序首页底部中间按钮点击行为和搜索无结果空态主操作。
  - 不修改饮食管家聊天协议、菜谱添加接口、AI Provider 路由配置或登录链路。
- **兼容性/风险**：
  - 默认开启，旧部署升级后行为保持不变；只有后台显式关闭后才改为添加菜谱。
  - 小程序拉取公开配置失败时使用本地缓存或默认开启兜底。
- **验证情况**：
  - 已执行 `go test ./internal/appsettings`，通过。
  - 已执行 `go test ./internal/app`，通过。
  - 已执行 `go test ./internal/admin`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/public-app-config-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析
    `pages/index/index.vue` 与 `admin-web/src/pages/SettingsPage.vue`，通过。
  - 已执行 `npm --prefix admin-web run build`，通过（仅有既有 chunk size warning）。

## 2026-05-11 (修复饮食管家 DeepSeek thinking tools 回传协议)

### Fixed

- **修改时间**：2026-05-11 11:24:03 +0800 CST
- **变更背景**：线上饮食管家切到 `deepseek-v4-flash` 后，tools 规划阶段在
  thinking mode 下会返回 `reasoning_content`；后续工具结果请求若只回传
  `tool_calls` 而未回传该字段，上游会返回
  `The 'reasoning_content' in the thinking mode must be passed back to the API.`。
- **核心改动**：
  - `backend/internal/dietassistant` 的 OpenAI-compatible assistant 消息结构新增
    `reasoning_content` 字段，并在 tools 调用后随原 assistant tool-call 消息原样带回。
  - 饮食管家 AI 请求新增可选环境变量
    `DIET_ASSISTANT_AI_THINKING_TYPE=enabled|disabled` 和
    `DIET_ASSISTANT_AI_REASONING_EFFORT=high|max`；留空时不发送对应字段，继续使用
    provider 默认行为。
  - `backend/configs/example.env`、`README.md` 与 `backend/README.md` 同步补充
    thinking 配置口径。
- **影响范围**：
  - 影响 `POST /api/diet-assistant/chat/stream` 对 DeepSeek-style thinking
    OpenAI-compatible 接口的请求上下文组装。
  - 不修改小程序 SSE 事件结构、聊天记录表结构、饮食管家 tools 契约或 LongCat
    内嵌工具标记兼容逻辑。
- **兼容性/风险**：
  - 默认不主动发送 `thinking` / `reasoning_effort`，避免影响 LongCat 或其它
    OpenAI-compatible provider。
  - 当 thinking mode 开启时，后端会按上游要求回传 `reasoning_content`；如果显式配置
    `DIET_ASSISTANT_AI_THINKING_TYPE=disabled`，则不会发送
    `reasoning_effort`。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./internal/config`，通过（无测试文件，仅编译检查）。
  - 已执行 `go test ./...`，通过。

## 2026-05-09 (优化 GPT Image b64_json 生图协议)

### Changed

- **修改时间**：2026-05-09 17:33:16 +0800 CST
- **变更背景**：`AI Provider -> flowchart` 接入 `gpt-image-*`
  图片生成节点时，继续把 `response_format=b64_json` 当成主请求字段会与
  GPT Image 默认 `b64_json` 响应口径冲突；同时原实现固定
  `output_format=png`，无法支持 `jpeg + output_compression`、尺寸、质量和背景等
  输出控制参数。
- **核心改动**：
  - `backend/internal/airouter` 为 `images/generations` 节点新增
    `size / quality / background / output_format / output_compression / n`
    参数解析、校验和持久化，参数来源为 Provider `extra`。
  - `gpt-image-*` 模型不再随请求发送 `response_format`；`responseFormat`
    保留为 DALL-E / 三方兼容字段和后端响应解码偏好。
  - 后端解析 `b64_json` 时会按实际 `output_format` 生成
    `data:image/jpeg|png|webp;base64,...`，避免 `jpeg` 响应被错误标记成
    `png`。
  - `admin-web` 的 AI Provider 页面为 `images/generations` 节点新增图片尺寸、
    质量、背景、输出格式、压缩率和生成数量配置，新建 flowchart 图片节点默认
    使用 `1536x1024 / high / opaque / jpeg / 60 / n=1`。
  - **追加调整**（2026-05-09 17:24:34 +0800 CST）：背景字段标题新增
    HelpTip，并在 `auto / opaque / transparent` 下拉选项内补充一句话说明，提示
    `jpeg` 建议使用不透明背景、透明背景更适合 `png / webp`。
  - **追加修复**（2026-05-09 17:33:16 +0800 CST）：为背景下拉框增加专属
    `popper-class`，修复 Element Plus 默认单行选项高度导致 tips 文案重叠和裁切的问题。
  - README、后端 README 与 AI 多 Provider 设计文档同步更新协议口径。
- **影响范围**：
  - 影响 `flowchart` 场景中配置为 `images_generations` 的 AI Provider 请求体和
    响应解析。
  - 不影响 `summary / title` 文本场景，不影响 `chat/completions` 形式的流程图节点。
- **兼容性/风险**：
  - 旧的 `responseFormat=b64_json` 配置仍可保留；对 `gpt-image-*` 只作为解码偏好，
    对非 GPT Image 模型仍按原逻辑随请求发送。
  - 旧节点未配置图片输出参数时继续默认 `output_format=png`；只有在后台保存
    Provider `extra` 后才会切到新的尺寸 / JPEG / 压缩配置。
- **验证情况**：
  - 已执行 `go test ./internal/airouter`，通过。
  - 已执行 `go test ./internal/recipe`，通过。
  - 已执行 `go test ./internal/appsettings`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `npm --prefix admin-web run build`，通过（仅有既有 chunk size warning）。
  - 已执行 `git diff --check`，通过。

## 2026-05-09 (配置中心移除旧 AI 单节点分组)

### Changed

- **修改时间**：2026-05-09 14:47:42 +0800 CST
- **变更背景**：`AI Provider` 页面已承接 `summary / title / flowchart`
  多节点路由配置，配置中心继续展示旧单节点 AI 分组会造成运维入口重复和生效口径混淆。
- **核心改动**：
  - `backend/internal/appsettings/runtime_provider.go` 将 `ai.summary`、
    `ai.title`、`ai.flowchart` 标记为后台不可见分组；`GET /api/admin/runtime-settings`
    不再返回这些旧单节点配置，`PUT/POST /api/admin/runtime-settings/groups/{group}`
    也不再允许编辑或测试这些分组。
  - 后端仍保留旧 `app_runtime_settings` / 环境变量读取能力作为内部兼容兜底；
    正常运维入口统一使用 `admin-web` 的 `AI Provider` 页面。
  - `admin-web/src/pages/SettingsPage.vue` 移除旧兼容模式提示分支，
    `admin-web/src/router/index.ts` 更新配置中心描述。
  - `README.md`、`backend/README.md` 与 AI Provider 相关设计文档同步更新配置入口口径。
- **影响范围**：
  - 影响后台配置中心展示、保存和测试的分组范围。
  - 不影响 `AI Provider` 多节点路由页面、路由配置表、AI 调用审计或小程序端功能。
- **兼容性/风险**：
  - 如果某个部署尚未在 `AI Provider` 页面启用对应场景，后端内部仍可按旧配置兜底；
    但后台不再提供旧单节点分组的可视化编辑入口。
- **验证情况**：
  - 已执行 `go test ./internal/appsettings`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `npm --prefix admin-web run build`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (修复饮食管家流式阶段 LongCat 工具标记泄漏)

### Fixed

- **修改时间**：2026-05-01 03:16:46 +0800 CST
- **变更背景**：用户截图反馈饮食管家最终回复中仍直接展示
  `<longcat_tool_call>get_recipe_by_id...</longcat_tool_call>`，说明此前只覆盖
  非流式工具规划阶段的兜底解析，未覆盖最终流式回复阶段继续吐出 LongCat
  内嵌工具标记的场景。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 为最终流式回复新增
    LongCat 工具标记流式拦截器，可处理 `<longcat_tool_call>` 被 delta 分片
    拆开的情况，避免内部标记继续转发给小程序前端。
  - 当最终流式回复中解析到 `get_recipe_by_id` 等工具标记时，后端会将其转换为
    真实工具调用，执行后把工具结果追加回上下文，并继续请求模型生成最终自然语言回复。
  - 为嵌套工具调用增加最多 3 轮保护，避免上游模型反复输出工具标记导致无限循环。
  - `backend/internal/dietassistant/service_test.go` 补充截图同类回归用例，覆盖
    流式 delta 分片、详情工具执行、继续生成回复以及不透传原始 LongCat 标记。
- **影响范围**：
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 的最终流式回复消费逻辑。
  - 不修改前端渲染协议、SSE 事件结构、菜谱详情工具契约或聊天记录表结构。
- **兼容性/风险**：
  - 若 LongCat 在最终回复阶段连续多轮要求工具调用，最多执行 3 轮；超过后返回上游异常，避免请求长时间挂起。
  - 已经被模型输出在工具标记之前的自然语言前缀仍会按流式体验展示；内部工具标记本身不会展示。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `git diff --check -- backend/internal/dietassistant/service.go backend/internal/dietassistant/service_test.go`，通过。

## 2026-05-01 (优化饮食管家弹层打开流畅度)

### Changed

- **修改时间**：2026-05-01 02:54:29 +0800 CST
- **变更背景**：用户反馈打开饮食管家 AI 聊天框后有卡顿和会话上下移动感，需要优化弹层打开阶段的布局稳定性与滚动体验。
- **核心改动**：
  - `pages/index/components/diet-assistant-sheet.vue` 将历史消息同步延后到弹层打开动画之后再触发，避免打开阶段同时执行弹层动画、网络请求、消息替换和滚动定位。
  - 饮食管家聊天区取消独立的“正在同步历史...”占位行，改为复用顶部固定灵感状态胶囊展示“今日灵感 / 同步中”，减少内容高度变化造成的跳动。
  - 将原“今天的灵感台”调整为更短的“今日灵感”，并增加轻量状态点，降低装饰感和视觉占位。
  - 新增 `shouldAnimateScroll` 和滚动锚点定时器：打开 / 历史回填 / 流式增量默认使用非动画滚动，发送消息时才保留一次轻量动画，并对连续 delta 更新做节流。
  - 降低弹层遮罩 blur 强度，减少低端设备打开弹层时的合成层压力。
- **影响范围**：
  - 仅影响饮食管家弹层前端打开、同步历史和滚动定位体验。
  - 不修改后端聊天接口、消息存储、function calling、SSE 协议或最终回复内容。
- **兼容性/风险**：
  - 历史消息同步会在弹层打开后约 220ms 触发，换取打开动画稳定；用户立刻发送消息时仍会跳过历史同步，避免覆盖当前流式会话。
- **验证情况**：
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue` 与 `pages/index/index.vue`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (修复饮食管家 LongCat 工具标记泄漏)

### Fixed

- **修改时间**：2026-05-01 03:02:59 +0800 CST
- **变更背景**：用户在饮食管家里咨询“清淡点的”之类问题时，偶发收到
  LongCat 内部 `<longcat_tool_call>...</longcat_tool_call>` 标记；本地
  `diet_assistant_messages` 也已查到真实脏数据，说明上游工具规划结果被当成正文透传并落库。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 为 tools 规划阶段新增
    LongCat 内嵌工具标记 fallback 解析：当上游未返回标准 `tool_calls`
    但正文里带有 `<longcat_tool_call>` 时，后端会解析成真实 tool call，
    补齐 `search_recipes_by_name` 等工具的默认参数后继续执行。
  - 同文件新增助手消息清洗逻辑：进入模型上下文、返回历史消息以及写库前，
    都会移除 LongCat 内部工具标记，避免脏回复污染后续多轮对话。
  - `backend/internal/dietassistant/service_test.go` 补充
    LongCat 标记 fallback 解析与历史消息清洗的回归用例。
- **影响范围**：
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 的上游工具规划兼容逻辑。
  - 影响 `GET /api/diet-assistant/messages` 返回的历史消息清洗结果；历史里的异常工具标记会被过滤，不再继续展示或带入上下文。
- **兼容性/风险**：
  - 当前 fallback 仅针对 LongCat 这类 XML 风格工具标记；若未来上游再出现新的非标准格式，仍需按样本继续补兼容。
  - 被完整识别为内部工具标记的历史助手消息会在读取时被过滤，极少数异常轮次可能只保留用户那一侧消息。
- **验证情况**：
  - 已执行 `gofmt -w backend/internal/dietassistant/service.go backend/internal/dietassistant/service_test.go`，通过。
  - 已执行 `git diff --check -- backend/internal/dietassistant/service.go backend/internal/dietassistant/service_test.go`，通过。
  - 未运行 `go test`；本轮按用户要求直接提交。

## 2026-05-01 (饮食管家菜谱详情与食材搜索 tools)

### Changed

- **修改时间**：2026-05-01 02:40:37 +0800 CST
- **变更背景**：饮食管家需要支持用户基于菜谱 ID 获取详情，同时现有按菜名搜索 tool 需要扩展到食材模糊查询，并限制默认返回数量避免性能风险。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 新增 `get_recipe_by_id` tool，支持根据菜谱 ID 获取当前空间菜谱详情，并在 SSE 工具执行阶段展示“正在读取菜谱详情”等实时状态。
  - `search_recipes_by_name` tool 扩展为按菜名或食材搜索，新增 `keyword`、`searchScope` 和 `ingredientKeyword` 入参口径，默认返回 5 条、最多 10 条。
  - `backend/internal/recipe/repository.go` 新增菜名 / 食材专用过滤条件，食材搜索同时匹配 `ingredient` 摘要字段和结构化 `ingredients_json`，避免命中备注或链接。
  - `backend/internal/app/app.go` 将详情 tool 串接到现有 `recipeService.GetByID()`，并校验菜谱属于当前会话空间；搜索 tool 改用菜名 / 食材专用过滤条件。
  - `README.md` 与 `backend/README.md` 更新饮食管家 tools 能力说明。
- **影响范围**：
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 的 function calling 工具集合、搜索入参和工具状态文案。
  - 不改变小程序正式菜谱详情接口、菜谱列表接口和数据库结构。
- **兼容性/风险**：
  - `search_recipes_by_name` 名称保持不变，但语义从“只按菜名”扩展为“菜名或食材”；旧模型若继续传 `titleKeyword` 仍兼容。
  - 食材搜索基于 SQLite `LIKE`，当前默认 5 / 最大 10 控制返回规模；数据量继续增长后可再考虑索引或 FTS。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./internal/recipe`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue` 与 `pages/index/index.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (饮食管家消息加粗渲染)

### Added

- **修改时间**：2026-05-01 02:24:47 +0800 CST
- **变更背景**：饮食管家 AI 回复中可能包含 Markdown 风格的 `**加粗**` 标记，原聊天气泡使用普通 `<text>` 渲染，导致标记原样展示。
- **核心改动**：
  - `pages/index/components/diet-assistant-sheet.vue` 将真实聊天消息内容从纯文本 `<text>` 改为 `rich-text` nodes 渲染。
  - 新增轻量 Markdown 解析，仅支持 `**...**` 加粗和换行；其它内容继续作为文本节点处理，避免引入完整 Markdown 依赖和额外 HTML 注入面。
- **影响范围**：
  - 影响饮食管家聊天消息展示；欢迎语、输入框、后端 SSE 协议和聊天记录存储不变。
- **兼容性/风险**：
  - 当前只支持加粗，不支持标题、列表、链接、图片等完整 Markdown 语法；未闭合的 `**` 会按普通文本展示。
- **验证情况**：
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (修复饮食管家单轮后发送状态卡住)

### Fixed

- **修改时间**：2026-05-01 02:09:58 +0800 CST
- **变更背景**：用户反馈在饮食管家当前窗口中问完一个问题后，发送按钮持续置灰，只能退出并重新进入 AI 窗口后才能继续提问。
- **核心改动**：
  - `pages/index/components/diet-assistant-sheet.vue` 将 `resetStreamState()` 的定义提前到发起流式请求之前，避免 SSE `done` 或 promise 回调在极端时序下访问尚未初始化的复位函数，导致 `isStreaming` 无法释放。
  - **追加调整**（2026-05-01 02:18:51 +0800 CST）：饮食管家前端改用当前助手消息 ID 作为流式请求释放状态的唯一标识，不再依赖 `activeStream` 对象引用匹配；同时历史同步状态不再让发送按钮置灰，避免历史请求卡住时影响继续提问。
  - `utils/diet-assistant-api.js` 新增流式回调安全调用封装，确保 `onDelta`、`onStatus`、`onError`、`onDone` 等 UI 回调即使抛错，也不会阻断内部 `finished` promise 的 resolve / reject。
- **影响范围**：
  - 影响饮食管家流式回复完成后的输入框和发送按钮状态释放。
  - 不修改后端聊天接口、聊天记录落库、工具调用结果或最终 AI 回复内容。
- **兼容性/风险**：
  - 回调异常现在会被吞掉并写入控制台 warning，优先保证聊天流状态可以收口；后续如需排查 UI 回调错误，可结合开发者工具 console 查看。
- **验证情况**：
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue` 与 `pages/index/index.vue`，通过。
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (饮食管家写库后列表自动刷新)

### Added

- **修改时间**：2026-05-01 02:06:15 +0800 CST
- **变更背景**：用户在饮食管家聊天窗口通过链接解析保存菜谱后，后端已写库，但首页美食库列表仍停留在旧前端状态，需在工具确认写入后自动刷新。
- **核心改动**：
  - `backend/internal/dietassistant/model.go` 为 SSE `StreamEvent` 新增 `mutation` 字段，当前支持 `recipe_created`，包含 `recipeId`、`recipeTitle`、`mealType` 和 `status`。
  - `backend/internal/dietassistant/service.go` 在 `parse_and_add_recipe_from_url` 工具成功后，于 `tool_done` 事件附带 `mutation.type=recipe_created`，失败时不发送 mutation。
  - `utils/diet-assistant-api.js` 解析 SSE 状态事件中的 `mutation`，并传给饮食管家弹层。
  - `pages/index/components/diet-assistant-sheet.vue` 收到 mutation 后向父页面派发 `recipes-mutated` 事件。
  - `pages/index/index.vue` 收到 `recipe_created` 后静默调用 `refreshRecipes({ silent: true })`，让首页列表、数量和缓存同步更新。
- **影响范围**：
  - 影响饮食管家链接解析保存菜谱后的前端同步体验。
  - 不改变菜谱写库逻辑、聊天记录存储内容或最终 AI 回复文本。
- **兼容性/风险**：
  - 旧客户端会忽略 `mutation` 字段，仍能正常消费原有 SSE 状态和回复。
  - 当前只对 `parse_and_add_recipe_from_url` 成功写入菜谱发送 `recipe_created`；后续新增删除 / 更新工具时可复用该 mutation 协议。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue` 与 `pages/index/index.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (饮食管家工具调用实时状态)

### Added

- **修改时间**：2026-05-01 01:54:50 +0800 CST
- **变更背景**：饮食管家执行 function calling 时，前端原本只能显示固定“正在整理...”，用户无法知道当前是在统计美食库、查找菜谱还是解析链接保存食材。
- **核心改动**：
  - `backend/internal/dietassistant/model.go` 为 SSE `StreamEvent` 增加 `toolName` 字段，保留既有 `delta` / `error` / `done` 事件兼容性。
  - `backend/internal/dietassistant/service.go` 在工具规划阶段发送 `status`，在工具执行前后发送 `tool_start`、`tool_done` 或 `tool_error`，分别展示“正在统计美食库”“正在查找菜谱”“正在解析链接并保存食材”等状态。
  - 纯 URL 输入直接走 `parse_and_add_recipe_from_url` 时同样会实时发送工具状态，不再只有最终 AI 流式回复。
  - `utils/diet-assistant-api.js` 识别 `status` / `tool_start` / `tool_done` / `tool_error` 事件并回调前端。
  - `pages/index/components/diet-assistant-sheet.vue` 为 pending 助手消息新增 `statusText`，在 AI 正文 delta 到达前动态更新气泡文案。
  - `README.md` 与 `backend/README.md` 补充饮食管家工具执行阶段 SSE 状态事件说明。
- **影响范围**：
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 的 SSE 事件内容；旧客户端会忽略新增事件，继续消费 `delta` / `done`。
  - 仅调整饮食管家弹层 pending 文案，不改变聊天记录落库内容和最终 AI 回复内容。
- **兼容性/风险**：
  - 新状态事件不会写入后端聊天历史；历史记录仍只保存用户消息和助手最终回复。
  - 如果某个工具失败，前端会先显示失败状态，随后仍由模型基于工具错误生成最终说明。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (饮食管家 URL 解析保存 tool)

### Changed

- **修改时间**：2026-05-01 01:34:59 +0800 CST
- **变更背景**：饮食管家当前暴露的直接添加菜谱 tool 入参较多，一版不需要单独添加食材 / 手填完整菜谱信息；更适合先支持用户只输入 B 站或小红书 URL，由系统解析内容并保存菜谱食材。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 从 tools 列表中移除 `add_recipe`，新增 `parse_and_add_recipe_from_url`，入参收敛为 `url`、`mealType`、`status`。
  - 饮食管家系统提示明确当前不提供单独添加食材工具；用户只输入 URL 时会直接走 URL 解析保存链路，无需等待模型自行判断是否调用工具。
  - `backend/internal/app/app.go` 将新 tool 串接到现有 `linkparse.Service.ParseRecipeLink()` 和 `recipeService.CreateFromInput()`：解析 B 站 / 小红书链接后写入菜谱标题、食材摘要、封面图、结构化主料 / 辅料 / 步骤和来源链接。
  - tool 返回中包含已保存菜谱、解析来源、主料 / 辅料摘要、步骤数量和解析警告，供模型生成最终流式回复。
  - `README.md` 与 `backend/README.md` 更新饮食管家 tools 口径，去除直接添加菜谱 / 单独添加食材的当前能力描述。
- **影响范围**：
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 的 function calling 工具集合和纯 URL 输入的处理路径。
  - 不修改正式菜谱创建接口 `POST /api/kitchens/{kitchenID}/recipes`，也不新增数据库表。
- **兼容性/风险**：
  - 只有 B 站 / 小红书等现有 `linkparse` 支持的平台链接可被自动解析保存；不支持的平台会通过工具错误交给模型解释。
  - 用户如果只想咨询某个链接但不想保存，纯 URL 输入会默认保存为想吃正餐；后续可通过更明确的话术或前端二次确认优化。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./internal/app ./internal/recipe ./internal/linkparse`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (饮食管家真实添加与菜名搜索 tools)

### Changed

- **修改时间**：2026-05-01 01:13:59 +0800 CST
- **变更背景**：饮食管家需要从模拟添加升级为可真正调用系统菜谱能力的 Agent，并支持用户按菜名模糊查询美食库中已有菜谱。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 将 function calling 工具从 `add_recipe_mock` 升级为真实写库的 `add_recipe`，并新增 `search_recipes_by_name`。
  - `add_recipe` 会按当前登录用户与当前空间写入菜谱，支持菜名、餐别、状态、食材、摘要、备注和来源链接；工具入参会做基础枚举归一与长度裁剪后复用菜谱创建链路。
  - `search_recipes_by_name` 只按菜谱名模糊查询，支持餐别和状态过滤，最多返回 10 条精简菜谱结果，避免误把食材、备注或链接命中当成菜名命中。
  - `backend/internal/recipe/model.go` / `service.go` 新增导出的 `CreateInput` 和 `CreateFromInput()`，供饮食管家在不绕过菜谱 service 校验的前提下创建菜谱；`ListFilter` 新增 `TitleKeyword`。
  - `backend/internal/app/app.go` 串接饮食管家工具回调到现有 `recipeService`，并把菜谱实体映射为工具返回的精简字段。
  - `README.md` 与 `backend/README.md` 更新饮食管家 tools 说明，去除“模拟添加不写库”的当前口径。
- **影响范围**：
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 的工具能力和模型系统提示。
  - 影响菜谱列表 repository 的可选过滤字段，但既有 `keyword` 查询和前端菜谱列表接口保持兼容。
- **兼容性/风险**：
  - 用户明确要求“添加 / 记录 / 保存”时，AI 现在会真正创建菜谱；误识别用户意图时可能产生真实数据，需要后续真机重点观察模型触发工具的边界。
  - 搜索工具按标题模糊查询，不会返回只在食材、备注或链接中命中的菜谱。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant ./internal/recipe`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (修复饮食管家回复完成后发送按钮未释放)

### Fixed

- **修改时间**：2026-05-01 01:01:03 +0800 CST
- **变更背景**：用户反馈 AI 回复内容已经显示完成，但第二次提问时发送按钮仍置灰，无法继续发送。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 调整流式收口顺序：模型流结束后先向前端发送 SSE `done`，再执行聊天记录落库，避免存储步骤延迟导致前端 `isStreaming` 长时间不释放。
  - `pages/index/components/diet-assistant-sheet.vue` 增加 `onDone` 回调处理，收到 SSE `done` 时立即完成助手消息并重置流式状态，不再只依赖 `stream.finished.then()` 的异步收口。
- **影响范围**：
  - 影响饮食管家流式回复完成后的输入框和发送按钮状态释放。
  - 不修改上游 AI 请求、工具调用、聊天历史请求体或后端存储表结构。
- **兼容性/风险**：
  - 聊天记录仍在同一请求内保存；如果保存失败，本轮回复已经释放给前端，后续历史可能缺少该轮记录，和既有“保存失败不阻断回复”策略一致。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-05-01 (饮食管家欢迎区 UI 优化)

### Changed

- **修改时间**：2026-05-01 00:57:07 +0800 CST
- **变更背景**：后端聊天记忆接入后，空历史欢迎区需要更轻量，同时保留一句必要的身份说明；截图中快捷建议卡片集中在气泡左半边，右侧留白过大，视觉重心不稳。
- **核心改动**：
  - `pages/index/components/diet-assistant-sheet.vue` 将欢迎气泡标记为 `chat-bubble--welcome`，并把三条快捷建议改为带图标、文案和右箭头的完整宽度操作行。
  - 欢迎区保留一句简短自我介绍“我是饮食管家，帮你想菜、记菜谱。”，移除多余说明文字和快捷建议副说明。
  - `pages/index/components/diet-assistant-sheet.scss` 重做欢迎气泡和建议项样式：气泡宽度适配头像后的剩余空间，建议项改为单列操作入口、固定触控高度、图标章、轻阴影和分层暖色背景。
- **影响范围**：
  - 仅影响饮食管家弹层空历史欢迎区与快捷建议卡片视觉。
  - 不修改后端存储、流式对话、工具调用或历史读取 / 清空接口。
- **兼容性/风险**：
  - 使用现有 `uview-plus` `up-icon` 图标能力，不新增静态资源。
  - 欢迎气泡更宽，需在微信开发者工具或真机确认极窄屏下文本换行和操作行右箭头不拥挤。
- **验证情况**：
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。
  - 未运行 HBuilderX 或微信开发者工具真机预览；建议真机确认空历史首屏、历史消息存在时欢迎区与真实消息的间距。

## 2026-05-01 (饮食管家聊天记忆后端存储)

### Added

- **修改时间**：2026-05-01 00:46:25 +0800 CST
- **变更背景**：饮食管家已有前端内存多轮上下文，但关闭 / 重新打开后不会从后端恢复；本轮补齐后端聊天记忆存储能力，并让清空会话同步清理后端记录。
- **核心改动**：
  - 新增迁移 `backend/migrations/019_add_diet_assistant_messages.sql`，创建 `diet_assistant_messages` 表，按 `user_id + kitchen_id` 存储饮食管家消息，并为用户空间查询和创建时间建立索引。
  - 新增 `backend/internal/dietassistant/repository.go`，支持追加单条消息、事务式保存完整一轮 `user + assistant`、按最近条数读取历史、清空当前用户当前空间历史。
  - `backend/internal/dietassistant/service.go` 在流式最终回复成功完成后保存本轮问答；上游失败、用户中断或助手空回复不保存半截记录；保存失败不阻断已经返回给用户的 AI 回复。
  - 新增受登录态保护的 `GET /api/diet-assistant/messages?kitchenId=...&limit=...` 与 `DELETE /api/diet-assistant/messages?kitchenId=...`，读取 / 清空前都会校验当前用户仍属于目标空间。
  - `utils/diet-assistant-api.js` 增加历史读取和清空 API；`pages/index/components/diet-assistant-sheet.vue` 打开弹层时同步后端历史，发送时继续携带历史上下文，点击“清空会话记录”会同步清空后端。
  - `pages/index/components/diet-assistant-sheet.vue` 移除空会话时的固定示例问答和示例菜谱卡，仅保留欢迎引导与快捷建议，避免与后端真实历史混淆。
  - `README.md` 与 `backend/README.md` 补充饮食管家聊天记忆存储、接口和清空口径。
- **影响范围**：
  - 影响饮食管家聊天流式接口的成功收口逻辑，以及饮食管家弹层打开 / 清空时的前后端交互。
  - 空历史用户打开饮食管家时不再看到模拟用户提问和模拟助手回复。
  - 新增数据库表和迁移；不修改菜谱正式创建接口，也不改变 `add_recipe_mock` 仍为模拟添加的策略。
- **兼容性/风险**：
  - 历史记录按当前登录用户隔离，同一空间内不同成员不会互相看到聊天记录；切换空间后读取对应空间的个人历史。
  - 每次成功回复会额外写入两条消息；默认读取最近 50 条，后端最多允许 100 条。
  - 如果保存聊天记录失败，用户仍能看到本次流式回复，但下次打开可能缺少该轮历史。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-04-30 (生产饮食管家 LongCat 配置与后端重部署)

### Changed

- **修改时间**：2026-04-30 23:53:05 +0800 CST
- **变更背景**：生产饮食管家需要切换到 LongCat OpenAI-compatible 上游，并按用户要求拉取最新代码后通过后端专用脚本重新发布。
- **核心改动**：
  - 已在未纳入 Git 的 `backend/configs/prod.env` 中更新 `DIET_ASSISTANT_AI_BASE_URL=https://api.longcat.chat/openai/v1`、`DIET_ASSISTANT_AI_MODEL=LongCat-2.0-Preview`、`DIET_ASSISTANT_AI_TIMEOUT_SECONDS=90`。
  - 已同步更新 `DIET_ASSISTANT_AI_API_KEY` 到用户提供的新密钥；变更记录不记录真实 API Key。
  - 首次按纯配置发布执行 `RUN_GIT_PULL=0 bash scripts/deploy-backend-on-server.sh`，随后按用户补充要求执行 `bash scripts/deploy-backend-on-server.sh` 拉取远端最新代码并重新构建、重启后端。
  - 代码从 `8ae91618342f9437d350a7929a884575e41a5d78` 快进到 `50f36f722eeed6d4a06bfead0f42f799067aafc0`。
- **影响范围**：
  - 生产后端运行配置：`backend/configs/prod.env`。
  - 生产后端二进制：`backend/bin/server`。
  - systemd 服务：`caipu-backend`。
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 的上游 AI Provider；不修改数据库结构或小程序端已发布包。
- **兼容性/风险**：
  - `DIET_ASSISTANT_AI_API_KEY` 仅保存在生产配置文件中，`backend/configs/prod.env` 已被 `.gitignore` 忽略。
  - 最新代码包含饮食管家 LongCat tools agent 逻辑；生产环境变量已显式指向 LongCat，避免继续使用旧 `dots2api` 上游。
  - 本次未使用真实用户 token 发送饮食管家对话，避免产生真实上游 token 消耗。
- **验证情况**：
  - `bash scripts/deploy-backend-on-server.sh` 完成，脚本摘要显示 `build backend: yes`、`build admin-web: no`、`restart backend: yes`。
  - `caipu-backend` 于 `2026-04-30 23:52:49 CST` 重启，`MainPID=426053`，`SubState=running`。
  - `curl -fsS http://127.0.0.1:8080/healthz` 与 `curl -fsS http://127.0.0.1:8080/api/healthz` 均返回 `code=0`。
  - 本机无 token 探测 `POST /api/diet-assistant/chat/stream` 返回 `401 Unauthorized`，确认接口已注册且仍受登录态保护。

## 2026-04-30 (饮食管家 LongCat Tools Agent 接入)

### Changed

- **修改时间**：2026-04-30 23:39:09 +0800 CST
- **变更背景**：饮食管家需要从单纯 AI 聊天升级为可调用系统能力的 Agent；本轮明确不再使用 `dots2api` 作为默认饮食管家上游，改为 LongCat OpenAI-compatible 接口，并先接入两个最小工具闭环。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 将原直接流式转发改为两段式 Agent loop：第一段非流式 `tools/tool_calls` 决策，执行工具后第二段再请求模型并通过 SSE 流式输出最终回复。
  - 新增工具 `get_recipe_count`：按当前登录用户与当前空间统计菜谱数量，支持餐别 `breakfast/main/all` 与状态 `wishlist/done/all` 过滤。
  - 新增工具 `add_recipe_mock`：模拟添加菜谱，只返回将要保存的菜名、餐别、状态、食材、摘要和备注，不真正调用创建菜谱接口、不写数据库。
  - `utils/diet-assistant-api.js` 在饮食管家流式请求体中带上当前 `kitchenId`，后端用它做当前空间工具上下文，并继续保持小程序端 `enableChunked` SSE 解析方式。
  - `backend/internal/config/config.go` 与 `backend/configs/example.env` 将饮食管家默认上游改为 `https://api.longcat.chat/openai/v1`，默认模型改为 `LongCat-2.0-Preview`。
  - `README.md` 与 `backend/README.md` 补充饮食管家 LongCat tools、模拟添加和菜谱数量统计口径。
- **影响范围**：
  - 影响 `POST /api/diet-assistant/chat/stream` 的上游请求模式和请求体契约；前端仍消费原有 `delta/error/done` SSE 事件。
  - 菜谱数量工具会读取当前空间菜谱列表；模拟添加工具不会修改菜谱、菜单或数据库。
  - 饮食管家默认配置不再指向 `dots2api`；如果部署环境变量仍显式设置旧地址，则仍以环境变量为准。
- **兼容性/风险**：
  - 每轮对话现在最多产生两次上游调用：一次工具决策、一次最终流式回复；相比原直接流式转发会增加一次延迟和 token 成本。
  - 依赖上游模型支持 OpenAI-compatible `tools/tool_calls`；已用 LongCat 手工探针验证 `LongCat-2.0-Preview` 与 `LongCat-Flash-Thinking-2601` 支持。
  - 当前只做一轮工具调用，不支持工具结果后继续递归调用更多工具；适合本轮最小闭环。
  - 正式保存菜谱尚未接入，`add_recipe_mock` 会在回复中明确“模拟添加，未真正保存”。
- **验证情况**：
  - 已执行 `go test ./internal/dietassistant`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `node --check utils/diet-assistant-api.js`，通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，通过。
  - 已执行 `git diff --check`，通过。

## 2026-04-30 (帮我选结果卡长图叠字优化)

### Changed

- **修改时间**：2026-04-30 15:36:53 +0800 CST
- **变更背景**：`帮我选` 弹出的菜品结果卡需要参考用户截图优化成更强图片感的移动端卡片：图片更长，标签、菜名、摘要叠在图片上，底部只保留操作按钮。
- **核心改动**：
  - `pages/index/components/random-pick-sheet.vue` 将菜品标签、菜名和摘要从独立正文区移动到封面图底部叠层。
  - `pages/index/components/random-pick-sheet.scss` 将封面高度从原 `290rpx` 调整为 `700rpx`，卡片背景收敛为暖白底，底部操作区独立成浅色按钮栏。
  - 为封面图新增上下方向深色渐变遮罩和文字阴影，保证复杂食物图片上仍能读取标签、标题和摘要。
  - 底部按钮按用户确认调整为左侧主按钮 `了解做法`、右侧次按钮 `换一个`。
- **追加调整**：
  - **修改时间**：2026-04-30 15:41:49 +0800 CST
  - `pages/index/components/random-pick-sheet.vue` 将 `up-popup` 内容层显式设为 `bgColor="transparent"`，并用 `overlayOpacity="1"` 承载 `overlayStyle` 的浅暖色半透明玻璃背景；`overlayStyle` 启用 `backdrop-filter: blur(30rpx) saturate(1.12)`，背景色降为 `rgba(255, 250, 244, 0.08)`。
  - `pages/index/components/random-pick-sheet.scss` 弱化全屏环境光层，去掉明显彩色遮罩，改成更轻的暖白玻璃层；卡片外发光同步降透明度，减少打开弹层后背景被“盖住”的感觉。
  - **修改时间**：2026-04-30 15:51:12 +0800 CST
  - 为提高弹层后方页面可见性，`randomPickOverlayStyle` 背景色从 `rgba(255, 250, 244, 0.08)` 降为 `rgba(255, 250, 244, 0.04)`，背景模糊从 `30rpx` 降为 `22rpx`；`.random-pick-stage__ambient` 移除二次 `backdrop-filter`，仅保留极轻暖白渐变。
- **影响范围**：
  - 仅影响首页点击 `帮我选` 后的随机菜品弹层展示与按钮排布。
  - 不修改随机抽取逻辑、菜品详情跳转逻辑、菜谱数据结构或图片缓存逻辑。
- **兼容性/风险**：
  - 结果卡高度增加，仍保留弹层固定宽度和底部按钮区；建议在小屏真机确认垂直居中与底部按钮可见性。
  - 图片文字叠层依赖深色渐变遮罩；极暗封面图上仍可能显得偏重，但文字可读性优先。
  - 背景玻璃模糊依赖小程序宿主对 `backdrop-filter` / `-webkit-backdrop-filter` 的支持；不支持时会降级为极浅暖色半透明遮罩。
- **验证情况**：
  - 已用 `node --check --input-type=module` 解析 `pages/index/components/random-pick-sheet.vue` 的 `<script>` 块，通过。
  - 已执行 `git diff --check -- pages/index/components/random-pick-sheet.vue pages/index/components/random-pick-sheet.scss`，通过。
  - 未运行 HBuilderX 或微信开发者工具真机预览；建议真机确认：① 点击 `帮我选` 后图片为长图；② 标签、菜名、摘要位于图片底部且可读；③ 左侧 `了解做法` 跳详情、右侧 `换一个` 重新抽取；④ 弹层后方页面信息可隐约看到但被明显模糊。

## 2026-04-30 (美食库搜索空态与列表标题微调)

### Changed

- **修改时间**：2026-04-30 15:27:26 +0800 CST
- **变更背景**：搜索无结果态需要按截图方向简化，减少说明文字和次动作；列表标题区的“正餐 · xx 道 · 帮我选”字号偏小，且与条件卡片 / 菜品列表之间留白不足。
- **核心改动**：
  - `pages/index/components/library-empty-state.vue` 支持 `primaryIconSrc`，搜索无结果 CTA 可使用项目内 SVG 图标而非 `up-icon`。
  - `pages/index/components/library-empty-state.scss` 为 `search-no-results` 单独收敛空态视觉：隐藏描述文案、缩小主按钮高度、降低标题权重，保持暖白卡片但更接近轻量搜索结果页。
  - `pages/index/index.vue` 搜索无结果文案改为 `库里没找到“xxx”相关的菜谱`，主动作改为 `问问 AI 怎么做`，取消“清除搜索”次动作；点击后打开饮食管家，并把搜索词预填成“想做这道菜”的提问。
  - 新增 `static/icons/sparkle-plus-warm.svg`，用于空态按钮的暖金色星闪图标；底部中间 AI 入口继续使用原白色 `sparkle-plus.svg`。
  - `pages/index/components/library-header-section.vue` / `.scss` 调大首页顶部标题区：`美食库` 标题 `40rpx → 44rpx`、图标章 `44rpx → 48rpx`、图标 `14 → 15`；右侧 `安排菜单` 按钮文字 `23rpx → 25rpx`，按钮高度、内边距和日历图标同步增大。
  - `pages/index/index.vue` 列表 caption 区字号上调：标题 `23rpx → 28rpx`、`帮我选` `21rpx → 24rpx`、图标 `12 → 14`；caption 顶部留白 `18rpx → 30rpx`、菜品列表顶部留白 `14rpx → 24rpx`，并移除 `帮我选` 图标外层白色底和边框。
  - `pages/index/components/diet-assistant-sheet.vue` 新增 `initialPrompt` 入参，弹层打开或入参变化时可预填输入框。
- **影响范围**：
  - 仅影响首页美食库搜索无结果态、饮食管家打开时的输入框预填、顶部标题区与列表标题区样式。
  - 不修改搜索过滤逻辑、菜品数据结构、后端接口、饮食管家流式接口或底部 AI 入口视觉。
- **兼容性/风险**：
  - 搜索无结果不再直接提供“添加这道菜 / 清除搜索”按钮；清除仍可通过搜索框右侧关闭按钮完成，新增菜仍可通过底部入口或饮食管家内记录入口完成。
  - `primaryIconSrc` 为可选入参，既有空态仍走原 `up-icon`，不影响其它空态类型。
  - `initialPrompt` 只在非流式回复中预填输入框；正在回复时不会覆盖用户当前会话。
- **验证情况**：
  - 已用 `node --check --input-type=module` 解析 `pages/index/index.vue`、`pages/index/components/library-empty-state.vue`、`pages/index/components/diet-assistant-sheet.vue` 的 `<script>` 块，通过。
  - 已执行 `git diff --check -- pages/index/index.vue pages/index/components/library-empty-state.vue pages/index/components/library-empty-state.scss pages/index/components/diet-assistant-sheet.vue static/icons/sparkle-plus-warm.svg`，通过。
  - 未运行 HBuilderX 或微信开发者工具真机预览；建议真机确认：① 搜索不存在关键词时空态为单按钮轻量样式；② 点击 `问问 AI 怎么做` 打开饮食管家且输入框预填关键词问题；③ `帮我选` 图标无白底、标题区字号与留白符合预期。

## 2026-04-30 (关于我们页面原型化重构)

### Changed

- **修改时间**：2026-04-30 14:50:47 +0800 CST
- **变更背景**：关于我们页需要参考原型图重构 UI，从旧的备案/设置信息卡改为更接近小程序移动端的品牌介绍页。
- **核心改动**：
  - `pages/about/index.vue` 重构为自定义顶部栏、居中品牌图标章、版本号、`产品的初心` 内容卡、功能列表和底部 `MADE WITH LOVE`。
  - 功能列表新增 `功能介绍与使用指南`、`反馈与吐槽`、`备案信息` 三个入口；管理员仍可看到 `应用设置` 入口。
  - 备案信息改为列表项触发 `uni.showActionSheet`，支持复制备案编号或备案查询网址。
  - 保留顶部标题长按触发的隐藏调试能力：清空首页图片缓存。
  - `pages.json` 将 `pages/about/index` 调整为 `navigationStyle: custom`，避免系统导航栏与页面内原型顶部栏重复。
- **追加调整**：
  - **修改时间**：2026-04-30 14:52:15 +0800 CST
  - `pages/about/index.vue` 的品牌标题与 `产品的初心` 卡片标题切换为宋体/衬线 fallback 字体栈，增强标题层级；导航标题与列表标题继续使用系统无衬线，保证移动端可读性。
  - **修改时间**：2026-04-30 14:57:03 +0800 CST
  - `pages/about/index.vue` 读取微信小程序右上角胶囊位置，动态调整自定义导航顶部高度与右侧安全间距，避免关闭按钮被系统胶囊遮挡；顶部 `关于我们` 标题同步切换为宋体/衬线 fallback 字体栈。
  - `产品的初心` 文案替换为“共享厨房”方向的新表述；功能列表收敛为单行标题样式，去掉每项下方说明，保持截图里的轻量列表观感。
  - **修改时间**：2026-04-30 15:00:11 +0800 CST
  - 移除页面内左上 `关于我们` 和右上自定义关闭按钮，保留系统胶囊下方安全留白；品牌爱心图标从双层阴影图标章简化为单层浅陶土色圆角底。
  - **修改时间**：2026-04-30 15:02:23 +0800 CST
  - `产品的初心` 正文中 `共享厨房`、`今天吃什么，要买什么菜` 两个关键词改为局部加粗和深色强调，保持正文行高不变。
  - **修改时间**：2026-04-30 15:04:07 +0800 CST
  - 上述两个正文强调词颜色从深棕调整为深陶土棕 `#8f4f42`，和页面爱心/陶土色体系保持一致。
  - **修改时间**：2026-04-30 15:13:04 +0800 CST
  - `pages/about/index.vue` 左上角新增轻量返回图标按钮，触控区为 `88rpx`；微信环境下按系统胶囊高度垂直对齐，点击优先 `uni.navigateBack()`，无上一页时回首页。
  - **修改时间**：2026-04-30 15:15:03 +0800 CST
  - 返回按钮去掉圆形背景、边框和阴影，保留裸图标视觉与 `88rpx` 触控区。
- **影响范围**：
  - 仅影响 `pages/about/index.vue` 的页面结构、样式与轻交互，以及该页面的导航栏渲染方式。
  - 不修改备案配置字段、应用设置接口、首页关于我们入口路由、后端接口。
- **兼容性/风险**：
  - 页面内自定义关闭按钮已移除；左上返回按钮只承担返回上一页/回首页能力，页面仍保留系统胶囊下方安全留白。
  - 页面使用 `var(--status-bar-height, 0px)` 和 `env(safe-area-inset-bottom)` 适配自定义导航与底部安全区；仍建议微信开发者工具和真机确认不同设备顶部留白。
  - `Version 1.0.0` 目前与 `package.json` 版本保持一致，未引入运行时读取 `package.json`，避免小程序构建兼容风险。
- **验证情况**：
  - 已用 `node --check --input-type=module` 解析 `pages/about/index.vue` 的 `<script>` 块，通过。
  - 已执行 `git diff --check -- CHANGELOG.md pages/about/index.vue pages.json`，通过。
  - 未运行 HBuilderX 或微信开发者工具真机预览；建议真机确认：① 页面顶部自定义标题和关闭按钮位置正确；② 品牌区、初心卡、功能列表贴近原型；③ 备案信息复制动作可用；④ 管理员账号仍显示应用设置入口；⑤ 长按标题仍能打开清空图片缓存动作。

## 2026-04-30 (空间页退出当前空间 · V1)

### Added

- **修改时间**：2026-04-30 14:35:04 +0800 CST
- **变更背景**：空间页缺少成员自行退出空间能力；本轮按“页脚底部轻量红色胶囊按钮 + 二次确认”的简版方案落地，优先满足非创建者退出当前空间的 V1 闭环。
- **核心改动**：
  - 后端新增 `DELETE /api/kitchens/{kitchenID}/members/me`：当前登录用户退出指定空间。
  - `backend/internal/kitchen/repository.go` 新增 `Leave()` 事务：读取成员角色、拒绝 `owner` 退出、删除当前用户成员关系、将该用户在该空间创建的 `active` 邀请置为 `revoked`、更新空间 `updated_at`。
  - `backend/internal/kitchen/service.go` / `handler.go` / `model.go` 串接 `LeaveCurrentMember`，返回用户退出后的空间列表与下一当前空间 ID。
  - `utils/kitchen-api.js` 新增 `leaveKitchen(kitchenId)`。
  - `pages/index/components/kitchen-section.vue` / `.scss` 新增页脚轻量红色胶囊按钮：左侧退出箭头图标、浅红微渐变背景、`退出当前空间` 文案、`退出中...` loading 态与按压反馈。
  - `pages/index/index.vue` 新增 `canLeaveCurrentKitchen`、`isLeavingKitchen`、`confirmLeaveCurrentKitchen()`、`leaveCurrentKitchen()`；仅 `member/admin` 展示退出入口，点击后 `uni.showModal` 二次确认，确认时轻触感反馈，成功后刷新 session / 菜谱 / 成员列表。
  - `README.md` 与 `backend/README.md` 同步新增自退出接口口径，并将后续待办收敛为“管理员移除成员 / 角色调整”。
- **追加调整**：
  - **修改时间**：2026-04-30 14:38:41 +0800 CST
  - `pages/index/index.vue` 空间页不再渲染底部 `关于我们` 入口；`pages/about/index.vue` 路由和美食库页底部入口保留。
- **影响范围**：
  - 前端空间页底部、空间成员状态刷新、当前空间切换后的菜谱刷新。
  - 后端空间成员关系与邀请状态；退出用户在该空间创建的仍有效邀请码会被撤销。
  - 不修改管理员移除成员、角色转让、空间删除、邀请列表 UI。
- **兼容性/风险**：
  - 创建者暂不允许退出，避免 `kitchens.owner_user_id` 指向非成员导致数据不一致；前端默认隐藏创建者退出入口，后端仍以 `409` 兜底。
  - 非创建者退出后如没有其他空间，后续 `/api/auth/me` 会按现有登录会话逻辑补默认空间；前端会刷新 session 并切到可用空间。
  - 退出入口使用浅红胶囊按钮和二次确认，不作为主按钮展示，降低误触概率。
- **验证情况**：
  - 已执行 `go test ./internal/kitchen`，通过。
  - 已执行 `go test ./...`，通过。
  - 已执行 `git diff --check`，通过。
  - 已用 `node --check --input-type=module` 解析 `pages/index/index.vue`、`pages/index/components/kitchen-section.vue` 的 `<script>` 块，通过。
  - 未运行 HBuilderX 或微信开发者工具真机预览；建议真机确认：① member/admin 空间页底部展示轻量红色胶囊按钮；② owner 不展示退出入口；③ 点击后弹窗二次确认；④ 确认后退出并切到其他空间或默认空间；⑤ 退出者之前发出的邀请码失效。

## 2026-04-30 (邀请成员邀请码字体微调)

### Changed

- **修改时间**：2026-04-30 14:04:07 +0800 CST
- **变更背景**：邀请成员弹层的邀请码字符原使用 `SF Mono / Menlo / monospace`，观感偏代码片段；本轮按“邀请码/票据编号”方向微调字体，使分享给成员时更贴合当前暖白空间 UI。
- **核心改动**：
  - `pages/index/components/invite-sheet.scss` 新增 `%invite-code-font` 字体占位，统一邀请码展示与输入框：`DIN Alternate`、`Avenir Next`、`Helvetica Neue`、`PingFang SC`、`sans-serif`。
  - `.invite-sheet__code` 字号 `34rpx → 36rpx`、字重 `700 → 800`、字距 `3rpx → 2.5rpx`，并补 `line-height: 1.12`，让字符更稳、更像编号。
  - `.invite-code-sheet__input` 同步换字体栈与字重 / 字距，保持用户输入和展示样式一致。
  - `pages/invite/index.vue` 的邀请预览页摘要邀请码同步换为同一字体栈，并保留 `tabular-nums` 数字对齐策略。
- **影响范围**：
  - 邀请成员弹层的邀请码展示卡、输入邀请码弹层输入框、邀请预览页摘要中的邀请码样式。
  - 不修改 `formatInviteCode()` 分组规则、邀请接口、分享路径、分享封面、复制/重新生成/加入逻辑。
- **兼容性/风险**：
  - `DIN Alternate`、`Avenir Next` 在部分设备可能不可用，会回退到 `Helvetica Neue` / `PingFang SC` / `sans-serif`；内容仍可读。
  - `font-variant-numeric` / `font-feature-settings` 在小程序宿主不支持时会被忽略，不影响主样式。
- **验证情况**：
  - 已执行 `git diff --check`，通过。
  - 已用 `rg` 确认邀请码相关样式中不再残留 `SF Mono` / `Menlo` / `monospace`。
  - 未运行 HBuilderX 或微信开发者工具真机预览；建议后续打开“空间 → 邀请成员”和“已有邀请码？去加入”确认 iOS / Android 字体 fallback 观感。

## 2026-04-30 (美食库跨模块 polish · Phase E)

### Added

- `pages/index/index.vue` token 块新增 `--color-accent-terracotta: #bf715f`：陶土橙 token 入仓，给"想吃 / 固顶徽"色调系统化提供色源。后续若加"想吃"色按钮等扩展点也直接复用，避免再 hard-code。

### Changed

- **修改时间**：2026-04-30 CST
- **背景**：按 `docs/food-library-ui-redesign-plan-2026-04-30.md` v2 §5.1 / §5.5 / §5.6 / §6 阶段 E（P2 可选）补四件跨模块 polish：从详情页返回美食库的上下文聚焦、菜单 spotlight 点击反馈、菜卡固顶书签色调统一到陶土橙、菜卡点击详情前的动态反馈。
- **核心改动**：
  - **E1 详情页返回上下文聚焦**（`pages/index/index.vue` + `recipe-card-item.*`）：
    - 按 `$ui-ux-pro-max` 的移动端微交互建议，把原整页 `library-shell` 返回动效降级为“只反馈刚才打开的菜卡”，避免整页位移或透明度变化被感知成闪屏。
    - 新增 data `returnFocusRecipeId` / `returnFocusPendingRecipeId` / `returnFocusTimer`，`openRecipeDetail()` 记录待聚焦菜卡，`onShow()` 延迟 `120ms` 触发，`1160ms` 后自动退场；`onUnload` 和离开美食库时清理 timer。
    - `recipe-card-item.vue` 新增 prop `isReturnFocus`，根节点 class 增 `recipe-card--return-focus`。
    - `recipe-card-item.scss` 新增 `.recipe-card--return-focus`：只做陶土暖色描边、柔和 halo、浅背景与封面阴影变化，不再移动整页、不改 `opacity`。
  - **E2 spotlight tap 反馈**（`pages/index/components/library-header-section.vue` + `library-header-section.scss`）：
    - 把模板的 `@tap="$emit('spotlight-tap')"` 改成 `@tap="handleSpotlightTap"`，本地先播一段 `220ms` 反馈再 emit 跳转。
    - 新增 data `tapPulseActive` / `tapPulseTimer`、method `handleSpotlightTap()`：节流式锁防重复点击；`160ms` 后 emit `spotlight-tap`，再 `60ms` 解锁。`beforeUnmount` 清 timer。
    - 模板 class 加 `'meal-order-spotlight--tap-pulse': tapPulseActive`。
    - SCSS 新增 `.meal-order-spotlight--tap-pulse` + `@keyframes meal-order-spotlight-tap-pulse`：`scale(1) → scale(0.98) → scale(0.985)` + `opacity 1 → 0.95 → 0.62`（淡出收尾），`220ms cubic-bezier(0.32, 0, 0.32, 1)`。`prefers-reduced-motion` 下 `animation: none` 仅保留 `opacity: 0.85` 弱反馈。
  - **E3 菜卡固顶书签双色渐变**（`pages/index/components/recipe-card-item.scss`）：
    - `.recipe-card::before`（书签条）背景从原本金棕单色三段渐变改为陶土橙双色：`linear-gradient(180deg, #d68a76 0%, var(--color-accent-terracotta, #bf715f) 50%, #a55a4a 100%)`。
    - 阴影 `rgba(133, 99, 54, 0.08) → rgba(133, 65, 54, 0.12)`、内部高光 `rgba(255, 247, 232, 0.42) → rgba(255, 232, 222, 0.46)`，整体偏向"想吃"陶土系。
    - `.recipe-card--pinned` 边框 / 阴影也同步调到陶土系（`rgba(186, 145, 81, 0.12) → rgba(191, 113, 95, 0.18)`、`rgba(98, 74, 44, 0.06) → rgba(133, 65, 54, 0.06)`），让固顶卡片整体色调与书签呼应。
  - **E4 菜卡 tap launch 反馈**（`pages/index/components/recipe-card-item.vue` + `.scss`）：
    - 根节点新增 `touchstart` / `touchmove` / `touchend` / `touchcancel` 处理，触摸开始立即给当前菜卡 `recipe-card--tap-press` 轻压缩，滑动列表时马上取消按压态；状态 switch 与点菜模式"加入菜单"区域阻止触摸冒泡，避免触发整卡打开反馈。
    - `handleCardTap()` 改为先播放 `recipe-card--tap-pulse`，`156ms` 后 emit `open` 进入详情；`360ms` 后清理暖光态，防止快速连点重复跳转。
    - SCSS 新增整卡 `::after` 陶土暖光扫过、浅 halo、边框 / 阴影强化；`prefers-reduced-motion` 下禁用扫光动画，只保留低强度静态反馈。
- **影响范围**：
  - `pages/index/index.vue`：token 增 `--color-accent-terracotta`、保留静态 `.library-shell` wrapper、data + method + onShow / onUnload 钩、`openRecipeDetail` pending 标记。
  - `pages/index/components/recipe-card-item.vue` / `.scss`：新增返回聚焦 prop 与 `recipe-card--return-focus` 视觉态；新增 `tapPressActive` / `tapPulseActive` 状态、触摸事件处理与 `recipe-card--tap-press` / `recipe-card--tap-pulse` 点击反馈。
  - `pages/index/components/library-header-section.vue`：data + method + beforeUnmount；模板 tap 处理由 emit 改为本地。
  - `pages/index/components/library-header-section.scss`：新增 tap-pulse class + keyframe + reduced-motion。
  - `pages/index/components/recipe-card-item.scss`：`.recipe-card::before` 渐变改色、阴影 / 高光微调，`.recipe-card--pinned` 边框 / 阴影同步陶土系；`.recipe-card::after` 用于 tap 暖光扫过，不参与布局。
  - 不修改 `recipe-store.js` 数据归一化、后端接口、邀请 / 上传链路、点菜模式判断。
- **兼容性/风险**：
  - 入场动画 wrapper 从 `<template>` 升级为 `<view>` 引入一层 DOM 节点。`.page-content` 仅用 padding，子级 SCSS 没有依赖直系子元素选择器（都是 descendant），`.page-content--meal-order .toolbar` 等仍能正常匹配，无视觉副作用。
  - 详情页返回不再做整页动效；局部菜卡高亮更克制，但仍能提示“回到刚才打开的菜”。
  - spotlight tap 加了 `160ms` 延迟再跳转。`tapPulseActive` 锁防止用户连点导致多次跳转；超时 `220ms` 内组件被 `v-if` 销毁时 `beforeUnmount` 会清 timer，无内存泄漏。
  - 菜卡点击新增 `156ms` 跳转延迟，用来让 `touchstart` 压感和暖光反馈可见；延迟低于常见可感知卡顿阈值，且 `tapPulseActive` 锁防止快速连点重复进入详情。
  - 陶土双色渐变只是局部视觉换色，无逻辑影响；`var(--color-accent-terracotta, #bf715f)` 带 fallback 值，即便未来 token 块被误删也不会破图。
- **验证情况**：
  - 已执行 `git diff --check`，通过。
  - 已用 `node --check --input-type=module` 解析改动相关 `.vue` 文件的 `<script>` 块（`pages/index/index.vue`、`library-header-section.vue`、`recipe-card-item.vue`、`library-empty-state.vue`），通过；本轮新增 tap 逻辑后已重新解析 `recipe-card-item.vue`。
  - 已确认 `pages/index/components/library-header-section.vue` 使用 Vue 3 `beforeUnmount` 清理 tap timer，未残留旧版销毁钩子。
  - 未运行 HBuilderX 真机预览或微信开发者工具构建；建议后续真机补测：① 详情页返回首页时整页稳定，刚才打开的菜卡出现暖色聚焦并自然退场；② 点击菜卡先轻压缩 + 暖光扫过再进入详情；③ 点击 spotlight 先收缩淡出再进入菜单详情；④ 左右滑 spotlight 后不会误触进入详情；⑤ 固顶菜卡书签色调与陶土橙一致。
- **后续待办（P2 / P3）**：
  - 列表 stagger（§5.2）首屏 6 张菜卡延迟从 `Math.min(index, 4) * 36ms` 改为 `Math.min(index, 6) * 32ms` —— 文档 §5.2 仍标 P1，可与首页性能 polish 一起做。
  - 状态 pill thumb 滑动（§5.3）—— 已经在 Phase B 第二轮 prototype rebuild 时去掉容器 + thumb 设计，文档 §5.3 不再适用，待文档下次维护时同步更新。

## 2026-04-30 (美食库菜卡微调 + 空态主 / 次动作组件化 · Phase D)

### Added

- `pages/index/components/library-empty-state.vue` + `library-empty-state.scss`：美食库专属的空态组件，承载图标章 + 标题 + 描述 + 主 / 次动作四区结构。
  - 主动作：`80rpx` 高、深棕渐变（`linear-gradient(180deg, --color-brand-brown 0%, #4a3a2c 100%)`）+ 内描边高光、`scale(0.96)` 按压反馈。
  - 次动作：`56rpx` 高暖灰胶囊（`var(--color-surface-warm)` 背景 + `var(--color-border-soft)` 边框），按压 `translateY(1rpx)` 轻反馈。
  - 仅依赖 token，不引入新色值。组件 `kind` 入参支持 `search-no-results` / `meal-empty` / `status-empty` 三类，分别决定图标与色调。
- `pages/index/recipe-card.js` 新增 `RECIPE_LIST_SUMMARY_PLACEHOLDER` 常量与 `pickFirstParsedStep` / `pickFirstNonEmptySummary` 私有 helper：菜卡摘要按 `summary → ingredient → note → 解析步骤第一条` 优先级回退，全部为空时落到占位文本 `"还没有备注，点开补一笔～"`，并通过 `card.listSummaryIsPlaceholder` 标记是否为占位。

### Changed

- **修改时间**：2026-04-30 CST
- **背景**：按 `docs/food-library-ui-redesign-plan-2026-04-30.md` v2 §4.5 / §4.6 落地阶段 D，把菜卡来源徽 / 状态 switch 触控扩面 / 摘要兜底三处微调，加上空态从"光秃图标 + 文字"升级到"主 / 次动作"组件化。
- **核心改动**：
  - `pages/index/index.vue` token 块新增 `--color-text-muted: #9f9387`，给摘要占位 / 空态描述提供弱文色源。
  - `pages/index/components/recipe-card-item.scss`：
    - `.recipe-card__source-badge` 背景 `rgba(39,31,25,0.52) → rgba(31,22,16,0.66)`，边框 `1px rgba(255,248,240,0.14) → 1rpx rgba(255,248,240,0.18)`，提升小红书 / B 站 / 链接来源徽在浅色封面上的对比度。
    - 新增 `.recipe-switch-shell`：`padding: 6rpx; margin: -6rpx;`，把状态 switch 的可点区域从 `52rpx` 扩到 `64rpx`，同时通过负 margin 抵消让外观保持原位。
    - `.recipe-switch` 高度 `52rpx → 56rpx`，`.recipe-switch__thumb` 尺寸 `46×46rpx → 50×50rpx`，对应 `--done` 态 `translateX(64rpx) → translateX(60rpx)` 与按压态 `translateX(64rpx) scale(0.95) → translateX(60rpx) scale(0.95)`，保证 thumb 在 done 槽位仍贴右内边距 4rpx。
    - 新增 `.recipe-card__summary--placeholder`：`color: var(--color-text-muted); font-style: italic`，承接 `card.listSummaryIsPlaceholder` 时的弱提示视觉。
  - `pages/index/components/recipe-card-item.vue`：
    - 状态 switch 模板从单层 `.recipe-switch` 重构为 `.recipe-switch-shell > .recipe-switch`，`@tap.stop="$emit('toggle-status')"` 移到外层 shell。
    - 摘要 `.recipe-card__summary` 增加 `:class="{ 'recipe-card__summary--placeholder': card.listSummaryIsPlaceholder }"`。
  - `pages/index/recipe-card.js`：`buildRecipeListSummary` 改为四级回退；`buildRecipeCard` 新增 `listSummaryIsPlaceholder` 字段，避免上层重复跑 fallback 逻辑。
  - `pages/index/index.vue`：
    - 模板把原 `<view class="empty-state"><up-icon ...>{{ title }}{{ desc }}</view>` 替换为 `<library-empty-state :kind :title :description :primary-text :primary-icon :secondary-text @primary @secondary />`。
    - 新增四个计算属性：`emptyStateKind`（推断 search / status / meal 三态）、`emptyStatePrimaryText`、`emptyStatePrimaryIcon`、`emptyStateSecondaryText`；`emptyStateDesc` 文案微调，把原来的"点中间的加号新增一道菜"改为"点下方按钮新增一道菜"，与组件主动作呼应。
    - 新增两个方法：`handleEmptyStatePrimary` / `handleEmptyStateSecondary`：搜索无结果 → 主"添加这道菜"调 `openAddSheet()`，次"清除搜索"清空 `searchKeyword`；状态空 → 主"查看全部"切到 `all`，次"添加这道菜"调 `openAddSheet()`；餐别空 → 主"添加这道菜"调 `openAddSheet()`，无次动作。`presetTitle` 预填关键词作为 P1 跟进，本轮先调通。
    - 删除原 `.empty-state` / `.empty-state__title` / `.empty-state__desc` SCSS 规则；同步删除 `.soft-empty` / `.soft-empty__text` / `.soft-empty--inline`（这三个在 `index.vue` 模板里没有使用点，是历史 dead code；其他用到 `.soft-empty` 的组件如 `kitchen-section.vue` / `meal-order-cart-sheet.vue` 都各自带 scoped 样式，不受影响）。
- **影响范围**：
  - `pages/index/components/library-empty-state.vue`、`library-empty-state.scss`（新建）。
  - `pages/index/recipe-card.js`：摘要 fallback + 占位标记字段。
  - `pages/index/components/recipe-card-item.vue`、`recipe-card-item.scss`：来源徽 / switch 触控 / switch 几何 / 摘要占位四处。
  - `pages/index/index.vue`：token 增 `--color-text-muted`、import 新组件、模板替换、computed 增 4 / methods 增 2、SCSS 清理。
  - 不修改 `recipe-store.js`、后端接口、邀请链路、点菜模式判断；点菜模式下 `<library-empty-state>` 仍只在 `filteredRecipes.length === 0` 时显示（与改前一致）。
- **兼容性/风险**：
  - `recipe-switch-shell` 通过负 margin 抵消 6rpx padding，外观保持原大小；如果未来把 `.recipe-card__top` 的 gap 调小，可能让 shell 与左侧标题区视觉贴边，目前 gap 12rpx 仍有富余。
  - thumb `translateX(60rpx)` 是按"track 内宽 110rpx + thumb 50rpx + 左右 padding 4rpx" 反算的，未来若改 track 宽度需同步重算；已在注释代码处通过几何对齐保留。
  - 摘要占位文本走 `font-style: italic`，鸿蒙部分系统对中文斜体回退到正体，是降级容忍可接受。
  - 同步失败 / 离线状态的空态在本轮未单独建模（`recipe-store.js` 未暴露独立 `syncStatus` 信号），保留组件 `kind` 入参可扩展空间，将来补上 `sync-error` 仅需新增一支分支。
- **验证情况**：
  - `grep` 校验：① `pages/index/index.vue` 已无 `.empty-state` / `.empty-state__title` / `.empty-state__desc` / `.soft-empty*` 残留 SCSS；模板上 `library-empty-state` 与所有 prop / 事件名一一对应。② `recipe-card-item.vue` 的 `.recipe-switch-shell` 包了原 `.recipe-switch`，`@tap.stop` 移动到 shell 上，inner switch 不再绑事件。③ `recipe-card.js` 的 `buildRecipeCard` 输出 `listSummaryIsPlaceholder` 与模板上的 `:class` 名一致。④ `recipe-card-item.scss` 的 `.recipe-switch--done` thumb `translateX(60rpx)` 与按压态对齐。
  - 未运行 HBuilderX 真机预览或微信开发者工具构建；建议下一轮真机走 §8：① 搜索 "番茄炒蛋" 等不存在关键词 → 显示"没有找到 xxx" + "添加这道菜" 主 + "清除搜索" 次；② 切到只有想吃但没有吃过的餐别 + 想吃 → 全部空（验证仅在某些数据状态下能复现，可手动改 status 触发）；③ 全空餐别 → 主"添加这道菜"，无次动作；④ 来源徽（B 站 / 小红书）在浅奶油封面上对比度更高；⑤ 状态 switch 触控边缘 6rpx 内点击仍能切换；⑥ 没有 summary / ingredient / note 也没有解析步骤的菜卡显示斜体占位"还没有备注，点开补一笔～"。
- **后续动作**：
  - Phase E（可选）：详情页返回入场动效、spotlight tap `scale(0.98) → 淡出` 反馈、固顶卡书签双色渐变。
  - P1：`openAddSheet` 接受 `presetTitle` 入参，让"添加这道菜"在搜索无结果时把关键词预填到新增弹层。

## 2026-04-30 (菜单 spotlight 升级为日历计划卡 · Phase C)

### Changed

- **修改时间**：2026-04-30 CST
- **背景**：按 `docs/food-library-ui-redesign-plan-2026-04-30.md` v2 §4.2 / §5.5 落地阶段 C，把美食库顶部的 `meal-order-spotlight` 从单行卡改为参考原型 `PlanCard`（`我们的美食空间/src/components/PlanCard.tsx`）的"日历图标章 + 衬线日期 + 周几 + 状态 chip + 进度 chip + 箭头"四区结构。
- **核心改动**：
  - `pages/index/meal-order.js` 新增 `formatMealOrderDateParts(value)`，返回 `{ dateText, weekday, isoDate }`，让上游可以分别渲染日期主体与周几（原 `formatMealOrderDateText` 保留，老调用点未改）。
  - `pages/index/index.vue` 引入新工具方法，把 `mealOrderSpotlightTitle` 拆为四个 computed：`mealOrderSpotlightDateText`（如 `04月18日`）、`mealOrderSpotlightWeekday`（如 `周六`）、`mealOrderSpotlightStatusText`（`已安排` / `草稿`）、`mealOrderSpotlightStatusKind`（`submitted` / `draft`）。`mealOrderSpotlightDesc` / `mealOrderSpotlightMetaText` 维持原契约。
  - `<library-header-section>` 调用点同步换掉旧 `:meal-order-spotlight-title`，新增 4 个 prop 绑定。
  - `pages/index/components/library-header-section.vue` 模板把 spotlight 拆为四块：
    - `.meal-order-spotlight__icon-mark`：56rpx 方块、橙色描边 + 内填浅奶白渐变，承载 `up-icon name="calendar"`。
    - `.meal-order-spotlight__main`：第一行 `.meal-order-spotlight__heading`（衬线日期 32rpx · 700 + 周几 22rpx · 600 + 状态 chip），第二行 `.meal-order-spotlight__desc`（前置 `·` 装饰，单行省略）。
    - `.meal-order-spotlight__aside`：竖排，顶部 `.meal-order-spotlight__progress` 衬线进度 chip（`1/3` 样式，多记录时才显示），底部 arrow icon。
    - 状态 chip 区分两种 kind：`submitted` 用陶土橙底 (`rgba(191, 113, 95, 0.10/0.18)`)、`draft` 用中性棕底 (`rgba(122, 102, 85, 0.10/0.18)`)。
  - `pages/index/components/library-header-section.scss` `meal-order-spotlight` 重写：
    - 容器圆角 `22rpx → 28rpx`、padding `18rpx 20rpx → 20rpx 22rpx`，背景从 `radial + rgba(255,250,244,0.94)` 改为 `radial + linear-gradient(145deg, #fffdf8 0%, #f4ecdf 100%)` 双层；边框换 `var(--color-border-soft)`；阴影沿用 clay-soft 深度。
    - 删除旧 `.meal-order-spotlight__title`、`.meal-order-spotlight__meta-text`，新增 `.meal-order-spotlight__icon-mark` / `__heading` / `__date` / `__weekday` / `__chip(__-submitted/--draft)` / `__chip-text` / `__progress` / `__progress-text` 9 条规则。
    - 衬线字体仅用于 `__date` 与 `__progress-text`：`font-family: "Songti SC", "STKaiti", "DejaVu Serif", serif`，符合 v2 §3.3 关于"刊物感数字"的限定使用。
    - `meal-order-spotlight-slide-next/previous` keyframe：`translateX 16rpx → 20rpx`，新增 `scale(0.98) → 1`，时长 `220ms → 240ms`，对应 v2 §5.5 的"翻书感"。
    - `spotlightMotionKey` 改为引用 `mealOrderSpotlightDateText`（旧 key 用 title，title 已下线）。
- **影响范围**：
  - `pages/index/meal-order.js`：新增导出函数；不动旧导出。
  - `pages/index/index.vue`：模板（spotlight prop 绑定）、computed（4 个新计算属性 + 删除 `mealOrderSpotlightTitle`）、import 新增 `formatMealOrderDateParts`。
  - `pages/index/components/library-header-section.vue`：模板 + props（删除 `mealOrderSpotlightTitle`，新增 4 个）。
  - `pages/index/components/library-header-section.scss`：spotlight 区块重写 + slide keyframe 调整。
  - 不修改 `recipe-store.js`、`recipe-card-item.*`、`pages/index/index.vue` 的工具卡 / 状态 pill / 菜卡逻辑、后端接口、邀请链路、点菜模式判断。
- **兼容性/风险**：
  - `meal-order-spotlight` 仅在 `!isLibraryMealOrderMode && hasMealOrderSpotlightRecord` 时渲染，新 prop 在无 record 时全为空字符串，`v-if` 防止空文字渲染。
  - 衬线字体（Songti SC / STKaiti）在部分鸿蒙设备 fallback 不一致，最终回落到 `serif`，仅日期数字 / 进度 chip 受影响，内容仍可读。
  - 状态 chip 的 `已安排 / 草稿` 文案绑定 `record.type === 'submitted'`；草稿（`draft`）使用中性棕底以避免和 `submitted` 抢眼。
  - `mealOrderSpotlightTitle` 仅在原 prop 链路使用，下线后无外部调用点（`grep` 已确认零残留）。
- **验证情况**：
  - `grep` 校验：① 三处文件均无 `mealOrderSpotlightTitle` / `meal-order-spotlight-title` / `meal-order-spotlight__title` / `meal-order-spotlight__meta-text` 残留；② 4 个新 prop 在 `index.vue` 模板 / `index.vue` computed / `library-header-section.vue` props / `library-header-section.vue` 模板四处一一对应。
  - 未运行 HBuilderX 真机预览或微信开发者工具构建；建议下一轮真机走 §8：① 顶部 spotlight 显示日历图标章 + 衬线日期 + 周几 + 状态 chip + 进度 chip + 箭头六要素全齐；② 切换多记录时左右滑动有 `translateX(20rpx) + scale(0.98) → 1` 翻书感；③ 单条 record 时进度 chip 不渲染，arrow 仍贴右下；④ submitted / draft 两种 kind 状态 chip 颜色区分明显；⑤ 点菜模式下 spotlight 不渲染（与改前一致）。
- **后续动作**：
  - Phase D：菜卡来源徽对比度 / 触控扩面 / 摘要兜底 + 空态主 / 次动作组件化（`recipe-card-item.scss`、`recipe-card.js` 摘要兜底、新建 `library-empty-state.vue`）。

## 2026-04-30 (美食库工具卡 / 状态 pill 视觉升级 · Phase B)

### Changed

- **修改时间**：2026-04-30 CST
- **背景**：按 `docs/food-library-ui-redesign-plan-2026-04-30.md` v2 §4.3 / §5.3 / §5.4 落地阶段 B（工具卡 + 状态 pill 视觉升级），是 Phase A token 抽取后第一波带视觉变化的 PR。同期把 `--color-surface` / `--color-surface-warm` / `--shadow-clay-soft` / `--shadow-clay-strong` 四个 token 挂上 `page` 选择器，方便 Phase C / D 复用。
- **核心改动**：
  - `pages/index/index.vue` token 块扩展到 10 个变量：新增 `--color-surface: #fffdf8`、`--color-surface-warm: #f4ecdf`、`--shadow-clay-soft`、`--shadow-clay-strong`；其中 `--color-surface` 与 `--shadow-clay-strong` 立即被工具卡使用，是 Phase B 的视觉切换点。
  - 工具卡 `.toolbar`：圆角 `22rpx → 32rpx`，padding `18rpx → 24rpx`，背景从 `rgba(255,255,255,0.86)` 换为 `var(--color-surface)`（`#fffdf8` 实色），边框从 `rgba(0,0,0,0.03)` 换为 `var(--color-border-soft)`，阴影从 `0 8rpx 20rpx rgba(56,44,30,0.04)` 换为 `var(--shadow-clay-strong)`，整体更"暖白立体"。`.page-content--meal-order .toolbar` override 保留（点菜模式仍是透明）。
  - 工具卡新增 `.toolbar--bounce-left` / `.toolbar--bounce-right` keyframe（140ms ease-out、最大位移 8rpx），由餐别 tab 切换时通过新增方法 `triggerToolbarBounce` 触发，给"内容已切"做轻反馈。
  - 搜索框 `.search-box`：高度 `68rpx → 76rpx`，圆角 `18rpx → 22rpx`，padding `0 18rpx → 0 18rpx 0 12rpx`，新增 `.search-box__icon-shell`（40rpx 圆形、内填 `rgba(91, 74, 59, 0.06)`），原裸 `up-icon` 套进图标章；输入区高度同步 `68rpx → 76rpx`。
  - 状态 pill 行重构：`.status-track` 从"3 个独立 pill + 10rpx gap"改为"分段控件容器 + 单 thumb 平移"。
    - `.status-track` 加 `padding: 4rpx`、`border-radius: 18rpx`、轻底色 `rgba(91, 74, 59, 0.04)` + token 边框；移除 gap。
    - 新增 `.status-track__thumb`：绝对定位、`width: calc((100% - 8rpx) / 3)`、`height: calc(100% - 8rpx)`、`transition: transform 0.22s cubic-bezier(0.2, 0.8, 0.2, 1)`。
    - 通过 `.status-track--all/--wishlist/--done .status-track__thumb` 切换 `transform: translateX(0/100%/200%)` 与背景渐变（沿用原"中性棕 / 想吃棕 / 鼠尾草绿"三色）。
    - `.status-pill` 自身去掉非激活态背景 / 激活态背景与 box-shadow（统一交给 thumb），保留按状态轻染 icon-shell 与 text 颜色（让未激活态依然能看出 wishlist / done 区分）。
    - 新增 `.status-pill__count` + `.status-pill__count-text`（18rpx · 600，未激活 `rgba(91, 74, 59, 0.06)` 浅底，激活态 `rgba(255, 255, 255, 0.18)` 浅白），数据由新增 `statusCount(status)` 方法计算（在当前 `activeMealType` 下分别统计 all / wishlist / done）。
  - 模板侧：`.toolbar` 加 `:class="toolbarBounceClass"`；状态 track 加 `:class="status-track--${activeStatus}"` 和 `<view class="status-track__thumb"></view>`；每个 pill 内加 count chip。
  - 数据 / 方法：`data()` 新增 `toolbarBounceClass: ''`、`toolbarBounceTimer: null`；`methods` 新增 `triggerToolbarBounce(direction)` 与 `statusCount(status)`；`handleMealTypeTabChange` 在切换前先按 tab 顺序判断方向并触发 bounce；`onUnload` 清理 `toolbarBounceTimer`。
- **影响范围**：
  - `pages/index/index.vue`：模板（搜索图标章、状态 thumb、count chip、toolbar bounce class）、`<style>` 块（toolbar / search-box / status-track / status-pill 重写）、`data()` / `methods` / `onUnload`。
  - 不修改 `pages/index/components/library-header-section.*`、`pages/index/components/recipe-card-item.*`、`recipe-store.js`、后端接口、邀请链路、点菜模式判断。
- **兼容性/风险**：
  - 点菜模式下 `.page-content--meal-order .toolbar` 把背景 / 边框 / 阴影 / padding / 圆角全部重置为透明 / 0，所以工具卡视觉升级不影响点菜模式头部。
  - `.status-track` 的 thumb 用 `transform: translateX(N * 100%)` 实现平移，依赖三个 pill 平均分宽（`flex: 1`）+ 容器固定 `padding: 4rpx`；不同设备宽度下不会错位。
  - count chip 数据由 `recipes` 数组实时过滤计算；菜谱量大（>500）时再考虑做缓存优化。
  - 餐别 tab 切换的 bounce 动画放在 `triggerToolbarBounce` 里，使用 `$nextTick` + `setTimeout(160)`，避免连点时 class 没卸下来再加回去；`onUnload` 已清理 timer。
  - `statusCount('all')` 等于"当前餐别 + 任意状态"的总数，与 `mealTypeCount` 在同一上下文下保持一致。
- **验证情况**：
  - 改后 `grep` 校验：旧的 `.status-pill--{status}` bg gradient、`.status-pill--active { box-shadow }`、`.status-pill--{x}.status-pill--active { bg/border }` 已全部删除；模板上的 `:class="status-track--${activeStatus}"` 与 CSS 上的 `.status-track--all/--wishlist/--done` 名称一一对应；`@keyframes toolbar-bounce-left/right` 与 `.toolbar--bounce-left/right` 名称一一对应；`statusCount`、`triggerToolbarBounce`、`toolbarBounceClass` 在模板 / data / methods 三处可联通。
  - 未运行 HBuilderX 真机预览或微信开发者工具构建；建议下一轮真机走 §8：① 切换早餐 / 正餐时工具卡轻位移并回弹；② 切换全部 / 想吃 / 吃过时 thumb 在三个槽位之间顺滑平移；③ 三个 pill 显示各自数量徽标，激活态徽标背景变浅白；④ 工具卡圆角 32rpx、阴影感明显比改前更立体；⑤ 搜索框图标章背景圆形可见；⑥ 点菜模式工具卡仍透明、不下沉。
- **后续动作**：
  - Phase C：菜单 spotlight 升级为日历计划卡（`library-header-section.vue` + `.scss`）。
  - Phase D：菜卡来源徽对比度 / 触控扩面 / 摘要兜底 + 空态主 / 次动作组件化。

## 2026-04-30 (美食库设计 token 抽取 · Phase A · 零视觉差异)

### Changed

- **修改时间**：2026-04-30 CST
- **背景**：按 `docs/food-library-ui-redesign-plan-2026-04-30.md`（v2 视觉与动效改造方案）阶段 A 落地设计 token，为后续 Phase B（工具卡 / 状态 pill 视觉升级）、Phase C（spotlight → PlanCard）、Phase D（菜卡 + 空态重做）做共享色板基础。本阶段只做"重构 + 视觉零差异"的别名替换，不调整任何具体取值。
- **核心改动**：
  - `pages/index/index.vue` 顶部新增一段非 scoped `<style>` 块，于 `page` 选择器声明 6 个 token 变量：`--color-bg`、`--color-text-primary`、`--color-text-on-brand`、`--color-brand-brown`、`--color-border-soft`、`--color-border-active`。token 值取**当前实际色值**（`#f6f4f1` / `#2f2923` / `#fffaf3` / `#5b4a3b` / `rgba(91, 74, 59, 0.07)` / `rgba(91, 74, 59, 0.16)`），而非 v2 §3.1 文档列出的目标值（`#F6F2EA`、`#5C4033` 等），避免本阶段引入视觉漂移；目标值的切换归 Phase B+。
  - 替换 `pages/index/index.vue` 内 13 处 hard-code 色值为 `var(--token)`：`.app-shell` 背景、`.search-box` 边框 / `--active` 边框 / 输入色、`.page-content--meal-order .search-box` 边框、`.status-pill--wishlist.status-pill--active` 渐变末端 + 边框、`.status-pill--active .status-pill__text`、`.empty-state__title` / `.meal-panel__title` / `.stat-box__value` / `.simple-panel__title` / `.simple-list__title` 标题色等。
  - 替换 `pages/index/components/library-header-section.scss` 内 3 处：`.page-header__title`、`.meal-order-spotlight__title`、`.meal-order-mode-bar__chip` 边框。
  - 替换 `pages/index/components/recipe-card-item.scss` 内 5 处：`.recipe-card--active` 边框、`.recipe-card__title`、`.recipe-switch` 边框、`.meal-order-add__text`、`.meal-order-add__text--active`。
- **影响范围**：
  - `pages/index/index.vue`（新增 token 块 + 风格层 token 引用）
  - `pages/index/components/library-header-section.scss`
  - `pages/index/components/recipe-card-item.scss`
  - 不修改后端接口、`recipe-store.js` 数据归一化、邀请链路、点菜模式判断逻辑。
  - CSS 变量挂在 `page` 选择器上，子组件 scoped 样式通过 CSS 自定义属性继承自动取值，无须额外 `@import`。
- **兼容性/风险**：
  - 微信小程序原生支持 CSS 变量，子组件可通过 `var(--token)` 跨 scoped 边界取值。
  - 仅做完全相等的色值替换（`#2f2923` → `--color-text-primary` 等），渲染结果应与改前像素级一致；建议改后用微信开发者工具或真机首屏对比一次确认。
  - 模板内 JS 表达式中的颜色字符串（如 `pages/index/index.vue:105` 的 `:color="..."`）保持原样，因为 CSS 变量不能在 JS 上下文求值；这部分留待后续如有需要再以计算属性承接。
- **验证情况**：
  - 替换全部使用 `Edit` 工具的精确字符串匹配完成；改后 `grep` 三个文件对应 hard-code pattern 已无残留（除 token 声明本身）。
  - 未运行 HBuilderX 真机预览或微信开发者工具构建；建议下一轮真机走 §8 视觉清单（首屏一屏五要素是否齐、菜卡颜色是否未漂移）后再开 Phase B。
- **后续动作**：
  - Phase B（工具卡圆角 22→32rpx、阴影升级、状态 pill 数量徽 + thumb 平移、餐别 tab 切换轻位移）。
  - Phase C（spotlight → PlanCard 化）。
  - Phase D（菜卡微调 + 空态重做）。
  - Phase E（可选 polish）。

## 2026-04-30 (饮食管家后端切换为单轮 user-only 上行)

### Changed

- **修改时间**：2026-04-30 00:41 CST
- **背景**：根据 2026-04-29 对 `dots-ai` Chat Completions 的验证结论，上游对 `system` 角色和多轮历史的处理不稳定（`system` 单独发送会 401、`system + user` 不能稳定遵循约束、显式多轮历史会超时或返回旧上下文残留），继续按原先"system 提示 + 截断历史"的方式上行已不再可靠。
- **核心改动**：
  - `backend/internal/dietassistant/service.go` 移除 `systemPrompt` 常量与 `buildUpstreamMessages` / `trimHistory`，改用 `buildSingleTurnUpstreamMessages`：仅取消息列表中最后一条 `role=user` 的消息上行，不再附带 `system` 提示，也不再向上游发送任何 `assistant` 历史。
  - 找不到可用 `user` 消息时直接返回 `400 user message is required`，避免空请求被打到上游。
  - 上行 payload 增加 `user: "new"` 字段（OpenAI 兼容协议的用户标识），用于辅助上游区分会话/降低串上下文概率。
  - 流式消费链路与 `emit` 回调未变。
- **影响范围**：
  - `backend/internal/dietassistant/service.go`
  - 改变了饮食管家与上游 AI 的对话契约：从"system + 多轮 user/assistant 历史"变为"单条 user 消息 + `user=new` 标识"。
  - 不修改前端 `utils/diet-assistant-api.js`、`pages/index/components/diet-assistant-sheet.vue`，也不动数据库、配置开关或部署脚本。
- **兼容性/风险**：
  - 饮食管家不再具备多轮对话上下文记忆能力，每条消息都是独立请求；如需"参考上一轮"的回复，需要用户在新一条消息里自己复述。
  - 不再注入身份系统提示，理论上模型可能回到通用助手语气；但根据前一日验证，原 `system` 提示在该上游也未能稳定生效，影响有限。
  - 与现有 `pages/index/components/diet-assistant-sheet.vue` 仍兼容：前端依然按完整消息列表上送，后端会自行截取最后一条 `user`。
- **验证情况**：
  - 已执行 `go build ./...` 通过编译。
  - 已执行 `git diff --check` 基础检查通过。
  - 未运行微信开发者工具或真机预览；建议后续补测连续两次提问是否各自独立、空消息或纯 assistant 消息列表是否正确返回 400。

## 2026-04-29 (验证 dots-ai Chat Completions 消息角色行为)

### Changed

- **修改时间**：2026-04-29 23:43 CST
- **背景**：用户要求确认 `https://www.gxm1227.top/dots2api/v1/chat/completions` 的 `messages` 是否实际支持 `system` 角色，还是只能使用 `user`。
- **核心结论**：
  - `system + user` 报文会被上游接受并返回 `200 OK`，说明接口层不是简单禁止 `system` 角色。
  - `system` 单独作为唯一消息时返回 `401 unauthorized`，该上游至少不适合作为无 `user` 消息的标准 Chat Completions 入口使用。
  - 使用强约束标记测试时，`system` 提示没有被稳定遵守；后续串行 `user` 强约束与普通问题也出现返回上一轮 `USER_OK` 内容的现象，说明当前 `dots-ai` 上游对 `messages` 内容存在明显不可靠或串上下文风险。
  - 补测多轮上下文时，显式携带 `user -> assistant -> user` 历史的非流式请求超时；同类流式请求返回 `401 unauthorized`，后续单轮流式请求也出现 `200 OK` 后无正文直到超时。
  - 为排除并发调用干扰，已单独重试一次显式上下文请求；上游返回 `200 OK`，但内容为旧测试相关的 `pong / USER_OK`，未回答当前请求中要求记住的“山竹”。
  - 单独发送 `user: 小红书 美食` 时，上游返回 `200 OK` 并生成“小红书美食风向标”类内容及多张图片代理链接，但正文开头仍带旧测试残留的 `USER_OK`。
  - 当前不能把该上游视为严格遵循 OpenAI Chat Completions 多角色语义的模型；饮食管家若继续使用该 Provider，应降低对 `system` 的依赖，并优先排查上游是否存在状态复用、缓存或代理转发实现问题。
- **影响范围**：
  - 饮食管家上游 AI 联调口径。
  - 不修改小程序前端、后端代码、配置文件、数据库或部署脚本。
- **兼容性/风险**：
  - 本次未记录真实 API Key；验证请求消耗少量上游调用额度。
  - 上游行为可能随 Provider 实现调整而变化，后续切换模型或代理后需重新验证。
- **验证情况**：
  - 已用脱敏 API Key 对 `dots-ai` 执行非流式 `system + user`、仅 `user`、仅 `system`、`developer + user` 和普通数学问题请求。
  - 已用真实项目上游形态执行 `stream: true` 请求，确认 `system + user` 可返回 SSE，但回复内容未稳定遵守饮食管家系统提示。
  - 已补测显式多轮上下文与单轮流式请求，未能得到可用、稳定的上下文响应。
  - 已按用户要求避免并发，只保留单次上下文请求重试；结果仍未能证明上游支持稳定上下文。
  - 已单独测试 `小红书 美食` 查询，确认该上游可生成图文类回答，但仍存在旧上下文残留。

## 2026-04-29 (修复饮食管家第二条消息无法发送)

### Fixed

- **修改时间**：2026-04-29 23:35 CST
- **背景**：用户反馈饮食管家首条消息发送后，第二条消息发不出去；从代码看，发送入口会在 `isStreaming=true` 时直接返回，而流式状态释放依赖 `Promise.prototype.finally()`，在部分微信小程序运行时可能不稳定。
- **核心改动**：
  - `pages/index/components/diet-assistant-sheet.vue` 将流式请求完成后的状态清理由 `.finally()` 改为 `then(success, failure)` 内显式调用，避免运行时缺少 `Promise.finally` 时 `isStreaming` 无法复位。
  - 保持原有成功完成、失败提示、手动中止和清空会话逻辑不变。
- **影响范围**：
  - `pages/index/components/diet-assistant-sheet.vue`
  - 仅影响饮食管家聊天输入框的流式状态释放；不改变后端接口、请求体、会话上下文或 UI 视觉结构。
- **兼容性/风险**：
  - 低。改动移除对较新 Promise API 的依赖，使用更基础的 Promise `then` 形式，兼容性更稳。
- **验证情况**：
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析并编译 `pages/index/components/diet-assistant-sheet.vue`，检查通过。
  - 已执行 `git diff --check`，基础 diff 检查通过。
  - 本次未运行微信开发者工具或真机预览；建议补测连续发送两条消息、第一条失败后再发第二条、流式中清空会话。

## 2026-04-29 (饮食管家新增清空会话入口)

### Added

- **修改时间**：2026-04-29 23:29 CST
- **背景**：用户希望在饮食管家对话框下方增加灰色文字入口，点击后可以清空当前会话记录。
- **核心改动**：
  - `pages/index/components/diet-assistant-sheet.vue` 在输入框下方新增“清空会话记录”文字入口，仅当当前已有真实对话消息时显示。
  - 点击清空时会先中止正在进行的流式回复，再清空 `localMessages`、重置流式状态，并让聊天窗恢复初始示例态。
  - `pages/index/components/diet-assistant-sheet.scss` 为清空入口增加灰色低优先级样式和轻微按压反馈。
- **影响范围**：
  - `pages/index/components/diet-assistant-sheet.vue`
  - `pages/index/components/diet-assistant-sheet.scss`
  - 仅影响饮食管家聊天抽屉的本地会话展示与前端内存上下文，不涉及后端接口、数据库、登录态或菜谱保存链路。
- **兼容性/风险**：
  - 当前会话记录本来只存在页面内存中，清空不会删除任何服务端数据。
  - 清空入口不做二次确认，存在误触后当前页面内对话无法恢复的低风险。
- **验证情况**：
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue`，SFC 结构检查通过。
  - 已执行 `git diff --check`，基础 diff 检查通过。
  - 本次未运行微信开发者工具或真机预览；建议补测有会话时入口显示、点击后恢复初始示例态、流式中点击可中止并清空。

## 2026-04-29 (修正饮食管家流式前端状态与异常上下文)

### Fixed

- **修改时间**：2026-04-29 23:16 CST
- **背景**：饮食管家流式接口已连通后，截图中仍出现旧 `请求失败(500)` 气泡、重试后回复带有“反代测试成功 / 服务正常 / 业务验证”等联调话术，且输入框可能在已显示回复后仍停留在“饮食管家正在回复...”状态。
- **核心改动**：
  - `utils/diet-assistant-api.js` 在解析到 SSE `done` 事件时立即 resolve 流式请求，不再只等待 `uni.request success`，降低真机上回复已完成但 UI 仍处于流式中的概率。
  - `pages/index/components/diet-assistant-sheet.vue` 为同一次用户消息与助手占位增加 `requestID`，接口失败或用户中止时将该轮请求排除出后续模型上下文，避免重试时把失败前的用户输入重复发送给后端。
  - 饮食管家后端系统提示补充约束，禁止在用户回复中提到反代、代理、接口、测试成功、服务正常、业务验证、模型提供商或部署状态。
- **影响范围**：
  - `utils/diet-assistant-api.js`
  - `pages/index/components/diet-assistant-sheet.vue`
  - `backend/internal/dietassistant/service.go`
  - 仅影响饮食管家聊天流式状态收口、失败重试上下文和模型输出口径；不改变接口路径、鉴权、请求体结构、数据库或其他业务链路。
- **兼容性/风险**：
  - 低。前端失败气泡仍保留在当前界面用于反馈用户，但不再作为有效助手上下文继续传给模型。
  - 上游模型仍可能偶发不遵守提示词，本次通过系统提示降低联调话术进入产品回复的概率。
- **验证情况**：
  - `node --check utils/diet-assistant-api.js` 通过。
  - 使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/components/diet-assistant-sheet.vue` 通过。
  - `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache GOCACHE=/tmp/caipu-go-build-cache go test ./...` 通过。
  - 本次未运行微信开发者工具或真机预览；建议上线/预览后重点复测失败后重试、正常流式完成后输入框状态恢复，以及简单问候不再输出联调话术。

## 2026-04-29 (修复饮食管家流式接口 500)

### Fixed

- **修改时间**：2026-04-29 23:10 CST
- **背景**：用户在小程序饮食管家输入“你好呀”后，前端提示 `请求失败(500)`；生产日志显示 `POST /api/diet-assistant/chat/stream` 在 `2026-04-29 23:05:55 CST` 以 `0ms` 返回 `500`。
- **核心改动**：
  - 定位原因是 `backend/internal/middleware/logger.go` 的 `statusRecorder` 包装了 `http.ResponseWriter`，但没有透传 `http.Flusher`；饮食管家 handler 判断流式响应不受支持后直接返回 `500`，请求尚未进入上游 AI 调用。
  - 为 `statusRecorder` 增加 `Flush()` 透传，保留底层 `ResponseWriter` 的 SSE 刷新能力。
  - 新增 `backend/internal/middleware/logger_test.go`，覆盖 `RequestLogger` 包装后仍支持 `http.Flusher` 的回归场景。
- **影响范围**：
  - `backend/internal/middleware/logger.go`
  - `backend/internal/middleware/logger_test.go`
  - 影响所有依赖 `http.Flusher` 的后端流式响应，当前主要是饮食管家 SSE 接口；不改变接口路径、请求体、鉴权策略、数据库结构或前端调用口径。
- **兼容性/风险**：
  - 低。修复仅补齐中间件对标准流式接口的能力透传，普通 JSON 接口状态记录逻辑不变。
  - 上游 AI 返回内容质量仍取决于 `DIET_ASSISTANT_AI_*` 当前配置的代理与模型，本次修复只解决后端本地 `500`。
- **验证情况**：
  - `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache GOCACHE=/tmp/caipu-go-build-cache go test ./...` 通过。
  - `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache GOCACHE=/tmp/caipu-go-build-cache go build -o bin/server ./cmd/server` 通过。
  - `systemctl restart caipu-backend` 已完成，服务于 `2026-04-29 23:08:10 CST` 重启为 `MainPID=259672`。
  - `curl -fsS http://127.0.0.1:8080/healthz` 与 `curl -fsS http://127.0.0.1:8080/api/healthz` 均返回 `code=0`。
  - 使用临时本机 JWT 对 `POST /api/diet-assistant/chat/stream` 发送“你好呀”，接口返回 `200 OK`，并收到 SSE `delta` 与 `done` 事件。

## 2026-04-29 (生产启用饮食管家 AI 配置)

### Changed

- **修改时间**：2026-04-29 23:04 CST
- **背景**：用户要求将饮食管家上游 AI 配置写入生产后端配置文件，并重启后端服务，使新接入的流式聊天接口可在线上使用。
- **核心改动**：
  - 在未纳入 Git 的 `backend/configs/prod.env` 中配置 `DIET_ASSISTANT_AI_BASE_URL`、`DIET_ASSISTANT_AI_API_KEY`、`DIET_ASSISTANT_AI_MODEL`、`DIET_ASSISTANT_AI_TIMEOUT_SECONDS`；变更记录不记录真实 API Key。
  - 首次仅重启后端后发现线上 `backend/bin/server` 仍是 2026-04-29 16:16 构建版本，`POST /api/diet-assistant/chat/stream` 返回 `404`，说明当前二进制尚未包含饮食管家路由。
  - 已使用当前仓库代码重新构建 `backend/bin/server`，并再次重启 `caipu-backend`，使饮食管家路由在生产服务中注册生效。
- **影响范围**：
  - 生产后端运行配置：`backend/configs/prod.env`
  - 生产后端二进制：`backend/bin/server`
  - systemd 服务：`caipu-backend`
  - 影响饮食管家 `POST /api/diet-assistant/chat/stream` 接口；不改变小程序前端代码、后台前端、数据库结构或其他业务接口契约。
- **兼容性/风险**：
  - `DIET_ASSISTANT_AI_API_KEY` 仅保存在生产配置文件中，未写入 Git 跟踪文件。
  - 已确认接口路由从 `404` 变为未登录时的 `401`，表示路由已生效且仍受登录态保护；本次未使用真实用户 token 调用上游，避免产生真实 AI token 消耗。
- **验证情况**：
  - `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache GOCACHE=/tmp/caipu-go-build-cache go test ./...` 通过。
  - `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache GOCACHE=/tmp/caipu-go-build-cache go build -o bin/server ./cmd/server` 通过。
  - `systemctl restart caipu-backend` 已完成，服务于 `2026-04-29 23:04:00 CST` 重启为 `MainPID=258622`。
  - `curl -fsS http://127.0.0.1:8080/healthz` 与 `curl -fsS http://127.0.0.1:8080/api/healthz` 均返回 `code=0`。
  - 本机无 token 探测 `POST /api/diet-assistant/chat/stream` 返回 `401 authorization header is required`，确认接口已注册并由登录态保护。

## 2026-04-29 (饮食管家流式 AI 接口与多轮对话接通)

### Added

- **修改时间**：2026-04-29 CST
- **背景**：用户希望后端新增一个通过 OpenAI Chat Completions 协议调用 `dots-ai` 的接口，并让小程序饮食管家聊天窗口先接通该接口，实现流式输出和多轮对话。
- **核心改动**：
  - 新增 `backend/internal/dietassistant/` 模块，提供受登录态保护的 `POST /api/diet-assistant/chat/stream` SSE 流式接口。
  - 后端接口向上游 OpenAI-compatible `/chat/completions` 发起 `stream: true` 请求，并把增量内容转换为本项目统一的 SSE 事件：`delta`、`done`、`error`。
  - 新增配置项 `DIET_ASSISTANT_AI_BASE_URL`、`DIET_ASSISTANT_AI_API_KEY`、`DIET_ASSISTANT_AI_MODEL`、`DIET_ASSISTANT_AI_TIMEOUT_SECONDS`；默认 baseURL 为 `https://www.gxm1227.top/dots2api/v1`，默认 model 为 `dots-ai`，API Key 只读环境变量，不写入仓库。
  - `backend/internal/app/router.go` 为饮食管家流式接口增加 3 分钟超时覆盖，避免默认 30 秒中断长回复。
  - 新增 `utils/diet-assistant-api.js`，使用 `uni.request({ enableChunked: true })` 处理小程序 chunk 流；若运行环境没有 chunk 回调，会在请求结束后尝试解析完整 SSE 文本。
  - `pages/index/components/diet-assistant-sheet.vue` 从本地假回复切换为真实流式回复：发送用户消息后创建助手占位气泡，逐段追加 delta 文本，并把历史 `user / assistant` 消息带给后端实现多轮上下文。
  - 前端流式解析新增 UTF-8 跨 chunk 解码兜底，降低中文增量片段在低版本运行时乱码或 SSE JSON 帧解析失败的概率。
  - 手动停止回复或接口失败产生的临时提示不再进入下一轮多轮上下文，避免把错误提示误当作助手回答继续发送给模型。
- **影响范围**：
  - `backend/internal/dietassistant/*`
  - `backend/internal/config/config.go`
  - `backend/internal/app/app.go`
  - `backend/internal/app/router.go`
  - `backend/configs/example.env`
  - `backend/README.md`
  - `utils/diet-assistant-api.js`
  - `pages/index/components/diet-assistant-sheet.vue`
  - `pages/index/components/diet-assistant-sheet.scss`
  - 影响饮食管家聊天链路；不改变现有菜谱保存、链接解析、点菜模式、邀请链路或后台 AI Provider 配置。
- **兼容性/风险**：
  - 线上必须配置 `DIET_ASSISTANT_AI_API_KEY` 后接口才可用；缺失时后端会通过 SSE 返回配置错误事件。
  - 小程序流式能力依赖微信运行时 `RequestTask.onChunkReceived`；已提供完整 SSE 文本解析兜底，但不同基础库和 nginx 缓冲策略仍需真机验证。
  - 前端已补充手写 UTF-8 解码兜底，但流式实时性仍取决于微信基础库、真机网络和网关是否关闭响应缓冲。
  - 当前接口只负责聊天建议，不会自动保存菜谱、自动解析外部链接或自动安排菜单；这些动作仍需用户走现有入口。
- **验证情况**：
  - `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./...` 通过。
  - `node --check utils/diet-assistant-api.js` 通过。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/index.vue` 与 `pages/index/components/diet-assistant-sheet.vue`，SFC 结构检查通过。
  - 已对 `pages/index/components/diet-assistant-sheet.vue` 的组件脚本做替换导入后的 `new Function` 语法检查，通过。
  - `git diff --check` 通过；已做敏感关键字扫描，确认未把用户提供的 API Key 或完整 curl 写入仓库。
  - 本次未把用户提供的 API Key 写入仓库，也未直接调用线上上游做真实 token 消耗测试；上线前需在运行环境配置 `DIET_ASSISTANT_AI_API_KEY` 后补测真机流式输出。

## 2026-04-29 (饮食管家聊天窗口 UI 初版)

### Changed

- **修改时间**：2026-04-29 CST
- **背景**：用户希望参考 `我们的美食空间/src/components/AIChatSheet.tsx` 中底部中心入口弹出的“饮食管家”UI，并结合 `ui-ux-pro-max` 的移动端聊天与触控建议，把当前小程序首页底部中心入口先改造成聊天窗口形态；当前阶段暂不接入 AI，只实现 UI 界面。
- **核心改动**：
  - 新增 `pages/index/components/diet-assistant-sheet.vue` 与 `diet-assistant-sheet.scss`，实现底部 80vh 聊天抽屉、饮食管家头部、欢迎气泡、模拟用户消息、模拟菜谱灵感卡、快捷建议、底部输入栏和本地发送占位反馈。
  - 饮食管家弹窗遮罩单独配置暖棕半透明背景与 `backdrop-filter` / `-webkit-backdrop-filter`，让弹窗背后的首页呈现玻璃虚化效果，不影响其他 `up-popup` 弹层。
  - 参考原型中 `饮食管家` 标题旁的 `brand-terracotta/10` 圆形 Lucide `Sparkles` 图标，新增浅红圆形 logo 视觉，并同步用于聊天管家头像。
  - 将底部中心 FAB 的 `static/icons/sparkle-plus.svg` 同步为原型同款白色线性 Lucide `Sparkles` 图标，不再使用原来的自定义星闪图形。
  - 新增 `static/icons/chat-send.svg` 作为聊天发送按钮图标，使用 Lucide `Send` 线性路径并做轻微左移，修正纸飞机图标视觉偏右的问题；`static/icons/diet-assistant-logo.svg` 用于饮食管家标题和聊天头像。
  - `pages/index/index.vue` 将底部中心 FAB 从直接打开 `add-recipe-sheet` 改为打开 `diet-assistant-sheet`。
  - 为避免丢失原有添加菜品链路，聊天窗口内保留“记录菜谱 / 先用现有表单加入美食库”入口，点击后关闭饮食管家并复用现有 `openAddSheet` 逻辑，不改变添加菜品表单、链接预填、图片上传或保存契约。
  - 聊天输入当前只做本地 UI：发送后追加用户消息和“暂未接入 AI”的本地占位回复，不发起后端请求，也不消耗 AI 调用。
- **影响范围**：
  - `pages/index/index.vue`
  - `pages/index/components/diet-assistant-sheet.vue`
  - `pages/index/components/diet-assistant-sheet.scss`
  - `static/icons/diet-assistant-logo.svg`
  - `static/icons/chat-send.svg`
  - `static/icons/sparkle-plus.svg`
  - 仅影响首页底部中心入口与新增饮食管家聊天抽屉 UI，不涉及后端接口、菜谱数据结构、点菜模式数据、邀请链路或现有添加菜品提交逻辑。
- **兼容性/风险**：
  - 首页中心 FAB 的首层语义从“添加菜品”变为“饮食管家”，用户新增菜品需要在聊天窗中再点“记录菜谱”；该入口已在顶部建议和消息卡片中保留，但操作路径比原先多一步。
  - 新增聊天抽屉与遮罩使用 `backdrop-filter`、渐变、阴影等视觉增强；低版本微信环境若弱化这些效果，仍有暖白背景、半透明遮罩和边框兜底。
  - 当前为 UI 占位，不应被理解为已具备真实 AI 推荐、链接解析或菜谱自动写入能力。
- **验证情况**：
  - 已运行 `python3 .codex/skills/ui-ux-pro-max/scripts/search.py "food wellness mobile assistant chat window elegant warm minimal" --design-system -p "Caipu Diet Assistant"` 获取设计系统建议，并补充移动触控与 Vue 栈建议。
  - 已使用 `admin-web/node_modules/@vue/compiler-sfc` 解析 `pages/index/index.vue` 与 `pages/index/components/diet-assistant-sheet.vue`，SFC 结构检查通过。
  - 已将 `pages/index/index.vue` 的 `<script>` 内容抽取到 `/tmp/caipu-index-script-check.mjs` 并执行 `node --check`，语法检查通过。
  - 已对 `pages/index/components/diet-assistant-sheet.vue` 的组件脚本做 `new Function` 语法检查，通过。
  - 已执行 `git diff --check`，基础 diff 检查通过。
  - 本次未运行微信开发者工具或真机预览，建议补测：首页底部中心 FAB 打开聊天窗、关闭抽屉、快捷建议填入输入框、本地发送占位回复、聊天窗内“记录菜谱”能继续打开原添加菜品表单，以及安全区/键盘遮挡表现。

## 2026-04-29 (美食库模块 UI 优化评估文档)

### Added

- **修改时间**：2026-04-29 CST
- **背景**：用户希望继续优化小程序 `美食库` 模块整体 UI，要求参考 `我们的美食空间/` 原型目录中较优秀的视觉风格，并结合 `ui-ux-pro-max` 的设计建议输出一份改动评估文档。
- **核心改动**：
  - 新增 `docs/food-library-ui-optimization-report-2026-04-29.md`，聚焦 `美食库` 模块本体，区别于此前“美食库与空间”总方案。
  - 文档对比 `我们的美食空间/src/App.tsx`、`RecipeCard.tsx`、`PlanCard.tsx`、`SpaceView.tsx` 与当前 `pages/index/index.vue`、`library-header-section`、`recipe-card-item`、`pages/index/recipe-card.js` 的 UI 差异。
  - 文档吸收 `ui-ux-pro-max` 中 `Marketplace / Directory`、`Nature Distilled`、`Bento Grids`、移动触控和搜索无结果建议，明确小程序侧不照搬 React/Tailwind/Lucide/Google Fonts 依赖。
  - 文档给出 `P0 / P1` 改造建议：搜索筛选面板组件化、状态筛选数量反馈、空态动作、菜谱卡触控与摘要兜底、菜单 spotlight 计划卡化、后续 AI 助手入口边界等。
- **影响范围**：
  - `docs/food-library-ui-optimization-report-2026-04-29.md`
  - 当前仅新增 UI 设计优化评估文档，不修改小程序运行时代码、后端接口、点菜模式、邀请链路、菜谱保存契约或构建配置。
- **兼容性/风险**：
  - 本次为文档沉淀，无运行时兼容风险。
  - 后续若进入实现，需要重点保护 `isLibraryMealOrderMode` 分支、搜索/筛选事件链路、菜卡 `@tap.stop`、空态新增流程和菜单 spotlight 点击/滑动逻辑。
- **验证情况**：
  - 已读取并对照 `CHANGELOG.md`、`README.md`、`backend/README.md`、相关 `docs/` 文档、`我们的美食空间/` 原型源码与当前小程序美食库/空间组件实现。
  - 已运行 `ui-ux-pro-max` 本地搜索脚本生成设计系统、移动 UX、Vue 栈、色彩和风格建议。
  - 本次未运行小程序构建或微信开发者工具预览，因为未修改运行时代码。

## 2026-04-29 (修复保存菜谱 500：`recipes` INSERT 占位符缺失)

### Fixed

- **修改时间**：2026-04-29 CST
- **背景**：线上小程序点击“保存菜品”时，后端 `POST /api/kitchens/{kitchenID}/recipes` 在 `2026-04-29 09:04:11`、`09:04:13`、`09:04:16` 连续返回 `500`。排查发现问题不在登录、预览或前端提交，而是 `backend/internal/recipe/repository.go` 的 `insertRecipe` 中 `recipes` 表 `INSERT` 已扩展到 `34` 列，但 `VALUES` 仍只有 `32` 个 `?` 占位符，SQLite 执行 SQL 时直接失败。
- **核心改动**：
  - 修正 `backend/internal/recipe/repository.go` 中 `insertRecipe` 的 `VALUES` 占位符数量，使其与当前列数保持一致。
  - 新增 `backend/internal/recipe/create_test.go` 回归测试，直接覆盖 `Repository.Create` 落库路径，并断言菜谱记录可写入、厨房 `updated_at` 会同步推进。
- **影响范围**：
  - 小程序首页新增菜谱保存链路。
  - 所有走 `POST /api/kitchens/{kitchenID}/recipes` 的创建入口。
- **兼容性/风险**：
  - 低。仅修正后端 SQL 模板并补充测试，不改变接口字段、前端提交口径或数据库结构。
  - 线上需要重新发布 `caipu-backend` 后修复才会生效。
- **验证情况**：
  - `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache GOCACHE=/tmp/caipu-go-build-cache go test ./internal/recipe -run 'TestRepository(CreatePersistsRecipeAndBumpsKitchenUpdatedAt|UpdateStatusDoesNotTouchRecipeUpdatedAt)$' -count=1` 通过。
  - `cd backend && GOMODCACHE=/tmp/caipu-go-mod-cache GOCACHE=/tmp/caipu-go-build-cache go test ./internal/recipe -count=1` 通过。
  - `RUN_GIT_PULL=0 bash scripts/deploy-backend-on-server.sh` 已完成后端发布；`caipu-backend` 于 `2026-04-29 16:16:33 CST` 重启为 `MainPID=203232`。
  - `curl -fsS http://127.0.0.1:8080/healthz` 返回 `code=0`，最近发布后日志未见新的 `POST /api/kitchens/{kitchenID}/recipes` 500。

## 2026-04-29 (空间模块 UI 整体改造)

### Changed

- **修改时间**：2026-04-29 CST
- **背景**：用户确认底部 bar 调整可用后，希望继续参考 `我们的美食空间/src/components/SpaceView.tsx` 原型，改造小程序 `空间` 模块整体 UI，并允许结合 `ui-ux-pro-max` 的移动端卡片与触控建议。
- **核心改动**：
  - `pages/index/components/kitchen-section.vue` 为空间页增加 `kitchen-section` 根布局、空间大卡装饰光晕、统计卡 tone class，以及当前用户头像徽标结构。
  - `pages/index/components/kitchen-section.scss` 将空间页从偏设置列表的白卡布局，改为“当前空间仪表盘”视觉：大卡承载当前空间、半透明统计格、深色邀请成员主按钮、成员卡片化列表和更轻的加入邀请码入口。
  - 邀请成员主按钮复现原型动效：增加小程序 `hover-class` 按压态、按钮轻压缩放、图标轻缩，以及从左到右的柔光 sweep 动画，替代原型 Web 端的 `whileHover / whileTap`。
  - 邀请成员主按钮图标从 uview `share` 改为本地 `lucide Share` 等价 SVG，统一为空间模块的线性图标语言。
  - 空间名标题尝试系统中文衬线字体栈 `"Songti SC", "STSong", "SimSun", "Noto Serif SC", serif`，不引入远程字体，优先在 iOS 真机上接近原型截图的宋体标题气质。
  - 空间统计数字单独增加系统衬线数字字体栈 `"Didot", "Baskerville", "Georgia", "Times New Roman", serif`，仅作用于成员数与空间数，不影响“创建者”等中文身份文本。
  - 保留所有现有事件与数据契约：空间切换、编辑空间名、邀请成员、查看全部成员、当前用户资料编辑、邀请码加入等行为不变。
  - 设计上沿用当前暖白、品牌棕、鼠尾草绿和陶土橙 token，不照搬原型的 React/Tailwind/Lucide 依赖；`ui-ux-pro-max` 的触控建议用于保证主要点击区不低于移动端可用尺寸。
- **影响范围**：
  - `pages/index/components/kitchen-section.vue`
  - `pages/index/components/kitchen-section.scss`
  - `static/icons/invite-share.svg`
  - 仅影响小程序首页 `空间` tab 的视觉层级和触控反馈，不涉及后端接口、空间数据结构、邀请链路、成员列表字段或底部导航行为。
- **兼容性/风险**：
  - 本次增加 `backdrop-filter` 和 `filter: blur` 作为视觉增强；低版本微信环境若弱化这些效果，卡片仍保留普通渐变、边框和阴影兜底。
  - 空间名标题的系统衬线字体在 Android 真机上不保证命中；未命中时会自然回退到系统默认字体，不影响功能和可读性。
  - 成员卡头像新增徽标仅在 `member.isCurrentUser` 时展示，不改变成员点击逻辑。
- **验证情况**：
  - 已对照 `我们的美食空间/src/components/SpaceView.tsx`、`pages/index/components/kitchen-section.vue` 和 `kitchen-section.scss` 完成实现。
  - 已执行 `git diff --check`，基础 diff 检查通过。
  - 已执行 `node --check` 校验 `pages/index/components/kitchen-section.vue` 与 `pages/index/index.vue` 抽取脚本片段，语法通过。
  - 本次未运行微信开发者工具或真机预览，建议补测空间切换、邀请弹层、当前用户资料编辑、成员为空/加载中状态和底部导航遮挡情况。

## 2026-04-29 (底部 Bar UI 优化落地与评估文档)

### Changed

- **修改时间**：2026-04-29 CST
- **背景**：用户希望基于 `我们的美食空间/` 新设计原型，对比当前小程序页面代码，优化小程序首页底部 bar UI，并输出一份变动评估文档。
- **核心改动**：
  - `pages/index/index.vue` 将首页底部导航从全宽贴底栏优化为左右内收的浮动胶囊形态，加入半透明暖白背景、毛玻璃、高光边框和更轻量的 tab 图标壳。
  - 根据预览截图二次收敛左右 tab：可见图标底壳改为透明触控容器，图标尺寸统一放大，inactive 颜色调浅，激活态改为只依赖图标与文字颜色，整体更接近原型中的轻量裸图标导航。
  - `空间` 入口从 uview `grid` 图标改为本地 `lucide Grid` 等价 SVG，修正旧图标视觉偏四格的问题，并保持 active / inactive 状态色一致。
  - 底部胶囊背景进一步收窄并降低白底不透明度，阴影改为更轻的玻璃浮层效果，让菜品列表能在底部 bar 后方隐约透出。
  - 左右 tab 从底部对齐改为胶囊内垂直居中，并降低 tab 内容最小高度，减少导航内部顶部空白。
  - 中间 FAB 增加小程序原生 `hover-class="nav-fab--pressed"` 与短暂 `hover-stay-time`，修复从空间切回美食库后立即点击添加菜品时，CSS `:active` 按压动效可能被弹层渲染打断的问题。
  - 中间 `添加` FAB 继续触发现有 `openAddSheet`，仅增强凸出层级、尺寸和阴影，不跟随原型改成 AI 助手入口，避免改变当前添加菜谱工作流。
  - 同步上移点菜模式 `meal-order-floating` 结算条，并增加普通/点菜模式页面底部留白，避免新底部 bar 与列表尾部或结算浮层互相遮挡。
  - 更新 `docs/bottom-nav-ui-upgrade.md`，沉淀原型对比、已落地变动、影响范围、兼容性风险和建议补测清单。
- **影响范围**：
  - `pages/index/index.vue`
  - `static/icons/space-grid.svg`
  - `static/icons/space-grid-active.svg`
  - `docs/bottom-nav-ui-upgrade.md`
  - 仅影响小程序首页底部 bar 与点菜模式底部浮层的视觉位置，不涉及后端接口、路由配置、添加菜谱弹层契约、空间切换逻辑或邀请链路。
- **兼容性/风险**：
  - 新增 `backdrop-filter` 作为毛玻璃增强；低版本微信环境若不支持，会自然降级为半透明暖白底。
  - 本轮未改变 FAB 功能语义；后续若要切换为 AI 助手入口，需要先补齐 AI sheet 与添加菜谱入口的共存策略。
- **验证情况**：
  - 已对照 `我们的美食空间/src/components/BottomNav.tsx`、`pages/index/index.vue` 底部导航与点菜模式浮动条完成代码评估。
  - 已执行 `node --check` 校验从 `pages/index/index.vue` 抽取的脚本片段，语法通过。
  - 已执行 `git diff --check`，基础 diff 检查通过。
  - 本次未运行微信开发者工具或真机预览，建议补测安全区、低端安卓毛玻璃降级和点菜模式底部浮层间距。

## 2026-04-29 (美食库与空间 UI 重构方案文档)

### Added

- **修改时间**：2026-04-29 CST
- **背景**：用户希望基于 `我们的美食空间/` 设计原型，评估当前小程序 `美食库` 与 `空间` 两个核心界面的可优化点，并输出一份后续可直接拆任务实施的 UI 界面重构文档。
- **核心改动**：
  - 新增 `docs/food-library-space-ui-refactor-plan-2026-04-29.md`，梳理原型目录中的美食库、空间页、底部导航和 AI 辅助入口设计意图，并对照当前 `pages/index/index.vue`、`library-header-section`、`recipe-card-item`、`kitchen-section` 等实现给出落地建议。
  - 文档明确重构策略为“保留现有业务链路，重整视觉层级和组件边界”，不建议直接照搬 React Web 原型依赖。
  - 文档按 `P0 / P1 / P2` 拆分美食库搜索筛选、空态动作、菜卡状态切换、底部导航、空间概览、邀请入口、成员卡和后续 AI/协作增强等改造项。
  - 文档补充设计 token、组件拆分建议、分阶段实施计划、验收清单与风险处理，方便后续按批次进入编码。
- **影响范围**：
  - `docs/food-library-space-ui-refactor-plan-2026-04-29.md`
  - 当前仅新增 UI 重构方案文档，不修改小程序运行时代码、后端接口、点菜模式、邀请链路或构建配置。
- **兼容性/风险**：
  - 本次为文档沉淀，无运行时兼容风险。
  - 后续若按文档进入实现，需要重点保护 `isLibraryMealOrderMode` 分支、空间切换/邀请事件、菜谱状态切换和搜索筛选既有行为。
- **验证情况**：
  - 已读取并对照 `我们的美食空间/` 原型源码、`README.md`、`docs/meal-order-mode-prototype-v1.md`、`pages/index/index.vue` 与相关组件实现。
  - 本次未运行小程序构建或微信开发者工具预览，因为未修改运行时代码。

## 2026-04-25 (首页菜单详情卡片无记录时直接隐藏)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：用户反馈首页头部的菜单详情卡片在当前空间没有任何菜单安排时不应继续显示空态卡片，否则会占视觉空间，也容易让“没有菜单”和“有一张可点开的菜单卡”混在一起。
- **核心改动**：
  - `pages/index/components/library-header-section.vue` 将首页头部菜单详情卡片改为仅在存在实际菜单记录时才渲染；当前空间没有草稿或已提交菜单时，首页不再展示空态菜单卡片。
  - `pages/index/components/library-header-section.scss` 同步删除仅用于空态菜单卡片的样式分支，保持头部结构更干净。
- **影响范围**：
  - `pages/index/components/library-header-section.vue`
  - `pages/index/components/library-header-section.scss`
  - 仅影响小程序首页头部的菜单详情卡片展示条件，不改变菜单草稿/提交数据结构、菜单详情页入口或“安排菜单”主操作。
- **兼容性/风险**：
  - 当前仍保留首页右上角 `安排菜单` 入口，因此“没有菜单时隐藏详情卡片”不会影响新建菜单路径。
  - 已有菜单记录时，轮播、滑动切换和进入详情页逻辑保持不变。
- **验证情况**：
  - 已人工核对首页菜单记录计算链路：`pages/index/index.vue` 中 `mealOrderSpotlightRecords` 仅会产出有实际菜品的草稿或已提交记录。
  - 已执行 `git diff --check` 进行基础变更校验。

## 2026-04-25 (Admin Web AI Provider 页面目录与快捷键位置再收口)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：在上一轮把 `页面目录` 和 `快捷键` 都收进底部固定 bar 之后，用户继续反馈这两类能力虽然“存在”，但语义位置仍然不顺手：`页面目录` 更像页内结构导航，不该和保存/测试混在执行条里；`快捷键` 作为高频动作加速器，也应该贴近动作本身，而不是再单独做一个帮助入口。
- **核心改动**：
  - `admin-web/src/pages/AIProvidersPage.vue` 将 `页面目录` 入口从底部固定 bar 挪到顶部 `routing-breadcrumb` 右侧，保留原有目录项、高亮联动和平滑滚动逻辑，但让它回到“当前页面结构导航”的语义层，不再占用底部操作带。
  - `admin-web/src/pages/AIProvidersPage.vue` 移除底部 bar 的独立 `快捷键` 入口与首次自动弹出的快捷键 coachmark，不再使用额外弹层承载快捷键信息。
  - `admin-web/src/pages/AIProvidersPage.vue` 保留 `保存` / `测试` 按钮上的键位 tooltip，并在场景卡区域上方增加一条轻量键位提示，专门提示 `Alt/⌥ + ←/→` 场景切换，让快捷键跟真实操作落点靠得更近。
  - `admin-web/src/pages/AIProvidersPage.vue` 同步补充顶部 breadcrumb 的响应式布局和目录按钮样式，确保目录入口不会压住正文，也不会在窄屏下打散主信息层级。
- **影响范围**：
  - `admin-web/src/pages/AIProvidersPage.vue`
  - 仅影响后台 `AI Provider` 页面中页内导航入口与快捷键提示的摆放方式，不改变既有快捷键守卫、保存/测试处理链路或锚点滚动行为。
- **兼容性/风险**：
  - `页面目录` 仍只在满足“长页面条件”的桌面宽屏下显示，本轮仅调整入口位置。
  - 由于移除了首次自动弹出的快捷键引导，用户主要通过场景切换提示和保存/测试 tooltip 认知快捷键；若后续数据表明仍然不够，可再考虑更轻的首次引导方式。
- **验证情况**：
  - `cd admin-web && npm exec -- vite build` 通过。
  - 本次未做浏览器人工回归；建议补测：顶部目录入口在宽屏下是否顺手、场景切换提示是否足够可见、底部 bar 去除辅助入口后主操作是否更聚焦。

## 2026-04-25 (Admin Web AI Provider 快捷键发现性与锚点目录遮挡优化)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：在上一轮为 `AI Provider` 页面补上最小集快捷键和锚点目录后，用户进一步反馈两个实际体验问题：底部整行快捷键提示虽然“看得见”，但并不利于真正被使用；右侧固定锚点目录在部分宽屏下会压住场景卡内容，影响首屏阅读。因此需要把“功能存在”继续优化到“信息不挡内容、又更容易被发现和采用”。
- **核心改动**：
  - `admin-web/src/pages/AIProvidersPage.vue` 移除右侧固定悬浮锚点目录，改为底部固定 bar 中的 `页面目录` 入口；目录点击后以上浮面板展示，并保留当前 section 高亮与平滑滚动能力，避免首屏内容被导航浮层遮挡。
  - `admin-web/src/pages/AIProvidersPage.vue` 移除底部常驻整行快捷键文案，改为底部 bar 中的 `快捷键` 入口；弹层内展示当前页实际快捷键、适用范围和禁用条件，不再用长提示文案持续占空间。
  - 保存与测试按钮的 tooltip 现在直接带上当前平台的实际键位文案（macOS 显示 `⌘` / `⌥`，其他平台显示 `Ctrl` / `Alt`），把快捷键提示贴近真正的动作触发点。
  - 页面首次在桌面端打开时，会自动弹出一次快捷键面板作为轻量引导，后续通过 `localStorage` 记忆，避免重复打扰。
- **影响范围**：
  - `admin-web/src/pages/AIProvidersPage.vue`
  - 仅影响后台 `AI Provider` 页面里快捷键发现方式与锚点目录呈现方式，不改变既有快捷键逻辑、保存/测试契约或锚点滚动行为。
- **兼容性/风险**：
  - `页面目录` 仍只在满足“长页面条件”的桌面宽屏下显示，只是从固定浮层改成底部 bar 入口。
  - 快捷键引导依赖浏览器 `localStorage` 记录“是否已看过”；若浏览器禁用存储，最多表现为首次引导可能重复出现，不影响主功能。
- **验证情况**：
  - `cd admin-web && npm exec -- vite build` 通过。
  - 本次未做浏览器人工回归；建议补测：目录面板打开后不遮挡关键内容、首次快捷键引导只弹一次、保存/测试 tooltip 的键位文案在 macOS / Windows 口径下是否符合预期。

## 2026-04-25 (Admin Web AI Provider P1 收口：锚点目录 / 健康概览 / 告警联动 / 快捷键)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：在上一轮把完整审计、`skip link` 和风险状态条收口后，`AI Provider` 页面仍缺少四个直接影响效率和可见性的点：长页面导航、场景卡健康摘要、最近告警状态联动，以及熟练用户的最小集快捷键。本轮目标是继续沿用现有控制台结构，把这四项补齐，而不重做页面布局或保存 / 测试契约。
- **核心改动**：
  - `backend/internal/aialert/model.go`、`backend/internal/aialert/repository.go`、`backend/internal/aialert/service.go` 新增告警概览 DTO 与只读 overview 聚合能力；基于 `ai_provider_alert_states` 按 `thresholdReached DESC, updatedAt DESC` 输出状态列表，补齐 `activeAlertCount`、`latestAlertedAt` 和 `hasDeliveryConfig` 等前端所需字段。
  - `backend/internal/admin/handler.go`、`backend/internal/app/router.go`、`backend/internal/app/app.go` 新增 `GET /api/admin/ai-routing/alerts/overview` 后台接口，并把告警概览服务注入到 admin handler；`backend/internal/admin/handler_test.go`、`backend/internal/aialert/service_test.go` 补了 handler 返回结构、阈值判断、排序、投递配置完整性和 active count 的测试覆盖。
  - `admin-web/src/api/admin.ts`、`admin-web/src/types.ts` 新增 `AIRoutingAlertOverview`、`AIRoutingAlertOverviewItem` 和 `SceneCardHealthSnapshot` 类型与接口封装，供 `AIProvidersPage` 统一消费。
  - `admin-web/src/pages/AIProvidersPage.vue` 在不重排主布局的前提下新增桌面宽屏条件化锚点目录、场景卡健康概览区、顶部状态条告警短状态、场景级告警聚合展示，以及页面级快捷键守卫；同时页面会并行补齐三类场景的详情、最近测试审计和告警概览数据，让三张场景卡都能显示“最近测试 / 配置风险 / 告警状态”三类信号。
  - `docs/admin-ai-provider-ui-design-report-2026-04-24.md` 同步更新剩余状态与后续优先级，避免继续把已完成的锚点目录、健康概览、告警联动和最小集快捷键误记为“尚未完成”。
- **影响范围**：
  - `admin-web/src/pages/AIProvidersPage.vue`
  - `admin-web/src/api/admin.ts`
  - `admin-web/src/types.ts`
  - `backend/internal/aialert/*`
  - `backend/internal/admin/handler.go`
  - `backend/internal/app/router.go`
  - `backend/internal/app/app.go`
  - `docs/admin-ai-provider-ui-design-report-2026-04-24.md`
  - 仅影响后台 `AI Provider` 页面与新增的只读告警概览接口，不改变现有场景保存接口、测试接口、审计接口语义，也不影响小程序前端和正式路由配置结构。
- **兼容性/风险**：
  - 告警概览仍以 `ai_provider_alert_states` 表为唯一事实来源，不引入新的告警事件流水表；因此当前“24h 内有告警”是基于最近告警状态聚合，而不是完整事件时间线。
  - 场景卡“最近测试”继续复用 `/api/admin/runtime-settings/audits` 的 `test` 记录，不新增专门的测试摘要接口；若后续测试摘要口径变化，需要同步审计写入语义。
  - 快捷键仅在 `AIProvidersPage` 生效，并对输入框聚焦、完整审计抽屉、保存/测试进行中和 `ElMessageBox` 打开状态做了守卫，避免误触发浏览器返回或重复提交。
- **验证情况**：
  - `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/aialert ./internal/admin ./internal/app` 通过。
  - `cd admin-web && npm exec -- vite build` 通过。
  - 本次未做浏览器人工回归；建议补测：宽屏锚点点击与高亮、三张场景卡健康摘要、顶部告警状态映射、`Mod + S` / `Mod + Enter` / `Alt + ←/→` 守卫场景。

## 2026-04-25 (Admin Web AI Provider 剩余 P0 收口：审计高级筛选 / 跳到主编辑区 / 风险状态条)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：在 `docs/admin-ai-provider-ui-design-report-2026-04-24.md` 的状态化复核基础上，`AI Provider` 页面还剩三项明确的 `P0`：完整审计高级筛选、`skip link` / 跳到主编辑区，以及“阻塞 / 高风险状态必须常驻可见”的收口。本轮目标是把这三项直接做完，避免后台继续停留在“能看 diff，但排障和键盘访问还差一口气”的状态。
- **核心改动**：
  - `backend/internal/appsettings/runtime_types.go`、`backend/internal/appsettings/runtime_provider.go`、`backend/internal/admin/handler.go` 扩展运行时审计列表筛选能力，新增操作人、配置键模糊匹配和时间范围参数，同时保留分页与动作筛选；`runtime_provider_test.go` 补充对应单测覆盖。
  - `admin-web/src/pages/AIProvidersPage.vue` 的完整审计抽屉新增操作人、配置键、时间范围、page size 和已应用筛选 chips，筛选请求直接透传到后端，分页组件也按真实 `pageSize` 计算页数，不再固定按默认值展示。
  - `admin-web/src/pages/AIProvidersPage.vue` 新增键盘可达的 `skip link`，“跳到主编辑区”会直接滚动并聚焦到主编辑区容器，补齐当前页面最明显的无障碍缺口。
  - `admin-web/src/pages/AIProvidersPage.vue` 在编辑区顶部新增风险状态条，直接常驻显示：无启用节点、启用节点缺密钥、最大尝试次数高于启用节点数、最近测试异常、所有节点都在冷却、保存将清空密钥、保存将删除节点等高风险信息，不再只依赖 tooltip 或保存前弹窗。
  - 同步更新 `docs/admin-ai-provider-ui-design-report-2026-04-24.md`，将上述三项从“尚未完成 / P0”改为“已实现”，并把剩余路线收敛到锚点目录、场景卡健康概览、最近告警状态联动等后续项。
- **影响范围**：
  - `admin-web/src/pages/AIProvidersPage.vue`
  - `backend/internal/admin/handler.go`
  - `backend/internal/appsettings/runtime_types.go`
  - `backend/internal/appsettings/runtime_provider.go`
  - `backend/internal/appsettings/runtime_provider_test.go`
  - `docs/admin-ai-provider-ui-design-report-2026-04-24.md`
  - 仅影响 `admin-web` 的 AI Provider 页面与其完整审计接口的筛选能力，不涉及小程序前端、AI 路由正式配置结构、后端保存接口契约或部署配置。
- **兼容性/风险**：
  - 审计高级筛选继续复用原有 `/api/admin/runtime-settings/audits` 接口，仅新增可选 query 参数；旧调用方不传这些参数时行为不变。
  - 风险状态条只做前端聚合展示，不新增后端状态字段；“所有节点冷却中”仍基于最近一次测试结果判断，暂不代表后端实时 breaker 视图。
  - `skip link` 只增加新的快捷入口，不改变现有鼠标交互和 Tab 顺序主链路。
- **验证情况**：
  - `cd backend && GOCACHE=/tmp/caipu-go-build-cache go test ./internal/appsettings ./internal/admin` 通过。
  - `cd admin-web && npm exec -- vite build` 通过。
  - 本次未做浏览器人工回归；建议补测：完整审计筛选组合、键盘 Tab 到 `skip link` 后跳转、风险条在缺密钥 / 清空密钥 / 删除节点 / 测试异常场景下的展示。

## 2026-04-25 (Admin Web AI Provider UI 设计报告补充：实施状态复核与剩余优先级)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：在核对 `docs/admin-ai-provider-ui-design-report-2026-04-24.md` 与当前 `admin-web` 实现是否完全一致时，发现文档里仍混杂着“原始设计建议”“已落地实现”“已被后续决策替换的方案”和“尚未完成的项”，继续直接拿来验收容易产生误判，因此需要把报告整理成更适合作为后续执行检查表的状态化版本。
- **核心改动**：
  - 更新 `docs/admin-ai-provider-ui-design-report-2026-04-24.md`，新增“实施状态复核（2026-04-25）”补充章节。
  - 将文档中的建议拆分为三类：`已落地`、`被后续口径替换`、`尚未完成`，明确指出哪些设计已在当前 `AIProvidersPage` / `HelpTip` 实现中覆盖。
  - 明确标注两项已被后续决策替换的口径：
    - “测试完成后自动滚动到结果卡”已改为“顶部结果 chip 手动跳转”。
    - “Provider ID blur 校验”已改为“隐藏内部 ID，校验 Provider Name 唯一性”。
  - 将原来的“优先级路线 / 推荐实施顺序”改写为基于当前实现状态的“剩余优先级路线 / 推荐后续实施顺序”，把后续任务收敛为：完整审计高级筛选、`skip link` 或等价主内容跳转、条件化锚点目录、场景卡健康概览增强、最近告警状态联动等尚未完成项。
- **影响范围**：
  - `docs/admin-ai-provider-ui-design-report-2026-04-24.md`
  - 当前仅影响 `admin-web` AI Provider 页面设计报告的验收口径与后续任务优先级，不涉及运行时代码、后端接口或部署配置。
- **兼容性/风险**：
  - 本次为文档收口，无运行时风险。
  - 更新后文档更适合做“当前状态核对”与“后续任务排期”，但仍保留原始设计分析段落，阅读时需区分“设计建议”和“已实施状态”。
- **验证情况**：
  - 已结合当前仓库中的 `admin-web/src/pages/AIProvidersPage.vue`、`admin-web/src/components/HelpTip.vue` 与现有 `CHANGELOG` 记录完成逐项复核。
  - 本次仅更新文档和变更记录，未触发新的运行时代码变更验证。

## 2026-04-25 (菜谱详情页流程图查看链路 P0 性能改造落地)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：根据 `docs/recipe-flowchart-performance-optimization.md` 的 `P0` 范围，本轮优先解决“流程图已经生成，但详情页和横屏查看打开仍然慢”的前端查看链路问题，同时去掉流程图首屏加载时就触发的分享封面裁图，避免无分享场景下的额外耗时。
- **核心改动**：
  - `pages/recipe-detail/index.vue` 为流程图新增独立缓存状态：按 `flowchartUpdatedAt` 优先、`flowchartImageUrl` 回退的版本规则接入现有 `utils/image-cache.js`，详情页流程图展示和轻点预览都改为“本地缓存优先，远程 URL 兜底”。
  - `pages/recipe-detail/index.vue` 的流程图缓存同步改为在 `applyRecipe` 后触发，并通过 `flowchartImageCacheRequestID` 丢弃旧版本异步结果；本地缓存文件失效时自动清理并回退远程图，不再因为缓存异常把整块流程图隐藏掉。
  - `pages/recipe-detail/index.vue` 的横屏查看 payload 现在同时写入远程 `imageUrl` 和本地 `localImagePath`，与详情页流程图展示口径保持一致。
  - `pages/recipe-detail/index.vue` 移除流程图 `<image>` 首屏 `@load` 裁图逻辑，改为仅在真正构造微信分享配置且当前渠道优先使用流程图封面时，才按需执行 canvas 正方形裁图；裁图失败时静默回退到远程 `flowchartImageUrl`，不阻塞分享。
  - `pages/flowchart-viewer/index.vue` 调整 payload 恢复顺序，优先读取 `localImagePath`，仅当本地图加载失败时再回退 `imageUrl`；只有本地和远程都不可用时才进入失败空态，其余缩放、拖动和关闭交互保持不变。
- **影响范围**：
  - `pages/recipe-detail/index.vue`
  - `pages/flowchart-viewer/index.vue`
  - 仅影响小程序菜谱详情页的流程图查看、轻点预览、横屏查看和微信分享封面准备链路，不涉及后端 API、worker 调度、流程图生成策略或成品图缓存逻辑。
- **兼容性/风险**：
  - 本次继续复用 `utils/image-cache.js`，缓存或本地文件失效时会静默回退远程 URL，主查看链路不应被阻断。
  - 收藏夹分享接口仍不支持 `promise` 异步配置，因此首次触发“收藏”时若正方形裁图尚未准备完成，会先回退到远程流程图；后续同版本流程图可直接复用已裁好的本地封面。
  - 本次未改动后端字段和 worker 行为，因此“生成开始前的排队 pickup delay”仍维持现状，属于当前 `P0` 非目标。
- **验证情况**：
  - 已对 `pages/recipe-detail/index.vue`、`pages/flowchart-viewer/index.vue` 的 `<script>` 片段分别执行 `node --check`，语法通过。
  - 已人工复核关键 diff，确认详情页流程图展示、`previewImage`、横屏 viewer payload、分享封面构造都改为本地缓存优先 / 懒裁图口径。
  - 本次未做真机或微信开发者工具手工回归；建议补测：首次进入详情、二次进入命中缓存、横屏查看本地图优先、轻点预览、好友分享 / 收藏分享裁图回退链路。

## 2026-04-25 (菜谱详情页流程图性能方案收口：暂不做 worker 即时唤醒)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：在继续细化菜谱详情页流程图性能方案时，用户明确提出“把流程图 worker 从轮询捞任务改成入队即唤醒”这一项先不做，原因是流程图生图本身耗时和成本都较高，当前更应该优先解决已生成流程图的查看慢问题。
- **核心改动**：
  - 更新 `docs/recipe-flowchart-performance-optimization.md`，移除 `P1` 中关于“手动生成请求增加即时唤醒能力”的实施方案、收益描述与后续推进表述。
  - 将文档中的当前工程目标从“展示版图片、轻量状态接口和即时唤醒 worker”调整为“展示版图片和轻量状态接口”。
  - 在文档 `非目标` 中明确补充：当前暂不纳入“流程图 worker 入队即唤醒”改造，并写明暂缓原因，避免后续误判为遗漏。
  - 同步调整实施顺序与结论描述，确保后续默认推进方向聚焦于 `P0` 查看链路优化，以及后续的展示版图片和轻量状态接口。
- **影响范围**：
  - `docs/recipe-flowchart-performance-optimization.md`
  - 当前仅影响文档口径和后续实施优先级，不涉及运行时代码。
- **兼容性/风险**：
  - 本次为文档调整，无运行时风险。
  - 保留当前 worker 轮询机制，短期内“提交后到真正开始处理”的空等现象仍会存在；该问题暂不作为当前阶段的优先处理项。
- **验证情况**：
  - 已核对文档中的方案编号、实施顺序、非目标和结论描述，确保不再把 worker 即时唤醒列入默认改造计划。

## 2026-04-25 (菜谱详情页流程图性能改造方案文档)

### Changed

- **修改时间**：2026-04-25 CST
- **背景**：用户反馈小程序菜谱详情页的流程图打开长期偏慢，需要先把“查看慢”和“生成慢”的问题拆开，明确现状链路、瓶颈判断、改造优先级和验收口径，再进入正式编码阶段。
- **核心改动**：
  - 新增 `docs/recipe-flowchart-performance-optimization.md`，沉淀菜谱详情页流程图性能改造方案。
  - 文档明确区分两类问题：已生成流程图的查看慢、手动生成后的总等待长。
  - 文档梳理当前前端查看链路与后端生成链路，确认现阶段主要瓶颈包括：流程图未复用本地图片缓存、分享封面裁图过早触发、worker 固定间隔轮询带来的 pickup delay、后端原图直出、状态轮询走完整详情。
  - 文档给出分阶段改造建议：
    - `P0`：流程图接入本地图片缓存、分享封面裁图改为懒执行
    - `P1`：后端增加展示版图片、增加轻量状态接口
    - `P2`：压缩策略、主动缓存预热、可观测性补强
  - 文档补充实施顺序、验收标准、风险点与回滚口径，作为后续正式改造的统一依据。
- **影响范围**：
  - `docs/recipe-flowchart-performance-optimization.md`
  - 当前仅影响项目文档与后续实施口径，不涉及运行时代码和接口变更。
- **兼容性/风险**：
  - 本次为文档改动，无运行时风险。
  - 后续若按文档实施，需要重点处理流程图缓存版本键和展示版图片兼容策略。
- **验证情况**：
  - 已结合当前前端详情页、横屏查看页、图片缓存工具、后端流程图 service/worker/upload 链路完成方案复核。
  - 本次未运行构建或测试；当前尚未进入代码改造阶段。

## 2026-04-24 (Admin Web AI Provider UI P0+P1 重构落地)

### Changed

- **修改时间**：2026-04-24 21:54 CST
- **背景**：根据 `docs/admin-ai-provider-ui-design-report-2026-04-24.md` 的 P0+P1 路线，`admin-web` 的 AI Provider 页面需要从“大表单 + 常驻说明”调整为更适合运维控制台的简洁扫描、渐进披露、贴近触发点反馈与可追溯审计体验。
- **核心改动**：
  - 新增 `admin-web/src/components/HelpTip.vue`，统一字段说明的 hover/focus tooltip 行为。
  - 重构 `admin-web/src/pages/AIProvidersPage.vue` 的场景状态、Provider 行、测试反馈、审计区与底部浮动操作条。
  - 将常驻告警横幅改为状态条里的按需 popover 入口，移除 emoji 状态符号。
  - Provider 默认行展示关键字段、密钥状态、启用状态与最近测试微状态；完整配置放入展开编辑区。
  - 增加 Provider 字段 blur 校验与保存/测试前阻断校验，覆盖 Provider Name、Base URL、Model、Timeout，并保留内部 ID 异常保护。
  - 测试中显示 elapsed timer，完成后自动定位测试结果；单节点测试结果回写到 Provider 行内微状态。
  - 审计区默认展示最近 5 条，完整审计迁入抽屉并支持展开查看旧值/新值对比。
  - 保存前增加 diff 摘要确认，底部 bar 提供草稿摘要、刷新、测试与保存入口。
  - Review 修复：单节点测试增加全局防重入；最近审计与完整审计分页/筛选状态拆分，避免抽屉操作污染主卡片。
  - 根据用户反馈移除顶部 `测试当前草稿 / 保存场景` 重复主按钮，保留底部浮动 bar 作为唯一高频测试与保存入口，顶部仅保留刷新、放弃草稿和测试结果状态。
  - 根据用户反馈将“最近审计”从横向表格改为事件时间线卡片，默认展示动作、目标对象、操作人、时间与字段级变化摘要，完整旧值/新值 diff 通过 popover 按需查看。
  - 继续优化“查看 diff”popover：默认展示字段级 diff 摘要、变更类型 chip、删除/新增/脱敏语义，原始旧值/新值改为折叠区按需展开。
  - 根据用户反馈优化保存确认 tips：将纯文本确认改为彩色结构化 diff 摘要，按新增、删除、修改、密钥、顺序调整展示语义标签与影响说明，并将主按钮文案调整为 `保存并发布`。
  - 根据用户反馈移除测试完成后的自动滚动跳转，测试结果仍同步到顶部状态 chip、底部状态条和 Provider 行内状态，用户可点击顶部 chip 手动查看详情。
  - 根据用户反馈优化 Provider 卡片：右侧常驻操作收敛为启用、测试和更多菜单，上下移动进入更多菜单；展开编辑区拆为基础信息、请求配置和密钥三组，卡片头部与状态 chips 更紧凑。
  - 根据用户反馈修复点击后页面下滑体验：场景切换仅在键盘方向键操作后恢复焦点，Provider 展开/密钥编辑时保持当前滚动位置，并禁用 Provider 列表局部 scroll anchoring。
  - 根据用户反馈将 Provider ID 调整为内部稳定标识：前端不再展示或编辑 ID，新建/复制节点自动生成唯一 ID；Provider Name 作为用户可见主标识并增加必填与场景内唯一校验；测试结果、审计列表和保存确认优先展示 Provider Name，避免误改 ID 导致密钥丢失。
  - 根据用户反馈优化 Provider 基础信息区：将 Provider Name 说明收敛到字段 HelpTip，并把 Provider Name 与 Model 放在同一行以提升编辑密度。
  - 根据用户反馈精简保存确认弹窗文案：按测试状态动态展示发布建议，弱化测试通过场景的警告色，Provider diff 改为对象 + 动作 + 当前/发布后状态表达。
  - 根据用户反馈优化审计可读性：最近审计与完整审计抽屉新增业务动作推导，支持展示新增节点、删除节点、修改配置、启停节点、密钥变化、场景策略调整、测试通过/异常等语义状态；审计详情改为统计 chip + 字段分组 + 旧值/新值可视化对比，原始值继续折叠保留。
  - 根据用户反馈将审计详情影响说明改为历史审计语气，并对新增节点按启用状态区分“参与调度 / 暂不参与调度”，避免“发布后”文案误导。
- **影响范围**：
  - 仅影响 `admin-web` AI Provider 配置页面前端交互与展示。
  - 不修改后端 API、数据结构、登录态、上传链路或部署配置。
- **兼容性·风险**：
  - 前端保持 Element Plus 与既有接口契约；未引入新的运行时依赖。
  - 保存与测试新增前置校验，历史上依赖“保存时才报错”的非法草稿会更早被阻断。
  - 构建仍存在既有 chunk size warning，不影响本次构建产物生成。
- **验证情况**：
  - `cd admin-web && npm exec -- vite build` 通过；Review 修复后再次构建通过。
  - 移除顶部重复按钮后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 最近审计改为时间线卡片后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - `查看 diff` popover 优化后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 保存确认 tips 结构化与颜色语义优化后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 移除测试完成自动滚动后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - Provider 卡片结构与视觉密度优化后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 点击后页面下滑体验修复后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - Provider ID 隐藏与 Provider Name 唯一校验调整后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - Provider Name HelpTip 与基础信息两列布局调整后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 保存确认弹窗文案与 diff 表达精简后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 审计业务动作推导与详情可视化优化后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 审计影响说明文案调整后再次执行 `cd admin-web && npm exec -- vite build` 通过。
  - 2026-04-24 22:03 CST 已执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，本地构建并上传到 `/srv/caipu-miniapp/admin-web/dist`，release id 为 `20260424-220313`。
  - 2026-04-24 22:07 CST 移除顶部重复按钮后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-220747`，远端 `dist/index.html` 更新时间为 `2026-04-24 22:07 CST`。
  - 2026-04-24 22:14 CST 最近审计改为时间线卡片后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-221405`，远端 `dist/index.html` 更新时间为 `2026-04-24 22:14 CST`。
  - 2026-04-24 22:22 CST `查看 diff` popover 优化后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-222244`，远端 `dist/index.html` 更新时间为 `2026-04-24 22:22 CST`。
  - 2026-04-24 22:31 CST 保存确认 tips 结构化优化后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-223145`，远端 `dist/index.html` 更新时间为 `2026-04-24 22:31 CST`，公网校验 `https://www.gxm1227.top/admin/` 返回 HTTP 200。
  - 2026-04-24 22:48 CST Provider 卡片结构与视觉密度优化后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-224802`，远端 `dist/index.html` 更新时间为 `2026-04-24 22:48 CST`，公网校验 `https://www.gxm1227.top/admin/` 返回 HTTP 200。
  - 2026-04-24 22:59 CST 点击后页面下滑体验修复后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-225900`，远端 `dist/index.html` 更新时间为 `2026-04-24 22:59 CST`，公网校验 `https://www.gxm1227.top/admin/` 返回 HTTP 200。
  - 2026-04-24 23:12 CST Provider ID 隐藏与 Provider Name 唯一校验调整后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-231234`，远端 `dist/index.html` 更新时间为 `2026-04-24 23:12 CST`，公网校验 `https://www.gxm1227.top/admin/` 返回 HTTP 200。
  - 2026-04-24 23:23 CST 保存确认弹窗文案与 diff 表达精简后再次执行 `SERVER_HOST=my-cloud DOMAIN=www.gxm1227.top bash scripts/upload-admin-web-dist.sh`，release id 为 `20260424-232342`，远端 `dist/index.html` 更新时间为 `2026-04-24 23:23 CST`，公网校验 `https://www.gxm1227.top/admin/` 返回 HTTP 200。
  - 脚本校验 `https://www.gxm1227.top/admin/` 返回 HTTP 200。
  - 未做浏览器人工回归；建议后续在后台页面手动检查无草稿、有草稿、测试中、测试失败和审计抽屉场景。

## 2026-04-24 (Admin Web AI Provider UI 设计报告补充：简洁优先与 Tips 渐进披露)

### Changed

- **修改时间**：2026-04-24 CST
- **背景**：用户明确希望 `admin-web` 的 AI Provider 改造保持整体界面简洁明了；如字段含义、策略说明、错误类型解释等需要补充说明，应优先通过 hover tips、click popover、详情抽屉等渐进披露方式展示，避免把长段说明常驻铺在页面中。
- **核心改动**：
  - 更新 `docs/admin-ai-provider-ui-design-report-2026-04-24.md`，新增“简洁优先与 Tips 策略”设计原则。
  - 明确常驻信息、hover tooltip、click popover、drawer/展开区、modal confirmation 的使用边界。
  - 将场景状态条、Provider 节点、测试结果、审计区、底部浮动 bar 的改造建议同步调整为“默认短文案 + 按需解释”。
  - 补充二次评估结论：新增统一 `HelpTip` 组件、说明文案配置表、字段 blur 校验、底部 bar 草稿摘要 popover、Provider 最近测试微状态、flowchart 长耗时测试计时反馈、按需出现的锚点目录、禁用 emoji 状态符号、统一 focus ring 等建议。
  - 调整 P0/P1 优先级，把“说明文案收敛为 tooltip / popover 策略”“HelpTip 组件”“字段前置校验”“底部 bar 草稿摘要”纳入首批前端优化。
- **影响范围**：
  - 文档与后续 UI 改造验收口径。
  - 不涉及前端代码、后端接口或部署配置变更。
- **兼容性·风险**：
  - 无运行时风险。
  - 后续实现时需注意：关键错误、阻塞状态、保存影响不能只藏在 tooltip 中，仍应常驻可见并满足无障碍要求。
- **验证情况**：
  - 已完成文档结构检查与改造建议复核。
  - 未运行构建或单测；本次仅更新设计文档与变更记录。

## 2026-04-23 (线上 nginx `/caipu-api/` 代理超时放宽到 300 秒)

### Changed

- **修改时间**：2026-04-23 17:34 CST
- **背景**：`admin-web` 在 `AI Provider -> flowchart 场景测试` 中新增 `gpt-image-2` 节点后，真实测试 prompt 会触发长耗时出图链路；后端 `/api/admin/ai-routing/scenes/{scene}/test` 已放宽到 3 分钟，但线上 nginx 的 `/caipu-api/` 代理段未单独配置长超时，导致请求在约 60 秒处被反向代理取消，表现为后台测试失败而上游节点本身并非不可用。
- **核心改动**：
  - 更新线上 nginx 站点配置 `/etc/nginx/conf.d/www.gxm1227.top.conf`，为 `location /caipu-api/` 增加：
    - `proxy_read_timeout 300;`
    - `proxy_send_timeout 300;`
  - 保持 `/caipu-api/` 的 upstream、header 转发与其他路径路由不变，仅放宽等待时间，覆盖后台 AI 路由测试与其他长耗时 API 请求。
  - 同步更新 `docs/cloud-server-config-overview.md`，把 `/caipu-api/` 的当前实际超时口径记入项目文档。
- **影响范围**：
  - 线上 nginx 反向代理。
  - `admin-web` 经 `/caipu-api/` 发起的长耗时后台测试请求，尤其是 `flowchart` 场景的图片生成测试。
- **兼容性·风险**：
  - 这是代理层等待时间放宽，不改变后端 API 契约、不改变业务逻辑。
  - `/caipu-api/` 下其他接口也会继承 300 秒等待时间；若未来出现真正卡死的请求，前端等待时间会更长，但当前优先解决 AI 测试被 60 秒截断的问题。
- **验证情况**：
  - `nginx -t` 通过。
  - `systemctl reload nginx` 成功。
  - `systemctl status nginx --no-pager` 显示 reload 于 `2026-04-23 17:34:20 CST` 成功执行。

## 2026-04-23 (AI Flowchart `images/generations` 去掉默认 `quality=high`)

### Changed

- **修改时间**：2026-04-23
- **背景**：`admin-web` 的流程图 Provider 配置页长期提示 `images/generations` 会固定带 `quality=high`、`output_format=png`。核对 OpenAI Images API 后，`quality` 属于可选参数，默认由上游按 `auto` 处理；继续强绑 `high` 会抬高请求成本，也让后台提示与真实意图「这里只配返回格式」混在一起。
- **核心改动**：
  - **后端正式路由**：`backend/internal/airouter/service.go` 在 `images/generations` 请求体中移除 `quality` 字段，仅保留固定 `output_format=png`，`response_format` 仍按后台节点配置决定是否传 `image_url / b64_json`。
  - **后端兼容回退链路**：`backend/internal/recipe/flowchart.go` 的直连流程图生成请求同步去掉默认 `quality=high`，确保新旧链路口径一致。
  - **后台连通性测试**：`backend/internal/appsettings/runtime_provider.go` 的 `testFlowchartCompatible` 也不再带 `quality`，避免“测试请求”和“真实请求”行为不一致。
  - **后台文案**：`admin-web/src/pages/AIProvidersPage.vue` 提示改为“当前固定按 `output_format=png` 发请求；这里仅配置 `response_format` 返回格式”，显式区分文件输出格式与响应返回格式。
  - **测试补强**：3 个相关单测新增请求体断言，明确要求 `quality` 不出现，同时继续校验 `output_format=png` 和已配置的 `response_format`。
- **影响范围**：`admin-web` AI Provider 页面文案、后端流程图正式/兼容请求构造、后台流程图 Provider 测试入口。
- **兼容性·风险**：低。对支持 OpenAI Images 兼容协议的上游，省略 `quality` 会回到服务端默认策略；若某些三方兼容网关历史上错误依赖显式 `quality` 字段，可能暴露兼容性差异，但后台保留 `output_format` 与 `response_format`，主行为不变。
- **验证情况**：`GOCACHE=/tmp/caipu-go-build-cache go test ./internal/airouter ./internal/appsettings ./internal/recipe` 全通过；`cd admin-web && npm exec -- vite build` 构建通过。补充说明：当前仓库直接跑 `tsc -p tsconfig.app.json --noEmit` 会因为缺少 `.vue` 模块声明在既有基线上失败，不属于本次改动引入的问题。

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
