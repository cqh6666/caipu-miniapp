# Project Changelog

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
