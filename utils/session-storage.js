const TOKEN_STORAGE_KEY = 'caipu-miniapp-token'
const SESSION_STORAGE_KEY = 'caipu-miniapp-session'

export function getAccessToken() {
	return uni.getStorageSync(TOKEN_STORAGE_KEY) || ''
}

export function setAccessToken(token = '') {
	if (token) {
		uni.setStorageSync(TOKEN_STORAGE_KEY, token)
		return
	}
	uni.removeStorageSync(TOKEN_STORAGE_KEY)
}

export function getSessionState() {
	const snapshot = uni.getStorageSync(SESSION_STORAGE_KEY)
	return snapshot && typeof snapshot === 'object' ? snapshot : null
}

export function setSessionState(session = null) {
	if (session && typeof session === 'object') {
		uni.setStorageSync(SESSION_STORAGE_KEY, session)
		return
	}
	uni.removeStorageSync(SESSION_STORAGE_KEY)
}

export function clearSessionState() {
	setAccessToken('')
	setSessionState(null)
}
