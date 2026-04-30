# 美食库 UI 视觉与动效改造方案 v2

修改时间：2026-04-30 CST
适用范围：`pages/index/index.vue`、`pages/index/components/library-header-section.*`、`pages/index/components/recipe-card-item.*`、`pages/index/recipe-card.js`
参考来源：`我们的美食空间/`（React 原型）、`docs/food-library-ui-optimization-report-2026-04-29.md`（v1 工程拆分文档）
目标平台：微信小程序，`uni-app + Vue 3 + uview-plus`

## 0. 与 v1 文档的关系

| 文档 | 定位 | 重点 |
| --- | --- | --- |
| `food-library-ui-optimization-report-2026-04-29.md` | v1 · 工程拆分清单 | 组件抽离、状态数量、空态动作链路、菜卡触控 |
| **本文（v2）** | **设计语言对标** | 视觉 token、布局重构、动效规范、装饰层级、跨模块一致性 |

两份文档配合使用：v1 定义"哪些组件该拆、补什么动作"，v2 定义"拆出来的组件长什么样、怎么动、跟空间页如何统一"。建议先按 v1 完成结构改造，再按本文做视觉与动效收口；不打算同时落两份文档的所有点。

## 1. 一句话目标

让美食库首屏在 **暖白底 / 粘土阴影 / 大圆角 / 衬线点缀 / 轻动效** 这套设计语言上，和参考原型 `我们的美食空间/` 与现有的 `kitchen-section`（空间页）保持同一气质，并且不引入跨端不稳定依赖。

## 2. 当前状态盘点

| 模块 | 当前状态（2026-04-30） | 与原型差距 | 优先级 |
| --- | --- | --- | --- |
| 标题区（`page-header`） | 已有左侧图标章 + 标题 + 副标题 + 右侧"安排菜单"按钮 | 副标题语气偏中性，按钮颜色和图标章统一度高，整体接近原型 | P2 微调 |
| 菜单 spotlight | 仅有记录时显示，单行展示标题 / 描述 / 元信息 | 离原型 `PlanCard` 的"日历图标 + 衬线日期 + 菜品摘要 + 进度 chip"还差一段 | **P1** |
| 工具卡（搜索 + 餐别 tab + 状态 pill） | 用 `.toolbar` 容器包裹，圆角 22rpx，阴影 0 8rpx 20rpx | 圆角和阴影都偏弱于原型的 `rounded-[32px] + clay-shadow + border-white` 卡 | **P0** |
| 状态 pill（全部 / 想吃 / 吃过） | 已三态颜色 + 激活态深色填充 | **缺数量徽标**，激活反馈较扁 | **P0** |
| 列表 caption | 已含筛选摘要 + 清除 + 帮我选 | "帮我选"按钮可读性 OK，但在长摘要时被挤压 | P1 |
| 菜卡 | 横向布局，封面 + 来源徽 + 标题 + 摘要 + 状态 switch | 整体方向贴近原型 `RecipeCard`，但摘要经常空缺、来源徽对比度偏弱 | P1 |
| 空态 | 仅图标 + 标题 + 描述，**无任何动作** | 原型空态有"问问 AI 怎么做"主 CTA，是首屏断点 | **P0** |
| 入场动效 | 卡片有 `recipe-card-enter`，spotlight 有 slide-next/previous | 模块切换、过滤切换缺整段平移/淡入；状态 pill 切换无 thumb 滑动 | P1 |
| 与 `kitchen-section` 一致性 | 共用色板，但卡片形态、阴影深度未对齐 | 空间页大卡用了渐变 + 子格统计，美食库工具卡仍偏 flat | P1 |

P0 = 本轮必做，P1 = 紧随其后，P2 = 视觉 polish 阶段。

## 3. 设计语言（Design Token）

> 全部以 rpx 落地小程序，避免引入远程字体或 backdrop-filter 在低端安卓上的性能风险。

### 3.1 色板（沉淀为变量）

