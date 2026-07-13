import { getCurrentKitchenId } from '../../utils/auth'
import { defineIndexPageModule } from './page-module'
import { listMealPlanStore, saveMealPlanDraft, submitMealPlan as submitMealPlanRequest } from '../../utils/meal-plan-api'
import {
	addDaysFromISODate,
	buildMealOrderDishSummary,
	buildMealPlanPayload,
	createEmptyMealOrderStore,
	formatMealOrderDateParts,
	formatMealOrderDateText,
	nextWeekendISODate,
	normalizeMealOrderDate,
	normalizeMealOrderDraft,
	normalizeMealOrderRecord,
	normalizeMealOrderStore,
	toISODate
} from './meal-order'

export function createMealOrderDraftSyncController(syncDraft, defaultDelay = 180) {
	let timer = null
	let generation = 0
	return {
		schedule(delay = defaultDelay) {
			this.cancel()
			const scheduledGeneration = generation
			timer = setTimeout(async () => {
				timer = null
				if (scheduledGeneration !== generation) return
				await syncDraft()
			}, Math.max(Number(delay) || 0, 0))
		},
		cancel() {
			generation += 1
			if (!timer) return
			clearTimeout(timer)
			timer = null
		},
		hasPending() {
			return !!timer
		}
	}
}

export function buildMealOrderItem(recipe = {}, extractImages = null) {
	const recipeId = String(recipe.id || '').trim()
	if (!recipeId) return null
	const resolvedImages = typeof extractImages === 'function' ? extractImages(recipe) : null
	const images = Array.isArray(resolvedImages)
		? resolvedImages
		: Array.isArray(recipe.imageUrls)
		? recipe.imageUrls
		: Array.isArray(recipe.images)
			? recipe.images
			: []
	return {
		recipeId,
		quantity: 1,
		titleSnapshot: String(recipe.title || '').trim() || '未命名菜品',
		imageSnapshot: String(images[0] || recipe.image || '').trim(),
		mealTypeSnapshot: String(recipe.mealType || '').trim() || 'main'
	}
}

export function upsertMealOrderItem(items = [], nextItem = null) {
	if (!nextItem?.recipeId) return [...items]
	const nextItems = [...items]
	const index = nextItems.findIndex((item) => item.recipeId === nextItem.recipeId)
	if (index < 0) nextItems.push(nextItem)
	else nextItems[index] = { ...nextItems[index], ...nextItem }
	return nextItems
}

