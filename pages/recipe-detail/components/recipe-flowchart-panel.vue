<template>
	<view v-if="showImage" class="flowchart-panel">
		<view class="flowchart-panel__image-shell">
			<image
				class="flowchart-panel__image"
				:src="imageUrl"
				mode="widthFix"
				hover-class="flowchart-panel__image--active"
				hover-stay-time="80"
				@error="emit('image-error', $event)"
				@tap="emit('preview')"
			></image>
			<view class="flowchart-panel__image-shadow"></view>
			<view class="flowchart-panel__cta" hover-class="flowchart-panel__cta--active" hover-stay-time="80" @tap.stop="emit('open-viewer')">
				<text class="flowchart-panel__cta-text">横屏查看</text>
				<text class="flowchart-panel__cta-arrow">›</text>
			</view>
		</view>
	</view>
	<view v-if="showEmpty" class="flowchart-empty" :class="{ 'flowchart-empty--disabled': !canGenerate }">
		<view class="flowchart-empty__icon"><up-icon name="photo" size="24" color="#b08c72"></up-icon></view>
		<text class="flowchart-empty__title">还没生成步骤图</text>
		<text class="flowchart-empty__desc">{{ emptyText }}</text>
	</view>
</template>

<script setup>
defineProps({ showImage: Boolean, showEmpty: Boolean, imageUrl: String, canGenerate: Boolean, emptyText: String })
const emit = defineEmits(['image-error', 'preview', 'open-viewer'])
</script>
