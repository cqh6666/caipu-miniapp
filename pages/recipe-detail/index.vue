<template>
	<view class="detail-page" :class="{ 'detail-page--public': isPublicView }">
		<PublicReadonlyBanner
			:visible="isPublicView && !!recipe && !publicViewLoadFailed"
			:banner-text="publicBannerText"
			:explain-body="publicExplainBody"
			:show-explain="showPublicReadOnlyExplain"
			@open="openPublicReadOnlyExplain"
			@close="closePublicReadOnlyExplain"
		/>
		<template v-if="recipe">
			<scroll-view class="detail-scroll" scroll-y>
				<RecipeHero
					:display-recipe-images="displayRecipeImages"
					:is-public-view="isPublicView"
					:recipe-images="recipeImages"
					:can-show-hero-action-menu="canShowHeroActionMenu"
					:meal-label="mealLabel"
					:status-label="statusLabel"
					:is-pinned="isPinned"
					:recipe="recipe"
					:hero-image-index="heroImageIndex"
					:is-uploading-hero-image="isUploadingHeroImage"
					@tap="handleHeroCardTap"
					@swiper-change="handleHeroSwiperChange"
					@image-error="handleRecipeImageError"
					@open-menu="openHeroActionMenu"
				/>

				<!--
					P2-A: 「一图看懂」与「做法整理」合并为统一「做法」卡片
					- 顶部 Tab：仅当有流程图时显示，默认选中「一图看懂」
					- 右上 ⋯：合并菜单（重新生成 / 重新整理 / 查看详情）
					- 底部：合并元信息行（AI 生成 · MM-DD · 来源）
				-->
				<RecipeCookingPanel
					:is-cooking-active="isCookingActive"
					:cooking-active-label="cookingActiveLabel"
					:has-cooking-menu-items="hasCookingMenuItems"
					:has-flowchart="hasFlowchart"
					:active-cooking-tab="activeCookingTab"
					:flowchart-status-meta="flowchartStatusMeta"
					:flowchart-status-value="flowchartStatusValue"
					:flowchart-status-description="flowchartStatusDescription"
					:parse-status-meta="parseStatusMeta"
					:parse-status-value="parseStatusValue"
					:parse-status-description="parseStatusDescription"
					:show-cooking-flowchart-view="showCookingFlowchartView"
					:flowchart-display-image-url="flowchartDisplayImageUrl"
					:can-request-flowchart="canRequestFlowchart"
					:can-generate-flowchart="canGenerateFlowchart"
					:is-flowchart-active="isFlowchartActive"
					:is-generating-flowchart="isGeneratingFlowchart"
					:flowchart-action-text="flowchartActionText"
					:show-flowchart-stale-hint="showFlowchartStaleHint"
					:has-meaningful-parsed-content="hasMeaningfulParsedContent"
					:is-public-view="isPublicView"
					:show-cooking-steps-view="showCookingStepsView"
					:parsed-main-ingredients="parsedMainIngredients"
					:can-copy-ingredient-list="canCopyIngredientList"
					:parsed-secondary-groups="parsedSecondaryGroups"
					:parsed-steps="parsedSteps"
					:completed-step-count="completedStepCount"
					:can-request-parse="canRequestParse"
					:flowchart-empty-text="flowchartEmptyText"
					:cooking-footer-text="cookingFooterText"
					:is-step-completed="isStepCompleted"
					:highlight-step-detail="highlightStepDetail"
					@open-menu="openCookingMenu"
					@switch-tab="switchCookingTab"
					@flowchart-image-error="handleFlowchartImageError"
					@preview-flowchart="previewFlowchartImage"
					@open-flowchart-viewer="openFlowchartViewer"
					@generate-flowchart="handleGenerateFlowchart"
					@copy-ingredients="copyIngredientList"
					@toggle-step="toggleStepCompleted"
					@reset-steps="resetCompletedSteps"
					@parse="handleParseAction"
				/>

				<!-- P2 修复：公开只读模式下后端 DTO 已剔除 link/note，
				     前端再隐藏整块卡片，避免出现「来源链接 / 暂无链接」「备注 / 暂无备注」空卡，
				     体验从「没有内容」变为「该内容不公开」 -->
				<view v-if="!isPublicView" class="detail-card detail-card--quiet">
					<view class="detail-card__header">
						<view class="detail-card__heading">
							<text class="detail-card__title">来源链接</text>
						</view>
						<view v-if="recipe.link" class="detail-card__action" @tap="copyLink">
							<text class="detail-card__action-text">复制</text>
						</view>
						</view>
						<view v-if="recipe.link" class="link-panel">
							<view class="detail-link-box">
								<text class="detail-link-text" selectable>{{ displayRecipeLink }}</text>
							</view>
						</view>
					<text v-else class="detail-empty">暂无链接。</text>
				</view>

				<view v-if="!isPublicView" class="detail-card detail-card--note detail-card--quiet">
					<view class="detail-card__header detail-card__header--stack">
						<text class="detail-card__title">备注</text>
					</view>
					<text v-if="recipe.note" class="detail-note">{{ recipe.note }}</text>
					<text v-else class="detail-empty">暂无备注。</text>
				</view>
			</scroll-view>

			<view v-if="!isPublicView" class="detail-footer">
				<view class="detail-footer__action detail-footer__action--ghost detail-footer__action--delete" @tap="confirmDeleteRecipe">
					<up-icon name="trash" size="18" color="#b4664c"></up-icon>
				</view>
				<view
					class="detail-footer__action detail-footer__action--soft detail-footer__action--pin"
					:class="{
						'detail-footer__action--soft-active': isPinned,
						'detail-footer__action--disabled': isPinSubmitting
					}"
					@tap="togglePinned"
				>
					<text
						class="detail-footer__text"
						:class="{ 'detail-footer__text--accent': isPinned }"
					>{{ pinActionText }}</text>
				</view>
				<view class="detail-footer__action detail-footer__action--primary detail-footer__action--edit" @tap="openEditSheet">
					<text class="detail-footer__text detail-footer__text--primary">编辑</text>
				</view>
			</view>
		</template>

		<template v-else-if="showRecipeLoadingState">
			<view class="detail-loading">
				<view class="detail-loading__hero detail-loading__pulse"></view>
				<view class="detail-loading__section">
					<view class="detail-loading__chips">
						<view class="detail-loading__chip detail-loading__pulse"></view>
						<view class="detail-loading__chip detail-loading__chip--short detail-loading__pulse"></view>
					</view>
					<view class="detail-loading__title detail-loading__pulse"></view>
					<view class="detail-loading__line detail-loading__pulse"></view>
				</view>
				<view class="detail-loading__card">
					<view class="detail-loading__card-title detail-loading__pulse"></view>
					<view class="detail-loading__line detail-loading__pulse"></view>
					<view class="detail-loading__line detail-loading__line--short detail-loading__pulse"></view>
				</view>
				<view class="detail-loading__card">
					<view class="detail-loading__card-title detail-loading__pulse"></view>
					<view class="detail-loading__row detail-loading__pulse"></view>
					<view class="detail-loading__row detail-loading__pulse"></view>
					<view class="detail-loading__row detail-loading__row--short detail-loading__pulse"></view>
				</view>
			</view>
		</template>

		<template v-else>
			<view class="missing-state">
				<up-icon name="info-circle" size="42" color="#b8aa9b"></up-icon>
				<!-- P2-D：区分公开链接失效 vs 私有未找到，给出不同文案 -->
				<text class="missing-state__title">{{ isPublicView ? '分享链接已失效' : '没找到这道菜' }}</text>
				<text class="missing-state__desc">{{ isPublicView ? '这道菜谱可能已被删除或分享已收回。' : '可能已删除或未保存。' }}</text>
				<view class="missing-state__action" @tap="handleMissingStateBack">
					<text class="missing-state__action-text">{{ isPublicView ? '返回上一页' : '返回列表' }}</text>
				</view>
			</view>
		</template>

		<action-feedback
			:visible="actionFeedbackVisible"
			:feedback-key="actionFeedbackKey"
			:tone="actionFeedbackTone"
			:title="actionFeedbackTitle"
			:description="actionFeedbackDescription"
		></action-feedback>


		<RecipeEditSheet
			ref="recipeEditSheet"
			:show-edit-sheet="showEditSheet"
			:initial-draft="editInitialDraft"
			:max-recipe-images="maxRecipeImages"
			:meal-tabs="mealTabs"
			:status-tabs="statusTabs"
			:is-saving="isSavingRecipe"
			@close="closeEditSheet"
			@save="saveEditDraft"
		/>
	</view>
	<canvas canvas-id="_flowchart_square_canvas" style="position:fixed;left:-9999px;top:-9999px;width:300px;height:300px;"></canvas>
