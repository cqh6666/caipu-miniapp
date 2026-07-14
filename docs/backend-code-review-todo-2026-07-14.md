# Go 后端代码审查与重构待办

- **审查时间**：2026-07-14 22:26:46 +0800
- **多 Agent 复审时间**：2026-07-14（安全、数据一致性、运维与 AI 链路专项）
- **审查范围**：`backend/` 生产代码、测试、迁移、配置、运维脚本及相关部署文档
- **代码基线**：`d5806cd`（复审收尾时 HEAD；本次未修改运行时代码）
- **关联文档**：`docs/backend-refactor-roadmap-2026-07-14.md`
- **状态**：多 Agent 复审完成，待排期；P0 项应优先于原路线图 R10～R15

## 1. 结论

后端已完成 R0～R9 的职责拆分，当前包边界、组合根和基础测试明显优于重构前，
没有必要再次按文件行数做大规模拆分。下一阶段应先处理已确认的安全与一致性问题，
再补自动化门禁和运维恢复能力。

两轮审查共确认 5 个 P0 工作包：

1. 用户 JWT 在默认配置下可被管理员中间件接受，存在权限提升风险。
2. 打卡点和菜谱图片镜像可访问任意网络地址，存在 SSRF 风险。
3. B 站直连解析的主机校验可被伪后缀绕过，任意登录用户可触发 SSRF。
4. 自动解析结果没有任务所有权和人工编辑保护，可覆盖用户刚保存的菜谱内容。
5. 当前工具链与依赖命中 12 个可达漏洞，需要升级 Go 补丁版本和
   `golang.org/x/image`。

其余高价值事项集中在配置 fail-closed、写入并发、成员权限事务边界、真实迁移契约、
请求资源边界、日志脱敏、错误可观测性、备份恢复和 CI。`admin` handler、外部 HTTP
客户端和 worker 生命周期仍值得重构，但必须排在行为修复与契约测试之后。

多 Agent 复审采用三条独立只读审查线：安全与 HTTP 边界、数据一致性与 SQLite、
运维与可观测性；主审负责逐项回读源码、去重和分级。小红书伪后缀当前只会被转发给
sidecar，后端本身不会访问该目标，因此仅列入输入校验加固，不作为已确认的后端 SSRF。

## 2. 验证基线

| 检查项 | 结果 |
| --- | --- |
| `go test ./... -count=1` | 通过 |
| `go test -race ./... -count=1` | 通过 |
| `go vet ./...` | 通过 |
| `go mod verify` | 通过 |
| `govulncheck ./...` | 失败，命中 12 个可达漏洞 |
| `bash -n scripts/*.sh backend/scripts/*.sh` | 通过 |
| 整体语句覆盖率 | 42.8% |
| SQLite 外键最小复现 | `updated_by=0` 插入失败，确认 Admin B 站配置缺陷 |

重点低覆盖包：`invite` 12.0%、`kitchen` 12.0%、`admin` 18.6%、
`auth` 27.0%、`upload` 29.9%、`common/profile` 0%。覆盖率只作为定位信号，
后续验收以关键行为断言为准，不设置脱离风险的统一数字目标。

## 3. 优先级定义

| 级别 | 定义 | 处理要求 |
| --- | --- | --- |
| P0 | 可造成越权、内网访问、已知漏洞利用或严重数据风险 | 修复后再发布 |
| P1 | 可造成拒绝服务、并发错误、敏感信息泄漏或故障不可恢复 | 下一迭代完成 |
| P2 | 中期可维护性、容量和治理问题 | 满足进入条件后实施 |

## 4. P0：发布前处理

### SEC-001 管理员令牌与用户令牌彻底隔离

**已确认问题**

- `internal/config/validation.go:64` 在未配置 `ADMIN_JWT_SECRET` 时回退到
  `JWT_SECRET`。
- `internal/admin/middleware.go:45` 同时接受管理员 Cookie 和 Bearer Token。
- `internal/admin/token.go:47` 只验证签名、有效期和非空 `sub`，没有验证
  `issuer`、`audience`、令牌用途或配置中的管理员账号。
