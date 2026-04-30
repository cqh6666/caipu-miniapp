<template>
	<view class="app-shell">
		<view
			class="page-content"
			:class="{
				'page-content--meal-order': showMealOrderFloatingBar,
				'page-content--meal-order-entering': mealOrderModeMotionState === 'entering',
				'page-content--meal-order-leaving': mealOrderModeMotionState === 'leaving'
			}"
		>
			<view
				v-if="activeSection === 'library'"
				class="library-shell"
				:class="{ 'library-shell--entering': libraryEnterMotionActive }"
			>
				<library-header-section
					:is-library-meal-order-mode="isLibraryMealOrderMode"
					:library-header-title="libraryHeaderTitle"
					:library-header-summary="libraryHeaderSummary"
					:has-meal-order-spotlight-record="!!mealOrderSpotlightRecord"
					:meal-order-spotlight-date-text="mealOrderSpotlightDateText"
					:meal-order-spotlight-weekday="mealOrderSpotlightWeekday"
					:meal-order-spotlight-status-text="mealOrderSpotlightStatusText"
					:meal-order-spotlight-status-kind="mealOrderSpotlightStatusKind"
					:meal-order-spotlight-desc="mealOrderSpotlightDesc"
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
									<up-icon name="reload" size="12" color="#6f5b4a"></up-icon>
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
					:secondary-text="emptyStateSecondaryText"
					@primary="handleEmptyStatePrimary"
					@secondary="handleEmptyStateSecondary"
				></library-empty-state>
			</view>

			<template v-else>
				<kitchen-section
					:current-kitchen-name="currentKitchenName"
					:can-switch-kitchen="canSwitchKitchen"
					:kitchen-connection-label="kitchenConnectionLabel"
					:is-kitchen-connected="isKitchenConnected"
					:current-kitchen-display-name="currentKitchenDisplayName"
					:current-kitchen-meta-text="currentKitchenMetaText"
					:kitchen-members-count="kitchenMembers.length || 0"
					:current-kitchen-role-label="currentKitchenRoleLabel"
					:kitchen-options-count="kitchenOptions.length"
					:invite-action-description="inviteActionDescription"
					:member-panel-summary="memberPanelSummary"
					:has-more-kitchen-members="hasMoreKitchenMembers"
					:visible-kitchen-members="visibleKitchenMembers"
					:is-loading-kitchen-members="isLoadingKitchenMembers"
					:member-initial="memberInitial"
					:member-display-name="memberDisplayName"
					:member-role-label="memberRoleLabel"
					:member-member-description="memberMemberDescription"
					@open-kitchen-selector="openKitchenSelector"
					@open-kitchen-name-sheet="openKitchenNameSheet"
					@open-invite-sheet="openInviteSheet"
					@show-all-members="showAllMembers"
					@member-tap="handleMemberCardTap"
					@open-invite-code-sheet="openInviteCodeSheet"
				></kitchen-section>

			</template>

			<view v-if="!isLibraryMealOrderMode" class="app-footer-links">
				<view class="app-footer-link" @tap="openAboutPage">
					<text class="app-footer-link__label">关于我们</text>
				</view>
			</view>
		</view>

		<view v-if="showMealOrderFloatingBar" class="meal-order-floating">
			<view class="meal-order-floating__summary" @tap="openMealOrderCartSheet">
				<view class="meal-order-floating__summary-main">
					<view class="meal-order-floating__pill" :class="{ 'meal-order-floating__pill--empty': !mealOrderCanCheckout }">
						<view class="meal-order-floating__pill-dot"></view>
						<text class="meal-order-floating__pill-text">{{ mealOrderFloatingTitle }}</text>
					</view>
					<view class="meal-order-floating__peek">
						<up-icon name="arrow-right" size="14" color="rgba(255, 246, 235, 0.58)"></up-icon>
					</view>
				</view>
			</view>
			<view
				class="meal-order-floating__action"
				:class="{ 'meal-order-floating__action--disabled': !mealOrderCanCheckout }"
				@tap="openMealOrderCheckoutSheet"
			>
				<text class="meal-order-floating__action-text">{{ mealOrderFloatingActionText }}</text>
			</view>
		</view>

		<view
			class="bottom-nav"
			:class="{
				'bottom-nav--meal-order': showMealOrderFloatingBar,
				'bottom-nav--meal-order-entering': mealOrderModeMotionState === 'entering',
				'bottom-nav--meal-order-leaving': mealOrderModeMotionState === 'leaving'
			}"
		>
			<view
				class="nav-item"
				:class="{ 'nav-item--active': activeSection === 'library' }"
				@tap="switchSection('library')"
			>
				<view class="nav-item__icon-shell">
					<up-icon
						:name="activeSection === 'library' ? 'home-fill' : 'home'"
						size="24"
						:color="activeSection === 'library' ? '#6b4d3d' : '#b9aea8'"
					></up-icon>
				</view>
				<text class="nav-item__label">美食库</text>
			</view>

			<view class="nav-center">
				<view
					class="nav-fab"
					hover-class="nav-fab--pressed"
					hover-start-time="0"
					hover-stay-time="140"
					hover-stop-propagation
					@tap="openDietAssistantSheet"
				>
					<!-- 星闪入口：打开饮食管家聊天窗 -->
					<image
						class="nav-fab__icon"
						src="/static/icons/sparkle-plus.svg"
						mode="aspectFit"
					/>
				</view>
			</view>

			<view
				class="nav-item"
				:class="{ 'nav-item--active': activeSection === 'kitchen' }"
				@tap="switchSection('kitchen')"
			>
				<view class="nav-item__icon-shell">
					<image
						class="nav-space-icon"
						:src="activeSection === 'kitchen' ? '/static/icons/space-grid-active.svg' : '/static/icons/space-grid.svg'"
						mode="aspectFit"
					/>
				</view>
				<text class="nav-item__label">空间</text>
			</view>
		</view>

		<meal-order-date-sheet
			:show="showMealOrderDateSheet"
			:quick-date-options="mealOrderQuickDateOptions"
			:picker-value="mealOrderDatePickerValue"
			:date-start="mealOrderDateStart"
			@close="closeMealOrderDateSheet"
			@pick-date="startMealOrderMode"
			@start="startMealOrderMode"
		></meal-order-date-sheet>

		<meal-order-cart-sheet
			:show="showMealOrderCartSheet"
			:date-text="mealOrderDateText"
			:dish-count="mealOrderCartDishCount"
			:items="mealOrderCartItems"
			:note="mealOrderDraftNote"
			:can-checkout="mealOrderCanCheckout"
			@close="closeMealOrderCartSheet"
			@open-recipe="openMealOrderRecipeDetail"
			@remove-recipe="removeMealOrderRecipe"
			@note-input="handleMealOrderNoteInput"
			@clear="clearMealOrderCart"
			@confirm="openMealOrderCheckoutSheet"
		></meal-order-cart-sheet>

		<meal-order-checkout-sheet
			:show="showMealOrderCheckoutSheet"
			:date-text="mealOrderDateText"
			:dish-count="mealOrderCartDishCount"
			:items="mealOrderCartItems"
			:note="mealOrderDraftNote"
			:helper-text="mealOrderCheckoutHelperText"
			:can-checkout="mealOrderCanCheckout"
			:is-submitting="isSubmittingMealOrder"
			@close="closeMealOrderCheckoutSheet"
			@open-recipe="openMealOrderRecipeDetail"
			@submit="submitMealOrder"
		></meal-order-checkout-sheet>

		<meal-order-success-sheet
			:show="showMealOrderSuccessSheet"
			:date-text="mealOrderSuccessDateText"
			:dish-count="mealOrderSuccessDishCount"
			:dish-summary="mealOrderSuccessDishSummary"
			:note="mealOrderSuccessNote"
			@close="closeMealOrderSuccessSheet"
			@view-record="viewMealOrderSuccessRecord"
			@plan-next="planNextMealOrder"
		></meal-order-success-sheet>

		<invite-sheet
			:show="showInviteSheet"
			:subtitle="inviteSheetSubtitle"
			:is-preparing="isPreparingInvite"
			:invite="activeInvite"
			:preparing-text="invitePreparingText"
			:formatted-code="formattedActiveInviteCode"
			:meta-line="inviteMetaLine"
			:copied="inviteCodeCopied"
			:show-share-action="showInviteShareAction"
			@close="closeInviteSheet"
			@copy-code="copyInviteCode"
			@regenerate="regenerateInviteCode"
		></invite-sheet>

		<invite-code-sheet
			:show="showInviteCodeSheet"
			:value="inviteCodeInput"
			:can-submit="canSubmitInviteCode"
			@close="closeInviteCodeSheet"
			@input-code="handleInviteCodeInput"
			@submit="submitInviteCode"
		></invite-code-sheet>

		<profile-sheet
			:show="showProfileSheet"
			:title="profileSheetTitle"
			:subtitle="profileSheetSubtitle"
			:avatar-preview="profileAvatarPreview"
			:avatar-fallback="profileAvatarFallback"
			:nickname="profileDraft.nickname"
			:secondary-action-text="profileSheetSecondaryActionText"
			:can-submit="canSubmitProfile"
			:is-submitting="isSubmittingProfile"
			@close="closeProfileSheet"
			@choose-avatar="handleChooseAvatar"
			@nickname-input="handleProfileNicknameInput"
			@submit="submitProfile"
		></profile-sheet>

		<add-recipe-sheet
			:show="showAddSheet"
			:draft="draft"
			:draft-link-assist-text="draftLinkAssistText"
			:is-link-previewing="isDraftLinkPreviewing"
			:has-link-preview-error="!!draftLinkPreviewError"
			:draft-title-assist-text="draftTitleAssistText"
			:has-auto-title="!!draftAutoTitle"
			:is-title-touched="draftTitleTouched"
			:max-recipe-images="maxRecipeImages"
			:meal-tabs="mealTabs"
			:draft-status-options="draftStatusOptions"
			:can-submit="canSubmitDraft"
			:is-submitting="isSubmittingDraft"
			@close="closeAddSheet"
			@link-input="handleDraftLinkInput"
			@title-input="handleDraftTitleInput"
			@preview-image="previewDraftImages"
			@remove-image="removeDraftImage"
			@choose-images="chooseDraftImages"
			@select-meal-type="handleDraftMealTypeSelect"
			@select-status="handleDraftStatusSelect"
			@note-input="handleDraftNoteInput"
			@submit="submitDraft"
		></add-recipe-sheet>

		<diet-assistant-sheet
			:show="showDietAssistantSheet"
			@close="closeDietAssistantSheet"
			@open-add-recipe="openAddSheetFromAssistant"
		></diet-assistant-sheet>

		<action-feedback
			:visible="recipeStatusFeedbackVisible && activeSection === 'library'"
			:feedback-key="recipeStatusFeedbackKey"
			:tone="recipeStatusFeedbackTone"
			:title="recipeStatusFeedbackTitle"
			:description="recipeStatusFeedbackRecipeTitle"
			:show-sparkles="recipeStatusFeedbackShowSparkles"
		></action-feedback>

		<random-pick-sheet
			:show="showRandomPickSheet && !!randomPickCard"
			:card="randomPickCard"
			:cover-src="randomPickCoverSrc"
			:context-text="randomPickContextText"
			:can-reroll="randomPickCanReroll"
			:motion-mode="randomPickMotionMode"
			:reveal-key="randomPickRevealKey"
			@close="closeRandomPickSheet"
			@reroll="rerollTonightPick"
			@open-detail="openRandomPickDetail"
		></random-pick-sheet>
	</view>
</template>

<script>
import ActionFeedback from '../../components/action-feedback.vue'
import { appConfig } from '../../utils/app-config'
import {
	listMealPlanStore,
	saveMealPlanDraft,
	submitMealPlan as submitMealPlanRequest
} from '../../utils/meal-plan-api'
import { previewRecipeLink } from '../../utils/recipe-api'
import { buildImageCacheKey, getCachedImagePath, invalidateCachedImage, warmImageCache } from '../../utils/image-cache'
import { ensureUploadedImage } from '../../utils/upload-api'
import {
	MAX_RECIPE_IMAGES,
	createRecipeFromDraft,
	getCachedRecipes,
	loadRecipes,
	mealTypeLabelMap,
	mealTypeOptions,
	statusOptions,
	toggleRecipeStatusById
} from '../../utils/recipe-store'
import { createKitchenInvite, formatInviteCode, listKitchenMembers, normalizeInviteCode, updateKitchen } from '../../utils/kitchen-api'
import {
	ensureSession,
	getCurrentKitchenId,
	getFriendlySessionErrorMessage,
	getSessionSnapshot,
	isProfileIncomplete,
	isPlaceholderNickname,
	saveCurrentUserProfile,
	setCurrentKitchenId,
	updateSessionKitchen
} from '../../utils/auth'
import { createEmptyDraft, MAX_RECENT_SEARCHES, searchSuggestionKeywordsByMeal, statusMap } from './constants'
import AddRecipeSheet from './components/add-recipe-sheet.vue'
import DietAssistantSheet from './components/diet-assistant-sheet.vue'
import { detectDraftLinkPlatform, extractSupportedDraftLink, guessDraftTitleFromShareText, normalizeDraftAutoTitle } from './draft-link'
import InviteCodeSheet from './components/invite-code-sheet.vue'
import InviteSheet from './components/invite-sheet.vue'
import KitchenSection from './components/kitchen-section.vue'
import LibraryEmptyState from './components/library-empty-state.vue'
import LibraryHeaderSection from './components/library-header-section.vue'
import MealOrderCartSheet from './components/meal-order-cart-sheet.vue'
import MealOrderCheckoutSheet from './components/meal-order-checkout-sheet.vue'
import MealOrderDateSheet from './components/meal-order-date-sheet.vue'
import MealOrderSuccessSheet from './components/meal-order-success-sheet.vue'
import ProfileSheet from './components/profile-sheet.vue'
import RandomPickSheet from './components/random-pick-sheet.vue'
import RecipeCardItem from './components/recipe-card-item.vue'
import {
	addDaysFromISODate,
	buildMealOrderDishSummary,
	buildMealPlanPayload,
	consumePendingMealOrderAction,
	createEmptyMealOrderStore,
	formatMealOrderDateParts,
	formatMealOrderDateText,
	formatMealOrderHeaderTitle,
	nextWeekendISODate,
	normalizeMealOrderDate,
	normalizeMealOrderDraft,
	normalizeMealOrderRecord,
	normalizeMealOrderStore,
	toISODate
} from './meal-order'
import { buildRecipeCard, buildRecipeCoverVersion, buildRecipeSearchText, extractRecipeImages } from './recipe-card'
import { readLastDraftLinkPrefill, readRecentSearches, writeLastDraftLinkPrefill, writeRecentSearches } from './storage'

const inviteShareFallbackImageUrl = '/static/invite-share-cover.png'

function replaceKitchenLabel(value = '') {
	return String(value || '').replace(/厨房/g, '空间')
}

function shortenInviteShareText(value = '', maxLength = 10) {
	const text = String(value || '').trim()
	if (!text) return ''
	if (text.length <= maxLength) return text
	return `${text.slice(0, maxLength)}...`
}

