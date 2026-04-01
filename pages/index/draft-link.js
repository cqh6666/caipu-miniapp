const firstURLPattern = /https?:\/\/[^\s]+/i
const draftLinkTrailingPunctuationPattern = /[).,，。！!？?】）'"”’]+$/
const draftPlatformPattern = /\s*-\s*(哔哩哔哩|小红书)\s*$/i
const draftShareSuffixPattern = /复制后打开【小红书】查看笔记!?/g
const draftWhitespacePattern = /\s+/g
const draftSplitPattern = /[!！?？~～|｜/·•,:，。；;、\s]+/
const draftLowConfidencePattern = /教程|做法|分享|来咯|来啦|来了|最好吃|就是这个味|超级软烂|超软烂|入口即化|香迷糊|巨好吃|真的绝了|一学就会|零失败|保姆级|超下饭|超级入味/i
const draftNarrativePattern = /我做了|我家|我们家|拿手菜|私房菜|祖传|开店|饭店|餐馆|摆摊|多年|[0-9一二三四五六七八九十两]+年/i
const draftDishPattern = /炖|炒|烧|煮|蒸|焖|拌|炸|卤|煎|烤|焗|煲|炝|凉拌|清蒸|红烧|糖醋|牛腩|牛肉|排骨|鸡翅|鸡腿|五花肉|里脊|番茄|西红柿|土豆|茄子|豆腐|虾|鱼|面|饭|粥|汤|蛋/i
const draftDescriptorPattern = /鲜香|入味|浓稠|软烂|下饭|香辣|酸甜|麻辣|清爽|酥脆|嫩滑|家常|科学/i
const draftNoisePatterns = [
	/(.+?)(?:最好吃的做法|家常做法|详细做法|做法分享|做法教程|做法来了|做法来咯|做法来啦|教程来咯|教程来啦|教程来了|教程分享|教程|做法).*$/i,
	/(.+?)(?:就是这个味|超级软烂|超软烂|入口即化|香迷糊了?|巨好吃|好吃到哭|一学就会|零失败|保姆级|超下饭|真的绝了?|超级入味).*$/i
]

export function detectDraftLinkPlatform(input = '') {
	const value = String(input || '').toLowerCase()
	if (!value) return ''
	if (value.includes('bilibili.com') || value.includes('b23.tv') || value.includes('bili2233.cn')) return 'bilibili'
	if (value.includes('xiaohongshu.com') || value.includes('xhslink.com')) return 'xiaohongshu'
	return ''
}

export function extractSupportedDraftLink(input = '') {
	const raw = String(input || '').trim()
	if (!raw) return ''

	const match = raw.match(firstURLPattern)
	let candidate = String(match?.[0] || '').trim()
	if (!candidate && detectDraftLinkPlatform(raw)) {
		candidate = raw
	}
	if (!candidate) return ''

	candidate = candidate.replace(draftLinkTrailingPunctuationPattern, '').trim()
	return detectDraftLinkPlatform(candidate) ? candidate : ''
}

export function normalizeDraftAutoTitle(input = '') {
	let title = String(input || '').trim()
	if (!title) return ''
	const bracketMatch = title.match(/[【\[]([^】\]]+)[】\]]/)
	if (bracketMatch && bracketMatch[1]) {
		title = bracketMatch[1].trim()
	}
	title = title.replace(draftPlatformPattern, '').replace(draftShareSuffixPattern, '').trim()
	title = trimTrailingDraftTag(title)
	title = title.replace(draftWhitespacePattern, ' ').trim()
	title = trimDraftTitleNoise(title)
	title = chooseDraftTitleCandidate(title)
	title = title.replace(/[。！!~～\s]+$/g, '').trim()
	return title
}

function trimDraftTitleNoise(title = '') {
	let value = String(title || '').trim()
	if (!value) return ''

	draftNoisePatterns.some((pattern) => {
		const match = value.match(pattern)
		if (!match || !match[1]) return false
		const candidate = String(match[1] || '').trim()
		if ([...candidate].length < 2) return false
		value = candidate
		return true
	})

	return value.replace(/[。！!~～\s]+$/g, '').trim()
}

function scoreDraftTitleCandidate(title = '') {
	const value = String(title || '').trim()
	if (!value) return -100
	const length = [...value].length
	let score = 0

	if (length < 2) {
		score -= 8
	} else if (length <= 12) {
		score += 4
	} else if (length <= 16) {
		score += 2
	} else if (length <= 20) {
		score -= 1
	} else {
		score -= 5
	}

	if (draftDishPattern.test(value)) score += 5
	if (draftLowConfidencePattern.test(value)) score -= 3
	if (draftNarrativePattern.test(value)) score -= 4
	if (value.includes('的') && draftDescriptorPattern.test(value)) score -= 1

	return score
}

function chooseDraftTitleCandidate(title = '') {
	const value = String(title || '').trim()
	if (!value) return ''

	const candidates = collectDraftTitleCandidates(value)
	let best = value
	let bestScore = scoreDraftTitleCandidate(value)
	let bestLength = [...value].length

	candidates.forEach((candidate) => {
		const score = scoreDraftTitleCandidate(candidate)
		const length = [...candidate].length
		if (score > bestScore || (score === bestScore && length < bestLength)) {
			best = candidate
			bestScore = score
			bestLength = length
		}
	})

	return best
}

function collectDraftTitleCandidates(title = '') {
	const candidates = []
	const appendCandidate = (raw = '') => {
		const candidate = trimDraftTitleNoise(trimTrailingDraftTag(String(raw || '').trim()))
			.replace(/[。！!~～\s]+$/g, '')
			.trim()
			.replace(/^[【\[]+|[】\]]+$/g, '')
			.trim()
		if ([...candidate].length < 2) return
		if (candidates.includes(candidate)) return
		candidates.push(candidate)
	}

	appendCandidate(title)
	String(title || '')
		.split(draftSplitPattern)
		.forEach((segment) => appendCandidate(segment))

	;[...candidates].forEach((candidate) => {
		const index = candidate.lastIndexOf('的')
		if (index >= 0 && index < candidate.length - 1) {
			appendCandidate(candidate.slice(index + 1))
		}
	})

	return candidates
}

function trimTrailingDraftTag(title = '') {
	const value = String(title || '').trim()
	const lastBracket = Math.max(value.lastIndexOf('【'), value.lastIndexOf('['))
	if (lastBracket > 0) {
		return value.slice(0, lastBracket).trim()
	}
	return value
}

export function guessDraftTitleFromShareText(input = '') {
	const raw = String(input || '').trim()
	if (!raw) return ''

	const text = raw
		.replace(firstURLPattern, ' ')
		.replace(/复制后打开【小红书】查看笔记!?/g, ' ')
		.replace(/发布了一篇小红书笔记.*$/g, ' ')
		.replace(/快来看吧.*$/g, ' ')
		.replace(/\s+/g, ' ')
		.trim()

	if (!text) return ''

	const bracketMatch = text.match(/[【\[]([^】\]]+)[】\]]/)
	if (bracketMatch && bracketMatch[1]) {
		return normalizeDraftAutoTitle(bracketMatch[1])
	}

	const line = text.split(/[。\n]/).map((item) => item.trim()).filter(Boolean)[0] || ''
	return normalizeDraftAutoTitle(line)
}
