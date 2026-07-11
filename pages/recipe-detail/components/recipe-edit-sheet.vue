<template>
<up-popup
	v-if="showEditSheet"
	:show="showEditSheet"
	mode="bottom"
	round="32"
	overlayOpacity="0.22"
	:closeOnClickOverlay="false"
	:safeAreaInsetBottom="false"
	@close="handleEditSheetPopupClose"
>
	<view class="editor-sheet">
		<view class="editor-sheet__header">
			<view class="editor-sheet__heading">
				<text class="editor-sheet__title">编辑菜品</text>
				<text class="editor-sheet__subtitle">把这道菜补充完整。</text>
			</view>
			<view class="editor-sheet__close" @tap="requestCloseEditSheet">
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
						<view
							v-if="editDraft.images.length > 1"
							class="editor-gallery__sort"
							@tap.stop="openEditImageOrderActions(index)"
						>
							<text class="editor-gallery__sort-text">排序</text>
						</view>
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
					{{ editDraft.images.length ? `已添加 ${editDraft.images.length} 张，首张作封面，可调整顺序。` : `最多上传 ${maxRecipeImages} 张，首张作封面。` }}
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
				<view class="editor-field__head">
					<text class="editor-field__label">食材清单</text>
					<text class="editor-field__meta">{{ editIngredientCount }} 项</text>
				</view>
				<view class="editor-structured">
					<view class="editor-structured__section">
						<view class="editor-structured__header">
							<view class="editor-structured__heading">
								<text class="editor-structured__title">主料</text>
								<text class="editor-structured__desc">核心食材和份量</text>
							</view>
							<view class="editor-structured__action" @tap="addEditIngredient('main')">
								<text class="editor-structured__action-text">添加</text>
							</view>
						</view>

						<view v-if="editDraft.mainIngredients.length" class="editor-ingredient-list">
							<view
								v-for="(ingredient, index) in editDraft.mainIngredients"
								:key="ingredient.id"
								class="editor-ingredient-item"
							>
								<view class="editor-ingredient-item__index">
									<text class="editor-ingredient-item__index-text">{{ index + 1 }}</text>
								</view>
								<input
									:value="ingredient.value"
									class="editor-ingredient-item__input"
									placeholder="例如：牛肉 500g"
									placeholder-class="editor-input__placeholder"
									maxlength="60"
									@input="handleEditIngredientInput('main', index, $event)"
								/>
								<view class="editor-ingredient-item__menu" @tap="openEditIngredientActions('main', index)">
									<view class="editor-ingredient-item__menu-dots">
										<view class="editor-ingredient-item__menu-dot"></view>
										<view class="editor-ingredient-item__menu-dot"></view>
										<view class="editor-ingredient-item__menu-dot"></view>
									</view>
								</view>
							</view>
						</view>
						<view v-else class="editor-structured__empty">
							<text class="editor-structured__empty-text">{{ ingredientGroupEmptyText('main') }}</text>
						</view>
					</view>

					<view class="editor-structured__section">
						<view class="editor-structured__header">
							<view class="editor-structured__heading">
								<text class="editor-structured__title">辅料 / 调味</text>
								<text class="editor-structured__desc">配菜、调味和辅助食材</text>
							</view>
							<view class="editor-structured__action" @tap="addEditIngredient('secondary')">
								<text class="editor-structured__action-text">添加</text>
							</view>
						</view>

						<view v-if="editDraft.secondaryIngredients.length" class="editor-ingredient-list">
							<view
								v-for="(ingredient, index) in editDraft.secondaryIngredients"
								:key="ingredient.id"
								class="editor-ingredient-item"
							>
								<view class="editor-ingredient-item__index">
									<text class="editor-ingredient-item__index-text">{{ index + 1 }}</text>
								</view>
								<input
									:value="ingredient.value"
									class="editor-ingredient-item__input"
									placeholder="例如：葱姜蒜、盐、生抽"
									placeholder-class="editor-input__placeholder"
									maxlength="60"
									@input="handleEditIngredientInput('secondary', index, $event)"
								/>
								<view class="editor-ingredient-item__menu" @tap="openEditIngredientActions('secondary', index)">
									<view class="editor-ingredient-item__menu-dots">
										<view class="editor-ingredient-item__menu-dot"></view>
										<view class="editor-ingredient-item__menu-dot"></view>
										<view class="editor-ingredient-item__menu-dot"></view>
									</view>
								</view>
							</view>
						</view>
						<view v-else class="editor-structured__empty">
							<text class="editor-structured__empty-text">{{ ingredientGroupEmptyText('secondary') }}</text>
						</view>
					</view>
				</view>
				<text class="editor-field__hint">
					{{ editIsUsingFallbackContent ? '还没添加食材，直接补充即可。' : '食材按主料和辅料分开显示，可调整顺序。' }}
				</text>
			</view>

			<view class="editor-field">
				<view class="editor-field__head">
					<text class="editor-field__label">制作步骤</text>
					<text class="editor-field__meta">{{ editStepCount }} 步</text>
				</view>
				<view class="editor-step-list">
					<view
						v-for="(step, index) in editDraft.steps"
						:key="step.id"
						class="editor-step-card"
					>
						<view class="editor-step-card__header">
							<view class="editor-step-card__badge">
								<text class="editor-step-card__badge-text">Step {{ index + 1 }}</text>
							</view>
							<view class="editor-step-card__actions">
								<view
									class="editor-step-card__action"
									:class="{ 'editor-step-card__action--disabled': index === 0 }"
									@tap="moveEditStep(index, index - 1)"
								>
									<text class="editor-step-card__action-text">上移</text>
								</view>
								<view
									class="editor-step-card__action"
									:class="{ 'editor-step-card__action--disabled': index === editDraft.steps.length - 1 }"
									@tap="moveEditStep(index, index + 1)"
								>
									<text class="editor-step-card__action-text">下移</text>
								</view>
								<view class="editor-step-card__action editor-step-card__action--danger" @tap="removeEditStep(index)">
									<text class="editor-step-card__action-text editor-step-card__action-text--danger">删除</text>
								</view>
							</view>
						</view>

						<view class="editor-step-card__field">
							<text class="editor-step-card__label">步骤标题</text>
							<input
								:value="step.title"
								class="editor-step-card__input"
								placeholder="例如：腌制入味"
								placeholder-class="editor-input__placeholder"
								maxlength="30"
								@input="handleEditStepFieldInput(index, 'title', $event)"
							/>
						</view>

						<view class="editor-step-card__field">
							<text class="editor-step-card__label">步骤内容</text>
							<textarea
								:value="step.detail"
								auto-height
								class="editor-step-card__textarea"
								placeholder="写清楚这一小步的动作、时间或火候"
								placeholder-class="editor-textarea__placeholder"
								maxlength="220"
								@input="handleEditStepFieldInput(index, 'detail', $event)"
							/>
						</view>
					</view>

					<view v-if="!editDraft.steps.length" class="editor-structured__empty editor-structured__empty--large">
						<text class="editor-structured__empty-text">{{ stepEmptyText() }}</text>
					</view>

					<view class="editor-step-add" @tap="addEditStep">
						<text class="editor-step-add__text">添加一步</text>
					</view>
				</view>
				<text class="editor-field__hint">
					{{ editIsUsingFallbackContent ? '还没添加步骤，直接补充即可。' : '步骤标题可选填，留空会自动补全。' }}
				</text>
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
			<view class="editor-sheet__action" @tap="requestCloseEditSheet">
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
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import {
	buildRecipeEditPayload,
	cloneStepDraftList,
	createEmptyDraft,
	createIngredientDraftList,
	createStepDraftItem,
	moveListItem,
	normalizeIngredientDraftItem,
	serializeComparableEditDraft
} from '../use-recipe-edit'

