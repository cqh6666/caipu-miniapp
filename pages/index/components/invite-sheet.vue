<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.22"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="invite-sheet">
			<view class="invite-sheet__header">
				<view class="invite-sheet__heading">
					<text class="invite-sheet__title">邀请成员</text>
					<text class="invite-sheet__subtitle">{{ subtitle }}</text>
				</view>
				<view class="invite-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<scroll-view class="invite-sheet__body" scroll-y>
				<view v-if="isPreparing" class="invite-sheet__state">
					<up-icon name="reload" size="22" color="#8d8074"></up-icon>
					<text class="invite-sheet__state-title">正在生成邀请</text>
					<text class="invite-sheet__state-desc">{{ preparingText }}</text>
				</view>

				<view v-else-if="invite" class="invite-sheet__stack">
					<view class="invite-sheet__code-card" @tap="$emit('copy-code')">
						<text class="invite-sheet__code-label">邀请码</text>
						<text class="invite-sheet__code">{{ formattedCode }}</text>
					</view>

					<text class="invite-sheet__meta-line">{{ metaLine }}</text>
				</view>

				<view v-else class="invite-sheet__state">
					<up-icon name="info-circle" size="22" color="#8d8074"></up-icon>
					<text class="invite-sheet__state-title">暂时没拿到邀请码</text>
					<text class="invite-sheet__state-desc">可以稍后重试，或直接重新生成一组新的邀请码。</text>
				</view>
			</scroll-view>

			<view class="invite-sheet__footer">
				<view class="invite-sheet__button-group">
					<button
						v-if="showShareAction"
						class="invite-sheet__action invite-sheet__action--primary"
						:class="{ 'invite-sheet__action--disabled': !invite || isPreparing }"
						open-type="share"
						:disabled="!invite || isPreparing"
					>
						<view class="invite-sheet__action-inner">
							<up-icon name="share" size="16" color="#ffffff"></up-icon>
							<text class="invite-sheet__action-text invite-sheet__action-text--primary">发送给微信好友</text>
						</view>
					</button>
					<button
						class="invite-sheet__action"
						:class="[
							showShareAction ? 'invite-sheet__action--secondary' : 'invite-sheet__action--primary',
							{ 'invite-sheet__action--disabled': !invite || isPreparing }
						]"
						:disabled="!invite || isPreparing"
						@tap="$emit('copy-code')"
					>
						<text
							class="invite-sheet__action-text"
							:class="
								showShareAction
									? 'invite-sheet__action-text--secondary'
									: 'invite-sheet__action-text--primary'
							"
						>
							{{ isPreparing ? '生成中...' : copied ? '邀请码已复制' : '复制邀请码' }}
						</text>
					</button>
				</view>
				<view class="invite-sheet__utility">
					<view class="invite-sheet__utility-link" @tap="$emit('regenerate')">
						<up-icon name="reload" size="14" color="#7f7265"></up-icon>
						<text class="invite-sheet__utility-text">重新生成邀请码</text>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'InviteSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		subtitle: {
			type: String,
			default: ''
		},
		isPreparing: {
			type: Boolean,
			default: false
		},
		invite: {
			type: Object,
			default: null
		},
		preparingText: {
			type: String,
			default: ''
		},
		formattedCode: {
			type: String,
			default: '--'
		},
		metaLine: {
			type: String,
			default: '--'
		},
		copied: {
			type: Boolean,
			default: false
		},
		showShareAction: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'copy-code', 'regenerate']
}
</script>

<style lang="scss" scoped>
@import './invite-sheet.scss';
</style>
