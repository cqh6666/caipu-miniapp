<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="0"
		overlayOpacity="0.42"
		:closeOnClickOverlay="!isSubmitting"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="place-detail">
			<view class="place-detail__hero" :class="{ 'place-detail__hero--empty': !heroImages.length }">
				<swiper
					v-if="heroImages.length"
					class="place-detail__hero-swiper"
					:circular="heroImages.length > 1"
					:autoplay="heroImages.length > 1"
					:interval="3600"
					:duration="420"
					@change="handleHeroSwiperChange"
				>
					<swiper-item v-for="(image, index) in heroImages" :key="`${image}-${index}`">
						<image
							class="place-detail__hero-image"
							:src="image"
							mode="aspectFill"
							@tap.stop="$emit('preview-image', index)"
						></image>
					</swiper-item>
				</swiper>
				<view v-else class="place-detail__hero-placeholder">
					<up-icon :name="typeIcon" size="34" color="#8f7b68"></up-icon>
				</view>
				<view class="place-detail__hero-shade"></view>
				<view v-if="heroImages.length > 1" class="place-detail__hero-indicator">
					<view class="place-detail__hero-dots">
						<view
							v-for="(_, index) in heroImages"
							:key="`hero-dot-${index}`"
							class="place-detail__hero-dot"
							:class="{ 'place-detail__hero-dot--active': index === heroCurrentIndex }"
						></view>
					</view>
					<text class="place-detail__hero-count">{{ heroCurrentIndex + 1 }}/{{ heroImages.length }}</text>
				</view>
				<view class="place-detail__source-pill" @tap.stop>
					<text class="place-detail__source-pill-text">{{ sourceBadgeLabel }}</text>
				</view>
				<view class="place-detail__close" @tap.stop="$emit('close')">
					<up-icon name="close" size="21" color="#fffaf3"></up-icon>
				</view>
			</view>

			<scroll-view class="place-detail__scroll" scroll-y>
				<view class="place-detail__panel">
					<view class="place-detail__summary-row">
						<view class="place-detail__type-pill">
							<up-icon :name="typeIcon" size="16" color="#a99b91"></up-icon>
							<text class="place-detail__type-pill-text">{{ typeDisplayLabel }}</text>
						</view>
						<text v-if="place.price" class="place-detail__price">{{ place.price }}</text>
					</view>

					<text class="place-detail__name">{{ place.name || '未命名打卡点' }}</text>

					<!-- 重访意愿（仅去过且有评分） -->
					<view v-if="isVisited && hasRevisitRating" class="place-detail__rating-section">
						<view class="place-detail__rating-stars">
							<up-icon
								v-for="star in 5"
								:key="`detail-star-${star}`"
								:name="star <= place.revisitRating ? 'star-fill' : 'star'"
								size="24"
								:color="star <= place.revisitRating ? '#f4a236' : '#d4c4b8'"
							></up-icon>
						</view>
						<text class="place-detail__rating-label">{{ revisitRatingText }}</text>
						<text v-if="place.rating" class="place-detail__external-rating">{{ externalRatingDisplay }}</text>
					</view>

					<!-- 推荐项 -->
					<view v-if="isVisited && displayRecommendedItems.length" class="place-detail__recommended-section">
						<text class="place-detail__section-title">🍴 推荐必点</text>
						<view class="place-detail__recommended-grid">
							<view
								v-for="(item, index) in displayRecommendedItems"
								:key="`rec-${index}`"
								class="place-detail__recommended-item"
							>
								<text class="place-detail__recommended-item-text">{{ item }}</text>
							</view>
						</view>
					</view>

					<view v-if="displayTags.length" class="place-detail__tags">
						<view v-for="tag in displayTags" :key="tag" class="place-detail__tag">
							<text class="place-detail__tag-text">{{ tag }}</text>
						</view>
					</view>

					<!-- 电话卡片 -->
					<view
						v-if="place.phone"
						class="place-detail__phone-card"
						@longpress="handlePhoneLongPress"
						@tap="handlePhoneTap"
					>
						<view class="place-detail__card-icon">
							<up-icon name="phone" size="27" color="#a28f81"></up-icon>
						</view>
						<view class="place-detail__phone-main">
							<text class="place-detail__card-label">联系电话</text>
							<text class="place-detail__phone-text">{{ place.phone }}</text>
						</view>
						<up-icon name="arrow-right" size="17" color="#c6b9af"></up-icon>
					</view>

					<view
						class="place-detail__location-card"
						:class="{ 'place-detail__location-card--disabled': !canOpenLocation || isSubmitting }"
						@tap="$emit('open-location', place.id)"
					>
						<view class="place-detail__card-icon">
							<up-icon name="map" size="27" color="#a28f81"></up-icon>
						</view>
						<view class="place-detail__location-main">
							<text class="place-detail__card-label">所在位置</text>
							<text class="place-detail__location-text">{{ place.address || '还没有记录地址' }}</text>
						</view>
						<up-icon name="arrow-right" size="17" color="#c6b9af"></up-icon>
					</view>

					<view
						v-if="hasSourceUrl"
						class="place-detail__source-card"
						:class="{ 'place-detail__source-card--disabled': isSubmitting }"
						@tap="$emit('open-source', place.id)"
					>
						<view class="place-detail__card-icon place-detail__card-icon--source">
							<up-icon class="place-detail__external-icon" name="arrow-upward" size="24" color="#9b8a7f"></up-icon>
						</view>
						<view class="place-detail__source-main">
							<text class="place-detail__source-title">查看店铺原网页</text>
							<text class="place-detail__source-desc">{{ sourceActionDesc }}</text>
						</view>
						<up-icon name="arrow-right" size="17" color="#c6b9af"></up-icon>
					</view>

					<view v-if="place.note" class="place-detail__note-card">
						<text class="place-detail__note-label">备注</text>
						<text class="place-detail__note-text">{{ place.note }}</text>
					</view>

					<!-- 就餐提示 -->
					<view v-if="place.diningTips" class="place-detail__info-card">
						<text class="place-detail__info-label">💡 就餐提示</text>
						<text class="place-detail__info-text">{{ place.diningTips }}</text>
					</view>

					<!-- 适合场景 -->
					<view v-if="displayScenes.length" class="place-detail__info-card">
						<text class="place-detail__info-label">🏷️ 适合场景</text>
						<view class="place-detail__inline-tags">
							<view v-for="scene in displayScenes" :key="scene" class="place-detail__inline-tag">
								<text class="place-detail__inline-tag-text">{{ scene }}</text>
							</view>
						</view>
					</view>

					<!-- 时间信息 -->
					<view v-if="place.bestTime || place.duration" class="place-detail__info-card">
						<view v-if="place.bestTime" class="place-detail__info-row">
							<text class="place-detail__info-label">⏰ 最佳时段</text>
							<text class="place-detail__info-text">{{ place.bestTime }}</text>
						</view>
						<view v-if="place.duration" class="place-detail__info-row" :class="{ 'place-detail__info-row--mt': place.bestTime }">
							<text class="place-detail__info-label">⏱️ 就餐时长</text>
							<text class="place-detail__info-text">{{ place.duration }}</text>
						</view>
					</view>

					<!-- 陪同人推荐 -->
					<view v-if="displayCompanionTags.length" class="place-detail__info-card">
						<text class="place-detail__info-label">👥 陪同人推荐</text>
						<view class="place-detail__inline-tags">
							<view v-for="tag in displayCompanionTags" :key="tag" class="place-detail__inline-tag">
								<text class="place-detail__inline-tag-text">{{ tag }}</text>
							</view>
						</view>
					</view>

					<!-- 停车/交通 -->
					<view v-if="place.parkingNote" class="place-detail__info-card">
						<text class="place-detail__info-label">🚗 出行信息</text>
						<text class="place-detail__info-text">{{ place.parkingNote }}</text>
					</view>

					<!-- 数据来源 -->
					<view v-if="place.externalProvider" class="place-detail__source-info">
						<text class="place-detail__source-info-text">地点信息来自{{ externalProviderLabel }}</text>
						<text v-if="place.externalPoiId" class="place-detail__source-info-id">POI: {{ place.externalPoiId }}</text>
					</view>

					<view class="place-detail__manage-row">
						<view
							class="place-detail__manage-action"
							:class="{ 'place-detail__manage-action--disabled': isSubmitting }"
							@tap="$emit('edit', place.id)"
						>
							<up-icon name="edit-pen" size="15" color="#745742"></up-icon>
							<text class="place-detail__manage-text">编辑信息</text>
						</view>
						<view
							class="place-detail__manage-action place-detail__manage-action--danger"
							:class="{ 'place-detail__manage-action--disabled': isSubmitting }"
							@tap="$emit('delete', place.id)"
						>
							<up-icon name="trash" size="15" color="#a95549"></up-icon>
							<text class="place-detail__manage-text place-detail__manage-text--danger">删除</text>
						</view>
					</view>
				</view>
			</scroll-view>

			<view class="place-detail__bottom-bar">
				<button class="place-detail__share-btn" open-type="share">
					<up-icon name="share" size="24" color="#6a4736"></up-icon>
				</button>
				<view
					class="place-detail__primary"
					:class="{ 'place-detail__primary--submitting': isSubmitting }"
					@tap="$emit('toggle-status', place.id)"
				>
					<up-icon :name="primaryActionIcon" size="22" color="#fffaf3"></up-icon>
					<text class="place-detail__primary-text">{{ primaryActionText }}</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
