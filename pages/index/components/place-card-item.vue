<template>
	<view
		class="place-card"
		:class="{
			'place-card--visited': isVisited,
			'place-card--no-location': !canOpenLocation,
			'place-card--low-rating': isLowRating
		}"
		@tap="$emit('open', place.id)"
	>
		<view class="place-card__cover-shell">
			<image
				v-if="coverSrc"
				class="place-card__cover"
				:class="{ 'place-card__cover--visited': isVisited }"
				:src="coverSrc"
				mode="aspectFill"
			/>
			<view v-else class="place-card__cover-placeholder">
				<view class="place-card__cover-icon">
					<up-icon :name="place.type === 'food' ? 'coupon' : 'map'" size="24" color="#806956"></up-icon>
				</view>
			</view>
			<view class="place-card__source-badge">
				<text class="place-card__source-text">{{ sourceLabel }}</text>
			</view>
			<view v-if="isHighlyRecommended" class="place-card__star-badge">
				<up-icon name="star-fill" size="13" color="#f4a236"></up-icon>
			</view>
		</view>

		<view class="place-card__body">
			<view class="place-card__content">
				<view class="place-card__meta-top">
					<view class="place-card__type-chip">
						<up-icon :name="place.type === 'food' ? 'coupon' : 'map'" size="12" color="#8b7460"></up-icon>
						<text class="place-card__type-text">{{ priceDisplayText }}</text>
					</view>
				</view>

				<text class="place-card__name" :class="{ 'place-card__name--visited': isVisited }">{{ place.name }}</text>

				<!-- 重访意愿（仅去过） -->
				<view v-if="isVisited && hasRevisitRating" class="place-card__rating-row">
					<view class="place-card__stars">
						<up-icon
							v-for="star in 5"
							:key="`star-${star}`"
							:name="star <= place.revisitRating ? 'star-fill' : 'star'"
							size="14"
							:color="star <= place.revisitRating ? '#f4a236' : '#d4c4b8'"
						></up-icon>
					</view>
					<text class="place-card__rating-text">{{ revisitRatingText }}</text>
				</view>

				<!-- 推荐项（仅去过且有推荐） -->
				<view v-if="isVisited && displayRecommendedItems.length" class="place-card__recommended-row">
					<text class="place-card__recommended-label">🍴</text>
					<text class="place-card__recommended-text">{{ displayRecommendedItemsText }}</text>
				</view>

				<!-- 位置 -->
				<view class="place-card__location-row">
					<up-icon name="map-fill" size="13" color="#a08775"></up-icon>
					<text class="place-card__location-text">{{ addressText }}</text>
				</view>
			</view>

			<view v-if="displayTags.length" class="place-card__footer">
				<view class="place-card__tags">
					<view v-for="(tag, i) in displayTags" :key="i" class="place-card__tag">
						<text class="place-card__tag-text">{{ tag }}</text>
					</view>
				</view>
			</view>
		</view>

		<view
			class="place-card__action-btn"
			:class="{
				'place-card__action-btn--visited': isVisited,
				'place-card__action-btn--unavailable': !canOpenLocation
			}"
			@tap.stop="$emit('open-location', place.id)"
		>
			<up-icon
				:class="{ 'place-card__nav-icon': !isVisited }"
				:name="isVisited ? 'checkmark-circle-fill' : 'arrow-upward'"
				:size="isVisited ? 22 : 19"
				:color="actionIconColor"
			></up-icon>
		</view>

		<view v-if="isVisited" class="place-card__visited-stamp">
			<up-icon name="checkmark-circle-fill" size="15" color="#8a9a5b"></up-icon>
			<text class="place-card__visited-stamp-text">已打卡</text>
		</view>
	</view>
</template>

