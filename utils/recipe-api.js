import { request } from './http'

export function listRecipes(kitchenId, filters = {}) {
	return request({
		url: `/api/kitchens/${kitchenId}/recipes`,
		method: 'GET',
		data: filters
	}).then((data) => data?.items || [])
}

export function createRecipe(kitchenId, payload) {
	return request({
		url: `/api/kitchens/${kitchenId}/recipes`,
		method: 'POST',
		data: payload
	}).then((data) => data?.recipe || null)
}

export function getRecipeDetail(recipeId) {
	return request({
		url: `/api/recipes/${recipeId}`,
		method: 'GET'
	}).then((data) => data?.recipe || null)
}

export function updateRecipe(recipeId, payload) {
	return request({
		url: `/api/recipes/${recipeId}`,
		method: 'PUT',
		data: payload
	}).then((data) => data?.recipe || null)
}

export function updateRecipeStatus(recipeId, status) {
	return request({
		url: `/api/recipes/${recipeId}/status`,
		method: 'PATCH',
		data: { status }
	}).then((data) => data?.recipe || null)
}

export function deleteRecipe(recipeId) {
	return request({
		url: `/api/recipes/${recipeId}`,
		method: 'DELETE'
	})
}
