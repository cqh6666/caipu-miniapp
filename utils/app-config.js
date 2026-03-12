const apiBaseURL = 'http://127.0.0.1:8080'

const requestedAuthMode = 'auto'

function resolveAuthMode(mode, baseURL) {
	if (mode === 'dev' || mode === 'wechat') {
		return mode
	}

	return /127\.0\.0\.1|localhost/.test(baseURL) ? 'dev' : 'wechat'
}

export const appConfig = {
	apiBaseURL,
	authMode: resolveAuthMode(requestedAuthMode, apiBaseURL),
	authModeSetting: requestedAuthMode,
	devLoginIdentity: 'alice',
	devLoginIdentityMode: 'fixed',
	requestTimeout: 15000
}

export function resolveAPIURL(path = '') {
	if (!path) return appConfig.apiBaseURL
	if (/^https?:\/\//.test(path)) return path
	return `${appConfig.apiBaseURL}${path.startsWith('/') ? path : `/${path}`}`
}
