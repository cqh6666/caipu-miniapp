// 空间统计能力设计 §7.1 聚合模块
// 纯函数，不依赖 uni / 页面实例 / 网络请求，方便单测和后续迁移到后端接口。

const DAY_MS = 24 * 60 * 60 * 1000
const RECENT_WINDOW_DAYS = 30
const TOP_REVISIT_PLACE_LIMIT = 3
const TOP_TAG_LIMIT = 5

function toTimestamp(value) {
	if (!value) return 0
	const time = new Date(value).getTime()
	return Number.isFinite(time) ? time : 0
}

function toISODateKey(date) {
	if (!(date instanceof Date) || Number.isNaN(date.getTime())) return ''
	const year = date.getFullYear()
	const month = String(date.getMonth() + 1).padStart(2, '0')
	const day = String(date.getDate()).padStart(2, '0')
	return `${year}-${month}-${day}`
}

function isWithinRecentWindow(value, nowTime, days = RECENT_WINDOW_DAYS) {
	const time = toTimestamp(value)
	if (!time || !nowTime) return false
	return time <= nowTime && nowTime - time <= days * DAY_MS
}

function countByField(list, field, buckets) {
	const result = {}
	buckets.forEach((bucket) => {
		result[bucket] = 0
	})
	list.forEach((item) => {
		const value = item?.[field]
		if (Object.prototype.hasOwnProperty.call(result, value)) {
			result[value] += 1
		}
	})
	return result
}

function buildFrequencyTop(lists, limit = TOP_TAG_LIMIT) {
	const counters = new Map()
	lists.forEach((items) => {
		;(Array.isArray(items) ? items : []).forEach((raw) => {
			const label = String(raw || '').trim()
			if (!label) return
			const key = label.toLowerCase()
			const existing = counters.get(key)
			if (existing) {
				existing.count += 1
			} else {
				counters.set(key, { label, count: 1 })
			}
		})
	})

	return Array.from(counters.values())
		.sort((left, right) => right.count - left.count || left.label.localeCompare(right.label))
		.slice(0, limit)
}

function buildTopRevisitPlaces(places, limit = TOP_REVISIT_PLACE_LIMIT) {
	return places
		.filter((place) => Number(place?.revisitRating) >= 4)
		.sort((left, right) => {
			const ratingDelta = Number(right.revisitRating) - Number(left.revisitRating)
			if (ratingDelta) return ratingDelta
			return String(right.visitedAt || right.updatedAt || '').localeCompare(String(left.visitedAt || left.updatedAt || ''))
		})
		.slice(0, limit)
		.map((place) => ({
			id: place.id || '',
			name: place.name || '未命名店铺',
			revisitRating: Number(place.revisitRating) || 0,
			recommendedItems: Array.isArray(place.recommendedItems) ? place.recommendedItems.slice(0, 2) : [],
			imageUrl: (Array.isArray(place.imageUrls) && place.imageUrls[0]) || ''
		}))
}

function hasLocation(place) {
	const latitude = Number(place?.latitude) || 0
	const longitude = Number(place?.longitude) || 0
	return latitude !== 0 && longitude !== 0
}

function hasPoiMatch(place) {
	return !!String(place?.externalProvider || '').trim() && !!String(place?.externalPoiId || '').trim()
}

function hasExperience(place) {
	if (place?.status !== 'visited') return false
	if (Number(place?.revisitRating) > 0) return true
	return Array.isArray(place?.recommendedItems) && place.recommendedItems.length > 0
}

function buildRevisitRatingDistribution(places) {
	const distribution = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0 }
	places.forEach((place) => {
		const rating = Math.round(Number(place?.revisitRating) || 0)
		if (rating >= 1 && rating <= 5) {
			distribution[rating] += 1
		}
	})
	return distribution
}

function buildItemsByMealType(submitted) {
	const result = { breakfast: 0, main: 0 }
	submitted.forEach((record) => {
		;(Array.isArray(record?.items) ? record.items : []).forEach((item) => {
			const mealType = item?.mealTypeSnapshot || item?.mealType
			if (Object.prototype.hasOwnProperty.call(result, mealType)) {
				result[mealType] += 1
			}
		})
	})
	return result
}