const typeLabels = {
	food: '餐厅',
	attraction: '景点',
	other: '其他'
}

const typeDisplayLabels = {
	food: '美食探店',
	attraction: '景点打卡',
	other: '地点收藏'
}

const sourceLabels = {
	manual: '手动记录',
	dianping: '大众点评',
	meituan: '美团',
	other: '其他来源'
}

export default {
	name: 'PlaceDetailSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		place: {
			type: Object,
			default: () => ({})
		},
		isSubmitting: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'delete', 'edit', 'open-location', 'open-source', 'preview-image', 'toggle-status'],
	data() {
		return {
			heroCurrentIndex: 0
		}
	},
	watch: {
		show(next) {
			if (next) {
				this.heroCurrentIndex = 0
			}
		},
		'place.id'() {
			this.heroCurrentIndex = 0
		}
	},
	computed: {
		isVisited() {
			return this.place?.status === 'visited'
		},
		heroImages() {
			const images = Array.isArray(this.place?.imageUrls) ? this.place.imageUrls : []
			const seen = new Set()
			return images
				.map((image) => String(image || '').trim())
				.filter((image) => {
					if (!image || seen.has(image)) return false
					seen.add(image)
					return true
				})
		},
		typeIcon() {
			return this.place?.type === 'food' ? 'coupon' : 'map'
		},
		typeLabel() {
			return typeLabels[this.place?.type] || '其他'
		},
		typeDisplayLabel() {
			return typeDisplayLabels[this.place?.type] || this.typeLabel
		},
		sourceLabel() {
			return sourceLabels[this.place?.source] || '手动记录'
		},
		sourceBadgeLabel() {
			return this.place?.source && this.place.source !== 'manual'
				? `来自${this.sourceLabel}`
				: this.sourceLabel
		},
		sourceActionDesc() {
			if (this.place?.source === 'dianping') return '前往大众点评查看更多真实评价'
			if (this.place?.source === 'meituan') return '前往美团查看更多店铺信息'
			return '查看保存的来源链接'
		},
		displayTags() {
			return (Array.isArray(this.place?.tags) ? this.place.tags : []).slice(0, 6)
		},
		hasSourceUrl() {
			return !!String(this.place?.sourceUrl || '').trim()
		},
		canOpenLocation() {
			return !!(Number(this.place?.latitude) && Number(this.place?.longitude))
		},
		primaryActionText() {
			return this.isVisited ? '改回想去' : '去打卡'
		},
		primaryActionIcon() {
			return this.isVisited ? 'heart' : 'calendar'
		},
		hasRevisitRating() {
			return this.isVisited && Number(this.place?.revisitRating) > 0
		},
		revisitRatingText() {
			const rating = Number(this.place?.revisitRating || 0)
			if (rating === 5) return '非常推荐'
			if (rating === 4) return '值得再去'
			if (rating === 3) return '还可以'
			if (rating === 2) return '不太推荐'
			if (rating === 1) return '不推荐'
			return ''
		},
		externalRatingDisplay() {
			const rating = this.place?.rating || ''
			const provider = this.place?.externalProvider || ''
			if (!rating) return ''
			if (provider === 'amap') return `高德评分 ${rating}`
			return `外部评分 ${rating}`
		},
		displayRecommendedItems() {
			if (!this.isVisited) return []
			const items = this.place?.recommendedItems || []
			return Array.isArray(items) ? items.slice(0, 6) : []
		},
		displayScenes() {
			const items = this.place?.scenes || []
			return Array.isArray(items) ? items : []
		},
		displayCompanionTags() {
			const items = this.place?.companionTags || []
			return Array.isArray(items) ? items : []
		},
		externalProviderLabel() {
			if (this.place?.externalProvider === 'amap') return '高德地图'
			return '外部数据源'
		}
	},
	methods: {
		handleHeroSwiperChange(event = {}) {
			const current = Number(event?.detail?.current)
			if (!Number.isFinite(current)) return
			this.heroCurrentIndex = Math.max(0, Math.min(current, this.heroImages.length - 1))
		},
		handlePhoneTap() {
			if (!this.place?.phone) return
			uni.makePhoneCall({
				phoneNumber: this.place.phone
			})
		},
		handlePhoneLongPress() {
			if (!this.place?.phone) return
			uni.showActionSheet({
				itemList: ['拨打电话', '复制号码'],
				success: (res) => {
					if (res.tapIndex === 0) {
						uni.makePhoneCall({
							phoneNumber: this.place.phone
						})
					} else if (res.tapIndex === 1) {
						uni.setClipboardData({
							data: this.place.phone,
							success: () => {
								uni.showToast({
									title: '已复制',
									icon: 'success'
								})
							}
						})
					}
				}
			})
		}
	}
}
</script>

