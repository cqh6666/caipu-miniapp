<template>
	<view class="invite-page">
		<view class="invite-shell">
			<view v-if="isLoading" class="invite-card invite-card--loading">
				<up-icon name="reload" size="26" color="#8c7e72"></up-icon>
				<text class="invite-loading__title">正在获取邀请信息</text>
				<text class="invite-loading__desc">稍等一下，我们先确认这个空间邀请是否可用。</text>
			</view>

			<view v-else-if="errorMessage" class="invite-card">
				<view class="invite-badge">共享空间邀请</view>
				<text class="invite-title">没能打开这份邀请</text>
				<text class="invite-description">{{ errorMessage }}</text>
				<view class="invite-actions invite-actions--stack">
					<view class="invite-action invite-action--primary" @tap="initializePage">
						<text class="invite-action__text invite-action__text--primary">重新加载</text>
					</view>
					<view class="invite-action" @tap="goHome">
						<text class="invite-action__text">返回首页</text>
					</view>
				</view>
			</view>

			<view v-else class="invite-card">
				<view class="invite-badge">共享空间邀请</view>
				<text class="invite-title">{{ inviteTitle }}</text>
				<text class="invite-description">{{ inviteDescription }}</text>

				<view class="invite-summary">
					<view class="invite-summary__item">
						<text class="invite-summary__label">空间名称</text>
						<text class="invite-summary__value">{{ inviteDisplayName || '未命名空间' }}</text>
					</view>
					<view class="invite-summary__item">
						<text class="invite-summary__label">邀请人</text>
						<text class="invite-summary__value">{{ inviterName }}</text>
					</view>
					<view class="invite-summary__item">
						<text class="invite-summary__label">有效期</text>
						<text class="invite-summary__value">{{ inviteExpiryText }}</text>
					</view>
					<view class="invite-summary__item">
						<text class="invite-summary__label">剩余名额</text>
						<text class="invite-summary__value">{{ inviteRemainingText }}</text>
					</view>
					<view v-if="inviteCodeText" class="invite-summary__item">
						<text class="invite-summary__label">邀请码</text>
						<text class="invite-summary__value invite-summary__value--code">{{ inviteCodeText }}</text>
					</view>
				</view>

				<view class="invite-notice" :class="{ 'invite-notice--success': hasAccepted || alreadyMember }">
					<up-icon
						:name="hasAccepted || alreadyMember ? 'checkmark-circle-fill' : 'info-circle-fill'"
						size="16"
						:color="hasAccepted || alreadyMember ? '#72856f' : '#a07d61'"
					></up-icon>
					<text class="invite-notice__text">{{ inviteNoticeText }}</text>
				</view>

				<view class="invite-actions">
					<view class="invite-action invite-action--primary" @tap="handlePrimaryAction">
						<text class="invite-action__text invite-action__text--primary">{{ primaryActionText }}</text>
					</view>
					<view v-if="showSecondaryAction" class="invite-action" @tap="goHome">
						<text class="invite-action__text">稍后再说</text>
					</view>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
import { ensureSession, getFriendlySessionErrorMessage, getSessionSnapshot, setCurrentKitchenId, updateSessionKitchens } from '../../utils/auth'
import { acceptInvite, acceptInviteByCode, formatInviteCode, normalizeInviteCode, previewInvite, previewInviteByCode } from '../../utils/kitchen-api'

const inviteStatusLabelMap = {
	active: '可加入',
	expired: '已过期',
	used_up: '名额已满',
	revoked: '已失效'
}

function replaceKitchenLabel(value = '') {
	return String(value || '').replace(/厨房/g, '空间')
}

