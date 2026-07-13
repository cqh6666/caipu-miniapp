import { getCurrentKitchenId } from './auth'
import { compareRecipesForDisplay, normalizeRecipe } from './recipe-model'

const RECIPE_STORAGE_PREFIX = 'caipu-miniapp-recipes'

export function getRecipeStorageKey(kitchenId) {
	return `${RECIPE_STORAGE_PREFIX}:${kitchenId}`
}

export function loadRecipesForKitchen(kitchenId, storage = uni) {
	if (!kitchenId) return []
	const storedRecipes = storage.getStorageSync(getRecipeStorageKey(kitchenId))
	if (!Array.isArray(storedRecipes)) return []
	return storedRecipes.map((recipe) => normalizeRecipe(recipe)).sort(compareRecipesForDisplay)
}

export function saveRecipesForKitchen(kitchenId, recipes = [], storage = uni) {
	if (!kitchenId) return []
	const normalizedRecipes = recipes.map((recipe) => normalizeRecipe(recipe)).sort(compareRecipesForDisplay)
	storage.setStorageSync(getRecipeStorageKey(kitchenId), normalizedRecipes)
	return normalizedRecipes
}

export function upsertRecipeInCache(recipe = {}, storage = uni) {
	const normalized = normalizeRecipe(recipe)
	const kitchenId = normalized.kitchenId || getCurrentKitchenId()
	const current = loadRecipesForKitchen(kitchenId, storage)
	const filtered = current.filter((item) => item.id !== normalized.id)
	saveRecipesForKitchen(kitchenId, [normalized, ...filtered], storage)
	return normalized
}

export function removeRecipeFromCache(recipeId, kitchenId = getCurrentKitchenId(), storage = uni) {
	const current = loadRecipesForKitchen(kitchenId, storage)
	const nextRecipes = current.filter((item) => item.id !== recipeId)
	saveRecipesForKitchen(kitchenId, nextRecipes, storage)
	return nextRecipes
}

export function getCachedRecipes(kitchenId = getCurrentKitchenId()) {
	return loadRecipesForKitchen(kitchenId)
}

export function getCachedRecipeById(recipeId, kitchenId = getCurrentKitchenId()) {
	return loadRecipesForKitchen(kitchenId).find((item) => item.id === recipeId) || null
}
