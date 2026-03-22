import { resolveAssetURL } from './app-config'
import { ensureSession, getCurrentKitchenId } from './auth'
import {
	createRecipe,
	deleteRecipe,
	generateRecipeFlowchart,
	getRecipeDetail,
	listRecipes,
	reparseRecipe,
	updateRecipe,
	updateRecipePinned,
	updateRecipeStatus
} from './recipe-api'
import { ensureUploadedImages } from './upload-api'

const RECIPE_STORAGE_PREFIX = 'caipu-miniapp-recipes'
export const MAX_RECIPE_IMAGES = 9

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

const secondaryIngredientPattern = /(常用配菜|基础调味|常用调味料|调味|葱|姜|蒜|香叶|桂皮|八角|花椒|胡椒|盐|糖|冰糖|白糖|红糖|生抽|老抽|蚝油|料酒|鸡精|味精|醋|陈醋|米醋|香醋|豆瓣酱|辣椒|小米椒|淀粉|清水|热水|食用油|香油|芝麻油|花椒粉|辣椒粉|五香粉|十三香|孜然|芝麻|香菜|葱花)/
const secondaryIngredientExceptionPattern = /^(洋葱|红葱头|葱头)/
const ingredientSuffixPattern = /\s*(?:\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)|半个|半颗|半根|半头|适量|少许)$/

export function normalizeTextList(items = []) {
	const source = Array.isArray(items) ? items : [items]
	const normalized = []
	const seen = new Set()

	source.forEach((item) => {
		const value = String(item || '').trim()
		if (!value || seen.has(value)) return
		seen.add(value)
		normalized.push(value)
	})

	return normalized
}

function inferStepTitle(detail = '', index = 0) {
	const text = String(detail || '').trim()
	if (!text) return ''
	if (text.includes('焯水') || text.includes('汆水')) {
		return text.includes('腥') || text.includes('浮沫') ? '焯水去腥' : '焯水备用'
	}
	if (text.includes('腌')) return '腌制入味'
	if (text.includes('糖色') || text.includes('冰糖')) return '炒糖上色'
	if (text.includes('爆香') || text.includes('炒香')) return '炒香底料'
	if (text.includes('切') || text.includes('改刀')) return '切配备料'
	if (text.includes('收汁')) return '收汁出锅'
	if (text.includes('炖') || text.includes('焖')) return '小火慢炖'
	if (text.includes('蒸')) return '上锅蒸熟'
	if (text.includes('炸')) return '炸至金黄'
	if (text.includes('煎')) return '煎香上色'
	if (text.includes('烤')) return '烤至上色'
	if (text.includes('煮')) return '煮至入味'
	if (text.includes('拌')) return '拌匀调味'
	if (text.includes('炒') || text.includes('翻炒')) return '翻炒入味'
	if (text.includes('出锅')) return '调味出锅'
	return index === 0 ? '处理食材' : '继续烹饪'
}

export function normalizeParsedSteps(steps = []) {
	const source = Array.isArray(steps) ? steps : []
	const normalized = []
	const seen = new Set()

	source.forEach((step) => {
		const title = typeof step === 'object' && step !== null ? String(step.title || '').trim() : ''
		const detail =
			typeof step === 'string'
				? step.trim()
				: String(step?.detail || step?.text || '').trim()
		const nextDetail = detail || title
		const nextTitle = title || inferStepTitle(nextDetail, normalized.length)
		if (!nextDetail) return
		const key = `${nextTitle}\u0000${nextDetail}`
		if (seen.has(key)) return
		seen.add(key)
		normalized.push({
			title: nextTitle,
			detail: nextDetail
		})
	})

	return normalized
}

function ingredientLabelFromLine(line = '') {
	return String(line || '').trim().replace(ingredientSuffixPattern, '').trim()
}

function splitIngredientLines(lines = []) {
	const cleaned = normalizeTextList(lines)
	if (!cleaned.length) {
		return {
			mainIngredients: [],
			secondaryIngredients: []
		}
	}

	const mainIngredients = []
	const secondaryIngredients = []
	cleaned.forEach((line) => {
		const label = ingredientLabelFromLine(line)
		if (secondaryIngredientPattern.test(label) && !secondaryIngredientExceptionPattern.test(label)) {
			secondaryIngredients.push(line)
			return
		}
		mainIngredients.push(line)
	})

	if (!mainIngredients.length) {
		return {
			mainIngredients: cleaned.slice(0, 3),
			secondaryIngredients: cleaned.slice(3)
		}
	}

	return {
		mainIngredients,
		secondaryIngredients
	}
}

