# Go 后端代码审查与重构待办

- **审查时间**：2026-07-14 22:26:46 +0800
- **多 Agent 复审时间**：2026-07-14（安全、数据一致性、运维与 AI 链路专项）
- **审查范围**：`backend/` 生产代码、测试、迁移、配置、运维脚本及相关部署文档
- **代码基线**：`d5806cd`（复审收尾时 HEAD；本次未修改运行时代码）
- **关联文档**：`docs/backend-refactor-roadmap-2026-07-14.md`
- **进度更新时间**：2026-07-16 13:42:26 +0800
- **状态**：P0/P1、阶段 3 维护性工作包及阶段 4 中有直接证据的 DATA-005、DB-001、
  OPS-003 仓库内实现、核心契约测试、恢复发布链路、开发机门禁、Backend CI 首次运行、
  生产 Go 工具链、最小权限版本化发布与 journald 容量治理已完成；待线上密钥轮换、
  仓库外 sidecar、异机备份/真实回滚/故障 readiness 和 ops-health 外部告警后关闭发布门禁

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

### 1.1 当前实施进度（2026-07-16）

| 工作包 | 仓库内进度 | 仍待完成 |
| --- | --- | --- |
| SEC-001 | JWT 全字段校验、三类密钥隔离、凭据轮换工具、配置权限默认收紧已落地；Admin Web 仅使用 HttpOnly Cookie，已删除管理员 Bearer 入口并补双向令牌隔离集成测试 | 执行线上三类密钥轮换并验证旧令牌失效 |
| SEC-002 | 受限 DNS/Dial、重定向复核、超时、脱敏安全日志和打卡点错误回传已落地；公网图片、大小、类型、超时、取消、重定向上限和公网转私网均有测试 | 仓库内完成；发布后确认安全日志正常采集 |
| SEC-003 | 请求日志改为路由模板且不记录 query；上传保持随机 URL 公开资源，目录/非生成文件不可访问并带安全缓存头；生产公网基址和服务层大小/半成品清理均已收口 | 线上已确认 `UPLOAD_PUBLIC_BASE_URL` 使用正式 `/caipu-uploads` HTTPS 前缀 |
| SEC-004 | B 站/小红书 host 标签边界、B 站直连 egress policy、字幕域白名单与 SESSDATA 限制已落地；直连长链、短链、字幕及 sidecar 模式均有回归 | 验证仓库外 sidecar 具备等价目的地址策略 |
| SEC-005 | Admin Cookie-only 写请求使用会话绑定 CSRF token 与 Fetch Metadata，Cookie 改为 Strict/窄 Path；用户 JWT 增加 jti/token version，注销与版本递增可立即撤销；可信代理边界已落地 | 仓库内完成；发布前设置现网 `ADMIN_COOKIE_PATH=/caipu-api/admin`，发布后确认旧用户 token 要求重新登录 |
| DATA-003 | claim token、lease、内容版本、CAS 终态、人工编辑保护和长任务续租已落地；自动解析与流程图均验证旧 claim 失效 | 仓库内完成；发布后观察 lease 续租失败日志 |
| DATA-004 | 默认空间使用显式标记和唯一部分索引原子 ensure；菜谱/地点使用独立 version CAS；菜谱、地点、菜单、邀请和空间关键写入在事务内复核成员资格 | 仓库内完成；发布后关注 409 比例，确认小程序刷新后重试提示符合实际协作场景 |
| OBS-001 | `WriteError` 原始错误通过 ResponseWriter 进入统一请求日志；请求 ID、路由、状态、耗时、错误类型链和安全对象 ID 已结构化，全局 slog 集中脱敏；JSON 预编码避免二次写 500 | 仓库内完成；发布后确认日志平台可按 `request_id` 检索且脱敏字段符合预期 |
| AI-001 | 文本、图片 base64、B 站/Sidecar、微信、地图和分享页上游均有 `max+1` 响应边界；饮食管家限制 SSE 单事件、累计可见内容和工具参数，公开错误只返回稳定文案与 request ID | 仓库内完成；发布后观察 `response_too_large` 与 SSE 失败日志，按真实 Provider 响应容量调整常量前必须补内存回归 |
| OPS-001 | SQLite Online Backup 打包 DB/uploads/校验和/元数据，支持保留期与 rsync；生产已完成本机备份、恢复校验、迁移预演、原子 release 并启用每日备份/每周恢复 timer | 配置异机目标，完成真实异机副本恢复和发布失败回滚演练 |
| OPS-002 | `/livez` 与 `/readyz` 已分离；生产已迁移专用用户/current unit，验证 UID、沙箱写边界、字体、WAL、重启和内外 readiness | 执行生产故障 readiness/liveness 分离与真实回滚演练 |
| TEST-001 | P0/P1 安全、并发、迁移、入口以及 admin/auth/profile/kitchen/invite/upload/common 契约均有明确断言；Backend CI 已在 `2ab697f` 与 `c928d19` 成功运行 | 仓库与 CI 完成；不设脱离风险的统一覆盖率阈值 |
| ARC-001 | Admin Handler 已按 auth、dashboard/audit、runtime、AI routing、AI alert 拆为域文件，依赖改为窄接口，各域可独立 httptest | 仓库内完成；发布后观察 Admin API 5xx/4xx 契约是否与基线一致 |
| HTTP-002 | airouter/appsettings 使用包内 HTTPDoer + Options 注入与共享 Transport，请求 timeout 由 context 管理，响应边界和错误分类有协议测试 | 仓库内完成；发布后观察 Provider timeout/network/response-too-large 分类 |
| WORKER-001 | 三个 recipe worker 复用包内 lifecycle，Start 校验 interval，Stop 接受 context；App 先停 HTTP、并发停 worker、最后关 DB | 仓库内完成；生产重启时确认 15 秒 systemd 停机上限与 worker 日志 |
| DATA-005 | B 站配置/审计已同事务提交；AI 告警使用持久化 outbox、原子 claim 和退避重试；AI 路由、运行时配置及 B 站管理写入使用整数 version CAS，值/version 从同一 SQLite 快照读取，冲突返回 409，Admin Web 刷新且不自动重放 | 仓库内完成；SMTP 在“发送成功、sent 状态落库前进程崩溃”时仍有协议层重复窗口，发布后观察 outbox 重试和 409 比例 |
| DB-001 | `schema_migrations` 已增加 SHA-256，旧表启动时自升级并回填；已应用文件不一致拒绝启动；除两个历史 `019_*` 外序号必须唯一；空库和 `014` 历史库升级均执行 integrity check | 仓库内完成；发布前保留 migration 文件原始字节，禁止修改已发布 SQL |
| OPS-003 | release 注入 release/commit/build time/Go toolchain；生产构建身份、备份/恢复 timer 与 journald 512M/14day 已验证，manifest 固化二进制和迁移集合 | 安装并实测 ops-health timer，接入外部告警接收端并验证 request/release ID 检索 |
| DEP-001 | 开发机、Backend CI 与生产模块构建均使用 Go `1.26.5`，生产二进制和 manifest 已核对；`x/image v0.39.0`，全量门禁通过 | 开发机、CI 与生产证据闭环 |

