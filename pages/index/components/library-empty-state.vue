<template>
	<view class="empty-state-card" :class="`empty-state-card--${kind}`">
		<view class="empty-state-card__icon-shell">
			<up-icon :name="iconName" size="22" :color="iconColor"></up-icon>
		</view>
		<text class="empty-state-card__title">{{ title }}</text>
		<text v-if="description" class="empty-state-card__desc">{{ description }}</text>
		<view v-if="primaryText || secondaryText" class="empty-state-card__actions">
			<view
				v-if="primaryText"
				class="empty-state-card__primary"
				@tap="$emit('primary')"
			>
				<up-icon
					v-if="primaryIcon && !primaryIconSrc"
					class="empty-state-card__primary-icon"
					:name="primaryIcon"
					size="14"
					color="#fffaf3"
				></up-icon>
				<image
					v-if="primaryIconSrc"
					class="empty-state-card__primary-image"
					:src="primaryIconSrc"
					mode="aspectFit"
				/>
				<text class="empty-state-card__primary-text">{{ primaryText }}</text>
			</view>
			<view
				v-if="secondaryText"
				class="empty-state-card__secondary"
				@tap="$emit('secondary')"
			>
				<text class="empty-state-card__secondary-text">{{ secondaryText }}</text>
			</view>
		</view>
	</view>
</template>

<script>
const ICON_MAP = {
	'search-no-results': 'search',
	'meal-empty': 'plus-circle',
	'status-empty': 'list-dot'
}

const ICON_COLOR_MAP = {
	'search-no-results': '#a08975',
	'meal-empty': '#a08975',
	'status-empty': '#8b6f5c'
}

export default {
	name: 'LibraryEmptyState',
	props: {
		kind: {
			type: String,
			default: 'meal-empty'
		},
		title: {
			type: String,
			default: ''
		},
		description: {
			type: String,
			default: ''
		},
		primaryText: {
			type: String,
			default: ''
		},
		primaryIcon: {
			type: String,
			default: ''
		},
		primaryIconSrc: {
			type: String,
			default: ''
		},
		secondaryText: {
			type: String,
			default: ''
		}
	},
	emits: ['primary', 'secondary'],
	computed: {
		iconName() {
			return ICON_MAP[this.kind] || 'empty-search'
		},
		iconColor() {
			return ICON_COLOR_MAP[this.kind] || '#a08975'
		}
	}
}
</script>

<style lang="scss" scoped>
@import './library-empty-state.scss';
</style>