<style lang="scss" scoped>
.place-detail {
	position: relative;
	height: calc(100vh - 18rpx);
	max-height: calc(100vh - 18rpx);
	margin-top: 18rpx;
	background: #f6f1eb;
	border-radius: 44rpx 44rpx 0 0;
	overflow: hidden;
	display: flex;
	flex-direction: column;
	box-shadow: 0 -14rpx 36rpx rgba(53, 39, 27, 0.18);
}

.place-detail__hero {
	position: relative;
	height: 420rpx;
	flex-shrink: 0;
	overflow: hidden;
	background: #d8cabc;
}

.place-detail__hero--empty {
	background:
		radial-gradient(circle at 28% 22%, rgba(255, 255, 255, 0.72) 0%, rgba(255, 255, 255, 0) 40%),
		linear-gradient(145deg, #eadfD2 0%, #cfbcaa 100%);
}

.place-detail__hero-swiper,
.place-detail__hero-image,
.place-detail__hero-placeholder {
	width: 100%;
	height: 100%;
}

.place-detail__hero-placeholder {
	display: flex;
	align-items: center;
	justify-content: center;
}

.place-detail__hero-image {
	filter: saturate(0.92);
}

.place-detail__hero-shade {
	position: absolute;
	inset: 0;
	z-index: 1;
	pointer-events: none;
	background:
		linear-gradient(180deg, rgba(34, 26, 22, 0.16) 0%, rgba(34, 26, 22, 0.04) 44%, rgba(246, 241, 235, 0.92) 100%),
		linear-gradient(90deg, rgba(41, 31, 26, 0.1) 0%, rgba(41, 31, 26, 0) 42%);
}

.place-detail__hero-indicator {
	position: absolute;
	z-index: 2;
	pointer-events: none;
	right: 28rpx;
	bottom: 112rpx;
	min-width: 112rpx;
	height: 44rpx;
	padding: 0 16rpx;
	border-radius: 999rpx;
	background: rgba(43, 32, 26, 0.52);
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 12rpx;
	box-sizing: border-box;
}

.place-detail__hero-dots {
	display: flex;
	align-items: center;
	gap: 6rpx;
	max-width: 96rpx;
	overflow: hidden;
}

.place-detail__hero-dot {
	width: 8rpx;
	height: 8rpx;
	border-radius: 50%;
	background: rgba(255, 250, 243, 0.46);
	flex-shrink: 0;
	transition: width 0.18s ease, background 0.18s ease;
}

.place-detail__hero-dot--active {
	width: 22rpx;
	border-radius: 999rpx;
	background: #fffaf3;
}

.place-detail__hero-count {
	font-size: 21rpx;
	font-weight: 900;
	line-height: 1;
	color: #fffaf3;
}

.place-detail__source-pill {
	position: absolute;
	z-index: 3;
	top: 30rpx;
	left: 26rpx;
	min-height: 56rpx;
	padding: 0 22rpx;
	border-radius: 999rpx;
	background: rgba(255, 250, 244, 0.94);
	box-shadow:
		0 8rpx 18rpx rgba(54, 42, 33, 0.14),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.74);
	display: flex;
	align-items: center;
	justify-content: center;
}