| 角色 | Token 建议名 | 色值 | 用途 |
| --- | --- | --- | --- |
| 页面底色 | `--color-bg` | `#F6F2EA` | 全屏背景，接近 `brand-cream` |
| 卡面 | `--color-surface` | `#FFFDF8` | 工具卡 / 菜卡 / 空态卡 |
| 卡面（柔和暖） | `--color-surface-warm` | `#F4ECDF` | spotlight 渐变末端、章节强调 |
| 主文 | `--color-text-primary` | `#2F2923` | 标题、菜名、强提示 |
| 次文 | `--color-text-secondary` | `#74685E` | 描述、副标题 |
| 弱文 | `--color-text-muted` | `#9F9387` | 筛选摘要、提示语 |
| 主品牌棕 | `--color-brand-brown` | `#5C4033` | 主操作填充、激活状态 |
| 陶土橙 | `--color-accent-terracotta` | `#BF715F` | 想吃 / 提醒 / 安排菜单点缀 |
| 鼠尾草绿 | `--color-accent-sage` | `#6F846A` | 吃过 / 完成 / 已记录 |
| 卡片描边 | `--color-border-soft` | `rgba(91, 74, 59, 0.07)` | 工具卡、菜卡、状态 pill |
| 卡片描边（强调） | `--color-border-active` | `rgba(91, 74, 59, 0.16)` | 选中、聚焦 |

落地点：在 `pages/index/index.vue` 的 `<style>` 顶部补一段 `:root`/`.app-shell` 级 CSS 变量声明（小程序支持 CSS 变量），让工具卡、菜卡、空态卡共享同一组令牌；后续若做 dark theme 也方便切换。

### 3.2 圆角与阴影

| 元素 | 圆角 | 阴影 |
| --- | --- | --- |
| 工具卡（搜索 + 筛选大容器） | **32rpx** | `0 14rpx 32rpx rgba(56, 44, 30, 0.06), inset 0 1rpx 0 rgba(255, 255, 255, 0.7)` |
| 菜单 spotlight（PlanCard 化） | 28rpx | `0 12rpx 26rpx rgba(56, 44, 30, 0.05), inset 0 1rpx 0 rgba(255, 255, 255, 0.65)` |
| 菜卡 | 26rpx（保持当前） | `0 12rpx 24rpx rgba(70, 54, 40, 0.045)` |
| 状态 pill 容器 | 18rpx | 内嵌 inset 高光，外阴影只在激活态出现 |
| 空态卡 | 30rpx | 与工具卡同档 |
| 餐别 tab 内格 | 16rpx | 不带外阴影 |
| 安排菜单按钮 / 帮我选 | 999rpx 胶囊 | 轻外阴影 + inset 高光 |

阴影统一为"上方更亮、下方更深"的 clay-shadow，调用方式建议抽成两个 mixin（或 SCSS 变量）：

```scss
@mixin clay-shadow-soft {
	box-shadow:
		0 12rpx 24rpx rgba(70, 54, 40, 0.05),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
}

@mixin clay-shadow-strong {
	box-shadow:
		0 14rpx 32rpx rgba(56, 44, 30, 0.07),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.7);
}
```

### 3.3 字体层级

不引入远程字体，但通过 **字号 / 字重 / 字距** 拉开层级：

| 角色 | 字号 | 字重 | 字距 |
| --- | --- | --- | --- |
| 页面大标题"美食库" | 40rpx | 700 | -0.2rpx |
| 菜单 spotlight 日期（数字部分） | 36rpx | 700 | 0 |
| 工具卡内餐别 tab | 25rpx | 700 | 0 |
| 状态 pill 文字 | 23rpx | 700 | 0 |
| 菜卡菜名 | 30rpx | 700 | 0 |
| 菜卡 eyebrow（餐别 · 整理状态） | 19rpx | 700 | 0.6rpx |
| 菜卡摘要 | 23rpx | 400 | 0 |
| 空态主标题 | 30rpx | 700 | 0 |
| 空态次行说明 | 24rpx | 400 | 0 |

> 数字部分（spotlight 日期、份数、统计数）建议引入系统衬线数字栈作为点缀，模拟原型 `Playfair Display` 的"刊物感"：
>
> `font-family: "Songti SC", "STKaiti", "DejaVu Serif", serif;`
>
> 仅用于数字与少量装饰文本，不用作正文，避免微信不同设备渲染抖动。

### 3.4 间距系统

统一节奏 = `4rpx → 8rpx → 12rpx → 16rpx → 24rpx → 32rpx`：

| 场景 | 建议值 |
| --- | --- |
| 卡内左右内边距 | 24rpx（重要卡）/ 18rpx（次要卡） |
| 模块间纵向间距 | 16rpx → 24rpx |
| 同一卡内分块间距 | 12rpx |
| 标题与正文 | 8rpx |
| 列表卡之间 | 14rpx（保持当前） |

## 4. 布局重构（自上而下）

### 4.1 标题区

```
┌─────────────────────────────────────────────┐
│  [□图标]  美食库               [📅 安排菜单] │
│           按餐别整理，想吃和吃过更清楚         │
└─────────────────────────────────────────────┘
```

