<template>
	<view class="detail-page">
		<template v-if="recipe">
			<scroll-view class="detail-scroll" scroll-y>
				<view
					class="hero-card"
					:class="{ 'hero-card--empty': !recipeImages.length }"
					@tap="handleHeroCardTap"
				>
					<swiper
						v-if="recipeImages.length"
						class="hero-card__swiper"
						:circular="recipeImages.length > 1"
						:autoplay="recipeImages.length > 1"
						:interval="3600"
						:duration="320"
						@change="handleHeroSwiperChange"
					>
						<swiper-item v-for="(image, index) in recipeImages" :key="`hero-image-${index}`">
							<image class="hero-card__image" :src="image" mode="aspectFill"></image>
						</swiper-item>
					</swiper>
					<view v-if="recipeImages.length" class="hero-card__preview-tip">
						<up-icon name="photo" size="14" color="#ffffff"></up-icon>
						<text class="hero-card__preview-tip-text">查看大图</text>
					</view>
					<view v-if="recipeImages.length > 1" class="hero-card__counter">
						<text class="hero-card__counter-text">{{ heroImageIndex + 1 }} / {{ recipeImages.length }}</text>
					</view>
					<view v-if="!recipeImages.length" class="hero-card__placeholder">
						<view class="hero-card__placeholder-mask"></view>
						<view class="hero-card__upload-action" :class="{ 'hero-card__upload-action--loading': isUploadingHeroImage }">
							<up-icon :name="isUploadingHeroImage ? 'reload' : 'plus'" size="18" color="#5b4a3b"></up-icon>
							<text class="hero-card__upload-action-text">{{ isUploadingHeroImage ? '上传中...' : '上传成品图' }}</text>
						</view>
					</view>
				</view>

				<view class="detail-head">
					<text class="detail-meta">{{ detailMetaLine }}</text>
					<text class="detail-title">{{ recipe.title }}</text>
					<text v-if="recipe.summary" class="detail-summary">{{ recipe.summary }}</text>
				</view>

				<view class="detail-card detail-card--flowchart">
					<view class="detail-card__header">
						<view class="detail-card__heading">
							<text class="detail-card__title">AI 流程图</text>
							<text class="detail-card__subtitle">把关键步骤整理成一张图，进来就能先看懂顺序。</text>
						</view>
						<view
							class="detail-card__action detail-card__action--accent"
							:class="{ 'detail-card__action--disabled': !canRequestFlowchart || isGeneratingFlowchart }"
							@tap="handleGenerateFlowchart"
						>
							<text class="detail-card__action-text detail-card__action-text--accent">{{ flowchartActionText }}</text>
						</view>
					</view>

					<view
						v-if="flowchartStatusMeta"
						class="detail-parse"
						:class="`detail-parse--${flowchartStatusMeta.tone}`"
					>
						<view class="detail-parse__body">
							<view class="detail-parse__badge">
								<text class="detail-parse__badge-text">{{ flowchartStatusMeta.label }}</text>
							</view>
							<text class="detail-parse__desc">{{ flowchartStatusDescription }}</text>
						</view>
					</view>

					<view v-if="showFlowchartStaleHint" class="flowchart-hint">
						<up-icon name="info-circle" size="14" color="#b4664c"></up-icon>
						<text class="flowchart-hint__text">做法已更新，建议重新生成</text>
					</view>

					<view v-if="hasFlowchart" class="flowchart-panel" @tap="previewFlowchartImage">
						<image class="flowchart-panel__image" :src="flowchartImageUrl" mode="widthFix"></image>
						<view class="flowchart-panel__footer">
							<text v-if="flowchartUpdatedAtText" class="flowchart-panel__meta">{{ flowchartUpdatedAtText }}</text>
							<text class="flowchart-panel__preview">点击查看大图</text>
						</view>
					</view>

					<view v-else class="flowchart-empty" :class="{ 'flowchart-empty--disabled': !canGenerateFlowchart }">
						<view class="flowchart-empty__icon">
							<up-icon name="photo" size="24" color="#b08c72"></up-icon>
						</view>
						<text class="flowchart-empty__title">还没有流程图</text>
						<text class="flowchart-empty__desc">{{ flowchartEmptyText }}</text>
					</view>
				</view>

				<view class="detail-card">
					<view class="detail-card__header">
						<text class="detail-card__title">做法整理</text>
						<view
							v-if="canRequestParse"
							class="detail-card__action detail-card__action--accent"
							:class="{ 'detail-card__action--disabled': isReparseSubmitting }"
							@tap="handleParseAction"
						>
							<text class="detail-card__action-text detail-card__action-text--accent">{{ parseActionText }}</text>
						</view>
					</view>

					<view
						v-if="parseStatusMeta"
						class="detail-parse"
						:class="`detail-parse--${parseStatusMeta.tone}`"
					>
						<view class="detail-parse__body">
							<view class="detail-parse__badge">
								<text class="detail-parse__badge-text">{{ parseStatusMeta.label }}</text>
							</view>
							<text class="detail-parse__desc">{{ parseStatusDescription }}</text>
							<text v-if="parseStatusSourceLabel" class="detail-parse__meta">{{ parseStatusSourceLabel }}</text>
						</view>
					</view>

					<view class="parsed-section">
						<text class="parsed-section__title">主料</text>
						<view
							v-for="(ingredient, index) in parsedMainIngredients"
							:key="`main-ingredient-${index}`"
							class="parsed-item"
						>
							<view class="parsed-item__index">
								<text class="parsed-item__index-text">{{ index + 1 }}</text>
							</view>
							<text class="parsed-item__text">{{ ingredient }}</text>
						</view>
					</view>

					<view v-if="parsedSecondaryGroups.length" class="parsed-section">
						<text class="parsed-section__title">辅料</text>
						<view
							v-for="group in parsedSecondaryGroups"
							:key="group.key"
							class="parsed-group"
						>
							<text class="parsed-group__label">{{ group.label }}</text>
							<text class="parsed-group__text">{{ group.text }}</text>
						</view>
					</view>

					<view class="parsed-section parsed-section--steps">
						<text class="parsed-section__title">制作步骤</text>
						<view
							v-for="(step, index) in parsedSteps"
							:key="`step-${index}`"
							class="step-item"
						>
							<view class="step-item__index">
								<text class="step-item__index-text">Step {{ index + 1 }}</text>
							</view>
							<view class="step-item__body">
								<text class="step-item__title">{{ step.title }}</text>
								<text class="step-item__text">{{ step.detail }}</text>
							</view>
						</view>
					</view>
				</view>

				<view class="detail-card">
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
							<text class="detail-link-text" selectable>{{ recipe.link }}</text>
						</view>
					</view>
					<text v-else class="detail-empty">暂无链接。</text>
				</view>

				<view class="detail-card detail-card--note">
					<view class="detail-card__header detail-card__header--stack">
						<text class="detail-card__title">备注</text>
					</view>
					<text v-if="recipe.note" class="detail-note">{{ recipe.note }}</text>
					<text v-else class="detail-empty">暂无备注。</text>
				</view>
			</scroll-view>

			<view class="detail-footer">
				<view class="detail-footer__action detail-footer__action--ghost" @tap="confirmDeleteRecipe">
					<text class="detail-footer__text detail-footer__text--danger">删除</text>
				</view>
				<view
					class="detail-footer__action detail-footer__action--soft"
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
				<view class="detail-footer__action detail-footer__action--primary" @tap="openEditSheet">
					<text class="detail-footer__text detail-footer__text--primary">编辑</text>
				</view>
			</view>
		</template>

		<template v-else>
			<view class="missing-state">
				<up-icon name="info-circle" size="42" color="#b8aa9b"></up-icon>
				<text class="missing-state__title">没找到这道菜</text>
				<text class="missing-state__desc">可能已删除或未保存。</text>
				<view class="missing-state__action" @tap="goBack">
					<text class="missing-state__action-text">返回列表</text>
				</view>
			</view>
		</template>

		<up-popup
			:show="showEditSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeEditSheet"
		>
			<view class="editor-sheet">
				<view class="editor-sheet__header">
					<view class="editor-sheet__heading">
						<text class="editor-sheet__title">编辑菜品</text>
						<text class="editor-sheet__subtitle">把这道菜补充完整。</text>
					</view>
					<view class="editor-sheet__close" @tap="closeEditSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<scroll-view class="editor-sheet__body" scroll-y>
					<view class="editor-field">
						<text class="editor-field__label">菜名</text>
						<input
							v-model="editDraft.title"
							class="editor-input editor-input--title"
							placeholder="输入菜名"
							placeholder-class="editor-input__placeholder"
							maxlength="40"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">主要食材</text>
						<input
							v-model="editDraft.ingredient"
							class="editor-input"
							placeholder="例如：牛肉"
							placeholder-class="editor-input__placeholder"
							maxlength="60"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">链接</text>
						<input
							v-model="editDraft.link"
							class="editor-input"
							placeholder="粘贴菜谱或视频链接"
							placeholder-class="editor-input__placeholder"
							maxlength="300"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">成品图</text>
						<view class="editor-gallery">
							<view
								v-for="(image, index) in editDraft.images"
								:key="`edit-image-${index}`"
								class="editor-gallery__item"
								@tap="previewEditImages(index)"
							>
								<image class="editor-gallery__thumb" :src="image" mode="aspectFill"></image>
								<view class="editor-gallery__badge">
									<text class="editor-gallery__badge-text">{{ index === 0 ? '封面' : index + 1 }}</text>
								</view>
								<view class="editor-gallery__remove" @tap.stop="removeEditImage(index)">
									<up-icon name="close" size="14" color="#ffffff"></up-icon>
								</view>
							</view>
							<view
								v-if="editDraft.images.length < maxRecipeImages"
								class="editor-gallery__add"
								@tap="chooseEditImages"
							>
								<view class="editor-gallery__plus">
									<up-icon name="plus" size="20" color="#8c8074"></up-icon>
								</view>
								<text class="editor-gallery__add-text">上传成品图</text>
							</view>
						</view>
						<text class="editor-field__hint">
							{{ editDraft.images.length ? `已添加 ${editDraft.images.length} 张，首张会作为封面。` : `最多上传 ${maxRecipeImages} 张，首张会作为封面。` }}
						</text>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">分类</text>
						<view class="segment">
							<view
								v-for="tab in mealTabs"
								:key="tab.value"
								class="segment__item"
								:class="{ 'segment__item--active': editDraft.mealType === tab.value }"
								@tap="editDraft.mealType = tab.value"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">状态</text>
						<view class="segment">
							<view
								v-for="tab in statusTabs"
								:key="tab.value"
								class="segment__item"
								:class="{
									'segment__item--active': editDraft.status === tab.value,
									'segment__item--wishlist': editDraft.status === tab.value && tab.value === 'wishlist',
									'segment__item--done': editDraft.status === tab.value && tab.value === 'done'
								}"
								@tap="editDraft.status = tab.value"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">食材清单</text>
						<textarea
							v-model="editDraft.ingredientsText"
							class="editor-textarea"
							placeholder="一行一个食材"
							placeholder-class="editor-textarea__placeholder"
							maxlength="500"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">制作步骤</text>
						<textarea
							v-model="editDraft.stepsText"
							class="editor-textarea editor-textarea--large"
							placeholder="一行一步"
							placeholder-class="editor-textarea__placeholder"
							maxlength="800"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">备注</text>
						<textarea
							v-model="editDraft.note"
							class="editor-textarea"
							placeholder="口味、火候或视频亮点"
							placeholder-class="editor-textarea__placeholder"
							maxlength="300"
						/>
					</view>
				</scroll-view>

				<view class="editor-sheet__footer">
					<view class="editor-sheet__action" @tap="closeEditSheet">
						<text class="editor-sheet__action-text">取消</text>
					</view>
					<view
						class="editor-sheet__action editor-sheet__action--primary"
						:class="{ 'editor-sheet__action--disabled': !canSaveEditDraft }"
						@tap="saveEditDraft"
					>
						<text class="editor-sheet__action-text editor-sheet__action-text--primary">保存</text>
					</view>
				</view>
			</view>
		</up-popup>
	</view>
