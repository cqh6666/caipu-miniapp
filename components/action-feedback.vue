<template>
	<view
		v-if="visible"
		:key="feedbackKey"
		class="action-feedback"
		:class="[`action-feedback--${normalizedTone}`]"
	>
		<view class="action-feedback__aura"></view>
		<view class="action-feedback__card">
			<view class="action-feedback__icon-shell">
				<up-icon :name="resolvedIconName" size="18" :color="resolvedIconColor"></up-icon>
			</view>
			<view class="action-feedback__body">
				<text class="action-feedback__title">{{ title }}</text>
				<text v-if="description" class="action-feedback__desc">{{ description }}</text>
			</view>
		</view>
		<template v-if="showSparkles">
			<view class="action-feedback__spark action-feedback__spark--one"></view>
			<view class="action-feedback__spark action-feedback__spark--two"></view>
			<view class="action-feedback__spark action-feedback__spark--three"></view>
		</template>
	</view>
</template>

<script>
const supportedTones = new Set(['done', 'wishlist', 'pending'])

export default {
	name: 'ActionFeedback',
	props: {
		visible: {
			type: Boolean,
			default: false
		},
		feedbackKey: {
			type: String,
			default: ''
		},
		tone: {
			type: String,
			default: 'done'
		},
		title: {
			type: String,
			default: ''
		},
		description: {
			type: String,
			default: ''
		},
		iconName: {
			type: String,
			default: ''
		},
		iconColor: {
			type: String,
			default: ''
		},
		showSparkles: {
			type: Boolean,
			default: false
		}
	},
	computed: {
		normalizedTone() {
			const value = String(this.tone || '').trim().toLowerCase()
			return supportedTones.has(value) ? value : 'pending'
		},
		resolvedIconName() {
			if (this.iconName) return this.iconName
			if (this.normalizedTone === 'wishlist') return 'heart-fill'
			if (this.normalizedTone === 'pending') return 'clock-fill'
			return 'checkmark-circle-fill'
		},
		resolvedIconColor() {
			if (this.iconColor) return this.iconColor
			if (this.normalizedTone === 'wishlist') return '#fff7ee'
			if (this.normalizedTone === 'pending') return '#fff8e8'
			return '#f6fff1'
		}
	}
}
</script>

<style lang="scss" scoped>
.action-feedback {
	position: fixed;
	top: calc(env(safe-area-inset-top) + 146rpx);
	left: 26rpx;
	right: 26rpx;
	z-index: 13;
	display: flex;
	justify-content: center;
	pointer-events: none;
}

.action-feedback__card {
	position: relative;
	z-index: 2;
	width: 100%;
	max-width: 560rpx;
	min-height: 92rpx;
	padding: 14rpx 18rpx;
	border-radius: 28rpx;
	display: flex;
	align-items: center;
	gap: 14rpx;
	box-shadow:
		0 18rpx 34rpx rgba(48, 36, 28, 0.14),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.36);
	animation: action-feedback-pop 260ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

.action-feedback__aura {
	position: absolute;
	top: 8rpx;
	left: 50%;
	width: 232rpx;
	height: 232rpx;
	border-radius: 999rpx;
	transform: translateX(-50%);
	filter: blur(22rpx);
	opacity: 0.36;
	animation: action-feedback-aura 420ms ease-out both;
}

.action-feedback__icon-shell {
	width: 54rpx;
	height: 54rpx;
	border-radius: 18rpx;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	border: 1px solid rgba(255, 255, 255, 0.18);
}

.action-feedback__body {
	min-width: 0;
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 4rpx;
}

.action-feedback__title {
	display: block;
	font-size: 24rpx;
	font-weight: 700;
	line-height: 1.2;
	color: #fffdf8;
}