改动建议：
- 图标章背景渐变维持当前棕色调，但描边色统一为 `--color-border-soft`。
- 副标题色阶下沉到 `--color-text-secondary`，避免与正文文字混淆。
- "安排菜单"按钮背景改为浅陶土渐变 `linear-gradient(145deg, #F1E3D5, #EAD8C6)`，按压时下沉 1rpx + 内阴影变浅；与原型的轻奶油圆角胶囊一致。

### 4.2 菜单 spotlight → PlanCard 化

参考原型 `PlanCard.tsx`：

```
┌─────────────────────────────────────────────┐
│  📅  04月18日 周六                  1/3  ›   │
│      多宝鱼 / 酸辣拍黄瓜 / 莫氏鸡煲同款        │
└─────────────────────────────────────────────┘
```

落地点 `library-header-section.vue` 的 `meal-order-spotlight` 块：

| 项 | 现状 | 改造 |
| --- | --- | --- |
| 背景 | `radial-gradient + #FFFAF4` | 改为 `linear-gradient(145deg, #FFFDF8 0%, #F4ECDF 100%)` 让卡面温度感更强 |
| 左侧 | 文字直起 | 加 `📅` 日历图标章（28rpx 方块、橙色描边、内填白） |
| 标题 | 单行文字 | 拆为「日期主体（衬线数字）+ 周几 + 状态 chip（已安排 / 已完成）」 |
| 摘要 | 单行省略 | 保留单行，但前置一个 `·` 装饰间隔 |
| 右侧 | 进度文字 + arrow-right | 把进度改为右上角小 chip `1/3`，arrow 下移到 chip 下方右对齐，竖排两行更稳 |
| 多记录滑动 | 已有 motion-next / previous | 保留，方向反转时颜色短暂明亮一拍（见 §5） |

### 4.3 工具卡（搜索 + 餐别 + 状态）

把现有 `.toolbar` 升级成"原型同款大白卡"：

```
╔══════════════════════════════════════════════╗
║  🔍  搜索菜名 / 食材，搜不到可问 AI       ✕  ║
║                                              ║
║  ┌──────────────┐ ┌──────────────┐           ║
║  │  早餐    1   │ │  正餐   37   │           ║
║  └──────────────┘ └──────────────┘           ║
║                                              ║
║ [全部 38] [♡ 想吃 12] [✓ 吃过 26]             ║
╚══════════════════════════════════════════════╝
```

| 项 | 现状 | 改造 |
| --- | --- | --- |
| 容器圆角 | 22rpx | **32rpx**，与空间页的当前空间大卡同档 |
| 容器背景 | `rgba(255, 255, 255, 0.86)` | `--color-surface` 实色 + `border 1rpx --color-border-soft` |
| 容器阴影 | 0 8rpx 20rpx | `clay-shadow-strong` |
| 内边距 | 18rpx | 24rpx，给搜索栏与筛选块更多呼吸 |
| 搜索框 | 高 68rpx，圆角 18rpx | 高 76rpx，圆角 22rpx，左侧搜索图标章背景换成 `rgba(91, 74, 59, 0.06)` 圆形 |
| 推荐词 chips | 已有 | 维持，但 `chip` 圆角 999rpx + 字号 22rpx 不变；激活态加深色边框 `--color-border-active` |
| 餐别 tab 内格 | `bg: rgba(255, 255, 255, 0.24)` | 未激活态保持低对比；激活态 **加 1rpx 内描边 + inset 高光**，让"被选中"更立体 |
| 餐别 tab 计数 | 已有 | 保留 |
| 状态 pill | 现行三色 | **正中加数字 chip**，未激活态 `bg: rgba(91, 74, 59, 0.06)`、激活态 `bg: rgba(255, 255, 255, 0.16)`；数字 18rpx · 600 |

数量来源（计算属性，不动业务）：

```js
statusCounts(activeMealType) {
	const list = this.recipesByMealType(activeMealType)
	return {
		all: list.length,
		wishlist: list.filter(r => r.status === 'wishlist').length,
		done: list.filter(r => r.status === 'done').length
	}
}
```

### 4.4 列表 caption

```
正餐 · 37 道                          [清除] [⟳ 帮我选]
```

| 项 | 现状 | 改造 |
| --- | --- | --- |
| 文案 | "正餐 12 道，想吃 4 道" 等长摘要 | 用 ` · ` 分隔，最多两段：`餐别 · 数量` 或 `搜索关键词 · 数量` |
| 长摘要省略 | `word-break: break-all` | 改 `white-space: nowrap; overflow: hidden; text-overflow: ellipsis;`，避免行高跳动 |
| 帮我选按钮 | 渐变胶囊 | 维持，但激活态压感动效从 `scale(0.992)` 改为 `scale(0.96)` 强化反馈 |