</template>

<script>
import ActionFeedback from '../../components/action-feedback.vue'
import PublicReadonlyBanner from './components/public-readonly-banner.vue'
import RecipeHero from './components/recipe-hero.vue'
import RecipeCookingPanel from './components/recipe-cooking-panel.vue'
import RecipeEditSheet from './components/recipe-edit-sheet.vue'
import {
	MAX_RECIPE_IMAGES,
	deleteRecipeById,
	fetchPublicRecipeByShareToken,
	generateRecipeFlowchartById,
	getCachedRecipeById,
	getRecipeById,
	isFallbackParsedContent as isFallbackLikeParsedContent,
	mealTypeLabelMap,
	mealTypeOptions,
	normalizeParsedContentView,
	reparseRecipeById,
	setRecipePinnedById,
	statusLabelMap,
	statusOptions,
	updateRecipeById
} from '../../utils/recipe-store'
import { buildImageCacheKey, getCachedImagePath, invalidateCachedImage, warmImageCache } from '../../utils/image-cache'
import {
	buildStepCompletedStorageKey,
	buildStepCompletionKeyList,
	cloneStepDraftList,
	createCompletedStepStoragePayload,
	createEmptyDraft,
	createIngredientDraftList,
	highlightStepDetailText,
	normalizeCompletedStepKeyMap
} from './use-recipe-edit'
import {
	ACTIVE_FLOWCHART_STATUSES,
	ACTIVE_PARSE_STATUSES,
	buildFlowchartWaitHint,
	buildParseResultHint,
	buildParseWaitHint,
	createRecipeAsyncJobsController,
	extractCopyableLink,
	flowchartStatusMetaMap,
	formatDateTime,
	formatParseSourceLabel,
	isAutoParseSupportedLink,
	parseStatusMetaMap,
	resolveRemainingWaitSeconds,
	toPositiveInteger
} from './use-recipe-async-jobs'
import {
	buildFlowchartImageCacheEntry as createFlowchartImageCacheEntry,
	buildRecipeImageVersion,
	createRecipeImageController
} from './use-recipe-images'
import {
	createRecipeShareController,
	FLOWCHART_VIEWER_STORAGE_KEY
} from './use-recipe-share'