const props = defineProps({
	showEditSheet: Boolean,
	initialDraft: {
		type: Object,
		default: () => ({})
	},
	maxRecipeImages: Number,
	mealTabs: Array,
	statusTabs: Array,
	isSaving: Boolean
})
const emit = defineEmits(['close', 'save'])
const editDraft = ref(createEmptyDraft())
const initialSnapshot = ref('')

const editIngredientCount = computed(() => {
	const mainCount = Array.isArray(editDraft.value.mainIngredients) ? editDraft.value.mainIngredients.length : 0
	const secondaryCount = Array.isArray(editDraft.value.secondaryIngredients) ? editDraft.value.secondaryIngredients.length : 0
	return mainCount + secondaryCount
})
const editStepCount = computed(() => Array.isArray(editDraft.value.steps) ? editDraft.value.steps.length : 0)
const editIsUsingFallbackContent = computed(() => editDraft.value.parsedContentMode === 'fallback')
const canSaveEditDraft = computed(() => !!String(editDraft.value.title || '').trim() && !props.isSaving)
const hasUnsavedChanges = computed(() => initialSnapshot.value !== serializeComparableEditDraft(editDraft.value))

function cloneDraft(source = {}) {
	return createEmptyDraft({
		...source,
		images: Array.isArray(source.images) ? [...source.images] : [],
		mainIngredients: createIngredientDraftList(source.mainIngredients),
		secondaryIngredients: createIngredientDraftList(source.secondaryIngredients),
		steps: cloneStepDraftList(source.steps)
	})
}