### 4.5 菜谱卡

保留现有横排结构，做三处增强：

1. **来源徽对比度**：`source-badge` 背景从 `rgba(39, 31, 25, 0.52)` 改为 `rgba(31, 22, 16, 0.66)`，添加 1rpx 内描边 `rgba(255, 248, 240, 0.18)`，让小红书 / 链接 / B 站等来源更可读。
2. **状态 switch 触控**：实际可点高度从 52rpx 提到 56rpx；外层包一个透明 padding 6rpx 容器，视觉不变但触控目标进入 64rpx，符合 §5 的触控规范。
3. **摘要兜底**（v1 P0-3 已经提到，这里只补视觉规则）：
   - 优先级：`recipe.summary → recipe.ingredient → recipe.note → 解析步骤第一条`
   - 兜底空时显示一行弱提示 `"还没有备注，点开补一笔～"`，颜色 `--color-text-muted`，斜体（仅小程序系统支持的斜体）

可选装饰：固顶菜（`isPinned`）维持现有"上方书签条"，但用 `--color-accent-terracotta` 的双色渐变取代金棕色，让"想吃"色调系统化。

### 4.6 空态卡（最大断点）

按 v1 §6 P0-2 的状态分类，本节只定义视觉：

```
┌─────────────────────────────────────────────┐
│              ┌──────┐                       │
│              │  🔍  │                       │
│              └──────┘                       │
│                                             │
│        库里没找到"番茄炒蛋"相关的菜谱          │
│                                             │
│       [清除搜索]   [✨ 添加这道菜]            │
└─────────────────────────────────────────────┘
```

| 元素 | 设计 |
| --- | --- |
| 容器 | `border-radius: 30rpx; padding: 56rpx 40rpx; background: --color-surface;` + `clay-shadow-strong` |
| 顶部图标 | 48rpx 方块圆形容器，内填 `--color-bg`，承载主图标 |
| 标题 | 30rpx · 700 · `--color-text-primary` |
| 描述（可选） | 24rpx · 400 · `--color-text-secondary` |
| 主动作 | 80rpx 高、圆角 24rpx、深棕 `--color-brand-brown` 实色，白字，左侧图标 |
| 次动作 | 56rpx 高、胶囊、`--color-surface-warm` 背景，`--color-text-secondary` 文字 |

按钮区 flex 布局，`gap: 12rpx`，宽度自适应；当主动作很长（如"添加这道菜"+ 关键词预填）允许换行。

## 5. 动效规范

### 5.1 入场（进入美食库 / 切回美食库）

```
transform-only return settle：详情页返回后延迟约 96ms，再 translateY(16rpx → -2rpx → 0) (320ms ease-out)
```

落地点：`pages/index/index.vue` 的 `template v-if="activeSection === 'library'"` 外层包一层 `<view class="library-shell" :class="enterMotionClass">`；详情页返回由 `openRecipeDetail` 标记 pending、`onShow` 延迟触发，避免微信原生返回转场叠加时出现整页透明闪屏。普通 `activeSection` 切回美食库仍可即时触发一次轻位移。

### 5.2 列表 stagger

保留现有 `--recipe-card-enter-delay`，但把 `Math.min(index, 4) * 36ms` 改为 `Math.min(index, 6) * 32ms`，让首屏 6 张菜卡都有节奏感，第 7 张及以后无延迟。

### 5.3 状态 pill 切换（thumb 滑动）

参考菜卡内 `.recipe-switch__thumb` 的写法，把状态 pill 三态的激活态改成"thumb 在三个槽位之间平移"：

```
.status-track {
	position: relative;
}

.status-track__thumb {
	position: absolute;
	top: 4rpx;
	left: 4rpx;
	width: calc((100% - 8rpx) / 3);
	height: calc(100% - 8rpx);
	border-radius: 14rpx;
	transition: transform 0.22s cubic-bezier(0.2, 0.8, 0.2, 1), background 0.22s ease;
}
```

切到 `wishlist` 时 `transform: translateX(0)`，`all` 时 `translateX(100%)`，`done` 时 `translateX(200%)`；颜色按状态切换。视觉上比当前"整块换底色"更连贯，也更接近原型胶囊感。

### 5.4 餐别 tab 切换

切换早餐 / 正餐时，整个工具卡向左 / 向右轻位移 8rpx 再回弹（120ms ease-out），让用户感知"内容已切"。

