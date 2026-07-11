import { buildRecipeCard, buildRecipeSearchText } from './recipe-card'
import { buildRecipeCoverVersion, extractRecipeImages } from './recipe-card'
import { loadPublicAppConfig } from '../../utils/public-app-config-api'
import { buildImageCacheKey, getCachedImagePath, invalidateCachedImage, warmImageCache } from '../../utils/image-cache'
import { mealTypeLabelMap, toggleRecipeStatusById } from '../../utils/recipe-store'
import { formatMealOrderHeaderTitle, normalizeMealOrderDate } from './meal-order'
import { writeRecentSearches } from './storage'
import { MAX_RECENT_SEARCHES, searchSuggestionKeywordsByMeal, statusMap } from './constants'
import { countPlacesByStatus } from './use-place-library'

export function filterRecipes(recipes = [], options = {}) {
	const {
		mealType = '',
		status = 'all',
		keyword = '',
		mealOrderMode = false
	} = options
	const normalizedKeyword = String(keyword || '').trim().toLowerCase()
	return recipes.filter((recipe) => {
		const matchedMealType = mealOrderMode || recipe.mealType === mealType
		const matchedStatus = mealOrderMode || status === 'all' || recipe.status === status
		const matchedKeyword = !normalizedKeyword || buildRecipeSearchText(recipe).includes(normalizedKeyword)
		return matchedMealType && matchedStatus && matchedKeyword
	})
}

export function buildRecipeCards(recipes = [], coverMap = {}) {
	return recipes.map((recipe) => buildRecipeCard(recipe, coverMap))
}

export function buildRandomPickPool(recipes = [], options = {}) {
	const { mealType = '', status = 'all', keyword = '' } = options
	const normalizedKeyword = String(keyword || '').trim().toLowerCase()
	return recipes.filter((recipe) => {
		if (mealType && recipe.mealType !== mealType) return false
		if (status !== 'all' && recipe.status !== status) return false
		return !normalizedKeyword || buildRecipeSearchText(recipe).includes(normalizedKeyword)
	})
}

export function pickRandomRecipe(pool = [], excludeRecipeId = '', random = Math.random) {
	if (!pool.length) return null
	const alternatives = pool.filter((recipe) => recipe.id !== excludeRecipeId)
	const candidates = alternatives.length ? alternatives : pool
	return candidates[Math.floor(random() * candidates.length)] || candidates[0] || null
}

export function countRecipesByMealType(recipes = [], mealType = '') {
	return recipes.filter((recipe) => recipe.mealType === mealType).length
}

export function countRecipesByStatus(recipes = [], mealType = '', status = 'all') {
	const list = recipes.filter((recipe) => recipe.mealType === mealType)
	return status === 'all' ? list.length : list.filter((recipe) => recipe.status === status).length
}

export function createSearchBlurController(onBlur, delay = 160) {
	let timer = null
	return {
		schedule() {
			this.cancel()
			timer = setTimeout(() => {
				timer = null
				onBlur()
			}, delay)
		},
		cancel() {
			if (!timer) return
			clearTimeout(timer)
			timer = null
		},
		hasPending() {
			return !!timer
		}
	}
}

