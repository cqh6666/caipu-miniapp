import { request } from './http'

export function listPlaces(kitchenId, filters = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/places`,
		method: 'GET',
		data: filters
	}).then((data) => data?.items || [])
}

export function createPlace(kitchenId, payload) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/places`,
		method: 'POST',
		data: payload
	}).then((data) => data?.place || null)
}

export function getPlaceDetail(placeId) {
	return request({
		url: `/caipu-api/places/${placeId}`,
		method: 'GET'
	}).then((data) => data?.place || null)
}

export function updatePlace(placeId, payload) {
	return request({
		url: `/caipu-api/places/${placeId}`,
		method: 'PUT',
		data: payload
	}).then((data) => data?.place || null)
}

export function updatePlaceStatus(placeId, status) {
	return request({
		url: `/caipu-api/places/${placeId}/status`,
		method: 'PATCH',
		data: { status }
	}).then((data) => data?.place || null)
}

export function deletePlace(placeId) {
	return request({
		url: `/caipu-api/places/${placeId}`,
		method: 'DELETE'
	})
}