function splitSecondaryIngredientLines(lines = []) {
	const cleaned = normalizeTextList(lines)
	const supportingIngredients = []
	const seasonings = []

	cleaned.forEach((line) => {
		const label = ingredientLabelFromLine(line)
		if (secondaryIngredientPattern.test(label) && !secondaryIngredientExceptionPattern.test(label)) {
			seasonings.push(line)
			return
		}
		supportingIngredients.push(line)
	})

	return {
		supportingIngredients,
		seasonings
	}
}

function stepDetailsToList(steps = []) {
	return normalizeParsedSteps(steps).map((step) => step.detail)
}

function hasParsedIngredientOverride(parsedContent = {}) {
	return ['mainIngredients', 'secondaryIngredients', 'ingredients'].some((key) =>
		Object.prototype.hasOwnProperty.call(parsedContent || {}, key)
	)
}

function hasParsedStepOverride(parsedContent = {}) {
	return Object.prototype.hasOwnProperty.call(parsedContent || {}, 'steps')
}

function mergeUpdatedParsedContent(currentParsedContent = {}, nextParsedContent = {}) {
	const current = cloneParsedContent(currentParsedContent)
	const next = cloneParsedContent(nextParsedContent)
	const preserveIngredients =
		!hasParsedIngredientOverride(nextParsedContent) ||
		stringSlicesEqual(next.ingredients, current.ingredients)
	const preserveSteps =
		!hasParsedStepOverride(nextParsedContent) ||
		stringSlicesEqual(stepDetailsToList(next.steps), stepDetailsToList(current.steps))
	const mainIngredients = preserveIngredients ? current.mainIngredients : next.mainIngredients
	const secondaryIngredients = preserveIngredients ? current.secondaryIngredients : next.secondaryIngredients
	const steps = preserveSteps ? current.steps : next.steps

	return {
		mainIngredients,
		secondaryIngredients,
		ingredients: [...mainIngredients, ...secondaryIngredients],
		steps
	}
}

export function normalizeParsedContentView(parsedContent = {}) {
	const mainIngredients = normalizeTextList(parsedContent.mainIngredients)
	const secondaryIngredients = normalizeTextList(parsedContent.secondaryIngredients)
	const legacyIngredients = normalizeTextList(parsedContent.ingredients)
	const groupedIngredients =
		mainIngredients.length || secondaryIngredients.length
			? { mainIngredients, secondaryIngredients }
			: splitIngredientLines(legacyIngredients)
	const secondaryGroups = splitSecondaryIngredientLines(groupedIngredients.secondaryIngredients)

	return {
		mainIngredients: groupedIngredients.mainIngredients,
		secondaryIngredients: groupedIngredients.secondaryIngredients,
		supportingIngredients: secondaryGroups.supportingIngredients,
		seasonings: secondaryGroups.seasonings,
		ingredients: [...groupedIngredients.mainIngredients, ...groupedIngredients.secondaryIngredients],
		steps: normalizeParsedSteps(parsedContent.steps)
	}
}

function cloneParsedContent(parsedContent = {}) {
	const normalized = normalizeParsedContentView(parsedContent)
	return {
		mainIngredients: normalized.mainIngredients,
		secondaryIngredients: normalized.secondaryIngredients,
		ingredients: normalized.ingredients,
		steps: normalized.steps
	}
}

function normalizeImageList(images = []) {
	const source = Array.isArray(images) ? images : [images]
	const normalized = []
	const seen = new Set()

	source.forEach((item) => {
		const value = String(item || '').trim()
		if (!value || seen.has(value)) return
		seen.add(value)
		normalized.push(value)
	})

	return normalized.slice(0, MAX_RECIPE_IMAGES)
}