</template>

<script>
import {
	MAX_RECIPE_IMAGES,
	buildFallbackParsedContent,
	deleteRecipeById,
	generateRecipeFlowchartById,
	getCachedRecipeById,
	getRecipeById,
	mealTypeLabelMap,
	mealTypeOptions,
	reparseRecipeById,
	setRecipePinnedById,
	statusLabelMap,
	statusOptions,
	updateRecipeById
} from '../../utils/recipe-store'

const createEmptyDraft = (overrides = {}) => ({
	title: '',
	ingredient: '',
	link: '',
	images: [],
	mealType: 'breakfast',
	status: 'wishlist',
	ingredientsText: '',
	stepsText: '',
	note: '',
	...overrides
})

const listToText = (items = []) => items.join('\n')
const textToList = (text = '') =>
	text
		.split('\n')
		.map((item) => item.trim())
		.filter(Boolean)
const secondaryIngredientPattern = /(常用配菜|基础调味|常用调味料|调味|葱|姜|蒜|香叶|桂皮|八角|花椒|胡椒|盐|糖|冰糖|白糖|红糖|生抽|老抽|蚝油|料酒|鸡精|味精|醋|陈醋|米醋|香醋|豆瓣酱|辣椒|小米椒|淀粉|清水|热水|食用油|香油|芝麻油|花椒粉|辣椒粉|五香粉|十三香|孜然|芝麻|香菜|葱花)/
const secondaryIngredientExceptionPattern = /^(洋葱|红葱头|葱头)/
const ingredientSuffixPattern = /\s*(?:\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)|半个|半颗|半根|半头|适量|少许)$/
const stringSlicesEqual = (left = [], right = []) => {
	if (left.length !== right.length) return false
	return left.every((item, index) => item === right[index])
}
const stepSlicesEqual = (left = [], right = []) => {
	if (left.length !== right.length) return false
	return left.every((item, index) => item.title === right[index]?.title && item.detail === right[index]?.detail)
}
const normalizeTextList = (items = []) => {
	const source = Array.isArray(items) ? items : [items]
	const normalized = []
	const seen = new Set()

	source.forEach((item) => {
		const value = String(item || '').trim()
		if (!value || seen.has(value)) return
		seen.add(value)
		normalized.push(value)
	})

	return normalized
}
const inferStepTitle = (detail = '', index = 0) => {
	const text = String(detail || '').trim()
	if (!text) return ''
	if (text.includes('焯水') || text.includes('汆水')) {
		return text.includes('腥') || text.includes('浮沫') ? '焯水去腥' : '焯水备用'
	}
	if (text.includes('腌')) return '腌制入味'
	if (text.includes('糖色') || text.includes('冰糖')) return '炒糖上色'
	if (text.includes('爆香') || text.includes('炒香')) return '炒香底料'
	if (text.includes('切') || text.includes('改刀')) return '切配备料'
	if (text.includes('收汁')) return '收汁出锅'
	if (text.includes('炖') || text.includes('焖')) return '小火慢炖'
	if (text.includes('蒸')) return '上锅蒸熟'
	if (text.includes('炸')) return '炸至金黄'
	if (text.includes('煎')) return '煎香上色'
	if (text.includes('烤')) return '烤至上色'
	if (text.includes('煮')) return '煮至入味'
	if (text.includes('拌')) return '拌匀调味'
	if (text.includes('炒') || text.includes('翻炒')) return '翻炒入味'
	if (text.includes('出锅')) return '调味出锅'
	return index === 0 ? '处理食材' : '继续烹饪'
}
const normalizeParsedSteps = (steps = []) => {
	const source = Array.isArray(steps) ? steps : []
	const normalized = []
	const seen = new Set()

	source.forEach((step) => {
		const title = typeof step === 'object' && step !== null ? String(step.title || '').trim() : ''
		const detail =
			typeof step === 'string'
				? step.trim()
				: String(step?.detail || step?.text || '').trim()
		const nextDetail = detail || title
		const nextTitle = title || inferStepTitle(nextDetail, normalized.length)
		if (!nextDetail) return
		const key = `${nextTitle}\u0000${nextDetail}`
		if (seen.has(key)) return
		seen.add(key)
		normalized.push({
			title: nextTitle,
			detail: nextDetail
		})
	})

	return normalized
}
const ingredientLabelFromLine = (line = '') => String(line || '').trim().replace(ingredientSuffixPattern, '').trim()
const splitIngredientLines = (lines = []) => {
	const cleaned = normalizeTextList(lines)
	if (!cleaned.length) {
		return {
			mainIngredients: [],
			secondaryIngredients: []
		}
	}

	const mainIngredients = []
	const secondaryIngredients = []
	cleaned.forEach((line) => {
		const label = ingredientLabelFromLine(line)
		if (secondaryIngredientPattern.test(label) && !secondaryIngredientExceptionPattern.test(label)) {
			secondaryIngredients.push(line)
			return
		}
		mainIngredients.push(line)
	})

	if (!mainIngredients.length) {
		return {
			mainIngredients: cleaned.slice(0, 3),
			secondaryIngredients: cleaned.slice(3)
		}
	}

	return {
		mainIngredients,
		secondaryIngredients
	}
}
const splitSecondaryIngredientLines = (lines = []) => {
	const cleaned = normalizeTextList(lines)
	const supportingIngredients = []
	const seasonings = []

	cleaned.forEach((line) => {
		const label = ingredientLabelFromLine(line)
		if (secondaryIngredientPattern.test(label) && !secondaryIngredientExceptionPattern.test(label)) {
			seasonings.push(line)
			return
		}
		supportingIngredients.push(line)
	})

	return {
		supportingIngredients,
		seasonings
	}
}
const normalizeParsedContentView = (parsedContent = {}) => {
	const mainIngredients = normalizeTextList(parsedContent.mainIngredients)
	const secondaryIngredients = normalizeTextList(parsedContent.secondaryIngredients)
	const legacyIngredients = normalizeTextList(parsedContent.ingredients)
	const groupedIngredients =
		mainIngredients.length || secondaryIngredients.length
			? { mainIngredients, secondaryIngredients }
			: splitIngredientLines(legacyIngredients)
	const secondaryGroups = splitSecondaryIngredientLines(groupedIngredients.secondaryIngredients)

	return {
		mainIngredients: groupedIngredients.mainIngredients,
		secondaryIngredients: groupedIngredients.secondaryIngredients,
		supportingIngredients: secondaryGroups.supportingIngredients,
		seasonings: secondaryGroups.seasonings,
		ingredients: [...groupedIngredients.mainIngredients, ...groupedIngredients.secondaryIngredients],
		steps: normalizeParsedSteps(parsedContent.steps)
	}
}
const stepListToText = (steps = []) => normalizeParsedSteps(steps).map((item) => item.detail).join('\n')