function round2(value) {
	return Math.round((Number(value) || 0) * 100) / 100
}

function ratio(numerator, denominator) {
	if (!denominator) return 0
	return round2(numerator / denominator)
}

// 空 trends 结构，保证 V2 组件在本地聚合（无趋势）时也能安全读取。
export function createEmptyTrends() {
	return {
		recipeCreated: [],
		recipeDone: [],
		placeCreated: [],
		placeVisited: [],
		mealPlanSubmitted: []
	}
}

function average(values) {
	if (!values.length) return 0
	const total = values.reduce((sum, value) => sum + value, 0)
	return Math.round((total / values.length) * 10) / 10
}

function buildNextMealPlan(submitted, nowTime) {
	const todayKey = toISODateKey(new Date(nowTime))
	const upcoming = submitted
		.filter((record) => record?.planDate && record.planDate >= todayKey)
		.sort((left, right) => left.planDate.localeCompare(right.planDate))

	const next = upcoming[0]
	if (!next) return null

	return {
		planDate: next.planDate,
		dishCount: Array.isArray(next.items) ? next.items.length : 0,
		titles: (Array.isArray(next.items) ? next.items : [])
			.map((item) => item?.titleSnapshot || '')
			.filter(Boolean)
			.slice(0, 3)
	}
}

// 统一的待行动文案，供本地聚合与后端 stats adapter 复用（后端 action label 仍是过时的"本周/周末"文案，前端一律重新生成）。
function buildActionLabel(actionType, count) {
	switch (actionType) {
		case 'view-wishlist-recipes':
			return `想吃 ${count} 道菜可选`
		case 'view-want-places':
			return `想去 ${count} 家店可去`
		case 'view-draft-meal-plan':
			return `还有 ${count} 份菜单草稿没提交`
		case 'view-missing-location-places':
			return `还有 ${count} 个打卡点缺定位`
		case 'view-visited-places':
			return `高分复访 ${count} 家可回访`
		default:
			return ''
	}
}

function buildActions({ wishlistRecipeTotal, wantPlaceTotal, draftMealPlanDays, missingLocationPlaceTotal }) {
	const actions = []

	if (wishlistRecipeTotal > 0) {
		actions.push({
			key: 'wishlist-recipes',
			label: buildActionLabel('view-wishlist-recipes', wishlistRecipeTotal),
			count: wishlistRecipeTotal,
			actionType: 'view-wishlist-recipes'
		})
	}

	if (wantPlaceTotal > 0) {
		actions.push({
			key: 'want-places',
			label: buildActionLabel('view-want-places', wantPlaceTotal),
			count: wantPlaceTotal,
			actionType: 'view-want-places'
		})
	}

	if (draftMealPlanDays > 0) {
		actions.push({
			key: 'draft-meal-plans',
			label: buildActionLabel('view-draft-meal-plan', draftMealPlanDays),
			count: draftMealPlanDays,
			actionType: 'view-draft-meal-plan'
		})
	}

	if (missingLocationPlaceTotal > 0) {
		actions.push({
			key: 'missing-location-places',
			label: buildActionLabel('view-missing-location-places', missingLocationPlaceTotal),
			count: missingLocationPlaceTotal,
			actionType: 'view-missing-location-places'
		})
	}

	return actions
}

export function formatRelativeUpdatedAt(updatedAt, now = new Date()) {
	const time = toTimestamp(updatedAt)
	if (!time) return ''
	const nowTime = now instanceof Date ? now.getTime() : toTimestamp(now)
	const diffMs = Math.max(0, nowTime - time)
	const diffMinutes = Math.floor(diffMs / 60000)
	if (diffMinutes < 1) return '刚刚同步'
	if (diffMinutes < 60) return `${diffMinutes} 分钟前`
	const diffHours = Math.floor(diffMinutes / 60)
	if (diffHours < 24) return `${diffHours} 小时前`
	const diffDays = Math.floor(diffHours / 24)
	return `${diffDays} 天前`
}

