<template>
	<view class="space-stats-sheet space-stats-sheet--page">
		<view class="space-stats-sheet__header">
			<view class="space-stats-sheet__meta space-stats-sheet__meta--inline">
				<text class="space-stats-sheet__updated-text">更新时间：{{ updatedLabel || '暂无同步记录' }}</text>
				<view v-if="syncErrorMessage" class="space-stats-sheet__source-chip space-stats-sheet__source-chip--warning">
					<up-icon name="warning" size="12" color="#bf715f"></up-icon>
					<text class="space-stats-sheet__source-text">后端暂不可用 · 本地聚合</text>
				</view>
				<view v-else-if="isCacheSnapshot" class="space-stats-sheet__source-chip">
					<up-icon name="clock" size="12" color="#8a7d70"></up-icon>
					<text class="space-stats-sheet__source-text">本地聚合</text>
				</view>
			</view>
			<view
				class="space-stats-sheet__refresh-btn"
				hover-class="space-stats-sheet__refresh-btn--hover"
				hover-start-time="0"
				hover-stay-time="160"
				@tap="handleRefresh"
			>
				<up-icon
					name="reload"
					size="15"
					color="#6e5f50"
					:class="{ 'space-stats-sheet__refresh-icon--spinning': isRefreshing }"
				></up-icon>
				<text class="space-stats-sheet__refresh-text">{{ isRefreshing ? '刷新中' : '刷新' }}</text>
			</view>
		</view>

		<!-- 分组 Tab（V1.1） -->
		<scroll-view class="space-stats-sheet__tabs" scroll-x :show-scrollbar="false">
			<view class="space-stats-sheet__tabs-inner">
				<view
					v-for="tab in tabs"
					:key="tab.value"
					class="stats-tab"
					:class="{ 'stats-tab--active': activeTab === tab.value }"
					hover-class="stats-tab--hover"
					hover-start-time="0"
					hover-stay-time="160"
					@tap="activeTab = tab.value"
				>
					<text class="stats-tab__text">{{ tab.label }}</text>
				</view>
			</view>
		</scroll-view>

		<scroll-view class="space-stats-sheet__scroll" scroll-y :show-scrollbar="false">
			<!-- ============ 总览 ============ -->
			<view v-if="isTabMounted('overview')" v-show="activeTab === 'overview'">
				<view class="stats-section">
					<text class="stats-section__title">高分复访推荐</text>
					<view v-if="topRevisitPlaces.length" class="revisit-list">
						<view
							v-for="(place, index) in topRevisitPlaces"
							:key="place.id || index"
							class="revisit-card"
							:style="{ animationDelay: `${index * 60}ms` }"
							hover-class="revisit-card--hover"
							hover-start-time="0"
							hover-stay-time="160"
							@tap="onAction({ actionType: 'view-place-detail', placeId: place.id })"
						>
							<view class="revisit-card__thumb">
								<image v-if="place.imageUrl" class="revisit-card__thumb-image" :src="place.imageUrl" mode="aspectFill"></image>
								<up-icon v-else name="map-fill" size="20" color="#a08975"></up-icon>
							</view>
							<view class="revisit-card__body">
								<text class="revisit-card__name">{{ place.name }}</text>
								<view class="revisit-card__stars">
									<up-icon
										v-for="star in 5"
										:key="`revisit-star-${index}-${star}`"
										:name="star <= place.revisitRating ? 'star-fill' : 'star'"
										size="13"
										:color="star <= place.revisitRating ? '#f4a236' : '#d4c4b8'"
									></up-icon>
								</view>
								<view v-if="place.recommendedItems.length" class="revisit-card__chips">
									<view v-for="item in place.recommendedItems" :key="item" class="revisit-card__chip">
										<text class="revisit-card__chip-text">{{ item }}</text>
									</view>
								</view>
							</view>
							<up-icon name="arrow-right" size="14" color="#a08975"></up-icon>
						</view>
					</view>
					<view v-else class="stats-empty-hint">
						<text class="stats-empty-hint__text">还没有 4 分以上的复访推荐，去打卡点记录一次体验吧。</text>
					</view>
				</view>

				<!-- 近期动态 + 迷你趋势（V2 有 trends 时展示 sparkline） -->
				<view class="insight-card" :class="{ 'insight-card--last': !showContributors }">
					<text class="insight-card__title">近期动态</text>
					<text class="recent-activity-text">{{ recentActivityText }}</text>
					<view v-if="hasTrends" class="trend-list">
						<view v-for="trend in trendRows" :key="trend.key" class="trend-row">
							<text class="trend-row__label">{{ trend.label }}</text>
							<view class="trend-row__bars">
								<view
									v-for="(point, index) in trend.points"
									:key="`${trend.key}-${index}`"
									class="trend-bar"
									:style="{ height: `${point.height}rpx`, background: trend.color }"
								></view>
							</view>
							<text class="trend-row__total">{{ trend.total }}</text>
						</view>
					</view>
				</view>

				<!-- 成员贡献（V2，多成员空间才展示） -->
				<view v-if="showContributors" class="insight-card insight-card--last">
					<text class="insight-card__title">成员贡献</text>
					<view class="contributor-list">
						<view v-for="member in contributors" :key="member.userId" class="contributor-item">
							<view class="contributor-item__avatar">
								<image v-if="member.avatarUrl" class="contributor-item__avatar-image" :src="member.avatarUrl" mode="aspectFill"></image>
								<text v-else class="contributor-item__avatar-text">{{ memberInitial(member) }}</text>
							</view>
							<view class="contributor-item__body">
								<text class="contributor-item__name">{{ member.nickname || ('厨友 ' + member.userId) }}</text>
								<text class="contributor-item__detail">菜 {{ member.recipeCreatedTotal }} · 店 {{ member.placeCreatedTotal }} · 菜单 {{ member.mealPlanSubmittedTotal }}</text>
							</view>
							<text class="contributor-item__total">{{ member.total }}</text>
						</view>
					</view>
				</view>
			</view>

			<!-- ============ 美食库 ============ -->
			<view v-if="isTabMounted('recipes')" v-show="activeTab === 'recipes'">
				<view class="ratio-grid">
					<view class="ratio-card">
						<text class="ratio-card__label">想吃 / 吃过</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main ratio-card__main--sage">{{ animatedInt('recipeWishlist') }}</text>
							<text class="ratio-card__slash">/</text>
							<text class="ratio-card__sub">{{ animatedInt('recipeDone') }}</text>
						</view>
					</view>
					<view class="ratio-card">
						<text class="ratio-card__label">早餐 / 正餐</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main ratio-card__main--caramel">{{ animatedInt('recipeBreakfast') }}</text>
							<text class="ratio-card__slash">/</text>
							<text class="ratio-card__sub">{{ animatedInt('recipeMain') }}</text>
						</view>
					</view>
				</view>

				<view class="insight-card insight-card--last">
					<text class="insight-card__title">内容完整度</text>
					<view class="progress-row">
						<view class="progress-row__head">
							<text class="progress-row__label">图片覆盖率</text>
							<text class="progress-row__value progress-row__value--sage">{{ animatedInt('imageCoveragePct') }}%</text>
						</view>
						<view class="progress-track">
							<view class="progress-fill progress-fill--sage" :style="{ width: imageCoveragePct + '%' }"></view>
						</view>
					</view>
					<view class="progress-row">
						<view class="progress-row__head">
							<text class="progress-row__label">已智能解析</text>
							<text class="progress-row__value progress-row__value--terracotta">{{ animatedInt('recipeParsedTotal') }} 道</text>
						</view>
						<view class="progress-track">
							<view class="progress-fill progress-fill--terracotta" :style="{ width: parsedPct + '%' }"></view>
						</view>
					</view>
				</view>
			</view>

			<!-- ============ 打卡库 ============ -->
			<view v-if="isTabMounted('places')" v-show="activeTab === 'places'">
				<view class="ratio-grid">
					<view class="ratio-card">
						<text class="ratio-card__label">想去 / 去过</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main ratio-card__main--terracotta">{{ animatedInt('placeWant') }}</text>
							<text class="ratio-card__slash">/</text>
							<text class="ratio-card__sub">{{ animatedInt('placeVisited') }}</text>
						</view>
					</view>
					<view class="ratio-card">
						<text class="ratio-card__label">有定位比例</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main ratio-card__main--sage">{{ animatedInt('placeLocated') }}</text>
							<text class="ratio-card__slash">/</text>
							<text class="ratio-card__sub">{{ animatedInt('placeTotal') }}</text>
						</view>
					</view>
				</view>

				<view class="insight-card" :class="{ 'insight-card--last': !sceneTags.length && !recommendedTags.length }">
					<text class="insight-card__title">重访意愿分布（已打卡 {{ animatedInt('ratedPlaceTotal') }} 家）</text>
					<view v-if="hasRevisitRatings" class="rating-dist">
						<view v-for="row in revisitBars" :key="row.rating" class="rating-row">
							<view class="rating-row__stars">
								<up-icon name="star-fill" size="12" color="#f4a236"></up-icon>
								<text class="rating-row__label">{{ row.rating }}</text>
							</view>
							<view class="rating-row__track">
								<view class="rating-row__fill" :style="{ width: row.pct + '%' }"></view>
							</view>
							<text class="rating-row__count">{{ row.count }}</text>
						</view>
					</view>
					<view v-else class="stats-empty-hint">
						<text class="stats-empty-hint__text">还没有重访评分，去过的店记一笔评分就能看到分布。</text>
					</view>
				</view>

				<view
					v-if="sceneTags.length"
					class="insight-card"
					:class="{ 'insight-card--last': !recommendedTags.length }"
				>
					<text class="insight-card__title">高频场景标签</text>
					<view class="tag-cloud">
						<view v-for="tag in sceneTags" :key="tag.label" class="tag-cloud__item tag-cloud__item--scene">
							<text class="tag-cloud__text">{{ tag.label }}</text>
							<text class="tag-cloud__count">{{ tag.count }}</text>
						</view>
					</view>
				</view>

				<view v-if="recommendedTags.length" class="insight-card insight-card--last">
					<text class="insight-card__title">常点推荐</text>
					<view class="tag-cloud">
						<view v-for="tag in recommendedTags" :key="tag.label" class="tag-cloud__item">
							<text class="tag-cloud__text">{{ tag.label }}</text>
							<text class="tag-cloud__count">{{ tag.count }}</text>
						</view>
					</view>
				</view>
			</view>

			<!-- ============ 菜单安排 ============ -->
			<view v-if="isTabMounted('mealPlans')" v-show="activeTab === 'mealPlans'">
				<view class="ratio-grid">
					<view class="ratio-card">
						<text class="ratio-card__label">已安排天数</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main">{{ animatedInt('mealSubmittedDays') }}</text>
							<text class="ratio-card__unit">天</text>
						</view>
					</view>
					<view class="ratio-card">
						<text class="ratio-card__label">草稿天数</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main ratio-card__main--terracotta">{{ animatedInt('mealDraftDays') }}</text>
							<text class="ratio-card__unit">天</text>
						</view>
					</view>
				</view>

				<view class="insight-card">
					<text class="insight-card__title">下一次安排</text>
					<view
						v-if="mealPlanStats.nextPlan"
						class="next-plan"
						hover-class="next-plan--hover"
						hover-start-time="0"
						hover-stay-time="160"
						@tap="onAction({ actionType: 'view-draft-meal-plan' })"
					>
						<view class="next-plan__body">
							<text class="next-plan__date">{{ mealPlanStats.nextPlan.planDate }}</text>
							<text class="next-plan__count">共 {{ mealPlanStats.nextPlan.dishCount }} 道菜</text>
							<text v-if="nextPlanTitles" class="next-plan__titles">{{ nextPlanTitles }}</text>
						</view>
						<up-icon name="arrow-right" size="14" color="#a08975"></up-icon>
					</view>
					<view v-else class="stats-empty-hint">
						<text class="stats-empty-hint__text">还没有未来的菜单安排，去点菜模式安排一餐吧。</text>
					</view>
				</view>

				<view class="ratio-grid ratio-grid--last">
					<view class="ratio-card">
						<text class="ratio-card__label">平均每餐</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main">{{ animatedFixed('mealAverageDishCount', 1) }}</text>
							<text class="ratio-card__unit">道菜</text>
						</view>
					</view>
					<view class="ratio-card">
						<text class="ratio-card__label">早餐 / 正餐</text>
						<view class="ratio-card__value-row">
							<text class="ratio-card__main ratio-card__main--caramel">{{ animatedInt('mealBreakfast') }}</text>
							<text class="ratio-card__slash">/</text>
							<text class="ratio-card__sub">{{ animatedInt('mealMain') }}</text>
						</view>
					</view>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script>