export default {
	data() {
		return {
			token: '',
			code: '',
			invite: null,
			session: getSessionSnapshot(),
			isLoading: true,
			isAccepting: false,
			errorMessage: '',
			hasAccepted: false
		}
	},
	computed: {
		inviteStatus() {
			return this.invite?.status || ''
		},
		inviteDisplayName() {
			return replaceKitchenLabel(this.invite?.kitchenName || '')
		},
		inviterName() {
			return this.invite?.inviter?.nickname || '空间成员'
		},
		inviteExpiryText() {
			return this.formatDateTime(this.invite?.expiresAt)
		},
		inviteRemainingText() {
			if (!this.invite) return '--'
			return `${this.invite.remainingUses} / ${this.invite.maxUses}`
		},
		inviteCodeText() {
			return formatInviteCode(this.invite?.code || this.code)
		},
		alreadyMember() {
			if (!this.invite?.kitchenId) return false
			const kitchens = Array.isArray(this.session?.kitchens) ? this.session.kitchens : []
			return kitchens.some((item) => Number(item.id) === Number(this.invite.kitchenId))
		},
		isInviteUnavailable() {
			return ['expired', 'used_up', 'revoked'].includes(this.inviteStatus)
		},
		inviteTitle() {
			if (this.hasAccepted) {
				return `已加入 ${this.inviteDisplayName || '这个空间'}`
			}
			if (this.alreadyMember) {
				return `你已经在 ${this.inviteDisplayName || '这个空间'}`
			}
			if (this.isInviteUnavailable) {
				return `${this.inviteDisplayName || '这份邀请'} 当前不可加入`
			}
			return `${this.inviterName} 邀请你加入`
		},
		inviteDescription() {
			if (this.hasAccepted) {
				return '现在你们可以一起维护这份菜单了，进入空间就能看到同一份菜谱。'
			}
			if (this.alreadyMember) {
				return '这说明你之前已经加入过，不用重复操作，直接进入就可以。'
			}
			if (this.isInviteUnavailable) {
				return `当前状态：${inviteStatusLabelMap[this.inviteStatus] || '不可用'}。你可以让邀请人重新发一份新的邀请。`
			}
			return '加入后你会把这个空间加入自己的列表，后续可以和对方一起维护菜谱。'
		},
		inviteNoticeText() {
			if (this.hasAccepted) {
				return '系统已经帮你切到这个空间，点下面按钮就能直接进入。'
			}
			if (this.alreadyMember) {
				return '你已经是这个空间的成员了，直接进入就好。'
			}
			if (this.isInviteUnavailable) {
				return '这份邀请已经不能再使用了，需要让对方重新发一个新的。'
			}
			return '加入后你的原有空间不会丢，你仍然可以在空间页里自由切换。'
		},
		primaryActionText() {
			if (this.isLoading) return '加载中'
			if (this.isInviteUnavailable || this.errorMessage) return '返回首页'
			if (this.hasAccepted || this.alreadyMember) return '进入这个空间'
			return this.isAccepting ? '加入中...' : '加入这个空间'
		},
		showSecondaryAction() {
			return !this.errorMessage && !this.isInviteUnavailable && !this.hasAccepted && !this.alreadyMember
		}
	},
	onLoad(options) {
		this.token = options?.token || ''
		this.code = normalizeInviteCode(options?.code || '')
	},
	onShow() {
		this.initializePage()
	},
	methods: {
		async initializePage() {
			if (!this.token && !this.code) {
				this.errorMessage = '缺少邀请参数，无法判断你要加入哪一个空间。'
				this.isLoading = false
				return
			}

			this.isLoading = true
			this.errorMessage = ''
			this.hasAccepted = false

			try {
				const invite = this.code ? await previewInviteByCode(this.code) : await previewInvite(this.token)
				this.invite = invite
			} catch (error) {
				this.errorMessage = error?.message || '这份邀请不存在，或已经被撤回。'
				this.invite = null
				this.isLoading = false
				return
			}

			try {
				const session = await ensureSession()
				this.session = session
			} catch (error) {
				this.session = getSessionSnapshot()
			} finally {
				this.isLoading = false
			}
		},
		async handlePrimaryAction() {
			if (this.errorMessage || this.isInviteUnavailable) {
				this.goHome()
				return
			}

			if (this.hasAccepted || this.alreadyMember) {
				this.enterKitchen()
				return
			}

			await this.acceptCurrentInvite()
		},
		async acceptCurrentInvite() {
			if (this.isAccepting) return

			this.isAccepting = true
			uni.showLoading({
				title: '加入中',
				mask: true
			})

			try {
				await ensureSession()
				const result = this.code ? await acceptInviteByCode(this.code) : await acceptInvite(this.token)
				this.invite = result?.invite || this.invite
				this.session = updateSessionKitchens({
					kitchens: result?.kitchens,
					currentKitchenId: result?.currentKitchenId
				})
				this.hasAccepted = true
				uni.showToast({
					title: result?.alreadyMember ? '已经在这个空间里了' : '已加入空间',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: getFriendlySessionErrorMessage(error) || '加入失败',
					icon: 'none'
				})
			} finally {
				this.isAccepting = false
				uni.hideLoading()
			}
		},
		enterKitchen() {
			if (this.invite?.kitchenId) {
				setCurrentKitchenId(this.invite.kitchenId)
			}

			uni.reLaunch({
				url: '/pages/index/index?section=kitchen'
			})
		},
		goHome() {
			uni.reLaunch({
				url: '/pages/index/index'
			})
		},
		formatDateTime(value = '') {
			if (!value) return '--'
			const normalized = value.replace('T', ' ').replace(/\+\d{2}:\d{2}$/, '')
			return normalized.slice(0, 16)
		}
	}
}
</script>

