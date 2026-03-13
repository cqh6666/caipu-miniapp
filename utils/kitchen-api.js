import { request } from './http'

export function normalizeInviteCode(code = '') {
	return String(code).trim().toUpperCase().replace(/[\s-]+/g, '')
}

export function formatInviteCode(code = '') {
	const normalized = normalizeInviteCode(code)
	if (!normalized) return ''
	return normalized.replace(/(.{4})(?=.)/g, '$1-')
}

export function listKitchenMembers(kitchenId) {
	return request({
		url: `/api/kitchens/${kitchenId}/members`,
		method: 'GET'
	}).then((data) => data?.items || [])
}

export function createKitchenInvite(kitchenId, payload = {}) {
	return request({
		url: `/api/kitchens/${kitchenId}/invites`,
		method: 'POST',
		data: payload
	}).then((data) => data?.invite || null)
}

export function updateKitchen(kitchenId, payload = {}) {
	return request({
		url: `/api/kitchens/${kitchenId}`,
		method: 'PATCH',
		data: payload
	}).then((data) => data?.kitchen || null)
}

export function previewInvite(token) {
	return request({
		url: `/api/invites/${token}`,
		method: 'GET',
		auth: false
	}).then((data) => data?.invite || null)
}

export function previewInviteByCode(code) {
	const normalized = normalizeInviteCode(code)
	return request({
		url: `/api/invite-codes/${encodeURIComponent(normalized)}`,
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

export function acceptInviteByCode(code) {
	const normalized = normalizeInviteCode(code)
	return request({
		url: `/api/invite-codes/${encodeURIComponent(normalized)}/accept`,
		method: 'POST',
		data: {}
	})
}