本表分别注明仓库、CI 与已取得的生产证据；未明确写为生产实测的运维项仍不视为完成。

## 2. 验证基线

| 检查项 | 结果 |
| --- | --- |
| `go test ./... -count=1` | 通过 |
| `go test -race ./... -count=1` | 通过 |
| `go vet ./...` | 通过 |
| `go mod verify` | 通过 |
| `govulncheck ./...` | Go `1.26.5` 下通过，可达漏洞 0 |
| `bash -n scripts/*.sh backend/scripts/*.sh` | 通过 |
| Backend CI workflow `actionlint` | 通过 |
| Backend CI 实际运行 | `2ab697f`、`c928d19` 均成功 |
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

- [x] 部署配置必须显式声明环境；非 `local` 环境强制配置独立、高熵的
  `JWT_SECRET`、`ADMIN_JWT_SECRET` 和 `CREDENTIALS_SECRET`，禁止互相回退，
  禁止缺失、公开默认值或弱密钥启动。
- [x] 用户与管理员 Claims 增加固定 `iss`、`aud`、`token_use`，解析时全部校验。
- [x] 管理员中间件校验 `sub` 必须等于当前配置的管理员账号。
- [x] Admin Web 仅使用 HttpOnly Cookie，删除管理员 Bearer Token 兼容入口。
- [ ] 轮换线上三类密钥，使已签发旧令牌立即失效。
- [x] 为凭据密文增加 key version 和重加密流程，避免轮换
  `CREDENTIALS_SECRET` 后历史配置全部不可读。
- [x] 将 `APP_SETTINGS_ACCESS_MODE` 的生产默认值由 `all` 调整为 `admin`，
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

- [x] 抽取图片下载专用 egress policy，只允许 `http`/`https`，并由调用链决定是否允许明文 HTTP。
- [x] 在实际 `DialContext` 解析并校验目标 IP，拒绝 loopback、private、
  link-local、multicast、unspecified 和云元数据网段，避免 DNS 重绑定绕过。
- [x] 每次重定向重新校验目标，限制重定向次数，并拒绝协议降级。
- [x] 保留响应字节上限，同时增加连接、TLS、响应头和总请求超时。
- [x] 对被拒绝地址记录脱敏安全日志，不记录 URL userinfo、路径或查询参数。
- [x] 打卡点同步链路不得静默丢弃原图片而不告知调用方。

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

- [x] 使用 `URL.Hostname()` 做精确主机或点边界后缀匹配，拒绝 userinfo、异常端口、
  伪后缀和非 HTTP(S) scheme。
- [x] 复用 SEC-002 的 egress policy，在 DNS/Dial 阶段拒绝私网、回环、链路本地、
  云元数据地址，并对每次重定向重新校验。
- [x] 对 B 站返回的字幕 URL 建立独立域名白名单；只对可信 B 站域发送 SESSDATA，
  防止异常上游响应把 Cookie 带到第三方主机。
- [x] 修正小红书 host 标签边界校验。
- [ ] 验证 sidecar 自身执行等价的目的地址策略；在拿到 sidecar 代码前，把它记为
  待验证边界，不宣称已修复。

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

- [x] 为任务持久化不可复用的 claim token/generation 与 lease；所有成功、失败、重试
  终态用 CAS 更新，`RowsAffected=0` 表示结果已过期并安全丢弃。
- [x] 自动解析完成写必须同时校验 claim、`processing` 状态和人工编辑版本；发现人工
  编辑后保留用户内容，不清空 `parsed_content_edited`，并记录 stale/conflict 结果。