.place-detail__source-pill-text {
	font-size: 24rpx;
	font-weight: 900;
	line-height: 1;
	color: #60483a;
}

.place-detail__close {
	position: absolute;
	z-index: 3;
	top: 28rpx;
	right: 26rpx;
	width: 66rpx;
	height: 66rpx;
	border-radius: 50%;
	background: rgba(57, 45, 38, 0.78);
	display: flex;
	align-items: center;
	justify-content: center;
	box-shadow: 0 8rpx 18rpx rgba(34, 24, 18, 0.22);
}

.place-detail__scroll {
	position: relative;
	z-index: 2;
	flex: 1;
	min-height: 0;
	margin-top: -82rpx;
}

.place-detail__panel {
	min-height: calc(100vh - 356rpx);
	padding: 40rpx 40rpx 178rpx;
	border-radius: 42rpx 42rpx 0 0;
	background: #fbf7f1;
	box-sizing: border-box;
}

.place-detail__summary-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 24rpx;
}

.place-detail__type-pill {
	min-height: 40rpx;
	padding: 0 18rpx;
	border-radius: 14rpx;
	background: #f3ede6;
	display: inline-flex;
	align-items: center;
	gap: 10rpx;
}

.place-detail__type-pill-text {
	font-size: 22rpx;
	font-weight: 900;
	line-height: 1;
	color: #a08f84;
}

