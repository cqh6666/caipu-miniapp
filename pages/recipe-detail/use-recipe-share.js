import { ensureRecipeShareTokenById } from '../../utils/recipe-store'
import { invalidateCachedImage } from '../../utils/image-cache'

export const FLOWCHART_VIEWER_STORAGE_KEY = 'recipe-flowchart-viewer-payload'
export function requestImageInfo(src = '') {
	const target = String(src || '').trim()
	if (!target) return Promise.resolve(null)
	return new Promise((resolve, reject) => uni.getImageInfo({ src: target, success: resolve, fail: reject }))
}
export function exportCanvasToTempFilePath(options = {}, component) {
	return new Promise((resolve, reject) => {
		uni.canvasToTempFilePath({ ...options, success: resolve, fail: reject }, component)
	})
}

export function buildRecipeShareConfig(options = {}) {
	const {
		channel = 'message', recipe = {}, recipeId = '', shareToken = '', publicViewToken = '',
		hasFlowchart = false, parsedStepCount = 0, flowchartShareImage = '', currentFlowchartImage = '',
		flowchartImageUrl = '', coverImage = ''
	} = options
	const dishName = String(recipe.title || '').trim() || '一道值得做的菜'
	const title = channel === 'message' && (hasFlowchart || parsedStepCount >= 3)
		? `${dishName} · 完整做法`
		: dishName
	const id = String(recipe.id || recipeId || '').trim()
	const token = String(shareToken || publicViewToken || '').trim()
	const tokenSegment = token ? `&shareToken=${token}` : ''
	const path = id ? `/pages/recipe-detail/index?id=${id}&from=share${tokenSegment}` : '/pages/index/index'
	const query = id ? `id=${id}&from=share${tokenSegment}` : ''
	const flowchart = String(flowchartShareImage || currentFlowchartImage || flowchartImageUrl || '').trim()
	const timelineFlowchart = String(flowchartImageUrl || '').trim()
	const visibleCover = String(coverImage || '').trim()
	const imageUrl = channel === 'timeline' ? visibleCover || timelineFlowchart : flowchart || visibleCover
	const config = channel === 'message' ? { title, path } : { title, query }
	if (imageUrl) config.imageUrl = imageUrl
	return config
}

export function createRecipeShareController(host) {
	function ensureToken() {
		if (host.isPublicView) return Promise.resolve(host.shareToken || null)
		if (host.shareToken) return Promise.resolve(host.shareToken)
		const recipeId = String(host.recipe?.id || host.recipeId || '').trim()
		if (!recipeId) return Promise.resolve(null)
		if (host._shareTokenEnsurePromise) return host._shareTokenEnsurePromise
		host._shareTokenEnsurePromise = ensureRecipeShareTokenById(recipeId)
			.then((token) => {
				if (token) host.shareToken = token
				return token || null
			})
			.catch(() => null)
			.finally(() => { host._shareTokenEnsurePromise = null })
		return host._shareTokenEnsurePromise
	}

	function config({ channel = 'message', flowchartShareImage = '' } = {}) {
		return buildRecipeShareConfig({
			channel,
			recipe: host.recipe || {},
			recipeId: host.recipeId,
			shareToken: host.shareToken,
			publicViewToken: host.publicViewToken,
			hasFlowchart: host.hasFlowchart,
			parsedStepCount: host.parsedSteps.length,
			flowchartShareImage,
			currentFlowchartImage: host.getCurrentFlowchartShareImagePath(),
			flowchartImageUrl: host.flowchartImageUrl,
			coverImage: host.visibleRecipeSourceImages?.[0] || ''
		})
	}

	async function configAsync({ channel = 'message' } = {}) {
		const shouldEnsureToken = !host.shareToken && !host.isPublicView && !!host.recipeId
		const imageTask = host.shouldPreferFlowchartShareCover(channel) && host.hasFlowchart
			? ensureFlowchartImage().catch(() => '')
			: Promise.resolve('')
		const [, flowchartShareImage] = await Promise.all([
			shouldEnsureToken ? ensureToken() : Promise.resolve(host.shareToken || host.publicViewToken || ''),
			imageTask
		])
		return config({ channel, flowchartShareImage })
	}

	async function createSquareImage() {
		if (typeof uni.createCanvasContext !== 'function') return ''
		const sourceURLs = host.buildFlowchartPreviewURLs()
		if (!sourceURLs.length) return ''
		const canvasId = '_flowchart_square_canvas'
		for (const sourceURL of sourceURLs) {
			try {
				const imageInfo = await requestImageInfo(sourceURL)
				const sourcePath = String(imageInfo?.path || sourceURL || '').trim()
				const width = Number(imageInfo?.width) || 0
				const height = Number(imageInfo?.height) || 0
				const size = Math.min(width, height)
				if (!sourcePath || !size) continue
				const ctx = uni.createCanvasContext(canvasId, host)
				ctx.clearRect?.(0, 0, size, size)
				ctx.drawImage(sourcePath, width > size ? -(width - size) / 2 : 0, height > size ? -(height - size) / 2 : 0, width, height)
				const tempFilePath = await new Promise((resolve) => {
					ctx.draw(false, async () => {
						try {
							const result = await exportCanvasToTempFilePath({
								canvasId, width: size, height: size, destWidth: size, destHeight: size
							}, host)
							resolve(String(result?.tempFilePath || '').trim())
						} catch (_) { resolve('') }
					})
				})
				if (tempFilePath) return tempFilePath
			} catch (_) {
				if (sourceURL === String(host.cachedFlowchartImagePath || '').trim()) {
					host.cachedFlowchartImagePath = ''
					try {
						await invalidateCachedImage(host.flowchartImageUrl, host.flowchartImageCacheVersion || host.buildFlowchartImageCacheVersion())
					} catch (_) {}
				}
			}
		}
		return ''
	}

	async function ensureFlowchartImage() {
		const entry = host.buildFlowchartImageCacheEntry()
		if (!entry.url) return ''
		const readyPath = host.getCurrentFlowchartShareImagePath()
		if (readyPath) return readyPath
		if (host._flowchartShareImagePromise && host.flowchartShareImagePendingKey === entry.cacheKey) return host._flowchartShareImagePromise
		host.flowchartShareImagePendingKey = entry.cacheKey
		const task = createSquareImage()
			.then((path) => {
				if (!path || host.buildFlowchartImageCacheEntry().cacheKey !== entry.cacheKey) return ''
				host.flowchartSquareImagePath = path
				host.flowchartSquareImageSourceKey = entry.cacheKey
				return path
			})
			.catch(() => '')
			.finally(() => {
				if (host.flowchartShareImagePendingKey === entry.cacheKey) host.flowchartShareImagePendingKey = ''
				if (host._flowchartShareImagePromise === task) host._flowchartShareImagePromise = null
			})
		host._flowchartShareImagePromise = task
		return task
	}

	return { config, configAsync, createSquareImage, ensureFlowchartImage, ensureToken }
}
