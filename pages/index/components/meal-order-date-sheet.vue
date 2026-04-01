<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.22"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="meal-order-sheet">
			<view class="meal-order-sheet__header">
				<view class="meal-order-sheet__heading">
					<text class="meal-order-sheet__title">哪天一起吃</text>
					<text class="meal-order-sheet__subtitle">先挑个日子，再把想吃的菜慢慢放进这天的小菜单里。</text>
				</view>
				<view class="meal-order-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<view class="meal-order-date-grid">
				<view
					v-for="option in quickDateOptions"
					:key="option.value"
					class="meal-order-date-card"
					:class="{ 'meal-order-date-card--active': option.value === pickerValue }"
					@tap="$emit('start', option.value)"
				>
					<text class="meal-order-date-card__label">{{ option.label }}</text>
					<text class="meal-order-date-card__date">{{ option.dateText }}</text>
				</view>
			</view>

			<picker mode="date" :value="pickerValue" :start="dateStart" @change="handlePickerChange">
				<view class="meal-order-date-picker">
					<text class="meal-order-date-picker__text">自选日期</text>
					<up-icon name="calendar" size="16" color="#6f5f50"></up-icon>
				</view>
			</picker>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'MealOrderDateSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		quickDateOptions: {
			type: Array,
			default: () => []
		},
		pickerValue: {
			type: String,
			default: ''
		},
		dateStart: {
			type: String,
			default: ''
		}
	},
	emits: ['close', 'pick-date', 'start'],
	methods: {
		handlePickerChange(event) {
			this.$emit('pick-date', event?.detail?.value || '')
		}
	}
}
</script>

<style lang="scss" scoped>
@import './meal-order-sheet.scss';
</style>
