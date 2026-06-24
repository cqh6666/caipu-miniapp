<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.34"
		:closeOnClickOverlay="true"
		:safeAreaInsetBottom="false"
		@close="handleClose"
	>
		<view class="sheet">
			<view class="sheet__handle"></view>
			<view class="sheet__header">
				<text class="sheet__title">找到 {{ candidates.length }} 个可能的地点</text>
				<text class="sheet__cancel" @tap="handleClose">取消</text>
			</view>

			<scroll-view class="sheet__body" scroll-y>
				<!-- 候选卡片列表 -->
				<view class="card-list">
					<view
						v-for="candidate in candidates"
						:key="candidate.candidateId"
						class="card"
					>
						<view class="card__top">
							<!-- 左侧图片 -->
							<image
								v-if="candidate.imageUrls && candidate.imageUrls.length > 0"
								class="card__thumb"
								:src="candidate.imageUrls[0]"
								mode="aspectFill"
							></image>
							<view v-else class="card__thumb card__thumb--empty">
								<up-icon name="photo" size="32" color="#b5a89a"></up-icon>
							</view>

							<!-- 右侧信息 -->
							<view class="card__info">
								<text class="card__name">{{ candidate.name }}</text>
								<text class="card__address">{{ candidate.address }}</text>

								<!-- 评分与人均 -->
								<view class="card__meta">
									<view v-if="candidate.rating" class="rating">
										<up-icon name="star-fill" size="14" color="#ff6b35"></up-icon>
										<text class="rating__text">{{ candidate.rating }}</text>
									</view>
									<text v-if="candidate.price" class="price">{{ candidate.price }}</text>
								</view>

								<!-- 匹配理由 -->
								<view v-if="candidate.matchReasons && candidate.matchReasons.length" class="tags">
									<view
										v-for="(reason, idx) in candidate.matchReasons"
										:key="`reason-${idx}`"
										class="tag"
									>
										<up-icon name="checkmark-circle" size="12" color="#7c9070"></up-icon>
										<text class="tag__text">{{ reason }}</text>
									</view>
								</view>
							</view>
						</view>

						<!-- 操作按钮 -->
						<view class="card__btn" @tap="handleSelectCandidate(candidate)">
							<text class="card__btn-text">使用这个地点</text>
						</view>
					</view>
				</view>

				<!-- 底部手动填写 -->
				<view class="footer-tip" @tap="handleManualEntry">
					<text class="footer-tip__text">没有合适的，手动填写</text>
				</view>
			</scroll-view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'PlaceCandidateSheetV2',
	props: {
		show: { type: Boolean, default: false },
		candidates: { type: Array, default: () => [] },
		extracted: { type: Object, default: () => ({}) },
		source: { type: String, default: 'meituan' }
	},
	emits: ['close', 'select-candidate', 'manual-entry'],
	methods: {
		handleClose() {
			this.$emit('close')
		},
		handleSelectCandidate(candidate) {
			this.$emit('select-candidate', candidate)
		},
		handleManualEntry() {
			this.$emit('manual-entry')
		}
	}
}
</script>

<style lang="scss" scoped>
.sheet {
	background: #f7f5f1;
	border-radius: 32rpx 32rpx 0 0;
	max-height: 85vh;
	display: flex;
	flex-direction: column;
}

.sheet__handle {
	width: 56rpx;
	height: 8rpx;
	background: rgba(138, 125, 112, 0.24);
	border-radius: 4rpx;
	margin: 16rpx auto 0;
}

.sheet__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 20rpx 24rpx;
}

.sheet__title {
	font-size: 32rpx;
	font-weight: 700;
	color: #41362d;
	font-family: 'Playfair Display', Georgia, 'Times New Roman', serif;
}

.sheet__cancel {
	font-size: 28rpx;
	color: #8a7d70;
	padding: 8rpx 16rpx;
}

.sheet__body {
	flex: 1;
	padding: 0 24rpx 32rpx;
	box-sizing: border-box;
}

.card-list {
	display: flex;
	flex-direction: column;
	gap: 16rpx;
	margin-bottom: 16rpx;
}

.card {
	background: #fff;
	border-radius: 24rpx;
	overflow: hidden;
	box-shadow: 0 2rpx 12rpx rgba(65, 54, 45, 0.06);
}

.card__top {
	display: flex;
	padding: 24rpx;
	gap: 20rpx;
}

.card__thumb {
	flex-shrink: 0;
	width: 140rpx;
	height: 140rpx;
	border-radius: 20rpx;
	background: #f4ece2;

	&--empty {
		display: flex;
		align-items: center;
		justify-content: center;
	}
}

.card__info {
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 12rpx;
	min-width: 0;
}

.card__name {
	font-size: 30rpx;
	font-weight: 700;
	color: #41362d;
	line-height: 1.4;
	word-wrap: break-word;
}

.card__address {
	font-size: 24rpx;
	color: #8a7d70;
	line-height: 1.5;
	overflow: hidden;
	text-overflow: ellipsis;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	word-wrap: break-word;
}

.card__meta {
	display: flex;
	align-items: center;
	gap: 16rpx;
	flex-wrap: wrap;
	margin-top: 4rpx;
}

.rating {
	display: flex;
	align-items: center;
	gap: 4rpx;
}

.rating__text {
	font-size: 26rpx;
	font-weight: 600;
	color: #41362d;
}

.price {
	font-size: 26rpx;
	font-weight: 600;
	color: #41362d;
}

.tags {
	display: flex;
	flex-wrap: wrap;
	gap: 10rpx;
	margin-top: 4rpx;
}

.tag {
	display: flex;
	align-items: center;
	gap: 6rpx;
	padding: 8rpx 12rpx;
	background: rgba(124, 144, 112, 0.08);
	border-radius: 12rpx;
}

.tag__text {
	font-size: 22rpx;
	color: #7c9070;
	line-height: 1;
}

.card__btn {
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 20rpx 32rpx;

	&:active {
		opacity: 0.8;
	}
}

.card__btn-text {
	display: block;
	width: 100%;
	padding: 24rpx;
	background: #745742;
	border-radius: 48rpx;
	font-size: 28rpx;
	font-weight: 700;
	color: #fff;
	text-align: center;
	transition: all 0.2s ease;

	&:active {
		background: #5c4033;
		transform: scale(0.98);
	}
}

.footer-tip {
	display: flex;
	justify-content: center;
	padding: 24rpx;

	&:active {
		opacity: 0.6;
	}
}

.footer-tip__text {
	font-size: 26rpx;
	color: #a08775;
}
</style>