.place-detail__price {
	flex-shrink: 0;
	font-size: 28rpx;
	font-weight: 900;
	line-height: 1;
	color: #c84000;
}

.place-detail__name {
	margin-top: 80rpx;
	font-family: "Songti SC", "STSong", "SimSun", "DejaVu Serif", serif;
	font-size: 42rpx;
	font-weight: 700;
	line-height: 1.38;
	color: #604236;
	word-break: break-all;
}

.place-detail__rating-section {
	display: flex;
	flex-direction: column;
	gap: 16rpx;
	margin-top: 32rpx;
	padding: 28rpx;
	border-radius: 28rpx;
	background: linear-gradient(135deg, rgba(244, 162, 54, 0.08) 0%, rgba(244, 162, 54, 0.04) 100%);
	border: 1px solid rgba(244, 162, 54, 0.15);
}

.place-detail__rating-stars {
	display: flex;
	align-items: center;
	gap: 8rpx;
}

.place-detail__rating-label {
	font-size: 28rpx;
	font-weight: 700;
	color: #f4a236;
	line-height: 1.3;
}

.place-detail__external-rating {
	font-size: 24rpx;
	color: rgba(92, 64, 51, 0.5);
	line-height: 1.3;
}

.place-detail__recommended-section {
	margin-top: 32rpx;
}