- [x] 业务字段与 worker 状态使用不同更新边界，避免普通菜谱更新回滚异步状态。
- [x] 外部调用较长时按 claim CAS 续租；续租失去所有权会取消调用，同时保留终态
  fencing 校验。

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

- [x] 本地与生产模块构建统一使用 Go `1.26.5`，生产二进制、manifest 和健康响应一致。
- [x] Backend CI 在 `2ab697f` 与 `c928d19` 实际使用 Go `1.26.5` 并成功通过全部门禁。
- [x] 将 `golang.org/x/image` 升级到 `v0.39.0` 或更高兼容版本并整理 `go.mod`。
- [x] 在 Go `1.26.5` 开发机重新执行全量测试、竞态测试、`go vet` 和
  `govulncheck`，可达漏洞为 0。
- [x] 在 CI 中固定 Go `1.26.5` 并增加格式、测试、覆盖率、vet、竞态、模块校验和
  漏洞扫描门禁。

**验收标准**

- `govulncheck ./...` 不再报告可达漏洞；若有接受项，必须记录影响、缓解措施和到期时间。
- 生产二进制通过 `go version -m` 可确认使用已修复工具链构建。

## 5. P1：下一迭代完成

### HTTP-001 建立请求资源边界与入口防护

**证据**：`internal/common/request.go:10` 的 JSON 解码没有字节上限；
`internal/app/app.go:208` 仅设置 `ReadHeaderTimeout` 和 `IdleTimeout`；登录入口没有
应用层限流。字段长度校验发生在完整解码之后，无法阻止大请求先占用内存。

- [x] 普通请求 body 统一限制为 1 MiB；饮食管家流式请求为 2 MiB；图片上传按
  `UPLOAD_MAX_IMAGE_MB + 1 MiB` 覆盖。
- [x] 超限统一返回 413；多 JSON 值、尾随垃圾、未知字段和空 body 均有契约测试。
- [x] Server 设置 30 秒 `ReadTimeout` 和 1 MiB `MaxHeaderBytes`；保持
  `WriteTimeout=0`，由路由级 deadline 管理 SSE，避免全局写超时截断流。
- [x] 管理员/微信/本地登录及邀请码预览、分享图和接受增加 IP + 凭据目标双维度
  内存限流与短期封禁；格式错误的登录请求也计入 IP 维度。
- [x] 仅信任本机 nginx 覆写的 `X-Real-IP`，公网直连不能伪造转发头绕过限流；
  bootstrap nginx 默认上限与上传上限联动，且应用继续独立执行更小的 JSON 上限。

**实现与验收说明**：限流状态当前为单进程内存态，符合现有单实例部署；未来若扩展为
多实例，必须切换到共享限流状态或由可信入口统一执行同等策略。应用集成测试覆盖 413、
管理员账号/IP 429、邀请码目标 429、流式/上传覆盖和 SSE 写超时边界。

### DATA-001 修复邀请接受的并发计数丢失

**证据**：`internal/invite/service.go:108` 在事务外读取邀请；
`internal/invite/repository.go:156` 使用旧 `UsedCount` 计算新值，并在
`repository.go:162` 无条件覆盖。并发请求可基于同一旧值加入多个成员，导致计数丢失
并突破 `maxUses`。

- [x] 在同一事务内重新读取并校验邀请状态、过期时间和使用次数。
- [x] 使用带 active、余量和过期条件的原子自增；失败时返回“已用完/已过期/
  已撤销”的确定错误。
- [x] 保持“已经是成员不重复消耗次数”的幂等语义，即使邀请随后已用完也返回
  当前成员结果。
- [x] 使用全量迁移 SQLite 和同一份过期快照覆盖 20 goroutine 竞争
  `maxUses=1`，只有一个新增成员成功；同时执行外键检查和竞态测试。

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

- [x] 增加显式默认空间标识和唯一部分索引，或等价 bootstrap marker；ensure 在事务中
  原子执行，唯一冲突后回读。20 goroutine 同用户并发时只能得到一个默认空间。
- [x] 为菜谱、地点增加 `version` 乐观锁并返回 409，或改成明确的 PATCH/字段级更新；
  两个客户端基于同一版本提交时必须一方冲突，禁止静默丢失更新。
- [x] create/update/delete 等关键写入在同一事务重新检查 membership，或在 SQL 加
  `EXISTS` 条件；用 barrier 让退出发生在权限检查和提交之间，断言写入失败。
- [x] 审查 mealplan、invite 等同类链路，但只在验证存在写入窗口后扩展改动范围。

**实现与验收说明**：新增 migration `027_add_consistency_versions.sql`。历史空间按
`name_source=auto` 优先、ID 次序回填一个 `is_default=1`，唯一部分索引保证每个 owner
最多一个默认空间；ensure 使用单事务 `INSERT ... ON CONFLICT ... DO NOTHING` 后回读并
幂等补 owner membership。菜谱和地点新增独立、从 1 开始的 `version`，PUT/PATCH 缺失
版本返回 400，CAS 失败返回 409；不复用 worker 的 `content_version`。小程序旧缓存缺少
版本时先拉详情，409 时刷新缓存但不自动重放旧表单。

