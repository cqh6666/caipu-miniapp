const apiBaseURL = 'https://www.gxm1227.top'

const requestedAuthMode = 'wechat'

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
	requestTimeout: 15000,
	inviteShareEnabled: true,
	// 例如：京ICP备2024000000号A
	miniProgramFilingNumber: '粤ICP备2026023717号-2X',
	miniProgramFilingSystemURL: 'https://beian.miit.gov.cn/'
}

export function resolveAPIURL(path = '') {
	if (!path) return appConfig.apiBaseURL
	if (/^https?:\/\//.test(path)) return path
	return `${appConfig.apiBaseURL}${path.startsWith('/') ? path : `/${path}`}`
}