export const recipeLibraryMethods = {
	clearRecipeStatusFeedbackTimer() {
	if (!this.recipeStatusFeedbackTimer) return
	clearTimeout(this.recipeStatusFeedbackTimer)
	this.recipeStatusFeedbackTimer = null
},
clearRecipeStatusFeedback() {
	this.clearRecipeStatusFeedbackTimer()
	this.recipeStatusFeedbackVisible = false
	this.recipeStatusFeedbackTone = ''
	this.recipeStatusFeedbackTitle = ''
	this.recipeStatusFeedbackRecipeTitle = ''
	this.recipeStatusFeedbackShowSparkles = false
},
buildTonightPickPool() {
	const visible = buildRandomPickPool(this.recipes, {
		mealType: this.activeMealType,
		status: this.activeStatus,
		keyword: this.trimmedSearchKeyword
	})
	if (!visible.length) return []
	if (this.activeStatus === 'all') {
		const wishlistVisible = visible.filter((recipe) => recipe.status === 'wishlist')
		if (wishlistVisible.length) return wishlistVisible
	}
	return visible
},
buildTonightPickContext(pool = []) {
	if (!pool.length) return ''
	if (this.hasSearchKeyword) {
		return `根据“${this.trimmedSearchKeyword}”挑了一道`
	}
	if (this.activeStatus !== 'all') {
		return `从${this.currentMealLabel}的${this.currentStatusLabel}里挑了一道`
	}
	if (pool.every((recipe) => recipe.status === 'wishlist')) {
		return `先从${this.currentMealLabel}里想吃的菜里挑了一道`
	}
	return `从${this.currentMealLabel}里挑了一道`
},
pickTonightRecipe(pool = [], excludeRecipeId = '') {
	return pickRandomRecipe(Array.isArray(pool) ? pool.filter(Boolean) : [], excludeRecipeId)
},
presentTonightPick(recipe = null, pool = [], contextText = '', motionMode = 'enter') {
	if (!recipe?.id) return
	this.randomPickRecipeId = recipe.id
	this.randomPickPoolRecipeIds = pool.map((item) => item.id).filter(Boolean)
	this.randomPickContextText = contextText
	this.randomPickMotionMode = motionMode === 'swap' ? 'swap' : 'enter'
	this.randomPickTick += 1
	this.showRandomPickSheet = true
	try {
		uni.vibrateShort({
			type: 'light'
		})
	} catch (_) {
		// Ignore unsupported vibration capabilities and keep the picker path stable.
	}
},
closeRandomPickSheet() {
	this.showRandomPickSheet = false
	this.randomPickRecipeId = ''
	this.randomPickContextText = ''
	this.randomPickPoolRecipeIds = []
	this.randomPickMotionMode = 'enter'
},
rerollTonightPick() {
	const pool = this.randomPickPoolRecipeIds
		.map((recipeId) => this.recipes.find((recipe) => recipe.id === recipeId))
		.filter(Boolean)
	if (pool.length < 2) return
	const picked = this.pickTonightRecipe(pool, this.randomPickRecipeId)
	this.presentTonightPick(picked, pool, this.randomPickContextText, 'swap')
},
openRandomPickDetail(recipeId = '') {
	const targetRecipeId = String(recipeId || this.randomPickRecipeId || '').trim()
	if (!targetRecipeId) return
	this.closeRandomPickSheet()
	setTimeout(() => {
		this.openRecipeDetail(targetRecipeId)
	}, 140)
},
playRecipeStatusHaptic(nextStatus = 'wishlist') {
	const vibrationType = nextStatus === 'done' ? 'medium' : 'light'
	try {
		uni.vibrateShort({
			type: vibrationType
		})
	} catch (_) {
		try {
			uni.vibrateShort()
		} catch (__) {
			// Ignore unsupported vibration capabilities to keep the toggle path stable.
		}
	}
},
showLibraryActionFeedback(options = {}) {
	const tone = String(options?.tone || 'done').trim() || 'done'
	const title = String(options?.title || '').trim()
	const description = String(options?.description || '').trim()
	const duration = Math.max(900, Number(options?.duration) || 1440)
	const showSparkles = !!options?.showSparkles
	if (!title) return
	this.clearRecipeStatusFeedbackTimer()
	this.recipeStatusFeedbackTone = tone
	this.recipeStatusFeedbackTitle = title
	this.recipeStatusFeedbackRecipeTitle = description
	this.recipeStatusFeedbackShowSparkles = showSparkles
	this.recipeStatusFeedbackVisible = true
	this.recipeStatusFeedbackTick += 1
	this.recipeStatusFeedbackTimer = setTimeout(() => {
		this.recipeStatusFeedbackVisible = false
		this.recipeStatusFeedbackTimer = null
	}, duration)
},
showRecipeStatusFeedback(recipe = {}, nextStatus = 'wishlist') {
	const tone = nextStatus === 'done' ? 'done' : 'wishlist'
	this.showLibraryActionFeedback({
		tone,
		title: tone === 'done' ? '已标记吃过' : '已改回想吃',
		description: String(recipe?.title || '').trim() || '这道菜',
		duration: tone === 'done' ? 1680 : 1440,
		showSparkles: tone === 'done'
	})
},
clearMealOrderModeMotionTimer() {
	if (!this.mealOrderModeMotionTimer) return
	clearTimeout(this.mealOrderModeMotionTimer)
	this.mealOrderModeMotionTimer = null
},
queueMealOrderModeMotion(state = '') {
	const nextState = state === 'leaving' ? 'leaving' : 'entering'
	this.clearMealOrderModeMotionTimer()
	this.mealOrderModeMotionState = nextState
	this.mealOrderModeMotionTimer = setTimeout(() => {
		this.mealOrderModeMotionState = ''
		this.mealOrderModeMotionTimer = null
	}, nextState === 'leaving' ? 220 : 260)
},
bumpRecipeListMotion() {
	this.recipeListMotionTick += 1
},
handleMealTypeTabChange(value) {
	if (!this.mealTabs.some((tab) => tab.value === value) || this.activeMealType === value) return
	const oldIdx = this.mealTabs.findIndex((tab) => tab.value === this.activeMealType)
	const newIdx = this.mealTabs.findIndex((tab) => tab.value === value)
	this.triggerToolbarBounce(newIdx > oldIdx ? 'right' : 'left')
	this.activeMealType = value
	this.bumpRecipeListMotion()
},
handleStatusTabChange(value) {
	if (!this.statusTabs.some((tab) => tab.value === value) || this.activeStatus === value) return
	this.activeStatus = value
	this.bumpRecipeListMotion()
},
handleEmptyStatePrimary() {
	if (this.hasSearchKeyword) {
		if (this.isDietAssistantEntryEnabled) {
			this.openDietAssistantSheet(this.buildSearchNoResultPrompt())
		} else {
			this.openAddSheet()
		}
		return
	}
	if (this.activeStatus !== 'all') {
		this.handleStatusTabChange('all')
		return
	}
	this.openAddSheet()
},
handleEmptyStateSecondary() {
	if (this.hasSearchKeyword) {
		this.searchKeyword = ''
		return
	}
	if (this.activeStatus !== 'all') {
		this.openAddSheet()
	}
},
buildSearchNoResultPrompt() {
	const keyword = this.trimmedSearchKeyword
	if (!keyword) {
		return '我想找一道菜的简单做法，可以给我一个适合家常复刻的版本吗？'
	}
	return `我在美食库搜不到「${keyword}」，想做这道菜，可以给我一个简单做法吗？`
},
refreshPublicAppConfig() {
	const requestID = ++this.publicAppConfigRequestID
	loadPublicAppConfig()
		.then((config) => {
			if (requestID !== this.publicAppConfigRequestID) return
			this.publicAppConfig = config
		})
		.catch((error) => {
			console.warn?.('load public app config failed', error)
		})
},
handlePrimaryFabTap() {
	if (this.isExplorePrimaryFab) {
		this.openPlaceCreateSheet()
		return
	}
	if (this.isDietAssistantEntryEnabled) {
		this.openDietAssistantSheet()
		return
	}
	this.openAddSheet()
},
triggerToolbarBounce(direction = 'right') {
	const cls = `toolbar--bounce-${direction === 'left' ? 'left' : 'right'}`
	this.toolbarBounceClass = ''
	if (this.toolbarBounceTimer) {
		clearTimeout(this.toolbarBounceTimer)
		this.toolbarBounceTimer = null
	}
	this.$nextTick(() => {
		this.toolbarBounceClass = cls
		this.toolbarBounceTimer = setTimeout(() => {
			this.toolbarBounceClass = ''
			this.toolbarBounceTimer = null
		}, 160)
	})
},
clearRecipeReturnFocusTimer() {
	if (!this.returnFocusTimer) return
	clearTimeout(this.returnFocusTimer)
	this.returnFocusTimer = null
},
clearRecipeReturnFocus() {
	this.clearRecipeReturnFocusTimer()
	this.returnFocusRecipeId = ''
	this.returnFocusPendingRecipeId = ''
},
showRecipeReturnFocus(recipeId = '') {
	const targetRecipeId = String(recipeId || '').trim()
	if (!targetRecipeId || this.activeSection !== 'library') return
	this.clearRecipeReturnFocusTimer()
	this.returnFocusRecipeId = ''
	this.$nextTick(() => {
		this.returnFocusRecipeId = targetRecipeId
		this.returnFocusTimer = setTimeout(() => {
			this.returnFocusRecipeId = ''
			this.returnFocusTimer = null
		}, 1160)
	})
},
playPendingRecipeReturnFocus() {
	const targetRecipeId = String(this.returnFocusPendingRecipeId || '').trim()
	this.returnFocusPendingRecipeId = ''
	if (!targetRecipeId || this.activeSection !== 'library') return
	this.clearRecipeReturnFocusTimer()
	this.returnFocusTimer = setTimeout(() => {
		this.returnFocusTimer = null
		this.showRecipeReturnFocus(targetRecipeId)
	}, 120)
},
bumpMealOrderSpotlightMotion(direction = 'next') {
	this.mealOrderSpotlightMotionDirection = direction === 'previous' ? 'previous' : 'next'
	this.mealOrderSpotlightMotionTick += 1
},
setRecipeStatusPending(recipeId = '', pending = false) {
	const targetRecipeId = String(recipeId || '').trim()
	if (!targetRecipeId) return
	const nextPendingMap = {
		...this.recipeStatusPendingMap
	}
	if (pending) {
		nextPendingMap[targetRecipeId] = true
	} else {
		delete nextPendingMap[targetRecipeId]
	}
	this.recipeStatusPendingMap = nextPendingMap
},
patchLocalRecipeById(recipeId = '', updater = null) {
	const targetRecipeId = String(recipeId || '').trim()
	if (!targetRecipeId || typeof updater !== 'function') {
		return {
			found: false,
			previousRecipe: null,
			nextRecipe: null
		}
	}

	let previousRecipe = null
	let nextRecipe = null
	let changed = false
	const nextRecipes = this.recipes.map((recipe) => {
		if (recipe.id !== targetRecipeId) return recipe
		previousRecipe = recipe
		nextRecipe = updater(recipe)
		if (!nextRecipe) {
			nextRecipe = recipe
			return recipe
		}
		changed = nextRecipe !== recipe
		return nextRecipe
	})

	if (changed) {
		this.recipes = nextRecipes
	}

	return {
		found: !!previousRecipe,
		previousRecipe,
		nextRecipe
	}
},
applyRecipes(recipes = []) {
	this.recipes = Array.isArray(recipes) ? recipes : []
	this.recipeCardCoverFallbackMap = {}
	this.recipeCardHiddenMap = {}
	this.syncRecipeCoverCache(this.recipes)
},
getRecipeCardDisplayCover(card = {}) {
	const recipeId = String(card?.id || '').trim()
	if (recipeId && this.recipeCardHiddenMap[recipeId]) return ''
	if (recipeId && this.recipeCardCoverFallbackMap[recipeId]) {
		return String(card?.remoteCover || '').trim()
	}
	return String(card?.cover || '').trim()
},
async handleRecipeCardImageError(card = {}) {
	const recipeId = String(card?.id || '').trim()
	if (!recipeId) return

	const displayedCover = this.getRecipeCardDisplayCover(card)
	const cachedCover = String(card?.cachedCover || '').trim()
	const remoteCover = String(card?.remoteCover || '').trim()

	if (
		cachedCover &&
		remoteCover &&
		displayedCover === cachedCover &&
		cachedCover !== remoteCover &&
		!this.recipeCardCoverFallbackMap[recipeId]
	) {
		this.recipeCardCoverFallbackMap = {
			...this.recipeCardCoverFallbackMap,
			[recipeId]: true
		}

		if (this.cachedRecipeCoverMap[recipeId]) {
			const nextCoverMap = { ...this.cachedRecipeCoverMap }
			delete nextCoverMap[recipeId]
			this.cachedRecipeCoverMap = nextCoverMap
		}

		try {
			await invalidateCachedImage(remoteCover, card.coverVersion)
		} catch (error) {
			// Ignore cache cleanup failures and keep the UI fallback path usable.
		}
		return
	}

	if (this.recipeCardHiddenMap[recipeId]) return
	this.recipeCardHiddenMap = {
		...this.recipeCardHiddenMap,
		[recipeId]: true
	}
}

}

