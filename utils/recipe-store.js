import { ensureSession, getCurrentKitchenId } from './auth'
import {
	createRecipe,
	deleteRecipe,
	getRecipeDetail,
	listRecipes,
	reparseRecipe,
	updateRecipe,
	updateRecipeStatus
} from './recipe-api'
import { ensureUploadedImage } from './upload-api'

const RECIPE_STORAGE_PREFIX = 'caipu-miniapp-recipes'

export const mealTypeOptions = [
	{ label: '早餐', value: 'breakfast', icon: 'clock-fill', activeColor: '#a06a3f' },
	{ label: '正餐', value: 'main', icon: 'grid-fill', activeColor: '#5b4a3b' }
]

export const statusOptions = [
	{ label: '想吃', value: 'wishlist' },
	{ label: '吃过', value: 'done' }
]

export const mealTypeLabelMap = {
	breakfast: '早餐',
	main: '正餐'
}

export const statusLabelMap = {
	wishlist: '想吃',
	done: '吃过'
}

function getRecipeStorageKey(kitchenId) {
	return `${RECIPE_STORAGE_PREFIX}:${kitchenId}`
}

function cloneParsedContent(parsedContent = {}) {
	const ingredients = Array.isArray(parsedContent.ingredients) ? parsedContent.ingredients.filter(Boolean) : []
	const steps = Array.isArray(parsedContent.steps) ? parsedContent.steps.filter(Boolean) : []
	return {
		ingredients,
		steps
	}
}

export function buildFallbackParsedContent(recipe = {}) {
	const mealLabel = mealTypeLabelMap[recipe.mealType] || '这道菜'
	const mainIngredient = (recipe.ingredient || recipe.title || '主食材').trim()

	return {
		ingredients: [
			`${mainIngredient} 1份`,
			`${mealLabel}常用配菜 适量`,
			'基础调味 适量'
		],
		steps: [
			`先从链接里抓出 ${recipe.title || '这道菜'} 的核心做法。`,
			'按自己的口味整理成容易复刻的家常版本。',
			'做完以后回来补充口感、火候和踩坑点。'
		]
	}
}

export function normalizeRecipe(recipe = {}) {
	const image = recipe.image || recipe.imageUrl || ''
	const normalized = {
		id: recipe.id || '',
		kitchenId: Number(recipe.kitchenId) || 0,
		title: (recipe.title || '').trim(),
		ingredient: (recipe.ingredient || '').trim(),
		link: (recipe.link || '').trim(),
		image,
		imageUrl: image,
		mealType: recipe.mealType || 'breakfast',
		status: recipe.status || 'wishlist',
		note: (recipe.note || '').trim(),
		parseStatus: (recipe.parseStatus || '').trim(),
		parseSource: (recipe.parseSource || '').trim(),
		parseError: (recipe.parseError || '').trim(),
		parseRequestedAt: recipe.parseRequestedAt || '',
		parseFinishedAt: recipe.parseFinishedAt || '',
		createdAt: recipe.createdAt || '',
		updatedAt: recipe.updatedAt || ''
	}

	const parsedContent = cloneParsedContent(recipe.parsedContent)
	const hasParsedContent = parsedContent.ingredients.length || parsedContent.steps.length

	return {
		...normalized,
		parsedContent: hasParsedContent ? parsedContent : buildFallbackParsedContent(normalized)
	}
}

function buildRecipePayload(recipe = {}) {
	const normalized = normalizeRecipe(recipe)
	return {
		title: normalized.title,
		ingredient: normalized.ingredient,
		link: normalized.link,
		imageUrl: normalized.imageUrl,
		mealType: normalized.mealType,
		status: normalized.status,
		note: normalized.note,
		parsedContent: normalized.parsedContent
	}
}

