<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.42"
		:closeOnClickOverlay="!isSubmitting"
		:safeAreaInsetBottom="false"
		@close="handleRequestClose"
	>
		<view class="experience-sheet" :class="{ 'experience-sheet--submitting': isSubmitting }">
			<view class="experience-sheet__handle"></view>

			<view class="experience-sheet__hero">
				<view class="experience-sheet__icon-shell">
					<up-icon name="checkmark-circle-fill" size="32" color="#8a9a5b"></up-icon>
				</view>
				<text class="experience-sheet__title">🎉 打卡完成！</text>
				<text class="experience-sheet__subtitle">记录一下这次体验吧</text>
			</view>

			<scroll-view class="experience-sheet__body" scroll-y>
				<view class="experience-panel">
					<!-- 重访意愿 -->
					<view class="experience-field">
						<text class="experience-field__label">这家店怎么样？</text>
						<view class="rating-stars">
							<view
								v-for="star in 5"
								:key="`star-${star}`"
								class="rating-star"
								@tap="handleStarTap(star)"
							>
								<up-icon
									:name="star <= localRevisitRating ? 'star-fill' : 'star'"
									size="32"
									:color="star <= localRevisitRating ? '#f4a236' : '#d4c4b8'"
								></up-icon>
							</view>
						</view>
						<view v-if="!localRevisitRating" class="rating-quick-tags">
							<view
								v-for="tag in quickRatingTags"
								:key="tag.value"
								class="rating-quick-tag"
								@tap="handleQuickRating(tag.value)"
							>
								<text class="rating-quick-tag__text">{{ tag.label }}</text>
							</view>
						</view>
						<view v-else class="rating-label">
							<text class="rating-label__text">{{ ratingLabelText }}</text>
						</view>
					</view>

					<!-- 推荐菜/推荐项 -->
					<view class="experience-field">
						<text class="experience-field__label">{{ recommendedItemsLabel }} (可选)</text>
						<textarea
							:value="recommendedItemsText"
							class="experience-textarea"
							:placeholder="recommendedItemsPlaceholder"
							placeholder-class="experience-textarea__placeholder"
							maxlength="288"
							:disabled="isSubmitting"
							@input="handleRecommendedItemsInput"
						/>
						<text class="experience-field__hint">逗号或换行分隔，最多12个</text>
					</view>

					<!-- 到访日期 -->
					<view class="experience-field">
						<text class="experience-field__label">打卡日期</text>
						<view class="date-picker" @tap="handleDatePickerTap">
							<view class="date-picker__main">
								<text class="date-picker__value">{{ visitedAtDisplayText }}</text>
							</view>
							<up-icon name="calendar" size="18" color="#745742"></up-icon>
						</view>
					</view>
				</view>
			</scroll-view>

			<view class="experience-sheet__footer">
				<view class="experience-sheet__actions">
					<view
						class="experience-action experience-action--secondary"
						:class="{ 'experience-action--disabled': isSubmitting }"
						@tap="handleSkip"
					>
						<text class="experience-action__text">暂时跳过</text>
					</view>
					<view
						class="experience-action experience-action--primary"
						:class="{ 'experience-action--disabled': isSubmitting }"
						@tap="handleSubmit"
					>
						<text class="experience-action__text experience-action__text--primary">
							{{ isSubmitting ? '保存中...' : '保存体验' }}
						</text>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'PlaceExperienceSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		placeType: {
			type: String,
			default: 'food'
		},
		isSubmitting: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'submit', 'skip'],
	data() {
		return {
			localRevisitRating: 0,
			recommendedItemsText: '',
			visitedAt: '',
			quickRatingTags: [
				{ label: '非常推荐', value: 5 },
				{ label: '还行', value: 3 },
				{ label: '不推荐', value: 2 }
			]
		}
	},
	computed: {
		recommendedItemsLabel() {
			if (this.placeType === 'food') return '推荐菜'
			if (this.placeType === 'attraction') return '推荐打卡点'
			return '推荐项'
		},
		recommendedItemsPlaceholder() {
			if (this.placeType === 'food') return '比如：碳烤肥牛，烤鸡翅，肥牛饭'
			if (this.placeType === 'attraction') return '比如：观景台，樱花大道'
			return '记录你推荐的项目'
		},
		visitedAtDisplayText() {
			if (!this.visitedAt) return '今天'
			const date = new Date(this.visitedAt)
			const year = date.getFullYear()
			const month = String(date.getMonth() + 1).padStart(2, '0')
			const day = String(date.getDate()).padStart(2, '0')
			const today = new Date()
			const isToday =
				date.getFullYear() === today.getFullYear() &&
				date.getMonth() === today.getMonth() &&
				date.getDate() === today.getDate()
			return isToday ? '今天' : `${year}-${month}-${day}`
		},
		ratingLabelText() {
			if (this.localRevisitRating === 5) return '非常推荐'
			if (this.localRevisitRating === 4) return '值得再去'
			if (this.localRevisitRating === 3) return '还可以'
			if (this.localRevisitRating === 2) return '不太推荐'
			if (this.localRevisitRating === 1) return '不推荐'
			return ''
		}
	},
	watch: {
		show(newVal) {
			if (newVal) {
				this.resetForm()
			}
		}
	},
	methods: {
		resetForm() {
			this.localRevisitRating = 0
			this.recommendedItemsText = ''
			this.visitedAt = new Date().toISOString()
		},
		handleStarTap(star) {
			if (this.isSubmitting) return
			this.localRevisitRating = star
		},
		handleQuickRating(rating) {
			if (this.isSubmitting) return
			this.localRevisitRating = rating
		},
		handleRecommendedItemsInput(e) {
			this.recommendedItemsText = e.detail.value || ''
		},
		handleDatePickerTap() {
			if (this.isSubmitting) return
			uni.showModal({
				title: '选择日期',
				content: '当前版本暂不支持修改日期，默认为今天',
				showCancel: false
			})
		},
		handleRequestClose() {
			if (this.isSubmitting) return
			this.$emit('close')
		},
		handleSkip() {
			if (this.isSubmitting) return
			this.$emit('skip')
		},
		handleSubmit() {
			if (this.isSubmitting) return

			const recommendedItems = this.recommendedItemsText
				.split(/[，,\n]/)
				.map((item) => item.trim())
				.filter(Boolean)
				.slice(0, 12)

			this.$emit('submit', {
				revisitRating: this.localRevisitRating,
				recommendedItems,
				visitedAt: this.visitedAt || new Date().toISOString()
			})
		}
	}
}
</script>