### 5.5 spotlight 多记录滑动

保留现有 `meal-order-spotlight-slide-next/previous`，但把 `transform: translateX(16rpx)` 改为 `translateX(20rpx) + scale(0.98) → 1`，让卡片切换更"翻书感"；持续 240ms。

### 5.6 微交互

| 控件 | 反馈 |
| --- | --- |
| 主按钮（添加这道菜、安排菜单、提交菜单） | `:active { transform: scale(0.96); }` |
| 次按钮（清除搜索、清除筛选） | `:active { transform: translateY(1rpx); }` |
| 状态 pill | thumb 滑动 + active 时整 pill `scale(0.992)` |
| 菜卡点击 | 整卡 `scale(0.992)`（已有，保持） |
| 状态 switch | thumb 平移（已有，保持） |
| 菜谱卡片插入 / 删除 | 新增 `recipe-card-leave` 反向动画（180ms 不阻塞列表） |

### 5.7 不要做的动效

- 不引入毛玻璃 `backdrop-filter` 到正文卡（仅底部导航 / FAB / 选中徽继续使用）。
- 不写连续无限循环的呼吸 / 闪烁动效，不打扰阅读。
- 不在切换 tab 时整页重绘，避免低端安卓掉帧。

## 6. 跨模块一致性

| 模块 | 与美食库的呼应点 |
| --- | --- |
| `kitchen-section`（空间页） | 大卡圆角同为 32rpx，使用同一 `clay-shadow-strong`；统计格的"白底圆角分格 + 衬线数字"思路被美食库 spotlight 借走。 |
| 底部浮动导航 | 仍保持毛玻璃胶囊，不改；FAB 沿用现有星闪图标。 |
| 菜谱详情页（`pages/recipe-detail`） | 进入时的过渡保留现有 transition，但"返回美食库"时走 §5.1 transform-only 归位动效，避免缓存页面透明闪屏。 |
| 菜单详情页（`pages/meal-plan-detail`） | spotlight tap 跳转时，spotlight 卡先 `scale(0.98)` 一拍再淡出，强化"被点中"反馈。 |

设计 token（§3）建议沉淀到 `pages/index/index.vue` 的 `<style>` 顶部，或新建 `styles/tokens.scss` 在 `index.vue` 与 `library-header-section.scss`、`recipe-card-item.scss`、`kitchen-section.scss` 中共享，避免色值漂移。

## 7. 实施分期

> 每一阶段都不修改后端接口、菜谱保存契约、邀请链路或点菜模式判断逻辑。每一阶段做完都建议在微信开发者工具或真机走 §8 验收清单。

### 阶段 A：设计 token 落地（1 个 PR，约 2 小时）

- 在 `pages/index/index.vue` 顶部 `<style>` 增加 token 变量（§3.1）。
- 把 `library-header-section.scss`、`recipe-card-item.scss`、`kitchen-section.scss` 内 hard-code 色值替换为变量（仅替换，不改取值，做一次"重构 + 视觉零差异"提交）。
- 抽 `clay-shadow-soft` / `clay-shadow-strong` SCSS mixin，至少在工具卡、空态卡、菜单 spotlight 三处复用。

### 阶段 B：工具卡 + 状态 pill 视觉升级（必做，对应 §4.3 + §5.3）

- 工具卡圆角、内边距、阴影按 §4.3 调整。
- 状态 pill 加数量徽标（依赖 v1 P0-1 抽组件之后再做也可以，先在 `index.vue` 写也可接受）。
- 状态 pill 增加 `status-track__thumb`，做横向 thumb 滑动。
- 餐别 tab 切换加 8rpx 回弹位移。

### 阶段 C：spotlight → PlanCard 化（对应 §4.2 + §5.5）

- `library-header-section.vue` 模板拆为 `main-icon / main-text / aside-chip` 三块。
- 衬线日期数字（仅日期，不影响其他文案）。
- 多记录滑动加 `scale(0.98) → 1` 收口。

### 阶段 D：菜卡微调 + 空态重做（对应 §4.5 + §4.6）

- 菜卡来源徽对比度调高、外层 padding 容器扩触控。
- 空态拆出 `library-empty-state.vue`（v1 P0-2），按 §4.6 视觉实现，主 / 次动作落到 `index.vue` 的处理函数。

### 阶段 E（P2，可选）：跨模块 polish

- 详情页返回入场动效。
- spotlight tap 反馈。
- 菜卡固顶徽改双色渐变。

### 落地 TODO 看板（执行视图）

