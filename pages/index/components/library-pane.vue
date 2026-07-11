<template>
<!-- 美食库模式 -->
<view
	class="library-mode-content"
>
	<library-header-section
		:is-library-meal-order-mode="isLibraryMealOrderMode"
		:library-header-title="libraryHeaderTitle"
		:library-header-summary="libraryHeaderSummary"
		:has-meal-order-spotlight-record="!!mealOrderSpotlightRecord"
		:meal-order-spotlight-date-text="mealOrderSpotlightDateText"
		:meal-order-spotlight-weekday="mealOrderSpotlightWeekday"
		:meal-order-spotlight-lead-text="mealOrderSpotlightLeadText"
		:meal-order-spotlight-status-text="mealOrderSpotlightStatusText"
		:meal-order-spotlight-status-kind="mealOrderSpotlightStatusKind"
		:meal-order-spotlight-desc="mealOrderSpotlightDesc"
		:meal-order-spotlight-count-text="mealOrderSpotlightCountText"
		:meal-order-spotlight-meta-text="mealOrderSpotlightMetaText"
		:meal-order-spotlight-motion-direction="mealOrderSpotlightMotionDirection"
		:meal-order-spotlight-motion-tick="mealOrderSpotlightMotionTick"
		@open-meal-order-date-sheet="openMealOrderDateSheet"
		@exit-meal-order-mode="exitMealOrderMode"
		@spotlight-tap="handleMealOrderSpotlightTap"
		@spotlight-touchstart="handleMealOrderSpotlightTouchStart"
		@spotlight-touchend="handleMealOrderSpotlightTouchEnd"
	></library-header-section>
	<view class="toolbar" :class="toolbarBounceClass">
		<view class="toolbar__search-row">
			<view
				class="search-box"
				:class="{ 'search-box--active': isSearchFocused || trimmedSearchKeyword }"
			>
				<view class="search-box__icon">
					<up-icon name="search" size="14" color="#a08775"></up-icon>
				</view>
				<input
					v-model="searchKeyword"
					class="search-box__input"
					:placeholder="searchPlaceholderText"
					placeholder-class="search-box__placeholder"
					confirm-type="search"
					@focus="handleSearchFocus"
					@blur="handleSearchBlur"
					@confirm="handleSearchConfirm"
				/>
				<view v-if="trimmedSearchKeyword" class="search-box__clear" @tap="clearSearchKeyword">
					<up-icon name="close" size="14" color="#a08775"></up-icon>
				</view>
			</view>
		</view>

		<view v-if="showSearchAssist && !isLibraryMealOrderMode" class="search-assist">
			<text class="search-assist__label">{{ searchAssistLabel }}</text>
			<view class="search-assist__chips">
				<view
					v-for="keyword in searchAssistKeywords"
					:key="`search-assist-${keyword}`"
					class="search-assist__chip"
					@tap="applySearchKeyword(keyword)"
				>
					<text class="search-assist__chip-text">{{ keyword }}</text>
				</view>
			</view>
		</view>

		<view v-if="!isLibraryMealOrderMode" class="filter-group">
			<view class="meal-tabs">
				<view
					v-for="tab in mealTabs"
					:key="tab.value"
					class="meal-tab"
					:class="{ 'meal-tab--active': activeMealType === tab.value }"
					@tap="handleMealTypeTabChange(tab.value)"
				>
					<view class="meal-tab__left">
						<up-icon
							:name="tab.icon"
							size="16"
							:color="activeMealType === tab.value ? tab.activeColor : '#a08775'"
						></up-icon>
						<text class="meal-tab__text">{{ tab.label }}</text>
					</view>
					<view class="meal-tab__count">
						<text class="meal-tab__count-text">{{ mealTypeCount(tab.value) }}</text>
					</view>
				</view>
			</view>
		</view>

		<view v-if="!isLibraryMealOrderMode" class="filter-group filter-group--compact">
			<view class="status-track">
				<view
					v-for="tab in statusTabs"
					:key="tab.value"
					class="status-pill"
					:class="[`status-pill--${tab.value}`, { 'status-pill--active': activeStatus === tab.value }]"
					@tap="handleStatusTabChange(tab.value)"
				>
					<up-icon
						:name="statusMap[tab.value].icon"
						size="16"
						:color="activeStatus === tab.value ? '#fffaf3' : tab.value === 'done' ? '#75866f' : '#a08775'"
					></up-icon>
					<text class="status-pill__text">{{ tab.label }}</text>
					<view v-if="tab.value !== 'all'" class="status-pill__count">
						<text class="status-pill__count-text">{{ statusCount(tab.value) }}</text>
					</view>
				</view>
			</view>
		</view>
	</view>

	<view v-if="!isLibraryMealOrderMode" class="list-caption">
		<view class="list-caption__top">
			<text class="list-caption__title">{{ currentFilterSummary }}</text>
			<view class="list-caption__actions">
				<view
					v-if="canResetLibraryFilters"
					class="list-caption__clear"
					@tap="resetLibraryFilters"
				>
					<text class="list-caption__clear-text">清除</text>
				</view>
				<view class="list-caption__pick" @tap="drawTonight">
					<view class="list-caption__pick-icon-shell">
						<up-icon name="reload" size="14" color="#6f5b4a"></up-icon>
					</view>
					<text class="list-caption__pick-text">帮我选</text>
				</view>
			</view>
		</view>
	</view>

	<view v-if="filteredRecipes.length" class="recipe-list">
		<recipe-card-item
			v-for="(card, index) in recipeCards"
			:key="card.id"
			:card="card"
			:cover-src="getRecipeCardDisplayCover(card)"
			:is-active="selectedRecipeId === card.id"
			:is-library-meal-order-mode="isLibraryMealOrderMode"
			:is-meal-order-selected="mealOrderHasRecipe(card.id)"
			:is-return-focus="returnFocusRecipeId === card.id"
			:motion-index="index"
			:motion-phase="recipeListMotionTick"
			:status-icon="statusMap[card.status].icon"
			@open="openRecipeDetail"
			@image-error="handleRecipeCardImageError"
			@toggle-status="toggleRecipeStatus"
			@toggle-meal-order="toggleMealOrderRecipe"
		></recipe-card-item>
	</view>

	<library-empty-state
		v-else
		:kind="emptyStateKind"
		:title="emptyStateTitle"
		:description="emptyStateDesc"
		:primary-text="emptyStatePrimaryText"
		:primary-icon="emptyStatePrimaryIcon"
		:primary-icon-src="emptyStatePrimaryIconSrc"
		:secondary-text="emptyStateSecondaryText"
		@primary="handleEmptyStatePrimary"
		@secondary="handleEmptyStateSecondary"
	></library-empty-state>
