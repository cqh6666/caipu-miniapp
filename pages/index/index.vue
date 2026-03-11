<template>
	<view class="app-shell">
		<view class="bg-orb bg-orb--peach"></view>
		<view class="bg-orb bg-orb--butter"></view>

		<view class="page-content">
			<template v-if="activeSection === 'library'">
				<view class="hero">
					<view class="hero__topline">
						<text class="hero__eyebrow">MEAL LIBRARY</text>
						<view class="hero__pill">
							<text class="hero__pill-text">一起干饭 {{ profile.daysTogether }} 天</text>
						</view>
					</view>
					<text class="hero__title">今晚吃什么？</text>
					<view class="hero__quick-stats">
						<view v-for="item in summaryCards" :key="item.label" class="quick-stat">
							<text class="quick-stat__value">{{ item.value }}</text>
							<text class="quick-stat__label">{{ item.label }}</text>
						</view>
					</view>
				</view>

				<view class="search-panel">
					<view class="search-panel__row">
						<view class="search-box">
							<up-icon name="search" size="16" color="#b08a62"></up-icon>
							<input
								v-model="searchKeyword"
								class="search-box__input"
								placeholder="搜菜名或食材"
								placeholder-class="search-box__placeholder"
							/>
						</view>
						<view class="dice-button" @tap="drawTonight">
							<up-icon name="reload" size="15" color="#c66a34"></up-icon>
							<text class="dice-button__text">抽一道</text>
						</view>
					</view>

					<scroll-view class="pill-scroll" scroll-x>
						<view class="pill-track">
							<view
								v-for="tab in statusTabs"
								:key="tab.value"
								class="pill"
								:class="{ 'pill--active': activeStatus === tab.value }"
								@tap="activeStatus = tab.value"
							>
								<text class="pill__text">{{ tab.label }}</text>
							</view>
						</view>
					</scroll-view>
				</view>

				<view class="section-heading">
					<view>
						<text class="section-heading__title">美食库</text>
					</view>
					<text class="section-heading__count">{{ filteredRecipes.length }} 道</text>
				</view>

				<view v-if="filteredRecipes.length" class="masonry">
					<view class="masonry__column">
						<view
							v-for="recipe in masonryColumns.left"
							:key="recipe.id"
							class="recipe-card"
						>
							<view
								class="recipe-card__cover"
								:style="coverStyle(recipe)"
							>
								<view class="recipe-card__cover-top">
									<view class="cover-pill">
										<text class="cover-pill__text">{{ statusMap[recipe.status].label }}</text>
									</view>
								</view>
								<view class="recipe-card__cover-bottom">
									<text class="recipe-card__cover-caption">{{ recipe.mealTag }}</text>
								</view>
							</view>
							<view class="recipe-card__body recipe-card__body--compact">
								<text class="recipe-card__title">{{ recipe.title }}</text>
								<text class="recipe-card__meta-line">{{ recipe.timeTag }} · {{ recipe.likedBy }}</text>
								<view class="recipe-card__actions">
									<view class="card-action card-action--primary" @tap="advanceStatus(recipe)">
										<text class="card-action__text">{{ nextStatusText(recipe.status) }}</text>
									</view>
									<view class="card-action" @tap="requestCook(recipe)">
										<text class="card-action__text">
											{{ hasPendingOrder(recipe.id) ? '已点单' : '向TA点单' }}
										</text>
									</view>
								</view>
							</view>
						</view>
					</view>

					<view class="masonry__column">
						<view
							v-for="recipe in masonryColumns.right"
							:key="recipe.id"
							class="recipe-card"
						>
							<view
								class="recipe-card__cover"
								:style="coverStyle(recipe)"
							>
								<view class="recipe-card__cover-top">
									<view class="cover-pill">
										<text class="cover-pill__text">{{ statusMap[recipe.status].label }}</text>
									</view>
								</view>
								<view class="recipe-card__cover-bottom">
									<text class="recipe-card__cover-caption">{{ recipe.mealTag }}</text>
								</view>
							</view>
							<view class="recipe-card__body recipe-card__body--compact">
								<text class="recipe-card__title">{{ recipe.title }}</text>
								<text class="recipe-card__meta-line">{{ recipe.timeTag }} · {{ recipe.likedBy }}</text>
								<view class="recipe-card__actions">
									<view class="card-action card-action--primary" @tap="advanceStatus(recipe)">
										<text class="card-action__text">{{ nextStatusText(recipe.status) }}</text>
									</view>
									<view class="card-action" @tap="requestCook(recipe)">
										<text class="card-action__text">
											{{ hasPendingOrder(recipe.id) ? '已点单' : '向TA点单' }}
										</text>
									</view>
								</view>
							</view>
						</view>
					</view>
				</view>

				<view v-else class="empty-state">
					<up-icon name="empty-search" size="42" color="#d2a87d"></up-icon>
					<text class="empty-state__title">没有找到匹配的菜谱</text>
					<text class="empty-state__desc">试试换个关键词，或者点中间的加号记录一条新灵感。</text>
				</view>
			</template>

			<template v-else>
				<view class="kitchen-hero">
					<text class="kitchen-hero__eyebrow">OUR KITCHEN</text>
					<view class="kitchen-hero__avatars">
						<up-avatar text="成" size="54" bgColor="#ff9f4f"></up-avatar>
						<view class="kitchen-hero__heart">
							<up-icon name="heart-fill" size="20" color="#ff7f70"></up-icon>
						</view>
						<up-avatar text="满" size="54" bgColor="#f3b7a1"></up-avatar>
					</view>
					<text class="kitchen-hero__title">你们已经一起做完 {{ completedRecipes.length }} 道菜，厨房越来越像家了。</text>
					<text class="kitchen-hero__desc">
						这里放互动提醒、拔草相册和周末采购清单。后续接真实数据后，这里就是你们的数字小家。
					</text>
				</view>

				<view class="stats-grid">
					<view class="metric-card">
						<text class="metric-card__value">{{ chefContribution }}</text>
						<text class="metric-card__label">大厨贡献</text>
					</view>
					<view class="metric-card">
						<text class="metric-card__value">{{ plannerContribution }}</text>
						<text class="metric-card__label">点单灵感</text>
					</view>
					<view class="metric-card">
						<text class="metric-card__value">{{ scheduledRecipes.length }}</text>
						<text class="metric-card__label">本周待做</text>
					</view>
					<view class="metric-card">
						<text class="metric-card__value">{{ pendingOrders.length }}</text>
						<text class="metric-card__label">待接单</text>
					</view>
				</view>

				<view class="panel-card">
					<view class="panel-card__header">
						<text class="panel-card__title">点菜与接单</text>
						<text class="panel-card__meta">{{ pendingOrders.length }} 条提醒</text>
					</view>
					<view v-if="pendingOrders.length" class="order-list">
						<view v-for="order in pendingOrders" :key="order.id" class="order-item">
							<view class="order-item__content">
								<text class="order-item__title">{{ order.from }} 点了《{{ order.title }}》</text>
								<text class="order-item__desc">{{ order.note }}</text>
							</view>
							<view class="order-item__actions">
								<view class="tiny-action tiny-action--strong" @tap="acceptOrder(order)">
									<text class="tiny-action__text">接单</text>
								</view>
								<view class="tiny-action" @tap="dismissOrder(order.id)">
									<text class="tiny-action__text">稍后</text>
								</view>
							</view>
						</view>
					</view>
					<view v-else class="soft-empty">
						<text class="soft-empty__text">暂时没有新的点单提醒，今晚可以按计划慢慢做。</text>
					</view>
				</view>

				<view class="panel-card">
					<view class="panel-card__header">
						<text class="panel-card__title">拔草相册</text>
						<text class="panel-card__meta">{{ completedRecipes.length }} 道已完成</text>
					</view>
					<view class="photo-wall">
						<view
							v-for="recipe in completedRecipes"
							:key="recipe.id"
							class="photo-wall__item"
							:style="coverStyle(recipe)"
						>
							<view class="photo-wall__overlay">
								<text class="photo-wall__title">{{ recipe.title }}</text>
								<text class="photo-wall__subtitle">{{ recipe.photoCount }} 张实拍</text>
							</view>
						</view>
					</view>
				</view>

				<view class="panel-card">
					<view class="panel-card__header">
						<text class="panel-card__title">周末采购清单</text>
						<text class="panel-card__meta">{{ shoppingList.length }} 项</text>
					</view>
					<view v-if="shoppingList.length" class="shopping-list">
						<view
							v-for="item in shoppingList"
							:key="item.key"
							class="shopping-item"
							@tap="toggleShopping(item.key)"
						>
							<view class="shopping-item__check" :class="{ 'shopping-item__check--active': shoppingChecks[item.key] }">
								<up-icon
									v-if="shoppingChecks[item.key]"
									name="checkmark"
									size="14"
									color="#ffffff"
								></up-icon>
							</view>
							<view class="shopping-item__content">
								<text class="shopping-item__title">{{ item.label }}</text>
								<text class="shopping-item__desc">来自 {{ item.from.join('、') }}</text>
							</view>
						</view>
					</view>
					<view v-else class="soft-empty">
						<text class="soft-empty__text">把菜谱标记为“准备做”后，食材会自动汇总到这里。</text>
					</view>
				</view>

				<view class="panel-card">
					<view class="panel-card__header">
						<text class="panel-card__title">本月小盘点</text>
						<text class="panel-card__meta">自动生成的生活线索</text>
					</view>
					<view class="insight-list">
						<view v-for="insight in monthlyInsights" :key="insight.title" class="insight-item">
							<view class="insight-item__icon">
								<up-icon :name="insight.icon" size="18" color="#ff8c4c"></up-icon>
							</view>
							<view class="insight-item__content">
								<text class="insight-item__title">{{ insight.title }}</text>
								<text class="insight-item__desc">{{ insight.desc }}</text>
							</view>
						</view>
					</view>
				</view>
			</template>
		</view>

		<view class="bottom-nav">
			<view
				class="nav-item"
				:class="{ 'nav-item--active': activeSection === 'library' }"
				@tap="activeSection = 'library'"
			>
				<view class="nav-item__icon-shell">
					<up-icon
						:name="activeSection === 'library' ? 'home-fill' : 'home'"
						size="22"
						:color="activeSection === 'library' ? '#ff8c4c' : '#94785c'"
					></up-icon>
				</view>
				<text class="nav-item__label">美食库</text>
			</view>

			<view class="nav-center">
				<view class="nav-fab" @tap="openAddSheet">
					<up-icon name="plus" size="26" color="#ffffff"></up-icon>
				</view>
				<text class="nav-center__label">快速添加</text>
			</view>

			<view
				class="nav-item"
				:class="{ 'nav-item--active': activeSection === 'kitchen' }"
				@tap="activeSection = 'kitchen'"
			>
				<view class="nav-item__icon-shell nav-item__icon-shell--duo">
					<view class="duo-icon duo-icon--left"></view>
					<view class="duo-icon duo-icon--right"></view>
				</view>
				<text class="nav-item__label">我们的厨房</text>
			</view>
		</view>

		<up-popup
			:show="showAddSheet"
			mode="bottom"
			round="28"
			overlayOpacity="0.42"
			:safeAreaInsetBottom="true"
			:touchable="true"
			@close="closeAddSheet"
		>
			<view class="sheet">
				<view class="sheet__header">
					<view>
						<text class="sheet__eyebrow">QUICK CAPTURE</text>
						<text class="sheet__title">捕捉一道新的美味灵感</text>
					</view>
					<view class="sheet__close" @tap="closeAddSheet">
						<up-icon name="close" size="18" color="#8a7157"></up-icon>
					</view>
				</view>

				<view class="sheet__body">
					<view class="form-field">
						<text class="form-field__label">视频链接</text>
						<up-input
							v-model="draft.link"
							placeholder="支持粘贴抖音、小红书或收藏链接"
							border="surround"
							clearable
						></up-input>
					</view>

					<view class="form-field">
						<text class="form-field__label">菜名</text>
						<up-input
							v-model="draft.title"
							placeholder="例如：蒜香黄油虾仁意面"
							border="surround"
							clearable
						></up-input>
					</view>

					<view class="form-field">
						<text class="form-field__label">状态</text>
						<view class="choice-row">
							<view
								v-for="tab in draftStatusOptions"
								:key="tab.value"
								class="choice-chip"
								:class="{ 'choice-chip--active': draft.status === tab.value }"
								@tap="draft.status = tab.value"
							>
								<text class="choice-chip__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="form-field form-field--split">
						<view class="split-col">
							<text class="form-field__label">烹饪节奏</text>
							<view class="choice-row">
								<view
									v-for="item in timeOptions"
									:key="item"
									class="choice-chip choice-chip--small"
									:class="{ 'choice-chip--active': draft.timeTag === item }"
									@tap="draft.timeTag = item"
								>
									<text class="choice-chip__text">{{ item }}</text>
								</view>
							</view>
						</view>
						<view class="split-col">
							<text class="form-field__label">偏爱人群</text>
							<view class="choice-row">
								<view
									v-for="item in whoOptions"
									:key="item"
									class="choice-chip choice-chip--small"
									:class="{ 'choice-chip--active': draft.likedBy === item }"
									@tap="draft.likedBy = item"
								>
									<text class="choice-chip__text">{{ item }}</text>
								</view>
							</view>
						</view>
					</view>

					<view class="form-field">
						<text class="form-field__label">私人备注</text>
						<textarea
							v-model="draft.note"
							class="form-field__textarea"
							placeholder="例如：UP 主说两勺盐，亲测一勺半更刚好。"
							auto-height
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">采购清单食材</text>
						<up-input
							v-model="draft.ingredientsText"
							placeholder="用逗号分隔，例如：虾仁, 意面, 黄油"
							border="surround"
						></up-input>
					</view>
				</view>

				<view class="sheet__footer">
					<up-button
						type="primary"
						text="存入我的美食库"
						shape="circle"
						:customStyle="sheetButtonStyle"
						@tap="submitDraft"
					></up-button>
				</view>
			</view>
		</up-popup>
	</view>
