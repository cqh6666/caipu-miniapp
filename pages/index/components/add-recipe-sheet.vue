<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.34"
		:closeOnClickOverlay="!isSubmitting"
		:safeAreaInsetBottom="false"
		@close="handleRequestClose"
	>
		<view
			class="sheet"
			:class="{ 'sheet--submitting': isSubmitting, 'sheet--expanded': optionalExpanded }"
		>
			<view class="sheet__handle"></view>
			<view class="sheet__header">
				<view class="sheet__heading">
					<text class="sheet__title">添加菜品</text>
				</view>
				<view
					class="sheet__close"
					:class="{ 'sheet__close--disabled': isSubmitting }"
					@tap="handleRequestClose"
				>
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<scroll-view class="sheet__body" scroll-y>
				<view class="sheet-panel sheet-panel--entry">
					<view class="form-field">
						<text class="form-field__label form-field__label--strong">菜谱链接</text>
						<input
							:value="draft.link"
							class="sheet-input"
							placeholder="粘贴 B 站或小红书分享文案 / 链接"
							placeholder-class="sheet-input__placeholder"
							maxlength="300"
							@input="$emit('link-input', $event)"
						/>
						<view
							v-if="draftLinkAssistText"
							class="field-status"
							:class="{
								'field-status--loading': isLinkPreviewing,
								'field-status--error': hasLinkPreviewError,
								'field-status--info': !isLinkPreviewing && !hasLinkPreviewError
							}"
						>
							<text class="field-status__text">{{ draftLinkAssistText }}</text>
						</view>
					</view>

					<view class="form-field">
						<text class="form-field__label form-field__label--strong">菜名</text>
						<input
							:value="draft.title"
							class="sheet-input sheet-input--title"
							placeholder="可手动填写，或等自动识别"
							placeholder-class="sheet-input__placeholder"
							maxlength="40"
							@input="$emit('title-input', $event)"
						/>
						<view
							v-if="titleFieldAssistText"
							class="field-status"
							:class="titleFieldStatusClasses"
						>
							<text class="field-status__text">{{ titleFieldAssistText }}</text>
						</view>
					</view>
				</view>

				<view class="sheet-panel sheet-panel--secondary">
					<view class="sheet-panel__toggle" @tap="toggleOptionalExpanded">
						<view class="sheet-panel__toggle-copy">
							<view class="sheet-panel__title-row sheet-panel__title-row--toggle">
								<text class="sheet-panel__title">补充信息（可选）</text>
								<text class="sheet-panel__meta">{{ optionalMetaText }}</text>
							</view>
							<text class="sheet-panel__desc">{{ optionalSummaryText }}</text>
						</view>
						<view
							class="sheet-panel__toggle-icon"
							:class="{ 'sheet-panel__toggle-icon--expanded': optionalExpanded }"
						>
							<up-icon name="arrow-down" size="16" color="#8b7a69"></up-icon>
						</view>
					</view>
					<view v-if="optionalExpanded" class="sheet-panel__content">
						<view class="form-field">
							<text class="form-field__label">成品图（可选）</text>
							<view
								class="upload-gallery"
								:class="{ 'upload-gallery--empty': !draft.images.length }"
							>
								<view
									v-for="(image, index) in draft.images"
									:key="`draft-image-${index}`"
									class="upload-gallery__item"
									@tap="$emit('preview-image', index)"
								>
									<image class="upload-gallery__thumb" :src="image" mode="aspectFill"></image>
									<view class="upload-gallery__badge">
										<text class="upload-gallery__badge-text">
											{{ index === 0 ? '封面' : index + 1 }}
										</text>
									</view>
									<view class="upload-gallery__remove" @tap.stop="$emit('remove-image', index)">
										<up-icon name="close" size="14" color="#ffffff"></up-icon>
									</view>
								</view>
								<view
									v-if="draft.images.length < maxRecipeImages"
									class="upload-gallery__add"
									:class="{ 'upload-gallery__add--compact': !draft.images.length }"
									@tap="$emit('choose-images')"
								>
									<view class="upload-gallery__plus">
										<up-icon name="plus" size="20" color="#8c8074"></up-icon>
									</view>
									<text class="upload-gallery__add-text">上传成品图</text>
								</view>
							</view>
							<text class="form-field__hint">{{ imageHint }}</text>
						</view>

						<view class="form-field">
							<text class="form-field__label">分类</text>
							<view class="segment">
								<view
									v-for="tab in mealTabs"
									:key="tab.value"
									class="segment__item"
									:class="{ 'segment__item--active': draft.mealType === tab.value }"
									@tap="$emit('select-meal-type', tab.value)"
								>
									<text class="segment__text">{{ tab.label }}</text>
								</view>
							</view>
						</view>

						<view class="form-field">
							<text class="form-field__label">状态</text>
							<view class="segment">
								<view
									v-for="tab in draftStatusOptions"
									:key="tab.value"
									class="segment__item"
									:class="{
										'segment__item--active': draft.status === tab.value,
										'segment__item--wishlist': draft.status === tab.value && tab.value === 'wishlist',
										'segment__item--done': draft.status === tab.value && tab.value === 'done'
									}"
									@tap="$emit('select-status', tab.value)"
								>
									<text class="segment__text">{{ tab.label }}</text>
								</view>
							</view>
						</view>

						<view class="form-field">
							<text class="form-field__label">备注</text>
							<textarea
								:value="draft.note"
								class="sheet-textarea"
								placeholder="比如口味、做法备注、视频亮点"
								placeholder-class="sheet-textarea__placeholder"
								maxlength="300"
								@input="$emit('note-input', $event)"
							/>
						</view>
					</view>
				</view>
			</scroll-view>

			<view class="sheet__footer">
				<text
					class="sheet__footer-hint"
					:class="{
						'sheet__footer-hint--loading': footerAssistTone === 'loading',
						'sheet__footer-hint--ready': footerAssistTone === 'ready',
						'sheet__footer-hint--warning': footerAssistTone === 'warning'
					}"
				>
					{{ footerAssistText }}
				</text>
				<view class="sheet__footer-actions">
					<view
						class="sheet-action sheet-action--secondary"
						:class="{ 'sheet-action--disabled': isSubmitting }"
						@tap="handleRequestClose"
					>
						<text class="sheet-action__text">取消</text>
					</view>
					<view
						class="sheet-action sheet-action--primary"
						:class="{ 'sheet-action--disabled': !canSubmit || isSubmitting }"
						@tap="handleSubmit"
					>
						<text class="sheet-action__text sheet-action__text--primary">
							{{ isSubmitting ? '保存中...' : '保存菜品' }}
						</text>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