export function buildSpaceStats(options = {}) {
	const recipes = Array.isArray(options.recipes) ? options.recipes : []
	const places = Array.isArray(options.places) ? options.places : []
	const members = Array.isArray(options.members) ? options.members : []
	const mealOrderStore = options.mealOrderStore && typeof options.mealOrderStore === 'object' ? options.mealOrderStore : {}
	const drafts = mealOrderStore.drafts && typeof mealOrderStore.drafts === 'object' ? mealOrderStore.drafts : {}
	const submitted = Array.isArray(mealOrderStore.submitted) ? mealOrderStore.submitted : []
	const now = options.now instanceof Date && !Number.isNaN(options.now.getTime()) ? options.now : new Date()
	const nowTime = now.getTime()
	const source = options.source === 'remote' ? 'remote' : 'cache'
	const isSyncing = !!options.isSyncing

	const wishlistRecipeTotal = recipes.filter((recipe) => recipe?.status === 'wishlist').length
	const wantPlaceTotal = places.filter((place) => place?.status === 'want').length
	const draftMealPlanDays = Object.values(drafts).filter((draft) => Array.isArray(draft?.items) && draft.items.length > 0).length
	const missingLocationPlaceTotal = places.filter((place) => place?.id && !hasLocation(place)).length
	const ratedPlaces = places.filter((place) => Number(place?.revisitRating) > 0)

	const imageCoveredTotal = recipes.filter((recipe) => Array.isArray(recipe?.imageUrls) && recipe.imageUrls.length).length
	const locatedTotal = places.filter((place) => hasLocation(place)).length
	const recentActivity = {
		newRecipeTotal: recipes.filter((recipe) => isWithinRecentWindow(recipe?.createdAt, nowTime)).length,
		newPlaceTotal: places.filter((place) => isWithinRecentWindow(place?.createdAt, nowTime)).length,
		visitedPlaceTotal: places.filter((place) => place?.status === 'visited' && isWithinRecentWindow(place?.visitedAt, nowTime)).length,
		submittedMealPlanTotal: submitted.filter((record) => isWithinRecentWindow(record?.submittedAt, nowTime)).length
	}

	return {
		updatedAt: now.toISOString(),
		source,
		isSyncing,
		window: options.window || '30d',
		overview: {
			recipeTotal: recipes.length,
			placeTotal: places.length,
			submittedMealPlanDays: submitted.length,
			memberTotal: members.length,
			wishlistRecipeTotal,
			wantPlaceTotal,
			topRevisitPlaces: buildTopRevisitPlaces(places)
		},
		recipes: {
			byMealType: countByField(recipes, 'mealType', ['breakfast', 'main']),
			byStatus: countByField(recipes, 'status', ['wishlist', 'done']),
			imageCoveredTotal,
			imageCoverage: ratio(imageCoveredTotal, recipes.length),
			parsedTotal: recipes.filter((recipe) => recipe?.parseStatus === 'done').length,
			recentCreatedTotal: recipes.filter((recipe) => isWithinRecentWindow(recipe?.createdAt, nowTime)).length
		},
		places: {
			byStatus: countByField(places, 'status', ['want', 'visited']),
			locatedTotal,
			locationCoverage: ratio(locatedTotal, places.length),
			experienceCompletedTotal: places.filter((place) => hasExperience(place)).length,
			highlyRecommendedTotal: places.filter((place) => Number(place?.revisitRating) >= 4).length,
			lowRatingTotal: places.filter((place) => Number(place?.revisitRating) > 0 && Number(place.revisitRating) <= 2).length,
			averageRevisitRating: average(ratedPlaces.map((place) => Number(place.revisitRating))),
			poiMatchedTotal: places.filter((place) => hasPoiMatch(place)).length,
			revisitRatingDistribution: buildRevisitRatingDistribution(places),
			topRecommendedItems: buildFrequencyTop(places.map((place) => place?.recommendedItems)),
			topScenes: buildFrequencyTop(places.map((place) => place?.scenes))
		},
		mealPlans: {
			draftDays: draftMealPlanDays,
			submittedDays: submitted.length,
			nextPlan: buildNextMealPlan(submitted, nowTime),
			averageDishCount: average(submitted.map((record) => (Array.isArray(record?.items) ? record.items.length : 0))),
			itemsByMealType: buildItemsByMealType(submitted)
		},
		members: {
			total: members.length,
			// V1/V1.1 本地聚合拿不到成员维度贡献，留空由 V2 后端 stats 填充。
			contributors: []
		},
		trends: createEmptyTrends(),
		recentActivity,
		actions: buildActions({ wishlistRecipeTotal, wantPlaceTotal, draftMealPlanDays, missingLocationPlaceTotal })
	}
}