.action-feedback__desc {
	display: block;
	font-size: 21rpx;
	line-height: 1.3;
	color: rgba(255, 250, 244, 0.82);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.action-feedback--done .action-feedback__card {
	background:
		radial-gradient(circle at top left, rgba(246, 255, 241, 0.24) 0%, rgba(246, 255, 241, 0) 42%),
		linear-gradient(145deg, rgba(88, 108, 88, 0.96) 0%, rgba(63, 80, 64, 0.94) 100%);
	border: 1px solid rgba(232, 246, 230, 0.18);
}

.action-feedback--done .action-feedback__aura {
	background: radial-gradient(circle at center, rgba(145, 187, 140, 0.38) 0%, rgba(145, 187, 140, 0) 72%);
}

.action-feedback--done .action-feedback__icon-shell {
	background: rgba(246, 255, 241, 0.14);
}

.action-feedback--wishlist .action-feedback__card {
	background:
		radial-gradient(circle at top left, rgba(255, 247, 238, 0.26) 0%, rgba(255, 247, 238, 0) 42%),
		linear-gradient(145deg, rgba(128, 92, 68, 0.96) 0%, rgba(101, 72, 53, 0.94) 100%);
	border: 1px solid rgba(255, 238, 221, 0.16);
}

.action-feedback--wishlist .action-feedback__aura {
	background: radial-gradient(circle at center, rgba(221, 169, 126, 0.34) 0%, rgba(221, 169, 126, 0) 72%);
}

.action-feedback--wishlist .action-feedback__icon-shell {
	background: rgba(255, 247, 238, 0.14);
}

.action-feedback--pending .action-feedback__card {
	background:
		radial-gradient(circle at top left, rgba(255, 249, 228, 0.24) 0%, rgba(255, 249, 228, 0) 42%),
		linear-gradient(145deg, rgba(133, 98, 60, 0.96) 0%, rgba(103, 74, 45, 0.94) 100%);
	border: 1px solid rgba(255, 242, 211, 0.16);
}

.action-feedback--pending .action-feedback__aura {
	background: radial-gradient(circle at center, rgba(236, 195, 116, 0.34) 0%, rgba(236, 195, 116, 0) 72%);
}

.action-feedback--pending .action-feedback__icon-shell {
	background: rgba(255, 248, 233, 0.14);
}

.action-feedback__spark {
	position: absolute;
	z-index: 1;
	width: 12rpx;
	height: 12rpx;
	border-radius: 999rpx;
	background: rgba(240, 255, 236, 0.95);
	box-shadow: 0 0 0 8rpx rgba(240, 255, 236, 0.08);
	opacity: 0;
}

.action-feedback__spark--one {
	top: 24rpx;
	left: 50%;
	animation: action-feedback-spark-one 560ms ease-out 40ms both;
}

.action-feedback__spark--two {
	top: 62rpx;
	left: calc(50% + 72rpx);
	animation: action-feedback-spark-two 620ms ease-out 70ms both;
}

.action-feedback__spark--three {
	top: 78rpx;
	left: calc(50% - 76rpx);
	animation: action-feedback-spark-three 620ms ease-out 90ms both;
}

@keyframes action-feedback-pop {
	from {
		opacity: 0;
		transform: translateY(-14rpx) scale(0.94);
	}
	to {
		opacity: 1;
		transform: translateY(0) scale(1);
	}
}

@keyframes action-feedback-aura {
	from {
		opacity: 0;
		transform: translateX(-50%) scale(0.7);
	}
	to {
		opacity: 0.36;
		transform: translateX(-50%) scale(1);
	}
}

@keyframes action-feedback-spark-one {
	0% {
		opacity: 0;
		transform: translate(-4rpx, 10rpx) scale(0.5);
	}
	35% {
		opacity: 0.9;
	}
	100% {
		opacity: 0;
		transform: translate(-88rpx, -26rpx) scale(1);
	}
}

@keyframes action-feedback-spark-two {
	0% {
		opacity: 0;
		transform: translate(-4rpx, 6rpx) scale(0.48);
	}
	32% {
		opacity: 0.86;
	}
	100% {
		opacity: 0;
		transform: translate(54rpx, -34rpx) scale(1);
	}
}

@keyframes action-feedback-spark-three {
	0% {
		opacity: 0;
		transform: translate(0, 6rpx) scale(0.48);
	}
	34% {
		opacity: 0.86;
	}
	100% {
		opacity: 0;
		transform: translate(-58rpx, -24rpx) scale(1);
	}
}
</style>
