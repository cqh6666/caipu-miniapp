import { resolveAssetURL } from './app-config'
import { ensureSession, getCurrentKitchenId } from './auth'
import {
	createPlace,
	deletePlace,
	getPlaceDetail,
	listPlaces,
	updatePlace,
	updatePlaceStatus
} from './place-api'
import { ensureUploadedImages } from './upload-api'

const PLACE_STORAGE_PREFIX = 'caipu-miniapp-places'
export const MAX_PLACE_IMAGES = 6

export const placeStatusOptions = [
	{ label: '想去', value: 'want' },
	{ label: '去过', value: 'visited' }
]

export const placeTypeOptions = [
	{ label: '餐厅', value: 'food' },
	{ label: '景点', value: 'attraction' },
	{ label: '其他', value: 'other' }
]

export const placeSourceOptions = [
	{ label: '手动记录', value: 'manual' },
	{ label: '大众点评', value: 'dianping' },
	{ label: '美团', value: 'meituan' },
	{ label: '其他', value: 'other' }
]

function getPlaceStorageKey(kitchenId) {
	return `${PLACE_STORAGE_PREFIX}:${kitchenId}`
}

function normalizeString(value = '') {
	return String(value || '').trim()
}

function normalizeNumber(value = 0) {
	const numeric = Number(value)
	return Number.isFinite(numeric) ? numeric : 0
}

function normalizeOption(value = '', options = [], fallback = '') {
	const normalized = normalizeString(value)
	return options.some((item) => item.value === normalized) ? normalized : fallback
}

function normalizeTextList(items = [], limit = 8) {
	const source = Array.isArray(items) ? items : String(items || '').split(/[，,\n]/)
	const normalized = []
	const seen = new Set()

	source.forEach((item) => {
		const value = normalizeString(item)
		if (!value) return
		const key = value.toLowerCase()
		if (seen.has(key)) return
		seen.add(key)
		normalized.push(value)
	})

	return normalized.slice(0, limit)
}

function normalizeImageList(items = []) {
	const source = Array.isArray(items) ? items : [items]
	return source
		.map((item) => resolveAssetURL(normalizeString(item)))
		.filter(Boolean)
		.slice(0, MAX_PLACE_IMAGES)
}

export function normalizePlace(place = {}) {
	const status = normalizeOption(place.status, placeStatusOptions, 'want')
	const type = normalizeOption(place.type, placeTypeOptions, 'food')
	const source = normalizeOption(place.source, placeSourceOptions, 'manual')

	return {
		id: normalizeString(place.id),
		kitchenId: Number(place.kitchenId) || 0,
		name: normalizeString(place.name),
		type,
		address: normalizeString(place.address),
		latitude: normalizeNumber(place.latitude),
		longitude: normalizeNumber(place.longitude),
		price: normalizeString(place.price),
		source,
		sourceUrl: normalizeString(place.sourceUrl),
		imageUrls: normalizeImageList(place.imageUrls || place.images || place.imageUrl),
		status,
		tags: normalizeTextList(place.tags, 8),
		note: normalizeString(place.note),
		visitedAt: normalizeString(place.visitedAt),
		// 新增字段
		phone: normalizeString(place.phone),
		revisitRating: Number(place.revisitRating) || 0,
		recommendedItems: normalizeTextList(place.recommendedItems, 12),
		externalProvider: normalizeString(place.externalProvider),
		externalPoiId: normalizeString(place.externalPoiId),
		rating: normalizeString(place.rating),
		diningTips: normalizeString(place.diningTips),
		scenes: normalizeTextList(place.scenes, 8),
		bestTime: normalizeString(place.bestTime),
		duration: normalizeString(place.duration),
		companionTags: normalizeTextList(place.companionTags, 6),
		parkingNote: normalizeString(place.parkingNote),
		createdAt: normalizeString(place.createdAt),
		updatedAt: normalizeString(place.updatedAt)
	}
}

function comparePlacesForDisplay(left, right) {
	const statusWeight = (item) => (item.status === 'want' ? 0 : 1)
	const weightDelta = statusWeight(left) - statusWeight(right)
	if (weightDelta !== 0) return weightDelta
	return String(right.updatedAt || right.createdAt || '').localeCompare(String(left.updatedAt || left.createdAt || ''))
}

function loadPlacesForKitchen(kitchenId) {
	if (!kitchenId) return []
	const storedPlaces = uni.getStorageSync(getPlaceStorageKey(kitchenId))
	if (!Array.isArray(storedPlaces)) return []
	return storedPlaces.map((item) => normalizePlace(item)).filter((item) => item.id).sort(comparePlacesForDisplay)
}

function savePlacesForKitchen(kitchenId, places = []) {
	if (!kitchenId) return []
	const normalizedPlaces = places.map((item) => normalizePlace(item)).filter((item) => item.id).sort(comparePlacesForDisplay)
	uni.setStorageSync(getPlaceStorageKey(kitchenId), normalizedPlaces)
	return normalizedPlaces
}

