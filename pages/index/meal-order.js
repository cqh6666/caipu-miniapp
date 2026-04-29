const mealOrderWeekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const mealOrderPendingActionStorageKey = 'caipu-miniapp-meal-order-pending-action'

function padDateNumber(value) {
	return String(Number(value) || 0).padStart(2, '0')
}

export function toISODate(value = new Date()) {
	const date = value instanceof Date ? value : new Date(value)
	if (Number.isNaN(date.getTime())) return ''
	return `${date.getFullYear()}-${padDateNumber(date.getMonth() + 1)}-${padDateNumber(date.getDate())}`
}

function parseISODate(value = '') {
	const match = String(value || '').trim().match(/^(\d{4})-(\d{2})-(\d{2})$/)
	if (!match) return null
	const year = Number(match[1])
	const month = Number(match[2]) - 1
	const day = Number(match[3])
	const date = new Date(year, month, day)
	if (Number.isNaN(date.getTime())) return null
	if (date.getFullYear() !== year || date.getMonth() !== month || date.getDate() !== day) return null
	return date
}

export function normalizeMealOrderDate(value = '') {
	const date = parseISODate(value)
	return date ? toISODate(date) : ''
}

export function addDaysFromISODate(baseDate = '', offset = 0) {
	const date = parseISODate(baseDate) || new Date()
	date.setDate(date.getDate() + Number(offset || 0))
	return toISODate(date)
}

export function nextWeekendISODate(baseDate = '') {
	const seed = parseISODate(baseDate) || new Date()
	for (let index = 0; index < 8; index += 1) {
		const candidate = new Date(seed)
		candidate.setDate(seed.getDate() + index)
		const day = candidate.getDay()
		if (day === 0 || day === 6) {
			return toISODate(candidate)
		}
	}
	return toISODate(seed)
}

export function formatMealOrderDateText(value = '') {
	const date = parseISODate(value)
	if (!date) return '--'
	const month = padDateNumber(date.getMonth() + 1)
	const day = padDateNumber(date.getDate())
	const weekday = mealOrderWeekdays[date.getDay()] || ''
	return `${month}月${day}日 ${weekday}`
}

export function formatMealOrderDateParts(value = '') {
	const date = parseISODate(value)
	if (!date) return { dateText: '', weekday: '', isoDate: '' }
	const month = padDateNumber(date.getMonth() + 1)
	const day = padDateNumber(date.getDate())
	const weekday = mealOrderWeekdays[date.getDay()] || ''
	return {
		dateText: `${month}月${day}日`,
		weekday,
		isoDate: toISODate(date)
	}
}

export function formatMealOrderHeaderTitle(value = '') {
	const date = parseISODate(value)
	if (!date) return '这天的小菜单'
	return `${date.getMonth() + 1}月${date.getDate()}日的小菜单`
}

export function createEmptyMealOrderStore() {
	return {
		drafts: {},
		submitted: []
	}
}

function normalizeMealOrderItem(raw = {}) {
	const quantity = Math.max(1, Math.min(9, Number(raw.quantity) || 1))
	const recipeId = String(raw.recipeId || '').trim()
	if (!recipeId) return null
	const titleSnapshot = String(raw.titleSnapshot || raw.title || '').trim() || '未命名菜品'
	const imageSnapshot = String(raw.imageSnapshot || raw.image || '').trim()
	const mealTypeSnapshot = String(raw.mealTypeSnapshot || raw.mealType || '').trim() || 'main'

	return {
		recipeId,
		quantity,
		titleSnapshot,
		imageSnapshot,
		mealTypeSnapshot
	}
}

export function normalizeMealOrderDraft(raw = {}, planDate = '') {
	const normalizedPlanDate = normalizeMealOrderDate(planDate || raw.planDate || '')
	const items = (Array.isArray(raw.items) ? raw.items : [])
		.map((item) => normalizeMealOrderItem(item))
		.filter(Boolean)
	const note = String(raw.note || '').trim()
	const updatedAt = String(raw.updatedAt || '').trim()

	return {
		planDate: normalizedPlanDate,
		items,
		note,
		updatedAt
	}
}

export function normalizeMealOrderRecord(raw = {}) {
	const planDate = normalizeMealOrderDate(raw.planDate || '')
	const items = (Array.isArray(raw.items) ? raw.items : [])
		.map((item) => normalizeMealOrderItem(item))
		.filter(Boolean)
	const note = String(raw.note || '').trim()
	const submittedAt = String(raw.submittedAt || '').trim()
	if (!planDate || !items.length) return null

	return {
		id: String(raw.id || '').trim() || `mord_${Date.now()}`,
		planDate,
		items,
		note,
		submittedAt
	}
}

export function normalizeMealOrderStore(raw = {}) {
	const source = raw && typeof raw === 'object' ? raw : {}
	const draftSource = source.drafts && typeof source.drafts === 'object' ? source.drafts : {}
	const drafts = {}
	Object.keys(draftSource).forEach((dateKey) => {
		const normalizedDate = normalizeMealOrderDate(dateKey)
		if (!normalizedDate) return
		const draft = normalizeMealOrderDraft(draftSource[dateKey], normalizedDate)
		if (!draft.items.length && !draft.note) return
		drafts[normalizedDate] = draft
	})

	const submitted = (Array.isArray(source.submitted) ? source.submitted : [])
		.map((record) => normalizeMealOrderRecord(record))
		.filter(Boolean)
		.sort((left, right) => String(right.submittedAt || '').localeCompare(String(left.submittedAt || '')))
		.slice(0, 60)

	return {
		drafts,
		submitted
	}
}

export function buildMealPlanPayload(raw = {}) {
	const draft = normalizeMealOrderDraft(raw, raw?.planDate)
	return {
		items: draft.items.map((item) => ({
			recipeId: item.recipeId,
			quantity: item.quantity,
			titleSnapshot: item.titleSnapshot,
			imageSnapshot: item.imageSnapshot,
			mealTypeSnapshot: item.mealTypeSnapshot
		})),
		note: draft.note
	}
}

export function buildMealOrderDishSummary(items = []) {
	const names = (Array.isArray(items) ? items : [])
		.map((item) => String(item?.titleSnapshot || '').trim())
		.filter(Boolean)
		.slice(0, 3)
	if (!names.length) return '还没有菜'
	return names.join(' / ')
}

function normalizePendingMealOrderAction(raw = {}) {
	const source = raw && typeof raw === 'object' ? raw : {}
	const kind = String(source.kind || '').trim()
	if (!kind) return null

	const normalized = {
		kind,
		kitchenId: Math.max(0, Number(source.kitchenId) || 0),
		planDate: normalizeMealOrderDate(source.planDate || ''),
		message: String(source.message || '').trim()
	}

	if (normalized.kind === 'resume' && !normalized.planDate) {
		return null
	}
	if (!['reload', 'resume'].includes(normalized.kind)) {
		return null
	}

	return normalized
}

export function writePendingMealOrderAction(raw = {}) {
	const action = normalizePendingMealOrderAction(raw)
	if (!action) {
		uni.removeStorageSync(mealOrderPendingActionStorageKey)
		return
	}
	uni.setStorageSync(mealOrderPendingActionStorageKey, action)
}

export function consumePendingMealOrderAction() {
	const action = normalizePendingMealOrderAction(uni.getStorageSync(mealOrderPendingActionStorageKey))
	uni.removeStorageSync(mealOrderPendingActionStorageKey)
	return action
}
