# Project Changelog

## 2026-04-04

### Fixed

- 修复个人资料页“提示资料已更新，但昵称和头像实际未生效”的回归问题：
  - 后端将“登录态补资料”和“用户主动改资料”拆分为两条更新策略，主动保存资料时允许替换已有的非占位昵称
  - 后端登录补资料不再用微信侧头像覆盖用户已经手动保存过的头像，只会补齐缺失头像
  - 前端登录态自动资料同步同样收窄为“只补缺失资料”，避免保存后立刻被微信侧资料回写覆盖

### Notes

- 修改时间：2026-04-04 00:00 CST
- 变更背景：个人资料弹层保存成功后，后端显式更新逻辑错误地沿用了“仅补占位昵称”的策略，导致已有昵称无法再次修改；同时前后端自动资料同步都可能把用户手动保存的头像重新覆盖为微信侧资料，造成“提示成功但界面没变”
- 核心改动：新增后端显式资料更新分支用于处理用户主动改资料；保留原有 `EnsureProfile` 作为登录补资料逻辑，但限制为只补齐缺失头像；前端 `ensureSession` 内的自动资料同步策略同步收窄为只补缺失头像，不再覆盖已有头像
- 影响范围：`backend/internal/auth/repository.go`、`backend/internal/auth/repository_test.go`、`backend/internal/auth/service.go`、`utils/auth.js`
- 兼容性/风险：本次不改接口契约；已有用户下次重新保存资料后才能把历史上未生效的昵称或头像修正到最新值；前端当前仍无自动化测试，需在微信开发者工具或真机上补一次“改昵称 + 选头像 + 重新进入页面”的联调确认
- 验证情况：已执行 `cd backend && go test ./internal/auth/...`；已执行 `git diff --check`

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