function resetDraft(source = props.initialDraft) {
	const nextDraft = cloneDraft(source)
	editDraft.value = nextDraft
	initialSnapshot.value = serializeComparableEditDraft(nextDraft)
}

watch(
	() => [props.showEditSheet, props.initialDraft],
	([visible]) => {
		if (visible) resetDraft()
	},
	{ immediate: true }
)

function handleEditSheetPopupClose() { requestCloseEditSheet() }
function requestCloseEditSheet() {
	if (!props.showEditSheet || props.isSaving) return
	if (!hasUnsavedChanges.value) {
		emit('close')
		return
	}
	uni.showModal({
		title: '放弃当前修改？',
		content: '未保存的食材、步骤和备注改动会丢失。',
		cancelText: '继续编辑',
		confirmText: '放弃修改',
		confirmColor: '#b4664c',
		success: ({ confirm }) => {
			if (confirm) emit('close')
		}
	})
}

function previewEditImages(index = 0) {
	const urls = Array.isArray(editDraft.value.images) ? editDraft.value.images.filter(Boolean) : []
	if (!urls.length) return
	uni.previewImage({ current: urls[index] || urls[0], urls })
}

function chooseEditImages() {
	const remaining = Math.max((props.maxRecipeImages || 0) - editDraft.value.images.length, 0)
	if (!remaining) {
		uni.showToast({ title: `最多上传 ${props.maxRecipeImages} 张`, icon: 'none' })
		return
	}
	uni.chooseImage({
		count: remaining,
		sizeType: ['compressed'],
		sourceType: ['album', 'camera'],
		success: ({ tempFilePaths }) => {
			const nextImages = [...editDraft.value.images]
			;(tempFilePaths || []).forEach((path) => {
				if (path && !nextImages.includes(path) && nextImages.length < props.maxRecipeImages) nextImages.push(path)
			})
			editDraft.value.images = nextImages
		}
	})
}

function removeEditImage(index) {
	if (typeof index !== 'number') return
	editDraft.value.images = editDraft.value.images.filter((_, currentIndex) => currentIndex !== index)
}

function moveEditImage(fromIndex, toIndex) {
	const nextImages = moveListItem(editDraft.value.images, fromIndex, toIndex)
	if (nextImages !== editDraft.value.images) editDraft.value.images = nextImages
}

function openEditImageOrderActions(index) {
	const images = editDraft.value.images.filter(Boolean)
	if (typeof index !== 'number' || images.length < 2 || index < 0 || index >= images.length) return
	const actions = []
	if (index > 0) {
		actions.push({ label: '设为封面', handler: () => moveEditImage(index, 0) })
		actions.push({ label: '左移一位', handler: () => moveEditImage(index, index - 1) })
	}
	if (index < images.length - 1) actions.push({ label: '右移一位', handler: () => moveEditImage(index, index + 1) })
	uni.showActionSheet({
		itemList: actions.map((item) => item.label),
		success: ({ tapIndex }) => actions[tapIndex]?.handler?.()
	})
}

