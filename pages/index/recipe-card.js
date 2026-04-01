import { mealTypeLabelMap } from '../../utils/recipe-store'

export function buildRecipeInfoLine(recipe = {}) {
	const mealLabel = mealTypeLabelMap[recipe.mealType] || '早餐'
	const parseStatus = String(recipe.parseStatus || '').trim()

	let parseLabel = '手动整理'
	if (parseStatus === 'done') {
		parseLabel = '已整理'
	} else if (parseStatus === 'pending' || parseStatus === 'processing') {
		parseLabel = '整理中'
	} else if (parseStatus === 'failed') {
		parseLabel = '待重试'
	} else if (String(recipe.link || '').trim()) {
		parseLabel = '可整理'
	}

	return `${mealLabel} · ${parseLabel}`
}

export function detectRecipeSource(recipe = {}) {
	const parseSource = String(recipe.parseSource || '').trim().toLowerCase()
	const link = String(recipe.link || '').trim().toLowerCase()
	if (parseSource.includes('bilibili') || link.includes('bilibili.com') || link.includes('b23.tv') || link.includes('bili2233.cn')) {
		return 'B站'
	}
	if (parseSource.includes('xiaohongshu') || link.includes('xiaohongshu.com') || link.includes('xhslink.com')) {
		if (parseSource.includes(':video')) return '小红书视频'
		if (parseSource.includes(':image')) return '小红书图文'
		return '小红书'
	}
	if (link) return '链接'
	return ''
}

export function pickRecipePlaceholderIcon(recipe = {}) {
	return recipe.mealType === 'main' ? 'grid-fill' : 'clock-fill'
}

export function extractRecipeImages(recipe = {}) {
	if (Array.isArray(recipe.imageUrls) && recipe.imageUrls.length) {
		return recipe.imageUrls.filter(Boolean)
	}
	if (Array.isArray(recipe.images) && recipe.images.length) {
		return recipe.images.filter(Boolean)
	}
	return [recipe.image, recipe.imageUrl].filter(Boolean)
}

function truncateTextByRune(value = '', maxLength = 15) {
	const items = Array.from(String(value || '').trim())
	if (items.length <= maxLength) return items.join('')
	return items.slice(0, maxLength).join('')
}

export function buildRecipeListSummary(recipe = {}) {
	return truncateTextByRune(String(recipe.summary || '').trim(), 24)
}

export function buildRecipeCoverVersion(recipe = {}) {
	return String(recipe.updatedAt || recipe.parseFinishedAt || '').trim()
}

export function buildRecipeSearchText(recipe = {}) {
	const parsedContent = recipe.parsedContent || {}
	const ingredientLines = [
		...(Array.isArray(parsedContent.ingredients) ? parsedContent.ingredients : []),
		...(Array.isArray(parsedContent.mainIngredients) ? parsedContent.mainIngredients : []),
		...(Array.isArray(parsedContent.secondaryIngredients) ? parsedContent.secondaryIngredients : [])
	]
	const stepLines = (Array.isArray(parsedContent.steps) ? parsedContent.steps : []).reduce((result, step) => {
		if (typeof step === 'string') {
			return result.concat(step)
		}
		return result.concat([step?.title, step?.detail, step?.text].filter(Boolean))
	}, [])
		.filter(Boolean)

	return [
		recipe.title,
		recipe.summary,
		recipe.ingredient,
		recipe.note,
		recipe.link,
		...ingredientLines,
		...stepLines
	]
		.filter(Boolean)
		.join('\n')
		.toLowerCase()
}

export function buildRecipeCard(recipe = {}, cachedCoverMap = {}) {
	const images = extractRecipeImages(recipe)
	const remoteCover = images[0] || ''
	const cachedCover = cachedCoverMap[recipe.id] || ''
	return {
		...recipe,
		cover: cachedCover || remoteCover,
		cachedCover,
		remoteCover,
		coverVersion: buildRecipeCoverVersion(recipe),
		isPinned: !!String(recipe.pinnedAt || '').trim(),
		imageCount: images.length,
		sourceBadge: detectRecipeSource(recipe),
		placeholderIcon: pickRecipePlaceholderIcon(recipe),
		mealTypeLabel: mealTypeLabelMap[recipe.mealType] || '正餐',
		infoLine: buildRecipeInfoLine(recipe),
		listSummary: buildRecipeListSummary(recipe)
	}
}
