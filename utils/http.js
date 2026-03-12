import { appConfig, resolveAPIURL } from './app-config'
import { getAccessToken } from './session-storage'

function parseResponseData(data) {
	if (typeof data !== 'string') return data
	try {
		return JSON.parse(data)
	} catch (error) {
		return data
	}
}

function createRequestError(message, extras = {}) {
	const error = new Error(message || '请求失败')
	return Object.assign(error, extras)
}

function normalizeFailure(error) {
	if (error instanceof Error) {
		return error
	}

	if (error && typeof error === 'object') {
		return createRequestError(error.message || '请求失败', error)
	}

	return createRequestError('请求失败')
}

export function request(options = {}) {
	const {
		url,
		method = 'GET',
		data,
		header = {},
		auth = true,
		timeout = appConfig.requestTimeout
	} = options

	return new Promise((resolve, reject) => {
		const token = auth ? getAccessToken() : ''
		const requestHeader = {
			...header
		}

		if (auth && token) {
			requestHeader.Authorization = `Bearer ${token}`
		}

		if (!requestHeader['Content-Type'] && method !== 'GET') {
			requestHeader['Content-Type'] = 'application/json'
		}

		uni.request({
			url: resolveAPIURL(url),
			method,
			data,
			header: requestHeader,
			timeout,
			success: (response) => {
				const payload = parseResponseData(response.data)
				const statusCode = response.statusCode || 0

				if (payload && typeof payload === 'object' && payload.code === 0) {
					resolve(payload.data)
					return
				}

				reject(
					createRequestError(payload?.message || `请求失败 (${statusCode})`, {
						code: payload?.code || statusCode || 50000,
						statusCode,
						payload
					})
				)
			},
			fail: (error) => {
				reject(
					createRequestError(error?.errMsg || '网络请求失败', {
						code: 50000,
						statusCode: 0,
						originalError: error
					})
				)
			}
		})
	}).catch((error) => Promise.reject(normalizeFailure(error)))
}

export function uploadFile(options = {}) {
	const {
		url,
		filePath,
		name = 'file',
		formData = {},
		header = {},
		auth = true,
		timeout = appConfig.requestTimeout
	} = options

	return new Promise((resolve, reject) => {
		const token = auth ? getAccessToken() : ''
		const requestHeader = {
			...header
		}

		if (auth && token) {
			requestHeader.Authorization = `Bearer ${token}`
		}

		uni.uploadFile({
			url: resolveAPIURL(url),
			filePath,
			name,
			formData,
			header: requestHeader,
			timeout,
			success: (response) => {
				const payload = parseResponseData(response.data)
				const statusCode = response.statusCode || 0

				if (payload && typeof payload === 'object' && payload.code === 0) {
					resolve(payload.data)
					return
				}

				reject(
					createRequestError(payload?.message || `上传失败 (${statusCode})`, {
						code: payload?.code || statusCode || 50000,
						statusCode,
						payload
					})
				)
			},
			fail: (error) => {
				reject(
					createRequestError(error?.errMsg || '上传失败', {
						code: 50000,
						statusCode: 0,
						originalError: error
					})
				)
			}
		})
	}).catch((error) => Promise.reject(normalizeFailure(error)))
}
