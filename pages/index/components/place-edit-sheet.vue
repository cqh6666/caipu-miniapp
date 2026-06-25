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
						<text class="form-field__label">电话</text>
						<input
							:value="draft.phone"
							class="sheet-input"
							placeholder="方便订位、询问营业"
							placeholder-class="sheet-input__placeholder"
							maxlength="40"
							type="text"
							@input="$emit('phone-input', $event)"
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

				<!-- 体验记录区域（仅去过时显示） -->
				<view v-if="isVisitedStatus" class="sheet-panel sheet-panel--experience">
					<view class="panel-divider">
						<text class="panel-divider__text">体验记录</text>
					</view>

					<view class="form-field">
						<text class="form-field__label">重访意愿</text>
						<view class="rating-stars-input">
							<view
								v-for="star in 5"
								:key="`draft-star-${star}`"
								class="rating-star-input"
								@tap="$emit('rating-input', star)"
							>
								<up-icon
									:name="star <= (draft.revisitRating || 0) ? 'star-fill' : 'star'"
									size="28"
									:color="star <= (draft.revisitRating || 0) ? '#f4a236' : '#d4c4b8'"
								></up-icon>
							</view>
						</view>
						<text v-if="draft.revisitRating" class="form-field__hint">{{ revisitRatingLabel }}</text>
					</view>

					<view class="form-field">
						<text class="form-field__label">{{ recommendedItemsLabel }}</text>
						<textarea
							:value="recommendedItemsText"
							class="sheet-textarea"
							:placeholder="recommendedItemsPlaceholder"
							placeholder-class="sheet-textarea__placeholder"
							maxlength="288"
							@input="$emit('recommended-items-input', $event)"
						/>
						<text class="form-field__hint">逗号或换行分隔，最多12个</text>
					</view>
				</view>

				<!-- 更多信息折叠区 -->
				<view class="sheet-panel sheet-panel--collapsible">
					<view class="collapsible-header" @tap="toggleMoreInfo">
						<view class="collapsible-header__left">
							<up-icon
								:name="showMoreInfo ? 'arrow-down' : 'arrow-right'"
								size="16"
								color="#745742"
							></up-icon>
							<text class="collapsible-header__text">更多信息{{ moreInfoSummary }}</text>
						</view>
					</view>

					<view v-if="showMoreInfo" class="collapsible-body">
						<view v-if="draft.rating" class="form-field">
							<text class="form-field__label">外部评分</text>
							<view class="readonly-field">
								<text class="readonly-field__text">{{ externalRatingDisplay }}</text>
							</view>
						</view>

						<view class="form-field">
							<text class="form-field__label">就餐提示</text>
							<textarea
								:value="draft.diningTips"
								class="sheet-textarea"
								placeholder="比如：周末要排队，建议17:30前到"
								placeholder-class="sheet-textarea__placeholder"
								maxlength="160"
								@input="$emit('dining-tips-input', $event)"
							/>
						</view>

						<view class="form-field">
							<text class="form-field__label">适合场景</text>
							<input
								:value="scenesText"
								class="sheet-input"
								placeholder="比如：聚餐，约会，朋友小聚"
								placeholder-class="sheet-input__placeholder"
								maxlength="96"
								@input="$emit('scenes-input', $event)"
							/>
						</view>

						<view class="form-field">
							<text class="form-field__label">最佳时段</text>
							<input
								:value="draft.bestTime"
								class="sheet-input"
								placeholder="比如：晚餐时段 18:00-20:00"
								placeholder-class="sheet-input__placeholder"
								maxlength="60"
								@input="$emit('best-time-input', $event)"
							/>
						</view>

						<view class="form-field">
							<text class="form-field__label">就餐/游玩时长</text>
							<input
								:value="draft.duration"
								class="sheet-input"
								placeholder="比如：1-2小时"
								placeholder-class="sheet-input__placeholder"
								maxlength="20"
								@input="$emit('duration-input', $event)"
							/>
						</view>

						<view class="form-field">
							<text class="form-field__label">陪同人推荐</text>
							<input
								:value="companionTagsText"
								class="sheet-input"
								placeholder="比如：适合2-4人，一个人也OK"
								placeholder-class="sheet-input__placeholder"
								maxlength="72"
								@input="$emit('companion-tags-input', $event)"
							/>
						</view>

						<view class="form-field">
							<text class="form-field__label">停车/交通</text>
							<textarea
								:value="draft.parkingNote"
								class="sheet-textarea"
								placeholder="比如：商场地下停车场免费2小时"
								placeholder-class="sheet-textarea__placeholder"
								maxlength="160"
								@input="$emit('parking-note-input', $event)"
							/>
						</view>
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
		'best-time-input',
		'choose-images',
		'choose-location',
		'close',
		'companion-tags-input',
		'dining-tips-input',
		'duration-input',
		'name-input',
		'note-input',
		'parking-note-input',
		'phone-input',
		'preview-image',
		'price-input',
		'rating-input',
		'recommended-items-input',
		'remove-image',
		'scenes-input',
		'select-source',
		'select-status',
		'select-type',
		'source-url-input',
		'submit',
		'tags-input'
	],
	data() {
		return {
			showMoreInfo: false
		}
	},
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
		},
		isVisitedStatus() {
			return this.draft?.status === 'visited'
		},
		revisitRatingLabel() {
			const rating = Number(this.draft?.revisitRating || 0)
			if (rating === 5) return '非常推荐'
			if (rating === 4) return '值得再去'
			if (rating === 3) return '还可以'
			if (rating === 2) return '不太推荐'
			if (rating === 1) return '不推荐'
			return ''
		},
		recommendedItemsLabel() {
			const type = this.draft?.type || 'food'
			if (type === 'food') return '推荐菜'
			if (type === 'attraction') return '推荐打卡点'
			return '推荐项'
		},
		recommendedItemsPlaceholder() {
			const type = this.draft?.type || 'food'
			if (type === 'food') return '比如：碳烤肥牛，烤鸡翅，肥牛饭'
			if (type === 'attraction') return '比如：观景台，樱花大道'
			return '记录你推荐的项目'
		},
		recommendedItemsText() {
			const items = this.draft?.recommendedItems || []
			return Array.isArray(items) ? items.join('，') : ''
		},
		scenesText() {
			const items = this.draft?.scenes || []
			return Array.isArray(items) ? items.join('，') : ''
		},
		companionTagsText() {
			const items = this.draft?.companionTags || []
			return Array.isArray(items) ? items.join('，') : ''
		},
		externalRatingDisplay() {
			const rating = this.draft?.rating || ''
			const provider = this.draft?.externalProvider || ''
			if (!rating) return '暂无'
			if (provider === 'amap') return `高德评分 ${rating}`
			return rating
		},
		moreInfoSummary() {
			let count = 0
			if (this.draft?.diningTips) count++
			if (this.draft?.scenes && this.draft.scenes.length) count++
			if (this.draft?.bestTime) count++
			if (this.draft?.duration) count++
			if (this.draft?.companionTags && this.draft.companionTags.length) count++
			if (this.draft?.parkingNote) count++
			return count > 0 ? ` (已填${count}项)` : ''
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
		toggleMoreInfo() {
			this.showMoreInfo = !this.showMoreInfo
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

.sheet-panel--experience {
	margin-top: 32rpx;
	padding-top: 32rpx;
	border-top: 1px solid rgba(92, 64, 51, 0.08);
}

.panel-divider {
	display: flex;
	align-items: center;
	justify-content: center;
	margin-bottom: 32rpx;
}

.panel-divider__text {
	font-size: 26rpx;
	font-weight: 700;
	color: rgba(92, 64, 51, 0.5);
	line-height: 1;
	padding: 0 24rpx;
	background: rgba(255, 250, 244, 0.95);
	border-radius: 12rpx;
	min-height: 44rpx;
	display: flex;
	align-items: center;
}

.rating-stars-input {
	display: flex;
	align-items: center;
	gap: 16rpx;
	padding: 16rpx 0;
}

.rating-star-input {
	transition: transform 0.15s ease;

	&:active {
		transform: scale(0.88);
	}
}

.sheet-panel--collapsible {
	margin-top: 32rpx;
	padding-top: 0;
}

.collapsible-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 24rpx 0;
	cursor: pointer;
	transition: all 0.15s ease;

	&:active {
		opacity: 0.7;
	}
}

.collapsible-header__left {
	display: flex;
	align-items: center;
	gap: 12rpx;
}

.collapsible-header__text {
	font-size: 28rpx;
	font-weight: 700;
	color: #5c4033;
	line-height: 1;
}

.collapsible-body {
	display: flex;
	flex-direction: column;
	gap: 32rpx;
	padding-bottom: 16rpx;
}

.readonly-field {
	padding: 20rpx 24rpx;
	background: rgba(244, 236, 226, 0.5);
	border-radius: 20rpx;
	border: 1px solid rgba(111, 86, 64, 0.08);
}

.readonly-field__text {
	font-size: 26rpx;
	color: rgba(92, 64, 51, 0.6);
	line-height: 1.4;
}
</style>
