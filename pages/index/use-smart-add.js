import { createEmptyPlaceDraft } from './use-place-library'
import { previewRecipeLink } from '../../utils/recipe-api'
import { createRecipeFromDraft, getCachedRecipes } from '../../utils/recipe-store'
import { createEmptyDraft } from './constants'
import { detectDraftLinkPlatform, extractSupportedDraftLink, guessDraftTitleFromShareText, normalizeDraftAutoTitle } from './draft-link'
import { formatMealOrderDateText } from './meal-order'
import { defineIndexPageModule } from './page-module'

export function normalizePreviewImageList(source = {}) {
	const values = source.images || source.imageUrls || source.imageURL || source.imageUrl || []
	const items = Array.isArray(values) ? values : [values]
	return items.map((item) => String(item || '').trim()).filter(Boolean)
}

export function stringifyRecipeParsedContent(content) {
	if (!content) return ''
	if (typeof content === 'string') return content.trim()
	const lines = []
	const mainIngredients = Array.isArray(content.mainIngredients) ? content.mainIngredients : []
	const secondaryIngredients = Array.isArray(content.secondaryIngredients) ? content.secondaryIngredients : []
	const steps = Array.isArray(content.steps) ? content.steps : []
	if (mainIngredients.length) lines.push(`主料：${mainIngredients.filter(Boolean).join('、')}`)
	if (secondaryIngredients.length) lines.push(`辅料：${secondaryIngredients.filter(Boolean).join('、')}`)
	if (steps.length) {
		lines.push('步骤：')
		steps.forEach((step, index) => {
			if (typeof step === 'string') {
				if (step.trim()) lines.push(`${index + 1}. ${step.trim()}`)
				return
			}
			const text = [String(step?.title || '').trim(), String(step?.detail || '').trim()]
				.filter(Boolean)
				.join('：')
			if (text) lines.push(`${index + 1}. ${text}`)
		})
	}
	return lines.join('\n').trim()
}

export function buildPlaceDraftFromPreview(draft = {}) {
	return {
		...createEmptyPlaceDraft(),
		...draft,
		images: normalizePreviewImageList(draft)
	}
}

export function buildPlaceDraftFromCandidate(candidate = {}, context = {}) {
	if (candidate.placeDraft) return buildPlaceDraftFromPreview(candidate.placeDraft)
	return createEmptyPlaceDraft({
		name: candidate.name || '',
		type: candidate.type || 'food',
		address: candidate.address || '',
		latitude: candidate.latitude || 0,
		longitude: candidate.longitude || 0,
		price: candidate.price || '',
		source: context.source || 'manual',
		sourceUrl: context.sourceUrl || '',
		images: normalizePreviewImageList(candidate),
		status: 'want'
	})
}

export const smartAddPlaceMethods = {
	closeAddLinkPreviewPanel() {
		this.showAddLinkPreviewPanel = false
	},
	handleManualEntry() {
		// 用户选择手动填写，关闭所有智能识别界面，打开传统编辑表单
		this.showAddLinkPreviewPanel = false
		this.showPlaceCandidateSheet = false
		this.placeEditMode = 'create'
		this.editingPlaceId = ''
		this.placeDraft = createEmptyPlaceDraft()
		this.showPlaceEditSheet = true
	},
	handleParseResult(result) {
		if (result.status === 'recipe_result') {
			this.showAddLinkPreviewPanel = false
			this.handleRecipeParseResult(result)
		} else if (result.status === 'place_candidates') {
			// 打卡地候选结果
			this.placeCandidates = result.candidates || []
			this.placeExtracted = result.extracted || {}
			this.placeParseSource = result.source || 'meituan'
			this.showPlaceCandidateSheet = true
		} else if (result.status === 'partial' && result.contentType === 'place') {
			this.placeExtracted = result.extracted || {}
			this.placeParseSource = result.source || 'other'
			const draft = result.draft || {}
			this.placeDraft = buildPlaceDraftFromPreview(draft)
			this.placeEditMode = 'create'
			this.editingPlaceId = ''
			this.showPlaceEditSheet = true
			uni.showToast({
				title: '已提取基础信息，可继续补充',
				icon: 'none',
				duration: 2000
			})
		} else if (result.status === 'failed') {
			uni.showToast({
				title: result.message || '解析失败，请手动填写',
				icon: 'none',
				duration: 2000
			})
		}
	},
	closePlaceCandidateSheet() {
		this.showPlaceCandidateSheet = false
	},
	handleSelectCandidate(candidate) {
		// 将候选转换为 draft 并打开编辑表单
		this.placeDraft = buildPlaceDraftFromCandidate(candidate, {
			source: this.placeParseSource || 'manual',
			sourceUrl: this.placeExtracted.sourceUrl || ''
		})

		this.showPlaceCandidateSheet = false
		this.placeEditMode = 'create'
		this.editingPlaceId = ''
		this.showPlaceEditSheet = true

		// 提示用户
		uni.showToast({
			title: '已根据分享链接填入，可继续修改后保存',
			icon: 'none',
			duration: 2000
		})
	}
}