</view>
</template>

<script setup>
import { computed } from 'vue'
import LibraryEmptyState from './library-empty-state.vue'
import LibraryHeaderSection from './library-header-section.vue'
import RecipeCardItem from './recipe-card-item.vue'

const props = defineProps({
	isLibraryMealOrderMode: Boolean,
	libraryHeaderTitle: String,
	libraryHeaderSummary: String,
	mealOrderSpotlightRecord: Object,
	mealOrderSpotlightDateText: String,
	mealOrderSpotlightWeekday: String,
	mealOrderSpotlightLeadText: String,
	mealOrderSpotlightStatusText: String,
	mealOrderSpotlightStatusKind: String,
	mealOrderSpotlightDesc: String,
	mealOrderSpotlightCountText: String,
	mealOrderSpotlightMetaText: String,
	mealOrderSpotlightMotionDirection: String,
	mealOrderSpotlightMotionTick: Number,
	toolbarBounceClass: String,
	modelValue: String,
	isSearchFocused: Boolean,
	trimmedSearchKeyword: String,
	searchPlaceholderText: String,
	showSearchAssist: Boolean,
	searchAssistLabel: String,
	searchAssistKeywords: Array,
	mealTabs: Array,
	activeMealType: String,
	statusTabs: Array,
	activeStatus: String,
	statusMap: Object,
	currentFilterSummary: String,
	canResetLibraryFilters: Boolean,
	filteredRecipes: Array,
	recipeCards: Array,
	selectedRecipeId: String,
	returnFocusRecipeId: String,
	recipeListMotionTick: Number,
	emptyStateKind: String,
	emptyStateTitle: String,
	emptyStateDesc: String,
	emptyStatePrimaryText: String,
	emptyStatePrimaryIcon: String,
	emptyStatePrimaryIconSrc: String,
	emptyStateSecondaryText: String,
	mealTypeCount: Function,
	statusCount: Function,
	getRecipeCardDisplayCover: Function,
	mealOrderHasRecipe: Function
})
const emit = defineEmits([
	'update:modelValue', 'open-meal-order-date', 'exit-meal-order-mode',
	'spotlight-tap', 'spotlight-touchstart', 'spotlight-touchend', 'search-focus',
	'search-blur', 'search-confirm', 'apply-search', 'meal-type-change',
	'status-change', 'reset-filters', 'draw-tonight', 'open-recipe', 'image-error',
	'toggle-status', 'toggle-meal-order', 'empty-primary', 'empty-secondary'
])
const searchKeyword = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
function openMealOrderDateSheet() { emit('open-meal-order-date') }
function exitMealOrderMode() { emit('exit-meal-order-mode') }
function handleMealOrderSpotlightTap() { emit('spotlight-tap') }
function handleMealOrderSpotlightTouchStart(event) { emit('spotlight-touchstart', event) }
function handleMealOrderSpotlightTouchEnd(event) { emit('spotlight-touchend', event) }
function handleSearchFocus(event) { emit('search-focus', event) }
function handleSearchBlur(event) { emit('search-blur', event) }
function handleSearchConfirm(event) { emit('search-confirm', event) }
function clearSearchKeyword() { emit('update:modelValue', '') }
function applySearchKeyword(keyword) { emit('apply-search', keyword) }
function handleMealTypeTabChange(value) { emit('meal-type-change', value) }
function handleStatusTabChange(value) { emit('status-change', value) }
function resetLibraryFilters() { emit('reset-filters') }
function drawTonight() { emit('draw-tonight') }
function openRecipeDetail(id) { emit('open-recipe', id) }
function handleRecipeCardImageError(card) { emit('image-error', card) }
function toggleRecipeStatus(recipe) { emit('toggle-status', recipe) }
function toggleMealOrderRecipe(recipe) { emit('toggle-meal-order', recipe) }
function handleEmptyStatePrimary() { emit('empty-primary') }
function handleEmptyStateSecondary() { emit('empty-secondary') }
</script>
