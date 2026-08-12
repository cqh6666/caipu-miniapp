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
		<view class="add-preview-panel" :class="{ 'add-preview-panel--parsing': isParsing }">
			<view class="add-preview-panel__handle"></view>
			<view class="add-preview-panel__header">
				<view class="add-preview-panel__heading">
					<text class="add-preview-panel__title">{{ title }}</text>
				</view>
				<view
					class="add-preview-panel__close"
					:class="{ 'add-preview-panel__close--disabled': isParsing }"
					@tap="handleClose"
				>
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<view v-if="isParsing" class="add-preview-panel__parsing-state">
				<view class="add-preview-panel__parsing-spinner">
					<up-loading-icon mode="circle" color="#745742" size="48"></up-loading-icon>
				</view>
				<text class="add-preview-panel__parsing-title">内容解析中</text>
				<text class="add-preview-panel__parsing-desc">{{ parsingText }}</text>
				<text v-if="parsingDuration > 3" class="add-preview-panel__parsing-hint">可能需要几秒，请稍等</text>
			</view>

			<scroll-view v-else class="add-preview-panel__body" scroll-y>
				<view class="add-preview-panel__main-entry" @tap="$emit('paste-request')">
					<view class="add-preview-panel__main-icon">
						<up-icon :name="entryIcon" size="32" color="#e67a3d"></up-icon>
					</view>
					<view class="add-preview-panel__main-content">
						<text class="add-preview-panel__main-title">{{ entryTitle }}</text>
						<text
							v-for="description in entryDescriptions"
							:key="description"
							class="add-preview-panel__main-desc"
						>
							{{ description }}
						</text>
					</view>
				</view>

				<view class="add-preview-panel__capabilities">
					<text class="add-preview-panel__capabilities-title">支持解析的平台</text>
					<view class="add-preview-panel__capabilities-list">
						<view
							v-for="capability in capabilities"
							:key="capability.key"
							class="add-preview-panel__capability-card"
						>
							<view
								class="add-preview-panel__capability-icon"
								:class="`add-preview-panel__capability-icon--${capability.key}`"
							>
								<up-icon :name="capability.icon" size="28" :color="capability.color"></up-icon>
							</view>
							<view class="add-preview-panel__capability-content">
								<text class="add-preview-panel__capability-title">{{ capability.title }}</text>
								<text class="add-preview-panel__capability-desc">{{ capability.description }}</text>
							</view>
						</view>
					</view>
				</view>

				<view class="add-preview-panel__manual-entry" @tap="$emit('manual-entry')">
					<up-icon name="edit-pen" size="18" color="#8a7d70"></up-icon>
					<text class="add-preview-panel__manual-text">{{ manualEntryText }}</text>
				</view>
			</scroll-view>

			<view v-if="!isParsing" class="add-preview-panel__footer">
				<view class="add-preview-panel__paste-input">
					<input
						:value="inputText"
						class="add-preview-panel__input-field"
						placeholder="粘贴链接..."
						placeholder-class="add-preview-panel__input-placeholder"
						confirm-type="send"
						@input="handleInput"
						@confirm="$emit('submit')"
					/>
					<view
						class="add-preview-panel__submit"
						:class="{ 'add-preview-panel__submit--disabled': !inputText.trim() }"
						@tap="$emit('submit')"
					>
						<up-icon name="arrow-right" size="20" color="#ffffff"></up-icon>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'AddPreviewPanel',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		isParsing: Boolean,
		parsingText: String,
		parsingDuration: Number,
		title: String,
		entryIcon: String,
		entryTitle: String,
		entryDescriptions: {
			type: Array,
			default: () => []
		},
		capabilities: {
			type: Array,
			default: () => []
		},
		manualEntryText: String,
		inputText: {
			type: String,
			default: ''
		}
	},
	emits: ['close', 'manual-entry', 'paste-request', 'input-change', 'submit'],
	methods: {
		handleClose() {
			if (this.isParsing) return
			this.$emit('close')
		},
		handleInput(event) {
			this.$emit('input-change', event.detail.value)
		}
	}
}
</script>

<style lang="scss" scoped>
@import './add-preview-panel.scss';
</style>
