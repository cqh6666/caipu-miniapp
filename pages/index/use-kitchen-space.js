import { createKitchenInvite, formatInviteCode, leaveKitchen, listKitchenMembers, normalizeInviteCode, updateKitchen } from '../../utils/kitchen-api'
import { buildSpaceStats } from '../../utils/space-stats'
import { getKitchenStats, normalizeStatsWindow } from '../../utils/space-stats-api'
import { setSpaceStatsContext } from '../../utils/space-stats-bridge'
import {
	ensureSession,
	getCurrentKitchenId,
	getFriendlySessionErrorMessage,
	getSessionSnapshot,
	isPlaceholderNickname,
	saveCurrentUserProfile,
	setCurrentKitchenId,
	updateSessionKitchen
} from '../../utils/auth'
import { getAccessToken } from '../../utils/session-storage'
import { ensureUploadedImage } from '../../utils/upload-api'
import { appConfig } from '../../utils/app-config'
import { defineIndexPageModule } from './page-module'

const inviteShareFallbackImageUrl = '/static/invite-share-cover.png'

export function replaceKitchenLabel(value = '') {
	return String(value || '').replace(/厨房/g, '空间')
}

function shortenText(value = '', maxLength = 10) {
	const text = String(value || '').trim()
	return text.length <= maxLength ? text : `${text.slice(0, maxLength)}...`
}

export function buildInviteShareTitle(invite = {}, fallbackKitchenName = '') {
	const kitchenName = shortenText(
		replaceKitchenLabel(invite?.kitchenName || fallbackKitchenName),
		8
	)
	return kitchenName ? `邀请你加入「${kitchenName}」` : '邀请你加入共享空间'
}

export function buildInviteShareImageURL(invite = {}, now = Date.now()) {
	const raw = String(invite?.shareImageUrl || '').trim()
	if (!raw) return inviteShareFallbackImageUrl
	return `${raw}${raw.includes('?') ? '&' : '?'}ts=${now}`
}

export function memberRoleLabel(role) {
	if (role === 'owner') return '创建者'
	if (role === 'admin') return '管理员'
	return '成员'
}

export function memberDisplayName(member = {}) {
	return member.nickname || `厨友 ${member.userId || ''}`.trim()
}

export function memberInitial(member = {}) {
	return memberDisplayName(member).slice(0, 1)
}

export function memberDescription(member = {}) {
	return member.isCurrentUser ? '你正在维护这个空间。' : '已加入这个共享空间。'
}

