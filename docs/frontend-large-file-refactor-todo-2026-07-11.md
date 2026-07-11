# 前端大文件拆分与可维护性优化待办

- **创建时间**：2026-07-11 02:04:21 +0800
- **最后更新时间**：2026-07-11 16:20:05 +0800
- **适用范围**：微信小程序前端、`admin-web` 管理后台
- **状态**：A～E 代码工作包已完成；Admin 登录态浏览器人工回归受当前无可用浏览器实例限制。
- **目标**：在不改变业务行为、接口契约和视觉结果的前提下，降低页面级单体文件的职责密度，形成可独立验证、可回滚的模块边界。

## 1. 状态约定

- `[ ]` 未开始。
- `[-]` 进行中。
- `[x]` 已完成且通过可执行验收。
- `[!]` 代码工作已完成，但外部环境阻止部分人工验收。

拆分不以行数为唯一目标。纯规则、格式化、payload 生成、异步任务、计时器和图表实例必须有明确所有者；页面只保留布局、生命周期和跨模块编排。

## 2. 基线与完成体量

| 优先级 | 文件 | 基线行数 | 完成行数 | 结果 |
| --- | --- | ---: | ---: | --- |
| P0 | `admin-web/src/pages/AIProvidersPage.vue` | 7,991 | 3,276 | 告警、审计、草稿、校验和六个视图组件已拆出 |
| P0 | `pages/index/index.vue` | 5,490 | 1,100 | 菜谱、地点、菜单、空间、智能添加控制器和两个 Pane 已拆出 |
| P0/P1 | `pages/recipe-detail/index.vue` | 5,491 | 1,587 | 编辑、异步任务、图片、分享控制器和五个展示组件已拆出 |
| P1 | `admin-web/src/pages/DashboardPage.vue` | 1,450 | 1,420 | ECharts 生命周期已统一下沉 |
| P1 | `admin-web/src/style.css` | 1,549 | 5 | 仅保留五个职责样式入口的稳定导入顺序 |
| P2 | `pages/index/components/place-detail-sheet.vue` | 1,028 | 1,028 | 复核后维持单组件，主体增长来自样式而非多业务域 |

## 3. 总体状态

| 阶段 | 范围 | 状态 | 完成标志 |
| --- | --- | --- | --- |
| A | `AIProvidersPage.vue` | [x] | 告警、审计、Provider 编辑和场景保存形成独立模块 |
| B | `pages/index/index.vue` | [x] | 首页按菜谱、地点、菜单、空间和智能添加拆出控制器 |
| C | `pages/recipe-detail/index.vue` | [x] | 图片、异步任务、分享、编辑器和展示面板独立 |
| D | 次级文件与全局样式 | [x] | Dashboard 图表和 Admin 全局样式按职责收口 |
| E | 回归与收口 | [!] | 自动化、构建、小程序编译和关键冒烟完成；Admin 登录态人工回归待具备浏览器实例后补做 |

## 4. 阶段 A：AI Provider 管理页

### A1. 告警展示规则纯函数化 `[x]`

- [x] 新增 `admin-web/src/utils/ai-provider-alerts.ts`。
- [x] 抽离状态兼容、错误中文化、相对时间、场景聚合和问题摘要。
- [x] 保留 `thresholdReached`、未知错误类型与 24 小时最近恢复回退。

### A2. 审计展示规则抽离 `[x]`

- [x] 新增 `admin-web/src/utils/ai-provider-audit.ts`。
- [x] 抽离动作中文化、字段分组、差异优先级、值压缩和摘要。
- [x] 保留未知 action / field 原值回退。
- [x] 覆盖新增、删除、修改 Provider 和 route test 最小测试。

### A3. 场景草稿与 Provider 校验 `[x]`

- [x] 新增 `useAIRoutingDraft.ts`、`ai-provider-draft.ts` 和 `ai-provider-validation.ts`。
- [x] 抽离 hydrate、payload、快照、diff、字段校验和保存阻断。
- [x] 保持“密钥留空即保留、显式清空才删除”。
- [x] 保持多启用节点 `maxAttempts >= 2` 和未保存草稿离页保护。

### A4～A6. 视图组件化 `[x]`