- `internal/auth/token.go:35` 签发的普通用户 JWT 同样使用 HS256，并带非空
  `sub`。当两类令牌共用默认密钥时，普通用户 Bearer Token 可通过管理员中间件。
- `internal/config/config.go:98,109` 将 `APP_ENV` 和 `JWT_SECRET` 分别默认成
  `local` 与公开值 `dev-secret-change-me`，`validation.go:8-74` 未拒绝生产环境沿用。
  漏配密钥时，攻击者可以自行签发任意用户 ID 的令牌。
- `internal/config/validation.go:61` 还会让 `CREDENTIALS_SECRET` 回退到
  `JWT_SECRET`；数据库或备份泄露后，公开/泄露的 JWT 密钥可用于离线解密 B 站、
  AI、SMTP 和 sidecar 等运行时凭据。
- 现有应用集成测试显式使用两把不同密钥，因此没有覆盖默认回退路径。

**待办**

- [ ] 部署配置必须显式声明环境；非 `local` 环境强制配置独立、高熵的
  `JWT_SECRET`、`ADMIN_JWT_SECRET` 和 `CREDENTIALS_SECRET`，禁止互相回退，
  禁止缺失、公开默认值或弱密钥启动。
- [ ] 用户与管理员 Claims 增加固定 `iss`、`aud`、`token_use`，解析时全部校验。
- [ ] 管理员中间件校验 `sub` 必须等于当前配置的管理员账号。
- [ ] 评估是否还需要管理员 Bearer Token；如后台仅使用 Cookie，删除该兼容入口。
- [ ] 轮换线上三类密钥，使已签发旧令牌立即失效。
- [ ] 为凭据密文增加 key version 和重加密流程，避免轮换
  `CREDENTIALS_SECRET` 后历史配置全部不可读。
- [ ] 将 `APP_SETTINGS_ACCESS_MODE` 的生产默认值由 `all` 调整为 `admin`，
  明确白名单模式的使用场景。

**验收标准**

- 普通用户 JWT 访问任意 `/api/admin/*` 均返回 401。
- 管理员 JWT 访问普通用户接口同样返回 401。
- 错误 `iss/aud/token_use/sub`、旧密钥和弱默认密钥都有回归测试。
- 非本地环境缺少任一必需密钥时，进程在监听端口前失败并给出明确配置错误。
- JWT 密钥不能解密运行时凭据；凭据密钥轮换可回滚且不丢失现有配置。

### SEC-002 封堵远程图片下载 SSRF

**已确认问题**

- `internal/place/service.go:227` 会在用户创建或更新打卡点时同步下载其图片 URL。
- `internal/recipe/image_mirror_worker.go:149` 会异步下载菜谱中的远程图片 URL。
- 两条链路最终进入 `internal/upload/service.go:68`；当前仅检查 URL 非空，默认
  `http.Client` 会跟随重定向，没有限制协议、回环地址、私网、链路本地地址、
  云元数据地址或重定向后的目标。

**待办**

- [ ] 抽取图片下载专用 egress policy，只允许明确的 `https`/必要 `http` 场景。
- [ ] 在实际 `DialContext` 解析并校验目标 IP，拒绝 loopback、private、
  link-local、multicast、unspecified 和云元数据网段，避免 DNS 重绑定绕过。
- [ ] 每次重定向重新校验目标，限制重定向次数，并拒绝协议降级。
- [ ] 保留响应字节上限，同时增加连接、TLS、响应头和总请求超时。
- [ ] 对被拒绝地址记录脱敏安全日志；打卡点同步链路不得静默丢弃原图片而不告知调用方。

**验收标准**

- `127.0.0.1`、`::1`、RFC1918、链路本地、`169.254.169.254`、私网域名解析、
  公网到私网重定向和 DNS 重绑定测试全部被拒绝。
- 合法公网图片、大小超限、非图片、超时、取消和重定向上限均有测试。
- `go test -race ./internal/upload ./internal/place ./internal/recipe` 通过。

### SEC-004 封堵 B 站直连解析 SSRF

**已确认问题**