const ACTIVE_PARSE_STATUSES = ['pending', 'processing']
const ACTIVE_FLOWCHART_STATUSES = ['pending', 'processing']
const parseStatusMetaMap = {
	idle: {
		label: '可自动整理',
		tone: 'pending',
		description: '支持链接自动整理，可手动开始整理当前做法。'
	},
	pending: {
		label: '等待解析',
		tone: 'pending',
		description: '已加入后台整理队列，稍后会自动补齐食材和步骤。'
	},
	processing: {
		label: '解析中',
		tone: 'processing',
		description: '后台正在整理链接内容，结果会自动更新。'
	},
	done: {
		label: '已自动整理',
		tone: 'done',
		description: '食材和步骤已自动整理完成。'
	},
	failed: {
		label: '解析失败',
		tone: 'failed',
		description: '这次自动整理没成功，可以再试一次。'
	}
}

const flowchartStatusMetaMap = {
	pending: {
		label: '等待生成',
		tone: 'pending',
		description: '已加入流程图生成队列，稍后会自动更新。'
	},
	processing: {
		label: '生成中',
		tone: 'processing',
		description: '后台正在生成流程图，完成后会自动刷新。'
	},
	failed: {
		label: '生成失败',
		tone: 'failed',
		description: '这次流程图生成没成功，可以再试一次。'
	}
}

