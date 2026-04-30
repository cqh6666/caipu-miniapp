<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="36"
		overlayOpacity="0.24"
		:overlayStyle="assistantOverlayStyle"
		:closeOnClickOverlay="true"
		:safeAreaInsetBottom="false"
		@close="handleClose"
	>
		<view class="diet-assistant-sheet">
			<view class="diet-assistant-sheet__handle"></view>

			<view class="diet-assistant-sheet__header">
				<view class="assistant-brand">
					<view class="assistant-brand__icon">
						<image class="assistant-brand__icon-image" src="/static/icons/diet-assistant-logo.svg" mode="aspectFit" />
					</view>
					<view class="assistant-brand__copy">
						<text class="assistant-brand__title">饮食管家</text>
					</view>
				</view>

				<view class="diet-assistant-sheet__close" hover-class="diet-assistant-sheet__close--hover" @tap="handleClose">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<scroll-view
				class="diet-assistant-chat"
				scroll-y
				scroll-with-animation
				:scroll-into-view="scrollAnchor"
				:show-scrollbar="false"
			>
				<view class="assistant-date-pill">
					<text class="assistant-date-pill__text">今天的灵感台</text>
				</view>

				<view v-if="isLoadingHistory" class="assistant-history-status">
					<text class="assistant-history-status__text">正在同步历史...</text>
				</view>

				<view class="chat-row chat-row--assistant">
					<view class="chat-avatar chat-avatar--assistant">
						<image class="chat-avatar__image" src="/static/icons/diet-assistant-logo.svg" mode="aspectFit" />
					</view>
					<view class="chat-bubble chat-bubble--assistant">
						<text class="chat-bubble__text">晚上想吃什么、家里剩什么食材、或者想把链接先记下来，都可以从这里开始。</text>
						<view class="suggestion-grid">
							<view
								v-for="item in quickSuggestions"
								:key="item.title"
								class="suggestion-card"
								hover-class="suggestion-card--hover"
								@tap="applySuggestion(item.text)"
							>
								<text class="suggestion-card__title">{{ item.title }}</text>
								<text class="suggestion-card__desc">{{ item.desc }}</text>
							</view>
						</view>
					</view>
				</view>

				<view
					v-for="message in localMessages"
					:key="message.id"
					class="chat-row"
					:class="message.role === 'user' ? 'chat-row--user' : 'chat-row--assistant'"
				>
					<view v-if="message.role === 'assistant'" class="chat-avatar chat-avatar--assistant">
						<image class="chat-avatar__image" src="/static/icons/diet-assistant-logo.svg" mode="aspectFit" />
					</view>
					<view
						class="chat-bubble"
						:class="message.role === 'user' ? 'chat-bubble--user' : 'chat-bubble--assistant'"
					>
						<text
							class="chat-bubble__text"
							:class="{
								'chat-bubble__text--user': message.role === 'user',
								'chat-bubble__text--pending': message.pending && !message.text
							}"
						>
							{{ message.text || (message.pending ? '正在整理...' : '') }}
						</text>
					</view>
				</view>

				<view id="diet-assistant-bottom" class="diet-assistant-chat__bottom"></view>
			</scroll-view>

			<view class="diet-assistant-composer">
				<view class="composer-shortcuts">
					<view class="composer-shortcut" hover-class="composer-shortcut--hover" @tap="$emit('open-add-recipe')">
						<text class="composer-shortcut__text">记录菜谱</text>
					</view>
					<view class="composer-shortcut" hover-class="composer-shortcut--hover" @tap="applySuggestion('今晚不知道吃什么，帮我从美食库里挑一个方向')">
						<text class="composer-shortcut__text">今晚吃什么</text>
					</view>
					<view class="composer-shortcut" hover-class="composer-shortcut--hover" @tap="applySuggestion('我想把一个菜谱链接先整理成待记录内容')">
						<text class="composer-shortcut__text">整理链接</text>
					</view>
				</view>

				<view class="composer-box" :class="{ 'composer-box--active': isComposerFocused }">
					<input
						:value="draftMessage"
						class="composer-box__input"
						:placeholder="composerPlaceholder"
						placeholder-class="composer-box__placeholder"
						confirm-type="send"
						cursor-spacing="18"
						maxlength="200"
						@input="handleInput"
						@focus="isComposerFocused = true"
						@blur="isComposerFocused = false"
						@confirm="handleSend"
					/>
					<view
						class="composer-send"
						:class="{ 'composer-send--disabled': isSendDisabled }"
						hover-class="composer-send--hover"
						@tap="handleSend"
					>
						<image class="composer-send__icon" src="/static/icons/chat-send.svg" mode="aspectFit" />
					</view>
				</view>

				<view
					v-if="hasConversationStarted"
					class="composer-clear"
					hover-class="composer-clear--hover"
					@tap="clearConversationMessages"
				>
					<text class="composer-clear__text">清空会话记录</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