<script>
export default {
	name: 'PlaceCardItem',
	props: {
		place: {
			type: Object,
			required: true
		}
	},
	emits: ['open', 'open-location'],
	computed: {
		isVisited() {
			return this.place.status === 'visited'
		},
		coverSrc() {
			return (Array.isArray(this.place.imageUrls) ? this.place.imageUrls : [])[0] || this.place.image || ''
		},
		addressText() {
			return this.place.address || this.place.location || '还没有记录地址'
		},
		canOpenLocation() {
			return !!(Number(this.place.latitude) && Number(this.place.longitude))
		},
		typeLabel() {
			if (this.place.type === 'attraction') return '景点'
			if (this.place.type === 'other') return '其他'
			return '餐厅'
		},
		priceText() {
			return String(this.place.price || '').trim()
		},
		priceDisplayText() {
			if (this.priceText) return this.priceText
			if (this.place.type === 'attraction') return '免费'
			return this.typeLabel
		},
		sourceLabel() {
			if (this.place.source === 'manual') return '手动记录'
			if (this.place.source === 'dianping') return '大众点评'
			if (this.place.source === 'meituan') return '美团'
			return '其他'
		},
		actionIconColor() {
			if (!this.canOpenLocation) return 'rgba(92, 64, 51, 0.42)'
			return this.isVisited ? '#8a9a5b' : '#fffaf3'
		},
		displayTags() {
			const scenes = this.place.scenes || []
			const tags = this.place.tags || []
			const combined = [...scenes, ...tags]
			return combined.slice(0, 2)
		},
		hasRevisitRating() {
			return this.isVisited && Number(this.place.revisitRating) > 0
		},
		isHighlyRecommended() {
			return this.hasRevisitRating && Number(this.place.revisitRating) === 5
		},
		isLowRating() {
			return this.hasRevisitRating && Number(this.place.revisitRating) <= 2
		},
		revisitRatingText() {
			const rating = Number(this.place.revisitRating)
			if (rating === 5) return '非常推荐'
			if (rating === 4) return '值得再去'
			if (rating === 3) return '还可以'
			if (rating === 2) return '不太推荐'
			if (rating === 1) return '不推荐'
			return ''
		},
		displayRecommendedItems() {
			if (!this.isVisited) return []
			const items = this.place.recommendedItems || []
			return Array.isArray(items) ? items.slice(0, 2) : []
		},
		displayRecommendedItemsText() {
			return this.displayRecommendedItems.join('，')
		}
	}
}
</script>

<style lang="scss" scoped>
.place-card {
	position: relative;
	display: flex;
	align-items: stretch;
	gap: 22rpx;
	padding: 18rpx;
	border-radius: 30rpx;
	background:
		radial-gradient(circle at top left, rgba(255, 255, 255, 0.86) 0%, rgba(255, 255, 255, 0) 48%),
		linear-gradient(180deg, rgba(255, 253, 249, 0.99) 0%, rgba(255, 250, 244, 0.97) 100%);
	border: 1px solid rgba(91, 74, 59, 0.075);
	box-shadow:
		0 16rpx 32rpx rgba(70, 54, 40, 0.06),
		0 3rpx 8rpx rgba(70, 54, 40, 0.06),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
	min-height: 208rpx;
	margin-bottom: 0;
	overflow: hidden;
	transition: transform 0.18s cubic-bezier(0.2, 0.8, 0.2, 1);

	&:active {
		transform: scale(0.992);
	}

	&--visited {
		border-color: rgba(138, 154, 91, 0.22);
		background:
			radial-gradient(circle at top right, rgba(232, 238, 217, 0.42) 0%, rgba(232, 238, 217, 0) 44%),
			linear-gradient(180deg, rgba(255, 253, 249, 0.99) 0%, rgba(250, 248, 242, 0.98) 100%);
	}

	&--low-rating {
		opacity: 0.7;
	}
}

.place-card__cover-shell {
	position: relative;
	width: 204rpx;
	min-height: 172rpx;
	max-height: 212rpx;
	align-self: stretch;
	flex-shrink: 0;
	border-radius: 24rpx;
	overflow: hidden;
	background: #eadfD2;
	box-shadow: 0 8rpx 18rpx rgba(61, 44, 34, 0.08);
}

.place-card__cover {
	width: 100%;
	height: 100%;
	border-radius: 24rpx;

	&--visited {
		opacity: 0.8;
		filter: grayscale(22%) saturate(0.82);
	}
}