function buildInviteShareTitle(invite = {}, fallbackKitchenName = '') {
	const kitchenName = shortenInviteShareText(replaceKitchenLabel(invite?.kitchenName || fallbackKitchenName), 8)

	if (kitchenName) {
		return `邀请你加入「${kitchenName}」`
	}
	return '邀请你加入共享空间'
}

function buildInviteShareImageURL(invite = {}) {
	const raw = String(invite?.shareImageUrl || '').trim()
	if (!raw) {
		return inviteShareFallbackImageUrl
	}
	const target = raw
	const separator = target.includes('?') ? '&' : '?'
	return `${target}${separator}ts=${Date.now()}`
}

export default {
	components: {
		ActionFeedback,
		AddRecipeSheet,
		DietAssistantSheet,
		InviteCodeSheet,
		InviteSheet,
		KitchenSection,
		LibraryEmptyState,
		LibraryHeaderSection,
		MealOrderCartSheet,
		MealOrderCheckoutSheet,
		MealOrderDateSheet,
		MealOrderSuccessSheet,
		ProfileSheet,
		RandomPickSheet,
		RecipeCardItem
	},
	data() {
		return {
			statusMap,
			activeSection: 'library',
			activeMealType: 'main',
			activeStatus: 'all',
			toolbarBounceClass: '',
			toolbarBounceTimer: null,
			libraryEnterMotionActive: false,
			libraryEnterMotionTimer: null,
			searchKeyword: '',
			recentSearches: readRecentSearches(),
			lastDraftLinkPrefill: readLastDraftLinkPrefill(),
			isSearchFocused: false,
			searchBlurTimer: null,
			selectedRecipeId: '',
			showAddSheet: false,
			showDietAssistantSheet: false,
			draftLinkPrefillSource: '',
			draftClipboardPrefillRequestID: 0,
			showInviteSheet: false,
			showInviteCodeSheet: false,
			showProfileSheet: false,
			showMealOrderDateSheet: false,
			showMealOrderCartSheet: false,
			showMealOrderCheckoutSheet: false,
			showMealOrderSuccessSheet: false,
			isMealOrderMode: false,
			mealOrderModeMotionState: '',
			mealOrderModeMotionTimer: null,
			currentKitchenId: 0,
			mealOrderDate: '',
			mealOrderLastSubmittedDate: '',
			mealOrderStore: createEmptyMealOrderStore(),
			mealOrderStoreLoadedKitchenId: 0,
			mealOrderSpotlightIndex: 0,
			mealOrderSpotlightMotionDirection: '',
			mealOrderSpotlightMotionTick: 0,
			mealOrderSpotlightTouchStartX: 0,
			mealOrderSpotlightTouchStartY: 0,
			mealOrderSpotlightSuppressTap: false,
			mealOrderDraftSyncTimer: null,
			mealOrderLocalVersion: 0,
			mealOrderSyncContextID: 0,
			mealOrderStoreRequestID: 0,
			profileSheetMode: 'prompt',
			mealTabs: mealTypeOptions,
			statusTabs: [
				{ label: '全部', value: 'all' },
				{ label: '想吃', value: 'wishlist' },
				{ label: '吃过', value: 'done' }
			],
			draftStatusOptions: statusOptions,
			maxRecipeImages: MAX_RECIPE_IMAGES,
			draft: createEmptyDraft(),
			recipes: [],
			kitchenOptions: [],
			currentUser: null,
			currentKitchenName: '',
			currentKitchenRole: '',
			kitchenMembers: [],
			kitchenMembersKitchenId: 0,
			activeInvite: null,
			inviteCodeCopied: false,
			inviteCodeInput: '',
			profileDraft: {
				nickname: '',
				avatarUrl: ''
			},
			draftAutoTitle: '',
			draftTitleTouched: false,
			draftLinkPreviewPlatform: '',
			draftLinkPreviewTitleSource: '',
			draftLinkPreviewError: '',
			draftLinkPreviewTimer: null,
			draftLinkPreviewRequestID: 0,
			isDraftLinkPreviewing: false,
			hasDismissedProfilePrompt: false,
			cachedRecipeCoverMap: {},
			recipeCardCoverFallbackMap: {},
			recipeCardHiddenMap: {},
			recipeCoverCacheRequestID: 0,
			recipeListMotionTick: 0,
			recipeStatusPendingMap: {},
			recipeStatusFeedbackVisible: false,
			recipeStatusFeedbackTone: '',
			recipeStatusFeedbackTitle: '',
			recipeStatusFeedbackRecipeTitle: '',
			recipeStatusFeedbackShowSparkles: false,
			recipeStatusFeedbackTick: 0,
			recipeStatusFeedbackTimer: null,
			showRandomPickSheet: false,
			randomPickRecipeId: '',
			randomPickContextText: '',
			randomPickPoolRecipeIds: [],
			randomPickMotionMode: 'enter',
			randomPickTick: 0,
			syncErrorMessage: '',
			isSyncing: false,
			isSubmittingDraft: false,
			isSubmittingMealOrder: false,
			isSubmittingKitchenName: false,
			isSubmittingProfile: false,
			isLoadingKitchenMembers: false,
			isPreparingInvite: false
		}
	},
	onLoad(options) {
		if (options?.section === 'kitchen') {
			this.activeSection = 'kitchen'
		}
	},
	onShow() {
		this.refreshRecipes()
		this.triggerLibraryEnterMotion()
	},
	onHide() {
		if (!this.isSubmittingMealOrder) {
			this.syncMealOrderDraft({ silent: true })
		}
		this.clearMealOrderDraftSyncTimer()
		this.clearMealOrderModeMotionTimer()
		this.clearRecipeStatusFeedback()
		this.closeRandomPickSheet()
		this.clearDraftLinkPreviewState()
		this.clearSearchBlurTimer()
		this.recipeCoverCacheRequestID += 1
	},
	onUnload() {
		if (!this.isSubmittingMealOrder) {
			this.syncMealOrderDraft({ silent: true })
		}
		this.clearMealOrderDraftSyncTimer()
		this.clearMealOrderModeMotionTimer()
		this.clearRecipeStatusFeedback()
		this.closeRandomPickSheet()
		this.clearDraftLinkPreviewState()
		this.clearSearchBlurTimer()
		if (this.toolbarBounceTimer) {
			clearTimeout(this.toolbarBounceTimer)
			this.toolbarBounceTimer = null
		}
		if (this.libraryEnterMotionTimer) {
			clearTimeout(this.libraryEnterMotionTimer)
			this.libraryEnterMotionTimer = null
		}
		this.recipeCoverCacheRequestID += 1
	},
	onShareAppMessage(res) {
		if (res?.from === 'button' && this.activeInvite?.sharePath) {
			return {
				title: buildInviteShareTitle(this.activeInvite, this.currentKitchenName),
				path: this.activeInvite.sharePath,
				imageUrl: buildInviteShareImageURL(this.activeInvite)
			}
		}

		return {
			title: '来看看我们的数字空间',
			path: '/pages/index/index'
		}
	},
	computed: {
		currentMealLabel() {
			return this.mealTabs.find((tab) => tab.value === this.activeMealType)?.label || '早餐'
		},
		currentStatusLabel() {
			return this.statusMap[this.activeStatus]?.label || '全部'
		},
		libraryHeaderTitle() {
			return this.isLibraryMealOrderMode ? formatMealOrderHeaderTitle(this.mealOrderDate) : '美食库'
		},
		libraryHeaderSummary() {
			if (this.isLibraryMealOrderMode) {
				return ''
			}
			return this.librarySummary
		},
		wishlistRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'wishlist')
		},
		canSwitchKitchen() {
			return this.kitchenOptions.length > 1
		},
		isKitchenConnected() {
			return !!this.currentKitchenName
		},
		kitchenConnectionLabel() {
			return this.isKitchenConnected ? '已连接' : '未连接'
		},
		currentKitchenDisplayName() {
			if (this.currentKitchenName) {
				return replaceKitchenLabel(this.currentKitchenName)
			}
			return this.isSyncing ? '正在获取空间信息' : replaceKitchenLabel(this.syncErrorMessage || '暂时无法连接空间')
		},
		currentKitchenRoleLabel() {
			if (this.currentKitchenRole === 'owner') return '创建者'
			if (this.currentKitchenRole === 'admin') return '管理员'
			if (this.currentKitchenRole === 'member') return '成员'
			return ''
		},
		currentKitchenMetaText() {
			if (!this.currentKitchenName) {
				return this.isSyncing ? '正在同步空间信息' : replaceKitchenLabel(this.syncErrorMessage || '创建或加入一个空间后，会显示在这里。')
			}

			if (this.canSwitchKitchen) {
				return '点击这张卡片，可以切换到其他空间。'
			}
			return '邀请成员后，大家会看到同一份菜单。'
		},
		doneRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'done')
		},
		trimmedSearchKeyword() {
			return String(this.searchKeyword || '').trim()
		},
		hasSearchKeyword() {
			return !!this.trimmedSearchKeyword
		},
		mealOrderDateStart() {
			return toISODate(new Date())
		},
		mealOrderDatePickerValue() {
			return normalizeMealOrderDate(this.mealOrderDate) || this.mealOrderDateStart
		},
		mealOrderDateText() {
			return formatMealOrderDateText(this.mealOrderDate)
		},
		mealOrderDateStatusMetaMap() {
			const result = {}

			Object.values(this.mealOrderStore?.drafts || {})
				.map((draft) => normalizeMealOrderDraft(draft, draft?.planDate))
				.filter((draft) => draft.planDate && draft.items.length)
				.forEach((draft) => {
					result[draft.planDate] = {
						tag: '草稿中',
						text: `已选 ${draft.items.length} 道`,
						tone: 'draft'
					}
				})

			;(Array.isArray(this.mealOrderStore?.submitted) ? this.mealOrderStore.submitted : [])
				.map((record) => normalizeMealOrderRecord(record))
				.filter(Boolean)
				.forEach((record) => {
					if (result[record.planDate]) {
						result[record.planDate] = {
							tag: '待修改',
							text: `草稿 ${result[record.planDate].text.replace('已选 ', '')} · 原安排保留`,
							tone: 'editing'
						}
						return
					}
					result[record.planDate] = {
						tag: '已安排',
						text: `已安排 ${record.items.length} 道`,
						tone: 'submitted'
					}
				})

			return result
		},
		mealOrderQuickDateOptions() {
			const today = this.mealOrderDateStart
			const options = [
				{ label: '今天', value: today },
				{ label: '明天', value: addDaysFromISODate(today, 1) },
				{ label: '周末', value: nextWeekendISODate(today) }
			]
			const seen = new Set()
			return options
				.filter((option) => {
					if (!option.value || seen.has(option.value)) return false
					seen.add(option.value)
					return true
				})
				.map((option) => ({
					...option,
					dateText: formatMealOrderDateText(option.value),
					statusTag: this.mealOrderDateStatusMetaMap[option.value]?.tag || '',
					statusText: this.mealOrderDateStatusMetaMap[option.value]?.text || '',
					statusTone: this.mealOrderDateStatusMetaMap[option.value]?.tone || ''
				}))
		},
		mealOrderCurrentDraft() {
			const date = normalizeMealOrderDate(this.mealOrderDate)
			if (!date) {
				return normalizeMealOrderDraft({}, '')
			}
			return normalizeMealOrderDraft(this.mealOrderStore?.drafts?.[date], date)
		},
		mealOrderCartItems() {
			const recipeMap = this.recipes.reduce((result, recipe) => {
				result[recipe.id] = recipe
				return result
			}, {})
			return this.mealOrderCurrentDraft.items.map((item) => {
				const recipe = recipeMap[item.recipeId] || {}
				const title = item.titleSnapshot || recipe.title || '未命名菜品'
				const mealType = item.mealTypeSnapshot || recipe.mealType || 'main'
				const mealTypeLabel = mealTypeLabelMap[mealType] || '正餐'
				return {
					...item,
					title,
					mealTypeLabel,
					imageSnapshot: String(item.imageSnapshot || '').trim()
				}
			})
		},
		mealOrderDraftNote() {
			return String(this.mealOrderCurrentDraft.note || '')
		},
		mealOrderCartDishCount() {
			return this.mealOrderCartItems.length
		},
		mealOrderCanCheckout() {
			return this.mealOrderCartDishCount > 0 && !this.isSubmittingMealOrder
		},
		mealOrderFloatingTitle() {
			if (this.mealOrderCanCheckout) {
				return `已选 ${this.mealOrderCartDishCount} 道`
			}
			return '还没选菜'
		},
		mealOrderFloatingActionText() {
			return '去确认'
		},
		mealOrderCheckoutHelperText() {
			return '提交后，这天菜单会立即同步给空间成员。之后想改，我们会先带出草稿，不会直接覆盖原安排。'
		},
		isLibraryMealOrderMode() {
			return this.activeSection === 'library' && this.isMealOrderMode && !!normalizeMealOrderDate(this.mealOrderDate)
		},
		showMealOrderFloatingBar() {
			return this.isLibraryMealOrderMode
		},
		mealOrderSpotlightRecords() {
			const today = this.mealOrderDateStart
			const drafts = Object.values(this.mealOrderStore?.drafts || {})
				.map((draft) => normalizeMealOrderDraft(draft, draft?.planDate))
				.filter((draft) => draft.planDate && draft.items.length)
				.map((draft) => ({
					id: `draft:${draft.planDate}`,
					type: 'draft',
					planDate: draft.planDate,
					items: draft.items,
					note: draft.note
				}))
			const submitted = (Array.isArray(this.mealOrderStore?.submitted) ? this.mealOrderStore.submitted : [])
				.map((record) => normalizeMealOrderRecord(record))
				.filter(Boolean)
				.map((record) => ({
					id: `submitted:${record.planDate}`,
					type: 'submitted',
					planDate: record.planDate,
					items: record.items,
					note: record.note
				}))
			const allRecords = [...drafts, ...submitted]
			const sortRecords = (left, right) => {
				const byDate = String(left.planDate || '').localeCompare(String(right.planDate || ''))
				if (byDate) return byDate
				if (left.type === right.type) return 0
				return left.type === 'draft' ? -1 : 1
			}
			const upcoming = allRecords
				.filter((record) => record.planDate >= today)
				.sort(sortRecords)
			const fallback = allRecords
				.filter((record) => record.planDate < today)
				.sort((left, right) => String(right.planDate || '').localeCompare(String(left.planDate || '')))
			return [...upcoming, ...fallback]
		},
		mealOrderSpotlightRecordIndex() {
			const total = this.mealOrderSpotlightRecords.length
			if (!total) return 0
			const current = Number(this.mealOrderSpotlightIndex) || 0
			return Math.min(Math.max(current, 0), total - 1)
		},
		mealOrderSpotlightRecord() {
			return this.mealOrderSpotlightRecords[this.mealOrderSpotlightRecordIndex] || null
		},
		mealOrderSpotlightDateText() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return ''
			return formatMealOrderDateParts(record.planDate).dateText
		},
		mealOrderSpotlightWeekday() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return ''
			return formatMealOrderDateParts(record.planDate).weekday
		},
		mealOrderSpotlightStatusText() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return ''
			return record.type === 'submitted' ? '已安排' : '草稿'
		},
		mealOrderSpotlightStatusKind() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return ''
			return record.type === 'submitted' ? 'submitted' : 'draft'
		},
		mealOrderSpotlightDesc() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return '点右侧安排菜单，先挑一天'
			return buildMealOrderDishSummary(record.items)
		},
		mealOrderSpotlightMetaText() {
			const total = this.mealOrderSpotlightRecords.length
			if (total < 2) return ''
			return `${this.mealOrderSpotlightRecordIndex + 1}/${total}`
		},
		mealOrderSuccessRecord() {
			return this.findMealOrderSubmittedByDate(this.mealOrderLastSubmittedDate)
		},
		mealOrderSuccessDateText() {
			return formatMealOrderDateText(this.mealOrderLastSubmittedDate)
		},
		mealOrderSuccessDishCount() {
			return Array.isArray(this.mealOrderSuccessRecord?.items) ? this.mealOrderSuccessRecord.items.length : 0
		},
		mealOrderSuccessDishSummary() {
			const record = this.mealOrderSuccessRecord
			if (!record) return '这天的菜单已经排好了。'
			return buildMealOrderDishSummary(record.items)
		},
		mealOrderSuccessNote() {
			return String(this.mealOrderSuccessRecord?.note || '').trim()
		},
		librarySummary() {
			if (!this.currentKitchenName && this.syncErrorMessage) {
				return this.syncErrorMessage
			}
			return this.isSyncing ? '正在同步这份菜单。' : '按餐别整理，想吃和吃过更清楚'
		},
		inviteActionDescription() {
			return this.showInviteShareAction ? '复制邀请码或直接分享给朋友' : '复制邀请码发给朋友'
		},
		invitePreparingText() {
			return this.showInviteShareAction ? '很快就好，生成后就能直接发给微信好友。' : '很快就好，生成后就能复制邀请码发给朋友。'
		},
		memberPanelSummary() {
			if (!this.currentKitchenName && this.isSyncing) {
				return '同步中'
			}
			if (this.isLoadingKitchenMembers) {
				return '加载中'
			}
			if (!this.kitchenMembers.length) {
				return '等待成员加入'
			}
			return `${this.kitchenMembers.length} 位成员`
		},
		visibleKitchenMembers() {
			return this.kitchenMembers.slice(0, 3)
		},
		hasMoreKitchenMembers() {
			return this.kitchenMembers.length > this.visibleKitchenMembers.length
		},
		filteredRecipes() {
			const keyword = this.trimmedSearchKeyword.toLowerCase()
			return this.recipes.filter((recipe) => {
				const matchedMealType = this.isLibraryMealOrderMode ? true : recipe.mealType === this.activeMealType
				const matchedStatus = this.isLibraryMealOrderMode ? true : this.activeStatus === 'all' || recipe.status === this.activeStatus
				const matchedKeyword = !keyword || buildRecipeSearchText(recipe).includes(keyword)
				return matchedMealType && matchedStatus && matchedKeyword
			})
		},
		recipeCards() {
			return this.filteredRecipes.map((recipe) => buildRecipeCard(recipe, this.cachedRecipeCoverMap))
		},
		randomPickRecipe() {
			return this.recipes.find((recipe) => recipe.id === this.randomPickRecipeId) || null
		},
		randomPickCard() {
			if (!this.randomPickRecipe) return null
			return buildRecipeCard(this.randomPickRecipe, this.cachedRecipeCoverMap)
		},
		randomPickCoverSrc() {
			return this.randomPickCard ? this.getRecipeCardDisplayCover(this.randomPickCard) : ''
		},
		randomPickCanReroll() {
			return this.randomPickPoolRecipeIds.length > 1
		},
		randomPickRevealKey() {
			return `${this.randomPickRecipeId || 'idle'}:${this.randomPickTick}`
		},
		recipeStatusFeedbackKey() {
			return `${this.recipeStatusFeedbackTone || 'idle'}:${this.recipeStatusFeedbackTick}`
		},
		searchAssistKeywords() {
			const keyword = this.trimmedSearchKeyword
			const recentKeywords = this.recentSearches
				.filter((item) => item !== keyword)
				.slice(0, 4)
			if (recentKeywords.length) {
				return recentKeywords
			}

			return (searchSuggestionKeywordsByMeal[this.activeMealType] || searchSuggestionKeywordsByMeal.main)
				.filter((item) => item !== keyword)
				.slice(0, 4)
		},
		searchAssistLabel() {
			const recentKeywords = this.recentSearches
				.filter((item) => item !== this.trimmedSearchKeyword)
				.slice(0, 4)
			return recentKeywords.length ? '最近搜索' : '可以试试'
		},
		searchPlaceholderText() {
			return this.isLibraryMealOrderMode ? '搜索菜名' : '搜菜名 / 食材'
		},
		showSearchAssist() {
			return this.isSearchFocused && !this.hasSearchKeyword && this.searchAssistKeywords.length > 0
		},
		currentFilterSummary() {
			const parts = [this.currentMealLabel]
			if (this.activeStatus !== 'all') {
				parts.push(this.currentStatusLabel)
			}
			if (this.hasSearchKeyword) {
				parts.push(`搜“${this.trimmedSearchKeyword}”`)
			}
			parts.push(`${this.filteredRecipes.length}道`)
			return parts.join(' · ')
		},
		canResetLibraryFilters() {
			return this.activeStatus !== 'all' || this.hasSearchKeyword
		},
		emptyStateKind() {
			if (this.hasSearchKeyword) return 'search-no-results'
			if (this.activeStatus !== 'all') return 'status-empty'
			return 'meal-empty'
		},
		emptyStateTitle() {
			if (this.hasSearchKeyword) {
				return `没有找到“${this.trimmedSearchKeyword}”`
			}
			if (this.activeStatus === 'all') {
				return `还没有${this.currentMealLabel}记录`
			}
			return `${this.currentMealLabel}里还没有${this.currentStatusLabel}的菜`
		},
		emptyStateDesc() {
			if (this.hasSearchKeyword) {
				if (this.searchAssistKeywords.length) {
					return `试试搜 ${this.searchAssistKeywords.join('、')}，或者换个关键词。`
				}
				return '试试换个关键词，或者点下方按钮新增一道菜。'
			}
			if (this.activeStatus === 'all') {
				return `试试切换到另一类餐别，或者点下方按钮新增一道${this.currentMealLabel}。`
			}
			return `可以先把${this.currentMealLabel}里的菜标记为${this.currentStatusLabel}，或者点下方按钮回到全部。`
		},
		emptyStatePrimaryText() {
			if (this.hasSearchKeyword) return '添加这道菜'
			if (this.activeStatus !== 'all') return '查看全部'
			return '添加这道菜'
		},
		emptyStatePrimaryIcon() {
			if (this.hasSearchKeyword) return 'plus'
			if (this.activeStatus !== 'all') return 'list-dot'
			return 'plus'
		},
		emptyStateSecondaryText() {
			if (this.hasSearchKeyword) return '清除搜索'
			if (this.activeStatus !== 'all') return '添加这道菜'
			return ''
		},
		inviteSheetSubtitle() {
			if (!this.currentKitchenName) {
				return '发给朋友后，对方输入邀请码即可加入。'
			}
			return `邀请朋友加入「${replaceKitchenLabel(this.currentKitchenName)}」`
		},
		showInviteShareAction() {
			return !!appConfig.inviteShareEnabled
		},
		inviteExpiresText() {
			if (!this.activeInvite?.expiresAt) return '--'
			const raw = this.activeInvite.expiresAt.replace(/\+\d{2}:\d{2}$/, '')
			const normalized = raw.includes('T') ? raw : raw.replace(' ', 'T')
			const expiresAt = new Date(normalized)
			if (Number.isNaN(expiresAt.getTime())) {
				return raw.replace('T', ' ').slice(5, 16)
			}
			const month = String(expiresAt.getMonth() + 1).padStart(2, '0')
			const day = String(expiresAt.getDate()).padStart(2, '0')
			const hours = String(expiresAt.getHours()).padStart(2, '0')
			const minutes = String(expiresAt.getMinutes()).padStart(2, '0')
			return `${month}-${day} ${hours}:${minutes}`
		},
		inviteRemainingUsesText() {
			if (!this.activeInvite) return '--'
			return `${this.activeInvite.remainingUses} 人`
		},
		formattedActiveInviteCode() {
			return formatInviteCode(this.activeInvite?.code || '') || '--'
		},
		inviteMetaLine() {
			if (!this.activeInvite) return '--'
			return `${this.inviteRemainingUsesText} 可加入 · ${this.inviteExpiresText} 过期`
		},
		profileAvatarPreview() {
			return this.profileDraft.avatarUrl || this.currentUser?.avatarUrl || ''
		},
		profileSheetTitle() {
			return this.profileSheetMode === 'edit' ? '个人资料' : '完善资料'
		},
		profileSheetSubtitle() {
			return this.profileSheetMode === 'edit'
				? '修改头像和昵称后，空间成员会更容易认出你。'
				: '设置头像和昵称后，空间成员会更容易认出你。'
		},
		profileSheetSecondaryActionText() {
			return this.profileSheetMode === 'edit' ? '取消' : '暂不设置'
		},
		profileAvatarFallback() {
			const name = (this.profileDraft.nickname || this.currentUser?.nickname || '厨友').trim()
			return name.slice(0, 1) || '厨'
		},
		canSubmitProfile() {
			return !!String(this.profileDraft.nickname || '').trim() || !!String(this.profileDraft.avatarUrl || '').trim()
		},
		canSubmitInviteCode() {
			return !!normalizeInviteCode(this.inviteCodeInput)
		},
		canSubmitDraft() {
			return !!this.draft.title.trim()
		},
		draftLinkPlatformLabel() {
			if (this.draftLinkPreviewPlatform === 'bilibili') return 'B 站'
			if (this.draftLinkPreviewPlatform === 'xiaohongshu') return '小红书'
			return '链接'
		},
		draftLinkTitleSourceLabel() {
			if (this.draftLinkPreviewTitleSource === 'ai') return 'AI 识别'
			if (this.draftLinkPreviewTitleSource === 'rule') return '规则识别'
			return ''
		},
		draftTitleAssistText() {
			if (!this.draftAutoTitle) return ''
			const platformLabel = this.draftLinkPlatformLabel
			const sourceLabel = this.draftLinkTitleSourceLabel
			const sourceParts = [platformLabel !== '链接' ? platformLabel : '', sourceLabel].filter(Boolean)
			const sourceSuffix = sourceParts.length ? `（${sourceParts.join(' · ')}）` : ''
			if (this.draftTitleTouched) {
				return `已识别菜名，保留当前填写${sourceSuffix}`.trim()
			}
			return `已识别菜名，可直接保存${sourceSuffix}`.trim()
		},
		draftLinkAssistText() {
			if (this.isDraftLinkPreviewing) {
				return this.draftLinkPlatformLabel === '链接'
					? '正在识别链接标题...'
					: `正在识别${this.draftLinkPlatformLabel}菜名...`
			}
			if (this.draftLinkPreviewError) {
				return this.draftLinkPreviewError
			}
			if (this.draft.link.trim()) {
				if (this.draftLinkPrefillSource === 'clipboard') {
					return '已带入剪贴板分享内容，保存时会原样保留。'
				}
				return '已粘贴来源内容，系统会自动补标题。'
			}
			return ''
		}
	},
	watch: {
		activeSection(next, prev) {
			if (next === prev) return
			if (next !== 'library') {
				this.clearRecipeStatusFeedback()
				this.closeRandomPickSheet()
			} else {
				this.triggerLibraryEnterMotion()
			}
		},
		isLibraryMealOrderMode(next, prev) {
			if (next === prev) return
			this.queueMealOrderModeMotion(next ? 'entering' : 'leaving')
			this.bumpRecipeListMotion()
		}
	},
	methods: {
		clearRecipeStatusFeedbackTimer() {
			if (!this.recipeStatusFeedbackTimer) return
			clearTimeout(this.recipeStatusFeedbackTimer)
			this.recipeStatusFeedbackTimer = null
		},
		clearRecipeStatusFeedback() {
			this.clearRecipeStatusFeedbackTimer()
			this.recipeStatusFeedbackVisible = false
			this.recipeStatusFeedbackTone = ''
			this.recipeStatusFeedbackTitle = ''
			this.recipeStatusFeedbackRecipeTitle = ''
			this.recipeStatusFeedbackShowSparkles = false
		},
		buildTonightPickPool() {
			const visible = this.filteredRecipes.slice()
			if (!visible.length) return []
			if (this.activeStatus === 'all') {
				const wishlistVisible = visible.filter((recipe) => recipe.status === 'wishlist')
				if (wishlistVisible.length) return wishlistVisible
			}
			return visible
		},
		buildTonightPickContext(pool = []) {
			if (!pool.length) return ''
			if (this.hasSearchKeyword) {
				return `根据“${this.trimmedSearchKeyword}”挑了一道`
			}
			if (this.activeStatus !== 'all') {
				return `从${this.currentMealLabel}的${this.currentStatusLabel}里挑了一道`
			}
			if (pool.every((recipe) => recipe.status === 'wishlist')) {
				return `先从${this.currentMealLabel}里想吃的菜里挑了一道`
			}
			return `从${this.currentMealLabel}里挑了一道`
		},
		pickTonightRecipe(pool = [], excludeRecipeId = '') {
			const recipes = Array.isArray(pool) ? pool.filter(Boolean) : []
			if (!recipes.length) return null
			if (recipes.length === 1) return recipes[0]
			const targetExcludeId = String(excludeRecipeId || '').trim()
			const candidates = targetExcludeId ? recipes.filter((recipe) => recipe.id !== targetExcludeId) : recipes
			const source = candidates.length ? candidates : recipes
			return source[Math.floor(Math.random() * source.length)] || null
		},
		presentTonightPick(recipe = null, pool = [], contextText = '', motionMode = 'enter') {
			if (!recipe?.id) return
			this.randomPickRecipeId = recipe.id
			this.randomPickPoolRecipeIds = pool.map((item) => item.id).filter(Boolean)
			this.randomPickContextText = contextText
			this.randomPickMotionMode = motionMode === 'swap' ? 'swap' : 'enter'
			this.randomPickTick += 1
			this.showRandomPickSheet = true
			try {
				uni.vibrateShort({
					type: 'light'
				})
			} catch (_) {
				// Ignore unsupported vibration capabilities and keep the picker path stable.
			}
		},
		closeRandomPickSheet() {
			this.showRandomPickSheet = false
			this.randomPickRecipeId = ''
			this.randomPickContextText = ''
			this.randomPickPoolRecipeIds = []
			this.randomPickMotionMode = 'enter'
		},
		rerollTonightPick() {
			const pool = this.randomPickPoolRecipeIds
				.map((recipeId) => this.recipes.find((recipe) => recipe.id === recipeId))
				.filter(Boolean)
			if (pool.length < 2) return
			const picked = this.pickTonightRecipe(pool, this.randomPickRecipeId)
			this.presentTonightPick(picked, pool, this.randomPickContextText, 'swap')
		},
		openRandomPickDetail(recipeId = '') {
			const targetRecipeId = String(recipeId || this.randomPickRecipeId || '').trim()
			if (!targetRecipeId) return
			this.closeRandomPickSheet()
			setTimeout(() => {
				this.openRecipeDetail(targetRecipeId)
			}, 140)
		},
		playRecipeStatusHaptic(nextStatus = 'wishlist') {
			const vibrationType = nextStatus === 'done' ? 'medium' : 'light'
			try {
				uni.vibrateShort({
					type: vibrationType
				})
			} catch (_) {
				try {
					uni.vibrateShort()
				} catch (__) {
					// Ignore unsupported vibration capabilities to keep the toggle path stable.
				}
			}
		},
		showLibraryActionFeedback(options = {}) {
			const tone = String(options?.tone || 'done').trim() || 'done'
			const title = String(options?.title || '').trim()
			const description = String(options?.description || '').trim()
			const duration = Math.max(900, Number(options?.duration) || 1440)
			const showSparkles = !!options?.showSparkles
			if (!title) return
			this.clearRecipeStatusFeedbackTimer()
			this.recipeStatusFeedbackTone = tone
			this.recipeStatusFeedbackTitle = title
			this.recipeStatusFeedbackRecipeTitle = description
			this.recipeStatusFeedbackShowSparkles = showSparkles
			this.recipeStatusFeedbackVisible = true
			this.recipeStatusFeedbackTick += 1
			this.recipeStatusFeedbackTimer = setTimeout(() => {
				this.recipeStatusFeedbackVisible = false
				this.recipeStatusFeedbackTimer = null
			}, duration)
		},
		showRecipeStatusFeedback(recipe = {}, nextStatus = 'wishlist') {
			const tone = nextStatus === 'done' ? 'done' : 'wishlist'
			this.showLibraryActionFeedback({
				tone,
				title: tone === 'done' ? '已标记吃过' : '已改回想吃',
				description: String(recipe?.title || '').trim() || '这道菜',
				duration: tone === 'done' ? 1680 : 1440,
				showSparkles: tone === 'done'
			})
		},
		clearMealOrderModeMotionTimer() {
			if (!this.mealOrderModeMotionTimer) return
			clearTimeout(this.mealOrderModeMotionTimer)
			this.mealOrderModeMotionTimer = null
		},
		queueMealOrderModeMotion(state = '') {
			const nextState = state === 'leaving' ? 'leaving' : 'entering'
			this.clearMealOrderModeMotionTimer()
			this.mealOrderModeMotionState = nextState
			this.mealOrderModeMotionTimer = setTimeout(() => {
				this.mealOrderModeMotionState = ''
				this.mealOrderModeMotionTimer = null
			}, nextState === 'leaving' ? 220 : 260)
		},
		bumpRecipeListMotion() {
			this.recipeListMotionTick += 1
		},
		handleMealTypeTabChange(value) {
			if (!this.mealTabs.some((tab) => tab.value === value) || this.activeMealType === value) return
			const oldIdx = this.mealTabs.findIndex((tab) => tab.value === this.activeMealType)
			const newIdx = this.mealTabs.findIndex((tab) => tab.value === value)
			this.triggerToolbarBounce(newIdx > oldIdx ? 'right' : 'left')
			this.activeMealType = value
			this.bumpRecipeListMotion()
		},
		handleStatusTabChange(value) {
			if (!this.statusTabs.some((tab) => tab.value === value) || this.activeStatus === value) return
			this.activeStatus = value
			this.bumpRecipeListMotion()
		},
		handleEmptyStatePrimary() {
			if (this.hasSearchKeyword) {
				this.openAddSheet()
				return
			}
			if (this.activeStatus !== 'all') {
				this.handleStatusTabChange('all')
				return
			}
			this.openAddSheet()
		},
		handleEmptyStateSecondary() {
			if (this.hasSearchKeyword) {
				this.searchKeyword = ''
				return
			}
			if (this.activeStatus !== 'all') {
				this.openAddSheet()
			}
		},
		triggerToolbarBounce(direction = 'right') {
			const cls = `toolbar--bounce-${direction === 'left' ? 'left' : 'right'}`
			this.toolbarBounceClass = ''
			if (this.toolbarBounceTimer) {
				clearTimeout(this.toolbarBounceTimer)
				this.toolbarBounceTimer = null
			}
			this.$nextTick(() => {
				this.toolbarBounceClass = cls
				this.toolbarBounceTimer = setTimeout(() => {
					this.toolbarBounceClass = ''
					this.toolbarBounceTimer = null
				}, 160)
			})
		},
		triggerLibraryEnterMotion() {
			if (this.activeSection !== 'library') return
			if (this.libraryEnterMotionTimer) {
				clearTimeout(this.libraryEnterMotionTimer)
				this.libraryEnterMotionTimer = null
			}
			this.libraryEnterMotionActive = false
			this.$nextTick(() => {
				this.libraryEnterMotionActive = true
				this.libraryEnterMotionTimer = setTimeout(() => {
					this.libraryEnterMotionActive = false
					this.libraryEnterMotionTimer = null
				}, 240)
			})
		},
		bumpMealOrderSpotlightMotion(direction = 'next') {
			this.mealOrderSpotlightMotionDirection = direction === 'previous' ? 'previous' : 'next'
			this.mealOrderSpotlightMotionTick += 1
		},
		setRecipeStatusPending(recipeId = '', pending = false) {
			const targetRecipeId = String(recipeId || '').trim()
			if (!targetRecipeId) return
			const nextPendingMap = {
				...this.recipeStatusPendingMap
			}
			if (pending) {
				nextPendingMap[targetRecipeId] = true
			} else {
				delete nextPendingMap[targetRecipeId]
			}
			this.recipeStatusPendingMap = nextPendingMap
		},
		patchLocalRecipeById(recipeId = '', updater = null) {
			const targetRecipeId = String(recipeId || '').trim()
			if (!targetRecipeId || typeof updater !== 'function') {
				return {
					found: false,
					previousRecipe: null,
					nextRecipe: null
				}
			}

			let previousRecipe = null
			let nextRecipe = null
			let changed = false
			const nextRecipes = this.recipes.map((recipe) => {
				if (recipe.id !== targetRecipeId) return recipe
				previousRecipe = recipe
				nextRecipe = updater(recipe)
				if (!nextRecipe) {
					nextRecipe = recipe
					return recipe
				}
				changed = nextRecipe !== recipe
				return nextRecipe
			})

			if (changed) {
				this.recipes = nextRecipes
			}

			return {
				found: !!previousRecipe,
				previousRecipe,
				nextRecipe
			}
		},
		applyRecipes(recipes = []) {
			this.recipes = Array.isArray(recipes) ? recipes : []
			this.recipeCardCoverFallbackMap = {}
			this.recipeCardHiddenMap = {}
			this.syncRecipeCoverCache(this.recipes)
		},
		getRecipeCardDisplayCover(card = {}) {
			const recipeId = String(card?.id || '').trim()
			if (recipeId && this.recipeCardHiddenMap[recipeId]) return ''
			if (recipeId && this.recipeCardCoverFallbackMap[recipeId]) {
				return String(card?.remoteCover || '').trim()
			}
			return String(card?.cover || '').trim()
		},
		async handleRecipeCardImageError(card = {}) {
			const recipeId = String(card?.id || '').trim()
			if (!recipeId) return

			const displayedCover = this.getRecipeCardDisplayCover(card)
			const cachedCover = String(card?.cachedCover || '').trim()
			const remoteCover = String(card?.remoteCover || '').trim()

			if (
				cachedCover &&
				remoteCover &&
				displayedCover === cachedCover &&
				cachedCover !== remoteCover &&
				!this.recipeCardCoverFallbackMap[recipeId]
			) {
				this.recipeCardCoverFallbackMap = {
					...this.recipeCardCoverFallbackMap,
					[recipeId]: true
				}

				if (this.cachedRecipeCoverMap[recipeId]) {
					const nextCoverMap = { ...this.cachedRecipeCoverMap }
					delete nextCoverMap[recipeId]
					this.cachedRecipeCoverMap = nextCoverMap
				}

				try {
					await invalidateCachedImage(remoteCover, card.coverVersion)
				} catch (error) {
					// Ignore cache cleanup failures and keep the UI fallback path usable.
				}
				return
			}

			if (this.recipeCardHiddenMap[recipeId]) return
			this.recipeCardHiddenMap = {
				...this.recipeCardHiddenMap,
				[recipeId]: true
			}
		},
		switchSection(nextSection = 'library') {
			const targetSection = String(nextSection || '').trim()
			if (!targetSection || targetSection === this.activeSection) return
			if (!this.isMealOrderMode || this.activeSection !== 'library' || targetSection === 'library') {
				this.activeSection = targetSection
				return
			}

			uni.showModal({
				title: '离开点菜模式',
				content: '当前菜单草稿会自动保存，确认先离开吗？',
				confirmText: '确认离开',
				success: ({ confirm }) => {
					if (!confirm) return
					this.syncMealOrderDraft({ silent: true })
					this.activeSection = targetSection
				}
			})
		},
		applyMealOrderStore(store = createEmptyMealOrderStore()) {
			const normalizedStore = normalizeMealOrderStore(store)
			this.mealOrderStore = normalizedStore
			if (
				this.mealOrderLastSubmittedDate &&
				!(Array.isArray(normalizedStore.submitted) ? normalizedStore.submitted : [])
					.some((record) => normalizeMealOrderDate(record?.planDate) === this.mealOrderLastSubmittedDate)
			) {
				this.mealOrderLastSubmittedDate = ''
				this.showMealOrderSuccessSheet = false
			}

			const normalizedDate = normalizeMealOrderDate(this.mealOrderDate)
			if (normalizedDate && normalizedStore.drafts[normalizedDate]) {
				this.mealOrderDate = normalizedDate
				return
			}

			const availableDraftDates = Object.keys(normalizedStore.drafts).sort((left, right) => left.localeCompare(right))
			if (availableDraftDates.length && (!this.isMealOrderMode || !normalizedDate)) {
				this.mealOrderDate = availableDraftDates[0]
				return
			}

			if (!this.isMealOrderMode) {
				this.mealOrderDate = ''
			}
		},
		async loadMealOrderStore(options = {}) {
			const { silent = true } = options
			const kitchenId = Number(getCurrentKitchenId()) || 0
			if (!kitchenId) {
				this.mealOrderStoreLoadedKitchenId = 0
				this.applyMealOrderStore(createEmptyMealOrderStore())
				return createEmptyMealOrderStore()
			}

			const requestID = this.mealOrderStoreRequestID + 1
			this.mealOrderStoreRequestID = requestID
			const contextID = this.mealOrderSyncContextID
			const localVersion = this.mealOrderLocalVersion

			try {
				const store = await listMealPlanStore(kitchenId)
				if (
					requestID !== this.mealOrderStoreRequestID ||
					contextID !== this.mealOrderSyncContextID ||
					localVersion !== this.mealOrderLocalVersion ||
					kitchenId !== Number(this.currentKitchenId)
				) {
					return normalizeMealOrderStore(this.mealOrderStore)
				}
				this.applyMealOrderStore(store)
				this.mealOrderStoreLoadedKitchenId = kitchenId
				return store
			} catch (error) {
				if (!silent) {
					uni.showToast({
						title: error?.message || '加载菜单失败',
						icon: 'none'
					})
				}
				return normalizeMealOrderStore(this.mealOrderStore)
			}
		},
		clearMealOrderDraftSyncTimer() {
			if (!this.mealOrderDraftSyncTimer) return
			clearTimeout(this.mealOrderDraftSyncTimer)
			this.mealOrderDraftSyncTimer = null
		},
		resetMealOrderState() {
			this.clearMealOrderDraftSyncTimer()
			this.clearMealOrderModeMotionTimer()
			this.clearRecipeStatusFeedback()
			this.mealOrderStore = createEmptyMealOrderStore()
			this.mealOrderDate = ''
			this.isMealOrderMode = false
			this.mealOrderModeMotionState = ''
			this.showMealOrderDateSheet = false
			this.showMealOrderCartSheet = false
			this.showMealOrderCheckoutSheet = false
			this.showMealOrderSuccessSheet = false
			this.mealOrderLastSubmittedDate = ''
			this.mealOrderSpotlightIndex = 0
			this.mealOrderSpotlightMotionDirection = ''
			this.mealOrderSpotlightMotionTick = 0
			this.mealOrderSpotlightTouchStartX = 0
			this.mealOrderSpotlightTouchStartY = 0
			this.mealOrderSpotlightSuppressTap = false
		},
		stageMealOrderDraft(updater) {
			const date = normalizeMealOrderDate(this.mealOrderDate)
			if (!date || typeof updater !== 'function') return

			const current = normalizeMealOrderDraft(this.mealOrderStore?.drafts?.[date], date)
			const nextRawDraft = updater({
				...current,
				items: current.items.map((item) => ({ ...item }))
			})
			const nextDraft = normalizeMealOrderDraft(nextRawDraft, date)
			const nextDrafts = {
				...(this.mealOrderStore?.drafts || {})
			}

			if (!nextDraft.items.length && !String(nextDraft.note || '').trim()) {
				delete nextDrafts[date]
			} else {
				nextDrafts[date] = {
					...nextDraft,
					updatedAt: new Date().toISOString()
				}
			}

			this.mealOrderStore = {
				...(this.mealOrderStore || createEmptyMealOrderStore()),
				drafts: nextDrafts
			}
			this.mealOrderLocalVersion += 1
		},
		scheduleMealOrderDraftSync(delay = 0) {
			const date = normalizeMealOrderDate(this.mealOrderDate)
			if (!date || !getCurrentKitchenId()) return
			this.clearMealOrderDraftSyncTimer()
			this.mealOrderDraftSyncTimer = setTimeout(() => {
				this.mealOrderDraftSyncTimer = null
				this.syncMealOrderDraft({ silent: true })
			}, Math.max(0, Number(delay) || 0))
		},
		async syncMealOrderDraft(options = {}) {
			const { silent = false } = options
			if (this.isSubmittingMealOrder) return null
			const kitchenId = Number(getCurrentKitchenId()) || 0
			const date = normalizeMealOrderDate(this.mealOrderDate)
			if (!kitchenId || !date) return null

			this.clearMealOrderDraftSyncTimer()
			const localVersion = this.mealOrderLocalVersion
			const contextID = this.mealOrderSyncContextID
			const draft = normalizeMealOrderDraft(this.mealOrderStore?.drafts?.[date], date)

			try {
				const store = await saveMealPlanDraft(kitchenId, date, buildMealPlanPayload(draft))
				if (
					localVersion === this.mealOrderLocalVersion &&
					contextID === this.mealOrderSyncContextID &&
					kitchenId === Number(this.currentKitchenId)
				) {
					this.applyMealOrderStore(store)
					this.mealOrderStoreLoadedKitchenId = kitchenId
				}
				return store
			} catch (error) {
				if (!silent) {
					uni.showToast({
						title: error?.message || '保存菜单失败',
						icon: 'none'
					})
				}
				return null
			}
		},
		buildMealOrderItemFromRecipe(recipe = {}) {
			const recipeId = String(recipe.id || '').trim()
			if (!recipeId) return null
			const image = (extractRecipeImages(recipe) || [])[0] || ''
			return {
				recipeId,
				quantity: 1,
				titleSnapshot: String(recipe.title || '').trim() || '未命名菜品',
				imageSnapshot: String(image || '').trim(),
				mealTypeSnapshot: String(recipe.mealType || '').trim() || 'main'
			}
		},
		findMealOrderSubmittedByDate(planDate = '') {
			const normalizedDate = normalizeMealOrderDate(planDate)
			if (!normalizedDate) return null
			const submitted = (Array.isArray(this.mealOrderStore?.submitted) ? this.mealOrderStore.submitted : [])
				.map((record) => normalizeMealOrderRecord(record))
				.filter(Boolean)
			return submitted.find((record) => record.planDate === normalizedDate) || null
		},
		focusMealOrderSpotlightRecord(planDate = '', type = 'submitted') {
			const normalizedDate = normalizeMealOrderDate(planDate)
			if (!normalizedDate) return false
			const currentIndex = this.mealOrderSpotlightRecordIndex
			const targetIndex = this.mealOrderSpotlightRecords.findIndex(
				(record) => record.planDate === normalizedDate && record.type === type
			)
			if (targetIndex < 0) return false
			if (targetIndex !== currentIndex) {
				this.bumpMealOrderSpotlightMotion(targetIndex > currentIndex ? 'next' : 'previous')
			}
			this.mealOrderSpotlightIndex = targetIndex
			return true
		},
		mealOrderHasRecipe(recipeId = '') {
			const targetRecipeId = String(recipeId || '').trim()
			if (!targetRecipeId) return false
			return this.mealOrderCurrentDraft.items.some((item) => item.recipeId === targetRecipeId)
		},
		handleMealOrderSpotlightTap() {
			if (this.mealOrderSpotlightSuppressTap) {
				this.mealOrderSpotlightSuppressTap = false
				return
			}
			const record = this.mealOrderSpotlightRecord
			if (!record) {
				this.openMealOrderDateSheet()
				return
			}
			this.openMealOrderDetail(record)
		},
		handleMealOrderSpotlightTouchStart(event) {
			const touch = event?.touches?.[0] || event?.changedTouches?.[0]
			if (!touch) return
			this.mealOrderSpotlightTouchStartX = Number(touch.clientX || touch.pageX || 0)
			this.mealOrderSpotlightTouchStartY = Number(touch.clientY || touch.pageY || 0)
			this.mealOrderSpotlightSuppressTap = false
		},
		handleMealOrderSpotlightTouchEnd(event) {
			const touch = event?.changedTouches?.[0] || event?.touches?.[0]
			const startX = Number(this.mealOrderSpotlightTouchStartX || 0)
			const startY = Number(this.mealOrderSpotlightTouchStartY || 0)
			this.mealOrderSpotlightTouchStartX = 0
			this.mealOrderSpotlightTouchStartY = 0
			if (!touch || this.mealOrderSpotlightRecords.length < 2 || (!startX && !startY)) return

			const endX = Number(touch.clientX || touch.pageX || 0)
			const endY = Number(touch.clientY || touch.pageY || 0)
			const diffX = endX - startX
			const diffY = endY - startY
			if (Math.abs(diffX) < 56 || Math.abs(diffX) <= Math.abs(diffY)) return

			this.shiftMealOrderSpotlight(diffX > 0 ? 'next' : 'previous')
			this.mealOrderSpotlightSuppressTap = true
		},
		shiftMealOrderSpotlight(direction = 'next') {
			const total = this.mealOrderSpotlightRecords.length
			if (total < 2) return
			const step = direction === 'previous' ? -1 : 1
			this.mealOrderSpotlightIndex = (this.mealOrderSpotlightRecordIndex + step + total) % total
			this.bumpMealOrderSpotlightMotion(direction)
		},
		closeMealOrderSuccessSheet() {
			this.showMealOrderSuccessSheet = false
		},
		viewMealOrderSuccessRecord() {
			this.showMealOrderSuccessSheet = false
			this.openMealOrderDetail({
				planDate: this.mealOrderLastSubmittedDate,
				type: 'submitted'
			})
		},
		planNextMealOrder() {
			this.showMealOrderSuccessSheet = false
			this.showMealOrderDateSheet = true
		},
		drawTonight() {
			const pool = this.buildTonightPickPool()
			if (!pool.length) {
				uni.showToast({
					title: this.hasSearchKeyword || this.activeStatus !== 'all' ? '当前筛选里还没有可选菜' : '先添加几道菜吧',
					icon: 'none'
				})
				return
			}
			const picked = this.pickTonightRecipe(pool)
			this.presentTonightPick(picked, pool, this.buildTonightPickContext(pool), 'enter')
		},
		openMealOrderDateSheet() {
			if (!getCurrentKitchenId()) {
				uni.showToast({
					title: '请先完成空间同步',
					icon: 'none'
				})
				return
			}
			this.showMealOrderSuccessSheet = false
			this.showMealOrderDateSheet = true
		},
		closeMealOrderDateSheet() {
			this.showMealOrderDateSheet = false
		},
		startMealOrderMode(planDate = '') {
			const normalizedDate = normalizeMealOrderDate(planDate)
			if (!normalizedDate) return
			this.mealOrderDate = normalizedDate
			this.activeSection = 'library'
			this.isMealOrderMode = true
			this.showMealOrderDateSheet = false
			this.showMealOrderSuccessSheet = false
		},
		exitMealOrderMode() {
			this.syncMealOrderDraft({ silent: true })
			this.isMealOrderMode = false
			this.showMealOrderCartSheet = false
			this.showMealOrderCheckoutSheet = false
			this.showMealOrderSuccessSheet = false
		},
		addMealOrderRecipe(recipe = {}) {
			if (!this.isMealOrderMode || !this.mealOrderDate) {
				this.openMealOrderDateSheet()
				return
			}
			const nextItem = this.buildMealOrderItemFromRecipe(recipe)
			if (!nextItem) return
			this.stageMealOrderDraft((draft) => {
				const nextItems = [...draft.items]
				const index = nextItems.findIndex((item) => item.recipeId === nextItem.recipeId)
				if (index < 0) {
					nextItems.push(nextItem)
				} else {
					nextItems[index] = {
						...nextItems[index],
						titleSnapshot: nextItem.titleSnapshot,
						imageSnapshot: nextItem.imageSnapshot,
						mealTypeSnapshot: nextItem.mealTypeSnapshot
					}
				}
				return {
					...draft,
					items: nextItems
				}
			})
			this.scheduleMealOrderDraftSync()
		},
		toggleMealOrderRecipe(recipe = {}) {
			const recipeId = String(recipe?.id || '').trim()
			if (!recipeId) return
			if (this.mealOrderHasRecipe(recipeId)) {
				this.removeMealOrderRecipe(recipeId)
				uni.showToast({
					title: '已移出这天菜单',
					icon: 'none'
				})
				return
			}
			this.addMealOrderRecipe(recipe)
			uni.showToast({
				title: '已加入这天菜单',
				icon: 'none'
			})
		},
		removeMealOrderRecipe(recipeId = '') {
			const targetRecipeId = String(recipeId || '').trim()
			if (!targetRecipeId || !this.mealOrderDate) return
			this.stageMealOrderDraft((draft) => {
				const nextItems = draft.items.filter((item) => item.recipeId !== targetRecipeId)
				return {
					...draft,
					items: nextItems
				}
			})
			this.scheduleMealOrderDraftSync()
		},
		openMealOrderCartSheet() {
			if (!this.isMealOrderMode || !this.mealOrderDate) {
				this.openMealOrderDateSheet()
				return
			}
			this.showMealOrderCartSheet = true
		},
		closeMealOrderCartSheet() {
			this.showMealOrderCartSheet = false
		},
		openMealOrderCheckoutSheet() {
			if (!this.mealOrderCanCheckout) return
			this.showMealOrderCartSheet = false
			this.showMealOrderCheckoutSheet = true
		},
		closeMealOrderCheckoutSheet() {
			this.showMealOrderCheckoutSheet = false
		},
		handleMealOrderNoteInput(event) {
			const value = String(event?.detail?.value || '')
			this.stageMealOrderDraft((draft) => ({
				...draft,
				note: value
			}))
			this.scheduleMealOrderDraftSync(320)
		},
		clearMealOrderCart() {
			if (!this.mealOrderCartItems.length && !String(this.mealOrderDraftNote || '').trim()) return
			uni.showModal({
				title: '清空菜单',
				content: '确认清空这一天已经安排的菜单吗？',
				confirmText: '清空',
				success: ({ confirm }) => {
					if (!confirm) return
					this.stageMealOrderDraft((draft) => ({
						...draft,
						items: [],
						note: ''
					}))
					this.scheduleMealOrderDraftSync()
				}
			})
		},
		async submitMealOrder() {
			if (!this.mealOrderCanCheckout || !this.mealOrderDate || this.isSubmittingMealOrder) return
			const kitchenId = Number(getCurrentKitchenId()) || 0
			if (!kitchenId) return
			const currentDraft = normalizeMealOrderDraft(this.mealOrderCurrentDraft, this.mealOrderDate)
			this.clearMealOrderDraftSyncTimer()
			const contextID = this.mealOrderSyncContextID + 1
			this.mealOrderSyncContextID = contextID
			this.isSubmittingMealOrder = true

			try {
				const store = await submitMealPlanRequest(kitchenId, this.mealOrderDate, buildMealPlanPayload(currentDraft))
				if (contextID !== this.mealOrderSyncContextID || kitchenId !== Number(this.currentKitchenId)) {
					return
				}
				this.applyMealOrderStore(store)
				this.mealOrderStoreLoadedKitchenId = kitchenId
				this.showMealOrderCheckoutSheet = false
				this.showMealOrderCartSheet = false
				this.isMealOrderMode = false
				this.mealOrderLastSubmittedDate = this.mealOrderDate
				this.focusMealOrderSpotlightRecord(this.mealOrderDate, 'submitted')
				this.showMealOrderSuccessSheet = true
			} catch (error) {
				uni.showToast({
					title: error?.message || '提交菜单失败',
					icon: 'none'
				})
			} finally {
				this.isSubmittingMealOrder = false
			}
		},
		clearSearchBlurTimer() {
			if (!this.searchBlurTimer) return
			clearTimeout(this.searchBlurTimer)
			this.searchBlurTimer = null
		},
		handleSearchFocus() {
			this.clearSearchBlurTimer()
			this.isSearchFocused = true
		},
		handleSearchBlur() {
			this.clearSearchBlurTimer()
			this.searchBlurTimer = setTimeout(() => {
				this.isSearchFocused = false
				this.searchBlurTimer = null
			}, 120)
			this.rememberSearchKeyword()
		},
		handleSearchConfirm() {
			this.rememberSearchKeyword()
		},
		rememberSearchKeyword() {
			const keyword = this.trimmedSearchKeyword
			if (!keyword) return

			const nextKeywords = [keyword, ...this.recentSearches.filter((item) => item !== keyword)].slice(0, MAX_RECENT_SEARCHES)
			this.recentSearches = nextKeywords
			writeRecentSearches(nextKeywords)
		},
		applySearchKeyword(keyword = '') {
			const nextKeyword = String(keyword || '').trim()
			if (!nextKeyword) return

			this.clearSearchBlurTimer()
			this.searchKeyword = nextKeyword
			this.isSearchFocused = false
			this.rememberSearchKeyword()
			this.bumpRecipeListMotion()
		},
		clearSearchKeyword() {
			this.searchKeyword = ''
			this.clearSearchBlurTimer()
			this.isSearchFocused = true
			this.bumpRecipeListMotion()
		},
		buildRecipeCoverCacheEntries(recipes = []) {
			return (Array.isArray(recipes) ? recipes : [])
				.map((recipe) => {
					const images = extractRecipeImages(recipe)
					const cover = images[0] || ''
					const version = buildRecipeCoverVersion(recipe)
					if (!cover || !recipe.id) return null
					return {
						recipeId: recipe.id,
						url: cover,
						version,
						cacheKey: buildImageCacheKey(cover, version)
					}
				})
				.filter(Boolean)
		},
		async syncRecipeCoverCache(recipes = []) {
			const entries = this.buildRecipeCoverCacheEntries(recipes)
			const requestID = this.recipeCoverCacheRequestID + 1
			this.recipeCoverCacheRequestID = requestID

			if (!entries.length) {
				this.cachedRecipeCoverMap = {}
				this.recipeCardCoverFallbackMap = {}
				this.recipeCardHiddenMap = {}
				return
			}

			const cachedEntries = await Promise.all(
				entries.map(async (entry) => ({
					recipeId: entry.recipeId,
					localPath: await getCachedImagePath(entry.url, entry.version)
				}))
			)

			if (requestID !== this.recipeCoverCacheRequestID) return

			const nextCoverMap = {}
			const nextFallbackMap = { ...this.recipeCardCoverFallbackMap }
			const nextHiddenMap = { ...this.recipeCardHiddenMap }
			cachedEntries.forEach((entry) => {
				if (!entry.localPath) return
				nextCoverMap[entry.recipeId] = entry.localPath
				delete nextFallbackMap[entry.recipeId]
				delete nextHiddenMap[entry.recipeId]
			})
			this.cachedRecipeCoverMap = nextCoverMap
			this.recipeCardCoverFallbackMap = nextFallbackMap
			this.recipeCardHiddenMap = nextHiddenMap

			const recipeIdsByCacheKey = entries.reduce((result, entry) => {
				if (!result[entry.cacheKey]) {
					result[entry.cacheKey] = []
				}
				result[entry.cacheKey].push(entry.recipeId)
				return result
			}, {})

			warmImageCache(entries, {
				concurrency: 2,
				onResolved: ({ cacheKey, localPath }) => {
					if (requestID !== this.recipeCoverCacheRequestID || !localPath) return
					const recipeIds = recipeIdsByCacheKey[cacheKey] || []
					if (!recipeIds.length) return

					let changed = false
					const updatedCoverMap = { ...this.cachedRecipeCoverMap }
					const updatedFallbackMap = { ...this.recipeCardCoverFallbackMap }
					const updatedHiddenMap = { ...this.recipeCardHiddenMap }
					recipeIds.forEach((recipeId) => {
						if (updatedCoverMap[recipeId] === localPath) return
						updatedCoverMap[recipeId] = localPath
						delete updatedFallbackMap[recipeId]
						delete updatedHiddenMap[recipeId]
						changed = true
					})

					if (changed) {
						this.cachedRecipeCoverMap = updatedCoverMap
						this.recipeCardCoverFallbackMap = updatedFallbackMap
						this.recipeCardHiddenMap = updatedHiddenMap
					}
				}
			})
		},
		applySession(session = getSessionSnapshot()) {
			const snapshot = session || getSessionSnapshot()
			const previousKitchenId = Number(this.currentKitchenId) || 0
			this.currentUser = snapshot?.user || null
			this.kitchenOptions = Array.isArray(snapshot?.kitchens) ? snapshot.kitchens : []
			this.currentKitchenName = snapshot?.currentKitchen?.name || ''
			this.currentKitchenRole = snapshot?.currentKitchen?.role || ''
			const nextKitchenId = Number(snapshot?.currentKitchenId) || 0
			this.currentKitchenId = nextKitchenId
			if (nextKitchenId !== this.kitchenMembersKitchenId) {
				this.kitchenMembers = []
				this.kitchenMembersKitchenId = nextKitchenId
			}
			if (previousKitchenId !== nextKitchenId) {
				this.mealOrderSyncContextID += 1
				this.mealOrderStoreLoadedKitchenId = 0
				this.mealOrderLocalVersion += 1
				this.resetMealOrderState()
			}
			if (!nextKitchenId) {
				this.mealOrderStoreLoadedKitchenId = 0
				this.resetMealOrderState()
			} else if (this.mealOrderStoreLoadedKitchenId !== nextKitchenId) {
				this.loadMealOrderStore({ silent: true })
			}
			this.activeInvite = null
			this.inviteCodeCopied = false
			this.maybePromptProfile()
		},
		async refreshRecipes(options = {}) {
			const { silent = true } = options
			const cachedRecipes = getCachedRecipes()
			this.applyRecipes(cachedRecipes)

			try {
				this.isSyncing = true
				const session = await ensureSession()
				this.syncErrorMessage = ''
				this.applySession(session)
				const kitchenId = getCurrentKitchenId()
				const [recipes] = await Promise.all([
					loadRecipes({ forceRefresh: true }),
					this.refreshKitchenMembers({ kitchenId, silent: true })
				])
				this.applyRecipes(recipes)
				await this.applyPendingMealOrderAction(kitchenId)
			} catch (error) {
				this.syncErrorMessage = getFriendlySessionErrorMessage(error)
				this.applySession()
				this.applyRecipes(getCachedRecipes())
				this.kitchenMembers = []
				this.kitchenMembersKitchenId = 0
				if (!silent) {
					uni.showToast({
						title: error?.message || '同步失败',
						icon: 'none'
					})
				}
			} finally {
				this.isSyncing = false
			}
		},
		async applyPendingMealOrderAction(kitchenId = 0) {
			const action = consumePendingMealOrderAction()
			if (!action) return
			if (Number(action.kitchenId) && Number(action.kitchenId) !== Number(kitchenId)) return

			if (kitchenId) {
				await this.loadMealOrderStore({ silent: true })
			}

			if (action.kind === 'resume' && action.planDate) {
				this.startMealOrderMode(action.planDate)
			}

			if (action.message) {
				this.showLibraryActionFeedback({
					tone: 'done',
					title: action.message,
					description: action.planDate ? formatMealOrderDateText(action.planDate) : '',
					duration: 1560,
					showSparkles: false
				})
			}
		},
		memberRoleLabel(role) {
			if (role === 'owner') return '创建者'
			if (role === 'admin') return '管理员'
			if (role === 'member') return '成员'
			return '成员'
		},
		memberDisplayName(member = {}) {
			return member.nickname || `厨友 ${member.userId || ''}`.trim()
		},
		memberInitial(member = {}) {
			const name = this.memberDisplayName(member)
			return name.slice(0, 1)
		},
		memberMemberDescription(member = {}) {
			if (member.isCurrentUser) {
				return '你正在维护这个空间。'
			}
			return '已加入这个共享空间。'
		},
		handleMemberCardTap(member = {}) {
			if (!member.isCurrentUser || !this.currentUser?.id) return
			this.openProfileSheetWithMode('edit')
		},
		openAboutPage() {
			uni.navigateTo({
				url: '/pages/about/index'
			})
		},
		openProfileSheetWithMode(mode = 'prompt') {
			this.profileSheetMode = mode === 'edit' ? 'edit' : 'prompt'
			this.profileDraft = {
				nickname: !isPlaceholderNickname(this.currentUser?.nickname) ? this.currentUser.nickname : '',
				avatarUrl: ''
			}
			this.showProfileSheet = true
		},
		resetProfileDraft() {
			this.profileDraft = {
				nickname: '',
				avatarUrl: ''
			}
		},
		maybePromptProfile() {
			if (appConfig.authMode !== 'wechat') return
			if (this.hasDismissedProfilePrompt || this.showProfileSheet) return
			if (!this.currentUser?.id) return
			if (!isProfileIncomplete(this.currentUser)) return
			this.openProfileSheetWithMode('prompt')
		},
		closeProfileSheet() {
			this.showProfileSheet = false
			this.profileSheetMode = 'prompt'
			this.hasDismissedProfilePrompt = true
			this.resetProfileDraft()
		},
		handleChooseAvatar(event) {
			const avatarUrl = String(event?.detail?.avatarUrl || '').trim()
			if (!avatarUrl) return
			this.profileDraft.avatarUrl = avatarUrl
		},
		handleProfileNicknameInput(event) {
			this.profileDraft.nickname = String(event?.detail?.value || '').trim()
		},
		async submitProfile(event) {
			if (this.isSubmittingProfile || !this.canSubmitProfile) return

			const submittedNickname = String(event?.detail?.value?.nickname || this.profileDraft.nickname || '').trim()
			this.isSubmittingProfile = true

			try {
				const session = await ensureSession()
				this.applySession(session)

				const avatarUrl = await ensureUploadedImage(this.profileDraft.avatarUrl)
				const user = await saveCurrentUserProfile({
					nickname: submittedNickname,
					avatarUrl
				})
				if (!user) {
					throw new Error('当前后端暂不支持保存资料')
				}
				let nextSession = null
				try {
					nextSession = await ensureSession()
				} catch (error) {
					// Keep the saved profile result even if the follow-up session refresh fails.
				}
				this.showProfileSheet = false
				this.profileSheetMode = 'prompt'
				this.hasDismissedProfilePrompt = true
				this.resetProfileDraft()
				this.applySession(nextSession || getSessionSnapshot())
				await this.refreshKitchenMembers({ silent: true })
				uni.showToast({
					title: '资料已更新',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '保存资料失败',
					icon: 'none'
				})
			} finally {
				this.isSubmittingProfile = false
			}
		},
		async refreshKitchenMembers(options = {}) {
			const { kitchenId = getCurrentKitchenId(), silent = true } = options
			const targetKitchenId = Number(kitchenId) || 0
			if (!targetKitchenId) {
				this.kitchenMembers = []
				this.kitchenMembersKitchenId = 0
				return []
			}

			this.isLoadingKitchenMembers = true

			try {
				const items = await listKitchenMembers(targetKitchenId)
				if (targetKitchenId === getCurrentKitchenId()) {
					this.kitchenMembers = items
					this.kitchenMembersKitchenId = targetKitchenId
				}
				return items
			} catch (error) {
				if (targetKitchenId === getCurrentKitchenId()) {
					this.kitchenMembers = []
					this.kitchenMembersKitchenId = targetKitchenId
				}
				if (!silent) {
					uni.showToast({
						title: error?.message || '获取成员失败',
						icon: 'none'
					})
				}
				return []
			} finally {
				if (targetKitchenId === getCurrentKitchenId()) {
					this.isLoadingKitchenMembers = false
				}
			}
		},
		createDraftFromContext() {
			const defaultStatus = ['wishlist', 'done'].includes(this.activeStatus) ? this.activeStatus : 'wishlist'
			return createEmptyDraft({
				mealType: this.activeMealType || 'breakfast',
				status: defaultStatus
			})
		},
		resetDraftAssistState() {
			this.clearDraftLinkPreviewState()
			this.draftAutoTitle = ''
			this.draftTitleTouched = false
			this.draftLinkPreviewPlatform = ''
			this.draftLinkPreviewTitleSource = ''
			this.draftLinkPreviewError = ''
			this.draftLinkPrefillSource = ''
		},
		readClipboardText() {
			return new Promise((resolve) => {
				uni.getClipboardData({
					success: (result) => {
						resolve(String(result?.data || '').trim())
					},
					fail: () => {
						resolve('')
					}
				})
			})
		},
		async tryAutoPrefillDraftLinkFromClipboard(requestID = 0) {
			try {
				const clipboardText = String(await this.readClipboardText() || '').trim()
				const detectedLink = extractSupportedDraftLink(clipboardText)
				if (!detectedLink) return false
				if (!clipboardText || clipboardText === this.lastDraftLinkPrefill) return false
				if (!this.showAddSheet || requestID !== this.draftClipboardPrefillRequestID) return false
				if (String(this.draft.link || '').trim()) return false

				this.draft.link = clipboardText
				this.draftLinkPrefillSource = 'clipboard'
				this.lastDraftLinkPrefill = clipboardText
				writeLastDraftLinkPrefill(clipboardText)

				const guessedTitle = guessDraftTitleFromShareText(clipboardText)
				if (guessedTitle) {
					this.applyDraftAutoTitle(guessedTitle)
				}
				this.scheduleDraftLinkPreview(clipboardText)
				return true
			} catch (_) {
				return false
			}
		},
		clearDraftLinkPreviewState() {
			if (this.draftLinkPreviewTimer) {
				clearTimeout(this.draftLinkPreviewTimer)
				this.draftLinkPreviewTimer = null
			}
			this.draftLinkPreviewRequestID += 1
			this.isDraftLinkPreviewing = false
			this.draftLinkPreviewTitleSource = ''
		},
		applyDraftAutoTitle(title = '') {
			const normalizedTitle = normalizeDraftAutoTitle(title)
			if (!normalizedTitle) return

			const currentTitle = String(this.draft.title || '').trim()
			const previousAutoTitle = String(this.draftAutoTitle || '').trim()
			const canReplace = !currentTitle || !this.draftTitleTouched || (previousAutoTitle && currentTitle === previousAutoTitle)

			this.draftAutoTitle = normalizedTitle
			if (canReplace) {
				this.draft.title = normalizedTitle
				this.draftTitleTouched = false
			}
		},
		handleDraftTitleInput(event) {
			const value = String(event?.detail?.value || '')
			this.draft.title = value

			const normalizedTitle = value.trim()
			if (!normalizedTitle) {
				this.draftTitleTouched = false
				return
			}

			const autoTitle = String(this.draftAutoTitle || '').trim()
			this.draftTitleTouched = autoTitle ? normalizedTitle !== autoTitle : true
		},
		handleDraftMealTypeSelect(value) {
			if (!this.mealTabs.some((tab) => tab.value === value)) return
			this.draft.mealType = value
		},
		handleDraftStatusSelect(value) {
			if (!this.draftStatusOptions.some((tab) => tab.value === value)) return
			this.draft.status = value
		},
		handleDraftNoteInput(event) {
			this.draft.note = String(event?.detail?.value || '')
		},
		handleDraftLinkInput(event) {
			const value = String(event?.detail?.value || '')
			this.draft.link = value
			this.draftLinkPrefillSource = ''
			this.scheduleDraftLinkPreview(value)
		},
		scheduleDraftLinkPreview(rawInput = '') {
			this.clearDraftLinkPreviewState()
			this.draftLinkPreviewError = ''

			const value = String(rawInput || '').trim()
			const previousAutoTitle = String(this.draftAutoTitle || '').trim()
			if (!value) {
				if (!this.draftTitleTouched && previousAutoTitle && String(this.draft.title || '').trim() === previousAutoTitle) {
					this.draft.title = ''
				}
				this.draftAutoTitle = ''
				this.draftLinkPreviewPlatform = ''
				this.draftLinkPreviewTitleSource = ''
				return
			}

			const platform = detectDraftLinkPlatform(value)
			this.draftLinkPreviewPlatform = platform

			const guessedTitle = guessDraftTitleFromShareText(value)
			if (guessedTitle) {
				this.applyDraftAutoTitle(guessedTitle)
			}

			const mayContainShareLink = /https?:\/\/|www\.|bilibili|b23\.tv|bili2233\.cn|xiaohongshu|xhslink/i.test(value)
			if (!platform && !mayContainShareLink) {
				if (!guessedTitle && !this.draftTitleTouched && previousAutoTitle && String(this.draft.title || '').trim() === previousAutoTitle) {
					this.draft.title = ''
					this.draftAutoTitle = ''
				}
				return
			}

			const requestID = this.draftLinkPreviewRequestID
			this.isDraftLinkPreviewing = true
			this.draftLinkPreviewTimer = setTimeout(async () => {
					try {
						const result = await previewRecipeLink(value)
						if (requestID !== this.draftLinkPreviewRequestID) return

						this.isDraftLinkPreviewing = false
						this.draftLinkPreviewTimer = null
						const resolvedLink = String(result?.canonicalUrl || result?.link || '').trim()
						this.draftLinkPreviewPlatform = detectDraftLinkPlatform(resolvedLink || value) || platform
						this.draftLinkPreviewTitleSource = String(result?.titleSource || '').trim().toLowerCase()

						const previewTitle = normalizeDraftAutoTitle(result?.title || '')
					if (previewTitle) {
						this.applyDraftAutoTitle(previewTitle)
						return
					}

					if (!guessedTitle) {
						this.draftLinkPreviewError = '暂时没识别到菜名，可继续手动填写。'
					}
				} catch (error) {
					if (requestID !== this.draftLinkPreviewRequestID) return
					this.isDraftLinkPreviewing = false
					this.draftLinkPreviewTimer = null
					if (!guessedTitle) {
						this.draftLinkPreviewError = error?.message || '暂时无法识别链接标题，可先手动填写。'
					}
				}
			}, 480)
		},
		mealTypeCount(type) {
			return this.recipes.filter((recipe) => recipe.mealType === type).length
		},
		statusCount(status) {
			const list = this.recipes.filter((recipe) => recipe.mealType === this.activeMealType)
			if (status === 'all') return list.length
			return list.filter((recipe) => recipe.status === status).length
		},
		resetLibraryFilters() {
			this.activeStatus = 'all'
			this.searchKeyword = ''
			this.clearSearchBlurTimer()
			this.isSearchFocused = false
			this.bumpRecipeListMotion()
		},
		openRecipeDetail(recipeId) {
			uni.navigateTo({
				url: `/pages/recipe-detail/index?id=${recipeId}`
			})
		},
		openMealOrderDetail(record = {}) {
			const planDate = normalizeMealOrderDate(record?.planDate || '')
			const type = String(record?.type || '').trim() === 'draft' ? 'draft' : 'submitted'
			if (!planDate) {
				uni.showToast({
					title: '这份菜单暂时打不开',
					icon: 'none'
				})
				return
			}
			uni.navigateTo({
				url: `/pages/meal-plan-detail/index?planDate=${encodeURIComponent(planDate)}&type=${type}`
			})
		},
		openMealOrderRecipeDetail(item = {}) {
			const recipeId = String(item?.recipeId || '').trim()
			if (!recipeId) {
				uni.showToast({
					title: '这道菜暂时打不开',
					icon: 'none'
				})
				return
			}
			this.openRecipeDetail(recipeId)
		},
		nextStatusText(status) {
			return status === 'done' ? '标记想吃' : '标记吃过'
		},
		toggleRecipeStatus(recipeId) {
			this.toggleRecipeStatusAsync(recipeId)
		},
		async toggleRecipeStatusAsync(recipeId) {
			const targetRecipeId = String(recipeId || '').trim()
			if (!targetRecipeId || this.recipeStatusPendingMap[targetRecipeId]) return

			const currentRecipe = this.recipes.find((recipe) => recipe.id === targetRecipeId)
			if (!currentRecipe) return

			const nextStatus = currentRecipe.status === 'done' ? 'wishlist' : 'done'
			this.setRecipeStatusPending(targetRecipeId, true)
			const optimisticUpdate = this.patchLocalRecipeById(targetRecipeId, (recipe) => ({
				...recipe,
				status: nextStatus
			}))
			if (!optimisticUpdate.found || !optimisticUpdate.previousRecipe) {
				this.setRecipeStatusPending(targetRecipeId, false)
				return
			}

			this.playRecipeStatusHaptic(nextStatus)

			try {
				const updatedRecipe = await toggleRecipeStatusById(targetRecipeId)
				this.patchLocalRecipeById(targetRecipeId, (recipe) => ({
					...recipe,
					...updatedRecipe
				}))
				this.showRecipeStatusFeedback(
					updatedRecipe || optimisticUpdate.nextRecipe,
					updatedRecipe?.status || nextStatus
				)
			} catch (error) {
				this.patchLocalRecipeById(targetRecipeId, () => optimisticUpdate.previousRecipe)
				uni.showToast({
					title: error?.message || '更新状态失败',
					icon: 'none'
				})
			} finally {
				this.setRecipeStatusPending(targetRecipeId, false)
			}
		},
		openAddSheet() {
			this.resetDraftAssistState()
			this.draft = this.createDraftFromContext()
			this.showAddSheet = true
			this.draftClipboardPrefillRequestID += 1
			this.tryAutoPrefillDraftLinkFromClipboard(this.draftClipboardPrefillRequestID)
		},
		openDietAssistantSheet() {
			this.showDietAssistantSheet = true
		},
		closeDietAssistantSheet() {
			this.showDietAssistantSheet = false
		},
		openAddSheetFromAssistant() {
			this.closeDietAssistantSheet()
			this.openAddSheet()
		},
		closeAddSheet() {
			if (this.isSubmittingDraft) return
			this.draftClipboardPrefillRequestID += 1
			this.resetDraftAssistState()
			this.showAddSheet = false
			this.draft = this.createDraftFromContext()
		},
		chooseDraftImages() {
			const remaining = Math.max(this.maxRecipeImages - this.draft.images.length, 0)
			if (!remaining) {
				uni.showToast({
					title: `最多上传 ${this.maxRecipeImages} 张`,
					icon: 'none'
				})
				return
			}

			uni.chooseImage({
				count: remaining,
				sizeType: ['compressed'],
				sourceType: ['album', 'camera'],
				success: ({ tempFilePaths }) => {
					if (!tempFilePaths || !tempFilePaths.length) return
					const nextImages = [...this.draft.images]
					tempFilePaths.forEach((path) => {
						if (path && !nextImages.includes(path) && nextImages.length < this.maxRecipeImages) {
							nextImages.push(path)
						}
					})
					this.draft.images = nextImages
				}
			})
		},
		removeDraftImage(index) {
			if (typeof index !== 'number') return
			this.draft.images = this.draft.images.filter((_, currentIndex) => currentIndex !== index)
		},
		previewDraftImages(index = 0) {
			const urls = Array.isArray(this.draft.images) ? this.draft.images.filter(Boolean) : []
			if (!urls.length) return
			uni.previewImage({
				current: urls[index] || urls[0],
				urls
			})
		},
		async submitDraft() {
			if (!this.canSubmitDraft || this.isSubmittingDraft) return

			this.isSubmittingDraft = true

			try {
				const newRecipe = await createRecipeFromDraft(this.draft)
				this.applyRecipes(getCachedRecipes())
				this.selectedRecipeId = newRecipe.id
				this.activeSection = 'library'
				this.activeMealType = newRecipe.mealType
				this.activeStatus = 'all'
				this.searchKeyword = ''
				this.showAddSheet = false
				this.resetDraftAssistState()
				this.draft = this.createDraftFromContext()
				try {
					uni.vibrateShort({
						type: 'light'
					})
				} catch (_) {
					// Ignore unsupported vibration capabilities and keep save stable.
				}
				this.showLibraryActionFeedback({
					tone: newRecipe.status === 'done' ? 'done' : 'wishlist',
					title: newRecipe.status === 'done' ? '已保存并标记吃过' : '已加入美食库',
					description: String(newRecipe?.title || '').trim() || '这道菜',
					duration: newRecipe.status === 'done' ? 1680 : 1440,
					showSparkles: newRecipe.status === 'done'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '保存失败',
					icon: 'none'
				})
			} finally {
				this.isSubmittingDraft = false
			}
		},
		async openInviteSheet() {
			if (!this.currentKitchenName) {
				await this.refreshRecipes({ silent: false })
			}

			if (!getCurrentKitchenId()) {
				uni.showToast({
					title: '还没拿到空间信息',
					icon: 'none'
				})
				return
			}

			this.showInviteSheet = true
			const canReuseInvite =
				this.activeInvite &&
				Number(this.activeInvite.kitchenId) === Number(getCurrentKitchenId()) &&
				this.activeInvite.status === 'active'
			if (!canReuseInvite) {
				await this.prepareInvite()
			}
		},
		closeInviteSheet() {
			this.showInviteSheet = false
			this.inviteCodeCopied = false
		},
		openInviteCodeSheet() {
			this.inviteCodeInput = ''
			this.showInviteCodeSheet = true
		},
		closeInviteCodeSheet() {
			this.showInviteCodeSheet = false
			this.inviteCodeInput = ''
		},
		openKitchenNameSheet() {
			if (!getCurrentKitchenId()) {
				uni.showToast({
					title: '还没拿到空间信息',
					icon: 'none'
				})
				return
			}

			this.promptKitchenName()
		},
		promptKitchenName() {
			if (this.isSubmittingKitchenName) return

			uni.showModal({
				title: '修改空间名',
				editable: true,
				content: this.currentKitchenName || '',
				placeholderText: '输入空间名称',
				confirmText: '保存',
				cancelText: '取消',
				success: async (result) => {
					if (!result?.confirm) return
					const submittedName = String(result?.content || '').trim()
					await this.submitKitchenName(submittedName)
				}
			})
		},
		async submitKitchenName(submittedName = '') {
			const nextName = String(submittedName || '').trim()
			if (this.isSubmittingKitchenName || !nextName) return

			this.isSubmittingKitchenName = true

			try {
				const kitchen = await updateKitchen(getCurrentKitchenId(), {
					name: nextName
				})
				if (!kitchen) {
					throw new Error('修改空间名失败')
				}

				const currentInvite = this.activeInvite
				const nextSession = updateSessionKitchen(kitchen)
				this.applySession(nextSession)
				if (Number(currentInvite?.kitchenId) === Number(kitchen.id)) {
					this.activeInvite = {
						...currentInvite,
						kitchenName: kitchen.name
					}
				}
				uni.showToast({
					title: '空间名已更新',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '修改空间名失败',
					icon: 'none'
				})
			} finally {
				this.isSubmittingKitchenName = false
			}
		},
		handleInviteCodeInput(event) {
			this.inviteCodeInput = formatInviteCode(event?.detail?.value || '')
		},
		async prepareInvite() {
			if (this.isPreparingInvite) return

			this.isPreparingInvite = true
			this.inviteCodeCopied = false
			this.activeInvite = null

			try {
				const invite = await createKitchenInvite(getCurrentKitchenId(), {})
				this.activeInvite = invite
			} catch (error) {
				uni.showToast({
					title: error?.message || '生成邀请失败',
					icon: 'none'
				})
			} finally {
				this.isPreparingInvite = false
			}
		},
		copyInviteCode() {
			if (!this.activeInvite?.code || this.isPreparingInvite) {
				uni.showToast({
					title: '请先生成邀请码',
					icon: 'none'
				})
				return
			}

			uni.setClipboardData({
				data: formatInviteCode(this.activeInvite.code),
				success: () => {
					this.inviteCodeCopied = true
					uni.showToast({
						title: '邀请码已复制',
						icon: 'none'
					})
				}
			})
		},
		regenerateInviteCode() {
			uni.showModal({
				title: '重新生成邀请码',
				content: '重新生成后，之前发出的邀请码会失效，是否继续？',
				confirmText: '重新生成',
				success: async ({ confirm }) => {
					if (!confirm) return
					await this.prepareInvite()
				}
			})
		},
		submitInviteCode() {
			const code = normalizeInviteCode(this.inviteCodeInput)
			if (!code) {
				uni.showToast({
					title: '请先输入邀请码',
					icon: 'none'
				})
				return
			}

			this.closeInviteCodeSheet()
			uni.navigateTo({
				url: `/pages/invite/index?code=${encodeURIComponent(code)}`
			})
		},
		showAllMembers() {
			if (!this.kitchenMembers.length) return

			uni.showActionSheet({
				itemList: this.kitchenMembers.map((member) => {
					const suffix = member.isCurrentUser ? ' · 你' : ''
					return `${this.memberDisplayName(member)} · ${this.memberRoleLabel(member.role)}${suffix}`
				})
			})
		},
		openKitchenSelector() {
			if (!this.kitchenOptions.length) return
			if (this.kitchenOptions.length <= 1) {
				uni.showToast({
					title: '当前只有一个空间',
					icon: 'none'
				})
				return
			}

			uni.showActionSheet({
				itemList: this.kitchenOptions.map((item) => item.name),
				success: async ({ tapIndex }) => {
					const nextKitchen = this.kitchenOptions[tapIndex]
					if (!nextKitchen || nextKitchen.id === getSessionSnapshot()?.currentKitchenId) return
					setCurrentKitchenId(nextKitchen.id)
					this.applySession()
					this.selectedRecipeId = ''
					this.searchKeyword = ''
					await this.refreshRecipes({ silent: false })
				}
			})
		}
	}
}
</script>

