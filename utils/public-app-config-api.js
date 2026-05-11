import { request } from './http'

const storageKey = 'caipu.publicAppConfig'

function normalizeBool(value, fallback = false) {
	if (typeof value === 'boolean') return value
	if (typeof value === 'string') {
		const normalized = value.trim().toLowerCase()
		if (normalized === 'true') return true
		if (normalized === 'false') return false
	}
	return fallback
}

export function normalizePublicAppConfig(value = {}) {
	const features = value?.features || {}
	return {
		features: {
			dietAssistantEnabled: normalizeBool(features.dietAssistantEnabled, true)
		}
	}
}

export function readCachedPublicAppConfig() {
	try {
		const cached = uni.getStorageSync(storageKey)
		if (!cached || typeof cached !== 'object') {
			return normalizePublicAppConfig()
		}
		return normalizePublicAppConfig(cached)
	} catch (error) {
		return normalizePublicAppConfig()
	}
}

export function loadPublicAppConfig() {
	return request({
		url: '/caipu-api/public/app-config',
		method: 'GET',
		auth: false
	}).then((data) => {
		const config = normalizePublicAppConfig(data)
		try {
			uni.setStorageSync(storageKey, config)
		} catch (error) {
			// 缓存失败不影响本次配置生效。
		}
		return config
	})
}
