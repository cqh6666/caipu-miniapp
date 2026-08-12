<template>
	<add-preview-panel
		:show="show"
		:is-parsing="isParsing"
		:parsing-text="parsingText"
		:parsing-duration="parsingDuration"
		title="添加菜品"
		entry-icon="grid-fill"
		entry-title="点此粘贴菜谱链接"
		:entry-descriptions="entryDescriptions"
		:capabilities="capabilities"
		manual-entry-text="手动填写菜谱信息"
		:input-text="manualInputText"
		@close="handleClose"
		@manual-entry="handleManualEntry"
		@paste-request="handlePasteLink"
		@input-change="handleInputTextChange"
		@submit="handleManualInputSubmit"
	></add-preview-panel>
</template>

<script>
import AddPreviewPanel from './add-preview-panel.vue'
import { hasParseableShareHint, readClipboardText } from '../use-add-preview-flow'

const ENTRY_DESCRIPTIONS = ['支持小红书、B站', '自动提取菜名、食材、步骤']
const CAPABILITIES = [
	{
		key: 'xiaohongshu',
		icon: 'star-fill',
		color: '#ff2442',
		title: '小红书',
		description: '图文菜谱 / 视频教程'
	},
	{
		key: 'bilibili',
		icon: 'play-circle-fill',
		color: '#00a1d6',
		title: 'B站',
		description: '视频字幕提取 / 菜谱整理'
	}
]

export default {
	name: 'AddRecipePreviewPanel',
	components: { AddPreviewPanel },
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
		return {
			manualInputText: '',
			entryDescriptions: ENTRY_DESCRIPTIONS,
			capabilities: CAPABILITIES
		}
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
		handleInputTextChange(value) {
			this.manualInputText = value
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