事务内成员复核覆盖菜谱/地点 create、update、delete，空间改名，菜单草稿/提交/删除和
邀请创建；mealplan 同时在提交事务复核所有 recipe 仍属于当前空间且未删除。测试覆盖
20 goroutine 默认空间 ensure、历史多空间迁移回填/唯一约束、菜谱与地点双客户端同版本
竞争，以及在服务预检和仓储提交之间删除 membership 后所有关键写入均失败。后端全量
测试、相关包竞态和根前端/Admin Web 测试通过。

### DB-002 修复 Admin B 站配置的真实迁移契约

**已确认问题**

`migrations/004_add_app_bilibili_settings.sql:9-11` 要求 `updated_by` 非空且引用
`users(id)`；Admin runtime settings handler 在 `internal/admin/handler.go:562,576`
固定传 `updatedBy=0`。正式数据库通过 `internal/db/db.go:55` 开启外键，因此管理员首次
更新或清空配置会失败。最小 SQLite 复现已得到 `FOREIGN KEY constraint failed`；现有
service 测试自建表没有复制该外键，因而产生假绿。

- [x] 新增 `025_fix_app_bilibili_settings_operator.sql`，将管理员操作者建模为可空
  用户 ID + 必填 subject；用户写入 `user:<id>`，管理员写入 `admin:<username>`，
  不创建伪造的 `users.id=0` 记录规避约束。
- [x] 测试从全量迁移创建真实 SQLite，覆盖用户端和管理员端 update/clear，最后执行
  `PRAGMA foreign_key_check`。
- [x] 原简化测试 DDL 同步为可空外键 + 必填 subject，并新增真实迁移契约测试，避免
  单元测试 schema 与生产漂移。

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

- [x] 仅进程明确设置 `APP_ENV=local` 时自动读取 local dotenv；进程环境优先。显式
  `APP_ENV_FILE` 不存在、不可读或格式错误时拒绝启动，并记录脱敏的配置来源摘要。
- [x] typed parser 返回并聚合错误；`flase`、`1O`、`NaN` 等非法值在监听前失败，
  错误包含变量名但不回显 secret。
- [x] 生产 app settings 默认并强制为非 `all`；集成测试确认 dev-login
  为 404、普通用户写全局配置为 403。
- [x] 显式和本地 env file 权限宽于 `0600` 时拒绝读取；当前未跟踪的
  `backend/configs/local.env` 已从 `0644` 收紧为 `0600`，部署脚本与文档同样固定
  server env file 为 `0600`，未读取或记录其中的值。
- [x] 以表驱动测试覆盖 OS env、默认 dotenv、显式 env file、缺失/格式/权限错误和
  typed 非法值的完整优先级矩阵。

### SEC-003 日志与上传公开边界收口

**原始证据**：`internal/middleware/logger.go` 曾记录原始 URL path 和 query；邀请 token、
邀请码、菜谱分享 token 都位于 path。`internal/app/router.go` 曾直接使用
`http.FileServer` 并提供目录列表。`internal/upload/handler.go` 曾信任
`X-Forwarded-Host`/`X-Forwarded-Proto` 生成公开 URL。

- [x] 日志只记录 Chi 路由模板，不记录原始 path/query；已覆盖邀请 token、敏感 query
  和未匹配路径不泄露测试。
- [x] 禁止上传目录列表；明确图片继续作为无需登录的公开资源，避免破坏现有小程序、
  分享页和第三方图片转存 URL。
- [x] 公开资源增加 `nosniff`、CSP、referrer/CORP 响应头；成功图片使用一年
  immutable 缓存，失败响应 `no-store`。静态路由只接受当前生成器的
  `年/月/img_时间戳_48位随机值.扩展名`，并验证目录、可预测文件名、非图片和猜测路径
  均返回 404。
- [x] 非本地环境强制配置无 userinfo/query/fragment 的 HTTPS
  `UPLOAD_PUBLIC_BASE_URL`；上传响应优先使用该固定基址，并忽略
  `X-Forwarded-Host`/`X-Forwarded-Proto`，不从任意转发头构造持久化 URL。
- [x] 上传服务自身用 `max+1` 限流读取；文件先写 `.part`，大小超限、读取/关闭/
  rename 失败均删除半成品，成功后才原子 rename 为随机公开文件名。

验证：`go test ./... -count=1`、`go test -race ./... -count=1`、`go vet ./...`、
`go mod verify` 和 `git diff --check` 均通过。

### SEC-005 管理端 CSRF、令牌撤销与可信代理边界

**原始证据/条件式风险**

- Admin Cookie 在 `internal/admin/service.go:60-69` 使用 `SameSite=Lax` 且 Path 为 `/`；
  管理写接口没有 Origin/Fetch Metadata/CSRF 校验。同一主域存在可控兄弟子域时，
  浏览器仍可能携带同站 Cookie 发起跨源表单 POST。
- 用户 Bearer Token 默认有效 720 小时，只有 `exp`，没有 `jti`、token version、
  refresh session 或服务端撤销机制；泄露后用户无法主动失效。
- `chi/middleware.RealIP` 无可信代理边界地接受转发头；当前会污染审计 IP，未来若直接
  用于登录限流，会被客户端伪造头绕过。

- [x] Admin 已是 Cookie-only，不再保留审查时设想的 Bearer 兼容。登录/`me` 返回与
  HttpOnly 会话 JWT 绑定的 HMAC CSRF token，Admin Web 对所有写方法发送
  `X-CSRF-Token`；中间件同时拒绝 `Sec-Fetch-Site` 非 `same-origin` 的写请求。
  Cookie 改为 `SameSite=Strict`，Path 由 `ADMIN_COOKIE_PATH` 收紧：开发/standalone
  使用 `/api/admin`，现网共享前缀使用 `/caipu-api/admin`。