// 后端 action.target -> 前端 actionType 映射（按稳定的 target，忽略后端过时的 label）。
const REMOTE_ACTION_TARGET_MAP = {
	'recipes:wishlist': 'view-wishlist-recipes',
	'recipes:done': 'view-done-recipes',
	'places:want': 'view-want-places',
	'places:visited': 'view-visited-places',
	'places:revisit': 'view-visited-places',
	'places:missing-location': 'view-missing-location-places',
	'mealPlans:draft': 'view-draft-meal-plan'
}

function mapRemoteActions(remoteActions) {
	return (Array.isArray(remoteActions) ? remoteActions : [])
		.map((action, index) => {
			const target = String(action?.target || '').trim()
			const actionType = REMOTE_ACTION_TARGET_MAP[target] || ''
			if (!actionType) return null
			const count = Number(action?.count) || 0
			return {
				key: `${actionType}-${index}`,
				actionType,
				count,
				// 优先用前端统一文案；无匹配文案时回退后端 label。
				label: buildActionLabel(actionType, count) || String(action?.label || '').trim()
			}
		})
		.filter(Boolean)
}

function normalizeCountedLabels(list) {
	return (Array.isArray(list) ? list : [])
		.map((entry) => ({
			label: String(entry?.label || '').trim(),
			count: Number(entry?.count) || 0
		}))
		.filter((entry) => entry.label)
}

function normalizeRemoteNextPlan(brief) {
	if (!brief || !brief.planDate) return null
	return {
		planDate: brief.planDate,
		dishCount: Number(brief.itemCount) || 0,
		titles: [] // 后端 stats 不下发菜品标题，保持结构一致即可。
	}
}

function normalizeTrends(trends) {
	const empty = createEmptyTrends()
	if (!trends || typeof trends !== 'object') return empty
	const pickSeries = (series) =>
		(Array.isArray(series) ? series : [])
			.map((point) => ({ date: String(point?.date || ''), count: Number(point?.count) || 0 }))
			.filter((point) => point.date)
	return {
		recipeCreated: pickSeries(trends.recipeCreated),
		recipeDone: pickSeries(trends.recipeDone),
		placeCreated: pickSeries(trends.placeCreated),
		placeVisited: pickSeries(trends.placeVisited),
		mealPlanSubmitted: pickSeries(trends.mealPlanSubmitted)
	}
}

