# 前端第二阶段全量重构清单

- **创建时间**：2026-07-13 22:44:09 +0800
- **最后更新时间**：2026-07-14 01:18:25 +0800
- **适用范围**：微信小程序前端、`admin-web` 管理后台、前端测试入口
- **当前状态**：已完成
- **背景**：承接 2026-07-11 第一阶段大文件拆分。在页面体量下降后，继续解决隐式页面依赖、重复流程、数据层职责混杂和 Admin 页面剩余编排过重问题。
- **原则**：保持业务行为、后端 API、持久化结构与现有视觉不变；优先提取有明确所有权且可独立测试的逻辑，不按行数机械拆分。

## 1. 状态约定

- `[ ]` 未开始。
- `[-]` 进行中。
- `[x]` 已完成并通过对应自动化验收。
- `[!]` 代码完成，但外部运行环境阻止部分人工验收。

## 2. 总体进度

| 工作包 | 范围 | 优先级 | 状态 | 完成标志 |
| --- | --- | --- | --- | --- |
| R1 | 前端测试入口与目录归属 | P0 | [x] | 根目录可直接运行 Miniapp 规则测试，Admin 与 Miniapp 测试职责分开 |
| R2 | 添加链接预览共享流程 | P1 | [x] | 两个预览组件复用剪贴板、计时器、阶段与收尾控制器 |
| R3 | 空间统计数字动效 | P2 | [x] | 首页卡片与洞察页复用无 UI 的 count-up controller |
| R4 | 饮食管家流式传输 | P1 | [x] | UTF-8 解码、SSE 解析、请求适配与消息 API 分层并具备边界测试 |
| R5 | 菜谱数据层 | P1 | [x] | 模型、缓存、远端仓储职责分离，旧导入契约兼容 |
| R6 | 菜谱详情剩余控制器 | P2 | [x] | 加载、步骤完成态与动作反馈计时器拥有明确所有者 |
| R7 | 首页业务模块显式依赖 | P0 | [x] | 五个 `use-*.js` 不再以无边界 method/computed bag 访问整个页面实例 |
| R8 | Admin Dashboard 分布面板 | P1 | [x] | 三组重复分布模板收口为同一组件，规则转换可独立测试 |
| R9 | Admin AI Provider 第二轮拆分 | P0 | [x] | 节点编辑、告警动作、场景策略与保存确认继续下沉，页面只保留跨域编排 |
| R10 | 全量回归与记忆闭环 | P0 | [x] | 自动化、类型、构建、小程序编译、diff 检查与变更记录全部完成 |

## 3. R1：前端测试入口与目录归属 `[x]`

- [x] 新增根级 Miniapp 测试目录与稳定脚本入口。
- [x] 将 `admin-web/tests/frontend-refactor.check.ts` 迁出 Admin 工程。
- [x] `admin-web` 测试脚本只运行 Admin 规则和组件渲染测试。
- [x] 根目录 `npm test` 不再固定失败，并覆盖 Miniapp 纯规则测试。
- [x] 修正根 `package.json` 过时的“初始化阶段”描述。

验收：`npm test`、`npm --prefix admin-web run test`、`npm --prefix admin-web run typecheck`。

## 4. R2：添加链接预览共享流程 `[x]`

- [x] 新增无 UI 的预览流程 controller，拥有阶段切换、计时器和停止清理。
- [x] 新增剪贴板读取、可解析提示判断等可复用纯函数。
- [x] `add-link-preview-panel.vue` 保留通用菜谱/地点识别分支。
- [x] `add-recipe-preview-panel.vue` 保留菜谱专用超时占位兜底。
- [x] 卸载时不遗留计时器，重复点击不产生并发流程。
- [x] 补充 controller 执行、取消和超时差异测试。

## 5. R3：空间统计数字动效 `[x]`

- [x] 新增无 UI 的 count-up controller，不使用自定义数字渲染组件。
- [x] 首页 `space-stats-card.vue` 继续由普通 `<text>` 渲染数字。
- [x] 洞察页 `space-stats/index.vue` 继续支持 Tab 首次挂载后播放动画。
- [x] controller 支持单键替换、批量清理和卸载清理。
- [x] 补充缓动终值、取消和多键并行测试。

## 6. R4：饮食管家流式传输 `[x]`

- [x] 提取 UTF-8 chunk decoder，保留无 `TextDecoder` 环境的手工回退。
- [x] 提取 SSE parser，覆盖半包、多事件、错误事件与结束事件。
- [x] `diet-assistant-api.js` 只保留消息 API 和小程序请求适配。
- [x] 回调异常不破坏请求生命周期，abort 行为保持不变。
- [x] 补充跨 chunk 中文、JSON 半包、`done/error/tool_status` 等测试。

## 7. R5：菜谱数据层 `[x]`

- [x] 提取菜谱模型归一化、食材/步骤规则与 payload 构建模块。
- [x] 提取按空间隔离的本地缓存模块。
- [x] 提取远端同步、上传与 CRUD 仓储模块。
- [x] `utils/recipe-store.js` 保留兼容导出层，避免一次性修改全部调用方。
- [x] 补充旧字段、fallback parsed content、图片上限、缓存隔离和更新合并测试。

## 8. R6：菜谱详情剩余控制器 `[x]`

- [x] 提取菜谱私有/公开加载 controller，保留缓存优先和公开只读语义。
- [x] 提取步骤完成态 controller，统一 key、读取、写入、重置和无效项清理。
- [x] 提取动作反馈 timer controller，页面只维护展示数据。
- [x] 页面保留生命周期、跨 controller 协调和平台分享钩子。
- [x] 补充加载分支、步骤持久化和 timer 清理测试。