- 任意已登录用户均可调用链接预览/解析入口；当前不要求目标空间成员资格。
- `internal/linkparse/bilibili.go:425` 使用 `strings.Contains` 判断主机，
  `bilibili.com.attacker.example`、`b23.tv.attacker.example` 等伪后缀会通过。
- 当 sidecar 未启用或未配置时，`bilibili.go:353-369,430-445` 会由后端直接 GET
  该 URL；默认 client 跟随重定向，未在每跳复核目的地址。示例配置默认关闭 sidecar。
- `internal/linkparse/platform.go:76` 对小红书也有相同的伪后缀判断，但当前后端只把
  输入转发到已配置 sidecar；sidecar 未配置时直接报错，不能据此认定后端直接 SSRF。

**待办**

- [ ] 使用 `URL.Hostname()` 做精确主机或点边界后缀匹配，拒绝 userinfo、异常端口、
  伪后缀和非 HTTP(S) scheme。
- [ ] 复用 SEC-002 的 egress policy，在 DNS/Dial 阶段拒绝私网、回环、链路本地、
  云元数据地址，并对每次重定向重新校验。
- [ ] 对 B 站返回的字幕 URL 建立独立域名白名单；只对可信 B 站域发送 SESSDATA，
  防止异常上游响应把 Cookie 带到第三方主机。
- [ ] 修正小红书 host 校验并验证 sidecar 自身执行等价的目的地址策略；在拿到
  sidecar 代码前，把它记为待验证边界，不宣称已修复。

**验收标准**

- 伪后缀、userinfo、IPv4/IPv6 字面量、DNS 重绑定、公网转私网重定向全部被拒绝。
- 合法 B 站长链、短链和字幕仍可解析；不可信字幕地址收不到 SESSDATA。
- `go test ./internal/linkparse -count=1` 覆盖直连与 sidecar 两种模式。

### DATA-003 为异步菜谱任务增加所有权与人工编辑保护

**已确认问题**

- `internal/recipe/service.go:171-210` 允许用户在 `processing` 期间保存人工编辑；
  `repository_auto_parse.go:248-307` 随后只按菜谱 ID 整块覆盖标题、摘要、图片、
  食材和步骤，并无条件把 `parsed_content_edited` 清零，可造成用户数据丢失。
- `repository_auto_parse.go:182-245` 和 `repository_flowchart.go:218-288` 会回收超时任务，
  但成功、失败和重试写入没有 claim 所有权条件。旧执行者 A 被回收、执行者 B 重新
  claim 后，A 仍能覆盖 B 的状态、解析结果或流程图。

**待办**

- [ ] 为任务持久化不可复用的 claim token/generation 与 lease；所有成功、失败、重试
  终态用 CAS 更新，`RowsAffected=0` 表示结果已过期并安全丢弃。
- [ ] 自动解析完成写必须同时校验 claim、`processing` 状态和人工编辑版本；发现人工
  编辑后保留用户内容，不清空 `parsed_content_edited`，并记录 stale/conflict 结果。
- [ ] 业务字段与 worker 状态使用不同更新边界，避免普通菜谱更新回滚异步状态。
- [ ] 外部调用较长时支持续租，但即使 stale 阈值大于超时也必须保留 fencing 校验。

**验收标准**

- 阻塞 parser，夹入人工 Update 后再释放，断言人工内容保留且旧任务不能改终态。
- A claim -> stale requeue -> B claim -> A 成功/失败的确定性并发测试中，A 的更新为 0，
  B 独占终态；自动解析和流程图均覆盖。
- 多进程或多 worker 实例下行为与单进程一致，`go test -race ./internal/recipe` 通过。

### DEP-001 升级存在可达漏洞的工具链与依赖

**已确认问题**

`govulncheck` 在当前环境 `go1.26.1` 下命中 12 个可达漏洞，其中标准库修复版本
最高要求 `go1.26.5`；`golang.org/x/image v0.38.0` 命中
`GO-2026-4962`，修复版本为 `v0.39.0`。调用链涉及 TLS、HTTP/2、SMTP、
证书校验和邀请图片字体解析。

**待办**

