export const ACTIVE_PARSE_STATUSES = ['pending', 'processing']
export const ACTIVE_FLOWCHART_STATUSES = ['pending', 'processing']
export const parseStatusMetaMap = {
	idle: { label: '可自动整理', tone: 'pending', description: '支持链接自动整理，可手动开始整理当前做法。' },
	pending: { label: '等待解析', tone: 'pending', description: '已加入后台整理队列，稍后会自动补齐食材和步骤。' },
	processing: { label: '解析中', tone: 'processing', description: '后台正在整理链接内容，结果会自动更新。' },
	done: { label: '已自动整理', tone: 'done', description: '食材和步骤已自动整理完成。' },
	failed: { label: '解析失败', tone: 'failed', description: '这次自动整理没成功，可以再试一次。' }
}
export const flowchartStatusMetaMap = {
	pending: { label: '等待出图', tone: 'pending', description: '已加入生成队列，稍后会自动补上步骤图。' },
	processing: { label: '正在出图', tone: 'processing', description: '后台正在整理步骤图，完成后会自动刷新。' },
	failed: { label: '生成失败', tone: 'failed', description: '这次步骤图生成没成功，可以重新再试。' }
}
export const isAutoParseSupportedLink = (link = '') => /(bilibili\.com|b23\.tv|bili2233\.cn|xiaohongshu\.com|xhslink\.com)/i.test(String(link).trim())
export function extractCopyableLink(value = '') {
	const source = String(value || '').trim()
	if (!source) return ''
	const link = String(source.match(/https?:\/\/[^\s]+/i)?.[0] || source).trim()
	return link.replace(/[)\]】》」'",，。；;!?！？]+$/g, '').trim()
}
export function formatParseSourceLabel(source = '') {
	const value = String(source).trim()
	if (!value) return ''
	if (value === 'bilibili') return '来源：B 站链接自动解析'
	if (value === 'bilibili:ai') return '来源：B 站内容 + AI 总结'
	if (value === 'bilibili:heuristic') return '来源：B 站规则整理'
	if (value.startsWith('xiaohongshu')) {
		const parts = value.toLowerCase().split(':').filter(Boolean)
		if (parts.includes('ai')) return '来源：小红书 + AI 总结'
		if (parts.includes('heuristic')) return '来源：小红书规则整理'
		return '来源：小红书链接自动解析'
	}
	return `来源：${value}`
}
export function buildParseResultHint(status = '', source = '') {
	return String(status).trim().toLowerCase() === 'done' && String(source).trim().toLowerCase() === 'bilibili:heuristic'
		? '这次先按规则整理，通常是因为字幕不可用，或 AI 总结暂时不可用；可以稍后再试一次。'
		: ''
}
export function toPositiveInteger(value = 0) {
	const parsed = Number(value)
	return Number.isFinite(parsed) && parsed > 0 ? Math.ceil(parsed) : 0
}
export function resolveRemainingWaitSeconds(value = 0, syncedAt = 0, now = 0) {
	const base = toPositiveInteger(value)
	if (!base) return 0
	const elapsed = Number(syncedAt) > 0 && Number(now) > Number(syncedAt)
		? Math.floor((Number(now) - Number(syncedAt)) / 1000)
		: 0
	return Math.max(base - elapsed, 0)
}
function formatApproxWait(seconds = 0) {
	const total = toPositiveInteger(seconds)
	if (!total) return ''
	if (total < 60) return `${Math.max(5, Math.ceil(total / 5) * 5)} 秒左右`
	if (total < 3600) return `${Math.max(1, Math.ceil(total / 60))} 分钟左右`
	const hours = Math.floor(total / 3600)
	const minutes = Math.ceil((total % 3600) / 60)
	return minutes ? `${hours} 小时 ${minutes} 分钟左右` : `${hours} 小时左右`
}
function buildWaitHint(kind, status = '', queueAhead = 0, waitSeconds = 0) {
	const normalizedStatus = String(status).trim().toLowerCase()
	const waitText = formatApproxWait(waitSeconds)
	if (!waitText) return ''
	const noun = kind === 'flowchart' ? '出图' : '整理'
	if (normalizedStatus === 'pending') return queueAhead > 0
		? `前面还有 ${queueAhead} 个任务，预计还要 ${waitText}，${noun}完成后会自动刷新。`
		: `已加入${noun}队列，预计 ${waitText} 后完成。`
	if (normalizedStatus === 'processing') return `后台正在${kind === 'flowchart' ? '生成步骤图' : '整理链接内容'}，预计还要 ${waitText}，完成后会自动刷新。`
	return ''
}
export const buildParseWaitHint = (status, queueAhead, waitSeconds) => buildWaitHint('parse', status, queueAhead, waitSeconds)
export const buildFlowchartWaitHint = (status, queueAhead, waitSeconds) => buildWaitHint('flowchart', status, queueAhead, waitSeconds)
export function formatDateTime(value = '') {
	const date = new Date(value)
	if (Number.isNaN(date.getTime())) return ''
	const pad = (value) => `${value}`.padStart(2, '0')
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}
export function createRecipeJobPollingController(poll, interval = 2200) {
	let timer = null
	return {
		start() { if (!timer) timer = setInterval(poll, interval) },
		stop() { if (timer) clearInterval(timer); timer = null },
		isActive() { return !!timer }
	}
}

export function createRecipeAsyncJobsController(options = {}) {
	const {
		poll = () => {},
		onEstimateTick = () => {},
		pollInterval = 4000,
		estimateInterval = 1000
	} = options
	const pollController = createRecipeJobPollingController(poll, pollInterval)
	const estimateController = createRecipeJobPollingController(onEstimateTick, estimateInterval)

	return {
		sync({ hasActiveJob = false, hasEstimate = false } = {}) {
			if (hasActiveJob) pollController.start()
			else pollController.stop()
			if (hasActiveJob && hasEstimate) estimateController.start()
			else estimateController.stop()
		},
		stop() {
			pollController.stop()
			estimateController.stop()
		},
		status() {
			return {
				polling: pollController.isActive(),
				estimating: estimateController.isActive()
			}
		}
	}
}
