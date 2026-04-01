<template>
	<up-popup
		:show="show"
		mode="bottom"
		round="32"
		overlayOpacity="0.22"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="profile-sheet">
			<view class="profile-sheet__header">
				<view class="profile-sheet__heading">
					<text class="profile-sheet__title">{{ title }}</text>
					<text class="profile-sheet__subtitle">{{ subtitle }}</text>
				</view>
				<view class="profile-sheet__close" @tap="$emit('close')">
					<up-icon name="close" size="18" color="#8a7d70"></up-icon>
				</view>
			</view>

			<form class="profile-sheet__body" @submit="$emit('submit', $event)">
				<button class="profile-sheet__avatar-button" open-type="chooseAvatar" @chooseavatar="$emit('choose-avatar', $event)">
					<image v-if="avatarPreview" class="profile-sheet__avatar-image" :src="avatarPreview" mode="aspectFill"></image>
					<view v-else class="profile-sheet__avatar-fallback">{{ avatarFallback }}</view>
				</button>
				<text class="profile-sheet__avatar-tip">点击头像选择你的微信头像</text>

				<view class="profile-sheet__field">
					<text class="profile-sheet__label">昵称</text>
					<input
						:value="nickname"
						class="profile-sheet__input"
						type="nickname"
						name="nickname"
						placeholder="输入昵称"
						placeholder-class="profile-sheet__placeholder"
						maxlength="20"
						@input="$emit('nickname-input', $event)"
					/>
					<text class="profile-sheet__hint">点击输入框时，键盘上方会出现微信昵称。</text>
				</view>

				<view class="profile-sheet__footer">
					<button class="sheet-action" form-type="reset" @tap="$emit('close')">
						<text class="sheet-action__text">{{ secondaryActionText }}</text>
					</button>
					<button
						class="sheet-action sheet-action--primary"
						:class="{ 'sheet-action--disabled': !canSubmit || isSubmitting }"
						form-type="submit"
						:disabled="!canSubmit || isSubmitting"
					>
						<text class="sheet-action__text sheet-action__text--primary">
							{{ isSubmitting ? '保存中...' : '保存资料' }}
						</text>
					</button>
				</view>
			</form>
		</view>
	</up-popup>
</template>

<script>
export default {
	name: 'ProfileSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		title: {
			type: String,
			default: ''
		},
		subtitle: {
			type: String,
			default: ''
		},
		avatarPreview: {
			type: String,
			default: ''
		},
		avatarFallback: {
			type: String,
			default: ''
		},
		nickname: {
			type: String,
			default: ''
		},
		secondaryActionText: {
			type: String,
			default: '取消'
		},
		canSubmit: {
			type: Boolean,
			default: false
		},
		isSubmitting: {
			type: Boolean,
			default: false
		}
	},
	emits: ['close', 'choose-avatar', 'nickname-input', 'submit']
}
</script>

<style lang="scss" scoped>
@import './sheet-action.scss';

.profile-sheet {
	padding: 26rpx 24rpx calc(env(safe-area-inset-bottom) + 24rpx);
	background: #f8f4ee;
}

.profile-sheet__header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 18rpx;
}

.profile-sheet__heading {
	flex: 1;
	min-width: 0;
}

.profile-sheet__title {
	display: block;
	font-size: 36rpx;
	font-weight: 700;
	color: #2f2923;
}

.profile-sheet__subtitle {
	display: block;
	margin-top: 10rpx;
	font-size: 24rpx;
	line-height: 1.6;
	color: #8a7d70;
}

.profile-sheet__close {
	width: 56rpx;
	height: 56rpx;
	border-radius: 999rpx;
	background: rgba(255, 255, 255, 0.75);
	display: flex;
	align-items: center;
	justify-content: center;
}

.profile-sheet__body {
	margin-top: 22rpx;
	padding: 24rpx;
	border-radius: 24rpx;
	background: rgba(255, 255, 255, 0.94);
	box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.04);
	display: flex;
	flex-direction: column;
	align-items: center;
}

.profile-sheet__avatar-button {
	width: 144rpx;
	height: 144rpx;
	padding: 0;
	border-radius: 999rpx;
	background: transparent;
	display: flex;
	align-items: center;
	justify-content: center;
	border: none;
}

.profile-sheet__avatar-button::after {
	border: none;
}

.profile-sheet__avatar-image,
.profile-sheet__avatar-fallback {
	width: 144rpx;
	height: 144rpx;
	border-radius: 999rpx;
}

.profile-sheet__avatar-image {
	display: block;
}

.profile-sheet__avatar-fallback {
	background: linear-gradient(180deg, #e8d8c5 0%, #dbc4a8 100%);
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 48rpx;
	font-weight: 700;
	color: #5b4a3b;
}

.profile-sheet__avatar-tip {
	margin-top: 16rpx;
	font-size: 22rpx;
	color: #877a6e;
}

.profile-sheet__field {
	width: 100%;
	margin-top: 28rpx;
}

.profile-sheet__label {
	display: block;
	font-size: 24rpx;
	font-weight: 600;
	color: #594c40;
}

.profile-sheet__input {
	margin-top: 12rpx;
	height: 92rpx;
	padding: 0 24rpx;
	border-radius: 20rpx;
	background: #f8f3ec;
	font-size: 28rpx;
	color: #2f2923;
}

.profile-sheet__placeholder {
	color: #b0a59a;
}

.profile-sheet__hint {
	display: block;
	margin-top: 12rpx;
	font-size: 22rpx;
	line-height: 1.6;
	color: #8a7d70;
}

.profile-sheet__footer {
	width: 100%;
	margin-top: 28rpx;
	display: flex;
	gap: 12rpx;
}
</style>
