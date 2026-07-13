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
			@touchstart="handleAppModeTouchStart"
			@touchend="handleAppModeTouchEnd"
			@touchcancel="resetAppModeTouch"
		>
			<!-- Mode Switcher: 美食库 / 打卡点 -->
			<view class="mode-switcher">
				<view class="mode-switcher__track">
					<view
						class="mode-switcher__btn"
						:class="{ 'mode-switcher__btn--active': appMode === 'cook' }"
						@tap="switchAppMode('cook')"
					>
						<up-icon name="grid-fill" size="14" :color="appMode === 'cook' ? '#5c4033' : 'rgba(92,64,51,0.4)'"></up-icon>
						<text class="mode-switcher__btn-text">美食库</text>
					</view>
					<view
						class="mode-switcher__btn"
						:class="{ 'mode-switcher__btn--active': appMode === 'explore' }"
						@tap="switchAppMode('explore')"
					>
						<up-icon name="map-fill" size="14" :color="appMode === 'explore' ? '#5c4033' : 'rgba(92,64,51,0.4)'"></up-icon>
						<text class="mode-switcher__btn-text">打卡点</text>
					</view>
				</view>
			</view>

			<!-- 模式描述行 + 安排菜单按钮 -->
			<view v-if="!isLibraryMealOrderMode" class="mode-desc-row">
				<text class="mode-desc-row__text">{{ appMode === 'cook' ? libraryHeaderSummary : '记录心动店铺，周末去哪儿不用愁' }}</text>
				<view v-if="appMode === 'cook'" class="mode-desc-row__action" @tap="openMealOrderDateSheet">
					<up-icon name="calendar" size="14" color="#745742"></up-icon>
					<text class="mode-desc-row__action-text">安排菜单</text>
				</view>
			</view>

			<LibraryPane
				v-if="appMode === 'cook'"
				v-model="searchKeyword"
				:class="appModePaneMotionClass"
				:is-library-meal-order-mode="isLibraryMealOrderMode"
				:library-header-title="libraryHeaderTitle"
				:library-header-summary="libraryHeaderSummary"
				:meal-order-spotlight-record="mealOrderSpotlightRecord"
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
				:toolbar-bounce-class="toolbarBounceClass"
				:is-search-focused="isSearchFocused"
				:trimmed-search-keyword="trimmedSearchKeyword"
				:search-placeholder-text="searchPlaceholderText"
				:show-search-assist="showSearchAssist"
				:search-assist-label="searchAssistLabel"
				:search-assist-keywords="searchAssistKeywords"
				:meal-tabs="mealTabs"
				:active-meal-type="activeMealType"
				:status-tabs="statusTabs"
				:active-status="activeStatus"
				:status-map="statusMap"
				:current-filter-summary="currentFilterSummary"
				:can-reset-library-filters="canResetLibraryFilters"
				:filtered-recipes="filteredRecipes"
				:recipe-cards="recipeCards"
				:selected-recipe-id="selectedRecipeId"
				:return-focus-recipe-id="returnFocusRecipeId"
				:recipe-list-motion-tick="recipeListMotionTick"
				:empty-state-kind="emptyStateKind"
				:empty-state-title="emptyStateTitle"
				:empty-state-desc="emptyStateDesc"
				:empty-state-primary-text="emptyStatePrimaryText"
				:empty-state-primary-icon="emptyStatePrimaryIcon"
				:empty-state-primary-icon-src="emptyStatePrimaryIconSrc"
				:empty-state-secondary-text="emptyStateSecondaryText"
				:meal-type-count="mealTypeCount"
				:status-count="statusCount"
				:get-recipe-card-display-cover="getRecipeCardDisplayCover"
				:meal-order-has-recipe="mealOrderHasRecipe"
				@open-meal-order-date="openMealOrderDateSheet"
				@exit-meal-order-mode="exitMealOrderMode"
				@spotlight-tap="handleMealOrderSpotlightTap"
				@spotlight-touchstart="handleMealOrderSpotlightTouchStart"
				@spotlight-touchend="handleMealOrderSpotlightTouchEnd"
				@search-focus="handleSearchFocus"
				@search-blur="handleSearchBlur"
				@search-confirm="handleSearchConfirm"
				@apply-search="applySearchKeyword"
				@meal-type-change="handleMealTypeTabChange"
				@status-change="handleStatusTabChange"
				@reset-filters="resetLibraryFilters"
				@draw-tonight="drawTonight"
				@open-recipe="openRecipeDetail"
				@image-error="handleRecipeCardImageError"
				@toggle-status="toggleRecipeStatus"
				@toggle-meal-order="toggleMealOrderRecipe"
				@empty-primary="handleEmptyStatePrimary"
				@empty-secondary="handleEmptyStateSecondary"
			/>

			<!-- 打卡点模式 -->
			<PlacePane
				v-else
				:app-mode-pane-motion-class="appModePaneMotionClass"
				:search-keyword="placeSearchKeyword"
				:is-place-search-focused="isPlaceSearchFocused"
				:trimmed-place-search-keyword="trimmedPlaceSearchKeyword"
				:place-status-tabs="placeStatusTabs"
				:active-status="activePlaceStatus"
				:filtered-places="filteredPlaces"
				:place-status-count="placeStatusCount"
				@update:search-keyword="placeSearchKeyword = $event"
				@update:active-status="activePlaceStatus = $event"
				@search-focus="isPlaceSearchFocused = true"
				@search-blur="isPlaceSearchFocused = false"
				@open-place="handlePlaceOpen"
				@open-location="openPlaceLocation"
				@add-place="openPlaceCreateSheet"
			/>
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
					:show-leave-action="canLeaveCurrentKitchen"
					:is-leaving-kitchen="isLeavingKitchen"
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
					@leave-kitchen="confirmLeaveCurrentKitchen"
				>
					<template #overview>
						<space-stats-card
							:stats="spaceStats"
							:has-kitchen="isKitchenConnected"
							:is-syncing="isRefreshingSpaceStats"
							@open-stats="openSpaceStatsPage"
							@action="handleSpaceStatsAction"
						></space-stats-card>
					</template>
				</kitchen-section>
			</template>

			<view v-if="!isLibraryMealOrderMode && activeSection !== 'kitchen'" class="app-footer-links">
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
				<text class="nav-item__label">清单</text>
			</view>

			<view class="nav-center">
				<view
					class="nav-fab"
					hover-class="nav-fab--pressed"
					hover-start-time="0"
					hover-stay-time="140"
					hover-stop-propagation
					@tap="handlePrimaryFabTap"
				>
					<!-- 底部主入口：打卡点模式添加打卡点；美食库模式按后台开关分流到饮食管家或添加菜品 -->
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

		<add-recipe-preview-panel
			:show="showAddRecipePreviewPanel"
			@close="closeAddRecipePreviewPanel"
			@manual-entry="handleRecipeManualEntry"
			@parse-result="handleRecipeParseResult"
			@preview-timeout="handleRecipePreviewTimeoutFallback"
		></add-recipe-preview-panel>

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

		<place-edit-sheet
			:show="showPlaceEditSheet"
			:is-edit="placeEditMode === 'edit'"
			:draft="placeDraft"
			:status-options="placeStatusOptions"
			:type-options="placeTypeOptions"
			:source-options="placeSourceOptions"
			:max-place-images="maxPlaceImages"
			:can-submit="canSubmitPlaceDraft"
			:is-submitting="isSubmittingPlace"
			@close="closePlaceEditSheet"
			@name-input="handlePlaceDraftNameInput"
			@select-status="handlePlaceDraftStatusSelect"
			@select-type="handlePlaceDraftTypeSelect"
			@choose-location="choosePlaceLocation"
			@address-input="handlePlaceDraftAddressInput"
			@phone-input="handlePlaceDraftPhoneInput"
			@price-input="handlePlaceDraftPriceInput"
			@choose-images="choosePlaceImages"
			@preview-image="previewPlaceDraftImages"
			@remove-image="removePlaceDraftImage"
			@tags-input="handlePlaceDraftTagsInput"
			@select-source="handlePlaceDraftSourceSelect"
			@source-url-input="handlePlaceDraftSourceUrlInput"
			@note-input="handlePlaceDraftNoteInput"
			@rating-input="handlePlaceDraftRatingInput"
			@recommended-items-input="handlePlaceDraftRecommendedItemsInput"
			@dining-tips-input="handlePlaceDraftDiningTipsInput"
			@scenes-input="handlePlaceDraftScenesInput"
			@best-time-input="handlePlaceDraftBestTimeInput"
			@duration-input="handlePlaceDraftDurationInput"
			@companion-tags-input="handlePlaceDraftCompanionTagsInput"
			@parking-note-input="handlePlaceDraftParkingNoteInput"
			@submit="submitPlaceDraft"
		></place-edit-sheet>

		<add-link-preview-panel
			:show="showAddLinkPreviewPanel"
			@close="closeAddLinkPreviewPanel"
			@manual-entry="handleManualEntry"
			@parse-result="handleParseResult"
		></add-link-preview-panel>

		<place-candidate-sheet
			:show="showPlaceCandidateSheet"
			:candidates="placeCandidates"
			:extracted="placeExtracted"
			:source="placeParseSource"
			@close="closePlaceCandidateSheet"
			@select-candidate="handleSelectCandidate"
			@manual-entry="handleManualEntry"
		></place-candidate-sheet>

		<place-detail-sheet
			:show="showPlaceDetailSheet"
			:place="selectedPlace"
			:is-submitting="isSubmittingPlace"
			@close="closePlaceDetailSheet"
			@preview-image="previewSelectedPlaceImages"
			@open-location="openPlaceLocation"
			@open-source="openPlaceSourceURL"
			@toggle-status="togglePlaceStatus"
			@edit="openPlaceEditSheet"
			@delete="confirmDeletePlace"
		></place-detail-sheet>

		<place-experience-sheet
			:show="showPlaceExperienceSheet"
			:place-type="pendingPlaceType"
			:is-submitting="isSubmittingPlace"
			@close="closePlaceExperienceSheet"
			@skip="handleExperienceSkip"
			@submit="handleExperienceSubmit"
		></place-experience-sheet>

		<diet-assistant-sheet
			:show="showDietAssistantSheet"
			:initial-prompt="dietAssistantInitialPrompt"
			@close="closeDietAssistantSheet"
			@open-add-recipe="openAddSheetFromAssistant"
			@recipes-mutated="handleDietAssistantRecipesMutated"
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
import { readCachedPublicAppConfig } from '../../utils/public-app-config-api'
import {
	MAX_PLACE_IMAGES,
	getCachedPlaces,
	loadPlaces,
	placeSourceOptions,
	placeStatusOptions,
	placeTypeOptions
} from '../../utils/place-store'
import {
	MAX_RECIPE_IMAGES,
	getCachedRecipes,
	loadRecipes,
	mealTypeOptions,
	statusOptions
} from '../../utils/recipe-store'
import { takePendingSpaceStatsAction } from '../../utils/space-stats-bridge'
import {
	ensureSession,
	getCurrentKitchenId,
	getFriendlySessionErrorMessage
} from '../../utils/auth'
import { getAccessToken } from '../../utils/session-storage'
import { createEmptyDraft, statusMap } from './constants'
import AddRecipeSheet from './components/add-recipe-sheet.vue'
import AddRecipePreviewPanel from './components/add-recipe-preview-panel.vue'
import DietAssistantSheet from './components/diet-assistant-sheet.vue'
import InviteCodeSheet from './components/invite-code-sheet.vue'
import InviteSheet from './components/invite-sheet.vue'
import KitchenSection from './components/kitchen-section.vue'
import LibraryPane from './components/library-pane.vue'
import MealOrderCartSheet from './components/meal-order-cart-sheet.vue'
import MealOrderCheckoutSheet from './components/meal-order-checkout-sheet.vue'
import MealOrderDateSheet from './components/meal-order-date-sheet.vue'
import MealOrderSuccessSheet from './components/meal-order-success-sheet.vue'
import PlaceDetailSheet from './components/place-detail-sheet.vue'
import PlaceEditSheet from './components/place-edit-sheet.vue'
import PlaceExperienceSheet from './components/place-experience-sheet.vue'
import AddLinkPreviewPanel from './components/add-link-preview-panel.vue'
import PlaceCandidateSheet from './components/place-candidate-sheet-v2.vue'
import ProfileSheet from './components/profile-sheet.vue'
import RandomPickSheet from './components/random-pick-sheet.vue'
import PlacePane from './components/place-pane.vue'
import SpaceStatsCard from './components/space-stats-card.vue'
import {
	consumePendingMealOrderAction,
	createEmptyMealOrderStore,
	formatMealOrderDateText,
	normalizeMealOrderDate,
	normalizeMealOrderStore
} from './meal-order'
import { readRecentSearches } from './storage'
import {
	createEmptyPlaceDraft,
	placeLibraryModule
} from './use-place-library'
import {
	createSearchBlurController,
	recipeLibraryModule
} from './use-recipe-library'
import { mealOrderModule } from './use-meal-order'
import {
	buildInviteShareImageURL,
	buildInviteShareTitle,
	kitchenSpaceModule
} from './use-kitchen-space'
import {
	smartAddModule
} from './use-smart-add'
import {
	installIndexPageModules,
	runIndexPageModuleLifecycle,
	validateIndexPageModuleContext
} from './page-module'

