<template>
	<view class="settings-page">
		<view class="settings-card">
			<view class="settings-card__header">
				<view>
					<text class="settings-card__title">B站字幕配置</text>
					<text class="settings-card__desc">维护整个应用共用的一份 SESSDATA，用来提升 B 站字幕命中率。</text>
				</view>
				<view class="settings-status" :class="`settings-status--${statusMeta.tone}`">
					<text class="settings-status__text">{{ statusMeta.label }}</text>
				</view>
			</view>

			<view class="settings-summary">
				<view class="settings-summary__item">
					<text class="settings-summary__label">当前值</text>
					<text class="settings-summary__value">{{ maskedSessdataText }}</text>
				</view>
				<view class="settings-summary__item">
					<text class="settings-summary__label">最近成功</text>
					<text class="settings-summary__value">{{ formatDateTime(setting?.lastSuccessAt) }}</text>
				</view>
				<view class="settings-summary__item">
					<text class="settings-summary__label">最近校验</text>
					<text class="settings-summary__value">{{ formatDateTime(setting?.lastCheckedAt) }}</text>
				</view>
			</view>

			<view v-if="setting?.lastError" class="settings-error">
				<text class="settings-error__text">{{ setting.lastError }}</text>
			</view>
		</view>

		<view class="settings-card">
			<view class="settings-card__header settings-card__header--stack">
				<text class="settings-card__title">更新 SESSDATA</text>
				<text class="settings-card__desc">可以直接粘贴 SESSDATA，或整段 Cookie，系统会自动提取。</text>
			</view>

			<textarea
				v-model="sessdataInput"
				class="settings-textarea"
				placeholder="粘贴新的 SESSDATA 或整段 Cookie"
				placeholder-class="settings-textarea__placeholder"
				maxlength="1200"
			/>

			<view class="settings-actions">
				<view class="settings-action" @tap="goBack">
					<text class="settings-action__text">返回</text>
				</view>
				<view
					class="settings-action settings-action--primary"
					:class="{ 'settings-action--disabled': !canSubmit || isSaving }"
					@tap="submitSessdata"
				>
					<text class="settings-action__text settings-action__text--primary">{{ isSaving ? '保存中...' : '验证并保存' }}</text>
				</view>
			</view>

			<view
				v-if="setting?.configured"
				class="settings-link"
				:class="{ 'settings-link--disabled': isClearing }"
				@tap="confirmClearSessdata"
			>
				<text class="settings-link__text">{{ isClearing ? '清空中...' : '清空当前配置' }}</text>
			</view>
		</view>

		<view class="settings-card settings-card--soft">
			<view class="settings-card__header settings-card__header--stack">
				<text class="settings-card__title">如何获取</text>
				<text class="settings-card__desc">登录 B 站后，在浏览器开发者工具里复制请求头中的 Cookie，或者只取其中的 SESSDATA。</text>
			</view>
			<text class="settings-help">建议只在服务端维护，不要放到前端代码里。更新成功后，新的解析任务会自动使用这份登录态。</text>
		</view>
	</view>
</template>

<script>
import {
	clearBilibiliSessionSetting,
	getBilibiliSessionSetting,
	updateBilibiliSessionSetting
} from '../../utils/app-settings-api'

const statusMetaMap = {
	unconfigured: { label: '未配置', tone: 'idle' },
	valid: { label: '已生效', tone: 'valid' }
}

export default {
	data() {
		return {
			setting: null,
			sessdataInput: '',
			isLoading: false,
			isSaving: false,
			isClearing: false
		}
	},
	onShow() {
		this.loadSetting()
	},
	computed: {
		statusMeta() {
			return statusMetaMap[this.setting?.status] || statusMetaMap.unconfigured
		},
		maskedSessdataText() {
			return this.setting?.maskedSessdata || '尚未配置'
		},
		canSubmit() {
			return !!String(this.sessdataInput || '').trim()
		}
	},
	methods: {
		async loadSetting() {
			if (this.isLoading) return
			this.isLoading = true
			try {
				this.setting = await getBilibiliSessionSetting()
			} catch (error) {
				uni.showToast({
					title: error?.message || '加载设置失败',
					icon: 'none'
				})
			} finally {
				this.isLoading = false
			}
		},
		async submitSessdata() {
			if (this.isSaving || !this.canSubmit) return
			this.isSaving = true
			uni.showLoading({
				title: '校验中',
				mask: true
			})

			try {
				this.setting = await updateBilibiliSessionSetting({
					sessdata: this.sessdataInput
				})
				this.sessdataInput = ''
				uni.showToast({
					title: '已保存',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '保存失败',
					icon: 'none'
				})
			} finally {
				this.isSaving = false
				uni.hideLoading()
			}
		},
		confirmClearSessdata() {
			if (this.isClearing) return
			uni.showModal({
				title: '清空配置',
				content: '清空后，B 站解析会回到匿名模式。后续可以再重新粘贴 SESSDATA。',
				confirmText: '确认清空',
				success: async ({ confirm }) => {
					if (!confirm) return
					await this.clearSessdata()
				}
			})
		},
		async clearSessdata() {
			if (this.isClearing) return
			this.isClearing = true
			try {
				this.setting = await clearBilibiliSessionSetting()
				this.sessdataInput = ''
				uni.showToast({
					title: '已清空',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '清空失败',
					icon: 'none'
				})
			} finally {
				this.isClearing = false
			}
		},
		goBack() {
			uni.navigateBack({
				fail: () => {
					uni.reLaunch({
						url: '/pages/index/index?section=kitchen'
					})
				}
			})
		},
		formatDateTime(value) {
			const raw = String(value || '').trim()
			if (!raw) return '暂无'
			const normalized = raw.includes('T') ? raw : raw.replace(' ', 'T')
			const date = new Date(normalized)
			if (Number.isNaN(date.getTime())) {
				return raw.replace('T', ' ').slice(0, 16)
			}
			const month = String(date.getMonth() + 1).padStart(2, '0')
			const day = String(date.getDate()).padStart(2, '0')
			const hours = String(date.getHours()).padStart(2, '0')
			const minutes = String(date.getMinutes()).padStart(2, '0')
			return `${month}-${day} ${hours}:${minutes}`
		}
	}
}
</script>