export const kitchenSpaceMethods = {
openSpaceStatsPage() {
	// 传给洞察页当前统计快照 + kitchenId（深拷贝，避免跨页响应式引用）。
	let snapshot = {}
	try {
		snapshot = JSON.parse(JSON.stringify(this.spaceStats || {}))
	} catch (error) {
		snapshot = {}
	}
	setSpaceStatsContext({ stats: snapshot, kitchenId: Number(this.currentKitchenId) || 0 })
	uni.navigateTo({ url: '/pages/space-stats/index' })
	// 后台补齐/刷新首页侧统计（卡片与下次进入洞察页时的快照），失败静默回退本地聚合。
	if (!getAccessToken() || !Number(this.currentKitchenId) || this.isRefreshingSpaceStats) return
	if (!this.hasSpaceStatsContent && this.spaceStatsAutoSyncKitchenId !== Number(this.currentKitchenId)) {
		this.spaceStatsAutoSyncKitchenId = Number(this.currentKitchenId)
		this.refreshSpaceStats({ silent: true })
	} else if (!this.spaceStatsRemote) {
		this.isRefreshingSpaceStats = true
		this.loadRemoteSpaceStats(this.spaceStatsWindow, { silent: true }).finally(() => {
			this.isRefreshingSpaceStats = false
		})
	}
},
async loadRemoteSpaceStats(window = this.spaceStatsWindow, options = {}) {
	// 仅负责拉取后端 stats 并落地，不管理 isRefreshingSpaceStats（由调用方统一管理 loading）。
	const { silent = true } = options
	const kitchenId = Number(this.currentKitchenId) || 0
	const normalized = normalizeStatsWindow(window)
	if (!kitchenId || !getAccessToken()) {
		this.spaceStatsRemote = null
		this.spaceStatsRemoteKitchenId = 0
		this.spaceStatsRemoteError = '当前登录态或空间信息未就绪'
		return false
	}
	try {
		const remote = await getKitchenStats(kitchenId, { window: normalized })
		// 防止请求期间空间已切换。
		if (Number(this.currentKitchenId) === kitchenId) {
			this.spaceStatsRemote = remote
			this.spaceStatsRemoteKitchenId = kitchenId
			this.spaceStatsWindow = normalized
			this.spaceStatsRemoteError = ''
		}
		return true
	} catch (error) {
		// 后端 stats 不可用：保留已有远端数据（窗口切换失败时不清空），无远端数据则走本地聚合。
		this.spaceStatsRemoteError = error?.message || '后端统计暂不可用'
		console.warn('[space-stats] remote stats unavailable', error)
		if (!silent) {
			uni.showToast({
				title: '后端统计暂不可用，已显示本地聚合',
				icon: 'none'
			})
		}
		return false
	}
},
async refreshSpaceStats(options = {}) {
	const { silent = true } = options
	if (this.isRefreshingSpaceStats) return
	this.isRefreshingSpaceStats = true
	this.spaceStatsRemoteError = ''
	try {
		// refreshRecipes 内部已串起 recipes / places / members 同步，并负责设置 spaceStatsSyncedAt。
		await this.refreshRecipes({ silent })
		await this.loadMealOrderStore({ silent })
		await this.loadRemoteSpaceStats(this.spaceStatsWindow, { silent })
	} finally {
		this.isRefreshingSpaceStats = false
	}
},
handleSpaceStatsAction(payload = {}) {
	const actionType = payload?.actionType || ''
	switch (actionType) {
		case 'open-add-recipe':
			this.activeSection = 'library'
			this.switchAppMode('cook')
			this.openAddSheet()
			break
		case 'view-wishlist-recipes':
			this.activeSection = 'library'
			this.switchAppMode('cook')
			this.activeStatus = 'wishlist'
			break
		case 'view-done-recipes':
			this.activeSection = 'library'
			this.switchAppMode('cook')
			this.activeStatus = 'done'
			break
		case 'view-want-places':
			this.activeSection = 'library'
			this.switchAppMode('explore')
			this.activePlaceStatus = 'want'
			break
		case 'view-visited-places':
			this.activeSection = 'library'
			this.switchAppMode('explore')
			this.activePlaceStatus = 'visited'
			break
		case 'view-missing-location-places':
			this.activeSection = 'library'
			this.switchAppMode('explore')
			this.activePlaceStatus = 'all'
			break
		case 'view-draft-meal-plan':
			this.activeSection = 'library'
			this.switchAppMode('cook')
			this.openMealOrderDateSheet()
			break
		case 'view-place-detail':
			if (payload?.placeId) {
				this.activeSection = 'library'
				this.switchAppMode('explore')
				this.handlePlaceOpen(payload.placeId)
			}
			break
		default:
			break
	}
},
memberRoleLabel(role) {
	return formatMemberRoleLabel(role)
},
memberDisplayName(member = {}) {
	return formatMemberDisplayName(member)
},
memberInitial(member = {}) {
	return formatMemberInitial(member)
},
memberMemberDescription(member = {}) {
	return memberDescription(member)
},
handleMemberCardTap(member = {}) {
	if (!member.isCurrentUser || !this.currentUser?.id) return
	this.openProfileSheetWithMode('edit')
},
confirmLeaveCurrentKitchen() {
	if (!this.canLeaveCurrentKitchen || this.isLeavingKitchen) return

	const kitchenName = replaceKitchenLabel(this.currentKitchenName || '当前空间')
	uni.showModal({
		title: '退出当前空间',
		content: `退出后你将无法查看「${kitchenName}」里的菜谱和菜单，重新加入需要成员邀请。空间数据不会被删除。`,
		cancelText: '再想想',
		confirmText: '退出空间',
		confirmColor: '#a95549',
		success: async ({ confirm }) => {
			if (!confirm) return
			await this.leaveCurrentKitchen()
		}
	})
},
async leaveCurrentKitchen() {
	if (this.isLeavingKitchen) return

	const kitchenId = Number(getCurrentKitchenId()) || 0
	if (!kitchenId) {
		uni.showToast({
			title: '暂时没有可退出的空间',
			icon: 'none'
		})
		return
	}

	this.isLeavingKitchen = true
	try {
		if (typeof uni.vibrateShort === 'function') {
			uni.vibrateShort({ type: 'light' })
		}
	} catch (error) {
		// 轻触感是增强反馈，不影响退出流程。
	}

	try {
		await leaveKitchen(kitchenId)
		const session = await ensureSession()
		this.applySession(session)
		this.selectedRecipeId = ''
		this.searchKeyword = ''
		await this.refreshRecipes({ silent: false })
		uni.showToast({
			title: '已退出空间',
			icon: 'none'
		})
	} catch (error) {
		const message = String(error?.message || '')
		uni.showToast({
			title: message.includes('owner') ? '创建者暂不能退出空间' : getFriendlySessionErrorMessage(error) || '退出失败',
			icon: 'none'
		})
	} finally {
		this.isLeavingKitchen = false
	}
},
openAboutPage() {
	uni.navigateTo({
		url: '/pages/about/index'
	})
},
openProfileSheetWithMode(mode = 'prompt') {
	this.profileSheetMode = mode === 'edit' ? 'edit' : 'prompt'
	this.profileDraft = {
		nickname: !isPlaceholderNickname(this.currentUser?.nickname) ? this.currentUser.nickname : '',
		avatarUrl: ''
	}
	this.showProfileSheet = true
},
resetProfileDraft() {
	this.profileDraft = {
		nickname: '',
		avatarUrl: ''
	}
},
closeProfileSheet() {
	this.showProfileSheet = false
	this.profileSheetMode = 'prompt'
	this.resetProfileDraft()
},
handleChooseAvatar(event) {
	const avatarUrl = String(event?.detail?.avatarUrl || '').trim()
	if (!avatarUrl) return
	this.profileDraft.avatarUrl = avatarUrl
},
handleProfileNicknameInput(event) {
	this.profileDraft.nickname = String(event?.detail?.value || '').trim()
},
async submitProfile(event) {
	if (this.isSubmittingProfile || !this.canSubmitProfile) return

	const submittedNickname = String(event?.detail?.value?.nickname || this.profileDraft.nickname || '').trim()
	this.isSubmittingProfile = true

	try {
		const session = await ensureSession()
		this.applySession(session)

		const avatarUrl = await ensureUploadedImage(this.profileDraft.avatarUrl)
		const user = await saveCurrentUserProfile({
			nickname: submittedNickname,
			avatarUrl
		})
		if (!user) {
			throw new Error('当前后端暂不支持保存资料')
		}
		let nextSession = null
		try {
			nextSession = await ensureSession()
		} catch (error) {
			// Keep the saved profile result even if the follow-up session refresh fails.
		}
		this.showProfileSheet = false
		this.profileSheetMode = 'prompt'
		this.resetProfileDraft()
		this.applySession(nextSession || getSessionSnapshot())
		await this.refreshKitchenMembers({ silent: true })
		uni.showToast({
			title: '资料已更新',
			icon: 'none'
		})
	} catch (error) {
		uni.showToast({
			title: error?.message || '保存资料失败',
			icon: 'none'
		})
	} finally {
		this.isSubmittingProfile = false
	}
},
async refreshKitchenMembers(options = {}) {
	const { kitchenId = getCurrentKitchenId(), silent = true } = options
	const targetKitchenId = Number(kitchenId) || 0
	if (!targetKitchenId) {
		this.kitchenMembers = []
		this.kitchenMembersKitchenId = 0
		return []
	}

	this.isLoadingKitchenMembers = true

	try {
		const items = await listKitchenMembers(targetKitchenId)
		if (targetKitchenId === getCurrentKitchenId()) {
			this.kitchenMembers = items
			this.kitchenMembersKitchenId = targetKitchenId
		}
		return items
	} catch (error) {
		if (targetKitchenId === getCurrentKitchenId()) {
			this.kitchenMembers = []
			this.kitchenMembersKitchenId = targetKitchenId
		}
		if (!silent) {
			uni.showToast({
				title: error?.message || '获取成员失败',
				icon: 'none'
			})
		}
		return []
	} finally {
		if (targetKitchenId === getCurrentKitchenId()) {
			this.isLoadingKitchenMembers = false
		}
	}
}

}