## 9. R7：首页业务模块显式依赖 `[x]`

- [x] 为菜谱、地点、菜单、空间、智能添加定义模块上下文边界。
- [x] 将纯规则与页面副作用分开；纯规则不得读取页面实例。
- [x] method/computed 安装使用显式模块清单，避免散落展开顺序决定覆盖关系。
- [x] 页面初始化与清理统一调用模块 lifecycle，减少 `onHide/onUnload` 重复。
- [x] 保持 Options API 与 uni-app 微信小程序编译兼容，不强制迁移全页 Composition API。
- [x] 扩大 Miniapp 规则测试，覆盖模块契约和关键清理动作。

## 10. R8：Admin Dashboard 分布面板 `[x]`

- [x] 新增 `DistributionPanel.vue`，统一标题、空态、排行、图表切换和告警提示。
- [x] 提取分布排行数据转换纯函数与类型。
- [x] 移除未被界面使用的 `table` 视图类型。
- [x] 保持三个 ECharts DOM 实例和 `useDashboardCharts` 生命周期不变。
- [x] 补充未知名称、并列排序、成功率 tone 与场景文案测试。

## 11. R9：Admin AI Provider 第二轮拆分 `[x]`

- [x] 新增 Provider 编辑 controller/composable，拥有本地 key、折叠、拖拽、密钥编辑、触碰与最近测试状态。
- [x] 新增告警动作 composable，拥有 pending、单项/批量处置与 overview 就地更新。
- [x] 提取场景策略表单组件，收口重试、熔断和请求参数模板。
- [x] 提取保存确认展示/格式化逻辑，避免页面内手写大段 VNode。
- [x] 缩减 `ProviderEditor` 的函数型 props 与事件数量，建立明确的 editor contract。
- [x] 保留密钥留空即保留、显式清空才删除、草稿离页保护和真实调用成本确认。
- [x] 扩充 SSR 渲染测试，覆盖首节点展开、保存确认和告警动作装配。

## 12. R10：全量回归与记忆闭环 `[x]`

- [x] Miniapp 纯规则测试通过。
- [x] Admin 纯规则、组件 SSR、TypeScript 与 Vite build 通过。
- [x] 全部修改过的 Miniapp SFC 可编译。
- [x] HBuilderX 微信小程序编译通过；微信开发者工具自动预览成功。
- [x] `git diff --check` 与 `git status --short` 完成审计。
- [x] 更新本文最后更新时间、总体状态和每个工作包结果。
- [x] 更新根目录 `CHANGELOG.md`，记录背景、核心改动、影响、风险与验证。

## 13. 验证命令

```bash
npm test
npm --prefix admin-web run test
npm --prefix admin-web run typecheck
npm --prefix admin-web run build
npm run wx:auto-preview
git diff --check
```

## 14. 实施记录

| 时间 | 工作包 | 状态 | 变更与验证 |
| --- | --- | --- | --- |
| 2026-07-13 22:44:09 +0800 | 初始化 | 进行中 | 建立第二阶段 R1～R10 全量看板、依赖顺序与验收门槛 |
| 2026-07-13 22:48:03 +0800 | R1 | 已完成 | Miniapp 测试迁至根 `tests/`，新增根级聚合测试与 Admin 独立测试入口；`npm test`、Admin TypeScript 均通过 |
| 2026-07-13 22:52:07 +0800 | R2 | 已完成 | 两个添加预览组件改用共享流程 controller；补充剪贴板、平台提示、阶段、计时与陈旧流程防护测试，Miniapp 测试通过 |
| 2026-07-13 22:55:49 +0800 | R3 | 已完成 | 首页概览卡和空间洞察页改用无 UI count-up controller；普通文本渲染与 Tab 懒挂载语义不变，缓动/终值/取消/并行测试通过 |
| 2026-07-13 23:00:13 +0800 | R4 | 已完成 | 饮食管家 API 拆出 UTF-8 decoder 与 SSE parser；API 文件降至 158 行，跨包中文、JSON 半包、工具状态、错误与结束事件测试通过 |
| 2026-07-13 23:07:21 +0800 | R5 | 已完成 | 菜谱数据层拆为 model/cache/repository，`recipe-store.js` 保留 16 行兼容出口；旧字段、fallback、图片上限和空间缓存隔离测试通过 |
| 2026-07-13 23:15:10 +0800 | R6 | 已完成 | 详情页新增加载、步骤完成态和动作反馈 controller；页面降至 1,550 行，缓存/公开加载、步骤持久化与 timer 清理测试通过 |
| 2026-07-13 23:51:36 +0800 | R7 | 已完成 | 首页五个业务域建立显式 module contract、统一安装与 lifecycle；页面降至 1,063 行，模块重复检测、依赖声明、销毁测试和 SFC 编译通过 |
| 2026-07-14 00:02:07 +0800 | R8 | 已完成 | Dashboard 三组分布面板提取为共享组件，排行规则独立成 util，移除未使用模式和死样式；页面降至 847 行，Admin 测试与类型检查通过 |
| 2026-07-14 01:10:30 +0800 | R9 | 已完成 | AI Provider 编辑与告警状态下沉为 composable，场景策略和保存确认拆为组件与纯规则；页面降至 2,357 行，新契约、密钥/顺序/测试状态及 SSR 装配测试通过 |
| 2026-07-14 01:18:25 +0800 | R10 | 已完成 | 根级聚合测试、Admin 类型检查与生产构建、6 个 Miniapp SFC 编译、HBuilderX 5.07 编译、微信自动预览及 diff/status 审计全部通过；根变更记录已同步 |