- [ ] 本地、CI 和生产构建机统一升级到 Go `1.26.5` 或更高兼容补丁版本。
- [ ] 将 `golang.org/x/image` 升级到 `v0.39.0` 或更高兼容版本并整理 `go.mod`。
- [ ] 重新执行全量测试、竞态测试、`go vet` 和 `govulncheck`。
- [ ] 在 CI 中固定 Go 补丁版本并增加漏洞扫描，避免只声明 `go.mod` 最低版本。

**验收标准**

- `govulncheck ./...` 不再报告可达漏洞；若有接受项，必须记录影响、缓解措施和到期时间。
- 生产二进制通过 `go version -m` 可确认使用已修复工具链构建。

## 5. P1：下一迭代完成

### HTTP-001 建立请求资源边界与入口防护

**证据**：`internal/common/request.go:10` 的 JSON 解码没有字节上限；
`internal/app/app.go:208` 仅设置 `ReadHeaderTimeout` 和 `IdleTimeout`；登录入口没有
应用层限流。字段长度校验发生在完整解码之后，无法阻止大请求先占用内存。

- [ ] 为 JSON 请求设置统一默认上限，并允许上传、AI 流式接口按路由覆盖。
- [ ] 超限统一返回 413；补多 JSON 值、尾随垃圾、未知字段和空 body 契约测试。
- [ ] 配置合理的 body 读取超时、最大 Header，并确认 SSE 不受全局写超时误伤。
- [ ] 对管理员登录、微信登录、邀请码预览/接受增加按 IP 和账号维度的限流与短期封禁。
- [ ] nginx 与应用层上限保持一致，应用不能依赖代理作为唯一防线。

### DATA-001 修复邀请接受的并发计数丢失

**证据**：`internal/invite/service.go:108` 在事务外读取邀请；
`internal/invite/repository.go:156` 使用旧 `UsedCount` 计算新值，并在
`repository.go:162` 无条件覆盖。并发请求可基于同一旧值加入多个成员，导致计数丢失
并突破 `maxUses`。

- [ ] 在同一事务内重新读取并校验邀请状态、过期时间和使用次数。
- [ ] 使用条件更新或原子自增，更新失败时返回“已用完/已过期/已撤销”的确定错误。
- [ ] 保持“已经是成员不重复消耗次数”的幂等语义。
- [ ] 增加多 goroutine 接受同一 `maxUses=1` 邀请的 SQLite 集成测试。

**验收标准**：任意并发度下新增成员数和 `used_count` 都不超过 `max_uses`，
且不存在已加入成员但计数未提交的部分成功状态。

### DATA-004 收紧默认空间、并发更新与成员权限的事务边界

**已确认问题**

- `internal/kitchen/service.go:46-56` 先查询再创建默认空间，创建事务没有重查，
  `migrations/001_init.sql:10-17` 也没有“每用户一个默认空间”的唯一约束；首次并发
  登录可创建多个默认空间。
- 菜谱 `service.go:171-210` -> `repository.go:147-226` 和地点
  `place/service.go:132-200` -> `place/repository.go:137-208` 都是先读完整对象再整行覆盖，
  两个成员修改不同字段时后提交者会静默恢复前一个成员的字段；菜谱还可能回滚 worker 状态。
- 菜谱、地点等写链路先调用 `EnsureMember`，之后在独立事务提交，UPDATE 只按实体 ID；
  权限检查后并发退出空间的用户仍可在竞态窗口完成写入。

**待办与验收**

- [ ] 增加显式默认空间标识和唯一部分索引，或等价 bootstrap marker；ensure 在事务中
  原子执行，唯一冲突后回读。20 goroutine 同用户并发时只能得到一个默认空间。
- [ ] 为菜谱、地点增加 `version` 乐观锁并返回 409，或改成明确的 PATCH/字段级更新；
  两个客户端基于同一版本提交时必须一方冲突，禁止静默丢失更新。
- [ ] create/update/delete 等关键写入在同一事务重新检查 membership，或在 SQL 加
  `EXISTS` 条件；用 barrier 让退出发生在权限检查和提交之间，断言写入失败。
- [ ] 审查 mealplan、invite 等同类链路，但只在验证存在写入窗口后扩展改动范围。

### DB-002 修复 Admin B 站配置的真实迁移契约

