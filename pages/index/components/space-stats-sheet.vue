<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="36"
		overlayOpacity="0.24"
		:closeOnClickOverlay="true"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="space-stats-sheet">
			<view class="space-stats-sheet__handle"></view>

			<view class="space-stats-sheet__header">
				<text class="space-stats-sheet__title">空间洞察</text>
				<view class="space-stats-sheet__header-actions">
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
					<view
						class="space-stats-sheet__close"
						hover-class="space-stats-sheet__close--hover"
						hover-start-time="0"
						hover-stay-time="160"
						@tap="$emit('close')"
					>
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>
			</view>

			<view class="space-stats-sheet__meta">
				<text class="space-stats-sheet__updated-text">更新时间：{{ updatedLabel || '暂无同步记录' }}</text>
				<view v-if="isCacheSnapshot" class="space-stats-sheet__source-chip">
					<up-icon name="clock" size="12" color="#8a7d70"></up-icon>
					<text class="space-stats-sheet__source-text">本地聚合</text>
				</view>
			</view>

			<!-- 窗口切换（V2：影响趋势与近期动态口径；本地聚合固定近 30 天） -->
			<view class="space-stats-sheet__window">
				<view
					v-for="option in windowOptions"
					:key="option.value"
					class="window-chip"
					:class="{ 'window-chip--active': activeWindow === option.value }"
					hover-class="window-chip--hover"
					hover-start-time="0"
					hover-stay-time="160"
					@tap="handleWindowChange(option.value)"
				>
					<text class="window-chip__text">{{ option.label }}</text>
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
				<block v-if="activeTab === 'overview'">
					<view class="stats-section">
						<text class="stats-section__title">资产总量</text>
						<view class="stats-section__grid">
							<view v-for="cell in assetCells" :key="cell.key" class="stats-section__cell">
								<text class="stats-section__cell-value">{{ cell.value }}</text>
								<text class="stats-section__cell-label">{{ cell.label }}</text>
							</view>
						</view>
					</view>

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
								@tap="$emit('action', { actionType: 'view-place-detail', placeId: place.id })"
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

					<view class="stats-section">
						<text class="stats-section__title">待行动</text>
						<view v-if="actions.length" class="action-list">
							<view
								v-for="item in actions"
								:key="item.key"
								class="action-item"
								hover-class="action-item--hover"
								hover-start-time="0"
								hover-stay-time="160"
								@tap="$emit('action', { actionType: item.actionType })"
							>
								<text class="action-item__label">{{ item.label }}</text>
								<up-icon name="arrow-right" size="14" color="#a08975"></up-icon>
							</view>
						</view>
						<view v-else class="stats-empty-hint">
							<text class="stats-empty-hint__text">暂时没有待行动的事项，空间状态很清爽。</text>
						</view>
					</view>

					<!-- 近期动态 + 迷你趋势（V2 有 trends 时展示 sparkline） -->
					<view class="stats-section">
						<text class="stats-section__title">近期动态（{{ windowLabel }}）</text>
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
					<view v-if="showContributors" class="stats-section stats-section--last">
						<text class="stats-section__title">成员贡献</text>
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
					<view v-else class="stats-section stats-section--last"></view>
				</block>

				<!-- ============ 美食库 ============ -->
				<block v-else-if="activeTab === 'recipes'">
					<view class="stats-section">
						<text class="stats-section__title">餐别结构</text>
						<view class="dual-bar">
							<view class="dual-bar__item">
								<text class="dual-bar__value">{{ recipeStats.byMealType.breakfast || 0 }}</text>
								<text class="dual-bar__label">早餐</text>
							</view>
							<view class="dual-bar__item">
								<text class="dual-bar__value">{{ recipeStats.byMealType.main || 0 }}</text>
								<text class="dual-bar__label">正餐</text>
							</view>
						</view>
					</view>
					<view class="stats-section">
						<text class="stats-section__title">想吃 / 吃过</text>
						<view class="split-bar">
							<view class="split-bar__segment split-bar__segment--want" :style="{ flexGrow: recipeStats.byStatus.wishlist || 0 }">
								<text class="split-bar__text">想吃 {{ recipeStats.byStatus.wishlist || 0 }}</text>
							</view>
							<view class="split-bar__segment split-bar__segment--done" :style="{ flexGrow: recipeStats.byStatus.done || 0 }">
								<text class="split-bar__text">吃过 {{ recipeStats.byStatus.done || 0 }}</text>
							</view>
						</view>
					</view>
					<view class="stats-section stats-section--last">
						<view class="health-toggle" @tap="showRecipeHealth = !showRecipeHealth">
							<text class="health-toggle__title">数据完整度</text>
							<up-icon :name="showRecipeHealth ? 'arrow-up' : 'arrow-down'" size="13" color="#a08975"></up-icon>
						</view>
						<view v-if="showRecipeHealth" class="health-body">
							<view
								v-if="recipeImageMissing > 0"
								class="health-hint"
								hover-class="health-hint--hover"
								hover-start-time="0"
								hover-stay-time="160"
								@tap="$emit('action', { actionType: 'view-done-recipes' })"
							>
								<text class="health-hint__text">还有 {{ recipeImageMissing }} 道菜没有图片，去补全</text>
								<up-icon name="arrow-right" size="13" color="#a08975"></up-icon>
							</view>
							<text v-else class="health-ok">菜谱图片已全部补齐。</text>
							<text class="health-note">已解析菜谱 {{ recipeStats.parsedTotal || 0 }} / {{ overview.recipeTotal || 0 }} 道</text>
						</view>
					</view>
				</block>

				<!-- ============ 打卡库 ============ -->
				<block v-else-if="activeTab === 'places'">
					<view class="stats-section">
						<text class="stats-section__title">想去 / 去过</text>
						<view class="split-bar">
							<view class="split-bar__segment split-bar__segment--want" :style="{ flexGrow: placeStats.byStatus.want || 0 }">
								<text class="split-bar__text">想去 {{ placeStats.byStatus.want || 0 }}</text>
							</view>
							<view class="split-bar__segment split-bar__segment--done" :style="{ flexGrow: placeStats.byStatus.visited || 0 }">
								<text class="split-bar__text">去过 {{ placeStats.byStatus.visited || 0 }}</text>
							</view>
						</view>
					</view>

					<view class="stats-section">
						<text class="stats-section__title">重访评分分布</text>
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

					<view v-if="recommendedTags.length" class="stats-section">
						<text class="stats-section__title">推荐项 Top</text>
						<view class="tag-cloud">
							<view v-for="tag in recommendedTags" :key="tag.label" class="tag-cloud__item">
								<text class="tag-cloud__text">{{ tag.label }}</text>
								<text class="tag-cloud__count">{{ tag.count }}</text>
							</view>
						</view>
					</view>

					<view v-if="sceneTags.length" class="stats-section">
						<text class="stats-section__title">场景标签 Top</text>
						<view class="tag-cloud">
							<view v-for="tag in sceneTags" :key="tag.label" class="tag-cloud__item tag-cloud__item--scene">
								<text class="tag-cloud__text">{{ tag.label }}</text>
								<text class="tag-cloud__count">{{ tag.count }}</text>
							</view>
						</view>
					</view>

					<view class="stats-section stats-section--last">
						<view class="health-toggle" @tap="showPlaceHealth = !showPlaceHealth">
							<text class="health-toggle__title">数据完整度</text>
							<up-icon :name="showPlaceHealth ? 'arrow-up' : 'arrow-down'" size="13" color="#a08975"></up-icon>
						</view>
						<view v-if="showPlaceHealth" class="health-body">
							<view
								v-if="placeLocationMissing > 0"
								class="health-hint"
								hover-class="health-hint--hover"
								hover-start-time="0"
								hover-stay-time="160"
								@tap="$emit('action', { actionType: 'view-missing-location-places' })"
							>
								<text class="health-hint__text">还有 {{ placeLocationMissing }} 个打卡点缺定位，去补全</text>
								<up-icon name="arrow-right" size="13" color="#a08975"></up-icon>
							</view>
							<text v-else class="health-ok">打卡点定位已全部补齐。</text>
							<text class="health-note">POI 已匹配 {{ placeStats.poiMatchedTotal || 0 }} / {{ overview.placeTotal || 0 }} 个</text>
						</view>
					</view>
				</block>

				<!-- ============ 菜单安排 ============ -->
				<block v-else>
					<view class="stats-section">
						<text class="stats-section__title">菜单概览</text>
						<view class="stats-section__grid">
							<view class="stats-section__cell">
								<text class="stats-section__cell-value">{{ mealPlanStats.submittedDays || 0 }}</text>
								<text class="stats-section__cell-label">已安排天数</text>
							</view>
							<view class="stats-section__cell">
								<text class="stats-section__cell-value">{{ mealPlanStats.draftDays || 0 }}</text>
								<text class="stats-section__cell-label">草稿天数</text>
							</view>
							<view class="stats-section__cell">
								<text class="stats-section__cell-value">{{ averageDishText }}</text>
								<text class="stats-section__cell-label">平均菜数</text>
							</view>
							<view class="stats-section__cell">
								<text class="stats-section__cell-value">{{ (mealPlanStats.itemsByMealType.breakfast || 0) + (mealPlanStats.itemsByMealType.main || 0) }}</text>
								<text class="stats-section__cell-label">累计菜品</text>
							</view>
						</view>
					</view>

					<view class="stats-section">
						<text class="stats-section__title">下一次安排</text>
						<view
							v-if="mealPlanStats.nextPlan"
							class="next-plan"
							hover-class="next-plan--hover"
							hover-start-time="0"
							hover-stay-time="160"
							@tap="$emit('action', { actionType: 'view-draft-meal-plan' })"
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

					<view class="stats-section stats-section--last">
						<text class="stats-section__title">菜品来源</text>
						<view class="dual-bar">
							<view class="dual-bar__item">
								<text class="dual-bar__value">{{ mealPlanStats.itemsByMealType.breakfast || 0 }}</text>
								<text class="dual-bar__label">早餐</text>
							</view>
							<view class="dual-bar__item">
								<text class="dual-bar__value">{{ mealPlanStats.itemsByMealType.main || 0 }}</text>
								<text class="dual-bar__label">正餐</text>
							</view>
						</view>
					</view>
				</block>
			</scroll-view>
		</view>
	</up-popup>
