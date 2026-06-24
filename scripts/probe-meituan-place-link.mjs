#!/usr/bin/env node

const DEFAULT_INPUT =
	'【旺记碳烤肥牛·烤肉大排档（北滘悦然里店）】快来试试这家餐厅吧！ 【地址：顺德区人昌路2号（华美达和悦然里中间停车场）】【电话：17303028852】@美团 http://dpurl.cn/4zWiEohz'

const MOBILE_UA =
	'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 MicroMessenger/8.0.49'

const DESKTOP_UA =
	'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36'

const MAX_BODY_BYTES = 800_000

function parseArgs(argv) {
	const args = {
		inputParts: [],
		amapKey: process.env.AMAP_WEB_SERVICE_KEY || process.env.AMAP_KEY || '',
		amapCity: process.env.AMAP_CITY || '佛山',
		amapLimit: Number(process.env.AMAP_LIMIT) || 10,
		amapMaxAttempts: Number(process.env.AMAP_MAX_ATTEMPTS) || 4,
		amapDelayMs: Number(process.env.AMAP_DELAY_MS) || 350
	}

	for (let index = 0; index < argv.length; index += 1) {
		const item = argv[index]
		if (item === '--amap-key') {
			args.amapKey = argv[index + 1] || ''
			index += 1
			continue
		}
		if (item.startsWith('--amap-key=')) {
			args.amapKey = item.slice('--amap-key='.length)
			continue
		}
		if (item === '--amap-city') {
			args.amapCity = argv[index + 1] || ''
			index += 1
			continue
		}
		if (item.startsWith('--amap-city=')) {
			args.amapCity = item.slice('--amap-city='.length)
			continue
		}
		if (item === '--amap-limit') {
			args.amapLimit = Number(argv[index + 1]) || args.amapLimit
			index += 1
			continue
		}
		if (item.startsWith('--amap-limit=')) {
			args.amapLimit = Number(item.slice('--amap-limit='.length)) || args.amapLimit
			continue
		}
		if (item === '--amap-max-attempts') {
			args.amapMaxAttempts = Number(argv[index + 1]) || args.amapMaxAttempts
			index += 1
			continue
		}
		if (item.startsWith('--amap-max-attempts=')) {
			args.amapMaxAttempts = Number(item.slice('--amap-max-attempts='.length)) || args.amapMaxAttempts
			continue
		}
		if (item === '--amap-delay-ms') {
			args.amapDelayMs = Number(argv[index + 1]) || args.amapDelayMs
			index += 1
			continue
		}
		if (item.startsWith('--amap-delay-ms=')) {
			args.amapDelayMs = Number(item.slice('--amap-delay-ms='.length)) || args.amapDelayMs
			continue
		}
		args.inputParts.push(item)
	}

	return args
}

function readStdin() {
	return new Promise((resolve) => {
		if (process.stdin.isTTY) {
			resolve('')
			return
		}

		let input = ''
		process.stdin.setEncoding('utf8')
		process.stdin.on('data', (chunk) => {
			input += chunk
		})
		process.stdin.on('end', () => resolve(input))
	})
}

function firstMatch(value, pattern) {
	const match = String(value || '').match(pattern)
	return match ? String(match[1] || '').trim() : ''
}

function extractURLs(value) {
	return Array.from(String(value || '').matchAll(/https?:\/\/[^\s，。；;）)]*/gi))
		.map((match) => String(match[0] || '').trim())
		.filter(Boolean)
}

function normalizeText(value) {
	return String(value || '')
		.replace(/\s+/g, ' ')
		.trim()
}

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, Math.max(0, Number(ms) || 0)))
}

function uniq(values) {
	const seen = new Set()
	const output = []
	for (const value of values) {
		const normalized = String(value || '').trim()
		if (!normalized || seen.has(normalized)) continue
		seen.add(normalized)
		output.push(normalized)
	}
	return output
}

function parseCopiedShareText(input) {
	const urls = extractURLs(input)
	const name = firstMatch(input, /【([^】]+)】/)
	const address = firstMatch(input, /【地址[:：]([^】]+)】/)
	const phone = firstMatch(input, /【电话[:：]([^】]+)】/)
	const source = /美团|meituan|dpurl\.cn/i.test(input) ? 'meituan' : ''

	return {
		name,
		address,
		phone,
		source,
		sourceUrl: urls[0] || '',
		urls
	}
}