<style lang="scss">
	/* 美食库设计 token v1
	 * Phase A（2026-04-30）：抽取常用色为 token 别名，零视觉差异。
	 * Phase B（2026-04-30）：追加表面色 / 阴影 token，工具卡 / 状态 pill 起开始使用目标值，引入视觉升级。
	 * Phase D（2026-04-30）：新增 --color-text-muted 用于摘要兜底占位与空态描述弱文。
	 * Phase E（2026-04-30）：新增 --color-accent-terracotta（陶土橙）用于"想吃 / 固顶徽"双色渐变。
	 * 规范：docs/food-library-ui-redesign-plan-2026-04-30.md §3.1 / §3.2 */
	page {
		--color-bg: #f6f4f1;
		--color-surface: #fffdf8;
		--color-surface-warm: #f4ecdf;
		--color-text-primary: #2f2923;
		--color-text-on-brand: #fffaf3;
		--color-text-muted: #9f9387;
		--color-brand-brown: #5b4a3b;
		--color-accent-terracotta: #bf715f;
		--color-border-soft: rgba(91, 74, 59, 0.07);
		--color-border-active: rgba(91, 74, 59, 0.16);
		--shadow-clay-soft:
			0 12rpx 24rpx rgba(70, 54, 40, 0.05),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
		--shadow-clay-strong:
			0 14rpx 32rpx rgba(56, 44, 30, 0.06),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.7);
	}
