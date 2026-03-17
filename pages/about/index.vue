<template>
	<view class="about-page">
		<view class="about-card">
			<view class="about-card__header">
				<text class="about-card__title">关于我们</text>
				<text class="about-card__desc">一份给家庭和小团队共用的数字厨房。</text>
			</view>

			<text class="about-card__body">
				和家人共享菜单，记录想吃和吃过，也能把视频菜谱自动整理成食材与步骤。
			</text>
		</view>

		<view class="about-card about-card--soft">
			<view class="about-card__header about-card__header--compact">
				<text class="about-card__title">备案信息</text>
				<text class="about-card__desc">本小程序已完成互联网信息服务备案，可复制备案编号或查询网址。</text>
			</view>

			<view v-if="hasMiniProgramFiling" class="filing-number-card" @tap="copyMiniProgramFilingNumber">
				<text class="filing-number-card__label">备案编号</text>
				<text class="filing-number-card__value">{{ miniProgramFilingNumber }}</text>
				<text class="filing-number-card__tip">点击备案编号可复制</text>
			</view>

			<view v-else class="filing-empty">
				<text class="filing-empty__text">暂未配置备案编号。</text>
			</view>

			<view class="filing-query">
				<text class="filing-query__label">查询网址</text>
				<text class="filing-query__value">{{ miniProgramFilingSystemURL }}</text>
				<text class="filing-query__hint">微信小程序内不直接打开外部备案网页，可复制网址后前往浏览器查询。</text>
			</view>

			<view class="filing-actions">
				<view
					v-if="hasMiniProgramFiling"
					class="filing-action filing-action--primary"
					@tap="copyMiniProgramFilingNumber"
				>
					<text class="filing-action__text filing-action__text--primary">复制备案编号</text>
				</view>
				<view class="filing-action" @tap="copyMiniProgramFilingSystemURL">
					<text class="filing-action__text">复制查询网址</text>
				</view>
			</view>
		</view>

		<view v-if="canOpenAppSettings" class="about-card about-card--soft">
			<view class="about-card__header about-card__header--compact">
				<text class="about-card__title">应用设置</text>
				<text class="about-card__desc">仅对具备权限的账号显示。</text>
			</view>

			<view class="filing-actions">
				<view class="filing-action filing-action--primary" @tap="openAppSettings">
					<text class="filing-action__text filing-action__text--primary">打开应用设置</text>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
import { appConfig } from '../../utils/app-config'
import { getBilibiliSessionSetting } from '../../utils/app-settings-api'
import { ensureSession, getSessionSnapshot } from '../../utils/auth'

export default {
	data() {
		return {
			currentUser: null
		}
	},
	onShow() {
		this.applySession()
		this.syncSession()
	},
	computed: {
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
		}
	}
}
</script>

<style scoped>
	.about-page {
		min-height: 100vh;
		padding: 24rpx;
		box-sizing: border-box;
		background: linear-gradient(180deg, #f6f1ea 0%, #efe7dd 100%);
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.about-card {
		padding: 24rpx;
		border-radius: 28rpx;
		background: rgba(255, 255, 255, 0.94);
		border: 1px solid rgba(201, 186, 170, 0.35);
		box-shadow: 0 18rpx 40rpx rgba(82, 63, 44, 0.08);
		display: flex;
		flex-direction: column;
		gap: 20rpx;
	}

	.about-card--soft {
		background: rgba(252, 248, 243, 0.94);
	}

	.about-card__header {
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.about-card__header--compact {
		gap: 8rpx;
	}

	.about-card__title {
		font-size: 32rpx;
		font-weight: 700;
		line-height: 1.35;
		color: #2f2923;
	}

	.about-card__desc {
		font-size: 23rpx;
		line-height: 1.65;
		color: #85786d;
	}

	.about-card__body {
		font-size: 24rpx;
		line-height: 1.75;
		color: #6f6053;
	}

	.filing-number-card {
		padding: 22rpx 20rpx;
		border-radius: 22rpx;
		background: linear-gradient(135deg, rgba(249, 242, 234, 0.98), rgba(244, 235, 225, 0.94));
		border: 1px solid rgba(180, 156, 131, 0.18);
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.filing-number-card__label,
	.filing-query__label {
		font-size: 21rpx;
		font-weight: 600;
		color: #927d6a;
	}

	.filing-number-card__value,
	.filing-query__value {
		font-size: 30rpx;
		font-weight: 700;
		line-height: 1.45;
		color: #47392d;
		word-break: break-all;
	}

	.filing-number-card__tip,
	.filing-query__hint,
	.filing-empty__text {
		font-size: 22rpx;
		line-height: 1.65;
		color: #85786d;
	}

	.filing-empty {
		padding: 22rpx 20rpx;
		border-radius: 22rpx;
		background: rgba(250, 246, 241, 0.9);
		border: 1px dashed rgba(180, 156, 131, 0.3);
	}

	.filing-query {
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.filing-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 12rpx;
	}

	.filing-action {
		padding: 16rpx 20rpx;
		border-radius: 18rpx;
		background: rgba(91, 74, 59, 0.08);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.filing-action--primary {
		background: linear-gradient(180deg, #5c493c 0%, #46362c 100%);
	}

	.filing-action__text {
		font-size: 23rpx;
		font-weight: 600;
		color: #5d4c3c;
	}

	.filing-action__text--primary {
		color: #ffffff;
	}
</style>