- [x] 新增 migration `026_add_user_token_version.sql`，历史用户回填版本 1；用户 JWT
  增加随机 `jti` 和 `ver`，每次鉴权查询数据库当前版本。新增
  `POST /api/auth/logout` 原子递增版本，注销、直接版本递增后旧 token 均返回 401；
  小程序提供 `logoutCurrentSession` 调用封装并保证本地状态最终清理。
- [x] HTTP-001 已用 `TrustedRealIP` 收口可信代理：仅 TCP 直连对端为 loopback 时接受
  nginx 覆写的 `X-Real-IP`，直连伪造转发头不改变客户端 IP；限流已复用该结果。

验证：Admin 中间件覆盖缺失/错误 CSRF、cross-site/same-site 拒绝与 same-origin 成功；
应用集成测试覆盖 Cookie 属性、Admin 注销、用户登录→注销→旧 token 401→重新登录→
版本递增→旧 token 401；真实 `026` SQL 覆盖历史用户回填与 CHECK 约束。后端全量测试、
全量竞态、`go vet`、`go mod verify`、Admin Web typecheck/测试、小程序与 shell 语法和
`git diff --check` 均通过。

### OBS-001 统一错误出口与结构化日志

**证据**：服务层普遍包装了根因，但 `internal/common/response.go:32` 丢弃内部错误，
`internal/middleware/logger.go:40` 只记录状态码；普通 500 很难通过 request ID 还原。

- [x] `common.WriteError` 通过 ResponseWriter error observer 把原始错误交给外层请求日志；
  客户端继续只收到稳定 AppError/通用 500，内部错误类型链和脱敏摘要只写日志。
- [x] 请求日志已移到 Chi Router 外层，未知路由也被记录；字段包含 response
  `X-Request-ID`、request ID、路由模板、状态码、耗时、错误 code/type/type chain，
  并仅从 allowlist URL 参数记录 kitchen/recipe/place/provider 等安全业务 ID，排除
  token/code 和所有 query。
- [x] 新增全局 `logging.RedactingHandler`，统一处理 record message、error/string/group
  属性；敏感 key、Bearer/JWT、URL userinfo/path/query 和 AI messages/prompt/body 均替换
  或省略，worker 与请求日志使用同一规则。
- [x] `WriteJSON` 改为内存预编码，成功后才写 Header/body；编码失败在 Header 未写出时
  一次性返回通用 500，不再追加无效 `http.Error`。测试覆盖 400/401/403/404/405/409/
  500 envelope、内部根因隔离、编码失败单次 500、错误链脱敏和未知路由日志。

验证：`go test ./... -count=1`、`go test -race ./... -count=1`、`go vet ./...`、
`go mod verify` 和 `git diff --check` 均通过。

### AI-001 限制上游响应并脱敏 SSE 错误

**已确认问题**

- `internal/airouter/openai_client.go:175-205`、`internal/linkparse/bilibili.go:581-596`
  等成功响应直接交给 JSON Decoder，没有响应体上限；图片生成响应可能包含大体积 base64，
  异常或被接管的上游可造成进程内存持续膨胀。
- `internal/dietassistant/service.go:282-288` 把上游非 2xx body 直接作为公开 AppError，
  `handler.go:122-129,192-198` 又通过 SSE 原样返回；上游 HTML、内部路由信息或供应商
  调试内容会暴露给用户。现有普通 JSON 错误脱敏不覆盖这条已写 200 的 SSE 链路。

- [x] 为各类上游成功响应设置按场景上限；使用可识别“刚好到上限”的 bounded reader，
  超限返回稳定的 bad gateway 错误并停止解码。图片 base64 上限需结合最终图片限制测算。
- [x] 上游原始错误只进入脱敏日志/审计，SSE 对客户端返回稳定业务文案和 request ID；
  不转发 HTML、API key、主机名或原始 provider body。
- [x] 对流式响应限制单事件大小、累计可见内容和工具参数大小，保持 context 取消有效。
- [x] 使用超大 JSON/base64、无限 SSE 事件和含敏感字段的错误体做定向回归及内存基准。

**实现与验收说明**：新增共享 `internal/upstream` bounded reader，以 `max+1` 区分恰好
到上限和超限，并拒绝多 JSON 值。文本 AI 为 2 MiB、饮食管家普通 JSON 与 B 站/
Sidecar 为 4 MiB、微信 `code2session` 为 64 KiB、地图为 2 MiB、分享页为 256 KiB；
图片生成 JSON 为 16 MiB，可容纳默认 10 MiB 图片经过约 4/3 base64 膨胀后的 JSON
开销。饮食管家限制单 SSE 事件和累计可见内容各 256 KiB、工具块和工具参数各
64 KiB。Provider/Sidecar/SMTP 探测只向 Admin 返回稳定状态文案，脱敏根因进入统一日志；
SSE error 事件固定中文文案并附 `requestId`，小程序错误对象保留并展示该 ID。