function normalizeOptionalImageList(images = []) {
	return (Array.isArray(images) ? images : []).map((item) => String(item || '').trim()).filter(Boolean)
}

function buildOptionalDraftSnapshot(draft = {}) {
	return {
		note: String(draft?.note || '').trim(),
		images: normalizeOptionalImageList(draft?.images),
		mealType: String(draft?.mealType || '').trim(),
		status: String(draft?.status || '').trim()
	}
}

function stringArraysEqual(left = [], right = []) {
	if (left.length !== right.length) return false
	return left.every((item, index) => item === right[index])
}

export default {
	name: 'AddRecipeSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		draft: {
			type: Object,
			default: () => ({
				link: '',
				title: '',
				images: [],
				mealType: '',
				status: '',
				note: ''
			})
		},
		draftLinkAssistText: {
			type: String,
			default: ''
		},
		isLinkPreviewing: {
			type: Boolean,
			default: false
		},
		hasLinkPreviewError: {
			type: Boolean,
			default: false
		},
		draftTitleAssistText: {
			type: String,
			default: ''
		},
		hasAutoTitle: {
			type: Boolean,
			default: false
		},
		isTitleTouched: {
			type: Boolean,
			default: false
		},
		maxRecipeImages: {
			type: Number,
			default: 0
		},
		mealTabs: {
			type: Array,
			default: () => []
		},
		draftStatusOptions: {
			type: Array,
			default: () => []
		},
		canSubmit: {
			type: Boolean,
			default: false
		},
		isSubmitting: {
			type: Boolean,
			default: false
		}
	},
	emits: [
		'choose-images',
		'close',
		'link-input',
		'note-input',
		'preview-image',
		'remove-image',
		'select-meal-type',
		'select-status',
		'submit',
		'title-input'
	],
	data() {
		return {
			optionalExpanded: false,
			initialOptionalState: buildOptionalDraftSnapshot()
		}
	},
	computed: {
		hasDraftLink() {
			return !!String(this.draft?.link || '').trim()
		},
		titleFieldAssistText() {
			return this.draftTitleAssistText || ''
		},
		titleFieldStatusClasses() {
			return {
				'field-status--success': this.hasAutoTitle,
				'field-status--info': !!this.draftTitleAssistText && !this.hasAutoTitle,
				'field-status--muted': this.isTitleTouched
			}
		},
		optionalChangedState() {
			const current = buildOptionalDraftSnapshot(this.draft)
			const initial = this.initialOptionalState || buildOptionalDraftSnapshot()
			return {
				note: !!current.note && current.note !== initial.note,
				images: !stringArraysEqual(current.images, initial.images),
				mealType: !!current.mealType && current.mealType !== initial.mealType,
				status: !!current.status && current.status !== initial.status
			}
		},
		optionalFilledCount() {
			return Object.values(this.optionalChangedState).filter(Boolean).length
		},
		optionalSummaryText() {
			if (this.optionalFilledCount) {
				return `已补充 ${this.optionalFilledCount} 项，点开继续完善。`
			}
			return '备注、图片、分类和状态都收在这里。'
		},
		optionalMetaText() {
			if (this.optionalExpanded) {
				return '收起'
			}
			return this.optionalFilledCount ? `已填 ${this.optionalFilledCount}/4` : '可选'
		},
		footerAssistText() {
			if (this.isSubmitting) {
				return '正在保存...'
			}
			if (this.canSubmit) {
				return '可直接保存。'
			}
			if (this.isLinkPreviewing) {
				return '识别中，也可以直接填菜名。'
			}
			if (this.hasLinkPreviewError && this.hasDraftLink) {
				return '链接未识别，补菜名即可保存。'
			}
			if (this.hasDraftLink) {
				return '再填菜名即可保存。'
			}
			return '填写菜名后可保存。'
		},
		footerAssistTone() {
			if (this.isSubmitting) {
				return 'loading'
			}
			if (this.canSubmit) {
				return 'ready'
			}
			if (this.hasLinkPreviewError) {
				return 'warning'
			}
			return 'pending'
		},
		imageHint() {
			const images = Array.isArray(this.draft?.images) ? this.draft.images : []
			if (images.length) {
				return `已添加 ${images.length} 张，首张会作为封面展示。`
			}
			return `最多上传 ${this.maxRecipeImages} 张，首张会作为封面展示。`
		}
	},
	watch: {
		show: {
			immediate: true,
			handler(next) {
				this.initialOptionalState = next ? buildOptionalDraftSnapshot(this.draft) : buildOptionalDraftSnapshot()
				if (!next) {
					this.optionalExpanded = false
					return
				}
				this.optionalExpanded = false
			}
		}
	},
	methods: {
		handleRequestClose() {
			if (this.isSubmitting) return
			this.$emit('close')
		},
		handleSubmit() {
			if (this.isSubmitting || !this.canSubmit) return
			this.$emit('submit')
		},
		toggleOptionalExpanded() {
			if (this.isSubmitting) return
			this.optionalExpanded = !this.optionalExpanded
		}
	}
}
</script>

<style lang="scss" scoped>
@import './add-recipe-sheet.scss';
@import './sheet-action.scss';
</style>
