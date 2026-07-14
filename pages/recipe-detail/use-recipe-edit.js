import { normalizeParsedSteps } from '../../utils/recipe-store'

export const createEmptyDraft = (overrides = {}) => ({
	title: '', ingredient: '', link: '', images: [], mealType: 'breakfast',
	status: 'wishlist', mainIngredients: [], secondaryIngredients: [], steps: [],
	parsedContentMode: 'empty', note: '', ...overrides
})

let editDraftItemSeed = 0
const createEditDraftItemId = (prefix = 'draft') => `${prefix}-${Date.now()}-${editDraftItemSeed += 1}`

export function normalizeIngredientDraftItem(item = '') {
	return typeof item === 'object' && item !== null
		? { id: String(item.id || createEditDraftItemId('ingredient')), value: String(item.value || '') }
		: { id: createEditDraftItemId('ingredient'), value: String(item || '') }
}

export const createIngredientDraftList = (items = []) =>
	(Array.isArray(items) ? items : []).map((item) => normalizeIngredientDraftItem(item))

export const getIngredientDraftValues = (items = []) =>
	(Array.isArray(items) ? items : []).map((item) =>
		typeof item === 'object' && item !== null ? String(item.value || '') : String(item || '')
	)

export function createStepDraftItem(step = {}) {
	const source = typeof step === 'object' && step !== null ? step : { detail: step }
	return {
		id: String(source.id || createEditDraftItemId('step')),
		title: String(source.title || ''),
		detail: String(source.detail || source.text || '')
	}
}

export function moveListItem(items = [], fromIndex = 0, toIndex = 0) {
	if (!Array.isArray(items) || !items.length) return Array.isArray(items) ? items : []
	if (fromIndex < 0 || fromIndex >= items.length || toIndex < 0 || toIndex >= items.length || fromIndex === toIndex) return items
	const list = [...items]
	const [item] = list.splice(fromIndex, 1)
	list.splice(toIndex, 0, item)
	return list
}

export const cloneStepDraftList = (steps = []) =>
	normalizeParsedSteps(steps).map((step) => createStepDraftItem(step))

const comparableTextList = (items = []) => getIngredientDraftValues(items).map((item) => item.trim()).filter(Boolean)
const comparableStepList = (steps = []) => (Array.isArray(steps) ? steps : [])
	.map((step) => {
		const normalized = createStepDraftItem(step)
		return { title: normalized.title.trim(), detail: normalized.detail.trim() }
	})
	.filter((step) => step.title || step.detail)

export const serializeComparableEditDraft = (draft = {}) => JSON.stringify({
	title: String(draft.title || '').trim(),
	ingredient: String(draft.ingredient || '').trim(),
	link: String(draft.link || '').trim(),
	images: (Array.isArray(draft.images) ? draft.images : []).map((item) => String(item || '').trim()).filter(Boolean),
	mealType: String(draft.mealType || '').trim(),
	status: String(draft.status || '').trim(),
	mainIngredients: comparableTextList(draft.mainIngredients),
	secondaryIngredients: comparableTextList(draft.secondaryIngredients),
	steps: comparableStepList(draft.steps),
	note: String(draft.note || '').trim()
})

export function buildRecipeEditPayload(draft = {}) {
	return {
		title: String(draft.title || '').trim(),
		ingredient: String(draft.ingredient || '').trim(),
		link: String(draft.link || '').trim(),
		images: (Array.isArray(draft.images) ? draft.images : [])
			.map((item) => String(item || '').trim())
			.filter(Boolean),
		mealType: String(draft.mealType || '').trim() || 'breakfast',
		status: String(draft.status || '').trim() || 'wishlist',
		parsedContent: {
			mainIngredients: comparableTextList(draft.mainIngredients),
			secondaryIngredients: comparableTextList(draft.secondaryIngredients),
			steps: normalizeParsedSteps(draft.steps)
		},
		parsedContentEdited: draft.parsedContentMode === 'manual',
		note: String(draft.note || '').trim()
	}
}

const STEP_HIGHLIGHT_REGEX = /(\d+(?:\.\d+)?\s?(?:分钟|秒|小时|分|克|斤|g|kg|毫升|ml|L|勺|匙|杯|碗|条|个|片|块|颗|粒|根|瓣|只|滴|圈)|大火|中火|小火|中小火|大小火|文火|武火|旺火|微火|小炖|\d+\s?°?C|\d+\s?度)/g

export function highlightStepDetailText(detail) {
	const raw = String(detail || '').trim()
	if (!raw) return [{ text: '', highlight: false }]
	const segments = []
	let lastIndex = 0
	STEP_HIGHLIGHT_REGEX.lastIndex = 0
	let match
	while ((match = STEP_HIGHLIGHT_REGEX.exec(raw)) !== null) {
		if (match.index > lastIndex) segments.push({ text: raw.slice(lastIndex, match.index), highlight: false })
		segments.push({ text: match[0], highlight: true })
		lastIndex = match.index + match[0].length
	}
	if (lastIndex < raw.length) segments.push({ text: raw.slice(lastIndex), highlight: false })
	return segments.length ? segments : [{ text: raw, highlight: false }]
}

const STEP_COMPLETED_STORAGE_PREFIX = 'recipe-step-done:'
const STEP_COMPLETED_STORAGE_VERSION = 2
export function buildStepCompletedStorageKey(recipeId) {
	const id = String(recipeId || '').trim()
	return id ? `${STEP_COMPLETED_STORAGE_PREFIX}${id}` : ''
}
function buildComparableStepIdentity(step = {}) {
	const normalized = createStepDraftItem(step)
	return JSON.stringify({ title: normalized.title.trim(), detail: normalized.detail.trim() })
}
export function buildStepCompletionKeyList(steps = []) {
	const occurrenceMap = {}
	return normalizeParsedSteps(steps).map((step) => {
		const identity = buildComparableStepIdentity(step)
		const occurrence = (occurrenceMap[identity] || 0) + 1
		occurrenceMap[identity] = occurrence
		return `${identity}#${occurrence}`
	})
}
export function normalizeCompletedStepKeyMap(rawValue, currentStepKeys = []) {
	const allowedKeys = new Set(Array.isArray(currentStepKeys) ? currentStepKeys : [])
	let payload = rawValue
	if (typeof payload === 'string' && payload) {
		try { payload = JSON.parse(payload) } catch (_) { return {} }
	}
	if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return {}
	if (Number(payload.version) < STEP_COMPLETED_STORAGE_VERSION || !Array.isArray(payload.completedKeys)) return {}
	return payload.completedKeys.reduce((result, value) => {
		const key = String(value || '').trim()
		if (allowedKeys.has(key)) result[key] = true
		return result
	}, {})
}
export function createCompletedStepStoragePayload(stepKeyMap = {}) {
	return { version: STEP_COMPLETED_STORAGE_VERSION, completedKeys: Object.keys(stepKeyMap).filter((key) => stepKeyMap[key]) }
}