验证覆盖文本/图片 base64/Sidecar/B 站/微信/饮食管家超限、无限无换行 reader、多个
小事件累计、工具块/参数、`context.Canceled`/`DeadlineExceeded` 和含 HTML、API key、
内部主机名的上游错误体。`BenchmarkReadAllBoundedOversize` 在 Apple M3 上为
`163674 ns/op`、`2236547 B/op`、27 allocs/op；全量测试、全量竞态、`go vet`、
`go mod verify`、根前端/Admin Web 测试、JS/Shell 语法与 `git diff --check` 均通过。

### OPS-001 建立可恢复的 SQLite 备份与发布流程

**证据**：`backend/scripts/backup.sh:9` 直接复制运行中 WAL 模式数据库的主文件，
没有在线备份协议、完整性校验或恢复演练。`backend/scripts/deploy.sh:70` 标注“重启”
但使用 `systemctl enable --now`，服务已运行时不会执行 restart；相关旧文档仍推荐该脚本。
当前服务器侧发布脚本也缺少发布前测试、数据库备份和健康失败自动回滚。

- [x] 使用 SQLite Online Backup API、`sqlite3 .backup` 或等价一致性快照，禁止只复制主文件。
- [x] 备份数据库和上传文件，增加保留周期、校验和、异机副本及定期恢复演练。
- [x] 发布前执行配置校验、迁移预检和一致性备份；使用版本化 release 目录原子切换二进制。
- [x] 健康检查失败时自动恢复上一二进制，并明确数据库迁移的前向兼容/人工回滚策略。
- [x] 修复或废弃 `backend/scripts/deploy.sh`，统一 README、部署文档和线上实际入口。

**实现与验收说明**：`backup.sh` 使用 `sqlite3 .backup` 捕获已提交 WAL 页，以隐藏 staging
目录生成 `app.db`、uploads 压缩包、元数据和 SHA-256 后原子 rename；支持保留天数、
`OFFSITE_BACKUP_TARGET` rsync 和生产强制异机开关。`verify-backup.sh` 校验哈希、SQLite
quick check 与 uploads 实际解包；bootstrap 安装每日备份和每周恢复校验 timer。测试用
长读事务把第二条提交留在 WAL，备份仍得到两条记录，篡改后校验必败。

实际发布入口构建带 release ID 的 `releases/<id>`，配置校验后先备份，再在备份副本预演
migration，生产只执行前向迁移；`current` 原子切换后连续核对 readiness release ID，失败
恢复上一二进制并保留失败 release/备份。回滚集成测试验证两次 restart 和 symlink 恢复。
旧 `backend/scripts/deploy.sh` 已改为废弃提示。生产异机副本与真实数据恢复/回滚仍属外部
验收，不因仓库脚本通过而视为完成。

### OPS-002 增加 readiness 并以最小权限运行服务

**已确认问题**

- `internal/app/router.go:79-91` 的健康接口是静态 200；发布脚本用它判断成功，数据库
  已关闭、路径权限错误或迁移不完整时仍可能绿。
- `backend/scripts/bootstrap-server.sh:92-107` 和部署文档生成的 systemd unit 未设置
  `User/Group`，默认以 root 运行，也没有文件系统与提权沙箱。

- [x] 分离 `/livez` 与 `/readyz`；readiness 用短超时检查 DB ping、迁移版本及必要目录，
  故障返回 503。发布脚本要求连续 N 次 ready，并核对目标 release ID。
- [x] 建立专用不可登录用户，只授权数据库/WAL/SHM、uploads 和必要临时目录；增加
  `NoNewPrivileges`、`PrivateTmp`、`ProtectSystem`、`ProtectHome`、`ReadWritePaths`、
  `UMask` 和匹配应用 deadline 的 `TimeoutStopSec`。
- [x] 生产迁移专用用户/current unit，验收 UID 非 0、无法写 `/etc` 或 release，data、
  字体和 SQLite WAL 正常，重启后内外 readiness 与 release ID 一致。
- [ ] 执行生产真实二进制回滚，并在受控窗口验证关闭 DB/破坏目录权限时 readiness 失败而
  liveness 保持可用。

**实现与验收说明**：`/livez` 不访问依赖；`/readyz` 在 2 秒上限内检查 DB ping、当前
release 全部 migration、SQLite 父目录与 uploads 可写性，失败只返回稳定 503，日志保留
根因。`/healthz` 与 `/api/healthz` 改为 readiness 兼容别名，响应头/JSON 均携带 release
ID。集成测试已覆盖关闭 DB 和删除最新 migration 记录时 readiness 失败而 liveness 成功。

bootstrap 创建不可登录 `caipu-backend` 用户，unit 指向 `current/server` 并只开放 data
写权限，停止上限 15 秒覆盖应用 10 秒 shutdown deadline。生产已于 2026-07-16 验证 UID、
字体、沙箱写边界、WAL、重启与 release-aware readiness；为避免在未告知窗口主动影响
可用性，真实回滚和故障 readiness/liveness 分离仍单独保留为未完成项。

### TEST-001 补核心契约测试并接入 CI

- [x] 优先补 SEC-001、SEC-002、SEC-004、DATA-003、DATA-001、DB-002、HTTP-001
  对应的回归测试。
- [x] 补 `admin/auth/invite/kitchen/upload/common/profile` 的 handler、service 和真实迁移
  SQLite 集成测试，覆盖状态码、JSON、Cookie、权限和事务边界。
- [x] CI 执行 `gofmt` 检查、`go test ./...`、`go vet ./...`、
  `go test -race ./...`、覆盖率采集、`go mod verify` 和 `govulncheck ./...`。
