import { resolveAssetURL } from './app-config'
import { request } from './http'

function normalizeKitchenMember(member = {}) {
	return {
		...member,
		avatarUrl: resolveAssetURL(member?.avatarUrl || '')
	}
}

function normalizeInvite(invite = {}) {
	if (!invite || typeof invite !== 'object') return null

	return {
		...invite,
		shareImageUrl: resolveAssetURL(invite?.shareImageUrl || '')
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
	}).then((data) => normalizeInvite(data?.invite || null))
}

export function updateKitchen(kitchenId, payload = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}`,
		method: 'PATCH',
		data: payload
	}).then((data) => data?.kitchen || null)
}

export function leaveKitchen(kitchenId) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/members/me`,
		method: 'DELETE'
	})
}

export function previewInvite(token) {
	return request({
		url: `/caipu-api/invites/${token}`,
		method: 'GET',
		auth: false
	}).then((data) => normalizeInvite(data?.invite || null))
}

export function previewInviteByCode(code) {
	const normalized = normalizeInviteCode(code)
	return request({
		url: `/caipu-api/invite-codes/${encodeURIComponent(normalized)}`,
		method: 'GET',
		auth: false
	}).then((data) => normalizeInvite(data?.invite || null))
}

export function acceptInvite(token) {
	return request({
		url: `/caipu-api/invites/${token}/accept`,
		method: 'POST',
		data: {}
	}).then((data) => ({
		...data,
		invite: normalizeInvite(data?.invite || null)
	}))
}

export function acceptInviteByCode(code) {
	const normalized = normalizeInviteCode(code)
	return request({
		url: `/caipu-api/invite-codes/${encodeURIComponent(normalized)}/accept`,
		method: 'POST',
		data: {}
	}).then((data) => ({
		...data,
		invite: normalizeInvite(data?.invite || null)
	}))
}
