<template>
	<add-preview-panel
		:show="show"
		:is-parsing="isParsing"
		:parsing-text="parsingText"
		:parsing-duration="parsingDuration"
		title="添加打卡点"
		entry-icon="file-text"
		entry-title="点此粘贴分享链接"
		:entry-descriptions="entryDescriptions"
		:capabilities="capabilities"
		manual-entry-text="手动填写信息"
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
import { readClipboardText } from '../use-add-preview-flow'

const ENTRY_DESCRIPTIONS = ['支持大众点评、美团', '自动提取地点信息']
const CAPABILITIES = [
	{
		key: 'place',
		icon: 'map-fill',
		color: '#7c9070',
		title: '打卡地',
		description: '大众点评 / 美团'
	}
]

export default {
	name: 'AddLinkPreviewPanel',
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
		handleInputTextChange(value) {
			this.manualInputText = value
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
