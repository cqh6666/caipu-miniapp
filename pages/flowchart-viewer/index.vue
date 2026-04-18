<template>
	<view class="flowchart-viewer">
		<template v-if="imageUrl && !imageFailed">
			<view class="flowchart-viewer__stage" @tap="toggleChrome">
				<view class="flowchart-viewer__image-shell">
					<image
						class="flowchart-viewer__image"
						:src="imageUrl"
						mode="aspectFit"
						@load="handleImageLoad"
						@error="handleImageError"
					></image>
				</view>

				<view
					class="flowchart-viewer__topbar"
					:class="{ 'flowchart-viewer__topbar--hidden': !chromeVisible }"
				>
					<view class="flowchart-viewer__topbar-inner">
						<view class="flowchart-viewer__back" @tap.stop="goBack">
							<up-icon name="arrow-left" size="18" color="#f8f3ed"></up-icon>
							<text class="flowchart-viewer__back-text">返回</text>
						</view>
						<view class="flowchart-viewer__meta">
							<text class="flowchart-viewer__title">{{ pageTitle }}</text>
							<text class="flowchart-viewer__subtitle">{{ pageSubtitle }}</text>
						</view>
					</view>
				</view>

				<view
					v-if="chromeVisible"
					class="flowchart-viewer__hint"
					:class="{ 'flowchart-viewer__hint--loading': imageLoading }"
				>
					<up-icon v-if="imageLoading" name="reload" size="14" color="#f2ded0" class="flowchart-viewer__hint-icon"></up-icon>
					<text class="flowchart-viewer__hint-text">{{ imageLoading ? '步骤图加载中' : '轻点画面可隐藏操作区' }}</text>
				</view>
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

export default {
	data() {
		return {
			viewerKey: '',
			imageUrl: '',
			title: '',
			updatedAtText: '',
			chromeVisible: true,
			imageLoading: true,
			imageFailed: false
		}
	},
	computed: {
		pageTitle() {
			const title = String(this.title || '').trim()
			return title ? `${title} · 一图看懂` : '菜品步骤图'
		},
		pageSubtitle() {
			return String(this.updatedAtText || '').trim() || '横屏沉浸查看'
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
			this.title = String(payload.title || '').trim()
			this.updatedAtText = String(payload.updatedAtText || '').trim()
			this.imageFailed = !this.imageUrl
			this.imageLoading = !!this.imageUrl
		},
		setKeepScreenOn(keepScreenOn = false) {
			if (typeof uni.setKeepScreenOn !== 'function') return
			uni.setKeepScreenOn({
				keepScreenOn: !!keepScreenOn
			})
		},
		toggleChrome() {
			if (!this.imageUrl || this.imageFailed) return
			this.chromeVisible = !this.chromeVisible
		},
		handleImageLoad() {
			this.imageLoading = false
			this.imageFailed = false
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
		background:
			radial-gradient(circle at top right, rgba(232, 164, 103, 0.16) 0%, rgba(232, 164, 103, 0) 34%),
			linear-gradient(180deg, #17110d 0%, #0d0907 100%);
	}

	.flowchart-viewer__stage {
		position: relative;
		width: 100vw;
		height: 100vh;
		overflow: hidden;
	}

	.flowchart-viewer__stage::before {
		content: '';
		position: absolute;
		inset: 0;
		background:
			radial-gradient(circle at center, rgba(255, 255, 255, 0.06), rgba(255, 255, 255, 0) 56%),
			linear-gradient(90deg, rgba(255, 255, 255, 0.025) 0, rgba(255, 255, 255, 0) 18%, rgba(255, 255, 255, 0) 82%, rgba(255, 255, 255, 0.025) 100%);
		pointer-events: none;
	}

	.flowchart-viewer__image-shell {
		position: absolute;
		inset: 0;
		padding:
			calc(env(safe-area-inset-top) + 30rpx)
			calc(env(safe-area-inset-right) + 28rpx)
			calc(env(safe-area-inset-bottom) + 30rpx)
			calc(env(safe-area-inset-left) + 28rpx);
		box-sizing: border-box;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.flowchart-viewer__image {
		width: 100%;
		height: 100%;
		display: block;
	}

	.flowchart-viewer__topbar {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		padding:
			calc(env(safe-area-inset-top) + 18rpx)
			calc(env(safe-area-inset-right) + 24rpx)
			0
			calc(env(safe-area-inset-left) + 24rpx);
		box-sizing: border-box;
		opacity: 1;
		transform: translateY(0);
		transition: opacity 0.18s ease, transform 0.18s ease;
	}

	.flowchart-viewer__topbar--hidden {
		opacity: 0;
		transform: translateY(-16rpx);
		pointer-events: none;
	}

	.flowchart-viewer__topbar-inner {
		display: flex;
		align-items: center;
		gap: 18rpx;
		padding: 16rpx 18rpx;
		border-radius: 24rpx;
		background: rgba(19, 15, 11, 0.56);
		border: 1px solid rgba(255, 255, 255, 0.08);
		backdrop-filter: blur(18rpx);
		box-shadow:
			0 18rpx 36rpx rgba(0, 0, 0, 0.18),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.08);
	}

	.flowchart-viewer__back {
		flex-shrink: 0;
		min-height: 64rpx;
		padding: 0 18rpx;
		border-radius: 999rpx;
		background:
			linear-gradient(180deg, rgba(124, 95, 73, 0.92), rgba(89, 67, 50, 0.96));
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 10rpx;
		box-shadow:
			0 10rpx 20rpx rgba(0, 0, 0, 0.14),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.12);
	}

	.flowchart-viewer__back-text {
		font-size: 24rpx;
		font-weight: 600;
		line-height: 1;
		color: #f8f3ed;
	}

	.flowchart-viewer__meta {
		flex: 1;
		min-width: 0;
	}

	.flowchart-viewer__title,
	.flowchart-viewer__subtitle {
		display: block;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.flowchart-viewer__title {
		font-size: 28rpx;
		font-weight: 700;
		line-height: 1.2;
		color: #fff8f0;
	}

	.flowchart-viewer__subtitle {
		margin-top: 8rpx;
		font-size: 21rpx;
		line-height: 1.3;
		color: rgba(255, 240, 226, 0.72);
	}

	.flowchart-viewer__hint {
		position: absolute;
		left: 50%;
		bottom: calc(env(safe-area-inset-bottom) + 20rpx);
		transform: translateX(-50%);
		max-width: calc(100vw - 80rpx);
		padding: 12rpx 18rpx;
		border-radius: 999rpx;
		background: rgba(21, 16, 11, 0.58);
		border: 1px solid rgba(255, 255, 255, 0.08);
		backdrop-filter: blur(18rpx);
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
		box-sizing: border-box;
	}

	.flowchart-viewer__hint--loading {
		background: rgba(62, 43, 31, 0.76);
	}

	.flowchart-viewer__hint-icon {
		animation: flowchart-viewer-rotate 0.9s linear infinite;
	}

	.flowchart-viewer__hint-text {
		font-size: 21rpx;
		line-height: 1.3;
		color: #f2ded0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
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
