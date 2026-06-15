<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.34"
		:closeOnClickOverlay="!isSubmitting"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="place-detail">
			<view class="place-detail__handle"></view>
			<view class="place-detail__cover" :class="{ 'place-detail__cover--empty': !cover }">
				<image v-if="cover" class="place-detail__cover-image" :src="cover" mode="aspectFill" @tap="$emit('preview-image', 0)"></image>
				<view v-else class="place-detail__cover-placeholder">
					<up-icon :name="typeIcon" size="28" color="#8f7b68"></up-icon>
				</view>
				<view class="place-detail__status" :class="{ 'place-detail__status--visited': isVisited }">
					<up-icon :name="isVisited ? 'checkmark-circle-fill' : 'heart-fill'" size="14" :color="isVisited ? '#5f7656' : '#bf715f'"></up-icon>
					<text class="place-detail__status-text">{{ isVisited ? '已打卡' : '想去' }}</text>
				</view>
			</view>

			<view class="place-detail__body">
				<view class="place-detail__header">
					<view class="place-detail__title-wrap">
						<text class="place-detail__name">{{ place.name || '未命名打卡点' }}</text>
						<text class="place-detail__meta">{{ typeLabel }} · {{ sourceLabel }}</text>
					</view>
					<view class="place-detail__close" @tap="$emit('close')">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<view class="place-detail__info">
					<view class="place-detail__line">
						<up-icon name="map-fill" size="14" color="#a08775"></up-icon>
						<text class="place-detail__line-text">{{ place.address || '还没有记录地址' }}</text>
					</view>
					<view v-if="place.price" class="place-detail__line">
						<up-icon name="coupon" size="14" color="#a08775"></up-icon>
						<text class="place-detail__line-text">{{ place.price }}</text>
					</view>
					<view v-if="place.note" class="place-detail__note">
						<text class="place-detail__note-text">{{ place.note }}</text>
					</view>
				</view>

				<view v-if="displayTags.length" class="place-detail__tags">
					<view v-for="tag in displayTags" :key="tag" class="place-detail__tag">
						<text class="place-detail__tag-text">{{ tag }}</text>
					</view>
				</view>

				<view class="place-detail__actions">
					<view
						class="place-detail__action"
						:class="{ 'place-detail__action--disabled': !canOpenLocation || isSubmitting }"
						@tap="$emit('open-location', place.id)"
					>
						<up-icon name="map" size="16" color="#745742"></up-icon>
						<text class="place-detail__action-text">地图</text>
					</view>
					<view
						class="place-detail__action"
						:class="{ 'place-detail__action--disabled': isSubmitting }"
						@tap="$emit('toggle-status', place.id)"
					>
						<up-icon :name="isVisited ? 'heart' : 'checkmark-circle'" size="16" color="#745742"></up-icon>
						<text class="place-detail__action-text">{{ isVisited ? '想去' : '去过' }}</text>
					</view>
					<view
						class="place-detail__action"
						:class="{ 'place-detail__action--disabled': isSubmitting }"
						@tap="$emit('edit', place.id)"
					>
						<up-icon name="edit-pen" size="16" color="#745742"></up-icon>
						<text class="place-detail__action-text">编辑</text>
					</view>
					<view
						class="place-detail__action place-detail__action--danger"
						:class="{ 'place-detail__action--disabled': isSubmitting }"
						@tap="$emit('delete', place.id)"
					>
						<up-icon name="trash" size="16" color="#a95549"></up-icon>
						<text class="place-detail__action-text">删除</text>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
const typeLabels = {
	food: '餐厅',
	attraction: '景点',
	other: '其他'
}

const sourceLabels = {
	manual: '手动记录',
	dianping: '大众点评',
	meituan: '美团',
	other: '其他来源'
}

export default {
	name: 'PlaceDetailSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		place: {
			type: Object,
			default: () => ({})
		},
		isSubmitting: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'delete', 'edit', 'open-location', 'preview-image', 'toggle-status'],
	computed: {
		isVisited() {
			return this.place?.status === 'visited'
		},
		cover() {
			return (Array.isArray(this.place?.imageUrls) ? this.place.imageUrls : [])[0] || ''
		},
		typeIcon() {
			return this.place?.type === 'food' ? 'coupon' : 'map'
		},
		typeLabel() {
			return typeLabels[this.place?.type] || '其他'
		},
		sourceLabel() {
			return sourceLabels[this.place?.source] || '手动记录'
		},
		displayTags() {
			return (Array.isArray(this.place?.tags) ? this.place.tags : []).slice(0, 6)
		},
		canOpenLocation() {
			return !!(Number(this.place?.latitude) && Number(this.place?.longitude))
		}
	}
}
</script>