export const smartAddDraftMethods = {
createDraftFromContext() {
	const defaultStatus = ['wishlist', 'done'].includes(this.activeStatus) ? this.activeStatus : 'wishlist'
	return createEmptyDraft({
		mealType: this.activeMealType || 'breakfast',
		status: defaultStatus
	})
},
resetDraftAssistState() {
	this.clearDraftLinkPreviewState()
	this.draftAutoTitle = ''
	this.draftTitleTouched = false
	this.draftLinkPreviewPlatform = ''
	this.draftLinkPreviewTitleSource = ''
	this.draftLinkPreviewError = ''
	this.draftLinkPrefillSource = ''
},
clearDraftLinkPreviewState() {
	if (this.draftLinkPreviewTimer) {
		clearTimeout(this.draftLinkPreviewTimer)
		this.draftLinkPreviewTimer = null
	}
	this.draftLinkPreviewRequestID += 1
	this.isDraftLinkPreviewing = false
	this.draftLinkPreviewTitleSource = ''
},
applyDraftAutoTitle(title = '') {
	const normalizedTitle = normalizeDraftAutoTitle(title)
	if (!normalizedTitle) return

	const currentTitle = String(this.draft.title || '').trim()
	const previousAutoTitle = String(this.draftAutoTitle || '').trim()
	const canReplace = !currentTitle || !this.draftTitleTouched || (previousAutoTitle && currentTitle === previousAutoTitle)

	this.draftAutoTitle = normalizedTitle
	if (canReplace) {
		this.draft.title = normalizedTitle
		this.draftTitleTouched = false
	}
},
handleDraftTitleInput(event) {
	const value = String(event?.detail?.value || '')
	this.draft.title = value

	const normalizedTitle = value.trim()
	if (!normalizedTitle) {
		this.draftTitleTouched = false
		return
	}

	const autoTitle = String(this.draftAutoTitle || '').trim()
	this.draftTitleTouched = autoTitle ? normalizedTitle !== autoTitle : true
},
handleDraftMealTypeSelect(value) {
	if (!this.mealTabs.some((tab) => tab.value === value)) return
	this.draft.mealType = value
},
handleDraftStatusSelect(value) {
	if (!this.draftStatusOptions.some((tab) => tab.value === value)) return
	this.draft.status = value
},
handleDraftNoteInput(event) {
	this.draft.note = String(event?.detail?.value || '')
},
handleDraftLinkInput(event) {
	const value = String(event?.detail?.value || '')
	this.draft.link = value
	this.draftLinkPrefillSource = ''
	this.scheduleDraftLinkPreview(value)
},
scheduleDraftLinkPreview(rawInput = '') {
	this.clearDraftLinkPreviewState()
	this.draftLinkPreviewError = ''

	const value = String(rawInput || '').trim()
	const previousAutoTitle = String(this.draftAutoTitle || '').trim()
	if (!value) {
		if (!this.draftTitleTouched && previousAutoTitle && String(this.draft.title || '').trim() === previousAutoTitle) {
			this.draft.title = ''
		}
		this.draftAutoTitle = ''
		this.draftLinkPreviewPlatform = ''
		this.draftLinkPreviewTitleSource = ''
		return
	}

	const platform = detectDraftLinkPlatform(value)
	this.draftLinkPreviewPlatform = platform

	const guessedTitle = guessDraftTitleFromShareText(value)
	if (guessedTitle) {
		this.applyDraftAutoTitle(guessedTitle)
	}

	const mayContainShareLink = /https?:\/\/|www\.|bilibili|b23\.tv|bili2233\.cn|xiaohongshu|xhslink/i.test(value)
	if (!platform && !mayContainShareLink) {
		if (!guessedTitle && !this.draftTitleTouched && previousAutoTitle && String(this.draft.title || '').trim() === previousAutoTitle) {
			this.draft.title = ''
			this.draftAutoTitle = ''
		}
		return
	}

	const requestID = this.draftLinkPreviewRequestID
	this.isDraftLinkPreviewing = true
	this.draftLinkPreviewTimer = setTimeout(async () => {
			try {
				const result = await previewRecipeLink(value)
				if (requestID !== this.draftLinkPreviewRequestID) return

				this.isDraftLinkPreviewing = false
				this.draftLinkPreviewTimer = null
				const resolvedLink = String(result?.canonicalUrl || result?.link || '').trim()
				this.draftLinkPreviewPlatform = detectDraftLinkPlatform(resolvedLink || value) || platform
				this.draftLinkPreviewTitleSource = String(result?.titleSource || '').trim().toLowerCase()

				const previewTitle = normalizeDraftAutoTitle(result?.title || '')
			if (previewTitle) {
				this.applyDraftAutoTitle(previewTitle)
				return
			}

			if (!guessedTitle) {
				this.draftLinkPreviewError = '暂时没识别到菜名，可继续手动填写。'
			}
		} catch (error) {
			if (requestID !== this.draftLinkPreviewRequestID) return
			this.isDraftLinkPreviewing = false
			this.draftLinkPreviewTimer = null
			if (!guessedTitle) {
				this.draftLinkPreviewError = error?.message || '暂时无法识别链接标题，可先手动填写。'
			}
		}
	}, 480)
}

}