export const kitchenInviteMethods = {
async openInviteSheet() {
	if (!this.currentKitchenName) {
		await this.refreshRecipes({ silent: false })
	}

	if (!getCurrentKitchenId()) {
		uni.showToast({
			title: '还没拿到空间信息',
			icon: 'none'
		})
		return
	}

	this.showInviteSheet = true
	const canReuseInvite =
		this.activeInvite &&
		Number(this.activeInvite.kitchenId) === Number(getCurrentKitchenId()) &&
		this.activeInvite.status === 'active'
	if (!canReuseInvite) {
		await this.prepareInvite()
	}
},
closeInviteSheet() {
	this.showInviteSheet = false
	this.inviteCodeCopied = false
},
openInviteCodeSheet() {
	this.inviteCodeInput = ''
	this.showInviteCodeSheet = true
},
closeInviteCodeSheet() {
	this.showInviteCodeSheet = false
	this.inviteCodeInput = ''
},
openKitchenNameSheet() {
	if (!getCurrentKitchenId()) {
		uni.showToast({
			title: '还没拿到空间信息',
			icon: 'none'
		})
		return
	}

	this.promptKitchenName()
},
promptKitchenName() {
	if (this.isSubmittingKitchenName) return

	uni.showModal({
		title: '修改空间名',
		editable: true,
		content: this.currentKitchenName || '',
		placeholderText: '输入空间名称',
		confirmText: '保存',
		cancelText: '取消',
		success: async (result) => {
			if (!result?.confirm) return
			const submittedName = String(result?.content || '').trim()
			await this.submitKitchenName(submittedName)
		}
	})
},
async submitKitchenName(submittedName = '') {
	const nextName = String(submittedName || '').trim()
	if (this.isSubmittingKitchenName || !nextName) return

	this.isSubmittingKitchenName = true

	try {
		const kitchen = await updateKitchen(getCurrentKitchenId(), {
			name: nextName
		})
		if (!kitchen) {
			throw new Error('修改空间名失败')
		}

		const currentInvite = this.activeInvite
		const nextSession = updateSessionKitchen(kitchen)
		this.applySession(nextSession)
		if (Number(currentInvite?.kitchenId) === Number(kitchen.id)) {
			this.activeInvite = {
				...currentInvite,
				kitchenName: kitchen.name
			}
		}
		uni.showToast({
			title: '空间名已更新',
			icon: 'none'
		})
	} catch (error) {
		uni.showToast({
			title: error?.message || '修改空间名失败',
			icon: 'none'
		})
	} finally {
		this.isSubmittingKitchenName = false
	}
},
handleInviteCodeInput(event) {
	this.inviteCodeInput = formatInviteCode(event?.detail?.value || '')
},
async prepareInvite() {
	if (this.isPreparingInvite) return

	this.isPreparingInvite = true
	this.inviteCodeCopied = false
	this.activeInvite = null

	try {
		const invite = await createKitchenInvite(getCurrentKitchenId(), {})
		this.activeInvite = invite
	} catch (error) {
		uni.showToast({
			title: error?.message || '生成邀请失败',
			icon: 'none'
		})
	} finally {
		this.isPreparingInvite = false
	}
},
copyInviteCode() {
	if (!this.activeInvite?.code || this.isPreparingInvite) {
		uni.showToast({
			title: '请先生成邀请码',
			icon: 'none'
		})
		return
	}

	uni.setClipboardData({
		data: formatInviteCode(this.activeInvite.code),
		success: () => {
			this.inviteCodeCopied = true
			uni.showToast({
				title: '邀请码已复制',
				icon: 'none'
			})
		}
	})
},
regenerateInviteCode() {
	uni.showModal({
		title: '重新生成邀请码',
		content: '重新生成后，之前发出的邀请码会失效，是否继续？',
		confirmText: '重新生成',
		success: async ({ confirm }) => {
			if (!confirm) return
			await this.prepareInvite()
		}
	})
},
submitInviteCode() {
	const code = normalizeInviteCode(this.inviteCodeInput)
	if (!code) {
		uni.showToast({
			title: '请先输入邀请码',
			icon: 'none'
		})
		return
	}

	this.closeInviteCodeSheet()
	uni.navigateTo({
		url: `/pages/invite/index?code=${encodeURIComponent(code)}`
	})
},
showAllMembers() {
	if (!this.kitchenMembers.length) return

	uni.showActionSheet({
		itemList: this.kitchenMembers.map((member) => {
			const suffix = member.isCurrentUser ? ' · 你' : ''
			return `${this.memberDisplayName(member)} · ${this.memberRoleLabel(member.role)}${suffix}`
		})
	})
},
openKitchenSelector() {
	if (!this.kitchenOptions.length) return
	if (this.kitchenOptions.length <= 1) {
		uni.showToast({
			title: '当前只有一个空间',
			icon: 'none'
		})
		return
	}

	uni.showActionSheet({
		itemList: this.kitchenOptions.map((item) => item.name),
		success: async ({ tapIndex }) => {
			const nextKitchen = this.kitchenOptions[tapIndex]
			if (!nextKitchen || nextKitchen.id === getSessionSnapshot()?.currentKitchenId) return
			setCurrentKitchenId(nextKitchen.id)
			this.applySession()
			this.selectedRecipeId = ''
			this.searchKeyword = ''
			await this.refreshRecipes({ silent: false })
		}
	})
}

}

