<template>
	<view class="about-page">
		<view class="about-shell" :style="aboutShellStyle">
			<view class="about-hero">
				<view class="about-hero__mark">
					<up-icon name="heart-fill" size="30" color="#bf715f"></up-icon>
				</view>
				<text class="about-hero__title" @longpress="openHiddenDebugActions">我们的数字空间</text>
				<text class="about-hero__version">Version {{ appVersion }}</text>
			</view>

			<view class="mission-card">
				<text class="mission-card__title">产品的初心</text>
				<text class="mission-card__paragraph">
					做饭不应该是一个人的烦恼。我们希望为情侣、家庭或室友提供一个轻量的「<text class="mission-card__emphasis">共享厨房</text>」。
				</text>
				<text class="mission-card__paragraph">
					把平时在B站、小红书看到的灵感统一沉淀，解决平时「<text class="mission-card__emphasis">今天吃什么，要买什么菜</text>」的真实烦恼，让一起准备一顿饭变成顺理成章的日常。
				</text>
			</view>

			<view class="about-menu">
				<view
					class="about-menu__item"
					hover-class="about-menu__item--hover"
					hover-stay-time="80"
					@tap="openFeatureGuide"
				>
					<view class="about-menu__icon about-menu__icon--guide">
						<up-icon name="file-text" size="18" color="#8b6b52"></up-icon>
					</view>
					<text class="about-menu__title">功能介绍与使用指南</text>
					<up-icon name="arrow-right" size="14" color="#b3a394"></up-icon>
				</view>

				<view
					class="about-menu__item"
					hover-class="about-menu__item--hover"
					hover-stay-time="80"
					@tap="openFeedback"
				>
					<view class="about-menu__icon about-menu__icon--feedback">
						<up-icon name="edit-pen" size="18" color="#a76655"></up-icon>
					</view>
					<text class="about-menu__title">反馈与吐槽</text>
					<up-icon name="arrow-right" size="14" color="#b3a394"></up-icon>
				</view>

				<view
					class="about-menu__item"
					hover-class="about-menu__item--hover"
					hover-stay-time="80"
					@tap="openFilingInfo"
				>
					<view class="about-menu__icon about-menu__icon--filing">
						<up-icon name="info-circle" size="18" color="#7f735f"></up-icon>
					</view>
					<text class="about-menu__title">备案信息</text>
					<up-icon name="arrow-right" size="14" color="#b3a394"></up-icon>
				</view>

				<view
					v-if="canOpenAppSettings"
					class="about-menu__item"
					hover-class="about-menu__item--hover"
					hover-stay-time="80"
					@tap="openAppSettings"
				>
					<view class="about-menu__icon about-menu__icon--settings">
						<up-icon name="grid-fill" size="17" color="#6e5f50"></up-icon>
					</view>
					<text class="about-menu__title">应用设置</text>
					<up-icon name="arrow-right" size="14" color="#b3a394"></up-icon>
				</view>
			</view>

			<text class="about-footer">MADE WITH LOVE</text>
		</view>
	</view>
</template>

<script>
import { appConfig } from '../../utils/app-config'
import { getBilibiliSessionSetting } from '../../utils/app-settings-api'
import { ensureSession, getSessionSnapshot } from '../../utils/auth'
import { clearImageCache } from '../../utils/image-cache'

const APP_VERSION = '1.0.0'