**已确认问题**

`migrations/004_add_app_bilibili_settings.sql:9-11` 要求 `updated_by` 非空且引用
`users(id)`；Admin runtime settings handler 在 `internal/admin/handler.go:562,576`
固定传 `updatedBy=0`。正式数据库通过 `internal/db/db.go:55` 开启外键，因此管理员首次
更新或清空配置会失败。最小 SQLite 复现已得到 `FOREIGN KEY constraint failed`；现有
service 测试自建表没有复制该外键，因而产生假绿。

- [ ] 将管理员操作者建模为可空用户 ID + 必填 subject，或迁移为统一 subject 字段；
  不创建伪造的 `users.id=0` 记录规避约束。
- [ ] 测试必须从全量迁移创建真实 SQLite，覆盖用户端和管理员端 update/clear，最后执行
  `PRAGMA foreign_key_check`。
- [ ] 将“schema 约束被简化的单元测试”纳入 TEST-001 清单，避免测试 DDL 与生产漂移。

### CFG-002 配置加载与生产默认值 fail-closed

**已确认问题**

- `internal/config/env.go:11-24` 固定读取 `.env`、`configs/local.env` 和
  `APP_ENV_FILE`，并用 `godotenv.Overload` 覆盖 systemd/容器已注入的变量；显式文件
  不存在、权限错误或解析失败也会被忽略。
- `env.go:36-77` 对非法 int/float/bool 静默使用默认值。例如镜像开关拼错后可能从
  预期关闭变成默认开启。
- `config.go:98,106` 默认 `APP_ENV=local`、`APP_SETTINGS_ACCESS_MODE=all`；若线上同时
  漏配，`router.go:96` 会暴露 dev-login，任意匿名者可取 token，随后按
  `auth/service.go:262-276` 修改全局 B 站凭据。

**待办与验收**

- [ ] 仅明确的 local 模式自动读取 local dotenv；进程环境优先。显式
  `APP_ENV_FILE` 不存在、不可读或格式错误时拒绝启动，并记录脱敏的配置来源摘要。
- [ ] typed parser 返回并聚合错误；`flase`、`1O`、`NaN` 等非法值必须在监听前失败，
  错误包含变量名但不回显 secret。
- [ ] 生产默认 app settings 权限改为 `admin`，禁止 `all`；部署验收必须确认 dev-login
  为 404、普通用户写全局配置为 403。
- [ ] 本地与服务器 secret 文件权限设为 `0600`；当前未跟踪的
  `backend/configs/local.env` 权限为 `0644`，应单独评估暴露面与密钥轮换，文档不记录值。
- [ ] 以表驱动测试覆盖 OS env、默认 dotenv、显式 env file 和非法值的完整优先级矩阵。

### SEC-003 日志与上传公开边界收口

**证据**：`internal/middleware/logger.go:43` 记录原始 URL path 和 query；邀请 token、
邀请码、菜谱分享 token 都位于 path。`internal/app/router.go:89` 直接使用
`http.FileServer`，会提供目录列表。`internal/upload/handler.go:43` 信任
`X-Forwarded-Host`/`X-Forwarded-Proto` 生成公开 URL。

- [ ] 日志记录路由模板或对 token/code 段做不可逆脱敏，不记录原始敏感 query。
- [ ] 禁止上传目录列表；明确图片是“公开资源”还是“空间成员资源”。
- [ ] 若保持公开资源，增加安全响应头、缓存策略和不可枚举验证；若改为私有资源，
  设计鉴权或短期签名 URL，不能直接在现有路径上突然加鉴权破坏前端。
- [ ] 非本地环境强制配置合法的 `UPLOAD_PUBLIC_BASE_URL`，不要从任意 Host Header
  构造持久化 URL；只信任明确的反向代理。
- [ ] 上传服务自身严格执行单文件大小限制，并在写入失败时删除半成品文件。

### SEC-005 管理端 CSRF、令牌撤销与可信代理边界

**已确认/条件式风险**

