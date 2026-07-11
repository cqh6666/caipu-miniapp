<template>
<view
	class="explore-mode-content"
	:class="appModePaneMotionClass"
>
	<!-- 打卡点搜索 & 筛选 -->
	<view class="toolbar">
		<view class="toolbar__search-row">
			<view
				class="search-box"
				:class="{ 'search-box--active': isPlaceSearchFocused || trimmedPlaceSearchKeyword }"
			>
				<view class="search-box__icon">
					<up-icon name="search" size="14" color="#a08775"></up-icon>
				</view>
				<input
					v-model="searchKeyword"
					class="search-box__input"
					placeholder="搜索店铺 / 地点..."
					placeholder-class="search-box__placeholder"
					confirm-type="search"
					@focus="emit('search-focus')"
					@blur="emit('search-blur')"
				/>
				<view v-if="trimmedPlaceSearchKeyword" class="search-box__clear" @tap="emit('update:searchKeyword', '')">
					<up-icon name="close" size="14" color="#a08775"></up-icon>
				</view>
			</view>
		</view>

		<view class="filter-group filter-group--compact">
			<view class="status-track">
				<view
					v-for="tab in placeStatusTabs"
					:key="tab.value"
					class="status-pill"
					:class="[`status-pill--${tab.value}`, { 'status-pill--active': activePlaceStatus === tab.value }]"
					@tap="emit('update:activeStatus', tab.value)"
				>
					<up-icon
						:name="tab.icon"
						size="16"
						:color="activePlaceStatus === tab.value ? '#fffaf3' : tab.value === 'visited' ? '#75866f' : '#a08775'"
					></up-icon>
					<text class="status-pill__text">{{ tab.label }}</text>
					<view class="status-pill__count">
						<text class="status-pill__count-text">{{ placeStatusCount(tab.value) }}</text>
					</view>
				</view>
			</view>
		</view>
	</view>

	<!-- 打卡点列表标题 -->
	<view class="list-caption">
		<view class="list-caption__top">
			<text class="list-caption__title">打卡心愿单 · {{ filteredPlaces.length }}个</text>
		</view>
	</view>

	<!-- 打卡点列表 -->
	<view v-if="filteredPlaces.length" class="place-list">
		<place-card-item
			v-for="place in filteredPlaces"
			:key="place.id"
			:place="place"
			@open="handlePlaceOpen"
			@open-location="openPlaceLocation"
		></place-card-item>
	</view>

	<view v-else class="explore-empty">
		<view class="explore-empty__icon-shell">
			<up-icon name="map-fill" size="24" color="rgba(92,64,51,0.25)"></up-icon>
		</view>
		<text class="explore-empty__text">还没有打卡点，去添加一些吧</text>
		<view class="explore-empty__action" @tap="openPlaceCreateSheet">
			<text class="explore-empty__action-text">添加打卡点</text>
		</view>
	</view>
</view>
</template>

<script setup>
import { computed } from 'vue'
import PlaceCardItem from './place-card-item.vue'
const props = defineProps({
	appModePaneMotionClass: String,
	searchKeyword: String,
	isPlaceSearchFocused: Boolean,
	trimmedPlaceSearchKeyword: String,
	placeStatusTabs: Array,
	activeStatus: String,
	filteredPlaces: Array,
	placeStatusCount: Function
})
const emit = defineEmits(['update:searchKeyword', 'update:activeStatus', 'search-focus', 'search-blur', 'open-place', 'open-location', 'add-place'])
const searchKeyword = computed({ get: () => props.searchKeyword, set: value => emit('update:searchKeyword', value) })
const activePlaceStatus = computed(() => props.activeStatus)
function handlePlaceOpen(id) { emit('open-place', id) }
function openPlaceLocation(id) { emit('open-location', id) }
function openPlaceCreateSheet() { emit('add-place') }
</script>