const APP_MODE_SWIPE_MIN_DISTANCE = 64
const APP_MODE_SWIPE_DOMINANCE_RATIO = 1.18
const INDEX_PAGE_MODULES = [
	placeLibraryModule,
	recipeLibraryModule,
	mealOrderModule,
	kitchenSpaceModule,
	smartAddModule
]
const INDEX_PAGE_MODULE_INSTALL = installIndexPageModules(INDEX_PAGE_MODULES)

function getGestureTouch(event = {}, preferChanged = false) {
	const primary = preferChanged ? event?.changedTouches : event?.touches
	const fallback = preferChanged ? event?.touches : event?.changedTouches
	return primary?.[0] || fallback?.[0] || null
}

export default {
	components: {
		ActionFeedback,
		AddRecipeSheet,
		AddRecipePreviewPanel,
		AddLinkPreviewPanel,
		DietAssistantSheet,
		InviteCodeSheet,
		InviteSheet,
		KitchenSection,
		LibraryPane,
		MealOrderCartSheet,
		MealOrderCheckoutSheet,
		MealOrderDateSheet,
		MealOrderSuccessSheet,
		PlacePane,
		PlaceCandidateSheet,
		PlaceDetailSheet,
		PlaceEditSheet,
		PlaceExperienceSheet,
		ProfileSheet,
		RandomPickSheet,
		SpaceStatsCard
	},
	data() {
		return {
			statusMap,
			activeSection: 'library',
			appMode: 'cook',
			appModeTouchTracking: false,
			appModeTouchStartX: 0,
			appModeTouchStartY: 0,
			appModeMotionDirection: '',
			appModeMotionTimer: null,
			activePlaceStatus: 'all',
			placeSearchKeyword: '',
			isPlaceSearchFocused: false,
			placeStatusTabs: [
				{ label: '全部', value: 'all', icon: 'map-fill' },
				{ label: '想去', value: 'want', icon: 'heart' },
				{ label: '去过', value: 'visited', icon: 'checkmark-circle' }
			],
			places: [],
			placeSyncErrorMessage: '',
			isLoadingPlaces: false,
			showPlaceEditSheet: false,
			showPlaceDetailSheet: false,
			showPlaceExperienceSheet: false,
			pendingPlaceStatusChangeId: '',
			showAddLinkPreviewPanel: false,
			showAddRecipePreviewPanel: false,
			showPlaceCandidateSheet: false,
			placeCandidates: [],
			placeExtracted: {},
			placeParseSource: '',
			placeEditMode: 'create',
			editingPlaceId: '',
			selectedPlaceId: '',
			placeDraft: createEmptyPlaceDraft(),
				placeStatusOptions,
				placeTypeOptions,
				placeSourceOptions,
				maxPlaceImages: MAX_PLACE_IMAGES,
				isSubmittingPlace: false,
				activeMealType: 'main',
			activeStatus: 'all',
			toolbarBounceClass: '',
			toolbarBounceTimer: null,
			returnFocusRecipeId: '',
			returnFocusPendingRecipeId: '',
			returnFocusTimer: null,
			searchKeyword: '',
			recentSearches: readRecentSearches(),
			isSearchFocused: false,
			searchBlurController: null,
			selectedRecipeId: '',
			showAddSheet: false,
			showDietAssistantSheet: false,
			dietAssistantInitialPrompt: '',
			publicAppConfig: readCachedPublicAppConfig(),
			publicAppConfigRequestID: 0,
			draftLinkPrefillSource: '',
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
			mealOrderDraftSyncController: null,
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
			isLeavingKitchen: false,
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
			recipePreviewTimeoutRefreshTimer: null,
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
			isPreparingInvite: false,
			isRefreshingSpaceStats: false,
			spaceStatsSyncedAt: '',
			spaceStatsWindow: 'all',
			spaceStatsRemote: null,
			spaceStatsRemoteKitchenId: 0,
			spaceStatsRemoteError: '',
			spaceStatsAutoSyncKitchenId: 0
		}
	},
	onLoad(options) {
		const missingModuleContext = validateIndexPageModuleContext(this, INDEX_PAGE_MODULES)
		if (missingModuleContext.length) {
			console.warn('[index-page] module context is incomplete:', missingModuleContext.join(', '))
		}
		this.searchBlurController = createSearchBlurController(() => {
			this.isSearchFocused = false
		}, 120)
		if (options?.section === 'kitchen') {
			this.activeSection = 'kitchen'
		}
	},
		onShow() {
			this.refreshPublicAppConfig()
			this.refreshRecipes()
			this.playPendingRecipeReturnFocus()
			// 从空间洞察页返回时，执行页内触发的动作（进店详情 / 看草稿等）。
			const pendingStatsAction = takePendingSpaceStatsAction()
			if (pendingStatsAction) {
				this.$nextTick(() => this.handleSpaceStatsAction(pendingStatsAction))
			}
		},
	async onPullDownRefresh() {
		// 空间页主卡片下拉刷新：串起数据同步 + 统计重新聚合（§11 V1）。
		try {
			if (this.activeSection === 'kitchen') {
				await this.refreshSpaceStats({ silent: false })
			} else {
				await this.refreshRecipes({ silent: false })
			}
		} finally {
			uni.stopPullDownRefresh()
		}
	},
	onHide() {
		runIndexPageModuleLifecycle(this, INDEX_PAGE_MODULES, 'deactivate')
		this.clearAppModeMotionTimer()
		this.resetAppModeTouch()
	},
	onUnload() {
		runIndexPageModuleLifecycle(this, INDEX_PAGE_MODULES, 'dispose')
		this.clearAppModeMotionTimer()
		this.resetAppModeTouch()
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
			appModePaneMotionClass() {
				if (this.appModeMotionDirection === 'forward') return 'app-mode-pane--forward'
				if (this.appModeMotionDirection === 'back') return 'app-mode-pane--back'
				return ''
			},
		...INDEX_PAGE_MODULE_INSTALL.computed
	},
	watch: {
		activeSection(next, prev) {
			if (next === prev) return
			if (next !== 'library') {
				this.clearRecipeStatusFeedback()
				this.clearRecipeReturnFocus()
				this.closeRandomPickSheet()
			}
		},
		isLibraryMealOrderMode(next, prev) {
			if (next === prev) return
			this.queueMealOrderModeMotion(next ? 'entering' : 'leaving')
			this.bumpRecipeListMotion()
		}
	},
		methods: {
			clearAppModeMotionTimer() {
				if (this.appModeMotionTimer) {
					clearTimeout(this.appModeMotionTimer)
					this.appModeMotionTimer = null
				}
				this.appModeMotionDirection = ''
			},
			queueAppModeMotion(nextMode = 'cook') {
				this.clearAppModeMotionTimer()
				this.appModeMotionDirection = nextMode === 'explore' ? 'forward' : 'back'
				this.appModeMotionTimer = setTimeout(() => {
					this.appModeMotionDirection = ''
					this.appModeMotionTimer = null
				}, 260)
			},
			resetAppModeTouch() {
				this.appModeTouchTracking = false
				this.appModeTouchStartX = 0
				this.appModeTouchStartY = 0
			},
			handleAppModeTouchStart(event) {
				const touch = getGestureTouch(event)
				if (
					!touch ||
					this.activeSection !== 'library' ||
					this.isSearchFocused ||
					this.isPlaceSearchFocused
				) {
					return
				}
				this.appModeTouchTracking = true
				this.appModeTouchStartX = Number(touch.clientX || touch.pageX || 0)
				this.appModeTouchStartY = Number(touch.clientY || touch.pageY || 0)
			},
			handleAppModeTouchEnd(event) {
				if (!this.appModeTouchTracking) return
				const touch = getGestureTouch(event, true)
				const startX = Number(this.appModeTouchStartX || 0)
				const startY = Number(this.appModeTouchStartY || 0)
				this.resetAppModeTouch()
				if (!touch || this.activeSection !== 'library') return

				const endX = Number(touch.clientX || touch.pageX || 0)
				const endY = Number(touch.clientY || touch.pageY || 0)
				const diffX = endX - startX
				const diffY = endY - startY
				const distanceX = Math.abs(diffX)
				const distanceY = Math.abs(diffY)
				if (
					distanceX < APP_MODE_SWIPE_MIN_DISTANCE ||
					distanceX < distanceY * APP_MODE_SWIPE_DOMINANCE_RATIO
				) {
					return
				}

				this.switchAppMode(diffX < 0 ? 'explore' : 'cook')
			},
			switchAppMode(mode) {
				if (this.appMode === mode) return
				this.queueAppModeMotion(mode)
				this.appMode = mode
				if (mode === 'explore') {
					this.refreshPlaces({ silent: true })
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
		async refreshRecipes(options = {}) {
			const { silent = true } = options
			const cachedRecipes = getCachedRecipes()
			this.applyRecipes(cachedRecipes)
			const storedToken = getAccessToken()
			if (!storedToken) {
				this.applySession(null)
				this.syncErrorMessage = ''
				this.applyPlaces(getCachedPlaces())
				this.kitchenMembers = []
				this.kitchenMembersKitchenId = 0
				return cachedRecipes
			}

			try {
				this.isSyncing = true
				const session = await ensureSession()
				this.syncErrorMessage = ''
				this.applySession(session)
					const kitchenId = getCurrentKitchenId()
					const [recipes, places] = await Promise.all([
						loadRecipes({ forceRefresh: true }),
						loadPlaces({ forceRefresh: true }),
						this.refreshKitchenMembers({ kitchenId, silent: true })
					])
					this.applyRecipes(recipes)
					this.applyPlaces(places)
					await this.applyPendingMealOrderAction(kitchenId)
					this.spaceStatsSyncedAt = new Date().toISOString()
				} catch (error) {
				this.syncErrorMessage = getFriendlySessionErrorMessage(error)
					this.applySession()
					this.applyRecipes(getCachedRecipes())
					this.applyPlaces(getCachedPlaces())
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
		...INDEX_PAGE_MODULE_INSTALL.methods
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

<style lang="scss" src="./index-page.scss"></style>
