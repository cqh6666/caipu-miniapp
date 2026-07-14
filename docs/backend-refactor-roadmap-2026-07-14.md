# Go 后端可维护性重构路线图

- **创建时间**：2026-07-14 06:51:41 +0800
- **最后更新**：2026-07-14 07:36:16 +0800
- **状态**：已完成
- **适用范围**：`backend/` Go 服务端

## 1. 背景与目标

后端已经形成按业务域划分的 `internal/*` 包，并具备较完整的单元测试，但随着 AI 路由、链接解析、
菜谱异步任务、饮食管家与运行时配置持续增长，部分文件同时承担编排、协议适配、规则计算、持久化和
兼容逻辑，阅读与修改成本明显上升。

本轮重构目标：

1. 保持现有 HTTP API、数据库结构、运行配置和业务行为兼容。
2. 将超大文件按稳定职责拆分，降低单文件认知负担与修改冲突。
3. 缩小应用装配层中匿名闭包和跨模块映射逻辑，明确依赖方向。
4. 为配置、启动装配和外部协议边界补齐测试，避免“能编译但接错线”。
5. 建立可重复执行的自验门禁，并在每个工作包完成后更新本文进度。

本轮不做：

- 不引入新的 Web 框架、ORM、依赖注入框架或目录范式。
- 不为了拆文件改变包名、公开 API、数据库迁移或前端契约。
- 不以追求行数为唯一目标；只有职责边界明确且能独立验证时才拆分。

## 2. 现状基线

### 2.1 规模与热点

2026-07-14 基线统计：后端 Go/SQL 约 40,166 行，主要热点如下。

| 文件 | 行数 | 当前主要职责 | 风险 |
| --- | ---: | --- | --- |
| `internal/linkparse/service.go` | 2,304 | 服务配置、B 站协议、AI 总结、启发式解析、标题清洗 | 高 |
| `internal/airouter/service.go` | 1,810 | 配置发布、路由策略、上游协议、错误分类、审计 | 高 |
| `internal/recipe/repository.go` | 1,735 | 菜谱 CRUD、队列、图片、状态事件、分享查询 | 高 |
| `internal/dietassistant/service.go` | 1,702 | 对话编排、流解析、工具协议、工具执行、清洗规则 | 高 |
| `internal/appsettings/runtime_provider.go` | 1,134 | 配置读取、更新、审计、定义、连通性测试 | 高 |
| `internal/recipe/service.go` | 1,060 | CRUD、解析、图片、分享和队列业务编排 | 中高 |
| `internal/recipe/flowchart.go` | 900 | 流程图配置、调用协议、响应解析、提示词 | 中 |
| `internal/audit/service.go` | 814 | 任务/调用写入、列表查询、Dashboard 聚合、筛选规则 | 中 |
| `internal/app/app.go` | 734 | 全量依赖装配、跨模块适配、生命周期 | 高 |

包级生产代码规模较大的模块：`recipe` 5,755 行、`linkparse` 3,580 行、`airouter` 2,850 行、
`dietassistant` 2,134 行、`appsettings` 1,878 行。

### 2.2 测试与静态检查基线

基线命令：

```bash
cd backend
go test ./... -cover
go vet ./...
```

两项均通过。重点覆盖率如下：

| 包 | 覆盖率 | 结论 |
| --- | ---: | --- |
| `aialert` | 70.9% | 较好 |
| `spacestats` | 67.5% | 较好 |
| `middleware` | 62.0% | 较好 |
| `dietassistant` | 62.0% | 较好，但职责集中 |
| `place` | 60.8% | 较好 |
| `audit` | 60.5% | 较好 |
| `appsettings` | 55.6% | 中等 |
| `linkparse` | 44.2% | 协议与回退分支仍有缺口 |
| `addpreview` | 39.8% | 中等偏低 |
| `airouter` | 39.1% | 路由与上游协议分支较多 |
| `recipe` | 34.0% | 数据与异步链路较多 |
| `app` / `config` / `bootstrap` / `db` / `wechat` | 0% | 关键基础边界缺少测试 |

