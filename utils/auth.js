import { appConfig, resolveAssetURL } from './app-config'
import { request } from './http'
import { clearSessionState, getAccessToken, getSessionState, setAccessToken, setSessionState } from './session-storage'
import { isTemporaryImagePath } from './upload-api'

let pendingSessionPromise = null
const DEV_IDENTITY_STORAGE_KEY = 'caipu-miniapp-dev-identity'
const FALLBACK_NICKNAME_PREFIX = '厨友'
const PLACEHOLDER_NICKNAMES = new Set(['微信用户', 'wechat user'])

function normalizeKitchen(kitchen = {}) {
	return {
		id: Number(kitchen.id) || 0,
		name: kitchen.name || '我的厨房',
		role: kitchen.role || 'member'
	}
}

function normalizeUser(user = null) {
	if (!user || typeof user !== 'object') {
		return null
	}

	return {
		...user,
		avatarUrl: resolveAssetURL(user.avatarUrl || '')
	}
}

function normalizeSessionPayload(payload = {}, tokenOverride = '') {
	const previous = getSessionState()
	const kitchens = Array.isArray(payload.kitchens) ? payload.kitchens.map((item) => normalizeKitchen(item)) : []
	const preferredKitchenId =
		Number(previous?.currentKitchenId) ||
		Number(payload.currentKitchenId) ||
		Number(kitchens[0]?.id) ||
		0

	const currentKitchenId = kitchens.some((item) => item.id === preferredKitchenId)
		? preferredKitchenId
		: Number(kitchens[0]?.id) || 0

	const token = tokenOverride || payload.token || getAccessToken() || ''
	const currentKitchen = kitchens.find((item) => item.id === currentKitchenId) || null

	const session = {
		token,
		user: normalizeUser(payload.user || previous?.user || null),
		kitchens,
		currentKitchenId,
		currentKitchen,
		syncedAt: new Date().toISOString()
	}

	setAccessToken(token)
	setSessionState(session)
	return session
}

function shouldUseDevLogin() {
	return appConfig.authMode === 'dev'
}

function getConfiguredDevLoginIdentity() {
	const value = String(appConfig.devLoginIdentity || '').trim()
	return value || 'demo'
}

function getExpectedDevOpenID() {
	return `dev:${getConfiguredDevLoginIdentity()}`
}

function getDevLoginIdentity() {
	const configuredIdentity = getConfiguredDevLoginIdentity()
	if (appConfig.devLoginIdentityMode === 'fixed') {
		uni.setStorageSync(DEV_IDENTITY_STORAGE_KEY, configuredIdentity)
		return configuredIdentity
	}

	const storedIdentity = uni.getStorageSync(DEV_IDENTITY_STORAGE_KEY)
	if (storedIdentity) {
		return storedIdentity
	}

	const generatedIdentity = `${configuredIdentity}-${Math.random().toString(36).slice(2, 8)}`
	uni.setStorageSync(DEV_IDENTITY_STORAGE_KEY, generatedIdentity)
	return generatedIdentity
}

function shouldRefreshDevSession(payload = {}) {
	if (!shouldUseDevLogin() || appConfig.devLoginIdentityMode !== 'fixed') {
		return false
	}

	const actualOpenID = String(payload?.user?.openid || '').trim()
	if (!actualOpenID) {
		return false
	}

	return actualOpenID !== getExpectedDevOpenID()
}

function loginWithWeChat() {
	return new Promise((resolve, reject) => {
		uni.login({
			provider: 'weixin',
			success: (result) => {
				if (!result?.code) {
					reject(new Error('微信登录失败'))
					return
				}
				resolve(result.code)
			},
			fail: (error) => {
				reject(new Error(error?.errMsg || '微信登录失败'))
			}
		})
	})
}

function getMiniProgramAppID() {
	if (typeof uni.getAccountInfoSync !== 'function') {
		return ''
	}

	try {
		const accountInfo = uni.getAccountInfoSync()
		return String(accountInfo?.miniProgram?.appId || '').trim()
	} catch (error) {
		return ''
	}
}

function getUserProfileFromPayload(payload = {}) {
	const userInfo = payload?.userInfo || payload
	const avatarUrl = String(userInfo?.avatarUrl || userInfo?.avatarURL || '').trim()
	return {
		nickname: String(userInfo?.nickName || userInfo?.nickname || '').trim(),
		avatarUrl: isTemporaryImagePath(avatarUrl) ? '' : avatarUrl
	}
}

