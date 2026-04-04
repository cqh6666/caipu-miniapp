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
					<text class="meal-order-sheet__title">这天的小菜单</text>
					<text class="meal-order-sheet__subtitle">{{ dateText }} · 已选 {{ dishCount }} 道</text>
				</view>
				<view class="meal-order-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<scroll-view class="meal-order-cart-list" scroll-y>
				<view v-if="items.length" class="meal-order-cart-stack">
					<view
						v-for="item in items"
						:key="`meal-order-cart-${item.recipeId}`"
						class="meal-order-cart-item meal-order-cart-item--link"
						hover-class="meal-order-cart-item--hover"
						@tap="$emit('open-recipe', item)"
					>
						<view class="meal-order-card-thumb">
							<image
								v-if="item.imageSnapshot"
								class="meal-order-card-thumb__image"
								:src="item.imageSnapshot"
								mode="aspectFill"
							></image>
							<view v-else class="meal-order-card-thumb__placeholder">
								<up-icon name="photo" size="18" color="#b69d86"></up-icon>
							</view>
						</view>
						<view class="meal-order-cart-item__main">
							<text class="meal-order-cart-item__title">{{ item.title }}</text>
							<view class="meal-order-card-meta">
								<text v-if="item.mealTypeLabel" class="meal-order-card-meta__text">{{ item.mealTypeLabel }}</text>
								<text v-if="item.quantity > 1" class="meal-order-card-meta__text">x{{ item.quantity }}</text>
							</view>
						</view>
						<view class="meal-order-cart-item__action" @tap.stop="$emit('remove-recipe', item.recipeId)">
							<text class="meal-order-cart-item__action-text">移出</text>
						</view>
					</view>
				</view>
				<view v-else class="soft-empty meal-order-cart-empty">
					<text class="soft-empty__text">还没选菜，先去美食库慢慢挑两道喜欢的吧。</text>
				</view>
			</scroll-view>

			<view class="meal-order-note">
				<text class="meal-order-note__label">想说的话</text>
				<textarea
					:value="note"
					class="meal-order-note__input"
					placeholder="比如：周六想吃热乎一点，提前把牛肉腌上"
					placeholder-class="meal-order-note__placeholder"
					maxlength="120"
					@input="$emit('note-input', $event)"
				/>
			</view>

			<view class="meal-order-sheet__footer">
				<view class="sheet-action" @tap="$emit('clear')">
					<text class="sheet-action__text">清空</text>
				</view>
				<view
					class="sheet-action sheet-action--primary"
					:class="{ 'sheet-action--disabled': !canCheckout }"
					@tap="$emit('confirm')"
				>
					<text class="sheet-action__text sheet-action__text--primary">确认菜单</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'MealOrderCartSheet',
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
		}
	},
	emits: ['close', 'open-recipe', 'remove-recipe', 'note-input', 'clear', 'confirm']
}
</script>

<style lang="scss" scoped>
@import './meal-order-sheet.scss';
</style>
