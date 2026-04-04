<template>
	<up-popup
		:show="show"
		mode="center"
		overlayOpacity="0.16"
		:closeOnClickOverlay="true"
		:safeAreaInsetBottom="false"
		@close="$emit('close')"
	>
		<view class="random-pick-stage" @tap.self="$emit('close')">
			<view class="random-pick-stage__ambient"></view>
			<view class="random-pick-shell" @tap.stop>
				<view class="random-pick-shell__glow random-pick-shell__glow--one"></view>
				<view class="random-pick-shell__glow random-pick-shell__glow--two"></view>
				<view class="random-pick-sheet">
					<view
						class="random-pick-sheet__content"
						:key="revealKey"
						:class="[`random-pick-sheet__content--${normalizedMotionMode}`]"
					>
						<view class="random-pick-sheet__hero" :class="{ 'random-pick-sheet__hero--empty': !coverSrc }">
							<image
								v-if="coverSrc"
								class="random-pick-sheet__image"
								:src="coverSrc"
								mode="aspectFill"
							></image>
							<view v-else class="random-pick-sheet__placeholder">
								<view class="random-pick-sheet__placeholder-icon">
									<up-icon :name="card?.placeholderIcon || 'grid-fill'" size="26" color="#896f59"></up-icon>
								</view>
								<text class="random-pick-sheet__placeholder-text">先看看这道</text>
							</view>
						</view>

						<view class="random-pick-sheet__body">
							<text v-if="contextText" class="random-pick-sheet__context">{{ contextText }}</text>
							<text class="random-pick-sheet__title">{{ card?.title || '今天吃什么好' }}</text>

							<view class="random-pick-sheet__meta">
								<view v-if="mealLabel" class="random-pick-chip">
									<text class="random-pick-chip__text">{{ mealLabel }}</text>
								</view>
								<view
									v-if="statusLabel"
									class="random-pick-chip"
									:class="statusToneClass"
								>
									<text class="random-pick-chip__text">{{ statusLabel }}</text>
								</view>
								<view v-if="card?.sourceBadge" class="random-pick-chip random-pick-chip--source">
									<text class="random-pick-chip__text">{{ card.sourceBadge }}</text>
								</view>
							</view>

							<text class="random-pick-sheet__summary" :class="{ 'random-pick-sheet__summary--muted': !summaryText }">
								{{ summaryText || '先看看食材和做法，说不定就是今天这口。' }}
							</text>
						</view>

						<view class="random-pick-sheet__actions">
							<view
								class="random-pick-action random-pick-action--ghost"
								@tap="$emit('open-detail', card?.id)"
							>
								<text class="random-pick-action__text">了解一下</text>
								<up-icon name="arrow-right" size="14" color="#715845"></up-icon>
							</view>
							<view
								class="random-pick-action random-pick-action--primary"
								:class="{ 'random-pick-action--disabled': !canReroll }"
								@tap="canReroll && $emit('reroll')"
							>
								<text class="random-pick-action__text random-pick-action__text--primary">换一个</text>
								<up-icon name="reload" size="13" color="#fffaf4"></up-icon>
							</view>
						</view>
					</view>
				</view>
			</view>
		</view>
	</up-popup>
</template>

<script>
import { mealTypeLabelMap, statusLabelMap } from '../../../utils/recipe-store'

export default {
	name: 'RandomPickSheet',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		card: {
			type: Object,
			default: () => ({})
		},
		coverSrc: {
			type: String,
			default: ''
		},
		contextText: {
			type: String,
			default: ''
		},
		canReroll: {
			type: Boolean,
			default: false
		},
		motionMode: {
			type: String,
			default: 'enter'
		},
		revealKey: {
			type: String,
			default: ''
		}
	},
	emits: ['close', 'reroll', 'open-detail'],
	computed: {
		normalizedMotionMode() {
			return this.motionMode === 'swap' ? 'swap' : 'enter'
		},
		mealLabel() {
			return this.card?.mealTypeLabel || mealTypeLabelMap[this.card?.mealType] || ''
		},
		statusLabel() {
			return statusLabelMap[this.card?.status] || ''
		},
		statusToneClass() {
			return this.card?.status === 'done' ? 'random-pick-chip--done' : 'random-pick-chip--wishlist'
		},
		summaryText() {
			const summary = String(this.card?.listSummary || '').trim()
			if (summary) return summary
			const ingredient = String(this.card?.ingredient || '').trim()
			const note = String(this.card?.note || '').trim()
			if (ingredient && note) return `${ingredient} · ${note}`
			return ingredient || note
		}
	}
}
</script>

<style lang="scss" scoped>
@import './random-pick-sheet.scss';
</style>