export default {
	data() {
		return {
			currentUser: null,
			pageTopInset: 0
		}
	},
	onLoad() {
		this.syncNavigationMetrics()
	},
	onShow() {
		this.syncNavigationMetrics()
		this.applySession()
		this.syncSession()
	},
	computed: {
		aboutShellStyle() {
			if (!this.pageTopInset) return ''
			return `padding-top: ${this.pageTopInset}px;`
		},
		appVersion() {
			return APP_VERSION
		},
		miniProgramFilingNumber() {
			return String(appConfig.miniProgramFilingNumber || '').trim()
		},
		miniProgramFilingSystemURL() {
			return String(appConfig.miniProgramFilingSystemURL || 'https://beian.miit.gov.cn/').trim()
		},
		hasMiniProgramFiling() {
			return !!this.miniProgramFilingNumber
		},
		canOpenAppSettings() {
			return !!this.currentUser?.canManageAppSettings
		}
	},
	methods: {
		applySession(session = getSessionSnapshot()) {
			this.currentUser = session?.user || null
		},
		async syncSession() {
			try {
				const session = await ensureSession()
				this.applySession(session)
			} catch (error) {
				this.applySession()
			}
		},
		syncNavigationMetrics() {
			if (typeof wx === 'undefined' || typeof wx.getMenuButtonBoundingClientRect !== 'function') {
				this.pageTopInset = 0
				return
			}

			try {
				const menuButton = wx.getMenuButtonBoundingClientRect()
				const menuBottom = Number(menuButton?.bottom) || 0
				if (!menuBottom) {
					this.pageTopInset = 0
					return
				}

				this.pageTopInset = Math.ceil(menuBottom + 4)
			} catch (error) {
				this.pageTopInset = 0
			}
		},
		openFeatureGuide() {
			uni.showModal({
				title: '功能介绍与使用指南',
				content: '可以在美食库记录菜谱，按想吃和吃过整理状态；也可以在空间里邀请成员，一起维护菜单计划。',
				showCancel: false,
				confirmText: '知道了'
			})
		},
		openFeedback() {
			uni.showModal({
				title: '反馈与吐槽',
				content: '现在可以先把问题或建议发给空间维护者。后续会接入更直接的反馈入口。',
				showCancel: false,
				confirmText: '知道了'
			})
		},
		openFilingInfo() {
			const itemList = this.hasMiniProgramFiling ? ['复制备案编号', '复制查询网址'] : ['复制查询网址']

			uni.showActionSheet({
				itemList,
				success: ({ tapIndex }) => {
					const action = itemList[tapIndex]
					if (action === '复制备案编号') {
						this.copyMiniProgramFilingNumber()
						return
					}
					this.copyMiniProgramFilingSystemURL()
				}
			})
		},
		copyMiniProgramFilingNumber() {
			if (!this.hasMiniProgramFiling) return

			uni.setClipboardData({
				data: this.miniProgramFilingNumber,
				success: () => {
					uni.showToast({
						title: '备案编号已复制',
						icon: 'none'
					})
				}
			})
		},
		copyMiniProgramFilingSystemURL() {
			uni.setClipboardData({
				data: this.miniProgramFilingSystemURL,
				success: () => {
					uni.showToast({
						title: '查询网址已复制',
						icon: 'none'
					})
				}
			})
		},
		async openAppSettings() {
			if (!this.canOpenAppSettings) return

			try {
				await getBilibiliSessionSetting()
			} catch (error) {
				uni.showToast({
					title: error?.message || '暂时无法打开设置',
					icon: 'none'
				})
				return
			}

			uni.navigateTo({
				url: '/pages/app-settings/index'
			})
		},
		openHiddenDebugActions() {
			uni.showActionSheet({
				itemList: ['清空首页图片缓存'],
				success: ({ tapIndex }) => {
					if (tapIndex !== 0) return
					this.confirmClearImageCache()
				}
			})
		},
		confirmClearImageCache() {
			uni.showModal({
				title: '清空图片缓存',
				content: '会删除首页已缓存的封面图，下次进入首页会重新下载，是否继续？',
				confirmText: '清空',
				success: async ({ confirm }) => {
					if (!confirm) return
					await this.clearRecipeImageCache()
				}
			})
		},
		async clearRecipeImageCache() {
			try {
				const result = await clearImageCache()
				const removedCount = Number(result?.removedCount) || 0
				uni.showToast({
					title: removedCount ? `已清空 ${removedCount} 张` : '缓存已清空',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '清空缓存失败',
					icon: 'none'
				})
			}
		}
	}
}
</script>