.place-detail__section-title {
	font-size: 28rpx;
	font-weight: 700;
	color: #5c4033;
	line-height: 1.4;
	margin-bottom: 20rpx;
	display: block;
}

.place-detail__recommended-grid {
	display: flex;
	flex-wrap: wrap;
	gap: 16rpx;
}

.place-detail__recommended-item {
	padding: 16rpx 24rpx;
	border-radius: 20rpx;
	background: rgba(255, 255, 255, 0.92);
	border: 1px solid rgba(104, 78, 62, 0.08);
	box-shadow: 0 8rpx 16rpx rgba(71, 54, 42, 0.06);
}

.place-detail__recommended-item-text {
	font-size: 26rpx;
	font-weight: 600;
	color: #5c4033;
	line-height: 1.3;
}

.place-detail__tags {
	display: flex;
	flex-wrap: wrap;
	gap: 14rpx;
	margin-top: 32rpx;
}

.place-detail__tag {
	min-height: 50rpx;
	padding: 0 22rpx;
	border-radius: 18rpx;
	background: #fffdf9;
	border: 1px solid rgba(104, 78, 62, 0.06);
	box-shadow:
		0 10rpx 18rpx rgba(71, 54, 42, 0.08),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
	display: flex;
	align-items: center;
	justify-content: center;
}

.place-detail__tag-text {
	font-size: 24rpx;
	font-weight: 900;
	line-height: 1;
	color: #8a776a;
}

.place-detail__location-card,
.place-detail__phone-card,
.place-detail__source-card,
.place-detail__note-card,
.place-detail__info-card {
	margin-top: 88rpx;
	border-radius: 36rpx;
	background: rgba(255, 255, 255, 0.98);
	border: 1px solid rgba(104, 78, 62, 0.06);
	box-shadow:
		0 12rpx 24rpx rgba(71, 54, 42, 0.07),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
}

.place-detail__location-card,
.place-detail__phone-card,
.place-detail__source-card {
	min-height: 132rpx;
	padding: 24rpx 26rpx;
	box-sizing: border-box;
	display: flex;
	align-items: center;
	gap: 22rpx;
}

.place-detail__location-card--disabled,
.place-detail__source-card--disabled {
	opacity: 0.58;
}

.place-detail__card-icon {
	width: 76rpx;
	height: 76rpx;
	border-radius: 24rpx;
	background: #f4f0eb;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	border: 1px solid rgba(104, 78, 62, 0.05);
}

.place-detail__card-icon--source {
	background: #f1ece6;
}

.place-detail__external-icon {
	transform: rotate(45deg);
}

.place-detail__location-main,
.place-detail__source-main {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	gap: 8rpx;
}

.place-detail__card-label {
	font-size: 23rpx;
	font-weight: 900;
	line-height: 1;
	color: #c0b4ad;
}

.place-detail__location-text {
	font-size: 29rpx;
	font-weight: 900;
	line-height: 1.35;
	color: #5e4035;
	word-break: break-all;
}

.place-detail__location-action {
	width: 58rpx;
	height: 58rpx;
	border-radius: 50%;
	background: #f5f1ed;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
}

.place-detail__source-card {
	margin-top: 24rpx;
}

.place-detail__source-title {
	font-size: 30rpx;
	font-weight: 900;
	line-height: 1.2;
	color: #5e4035;
}

.place-detail__source-desc {
	font-size: 24rpx;
	font-weight: 600;
	line-height: 1.25;
	color: #b2a59d;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.place-detail__note-card {
	margin-top: 32rpx;
	padding: 24rpx 26rpx;
	display: flex;
	flex-direction: column;
	gap: 12rpx;
}

.place-detail__note-label {
	font-size: 23rpx;
	font-weight: 900;
	line-height: 1;
	color: #b6aaa1;
}

.place-detail__note-text {
	font-size: 26rpx;
	line-height: 1.56;
	color: #6c5f56;
}

