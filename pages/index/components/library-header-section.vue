<template>
	<view>
		<view v-if="isLibraryMealOrderMode" class="page-header page-header--meal-order">
			<view class="page-header__top">
				<view class="page-header__heading">
					<view class="page-header__title-row">
						<view class="page-header__title-mark page-header__title-mark--meal-order">
							<up-icon
								name="heart-fill"
								size="14"
								color="#bf715f"
							></up-icon>
						</view>
						<text class="page-header__title">{{ libraryHeaderTitle }}</text>
					</view>
				</view>
				<view class="meal-order-mode-bar__actions page-header__mode-actions">
					<view class="meal-order-mode-bar__chip meal-order-mode-bar__chip--accent" @tap="$emit('open-meal-order-date-sheet')">
						<up-icon name="calendar" size="14" color="#765948"></up-icon>
						<text class="meal-order-mode-bar__chip-text">改日期</text>
					</view>
					<view class="meal-order-mode-bar__chip meal-order-mode-bar__chip--ghost" @tap="$emit('exit-meal-order-mode')">
						<up-icon name="arrow-left" size="14" color="#948476"></up-icon>
						<text class="meal-order-mode-bar__chip-text">返回</text>
					</view>
				</view>
			</view>
		</view>

		<view
			v-if="!isLibraryMealOrderMode && hasMealOrderSpotlightRecord"
			class="meal-order-spotlight"
			:key="spotlightMotionKey"
			:class="[spotlightMotionClass, { 'meal-order-spotlight--tap-pulse': tapPulseActive }]"
			@tap="handleSpotlightTap"
			@touchstart.stop="$emit('spotlight-touchstart', $event)"
			@touchend.stop="$emit('spotlight-touchend', $event)"
		>
			<view class="meal-order-spotlight__date-card">
				<text v-if="mealOrderSpotlightLeadText" class="meal-order-spotlight__lead">{{ mealOrderSpotlightLeadText }}</text>
				<text class="meal-order-spotlight__date">{{ mealOrderSpotlightDateText }}</text>
				<text v-if="mealOrderSpotlightWeekday" class="meal-order-spotlight__weekday">{{ mealOrderSpotlightWeekday }}</text>
			</view>
			<view class="meal-order-spotlight__main">
				<view class="meal-order-spotlight__meta-row">
					<view
						v-if="mealOrderSpotlightStatusText"
						class="meal-order-spotlight__chip"
						:class="`meal-order-spotlight__chip--${mealOrderSpotlightStatusKind}`"
					>
						<text class="meal-order-spotlight__chip-text">{{ mealOrderSpotlightStatusText }}</text>
					</view>
					<text v-if="mealOrderSpotlightCountText" class="meal-order-spotlight__count">{{ mealOrderSpotlightCountText }}</text>
				</view>
				<text class="meal-order-spotlight__title">这天的小菜单</text>
				<text class="meal-order-spotlight__desc">{{ mealOrderSpotlightDesc }}</text>
			</view>
			<view class="meal-order-spotlight__aside">
				<view v-if="mealOrderSpotlightMetaText" class="meal-order-spotlight__progress">
					<text class="meal-order-spotlight__progress-text">{{ mealOrderSpotlightMetaText }}</text>
				</view>
				<view class="meal-order-spotlight__arrow">
					<up-icon name="arrow-right" size="15" color="#fff9f1"></up-icon>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
export default {
	name: 'LibraryHeaderSection',
	props: {
		isLibraryMealOrderMode: {
			type: Boolean,
			default: false
		},
		libraryHeaderTitle: {
			type: String,
			default: ''
		},
		libraryHeaderSummary: {
			type: String,
			default: ''
		},
		hasMealOrderSpotlightRecord: {
			type: Boolean,
			default: false
		},
		mealOrderSpotlightDateText: {
			type: String,
			default: ''
		},
		mealOrderSpotlightWeekday: {
			type: String,
			default: ''
		},
		mealOrderSpotlightLeadText: {
			type: String,
			default: ''
		},
		mealOrderSpotlightStatusText: {
			type: String,
			default: ''
		},
		mealOrderSpotlightStatusKind: {
			type: String,
			default: ''
		},
		mealOrderSpotlightDesc: {
			type: String,
			default: ''
		},
		mealOrderSpotlightCountText: {
			type: String,
			default: ''
		},
		mealOrderSpotlightMetaText: {
			type: String,
			default: ''
		},
		mealOrderSpotlightMotionDirection: {
			type: String,
			default: ''
		},
		mealOrderSpotlightMotionTick: {
			type: Number,
			default: 0
		}
	},
	computed: {
		spotlightMotionClass() {
			if (!this.mealOrderSpotlightMotionTick) return ''
			return this.mealOrderSpotlightMotionDirection === 'previous'
				? 'meal-order-spotlight--motion-previous'
				: 'meal-order-spotlight--motion-next'
		},
		spotlightMotionKey() {
			return [
				this.mealOrderSpotlightMotionDirection || 'idle',
				this.mealOrderSpotlightMotionTick,
				this.mealOrderSpotlightDateText,
				this.mealOrderSpotlightMetaText
			].join(':')
		}
	},
	data() {
		return {
			tapPulseActive: false,
			tapPulseTimer: null
		}
	},
	beforeUnmount() {
		if (this.tapPulseTimer) {
			clearTimeout(this.tapPulseTimer)
			this.tapPulseTimer = null
		}
	},
	methods: {
		handleSpotlightTap() {
			if (this.tapPulseActive) return
			this.tapPulseActive = true
			if (this.tapPulseTimer) clearTimeout(this.tapPulseTimer)
			this.tapPulseTimer = setTimeout(() => {
				this.$emit('spotlight-tap')
				this.tapPulseTimer = setTimeout(() => {
					this.tapPulseActive = false
					this.tapPulseTimer = null
				}, 60)
			}, 160)
		}
	},
	emits: [
		'exit-meal-order-mode',
		'open-meal-order-date-sheet',
		'spotlight-tap',
		'spotlight-touchend',
		'spotlight-touchstart'
	]
}
</script>

<style lang="scss" scoped>
@import './library-header-section.scss';
</style>