function isAutoParseSupportedLink(link = '') {
	return /(bilibili\.com|b23\.tv|bili2233\.cn|xiaohongshu\.com|xhslink\.com)/i.test(String(link).trim())
}

function extractCopyableLink(value = '') {
	const source = String(value || '').trim()
	if (!source) return ''
	const matched = source.match(/https?:\/\/[^\s]+/i)
	const link = String(matched?.[0] || source).trim()
	return link.replace(/[)\]】》」'",，。；;!?！？]+$/g, '').trim()
}

function isFallbackLikeParsedContent(recipe = {}, parsedContent = {}) {
	const current = normalizeParsedContentView(parsedContent)
	if (!current.ingredients.length && !current.steps.length) return true
	const fallback = buildFallbackParsedContent(recipe)
	return (
		stringSlicesEqual(current.mainIngredients, fallback.mainIngredients || []) &&
		stringSlicesEqual(current.secondaryIngredients, fallback.secondaryIngredients || []) &&
		stepSlicesEqual(current.steps, fallback.steps || [])
	)
}

function formatParseSourceLabel(source = '') {
	const value = String(source).trim()
	if (!value) return ''
	if (value === 'bilibili') return '来源：B 站链接自动解析'
	if (value === 'bilibili:ai') return '来源：B 站内容 + AI 总结'
	if (value === 'bilibili:heuristic') return '来源：B 站简介规则整理'
	if (value === 'xiaohongshu') return '来源：小红书链接自动解析'
	if (value === 'xiaohongshu:ai') return '来源：小红书图文 + AI 总结'
	if (value === 'xiaohongshu:heuristic') return '来源：小红书正文规则整理'
	return `来源：${value}`
}

function formatDateTime(value = '') {
	const date = new Date(value)
	if (Number.isNaN(date.getTime())) return ''
	const year = date.getFullYear()
	const month = `${date.getMonth() + 1}`.padStart(2, '0')
	const day = `${date.getDate()}`.padStart(2, '0')
	const hours = `${date.getHours()}`.padStart(2, '0')
	const minutes = `${date.getMinutes()}`.padStart(2, '0')
	return `${year}-${month}-${day} ${hours}:${minutes}`
}

