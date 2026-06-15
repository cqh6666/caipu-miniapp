<template>
	<view
		class="place-card"
		:class="{ 'place-card--visited': isVisited }"
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
				<up-icon :name="place.type === 'food' ? 'coupon' : 'map'" size="26" color="#866d58"></up-icon>
			</view>
			<view class="place-card__source-badge">
				<text class="place-card__source-text">{{ sourceLabel }}</text>
			</view>
		</view>

		<view class="place-card__body">
			<view class="place-card__meta-top">
				<view class="place-card__type-price">
					<up-icon :name="place.type === 'food' ? 'coupon' : 'map' " size="12" color="#a08775"></up-icon>
					<text class="place-card__price-text">{{ place.price }}</text>
				</view>
			</view>

			<text class="place-card__name" :class="{ 'place-card__name--visited': isVisited }">{{ place.name }}</text>

			<view class="place-card__location-row">
				<up-icon name="map-fill" size="13" color="#a08775"></up-icon>
				<text class="place-card__location-text">{{ addressText }}</text>
			</view>

			<view class="place-card__footer">
				<view class="place-card__tags">
					<view v-for="(tag, i) in displayTags" :key="i" class="place-card__tag">
						<text class="place-card__tag-text">{{ tag }}</text>
					</view>
				</view>

				<view
					class="place-card__action-btn"
					:class="{ 'place-card__action-btn--visited': isVisited }"
					@tap.stop="$emit('open-location', place.id)"
				>
					<up-icon
						:name="isVisited ? 'checkmark-circle-fill' : 'nav-fill'"
						size="18"
						:color="isVisited ? '#8a9a5b' : '#ffffff'"
					></up-icon>
				</view>
			</view>
		</view>

		<!-- Visited Stamp -->
		<view v-if="isVisited" class="place-card__stamp">
			<up-icon name="checkmark-circle-fill" size="14" color="#8a9a5b"></up-icon>
			<text class="place-card__stamp-text">已打卡</text>
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
		sourceLabel() {
			if (this.place.source === 'manual') return '手动记录'
			if (this.place.source === 'dianping') return '大众点评'
			if (this.place.source === 'meituan') return '美团'
			return '其他'
		},
		displayTags() {
			return (this.place.tags || []).slice(0, 2)
		}
	}
}
</script>

<style lang="scss" scoped>
.place-card {
	position: relative;
	display: flex;
	align-items: stretch;
	gap: 18rpx;
	padding: 16rpx;
	border-radius: 26rpx;
	background: rgba(255, 253, 249, 0.96);
	border: 1px solid rgba(100, 78, 58, 0.05);
	box-shadow:
		0 12rpx 24rpx rgba(70, 54, 40, 0.045),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.6);
	margin-bottom: 20rpx;
	overflow: hidden;
	transition: transform 0.18s cubic-bezier(0.2, 0.8, 0.2, 1);

	&:active {
		transform: scale(0.992);
	}

	&--visited {
		border-color: rgba(138, 154, 91, 0.15);
	}
}

.place-card__cover-shell {
	position: relative;
	width: 180rpx;
	height: 180rpx;
	flex-shrink: 0;
	border-radius: 20rpx;
	overflow: hidden;
}

.place-card__cover {
	width: 100%;
	height: 100%;
	border-radius: 20rpx;

	&--visited {
		opacity: 0.8;
		filter: grayscale(30%);
	}
}

.place-card__cover-placeholder {
	width: 100%;
	height: 100%;
	border-radius: 20rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	background: linear-gradient(145deg, #f0e6db, #e4d5c5);
}

.place-card__source-badge {
	position: absolute;
	top: 8rpx;
	left: 8rpx;
	padding: 4rpx 12rpx;
	background: rgba(255, 255, 255, 0.85);
	backdrop-filter: blur(8px);
	border-radius: 10rpx;
	border: 1px solid rgba(255, 255, 255, 0.3);
}

.place-card__source-text {
	font-size: 18rpx;
	font-weight: 700;
	color: #5c4033;
}

.place-card__body {
	flex: 1;
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	padding: 6rpx 8rpx 6rpx 0;
}

.place-card__meta-top {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
}

.place-card__type-price {
	display: flex;
	align-items: center;
	gap: 6rpx;
}

.place-card__price-text {
	font-size: 20rpx;
	color: rgba(92, 64, 51, 0.5);
	font-weight: 500;
	letter-spacing: 0.5rpx;
}

.place-card__name {
	font-size: 30rpx;
	font-weight: 700;
	color: #5c4033;
	line-height: 1.3;
	margin-top: 4rpx;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;

	&--visited {
		color: rgba(92, 64, 51, 0.6);
	}
}

.place-card__location-row {
	display: flex;
	align-items: flex-start;
	gap: 6rpx;
	margin-top: 10rpx;
}

.place-card__location-text {
	font-size: 22rpx;
	color: rgba(92, 64, 51, 0.6);
	line-height: 1.5;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	overflow: hidden;
}

.place-card__footer {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-top: 12rpx;
}

.place-card__tags {
	display: flex;
	gap: 10rpx;
	overflow: hidden;
}

.place-card__tag {
	padding: 6rpx 14rpx;
	background: #f9f7f2;
	border-radius: 8rpx;
}

.place-card__tag-text {
	font-size: 18rpx;
	font-weight: 700;
	color: rgba(92, 64, 51, 0.6);
	white-space: nowrap;
}

.place-card__action-btn {
	width: 56rpx;
	height: 56rpx;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	background: #5c4033;
	box-shadow: 0 6rpx 12rpx rgba(92, 64, 51, 0.2);
	transition: all 0.2s ease;

	&--visited {
		background: rgba(138, 154, 91, 0.1);
		box-shadow: none;
	}
}

.place-card__stamp {
	position: absolute;
	top: 24rpx;
	right: 24rpx;
	display: flex;
	align-items: center;
	gap: 6rpx;
	padding: 6rpx 16rpx;
	border: 2px solid #8a9a5b;
	border-radius: 12rpx;
	background: rgba(255, 255, 255, 0.85);
	backdrop-filter: blur(4px);
	transform: rotate(-12deg);
	box-shadow: 0 4rpx 8rpx rgba(0, 0, 0, 0.05);
}

.place-card__stamp-text {
	font-size: 20rpx;
	font-weight: 900;
	color: #8a9a5b;
	letter-spacing: 2rpx;
}
</style>