</template>

<script>
const statusMap = {
	all: { label: '全部' },
	wishlist: { label: '想吃' },
	scheduled: { label: '准备做' },
	done: { label: '已拔草' }
}

const tonePairs = [
	['linear-gradient(160deg, #f7c38a, #ee8c6a)', '#fff8f1'],
	['linear-gradient(160deg, #f2d5a0, #d19b62)', '#fffcf4'],
	['linear-gradient(160deg, #f1b5a0, #d56a5d)', '#fff6f5'],
	['linear-gradient(160deg, #a8d6cb, #5ea596)', '#f5fbf9'],
	['linear-gradient(160deg, #d8c3ee, #9b7fcb)', '#faf7ff'],
	['linear-gradient(160deg, #f4b7ba, #ff8c6a)', '#fff7f5']
]

export default {
	data() {
		return {
			statusMap,
			activeSection: 'library',
			activeStatus: 'all',
			searchKeyword: '',
			showAddSheet: false,
			shoppingChecks: {},
			profile: {
				daysTogether: 286,
				chef: '阿成',
				planner: '小满'
			},
			statusTabs: [
				{ label: '全部', value: 'all' },
				{ label: '想吃', value: 'wishlist' },
				{ label: '准备做', value: 'scheduled' },
				{ label: '已拔草', value: 'done' }
			],
			draftStatusOptions: [
				{ label: '先存到心愿单', value: 'wishlist' },
				{ label: '直接提上日程', value: 'scheduled' }
			],
			timeOptions: ['快手菜', '慢熬汤'],
			whoOptions: ['她最爱', '他最爱', '一起都爱'],
			draft: {
				link: '',
				title: '',
				status: 'wishlist',
				timeTag: '快手菜',
				likedBy: '一起都爱',
				note: '',
				ingredientsText: ''
			},
			orders: [
				{
					id: 'order-1',
					recipeId: 'tangcu',
					title: '神仙糖醋排骨',
					from: '小满',
					note: '周末想吃点浓郁的，能不能先帮我接单？'
				}
			],
			recipes: [
				{
					id: 'tangcu',
					title: '神仙糖醋排骨',
					status: 'wishlist',
					timeTag: '慢熬汤',
					likedBy: '她最爱',
					ingredient: '排骨',
					mealTag: '周末大餐',
					note: '先煎再焖，最后大火收汁，颜色会特别亮。',
					ingredients: ['排骨 500g', '冰糖 20g', '香醋 30ml'],
					author: '阿成',
					photoCount: 0,
					coverLabel: '糖',
					coverHeight: 260,
					toneIndex: 0
				},
				{
					id: 'oat-pancake',
					title: '低卡燕麦松饼',
					status: 'done',
					timeTag: '快手菜',
					likedBy: '他最爱',
					ingredient: '鸡蛋',
					mealTag: '早餐回血',
					note: '牛奶别放太多，面糊略稠一点更容易摊出蓬松感。',
					ingredients: ['燕麦 80g', '鸡蛋 2个', '香蕉 1根'],
					author: '小满',
					photoCount: 4,
					coverLabel: '燕',
					coverHeight: 228,
					toneIndex: 1
				},
				{
					id: 'sukiyaki',
					title: '周末寿喜锅',
					status: 'scheduled',
					timeTag: '慢熬汤',
					likedBy: '一起都爱',
					ingredient: '牛肉',
					mealTag: '围炉夜话',
					note: '先把酱汁和洋葱煮香，再下肥牛和豆腐，锅气会很足。',
					ingredients: ['肥牛卷 350g', '豆腐 1盒', '香菇 6朵', '茼蒿 1把'],
					author: '阿成',
					photoCount: 1,
					coverLabel: '锅',
					coverHeight: 282,
					toneIndex: 2
				},
				{
					id: 'garlic-pasta',
					title: '蒜香虾仁意面',
					status: 'wishlist',
					timeTag: '快手菜',
					likedBy: '一起都爱',
					ingredient: '海鲜',
					mealTag: '工作日晚餐',
					note: '黄油和蒜末一定要先小火煸香，虾仁最后再回锅。',
					ingredients: ['意面 180g', '虾仁 200g', '黄油 20g'],
					author: '小满',
					photoCount: 0,
					coverLabel: '意',
					coverHeight: 246,
					toneIndex: 3
				},
				{
					id: 'tomato-beef',
					title: '番茄滑蛋牛肉',
					status: 'done',
					timeTag: '快手菜',
					likedBy: '她最爱',
					ingredient: '牛肉',
					mealTag: '下饭王者',
					note: '牛肉切薄一点腌 10 分钟，鸡蛋最后半熟出锅最嫩。',
					ingredients: ['牛里脊 200g', '番茄 2个', '鸡蛋 3个'],
					author: '阿成',
					photoCount: 3,
					coverLabel: '番',
					coverHeight: 266,
					toneIndex: 4
				},
				{
					id: 'mango-dessert',
					title: '杨枝甘露杯',
					status: 'wishlist',
					timeTag: '快手菜',
					likedBy: '她最爱',
					ingredient: '芒果',
					mealTag: '饭后甜口',
					note: '西米提前煮好冰镇，芒果块最后再拌进去口感最好。',
					ingredients: ['芒果 2个', '椰奶 1盒', '西米 50g'],
					author: '小满',
					photoCount: 0,
					coverLabel: '芒',
					coverHeight: 214,
					toneIndex: 5
				}
			]
		}
	},
	computed: {
		wishlistRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'wishlist')
		},
		scheduledRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'scheduled')
		},
		completedRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'done')
		},
		summaryCards() {
			return [
				{ label: '想吃清单', value: this.wishlistRecipes.length },
				{ label: '提上日程', value: this.scheduledRecipes.length },
				{ label: '已拔草', value: this.completedRecipes.length }
			]
		},
		filteredRecipes() {
			const keyword = this.searchKeyword.trim().toLowerCase()
			return this.recipes.filter((recipe) => {
				const matchedStatus = this.activeStatus === 'all' || recipe.status === this.activeStatus
				const matchedKeyword =
					!keyword ||
					recipe.title.toLowerCase().includes(keyword) ||
					recipe.note.toLowerCase().includes(keyword) ||
					recipe.ingredient.toLowerCase().includes(keyword)
				return matchedStatus && matchedKeyword
			})
		},
		masonryColumns() {
			return this.filteredRecipes.reduce(
				(columns, recipe) => {
					const weight = recipe.coverHeight + 140
					if (columns.leftHeight <= columns.rightHeight) {
						columns.left.push(recipe)
						columns.leftHeight += weight
					} else {
						columns.right.push(recipe)
						columns.rightHeight += weight
					}
					return columns
				},
				{ left: [], right: [], leftHeight: 0, rightHeight: 0 }
			)
		},
		pendingOrders() {
			return this.orders
		},
		chefContribution() {
			return this.recipes.filter((recipe) => recipe.author === this.profile.chef).length
		},
		plannerContribution() {
			return this.recipes.filter((recipe) => recipe.author === this.profile.planner).length
		},
		shoppingList() {
			const sourceMap = {}
			this.scheduledRecipes.forEach((recipe) => {
				recipe.ingredients.forEach((ingredient) => {
					if (!sourceMap[ingredient]) {
						sourceMap[ingredient] = {
							key: ingredient,
							label: ingredient,
							from: [recipe.title]
						}
					} else if (!sourceMap[ingredient].from.includes(recipe.title)) {
						sourceMap[ingredient].from.push(recipe.title)
					}
				})
			})
			return Object.values(sourceMap)
		},
		monthlyInsights() {
			const favoriteIngredient = this.completedRecipes.length
				? this.completedRecipes.reduce((map, recipe) => {
						map[recipe.ingredient] = (map[recipe.ingredient] || 0) + 1
						return map
					}, {})
				: {}
			const ingredientName = Object.keys(favoriteIngredient).sort(
				(a, b) => favoriteIngredient[b] - favoriteIngredient[a]
			)[0] || '牛肉'
			return [
				{
					title: '本月高频食材',
					desc: `你们最近最偏爱的食材是 ${ingredientName}，适合继续扩充这一类菜谱。`,
					icon: 'tags-fill'
				},
				{
					title: '新挑战进度',
					desc: `这个月已经完成 ${this.completedRecipes.length} 道拔草，离满分生活只差继续动手。`,
					icon: 'checkmark-circle-fill'
				},
				{
					title: '周末准备度',
					desc: `${this.scheduledRecipes.length} 道菜已经排进计划，采购清单也同步生成好了。`,
					icon: 'calendar-fill'
				}
			]
		},
		buttonPrimaryStyle() {
			return {
				width: '100%',
				height: '78rpx',
				fontWeight: '600'
			}
		},
		buttonGhostStyle() {
			return {
				width: '100%',
				height: '78rpx',
				fontWeight: '600',
				backgroundColor: 'rgba(255, 248, 241, 0.18)',
				borderColor: 'rgba(255, 252, 248, 0.38)'
			}
		},
		sheetButtonStyle() {
			return {
				width: '100%',
				height: '84rpx',
				fontWeight: '600'
			}
		}
	},
	methods: {
		coverStyle(recipe) {
			const pair = tonePairs[recipe.toneIndex % tonePairs.length]
			return {
				height: '132rpx',
				backgroundImage: pair[0]
			}
		},
		nextStatusText(status) {
			if (status === 'wishlist') return '提上日程'
			if (status === 'scheduled') return '标记拔草'
			return '重新想吃'
		},
		advanceStatus(recipe) {
			const nextStatus =
				recipe.status === 'wishlist'
					? 'scheduled'
					: recipe.status === 'scheduled'
					? 'done'
					: 'wishlist'
			this.setRecipeStatus(recipe.id, nextStatus)
		},
		setRecipeStatus(recipeId, status) {
			this.recipes = this.recipes.map((recipe) => {
				if (recipe.id !== recipeId) return recipe
				const photoCount = status === 'done' ? Math.max(recipe.photoCount, 1) : recipe.photoCount
				return { ...recipe, status, photoCount }
			})
			if (status === 'scheduled') {
				this.orders = this.orders.filter((order) => order.recipeId !== recipeId)
			}
			this.activeStatus = status === 'done' ? 'done' : this.activeStatus
			uni.showToast({
				title: `已更新为${this.statusMap[status].label}`,
				icon: 'none'
			})
		},
		hasPendingOrder(recipeId) {
			return this.orders.some((order) => order.recipeId === recipeId)
		},
		requestCook(recipe) {
			if (this.hasPendingOrder(recipe.id)) {
				uni.showToast({
					title: '这道菜已经在待接单列表里',
					icon: 'none'
				})
				return
			}
			this.orders = [
				{
					id: `order-${Date.now()}`,
					recipeId: recipe.id,
					title: recipe.title,
					from: this.profile.planner,
					note: `想吃这道${recipe.mealTag}，有空的话帮我安排进本周计划。`
				},
				...this.orders
			]
			uni.showToast({
				title: '已向TA点单',
				icon: 'none'
			})
		},
		acceptOrder(order) {
			this.setRecipeStatus(order.recipeId, 'scheduled')
			this.orders = this.orders.filter((item) => item.id !== order.id)
			this.activeSection = 'kitchen'
		},
		dismissOrder(orderId) {
			this.orders = this.orders.filter((item) => item.id !== orderId)
		},
		drawTonight() {
			const pool = this.wishlistRecipes.length ? this.wishlistRecipes : this.recipes
			if (!pool.length) {
				uni.showToast({
					title: '先添加几道想吃的菜吧',
					icon: 'none'
				})
				return
			}
			const picked = pool[Math.floor(Math.random() * pool.length)]
			uni.showToast({
				title: `今晚试试：${picked.title}`,
				icon: 'none'
			})
		},
		openAddSheet() {
			this.showAddSheet = true
		},
		closeAddSheet() {
			this.showAddSheet = false
		},
		submitDraft() {
			const title = this.draft.title.trim()
			if (!title) {
				uni.showToast({
					title: '先给这道菜起个名字',
					icon: 'none'
				})
				return
			}
			const ingredients = this.draft.ingredientsText
				.split(/[,，]/)
				.map((item) => item.trim())
				.filter(Boolean)
			const newRecipe = {
				id: `recipe-${Date.now()}`,
				title,
				status: this.draft.status,
				timeTag: this.draft.timeTag,
				likedBy: this.draft.likedBy,
				ingredient: ingredients[0] || '灵感',
				mealTag: this.draft.status === 'scheduled' ? '本周准备做' : '新鲜收藏',
				note: this.draft.note.trim() || '刚刚捕捉到的新灵感，等你们一起把它变成好吃的日常。',
				ingredients: ingredients.length ? ingredients : ['待补充食材'],
				author: this.profile.planner,
				photoCount: 0,
				coverLabel: title.slice(0, 1),
				coverHeight: 220 + (this.recipes.length % 4) * 18,
				toneIndex: this.recipes.length % tonePairs.length
			}
			this.recipes = [newRecipe, ...this.recipes]
			this.activeSection = 'library'
			this.activeStatus = 'all'
			this.searchKeyword = ''
			this.draft = {
				link: '',
				title: '',
				status: 'wishlist',
				timeTag: '快手菜',
				likedBy: '一起都爱',
				note: '',
				ingredientsText: ''
			}
			this.showAddSheet = false
			uni.showToast({
				title: '已存入美食库',
				icon: 'none'
			})
		},
		toggleShopping(key) {
			this.shoppingChecks = {
				...this.shoppingChecks,
				[key]: !this.shoppingChecks[key]
			}
		}
	}
}
</script>

