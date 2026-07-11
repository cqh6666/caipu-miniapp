<template>
<!-- P2 修复：公开只读模式下「无成品图」菜谱整块跳过 Hero 区，
     避免被 .hero-card 的 min-height: 380rpx 撑出大块空白；
     无图时下方 detail-head 兜底分支接管标题与 meta 渲染。
     私有模式保留原 placeholder（含「上传成品图」CTA），仍需要 380rpx 撑住按钮 -->
<view
	v-if="displayRecipeImages.length || !isPublicView"
	class="hero-card"
	:class="{ 'hero-card--empty': !recipeImages.length, 'hero-card--with-overlay': displayRecipeImages.length > 0 }"
	@tap="handleHeroCardTap"
>
	<swiper
		v-if="displayRecipeImages.length"
		class="hero-card__swiper"
		:circular="displayRecipeImages.length > 1"
		:autoplay="false"
		:duration="280"
		@change="handleHeroSwiperChange"
	>
		<swiper-item v-for="(image, index) in displayRecipeImages" :key="image.cacheKey || `hero-image-${index}`">
			<image
				class="hero-card__image"
				:src="image.displayURL"
				mode="aspectFill"
				@error="handleRecipeImageError(image)"
			></image>
		</swiper-item>
	</swiper>
	<!-- H4：底部渐变蒙层，为压图标题/分页器提供读性 -->
	<view v-if="displayRecipeImages.length" class="hero-card__overlay" @tap.stop="handleHeroCardTap"></view>
	<!-- Hero 操作菜单：右下角 ⋯ 按钮，按当前位置动态装配「设为封面 / 添加 / 删除」 -->
	<!-- P2-D：公开只读模式下隐藏，避免接收者点击触发编辑路径 -->
	<view
		v-if="canShowHeroActionMenu && !isPublicView"
		class="hero-card__action"
		hover-class="hero-card__action--hover"
		hover-stay-time="80"
		@tap.stop="openHeroActionMenu"
	>
		<up-icon name="more-dot-fill" size="14" color="#ffffff"></up-icon>
	</view>
	<!-- H5：标题 + meta chips 压图（仅在有图时） -->
	<view v-if="displayRecipeImages.length" class="hero-card__title-block" @tap.stop="handleHeroCardTap">
		<view class="hero-card__meta">
			<view class="hero-card__chip hero-card__chip--meal">
				<text class="hero-card__chip-text">{{ mealLabel }}</text>
			</view>
			<view
				class="hero-card__chip"
				:class="recipe?.status === 'done' ? 'hero-card__chip--done' : 'hero-card__chip--wishlist'"
			>
				<text class="hero-card__chip-text">{{ statusLabel }}</text>
			</view>
			<view v-if="isPinned" class="hero-card__chip hero-card__chip--pin">
				<text class="hero-card__chip-text">已置顶</text>
			</view>
		</view>
		<text class="hero-card__title">{{ recipe.title }}</text>
	</view>
	<!-- H3：底部居中分页器；≤5 张用圆点 dots，>5 张用数字 chip -->
	<view v-if="displayRecipeImages.length > 1" class="hero-card__pager">
		<template v-if="displayRecipeImages.length <= 5">
			<view
				v-for="i in displayRecipeImages.length"
				:key="`hero-dot-${i}`"
				class="hero-card__dot"
				:class="{ 'hero-card__dot--active': i - 1 === heroImageIndex }"
			></view>
		</template>
		<view v-else class="hero-card__counter">
			<text class="hero-card__counter-text">{{ heroImageIndex + 1 }} / {{ displayRecipeImages.length }}</text>
		</view>
	</view>
	<view v-if="!displayRecipeImages.length && !isPublicView" class="hero-card__placeholder">
		<view class="hero-card__placeholder-mask"></view>
		<view class="hero-card__upload-action" :class="{ 'hero-card__upload-action--loading': isUploadingHeroImage }">
			<up-icon :name="recipeImages.length ? 'photo' : (isUploadingHeroImage ? 'reload' : 'plus')" size="18" color="#5b4a3b"></up-icon>
			<text class="hero-card__upload-action-text">{{ recipeImages.length ? '封面加载失败，点查看原图' : (isUploadingHeroImage ? '上传中...' : '上传成品图') }}</text>
		</view>
	</view>
</view>

<!-- H5：无图时回退到外部 detail-head 渲染标题 + meta；有图时压在图上 -->
<view v-if="!displayRecipeImages.length" class="detail-head">
	<view class="detail-head__meta">
		<view class="detail-chip detail-chip--meal">
			<text class="detail-chip__text">{{ mealLabel }}</text>
		</view>
		<view
			class="detail-chip"
			:class="recipe?.status === 'done' ? 'detail-chip--done' : 'detail-chip--wishlist'"
		>
			<text class="detail-chip__text">{{ statusLabel }}</text>
		</view>
		<view v-if="isPinned" class="detail-chip detail-chip--pin">
			<text class="detail-chip__text">已置顶</text>
		</view>
	</view>
	<text class="detail-title">{{ recipe.title }}</text>
	<view v-if="recipe.summary" class="detail-summary-card">
		<text class="detail-summary">{{ recipe.summary }}</text>
	</view>
</view>
<!-- H5：有图时只渲染 summary（标题已经在图上） -->
<view v-else-if="recipe.summary" class="detail-head detail-head--summary-only">
	<view class="detail-summary-card">
		<text class="detail-summary">{{ recipe.summary }}</text>
	</view>
</view>
</template>

<script setup>
defineProps({
	displayRecipeImages: Array,
	isPublicView: Boolean,
	recipeImages: Array,
	canShowHeroActionMenu: Boolean,
	mealLabel: String,
	statusLabel: String,
	isPinned: Boolean,
	recipe: Object,
	heroImageIndex: Number,
	isUploadingHeroImage: Boolean
})
const emit = defineEmits(['tap', 'swiper-change', 'image-error', 'open-menu'])
function handleHeroCardTap() { emit('tap') }
function handleHeroSwiperChange(event) { emit('swiper-change', event) }
function handleRecipeImageError(image) { emit('image-error', image) }
function openHeroActionMenu() { emit('open-menu') }
</script>