.place-card__cover-placeholder {
	width: 100%;
	height: 100%;
	border-radius: 24rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	background:
		radial-gradient(circle at 28% 18%, rgba(255, 255, 255, 0.72) 0%, rgba(255, 255, 255, 0) 42%),
		linear-gradient(145deg, #f0e5d8, #e3d3c2);
}

.place-card__cover-icon {
	width: 64rpx;
	height: 64rpx;
	border-radius: 20rpx;
	background: rgba(255, 250, 244, 0.58);
	display: flex;
	align-items: center;
	justify-content: center;
	box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
}

.place-card__source-badge {
	position: absolute;
	top: 10rpx;
	left: 10rpx;
	display: flex;
	align-items: center;
	max-width: calc(100% - 20rpx);
	min-height: 36rpx;
	padding: 0 14rpx;
	background: rgba(255, 250, 244, 0.72);
	backdrop-filter: blur(8px);
	border-radius: 13rpx;
	border: 1px solid rgba(255, 255, 255, 0.28);
	box-shadow: 0 4rpx 10rpx rgba(54, 42, 33, 0.08);
}

.place-card__source-text {
	font-size: 21rpx;
	font-weight: 900;
	line-height: 1;
	color: #5c4033;
	white-space: nowrap;
}

.place-card__star-badge {
	position: absolute;
	top: 10rpx;
	right: 10rpx;
	width: 36rpx;
	height: 36rpx;
	border-radius: 50%;
	background: rgba(244, 162, 54, 0.95);
	display: flex;
	align-items: center;
	justify-content: center;
	box-shadow: 0 4rpx 10rpx rgba(244, 162, 54, 0.3);
}

.place-card__body {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	justify-content: flex-start;
	gap: 14rpx;
	padding: 6rpx 82rpx 6rpx 0;
	overflow: hidden;
}

.place-card__content {
	min-width: 0;
}

.place-card__meta-top {
	display: flex;
	min-width: 0;
	justify-content: space-between;
	align-items: center;
	gap: 10rpx;
}

.place-card__type-chip {
	display: flex;
	align-items: center;
	gap: 6rpx;
	min-width: 0;
}

.place-card__type-text {
	font-size: 22rpx;
	color: rgba(92, 64, 51, 0.5);
	font-weight: 700;
	line-height: 1.2;
	white-space: nowrap;
}

.place-card__name {
	font-family: "Songti SC", "STSong", "SimSun", "DejaVu Serif", serif;
	font-size: 34rpx;
	font-weight: 900;
	color: #5c4033;
	line-height: 1.22;
	margin-top: 10rpx;
	display: -webkit-box;
	overflow: hidden;
	word-break: break-all;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;

	&--visited {
		color: rgba(92, 64, 51, 0.68);
	}
}

.place-card--visited .place-card__content {
	padding-right: 42rpx;
}

.place-card__rating-row {
	display: flex;
	align-items: center;
	gap: 10rpx;
	margin-top: 10rpx;
}

.place-card__stars {
	display: flex;
	align-items: center;
	gap: 4rpx;
}

.place-card__rating-text {
	font-size: 22rpx;
	font-weight: 700;
	color: #f4a236;
	line-height: 1.2;
}

.place-card__recommended-row {
	display: flex;
	align-items: flex-start;
	gap: 6rpx;
	margin-top: 10rpx;
}

.place-card__recommended-label {
	font-size: 20rpx;
	line-height: 1.4;
	flex-shrink: 0;
}

.place-card__recommended-text {
	flex: 1;
	min-width: 0;
	font-size: 22rpx;
	font-weight: 600;
	color: rgba(92, 64, 51, 0.72);
	line-height: 1.4;
	display: -webkit-box;
	overflow: hidden;
	word-break: break-all;
	-webkit-line-clamp: 1;
	-webkit-box-orient: vertical;
}

.place-card__location-row {
	display: flex;
	min-width: 0;
	align-items: flex-start;
	gap: 8rpx;
	margin-top: 10rpx;
}

.place-card__location-text {
	flex: 1;
	min-width: 0;
	font-size: 24rpx;
	color: rgba(92, 64, 51, 0.58);
	line-height: 1.35;
	display: -webkit-box;
	overflow: hidden;
	word-break: break-all;
	-webkit-line-clamp: 1;
	-webkit-box-orient: vertical;
}

.place-card__footer {
	display: flex;
	align-items: center;
	justify-content: flex-start;
	min-width: 0;
}

.place-card__tags {
	display: flex;
	gap: 8rpx;
	min-width: 0;
	overflow: hidden;
}

.place-card__tag {
	flex-shrink: 1;
	min-width: 0;
	min-height: 40rpx;
	padding: 0 16rpx;
	background: rgba(249, 247, 242, 0.95);
	border-radius: 12rpx;
	display: flex;
	align-items: center;
	justify-content: center;
}

.place-card__tag-text {
	font-size: 21rpx;
	font-weight: 800;
	color: rgba(92, 64, 51, 0.6);
	display: block;
	max-width: 116rpx;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.place-card__action-btn {
	position: absolute;
	right: 24rpx;
	bottom: 24rpx;
	z-index: 3;
	width: 68rpx;
	height: 68rpx;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	background: #5c4033;
	box-shadow: 0 10rpx 18rpx rgba(92, 64, 51, 0.22);
	transition: all 0.2s ease;

	&--visited {
		background: rgba(138, 154, 91, 0.12);
		box-shadow: none;
	}

	&--unavailable {
		background: rgba(92, 64, 51, 0.08);
		box-shadow: none;
	}
}

.place-card__nav-icon {
	transform: rotate(45deg);
}

.place-card__visited-stamp {
	position: absolute;
	top: 22rpx;
	right: 26rpx;
	z-index: 2;
	min-height: 50rpx;
	padding: 0 24rpx;
	border-radius: 14rpx;
	border: 4rpx solid #8a9a5b;
	background: rgba(255, 253, 249, 0.88);
	box-shadow: 0 6rpx 14rpx rgba(74, 84, 49, 0.1);
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 8rpx;
	transform: rotate(-13deg);
}

.place-card__visited-stamp-text {
	font-size: 25rpx;
	font-weight: 900;
	line-height: 1;
	color: #8a9a5b;
	letter-spacing: 1rpx;
	white-space: nowrap;
}
</style>