.place-detail__manage-row {
	display: flex;
	gap: 16rpx;
	margin-top: 58rpx;
}

.place-detail__manage-action {
	flex: 1;
	min-height: 74rpx;
	border-radius: 24rpx;
	background: rgba(255, 255, 255, 0.64);
	border: 1px solid rgba(104, 78, 62, 0.07);
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 8rpx;
}

.place-detail__manage-action--danger {
	background: rgba(255, 248, 246, 0.74);
	border-color: rgba(169, 85, 73, 0.12);
}

.place-detail__manage-action--disabled {
	opacity: 0.48;
}

.place-detail__manage-text {
	font-size: 23rpx;
	font-weight: 800;
	color: #745742;
}

.place-detail__manage-text--danger {
	color: #a95549;
}

.place-detail__bottom-bar {
	position: absolute;
	z-index: 5;
	left: 0;
	right: 0;
	bottom: 0;
	padding: 18rpx 40rpx calc(env(safe-area-inset-bottom) + 34rpx);
	background: linear-gradient(180deg, rgba(251, 247, 241, 0) 0%, rgba(251, 247, 241, 0.94) 28%, #fbf7f1 100%);
	display: flex;
	align-items: center;
	gap: 20rpx;
}

.place-detail__share-btn {
	width: 92rpx;
	height: 92rpx;
	padding: 0;
	margin: 0;
	border-radius: 32rpx;
	background: #fffefa;
	border: 0;
	box-shadow:
		0 10rpx 22rpx rgba(71, 54, 42, 0.1),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.76);
	display: flex;
	align-items: center;
	justify-content: center;
	line-height: 1;
}

.place-detail__share-btn::after {
	border: 0;
}

.place-detail__primary {
	flex: 1;
	height: 92rpx;
	border-radius: 34rpx;
	background: #684635;
	box-shadow: 0 12rpx 24rpx rgba(74, 48, 35, 0.2);
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 14rpx;
}

.place-detail__primary--submitting {
	opacity: 0.62;
}

.place-detail__primary-text {
	font-size: 30rpx;
	font-weight: 900;
	line-height: 1;
	color: #fffaf3;
}

.place-detail__phone-card {
	margin-top: 32rpx;
}

.place-detail__phone-main {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	gap: 8rpx;
}

.place-detail__phone-text {
	font-size: 29rpx;
	font-weight: 900;
	line-height: 1.35;
	color: #5e4035;
	word-break: break-all;
}

.place-detail__info-card {
	margin-top: 32rpx;
	padding: 24rpx 26rpx;
	display: flex;
	flex-direction: column;
	gap: 16rpx;
}

.place-detail__info-label {
	font-size: 26rpx;
	font-weight: 700;
	color: #5c4033;
	line-height: 1.4;
}

.place-detail__info-text {
	font-size: 26rpx;
	line-height: 1.56;
	color: #6c5f56;
}

.place-detail__info-row {
	display: flex;
	flex-direction: column;
	gap: 10rpx;
}

.place-detail__info-row--mt {
	margin-top: 20rpx;
}

.place-detail__inline-tags {
	display: flex;
	flex-wrap: wrap;
	gap: 12rpx;
	margin-top: 4rpx;
}

.place-detail__inline-tag {
	padding: 10rpx 18rpx;
	border-radius: 14rpx;
	background: rgba(244, 236, 226, 0.6);
	border: 1px solid rgba(92, 64, 51, 0.08);
}

.place-detail__inline-tag-text {
	font-size: 24rpx;
	font-weight: 600;
	color: #745742;
	line-height: 1;
}

.place-detail__source-info {
	margin-top: 48rpx;
	padding: 20rpx;
	display: flex;
	flex-direction: column;
	gap: 8rpx;
	align-items: center;
}

.place-detail__source-info-text {
	font-size: 22rpx;
	color: rgba(92, 64, 51, 0.4);
	line-height: 1.4;
}

.place-detail__source-info-id {
	font-size: 20rpx;
	font-family: monospace;
	color: rgba(92, 64, 51, 0.3);
	line-height: 1;
}
</style>
