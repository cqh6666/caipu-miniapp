import { updateRecipeById } from '../../utils/recipe-store'
import { buildRecipeImageVersion, getRecipeImageSources } from '../../utils/recipe-model'
import { buildImageCacheKey, createImageDisplayController } from '../../utils/image-cache'

export { buildRecipeImageVersion } from '../../utils/recipe-model'

export function buildRecipeImageCacheEntries(recipe = {}, buildCacheKey) {
	const images = getRecipeImageSources(recipe)
	const version = buildRecipeImageVersion(recipe)
	return images
		.map((url) => ({ url: String(url || '').trim(), version, cacheKey: buildCacheKey(url, version) }))
		.filter((entry) => entry.url)
}

export function buildFlowchartImageCacheEntry(recipe = {}, buildCacheKey) {
	const url = String(recipe?.flowchartImageUrl || '').trim()
	const version = String(recipe?.flowchartUpdatedAt || url).trim()
	return { url, version, cacheKey: buildCacheKey(url, version) }
}

export function resolveVisibleImageIndex(visibleList = [], sourceImages = [], version = '', buildCacheKey, visibleIndex = 0) {
	if (!Array.isArray(visibleList) || visibleIndex < 0 || visibleIndex >= visibleList.length) return -1
	const targetKey = visibleList[visibleIndex]?.cacheKey
	if (!targetKey) return -1
	return sourceImages.findIndex((url) => buildCacheKey(url, version) === targetKey)
}

export function createRecipeImageController(host) {
	const displayController = createImageDisplayController({
		concurrency: 2,
		onState: ({ cachedMap, fallbackMap, hiddenMap }) => {
			host.cachedRecipeImageMap = cachedMap
			host.recipeImageFallbackMap = fallbackMap
			host.recipeImageHiddenMap = hiddenMap
		}
	})

	function originalIndex(visibleIndex = host.heroImageIndex) {
		return resolveVisibleImageIndex(
			host.displayRecipeImages,
			host.recipeImages,
			host.recipeImageVersion,
			buildImageCacheKey,
			visibleIndex
		)
	}

	async function syncRecipeCache(recipe = host.recipe) {
		const entries = buildRecipeImageCacheEntries(recipe || {}, buildImageCacheKey)
		return displayController.sync(entries.map((entry) => ({ ...entry, key: entry.cacheKey })))
	}

	async function handleImageError(image = {}) {
		const remoteURL = String(image?.remoteURL || '').trim()
		if (!remoteURL) return
		const cacheKey = String(image?.cacheKey || buildImageCacheKey(remoteURL, host.recipeImageVersion)).trim()
		const result = await displayController.handleError({ key: cacheKey, displayURL: image?.displayURL })
		if (result === 'hidden') host.heroImageIndex = 0
	}

	async function saveHeroImages(imagePaths = []) {
		const incoming = (Array.isArray(imagePaths) ? imagePaths : [imagePaths]).filter(Boolean)
		if (!incoming.length || !host.recipeId || host.isUploadingHeroImage) return
		host.isUploadingHeroImage = true
		uni.showLoading({ title: '上传中', mask: true })
		try {
			const nextImages = [...host.visibleRecipeSourceImages]
			incoming.forEach((path) => {
				if (path && !nextImages.includes(path) && nextImages.length < host.maxRecipeImages) nextImages.push(path)
			})
			host.applyRecipe(await updateRecipeById(host.recipeId, { images: nextImages }))
			uni.showToast({ title: `已添加 ${incoming.length} 张`, icon: 'none' })
		} catch (error) {
			uni.showToast({ title: error?.message || '上传失败', icon: 'none' })
		} finally {
			host.isUploadingHeroImage = false
			uni.hideLoading()
		}
	}

	function chooseHeroImages() {
		if (host.isPublicView || !host.recipe || host.isUploadingHeroImage) return
		const remaining = Math.max(host.maxRecipeImages - host.visibleRecipeSourceImages.length, 0)
		if (!remaining) return
		uni.chooseImage({
			count: remaining,
			sizeType: ['compressed'],
			sourceType: ['album', 'camera'],
			success: ({ tempFilePaths }) => saveHeroImages(tempFilePaths || [])
		})
	}

	function previewCurrent() {
		const urls = host.displayRecipeImages.map((item) => item.displayURL).filter(Boolean)
		if (urls.length) uni.previewImage({ current: urls[host.heroImageIndex] || urls[0], urls })
	}

	async function setCurrentAsCover() {
		if (!host.canSetCurrentAsCover || !host.recipeId || host.isUploadingHeroImage) return
		const fromIndex = originalIndex()
		const images = [...host.recipeImages]
		if (fromIndex <= 0 || fromIndex >= images.length) return
		const [picked] = images.splice(fromIndex, 1)
		images.unshift(picked)
		host.isUploadingHeroImage = true
		uni.showLoading({ title: '设置中', mask: true })
		try {
			host.applyRecipe(await updateRecipeById(host.recipeId, { images }))
			host.heroImageIndex = 0
			uni.vibrateShort?.({ type: 'medium' })
			host.showActionFeedback({ tone: 'done', title: '已设为封面' })
		} catch (error) {
			uni.showToast({ title: error?.message || '设置失败，请重试', icon: 'none' })
		} finally {
			host.isUploadingHeroImage = false
			uni.hideLoading()
		}
	}

	async function deleteCurrent() {
		if (!host.canDeleteCurrentImage || !host.recipeId || host.isUploadingHeroImage) return
		const removeIndex = originalIndex()
		const images = [...host.recipeImages]
		if (removeIndex < 0 || removeIndex >= images.length) return
		images.splice(removeIndex, 1)
		host.isUploadingHeroImage = true
		uni.showLoading({ title: '删除中', mask: true })
		try {
			host.applyRecipe(await updateRecipeById(host.recipeId, { images }))
			host.heroImageIndex = Math.min(host.heroImageIndex, Math.max(host.displayRecipeImages.length - 1, 0))
			uni.vibrateShort?.({ type: 'light' })
			host.showActionFeedback({ tone: 'done', title: '已删除' })
		} catch (error) {
			uni.showToast({ title: error?.message || '删除失败，请重试', icon: 'none' })
		} finally {
			host.isUploadingHeroImage = false
			uni.hideLoading()
		}
	}

	return {
		cancel: displayController.cancel,
		chooseHeroImages,
		deleteCurrent,
		handleImageError,
		originalIndex,
		previewCurrent,
		saveHeroImages,
		setCurrentAsCover,
		syncRecipeCache
	}
}
