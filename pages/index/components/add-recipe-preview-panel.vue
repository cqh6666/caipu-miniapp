<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.34"
		:closeOnClickOverlay="!isParsing"
		:safeAreaInsetBottom="false"
		@close="handleClose"
	>
		<view class="panel" :class="{ 'panel--parsing': isParsing }">
			<view class="panel__handle"></view>
			<view class="panel__header">
				<view class="panel__heading">
					<text class="panel__title">添加菜品</text>
				</view>
				<view class="panel__close" :class="{ 'panel__close--disabled': isParsing }" @tap="handleClose">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<!-- 智能解析中状态 -->
			<view v-if="isParsing" class="parsing-state">
				<view class="parsing-state__spinner">
					<up-loading-icon mode="circle" color="#745742" size="48"></up-loading-icon>
				</view>
				<text class="parsing-state__title">智能解析中</text>
				<text class="parsing-state__desc">{{ parsingText }}</text>
				<text v-if="parsingDuration > 3" class="parsing-state__hint">可能需要几秒，请稍等</text>
			</view>

			<!-- 主入口区域 -->
			<scroll-view v-else class="panel__body" scroll-y>
				<view class="main-entry" @tap="handlePasteLink">
					<view class="main-entry__icon">
						<up-icon name="grid-fill" size="32" color="#e67a3d"></up-icon>
					</view>
					<view class="main-entry__content">
						<text class="main-entry__title">点此粘贴菜谱链接</text>
						<text class="main-entry__desc">支持小红书、B站</text>
						<text class="main-entry__desc">自动提取菜名、食材、步骤</text>
					</view>
				</view>

				<view class="capabilities">
					<text class="capabilities__title">支持解析的平台</text>
					<view class="capabilities__list">
						<view class="capability-card">
							<view class="capability-card__icon capability-card__icon--xiaohongshu">
								<up-icon name="star-fill" size="28" color="#ff2442"></up-icon>
							</view>
							<view class="capability-card__content">
								<text class="capability-card__title">小红书</text>
								<text class="capability-card__desc">图文菜谱 / 视频教程</text>
							</view>
						</view>
						<view class="capability-card">
							<view class="capability-card__icon capability-card__icon--bilibili">
								<up-icon name="play-circle-fill" size="28" color="#00a1d6"></up-icon>
							</view>
							<view class="capability-card__content">
								<text class="capability-card__title">B站</text>
								<text class="capability-card__desc">视频 + AI字幕提取</text>
							</view>
						</view>
					</view>
				</view>

				<view class="manual-entry" @tap="handleManualEntry">
					<up-icon name="edit-pen" size="18" color="#8a7d70"></up-icon>
					<text class="manual-entry__text">手动填写菜谱信息</text>
				</view>
			</scroll-view>

			<!-- 底部输入框 -->
			<view v-if="!isParsing" class="panel__footer">
				<view class="paste-input">
					<input
						v-model="manualInputText"
						class="paste-input__field"
						placeholder="粘贴链接..."
						placeholder-class="paste-input__placeholder"
						confirm-type="send"
						@confirm="handleManualInputSubmit"
					/>
					<view
						class="paste-input__submit"
						:class="{ 'paste-input__submit--disabled': !manualInputText.trim() }"
						@tap="handleManualInputSubmit"
					>
						<up-icon name="arrow-right" size="20" color="#ffffff"></up-icon>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
import { hasParseableShareHint, readClipboardText } from '../use-add-preview-flow'

export default {
	name: 'AddRecipePreviewPanel',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		isParsing: Boolean,
		parsingText: String,
		parsingDuration: Number
	},
	emits: ['close', 'manual-entry', 'paste'],
	data() {
		return { manualInputText: '' }
	},
	methods: {
		handleClose() {
			if (this.isParsing) return
			this.$emit('close')
		},
		handleManualEntry() {
			this.$emit('manual-entry')
			this.$emit('close')
		},
		async handlePasteLink() {
			const text = await readClipboardText(uni, (error) => {
				console.warn('读取剪贴板失败:', error)
			})
			if (!text) {
				uni.showToast({
					title: '没读到剪贴板内容，复制链接后再试',
					icon: 'none',
					duration: 2000
				})
				return
			}

			if (!hasParseableShareHint(text)) {
				uni.showToast({
					title: '剪贴板里没找到链接，复制小红书 / B 站分享后再试',
					icon: 'none',
					duration: 2400
				})
				return
			}

			this.$emit('paste', text)
		},
		// 本地轻量预判：含 http(s) 链接或平台关键词才放行，避免空剪贴板 / 纯文字
		// 也走一遍 loading 再被后端笼统驳回。含链接一律放行，交后端精判，防误杀。
		handleManualInputSubmit() {
			const text = this.manualInputText.trim()
			if (!text) {
				uni.showToast({
					title: '请粘贴菜谱链接',
					icon: 'none'
				})
				return
			}

			if (!hasParseableShareHint(text)) {
				uni.showToast({
					title: '没识别到链接，请粘贴小红书 / B 站分享文案',
					icon: 'none',
					duration: 2400
				})
				return
			}

			this.$emit('paste', text)
			this.manualInputText = ''
		}
	}
}
</script>

<style lang="scss" scoped>
@import './add-preview-panel.scss';
</style>