</template>

<script>
import { formatRelativeUpdatedAt } from '../../../utils/space-stats'

const WINDOW_LABELS = {
	'7d': '近 7 天',
	'30d': '近 30 天',
	'90d': '近 90 天',
	all: '全部'
}
const TREND_BAR_MAX_HEIGHT = 40
const TREND_BAR_MIN_HEIGHT = 4

export default {
	name: 'SpaceStatsSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		stats: {
			type: Object,
			default: () => ({})
		},
		isRefreshing: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'refresh', 'action', 'change-window'],
	data() {
		return {
			activeTab: 'overview',
			showRecipeHealth: false,
			showPlaceHealth: false,
			windowOptions: [
				{ value: '7d', label: '7天' },
				{ value: '30d', label: '30天' },
				{ value: '90d', label: '90天' },
				{ value: 'all', label: '全部' }
			],
			tabs: [
				{ value: 'overview', label: '总览' },
				{ value: 'recipes', label: '美食库' },
				{ value: 'places', label: '打卡库' },
				{ value: 'mealPlans', label: '菜单安排' }
			]
		}
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
		assetCells() {
			return [
				{ key: 'recipe', label: '菜品', value: this.overview.recipeTotal || 0 },
				{ key: 'place', label: '打卡点', value: this.overview.placeTotal || 0 },
				{ key: 'meal-plan', label: '菜单', value: this.overview.submittedMealPlanDays || 0 },
				{ key: 'member', label: '成员', value: this.overview.memberTotal || 0 }
			]
		},
		topRevisitPlaces() {
			return Array.isArray(this.overview.topRevisitPlaces) ? this.overview.topRevisitPlaces : []
		},
		actions() {
			return Array.isArray(this.stats?.actions) ? this.stats.actions : []
		},
		isCacheSnapshot() {
			return this.stats?.source !== 'remote'
		},
		activeWindow() {
			return this.stats?.window || '30d'
		},
		windowLabel() {
			return WINDOW_LABELS[this.activeWindow] || '近 30 天'
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
		recommendedTags() {
			return this.placeStats.topRecommendedItems
		},
		sceneTags() {
			return this.placeStats.topScenes
		},
		recipeImageMissing() {
			return Math.max(0, (this.overview.recipeTotal || 0) - (this.recipeStats.imageCoveredTotal || 0))
		},
		placeLocationMissing() {
			return Math.max(0, (this.overview.placeTotal || 0) - (this.placeStats.locatedTotal || 0))
		},
		averageDishText() {
			const value = Number(this.mealPlanStats.averageDishCount) || 0
			return value ? value.toFixed(1) : '0'
		},
		nextPlanTitles() {
			const titles = this.mealPlanStats.nextPlan?.titles
			return Array.isArray(titles) && titles.length ? titles.join(' · ') : ''
		}
	},
	methods: {
		handleRefresh() {
			if (this.isRefreshing) return
			this.$emit('refresh')
		},
		handleWindowChange(window) {
			if (window === this.activeWindow) return
			this.$emit('change-window', window)
		},
		memberInitial(member = {}) {
			const name = member.nickname || `${member.userId || ''}`
			return String(name).slice(0, 1) || '厨'
		}
	}
}
</script>

<style lang="scss" scoped>
@import './space-stats-sheet.scss';
</style>
