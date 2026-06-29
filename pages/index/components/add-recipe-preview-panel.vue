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
import { isPreviewTimeoutError, previewAddLink } from '../../../utils/add-preview-api'
import { getCurrentKitchenId } from '../../../utils/auth'

export default {
	name: 'AddRecipePreviewPanel',
	props: {
		show: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'manual-entry', 'parse-result', 'preview-timeout'],
	data() {
		return {
			isParsing: false,
			parsingStage: 'extracting',
			parsingDuration: 0,
			parsingTimer: null,
			manualInputText: ''
		}
	},
	computed: {
		parsingText() {
			const stages = {
				extracting: '正在提取链接信息...',
				identifying: '正在识别平台类型...',
				fetching: '正在获取菜谱内容...',
				parsing: '正在解析食材和步骤...',
				finalizing: '正在整理图片和补充信息...'
			}
			return stages[this.parsingStage] || '正在处理...'
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
			const text = await this.readClipboardText()
			if (text) {
				this.startParsing(text)
				return
			}

			uni.showToast({
				title: '未读取到剪贴板，请粘贴到输入框',
				icon: 'none',
				duration: 2000
			})
		},
		readClipboardText() {
			return new Promise((resolve) => {
				uni.getClipboardData({
					success: (result) => {
						resolve(String(result?.data || '').trim())
					},
					fail: (error) => {
						console.warn('读取剪贴板失败:', error)
						resolve('')
					}
				})
			})
		},
		handleManualInputSubmit() {
			const text = this.manualInputText.trim()
			if (!text) {
				uni.showToast({
					title: '请输入菜谱链接',
					icon: 'none'
				})
				return
			}

			this.startParsing(text)
			this.manualInputText = ''
		},
		startParsing(text) {
			this.isParsing = true
			this.parsingStage = 'extracting'
			this.parsingDuration = 0

			// 启动计时器
			this.parsingTimer = setInterval(() => {
				this.parsingDuration++
			}, 1000)

			// 调用解析流程
			this.parseRecipeLink(text)
		},
		async parseRecipeLink(text) {
			try {
				const kitchenId = Number(getCurrentKitchenId()) || 0
				if (!kitchenId) {
					this.finishParsing({ status: 'failed', message: '请先完成空间同步' })
					return
				}

				await this.sleep(200)
				this.parsingStage = 'identifying'

				await this.sleep(200)
				this.parsingStage = 'fetching'

				const result = await previewAddLink(kitchenId, {
					text,
					city: '佛山',
					limit: 3
				})

				this.parsingStage = 'parsing'
				await this.sleep(240)
				this.parsingStage = 'finalizing'
				await this.sleep(160)
				this.finishParsing(result || { status: 'failed', message: '解析结果为空' })
			} catch (error) {
				console.error('解析失败:', error)
				// 解析请求超时：不再只弹失败，转交父级用「链接+猜测标题」建占位菜谱走后台补全。
				// 仅菜谱入口启用此兜底，避免把内容误存（地点链路走 add-link-preview-panel，不走这里）。
				if (isPreviewTimeoutError(error)) {
					this.stopParsing()
					this.$emit('preview-timeout', { text })
					this.$emit('close')
					return
				}
				this.finishParsing({
					status: 'failed',
					message: error.message || '解析失败，请手动填写'
				})
			}
		},
		stopParsing() {
			clearInterval(this.parsingTimer)
			this.parsingTimer = null

			this.isParsing = false
			this.parsingStage = 'extracting'
			this.parsingDuration = 0
		},
		finishParsing(result) {
			this.stopParsing()

			if (result.status === 'failed') {
				uni.showToast({
					title: result.message || '解析失败',
					icon: 'none',
					duration: 2000
				})
			} else {
				// 向父组件传递解析结果
				this.$emit('parse-result', result)
				this.$emit('close')
			}
		},
		sleep(ms) {
			return new Promise(resolve => setTimeout(resolve, ms))
		}
	},
	beforeUnmount() {
		if (this.parsingTimer) {
			clearInterval(this.parsingTimer)
		}
	}
}
</script>

<style lang="scss" scoped>
.panel {
	position: relative;
	background: #fffcf8;
	border-radius: 32rpx 32rpx 0 0;
	max-height: 80vh;
	display: flex;
	flex-direction: column;

	&--parsing {
		min-height: 520rpx;
	}
}

.panel__handle {
	width: 56rpx;
	height: 8rpx;
	background: rgba(138, 125, 112, 0.24);
	border-radius: 4rpx;
	margin: 16rpx auto 0;
}

.panel__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 24rpx 24rpx 20rpx;
}

.panel__heading {
	flex: 1;
	min-width: 0;
}

.panel__title {
	font-size: 32rpx;
	font-weight: 700;
	color: #41362d;
	line-height: 1.3;
	font-family: 'Playfair Display', Georgia, 'Times New Roman', serif;
}

