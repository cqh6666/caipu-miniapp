<template>
	<view v-if="visible" class="public-banner">
		<view class="public-banner__body">
			<up-icon name="eye" size="14" color="#7b6d62"></up-icon>
			<text class="public-banner__text">{{ bannerText }}</text>
		</view>
		<view class="public-banner__action" hover-class="public-banner__action--hover" hover-stay-time="80" @tap="emit('open')">
			<text class="public-banner__action-text">了解</text>
		</view>
	</view>
	<up-popup
		v-if="showExplain"
		:show="showExplain"
		mode="center"
		round="20"
		overlayOpacity="0.32"
		:closeOnClickOverlay="true"
		@close="emit('close')"
	>
		<view class="public-explain">
			<view class="public-explain__icon"><up-icon name="eye" size="22" color="#7b6d62"></up-icon></view>
			<text class="public-explain__title">这是一份只读分享</text>
			<text class="public-explain__desc">{{ explainBody }}</text>
			<view class="public-explain__action" hover-class="public-explain__action--hover" hover-stay-time="80" @tap="emit('close')">
				<text class="public-explain__action-text">我知道了</text>
			</view>
		</view>
	</up-popup>
</template>

<script setup>
defineProps({ visible: Boolean, bannerText: String, explainBody: String, showExplain: Boolean })
const emit = defineEmits(['open', 'close'])
</script>