export const smartAddRecipeMethods = {
openAddSheet() {
	// V1.1: 打开智能识别面板，而不是直接打开表单
	this.showAddRecipePreviewPanel = true
},
openDietAssistantSheet(initialPrompt = '') {
	this.dietAssistantInitialPrompt = typeof initialPrompt === 'string' ? initialPrompt : ''
	this.showDietAssistantSheet = true
},
closeDietAssistantSheet() {
	this.showDietAssistantSheet = false
},
openAddSheetFromAssistant() {
	this.closeDietAssistantSheet()
	this.openAddSheet()
},
handleDietAssistantRecipesMutated(mutation = {}) {
	if (mutation?.type !== 'recipe_created') return
	this.refreshRecipes({ silent: true })
},
closeAddRecipePreviewPanel() {
	this.showAddRecipePreviewPanel = false
},
handleRecipeManualEntry() {
	// 关闭智能识别面板，打开原始手动表单
	this.showAddRecipePreviewPanel = false
	this.resetDraftAssistState()
	this.draft = this.createDraftFromContext()
	this.showAddSheet = true
},
handleRecipeParseResult(result) {
	// 处理菜品解析结果
	this.showAddRecipePreviewPanel = false

	if (result.status === 'recipe_result' && result.recipeDraft) {
		// 预填表单字段
		this.resetDraftAssistState()
		this.draft = this.createDraftFromContext()

		// 预填菜谱名
		if (result.recipeDraft.title) {
			this.draft.title = result.recipeDraft.title
			this.draftTitleTouched = true
		}

		// 预填链接
		if (result.recipeDraft.link) {
			this.draft.link = result.recipeDraft.link
		}

		// 预填图片
		const recipeImages = normalizePreviewImageList(result.recipeDraft)
		if (recipeImages.length) {
			this.draft.images = recipeImages.slice(0, this.maxRecipeImages)
		}

		// 预填备注（食材 + 步骤）
		const noteParts = []
		if (result.recipeDraft.ingredient) {
			noteParts.push(`食材：${result.recipeDraft.ingredient}`)
		}
		const parsedContentText = stringifyRecipeParsedContent(result.recipeDraft.parsedContent)
		if (parsedContentText) {
			noteParts.push(parsedContentText)
		}
		if (result.recipeDraft.note) {
			noteParts.push(result.recipeDraft.note)
		}
		if (noteParts.length) {
			this.draft.note = noteParts.join('\n\n')
		}

		// 打开表单
		this.showAddSheet = true

		// 提示用户
		uni.showToast({
			title: '已自动填入，可继续修改',
			icon: 'none',
			duration: 2000
		})
	} else {
		// 解析失败或部分成功，直接打开空表单
		this.handleRecipeManualEntry()
	}
},
// handleRecipePreviewTimeoutFallback：preview 请求超时时由菜谱面板触发。
// 用「原始链接 + guessDraftTitleFromShareText 猜测标题」先建占位菜谱（parsedContent 留空，
// 让后端置 parse_status=pending 进自动解析队列），首页立即出现「解析中」徽标，稍后自动补全。
// 提取不到可支持链接则不建占位，回退手动填写，避免误存无链接菜谱。
async handleRecipePreviewTimeoutFallback(payload = {}) {
	this.showAddRecipePreviewPanel = false

	const rawText = String(payload?.text || '').trim()
	const link = extractSupportedDraftLink(rawText)
	if (!link) {
		uni.showToast({
			title: '解析超时，请手动填写',
			icon: 'none',
			duration: 2000
		})
		return
	}

	const guessedTitle = guessDraftTitleFromShareText(rawText)
	const title = guessedTitle || '菜谱整理中'
	const status = this.activeStatus === 'done' ? 'done' : 'wishlist'

	try {
		const recipe = await createRecipeFromDraft({
			title,
			titleSource: 'placeholder',
			link,
			mealType: this.activeMealType || 'breakfast',
			status,
			// 必须保持空内容，让后端 shouldQueueAutoParse 判定为需要自动解析；
			// parsedContentEdited=false 避免被视作用户手动整理。
			parsedContent: {
				mainIngredients: [],
				secondaryIngredients: [],
				steps: []
			},
			parsedContentEdited: false
		})

		// createRecipeFromDraft 已写入本地缓存，这里同步到首页列表，让占位卡立即出现。
		this.applyRecipes(getCachedRecipes())
		if (recipe?.mealType) {
			this.activeMealType = recipe.mealType
		}

		uni.showToast({
			title: '模型繁忙，已转后台，稍后自动补全',
			icon: 'none',
			duration: 2200
		})

		// 15s 后静默刷新一次，尝试拿到后台 worker 已补全的内容；首页不做高频轮询。
		this.clearRecipePreviewTimeoutRefreshTimer()
		this.recipePreviewTimeoutRefreshTimer = setTimeout(() => {
			this.recipePreviewTimeoutRefreshTimer = null
			this.refreshRecipes({ silent: true })
		}, 15000)
	} catch (error) {
		console.error('超时转后台占位菜谱创建失败:', error)
		uni.showToast({
			title: '后台保存失败，请稍后重试',
			icon: 'none',
			duration: 2000
		})
	}
},
clearRecipePreviewTimeoutRefreshTimer() {
	if (!this.recipePreviewTimeoutRefreshTimer) return
	clearTimeout(this.recipePreviewTimeoutRefreshTimer)
	this.recipePreviewTimeoutRefreshTimer = null
},
closeAddSheet() {
	if (this.isSubmittingDraft) return
	this.resetDraftAssistState()
	this.showAddSheet = false
	this.draft = this.createDraftFromContext()
},
chooseDraftImages() {
	const remaining = Math.max(this.maxRecipeImages - this.draft.images.length, 0)
	if (!remaining) {
		uni.showToast({
			title: `最多上传 ${this.maxRecipeImages} 张`,
			icon: 'none'
		})
		return
	}

	uni.chooseImage({
		count: remaining,
		sizeType: ['compressed'],
		sourceType: ['album', 'camera'],
		success: ({ tempFilePaths }) => {
			if (!tempFilePaths || !tempFilePaths.length) return
			const nextImages = [...this.draft.images]
			tempFilePaths.forEach((path) => {
				if (path && !nextImages.includes(path) && nextImages.length < this.maxRecipeImages) {
					nextImages.push(path)
				}
			})
			this.draft.images = nextImages
		}
	})
},
removeDraftImage(index) {
	if (typeof index !== 'number') return
	this.draft.images = this.draft.images.filter((_, currentIndex) => currentIndex !== index)
},
previewDraftImages(index = 0) {
	const urls = Array.isArray(this.draft.images) ? this.draft.images.filter(Boolean) : []
	if (!urls.length) return
	uni.previewImage({
		current: urls[index] || urls[0],
		urls
	})
},
async submitDraft() {
	if (!this.canSubmitDraft || this.isSubmittingDraft) return

	this.isSubmittingDraft = true

	try {
		const newRecipe = await createRecipeFromDraft(this.draft)
		this.applyRecipes(getCachedRecipes())
		this.selectedRecipeId = newRecipe.id
		this.activeSection = 'library'
		this.activeMealType = newRecipe.mealType
		this.activeStatus = 'all'
		this.searchKeyword = ''
		this.showAddSheet = false
		this.resetDraftAssistState()
		this.draft = this.createDraftFromContext()
		try {
			uni.vibrateShort({
				type: 'light'
			})
		} catch (_) {
			// Ignore unsupported vibration capabilities and keep save stable.
		}
		this.showLibraryActionFeedback({
			tone: newRecipe.status === 'done' ? 'done' : 'wishlist',
			title: newRecipe.status === 'done' ? '已保存并标记吃过' : '已加入美食库',
			description: String(newRecipe?.title || '').trim() || '这道菜',
			duration: newRecipe.status === 'done' ? 1680 : 1440,
			showSparkles: newRecipe.status === 'done'
		})
	} catch (error) {
		uni.showToast({
			title: error?.message || '保存失败',
			icon: 'none'
		})
	} finally {
		this.isSubmittingDraft = false
	}
}

}