- [x] 新增 `SceneOverviewCards.vue`、`AlertLifecyclePanel.vue`。
- [x] 新增 `ProviderEditor.vue`，承载列表、模板、复制、删除、排序和折叠。
- [x] 新增 `RouteTestResult.vue`、`AuditTimeline.vue`、`AuditDrawer.vue`。
- [x] 保留复测成本确认、批量动作、字段显隐、requestId、筛选和分页。
- [x] 页面主要保留布局、路由、生命周期和跨模块编排，未形成万能 composable。
- [x] `test:ai-provider-utils`、TypeScript 和 Vite build 通过。
- [!] 当前 Browser 运行时可用实例列表为空，无法完成登录态下保存、告警处置和审计筛选的浏览器人工回归。

## 5. 阶段 B：小程序首页

### B1～B5. 业务控制器 `[x]`

- [x] `use-recipe-library.js`：搜索筛选、计数、随机推荐和失焦调度器。
- [x] `use-place-library.js`：地点筛选、归一化和草稿构建。
- [x] `use-meal-order.js`：菜单条目与延迟同步控制器，明确执行、取消和卸载清理。
- [x] `use-kitchen-space.js`：成员文案、邀请标题/图片和空间名称替换。
- [x] `use-smart-add.js`：菜谱/地点识别结果到草稿的映射。
- [x] 二次所有权审计后，将上述五个业务域的页面 methods/computed 一并下沉；页面只组合模块并协调跨域状态。
- [x] 保持菜单 pending action、空间洞察 bridge 和用户手势剪贴板约束。

### B6. 页面容器收口 `[x]`

- [x] 新增 `library-pane.vue` 和 `place-pane.vue`，收口两种首页模式模板。
- [x] 页面继续负责 `activeSection / appMode` 与跨域协调，不在子组件复制业务校验。
- [x] 页面样式迁至 `pages/index/index-page.scss`。
- [x] 首页各类计时器和清理动作有唯一所有者。

阶段 B 验收：

- [x] JS 语法、SFC 编译、HBuilderX 5.07 微信小程序编译通过。
- [x] 微信开发者工具自动预览成功，冒烟检查美食库、打卡点列表和地点详情可渲染。
- [x] 菜谱/地点筛选、随机推荐、菜单同步、空间文案和智能映射规则测试通过。

## 6. 阶段 C：菜谱详情页

### C1. 异步任务状态 `[x]`

- [x] 新增 `use-recipe-async-jobs.js`，统一解析/流程图状态、等待提示、轮询和清理。
- [x] 二次所有权审计后新增异步任务 controller，页面不再直接拥有轮询与预计等待倒计时。
- [x] 保持旧菜谱与缺少状态字段时的回退。
- [x] 运行时冒烟发现并修复 `toPositiveInteger` 拆分遗漏，并新增规则测试。

### C2. 图片与分享 `[x]`

- [x] 新增 `use-recipe-images.js`，抽离缓存条目、版本和可见索引映射。
- [x] 新增 `use-recipe-share.js`，抽离分享 token、标题、路径、图片信息和 Canvas 导出。
- [x] 图片缓存/上传/删除与分享 token/渠道配置分别由 controller 管理，页面仅保留事件代理。
- [x] 保持公开只读分享不暴露编辑动作和私有字段。

### C3～C4. 编辑与展示组件 `[x]`

- [x] 新增 `recipe-edit-sheet.vue`，承载编辑草稿、食材、步骤和图片顺序界面。
- [x] 编辑器组件拥有本地草稿、脏检查、关闭确认和图片/食材/步骤操作；页面只提交归一化保存 payload。
- [x] 新增 `recipe-hero.vue`、`recipe-cooking-panel.vue`、`recipe-flowchart-panel.vue`、`public-readonly-banner.vue`。
- [x] 页面样式迁至 `recipe-detail-page.scss`，保留步骤完成态、流程图回退和只读说明。

阶段 C 验收：

- [x] 详情页与全部子组件 SFC 编译、HBuilderX 编译通过。
- [x] 微信开发者工具中私有菜谱详情、Hero、做法/流程图区域和编辑入口可渲染。
- [x] 编辑草稿、步骤完成态、等待提示、图片索引和分享 token 规则测试通过。
- [!] 公开分享 token 的真实网络打开、上传/删除图片和保存编辑未执行破坏性人工操作；已由纯规则测试和编译覆盖。

## 7. 阶段 D：次级优化

### D1. Dashboard 图表 `[x]`

