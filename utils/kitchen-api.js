import { resolveAssetURL } from './app-config'
import { request } from './http'

function normalizeKitchenMember(member = {}) {
	return {
		...member,
		avatarUrl: resolveAssetURL(member?.avatarUrl || '')
	}
}

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
		url: `/caipu-api/kitchens/${kitchenId}/members`,
		method: 'GET'
	}).then((data) => (Array.isArray(data?.items) ? data.items.map((item) => normalizeKitchenMember(item)) : []))
}

export function createKitchenInvite(kitchenId, payload = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/invites`,
		method: 'POST',
		data: payload
	}).then((data) => data?.invite || null)
}

export function updateKitchen(kitchenId, payload = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}`,
		method: 'PATCH',
		data: payload
	}).then((data) => data?.kitchen || null)
}

export function previewInvite(token) {
	return request({
		url: `/caipu-api/invites/${token}`,
		method: 'GET',
		auth: false
	}).then((data) => data?.invite || null)
}

export function previewInviteByCode(code) {
	const normalized = normalizeInviteCode(code)
	return request({
		url: `/caipu-api/invite-codes/${encodeURIComponent(normalized)}`,
		method: 'GET',
		auth: false
	}).then((data) => data?.invite || null)
}

export function acceptInvite(token) {
	return request({
		url: `/caipu-api/invites/${token}/accept`,
		method: 'POST',
		data: {}
	})
}

export function acceptInviteByCode(code) {
	const normalized = normalizeInviteCode(code)
	return request({
		url: `/caipu-api/invite-codes/${encodeURIComponent(normalized)}/accept`,
		method: 'POST',
		data: {}
	})
}
