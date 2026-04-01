<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.22"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="invite-code-sheet">
			<view class="invite-code-sheet__header">
				<view class="invite-code-sheet__heading">
					<text class="invite-code-sheet__title">输入邀请码</text>
					<text class="invite-code-sheet__subtitle">让朋友把邀请码发给你，输入后就能进入邀请页确认加入。</text>
				</view>
				<view class="invite-code-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<view class="invite-code-sheet__body">
				<input
					:value="value"
					class="invite-code-sheet__input"
					placeholder="输入邀请码，例如 AB12-CD34"
					placeholder-class="invite-code-sheet__placeholder"
					maxlength="9"
					@input="$emit('input-code', $event)"
				/>
				<text class="invite-code-sheet__hint">输入后会先打开邀请页，再由你确认是否加入。</text>
			</view>

			<view class="invite-code-sheet__footer">
				<view class="sheet-action" @tap="$emit('close')">
					<text class="sheet-action__text">取消</text>
				</view>
				<view
					class="sheet-action sheet-action--primary"
					:class="{ 'sheet-action--disabled': !canSubmit }"
					@tap="$emit('submit')"
				>
					<text class="sheet-action__text sheet-action__text--primary">继续</text>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'InviteCodeSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		value: {
			type: String,
			default: ''
		},
		canSubmit: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'input-code', 'submit']
}
</script>

<style lang="scss" scoped>
@import './invite-sheet.scss';
@import './sheet-action.scss';
</style>
