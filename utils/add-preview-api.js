import { request } from './http'

export function previewAddLink(kitchenId, payload = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/add-link-previews`,
		method: 'POST',
		data: {
			text: payload.text || '',
			city: payload.city || '',
			latitude: Number(payload.latitude) || 0,
			longitude: Number(payload.longitude) || 0,
			limit: Number(payload.limit) || 3
		},
		timeout: 20000
	}).then((data) => data?.result || null)
}