<style scoped>
	.about-page {
		min-height: 100vh;
		background:
			radial-gradient(circle at 50% 0%, rgba(255, 248, 239, 0.98) 0%, rgba(255, 248, 239, 0) 52%),
			linear-gradient(180deg, #f8f3ec 0%, #efe5da 100%);
		box-sizing: border-box;
		color: #2f2923;
	}

	.about-shell {
		min-height: 100vh;
		padding: calc(var(--status-bar-height, 0px) + 18rpx) 28rpx calc(env(safe-area-inset-bottom) + 36rpx);
		box-sizing: border-box;
		display: flex;
		flex-direction: column;
		gap: 24rpx;
	}

	.about-hero {
		padding: 18rpx 16rpx 30rpx;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}

	.about-hero__mark {
		width: 78rpx;
		height: 78rpx;
		border-radius: 24rpx;
		background: rgba(191, 113, 95, 0.1);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.about-hero__title {
		margin-top: 28rpx;
		font-size: 42rpx;
		font-family: "Songti SC", "STKaiti", "Source Han Serif SC", "Noto Serif CJK SC", serif;
		font-weight: 700;
		line-height: 1.2;
		color: #2f2923;
		letter-spacing: 0;
	}

	.about-hero__version {
		margin-top: 12rpx;
		padding: 8rpx 18rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.66);
		border: 1rpx solid rgba(189, 169, 150, 0.24);
		font-size: 22rpx;
		font-weight: 600;
		line-height: 1;
		color: #9c8b7b;
	}

	.mission-card,
	.about-menu {
		background: rgba(255, 253, 249, 0.96);
		border: 1rpx solid rgba(205, 187, 169, 0.34);
		box-shadow:
			0 16rpx 36rpx rgba(93, 69, 49, 0.08),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.76);
	}

	.mission-card {
		padding: 30rpx;
		border-radius: 32rpx;
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.mission-card__title {
		font-size: 31rpx;
		font-family: "Songti SC", "STKaiti", "Source Han Serif SC", "Noto Serif CJK SC", serif;
		font-weight: 700;
		line-height: 1.25;
		color: #2f2923;
	}

	.mission-card__paragraph {
		font-size: 26rpx;
		line-height: 1.72;
		color: #6f6256;
	}

	.mission-card__emphasis {
		font-weight: 800;
		color: #8f4f42;
	}

	.about-menu {
		border-radius: 30rpx;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.about-menu__item {
		min-height: 96rpx;
		padding: 16rpx 24rpx;
		box-sizing: border-box;
		display: flex;
		align-items: center;
		gap: 16rpx;
		background: rgba(255, 253, 249, 0);
		transition: background 0.16s ease, transform 0.16s ease;
	}

	.about-menu__item + .about-menu__item {
		border-top: 1rpx solid rgba(216, 201, 185, 0.48);
	}

	.about-menu__item--hover {
		background: rgba(244, 235, 225, 0.62);
		transform: scale(0.992);
	}

	.about-menu__icon {
		flex: 0 0 auto;
		width: 50rpx;
		height: 50rpx;
		border-radius: 18rpx;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.about-menu__icon--guide {
		background: rgba(191, 113, 95, 0.1);
	}

	.about-menu__icon--feedback {
		background: rgba(166, 102, 85, 0.1);
	}

	.about-menu__icon--filing {
		background: rgba(127, 115, 95, 0.1);
	}

	.about-menu__icon--settings {
		background: rgba(91, 74, 59, 0.1);
	}

	.about-menu__title {
		min-width: 0;
		flex: 1 1 auto;
		font-size: 28rpx;
		font-weight: 700;
		line-height: 1.35;
		color: #352b24;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.about-footer {
		margin-top: auto;
		padding-top: 36rpx;
		text-align: center;
		font-size: 21rpx;
		font-weight: 700;
		line-height: 1.2;
		color: rgba(126, 111, 96, 0.38);
		letter-spacing: 0;
	}
</style>