function upsertPlaceInCache(place = {}) {
	const normalized = normalizePlace(place)
	const kitchenId = normalized.kitchenId || getCurrentKitchenId()
	if (!kitchenId || !normalized.id) return normalized
	const current = loadPlacesForKitchen(kitchenId)
	const nextPlaces = [normalized, ...current.filter((item) => item.id !== normalized.id)]
	savePlacesForKitchen(kitchenId, nextPlaces)
	return normalized
}

function removePlaceFromCache(placeId, kitchenId = getCurrentKitchenId()) {
	const current = loadPlacesForKitchen(kitchenId)
	savePlacesForKitchen(kitchenId, current.filter((item) => item.id !== placeId))
}

export function getCachedPlaces(kitchenId = getCurrentKitchenId()) {
	return loadPlacesForKitchen(kitchenId)
}

export function getCachedPlaceById(placeId, kitchenId = getCurrentKitchenId()) {
	return loadPlacesForKitchen(kitchenId).find((item) => item.id === placeId) || null
}

export async function syncPlaces(filters = {}) {
	const session = await ensureSession()
	const kitchenId = Number(session?.currentKitchenId) || 0
	if (!kitchenId) return []

	const items = await listPlaces(kitchenId, filters)
	return savePlacesForKitchen(kitchenId, items)
}

export async function loadPlaces(options = {}) {
	const { forceRefresh = false, filters = {} } = options
	await ensureSession()

	if (!forceRefresh) {
		const cached = getCachedPlaces()
		if (cached.length) {
			return cached
		}
	}

	return syncPlaces(filters)
}

export async function getPlaceById(placeId, options = {}) {
	const { preferCache = true } = options
	await ensureSession()

	if (preferCache) {
		const cached = getCachedPlaceById(placeId)
		if (cached) return cached
	}

	const item = await getPlaceDetail(placeId)
	return upsertPlaceInCache(item)
}

function buildPlacePayload(draft = {}) {
	return {
		name: normalizeString(draft.name),
		type: normalizeOption(draft.type, placeTypeOptions, 'food'),
		address: normalizeString(draft.address),
		latitude: normalizeNumber(draft.latitude),
		longitude: normalizeNumber(draft.longitude),
		price: normalizeString(draft.price),
		source: normalizeOption(draft.source, placeSourceOptions, 'manual'),
		sourceUrl: normalizeString(draft.sourceUrl),
		imageUrls: normalizeImageList(draft.imageUrls || draft.images),
		status: normalizeOption(draft.status, placeStatusOptions, 'want'),
		tags: normalizeTextList(draft.tags, 8),
		note: normalizeString(draft.note),
		// 新增字段
		phone: normalizeString(draft.phone),
		revisitRating: Number(draft.revisitRating) || 0,
		recommendedItems: normalizeTextList(draft.recommendedItems, 12),
		externalProvider: normalizeString(draft.externalProvider),
		externalPoiId: normalizeString(draft.externalPoiId),
		rating: normalizeString(draft.rating),
		diningTips: normalizeString(draft.diningTips),
		scenes: normalizeTextList(draft.scenes, 8),
		bestTime: normalizeString(draft.bestTime),
		duration: normalizeString(draft.duration),
		companionTags: normalizeTextList(draft.companionTags, 6),
		parkingNote: normalizeString(draft.parkingNote)
	}
}

export async function createPlaceFromDraft(draft = {}) {
	const session = await ensureSession()
	const kitchenId = Number(session?.currentKitchenId) || 0
	const imageUrls = await ensureUploadedImages(draft.images || draft.imageUrls || [])
	const payload = buildPlacePayload({
		...draft,
		imageUrls
	})
	const item = await createPlace(kitchenId, payload)
	return upsertPlaceInCache(item)
}

export async function updatePlaceById(placeId, updates = {}) {
	const current = await getPlaceById(placeId)
	const hasImages =
		Object.prototype.hasOwnProperty.call(updates, 'images') ||
		Object.prototype.hasOwnProperty.call(updates, 'imageUrls')
	const imageUrls = hasImages
		? await ensureUploadedImages(updates.images || updates.imageUrls || [])
		: current.imageUrls
	const payload = buildPlacePayload({
		...current,
		...updates,
		imageUrls
	})
	const item = await updatePlace(placeId, payload)
	return upsertPlaceInCache(item)
}

export async function updatePlaceStatusById(placeId, nextStatus, experienceData = {}) {
	const item = await updatePlaceStatus(placeId, nextStatus, experienceData)
	return upsertPlaceInCache(item)
}

export async function deletePlaceById(placeId) {
	const current = getCachedPlaceById(placeId)
	await ensureSession()
	await deletePlace(placeId)
	removePlaceFromCache(placeId, current?.kitchenId || getCurrentKitchenId())
}