<style lang="scss" scoped>
	$bg: #fdf7f0;
	$bg-soft: #fffaf4;
	$surface: rgba(255, 255, 255, 0.9);
	$surface-strong: #ffffff;
	$text-main: #2e241d;
	$text-soft: #7d6958;
	$text-faint: #a28a76;
	$accent-deep: #df6d2f;
	$shadow: 0 20rpx 50rpx rgba(120, 76, 35, 0.1);

	.app-shell {
		min-height: 100vh;
		position: relative;
		background:
			radial-gradient(circle at top left, rgba(255, 210, 160, 0.55), transparent 36%),
			linear-gradient(180deg, #fffdf9 0%, #fdf7f0 22%, #f9f3ec 100%);
		overflow: hidden;
	}

	.bg-orb {
		position: absolute;
		border-radius: 999rpx;
		filter: blur(6rpx);
		pointer-events: none;
	}

	.bg-orb--peach {
		top: 120rpx;
		right: -60rpx;
		width: 260rpx;
		height: 260rpx;
		background: rgba(255, 181, 124, 0.18);
	}

	.bg-orb--butter {
		top: 540rpx;
		left: -110rpx;
		width: 280rpx;
		height: 280rpx;
		background: rgba(255, 235, 175, 0.22);
	}

	.page-content {
		position: relative;
		z-index: 1;
		padding: 34rpx 24rpx 178rpx;
	}

	.hero,
	.kitchen-hero {
		padding: 20rpx 6rpx 8rpx;
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.hero__topline,
	.kitchen-hero__avatars {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18rpx;
	}

	.hero__eyebrow,
	.kitchen-hero__eyebrow,
	.sheet__eyebrow {
		font-size: 22rpx;
		letter-spacing: 3rpx;
		font-weight: 700;
		color: #df6d2f;
	}

	.hero__pill {
		padding: 12rpx 20rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.65);
		backdrop-filter: blur(14rpx);
	}

	.hero__pill-text {
		font-size: 22rpx;
		color: #7d6958;
	}

	.hero__title,
	.kitchen-hero__title {
		font-size: 56rpx;
		line-height: 1.14;
		font-weight: 700;
		color: #2e241d;
	}

	.hero__desc,
	.kitchen-hero__desc {
		font-size: 28rpx;
		line-height: 1.72;
		color: #7d6958;
	}

	.hero__stats,
	.stats-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 16rpx;
	}

	.summary-card,
	.metric-card {
		padding: 24rpx 20rpx;
		border-radius: 28rpx;
		background: rgba(255, 255, 255, 0.9);
		box-shadow: 0 20rpx 50rpx rgba(120, 76, 35, 0.1);
		border: 1px solid rgba(255, 255, 255, 0.78);
		backdrop-filter: blur(18rpx);
	}

	.summary-card__value,
	.metric-card__value {
		display: block;
		font-size: 40rpx;
		font-weight: 700;
		color: #2e241d;
	}

	.summary-card__label,
	.metric-card__label {
		display: block;
		margin-top: 10rpx;
		font-size: 22rpx;
		color: #a28a76;
	}

	.search-panel,
	.panel-card {
		margin-top: 26rpx;
		padding: 26rpx;
		border-radius: 32rpx;
		background: rgba(255, 255, 255, 0.9);
		box-shadow: 0 20rpx 50rpx rgba(120, 76, 35, 0.1);
		border: 1px solid rgba(255, 255, 255, 0.78);
		backdrop-filter: blur(18rpx);
	}

	.search-panel__row {
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.search-box {
		flex: 1;
		height: 76rpx;
		display: flex;
		align-items: center;
		gap: 10rpx;
		padding: 0 18rpx;
		border-radius: 18rpx;
		background: rgba(255, 255, 255, 0.78);
		border: 1px solid rgba(196, 153, 112, 0.08);
	}

	.search-box__input {
		flex: 1;
		height: 76rpx;
		font-size: 26rpx;
		color: #2e241d;
	}

	.search-box__placeholder {
		color: #b89d84;
	}

	.dice-button {
		width: 132rpx;
		height: 76rpx;
		border-radius: 18rpx;
		background: rgba(255, 245, 235, 0.92);
		border: 1px solid rgba(228, 174, 129, 0.32);
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8rpx;
	}

	.dice-button__text {
		font-size: 22rpx;
		font-weight: 600;
		color: #c66a34;
	}

	.pill-scroll {
		margin-top: 18rpx;
		white-space: nowrap;
	}

	.pill-track {
		display: inline-flex;
		gap: 14rpx;
		padding-right: 24rpx;
	}

	.pill,
	.choice-chip {
		flex-shrink: 0;
		padding: 14rpx 24rpx;
		border-radius: 999rpx;
		background: #f2e8dc;
		border: 1px solid transparent;
		transition: all 0.2s ease;
	}

	.pill--active,
	.choice-chip--active {
		background: linear-gradient(135deg, #ff9a58, #f47d41);
		box-shadow: 0 12rpx 24rpx rgba(244, 125, 65, 0.18);
	}

	.pill__text,
	.choice-chip__text {
		font-size: 24rpx;
		font-weight: 600;
		color: #8e6b4b;
	}

	.pill--active .pill__text,
	.choice-chip--active .choice-chip__text {
		color: #fffaf5;
	}

	.choice-chip--small {
		padding: 12rpx 18rpx;
	}

	.panel-card__header,
	.section-heading,
	.sheet__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18rpx;
	}

	.choice-row,
	.recipe-card__actions,
	.insight-list {
		display: flex;
		flex-wrap: wrap;
		gap: 14rpx;
	}

	.meta-chip {
		padding: 10rpx 18rpx;
		border-radius: 999rpx;
		background: rgba(255, 248, 241, 0.14);
	}

	.meta-chip--soft {
		background: #f8eee2;
	}

	.meta-chip__text {
		font-size: 22rpx;
		color: #fff8f1;
	}

	.meta-chip--soft .meta-chip__text {
		color: #8b6a4c;
	}

	.section-heading {
		margin-top: 30rpx;
	}

	.section-heading__title,
	.panel-card__title {
		display: block;
		font-size: 34rpx;
		font-weight: 700;
		color: #2e241d;
	}

	.panel-card__meta {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		color: #a28a76;
	}

	.section-heading__count {
		font-size: 24rpx;
		font-weight: 600;
		color: #df6d2f;
	}

	.masonry {
		margin-top: 18rpx;
		display: flex;
		gap: 16rpx;
		align-items: flex-start;
	}

	.masonry__column {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}

	.recipe-card {
		border-radius: 28rpx;
		overflow: hidden;
		background: #ffffff;
		box-shadow: 0 20rpx 50rpx rgba(120, 76, 35, 0.1);
	}

	.recipe-card__cover {
		position: relative;
		padding: 20rpx;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}

	.recipe-card__cover-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.cover-pill {
		padding: 10rpx 16rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.22);
		backdrop-filter: blur(12rpx);
	}

	.cover-pill__text {
		font-size: 20rpx;
		font-weight: 600;
		color: #fffdf9;
	}

	.photo-wall__title {
		font-size: 62rpx;
		font-weight: 700;
		color: rgba(255, 255, 255, 0.88);
	}

	.recipe-card__body {
		padding: 24rpx 22rpx;
	}

	.recipe-card__title,
	.order-item__title,
	.shopping-item__title,
	.insight-item__title,
	.sheet__title {
		display: block;
		font-size: 30rpx;
		font-weight: 700;
		color: #2e241d;
		line-height: 1.34;
	}

	.order-item__desc,
	.shopping-item__desc,
	.insight-item__desc,
	.soft-empty__text,
	.empty-state__desc,
	.photo-wall__subtitle {
		display: block;
		margin-top: 10rpx;
		font-size: 24rpx;
		line-height: 1.64;
		color: #7d6958;
	}

	.recipe-card__actions {
		margin-top: 18rpx;
	}

	.card-action,
	.tiny-action {
		flex: 1;
		min-width: 0;
		height: 70rpx;
		border-radius: 20rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #f6eee4;
	}

	.card-action--primary,
	.tiny-action--strong {
		background: linear-gradient(135deg, #ff9a58, #f47d41);
	}

	.card-action__text,
	.tiny-action__text {
		font-size: 24rpx;
		font-weight: 600;
		color: #885f3b;
	}

	.card-action--primary .card-action__text,
	.tiny-action--strong .tiny-action__text {
		color: #fffaf6;
	}

	.empty-state,
	.soft-empty {
		margin-top: 22rpx;
		padding: 60rpx 32rpx;
		border-radius: 28rpx;
		background: rgba(255, 255, 255, 0.64);
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 14rpx;
	}

	.empty-state__title {
		font-size: 30rpx;
		font-weight: 700;
		color: #2e241d;
	}

	.kitchen-hero__avatars {
		justify-content: center;
	}

	.kitchen-hero__heart {
		width: 60rpx;
		height: 60rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.78);
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 12rpx 28rpx rgba(255, 127, 112, 0.12);
	}

	.panel-card {
		margin-top: 24rpx;
		padding: 26rpx 24rpx;
	}

	.order-list,
	.shopping-list {
		margin-top: 18rpx;
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.order-item,
	.shopping-item,
	.insight-item {
		padding: 20rpx;
		border-radius: 24rpx;
		background: #fff8f1;
		border: 1px solid rgba(222, 192, 160, 0.22);
	}

	.order-item {
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.order-item__actions {
		display: flex;
		gap: 12rpx;
	}

	.photo-wall {
		margin-top: 18rpx;
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 14rpx;
	}

	.photo-wall__item {
		height: 206rpx;
		border-radius: 24rpx;
		overflow: hidden;
		display: flex;
		align-items: flex-end;
	}

	.photo-wall__overlay {
		width: 100%;
		padding: 18rpx;
		background: linear-gradient(180deg, transparent, rgba(45, 33, 25, 0.7));
	}

	.photo-wall__title {
		font-size: 26rpx;
	}

	.photo-wall__subtitle {
		margin-top: 6rpx;
		color: rgba(255, 248, 241, 0.72);
	}

	.shopping-item {
		display: flex;
		align-items: center;
		gap: 16rpx;
	}

	.shopping-item__check {
		width: 42rpx;
		height: 42rpx;
		border-radius: 14rpx;
		border: 2rpx solid rgba(255, 140, 76, 0.32);
		display: flex;
		align-items: center;
		justify-content: center;
		background: #fffdf9;
		flex-shrink: 0;
	}

	.shopping-item__check--active {
		background: linear-gradient(135deg, #ff9a58, #f47d41);
		border-color: transparent;
	}

	.shopping-item__content,
	.insight-item__content {
		flex: 1;
		min-width: 0;
	}

	.insight-list {
		margin-top: 18rpx;
		flex-direction: column;
	}

	.insight-item {
		display: flex;
		align-items: flex-start;
		gap: 16rpx;
	}

	.insight-item__icon {
		width: 54rpx;
		height: 54rpx;
		border-radius: 18rpx;
		background: rgba(255, 140, 76, 0.12);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.bottom-nav {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 9;
		padding: 12rpx 24rpx calc(env(safe-area-inset-bottom) + 12rpx);
		background: linear-gradient(180deg, rgba(253, 247, 240, 0), rgba(253, 247, 240, 0.82) 14%, rgba(255, 255, 255, 0.98) 28%);
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
	}

	.nav-item,
	.nav-center {
		width: 30%;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10rpx;
	}

	.nav-item__icon-shell {
		width: 82rpx;
		height: 82rpx;
		border-radius: 26rpx;
		background: rgba(255, 255, 255, 0.94);
		box-shadow: 0 12rpx 28rpx rgba(120, 76, 35, 0.08);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.nav-item__icon-shell--duo {
		gap: 0;
	}

	.duo-icon {
		width: 26rpx;
		height: 26rpx;
		border-radius: 999rpx;
		background: #f2b49a;
	}

	.duo-icon--left {
		transform: translateX(8rpx);
		background: #ff9a58;
	}

	.duo-icon--right {
		transform: translateX(-8rpx);
		background: #f3c3b3;
	}

	.nav-item__label,
	.nav-center__label {
		font-size: 22rpx;
		color: #927761;
		font-weight: 600;
	}

	.nav-item--active .nav-item__label {
		color: #df6d2f;
	}

	.nav-center {
		transform: translateY(-20rpx);
	}

	.nav-fab {
		width: 112rpx;
		height: 112rpx;
		border-radius: 999rpx;
		border: 8rpx solid rgba(255, 255, 255, 0.96);
		background: linear-gradient(135deg, #ff9a58, #f47d41);
		box-shadow: 0 24rpx 38rpx rgba(244, 125, 65, 0.24);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.sheet {
		padding: 26rpx 24rpx 34rpx;
		background: #fffdf8;
	}

	.sheet__close {
		width: 72rpx;
		height: 72rpx;
		border-radius: 22rpx;
		background: #f5ebdf;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.sheet__body {
		margin-top: 24rpx;
		display: flex;
		flex-direction: column;
		gap: 20rpx;
	}

	.form-field {
		display: flex;
		flex-direction: column;
		gap: 12rpx;
	}

	.form-field--split {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 18rpx;
	}

	.split-col {
		display: flex;
		flex-direction: column;
		gap: 12rpx;
	}

	.form-field__label {
		font-size: 24rpx;
		font-weight: 600;
		color: #7d6958;
	}

	.form-field__textarea {
		width: 100%;
		min-height: 128rpx;
		padding: 22rpx;
		box-sizing: border-box;
		border-radius: 24rpx;
		background: #fbf4ec;
		border: 1px solid rgba(196, 153, 112, 0.12);
		font-size: 28rpx;
		line-height: 1.6;
		color: #2e241d;
	}

	.sheet__footer {
		margin-top: 26rpx;
	}

	.hero {
		gap: 14rpx;
	}

	.hero__title {
		font-size: 44rpx;
		line-height: 1.18;
	}

	.hero__quick-stats {
		display: flex;
		flex-wrap: wrap;
		gap: 12rpx;
	}

	.quick-stat {
		padding: 12rpx 18rpx;
		border-radius: 20rpx;
		background: rgba(255, 255, 255, 0.74);
		border: 1px solid rgba(223, 190, 157, 0.22);
		display: flex;
		align-items: center;
		gap: 8rpx;
	}

	.quick-stat__value {
		font-size: 24rpx;
		font-weight: 700;
		color: #2e241d;
	}

	.quick-stat__label {
		font-size: 22rpx;
		color: #8f7762;
	}

	.search-panel {
		margin-top: 18rpx;
		padding: 22rpx;
	}

	.section-heading {
		margin-top: 24rpx;
	}

	.recipe-card {
		border-radius: 24rpx;
		box-shadow: 0 12rpx 28rpx rgba(120, 76, 35, 0.08);
	}

	.recipe-card__cover {
		padding: 16rpx 18rpx;
	}

	.recipe-card__cover-bottom {
		margin-top: auto;
	}

	.recipe-card__cover-caption {
		font-size: 22rpx;
		font-weight: 600;
		color: rgba(255, 253, 249, 0.92);
	}

	.recipe-card__body--compact {
		padding: 18rpx;
	}

	.recipe-card__title {
		font-size: 28rpx;
	}

	.recipe-card__meta-line {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		color: #7d6958;
	}

	.recipe-card__actions {
		margin-top: 14rpx;
		gap: 10rpx;
	}

	.card-action {
		height: 62rpx;
		border-radius: 18rpx;
	}
</style>