- [x] 不把整体覆盖率阈值作为首要门禁；先对关键安全与业务分支设置明确断言。

**实现与验收说明**：安全边界已有管理员/用户 token 双向隔离、CSRF、SSRF DNS/Dial/
redirect、B 站 host/字幕、worker claim fencing、邀请 20 goroutine 原子接受、真实 B 站
migration 和 HTTP body/JSON/限流测试。新增全路由 SQLite 集成流覆盖 profile 更新、空间
创建/改名、邀请创建/公开预览/第二用户接受和成员空间列表；结合 Admin Strict Cookie、
upload handler/service、common envelope 与 migration 测试，对状态码、JSON、Cookie、权限
和事务边界均设置显式断言。覆盖率继续作为定位信号，不设统一百分比阈值。

## 6. P2：满足进入条件后实施

### ARC-001 Admin HTTP 边界收口

- [x] 在 TEST-001 锁定契约后，将 `internal/admin/handler.go` 按 auth、dashboard、audit、
runtime settings、AI routing/alert 拆分，并以窄接口替代大范围具体类型依赖。
保留统一 `Handler` 和现有路由签名，不改前端契约。

**实现与验收说明**：原 650 行 handler 已拆为 91 行组合边界及 auth、dashboard/audit、
runtime、AI routing、AI alert 五个 58～180 行域文件。Audit、Runtime、Bilibili、Server
Health、AI Routing 和 AI Alert 均按实际调用面定义接口，不为无替代边界的模块泛化接口。
新增各域独立 `httptest`，并由真实 App 集成测试继续锁定 Cookie/CSRF、JSON、状态码和
路由；Admin/App 定向测试及竞态通过。

### HTTP-002 外部 HTTP 适配器可测性

- [x] 为 `airouter`、`appsettings/runtime_probe` 等请求路径临时创建客户端的模块增加包内
`HTTPDoer`/Options 注入，复用 Transport，统一响应上限和错误分类。不要建立跨业务的
“万能 HTTP 客户端”；SSRF egress policy 只用于处理不可信目标地址的链路。

**实现与验收说明**：两个业务包各自定义窄 `HTTPDoer` 与 Options constructor，默认
client 共享 `http.DefaultTransport`，不新增跨业务万能客户端。provider/probe timeout 由
派生 request context 管理；运行时探测统一限制 64 KiB 并完整读取成功体以复用连接。
注入测试覆盖 method/path/body、Authorization、非 2xx、畸形 JSON、网络错误、超时、取消、
响应超限和损坏密文；数据库密文解密失败会在发出请求前按 auth fail-closed。

### WORKER-001 Worker 生命周期一致化

- [x] 先补三个菜谱 worker 的启停、取消、非法间隔和 goroutine 退出测试，再提取最小的
包内 loop/clock 能力。`Stop` 应接受 context 或有明确等待上限，避免 worker 不退出时
阻塞整个优雅停机流程。当前 `internal/app/app.go:241-258` 在调用
`http.Server.Shutdown(ctx)` 前先同步执行三个无 context 的 `Stop()`，所以 main 创建的
10 秒 deadline 实际约束不到 worker，等待期间 HTTP 还可能继续接流量。验收应先停止
接流量，再在统一 deadline 内取消并等待 worker，最后关闭 DB；注入不退出 worker 时
进程也必须在 deadline 内返回明确错误。

**实现与验收说明**：三个 worker 删除重复的 cancel/done/once/ticker 实现，统一使用
recipe 包内 lifecycle；保持“立即执行一次 + 周期 tick”语义，非法 interval 返回错误而非
ticker panic，重复并发 Start/Stop 幂等。`Stop(ctx)` 在不合作任务下返回 deadline error。
App 先调用 `http.Server.Shutdown(ctx)` 停止接流量，再并发取消/等待 worker，最后关闭 DB；
注入忽略取消的 worker 时，测试证明 25ms deadline 内返回且 DB 仍被关闭。定向 `-race` 与
后端全量测试、vet、diff 检查通过。

### DATA-002 数据生命周期与查询容量治理

- 为 `ai_job_runs`、`ai_call_logs`、配置审计和告警事件定义保留期、清理批次与归档策略。
- 当单空间菜谱、打卡点或菜单达到实测阈值后，再为列表接口增加游标分页；当前不为未来
  规模提前引入复杂搜索系统。
- 只有统计维度继续增长且查询计划出现退化时，才拆分 `spacestats/repository.go` 或调整索引。

### DATA-005 配置审计、告警通知与管理写入并发

- [x] `internal/appsettings/service.go:79-103,122-141` 先提交 B 站配置，再单独插入审计且
  忽略错误；应在同一事务读取旧值、写配置和审计，审计失败整体回滚。
- [x] `internal/aialert/service.go:238-283` 在发送邮件后才标记已告警，并发失败跨过阈值时
  可能重复发信；使用原子 claim/outbox 和 event ID 幂等，发送失败可重试。
- [x] AI 路由和运行时配置是整组 last-write-wins；两个管理员同时保存会静默覆盖且审计
  oldValue 过时。响应携带 version/updatedAt，保存 CAS，冲突返回 409。
- [x] 餐单在事务外验证 recipe 后保存快照。先确认产品是否允许引用随后删除的菜谱；若允许，
  把它记录成历史快照契约；若不允许，在 Replace 事务内复查，不做含糊修复。