### 2.3 主要结构问题

1. **组合根过载**：`app.New` 同时创建所有模块、定义跨模块映射闭包和配置适配，接线错误只能依赖编译或人工发现。
2. **协议与规则混放**：`linkparse`、`airouter`、`dietassistant` 的 HTTP/SSE 协议细节与领域规则集中在单个 service 文件。
3. **仓储职责过宽**：`recipe.Repository` 同时管理主实体、自动解析队列、流程图队列、图片镜像、状态事件与分享令牌。
4. **运行时配置混合读写与探测**：`RuntimeProvider` 同时处理配置定义、缓存、事务写入、审计和外部连通性测试。
5. **基础设施验证不足**：配置默认值、迁移顺序、应用路由装配与微信客户端没有自动化回归保护。

## 3. 重构原则与门禁

每个工作包必须满足：

1. 先锁定行为测试，再调整结构；新发现的边界缺口必须补测试。
2. 保持包级公开契约稳定；确需调整时先记录影响与兼容策略。
3. 对修改包执行定向测试，对后端执行 `go test ./...` 和 `go vet ./...`。
4. 执行 `gofmt` 与 `git diff --check`，禁止遗留 TODO 或临时兼容代码。
5. 更新本路线图的状态、完成时间、核心改动、风险和验证结果。

阶段性质量目标：

- 核心编排文件尽量控制在约 800 行以内；超过时必须能说明单一职责为何成立。
- `app.New` 只保留启动资源管理和顶层装配顺序，跨模块适配逻辑移入有名称、可测试的构造函数。
- 外部协议解析、领域规则和持久化语句按文件边界可独立定位。
- 最终所有现有测试通过，且新增测试覆盖本轮提取出的关键纯逻辑和装配边界。

## 4. 执行路线图

| 编号 | 优先级 | 工作包 | 核心内容 | 验收标准 | 状态 |
| --- | --- | --- | --- | --- | --- |
| R0 | P0 | 基线与路线图 | 盘点规模、依赖、测试与风险，固化执行顺序 | 基线命令通过；路线图可检索、可更新 | **已完成** |
| R1 | P0 | 应用装配层收口 | 提取链接解析、流程图、饮食管家与 AI 路由适配器；缩短 `app.New`；补纯映射/配置适配测试 | `app.New` 聚焦装配；跨模块转换有命名入口；`internal/app` 有测试 | **已完成** |
| R2 | P0 | `linkparse` 拆分 | 按 B 站协议、AI 总结、启发式规则、标题清洗拆分 `service.go` | 原服务契约不变；核心文件职责单一；定向与全量测试通过 | **已完成** |
| R3 | P0 | `dietassistant` 拆分 | 分离对话编排、LongCat 标记流、OpenAI SSE、工具定义与执行 | 流事件顺序、工具结果和历史存储行为不变 | **已完成** |
| R4 | P1 | `airouter` 拆分 | 分离配置变更、路由策略、OpenAI 请求适配、错误分类与审计摘要 | 多 Provider 轮询/降级/重试/密钥语义不变 | **已完成** |
| R5 | P1 | `recipe` 数据与服务拆分 | 按 CRUD、异步队列、图片、分享、序列化拆分 repository/service | 事务边界、状态事件、队列抢占和兼容字段不变 | **已完成** |
| R6 | P1 | `appsettings` 拆分 | 分离配置定义、读缓存、事务更新/审计和连通性探测 | 清空密钥、默认值回退、审计分页和测试接口不变 | **已完成** |
| R7 | P1 | 配置与启动基础设施补强 | 拆分配置分组与校验；补 config、bootstrap、db、wechat 的关键测试 | 默认值、环境覆盖、迁移顺序、超时与错误映射有测试 | **已完成** |
| R8 | P2 | HTTP/装配集成门禁 | 为核心路由、中间件链、鉴权边界和健康检查补最小集成测试 | 公开/登录/管理员路由边界可自动验证 | **已完成** |
| R9 | P0 | 全量收口 | 全量测试、竞态检查、文件体量和依赖审计、README/CHANGELOG 更新 | 所有门禁通过；无未决高风险项；文档状态完成 | **已完成** |