</style>

<style lang="scss" scoped>
	.app-shell {
		min-height: 100vh;
		background: var(--color-bg);
	}

	.page-content {
		padding: 24rpx 24rpx 214rpx;
	}

	.library-shell {
		display: block;
		will-change: transform, opacity;
	}

	.library-shell--entering {
		animation: library-shell-enter 220ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
	}

	@keyframes library-shell-enter {
		from {
			opacity: 0;
			transform: translateY(8rpx);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.library-shell--entering {
			animation: none;
		}
	}

	.page-content--meal-order-entering {
		animation: page-content-meal-order-enter 260ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
	}

	.page-content--meal-order-leaving {
		animation: page-content-meal-order-leave 220ms ease both;
	}

	.page-content--meal-order {
		padding-bottom: 342rpx;
	}

	@keyframes page-content-meal-order-enter {
		from {
			opacity: 0.76;
			transform: translateY(14rpx);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	@keyframes page-content-meal-order-leave {
		from {
			opacity: 0.94;
			transform: translateY(-6rpx);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.app-footer-links {
		margin-top: 18rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 18rpx;
	}

	.app-footer-link {
		padding: 10rpx 18rpx 0;
		opacity: 0.82;
	}

	.app-footer-link__label {
		font-size: 22rpx;
		font-weight: 600;
		color: #7b6c5f;
		letter-spacing: 1rpx;
	}

	.toolbar {
		margin-top: 20rpx;
		padding: 22rpx;
		border-radius: 32rpx;
		background: var(--color-surface);
		border: 1px solid var(--color-border-soft);
		box-shadow: var(--shadow-clay-strong);
		transition: transform 0.18s ease;
	}

	.toolbar--bounce-left {
		animation: toolbar-bounce-left 140ms ease-out;
	}

	.toolbar--bounce-right {
		animation: toolbar-bounce-right 140ms ease-out;
	}

	@keyframes toolbar-bounce-left {
		0% { transform: translateX(0); }
		45% { transform: translateX(-8rpx); }
		100% { transform: translateX(0); }
	}

	@keyframes toolbar-bounce-right {
		0% { transform: translateX(0); }
		45% { transform: translateX(8rpx); }
		100% { transform: translateX(0); }
	}

	.page-content--meal-order .toolbar {
		margin-top: 14rpx;
		padding: 0;
		border-radius: 0;
		background: transparent;
		border: 0;
		box-shadow: none;
	}

	.toolbar__search-row {
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.filter-group {
		margin-top: 14rpx;
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.filter-group--compact {
		margin-top: 14rpx;
	}

	.meal-tabs {
		display: flex;
		gap: 16rpx;
	}

	.meal-tab {
		flex: 1;
		min-width: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 28rpx 26rpx;
		box-sizing: border-box;
		border-radius: 36rpx;
		background: rgba(249, 247, 242, 0.4);
		border: 2rpx solid transparent;
		transition: background 0.18s ease, border-color 0.18s ease;
	}

	.meal-tab--active {
		background: var(--color-surface-warm);
		border-color: rgba(91, 74, 59, 0.05);
	}

	.meal-tab__left {
		display: flex;
		align-items: center;
		gap: 10rpx;
		min-width: 0;
	}

	.meal-tab__text {
		font-size: 30rpx;
		font-weight: 700;
		color: rgba(92, 64, 51, 0.4);
	}

	.meal-tab__count {
		padding: 6rpx 14rpx;
		border-radius: 10rpx;
		background: rgba(255, 255, 255, 0.5);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.meal-tab__count-text {
		font-size: 24rpx;
		font-weight: 700;
		line-height: 1;
		color: rgba(92, 64, 51, 0.6);
	}

	.meal-tab--active .meal-tab__text {
		color: var(--color-brand-brown);
	}

	.meal-tab--active .meal-tab__count-text {
		color: var(--color-brand-brown);
	}

	.search-box {
		flex: 1;
		min-width: 0;
		height: 76rpx;
		display: flex;
		align-items: center;
		gap: 10rpx;
		padding: 0 20rpx;
		border-radius: 32rpx;
		background: rgba(249, 247, 242, 0.5);
		transition: background 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
	}

	.search-box__icon {
		width: 32rpx;
		height: 32rpx;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.search-box:active {
		transform: translateY(1rpx);
	}

	.search-box--active {
		background: rgba(249, 247, 242, 0.85);
		box-shadow: 0 0 0 3rpx rgba(91, 74, 59, 0.12);
	}

	.search-box__input {
		flex: 1;
		height: 76rpx;
		font-size: 28rpx;
		color: var(--color-text-primary);
	}

	.search-box__placeholder {
		color: rgba(92, 64, 51, 0.32);
	}

	.search-box__clear {
		width: 36rpx;
		height: 36rpx;
		border-radius: 999rpx;
		background: #f0ece6;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: transform 0.16s ease, background 0.16s ease;
	}

	.search-box__clear:active {
		transform: scale(0.92);
		background: #e7dfd5;
	}

	.page-content--meal-order .search-box {
		height: 72rpx;
		padding: 0 22rpx;
		border-radius: 22rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.84) 0%, rgba(255, 255, 255, 0) 44%),
			rgba(255, 255, 255, 0.98);
		border-color: var(--color-border-soft);
		box-shadow: 0 14rpx 22rpx rgba(56, 44, 30, 0.045);
	}

	.page-content--meal-order .search-box__input {
		height: 72rpx;
		font-size: 24rpx;
	}

	.search-assist {
		margin-top: 12rpx;
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.search-assist__label {
		min-height: 56rpx;
		display: inline-flex;
		align-items: center;
		font-size: 22rpx;
		font-weight: 600;
		line-height: 1;
		color: #8d847a;
		flex-shrink: 0;
	}

	.search-assist__chips {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10rpx;
	}

	.search-assist__chip {
		min-height: 56rpx;
		box-sizing: border-box;
		padding: 10rpx 16rpx;
		border-radius: 999rpx;
		background: #f2ede7;
		border: 1px solid rgba(91, 74, 59, 0.04);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.16s ease, background 0.16s ease, border-color 0.16s ease;
	}

	.search-assist__chip:active {
		transform: translateY(1rpx);
		background: #ece5dd;
		border-color: rgba(91, 74, 59, 0.08);
	}

	.search-assist__chip-text {
		font-size: 22rpx;
		line-height: 1;
		color: #6e6155;
	}

	.status-track {
		display: flex;
		align-items: stretch;
		gap: 14rpx;
	}

	.status-pill {
		flex: 1;
		min-width: 0;
		min-height: 72rpx;
		padding: 0 22rpx;
		box-sizing: border-box;
		border-radius: 28rpx;
		background: rgba(249, 247, 242, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8rpx;
		transition: background 0.16s ease, transform 0.16s ease;
	}

	.status-pill:active {
		transform: scale(0.98);
	}

	.status-pill--all {
		flex: 0 0 auto;
		padding: 0 36rpx;
		gap: 8rpx;
	}

	.status-pill__text {
		font-size: 30rpx;
		font-weight: 700;
		color: rgba(92, 64, 51, 0.4);
		line-height: 1;
	}

	.status-pill__count {
		padding: 6rpx 12rpx;
		border-radius: 10rpx;
		background: rgba(255, 255, 255, 0.5);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.status-pill__count-text {
		font-size: 24rpx;
		font-weight: 700;
		line-height: 1;
		color: rgba(92, 64, 51, 0.6);
	}

	.status-pill--active {
		background: var(--color-brand-brown);
	}

	.status-pill--active .status-pill__text {
		color: var(--color-text-on-brand);
	}

	.status-pill--active .status-pill__count {
		background: rgba(255, 255, 255, 0.18);
	}

	.status-pill--active .status-pill__count-text {
		color: var(--color-text-on-brand);
	}

	.list-caption {
		margin-top: 18rpx;
		padding: 0 2rpx;
	}

	.list-caption__top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.list-caption__title {
		flex: 1;
		min-width: 0;
		font-size: 23rpx;
		font-weight: 600;
		line-height: 1.35;
		color: #695d51;
		word-break: break-all;
	}

	.list-caption__actions {
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
		flex-shrink: 0;
	}

	.list-caption__clear {
		min-height: 48rpx;
		padding: 0 16rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.88);
		border: 1px solid rgba(91, 74, 59, 0.06);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.list-caption__clear:active {
		transform: scale(0.992);
	}

	.list-caption__clear-text {
		font-size: 21rpx;
		font-weight: 600;
		line-height: 1;
		color: #8a7b6e;
	}

	.list-caption__pick {
		min-height: 52rpx;
		padding: 0 18rpx 0 12rpx;
		border-radius: 999rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.76) 0%, rgba(255, 255, 255, 0) 44%),
			linear-gradient(180deg, #f6ede3 0%, #efe3d4 100%);
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow:
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62),
			0 8rpx 16rpx rgba(63, 52, 42, 0.06);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8rpx;
		flex-shrink: 0;
	}

	.list-caption__pick:active {
		transform: scale(0.992);
	}

	.list-caption__pick-icon-shell {
		width: 28rpx;
		height: 28rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.58);
		border: 1px solid rgba(111, 97, 84, 0.08);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.list-caption__pick-text {
		font-size: 21rpx;
		font-weight: 700;
		line-height: 1;
		color: #6a5848;
	}

	.recipe-list {
		margin-top: 14rpx;
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.stats-panel {
		margin-top: 16rpx;
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 12rpx;
	}

	.meal-panel-list {
		margin-top: 16rpx;
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.meal-panel {
		border-radius: 20rpx;
		background: rgba(255, 255, 255, 0.88);
		border: 1px solid rgba(0, 0, 0, 0.03);
		box-shadow: 0 8rpx 18rpx rgba(56, 44, 30, 0.04);
		padding: 18rpx;
	}

	.meal-panel__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.meal-panel__title {
		font-size: 28rpx;
		font-weight: 700;
		color: var(--color-text-primary);
	}

	.meal-panel__meta {
		font-size: 22rpx;
		color: #8d847a;
	}

	.meal-panel__block {
		margin-top: 14rpx;
	}

	.meal-panel__block-header {
		display: flex;
		align-items: center;
		gap: 8rpx;
		margin-bottom: 8rpx;
	}

	.meal-panel__block-title {
		font-size: 22rpx;
		font-weight: 600;
		color: #6d6257;
	}

	.stat-box,
	.simple-panel {
		border-radius: 20rpx;
		background: rgba(255, 255, 255, 0.88);
		border: 1px solid rgba(0, 0, 0, 0.03);
		box-shadow: 0 8rpx 18rpx rgba(56, 44, 30, 0.04);
	}

	.stat-box {
		padding: 22rpx 18rpx;
	}

	.stat-box__value {
		display: block;
		font-size: 36rpx;
		font-weight: 700;
		color: var(--color-text-primary);
	}

	.stat-box__label {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		color: #8d847a;
	}

	.simple-panel {
		margin-top: 14rpx;
		padding: 18rpx;
	}

	.simple-panel__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.simple-panel__title {
		font-size: 28rpx;
		font-weight: 700;
		color: var(--color-text-primary);
	}

	.simple-panel__meta {
		font-size: 22rpx;
		color: #8d847a;
	}

	.simple-list {
		margin-top: 12rpx;
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.simple-list__item {
		padding: 14rpx 0;
		border-bottom: 1px solid rgba(0, 0, 0, 0.05);
	}

	.simple-list__item--link:active {
		opacity: 0.82;
	}

	.simple-list__item:last-child {
		border-bottom: 0;
	}

	.simple-list__title {
		display: block;
		font-size: 25rpx;
		font-weight: 600;
		color: var(--color-text-primary);
	}

	.simple-list__meta {
		display: block;
		margin-top: 6rpx;
		font-size: 22rpx;
		color: #8d847a;
	}

	.meal-order-floating {
		position: fixed;
		left: 24rpx;
		right: 24rpx;
		bottom: calc(env(safe-area-inset-bottom) + 168rpx);
		z-index: 11;
		padding: 8rpx;
		border-radius: 30rpx;
		background:
			radial-gradient(circle at top right, rgba(255, 224, 188, 0.22) 0%, rgba(255, 224, 188, 0) 38%),
			linear-gradient(145deg, rgba(72, 56, 44, 0.9) 0%, rgba(44, 34, 29, 0.86) 100%);
		border: 1px solid rgba(255, 233, 207, 0.12);
		box-shadow:
			0 20rpx 34rpx rgba(45, 36, 29, 0.18),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.08);
		backdrop-filter: blur(22rpx);
		display: flex;
		align-items: center;
		gap: 8rpx;
		animation: meal-order-floating-enter 220ms ease both;
	}

	.meal-order-floating__summary {
		flex: 1;
		min-width: 0;
		min-height: 72rpx;
		padding: 0 10rpx 0 6rpx;
		border-radius: 22rpx;
		background: rgba(255, 248, 238, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.06);
		display: flex;
		align-items: center;
		transition: transform 0.18s ease, background 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
	}

	.meal-order-floating__summary:active {
		transform: translateY(1rpx);
		background: rgba(255, 248, 238, 0.12);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.08);
	}

	.meal-order-floating__summary-main {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
		width: 100%;
	}

	.meal-order-floating__pill {
		max-width: 100%;
		min-height: 42rpx;
		padding: 0 14rpx;
		border-radius: 999rpx;
		background: rgba(255, 248, 238, 0.12);
		border: 1px solid rgba(255, 255, 255, 0.07);
		display: inline-flex;
		align-items: center;
		gap: 8rpx;
	}

	.meal-order-floating__pill--empty {
		background: rgba(255, 248, 238, 0.1);
	}

	.meal-order-floating__pill-dot {
		width: 10rpx;
		height: 10rpx;
		border-radius: 999rpx;
		background: #f2d2ae;
		flex-shrink: 0;
	}

	.meal-order-floating__pill-text {
		font-size: 20rpx;
		font-weight: 700;
		line-height: 1;
		color: #fff7ed;
		white-space: nowrap;
	}

	.meal-order-floating__peek {
		width: 36rpx;
		height: 36rpx;
		border-radius: 999rpx;
		background: rgba(255, 248, 238, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.05);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: transform 0.18s ease, background 0.18s ease, opacity 0.18s ease;
	}

	.meal-order-floating__summary:active .meal-order-floating__peek {
		transform: translateX(2rpx);
		background: rgba(255, 248, 238, 0.12);
		opacity: 0.92;
	}

	.meal-order-floating__action {
		min-width: 154rpx;
		height: 70rpx;
		padding: 0 18rpx;
		border-radius: 20rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.9) 0%, rgba(255, 255, 255, 0) 44%),
			linear-gradient(180deg, rgba(255, 242, 227, 0.98) 0%, rgba(243, 224, 201, 0.96) 100%);
		border: 1px solid rgba(255, 255, 255, 0.28);
		box-shadow:
			inset 0 1rpx 0 rgba(255, 255, 255, 0.92),
			inset 0 -1rpx 0 rgba(183, 142, 100, 0.12),
			0 10rpx 18rpx rgba(34, 25, 20, 0.1);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: transform 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
	}

	.meal-order-floating__action--disabled {
		background: rgba(191, 180, 168, 0.84);
		border-color: rgba(255, 255, 255, 0.08);
		pointer-events: none;
		box-shadow: none;
	}

	.meal-order-floating__action:active {
		transform: translateY(2rpx) scale(0.986);
		box-shadow:
			inset 0 1rpx 0 rgba(255, 255, 255, 0.88),
			inset 0 -1rpx 0 rgba(183, 142, 100, 0.1),
			0 6rpx 12rpx rgba(34, 25, 20, 0.09);
	}

	.meal-order-floating__action-text {
		font-size: 23rpx;
		font-weight: 700;
		line-height: 1;
		color: #4b3728;
	}

	@keyframes meal-order-floating-enter {
		from {
			opacity: 0;
			transform: translateY(18rpx);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.bottom-nav {
		position: fixed;
		left: 46rpx;
		right: 46rpx;
		bottom: calc(env(safe-area-inset-bottom) + 20rpx);
		z-index: 9;
		min-height: 92rpx;
		padding: 8rpx 42rpx 10rpx;
		border-radius: 46rpx;
		background:
			radial-gradient(circle at 50% -46%, rgba(255, 250, 242, 0.52) 0%, rgba(255, 250, 242, 0) 48%),
			linear-gradient(180deg, rgba(255, 255, 255, 0.52) 0%, rgba(255, 253, 249, 0.62) 100%);
		border: 1px solid rgba(255, 255, 255, 0.56);
		box-shadow:
			0 18rpx 44rpx rgba(56, 44, 30, 0.1),
			0 2rpx 10rpx rgba(56, 44, 30, 0.04),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.74);
		backdrop-filter: blur(18rpx);
		-webkit-backdrop-filter: blur(18rpx);
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.bottom-nav--meal-order-entering {
		animation: bottom-nav-meal-order-enter 260ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
	}

	.bottom-nav--meal-order-leaving {
		animation: bottom-nav-meal-order-leave 220ms ease both;
	}

	@keyframes bottom-nav-meal-order-enter {
		from {
			transform: translateY(18rpx);
			opacity: 0.82;
		}
		to {
			transform: translateY(0);
			opacity: 1;
		}
	}

	@keyframes bottom-nav-meal-order-leave {
		from {
			transform: translateY(-8rpx);
			opacity: 0.94;
		}
		to {
			transform: translateY(0);
			opacity: 1;
		}
	}

	.bottom-nav--meal-order .nav-center {
		transform: translateY(-44rpx);
	}

	.bottom-nav--meal-order .nav-fab {
		width: 96rpx;
		height: 96rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 248, 237, 0.2) 0%, rgba(255, 248, 237, 0) 34%),
			linear-gradient(180deg, #6b594b 0%, #5a4739 100%);
		box-shadow:
			0 14rpx 22rpx rgba(91, 74, 59, 0.12),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.14);
	}

	.bottom-nav--meal-order .nav-item__icon-shell {
		background: transparent;
		box-shadow: none;
	}

	.nav-item,
	.nav-center {
		width: 30%;
		min-height: 76rpx;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 6rpx;
	}

	.nav-item__icon-shell {
		width: 68rpx;
		height: 46rpx;
		border-radius: 999rpx;
		background: transparent;
		border: 0;
		box-shadow: none;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease, background 0.18s ease;
	}

	.nav-item:active .nav-item__icon-shell {
		transform: translateY(1rpx);
		box-shadow: none;
	}

	.nav-item__label,
	.nav-center__label {
		font-size: 24rpx;
		line-height: 1;
		color: #b9aea8;
		font-weight: 700;
		transition: color 0.18s ease;
	}

	.nav-item--active .nav-item__icon-shell {
		transform: none;
		background: transparent;
		border-color: transparent;
		box-shadow: none;
	}

	.nav-item--active .nav-item__label {
		color: #6b4d3d;
		font-weight: 700;
	}

	.nav-space-icon {
		width: 50rpx;
		height: 50rpx;
		transition: transform 0.18s ease;
	}

	.nav-item:active .nav-space-icon {
		transform: translateY(1rpx);
	}

	.nav-center {
		position: relative;
		transform: translateY(-56rpx);
	}

	.nav-fab {
		width: 112rpx;
		height: 112rpx;
		border-radius: 999rpx;
		border: 8rpx solid rgba(255, 252, 247, 0.98);
		background:
			radial-gradient(circle at top left, rgba(255, 248, 237, 0.22) 0%, rgba(255, 248, 237, 0) 34%),
			linear-gradient(180deg, #6a5849 0%, #534133 100%);
		box-shadow:
			0 20rpx 32rpx rgba(91, 74, 59, 0.18),
			0 6rpx 12rpx rgba(91, 74, 59, 0.1),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.14);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
	}

	.nav-fab__icon {
		width: 52rpx;
		height: 52rpx;
	}

	.nav-center:active .nav-fab,
	.nav-fab--pressed {
		transform: translateY(2rpx) scale(0.972);
		box-shadow:
			0 10rpx 18rpx rgba(91, 74, 59, 0.12),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.14);
	}

	.nav-center:active .nav-center__label {
		color: #6a5848;
	}

</style>
