const apiBaseURL = 'https://www.gxm1227.top'
const serverRouteMappings = [
	['/api/', '/caipu-api/'],
	['/uploads/', '/caipu-uploads/']
]
const exactServerRouteMappings = new Map([['/healthz', '/caipu-healthz']])

const requestedAuthMode = 'wechat'

function resolveAuthMode(mode, baseURL) {
	if (mode === 'dev' || mode === 'wechat') {
		return mode
	}

	return /127\.0\.0\.1|localhost/.test(baseURL) ? 'dev' : 'wechat'
}

function isAbsoluteHTTPURL(value = '') {
	return /^https?:\/\//i.test(String(value || '').trim())
}

function extractURLOrigin(value = '') {
	const match = String(value || '').trim().match(/^(https?:\/\/[^/?#]+)/i)
	return match ? match[1] : ''
}

function parseAbsoluteHTTPURL(value = '') {
	const match = String(value || '').trim().match(/^(https?:\/\/[^/?#]+)(\/[^?#]*)?([?#].*)?$/i)
	if (!match) {
		return null
	}

	return {
		origin: match[1],
		pathname: match[2] || '/',
		suffix: match[3] || ''
	}
}

function splitPathSuffix(value = '') {
	const input = String(value || '').trim()
	const suffixIndex = input.search(/[?#]/)
	if (suffixIndex === -1) {
		return {
			pathname: input,
			suffix: ''
		}
	}

	return {
		pathname: input.slice(0, suffixIndex),
		suffix: input.slice(suffixIndex)
	}
}

function normalizeServerPath(value = '') {
	const normalizedInput = String(value || '').trim()
	if (!normalizedInput) return ''

	const { pathname, suffix } = splitPathSuffix(normalizedInput)
	const exactMatch = exactServerRouteMappings.get(pathname)
	if (exactMatch) {
		return `${exactMatch}${suffix}`
	}

	for (const [legacyPrefix, nextPrefix] of serverRouteMappings) {
		if (pathname.startsWith(legacyPrefix)) {
			return `${nextPrefix}${pathname.slice(legacyPrefix.length)}${suffix}`
		}
	}

	return `${pathname}${suffix}`
}

const apiBaseOrigin = extractURLOrigin(apiBaseURL)

export const appConfig = {
	apiBaseURL,
	authMode: resolveAuthMode(requestedAuthMode, apiBaseURL),
	authModeSetting: requestedAuthMode,
	devLoginIdentity: 'alice',
	devLoginIdentityMode: 'fixed',
	requestTimeout: 15000,
	inviteShareEnabled: true,
	// 例如：京ICP备2024000000号A
	miniProgramFilingNumber: '粤ICP备2026023717号-2X',
	miniProgramFilingSystemURL: 'https://beian.miit.gov.cn/'
}

export function isServerAssetPath(path = '') {
	const value = String(path || '').trim()
	if (!value) return false

	if (isAbsoluteHTTPURL(value)) {
		const target = parseAbsoluteHTTPURL(value)
		return !!target && target.origin === apiBaseOrigin && isServerAssetPath(target.pathname)
	}

	const normalizedPath = value.startsWith('/') ? value : `/${value}`
	const { pathname } = splitPathSuffix(normalizedPath)
	return pathname.startsWith('/uploads/') || pathname.startsWith('/caipu-uploads/')
}

export function resolveServerURL(path = '') {
	const value = String(path || '').trim()
	if (!value) return appConfig.apiBaseURL

	if (isAbsoluteHTTPURL(value)) {
		const target = parseAbsoluteHTTPURL(value)
		if (!target) {
			return value
		}
		if (target.origin !== apiBaseOrigin) {
			return value
		}
		const normalizedPath = normalizeServerPath(`${target.pathname}${target.suffix}`)
		return `${target.origin}${normalizedPath}`
	}

	const normalizedPath = normalizeServerPath(value.startsWith('/') ? value : `/${value}`)
	return `${appConfig.apiBaseURL}${normalizedPath}`
}

export function resolveAPIURL(path = '') {
	return resolveServerURL(path)
}

export function resolveAssetURL(path = '') {
	if (!path) return ''
	return resolveServerURL(path)
}
