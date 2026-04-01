import { MAX_RECENT_SEARCHES } from './constants'

const recentSearchStorageKey = 'caipu-miniapp-recent-searches'
const lastDraftLinkPrefillStorageKey = 'caipu-miniapp-last-draft-link-prefill'

export function readRecentSearches() {
	try {
		const stored = uni.getStorageSync(recentSearchStorageKey)
		if (!Array.isArray(stored)) return []
		return stored
			.map((item) => String(item || '').trim())
			.filter(Boolean)
			.slice(0, MAX_RECENT_SEARCHES)
	} catch (error) {
		return []
	}
}

export function writeRecentSearches(items = []) {
	try {
		uni.setStorageSync(recentSearchStorageKey, items)
	} catch (error) {
		// Ignore storage write failures and keep search usable.
	}
}

export function readLastDraftLinkPrefill() {
	try {
		return String(uni.getStorageSync(lastDraftLinkPrefillStorageKey) || '').trim()
	} catch (error) {
		return ''
	}
}

export function writeLastDraftLinkPrefill(value = '') {
	try {
		uni.setStorageSync(lastDraftLinkPrefillStorageKey, String(value || '').trim())
	} catch (error) {
		// Ignore storage write failures and keep prefill usable.
	}
}