<style lang="scss" scoped>
.experience-sheet {
	display: flex;
	flex-direction: column;
	max-height: 85vh;
	background: linear-gradient(180deg, rgba(255, 253, 249, 0.99) 0%, rgba(250, 246, 239, 0.98) 100%);
	border-radius: 32rpx 32rpx 0 0;
	overflow: hidden;

	&--submitting {
		pointer-events: none;
		opacity: 0.6;
	}
}

.experience-sheet__handle {
	width: 72rpx;
	height: 8rpx;
	border-radius: 4rpx;
	background: rgba(92, 64, 51, 0.15);
	margin: 16rpx auto 0;
}

.experience-sheet__hero {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 32rpx 48rpx 24rpx;
	gap: 12rpx;
}

.experience-sheet__icon-shell {
	width: 88rpx;
	height: 88rpx;
	border-radius: 50%;
	background: rgba(138, 154, 91, 0.12);
	display: flex;
	align-items: center;
	justify-content: center;
	margin-bottom: 8rpx;
}

.experience-sheet__title {
	font-size: 36rpx;
	font-weight: 700;
	color: #5c4033;
	line-height: 1.3;
}

.experience-sheet__subtitle {
	font-size: 26rpx;
	color: rgba(92, 64, 51, 0.6);
	line-height: 1.4;
}

.experience-sheet__body {
	flex: 1;
	min-height: 0;
	overflow-y: auto;
	padding: 0 32rpx;
}

