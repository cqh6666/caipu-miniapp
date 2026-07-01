<template>
	<view v-if="hasKitchen" class="space-stats-card">
		<view class="space-stats-card__header">
			<view class="space-stats-card__heading">
				<text class="space-stats-card__title">空间概览</text>
				<text v-if="!isEmpty" class="space-stats-card__summary">
					这个空间已经沉淀了 {{ overview.recipeTotal || 0 }} 道菜和 {{ overview.placeTotal || 0 }} 个打卡点
				</text>
			</view>
			<view v-if="isSyncing" class="space-stats-card__sync-badge">
				<up-icon name="reload" size="14" color="#8a7d70" class="space-stats-card__sync-icon"></up-icon>
				<text class="space-stats-card__sync-text">同步中</text>
			</view>
		</view>

		<view v-if="isCacheSnapshot && !isSyncing" class="space-stats-card__cache-chip">
			<text class="space-stats-card__cache-text">本地缓存 · {{ updatedAtLabel || '刚刚' }}</text>
			<view
				class="space-stats-card__cache-action"
				hover-class="space-stats-card__cache-action--hover"
				hover-start-time="0"
				hover-stay-time="160"
				@tap.stop="$emit('refresh')"
			>
				<text class="space-stats-card__cache-action-text">重新同步</text>
			</view>
		</view>

		<library-empty-state
			v-if="isEmpty"
			class="space-stats-card__empty"
			kind="meal-empty"
			title="添加第一道想吃的菜，开启你的美食空间"
			primary-text="去添加"
			primary-icon="plus"
			@primary="$emit('action', { actionType: 'open-add-recipe' })"
		></library-empty-state>

		<template v-else>
			<view class="space-stats-card__grid">
				<view class="space-stats-card__cell">
					<view class="space-stats-card__cell-value-row">
						<text class="space-stats-card__cell-value">{{ animated.recipeTotal }}</text>
						<text class="space-stats-card__cell-unit">道</text>
					</view>
					<text class="space-stats-card__cell-label">菜品</text>
				</view>
				<view class="space-stats-card__cell">
					<view class="space-stats-card__cell-value-row">
						<text class="space-stats-card__cell-value">{{ animated.placeTotal }}</text>
						<text class="space-stats-card__cell-unit">家</text>
					</view>
					<text class="space-stats-card__cell-label">打卡点</text>
				</view>
				<view class="space-stats-card__cell">
					<view class="space-stats-card__cell-value-row">
						<text class="space-stats-card__cell-value">{{ animated.submittedMealPlanDays }}</text>
						<text class="space-stats-card__cell-unit">天</text>
					</view>
					<text class="space-stats-card__cell-label">已安排</text>
				</view>
				<view class="space-stats-card__cell">
					<view class="space-stats-card__cell-value-row">
						<text class="space-stats-card__cell-value">{{ animated.memberTotal }}</text>
						<text class="space-stats-card__cell-unit">人</text>
					</view>
					<text class="space-stats-card__cell-label">成员</text>
				</view>
			</view>

			<view class="space-stats-card__action-row">
				<view
					class="space-stats-card__action-chip"
					hover-class="space-stats-card__action-chip--hover"
					hover-start-time="0"
					hover-stay-time="160"
					@tap="$emit('action', { actionType: 'view-wishlist-recipes' })"
				>
					<up-icon name="heart-fill" size="15" color="#f4a236"></up-icon>
					<text class="space-stats-card__action-text">想吃</text>
					<text class="space-stats-card__action-count">{{ animated.wishlistRecipeTotal }}</text>
					<text class="space-stats-card__action-text">道</text>
				</view>
				<view
					class="space-stats-card__action-chip"
					hover-class="space-stats-card__action-chip--hover"
					hover-start-time="0"
					hover-stay-time="160"
					@tap="$emit('action', { actionType: 'view-want-places' })"
				>
					<up-icon name="heart" size="15" color="#f4a236"></up-icon>
					<text class="space-stats-card__action-text">想去</text>
					<text class="space-stats-card__action-count">{{ animated.wantPlaceTotal }}</text>
					<text class="space-stats-card__action-text">家</text>
				</view>
			</view>
		</template>

		<view
			class="space-stats-card__footer"
			hover-class="space-stats-card__footer--hover"
			hover-start-time="0"
			hover-stay-time="160"
			@tap="$emit('open-stats')"
		>
			<text class="space-stats-card__footer-text">查看洞察</text>
			<up-icon name="arrow-right" size="14" color="#bf715f"></up-icon>
		</view>
	</view>
</template>

<script>
import { formatRelativeUpdatedAt } from '../../../utils/space-stats'
import LibraryEmptyState from './library-empty-state.vue'

const ANIMATED_KEYS = ['recipeTotal', 'placeTotal', 'submittedMealPlanDays', 'memberTotal', 'wishlistRecipeTotal', 'wantPlaceTotal']
const COUNT_UP_DURATION_MS = 640
const COUNT_UP_STEP_MS = 40

export default {
	name: 'SpaceStatsCard',
	components: {
		LibraryEmptyState
	},
	props: {
		stats: {
			type: Object,
			default: () => ({})
		},
		hasKitchen: {
			type: Boolean,
			default: false
		},
		isSyncing: {
			type: Boolean,
			default: false
		},
		isCacheSnapshot: {
			type: Boolean,
			default: false
		}
	},
	emits: ['open-stats', 'action', 'refresh'],
	data() {
		return {
			animated: {
				recipeTotal: 0,
				placeTotal: 0,
				submittedMealPlanDays: 0,
				memberTotal: 0,
				wishlistRecipeTotal: 0,
				wantPlaceTotal: 0
			},
			countUpTimers: {}
		}
	},
	computed: {
		overview() {
			return this.stats?.overview || {}
		},
		isEmpty() {
			return !this.overview.recipeTotal && !this.overview.placeTotal && !this.overview.submittedMealPlanDays
		},
		updatedAtLabel() {
			return formatRelativeUpdatedAt(this.stats?.updatedAt)
		}
	},
	watch: {
		'stats.updatedAt': {
			immediate: true,
			handler() {
				this.runCountUp()
			}
		}
	},
	beforeUnmount() {
		Object.values(this.countUpTimers).forEach((timerId) => clearInterval(timerId))
	},
	methods: {
		runCountUp() {
			ANIMATED_KEYS.forEach((key) => this.animateNumber(key, Number(this.overview[key]) || 0))
		},
		animateNumber(key, target) {
			if (this.countUpTimers[key]) {
				clearInterval(this.countUpTimers[key])
				delete this.countUpTimers[key]
			}

			const start = Number(this.animated[key]) || 0
			if (start === target) {
				this.animated[key] = target
				return
			}

			const totalSteps = Math.max(1, Math.round(COUNT_UP_DURATION_MS / COUNT_UP_STEP_MS))
			let currentStep = 0

			this.countUpTimers[key] = setInterval(() => {
				currentStep += 1
				const progress = Math.min(1, currentStep / totalSteps)
				const eased = 1 - Math.pow(1 - progress, 3)
				this.animated[key] = Math.round(start + (target - start) * eased)

				if (progress >= 1) {
					clearInterval(this.countUpTimers[key])
					delete this.countUpTimers[key]
					this.animated[key] = target
				}
			}, COUNT_UP_STEP_MS)
		}
	}
}
</script>

<style lang="scss" scoped>
@import './space-stats-card.scss';
</style>
