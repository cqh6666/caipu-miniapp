<template>
	<view
		class="place-card"
		:class="{ 'place-card--visited': isVisited, 'place-card--no-location': !canOpenLocation }"
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

				<view class="place-card__location-row">
					<up-icon name="map-fill" size="13" color="#a08775"></up-icon>
					<text class="place-card__location-text">{{ addressText }}</text>
				</view>
			</view>

			<view class="place-card__footer">
				<view class="place-card__tags">
					<view v-for="(tag, i) in displayTags" :key="i" class="place-card__tag">
						<text class="place-card__tag-text">{{ tag }}</text>
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
			</view>
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
	gap: 24rpx;
	padding: 22rpx;
	border-radius: 34rpx;
	background:
		radial-gradient(circle at top left, rgba(255, 255, 255, 0.86) 0%, rgba(255, 255, 255, 0) 48%),
		linear-gradient(180deg, rgba(255, 253, 249, 0.99) 0%, rgba(255, 250, 244, 0.97) 100%);
	border: 1px solid rgba(91, 74, 59, 0.075);
	box-shadow:
		0 16rpx 32rpx rgba(70, 54, 40, 0.06),
		0 3rpx 8rpx rgba(70, 54, 40, 0.06),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
	min-height: 220rpx;
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
}

.place-card__cover-shell {
	position: relative;
	width: 176rpx;
	height: 176rpx;
	flex-shrink: 0;
	border-radius: 26rpx;
	overflow: hidden;
	align-self: center;
	background: #eadfD2;
	box-shadow: 0 8rpx 18rpx rgba(61, 44, 34, 0.08);
}

.place-card__cover {
	width: 100%;
	height: 100%;
	border-radius: 26rpx;

	&--visited {
		opacity: 0.8;
		filter: grayscale(22%) saturate(0.82);
	}
}

.place-card__cover-placeholder {
	width: 100%;
	height: 100%;
	border-radius: 26rpx;
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

.place-card__body {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	gap: 16rpx;
	padding: 8rpx 4rpx 8rpx 0;
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
	font-size: 36rpx;
	font-weight: 900;
	color: #5c4033;
	line-height: 1.24;
	margin-top: 12rpx;
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
	padding-right: 120rpx;
}

.place-card__location-row {
	display: flex;
	min-width: 0;
	align-items: flex-start;
	gap: 8rpx;
	margin-top: 12rpx;
}

.place-card__location-text {
	flex: 1;
	min-width: 0;
	font-size: 25rpx;
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
	justify-content: space-between;
	gap: 12rpx;
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
	width: 64rpx;
	height: 64rpx;
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