import {
	clearDietAssistantMessages,
	listDietAssistantMessages,
	streamDietAssistantChat
} from '../../../utils/diet-assistant-api'

export default {
	name: 'DietAssistantSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		initialPrompt: {
			type: String,
			default: ''
		}
	},
	emits: ['close', 'open-add-recipe'],
	data() {
		return {
			draftMessage: '',
			isComposerFocused: false,
			isStreaming: false,
			activeStream: null,
			activeAssistantMessageID: '',
			streamAbortExpected: false,
			isLoadingHistory: false,
			historyLoadSerial: 0,
			localMessages: [],
			messageSerial: 0,
			scrollAnchor: '',
			assistantOverlayStyle: {
				'background-color': 'rgba(68, 48, 35, 0.24)',
				'backdrop-filter': 'blur(18rpx) saturate(1.08)',
				'-webkit-backdrop-filter': 'blur(18rpx) saturate(1.08)'
			},
			quickSuggestions: [
				{
					title: '用剩菜找灵感',
					desc: '输入食材，先占位展示',
					text: '我家里有鸡蛋、番茄和一点青菜，可以做什么？'
				},
				{
					title: '安排一顿菜单',
					desc: '从美食库挑方向',
					text: '今晚想吃清爽一点，帮我整理一个菜单方向'
				},
				{
					title: '记录外部链接',
					desc: '后续可接链接解析',
					text: '我想把一个小红书/B站菜谱链接先记下来'
				}
			]
		}
	},
	computed: {
		hasConversationStarted() {
			return this.localMessages.length > 0
		},
		isSendDisabled() {
			return this.isStreaming || this.isLoadingHistory || !String(this.draftMessage || '').trim()
		},
		composerPlaceholder() {
			if (this.isLoadingHistory) return '正在同步历史...'
			if (this.isStreaming) return '饮食管家正在回复...'
			return '贴链接，或先写下想吃什么...'
		}
	},
	watch: {
		show(value) {
			if (value) {
				this.applyInitialPrompt()
				this.loadStoredMessages()
				this.bumpScrollAnchor()
			}
		},
		initialPrompt() {
			if (this.show) {
				this.applyInitialPrompt()
			}
		}
	},
	methods: {
		handleClose() {
			this.abortActiveStream()
			this.$emit('close')
		},
		handleInput(event) {
			this.draftMessage = String(event?.detail?.value || '')
		},
		applySuggestion(text = '') {
			this.draftMessage = String(text || '')
		},
		async loadStoredMessages() {
			if (this.isStreaming) return
			const serial = this.historyLoadSerial + 1
			this.historyLoadSerial = serial
			this.isLoadingHistory = true
			try {
				const items = await listDietAssistantMessages()
				if (serial !== this.historyLoadSerial || !this.show || this.isStreaming) return
				this.localMessages = this.mapStoredMessages(items)
				this.bumpScrollAnchor()
			} catch (error) {
				if (serial === this.historyLoadSerial && this.show) {
					uni.showToast({
						title: error?.message || '会话记录同步失败',
						icon: 'none'
					})
				}
			} finally {
				if (serial === this.historyLoadSerial) {
					this.isLoadingHistory = false
				}
			}
		},
		mapStoredMessages(items = []) {
			return (Array.isArray(items) ? items : [])
				.map((item, index) => {
					const role = String(item?.role || '').trim().toLowerCase()
					const text = String(item?.content || '').trim()
					if ((role !== 'user' && role !== 'assistant') || !text) return null
					return {
						id: `remote-${item?.id || `${item?.createdAt || 'message'}-${index}`}`,
						role,
						text,
						pending: false,
						contextExcluded: false
					}
				})
				.filter(Boolean)
		},
		applyInitialPrompt() {
			const text = String(this.initialPrompt || '').trim()
			if (!text || this.isStreaming) return
			this.draftMessage = text
		},
		handleSend() {
			const text = String(this.draftMessage || '').trim()
			if (!text || this.isStreaming) return

			const requestID = `diet-assistant-request-${Date.now()}-${this.messageSerial++}`
			const userID = `local-user-${Date.now()}-${this.messageSerial++}`
			const assistantID = `local-assistant-${Date.now()}-${this.messageSerial++}`
			const nextMessages = this.buildConversationMessages(text)

			this.localMessages.push({
				id: userID,
				requestID,
				role: 'user',
				text,
				contextExcluded: false
			})
			this.localMessages.push({
				id: assistantID,
				requestID,
				role: 'assistant',
				text: '',
				pending: true,
				contextExcluded: false
			})
			this.draftMessage = ''
			this.bumpScrollAnchor()
			this.startStreamResponse(assistantID, nextMessages)
		},
		buildConversationMessages(nextUserText = '') {
			const messages = this.localMessages
				.filter((message) => !message.pending && !message.transient && !message.contextExcluded && message.text)
				.map((message) => ({
					role: message.role,
					content: message.text
				}))
			messages.push({
				role: 'user',
				content: nextUserText
			})
			return messages
		},
		startStreamResponse(assistantID, messages) {
			this.isStreaming = true
			this.streamAbortExpected = false
			const stream = streamDietAssistantChat(messages, {
				onDelta: (delta) => {
					this.appendAssistantDelta(assistantID, delta)
				},
				onError: (error) => {
					this.failAssistantMessage(assistantID, error?.message || '饮食管家暂时不可用，请稍后再试。')
				}
			})
			this.activeStream = stream
			this.activeAssistantMessageID = assistantID
			const resetStreamState = () => {
				if (this.activeStream === stream) {
					this.activeStream = null
					this.activeAssistantMessageID = ''
					this.isStreaming = false
					this.streamAbortExpected = false
				}
			}
			stream.finished
				.then(() => {
					this.finishAssistantMessage(assistantID)
					resetStreamState()
				}, (error) => {
					if (this.streamAbortExpected) return
					this.failAssistantMessage(assistantID, error?.message || '饮食管家暂时不可用，请稍后再试。')
					resetStreamState()
				})
		},
		findMessage(id = '') {
			return this.localMessages.find((message) => message.id === id)
		},
		appendAssistantDelta(id = '', delta = '') {
			const message = this.findMessage(id)
			if (!message || !delta) return
			message.text = `${message.text || ''}${delta}`
			message.pending = true
			this.bumpScrollAnchor()
		},
		setAssistantMessage(id = '', text = '', pending = false, transient = false) {
			const message = this.findMessage(id)
			if (!message) return
			message.text = String(text || '')
			message.pending = pending
			message.transient = transient
			this.bumpScrollAnchor()
		},
		failAssistantMessage(id = '', text = '') {
			this.excludeMessageRequestFromContext(id)
			this.setAssistantMessage(id, text, false, true)
		},
		excludeMessageRequestFromContext(id = '') {
			const message = this.findMessage(id)
			if (!message?.requestID) return
			this.localMessages.forEach((item) => {
				if (item.requestID === message.requestID) {
					item.contextExcluded = true
				}
			})
		},
		finishAssistantMessage(id = '') {
			const message = this.findMessage(id)
			if (!message) return
			if (!String(message.text || '').trim()) {
				message.text = '我这边没有收到有效回复，可以换个问法再试一次。'
			}
			message.pending = false
			message.transient = false
			this.bumpScrollAnchor()
		},
		abortActiveStream() {
			if (!this.activeStream) return
			this.streamAbortExpected = true
			this.activeStream.abort?.()
			if (this.activeAssistantMessageID) {
				this.excludeMessageRequestFromContext(this.activeAssistantMessageID)
				const message = this.findMessage(this.activeAssistantMessageID)
				if (message?.pending && !String(message.text || '').trim()) {
					message.text = '已停止回复。'
					message.pending = false
					message.transient = true
				} else if (message) {
					message.pending = false
					message.transient = true
				}
			}
			this.activeStream = null
			this.activeAssistantMessageID = ''
			this.isStreaming = false
		},
		clearConversationMessages() {
			this.historyLoadSerial += 1
			this.abortActiveStream()
			this.localMessages = []
			this.activeStream = null
			this.activeAssistantMessageID = ''
			this.streamAbortExpected = false
			this.isStreaming = false
			this.isLoadingHistory = false
			this.bumpScrollAnchor()
			clearDietAssistantMessages().catch((error) => {
				uni.showToast({
					title: error?.message || '后端会话清空失败',
					icon: 'none'
				})
			})
		},
		bumpScrollAnchor() {
			this.$nextTick(() => {
				this.scrollAnchor = ''
				this.$nextTick(() => {
					this.scrollAnchor = 'diet-assistant-bottom'
				})
			})
		}
	}
}
</script>

<style lang="scss" scoped>
@import './diet-assistant-sheet.scss';
</style>