- Admin Cookie 在 `internal/admin/service.go:60-69` 使用 `SameSite=Lax` 且 Path 为 `/`；
  管理写接口没有 Origin/Fetch Metadata/CSRF 校验。同一主域存在可控兄弟子域时，
  浏览器仍可能携带同站 Cookie 发起跨源表单 POST。
- 用户 Bearer Token 默认有效 720 小时，只有 `exp`，没有 `jti`、token version、
  refresh session 或服务端撤销机制；泄露后用户无法主动失效。
- `chi/middleware.RealIP` 无可信代理边界地接受转发头；当前会污染审计 IP，未来若直接
  用于登录限流，会被客户端伪造头绕过。

- [ ] 管理写接口校验 Origin/Fetch Metadata 或使用 CSRF token；Cookie 收紧为
  `SameSite=Strict`、`Path=/api/admin`，同时保持受控 Bearer API 客户端兼容。
- [ ] 采用短期 access token + 可撤销 refresh/session，或至少增加用户 token version；
  logout、撤销、版本递增后旧 token 必须返回 401。
- [ ] 仅对明确可信反向代理解析转发头，直连伪造 `X-Forwarded-For` 不改变客户端 IP。

### OBS-001 统一错误出口与结构化日志

**证据**：服务层普遍包装了根因，但 `internal/common/response.go:32` 丢弃内部错误，
`internal/middleware/logger.go:40` 只记录状态码；普通 500 很难通过 request ID 还原。

- [ ] 引入请求级错误响应器或最小中间件，在保持客户端通用错误的同时记录错误链。
- [ ] 日志至少包含 request ID、路由模板、错误类型、状态码、耗时和安全的业务对象 ID。
- [ ] 建立集中脱敏规则，禁止记录 JWT、Cookie、API Key、SESSDATA、完整 AI 请求体。
- [ ] 修正 `WriteJSON` 在响应已写出后再次尝试发送 500 的无效行为，并测试编码失败路径。

### AI-001 限制上游响应并脱敏 SSE 错误

**已确认问题**

- `internal/airouter/openai_client.go:175-205`、`internal/linkparse/bilibili.go:581-596`
  等成功响应直接交给 JSON Decoder，没有响应体上限；图片生成响应可能包含大体积 base64，
  异常或被接管的上游可造成进程内存持续膨胀。
- `internal/dietassistant/service.go:282-288` 把上游非 2xx body 直接作为公开 AppError，
  `handler.go:122-129,192-198` 又通过 SSE 原样返回；上游 HTML、内部路由信息或供应商
  调试内容会暴露给用户。现有普通 JSON 错误脱敏不覆盖这条已写 200 的 SSE 链路。

- [ ] 为各类上游成功响应设置按场景上限；使用可识别“刚好到上限”的 bounded reader，
  超限返回稳定的 bad gateway 错误并停止解码。图片 base64 上限需结合最终图片限制测算。
- [ ] 上游原始错误只进入脱敏日志/审计，SSE 对客户端返回稳定业务文案和 request ID；
  不转发 HTML、API key、主机名或原始 provider body。
- [ ] 对流式响应限制单事件大小、累计可见内容和工具参数大小，保持 context 取消有效。
- [ ] 使用超大 JSON/base64、无限 SSE 事件和含敏感字段的错误体做定向回归及内存基准。

### OPS-001 建立可恢复的 SQLite 备份与发布流程

**证据**：`backend/scripts/backup.sh:9` 直接复制运行中 WAL 模式数据库的主文件，
没有在线备份协议、完整性校验或恢复演练。`backend/scripts/deploy.sh:70` 标注“重启”
但使用 `systemctl enable --now`，服务已运行时不会执行 restart；相关旧文档仍推荐该脚本。
当前服务器侧发布脚本也缺少发布前测试、数据库备份和健康失败自动回滚。

- [ ] 使用 SQLite Online Backup API、`sqlite3 .backup` 或等价一致性快照，禁止只复制主文件。
- [ ] 备份数据库和上传文件，增加保留周期、校验和、异机副本及定期恢复演练。
- [ ] 发布前执行配置校验、迁移预检和一致性备份；使用版本化 release 目录原子切换二进制。
- [ ] 健康检查失败时自动恢复上一二进制，并明确数据库迁移的前向兼容/人工回滚策略。
- [ ] 修复或废弃 `backend/scripts/deploy.sh`，统一 README、部署文档和线上实际入口。

