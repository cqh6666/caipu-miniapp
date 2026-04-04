import { request } from './http'

function normalizeStore(store = {}) {
	return {
		drafts: store?.drafts || {},
		submitted: Array.isArray(store?.submitted) ? store.submitted : []
	}
}

export function listMealPlanStore(kitchenId) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/meal-plans`,
		method: 'GET'
	}).then((data) => normalizeStore(data?.store))
}

export function saveMealPlanDraft(kitchenId, planDate, payload = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/meal-plans/${encodeURIComponent(planDate)}/draft`,
		method: 'PUT',
		data: payload
	}).then((data) => normalizeStore(data?.store))
}

export function submitMealPlan(kitchenId, planDate, payload = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/meal-plans/${encodeURIComponent(planDate)}/submit`,
		method: 'POST',
		data: payload
	}).then((data) => normalizeStore(data?.store))
}

export function deleteMealPlanDraft(kitchenId, planDate) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/meal-plans/${encodeURIComponent(planDate)}/draft`,
		method: 'DELETE'
	}).then((data) => normalizeStore(data?.store))
}

export function createMealPlanDraftFromSubmitted(kitchenId, planDate) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/meal-plans/${encodeURIComponent(planDate)}/draft-from-submitted`,
		method: 'POST'
	}).then((data) => normalizeStore(data?.store))
}

export function deleteSubmittedMealPlan(kitchenId, planDate) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/meal-plans/${encodeURIComponent(planDate)}/submitted`,
		method: 'DELETE'
	}).then((data) => normalizeStore(data?.store))
}