// 把后端 GET /kitchens/{id}/stats 响应（Stats）归一为与 buildSpaceStats 一致的视图模型，
// 让 space-stats-card / space-stats-sheet 无需感知数据来源。
export function mapRemoteStatsToViewModel(rawRemote = {}, options = {}) {
	const remote = rawRemote && typeof rawRemote === 'object' ? rawRemote : {}
	const overview = remote.overview && typeof remote.overview === 'object' ? remote.overview : {}
	const recipes = remote.recipes && typeof remote.recipes === 'object' ? remote.recipes : {}
	const places = remote.places && typeof remote.places === 'object' ? remote.places : {}
	const mealPlans = remote.mealPlans && typeof remote.mealPlans === 'object' ? remote.mealPlans : {}
	const membersStats = remote.members && typeof remote.members === 'object' ? remote.members : {}

	return {
		updatedAt: remote.updatedAt || new Date().toISOString(),
		source: 'remote',
		isSyncing: !!options.isSyncing,
		window: remote.window || options.window || '30d',
		overview: {
			recipeTotal: Number(overview.recipeTotal) || 0,
			placeTotal: Number(overview.placeTotal) || 0,
			submittedMealPlanDays: Number(overview.submittedMealPlanDays) || 0,
			memberTotal: Number(overview.memberTotal) || 0,
			// 兼容部署过渡期：新契约 wishlistRecipeTotal / wantPlaceTotal，旧二进制仍是 weeklyAvailableRecipes / weekendAvailablePlaces。
			wishlistRecipeTotal: Number(overview.wishlistRecipeTotal ?? overview.weeklyAvailableRecipes) || 0,
			wantPlaceTotal: Number(overview.wantPlaceTotal ?? overview.weekendAvailablePlaces) || 0,
			topRevisitPlaces: (Array.isArray(overview.topRevisitPlaces) ? overview.topRevisitPlaces : []).map((place) => ({
				id: place?.id || '',
				name: place?.name || '未命名店铺',
				revisitRating: Number(place?.revisitRating) || 0,
				recommendedItems: Array.isArray(place?.recommendedItems) ? place.recommendedItems.slice(0, 2) : [],
				imageUrl: place?.imageUrl || ''
			}))
		},
		recipes: {
			byMealType: recipes.byMealType && typeof recipes.byMealType === 'object' ? recipes.byMealType : { breakfast: 0, main: 0 },
			byStatus: recipes.byStatus && typeof recipes.byStatus === 'object' ? recipes.byStatus : { wishlist: 0, done: 0 },
			imageCoveredTotal: Number(recipes.imageCoveredTotal) || 0,
			imageCoverage: Number(recipes.imageCoverage) || 0,
			parsedTotal: Number(recipes.parsedTotal) || 0,
			recentCreatedTotal: Number(recipes.recentCreatedTotal) || 0
		},
		places: {
			byStatus: places.byStatus && typeof places.byStatus === 'object' ? places.byStatus : { want: 0, visited: 0 },
			locatedTotal: Number(places.locatedTotal) || 0,
			locationCoverage: Number(places.locationCoverage) || 0,
			experienceCompletedTotal: Number(places.experienceCompletedTotal) || 0,
			highlyRecommendedTotal: Number(places.highlyRecommendedTotal) || 0,
			lowRatingTotal: Number(places.lowRatingTotal) || 0,
			averageRevisitRating: Number(places.averageRevisitRating) || 0,
			poiMatchedTotal: Number(places.poiMatchedTotal) || 0,
			revisitRatingDistribution:
				places.revisitRatingDistribution && typeof places.revisitRatingDistribution === 'object'
					? places.revisitRatingDistribution
					: { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0 },
			topRecommendedItems: normalizeCountedLabels(places.topRecommendedItems),
			topScenes: normalizeCountedLabels(places.topScenes)
		},
		mealPlans: {
			draftDays: Number(mealPlans.draftDays) || 0,
			submittedDays: Number(mealPlans.submittedDays) || 0,
			nextPlan: normalizeRemoteNextPlan(mealPlans.nextPlan),
			averageDishCount: Number(mealPlans.averageDishCount) || 0,
			itemsByMealType:
				mealPlans.itemsByMealType && typeof mealPlans.itemsByMealType === 'object'
					? mealPlans.itemsByMealType
					: { breakfast: 0, main: 0 }
		},
		members: {
			total: Number(membersStats.total) || Number(overview.memberTotal) || 0,
			contributors: (Array.isArray(membersStats.contributors) ? membersStats.contributors : []).map((member) => ({
				userId: member?.userId || 0,
				nickname: member?.nickname || '',
				avatarUrl: member?.avatarUrl || '',
				role: member?.role || 'member',
				recipeCreatedTotal: Number(member?.recipeCreatedTotal) || 0,
				placeCreatedTotal: Number(member?.placeCreatedTotal) || 0,
				mealPlanSubmittedTotal: Number(member?.mealPlanSubmittedTotal) || 0,
				total: Number(member?.total) || 0
			}))
		},
		trends: normalizeTrends(remote.trends),
		recentActivity: {
			newRecipeTotal: Number(overview.recentCreatedRecipes) || 0,
			newPlaceTotal: Number(overview.recentCreatedPlaces) || 0,
			visitedPlaceTotal: Number(overview.recentVisitedPlaces) || 0,
			submittedMealPlanTotal: Number(overview.recentSubmittedMealPlans) || 0
		},
		actions: mapRemoteActions(remote.actions)
	}
}
