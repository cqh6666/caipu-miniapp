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
		<view class="sheet" :class="{ 'sheet--submitting': isSubmitting, 'sheet--expanded': true }">
			<view class="sheet__handle"></view>
			<view class="sheet__header">
				<view class="sheet__heading">
					<text class="sheet__title">{{ isEdit ? '编辑打卡点' : '添加打卡点' }}</text>
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
						<text class="form-field__label form-field__label--strong">名称</text>
						<input
							:value="draft.name"
							class="sheet-input sheet-input--title"
							placeholder="比如：周末想去的店"
							placeholder-class="sheet-input__placeholder"
							maxlength="40"
							@input="$emit('name-input', $event)"
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">状态</text>
						<view class="segment">
							<view
								v-for="tab in statusOptions"
								:key="tab.value"
								class="segment__item"
								:class="{
									'segment__item--active': draft.status === tab.value,
									'segment__item--wishlist': draft.status === tab.value && tab.value === 'want',
									'segment__item--done': draft.status === tab.value && tab.value === 'visited'
								}"
								@tap="$emit('select-status', tab.value)"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>
				</view>

				<view class="sheet-panel sheet-panel--secondary">
					<view class="form-field">
						<text class="form-field__label">类型</text>
						<view class="segment">
							<view
								v-for="tab in typeOptions"
								:key="tab.value"
								class="segment__item"
								:class="{ 'segment__item--active': draft.type === tab.value }"
								@tap="$emit('select-type', tab.value)"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="form-field">
						<text class="form-field__label">位置</text>
						<view class="location-picker" @tap="$emit('choose-location')">
							<view class="location-picker__main">
								<text class="location-picker__title">{{ locationTitle }}</text>
								<text class="location-picker__desc">{{ locationDesc }}</text>
							</view>
							<view class="location-picker__action">
								<up-icon name="map-fill" size="16" color="#745742"></up-icon>
								<text class="location-picker__action-text">选点</text>
							</view>
						</view>
						<input
							:value="draft.address"
							class="sheet-input"
							placeholder="也可以手动补充地址"
							placeholder-class="sheet-input__placeholder"
							maxlength="120"
							@input="$emit('address-input', $event)"
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">费用</text>
						<input
							:value="draft.price"
							class="sheet-input"
							placeholder="比如：¥98/人、免费"
							placeholder-class="sheet-input__placeholder"
							maxlength="40"
							@input="$emit('price-input', $event)"
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">图片</text>
						<view class="upload-gallery" :class="{ 'upload-gallery--empty': !draft.images.length }">
							<view
								v-for="(image, index) in draft.images"
								:key="`place-draft-image-${index}`"
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
							<view
								v-if="draft.images.length < maxPlaceImages"
								class="upload-gallery__add"
								:class="{ 'upload-gallery__add--compact': !draft.images.length }"
								@tap="$emit('choose-images')"
							>
								<view class="upload-gallery__plus">
									<up-icon name="plus" size="20" color="#8c8074"></up-icon>
								</view>
								<text class="upload-gallery__add-text">上传照片</text>
							</view>
						</view>
						<text class="form-field__hint">{{ imageHint }}</text>
					</view>

					<view class="form-field">
						<text class="form-field__label">标签</text>
						<input
							:value="tagText"
							class="sheet-input"
							placeholder="用逗号分隔，比如：早午餐，拍照"
							placeholder-class="sheet-input__placeholder"
							maxlength="120"
							@input="$emit('tags-input', $event)"
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">来源</text>
						<view class="segment">
							<view
								v-for="tab in sourceOptions"
								:key="tab.value"
								class="segment__item"
								:class="{ 'segment__item--active': draft.source === tab.value }"
								@tap="$emit('select-source', tab.value)"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
						<input
							:value="draft.sourceUrl"
							class="sheet-input"
							placeholder="来源链接（可选）"
							placeholder-class="sheet-input__placeholder"
							maxlength="300"
							@input="$emit('source-url-input', $event)"
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">备注</text>
						<textarea
							:value="draft.note"
							class="sheet-textarea"
							placeholder="比如推荐菜、排队情况、同行人的小偏好"
							placeholder-class="sheet-textarea__placeholder"
							maxlength="300"
							@input="$emit('note-input', $event)"
						/>
					</view>
				</view>
			</scroll-view>

			<view class="sheet__footer">
				<text
					class="sheet__footer-hint"
					:class="{
						'sheet__footer-hint--loading': isSubmitting,
						'sheet__footer-hint--ready': canSubmit
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
							{{ isSubmitting ? '保存中...' : '保存打卡点' }}
						</text>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'PlaceEditSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		isEdit: {
			type: Boolean,
			default: false
		},
		draft: {
			type: Object,
			default: () => ({
				name: '',
				type: 'food',
				address: '',
				latitude: 0,
				longitude: 0,
				price: '',
				source: 'manual',
				sourceUrl: '',
				images: [],
				status: 'want',
				tags: [],
				note: ''
			})
		},
		statusOptions: {
			type: Array,
			default: () => []
		},
		typeOptions: {
			type: Array,
			default: () => []
		},
		sourceOptions: {
			type: Array,
			default: () => []
		},
		maxPlaceImages: {
			type: Number,
			default: 0
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
		'address-input',
		'choose-images',
		'choose-location',
		'close',
		'name-input',
		'note-input',
		'preview-image',
		'price-input',
		'remove-image',
		'select-source',
		'select-status',
		'select-type',
		'source-url-input',
		'submit',
		'tags-input'
	],
	computed: {
		tagText() {
			return (Array.isArray(this.draft?.tags) ? this.draft.tags : []).join('，')
		},
		hasLocation() {
			return !!(Number(this.draft?.latitude) && Number(this.draft?.longitude))
		},
		locationTitle() {
			if (this.draft?.address) return this.draft.address
			return this.hasLocation ? '已选择位置' : '选择地图位置'
		},
		locationDesc() {
			if (this.hasLocation) {
				return `${Number(this.draft.latitude).toFixed(5)}, ${Number(this.draft.longitude).toFixed(5)}`
			}
			return '保存坐标后，可以直接打开地图导航。'
		},
		imageHint() {
			const images = Array.isArray(this.draft?.images) ? this.draft.images : []
			if (images.length) {
				return `已添加 ${images.length} 张，首张会作为封面展示。`
			}
			return `最多上传 ${this.maxPlaceImages} 张。`
		},
		footerAssistText() {
			if (this.isSubmitting) return '正在保存...'
			if (this.canSubmit) return '位置可以稍后再补，名称填好就能保存。'
			return '填写名称后可保存。'
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
		}
	}
}
</script>

<style lang="scss" scoped>
@import './add-recipe-sheet.scss';
@import './sheet-action.scss';

.location-picker {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 18rpx;
	padding: 20rpx;
	border-radius: 24rpx;
	background: rgba(255, 253, 249, 0.98);
	border: 1px solid rgba(111, 86, 64, 0.1);
}

.location-picker__main {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	gap: 6rpx;
}

.location-picker__title {
	font-size: 26rpx;
	font-weight: 700;
	line-height: 1.4;
	color: #41362d;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.location-picker__desc {
	font-size: 22rpx;
	line-height: 1.4;
	color: #918578;
}

.location-picker__action {
	display: flex;
	align-items: center;
	gap: 8rpx;
	flex-shrink: 0;
	padding: 12rpx 16rpx;
	border-radius: 18rpx;
	background: #f4ece2;
}

.location-picker__action-text {
	font-size: 22rpx;
	font-weight: 700;
	color: #745742;
}
</style>