export default {
	data() {
		return {
			recipeId: '',
			recipe: null,
			showEditSheet: false,
			editDraft: createEmptyDraft(),
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
			parsePollingTimer: null
		}
	},
	computed: {
		mealLabel() {
			return mealTypeLabelMap[this.recipe?.mealType] || '早餐'
		},
		statusLabel() {
			return statusLabelMap[this.recipe?.status] || '想吃'
		},
		isPinned() {
			return !!String(this.recipe?.pinnedAt || '').trim()
		},
		detailMetaLine() {
			return this.isPinned ? `${this.mealLabel} · ${this.statusLabel} · 已置顶` : `${this.mealLabel} · ${this.statusLabel}`
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
		recipeImages() {
			if (Array.isArray(this.recipe?.imageUrls) && this.recipe.imageUrls.length) {
				return this.recipe.imageUrls.filter(Boolean)
			}
			const fallbackImage = String(this.recipe?.image || this.recipe?.imageUrl || '').trim()
			return fallbackImage ? [fallbackImage] : []
		},
		flowchartImageUrl() {
			return String(this.recipe?.flowchartImageUrl || '').trim()
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
		flowchartActionText() {
			if (this.isGeneratingFlowchart) return '提交中...'
			if (ACTIVE_FLOWCHART_STATUSES.includes(this.flowchartStatusValue)) return '生成中...'
			return this.hasFlowchart ? '重新生成' : '生成流程图'
		},
		flowchartEmptyText() {
			if (this.canGenerateFlowchart) {
				return '生成后会把主流程整理成一张更直观的步骤图。'
			}
			return '先补充至少 3 个关键步骤，再生成流程图。'
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
			return this.flowchartStatusMeta.description
		},
		showFlowchartStaleHint() {
			return this.hasFlowchart && !!this.recipe?.flowchartStale
		},
		flowchartUpdatedAtText() {
			const value = formatDateTime(this.recipe?.flowchartUpdatedAt || '')
			return value ? `上次生成：${value}` : ''
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
			return this.parseStatusMeta.description
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
			return this.parseStatusValue === 'done' || this.parseStatusValue === 'failed' || this.hasMeaningfulParsedContent
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
		canSaveEditDraft() {
			return !!this.editDraft.title.trim()
		}
	},
	onLoad(options) {
		this.recipeId = options?.id || ''
	},
	onShow() {
		this.loadRecipe()
	},
	onHide() {
		this.stopParsePolling()
	},
	onUnload() {
		this.stopParsePolling()
	},
	methods: {
		async loadRecipe() {
			if (!this.recipeId) {
				this.recipe = null
				return
			}

			const cachedRecipe = getCachedRecipeById(this.recipeId)
			if (cachedRecipe) {
				this.applyRecipe(cachedRecipe)
			}

			try {
				this.isLoadingRecipe = true
				const recipe = await getRecipeById(this.recipeId, { preferCache: !cachedRecipe })
				this.applyRecipe(recipe)
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
			}
		},
		applyRecipe(recipe) {
			this.recipe = recipe
			if (this.heroImageIndex >= this.recipeImages.length) {
				this.heroImageIndex = 0
			}
			if (this.recipe?.title) {
				uni.setNavigationBarTitle({
					title: this.recipe.title
				})
			}
			this.syncParsePolling()
		},
		syncParsePolling() {
			const parseStatus = String(this.recipe?.parseStatus || '').trim()
			const flowchartStatus = String(this.recipe?.flowchartStatus || '').trim()
			if (!ACTIVE_PARSE_STATUSES.includes(parseStatus) && !ACTIVE_FLOWCHART_STATUSES.includes(flowchartStatus)) {
				this.stopParsePolling()
				return
			}

			if (this.parsePollingTimer) return

			this.parsePollingTimer = setInterval(() => {
				this.refreshParseStatus()
			}, 4000)
		},
		stopParsePolling() {
			if (!this.parsePollingTimer) return
			clearInterval(this.parsePollingTimer)
			this.parsePollingTimer = null
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
				ingredientsText: listToText(parsedContentView.ingredients || []),
				stepsText: stepListToText(parsedContentView.steps || []),
				note: recipe.note || ''
			})
		},
		openEditSheet() {
			if (!this.recipe) return
			this.editDraft = this.createDraftFromRecipe(this.recipe)
			this.showEditSheet = true
		},
		handleHeroCardTap() {
			if (!this.recipe) return
			if (this.recipeImages.length) {
				this.previewRecipeImage()
				return
			}
			this.chooseHeroImages()
		},
		handleHeroSwiperChange(event) {
			this.heroImageIndex = Number(event?.detail?.current) || 0
		},
		closeEditSheet() {
			this.showEditSheet = false
			this.editDraft = createEmptyDraft()
		},
		chooseHeroImages() {
			if (!this.recipe || this.isUploadingHeroImage) return
			const remaining = Math.max(this.maxRecipeImages - this.recipeImages.length, 0)
			if (!remaining) return

			uni.chooseImage({
				count: remaining,
				sizeType: ['compressed'],
				sourceType: ['album', 'camera'],
				success: ({ tempFilePaths }) => {
					if (!tempFilePaths || !tempFilePaths.length) return
					this.saveHeroImages(tempFilePaths)
				}
			})
		},
		async saveHeroImages(imagePaths = []) {
			const incoming = Array.isArray(imagePaths) ? imagePaths.filter(Boolean) : [imagePaths].filter(Boolean)
			if (!incoming.length || !this.recipeId || this.isUploadingHeroImage) return

			this.isUploadingHeroImage = true
			uni.showLoading({
				title: '上传中',
				mask: true
			})

			try {
				const nextImages = [...this.recipeImages]
				incoming.forEach((path) => {
					if (path && !nextImages.includes(path) && nextImages.length < this.maxRecipeImages) {
						nextImages.push(path)
					}
				})
				const recipe = await updateRecipeById(this.recipeId, {
					images: nextImages
				})
				this.applyRecipe(recipe)
				uni.showToast({
					title: `已添加 ${incoming.length} 张`,
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '上传失败',
					icon: 'none'
				})
			} finally {
				this.isUploadingHeroImage = false
				uni.hideLoading()
			}
		},
		chooseEditImages() {
			const remaining = Math.max(this.maxRecipeImages - this.editDraft.images.length, 0)
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
					const nextImages = [...this.editDraft.images]
					tempFilePaths.forEach((path) => {
						if (path && !nextImages.includes(path) && nextImages.length < this.maxRecipeImages) {
							nextImages.push(path)
						}
					})
					this.editDraft.images = nextImages
				}
			})
		},
		removeEditImage(index) {
			if (typeof index !== 'number') return
			this.editDraft.images = this.editDraft.images.filter((_, currentIndex) => currentIndex !== index)
		},
		previewEditImages(index = 0) {
			const urls = Array.isArray(this.editDraft.images) ? this.editDraft.images.filter(Boolean) : []
			if (!urls.length) return
			uni.previewImage({
				current: urls[index] || urls[0],
				urls
			})
		},
		async saveEditDraft() {
			if (!this.canSaveEditDraft || this.isSavingRecipe) return

			this.isSavingRecipe = true
			uni.showLoading({
				title: '保存中',
				mask: true
			})

			try {
				const recipe = await updateRecipeById(this.recipeId, {
					title: this.editDraft.title.trim(),
					ingredient: this.editDraft.ingredient.trim(),
					link: this.editDraft.link.trim(),
					images: this.editDraft.images,
					mealType: this.editDraft.mealType,
					status: this.editDraft.status,
					parsedContent: {
						ingredients: textToList(this.editDraft.ingredientsText),
						steps: textToList(this.editDraft.stepsText)
					},
					note: this.editDraft.note.trim()
				})
				this.closeEditSheet()
				this.applyRecipe(recipe)
				uni.showToast({
					title: '已保存',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '保存失败',
					icon: 'none'
				})
			} finally {
				this.isSavingRecipe = false
				uni.hideLoading()
			}
		},
		handleParseAction() {
			if (!this.canRequestParse || this.isReparseSubmitting) return
			if (!this.needsParseOverwriteConfirm) {
				this.requestAutoParse()
				return
			}

			uni.showModal({
				title: '更新做法整理',
				content: '将根据来源链接更新当前食材和步骤。',
				confirmText: '继续整理',
				confirmColor: '#b4664c',
				success: ({ confirm }) => {
					if (!confirm) return
					this.requestAutoParse()
				}
			})
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
				uni.showToast({
					title: '已加入整理队列',
					icon: 'none'
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
		async handleGenerateFlowchart() {
			if (!this.recipeId || this.isGeneratingFlowchart || !this.canRequestFlowchart) return
			if (!this.canGenerateFlowchart) {
				uni.showToast({
					title: '先补充至少 3 个关键步骤',
					icon: 'none'
				})
				return
			}

			this.isGeneratingFlowchart = true
			uni.showLoading({
				title: '提交中',
				mask: true
			})

			try {
				const recipe = await generateRecipeFlowchartById(this.recipeId)
				this.applyRecipe(recipe)
				uni.showToast({
					title: '已加入生成队列',
					icon: 'none'
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
			const link = extractCopyableLink(this.recipe?.link)
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
			const urls = this.recipeImages
			if (!urls.length) return

			uni.previewImage({
				current: urls[this.heroImageIndex] || urls[0],
				urls
			})
		},
		previewFlowchartImage() {
			if (!this.flowchartImageUrl) return
			uni.previewImage({
				current: this.flowchartImageUrl,
				urls: [this.flowchartImageUrl]
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
		}
	}
}
</script>

<style lang="scss" scoped>
	.detail-page {
		min-height: 100vh;
		background: #f6f4f1;
	}

	.detail-scroll {
		height: 100vh;
		box-sizing: border-box;
		padding: 24rpx 24rpx calc(env(safe-area-inset-bottom) + 188rpx);
	}

	.hero-card,
	.detail-card,
	.missing-state {
		border-radius: 28rpx;
		background: #ffffff;
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.05);
	}

	.hero-card {
		position: relative;
		overflow: hidden;
		min-height: 380rpx;
	}

	.hero-card--empty {
		min-height: 380rpx;
	}

	.hero-card__swiper {
		width: 100%;
		height: 380rpx;
	}

	.hero-card__image {
		width: 100%;
		height: 380rpx;
		display: block;
	}

	.hero-card__preview-tip {
		position: absolute;
		right: 22rpx;
		bottom: 22rpx;
		padding: 10rpx 16rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.46);
		display: flex;
		align-items: center;
		gap: 8rpx;
	}

	.hero-card__preview-tip-text {
		font-size: 21rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.hero-card__counter {
		position: absolute;
		left: 22rpx;
		bottom: 22rpx;
		padding: 10rpx 16rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.46);
	}

	.hero-card__counter-text {
		font-size: 21rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.hero-card__placeholder {
		position: relative;
		min-height: 380rpx;
		box-sizing: border-box;
		background:
			linear-gradient(135deg, rgba(255, 255, 255, 0.22), rgba(255, 255, 255, 0.08)),
			linear-gradient(135deg, #ddd2c4 0%, #cfbfae 100%);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.hero-card__placeholder-mask {
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		left: 0;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.16), rgba(255, 255, 255, 0.04)),
			radial-gradient(circle at center, rgba(255, 255, 255, 0.2), transparent 60%);
	}

	.hero-card__upload-action {
		position: relative;
		z-index: 1;
		padding: 16rpx 28rpx;
		border-radius: 999rpx;
		border: 1px solid rgba(255, 255, 255, 0.58);
		background: rgba(255, 255, 255, 0.74);
		box-shadow: 0 8rpx 18rpx rgba(91, 74, 59, 0.08);
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
	}

	.hero-card__upload-action--loading {
		background: rgba(246, 242, 237, 0.9);
	}

	.hero-card__upload-action-text {
		font-size: 25rpx;
		font-weight: 600;
		line-height: 1;
		color: #5b4a3b;
	}

	.detail-head {
		padding: 24rpx 6rpx 8rpx;
	}

	.detail-meta {
		display: block;
		font-size: 22rpx;
		font-weight: 600;
		color: #8c8176;
	}

	.detail-title {
		display: block;
		margin-top: 18rpx;
		font-size: 40rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.detail-summary {
		display: block;
		margin-top: 16rpx;
		font-size: 26rpx;
		line-height: 1.7;
		color: #5e544b;
	}

	.detail-card {
		margin-top: 18rpx;
		padding: 26rpx;
	}

	.detail-card__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
	}

	.detail-card__heading {
		flex: 1;
		min-width: 0;
	}

	.detail-card__header--stack {
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.detail-card__title {
		font-size: 30rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.detail-card__subtitle {
		display: block;
		margin-top: 10rpx;
		font-size: 22rpx;
		line-height: 1.6;
		color: #9b9186;
	}

	.detail-card__action {
		padding: 12rpx 20rpx;
		border-radius: 999rpx;
		background: #f2ece5;
	}

	.detail-card__action--accent {
		background: #fff2ea;
		border: 1px solid rgba(180, 102, 76, 0.12);
	}

	.detail-card__action--disabled {
		opacity: 0.6;
		pointer-events: none;
	}

	.detail-card__action-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #6d6155;
	}

	.detail-card__action-text--accent {
		color: #b4664c;
	}

	.link-panel {
		margin-top: 18rpx;
	}

	.detail-link-box {
		width: 100%;
		box-sizing: border-box;
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		background: #f8f5f1;
		border: 1px solid rgba(91, 74, 59, 0.08);
	}

	.detail-link-text {
		display: block;
		font-size: 24rpx;
		line-height: 1.7;
		color: #5e544b;
		white-space: normal;
		word-break: break-all;
	}

	.detail-note,
	.detail-empty {
		display: block;
		margin-top: 16rpx;
		font-size: 25rpx;
		line-height: 1.7;
		color: #5e544b;
	}

	.detail-empty {
		color: #9e9387;
	}

	.detail-card--flowchart {
		overflow: hidden;
	}

	.flowchart-hint {
		margin-top: 18rpx;
		padding: 14rpx 18rpx;
		border-radius: 18rpx;
		background: #fff2ea;
		display: flex;
		align-items: center;
		gap: 10rpx;
	}

	.flowchart-hint__text {
		font-size: 22rpx;
		line-height: 1.5;
		color: #b4664c;
	}

	.flowchart-panel {
		margin-top: 18rpx;
		border-radius: 24rpx;
		overflow: hidden;
		background: #f6f2ed;
		border: 1px solid rgba(91, 74, 59, 0.08);
	}

	.flowchart-panel__image {
		width: 100%;
		display: block;
		background: #f6f2ed;
	}

	.flowchart-panel__footer {
		padding: 16rpx 18rpx 18rpx;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16rpx;
	}

	.flowchart-panel__meta,
	.flowchart-panel__preview {
		font-size: 21rpx;
		line-height: 1.5;
		color: #8f8275;
	}

	.flowchart-panel__preview {
		font-weight: 600;
		color: #6d6155;
	}

	.flowchart-empty {
		margin-top: 18rpx;
		padding: 34rpx 28rpx;
		border-radius: 24rpx;
		background: linear-gradient(135deg, #f9f4ee, #f4ede4);
		border: 1px dashed rgba(180, 102, 76, 0.2);
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}

	.flowchart-empty--disabled {
		opacity: 0.72;
	}

	.flowchart-empty__icon {
		width: 78rpx;
		height: 78rpx;
		border-radius: 22rpx;
		background: rgba(255, 255, 255, 0.72);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.flowchart-empty__title {
		margin-top: 18rpx;
		font-size: 27rpx;
		font-weight: 700;
		color: #4e4339;
	}

	.flowchart-empty__desc {
		margin-top: 10rpx;
		font-size: 23rpx;
		line-height: 1.6;
		color: #8f8275;
	}

	.detail-parse {
		margin-top: 18rpx;
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18rpx;
	}

	.detail-parse--pending,
	.detail-parse--processing {
		background: #f7f1e7;
		border: 1px solid rgba(195, 150, 89, 0.16);
	}

	.detail-parse--done {
		background: #eef5ee;
		border: 1px solid rgba(111, 130, 109, 0.16);
	}

	.detail-parse--failed {
		background: #fbefec;
		border: 1px solid rgba(193, 106, 81, 0.14);
	}

	.detail-parse__body {
		flex: 1;
		min-width: 0;
	}

	.detail-parse__badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 6rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.72);
	}

	.detail-parse__badge-text {
		font-size: 20rpx;
		font-weight: 700;
		color: #6e5f50;
	}

	.detail-parse__desc,
	.detail-parse__meta {
		display: block;
		line-height: 1.6;
	}

	.detail-parse__desc {
		margin-top: 10rpx;
		font-size: 23rpx;
		color: #5e544b;
		word-break: break-all;
	}

	.detail-parse__meta {
		margin-top: 6rpx;
		font-size: 21rpx;
		color: #978b80;
	}

	.parsed-section {
		margin-top: 24rpx;
	}

	.parsed-section--steps {
		margin-top: 30rpx;
	}

	.parsed-section__title {
		display: block;
		font-size: 24rpx;
		font-weight: 700;
		color: #76695d;
	}

	.parsed-item,
	.step-item {
		margin-top: 14rpx;
		display: flex;
		align-items: flex-start;
		gap: 14rpx;
	}

	.parsed-group {
		margin-top: 14rpx;
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		background: #f8f5f1;
		border: 1px solid rgba(91, 74, 59, 0.08);
	}

	.parsed-group__label {
		display: inline-flex;
		padding: 6rpx 14rpx;
		border-radius: 999rpx;
		background: #efe8df;
		font-size: 20rpx;
		font-weight: 700;
		line-height: 1.2;
		color: #7a6c60;
	}

	.parsed-group__text {
		display: block;
		margin-top: 12rpx;
		font-size: 25rpx;
		line-height: 1.7;
		color: #4d433a;
	}

	.parsed-item__index {
		width: 40rpx;
		height: 40rpx;
		border-radius: 12rpx;
		background: #f1ebe4;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.parsed-item__index-text {
		font-size: 21rpx;
		font-weight: 700;
		color: #7d7064;
	}

	.parsed-item__text,
	.step-item__text {
		flex: 1;
		min-width: 0;
		font-size: 25rpx;
		line-height: 1.7;
		color: #4d433a;
	}

	.step-item__body {
		flex: 1;
		min-width: 0;
	}

	.step-item__title {
		display: block;
		font-size: 26rpx;
		font-weight: 700;
		line-height: 1.4;
		color: #2f2923;
	}

	.step-item__text {
		display: block;
		margin-top: 8rpx;
	}

	.step-item__index {
		flex-shrink: 0;
		min-height: 52rpx;
		padding: 0 14rpx;
		box-sizing: border-box;
		border-radius: 999rpx;
		background: #efe8df;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.step-item__index-text {
		display: block;
		font-size: 20rpx;
		line-height: 1;
		font-weight: 700;
		color: #786b5f;
	}

	.detail-card--note {
		margin-bottom: 6rpx;
	}

	.detail-footer {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 10;
		padding: 18rpx 24rpx calc(env(safe-area-inset-bottom) + 20rpx);
		background: linear-gradient(180deg, rgba(246, 244, 241, 0), rgba(246, 244, 241, 0.92) 20%, rgba(255, 255, 255, 0.98) 42%);
		display: flex;
		gap: 16rpx;
	}

	.detail-footer__action {
		flex: 1;
		height: 88rpx;
		border-radius: 24rpx;
		background: #f1ede8;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.detail-footer__action--ghost {
		background: #f7efec;
	}

	.detail-footer__action--primary {
		background: #5b4a3b;
		box-shadow: 0 12rpx 20rpx rgba(91, 74, 59, 0.16);
	}

	.detail-footer__action--soft {
		background: #f3ede5;
	}

	.detail-footer__action--soft-active {
		background: #f7efe2;
		box-shadow: inset 0 0 0 1px rgba(186, 145, 81, 0.16);
	}

	.detail-footer__action--disabled {
		opacity: 0.62;
		pointer-events: none;
	}

	.detail-footer__text {
		font-size: 28rpx;
		font-weight: 600;
		color: #675c51;
	}

	.detail-footer__text--danger {
		color: #b4664c;
	}

	.detail-footer__text--primary {
		color: #ffffff;
	}

	.detail-footer__text--accent {
		color: #9a7343;
	}

	.editor-sheet {
		height: 78vh;
		background: #ffffff;
		display: flex;
		flex-direction: column;
	}

	.editor-sheet__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
		padding: 28rpx 28rpx 18rpx;
	}

	.editor-sheet__heading {
		flex: 1;
		min-width: 0;
	}

	.editor-sheet__title {
		font-size: 38rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.editor-sheet__subtitle {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		line-height: 1.5;
		color: #9b9186;
	}

	.editor-sheet__close {
		width: 68rpx;
		height: 68rpx;
		border-radius: 18rpx;
		background: #f4f0eb;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.editor-sheet__body {
		flex: 1;
		min-height: 0;
		padding: 0 28rpx 28rpx;
		box-sizing: border-box;
	}

	.editor-field {
		display: flex;
		flex-direction: column;
		gap: 12rpx;
		margin-top: 26rpx;
	}

	.editor-field:first-child {
		margin-top: 0;
	}

	.editor-field__label {
		font-size: 22rpx;
		font-weight: 500;
		color: #9b9186;
	}

	.editor-field__hint {
		font-size: 22rpx;
		line-height: 1.6;
		color: #9b9186;
	}

	.editor-input,
	.editor-textarea {
		width: 100%;
		box-sizing: border-box;
		border-radius: 24rpx;
		background: #f7f4f0;
		border: 1px solid #ebe4db;
		color: #2f2923;
	}

	.editor-input {
		height: 88rpx;
		padding: 0 24rpx;
		font-size: 27rpx;
	}

	.editor-input--title {
		height: 96rpx;
		font-size: 30rpx;
		font-weight: 600;
		background: #ffffff;
		border-color: #e3dbd2;
	}

	.editor-input__placeholder,
	.editor-textarea__placeholder {
		color: #b7aea3;
	}

	.editor-textarea {
		min-height: 180rpx;
		padding: 22rpx 24rpx;
		font-size: 26rpx;
		line-height: 1.6;
	}

	.editor-textarea--large {
		min-height: 220rpx;
	}

	.editor-gallery {
		display: flex;
		flex-wrap: wrap;
		gap: 16rpx;
	}

	.editor-gallery__item,
	.editor-gallery__add {
		position: relative;
		width: calc((100% - 32rpx) / 3);
		height: 176rpx;
		border-radius: 24rpx;
		overflow: hidden;
	}

	.editor-gallery__item {
		background: #ebe4db;
	}

	.editor-gallery__thumb {
		width: 100%;
		height: 100%;
		display: block;
	}

	.editor-gallery__badge {
		position: absolute;
		left: 12rpx;
		bottom: 12rpx;
		padding: 8rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.58);
		backdrop-filter: blur(10rpx);
	}

	.editor-gallery__badge-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.editor-gallery__remove {
		position: absolute;
		top: 12rpx;
		right: 12rpx;
		width: 40rpx;
		height: 40rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.editor-gallery__add {
		border: 1px dashed #d8cec3;
		background: #faf7f3;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12rpx;
	}

	.editor-gallery__plus {
		width: 64rpx;
		height: 64rpx;
		border-radius: 20rpx;
		background: #f1ebe4;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.editor-gallery__add-text {
		font-size: 24rpx;
		font-weight: 600;
		color: #75685c;
	}

	.segment {
		display: flex;
		gap: 10rpx;
		padding: 8rpx;
		border-radius: 24rpx;
		background: #f3efea;
	}

	.segment__item {
		flex: 1;
		height: 76rpx;
		border-radius: 18rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
	}

	.segment__item--active {
		background: #ffffff;
		box-shadow: 0 8rpx 18rpx rgba(59, 47, 36, 0.06);
	}

	.segment__item--wishlist {
		background: #f3e7de;
	}

	.segment__item--done {
		background: #e8efe5;
	}

	.segment__text {
		font-size: 24rpx;
		font-weight: 600;
		color: #867a6f;
	}

	.segment__item--active .segment__text {
		color: #5b4a3b;
	}

	.editor-sheet__footer {
		padding: 18rpx 28rpx calc(env(safe-area-inset-bottom) + 20rpx);
		border-top: 1px solid rgba(91, 74, 59, 0.08);
		background: #ffffff;
		display: flex;
		gap: 16rpx;
	}

	.editor-sheet__action {
		flex: 1;
		height: 88rpx;
		border-radius: 24rpx;
		background: #f1ede8;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.editor-sheet__action--primary {
		background: #5b4a3b;
		box-shadow: 0 12rpx 20rpx rgba(91, 74, 59, 0.16);
	}

	.editor-sheet__action--disabled {
		background: #d9d1c8;
		box-shadow: none;
		pointer-events: none;
	}

	.editor-sheet__action-text {
		font-size: 28rpx;
		font-weight: 600;
		color: #675c51;
	}

	.editor-sheet__action-text--primary {
		color: #ffffff;
	}

	.missing-state {
		margin: 180rpx 24rpx 0;
		padding: 52rpx 32rpx;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}

	.missing-state__title {
		margin-top: 18rpx;
		font-size: 32rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.missing-state__desc {
		margin-top: 12rpx;
		font-size: 24rpx;
		line-height: 1.6;
		color: #8d847a;
	}

	.missing-state__action {
		margin-top: 24rpx;
		padding: 16rpx 28rpx;
		border-radius: 999rpx;
		background: #5b4a3b;
	}

	.missing-state__action-text {
		font-size: 24rpx;
		font-weight: 600;
		color: #ffffff;
	}
</style>