> 把上面五个阶段拆成可勾选条目，每个条目都要落到具体文件 / 函数 / 数值，避免开工时还要二次拆解。
> 完成节奏：A → B → C → D → E，每个阶段独立 commit + CHANGELOG，不混搭。

#### 阶段 A · token 落地（零视觉差异）

- [ ] **A1** 在 `pages/index/index.vue` 顶部追加一段非 scoped `<style>` 块，于 `page` 选择器声明 token 变量；首版 **采用当前实际色值** 而非 §3.1 目标值，避免 Phase A 引入视觉漂移
- [ ] **A2** 替换 `pages/index/index.vue` 内 `.search-box`、`.empty-state__title`、`.search-box__input`、`.list-caption__title`、`.empty-state__desc`、`.meal-panel__title`、`.status-pill--active .status-pill__text`、`.status-pill--wishlist.status-pill--active`、`.status-pill__text` 等处的 hard-code 色为 `var(--token)`
- [ ] **A3** 替换 `pages/index/components/library-header-section.scss` 中 `.page-header__title`、`.meal-order-spotlight__title`、`.meal-order-spotlight` 的 token 安全替换点
- [ ] **A4** 替换 `pages/index/components/recipe-card-item.scss` 中 `.recipe-card`（border 0.07）、`.recipe-card--active`（border 0.16）、`.recipe-card__title`、`.recipe-switch`（border 0.07）、`.meal-order-add__text--active`（#fffaf3）等点
- [ ] **A5** 真机或微信开发者工具走一次首屏：暖白底 + 工具卡 + 至少一张菜卡 + spotlight，与改前对比无可见差异
- [ ] **A6** `CHANGELOG.md` 追加：`Changed: 美食库设计 token 抽取（page 级 CSS 变量），零视觉差异`

#### 阶段 B · 工具卡 + 状态 pill 视觉升级

- [ ] **B1** `pages/index/index.vue` `.toolbar`：圆角 `22rpx → 32rpx`，padding `18rpx → 24rpx`，背景 `rgba(255,255,255,0.86) → var(--color-surface)`，阴影换 `--shadow-clay-strong`
- [ ] **B2** `.search-box`：高度 `68rpx → 76rpx`，圆角 `18rpx → 22rpx`，左侧加 `40rpx` 圆形图标章（`background: rgba(91, 74, 59, 0.06); border-radius: 999rpx`）
- [ ] **B3** `pages/index/recipe-card.js` 或 store 暴露 `statusCounts(activeMealType)` 计算属性，返回 `{ all, wishlist, done }`
- [ ] **B4** `pages/index/index.vue` 状态 pill 模板加数字徽标（18rpx · 600，未激活态 `rgba(91, 74, 59, 0.06)`，激活态 `rgba(255, 255, 255, 0.16)`）
- [ ] **B5** 给 `.status-track` 加 `.status-track__thumb`（绝对定位、`width: calc((100% - 8rpx) / 3)`，`transform: translateX(N * 100%)`，0.22s 缓动）；状态切换不再"整块换底色"
- [ ] **B6** 餐别 tab 切换：监听 `activeMealType` 变化，给 `.toolbar` 加 120ms 轻位移 keyframe（`translateX(8rpx) → 0`）
- [ ] **B7** 真机走 §8 的"切换早餐 / 正餐"、"切换全部 / 想吃 / 吃过"两条交互线
- [ ] **B8** `CHANGELOG.md` 追加：`Changed: 美食库工具卡圆角 / 阴影 / 状态 pill 视觉升级 + 状态计数`

#### 阶段 C · spotlight → PlanCard 化

- [ ] **C1** `library-header-section.vue` 的 `meal-order-spotlight` 模板拆为 `__main-icon` / `__main-text` / `__aside-chip` / `__aside-arrow` 四块，左右纵列对齐
- [ ] **C2** 加日历图标章（28rpx 方块、橙色描边、内填白），icon 使用现有 `u-icon` `calendar` 或 `clock` fallback
- [ ] **C3** `__main-text` 第一行：日期数字部分套 `font-family: "Songti SC", "STKaiti", "DejaVu Serif", serif`，36rpx · 700；周几尾随 24rpx · 400；状态 chip 同行右贴
- [ ] **C4** 进度从原本 `meta-text` 改成右上角小 chip（`1/3` · 18rpx · 600 · 圆角 999rpx），arrow 下移到 chip 下方右对齐
- [ ] **C5** spotlight 背景从 `radial + #FFFAF4` 改为 `linear-gradient(145deg, #FFFDF8 0%, #F4ECDF 100%)`
- [ ] **C6** 更新 `meal-order-spotlight-slide-next/previous` keyframe：`translateX 16rpx → 20rpx`、加 `scale(0.98) → 1`、时长 `220ms → 240ms`
- [ ] **C7** 真机滑动多记录，方向感清楚、无闪烁；点击跳转菜单详情链路保持
- [ ] **C8** `CHANGELOG.md` 追加：`Changed: 菜单 spotlight 升级为日历计划卡`