function parseQueryFromURL(rawURL) {
	try {
		const url = new URL(rawURL)
		return Object.fromEntries(url.searchParams.entries())
	} catch {
		return {}
	}
}

function extractNestedTargetURL(rawURL) {
	const params = parseQueryFromURL(rawURL)
	if (!params.url) return ''
	try {
		return decodeURIComponent(params.url)
	} catch {
		return params.url
	}
}

function extractPoiFromText(text) {
	const source = String(text || '')
	return {
		poiId: firstMatch(source, /(?:poiId|poiid|mtShopId)=([0-9]+)/i) || firstMatch(source, /\/(?:poi|shop|meishi)\/([0-9]+)/i),
		poiIdEncrypt: firstMatch(source, /poiIdEncrypt=([^&#"'\s]+)/i)
	}
}

function decodeHTML(value) {
	return String(value || '')
		.replace(/&amp;/g, '&')
		.replace(/&quot;/g, '"')
		.replace(/&#39;/g, "'")
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
}

function extractMeta(html) {
	const title = decodeHTML(firstMatch(html, /<title[^>]*>([\s\S]*?)<\/title>/i))
	const metas = {}
	const metaPattern = /<meta\b[^>]*>/gi
	const attrPattern = /\b(name|property|content)=["']([^"']*)["']/gi

	for (const tag of String(html || '').match(metaPattern) || []) {
		const attrs = {}
		for (const match of tag.matchAll(attrPattern)) {
			attrs[match[1].toLowerCase()] = decodeHTML(match[2])
		}
		const key = attrs.property || attrs.name
		if (key && attrs.content) metas[key] = attrs.content
	}

	return {
		title: normalizeText(title),
		description: normalizeText(metas.description || metas['og:description'] || ''),
		ogTitle: normalizeText(metas['og:title'] || ''),
		ogImage: normalizeText(metas['og:image'] || metas.image || '')
	}
}

function extractImageURLs(text) {
	const imagePattern = /https?:\/\/[^"'()\s<>]+?\.(?:jpg|jpeg|png|webp)(?:[@?][^"'()\s<>]*)?/gi
	return uniq(Array.from(String(text || '').matchAll(imagePattern)).map((match) => match[0]))
}

function isLikelyNonShopImage(url) {
	const value = String(url || '')
	if (!value) return true
	if (value.includes('${')) return true
	return /\/smartvenus\/|\/travelcube\/|\/keeper\/evoke-image-|favicon|QRCode|qr/i.test(value)
}

function extractJSONLikeFields(text) {
	const fields = {}
	const pairs = [
		['frontImg', /"frontImg"\s*:\s*"([^"]+)"/i],
		['avgScore', /"avgScore"\s*:\s*"?([^",}\]]+)/i],
		['avgPrice', /"avgPrice"\s*:\s*"?([^",}\]]+)/i],
		['address', /"address"\s*:\s*"([^"]+)"/i],
		['phone', /"phone"\s*:\s*"([^"]+)"/i],
		['openTime', /"openTime"\s*:\s*"([^"]+)"/i]
	]

	for (const [key, pattern] of pairs) {
		const value = firstMatch(text, pattern)
		if (value) fields[key] = decodeHTML(value)
	}

	return fields
}

async function fetchWithLimit(url, options = {}) {
	const controller = new AbortController()
	const timeout = setTimeout(() => controller.abort(), options.timeoutMs || 12_000)

	try {
		const response = await fetch(url, {
			method: options.method || 'GET',
			redirect: options.redirect || 'follow',
			signal: controller.signal,
			headers: {
				'user-agent': options.userAgent || MOBILE_UA,
				accept: options.accept || 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
				...options.headers
			}
		})

		const arrayBuffer = await response.arrayBuffer()
		const limited = arrayBuffer.slice(0, MAX_BODY_BYTES)
		const body = new TextDecoder('utf8', { fatal: false }).decode(limited)

		return {
			ok: response.ok,
			status: response.status,
			url: response.url,
			headers: Object.fromEntries(response.headers.entries()),
			body,
			truncated: arrayBuffer.byteLength > MAX_BODY_BYTES
		}
	} catch (error) {
		return {
			ok: false,
			status: 0,
			url,
			error: error instanceof Error ? error.message : String(error)
		}
	} finally {
		clearTimeout(timeout)
	}
}