## 5. 进度记录

### 2026-07-14：R0 完成

- 读取 `CHANGELOG.md`、根目录与后端 README、现有设计/部署文档。
- 统计文件与包规模，检查内部包依赖和现有测试分布。
- 执行 `go test ./... -cover` 与 `go vet ./...`，均通过。
- 确认重构优先从组合根和跨模块适配开始，再处理高复杂度业务包，最后补基础设施与集成门禁。

### 2026-07-14：R1 完成

- 将链接解析运行时配置、流程图运行时配置、旧单节点 AI 兼容路由和场景测试输入提取到
  `internal/app/ai_wiring.go`。
- 将饮食管家菜谱工具桥接与字段清洗提取到 `dietassistant_wiring.go`，通过窄接口隔离菜谱服务和链接解析器。
- 将三个菜谱后台 worker 的构造提取到 `recipe_wiring.go`；`app.go` 从 734 行降至 293 行。
- 新增 `wiring_test.go`，覆盖配置映射、标题配置回退、密钥掩码、流程图协议归一、测试输入分派、
  菜谱查询/创建映射和空间隔离；`internal/app` 覆盖率由 0% 提升到 30.6%。
- 验证通过：`go test ./...`、`go vet ./...`、
  `go test -race ./internal/app ./internal/linkparse ./internal/recipe ./internal/dietassistant`、
  `gofmt`、`git diff --check`。

### 2026-07-14：R2 完成

- 将 `linkparse/service.go` 按职责拆为服务配置、B 站协议、AI 总结、启发式菜谱规则和标题清洗五个文件。
- B 站探测参数、食材/步骤规则和预览标题正则移动到各自职责文件，跨平台共用 URL/代码块规则保留在服务核心。
- 原 2,304 行 `service.go` 降至 380 行；新文件分别为 626、542、415、402 行，无文件超过 800 行。
- 保持 `linkparse.NewService`、解析/预览方法和运行时配置契约不变，包覆盖率保持 44.2%。
- 验证通过：`go test ./...`、`go vet ./...`、
  `go test -race ./internal/linkparse ./internal/app ./internal/addpreview ./internal/recipe`、
  `gofmt`、`git diff --check`。

### 2026-07-14：R3 完成

- 将 `dietassistant/service.go` 拆为对话/存储编排、LongCat 工具标记协议、OpenAI 客户端和菜谱工具执行四个文件。
- 原 1,702 行 `service.go` 降至 466 行，其余职责文件为 403、274、586 行，无文件超过 800 行。
- 保持 SSE 事件、LongCat 隐藏标记过滤、多轮工具调用、历史消息清洗和菜谱工具输入输出不变，包覆盖率保持 62.0%。
- 验证通过：`go test ./...`、`go vet ./...`、
  `go test -race ./internal/dietassistant ./internal/app`、`gofmt`、`git diff --check`。

### 2026-07-14：R4 完成

- 将 `airouter/service.go` 拆为场景服务、配置变更、运行时路由、OpenAI 客户端、响应校验和配置规则六个文件。
- 图片引用识别规则归入响应校验文件；配置密钥保留/清空、Provider extra 持久化和审计摘要归入配置规则边界。
- 原 1,810 行 `service.go` 降至 429 行，其余职责文件为 242、349、364、206、276 行。
- 保持多 Provider 优先级/轮询降级、熔断、重试、复测、密钥加解密和审计行为不变，包覆盖率保持 39.1%。
- 验证通过：`go test ./...`、`go vet ./...`、
  `go test -race ./internal/airouter ./internal/aialert ./internal/appsettings ./internal/app ./internal/linkparse ./internal/recipe`、
  `gofmt`、`git diff --check`。

### 2026-07-14：R5 完成

