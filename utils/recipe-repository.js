import { ensureSession, getCurrentKitchenId } from './auth'
import {
	createRecipe,
	deleteRecipe,
	ensureRecipeShareToken,
	generateRecipeFlowchart,
	getRecipeByShareToken,
	getRecipeDetail,
	listRecipes,
	reparseRecipe,
	updateRecipe,
	updateRecipePinned,
	updateRecipeStatus
} from './recipe-api'
import {
	getCachedRecipeById,
	getCachedRecipes,
	removeRecipeFromCache,
	saveRecipesForKitchen,
	upsertRecipeInCache
} from './recipe-cache'
import {
	buildRecipePayload,
	getRecipeImageSources,
	mergeUpdatedParsedContent,
	normalizeImageList,
	normalizeRecipe
} from './recipe-model'
import { ensureUploadedImages } from './upload-api'

async function resolveRecipeImages(recipe = {}) {
	return ensureUploadedImages(normalizeImageList(getRecipeImageSources(recipe)))
}

export async function syncRecipes(filters = {}) {
	const session = await ensureSession()
	const kitchenId = Number(session?.currentKitchenId) || 0
	if (!kitchenId) return []
	const items = await listRecipes(kitchenId, filters)
	return saveRecipesForKitchen(kitchenId, items)
}

export async function loadRecipes(options = {}) {
	const { forceRefresh = false, filters = {} } = options
	await ensureSession()
	if (!forceRefresh) {
		const cached = getCachedRecipes()
		if (cached.length) return cached
	}
	return syncRecipes(filters)
}

export async function getRecipeById(recipeId, options = {}) {
	const { preferCache = true } = options
	await ensureSession()
	if (preferCache) {
		const cached = getCachedRecipeById(recipeId)
		if (cached) return cached
	}
	const item = await getRecipeDetail(recipeId)
	return upsertRecipeInCache(item)
}

export async function ensureRecipeShareTokenById(recipeId) {
	await ensureSession()
	return ensureRecipeShareToken(recipeId)
}

export async function fetchPublicRecipeByShareToken(token) {
	const view = await getRecipeByShareToken(token)
	if (!view || !view.recipe) {
		return { recipe: null, kitchenName: '', creatorName: '' }
	}
	return {
		recipe: normalizeRecipe(view.recipe),
		kitchenName: view.kitchenName || '',
		creatorName: view.creatorName || ''
	}
}

export async function createRecipeFromDraft(draft = {}) {
	const session = await ensureSession()
	const kitchenId = Number(session?.currentKitchenId) || 0
	const imageUrls = await resolveRecipeImages(draft)
	const item = await createRecipe(kitchenId, buildRecipePayload({ ...draft, images: imageUrls }))
	return upsertRecipeInCache(item)
}

export async function updateRecipeById(recipeId, updates = {}) {
	const current = await getRecipeById(recipeId)
	const hasImageArray = Object.prototype.hasOwnProperty.call(updates, 'images') || Object.prototype.hasOwnProperty.call(updates, 'imageUrls')
	const hasSingleImage = Object.prototype.hasOwnProperty.call(updates, 'image') || Object.prototype.hasOwnProperty.call(updates, 'imageUrl')
	const imageSources = hasImageArray
		? updates.images || updates.imageUrls || []
		: hasSingleImage ? [updates.image || updates.imageUrl || ''] : current.imageUrls
	const imageUrls = await ensureUploadedImages(normalizeImageList(imageSources))
	const parsedContent = Object.prototype.hasOwnProperty.call(updates, 'parsedContent')
		? mergeUpdatedParsedContent(current.parsedContent, updates.parsedContent || {})
		: current.parsedContent
	const payload = buildRecipePayload({ ...current, ...updates, parsedContent, images: imageUrls })
	const item = await updateRecipe(recipeId, payload)
	return upsertRecipeInCache(item)
}

export async function toggleRecipeStatusById(recipeId) {
	const current = await getRecipeById(recipeId)
	const item = await updateRecipeStatus(recipeId, current.status === 'done' ? 'wishlist' : 'done')
	return upsertRecipeInCache(item)
}

export async function setRecipePinnedById(recipeId, pinned) {
	return upsertRecipeInCache(await updateRecipePinned(recipeId, pinned))
}

export async function reparseRecipeById(recipeId) {
	return upsertRecipeInCache(await reparseRecipe(recipeId))
}

export async function generateRecipeFlowchartById(recipeId) {
	return upsertRecipeInCache(await generateRecipeFlowchart(recipeId))
}

export async function deleteRecipeById(recipeId) {
	const current = await getRecipeById(recipeId, { preferCache: true })
	await deleteRecipe(recipeId)
	removeRecipeFromCache(recipeId, current?.kitchenId || getCurrentKitchenId())
	return true
}