async function expandURL(rawURL) {
	if (!rawURL) return null
	const response = await fetchWithLimit(rawURL, {
		redirect: 'manual',
		userAgent: MOBILE_UA,
		timeoutMs: 12_000
	})

	const location = response.headers?.location || ''
	let absoluteLocation = location
	try {
		absoluteLocation = location ? new URL(location, rawURL).toString() : ''
	} catch {
		absoluteLocation = location
	}

	const nestedTargetURL = extractNestedTargetURL(absoluteLocation)
	const poiFromLocation = extractPoiFromText(`${absoluteLocation}\n${nestedTargetURL}`)

	return {
		status: response.status,
		location: absoluteLocation,
		nestedTargetURL,
		...poiFromLocation
	}
}

function buildCandidateURLs(expanded) {
	const urls = []
	const poiId = expanded?.poiId
	const poiIdEncrypt = expanded?.poiIdEncrypt

	if (expanded?.location) urls.push(expanded.location)
	if (poiId) {
		urls.push(`https://i.meituan.com/shop/${poiId}.html`)
		urls.push(`https://i.meituan.com/poi/${poiId}${poiIdEncrypt ? `?poiIdEncrypt=${poiIdEncrypt}` : ''}`)
		urls.push(`https://www.meituan.com/meishi/${poiId}/`)
		urls.push(`https://www.meituan.com/meishi/api/poi/getPoiInfo?poiId=${poiId}`)
	}

	return uniq(urls)
}

function summarizeProbe(url, result) {
	const body = result.body || ''
	const meta = extractMeta(body)
	const jsonFields = extractJSONLikeFields(body)
	const imageURLs = extractImageURLs(body).slice(0, 12)
	const poi = extractPoiFromText(`${result.url || ''}\n${body}`)
	const blocked = /verify\.meituan\.com|spiderindefence|验证码|安全验证|Status page/i.test(`${result.url || ''}\n${body}`)

	return {
		url,
		finalUrl: result.url,
		status: result.status,
		ok: result.ok,
		blocked,
		title: meta.title,
		ogTitle: meta.ogTitle,
		description: meta.description,
		ogImage: meta.ogImage,
		poiId: poi.poiId,
		poiIdEncrypt: poi.poiIdEncrypt,
		jsonFields,
		imageURLs,
		error: result.error || '',
		truncated: !!result.truncated
	}
}

async function probePages(candidateURLs) {
	const results = []
	for (const url of candidateURLs) {
		const userAgent = /www\.meituan\.com/.test(url) ? DESKTOP_UA : MOBILE_UA
		const response = await fetchWithLimit(url, {
			userAgent,
			timeoutMs: 12_000
		})
		results.push(summarizeProbe(url, response))
	}
	return results
}

function normalizeAmapKeyword(value) {
	return normalizeText(
		String(value || '')
			.replace(/[【】()[\]（）·.,，。/\\|:：;；"'“”‘’!！?？_-]+/g, ' ')
	)
}

function stripBranchFromName(value) {
	return String(value || '').replace(/[（(][^）)]*[）)]/g, ' ')
}

function buildAmapKeywords(copied) {
	const cleanName = normalizeAmapKeyword(copied.name)
	const nameWithoutBranch = normalizeAmapKeyword(stripBranchFromName(copied.name))
	const cleanAddress = normalizeAmapKeyword(copied.address)

	return uniq([
		copied.name,
		cleanName,
		nameWithoutBranch,
		[copied.name, copied.address].filter(Boolean).join(' '),
		copied.phone,
		cleanAddress
	]).filter(Boolean)
}

function normalizeComparable(value) {
	return String(value || '')
		.toLowerCase()
		.replace(/[^a-z0-9\u4e00-\u9fff]/g, '')
}

function uniqueCharacters(value) {
	return Array.from(new Set(Array.from(normalizeComparable(value))))
}

function diceCoefficient(left, right) {
	const leftChars = uniqueCharacters(left)
	const rightChars = uniqueCharacters(right)
	if (!leftChars.length || !rightChars.length) return 0

	const rightSet = new Set(rightChars)
	let overlap = 0
	for (const char of leftChars) {
		if (rightSet.has(char)) overlap += 1
	}
	return (2 * overlap) / (leftChars.length + rightChars.length)
}

function buildNameTerms(name) {
	const cleanName = normalizeAmapKeyword(stripBranchFromName(name))
	const parts = cleanName.split(/\s+/).filter((part) => normalizeComparable(part).length >= 2)
	const compact = normalizeComparable(cleanName)
	return uniq([
		compact,
		...parts,
		parts.slice(0, 2).join('')
	]).filter((term) => normalizeComparable(term).length >= 2)
}

function extractBracketTexts(value) {
	const output = []
	const pattern = /[（(]([^）)]{2,})[）)]/g
	for (const match of String(value || '').matchAll(pattern)) {
		output.push(match[1])
	}
	return output
}