export default {
	components: {
		ActionFeedback,
		PublicReadonlyBanner,
		RecipeHero,
		RecipeCookingPanel,
		RecipeEditSheet
	},
	data() {
		return {
			recipeId: '',
			recipe: null,
			showEditSheet: false,
			editInitialDraft: createEmptyDraft(),
			maxRecipeImages: MAX_RECIPE_IMAGES,
			mealTabs: mealTypeOptions,
			statusTabs: statusOptions,
			isLoadingRecipe: false,
			isUploadingHeroImage: false,
			isSavingRecipe: false,
			isDeletingRecipe: false,
			isReparseSubmitting: false,
			isGeneratingFlowchart: false,
			isPinSubmitting: false,
			heroImageIndex: 0,
			recipeAsyncJobsController: null,
			recipeImageController: null,
			recipeShareController: null,
			statusEstimateSyncedAt: 0,
			statusEstimateNow: 0,
			actionFeedbackVisible: false,
			actionFeedbackTone: '',
			actionFeedbackTitle: '',
			actionFeedbackDescription: '',
			actionFeedbackTick: 0,
			actionFeedbackTimer: null,
			hasResolvedInitialRecipeLoad: false,
			cachedRecipeImageMap: {},
			recipeImageFallbackMap: {},
			recipeImageHiddenMap: {},
			recipeImageCacheRequestID: 0,
			// P2-A：「做法」卡片当前激活的 Tab；'flowchart' | 'steps'
			// 默认值在 watch hasFlowchart / 初次加载时由 ensureCookingTabValid 校正
			activeCookingTab: 'flowchart',
			// B2-6：步骤完成状态，按「步骤内容签名」持久化，避免改顺序后串位
			completedStepKeyMap: {},
			// P2-D 分享路径升级：share_token 公开只读机制
			// shareToken：当前菜谱的永久 share_token（私有模式下后台 ensure，分享时拼到 path）
			// publicViewToken：从 onLoad query 读到的 shareToken，存在则强制走公开只读
			// isPublicView：是否处于公开只读模式（隐藏所有编辑入口、显示顶部 banner）
			// publicKitchenName / publicCreatorName：公开模式下 banner 显示的上下文
			// publicViewLoadFailed：公开拉取失败（token 失效 / 菜谱已删），显示兜底空态
			// showPublicReadOnlyExplain：banner 「了解」按钮控制的 popup 开关
			shareToken: '',
			// 进行中的 ensure share_token Promise（去重 + 供分享时 await 兜底）
			_shareTokenEnsurePromise: null,
			publicViewToken: '',
			isPublicView: false,
			publicKitchenName: '',
			publicCreatorName: '',
			publicViewLoadFailed: false,
			showPublicReadOnlyExplain: false,
			cachedFlowchartImagePath: '',
			flowchartImageCacheVersion: '',
			flowchartImageCacheRequestID: 0,
			flowchartSquareImagePath: '',
			flowchartSquareImageSourceKey: '',
			flowchartShareImagePendingKey: '',
			_flowchartShareImagePromise: null
		}
	},
	computed: {
		mealLabel() {
			return mealTypeLabelMap[this.recipe?.mealType] || '早餐'
		},
		// P2-D 分享路径升级：公开只读 banner 文案
		// 优先「来自 XX 的空间」；空间名缺失时退化为「来自他人的菜谱分享」
		publicBannerText() {
			const kitchen = String(this.publicKitchenName || '').trim()
			if (kitchen) return `来自「${kitchen}」的菜谱 · 加入空间可参与编辑`
			return '来自他人的菜谱分享 · 加入空间可参与编辑'
		},
		// 只读规则 popup 正文，按是否有创建者昵称差异化
		publicExplainBody() {
			const creator = String(this.publicCreatorName || '').trim()
			const owner = creator ? `「${creator}」` : '原作者'
			return `这道菜由${owner}整理，分享出来仅供查看。如果想一起编辑、调整步骤或补充心得，可以请对方把你加入空间。`
		},
		statusLabel() {
			return statusLabelMap[this.recipe?.status] || '想吃'
		},
		isPinned() {
			return !!String(this.recipe?.pinnedAt || '').trim()
		},
		parsedContentView() {
			return normalizeParsedContentView(this.recipe?.parsedContent || {})
		},
		parsedMainIngredients() {
			return this.parsedContentView.mainIngredients
		},
		parsedSecondaryIngredients() {
			return this.parsedContentView.secondaryIngredients
		},
		parsedSecondaryGroups() {
			const groups = []
			const supportingIngredients = this.parsedContentView.supportingIngredients || []
			const seasonings = this.parsedContentView.seasonings || []

			if (supportingIngredients.length) {
				groups.push({
					key: 'supporting',
					label: '配菜',
					text: supportingIngredients.join('、')
				})
			}

			if (seasonings.length) {
				groups.push({
					key: 'seasonings',
					label: '调味',
					text: seasonings.join('、')
				})
			}

			return groups
		},
		parsedSteps() {
			return this.parsedContentView.steps
		},
		hasMeaningfulParsedContent() {
			return !isFallbackLikeParsedContent(this.recipe || {}, {
				mainIngredients: this.parsedMainIngredients,
				secondaryIngredients: this.parsedSecondaryIngredients,
				steps: this.parsedSteps
			})
		},
		hasManualParsedContentEdits() {
			return !!this.recipe?.parsedContentEdited
		},
		recipeImageVersion() {
			return buildRecipeImageVersion(this.recipe || {})
		},
		recipeImages() {
			if (Array.isArray(this.recipe?.imageUrls) && this.recipe.imageUrls.length) {
				return this.recipe.imageUrls.filter(Boolean)
			}
			const fallbackImage = String(this.recipe?.image || this.recipe?.imageUrl || '').trim()
			return fallbackImage ? [fallbackImage] : []
		},
		displayRecipeLink() {
			const rawLink = String(this.recipe?.link || '').trim()
			return extractCopyableLink(rawLink) || rawLink
		},
		displayRecipeImages() {
			const version = this.recipeImageVersion
			return this.recipeImages
				.map((remoteURL) => {
					const cacheKey = buildImageCacheKey(remoteURL, version)
					if (this.recipeImageHiddenMap[cacheKey]) return null

					const cachedURL = String(this.cachedRecipeImageMap[cacheKey] || '').trim()
					return {
						cacheKey,
						remoteURL,
						displayURL: this.recipeImageFallbackMap[cacheKey] ? remoteURL : (cachedURL || remoteURL)
					}
				})
				.filter(Boolean)
		},
		visibleRecipeSourceImages() {
			const version = this.recipeImageVersion
			return this.recipeImages.filter((remoteURL) => {
				const cacheKey = buildImageCacheKey(remoteURL, version)
				return !this.recipeImageHiddenMap[cacheKey]
			})
		},
		flowchartImageUrl() {
			return String(this.recipe?.flowchartImageUrl || '').trim()
		},
		flowchartDisplayImageUrl() {
			return String(this.cachedFlowchartImagePath || '').trim() || this.flowchartImageUrl
		},
		flowchartStatusValue() {
			return String(this.recipe?.flowchartStatus || '').trim()
		},
		hasFlowchart() {
			return !!this.flowchartImageUrl
		},
		canGenerateFlowchart() {
			return this.hasMeaningfulParsedContent && this.parsedSteps.length >= 3
		},
		canRequestFlowchart() {
			return this.canGenerateFlowchart && !ACTIVE_FLOWCHART_STATUSES.includes(this.flowchartStatusValue)
		},
		isFlowchartActive() {
			// 后台正在生成 / 排队中
			return ACTIVE_FLOWCHART_STATUSES.includes(this.flowchartStatusValue)
		},
		flowchartActionText() {
			if (this.isGeneratingFlowchart) return '提交中...'
			if (ACTIVE_FLOWCHART_STATUSES.includes(this.flowchartStatusValue)) return '生成中...'
			return this.hasFlowchart ? '重新生成' : '生成步骤图'
		},
		flowchartEmptyText() {
			if (this.canGenerateFlowchart) {
				return '生成后会把关键步骤整理成一张图，先看懂再下厨。'
			}
			return '先补充至少 3 个关键步骤，再生成步骤图。'
		},
		flowchartStatusMeta() {
			const status = this.flowchartStatusValue
			if (!status || status === 'done') return null
			return flowchartStatusMetaMap[status] || null
		},
		flowchartStatusDescription() {
			if (!this.flowchartStatusMeta) return ''
			const errorMessage = String(this.recipe?.flowchartError || '').trim()
			if (this.flowchartStatusValue === 'failed' && errorMessage) {
				return errorMessage
			}
			const waitHint = buildFlowchartWaitHint(this.flowchartStatusValue, this.flowchartQueueAhead, this.flowchartEstimatedWaitSeconds)
			if (waitHint) {
				return waitHint
			}
			return this.flowchartStatusMeta.description
		},
		flowchartQueueAhead() {
			return toPositiveInteger(this.recipe?.flowchartQueueAhead || 0)
		},
		flowchartEstimatedWaitSeconds() {
			return resolveRemainingWaitSeconds(
				this.recipe?.flowchartEstimatedWaitSeconds || 0,
				this.statusEstimateSyncedAt,
				this.statusEstimateNow
			)
		},
		showFlowchartStaleHint() {
			return this.hasFlowchart && !!this.recipe?.flowchartStale
		},
		flowchartUpdatedAtText() {
			const value = formatDateTime(this.recipe?.flowchartUpdatedAt || '')
			return value ? `已生成：${value}` : ''
		},
		flowchartCaptionText() {
			const raw = String(this.recipe?.flowchartUpdatedAt || '').trim()
			if (!raw) return ''
			// 仅取月-日，作为卡片底部一行 caption（完整时间放到「查看生成详情」里）
			const date = new Date(raw)
			if (Number.isNaN(date.getTime())) {
				return 'AI 生成'
			}
			const mm = String(date.getMonth() + 1).padStart(2, '0')
			const dd = String(date.getDate()).padStart(2, '0')
			return `AI 生成 · ${mm}-${dd}`
		},
		flowchartModelTip() {
			const model = String(this.recipe?.flowchartModel || '').trim()
			return model ? `由 ${model} 生成` : ''
		},
		parseStatusValue() {
			return String(this.recipe?.parseStatus || '').trim()
		},
		parseStatusMeta() {
			const status = this.parseStatusValue
			if (status && parseStatusMetaMap[status]) {
				return parseStatusMetaMap[status]
			}
			if (this.isAutoParseRecipe) {
				return parseStatusMetaMap.idle
			}
			return null
		},
		parseStatusDescription() {
			if (!this.parseStatusMeta) return ''
			const errorMessage = String(this.recipe?.parseError || '').trim()
			if (this.parseStatusValue === 'failed' && errorMessage) {
				return errorMessage
			}
			if (this.parseStatusValue === 'done' && errorMessage && String(this.recipe?.parseSource || '').toLowerCase().includes('heuristic')) {
				return errorMessage
			}
			const waitHint = buildParseWaitHint(this.parseStatusValue, this.parseQueueAhead, this.parseEstimatedWaitSeconds)
			if (waitHint) {
				return waitHint
			}
			const resultHint = buildParseResultHint(this.parseStatusValue, this.recipe?.parseSource || '')
			if (resultHint) {
				return resultHint
			}
			return this.parseStatusMeta.description
		},
		parseQueueAhead() {
			return toPositiveInteger(this.recipe?.parseQueueAhead || 0)
		},
		parseEstimatedWaitSeconds() {
			return resolveRemainingWaitSeconds(
				this.recipe?.parseEstimatedWaitSeconds || 0,
				this.statusEstimateSyncedAt,
				this.statusEstimateNow
			)
		},
		parseStatusSourceLabel() {
			return formatParseSourceLabel(this.recipe?.parseSource || '')
		},
		isAutoParseRecipe() {
			return isAutoParseSupportedLink(this.recipe?.link || '')
		},
		canRequestParse() {
			return this.isAutoParseRecipe && !ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)
		},
		needsParseOverwriteConfirm() {
			// 仅在「手动改过」时才走覆盖警告；纯 AI 结果走下方的「轻确认」分支
			return this.hasManualParsedContentEdits
		},
		parseOverwriteModalContent() {
			if (this.hasManualParsedContentEdits) {
				return '你手动修改过食材或制作步骤，重新整理后可能会覆盖这些内容。'
			}
			return '将根据来源链接更新当前食材和步骤。'
		},
		parseActionText() {
			if (this.isReparseSubmitting) return '整理中...'
			if (!this.parseStatusValue) return '开始整理'
			if (this.parseStatusValue === 'failed') return '再试一次'
			return '重新整理'
		},
		pinActionText() {
			if (this.isPinSubmitting) return '处理中...'
			return this.isPinned ? '取消置顶' : '置顶'
		},
		showRecipeLoadingState() {
			return !this.recipe && (!this.hasResolvedInitialRecipeLoad || this.isLoadingRecipe)
		},
		actionFeedbackKey() {
			return `${this.actionFeedbackTone || 'idle'}:${this.actionFeedbackTick}`
		},
		// ===== P2-A：「做法」合并卡片相关 =====
		// 是否处于后台异步进行中（一图生成 or 步骤整理），用于把右上 ⋯ 替换为 chip
		isCookingActive() {
			return this.isFlowchartActive || ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)
		},
		// chip 文案，根据当前激活的 Tab 选择对应任务的标签
		cookingActiveLabel() {
			if (this.activeCookingTab === 'flowchart' && this.isFlowchartActive) {
				return '生成中…'
			}
			if (this.activeCookingTab === 'steps' && ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)) {
				return '整理中…'
			}
			// 跨 Tab 的兜底：另一 Tab 在跑也提示用户
			if (this.isFlowchartActive) return '一图生成中…'
			if (ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)) return '步骤整理中…'
			return ''
		},
		// 统一 ⋯ 菜单是否至少有一个可执行项；为空则隐藏入口
		hasCookingMenuItems() {
			if (this.canRequestFlowchart && !this.isFlowchartActive) return true
			if (this.canRequestParse) return true
			if (this.flowchartUpdatedAtText || this.flowchartModelTip) return true
			if (this.parseStatusSourceLabel) return true
			return false
		},
		showCookingFlowchartView() {
			return this.hasFlowchart && this.activeCookingTab === 'flowchart'
		},
		showCookingStepsView() {
			return !this.hasFlowchart || this.activeCookingTab === 'steps'
		},
		// 卡片底部一行 caption：根据当前 Tab 选择对应来源
		cookingFooterText() {
			if (this.showCookingFlowchartView) {
				return this.flowchartCaptionText || ''
			}
			if (this.showCookingStepsView) {
				return this.parseStatusSourceLabel || ''
			}
			return ''
		},
		// B1-2：是否可复制食材清单（至少要有 1 项主料或 1 个辅料分组）
		canCopyIngredientList() {
			return this.parsedMainIngredients.length > 0 || this.parsedSecondaryGroups.length > 0
		},
		// B2-6：已完成步骤数（用于「2 / 4」进度提示）
		completedStepCount() {
			const stepKeys = this.buildCurrentStepCompletionKeys()
			if (!stepKeys.length) return 0
			let count = 0
			for (let i = 0; i < stepKeys.length; i += 1) {
				if (this.completedStepKeyMap[stepKeys[i]]) count += 1
			}
			return count
		},
		// ===== Hero 操作菜单：当前位置图片是否可设为封面 =====
		canSetCurrentAsCover() {
			return this.recipeImages.length > 1 && this.resolveOriginalImageIndex(this.heroImageIndex) > 0
		},
		// 是否可以再添加图片（未到上限）
		canAddMoreHeroImages() {
			return this.recipeImages.length > 0 && this.visibleRecipeSourceImages.length < this.maxRecipeImages
		},
		// 是否可以删除当前图片（至少要有 1 张存在）
		canDeleteCurrentImage() {
			return this.resolveOriginalImageIndex(this.heroImageIndex) >= 0
		},
		// 是否显示 Hero ⋯ 按钮：上传中隐藏；菜单全空（理论上 length > 0 至少有「删除」）也隐藏
		canShowHeroActionMenu() {
			if (this.isUploadingHeroImage) return false
			if (!this.displayRecipeImages.length) {
				return this.canAddMoreHeroImages
			}
			return this.canSetCurrentAsCover || this.canAddMoreHeroImages || this.canDeleteCurrentImage
		}
	},
	watch: {
		// P2-A：当流程图从「有」变「无」（如生成失败被清空）且当前正停留在「一图」Tab，
		// 自动回退到「详细步骤」Tab，避免显示空内容
		hasFlowchart() {
			this.ensureCookingTabValid()
		},
		// P2-A 修复：流程图任务终止（active -> idle/failed）后也要回退一次
		isFlowchartActive() {
			this.ensureCookingTabValid()
		}
	},
	onLoad(options) {
		this.recipeId = options?.id || ''
		// P2-D 分享路径升级：query 带 shareToken 即视为公开只读访问
		// 不论是否同空间成员，统一走公开只读，避免拉起登录、避免编辑误触
		const shareToken = String(options?.shareToken || '').trim()
		if (shareToken) {
			this.publicViewToken = shareToken
			this.isPublicView = true
			// P1 修复：把入参 shareToken 同步写到 this.shareToken
			// 否则公开模式下二次转发时 buildRecipeShareConfig 拿不到 token，发出去的链接会断在第二跳
			this.shareToken = shareToken
		}
	},
	onShow() {
		this.loadRecipe()
	},
	onHide() {
		this.stopParsePolling()
		this.clearActionFeedback()
	},
	onUnload() {
		this.stopParsePolling()
		this.clearActionFeedback()
	},
	onBackPress() {
		if (!this.showEditSheet) return false
		this.$refs.recipeEditSheet?.requestCloseEditSheet?.()
		return true
	},
	// P2-D 分享路径升级：开启微信原生右上角胶囊菜单的「转发 / 分享到朋友圈 / 收藏」三项能力
	// 只要定义这三个生命周期函数，对应菜单项就会出现，无需 UI 改动
	// P2 修复：分享窗口期兜底
	//   - 若 token 已就绪：同步返回完整 config（含 shareToken）
	//   - 若 token 还在 ensure 中：通过 promise 字段（基础库 2.12.0+）等 token 到位再返回
	//   - 微信侧 promise 超时（约 5s）会回退使用同步返回的兜底 config（不带 token，行为退化为旧版鉴权墙）
	onShareAppMessage(res) {
		const fallback = this.buildRecipeShareConfig({ from: res?.from, channel: 'message' })
		const needsShareToken = !this.shareToken && !this.isPublicView && !!this.recipeId
		const needsFlowchartShareCover =
			this.shouldPreferFlowchartShareCover('message') && this.hasFlowchart && !this.getCurrentFlowchartShareImagePath()
		if (!needsShareToken && !needsFlowchartShareCover) return fallback
		return {
			...fallback,
			promise: this.buildRecipeShareConfigAsync({ from: res?.from, channel: 'message' })
				.catch(() => fallback)
		}
	},
	onShareTimeline() {
		const fallback = this.buildRecipeShareConfig({ channel: 'timeline' })
		if (this.shareToken || this.isPublicView || !this.recipeId) return fallback
		// 朋友圈 promise 字段需基础库 3.12.0+，老版本会忽略 promise，自动回退到同步配置
		return {
			...fallback,
			promise: this.ensureShareTokenIfNeeded()
				.then(() => this.buildRecipeShareConfig({ channel: 'timeline' }))
				.catch(() => fallback)
		}
	},
	onAddToFavorites() {
		// 收藏夹接口不支持 promise 字段，只能同步返回当前最佳配置。
		// 若这次优先走流程图封面，则在真正触发分享时后台懒裁一份供后续复用。
		if (this.shouldPreferFlowchartShareCover('favorite') && this.hasFlowchart && !this.getCurrentFlowchartShareImagePath()) {
			this.ensureFlowchartShareImagePath().catch(() => {})
		}
		return this.buildRecipeShareConfig({ channel: 'favorite' })
	},
	methods: {
		clearActionFeedbackTimer() {
			if (!this.actionFeedbackTimer) return
			clearTimeout(this.actionFeedbackTimer)
			this.actionFeedbackTimer = null
		},
		clearActionFeedback() {
			this.clearActionFeedbackTimer()
			this.actionFeedbackVisible = false
			this.actionFeedbackTone = ''
			this.actionFeedbackTitle = ''
			this.actionFeedbackDescription = ''
		},
		showActionFeedback(options = {}) {
			const title = String(options?.title || '').trim()
			if (!title) return
			this.clearActionFeedbackTimer()
			this.actionFeedbackTone = String(options?.tone || 'done').trim() || 'done'
			this.actionFeedbackTitle = title
			this.actionFeedbackDescription = String(options?.description || '').trim()
			this.actionFeedbackVisible = true
			this.actionFeedbackTick += 1
			this.actionFeedbackTimer = setTimeout(() => {
				this.actionFeedbackVisible = false
				this.actionFeedbackTimer = null
			}, Math.max(1200, Number(options?.duration) || 1680))
		},
		async loadRecipe() {
			// P2-D 分享路径升级：公开只读模式优先走 share_token 公开接口
			// 不进缓存（避免污染同 id 的私有缓存）、不触发 ensureSession
			if (this.isPublicView && this.publicViewToken) {
				try {
					this.isLoadingRecipe = true
					const view = await fetchPublicRecipeByShareToken(this.publicViewToken)
					if (!view || !view.recipe) {
						this.recipe = null
						this.publicViewLoadFailed = true
						this.hasResolvedInitialRecipeLoad = true
						return
					}
					this.recipeId = view.recipe.id || this.recipeId
					this.publicKitchenName = view.kitchenName || ''
					this.publicCreatorName = view.creatorName || ''
					this.publicViewLoadFailed = false
					this.applyRecipe(view.recipe)
					this.hasResolvedInitialRecipeLoad = true
				} catch (error) {
					this.recipe = null
					this.publicViewLoadFailed = true
					this.hasResolvedInitialRecipeLoad = true
				} finally {
					this.isLoadingRecipe = false
				}
				return
			}

			if (!this.recipeId) {
				this.recipe = null
				this.hasResolvedInitialRecipeLoad = true
				return
			}

			// P2 修复：进页就立刻 fire ensure share_token（与缓存读取并行）
			// 缩短「打开详情秒分享」窗口，applyRecipe 末尾的 ensure 作为兜底
			this.ensureShareTokenIfNeeded()

			const cachedRecipe = getCachedRecipeById(this.recipeId)
			if (cachedRecipe) {
				this.applyRecipe(cachedRecipe)
				this.hasResolvedInitialRecipeLoad = true
			}

			try {
				this.isLoadingRecipe = true
				const recipe = await getRecipeById(this.recipeId, { preferCache: !cachedRecipe })
				this.applyRecipe(recipe)
				this.hasResolvedInitialRecipeLoad = true
			} catch (error) {
				if (!cachedRecipe) {
					this.recipe = null
					uni.showToast({
						title: error?.message || '加载失败',
						icon: 'none'
					})
				}
			} finally {
				this.isLoadingRecipe = false
				this.hasResolvedInitialRecipeLoad = true
			}
		},
		buildFlowchartImageCacheVersion(recipe = this.recipe) {
			return createFlowchartImageCacheEntry(recipe, buildImageCacheKey).version
		},
		buildFlowchartImageCacheEntry(recipe = this.recipe) {
			return createFlowchartImageCacheEntry(recipe, buildImageCacheKey)
		},
		buildFlowchartPreviewURLs() {
			const urls = []
			const appendURL = (value = '') => {
				const target = String(value || '').trim()
				if (!target || urls.includes(target)) return
				urls.push(target)
			}
			appendURL(this.cachedFlowchartImagePath)
			appendURL(this.flowchartImageUrl)
			return urls
		},
		shouldPreferFlowchartShareCover(channel = 'message') {
			return channel === 'message' || channel === 'favorite'
		},
		getCurrentFlowchartShareImagePath() {
			const currentKey = this.buildFlowchartImageCacheEntry().cacheKey
			if (!currentKey || this.flowchartSquareImageSourceKey !== currentKey) return ''
			return String(this.flowchartSquareImagePath || '').trim()
		},
		getRecipeShareController() {
			if (!this.recipeShareController) this.recipeShareController = createRecipeShareController(this)
			return this.recipeShareController
		},
		async buildRecipeShareConfigAsync({ channel = 'message' } = {}) {
			return this.getRecipeShareController().configAsync({ channel })
		},
		async ensureFlowchartShareImagePath() {
			return this.getRecipeShareController().ensureFlowchartImage()
		},
		async createFlowchartSquareImage() {
			return this.getRecipeShareController().createSquareImage()
		},
		buildRecipeShareConfig({ channel = 'message', flowchartShareImage = '' } = {}) {
			return this.getRecipeShareController().config({ channel, flowchartShareImage })
		},
		applyRecipe(recipe) {
			const previousFlowchartCacheKey = this.buildFlowchartImageCacheEntry(this.recipe).cacheKey
			const nextFlowchartCacheKey = this.buildFlowchartImageCacheEntry(recipe).cacheKey
			if (previousFlowchartCacheKey !== nextFlowchartCacheKey) {
				this.flowchartSquareImagePath = ''
				this.flowchartSquareImageSourceKey = ''
				this.flowchartShareImagePendingKey = ''
			}
			this.recipe = recipe
			const now = Date.now()
			this.statusEstimateSyncedAt = now
			this.statusEstimateNow = now
			this.syncRecipeImageCache(recipe)
			this.syncFlowchartImageCache(recipe, {
				previousCacheKey: previousFlowchartCacheKey
			})
			// B2-6：每次加载菜谱时同步读取本地步骤完成进度
			this.loadCompletedSteps()
			if (this.heroImageIndex >= this.displayRecipeImages.length) {
				this.heroImageIndex = 0
			}
			// P2-A 修复：菜谱数据落定后再校正一次「做法」Tab，避免无图时初次加载落到空态
			this.ensureCookingTabValid()
			if (this.recipe?.title) {
				uni.setNavigationBarTitle({
					title: this.recipe.title
				})
			}
			this.syncParsePolling()
			// P2-D 分享路径升级：私有模式下后台静默 ensure share_token
			// 不阻塞渲染、失败静默；分享时直接用 this.shareToken 拼 path
			this.ensureShareTokenIfNeeded()
		},
		// 后台幂等获取菜谱永久 share_token，仅私有模式且尚未拿到时触发
		// 失败不打扰用户，分享路径会兜底为不带 token 的链接（行为退化为旧版）
		// P2 修复：返回 Promise<string|null>，供 onShareAppMessage 在 token 未就绪时 await 兜底
		ensureShareTokenIfNeeded() {
			return this.getRecipeShareController().ensureToken()
		},
		ensureCookingTabValid() {
			if (this.activeCookingTab !== 'flowchart' && this.activeCookingTab !== 'steps') {
				this.activeCookingTab = this.hasFlowchart ? 'flowchart' : 'steps'
				return
			}
			if (!this.hasFlowchart && this.activeCookingTab !== 'steps') {
				this.activeCookingTab = 'steps'
			}
		},
		async syncFlowchartImageCache(recipe = this.recipe, options = {}) {
			const entry = this.buildFlowchartImageCacheEntry(recipe)
			const requestID = this.flowchartImageCacheRequestID + 1
			const previousCacheKey = String(options.previousCacheKey || '').trim()

			this.flowchartImageCacheRequestID = requestID
			this.flowchartImageCacheVersion = entry.version

			if (entry.cacheKey !== previousCacheKey) {
				this.cachedFlowchartImagePath = ''
			}

			if (!entry.url) {
				this.cachedFlowchartImagePath = ''
				this.flowchartImageCacheVersion = ''
				return
			}

			const localPath = await getCachedImagePath(entry.url, entry.version)
			if (requestID !== this.flowchartImageCacheRequestID) return

			if (localPath) {
				this.cachedFlowchartImagePath = localPath
				return
			}

			this.cachedFlowchartImagePath = ''
			warmImageCache([entry], {
				concurrency: 1,
				onResolved: ({ localPath: resolvedPath }) => {
					if (requestID !== this.flowchartImageCacheRequestID || !resolvedPath) return
					if (this.buildFlowchartImageCacheEntry().cacheKey !== entry.cacheKey) return
					this.cachedFlowchartImagePath = resolvedPath
				}
			})
		},
		getRecipeImageController() {
			if (!this.recipeImageController) this.recipeImageController = createRecipeImageController(this)
			return this.recipeImageController
		},
		async syncRecipeImageCache(recipe = this.recipe) {
			return this.getRecipeImageController().syncRecipeCache(recipe)
		},
		syncParsePolling() {
			const parseStatus = String(this.recipe?.parseStatus || '').trim()
			const flowchartStatus = String(this.recipe?.flowchartStatus || '').trim()
			if (!this.recipeAsyncJobsController) {
				this.recipeAsyncJobsController = createRecipeAsyncJobsController({
					poll: () => this.refreshParseStatus(),
					onEstimateTick: () => { this.statusEstimateNow = Date.now() }
				})
			}
			this.recipeAsyncJobsController.sync({
				hasActiveJob:
					ACTIVE_PARSE_STATUSES.includes(parseStatus) ||
					ACTIVE_FLOWCHART_STATUSES.includes(flowchartStatus),
				hasEstimate: !!(this.parseEstimatedWaitSeconds || this.flowchartEstimatedWaitSeconds)
			})
		},
		stopParsePolling() {
			this.recipeAsyncJobsController?.stop()
		},
		async refreshParseStatus() {
			if (!this.recipeId || this.isLoadingRecipe || this.isSavingRecipe || this.isDeletingRecipe || this.isReparseSubmitting || this.isGeneratingFlowchart || this.isPinSubmitting) {
				return
			}

			try {
				const recipe = await getRecipeById(this.recipeId, { preferCache: false })
				this.applyRecipe(recipe)
			} catch (error) {
				// Ignore transient polling errors and keep the last known state on screen.
			}
		},
		createDraftFromRecipe(recipe = {}) {
			const parsedContentView = normalizeParsedContentView(recipe.parsedContent || {})
			const hasStructuredContent = !isFallbackLikeParsedContent(recipe, recipe.parsedContent || {})
			return createEmptyDraft({
				title: recipe.title || '',
				ingredient: recipe.ingredient || '',
				link: recipe.link || '',
				images:
					Array.isArray(recipe.imageUrls) && recipe.imageUrls.length
						? [...recipe.imageUrls]
						: recipe.image
							? [recipe.image]
							: [],
				mealType: recipe.mealType || 'breakfast',
				status: recipe.status || 'wishlist',
				mainIngredients: hasStructuredContent ? createIngredientDraftList(parsedContentView.mainIngredients) : [],
				secondaryIngredients: hasStructuredContent ? createIngredientDraftList(parsedContentView.secondaryIngredients) : [],
				steps: hasStructuredContent ? cloneStepDraftList(parsedContentView.steps) : [],
				parsedContentMode: hasStructuredContent ? 'existing' : 'fallback',
				note: recipe.note || ''
			})
		},
		openEditSheet() {
			// P1 修复：公开只读模式禁止打开编辑面板
			if (this.isPublicView) return
			if (!this.recipe) return
			this.editInitialDraft = this.createDraftFromRecipe(this.recipe)
			this.showEditSheet = true
		},
		handleHeroCardTap() {
			if (!this.recipe) return
			if (this.displayRecipeImages.length) {
				this.previewRecipeImage()
				return
			}
			this.chooseHeroImages()
		},
		handleHeroSwiperChange(event) {
			this.heroImageIndex = Number(event?.detail?.current) || 0
		},
		async handleRecipeImageError(image = {}) {
			return this.getRecipeImageController().handleImageError(image)
		},
		closeEditSheet() {
			this.showEditSheet = false
			this.editInitialDraft = createEmptyDraft()
		},
		chooseHeroImages() {
			return this.getRecipeImageController().chooseHeroImages()
		},
		async saveHeroImages(imagePaths = []) {
			return this.getRecipeImageController().saveHeroImages(imagePaths)
		},
		handleParseAction() {
			// P1 修复：公开只读模式禁止触发 AI 整理（消耗额度的写入口）
			if (this.isPublicView) return
			if (!this.canRequestParse || this.isReparseSubmitting) return
			if (this.needsParseOverwriteConfirm) {
				uni.showModal({
					title: '更新做法整理',
					content: this.parseOverwriteModalContent,
					confirmText: '继续整理',
					confirmColor: '#b4664c',
					success: ({ confirm }) => {
						if (!confirm) return
						this.requestAutoParse()
					}
				})
				return
			}

			// 已有整理结果时，重新整理也会消耗一次 AI 额度，做轻确认
			if (this.hasMeaningfulParsedContent) {
				uni.showModal({
					title: '重新整理？',
					content: '将再次调用 AI 整理食材与步骤，消耗 1 次额度。',
					confirmText: '继续整理',
					confirmColor: '#b4664c',
					success: ({ confirm }) => {
						if (!confirm) return
						this.requestAutoParse()
					}
				})
				return
			}

			this.requestAutoParse()
		},
		async requestAutoParse() {
			if (!this.canRequestParse || this.isReparseSubmitting) return

			this.isReparseSubmitting = true
			uni.showLoading({
				title: '整理中',
				mask: true
			})

			try {
				const recipe = await reparseRecipeById(this.recipeId)
				this.applyRecipe(recipe)
				this.showActionFeedback({
					tone: 'pending',
					title: '已加入整理队列',
					description:
						buildParseWaitHint(recipe?.parseStatus, recipe?.parseQueueAhead, recipe?.parseEstimatedWaitSeconds) ||
						parseStatusMetaMap.pending.description,
					duration: 1800
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '发起整理失败',
					icon: 'none'
				})
			} finally {
				this.isReparseSubmitting = false
				uni.hideLoading()
			}
		},
		// ===== P2-A：合并卡片 Tab 切换 + 统一 ⋯ 菜单 =====
		// 备注：原 openFlowchartMenu / openParseMenu 已被 openCookingMenu 取代并移除
		switchCookingTab(tab) {
			if (tab !== 'flowchart' && tab !== 'steps') return
			if (this.activeCookingTab === tab) return
			// 「一图看懂」Tab 仅在有图时可用；无图时静默忽略以防误触
			if (tab === 'flowchart' && !this.hasFlowchart) return
			this.activeCookingTab = tab
			// 轻触觉反馈，与底部「横屏查看」胶囊一致
			if (typeof uni !== 'undefined' && typeof uni.vibrateShort === 'function') {
				uni.vibrateShort({ type: 'light' })
			}
		},
		openCookingMenu() {
			// P1 修复：公开只读模式禁止打开做法菜单（含重新生成 / 重整理 / 查看详情写入口）
			if (this.isPublicView) return
			// 任一异步任务进行中时，⋯ 已被替换为 chip，此处只是双保险
			if (this.isCookingActive) return
			if (this.isGeneratingFlowchart || this.isReparseSubmitting) return

			const items = []
			const actions = []

			// 1) 重新生成一图（仅可执行时暴露）
			if (this.canRequestFlowchart && !this.isFlowchartActive) {
				items.push(this.hasFlowchart ? '重新生成一图看懂' : '生成一图看懂')
				actions.push('regen-flowchart')
			}

			// 2) 重新整理步骤
			if (this.canRequestParse) {
				items.push('重新整理步骤')
				actions.push('reparse')
			}

			// 3) 查看详情（合并 flowchart + parse 来源信息）
			const detailLines = [
				this.flowchartUpdatedAtText,
				this.flowchartModelTip,
				this.parseStatusSourceLabel
			].filter(Boolean)
			if (detailLines.length) {
				items.push('查看生成详情')
				actions.push('detail')
			}

			if (!items.length) {
				uni.showToast({ title: '当前无可执行操作', icon: 'none' })
				return
			}

			uni.showActionSheet({
				itemList: items,
				success: ({ tapIndex }) => {
					const action = actions[tapIndex]
					if (action === 'regen-flowchart') {
						this.handleGenerateFlowchart()
					} else if (action === 'reparse') {
						this.handleParseAction()
					} else if (action === 'detail') {
						uni.showModal({
							title: '生成详情',
							content: detailLines.join('\n'),
							showCancel: false,
							confirmText: '知道了',
							confirmColor: '#5b4a3b'
						})
					}
				}
			})
		},
		// ===== B1-2：复制食材清单到剪贴板 =====
		copyIngredientList() {
			if (!this.canCopyIngredientList) return

			const lines = []
			const title = String(this.recipe?.title || '').trim()
			if (title) {
				lines.push(`${title} · 食材清单`)
				lines.push('')
			}

			if (this.parsedMainIngredients.length) {
				lines.push(`主料：${this.parsedMainIngredients.join('、')}`)
			}

			this.parsedSecondaryGroups.forEach((group) => {
				if (group?.text) {
					lines.push(`${group.label}：${group.text}`)
				}
			})

			const text = lines.join('\n').trim()
			if (!text) {
				uni.showToast({ title: '清单为空', icon: 'none' })
				return
			}

			uni.setClipboardData({
				data: text,
				success: () => {
					if (typeof uni.vibrateShort === 'function') {
						uni.vibrateShort({ type: 'light' })
					}
					// uni.setClipboardData 默认会弹「已复制」toast，这里再补一句更具体的提示
					uni.showToast({
						title: '已复制，可去备忘录粘贴',
						icon: 'none',
						duration: 1600
					})
				},
				fail: () => {
					uni.showToast({ title: '复制失败，请重试', icon: 'none' })
				}
			})
		},
		// ===== B2-5：步骤详情高亮切片代理（透传到外部纯函数）=====
		highlightStepDetail(detail) {
			return highlightStepDetailText(detail)
		},
		buildCurrentStepCompletionKeys() {
			return buildStepCompletionKeyList(this.parsedSteps)
		},
		getStepCompletionKey(index) {
			const stepKeys = this.buildCurrentStepCompletionKeys()
			return stepKeys[index] || ''
		},
		// ===== B2-6：步骤完成状态管理 =====
		isStepCompleted(index) {
			const stepKey = this.getStepCompletionKey(index)
			return !!(stepKey && this.completedStepKeyMap[stepKey])
		},
		toggleStepCompleted(index) {
			if (typeof index !== 'number' || index < 0) return
			const stepKey = this.getStepCompletionKey(index)
			if (!stepKey) return
			// 触发 Vue 响应式更新：使用 $set 或对象重建（这里直接重建以兼容 Vue 3 / 2.x）
			const next = { ...this.completedStepKeyMap }
			if (next[stepKey]) {
				delete next[stepKey]
			} else {
				next[stepKey] = true
			}
			this.completedStepKeyMap = next
			this.persistCompletedSteps()
			if (typeof uni.vibrateShort === 'function') {
				uni.vibrateShort({ type: 'light' })
			}
		},
		resetCompletedSteps() {
			if (!this.completedStepCount) return
			uni.showModal({
				title: '重置完成进度？',
				content: '将清除当前菜谱所有步骤的「已完成」标记。',
				confirmText: '重置',
				confirmColor: '#b4664c',
				success: ({ confirm }) => {
					if (!confirm) return
					this.completedStepKeyMap = {}
					this.persistCompletedSteps()
				}
			})
		},
		loadCompletedSteps() {
			const key = buildStepCompletedStorageKey(this.recipeId)
			if (!key) {
				this.completedStepKeyMap = {}
				return
			}
			const currentStepKeys = this.buildCurrentStepCompletionKeys()
			try {
				const raw = uni.getStorageSync(key)
				this.completedStepKeyMap = normalizeCompletedStepKeyMap(raw, currentStepKeys)
			} catch (error) {
				// 存储读失败不致命，回退到空状态
				this.completedStepKeyMap = {}
			}
		},
		persistCompletedSteps() {
			const key = buildStepCompletedStorageKey(this.recipeId)
			if (!key) return
			try {
				if (Object.keys(this.completedStepKeyMap).length === 0) {
					uni.removeStorageSync(key)
				} else {
					uni.setStorageSync(key, createCompletedStepStoragePayload(this.completedStepKeyMap))
				}
			} catch (error) {
				// 存储写失败不致命，仅记录到 console（生产环境无影响）
				// eslint-disable-next-line no-console
				console.warn('[recipe-detail] persistCompletedSteps failed:', error)
			}
		},
		async handleGenerateFlowchart() {
			// P1 修复：公开只读模式禁止生成流程图（防御性兜底，模板已 v-if 隐藏 CTA）
			if (this.isPublicView) return
			if (!this.recipeId || this.isGeneratingFlowchart || !this.canRequestFlowchart) return
			if (!this.canGenerateFlowchart) {
				uni.showToast({
					title: '先补充至少 3 个关键步骤',
					icon: 'none'
				})
				return
			}

			// 已有流程图时，重新生成会消耗一次 AI 额度，做二次确认
			if (this.hasFlowchart) {
				const confirmed = await new Promise((resolve) => {
					uni.showModal({
						title: '重新生成步骤图？',
						content: '将再次调用 AI 生成，消耗 1 次额度，约需 15 秒。',
						confirmText: '继续生成',
						confirmColor: '#b4664c',
						success: ({ confirm }) => resolve(!!confirm),
						fail: () => resolve(false)
					})
				})
				if (!confirmed) return
			}

			await this.submitFlowchartGeneration()
		},
		async submitFlowchartGeneration() {
			this.isGeneratingFlowchart = true
			uni.showLoading({
				title: '提交中',
				mask: true
			})

			try {
				const recipe = await generateRecipeFlowchartById(this.recipeId)
				this.applyRecipe(recipe)
				this.showActionFeedback({
					tone: 'pending',
					title: '已加入生成队列',
					description:
						buildFlowchartWaitHint(recipe?.flowchartStatus, recipe?.flowchartQueueAhead, recipe?.flowchartEstimatedWaitSeconds) ||
						flowchartStatusMetaMap.pending.description,
					duration: 1800
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '生成流程图失败',
					icon: 'none'
				})
			} finally {
				this.isGeneratingFlowchart = false
				uni.hideLoading()
			}
		},
		async togglePinned() {
			if (!this.recipeId || !this.recipe || this.isPinSubmitting) return

			const nextPinned = !this.isPinned
			this.isPinSubmitting = true
			uni.showLoading({
				title: nextPinned ? '置顶中' : '更新中',
				mask: true
			})

			try {
				const recipe = await setRecipePinnedById(this.recipeId, nextPinned)
				this.applyRecipe(recipe)
				uni.showToast({
					title: nextPinned ? '已置顶' : '已取消置顶',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '更新置顶失败',
					icon: 'none'
				})
			} finally {
				this.isPinSubmitting = false
				uni.hideLoading()
			}
		},
		confirmDeleteRecipe() {
			// P1 修复：公开只读模式禁止删除菜谱
			if (this.isPublicView) return
			if (!this.recipe) return
			uni.showModal({
				title: '删除菜品',
				content: '删除后会从列表和详情页移除。',
				confirmColor: '#c16a51',
				success: async ({ confirm }) => {
					if (!confirm) return
					await this.deleteCurrentRecipe()
				}
			})
		},
		async deleteCurrentRecipe() {
			if (this.isDeletingRecipe) return

			this.isDeletingRecipe = true
			uni.showLoading({
				title: '删除中',
				mask: true
			})

			try {
				await deleteRecipeById(this.recipeId)
				uni.showToast({
					title: '已删除',
					icon: 'none'
				})
				setTimeout(() => {
					this.goBack()
				}, 280)
			} catch (error) {
				uni.showToast({
					title: error?.message || '删除失败',
					icon: 'none'
				})
			} finally {
				this.isDeletingRecipe = false
				uni.hideLoading()
			}
		},
		copyLink() {
			const link = this.displayRecipeLink
			if (!link) {
				uni.showToast({
					title: '暂无链接',
					icon: 'none'
				})
				return
			}
			uni.setClipboardData({
				data: link,
				success: () => {
					uni.showToast({
						title: '已复制链接',
						icon: 'none'
					})
				}
			})
		},
		previewRecipeImage() {
			return this.getRecipeImageController().previewCurrent()
		},
		openHeroActionMenu() {
			// P1 修复：公开只读模式禁止 Hero 操作菜单（替换/重排/删除封面图入口）
			if (this.isPublicView) return
			if (!this.canShowHeroActionMenu) return

			const items = []
			const actions = []

			if (this.canSetCurrentAsCover) {
				items.push('设为封面')
				actions.push('set-cover')
			}
			if (this.canAddMoreHeroImages) {
				items.push('添加更多图片')
				actions.push('add-more')
			}
			if (this.canDeleteCurrentImage) {
				items.push('删除这张图')
				actions.push('delete')
			}

			if (!items.length) return

			uni.showActionSheet({
				itemList: items,
				success: ({ tapIndex }) => {
					const action = actions[tapIndex]
					if (action === 'set-cover') {
						this.setCurrentImageAsCover()
					} else if (action === 'add-more') {
						this.chooseHeroImages()
					} else if (action === 'delete') {
						this.confirmDeleteCurrentImage()
					}
				}
			})
		},
		// Hero 修复：把「可见列表」(displayRecipeImages) 索引映射回 recipeImages 原始索引
		// 当某些图加载失败被 recipeImageHiddenMap 标记时，两个数组长度/顺序会错位
		// 返回 -1 表示无法映射（越界或可见列表为空）
		resolveOriginalImageIndex(visibleIndex) {
			return this.getRecipeImageController().originalIndex(visibleIndex)
		},
		async setCurrentImageAsCover() {
			return this.getRecipeImageController().setCurrentAsCover()
		},
		confirmDeleteCurrentImage() {
			if (!this.canDeleteCurrentImage) return
			uni.showModal({
				title: '删除这张图？',
				content: '删除后无法恢复，仍可重新上传。',
				confirmText: '删除',
				confirmColor: '#b4664c',
				success: ({ confirm }) => {
					if (!confirm) return
					this.deleteCurrentImage()
				}
			})
		},
		async deleteCurrentImage() {
			return this.getRecipeImageController().deleteCurrent()
		},
		openFlowchartViewer() {
			if (!this.flowchartDisplayImageUrl) return
			const key = `${this.recipeId || 'recipe'}-${Date.now()}`
			uni.setStorageSync(FLOWCHART_VIEWER_STORAGE_KEY, {
				key,
				imageUrl: this.flowchartImageUrl,
				localImagePath: String(this.cachedFlowchartImagePath || '').trim(),
				title: String(this.recipe?.title || '').trim(),
				updatedAtText: this.flowchartUpdatedAtText
			})
			uni.navigateTo({
				url: `/pages/flowchart-viewer/index?key=${encodeURIComponent(key)}`
			})
		},
		async handleFlowchartImageError() {
			const localPath = String(this.cachedFlowchartImagePath || '').trim()
			if (!localPath || this.flowchartDisplayImageUrl !== localPath) return
			this.cachedFlowchartImagePath = ''
			try {
				await invalidateCachedImage(this.flowchartImageUrl, this.flowchartImageCacheVersion || this.buildFlowchartImageCacheVersion())
			} catch (error) {
				// Ignore stale cache cleanup failures and keep remote fallback usable.
			}
		},
		previewFlowchartImage() {
			// 轻点图片：用系统原生 previewImage 做快速预览（双指缩放、保存、长按菜单）
			// 与右下「横屏查看 ›」胶囊的横屏沉浸模式区分：轻 = 快看，重 = 横屏沉浸
			const urls = this.buildFlowchartPreviewURLs()
			if (!urls.length) return
			uni.vibrateShort && uni.vibrateShort({ type: 'light' })
			uni.previewImage({
				urls,
				current: urls[0]
			})
		},
		goBack() {
			if (getCurrentPages().length > 1) {
				uni.navigateBack()
				return
			}
			uni.reLaunch({
				url: '/pages/index/index'
			})
		},
		// P2-D 分享路径升级：失效空态 CTA
		// 公开模式下优先 navigateBack 关闭当前页（用户期望「关掉这个失效页面」），
		// 私有路径走 goBack 兜底（reLaunch 到首页）
		handleMissingStateBack() {
			if (this.isPublicView) {
				if (getCurrentPages().length > 1) {
					uni.navigateBack()
					return
				}
				uni.reLaunch({ url: '/pages/index/index' })
				return
			}
			this.goBack()
		},
		// P2-D 分享路径升级：banner 「了解」按钮，弹出只读规则说明
		openPublicReadOnlyExplain() {
			this.showPublicReadOnlyExplain = true
		},
		closePublicReadOnlyExplain() {
			this.showPublicReadOnlyExplain = false
		}
	}
}
</script>

<style lang="scss" src="./recipe-detail-page.scss"></style>