export const kitchenMemberComputed = {
	inviteActionDescription() {
		return this.showInviteShareAction ? '复制邀请码或直接分享给朋友' : '复制邀请码发给朋友'
	},
	invitePreparingText() {
		return this.showInviteShareAction ? '很快就好，生成后就能直接发给微信好友。' : '很快就好，生成后就能复制邀请码发给朋友。'
	},
	memberPanelSummary() {
		if (!this.currentKitchenName && this.isSyncing) {
			return '同步中'
		}
		if (this.isLoadingKitchenMembers) {
			return '加载中'
		}
		if (!this.kitchenMembers.length) {
			return '等待成员加入'
		}
		return `${this.kitchenMembers.length} 位成员`
	},
	visibleKitchenMembers() {
		return this.kitchenMembers.slice(0, 3)
	},
	hasMoreKitchenMembers() {
		return this.kitchenMembers.length > this.visibleKitchenMembers.length
	}
}

export const kitchenInviteComputed = {
	inviteSheetSubtitle() {
		if (!this.currentKitchenName) {
			return '发给朋友后，对方输入邀请码即可加入。'
		}
		return `邀请朋友加入「${replaceKitchenLabel(this.currentKitchenName)}」`
	},
	showInviteShareAction() {
		return !!appConfig.inviteShareEnabled
	},
	inviteExpiresText() {
		if (!this.activeInvite?.expiresAt) return '--'
		const raw = this.activeInvite.expiresAt.replace(/\+\d{2}:\d{2}$/, '')
		const normalized = raw.includes('T') ? raw : raw.replace(' ', 'T')
		const expiresAt = new Date(normalized)
		if (Number.isNaN(expiresAt.getTime())) {
			return raw.replace('T', ' ').slice(5, 16)
		}
		const month = String(expiresAt.getMonth() + 1).padStart(2, '0')
		const day = String(expiresAt.getDate()).padStart(2, '0')
		const hours = String(expiresAt.getHours()).padStart(2, '0')
		const minutes = String(expiresAt.getMinutes()).padStart(2, '0')
		return `${month}-${day} ${hours}:${minutes}`
	},
	inviteRemainingUsesText() {
		if (!this.activeInvite) return '--'
		return `${this.activeInvite.remainingUses} 人`
	},
	formattedActiveInviteCode() {
		return formatInviteCode(this.activeInvite?.code || '') || '--'
	},
	inviteMetaLine() {
		if (!this.activeInvite) return '--'
		return `${this.inviteRemainingUsesText} 可加入 · ${this.inviteExpiresText} 过期`
	},
	profileAvatarPreview() {
		return this.profileDraft.avatarUrl || this.currentUser?.avatarUrl || ''
	},
	profileSheetTitle() {
		return this.profileSheetMode === 'edit' ? '个人资料' : '完善资料'
	},
	profileSheetSubtitle() {
		return this.profileSheetMode === 'edit'
			? '修改头像和昵称后，空间成员会更容易认出你。'
			: '设置头像和昵称后，空间成员会更容易认出你。'
	},
	profileSheetSecondaryActionText() {
		return this.profileSheetMode === 'edit' ? '取消' : '暂不设置'
	},
	profileAvatarFallback() {
		const name = (this.profileDraft.nickname || this.currentUser?.nickname || '厨友').trim()
		return name.slice(0, 1) || '厨'
	},
	canSubmitProfile() {
		return !!String(this.profileDraft.nickname || '').trim() || !!String(this.profileDraft.avatarUrl || '').trim()
	},
	canSubmitInviteCode() {
		return !!normalizeInviteCode(this.inviteCodeInput)
	}
}