.experience-panel {
	display: flex;
	flex-direction: column;
	gap: 36rpx;
	padding-bottom: 32rpx;
}

.experience-field {
	display: flex;
	flex-direction: column;
	gap: 16rpx;
}

.experience-field__label {
	font-size: 28rpx;
	font-weight: 700;
	color: #5c4033;
	line-height: 1.4;
}

.experience-field__hint {
	font-size: 22rpx;
	color: rgba(92, 64, 51, 0.45);
	line-height: 1.4;
	margin-top: -8rpx;
}

.rating-stars {
	display: flex;
	align-items: center;
	gap: 12rpx;
	padding: 16rpx 0;
}

.rating-star {
	transition: transform 0.15s ease;

	&:active {
		transform: scale(0.88);
	}
}

.rating-quick-tags {
	display: flex;
	gap: 12rpx;
	flex-wrap: wrap;
}

.rating-quick-tag {
	padding: 14rpx 24rpx;
	border-radius: 16rpx;
	background: rgba(255, 250, 244, 0.95);
	border: 1px solid rgba(92, 64, 51, 0.12);
	transition: all 0.15s ease;

	&:active {
		transform: scale(0.96);
		background: rgba(244, 236, 226, 0.95);
	}
}

.rating-quick-tag__text {
	font-size: 24rpx;
	font-weight: 700;
	color: #745742;
	line-height: 1;
}

.rating-label {
	padding: 12rpx 0;
}

.rating-label__text {
	font-size: 26rpx;
	font-weight: 700;
	color: #f4a236;
	line-height: 1.4;
}

.experience-textarea {
	width: 100%;
	min-height: 160rpx;
	padding: 20rpx 24rpx;
	font-size: 26rpx;
	color: #41362d;
	line-height: 1.6;
	background: rgba(255, 253, 249, 0.98);
	border: 1px solid rgba(111, 86, 64, 0.12);
	border-radius: 20rpx;
	box-sizing: border-box;

	&::placeholder {
		color: rgba(92, 64, 51, 0.3);
	}
}

.experience-textarea__placeholder {
	color: rgba(92, 64, 51, 0.3);
}

.date-picker {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 20rpx 24rpx;
	border-radius: 20rpx;
	background: rgba(255, 253, 249, 0.98);
	border: 1px solid rgba(111, 86, 64, 0.12);
	transition: all 0.15s ease;

	&:active {
		background: rgba(244, 236, 226, 0.95);
	}
}

.date-picker__main {
	flex: 1;
	min-width: 0;
}

.date-picker__value {
	font-size: 26rpx;
	font-weight: 600;
	color: #41362d;
	line-height: 1.4;
}

.experience-sheet__footer {
	padding: 24rpx 32rpx;
	padding-bottom: calc(24rpx + env(safe-area-inset-bottom));
	background: rgba(255, 250, 244, 0.98);
	border-top: 1px solid rgba(92, 64, 51, 0.06);
}

.experience-sheet__actions {
	display: flex;
	gap: 16rpx;
}

.experience-action {
	flex: 1;
	min-height: 88rpx;
	border-radius: 22rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	transition: all 0.18s ease;

	&--secondary {
		background: rgba(255, 253, 249, 0.98);
		border: 1px solid rgba(92, 64, 51, 0.15);

		&:active:not(.experience-action--disabled) {
			transform: scale(0.98);
			background: rgba(244, 236, 226, 0.95);
		}
	}

	&--primary {
		background: linear-gradient(135deg, #5c4033 0%, #6f5544 100%);
		box-shadow: 0 12rpx 24rpx rgba(92, 64, 51, 0.22);

		&:active:not(.experience-action--disabled) {
			transform: scale(0.98);
			box-shadow: 0 6rpx 12rpx rgba(92, 64, 51, 0.18);
		}
	}

	&--disabled {
		opacity: 0.4;
		pointer-events: none;
	}
}

.experience-action__text {
	font-size: 28rpx;
	font-weight: 700;
	color: #5c4033;
	line-height: 1;

	&--primary {
		color: #fffaf3;
	}
}
</style>