function compareRecipesForDisplay(left = {}, right = {}) {
	const leftPinnedAt = String(left.pinnedAt || '').trim()
	const rightPinnedAt = String(right.pinnedAt || '').trim()
	const leftPinned = !!leftPinnedAt
	const rightPinned = !!rightPinnedAt

	if (leftPinned !== rightPinned) {
		return leftPinned ? -1 : 1
	}

	if (leftPinned && rightPinned) {
		const byPinnedAt = rightPinnedAt.localeCompare(leftPinnedAt)
		if (byPinnedAt) return byPinnedAt
	}

	return (right.updatedAt || '').localeCompare(left.updatedAt || '') || right.id.localeCompare(left.id)
}

export function buildFallbackParsedContent(recipe = {}) {
	const mealLabel = recipe.mealType === 'main' ? '正餐' : '早餐'
	const mainIngredient = (recipe.ingredient || recipe.title || '主食材').trim()

	return {
		mainIngredients: [
			`${mainIngredient} 1份`,
		],
		secondaryIngredients: [
			`${mealLabel}常用配菜 适量`,
			'基础调味 适量'
		],
		ingredients: [
			`${mainIngredient} 1份`,
			`${mealLabel}常用配菜 适量`,
			'基础调味 适量'
		],
		steps: [
			{ title: '整理做法', detail: '先整理这道菜的核心做法。' },
			{ title: '调整口味', detail: '按自己的口味调整成容易复刻的版本。' },
			{ title: '补充记录', detail: '做完以后补充口感和火候记录。' }
		]
	}
}

function stringSlicesEqual(left = [], right = []) {
	if (left.length !== right.length) return false
	return left.every((item, index) => item === right[index])
}

function stepSlicesEqual(left = [], right = []) {
	if (left.length !== right.length) return false
	return left.every((item, index) => item.title === right[index]?.title && item.detail === right[index]?.detail)
}

export function isFallbackParsedContent(recipe = {}, parsedContent = {}) {
	const current = cloneParsedContent(parsedContent)
	const fallback = buildFallbackParsedContent(recipe)
	return (
		stringSlicesEqual(current.mainIngredients, fallback.mainIngredients) &&
		stringSlicesEqual(current.secondaryIngredients, fallback.secondaryIngredients) &&
		stepSlicesEqual(current.steps, fallback.steps)
	)
}