function getIngredientFieldKey(group = 'main') {
	return group === 'secondary' ? 'secondaryIngredients' : 'mainIngredients'
}
function markParsedContentEdited() {
	if (editDraft.value.parsedContentMode !== 'manual') editDraft.value.parsedContentMode = 'manual'
}
function ingredientGroupEmptyText(group = 'main') {
	if (editIsUsingFallbackContent.value) return group === 'secondary' ? '还没添加辅料或调味。' : '还没添加主料。'
	return group === 'secondary'
		? '还没添加辅料或调味，比如葱姜蒜、盐、生抽。'
		: '还没添加主料，比如牛肉 500g。'
}
function stepEmptyText() {
	return editIsUsingFallbackContent.value ? '还没添加步骤。' : '还没添加步骤，可先补 3 到 6 步。'
}
function addEditIngredient(group = 'main') {
	const key = getIngredientFieldKey(group)
	editDraft.value[key] = [...editDraft.value[key], normalizeIngredientDraftItem()]
	markParsedContentEdited()
}
function handleEditIngredientInput(group = 'main', index = 0, event) {
	const key = getIngredientFieldKey(group)
	const list = [...editDraft.value[key]]
	if (index < 0 || index >= list.length) return
	list[index] = { ...normalizeIngredientDraftItem(list[index]), value: String(event?.detail?.value || '') }
	editDraft.value[key] = list
	markParsedContentEdited()
}
function removeEditIngredient(group = 'main', index = 0) {
	const key = getIngredientFieldKey(group)
	editDraft.value[key] = editDraft.value[key].filter((_, currentIndex) => currentIndex !== index)
	markParsedContentEdited()
}
function moveEditIngredient(group = 'main', fromIndex = 0, toIndex = 0) {
	const key = getIngredientFieldKey(group)
	const next = moveListItem(editDraft.value[key], fromIndex, toIndex)
	if (next === editDraft.value[key]) return
	editDraft.value[key] = next
	markParsedContentEdited()
}
function moveEditIngredientToGroup(fromGroup = 'main', index = 0, toGroup = 'secondary') {
	const fromKey = getIngredientFieldKey(fromGroup)
	const toKey = getIngredientFieldKey(toGroup)
	const source = [...editDraft.value[fromKey]]
	if (index < 0 || index >= source.length) return
	const [item] = source.splice(index, 1)
	editDraft.value[fromKey] = source
	editDraft.value[toKey] = [...editDraft.value[toKey], item]
	markParsedContentEdited()
}
function openEditIngredientActions(group = 'main', index = 0) {
	const key = getIngredientFieldKey(group)
	const ingredients = editDraft.value[key]
	if (index < 0 || index >= ingredients.length) return
	const actions = []
	if (index > 0) actions.push({ label: '上移一位', handler: () => moveEditIngredient(group, index, index - 1) })
	if (index < ingredients.length - 1) actions.push({ label: '下移一位', handler: () => moveEditIngredient(group, index, index + 1) })
	actions.push({
		label: group === 'secondary' ? '移到主料' : '移到辅料 / 调味',
		handler: () => moveEditIngredientToGroup(group, index, group === 'secondary' ? 'main' : 'secondary')
	})
	actions.push({ label: '删除', handler: () => removeEditIngredient(group, index) })
	uni.showActionSheet({
		itemList: actions.map((item) => item.label),
		success: ({ tapIndex }) => actions[tapIndex]?.handler?.()
	})
}
function addEditStep() {
	editDraft.value.steps = [...editDraft.value.steps, createStepDraftItem()]
	markParsedContentEdited()
}
function handleEditStepFieldInput(index = 0, field = 'title', event) {
	const steps = [...editDraft.value.steps]
	if (index < 0 || index >= steps.length) return
	steps[index] = { ...createStepDraftItem(steps[index]), [field]: String(event?.detail?.value || '') }
	editDraft.value.steps = steps
	markParsedContentEdited()
}
function moveEditStep(fromIndex = 0, toIndex = 0) {
	const next = moveListItem(editDraft.value.steps, fromIndex, toIndex)
	if (next === editDraft.value.steps) return
	editDraft.value.steps = next
	markParsedContentEdited()
}
function removeEditStep(index = 0) {
	editDraft.value.steps = editDraft.value.steps.filter((_, currentIndex) => currentIndex !== index)
	markParsedContentEdited()
}
function saveEditDraft() {
	if (!canSaveEditDraft.value) return
	emit('save', buildRecipeEditPayload(editDraft.value))
}
defineExpose({ requestCloseEditSheet })
</script>
