<template>
	<view
		class="recipe-card"
		:class="{
			'recipe-card--active': isActive,
			'recipe-card--pinned': card.isPinned,
			'recipe-card--meal-order-selected': isMealOrderSelected,
			'recipe-card--meal-order-mode': isLibraryMealOrderMode
		}"
		@tap="$emit('open', card.id)"
	>
		<view class="recipe-card__media" :class="{ 'recipe-card__media--empty': !coverSrc }">
			<image v-if="coverSrc" class="recipe-card__image" :src="coverSrc" mode="aspectFill" @error="$emit('image-error', card)"></image>
			<view v-else class="recipe-card__placeholder">
				<view class="recipe-card__placeholder-icon">
					<up-icon :name="card.placeholderIcon" size="26" color="#866d58"></up-icon>
				</view>
				<text class="recipe-card__placeholder-text">暂无图片</text>
			</view>
			<view v-if="isLibraryMealOrderMode && isMealOrderSelected" class="recipe-card__selected-badge">
				<up-icon name="checkmark" size="10" color="#fff9f1"></up-icon>
				<text class="recipe-card__selected-badge-text">已选</text>
			</view>
			<view v-if="card.sourceBadge && !isLibraryMealOrderMode" class="recipe-card__source-badge">
				<text class="recipe-card__source-badge-text">{{ card.sourceBadge }}</text>
			</view>
			<view v-if="card.imageCount > 1 && !isLibraryMealOrderMode" class="recipe-card__count">
				<text class="recipe-card__count-text">{{ card.imageCount }}</text>
			</view>
		</view>
		<view class="recipe-card__body">
			<view class="recipe-card__top">
				<view class="recipe-card__title-wrap">
					<view class="recipe-card__title-row">
						<view v-if="card.isPinned" class="recipe-card__pin-badge">
							<text class="recipe-card__pin-badge-text">置顶</text>
						</view>
						<text class="recipe-card__title">{{ card.title }}</text>
					</view>
				</view>
				<view
					v-if="!isLibraryMealOrderMode"
					class="recipe-switch"
					:class="'recipe-switch--' + card.status"
					@tap.stop="$emit('toggle-status', card.id)"
				>
					<view class="recipe-switch__track">
						<view class="recipe-switch__slot">
							<up-icon name="heart-fill" size="12" color="#b8aa9b"></up-icon>
						</view>
						<view class="recipe-switch__slot">
							<up-icon name="checkmark-circle-fill" size="12" color="#b8aa9b"></up-icon>
						</view>
					</view>
					<view class="recipe-switch__thumb">
						<up-icon :name="statusIcon" size="12" :color="statusIconColor"></up-icon>
					</view>
				</view>
				<view v-else class="meal-order-control" @tap.stop>
					<view
						class="meal-order-add"
						:class="{ 'meal-order-add--active': isMealOrderSelected }"
						@tap.stop="$emit('toggle-meal-order', card)"
					>
						<up-icon
							v-if="isMealOrderSelected"
							class="meal-order-add__icon"
							name="checkmark"
							size="12"
							color="#fffaf3"
						></up-icon>
						<text class="meal-order-add__text" :class="{ 'meal-order-add__text--active': isMealOrderSelected }">
							{{ isMealOrderSelected ? '已加入' : '加入菜单' }}
						</text>
					</view>
				</view>
			</view>
			<text v-if="!isLibraryMealOrderMode" class="recipe-card__info">{{ card.infoLine }}</text>
			<text v-if="!isLibraryMealOrderMode" class="recipe-card__summary">{{ card.listSummary }}</text>
		</view>
	</view>
</template>

<script>
export default {
	name: 'RecipeCardItem',
	props: {
		card: {
			type: Object,
			default: () => ({})
		},
		coverSrc: {
			type: String,
			default: ''
		},
		isActive: {
			type: Boolean,
			default: false
		},
		isLibraryMealOrderMode: {
			type: Boolean,
			default: false
		},
		isMealOrderSelected: {
			type: Boolean,
			default: false
		},
		statusIcon: {
			type: String,
			default: 'heart-fill'
		}
	},
	emits: ['image-error', 'open', 'toggle-meal-order', 'toggle-status'],
	computed: {
		statusIconColor() {
			return this.card?.status === 'done' ? '#6f826d' : '#9a7b65'
		}
	}
}
</script>

<style lang="scss" scoped>
@import './recipe-card-item.scss';
</style>
