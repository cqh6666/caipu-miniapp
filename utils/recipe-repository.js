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
	let current = await getRecipeById(recipeId)
	if (Number(current?.version) < 1) {
		current = await getRecipeById(recipeId, { preferCache: false })
	}
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
	try {
		const item = await updateRecipe(recipeId, payload)
		return upsertRecipeInCache(item)
	} catch (error) {
		return handleRecipeVersionConflict(recipeId, error)
	}
}

export async function toggleRecipeStatusById(recipeId) {
	let current = await getRecipeById(recipeId)
	if (Number(current?.version) < 1) {
		current = await getRecipeById(recipeId, { preferCache: false })
	}
	try {
		const item = await updateRecipeStatus(recipeId, current.status === 'done' ? 'wishlist' : 'done', current.version)
		return upsertRecipeInCache(item)
	} catch (error) {
		return handleRecipeVersionConflict(recipeId, error)
	}
}

export async function setRecipePinnedById(recipeId, pinned) {
	let current = await getRecipeById(recipeId)
	if (Number(current?.version) < 1) {
		current = await getRecipeById(recipeId, { preferCache: false })
	}
	try {
		return upsertRecipeInCache(await updateRecipePinned(recipeId, pinned, current.version))
	} catch (error) {
		return handleRecipeVersionConflict(recipeId, error)
	}
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

async function handleRecipeVersionConflict(recipeId, error) {
	if (Number(error?.code) !== 40900) throw error
	try {
		const latest = await getRecipeDetail(recipeId)
		upsertRecipeInCache(latest)
	} catch (_) {}
	const conflict = new Error('菜谱已被其他成员更新，已刷新最新内容，请确认后重试')
	conflict.code = 40900
	conflict.statusCode = 409
	conflict.originalError = error
	throw conflict
}
