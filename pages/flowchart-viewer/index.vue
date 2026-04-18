<template>
	<view class="flowchart-viewer">
		<template v-if="imageUrl && !imageFailed">
			<movable-area class="flowchart-viewer__stage">
				<movable-view
					class="flowchart-viewer__canvas"
					direction="all"
					:animation="false"
					:inertia="true"
					:out-of-bounds="false"
					:scale="true"
					:scale-min="minScale"
					:scale-max="maxScale"
					:scale-value="imageScale"
					@scale="handleViewerScale"
				>
					<image
						class="flowchart-viewer__image"
						:src="imageUrl"
						mode="aspectFit"
						@load="handleImageLoad"
						@error="handleImageError"
					></image>
				</movable-view>
			</movable-area>

			<view v-if="imageLoading" class="flowchart-viewer__loading">
				<view class="flowchart-viewer__loading-chip">
					<up-icon name="reload" size="14" color="#f6e5d7" class="flowchart-viewer__loading-icon"></up-icon>
					<text class="flowchart-viewer__loading-text">步骤图加载中</text>
				</view>
			</view>

			<view class="flowchart-viewer__close" @tap="goBack">
				<up-icon name="close" size="15" color="#fbf5ee"></up-icon>
			</view>
		</template>

		<template v-else>
			<view class="flowchart-viewer__empty">
				<view class="flowchart-viewer__empty-icon">
					<up-icon name="photo" size="28" color="#cfbaa9"></up-icon>
				</view>
				<text class="flowchart-viewer__empty-title">步骤图暂时不可查看</text>
				<text class="flowchart-viewer__empty-desc">{{ imageFailed ? '图片加载失败，可以返回详情页后重试。' : '没有拿到可用的流程图数据。' }}</text>
				<view class="flowchart-viewer__empty-action" @tap="goBack">
					<text class="flowchart-viewer__empty-action-text">返回详情</text>
				</view>
			</view>
		</template>
	</view>
</template>

<script>
const FLOWCHART_VIEWER_STORAGE_KEY = 'recipe-flowchart-viewer-payload'

function clampScale(value = 1, min = 1, max = 4) {
	const parsed = Number(value)
	if (!Number.isFinite(parsed)) return min
	return Math.min(Math.max(parsed, min), max)
}

export default {
	data() {
		return {
			viewerKey: '',
			imageUrl: '',
			imageLoading: true,
			imageFailed: false,
			minScale: 1,
			maxScale: 4,
			imageScale: 1
		}
	},
	onLoad(options = {}) {
		this.viewerKey = String(options.key || '').trim()
		this.restorePayload()
	},
	onShow() {
		this.setKeepScreenOn(true)
	},
	onHide() {
		this.setKeepScreenOn(false)
	},
	onUnload() {
		this.setKeepScreenOn(false)
		const payload = uni.getStorageSync(FLOWCHART_VIEWER_STORAGE_KEY)
		if (payload && payload.key === this.viewerKey) {
			uni.removeStorageSync(FLOWCHART_VIEWER_STORAGE_KEY)
		}
	},
	methods: {
		restorePayload() {
			const payload = uni.getStorageSync(FLOWCHART_VIEWER_STORAGE_KEY)
			if (!payload || payload.key !== this.viewerKey) {
				this.imageLoading = false
				return
			}

			this.imageUrl = String(payload.imageUrl || '').trim()
			this.imageFailed = !this.imageUrl
			this.imageLoading = !!this.imageUrl
			this.imageScale = this.minScale
		},
		setKeepScreenOn(keepScreenOn = false) {
			if (typeof uni.setKeepScreenOn !== 'function') return
			uni.setKeepScreenOn({
				keepScreenOn: !!keepScreenOn
			})
		},
		handleViewerScale(event) {
			this.imageScale = clampScale(event?.detail?.scale, this.minScale, this.maxScale)
		},
		handleImageLoad() {
			this.imageLoading = false
			this.imageFailed = false
			this.imageScale = this.minScale
		},
		handleImageError() {
			this.imageLoading = false
			this.imageFailed = true
		},
		goBack() {
			if (getCurrentPages().length > 1) {
				uni.navigateBack()
				return
			}
			uni.reLaunch({
				url: '/pages/index/index'
			})
		}
	}
}
</script>

<style lang="scss" scoped>
	.flowchart-viewer {
		min-height: 100vh;
		background: #050505;
	}

	.flowchart-viewer__stage {
		width: 100vw;
		height: 100vh;
		overflow: hidden;
		background:
			radial-gradient(circle at center, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0) 58%),
			linear-gradient(180deg, #120e0b 0%, #050505 100%);
	}

	.flowchart-viewer__canvas {
		width: 100%;
		height: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.flowchart-viewer__image {
		width: 100%;
		height: 100%;
		display: block;
	}

	.flowchart-viewer__loading {
		position: fixed;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		pointer-events: none;
	}

	.flowchart-viewer__loading-chip {
		padding: 14rpx 20rpx;
		border-radius: 999rpx;
		background: rgba(18, 13, 10, 0.72);
		border: 1px solid rgba(255, 255, 255, 0.08);
		backdrop-filter: blur(16rpx);
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
		box-shadow:
			0 18rpx 32rpx rgba(0, 0, 0, 0.2),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.08);
	}

	.flowchart-viewer__loading-icon {
		animation: flowchart-viewer-rotate 0.9s linear infinite;
	}

	.flowchart-viewer__loading-text {
		font-size: 22rpx;
		line-height: 1;
		color: #f6e5d7;
	}

	.flowchart-viewer__close {
		position: fixed;
		top: calc(env(safe-area-inset-top) + 18rpx);
		left: calc(env(safe-area-inset-left) + 18rpx);
		z-index: 5;
		width: 54rpx;
		height: 54rpx;
		border-radius: 999rpx;
		background: rgba(14, 11, 8, 0.36);
		border: 1px solid rgba(255, 255, 255, 0.06);
		backdrop-filter: blur(10rpx);
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow:
			0 8rpx 16rpx rgba(0, 0, 0, 0.14),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.06);
	}

	.flowchart-viewer__empty {
		min-height: 100vh;
		padding: 48rpx;
		box-sizing: border-box;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
	}

	.flowchart-viewer__empty-icon {
		width: 96rpx;
		height: 96rpx;
		border-radius: 28rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0.06));
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.flowchart-viewer__empty-title {
		margin-top: 22rpx;
		font-size: 34rpx;
		font-weight: 700;
		line-height: 1.3;
		color: #fff3e7;
	}

	.flowchart-viewer__empty-desc {
		margin-top: 12rpx;
		max-width: 620rpx;
		font-size: 24rpx;
		line-height: 1.7;
		color: rgba(255, 236, 219, 0.72);
	}

	.flowchart-viewer__empty-action {
		margin-top: 24rpx;
		min-height: 78rpx;
		padding: 0 30rpx;
		border-radius: 999rpx;
		background:
			linear-gradient(180deg, rgba(126, 96, 73, 0.96), rgba(87, 65, 48, 0.98));
		display: inline-flex;
		align-items: center;
		justify-content: center;
		box-shadow:
			0 14rpx 28rpx rgba(0, 0, 0, 0.2),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.14);
	}

	.flowchart-viewer__empty-action-text {
		font-size: 24rpx;
		font-weight: 600;
		line-height: 1;
		color: #fff8f0;
	}

	@keyframes flowchart-viewer-rotate {
		from {
			transform: rotate(0deg);
		}

		to {
			transform: rotate(360deg);
		}
	}
</style>
