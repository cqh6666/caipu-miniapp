<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.22"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view
			v-if="record"
			class="meal-order-sheet"
			@touchstart="$emit('touchstart-sheet', $event)"
			@touchend="$emit('touchend-sheet', $event)"
		>
			<view class="meal-order-sheet__header">
				<view class="meal-order-sheet__heading">
					<text class="meal-order-sheet__eyebrow">{{ eyebrow }}</text>
					<text class="meal-order-sheet__title">{{ title }}</text>
					<text class="meal-order-sheet__subtitle">{{ subtitle }}</text>
				</view>
				<view class="meal-order-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<scroll-view class="meal-order-cart-list" scroll-y>
				<view class="meal-order-checkout-list">
					<view
						v-for="item in items"
						:key="`meal-order-spotlight-${item.recipeId}`"
						class="meal-order-checkout-item meal-order-checkout-item--link"
						hover-class="meal-order-checkout-item--hover"
						@tap="$emit('open-recipe', item)"
					>
						<text class="meal-order-checkout-item__title">{{ item.title }}</text>
						<up-icon class="meal-order-checkout-item__chevron" name="arrow-right" size="14" color="#9d8c7a"></up-icon>
					</view>
				</view>
			</scroll-view>

			<view v-if="note" class="meal-order-checkout-note">
				<text class="meal-order-checkout-note__label">备注</text>
				<text class="meal-order-checkout-note__text">{{ note }}</text>
			</view>

			<view class="meal-order-sheet__footer">
				<view class="sheet-action" @tap="$emit('close')">
					<text class="sheet-action__text">关闭</text>
				</view>
				<view
					v-if="canResume"
					class="sheet-action sheet-action--primary"
					@tap="$emit('resume')"
				>
					<text class="sheet-action__text sheet-action__text--primary">继续安排</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'MealOrderSpotlightSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		record: {
			type: Object,
			default: null
		},
		eyebrow: {
			type: String,
			default: ''
		},
		title: {
			type: String,
			default: ''
		},
		subtitle: {
			type: String,
			default: ''
		},
		items: {
			type: Array,
			default: () => []
		},
		note: {
			type: String,
			default: ''
		},
		canResume: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'resume', 'open-recipe', 'touchstart-sheet', 'touchend-sheet']
}
</script>

<style lang="scss" scoped>
@import './meal-order-sheet.scss';
</style>