.panel__close {
	width: 56rpx;
	height: 56rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	border-radius: 50%;
	background: rgba(138, 125, 112, 0.08);

	&--disabled {
		opacity: 0.4;
		pointer-events: none;
	}
}

.panel__body {
	flex: 1;
	overflow-y: auto;
	padding: 0 24rpx 32rpx;
	box-sizing: border-box;
}

/* 智能解析中状态 */
.parsing-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 80rpx 24rpx;
	gap: 24rpx;
}

.parsing-state__spinner {
	margin-bottom: 16rpx;
}

.parsing-state__title {
	font-size: 32rpx;
	font-weight: 700;
	color: #41362d;
}

.parsing-state__desc {
	font-size: 26rpx;
	color: #8a7d70;
	line-height: 1.5;
}

.parsing-state__hint {
	margin-top: 8rpx;
	font-size: 24rpx;
	color: #b5a89a;
}

/* 主入口 */
.main-entry {
	display: flex;
	align-items: center;
	gap: 24rpx;
	padding: 40rpx 32rpx;
	background: #ffffff;
	border: 2px solid rgba(230, 122, 61, 0.15);
	border-radius: 24rpx;
	margin-bottom: 32rpx;
	transition: all 0.3s ease;

	&:active {
		background: rgba(230, 122, 61, 0.02);
		transform: scale(0.98);
	}
}

.main-entry__icon {
	flex-shrink: 0;
	width: 88rpx;
	height: 88rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(230, 122, 61, 0.08);
	border-radius: 50%;
	border: 2px solid rgba(230, 122, 61, 0.15);
}

.main-entry__content {
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 8rpx;
}

.main-entry__title {
	font-size: 32rpx;
	font-weight: 700;
	color: #41362d;
	line-height: 1.3;
}

.main-entry__desc {
	font-size: 24rpx;
	color: #a08775;
	line-height: 1.5;
}

/* 能力说明 */
.capabilities {
	margin-bottom: 32rpx;
}

.capabilities__title {
	display: block;
	font-size: 24rpx;
	font-weight: 500;
	color: #a08775;
	margin-bottom: 20rpx;
	padding-left: 4rpx;
}

.capabilities__list {
	display: flex;
	flex-direction: column;
	gap: 12rpx;
}

.capability-card {
	flex: 1;
	display: flex;
	align-items: center;
	gap: 16rpx;
	padding: 20rpx;
	background: #ffffff;
	border: 1px solid rgba(160, 135, 117, 0.12);
	border-radius: 20rpx;
	box-sizing: border-box;
	min-width: 0;
}

.capability-card__icon {
	flex-shrink: 0;
	width: 56rpx;
	height: 56rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	border-radius: 50%;

	&--xiaohongshu {
		background: rgba(255, 36, 66, 0.1);
	}

	&--bilibili {
		background: rgba(0, 161, 214, 0.1);
	}
}

.capability-card__content {
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 6rpx;
	min-width: 0;
	overflow: hidden;
}

.capability-card__title {
	font-size: 26rpx;
	font-weight: 700;
	color: #41362d;
	line-height: 1.3;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.capability-card__desc {
	font-size: 22rpx;
	color: #a08775;
	line-height: 1.4;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

/* 手动填写入口 */
.manual-entry {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 10rpx;
	padding: 24rpx;
	background: transparent;
	border: none;
	border-radius: 16rpx;
	transition: all 0.2s ease;

	&:active {
		background: rgba(138, 125, 112, 0.05);
	}
}

.manual-entry__text {
	font-size: 26rpx;
	font-weight: 500;
	color: #8a7d70;
}

/* 底部输入框 */
.panel__footer {
	padding: 16rpx 24rpx;
	padding-bottom: calc(16rpx + env(safe-area-inset-bottom));
	background: #ffffff;
	border-top: 1px solid rgba(160, 135, 117, 0.08);
}

.paste-input {
	display: flex;
	align-items: center;
	gap: 12rpx;
	background: #f7f5f1;
	border-radius: 48rpx;
	padding: 8rpx 8rpx 8rpx 24rpx;
}

.paste-input__field {
	flex: 1;
	height: 64rpx;
	font-size: 26rpx;
	color: #41362d;
	background: transparent;
	border: none;
	outline: none;
}

.paste-input__placeholder {
	color: #b5a89a;
}

.paste-input__submit {
	flex-shrink: 0;
	width: 64rpx;
	height: 64rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	background: #745742;
	border-radius: 50%;
	transition: all 0.2s ease;

	&:active {
		background: #5c4033;
		transform: scale(0.95);
	}

	&--disabled {
		background: #d4c8bb;
		opacity: 0.6;
		pointer-events: none;
	}
}
</style>