<style lang="scss" scoped>
	.invite-page {
		min-height: 100vh;
		padding: 28rpx 24rpx 40rpx;
		box-sizing: border-box;
		background:
			radial-gradient(circle at top left, rgba(255, 244, 228, 0.88) 0, rgba(255, 244, 228, 0) 44%),
			linear-gradient(180deg, #f8f4ee 0%, #f2ede6 100%);
	}

	.invite-shell {
		min-height: calc(100vh - 68rpx);
		display: flex;
		align-items: center;
	}

	.invite-card {
		width: 100%;
		padding: 34rpx 30rpx 32rpx;
		border-radius: 32rpx;
		background: rgba(255, 255, 255, 0.96);
		box-shadow: 0 18rpx 36rpx rgba(61, 48, 34, 0.08);
	}

	.invite-card--loading {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 16rpx;
	}

	.invite-badge {
		display: inline-flex;
		align-items: center;
		padding: 10rpx 18rpx;
		border-radius: 999rpx;
		background: rgba(91, 74, 59, 0.08);
		font-size: 21rpx;
		font-weight: 600;
		color: #6f5f51;
	}

	.invite-title {
		display: block;
		margin-top: 24rpx;
		font-size: 42rpx;
		font-weight: 700;
		line-height: 1.28;
		color: #2f2923;
	}

	.invite-description,
	.invite-loading__desc {
		display: block;
		margin-top: 14rpx;
		font-size: 25rpx;
		line-height: 1.7;
		color: #7d7268;
	}

	.invite-loading__title {
		font-size: 34rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.invite-summary {
		margin-top: 28rpx;
		padding: 8rpx 0;
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.invite-summary__item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 20rpx;
	}

	.invite-summary__label {
		font-size: 23rpx;
		color: #90857a;
	}

	.invite-summary__value {
		flex: 1;
		text-align: right;
		font-size: 24rpx;
		font-weight: 600;
		color: #3f3730;
	}

	.invite-summary__value--code {
		font-family: 'SF Mono', 'Menlo', monospace;
		letter-spacing: 2rpx;
	}

	.invite-notice {
		margin-top: 28rpx;
		padding: 18rpx 20rpx;
		border-radius: 22rpx;
		background: #faf4ed;
		display: flex;
		align-items: flex-start;
		gap: 12rpx;
	}

	.invite-notice--success {
		background: #eef5ec;
	}

	.invite-notice__text {
		flex: 1;
		font-size: 23rpx;
		line-height: 1.65;
		color: #6f6458;
	}

	.invite-actions {
		margin-top: 30rpx;
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.invite-actions--stack {
		margin-top: 34rpx;
	}

	.invite-action {
		padding: 22rpx 26rpx;
		border-radius: 20rpx;
		background: #f3eee8;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.invite-action--primary {
		background: #3e342c;
	}

	.invite-action__text {
		font-size: 26rpx;
		font-weight: 700;
		color: #5d5247;
	}

	.invite-action__text--primary {
		color: #ffffff;
	}
</style>