import { createCountUpController } from '../../utils/count-up'
import { formatRelativeUpdatedAt } from '../../utils/space-stats'
import { getKitchenStats } from '../../utils/space-stats-api'
import { takeSpaceStatsContext, setPendingSpaceStatsAction } from '../../utils/space-stats-bridge'

const TREND_BAR_MAX_HEIGHT = 40
const TREND_BAR_MIN_HEIGHT = 4
const STATS_WINDOW = 'all'
const TAB_ANIMATED_KEYS = {
	recipes: ['recipeWishlist', 'recipeDone', 'recipeBreakfast', 'recipeMain', 'imageCoveragePct', 'recipeParsedTotal'],
	places: ['placeWant', 'placeVisited', 'placeLocated', 'placeTotal', 'ratedPlaceTotal'],
	mealPlans: ['mealSubmittedDays', 'mealDraftDays', 'mealAverageDishCount', 'mealBreakfast', 'mealMain']
}

export default {
	name: 'SpaceStatsPage',
	data() {
		return {
			stats: {},
			kitchenId: 0,
			isRefreshing: false,
			syncErrorMessage: '',
			activeTab: 'overview',
			mountedTabs: {
				overview: true
			},
			animatedNumbers: {
				recipeWishlist: 0,
				recipeDone: 0,
				recipeBreakfast: 0,
				recipeMain: 0,
				imageCoveragePct: 0,
				recipeParsedTotal: 0,
				placeWant: 0,
				placeVisited: 0,
				placeLocated: 0,
				placeTotal: 0,
				ratedPlaceTotal: 0,
				mealSubmittedDays: 0,
				mealDraftDays: 0,
				mealAverageDishCount: 0,
				mealBreakfast: 0,
				mealMain: 0
			},
			countUpController: null,
			tabs: [
				{ value: 'overview', label: '总览' },
				{ value: 'recipes', label: '美食库' },
				{ value: 'places', label: '打卡库' },
				{ value: 'mealPlans', label: '菜单安排' }
			]
		}
	},
	watch: {
		activeTab: {
			immediate: true,
			handler(tab) {
				this.markTabMounted(tab)
				this.$nextTick(() => this.runCountUpForTab(tab))
			}
		},
		stats() {
			this.$nextTick(() => this.runCountUpForMountedTabs())
		}
	},
	onLoad() {
		const context = takeSpaceStatsContext() || {}
		this.stats = context.stats && typeof context.stats === 'object' ? context.stats : {}
		this.kitchenId = Number(context.kitchenId) || 0
		// 进入时静默补拉后端 stats（趋势 / 成员贡献 / 评分分布），失败保留快照。
		this.loadRemote({ silent: true })
	},
	beforeUnmount() {
		this.countUpController?.clear()
	},
	onUnload() {
		this.countUpController?.clear()
	},
	computed: {
		overview() {
			return this.stats?.overview || {}
		},
		recipeStats() {
			const recipes = this.stats?.recipes || {}
			return {
				byMealType: recipes.byMealType || { breakfast: 0, main: 0 },
				byStatus: recipes.byStatus || { wishlist: 0, done: 0 },
				imageCoveredTotal: recipes.imageCoveredTotal || 0,
				parsedTotal: recipes.parsedTotal || 0
			}
		},
		placeStats() {
			const places = this.stats?.places || {}
			return {
				byStatus: places.byStatus || { want: 0, visited: 0 },
				locatedTotal: places.locatedTotal || 0,
				poiMatchedTotal: places.poiMatchedTotal || 0,
				revisitRatingDistribution: places.revisitRatingDistribution || { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0 },
				topRecommendedItems: Array.isArray(places.topRecommendedItems) ? places.topRecommendedItems : [],
				topScenes: Array.isArray(places.topScenes) ? places.topScenes : []
			}
		},
		mealPlanStats() {
			const mealPlans = this.stats?.mealPlans || {}
			return {
				draftDays: mealPlans.draftDays || 0,
				submittedDays: mealPlans.submittedDays || 0,
				averageDishCount: mealPlans.averageDishCount || 0,
				nextPlan: mealPlans.nextPlan || null,
				itemsByMealType: mealPlans.itemsByMealType || { breakfast: 0, main: 0 }
			}
		},
		topRevisitPlaces() {
			return Array.isArray(this.overview.topRevisitPlaces) ? this.overview.topRevisitPlaces : []
		},
		isCacheSnapshot() {
			return this.stats?.source !== 'remote'
		},
		updatedLabel() {
			return formatRelativeUpdatedAt(this.stats?.updatedAt)
		},
		recentActivityText() {
			const activity = this.stats?.recentActivity || {}
			const parts = []
			if (activity.newRecipeTotal) parts.push(`新增 ${activity.newRecipeTotal} 道菜`)
			if (activity.newPlaceTotal) parts.push(`新增 ${activity.newPlaceTotal} 个打卡点`)
			if (activity.visitedPlaceTotal) parts.push(`打卡 ${activity.visitedPlaceTotal} 次`)
			if (activity.submittedMealPlanTotal) parts.push(`安排了 ${activity.submittedMealPlanTotal} 次菜单`)

			if (!parts.length) return '这个空间最近比较安静，还没有新增或打卡记录。'
			return `${parts.join('，')}。`
		},
		trendRows() {
			const trends = this.stats?.trends || {}
			const definitions = [
				{ key: 'recipeCreated', label: '新增菜谱', color: '#bf715f', series: trends.recipeCreated },
				{ key: 'placeVisited', label: '打卡', color: '#8a9a5b', series: trends.placeVisited },
				{ key: 'mealPlanSubmitted', label: '安排菜单', color: '#f4a236', series: trends.mealPlanSubmitted }
			]
			return definitions
				.map((definition) => {
					const series = Array.isArray(definition.series) ? definition.series : []
					const points = series.slice(-30)
					const max = points.reduce((peak, point) => Math.max(peak, Number(point.count) || 0), 0)
					const total = points.reduce((sum, point) => sum + (Number(point.count) || 0), 0)
					return {
						key: definition.key,
						label: definition.label,
						color: definition.color,
						total,
						points: points.map((point) => {
							const count = Number(point.count) || 0
							const height = max > 0 ? Math.max(TREND_BAR_MIN_HEIGHT, Math.round((count / max) * TREND_BAR_MAX_HEIGHT)) : TREND_BAR_MIN_HEIGHT
							return { height }
						})
					}
				})
				.filter((row) => row.total > 0)
		},
		hasTrends() {
			return this.trendRows.length > 0
		},
		contributors() {
			const members = this.stats?.members || {}
			return (Array.isArray(members.contributors) ? members.contributors : [])
				.slice()
				.sort((left, right) => (Number(right.total) || 0) - (Number(left.total) || 0))
		},
		showContributors() {
			return this.contributors.length > 1
		},
		revisitBars() {
			const distribution = this.placeStats.revisitRatingDistribution
			const max = Object.values(distribution).reduce((peak, count) => Math.max(peak, Number(count) || 0), 0)
			return [5, 4, 3, 2, 1].map((rating) => {
				const count = Number(distribution[rating]) || 0
				return {
					rating,
					count,
					pct: max > 0 ? Math.round((count / max) * 100) : 0
				}
			})
		},
		hasRevisitRatings() {
			return this.revisitBars.some((row) => row.count > 0)
		},
		ratedPlaceTotal() {
			return this.revisitBars.reduce((sum, row) => sum + (Number(row.count) || 0), 0)
		},
		recommendedTags() {
			return this.placeStats.topRecommendedItems
		},
		sceneTags() {
			return this.placeStats.topScenes
		},
		imageCoveragePct() {
			const total = Number(this.overview.recipeTotal) || 0
			if (total <= 0) return 0
			return Math.round(((this.recipeStats.imageCoveredTotal || 0) / total) * 100)
		},
		parsedPct() {
			const total = Number(this.overview.recipeTotal) || 0
			if (total <= 0) return 0
			return Math.min(100, Math.round(((this.recipeStats.parsedTotal || 0) / total) * 100))
		},
		nextPlanTitles() {
			const titles = this.mealPlanStats.nextPlan?.titles
			return Array.isArray(titles) && titles.length ? titles.join(' · ') : ''
		}
	},
	methods: {
		getCountUpController() {
			if (!this.countUpController) {
				this.countUpController = createCountUpController({
					read: (key) => this.animatedNumbers[key],
					write: (key, value) => {
						this.animatedNumbers[key] = value
					}
				})
			}
			return this.countUpController
		},
		handleRefresh() {
			this.loadRemote({ silent: false })
		},
		animatedInt(key) {
			return String(Math.round(Number(this.animatedNumbers[key]) || 0))
		},
		animatedFixed(key, decimals = 1) {
			const value = Number(this.animatedNumbers[key]) || 0
			return value.toFixed(decimals)
		},
		getAnimatedTargets() {
			return {
				recipeWishlist: this.recipeStats.byStatus.wishlist || 0,
				recipeDone: this.recipeStats.byStatus.done || 0,
				recipeBreakfast: this.recipeStats.byMealType.breakfast || 0,
				recipeMain: this.recipeStats.byMealType.main || 0,
				imageCoveragePct: this.imageCoveragePct,
				recipeParsedTotal: this.recipeStats.parsedTotal || 0,
				placeWant: this.placeStats.byStatus.want || 0,
				placeVisited: this.placeStats.byStatus.visited || 0,
				placeLocated: this.placeStats.locatedTotal || 0,
				placeTotal: this.overview.placeTotal || 0,
				ratedPlaceTotal: this.ratedPlaceTotal,
				mealSubmittedDays: this.mealPlanStats.submittedDays || 0,
				mealDraftDays: this.mealPlanStats.draftDays || 0,
				mealAverageDishCount: this.mealPlanStats.averageDishCount || 0,
				mealBreakfast: this.mealPlanStats.itemsByMealType.breakfast || 0,
				mealMain: this.mealPlanStats.itemsByMealType.main || 0
			}
		},
		runCountUpForMountedTabs() {
			Object.keys(TAB_ANIMATED_KEYS).forEach((tab) => {
				if (this.mountedTabs[tab]) {
					this.runCountUpForTab(tab)
				}
			})
		},
		runCountUpForTab(tab) {
			const keys = TAB_ANIMATED_KEYS[tab] || []
			if (!keys.length) return
			const targets = this.getAnimatedTargets()
			keys.forEach((key) => this.animateNumber(key, Number(targets[key]) || 0))
		},
		animateNumber(key, target) {
			this.getCountUpController().animate(key, target)
		},
		markTabMounted(tab) {
			if (!tab || this.mountedTabs[tab]) return
			this.mountedTabs[tab] = true
		},
		isTabMounted(tab) {
			return !!this.mountedTabs[tab]
		},
		async loadRemote(options = {}) {
			const { silent = true } = options
			if (this.isRefreshing) return
			if (!this.kitchenId) {
				this.syncErrorMessage = '当前空间信息未就绪'
				return
			}
			this.isRefreshing = true
			try {
				const remote = await getKitchenStats(this.kitchenId, { window: STATS_WINDOW })
				this.stats = remote
				this.syncErrorMessage = ''
			} catch (error) {
				this.syncErrorMessage = error?.message || '后端统计暂不可用'
				console.warn('[space-stats-page] remote stats unavailable', error)
				if (!silent) {
					uni.showToast({ title: '后端统计暂不可用，已显示本地聚合', icon: 'none' })
				}
			} finally {
				this.isRefreshing = false
			}
		},
		onAction(payload = {}) {
			// 页内动作交回首页执行（切换分区 / 打开详情），随后返回首页。
			setPendingSpaceStatsAction(payload)
			uni.navigateBack()
		},
		memberInitial(member = {}) {
			const name = member.nickname || `${member.userId || ''}`
			return String(name).slice(0, 1) || '厨'
		}
	}
}
</script>

<style lang="scss" scoped>
@import '../index/components/space-stats-sheet.scss';

/* 页面态覆写：由半屏改为整页展示 */
.space-stats-sheet--page {
	height: 100vh;
	max-height: none;
	border-radius: 0;
	padding-top: 20rpx;
}

.space-stats-sheet__meta--inline {
	flex: 1;
	min-width: 0;
	margin-top: 0;
}
</style>