### OPS-002 增加 readiness 并以最小权限运行服务

**已确认问题**

- `internal/app/router.go:79-91` 的健康接口是静态 200；发布脚本用它判断成功，数据库
  已关闭、路径权限错误或迁移不完整时仍可能绿。
- `backend/scripts/bootstrap-server.sh:92-107` 和部署文档生成的 systemd unit 未设置
  `User/Group`，默认以 root 运行，也没有文件系统与提权沙箱。

- [ ] 分离 `/livez` 与 `/readyz`；readiness 用短超时检查 DB ping、迁移版本及必要目录，
  故障返回 503。发布脚本要求连续 N 次 ready，并核对目标 release ID。
- [ ] 建立专用不可登录用户，只授权数据库/WAL/SHM、uploads 和必要临时目录；增加
  `NoNewPrivileges`、`PrivateTmp`、`ProtectSystem`、`ProtectHome`、`ReadWritePaths`、
  `UMask` 和匹配应用 deadline 的 `TimeoutStopSec`。
- [ ] 验收进程 UID 非 0、无法写 `/etc` 或 release 目录，同时上传、字体读取、SQLite WAL、
  重启与回滚正常；关闭 DB/破坏权限时 readiness 必须失败而 liveness 保持可用。

### TEST-001 补核心契约测试并接入 CI

- [ ] 优先补 SEC-001、SEC-002、SEC-004、DATA-003、DATA-001、DB-002、HTTP-001
  对应的回归测试。
- [ ] 补 `admin/auth/invite/kitchen/upload/common/profile` 的 handler、service 和真实迁移
  SQLite 集成测试，覆盖状态码、JSON、Cookie、权限和事务边界。
- [ ] CI 执行 `gofmt` 检查、`go test ./...`、`go vet ./...`、
  `go test -race ./...`、覆盖率采集、`go mod verify` 和 `govulncheck ./...`。
- [ ] 不把整体覆盖率阈值作为首要门禁；先对关键安全与业务分支设置明确断言。

## 6. P2：满足进入条件后实施

### ARC-001 Admin HTTP 边界收口

在 TEST-001 锁定契约后，将 `internal/admin/handler.go` 按 auth、dashboard、audit、
runtime settings、AI routing/alert 拆分，并以窄接口替代大范围具体类型依赖。
保留统一 `Handler` 和现有路由签名，不改前端契约。

### HTTP-002 外部 HTTP 适配器可测性

为 `airouter`、`appsettings/runtime_probe` 等请求路径临时创建客户端的模块增加包内
`HTTPDoer`/Options 注入，复用 Transport，统一响应上限和错误分类。不要建立跨业务的
“万能 HTTP 客户端”；SSRF egress policy 只用于处理不可信目标地址的链路。

### WORKER-001 Worker 生命周期一致化

先补三个菜谱 worker 的启停、取消、非法间隔和 goroutine 退出测试，再提取最小的
包内 loop/clock 能力。`Stop` 应接受 context 或有明确等待上限，避免 worker 不退出时
阻塞整个优雅停机流程。当前 `internal/app/app.go:241-258` 在调用
`http.Server.Shutdown(ctx)` 前先同步执行三个无 context 的 `Stop()`，所以 main 创建的
10 秒 deadline 实际约束不到 worker，等待期间 HTTP 还可能继续接流量。验收应先停止
接流量，再在统一 deadline 内取消并等待 worker，最后关闭 DB；注入不退出 worker 时
进程也必须在 deadline 内返回明确错误。

### DATA-002 数据生命周期与查询容量治理

- 为 `ai_job_runs`、`ai_call_logs`、配置审计和告警事件定义保留期、清理批次与归档策略。
- 当单空间菜谱、打卡点或菜单达到实测阈值后，再为列表接口增加游标分页；当前不为未来
  规模提前引入复杂搜索系统。
- 只有统计维度继续增长且查询计划出现退化时，才拆分 `spacestats/repository.go` 或调整索引。

### DATA-005 配置审计、告警通知与管理写入并发