#### 阶段 D · 菜卡微调 + 空态重做

- [ ] **D1** `recipe-card-item.scss` `.recipe-card__source-badge` 背景 `rgba(39,31,25,0.52) → rgba(31,22,16,0.66)`，加 `border: 1rpx solid rgba(255, 248, 240, 0.18)`
- [ ] **D2** 状态 switch 外层包透明 padding 6rpx 容器（视觉不变、触控扩到 64rpx）；`.recipe-switch` 高度 `52rpx → 56rpx`，thumb `46rpx → 50rpx`，对应 `translateX(64rpx)` 调整为 `translateX(62rpx)` 保持视觉对齐
- [ ] **D3** `recipe-card.js` 的 `buildRecipeListSummary`：优先级 `summary → ingredient → note → 解析步骤第一条 → ""`；空时返回 `"还没有备注，点开补一笔～"`（弱文字、斜体）
- [ ] **D4** 新建 `pages/index/components/library-empty-state.vue` + `.scss`：标题 30rpx · 700、描述 24rpx · 400、主动作 80rpx 高深棕实色、次动作 56rpx 暖灰胶囊
- [ ] **D5** 在 `pages/index/index.vue` 内：根据 `searchKeyword` / `activeMealType` / `activeStatus` / `recipes.length` / 同步状态分发四类空态（搜索无结果 / 当前餐别空 / 当前状态空 / 同步失败）
- [ ] **D6** 接入 `handleEmptyPrimary` / `handleEmptySecondary`：清除搜索 → 清空 `searchKeyword`；添加这道菜 → `openAddSheet({ presetTitle: searchKeyword })`（预填扩展可作 P1）；切到正餐 → `setActiveMealType('main')`；查看全部 → `setActiveStatus('all')`；重试同步 → 现有同步入口
- [ ] **D7** 真机走 §8 验收清单的"空态主动作能正常触发"
- [ ] **D8** `CHANGELOG.md` 追加：`Added: 美食库空态主 / 次动作 + 视觉重做`

#### 阶段 E · 跨模块 polish（可选）

- [x] **E1** 详情页返回时美食库走 §5.1 transform-only 归位动效（延迟触发，避免闪屏）
- [x] **E2** spotlight tap 时先 `scale(0.98)` 一拍再淡出再跳转
- [x] **E3** 菜卡 `recipe-card::before` 固顶书签：金棕 → `--color-accent-terracotta` 双色渐变
- [x] **E4** 视觉走查 + 是否回写 `CHANGELOG.md` 由产品决定

#### 共用约束（每阶段都要遵守）

- [ ] 不修改后端接口、`recipe-store.js` 数据归一化、邀请链路、点菜模式判断
- [ ] 不引入远程字体、Tailwind、Lucide、毛玻璃到正文卡
- [ ] `git diff --check` 通过；`pages/index/index.vue` 模板体积不显著膨胀
- [ ] 任一阶段完成后回写 `CHANGELOG.md`，并在阶段提交说明里点名对应 §

## 8. 验收清单

### 视觉

- [ ] 进入美食库首屏：标题、安排菜单按钮、菜单 spotlight（如有）、工具卡（搜索 + 餐别 + 状态）、列表 caption、至少一张菜卡都能在一屏内可读。
- [ ] 工具卡圆角 32rpx、阴影感明显比当前版本更"立体"，但仍保持暖白基调。
- [ ] 状态 pill 显示 `全部 / 想吃 / 吃过` + 各自数量；激活态背景为深色实色 + 白字。
- [ ] 菜单 spotlight 显示日历图标 + 衬线日期 + 周几 + 菜品摘要 + 进度 chip + 箭头。
- [ ] 空态卡有图标 + 标题 + 主动作 + 次动作；主动作背景为 `--color-brand-brown`。
- [ ] 美食库与空间页的卡片圆角、阴影深度一致。

### 交互