export function formatRecipeLink(link = '') {
	const cleaned = link.replace(/^https?:\/\//, '').replace(/^www\./, '').split('?')[0]
	return cleaned.length > 32 ? `${cleaned.slice(0, 29)}...` : cleaned
}

export function getRecipeSecondaryText(recipe = {}) {
	const ingredient = (recipe.ingredient || '').trim()
	const note = (recipe.note || '').trim()
	const link = (recipe.link || '').trim()

	if (ingredient && note) return `${ingredient} · ${note}`
	if (ingredient) return ingredient
	if (note) return note
	if (link) return formatRecipeLink(link)

	const mealLabel = mealTypeLabelMap[recipe.mealType] || '早餐'
	const statusLabel = statusLabelMap[recipe.status] || '想吃'
	return `${mealLabel} · ${statusLabel}`
}

function loadRecipesForKitchen(kitchenId) {
	if (!kitchenId) return []
	const storedRecipes = uni.getStorageSync(getRecipeStorageKey(kitchenId))
	if (!Array.isArray(storedRecipes)) return []
	return storedRecipes.map((recipe) => normalizeRecipe(recipe))
}

function saveRecipesForKitchen(kitchenId, recipes = []) {
	if (!kitchenId) return []
	const normalizedRecipes = recipes.map((recipe) => normalizeRecipe(recipe))
	uni.setStorageSync(getRecipeStorageKey(kitchenId), normalizedRecipes)
	return normalizedRecipes
}

function upsertRecipeInCache(recipe = {}) {
	const normalized = normalizeRecipe(recipe)
	const kitchenId = normalized.kitchenId || getCurrentKitchenId()
	const current = loadRecipesForKitchen(kitchenId)
	const filtered = current.filter((item) => item.id !== normalized.id)
	const nextRecipes = [normalized, ...filtered].sort((left, right) => {
		return (right.updatedAt || '').localeCompare(left.updatedAt || '') || right.id.localeCompare(left.id)
	})
	saveRecipesForKitchen(kitchenId, nextRecipes)
	return normalized
}

function removeRecipeFromCache(recipeId, kitchenId = getCurrentKitchenId()) {
	const current = loadRecipesForKitchen(kitchenId)
	const nextRecipes = current.filter((item) => item.id !== recipeId)
	saveRecipesForKitchen(kitchenId, nextRecipes)
	return nextRecipes
}

async function resolveRecipeImage(recipe = {}) {
	return ensureUploadedImage(recipe.image || recipe.imageUrl || '')
}

export function getCachedRecipes(kitchenId = getCurrentKitchenId()) {
	return loadRecipesForKitchen(kitchenId)
}

export function getCachedRecipeById(recipeId, kitchenId = getCurrentKitchenId()) {
	return loadRecipesForKitchen(kitchenId).find((item) => item.id === recipeId) || null
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
		if (cached.length) {
			return cached
		}
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

export async function createRecipeFromDraft(draft = {}) {
	const session = await ensureSession()
	const kitchenId = Number(session?.currentKitchenId) || 0
	const imageUrl = await resolveRecipeImage(draft)
	const payload = buildRecipePayload({
		...draft,
		image: imageUrl
	})
	const item = await createRecipe(kitchenId, payload)
	return upsertRecipeInCache(item)
}

export async function updateRecipeById(recipeId, updates = {}) {
	const current = await getRecipeById(recipeId)
	const image = Object.prototype.hasOwnProperty.call(updates, 'image') ? updates.image : current.image
	const imageUrl = await ensureUploadedImage(image)
	const payload = buildRecipePayload({
		...current,
		...updates,
		image: imageUrl
	})
	const item = await updateRecipe(recipeId, payload)
	return upsertRecipeInCache(item)
}

export async function toggleRecipeStatusById(recipeId) {
	const current = await getRecipeById(recipeId)
	const nextStatus = current.status === 'done' ? 'wishlist' : 'done'
	const item = await updateRecipeStatus(recipeId, nextStatus)
	return upsertRecipeInCache(item)
}

export async function reparseRecipeById(recipeId) {
	const item = await reparseRecipe(recipeId)
	return upsertRecipeInCache(item)
}

export async function deleteRecipeById(recipeId) {
	const current = await getRecipeById(recipeId, { preferCache: true })
	await deleteRecipe(recipeId)
	removeRecipeFromCache(recipeId, current?.kitchenId || getCurrentKitchenId())
	return true
}
