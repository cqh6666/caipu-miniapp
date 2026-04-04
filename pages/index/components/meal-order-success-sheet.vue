<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.22"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="meal-order-sheet meal-order-sheet--success">
			<view class="meal-order-sheet__header">
				<view class="meal-order-sheet__heading">
					<text class="meal-order-sheet__title">这天吃什么已经安排好</text>
					<text class="meal-order-sheet__subtitle">{{ dateText }} · 共 {{ dishCount }} 道</text>
				</view>
				<view class="meal-order-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<view class="meal-order-success-hero">
				<view class="meal-order-success-hero__icon">
					<up-icon name="checkmark" size="22" color="#ffffff"></up-icon>
				</view>
				<text class="meal-order-success-hero__title">安排已同步到这间厨房</text>
				<text class="meal-order-success-hero__desc">后面如果想改，我们会先带出草稿，不会直接覆盖原安排。</text>
			</view>

			<view class="meal-order-success-card">
				<text class="meal-order-success-card__label">菜单速览</text>
				<text class="meal-order-success-card__summary">{{ dishSummary }}</text>
				<text v-if="note" class="meal-order-success-card__note">备注：{{ note }}</text>
			</view>

			<view class="meal-order-sheet__footer">
				<view class="sheet-action" @tap="$emit('plan-next')">
					<text class="sheet-action__text">继续安排别天</text>
				</view>
				<view class="sheet-action sheet-action--primary" @tap="$emit('view-record')">
					<text class="sheet-action__text sheet-action__text--primary">查看当天菜单</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'MealOrderSuccessSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		dateText: {
			type: String,
			default: ''
		},
		dishCount: {
			type: Number,
			default: 0
		},
		dishSummary: {
			type: String,
			default: ''
		},
		note: {
			type: String,
			default: ''
		}
	},
	emits: ['close', 'view-record', 'plan-next']
}
</script>

<style lang="scss" scoped>
@import './meal-order-sheet.scss';
</style>