- [ ] 切换早餐 / 正餐：餐别 tab 激活态切换 + 工具卡轻位移。
- [ ] 切换 全部 / 想吃 / 吃过：thumb 在三个槽位之间平移，颜色顺滑过渡。
- [ ] 输入搜索词：搜索框聚焦态加深；清除按钮按压有反馈。
- [ ] 多菜单 spotlight 左右滑动：方向感清楚，无闪烁。
- [ ] 菜卡点击 → 进入详情；状态 switch / 加入菜单按钮不触发详情。
- [ ] 空态主动作（清除 / 添加这道菜 / 重试）能正常触发对应处理函数。

### 性能与稳定性

- [ ] 微信开发者工具 iOS / Android 模拟器各跑一次首屏，无掉帧抖动。
- [ ] 真机（任意一台中低端 Android）切换 tab、滚动列表、点击 spotlight 不卡顿。
- [ ] `git diff --check` 通过；`pages/index/index.vue` 模板体积没有显著膨胀（建议借 v1 拆分继续控制）。

### 工程

- [ ] 所有色值通过 token 引用，没有新的 hard-code 色值出现。
- [ ] 不新增后端接口；不调整 `recipe-store.js` 数据归一化逻辑。
- [ ] 点菜模式（`isLibraryMealOrderMode`）下的工具卡、菜卡、spotlight 显示规则保持现状。
- [ ] `CHANGELOG.md` 在阶段 B 与阶段 D 收尾时各回写一次。

## 9. 风险与处理

| 风险 | 描述 | 处理 |
| --- | --- | --- |
| token 引入导致色值漂移 | 替换过程中容易把 `rgba(91,74,59,0.07)` 改成同名变量但少了透明度 | 阶段 A 单独提一次"零视觉变化"PR；改完用真机比对，再开始阶段 B |
| status thumb 在不同设备宽度下定位错位 | 三等分用 `calc()` 容易因父级 padding 变化偏移 | thumb 使用 `transform: translateX(N * 100%)`，配合 grid `repeat(3, 1fr)` + 容器固定 padding |
| spotlight 衬线字体在部分设备 fallback 不一致 | iOS 用 Songti、Android 用 STKaiti，部分鸿蒙缺失 | 仅用于日期数字，最终 fallback 到 `serif`，可读性即可，不追求像素级一致 |
| 入场动效过强引起晕动症 | translateY 与 stagger 叠加可能让低端机出现整页颤动 | 详情页返回只做 transform，不改 opacity；位移控制在 16rpx 内，普通切换仍保持轻量 |
| 空态主 / 次动作和现有数据流不衔接 | "添加这道菜"想预填关键词需要扩展 `openAddSheet` | 第一版只打开新增弹层；预填作为 v1 P1 跟进 |
| 视觉升级后 v1 旧文档过时 | 工具卡圆角、阴影等数值与 v1 §4.2 不一致 | 阶段 A 落地后，在 v1 文档头部加一行"细节请参考 v2"指引；不删除 v1 |

## 10. 不建议本轮做的事

| 不做 | 原因 |
| --- | --- |
| 引入 Tailwind / Lucide 等原型依赖 | 微信小程序与 uview-plus 体系冲突，迁移成本高 |
| 引入远程字体（Playfair Display、Inter 等） | 小程序加载体验差，且无显著体感增益 |
| 把工具卡内的搜索框换成"全屏搜索页" | 与现有最近搜索 / 推荐词链路冲突，留作后续探索 |
| 把底部 FAB 改成 AI 入口 | FAB 当前是饮食管家，已经走流式协议；不本轮重做 |
| 把状态颜色从棕 / 陶土橙 / 鼠尾草绿改成红 / 黄 / 绿高饱和 | 与共享空间气质冲突，也会挤压空间页的深棕主调 |
| 给菜卡加翻转 / 滑动展开等"卡片动效套件" | 列表密度受影响，且现有点击进入详情已能满足 |

## 11. 收口动作

1. 完成阶段 A 后回写 `CHANGELOG.md`：`Changed: 美食库设计 token 抽取，零视觉差异`。
2. 完成阶段 B 后回写：`Changed: 美食库工具卡圆角 / 阴影 / 状态 pill 视觉升级 + 状态计数`。
3. 完成阶段 C 后回写：`Changed: 菜单 spotlight 升级为日历计划卡`。
4. 完成阶段 D 后回写：`Added: 美食库空态主 / 次动作 + 视觉重做`。
5. 阶段 E 视产品节奏，按需立项；不强制本轮。

> 完成上述阶段后，本文档与 v1 文档共同构成美食库视觉与工程的双视角参考。后续如再接 dark mode 或者大屏适配，可直接在本文 §3 与 §5 上做增量。