- `internal/appsettings/service.go:79-103,122-141` 先提交 B 站配置，再单独插入审计且
  忽略错误；应在同一事务读取旧值、写配置和审计，审计失败整体回滚。
- `internal/aialert/service.go:238-283` 在发送邮件后才标记已告警，并发失败跨过阈值时
  可能重复发信；使用原子 claim/outbox 和 event ID 幂等，发送失败可重试。
- AI 路由和运行时配置是整组 last-write-wins；两个管理员同时保存会静默覆盖且审计
  oldValue 过时。响应携带 version/updatedAt，保存 CAS，冲突返回 409。
- 餐单在事务外验证 recipe 后保存快照。先确认产品是否允许引用随后删除的菜谱；若允许，
  把它记录成历史快照契约；若不允许，在 Replace 事务内复查，不做含糊修复。

### DB-001 迁移治理

- `schema_migrations` 增加文件校验和，启动时拒绝已应用迁移被静默修改。
- 现有两个 `019_*` 文件保持原名，避免破坏已部署记录；对后续迁移增加序号唯一性检查。
- 在 CI 中从空库顺序应用全量迁移，并验证代表性历史库升级和 `PRAGMA integrity_check`。

### CFG-001 配置分组

仅当配置继续增长或出现模块参数漏接时，将扁平 `config.Config` 映射为 Auth、AI、
Linkparse、Recipe、Upload 等只读模块配置。保留环境变量名称与默认业务策略，不引入
DI 框架，不让业务包反向依赖全局配置。

### OPS-003 发布身份、健康探测配置与日志留存

- 用 `-trimpath` 和 ldflags 注入 commit、build time、Go toolchain、release ID；发布清单
  保存二进制 SHA-256 和迁移集合，启动日志/受控版本接口可映射到唯一构建。
- `internal/admin/server_health.go:104-110` 硬编码 `caipu-backend` 与 8080，但部署脚本
  允许覆盖服务名/端口，另一 bootstrap 默认名也不同；改为显式配置或进程内依赖检查。
- 明确 journald 容量和保留期或集中采集策略，为 5xx、worker 连续失败、磁盘空间、
  备份年龄与恢复校验建立告警；日志必须可按 request ID/release ID 查询。

## 7. 推荐执行顺序

| 阶段 | 工作包 | 完成门禁 |
| --- | --- | --- |
| 0：紧急安全 | SEC-001、SEC-002、SEC-004、DATA-003、DEP-001 | 安全、数据保护、竞态和漏洞门禁通过 |
| 1：入口与一致性 | HTTP-001、DATA-001、DATA-004、DB-002、CFG-002、SEC-003、SEC-005、AI-001 | 契约、并发、配置和脱敏测试通过 |
| 2：恢复与门禁 | OBS-001、OPS-001、OPS-002、TEST-001 | CI 绿；完成备份恢复、最小权限和回滚演练 |
| 3：维护性 | ARC-001、HTTP-002、WORKER-001 | 现有契约不变，定向与全量门禁通过 |
| 4：条件演进 | DATA-002、DATA-005、DB-001、CFG-001、OPS-003 | 有容量、并发或运维证据后启动 |

## 8. 当前不建议做

- 不继续按“超过多少行”机械拆文件；当前最大生产文件职责仍可解释。
- 不引入 ORM、DI 框架、消息队列或新 Web 框架。
- 不为所有 repository/service 预先创建接口，只在替代实现或测试边界明确时引入。
- 不在缺少 `EXPLAIN QUERY PLAN` 和生产数据量证据时调整 SQLite 索引或加缓存。
- 不把 SQLite 立即迁移到 MySQL 作为上述安全、并发和备份问题的替代方案。

## 9. 每个工作包的统一收尾门禁

```bash
cd backend
gofmt -w <本次修改的 Go 文件>
go test ./... -count=1
go vet ./...
go test -race ./... -count=1
go mod verify
govulncheck ./...
git diff --check
```

涉及迁移、备份或部署时，还必须补充空库迁移、历史库升级、备份恢复、健康失败回滚的
实际演练记录；未验证项不能仅以“代码看起来正确”关闭。