function getOptionalWeChatProfile() {
	return new Promise((resolve) => {
		if (typeof uni.getUserInfo !== 'function') {
			resolve({})
			return
		}

		uni.getUserInfo({
			provider: 'weixin',
			success: (result) => {
				resolve(getUserProfileFromPayload(result))
			},
			fail: () => {
				resolve({})
			}
		})
	})
}

function isFallbackNickname(value = '') {
	return String(value).trim().startsWith(FALLBACK_NICKNAME_PREFIX)
}

export function isPlaceholderNickname(value = '') {
	const nickname = String(value).trim()
	if (!nickname) return true
	return isFallbackNickname(nickname) || PLACEHOLDER_NICKNAMES.has(nickname.toLowerCase())
}

export function isProfileIncomplete(user = {}) {
	const nickname = String(user?.nickname || '').trim()
	const avatarUrl = String(user?.avatarUrl || '').trim()
	return !avatarUrl || isPlaceholderNickname(nickname)
}

function shouldSyncUserProfile(currentUser = {}, profile = {}) {
	const nickname = String(profile.nickname || '').trim()
	const avatarUrl = String(profile.avatarUrl || '').trim()
	if (!nickname && !avatarUrl) {
		return false
	}

	const currentNickname = String(currentUser?.nickname || '').trim()
	const currentAvatarUrl = String(currentUser?.avatarUrl || '').trim()
	if (nickname && !isPlaceholderNickname(nickname) && (!currentNickname || isPlaceholderNickname(currentNickname))) {
		return true
	}
	if (avatarUrl && currentAvatarUrl !== avatarUrl) {
		return true
	}
	return false
}

function isLegacyProfileRequestError(error) {
	const message = String(error?.message || '').trim().toLowerCase()
	const statusCode = Number(error?.statusCode) || 0
	return statusCode === 404 || statusCode === 405 || message === 'invalid request body' || message === 'request body is required'
}

async function updateSessionUserProfile(profile = {}) {
	try {
		const payload = await request({
			url: '/caipu-api/auth/profile',
			method: 'PATCH',
			data: {
				nickname: profile.nickname || '',
				avatarUrl: profile.avatarUrl || ''
			}
		})

		return payload?.user || null
	} catch (error) {
		if (isLegacyProfileRequestError(error)) {
			return null
		}
		throw error
	}
}

export async function saveCurrentUserProfile(profile = {}) {
	const user = await updateSessionUserProfile(profile)
	if (!user) return null
	updateSessionUser(user)
	return user
}

async function createSession() {
	if (shouldUseDevLogin()) {
		const payload = await request({
			url: '/caipu-api/auth/dev-login',
			method: 'POST',
			data: {
				identity: getDevLoginIdentity()
			},
			auth: false
		})

		return normalizeSessionPayload(payload, payload?.token || '')
	}

	const code = await loginWithWeChat()
	const profile = await getOptionalWeChatProfile()
	let payload
	try {
		payload = await request({
			url: '/caipu-api/auth/wechat/login',
			method: 'POST',
			data: {
				code,
				appId: getMiniProgramAppID(),
				nickname: profile.nickname || '',
				avatarUrl: profile.avatarUrl || ''
			},
			auth: false
		})
	} catch (error) {
		if (!isLegacyProfileRequestError(error)) {
			throw error
		}

		payload = await request({
			url: '/caipu-api/auth/wechat/login',
			method: 'POST',
			data: {
				code,
				appId: getMiniProgramAppID()
			},
			auth: false
		})
	}

	return normalizeSessionPayload(payload, payload?.token || '')
}

export async function ensureSession(options = {}) {
	const { force = false } = options
	if (pendingSessionPromise) {
		return pendingSessionPromise
	}

	pendingSessionPromise = (async () => {
		if (!force) {
			const storedToken = getAccessToken()
			if (storedToken) {
				try {
					let payload = await request({
						url: '/caipu-api/auth/me',
						method: 'GET'
					})
					let profileSynced = false
					const profile = await getOptionalWeChatProfile()
					try {
						if (shouldSyncUserProfile(payload?.user, profile)) {
							const user = await updateSessionUserProfile(profile)
							if (user) {
								payload.user = user
								profileSynced = true
							}
						}
					} catch (error) {
						if (!isLegacyProfileRequestError(error)) {
							throw error
						}
					}
					if (profileSynced) {
						try {
							payload = await request({
								url: '/caipu-api/auth/me',
								method: 'GET'
							})
						} catch (error) {
							// Keep the refreshed user payload even if this follow-up session sync fails.
						}
					}
					if (shouldRefreshDevSession(payload)) {
						clearSessionState()
						uni.setStorageSync(DEV_IDENTITY_STORAGE_KEY, getConfiguredDevLoginIdentity())
						return createSession()
					}
					return normalizeSessionPayload(payload, storedToken)
				} catch (error) {
					clearSessionState()
				}
			}
		}

		return createSession()
	})()

	try {
		return await pendingSessionPromise
	} finally {
		pendingSessionPromise = null
	}
}

