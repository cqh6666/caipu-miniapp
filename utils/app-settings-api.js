import { request } from './http'

export function getBilibiliSessionSetting() {
	return request({
		url: '/api/app-settings/bilibili-session',
		method: 'GET'
	}).then((data) => data?.setting || null)
}

export function updateBilibiliSessionSetting(payload = {}) {
	return request({
		url: '/api/app-settings/bilibili-session',
		method: 'PUT',
		data: payload
	}).then((data) => data?.setting || null)
}

export function clearBilibiliSessionSetting() {
	return request({
		url: '/api/app-settings/bilibili-session',
		method: 'DELETE'
	}).then((data) => data?.setting || null)
}