export const recipeSearchMethods = {
clearSearchBlurTimer() {
	this.searchBlurController?.cancel()
},
handleSearchFocus() {
	this.clearSearchBlurTimer()
	this.isSearchFocused = true
},
handleSearchBlur() {
	this.searchBlurController?.schedule()
	this.rememberSearchKeyword()
},
handleSearchConfirm() {
	this.rememberSearchKeyword()
},
rememberSearchKeyword() {
	const keyword = this.trimmedSearchKeyword
	if (!keyword) return

	const nextKeywords = [keyword, ...this.recentSearches.filter((item) => item !== keyword)].slice(0, MAX_RECENT_SEARCHES)
	this.recentSearches = nextKeywords
	writeRecentSearches(nextKeywords)
},
applySearchKeyword(keyword = '') {
	const nextKeyword = String(keyword || '').trim()
	if (!nextKeyword) return

	this.clearSearchBlurTimer()
	this.searchKeyword = nextKeyword
	this.isSearchFocused = false
	this.rememberSearchKeyword()
	this.bumpRecipeListMotion()
},
clearSearchKeyword() {
	this.searchKeyword = ''
	this.clearSearchBlurTimer()
	this.isSearchFocused = true
	this.bumpRecipeListMotion()
},
buildRecipeCoverCacheEntries(recipes = []) {
	return (Array.isArray(recipes) ? recipes : [])
		.map((recipe) => {
			const images = extractRecipeImages(recipe)
			const cover = images[0] || ''
			const version = buildRecipeCoverVersion(recipe)
			if (!cover || !recipe.id) return null
			return {
				recipeId: recipe.id,
				url: cover,
				version,
				cacheKey: buildImageCacheKey(cover, version)
			}
		})
		.filter(Boolean)
},
async syncRecipeCoverCache(recipes = []) {
	const entries = this.buildRecipeCoverCacheEntries(recipes)
	const requestID = this.recipeCoverCacheRequestID + 1
	this.recipeCoverCacheRequestID = requestID

	if (!entries.length) {
		this.cachedRecipeCoverMap = {}
		this.recipeCardCoverFallbackMap = {}
		this.recipeCardHiddenMap = {}
		return
	}

	const cachedEntries = await Promise.all(
		entries.map(async (entry) => ({
			recipeId: entry.recipeId,
			localPath: await getCachedImagePath(entry.url, entry.version)
		}))
	)

	if (requestID !== this.recipeCoverCacheRequestID) return

	const nextCoverMap = {}
	const nextFallbackMap = { ...this.recipeCardCoverFallbackMap }
	const nextHiddenMap = { ...this.recipeCardHiddenMap }
	cachedEntries.forEach((entry) => {
		if (!entry.localPath) return
		nextCoverMap[entry.recipeId] = entry.localPath
		delete nextFallbackMap[entry.recipeId]
		delete nextHiddenMap[entry.recipeId]
	})
	this.cachedRecipeCoverMap = nextCoverMap
	this.recipeCardCoverFallbackMap = nextFallbackMap
	this.recipeCardHiddenMap = nextHiddenMap

	const recipeIdsByCacheKey = entries.reduce((result, entry) => {
		if (!result[entry.cacheKey]) {
			result[entry.cacheKey] = []
		}
		result[entry.cacheKey].push(entry.recipeId)
		return result
	}, {})

	warmImageCache(entries, {
		concurrency: 2,
		onResolved: ({ cacheKey, localPath }) => {
			if (requestID !== this.recipeCoverCacheRequestID || !localPath) return
			const recipeIds = recipeIdsByCacheKey[cacheKey] || []
			if (!recipeIds.length) return

			let changed = false
			const updatedCoverMap = { ...this.cachedRecipeCoverMap }
			const updatedFallbackMap = { ...this.recipeCardCoverFallbackMap }
			const updatedHiddenMap = { ...this.recipeCardHiddenMap }
			recipeIds.forEach((recipeId) => {
				if (updatedCoverMap[recipeId] === localPath) return
				updatedCoverMap[recipeId] = localPath
				delete updatedFallbackMap[recipeId]
				delete updatedHiddenMap[recipeId]
				changed = true
			})

			if (changed) {
				this.cachedRecipeCoverMap = updatedCoverMap
				this.recipeCardCoverFallbackMap = updatedFallbackMap
				this.recipeCardHiddenMap = updatedHiddenMap
			}
		}
	})
},
applySession(session = getSessionSnapshot()) {
	const snapshot = session || getSessionSnapshot()
	const previousKitchenId = Number(this.currentKitchenId) || 0
	this.currentUser = snapshot?.user || null
	this.kitchenOptions = Array.isArray(snapshot?.kitchens) ? snapshot.kitchens : []
	this.currentKitchenName = snapshot?.currentKitchen?.name || ''
	this.currentKitchenRole = snapshot?.currentKitchen?.role || ''
	const nextKitchenId = Number(snapshot?.currentKitchenId) || 0
	this.currentKitchenId = nextKitchenId
		if (nextKitchenId !== this.kitchenMembersKitchenId) {
			this.kitchenMembers = []
			this.kitchenMembersKitchenId = nextKitchenId
		}
		if (previousKitchenId !== nextKitchenId) {
			this.mealOrderSyncContextID += 1
			this.mealOrderStoreLoadedKitchenId = 0
			this.mealOrderLocalVersion += 1
			this.resetMealOrderState()
			this.selectedPlaceId = ''
			this.showPlaceDetailSheet = false
			this.showPlaceEditSheet = false
			// 切换空间时清空上一空间的后端统计快照，避免串味（本地聚合会随新空间数据重新计算）。
			this.spaceStatsRemote = null
			this.spaceStatsRemoteKitchenId = 0
			this.spaceStatsRemoteError = ''
			this.spaceStatsAutoSyncKitchenId = 0
		}
		if (!nextKitchenId) {
			this.mealOrderStoreLoadedKitchenId = 0
			this.resetMealOrderState()
			this.applyPlaces([])
		} else {
			this.applyPlaces(getCachedPlaces(nextKitchenId))
			if (this.mealOrderStoreLoadedKitchenId !== nextKitchenId) {
				this.loadMealOrderStore({ silent: true })
			}
	}
	this.activeInvite = null
	this.inviteCodeCopied = false
}

}