export const smartAddComputed = {
	canSubmitDraft() {
		return !!this.draft.title.trim()
	},
	draftLinkPlatformLabel() {
		if (this.draftLinkPreviewPlatform === 'bilibili') return 'B 站'
		if (this.draftLinkPreviewPlatform === 'xiaohongshu') return '小红书'
		return '链接'
	},
	draftLinkTitleSourceLabel() {
		if (this.draftLinkPreviewTitleSource === 'ai') return 'AI 识别'
		if (this.draftLinkPreviewTitleSource === 'rule') return '规则识别'
		return ''
	},
	draftTitleAssistText() {
		if (!this.draftAutoTitle) return ''
		const platformLabel = this.draftLinkPlatformLabel
		const sourceLabel = this.draftLinkTitleSourceLabel
		const sourceParts = [platformLabel !== '链接' ? platformLabel : '', sourceLabel].filter(Boolean)
		const sourceSuffix = sourceParts.length ? `（${sourceParts.join(' · ')}）` : ''
		if (this.draftTitleTouched) {
			return `已识别菜名，保留当前填写${sourceSuffix}`.trim()
		}
		return `已识别菜名，可直接保存${sourceSuffix}`.trim()
	},
	draftLinkAssistText() {
		if (this.isDraftLinkPreviewing) {
			return this.draftLinkPlatformLabel === '链接'
				? '正在识别链接标题...'
				: `正在识别${this.draftLinkPlatformLabel}菜名...`
		}
		if (this.draftLinkPreviewError) {
			return this.draftLinkPreviewError
		}
		if (this.draft.link.trim()) {
			return '已粘贴来源内容，系统会自动补标题。'
		}
		return ''
	}
}

export const smartAddModule = defineIndexPageModule({
	name: 'smart-add',
	requires: [
		'draft', 'draftLinkPreviewRequestID', 'clearDraftLinkPreviewState',
		'clearRecipePreviewTimeoutRefreshTimer'
	],
	methods: {
		...smartAddPlaceMethods,
		...smartAddDraftMethods,
		...smartAddRecipeMethods
	},
	computed: smartAddComputed,
	lifecycle: {
		deactivate() {
			this.clearDraftLinkPreviewState()
		},
		dispose() {
			this.clearRecipePreviewTimeoutRefreshTimer()
		}
	}
})
