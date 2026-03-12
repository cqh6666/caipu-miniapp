const apiBaseURL = 'http://127.0.0.1:8080'

const autoAuthMode = /127\.0\.0\.1|localhost/.test(apiBaseURL) ? 'dev' : 'wechat'

export const appConfig = {
	apiBaseURL,
	authMode: autoAuthMode,
	devLoginIdentity: 'alice',
	devLoginIdentityMode: 'fixed',
	requestTimeout: 15000
}

export function resolveAPIURL(path = '') {
	if (!path) return appConfig.apiBaseURL
	if (/^https?:\/\//.test(path)) return path
	return `${appConfig.apiBaseURL}${path.startsWith('/') ? path : `/${path}`}`
}