export function normalizeRecipe(recipe = {}) {
	const imageUrls = normalizeImageList(
		(Array.isArray(recipe.imageUrls) && recipe.imageUrls.length
			? recipe.imageUrls
			: Array.isArray(recipe.images) && recipe.images.length
				? recipe.images
				: [recipe.image, recipe.imageUrl]
		).map((item) => resolveAssetURL(item || ''))
	)
	const image = imageUrls[0] || ''
	const flowchartImageUrl = resolveAssetURL(recipe.flowchartImageUrl || '')
	const normalized = {
		id: recipe.id || '',
		kitchenId: Number(recipe.kitchenId) || 0,
		title: (recipe.title || '').trim(),
		ingredient: (recipe.ingredient || '').trim(),
		summary: (recipe.summary || '').trim(),
		link: (recipe.link || '').trim(),
		image,
		imageUrl: image,
		images: imageUrls,
		imageUrls,
		flowchartImageUrl,
		flowchartStatus: (recipe.flowchartStatus || '').trim(),
		flowchartError: (recipe.flowchartError || '').trim(),
		flowchartRequestedAt: recipe.flowchartRequestedAt || '',
		flowchartFinishedAt: recipe.flowchartFinishedAt || '',
		flowchartUpdatedAt: recipe.flowchartUpdatedAt || '',
		flowchartStale: !!recipe.flowchartStale,
		mealType: recipe.mealType || 'breakfast',
		status: recipe.status || 'wishlist',
		note: (recipe.note || '').trim(),
		parsedContentEdited: !!recipe.parsedContentEdited,
		parseStatus: (recipe.parseStatus || '').trim(),
		parseSource: (recipe.parseSource || '').trim(),
		parseError: (recipe.parseError || '').trim(),
		parseRequestedAt: recipe.parseRequestedAt || '',
		parseFinishedAt: recipe.parseFinishedAt || '',
		pinnedAt: recipe.pinnedAt || '',
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
	const parsedContent = isFallbackParsedContent(normalized, normalized.parsedContent)
		? { mainIngredients: [], secondaryIngredients: [], steps: [] }
		: {
			mainIngredients: normalized.parsedContent.mainIngredients,
			secondaryIngredients: normalized.parsedContent.secondaryIngredients,
			steps: normalized.parsedContent.steps
		}

	const payload = {
		title: normalized.title,
		ingredient: normalized.ingredient,
		summary: normalized.summary,
		link: normalized.link,
		imageUrl: normalized.imageUrl,
		imageUrls: normalized.imageUrls,
		mealType: normalized.mealType,
		status: normalized.status,
		note: normalized.note,
		parsedContent
	}

	if (Object.prototype.hasOwnProperty.call(recipe, 'parsedContentEdited')) {
		payload.parsedContentEdited = !!recipe.parsedContentEdited
	}

	return payload
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
	return storedRecipes.map((recipe) => normalizeRecipe(recipe)).sort(compareRecipesForDisplay)
}

function saveRecipesForKitchen(kitchenId, recipes = []) {
	if (!kitchenId) return []
	const normalizedRecipes = recipes.map((recipe) => normalizeRecipe(recipe)).sort(compareRecipesForDisplay)
	uni.setStorageSync(getRecipeStorageKey(kitchenId), normalizedRecipes)
	return normalizedRecipes
}

function upsertRecipeInCache(recipe = {}) {
	const normalized = normalizeRecipe(recipe)
	const kitchenId = normalized.kitchenId || getCurrentKitchenId()
	const current = loadRecipesForKitchen(kitchenId)
	const filtered = current.filter((item) => item.id !== normalized.id)
	const nextRecipes = [normalized, ...filtered].sort(compareRecipesForDisplay)
	saveRecipesForKitchen(kitchenId, nextRecipes)
	return normalized
}

function removeRecipeFromCache(recipeId, kitchenId = getCurrentKitchenId()) {
	const current = loadRecipesForKitchen(kitchenId)
	const nextRecipes = current.filter((item) => item.id !== recipeId)
	saveRecipesForKitchen(kitchenId, nextRecipes)
	return nextRecipes
}

async function resolveRecipeImages(recipe = {}) {
	const imageSources =
		Array.isArray(recipe.images) && recipe.images.length
			? recipe.images
			: Array.isArray(recipe.imageUrls) && recipe.imageUrls.length
				? recipe.imageUrls
				: [recipe.image || recipe.imageUrl || '']

	return ensureUploadedImages(normalizeImageList(imageSources))
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
	const imageUrls = await resolveRecipeImages(draft)
	const payload = buildRecipePayload({
		...draft,
		images: imageUrls
	})
	const item = await createRecipe(kitchenId, payload)
	return upsertRecipeInCache(item)
}

export async function updateRecipeById(recipeId, updates = {}) {
	const current = await getRecipeById(recipeId)
	const hasImageArray =
		Object.prototype.hasOwnProperty.call(updates, 'images') ||
		Object.prototype.hasOwnProperty.call(updates, 'imageUrls')
	const hasSingleImage =
		Object.prototype.hasOwnProperty.call(updates, 'image') ||
		Object.prototype.hasOwnProperty.call(updates, 'imageUrl')
	const imageSources = hasImageArray
		? updates.images || updates.imageUrls || []
		: hasSingleImage
			? [updates.image || updates.imageUrl || '']
			: current.imageUrls
	const imageUrls = await ensureUploadedImages(normalizeImageList(imageSources))
	const parsedContent = Object.prototype.hasOwnProperty.call(updates, 'parsedContent')
		? mergeUpdatedParsedContent(current.parsedContent, updates.parsedContent || {})
		: current.parsedContent
	const payload = buildRecipePayload({
		...current,
		...updates,
		parsedContent,
		images: imageUrls
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

export async function setRecipePinnedById(recipeId, pinned) {
	const item = await updateRecipePinned(recipeId, pinned)
	return upsertRecipeInCache(item)
}

export async function reparseRecipeById(recipeId) {
	const item = await reparseRecipe(recipeId)
	return upsertRecipeInCache(item)
}

export async function generateRecipeFlowchartById(recipeId) {
	const item = await generateRecipeFlowchart(recipeId)
	return upsertRecipeInCache(item)
}

export async function deleteRecipeById(recipeId) {
	const current = await getRecipeById(recipeId, { preferCache: true })
	await deleteRecipe(recipeId)
	removeRecipeFromCache(recipeId, current?.kitchenId || getCurrentKitchenId())
	return true
}