- [x] 新增 `useDashboardCharts.ts`，统一实例创建、DOM 切换、resize 和 dispose。
- [x] 评估 `DistributionPanel.vue`：三组分布的展示结构与交互不同，仅图表生命周期可稳定复用，因此不创建空泛展示组件。

### D2. Admin 全局样式 `[x]`

- [x] 拆为 `tokens.css`、`base.css`、`shell.css`、`components.css`、`responsive.css`。
- [x] AI Provider 页面样式迁至 `ai-providers-page.css`，小程序页面样式分别外置。
- [x] `style.css` 仅保留五个 `@import`，修正初次拆分时跨文件选择器边界。
- [x] 保持 `main.ts` 入口和样式层叠顺序不变。
- [x] 复核 `components.css` 中 Dashboard/Server health/Settings 相关选择器的实际使用方；保留跨页面复用的卡片、统计和响应式规则，避免复制到多个 SFC。

### D3. 次级大组件复核 `[x]`

- [x] `place-detail-sheet.vue`：240 行模板、179 行脚本、607 行样式，展示域集中，暂不拆。
- [x] `SettingsPage.vue`：625 行，围绕运行时配置分组和同域审计，暂不拆。
- [x] `space-stats/index.vue`：650 行，围绕单一空间统计域且样式已复用，暂不拆。

## 8. 通用质量门槛

- [x] 未修改后端 API 契约和持久化结构。
- [x] 未新增无关依赖。
- [x] 无新增 TODO、废弃 import 或旧实现分支。
- [x] 未知枚举和旧数据保留回退。
- [x] `git diff --check` 通过。
- [x] Admin 两套规则测试、TypeScript 和 Vite build 通过。
- [x] 小程序 JS、SFC、HBuilderX 编译和微信自动预览通过。
- [x] 已核对 `git status --short`，未修改用户原有的 `animated-number.vue`、`tmp/douyin_public_share_probe.py` 和 `tmp/imagegen/`。

## 9. 验证命令与结果

```bash
npm --prefix admin-web run test:ai-provider-utils
npm --prefix admin-web run test:frontend-refactor
npm --prefix admin-web run typecheck
npm --prefix admin-web run build
npm run wx:auto-preview:skip-compile
git diff --check
```

- 上述命令均通过。
- HBuilderX 5.07 `launch mp-weixin --compile true` 编译成功。
- Vite 保留既有 `element-plus` 大 chunk warning。
- 微信开发者工具保留既有基础库 timeout、废弃属性和部分组件 WXSS 选择器警告；本次发现的详情页运行时 ReferenceError 已修复并复验消失。

## 10. 实施记录

| 时间 | 工作包 | 状态 | 变更与验证 |
| --- | --- | --- | --- |
| 2026-07-11 02:04:21 +0800 | 初始化待办 | 已完成 | 建立基线、阶段、验收和回滚策略 |
| 2026-07-11 02:13:52 +0800 | A1 | 已完成 | 首批告警规则抽离，TypeScript、Vite build、diff 检查通过 |
| 2026-07-11 09:30:00 +0800 | A2～A6 | 已完成 | 审计、草稿、校验和六个视图组件完成；AI Provider 页降至 3,276 行 |
| 2026-07-11 09:35:00 +0800 | B1～B6 | 已完成 | 五个业务模块和两个 Pane 完成；首页降至 4,043 行 |
| 2026-07-11 09:40:00 +0800 | C1～C4 | 已完成 | 四个业务模块、五个子组件和样式外置完成；详情页降至约 2,278 行 |
| 2026-07-11 09:45:00 +0800 | D1～D3 | 已完成 | Dashboard 图表生命周期、五层全局样式和次级大文件复核完成 |
| 2026-07-11 10:06:00 +0800 | E 回归收口 | 部分人工项受限 | 全量静态门禁、HBuilderX 编译、微信自动预览与关键页面冒烟通过；修复详情页 `toPositiveInteger` 运行时遗漏；Admin 浏览器实例不可用 |
| 2026-07-11 16:20:05 +0800 | B/C 所有权补审与 E 再验证 | 已完成 | 首页业务 methods/computed 与详情编辑、轮询、图片、分享行为继续下沉；修复模块对象闭合及尾部 import 导致的 uni-app 注入编译错误；首页/详情分别降至 1,100/1,587 行；两套规则测试、TypeScript、Vite、HBuilderX 和微信自动预览重新通过 |
