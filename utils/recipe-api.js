import { request } from './http'

export function listRecipes(kitchenId, filters = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/recipes`,
		method: 'GET',
		data: filters
	}).then((data) => data?.items || [])
}

export function previewRecipeLink(url) {
	return request({
		url: '/caipu-api/link-parsers/preview',
		method: 'POST',
		data: { url }
	}).then((data) => data?.result || null)
}

export function createRecipe(kitchenId, payload) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/recipes`,
		method: 'POST',
		data: payload
	}).then((data) => data?.recipe || null)
}

export function getRecipeDetail(recipeId) {
	return request({
		url: `/caipu-api/recipes/${recipeId}`,
		method: 'GET'
	}).then((data) => data?.recipe || null)
}

export function updateRecipe(recipeId, payload) {
	return request({
		url: `/caipu-api/recipes/${recipeId}`,
		method: 'PUT',
		data: payload
	}).then((data) => data?.recipe || null)
}

export function reparseRecipe(recipeId) {
	return request({
		url: `/caipu-api/recipes/${recipeId}/reparse`,
		method: 'POST',
		data: {}
	}).then((data) => data?.recipe || null)
}

export function generateRecipeFlowchart(recipeId) {
	return request({
		url: `/caipu-api/recipes/${recipeId}/flowchart`,
		method: 'POST',
		data: {}
	}).then((data) => data?.recipe || null)
}

export function updateRecipeStatus(recipeId, status) {
	return request({
		url: `/caipu-api/recipes/${recipeId}/status`,
		method: 'PATCH',
		data: { status }
	}).then((data) => data?.recipe || null)
}

export function updateRecipePinned(recipeId, pinned) {
	return request({
		url: `/caipu-api/recipes/${recipeId}/pin`,
		method: 'PATCH',
		data: { pinned: !!pinned }
	}).then((data) => data?.recipe || null)
}

export function deleteRecipe(recipeId) {
	return request({
		url: `/caipu-api/recipes/${recipeId}`,
		method: 'DELETE'
	})
}

// ensureRecipeShareToken：为指定菜谱获取或生成永久 share_token
// 后端幂等：已有 token 直接返回；无则生成后写库返回
export function ensureRecipeShareToken(recipeId) {
	return request({
		url: `/caipu-api/recipes/${recipeId}/share-token`,
		method: 'POST',
		data: {}
	}).then((data) => data?.shareToken || '')
}

// getRecipeByShareToken：通过 share_token 公开只读访问菜谱
// 不走鉴权（auth: false），返回 { recipe, kitchenName, creatorName }
export function getRecipeByShareToken(token) {
	return request({
		url: `/caipu-api/public/recipes/by-share-token/${token}`,
		method: 'GET',
		auth: false
	}).then((data) => ({
		recipe: data?.recipe || null,
		kitchenName: data?.kitchenName || '',
		creatorName: data?.creatorName || ''
	}))
}