export function getSessionSnapshot() {
	const session = getSessionState()
	if (!session) return null

	const kitchens = Array.isArray(session.kitchens) ? session.kitchens.map((item) => normalizeKitchen(item)) : []
	const currentKitchenId = Number(session.currentKitchenId) || Number(kitchens[0]?.id) || 0
	const currentKitchen = kitchens.find((item) => item.id === currentKitchenId) || null

	return {
		...session,
		user: normalizeUser(session.user),
		kitchens,
		currentKitchenId,
		currentKitchen,
		token: getAccessToken() || session.token || ''
	}
}

export function getCurrentKitchenId() {
	return Number(getSessionSnapshot()?.currentKitchenId) || 0
}

export function getCurrentKitchen() {
	return getSessionSnapshot()?.currentKitchen || null
}

export function setCurrentKitchenId(kitchenId) {
	const session = getSessionSnapshot()
	if (!session) return null

	const value = Number(kitchenId) || 0
	const currentKitchen = session.kitchens.find((item) => item.id === value)
	if (!currentKitchen) {
		return session
	}

	const nextSession = {
		...session,
		currentKitchenId: currentKitchen.id,
		currentKitchen,
		syncedAt: new Date().toISOString()
	}
	setSessionState(nextSession)
	return nextSession
}

export function updateSessionKitchens(payload = {}) {
	const session = getSessionSnapshot()
	const kitchens = Array.isArray(payload.kitchens) ? payload.kitchens.map((item) => normalizeKitchen(item)) : session?.kitchens || []
	const nextKitchenID = Number(payload.currentKitchenId) || Number(session?.currentKitchenId) || Number(kitchens[0]?.id) || 0
	const currentKitchen = kitchens.find((item) => item.id === nextKitchenID) || kitchens[0] || null

	const nextSession = {
		...(session || {}),
		kitchens,
		currentKitchenId: currentKitchen?.id || 0,
		currentKitchen,
		syncedAt: new Date().toISOString()
	}

	setSessionState(nextSession)
	return nextSession
}

export function updateSessionKitchen(kitchen = {}) {
	const session = getSessionSnapshot()
	if (!session) return null

	const targetKitchenId = Number(kitchen.id) || 0
	if (!targetKitchenId) {
		return session
	}

	let found = false
	const kitchens = (session.kitchens || []).map((item) => {
		if (Number(item.id) !== targetKitchenId) {
			return normalizeKitchen(item)
		}

		found = true
		return normalizeKitchen({
			...item,
			...kitchen
		})
	})

	if (!found) {
		return session
	}

	const currentKitchenId = Number(session.currentKitchenId) || 0
	const currentKitchen = kitchens.find((item) => item.id === currentKitchenId) || null
	const nextSession = {
		...session,
		kitchens,
		currentKitchen,
		syncedAt: new Date().toISOString()
	}

	setSessionState(nextSession)
	return nextSession
}

export function updateSessionUser(user = {}) {
	const session = getSessionSnapshot()
	const nextSession = {
		...(session || {}),
		user: normalizeUser({
			...(session?.user || {}),
			...(user || {})
		}),
		syncedAt: new Date().toISOString()
	}

	setSessionState(nextSession)
	return nextSession
}

export function getFriendlySessionErrorMessage(error) {
	const message = String(error?.message || '').trim()
	if (!message) {
		return '暂时无法登录，请稍后再试'
	}
	if (message === 'wechat login is not configured') {
		return '微信登录尚未配置'
	}
	if (message === 'mini program appId does not match backend wechat config') {
		return '小程序 AppID 与后端配置不一致'
	}
	if (message === 'wechat login failed') {
		return '微信登录失败，请稍后重试'
	}
	if (message === '网络请求失败') {
		return '暂时无法连接服务器'
	}
	return message
}

export function clearSession() {
	clearSessionState()
}