function extractDoorplates(value) {
	return uniq(Array.from(String(value || '').matchAll(/[\u4e00-\u9fffA-Za-z0-9]{1,12}(?:路|街|道|大道|巷)\d+(?:号)?/g)).map((match) => match[0]))
}

function splitUsefulLocationTerms(value) {
	const terms = []
	const withoutStoreSuffix = String(value || '').replace(/店$/g, '')
	const compact = normalizeComparable(withoutStoreSuffix)
	const branchMatch = compact.match(/^([\u4e00-\u9fff]{2,4}?)([\u4e00-\u9fff]{2,8}(?:里|城|广场|中心|园区|商场))$/)
	if (branchMatch) {
		terms.push(branchMatch[1], branchMatch[2])
	}

	const cleaned = normalizeAmapKeyword(withoutStoreSuffix)
		.replace(/停车场|中间|附近|旁边|入口|出口|首层|一层|二层|店/g, ' ')
		.replace(/[和与及]/g, ' ')
	for (const part of cleaned.split(/\s+/)) {
		const normalized = normalizeComparable(part)
		if (normalized.length >= 2 && normalized.length <= 12) terms.push(part)
	}

	return uniq(terms)
}

function buildLocationHints(copied) {
	const address = String(copied.address || '')
	const bracketTexts = extractBracketTexts(copied.name)
	const parentheticalAddressTexts = extractBracketTexts(address)
	const doorplates = extractDoorplates(`${copied.name} ${address}`)
	const branchTerms = bracketTexts.flatMap(splitUsefulLocationTerms)
	const addressTerms = [
		...parentheticalAddressTexts.flatMap(splitUsefulLocationTerms),
		...doorplates
	]

	return {
		doorplates,
		branchTerms: uniq(branchTerms),
		addressTerms: uniq(addressTerms),
		allTerms: uniq([...branchTerms, ...addressTerms])
	}
}

function hasSamePhone(copiedPhone, poiPhone) {
	const expected = String(copiedPhone || '').replace(/\D/g, '')
	if (!expected) return false
	return String(poiPhone || '')
		.split(/[;,，、/]/)
		.some((item) => item.replace(/\D/g, '') === expected)
}

function includesComparable(haystack, needle) {
	const normalizedNeedle = normalizeComparable(needle)
	return normalizedNeedle ? normalizeComparable(haystack).includes(normalizedNeedle) : false
}