- 将 `recipe/repository.go` 拆为主实体 CRUD、自动解析队列、流程图队列、图片镜像、扫描/序列化和分享查询六个文件。
- 将 `recipe/service.go` 拆为主服务、解析状态、状态/置顶/删除、流程图、输入校验和结构化菜谱规则六个文件。
- 将 `recipe/flowchart.go` 继续拆为生成器编排、上游客户端/响应解析和提示词/哈希规则三层。
- 原 1,735/1,060/900 行三个热点文件分别降至 402/219/212 行；`recipe` 最大生产文件为 469 行。
- 保持事务、状态事件、自动解析重试、流程图队列、图片镜像、分享令牌与兼容内容规则不变，覆盖率保持 34.0%。
- 验证通过：`go test ./...`、`go vet ./...`、
  `go test -race ./internal/recipe ./internal/app ./internal/linkparse ./internal/dietassistant`、
  `gofmt`、`git diff --check`。

### 2026-07-14：R6 完成

- 将 `appsettings/runtime_provider.go` 拆为运行时读取入口、更新/审计、缓存、配置定义/规范化和外部连通性探测五个文件。
- 缓存 TTL 归入缓存职责；密钥清空、事务更新、审计分页和探测请求分别保留在明确边界。
- 原 1,134 行文件降至 174 行，其余职责文件为 266、186、232、313 行。
- 保持配置默认值回退、缓存失效、密钥加解密、审计与测试接口行为不变，覆盖率保持 55.6%。
- 验证通过：`go test ./...`、`go vet ./...`、
  `go test -race ./internal/appsettings ./internal/airouter ./internal/aialert ./internal/admin ./internal/app`、
  `gofmt`、`git diff --check`。

### 2026-07-14：R7 完成

- 将配置环境文件/类型解析和配置校验/回退从 `config.go` 分离到 `env.go`、`validation.go`。
- 新增配置默认值、环境覆盖、env 文件优先级和非法值测试，`config` 覆盖率由 0% 提升至 93.1%。
- 新增迁移排序、幂等、失败回滚和目录错误测试，`bootstrap` 覆盖率由 0% 提升至 81.0%。
- 新增 SQLite 目录、连接池、WAL、外键和 busy timeout 测试，`db` 覆盖率由 0% 提升至 73.1%。
- 新增微信请求参数、成功响应、配置缺失、微信错误、网络和解码错误测试，`wechat` 覆盖率由 0% 提升至 94.7%。

### 2026-07-14：R8 完成

- 使用临时 SQLite 与全量迁移创建真实 `App`，通过 `Server.Handler` 验证完整路由和中间件链。
- 覆盖双健康检查、公开小程序配置、匿名用户/管理员拒绝、开发登录、Bearer 用户访问、管理员 cookie 登录与
  Dashboard 访问，以及生产环境不注册 `dev-login`。
- `internal/app` 覆盖率由 R1 的 30.6% 提升至 84.9%。
- R7/R8 联合验证通过：`go test ./...`、`go vet ./...`、
  `go test -race ./internal/config ./internal/bootstrap ./internal/db ./internal/wechat ./internal/app`、
  `gofmt`、`git diff --check`。

### 2026-07-14：R9 完成

- 全量文件审计发现 `audit/service.go` 仍有 814 行，继续按写入核心、任务查询、调用查询、Dashboard 聚合和
  筛选规则拆为五个文件；最大文件降至 261 行。
- 最终后端最大生产文件为 `spacestats/repository.go`（652 行），不存在超过 800 行的生产 Go 文件。
- 最终整体语句覆盖率为 42.8%；新增重点覆盖率：`app` 84.9%、`config` 93.1%、`bootstrap` 81.0%、
  `db` 73.1%、`wechat` 94.7%。
- 更新 `backend/README.md`，补充后端测试/竞态/静态门禁和本路线图入口。
- 最终验证通过：`go test ./... -count=1`、`go test -race ./... -count=1`、`go vet ./...`、
  `gofmt -d internal cmd`、`git diff --check`；真实应用集成测试已顺序应用全部现有迁移。
- 兼容性结论：未修改 HTTP 路由、JSON 契约、数据库迁移、环境变量名称或默认业务策略；本轮仅调整内部职责边界、
  提升测试覆盖并统一两个既有文件的 Go 格式。