export const recipeStatusMethods = {
mealTypeCount(type) {
	return countRecipesByMealType(this.recipes, type)
},
statusCount(status) {
	return countRecipesByStatus(this.recipes, this.activeMealType, status)
},
placeStatusCount(status) {
	return countPlacesByStatus(this.places, status)
},
resetLibraryFilters() {
	this.activeStatus = 'all'
	this.searchKeyword = ''
	this.clearSearchBlurTimer()
	this.isSearchFocused = false
	this.bumpRecipeListMotion()
},
openRecipeDetail(recipeId) {
	const targetRecipeId = String(recipeId || '').trim()
	this.returnFocusPendingRecipeId = this.activeSection === 'library' ? targetRecipeId : ''
	uni.navigateTo({
		url: `/pages/recipe-detail/index?id=${targetRecipeId}`,
		fail: () => {
			this.returnFocusPendingRecipeId = ''
		}
	})
},
openMealOrderDetail(record = {}) {
	const planDate = normalizeMealOrderDate(record?.planDate || '')
	const type = String(record?.type || '').trim() === 'draft' ? 'draft' : 'submitted'
	if (!planDate) {
		uni.showToast({
			title: '这份菜单暂时打不开',
			icon: 'none'
		})
		return
	}
	uni.navigateTo({
		url: `/pages/meal-plan-detail/index?planDate=${encodeURIComponent(planDate)}&type=${type}`
	})
},
openMealOrderRecipeDetail(item = {}) {
	const recipeId = String(item?.recipeId || '').trim()
	if (!recipeId) {
		uni.showToast({
			title: '这道菜暂时打不开',
			icon: 'none'
		})
		return
	}
	this.openRecipeDetail(recipeId)
},
nextStatusText(status) {
	return status === 'done' ? '标记想吃' : '标记吃过'
},
toggleRecipeStatus(recipeId) {
	this.toggleRecipeStatusAsync(recipeId)
},
async toggleRecipeStatusAsync(recipeId) {
	const targetRecipeId = String(recipeId || '').trim()
	if (!targetRecipeId || this.recipeStatusPendingMap[targetRecipeId]) return

	const currentRecipe = this.recipes.find((recipe) => recipe.id === targetRecipeId)
	if (!currentRecipe) return

	const nextStatus = currentRecipe.status === 'done' ? 'wishlist' : 'done'
	this.setRecipeStatusPending(targetRecipeId, true)
	const optimisticUpdate = this.patchLocalRecipeById(targetRecipeId, (recipe) => ({
		...recipe,
		status: nextStatus
	}))
	if (!optimisticUpdate.found || !optimisticUpdate.previousRecipe) {
		this.setRecipeStatusPending(targetRecipeId, false)
		return
	}

	this.playRecipeStatusHaptic(nextStatus)

	try {
		const updatedRecipe = await toggleRecipeStatusById(targetRecipeId)
		this.patchLocalRecipeById(targetRecipeId, (recipe) => ({
			...recipe,
			...updatedRecipe
		}))
		this.showRecipeStatusFeedback(
			updatedRecipe || optimisticUpdate.nextRecipe,
			updatedRecipe?.status || nextStatus
		)
	} catch (error) {
		this.patchLocalRecipeById(targetRecipeId, () => optimisticUpdate.previousRecipe)
		uni.showToast({
			title: error?.message || '更新状态失败',
			icon: 'none'
		})
	} finally {
		this.setRecipeStatusPending(targetRecipeId, false)
	}
}

}