function scoreAmapPOI(copied, poi) {
	const reasons = []
	let score = 0

	const nameScore = Math.round(diceCoefficient(copied.name, poi.name) * 80)
	const addressScore = Math.round(diceCoefficient(copied.address, poi.address) * 90)
	score += nameScore + addressScore
	if (nameScore) reasons.push(`名称相似度 +${nameScore}`)
	if (addressScore) reasons.push(`地址相似度 +${addressScore}`)

	if (/餐饮服务/.test(poi.type)) {
		score += 45
		reasons.push('餐饮类目 +45')
	}
	if (/购物服务|交通设施服务|停车场|汽车服务/.test(poi.type)) {
		score -= 30
		reasons.push('非餐饮/停车等类目 -30')
	}

	if (hasSamePhone(copied.phone, poi.tel)) {
		score += 80
		reasons.push('电话一致 +80')
	}

	const poiName = normalizeComparable(poi.name)
	for (const term of buildNameTerms(copied.name)) {
		const normalizedTerm = normalizeComparable(term)
		if (!normalizedTerm || !poiName.includes(normalizedTerm)) continue
		const bonus = Math.min(25, normalizedTerm.length * 4)
		score += bonus
		reasons.push(`命中名称词「${term}」 +${bonus}`)
	}

	const locationHints = buildLocationHints(copied)
	const poiLocationText = `${poi.name} ${poi.address} ${poi.adname} ${poi.cityname}`
	let locationHitCount = 0
	for (const doorplate of locationHints.doorplates) {
		if (includesComparable(poiLocationText, doorplate)) {
			score += 70
			locationHitCount += 1
			reasons.push(`门牌匹配「${doorplate}」 +70`)
			continue
		}

		const road = firstMatch(doorplate, /^(.+?(?:路|街|道|大道|巷))/)
		if (road && includesComparable(poiLocationText, road)) {
			score += 25
			locationHitCount += 1
			reasons.push(`道路匹配「${road}」 +25`)
		}
	}
	for (const term of locationHints.branchTerms) {
		if (!includesComparable(poiLocationText, term)) continue
		const bonus = Math.min(35, normalizeComparable(term).length * 10)
		score += bonus
		locationHitCount += 1
		reasons.push(`分店/商圈词匹配「${term}」 +${bonus}`)
	}
	for (const term of locationHints.addressTerms) {
		if (locationHints.doorplates.includes(term) || !includesComparable(poiLocationText, term)) continue
		const bonus = Math.min(30, normalizeComparable(term).length * 8)
		score += bonus
		locationHitCount += 1
		reasons.push(`地址补充词匹配「${term}」 +${bonus}`)
	}
	if (locationHints.allTerms.length && locationHitCount === 0) {
		score -= 35
		reasons.push('未命中地址/分店线索 -35')
	}

	if (poi.photos?.length) {
		score += 5
		reasons.push('有图片 +5')
	}
	if (poi.rating) {
		score += 3
		reasons.push('有评分 +3')
	}

	return {
		score,
		reasons
	}
}

function mergeAmapPOI(existing, poi, keyword) {
	const merged = {
		...(existing || {}),
		...poi,
		photos: existing?.photos?.length > poi.photos.length ? existing.photos : poi.photos,
		sourceKeywords: uniq([...(existing?.sourceKeywords || []), keyword])
	}
	for (const key of ['rating', 'cost', 'tel', 'address', 'location']) {
		if (!merged[key] && existing?.[key]) merged[key] = existing[key]
	}
	return merged
}

function rankAmapPOIs(copied, attempts) {
	const byId = new Map()
	for (const attempt of attempts) {
		for (const poi of attempt.pois || []) {
			const key = poi.id || `${poi.name}|${poi.address}|${poi.location}`
			byId.set(key, mergeAmapPOI(byId.get(key), poi, attempt.keyword))
		}
	}

	return Array.from(byId.values())
		.map((poi) => {
			const match = scoreAmapPOI(copied, poi)
			return {
				...poi,
				matchScore: match.score,
				matchReasons: match.reasons
			}
		})
		.sort((left, right) => right.matchScore - left.matchScore)
}

function normalizeAmapPOI(poi = {}) {
	const photos = Array.isArray(poi.photos) ? poi.photos : []
	const bizExt = poi.biz_ext && typeof poi.biz_ext === 'object' ? poi.biz_ext : {}
	const [longitude = '', latitude = ''] = String(poi.location || '').split(',')

	return {
		id: String(poi.id || ''),
		name: String(poi.name || ''),
		type: String(poi.type || ''),
		typecode: String(poi.typecode || ''),
		address: Array.isArray(poi.address) ? poi.address.join(' ') : String(poi.address || ''),
		location: String(poi.location || ''),
		latitude,
		longitude,
		tel: Array.isArray(poi.tel) ? poi.tel.join(';') : String(poi.tel || ''),
		distance: String(poi.distance || ''),
		pname: String(poi.pname || ''),
		cityname: String(poi.cityname || ''),
		adname: String(poi.adname || ''),
		rating: String(bizExt.rating || ''),
		cost: String(bizExt.cost || ''),
		photos: photos.map((photo) => ({
			title: String(photo.title || ''),
			url: String(photo.url || '')
		})).filter((photo) => photo.url)
	}
}

