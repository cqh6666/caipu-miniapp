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
					<text class="panel__title">添加打卡点</text>
				</view>
				<view class="panel__close" :class="{ 'panel__close--disabled': isParsing }" @tap="handleClose">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<!-- 内容解析中状态 -->
			<view v-if="isParsing" class="parsing-state">
				<view class="parsing-state__spinner">
					<up-loading-icon mode="circle" color="#745742" size="48"></up-loading-icon>
				</view>
				<text class="parsing-state__title">内容解析中</text>
				<text class="parsing-state__desc">{{ parsingText }}</text>
				<text v-if="parsingDuration > 3" class="parsing-state__hint">可能需要几秒，请稍等</text>
			</view>

			<!-- 主入口区域 -->
			<scroll-view v-else class="panel__body" scroll-y>
				<view class="main-entry" @tap="handlePasteLink">
					<view class="main-entry__icon">
						<up-icon name="file-text" size="32" color="#e67a3d"></up-icon>
					</view>
					<view class="main-entry__content">
						<text class="main-entry__title">点此粘贴分享链接</text>
						<text class="main-entry__desc">支持大众点评、美团</text>
						<text class="main-entry__desc">自动提取地点信息</text>
					</view>
				</view>

				<view class="capabilities">
					<text class="capabilities__title">支持解析的平台</text>
					<view class="capabilities__list">
						<view class="capability-card">
							<view class="capability-card__icon capability-card__icon--place">
								<up-icon name="map-fill" size="28" color="#7c9070"></up-icon>
							</view>
							<view class="capability-card__content">
								<text class="capability-card__title">打卡地</text>
								<text class="capability-card__desc">大众点评 / 美团</text>
							</view>
						</view>
					</view>
				</view>

				<view class="manual-entry" @tap="handleManualEntry">
					<up-icon name="edit-pen" size="18" color="#8a7d70"></up-icon>
					<text class="manual-entry__text">手动填写信息</text>
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
import { readClipboardText } from '../use-add-preview-flow'

export default {
	name: 'AddLinkPreviewPanel',
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
			if (text) {
				this.$emit('paste', text)
				return
			}

			uni.showToast({
				title: '未读取到剪贴板，请粘贴到输入框',
				icon: 'none',
				duration: 2000
			})
		},
		handleManualInputSubmit() {
			const text = this.manualInputText.trim()
			if (!text) {
				uni.showToast({
					title: '请输入分享链接或文案',
					icon: 'none'
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