export const mealOrderMethods = {
async loadMealOrderStore(options = {}) {
	const { silent = true } = options
	const kitchenId = Number(getCurrentKitchenId()) || 0
	if (!kitchenId) {
		this.mealOrderStoreLoadedKitchenId = 0
		this.applyMealOrderStore(createEmptyMealOrderStore())
		return createEmptyMealOrderStore()
	}

	const requestID = this.mealOrderStoreRequestID + 1
	this.mealOrderStoreRequestID = requestID
	const contextID = this.mealOrderSyncContextID
	const localVersion = this.mealOrderLocalVersion

	try {
		const store = await listMealPlanStore(kitchenId)
		if (
			requestID !== this.mealOrderStoreRequestID ||
			contextID !== this.mealOrderSyncContextID ||
			localVersion !== this.mealOrderLocalVersion ||
			kitchenId !== Number(this.currentKitchenId)
		) {
			return normalizeMealOrderStore(this.mealOrderStore)
		}
		this.applyMealOrderStore(store)
		this.mealOrderStoreLoadedKitchenId = kitchenId
		return store
	} catch (error) {
		if (!silent) {
			uni.showToast({
				title: error?.message || '加载菜单失败',
				icon: 'none'
			})
		}
		return normalizeMealOrderStore(this.mealOrderStore)
	}
},
clearMealOrderDraftSyncTimer() {
	this.mealOrderDraftSyncController?.cancel()
},
ensureMealOrderDraftSyncController() {
	if (!this.mealOrderDraftSyncController) {
		this.mealOrderDraftSyncController = createMealOrderDraftSyncController(
			() => this.syncMealOrderDraft({ silent: true }),
			0
		)
	}
	return this.mealOrderDraftSyncController
},
resetMealOrderState() {
	this.clearMealOrderDraftSyncTimer()
	this.clearMealOrderModeMotionTimer()
	this.clearRecipeStatusFeedback()
	this.mealOrderStore = createEmptyMealOrderStore()
	this.mealOrderDate = ''
	this.isMealOrderMode = false
	this.mealOrderModeMotionState = ''
	this.showMealOrderDateSheet = false
	this.showMealOrderCartSheet = false
	this.showMealOrderCheckoutSheet = false
	this.showMealOrderSuccessSheet = false
	this.mealOrderLastSubmittedDate = ''
	this.mealOrderSpotlightIndex = 0
	this.mealOrderSpotlightMotionDirection = ''
	this.mealOrderSpotlightMotionTick = 0
	this.mealOrderSpotlightTouchStartX = 0
	this.mealOrderSpotlightTouchStartY = 0
	this.mealOrderSpotlightSuppressTap = false
},
stageMealOrderDraft(updater) {
	const date = normalizeMealOrderDate(this.mealOrderDate)
	if (!date || typeof updater !== 'function') return

	const current = normalizeMealOrderDraft(this.mealOrderStore?.drafts?.[date], date)
	const nextRawDraft = updater({
		...current,
		items: current.items.map((item) => ({ ...item }))
	})
	const nextDraft = normalizeMealOrderDraft(nextRawDraft, date)
	const nextDrafts = {
		...(this.mealOrderStore?.drafts || {})
	}

	if (!nextDraft.items.length && !String(nextDraft.note || '').trim()) {
		delete nextDrafts[date]
	} else {
		nextDrafts[date] = {
			...nextDraft,
			updatedAt: new Date().toISOString()
		}
	}

	this.mealOrderStore = {
		...(this.mealOrderStore || createEmptyMealOrderStore()),
		drafts: nextDrafts
	}
	this.mealOrderLocalVersion += 1
},
scheduleMealOrderDraftSync(delay = 0) {
	const date = normalizeMealOrderDate(this.mealOrderDate)
	if (!date || !getCurrentKitchenId()) return
	this.ensureMealOrderDraftSyncController().schedule(delay)
},
async syncMealOrderDraft(options = {}) {
	const { silent = false } = options
	if (this.isSubmittingMealOrder) return null
	const kitchenId = Number(getCurrentKitchenId()) || 0
	const date = normalizeMealOrderDate(this.mealOrderDate)
	if (!kitchenId || !date) return null

	this.clearMealOrderDraftSyncTimer()
	const localVersion = this.mealOrderLocalVersion
	const contextID = this.mealOrderSyncContextID
	const draft = normalizeMealOrderDraft(this.mealOrderStore?.drafts?.[date], date)

	try {
		const store = await saveMealPlanDraft(kitchenId, date, buildMealPlanPayload(draft))
		if (
			localVersion === this.mealOrderLocalVersion &&
			contextID === this.mealOrderSyncContextID &&
			kitchenId === Number(this.currentKitchenId)
		) {
			this.applyMealOrderStore(store)
			this.mealOrderStoreLoadedKitchenId = kitchenId
		}
		return store
	} catch (error) {
		if (!silent) {
			uni.showToast({
				title: error?.message || '保存菜单失败',
				icon: 'none'
			})
		}
		return null
	}
},
buildMealOrderItemFromRecipe(recipe = {}) {
	return buildMealOrderItem(recipe, extractRecipeImages)
},
findMealOrderSubmittedByDate(planDate = '') {
	const normalizedDate = normalizeMealOrderDate(planDate)
	if (!normalizedDate) return null
	const submitted = (Array.isArray(this.mealOrderStore?.submitted) ? this.mealOrderStore.submitted : [])
		.map((record) => normalizeMealOrderRecord(record))
		.filter(Boolean)
	return submitted.find((record) => record.planDate === normalizedDate) || null
},
focusMealOrderSpotlightRecord(planDate = '', type = 'submitted') {
	const normalizedDate = normalizeMealOrderDate(planDate)
	if (!normalizedDate) return false
	const currentIndex = this.mealOrderSpotlightRecordIndex
	const targetIndex = this.mealOrderSpotlightRecords.findIndex(
		(record) => record.planDate === normalizedDate && record.type === type
	)
	if (targetIndex < 0) return false
	if (targetIndex !== currentIndex) {
		this.bumpMealOrderSpotlightMotion(targetIndex > currentIndex ? 'next' : 'previous')
	}
	this.mealOrderSpotlightIndex = targetIndex
	return true
},
mealOrderHasRecipe(recipeId = '') {
	const targetRecipeId = String(recipeId || '').trim()
	if (!targetRecipeId) return false
	return this.mealOrderCurrentDraft.items.some((item) => item.recipeId === targetRecipeId)
},
handleMealOrderSpotlightTap() {
	if (this.mealOrderSpotlightSuppressTap) {
		this.mealOrderSpotlightSuppressTap = false
		return
	}
	const record = this.mealOrderSpotlightRecord
	if (!record) {
		this.openMealOrderDateSheet()
		return
	}
	this.openMealOrderDetail(record)
},
handleMealOrderSpotlightTouchStart(event) {
	const touch = event?.touches?.[0] || event?.changedTouches?.[0]
	if (!touch) return
	this.mealOrderSpotlightTouchStartX = Number(touch.clientX || touch.pageX || 0)
	this.mealOrderSpotlightTouchStartY = Number(touch.clientY || touch.pageY || 0)
	this.mealOrderSpotlightSuppressTap = false
},
handleMealOrderSpotlightTouchEnd(event) {
	const touch = event?.changedTouches?.[0] || event?.touches?.[0]
	const startX = Number(this.mealOrderSpotlightTouchStartX || 0)
	const startY = Number(this.mealOrderSpotlightTouchStartY || 0)
	this.mealOrderSpotlightTouchStartX = 0
	this.mealOrderSpotlightTouchStartY = 0
	if (!touch || this.mealOrderSpotlightRecords.length < 2 || (!startX && !startY)) return

	const endX = Number(touch.clientX || touch.pageX || 0)
	const endY = Number(touch.clientY || touch.pageY || 0)
	const diffX = endX - startX
	const diffY = endY - startY
	if (Math.abs(diffX) < 56 || Math.abs(diffX) <= Math.abs(diffY)) return

	this.shiftMealOrderSpotlight(diffX > 0 ? 'next' : 'previous')
	this.mealOrderSpotlightSuppressTap = true
},
shiftMealOrderSpotlight(direction = 'next') {
	const total = this.mealOrderSpotlightRecords.length
	if (total < 2) return
	const step = direction === 'previous' ? -1 : 1
	this.mealOrderSpotlightIndex = (this.mealOrderSpotlightRecordIndex + step + total) % total
	this.bumpMealOrderSpotlightMotion(direction)
},
closeMealOrderSuccessSheet() {
	this.showMealOrderSuccessSheet = false
},
viewMealOrderSuccessRecord() {
	this.showMealOrderSuccessSheet = false
	this.openMealOrderDetail({
		planDate: this.mealOrderLastSubmittedDate,
		type: 'submitted'
	})
},
planNextMealOrder() {
	this.showMealOrderSuccessSheet = false
	this.showMealOrderDateSheet = true
},
drawTonight() {
	const pool = this.buildTonightPickPool()
	if (!pool.length) {
		uni.showToast({
			title: this.hasSearchKeyword || this.activeStatus !== 'all' ? '当前筛选里还没有可选菜' : '先添加几道菜吧',
			icon: 'none'
		})
		return
	}
	const picked = this.pickTonightRecipe(pool)
	this.presentTonightPick(picked, pool, this.buildTonightPickContext(pool), 'enter')
},
openMealOrderDateSheet() {
	if (!getCurrentKitchenId()) {
		uni.showToast({
			title: '请先完成空间同步',
			icon: 'none'
		})
		return
	}
	this.showMealOrderSuccessSheet = false
	this.showMealOrderDateSheet = true
},
closeMealOrderDateSheet() {
	this.showMealOrderDateSheet = false
},
startMealOrderMode(planDate = '') {
	const normalizedDate = normalizeMealOrderDate(planDate)
	if (!normalizedDate) return
	this.mealOrderDate = normalizedDate
	this.activeSection = 'library'
	this.isMealOrderMode = true
	this.showMealOrderDateSheet = false
	this.showMealOrderSuccessSheet = false
},
exitMealOrderMode() {
	this.syncMealOrderDraft({ silent: true })
	this.isMealOrderMode = false
	this.showMealOrderCartSheet = false
	this.showMealOrderCheckoutSheet = false
	this.showMealOrderSuccessSheet = false
},
addMealOrderRecipe(recipe = {}) {
	if (!this.isMealOrderMode || !this.mealOrderDate) {
		this.openMealOrderDateSheet()
		return
	}
	const nextItem = this.buildMealOrderItemFromRecipe(recipe)
	if (!nextItem) return
	this.stageMealOrderDraft((draft) => {
		return {
			...draft,
			items: upsertMealOrderItem(draft.items, nextItem)
		}
	})
	this.scheduleMealOrderDraftSync()
},
toggleMealOrderRecipe(recipe = {}) {
	const recipeId = String(recipe?.id || '').trim()
	if (!recipeId) return
	if (this.mealOrderHasRecipe(recipeId)) {
		this.removeMealOrderRecipe(recipeId)
		uni.showToast({
			title: '已移出这天菜单',
			icon: 'none'
		})
		return
	}
	this.addMealOrderRecipe(recipe)
	uni.showToast({
		title: '已加入这天菜单',
		icon: 'none'
	})
},
removeMealOrderRecipe(recipeId = '') {
	const targetRecipeId = String(recipeId || '').trim()
	if (!targetRecipeId || !this.mealOrderDate) return
	this.stageMealOrderDraft((draft) => {
		const nextItems = draft.items.filter((item) => item.recipeId !== targetRecipeId)
		return {
			...draft,
			items: nextItems
		}
	})
	this.scheduleMealOrderDraftSync()
},
openMealOrderCartSheet() {
	if (!this.isMealOrderMode || !this.mealOrderDate) {
		this.openMealOrderDateSheet()
		return
	}
	this.showMealOrderCartSheet = true
},
closeMealOrderCartSheet() {
	this.showMealOrderCartSheet = false
},
openMealOrderCheckoutSheet() {
	if (!this.mealOrderCanCheckout) return
	this.showMealOrderCartSheet = false
	this.showMealOrderCheckoutSheet = true
},
closeMealOrderCheckoutSheet() {
	this.showMealOrderCheckoutSheet = false
},
handleMealOrderNoteInput(event) {
	const value = String(event?.detail?.value || '')
	this.stageMealOrderDraft((draft) => ({
		...draft,
		note: value
	}))
	this.scheduleMealOrderDraftSync(320)
},
clearMealOrderCart() {
	if (!this.mealOrderCartItems.length && !String(this.mealOrderDraftNote || '').trim()) return
	uni.showModal({
		title: '清空菜单',
		content: '确认清空这一天已经安排的菜单吗？',
		confirmText: '清空',
		success: ({ confirm }) => {
			if (!confirm) return
			this.stageMealOrderDraft((draft) => ({
				...draft,
				items: [],
				note: ''
			}))
			this.scheduleMealOrderDraftSync()
		}
	})
},
async submitMealOrder() {
	if (!this.mealOrderCanCheckout || !this.mealOrderDate || this.isSubmittingMealOrder) return
	const kitchenId = Number(getCurrentKitchenId()) || 0
	if (!kitchenId) return
	const currentDraft = normalizeMealOrderDraft(this.mealOrderCurrentDraft, this.mealOrderDate)
	this.clearMealOrderDraftSyncTimer()
	const contextID = this.mealOrderSyncContextID + 1
	this.mealOrderSyncContextID = contextID
	this.isSubmittingMealOrder = true

	try {
		const store = await submitMealPlanRequest(kitchenId, this.mealOrderDate, buildMealPlanPayload(currentDraft))
		if (contextID !== this.mealOrderSyncContextID || kitchenId !== Number(this.currentKitchenId)) {
			return
		}
		this.applyMealOrderStore(store)
		this.mealOrderStoreLoadedKitchenId = kitchenId
		this.showMealOrderCheckoutSheet = false
		this.showMealOrderCartSheet = false
		this.isMealOrderMode = false
		this.mealOrderLastSubmittedDate = this.mealOrderDate
		this.focusMealOrderSpotlightRecord(this.mealOrderDate, 'submitted')
		this.showMealOrderSuccessSheet = true
	} catch (error) {
		uni.showToast({
			title: error?.message || '提交菜单失败',
			icon: 'none'
		})
	} finally {
		this.isSubmittingMealOrder = false
	}
}

}

