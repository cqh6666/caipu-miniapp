import { request } from './http'

export function createKitchenInvite(kitchenId, payload = {}) {
	return request({
		url: `/api/kitchens/${kitchenId}/invites`,
		method: 'POST',
		data: payload
	}).then((data) => data?.invite || null)
}

export function previewInvite(token) {
	return request({
		url: `/api/invites/${token}`,
		method: 'GET',
		auth: false
	}).then((data) => data?.invite || null)
}

export function acceptInvite(token) {
	return request({
		url: `/api/invites/${token}/accept`,
		method: 'POST',
		data: {}
	})
}