<style scoped>
	.settings-page {
		min-height: 100vh;
		padding: 24rpx;
		box-sizing: border-box;
		background: linear-gradient(180deg, #f6f1ea 0%, #efe7dd 100%);
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.settings-card {
		padding: 24rpx;
		border-radius: 28rpx;
		background: rgba(255, 255, 255, 0.92);
		border: 1px solid rgba(201, 186, 170, 0.35);
		box-shadow: 0 18rpx 40rpx rgba(82, 63, 44, 0.08);
		display: flex;
		flex-direction: column;
		gap: 20rpx;
	}

	.settings-card--soft {
		background: rgba(252, 248, 243, 0.92);
	}

	.settings-card__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 20rpx;
	}

	.settings-card__header--stack {
		flex-direction: column;
		align-items: stretch;
	}

	.settings-card__title {
		display: block;
		font-size: 32rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.settings-card__desc {
		display: block;
		margin-top: 10rpx;
		font-size: 23rpx;
		line-height: 1.7;
		color: #85786b;
	}

	.settings-status {
		padding: 10rpx 16rpx;
		border-radius: 999rpx;
		background: #ede7df;
		flex-shrink: 0;
	}

	.settings-status--valid {
		background: #e5efe2;
	}

	.settings-status__text {
		font-size: 22rpx;
		font-weight: 700;
		color: #726555;
	}

	.settings-status--valid .settings-status__text {
		color: #5f7a5b;
	}

	.settings-summary {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 12rpx;
	}

	.settings-summary__item {
		padding: 18rpx;
		border-radius: 20rpx;
		background: #f7f2ec;
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.settings-summary__label {
		font-size: 21rpx;
		color: #998d81;
	}

	.settings-summary__value {
		font-size: 24rpx;
		font-weight: 600;
		line-height: 1.5;
		color: #4d4034;
		word-break: break-all;
	}

	.settings-error {
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		background: #fff1eb;
	}

	.settings-error__text {
		font-size: 23rpx;
		line-height: 1.6;
		color: #a15d44;
	}

	.settings-textarea {
		width: 100%;
		min-height: 220rpx;
		padding: 22rpx 24rpx;
		box-sizing: border-box;
		border-radius: 24rpx;
		background: #f7f4f0;
		border: 1px solid #e4ddd5;
		font-size: 25rpx;
		line-height: 1.6;
		color: #312a24;
	}

	.settings-textarea__placeholder {
		color: #b0a79d;
	}

	.settings-actions {
		display: flex;
		gap: 14rpx;
	}

	.settings-action {
		flex: 1;
		height: 88rpx;
		border-radius: 24rpx;
		background: #f1ece6;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.settings-action--primary {
		background: #5b4a3b;
		box-shadow: 0 14rpx 24rpx rgba(91, 74, 59, 0.16);
	}

	.settings-action--disabled {
		pointer-events: none;
		background: #d9d1c8;
		box-shadow: none;
	}

	.settings-action__text {
		font-size: 28rpx;
		font-weight: 600;
		color: #6c6156;
	}

	.settings-action__text--primary {
		color: #ffffff;
	}

	.settings-link {
		align-self: center;
		padding: 10rpx 12rpx;
	}

	.settings-link--disabled {
		opacity: 0.6;
		pointer-events: none;
	}

	.settings-link__text {
		font-size: 24rpx;
		font-weight: 600;
		color: #9a644c;
	}

	.settings-help {
		font-size: 23rpx;
		line-height: 1.7;
		color: #7d7063;
	}
</style>
