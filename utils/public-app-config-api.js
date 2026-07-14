import { request } from './http'

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
			dietAssistantEnabled: normalizeBool(features.dietAssistantEnabled, false)
		}
	}
}

export function loadPublicAppConfig() {
	return request({
		url: '/caipu-api/public/app-config',
		method: 'GET',
		auth: false
	}).then((data) => normalizePublicAppConfig(data))
}