<style lang="scss" scoped>
.place-detail {
	max-height: 84vh;
	background: linear-gradient(180deg, #fcf8f3 0%, #f7f2eb 100%);
	overflow: hidden;
	display: flex;
	flex-direction: column;
	box-shadow: 0 -12rpx 30rpx rgba(53, 39, 27, 0.12);
}

.place-detail__handle {
	width: 84rpx;
	height: 8rpx;
	margin: 18rpx auto 14rpx;
	border-radius: 999rpx;
	background: rgba(127, 109, 92, 0.28);
}

.place-detail__cover {
	position: relative;
	height: 320rpx;
	margin: 0 24rpx;
	border-radius: 30rpx;
	overflow: hidden;
	background: #eadfD2;
}

.place-detail__cover--empty {
	height: 180rpx;
}

.place-detail__cover-image,
.place-detail__cover-placeholder {
	width: 100%;
	height: 100%;
}

.place-detail__cover-placeholder {
	display: flex;
	align-items: center;
	justify-content: center;
	background: linear-gradient(145deg, #f0e6db, #e4d5c5);
}

.place-detail__status {
	position: absolute;
	top: 18rpx;
	left: 18rpx;
	display: flex;
	align-items: center;
	gap: 8rpx;
	padding: 8rpx 16rpx;
	border-radius: 999rpx;
	background: rgba(255, 255, 255, 0.86);
	backdrop-filter: blur(8px);
}

.place-detail__status--visited {
	background: rgba(243, 248, 242, 0.9);
}

.place-detail__status-text {
	font-size: 22rpx;
	font-weight: 800;
	color: #5c4033;
}

.place-detail__body {
	padding: 24rpx 24rpx calc(env(safe-area-inset-bottom) + 24rpx);
	overflow-y: auto;
}

.place-detail__header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 18rpx;
}

.place-detail__title-wrap {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	gap: 8rpx;
}

.place-detail__name {
	font-size: 38rpx;
	font-weight: 900;
	line-height: 1.25;
	color: #2f2923;
}

.place-detail__meta {
	font-size: 22rpx;
	font-weight: 600;
	color: #9b9186;
}

.place-detail__close {
	width: 72rpx;
	height: 72rpx;
	border-radius: 22rpx;
	background: rgba(255, 255, 255, 0.92);
	border: 1px solid rgba(118, 94, 72, 0.1);
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
}

.place-detail__info {
	margin-top: 22rpx;
	padding: 20rpx;
	border-radius: 26rpx;
	background: rgba(255, 255, 255, 0.92);
	border: 1px solid rgba(118, 94, 72, 0.08);
}

.place-detail__line {
	display: flex;
	align-items: flex-start;
	gap: 10rpx;
}

.place-detail__line + .place-detail__line {
	margin-top: 14rpx;
}

.place-detail__line-text {
	flex: 1;
	font-size: 25rpx;
	line-height: 1.5;
	color: #5c4033;
}

.place-detail__note {
	margin-top: 18rpx;
	padding-top: 18rpx;
	border-top: 1px solid rgba(111, 86, 64, 0.07);
}

.place-detail__note-text {
	font-size: 25rpx;
	line-height: 1.6;
	color: #6f6358;
}

.place-detail__tags {
	display: flex;
	flex-wrap: wrap;
	gap: 10rpx;
	margin-top: 18rpx;
}

.place-detail__tag {
	padding: 8rpx 16rpx;
	border-radius: 10rpx;
	background: #fffdfa;
	border: 1px solid rgba(118, 94, 72, 0.08);
}

.place-detail__tag-text {
	font-size: 21rpx;
	font-weight: 700;
	color: rgba(92, 64, 51, 0.64);
}

.place-detail__actions {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 12rpx;
	margin-top: 24rpx;
}

.place-detail__action {
	min-height: 88rpx;
	border-radius: 22rpx;
	background: rgba(255, 255, 255, 0.96);
	border: 1px solid rgba(100, 78, 58, 0.08);
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: 6rpx;
}

.place-detail__action--danger {
	background: rgba(255, 248, 246, 0.96);
	border-color: rgba(169, 85, 73, 0.12);
}

.place-detail__action--disabled {
	opacity: 0.45;
}

.place-detail__action-text {
	font-size: 21rpx;
	font-weight: 800;
	color: #745742;
}

.place-detail__action--danger .place-detail__action-text {
	color: #a95549;
}
</style>