export const kitchenSpaceComputed = {
	canSwitchKitchen() {
		return this.kitchenOptions.length > 1
	},
	isKitchenConnected() {
		return !!this.currentKitchenName
	},
	spaceStats() {
		// V2：优先使用后端 stats（含趋势 / 成员贡献 / 消费统计）；不可用时回退本地聚合。
		const remote = this.spaceStatsRemote
		if (remote && Number(this.spaceStatsRemoteKitchenId) === Number(this.currentKitchenId)) {
			return { ...remote, isSyncing: this.isRefreshingSpaceStats }
		}
		// 本地聚合：以「最近一次同步时间」作为统计快照基准时间，既用于近 30 天窗口聚合，也用于「更新时间」新鲜度展示。
		const syncedTime = this.spaceStatsSyncedAt ? new Date(this.spaceStatsSyncedAt) : new Date()
		return buildSpaceStats({
			recipes: this.recipes,
			places: this.places,
			mealOrderStore: this.mealOrderStore,
			members: this.kitchenMembers,
			now: Number.isNaN(syncedTime.getTime()) ? new Date() : syncedTime,
			source: 'cache',
			isSyncing: this.isRefreshingSpaceStats,
			window: this.spaceStatsWindow
		})
	},
	hasSpaceStatsContent() {
		const overview = this.spaceStats?.overview || {}
		return !!(
			Number(overview.recipeTotal) ||
			Number(overview.placeTotal) ||
			Number(overview.submittedMealPlanDays) ||
			Number(overview.wishlistRecipeTotal) ||
			Number(overview.wantPlaceTotal)
		)
	},
	kitchenConnectionLabel() {
		return this.isKitchenConnected ? '已连接' : '未连接'
	},
	currentKitchenDisplayName() {
		if (this.currentKitchenName) {
			return replaceKitchenLabel(this.currentKitchenName)
		}
		return this.isSyncing ? '正在获取空间信息' : replaceKitchenLabel(this.syncErrorMessage || '暂时无法连接空间')
	},
	currentKitchenRoleLabel() {
		if (this.currentKitchenRole === 'owner') return '创建者'
		if (this.currentKitchenRole === 'admin') return '管理员'
		if (this.currentKitchenRole === 'member') return '成员'
		return ''
	},
	canLeaveCurrentKitchen() {
		return !!this.currentKitchenName && ['admin', 'member'].includes(this.currentKitchenRole)
	},
	currentKitchenMetaText() {
		if (!this.currentKitchenName) {
			return this.isSyncing ? '正在同步空间信息' : replaceKitchenLabel(this.syncErrorMessage || '创建或加入一个空间后，会显示在这里。')
		}

		if (this.canSwitchKitchen) {
			return '点击这张卡片，可以切换到其他空间。'
		}
		return '邀请成员后，大家会看到同一份菜单。'
	}
}

export const kitchenSpaceModule = defineIndexPageModule({
	name: 'kitchen-space',
	requires: [
		'currentKitchenId', 'currentKitchenName', 'currentKitchenRole', 'kitchenMembers',
		'refreshRecipes', 'loadMealOrderStore'
	],
	methods: {
		...kitchenSpaceMethods,
		...kitchenInviteMethods
	},
	computed: {
		...kitchenSpaceComputed,
		...kitchenMemberComputed,
		...kitchenInviteComputed
	}
})