async function queryAmapPOI(copied, options) {
	const key = String(options?.key || '').trim()
	if (!key) {
		return {
			enabled: false,
			reason: 'missing AMAP WebService key; pass --amap-key or set AMAP_WEB_SERVICE_KEY'
		}
	}

	const keywords = buildAmapKeywords(copied)
	const attempts = []
	const maxAttempts = Math.max(1, Math.min(Number(options.maxAttempts) || 4, keywords.length))
	const delayMs = Math.max(0, Number(options.delayMs) || 0)

	for (let index = 0; index < maxAttempts; index += 1) {
		if (index > 0 && delayMs) await sleep(delayMs)
		const keyword = keywords[index]
		const params = new URLSearchParams({
			key,
			keywords: keyword,
			city: String(options.city || ''),
			citylimit: 'true',
			offset: String(Math.max(1, Math.min(Number(options.limit) || 5, 25))),
			page: '1',
			extensions: 'all',
			output: 'json'
		})
		const url = `https://restapi.amap.com/v3/place/text?${params.toString()}`
		const response = await fetchWithLimit(url, {
			accept: 'application/json',
			userAgent: DESKTOP_UA,
			timeoutMs: 12_000
		})

		let payload = null
		try {
			payload = JSON.parse(response.body || '{}')
		} catch {
			payload = null
		}

		const pois = Array.isArray(payload?.pois) ? payload.pois.map(normalizeAmapPOI) : []
		const attempt = {
			keyword,
			status: response.status,
			ok: response.ok,
			amapStatus: String(payload?.status || ''),
			info: String(payload?.info || ''),
			count: String(payload?.count || ''),
			pois
		}
		attempts.push(attempt)

		if (/QPS|LIMIT|EXCEEDED/i.test(attempt.info) && delayMs < 1000) {
			await sleep(1000)
		}
	}
	const rankedPois = rankAmapPOIs(copied, attempts)

	return {
		enabled: true,
		city: String(options.city || ''),
		keywords,
		attempts,
		rankedPois,
		best: rankedPois[0] || null
	}
}

async function main() {
	const args = parseArgs(process.argv.slice(2))
	const stdin = await readStdin()
	const input = normalizeText(args.inputParts.join(' ') || stdin || DEFAULT_INPUT)
	const copied = parseCopiedShareText(input)
	const expanded = await expandURL(copied.sourceUrl)
	const candidateURLs = buildCandidateURLs(expanded)
	const probes = await probePages(candidateURLs)
	const amap = await queryAmapPOI(copied, {
		key: args.amapKey,
		city: args.amapCity,
		limit: args.amapLimit,
		maxAttempts: args.amapMaxAttempts,
		delayMs: args.amapDelayMs
	})
	const imageURLs = uniq([
		...probes.map((item) => item.ogImage),
		...probes.flatMap((item) => item.imageURLs),
		...probes.map((item) => item.jsonFields.frontImg),
		...(amap.best?.photos || []).map((item) => item.url)
	]).filter(Boolean)
	const amapImageURLs = (amap.best?.photos || []).map((item) => item.url).filter(Boolean)
	const usableImageURLs = uniq([
		...imageURLs.filter((url) => !isLikelyNonShopImage(url)),
		...amapImageURLs
	])

	const result = {
		input,
		extractedFromCopiedText: copied,
		expandedShortLink: expanded,
		candidateURLs,
		probes,
		amap,
		aggregated: {
			poiId: expanded?.poiId || probes.find((item) => item.poiId)?.poiId || '',
			poiIdEncrypt: expanded?.poiIdEncrypt || probes.find((item) => item.poiIdEncrypt)?.poiIdEncrypt || '',
			amapPoiId: amap.best?.id || '',
			amapRating: amap.best?.rating || '',
			amapCost: amap.best?.cost || '',
			amapImageURLs,
			imageURLs,
			usableImageURLs,
			jsonFields: probes.reduce((acc, item) => ({ ...acc, ...item.jsonFields }), {})
		}
	}

	process.stdout.write(`${JSON.stringify(result, null, 2)}\n`)
}

main().catch((error) => {
	console.error(error)
	process.exit(1)
})