export const recipeListComputed = {
	filteredRecipes() {
		return filterRecipes(this.recipes, {
			mealType: this.activeMealType,
			status: this.activeStatus,
			keyword: this.trimmedSearchKeyword,
			mealOrderMode: this.isLibraryMealOrderMode
		})
	},
	recipeCards() {
		return buildRecipeCards(this.filteredRecipes, this.cachedRecipeCoverMap)
	},
	randomPickRecipe() {
		return this.recipes.find((recipe) => recipe.id === this.randomPickRecipeId) || null
	},
	randomPickCard() {
		if (!this.randomPickRecipe) return null
		return buildRecipeCard(this.randomPickRecipe, this.cachedRecipeCoverMap)
	},
	randomPickCoverSrc() {
		return this.randomPickCard ? this.getRecipeCardDisplayCover(this.randomPickCard) : ''
	},
	randomPickCanReroll() {
		return this.randomPickPoolRecipeIds.length > 1
	},
	randomPickRevealKey() {
		return `${this.randomPickRecipeId || 'idle'}:${this.randomPickTick}`
	},
	recipeStatusFeedbackKey() {
		return `${this.recipeStatusFeedbackTone || 'idle'}:${this.recipeStatusFeedbackTick}`
	},
	searchAssistKeywords() {
		const keyword = this.trimmedSearchKeyword
		const recentKeywords = this.recentSearches
			.filter((item) => item !== keyword)
			.slice(0, 4)
		if (recentKeywords.length) {
			return recentKeywords
		}

		return (searchSuggestionKeywordsByMeal[this.activeMealType] || searchSuggestionKeywordsByMeal.main)
			.filter((item) => item !== keyword)
			.slice(0, 4)
	},
	searchAssistLabel() {
		const recentKeywords = this.recentSearches
			.filter((item) => item !== this.trimmedSearchKeyword)
			.slice(0, 4)
		return recentKeywords.length ? '最近搜索' : '可以试试'
	},
	searchPlaceholderText() {
		return this.isLibraryMealOrderMode ? '搜索菜名' : '搜菜名 / 食材'
	},
	showSearchAssist() {
		return this.isSearchFocused && !this.hasSearchKeyword && this.searchAssistKeywords.length > 0
	},
	currentFilterSummary() {
		const parts = [this.currentMealLabel]
		if (this.activeStatus !== 'all') {
			parts.push(this.currentStatusLabel)
		}
		if (this.hasSearchKeyword) {
			parts.push(`搜“${this.trimmedSearchKeyword}”`)
		}
		parts.push(`${this.filteredRecipes.length}道`)
		return parts.join(' · ')
	},
	canResetLibraryFilters() {
		return this.activeStatus !== 'all' || this.hasSearchKeyword
	},
	isDietAssistantEntryEnabled() {
		return this.publicAppConfig?.features?.dietAssistantEnabled !== false
	},
	isExplorePrimaryFab() {
		return this.activeSection === 'library' && this.appMode === 'explore'
	},
	emptyStateKind() {
		if (this.hasSearchKeyword) return 'search-no-results'
		if (this.activeStatus !== 'all') return 'status-empty'
		return 'meal-empty'
	},
	emptyStateTitle() {
		if (this.hasSearchKeyword) {
			return `库里没找到“${this.trimmedSearchKeyword}”相关的菜谱`
		}
		if (this.activeStatus === 'all') {
			return `还没有${this.currentMealLabel}记录`
		}
		return `${this.currentMealLabel}里还没有${this.currentStatusLabel}的菜`
	},
	emptyStateDesc() {
		if (this.hasSearchKeyword) {
			return ''
		}
		if (this.activeStatus === 'all') {
			return `试试切换到另一类餐别，或者点下方按钮新增一道${this.currentMealLabel}。`
		}
		return `可以先把${this.currentMealLabel}里的菜标记为${this.currentStatusLabel}，或者点下方按钮回到全部。`
	},
	emptyStatePrimaryText() {
		if (this.hasSearchKeyword) return this.isDietAssistantEntryEnabled ? '问问 AI 怎么做' : '添加这道菜'
		if (this.activeStatus !== 'all') return '查看全部'
		return '添加这道菜'
	},
	emptyStatePrimaryIcon() {
		if (this.hasSearchKeyword) return this.isDietAssistantEntryEnabled ? '' : 'plus'
		if (this.activeStatus !== 'all') return 'list-dot'
		return 'plus'
	},
	emptyStatePrimaryIconSrc() {
		if (this.hasSearchKeyword && this.isDietAssistantEntryEnabled) return '/static/icons/sparkle-plus-warm.svg'
		return ''
	},
	emptyStateSecondaryText() {
		if (this.hasSearchKeyword) return ''
		if (this.activeStatus !== 'all') return '添加这道菜'
		return ''
	}
}

export const recipeSearchComputed = {
	doneRecipes() {
		return this.recipes.filter((recipe) => recipe.status === 'done')
	},
	trimmedSearchKeyword() {
		return String(this.searchKeyword || '').trim()
	},
	hasSearchKeyword() {
		return !!this.trimmedSearchKeyword
	}
}

export const recipeHeaderComputed = {
		currentMealLabel() {
		return this.mealTabs.find((tab) => tab.value === this.activeMealType)?.label || '早餐'
	},
	currentStatusLabel() {
		return this.statusMap[this.activeStatus]?.label || '全部'
	},
	libraryHeaderTitle() {
		return this.isLibraryMealOrderMode ? formatMealOrderHeaderTitle(this.mealOrderDate) : '美食库'
	},
	libraryHeaderSummary() {
		if (this.isLibraryMealOrderMode) {
			return ''
		}
		return this.librarySummary
	},
	wishlistRecipes() {
		return this.recipes.filter((recipe) => recipe.status === 'wishlist')
	}
}