export const mealOrderComputed = {
	mealOrderDateStart() {
		return toISODate(new Date())
	},
	mealOrderDatePickerValue() {
		return normalizeMealOrderDate(this.mealOrderDate) || this.mealOrderDateStart
	},
	mealOrderDateText() {
		return formatMealOrderDateText(this.mealOrderDate)
	},
	mealOrderDateStatusMetaMap() {
		const result = {}

		Object.values(this.mealOrderStore?.drafts || {})
			.map((draft) => normalizeMealOrderDraft(draft, draft?.planDate))
			.filter((draft) => draft.planDate && draft.items.length)
			.forEach((draft) => {
				result[draft.planDate] = {
					tag: '草稿中',
					text: `已选 ${draft.items.length} 道`,
					tone: 'draft'
				}
			})

		;(Array.isArray(this.mealOrderStore?.submitted) ? this.mealOrderStore.submitted : [])
			.map((record) => normalizeMealOrderRecord(record))
			.filter(Boolean)
			.forEach((record) => {
				if (result[record.planDate]) {
					result[record.planDate] = {
						tag: '待修改',
						text: `草稿 ${result[record.planDate].text.replace('已选 ', '')} · 原安排保留`,
						tone: 'editing'
					}
					return
				}
				result[record.planDate] = {
					tag: '已安排',
					text: `已安排 ${record.items.length} 道`,
					tone: 'submitted'
				}
			})

		return result
	},
	mealOrderQuickDateOptions() {
		const today = this.mealOrderDateStart
		const options = [
			{ label: '今天', value: today },
			{ label: '明天', value: addDaysFromISODate(today, 1) },
			{ label: '周末', value: nextWeekendISODate(today) }
		]
		const seen = new Set()
		return options
			.filter((option) => {
				if (!option.value || seen.has(option.value)) return false
				seen.add(option.value)
				return true
			})
			.map((option) => ({
				...option,
				dateText: formatMealOrderDateText(option.value),
				statusTag: this.mealOrderDateStatusMetaMap[option.value]?.tag || '',
				statusText: this.mealOrderDateStatusMetaMap[option.value]?.text || '',
				statusTone: this.mealOrderDateStatusMetaMap[option.value]?.tone || ''
			}))
	},
	mealOrderCurrentDraft() {
		const date = normalizeMealOrderDate(this.mealOrderDate)
		if (!date) {
			return normalizeMealOrderDraft({}, '')
		}
		return normalizeMealOrderDraft(this.mealOrderStore?.drafts?.[date], date)
	},
	mealOrderCartItems() {
		const recipeMap = this.recipes.reduce((result, recipe) => {
			result[recipe.id] = recipe
			return result
		}, {})
		return this.mealOrderCurrentDraft.items.map((item) => {
			const recipe = recipeMap[item.recipeId] || {}
			const title = item.titleSnapshot || recipe.title || '未命名菜品'
			const mealType = item.mealTypeSnapshot || recipe.mealType || 'main'
			const mealTypeLabel = mealTypeLabelMap[mealType] || '正餐'
			return {
				...item,
				title,
				mealTypeLabel,
				imageSnapshot: String(item.imageSnapshot || '').trim()
			}
		})
	},
	mealOrderDraftNote() {
		return String(this.mealOrderCurrentDraft.note || '')
	},
	mealOrderCartDishCount() {
		return this.mealOrderCartItems.length
	},
	mealOrderCanCheckout() {
		return this.mealOrderCartDishCount > 0 && !this.isSubmittingMealOrder
	},
	mealOrderFloatingTitle() {
		if (this.mealOrderCanCheckout) {
			return `已选 ${this.mealOrderCartDishCount} 道`
		}
		return '还没选菜'
	},
	mealOrderFloatingActionText() {
		return '去确认'
	},
	mealOrderCheckoutHelperText() {
		return '提交后，这天菜单会立即同步给空间成员。之后想改，我们会先带出草稿，不会直接覆盖原安排。'
	},
	isLibraryMealOrderMode() {
		return this.activeSection === 'library' && this.isMealOrderMode && !!normalizeMealOrderDate(this.mealOrderDate)
	},
	showMealOrderFloatingBar() {
		return this.isLibraryMealOrderMode
	},
	mealOrderSpotlightRecords() {
		const today = this.mealOrderDateStart
		const drafts = Object.values(this.mealOrderStore?.drafts || {})
			.map((draft) => normalizeMealOrderDraft(draft, draft?.planDate))
			.filter((draft) => draft.planDate && draft.items.length)
			.map((draft) => ({
				id: `draft:${draft.planDate}`,
				type: 'draft',
				planDate: draft.planDate,
				items: draft.items,
				note: draft.note
			}))
		const submitted = (Array.isArray(this.mealOrderStore?.submitted) ? this.mealOrderStore.submitted : [])
			.map((record) => normalizeMealOrderRecord(record))
			.filter(Boolean)
			.map((record) => ({
				id: `submitted:${record.planDate}`,
				type: 'submitted',
				planDate: record.planDate,
				items: record.items,
				note: record.note
			}))
		const allRecords = [...drafts, ...submitted]
		const sortRecords = (left, right) => {
			const byDate = String(left.planDate || '').localeCompare(String(right.planDate || ''))
			if (byDate) return byDate
			if (left.type === right.type) return 0
			return left.type === 'draft' ? -1 : 1
		}
		const upcoming = allRecords
			.filter((record) => record.planDate >= today)
			.sort(sortRecords)
		return upcoming
	},
	mealOrderSpotlightRecordIndex() {
		const total = this.mealOrderSpotlightRecords.length
		if (!total) return 0
		const current = Number(this.mealOrderSpotlightIndex) || 0
		return Math.min(Math.max(current, 0), total - 1)
	},
	mealOrderSpotlightRecord() {
		return this.mealOrderSpotlightRecords[this.mealOrderSpotlightRecordIndex] || null
	},
	mealOrderSpotlightDateText() {
		const record = this.mealOrderSpotlightRecord
		if (!record) return ''
		return formatMealOrderDateParts(record.planDate).dateText
	},
	mealOrderSpotlightWeekday() {
		const record = this.mealOrderSpotlightRecord
		if (!record) return ''
		return formatMealOrderDateParts(record.planDate).weekday
	},
	mealOrderSpotlightLeadText() {
		const record = this.mealOrderSpotlightRecord
		if (!record) return ''
		return record.planDate === this.mealOrderDateStart ? '今日菜单' : '接下来'
	},
	mealOrderSpotlightStatusText() {
		const record = this.mealOrderSpotlightRecord
		if (!record) return ''
		return record.type === 'submitted' ? '已安排' : '草稿'
	},
	mealOrderSpotlightStatusKind() {
		const record = this.mealOrderSpotlightRecord
		if (!record) return ''
		return record.type === 'submitted' ? 'submitted' : 'draft'
	},
	mealOrderSpotlightDesc() {
		const record = this.mealOrderSpotlightRecord
		if (!record) return '点右侧安排菜单，先挑一天'
		return buildMealOrderDishSummary(record.items)
	},
	mealOrderSpotlightCountText() {
		const record = this.mealOrderSpotlightRecord
		const count = Array.isArray(record?.items) ? record.items.length : 0
		if (!count) return ''
		return record.type === 'draft' ? `已选 ${count} 道` : `${count} 道菜`
	},
	mealOrderSpotlightMetaText() {
		const total = this.mealOrderSpotlightRecords.length
		if (total < 2) return ''
		return `${this.mealOrderSpotlightRecordIndex + 1}/${total}`
	},
	mealOrderSuccessRecord() {
		return this.findMealOrderSubmittedByDate(this.mealOrderLastSubmittedDate)
	},
	mealOrderSuccessDateText() {
		return formatMealOrderDateText(this.mealOrderLastSubmittedDate)
	},
	mealOrderSuccessDishCount() {
		return Array.isArray(this.mealOrderSuccessRecord?.items) ? this.mealOrderSuccessRecord.items.length : 0
	},
	mealOrderSuccessDishSummary() {
		const record = this.mealOrderSuccessRecord
		if (!record) return '这天的菜单已经排好了。'
		return buildMealOrderDishSummary(record.items)
	},
	mealOrderSuccessNote() {
		return String(this.mealOrderSuccessRecord?.note || '').trim()
	},
	librarySummary() {
		if (!this.currentKitchenName && this.syncErrorMessage) {
			return this.syncErrorMessage
		}
		return this.isSyncing ? '正在同步这份菜单。' : '按餐别整理，想吃和吃过更清楚'
	}
}

export const mealOrderModule = defineIndexPageModule({
	name: 'meal-order',
	requires: [
		'mealOrderStore', 'mealOrderDate', 'isSubmittingMealOrder',
		'syncMealOrderDraft', 'clearMealOrderDraftSyncTimer', 'clearMealOrderModeMotionTimer'
	],
	methods: mealOrderMethods,
	computed: mealOrderComputed,
	lifecycle: {
		deactivate() {
			if (!this.isSubmittingMealOrder) this.syncMealOrderDraft({ silent: true })
			this.clearMealOrderDraftSyncTimer()
			this.clearMealOrderModeMotionTimer()
		}
	}
})