**实现与验收说明**：B 站设置写入和审计在同一 SQLite 事务内读取旧值、CAS 更新版本并
提交，审计 trigger 拒绝时配置保持旧值；B 站设置使用单条查询、AI route scene 与 runtime
group 使用只读事务，保证返回值和 version 来自同一 SQLite 快照，并有并发读写压力测试。
migration `028` 增加告警 failure streak 与 delivery outbox；失败状态和 enqueue 同事务，
claim token/lease 由单条 SQL 原子认领，失败按上限 15 分钟退避，后台 worker 在没有新失败
请求时继续重试。测试直接覆盖首次发送失败后回到 pending、到期重试成功、无新失败请求时
worker 自动派发、过期 lease 重领、Stop 取消发送，以及 SMTP 等待 greeting 时响应 context
取消。20 goroutine 同时跨过阈值只产生一条 delivery 且只发送一次。AI route scene 与
runtime group（含 B 站）均返回整数 version，过期保存返回 409；双客户端同版本测试均为
一个成功、一个冲突。餐单 recipe 归属与未删除状态已由 DATA-004 在 Replace 事务内复核。
SMTP 不提供幂等键，若邮件已被上游接受但进程在 sent 落库前崩溃，恢复 claim 后仍可能
重复发送，这是外部协议残余风险。

### DB-001 迁移治理

- [x] `schema_migrations` 增加文件校验和，启动时拒绝已应用迁移被静默修改。
- [x] 现有两个 `019_*` 文件保持原名，避免破坏已部署记录；对后续迁移增加序号唯一性检查。
- [x] 在 CI 中从空库顺序应用全量迁移，并验证代表性历史库升级和 `PRAGMA integrity_check`。

**实现与验收说明**：bootstrap 自升级旧 `schema_migrations` 表并为历史空 checksum 首次
回填当前文件 SHA-256；后续 Run/ready 检查均严格比对。迁移目录拒绝非法命名和重复序号，
只对白名单中的两个既有 `019_*` 放行。真实迁移测试覆盖空库全量、从 `001`～`014` 的代表性
历史库继续升级、运行时 group version 回填、outbox 建表、所有记录有 checksum 以及
`PRAGMA integrity_check=ok`；修改已应用文件的测试会在执行任何新迁移前失败。

### CFG-001 配置分组

仅当配置继续增长或出现模块参数漏接时，将扁平 `config.Config` 映射为 Auth、AI、
Linkparse、Recipe、Upload 等只读模块配置。保留环境变量名称与默认业务策略，不引入
DI 框架，不让业务包反向依赖全局配置。

### OPS-003 发布身份、健康探测配置与日志留存

- [x] 用 `-trimpath` 和 ldflags 注入 commit、build time、Go toolchain、release ID；发布清单
  保存二进制 SHA-256 和迁移集合，启动日志/受控版本接口可映射到唯一构建。
- [x] `internal/admin/server_health.go:104-110` 硬编码 `caipu-backend` 与 8080，但部署脚本
  允许覆盖服务名/端口，另一 bootstrap 默认名也不同；改为显式配置或进程内依赖检查。
- [x] 明确 journald 容量和保留期或集中采集策略，为 5xx、worker 连续失败、磁盘空间、
  备份年龄与恢复校验建立告警；日志必须可按 request ID/release ID 查询。

**实现与验收说明**：`server -version`、live/ready JSON 和启动日志均输出 release ID、commit、
build time、Go toolchain；release 同时保存 `manifest.env`、逐文件 `migrations.sha256`、迁移
数量与集合摘要。Server Health 的三个 unit 名和后端 base URL 改由环境配置。bootstrap
默认将 journald 全局上限设为 512 MiB/14 天，并安装每五分钟巡检：5xx、worker error、
磁盘 80%/90%、备份 26 小时、恢复演练 8 天；详细阈值和外部接收端边界见
`docs/backend-operations-alert-policy.md`。仓库只保证巡检以 1/2 退出码暴露告警，生产告警
平台订阅和真实投递必须实机验收后才能关闭外部项。

生产已于 2026-07-16 13:42 单独应用 journald `512M/14day` 并完成历史日志清理：占用从
3.9 GiB 降至 456 MiB，根分区从 96% 降至 87%；`ops-health` unit 尚未安装，仍需后续实测。

## 7. 推荐执行顺序

| 阶段 | 工作包 | 完成门禁 |
| --- | --- | --- |
| 0：紧急安全 | SEC-001、SEC-002、SEC-004、DATA-003、DEP-001 | 安全、数据保护、竞态和漏洞门禁通过 |
| 1：入口与一致性 | HTTP-001、DATA-001、DATA-004、DB-002、CFG-002、SEC-003、SEC-005、AI-001 | 契约、并发、配置和脱敏测试通过 |
| 2：恢复与门禁 | OBS-001、OPS-001、OPS-002、TEST-001 | CI 绿；完成备份恢复、最小权限和回滚演练 |
| 3：维护性 | ARC-001、HTTP-002、WORKER-001 | 现有契约不变，定向与全量门禁通过 |
| 4：条件演进 | DATA-005、DB-001、OPS-003 已完成；DATA-002、CFG-001 继续观察 | 有容量、并发或运维证据后启动；未满足条件的不机械演进 |

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
