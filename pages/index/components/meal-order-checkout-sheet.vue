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
					<text class="meal-order-sheet__title">一起确认菜单</text>
					<text class="meal-order-sheet__subtitle">{{ dateText }} · 共 {{ dishCount }} 道</text>
				</view>
				<view class="meal-order-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<scroll-view class="meal-order-cart-list" scroll-y>
				<view v-if="items.length" class="meal-order-checkout-list">
					<view
						v-for="item in items"
						:key="`meal-order-checkout-${item.recipeId}`"
						class="meal-order-checkout-item meal-order-checkout-item--link"
						hover-class="meal-order-checkout-item--hover"
						@tap="$emit('open-recipe', item)"
					>
						<text class="meal-order-checkout-item__title">{{ item.title }}</text>
						<up-icon class="meal-order-checkout-item__chevron" name="arrow-right" size="14" color="#9d8c7a"></up-icon>
					</view>
				</view>
				<view v-else class="soft-empty meal-order-cart-empty">
					<text class="soft-empty__text">这天还没有安排菜，先回去挑一挑。</text>
				</view>
			</scroll-view>

			<view v-if="note" class="meal-order-checkout-note">
				<text class="meal-order-checkout-note__label">备注</text>
				<text class="meal-order-checkout-note__text">{{ note }}</text>
			</view>

			<view class="meal-order-sheet__footer">
				<view class="sheet-action" @tap="$emit('close')">
					<text class="sheet-action__text">返回修改</text>
				</view>
				<view
					class="sheet-action sheet-action--primary"
					:class="{ 'sheet-action--disabled': !canCheckout }"
					@tap="$emit('submit')"
				>
					<text class="sheet-action__text sheet-action__text--primary">
						{{ isSubmitting ? '安排中...' : '安排这天菜单' }}
					</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'MealOrderCheckoutSheet',
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
		items: {
			type: Array,
			default: () => []
		},
		note: {
			type: String,
			default: ''
		},
		canCheckout: {
			type: Boolean,
			default: false
		},
		isSubmitting: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'open-recipe', 'submit']
}
</script>

<style lang="scss" scoped>
@import './meal-order-sheet.scss';
</style>
