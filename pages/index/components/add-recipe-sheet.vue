<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.22"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="sheet">
			<view class="sheet__header">
				<view class="sheet__heading">
					<text class="sheet__title">添加菜品</text>
					<text class="sheet__subtitle">先记下来，后面再慢慢补全</text>
				</view>
				<view class="sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<scroll-view class="sheet__body" scroll-y>
				<view class="form-field">
					<text class="form-field__label">菜谱链接</text>
					<input
						:value="draft.link"
						class="sheet-input"
						placeholder="支持直接粘贴 B 站或小红书分享链接"
						placeholder-class="sheet-input__placeholder"
						maxlength="300"
						@input="$emit('link-input', $event)"
					/>
					<text v-if="draftLinkAssistText" class="form-field__hint">{{ draftLinkAssistText }}</text>
				</view>

				<view class="form-field">
					<text class="form-field__label">菜名</text>
					<input
						:value="draft.title"
						class="sheet-input sheet-input--title"
						placeholder="可手动填写，或等待系统自动识别"
						placeholder-class="sheet-input__placeholder"
						maxlength="40"
						@input="$emit('title-input', $event)"
					/>
					<text v-if="draftTitleAssistText" class="form-field__hint">{{ draftTitleAssistText }}</text>
				</view>

				<view class="form-field">
					<text class="form-field__label">成品图（可选）</text>
					<view class="upload-gallery">
						<view
							v-for="(image, index) in draft.images"
							:key="`draft-image-${index}`"
							class="upload-gallery__item"
							@tap="$emit('preview-image', index)"
						>
							<image class="upload-gallery__thumb" :src="image" mode="aspectFill"></image>
							<view class="upload-gallery__badge">
								<text class="upload-gallery__badge-text">{{ index === 0 ? '封面' : index + 1 }}</text>
							</view>
							<view class="upload-gallery__remove" @tap.stop="$emit('remove-image', index)">
								<up-icon name="close" size="14" color="#ffffff"></up-icon>
							</view>
						</view>
						<view v-if="draft.images.length < maxRecipeImages" class="upload-gallery__add" @tap="$emit('choose-images')">
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
			</scroll-view>

			<view class="sheet__footer">
				<view class="sheet-action" @tap="$emit('close')">
					<text class="sheet-action__text">取消</text>
				</view>
				<view class="sheet-action sheet-action--primary" :class="{ 'sheet-action--disabled': !canSubmit }" @tap="$emit('submit')">
					<text class="sheet-action__text sheet-action__text--primary">保存</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
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
		draftTitleAssistText: {
			type: String,
			default: ''
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
	computed: {
		imageHint() {
			const images = Array.isArray(this.draft?.images) ? this.draft.images : []
			if (images.length) {
				return `已添加 ${images.length} 张，首张会作为封面展示。`
			}
			return `最多上传 ${this.maxRecipeImages} 张，首张会作为封面展示。`
		}
	}
}
</script>

<style lang="scss" scoped>
@import './add-recipe-sheet.scss';
@import './sheet-action.scss';
</style>
