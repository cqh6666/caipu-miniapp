<template>
	<view class="app-shell">
		<view class="page-content">
			<template v-if="activeSection === 'library'">
				<view class="page-header">
					<text class="page-header__title">美食库</text>
					<text class="page-header__summary">{{ librarySummary }}</text>
				</view>

				<view class="toolbar">
					<view class="toolbar__row">
						<view class="search-box">
							<up-icon name="search" size="15" color="#8f8377"></up-icon>
							<input
								v-model="searchKeyword"
								class="search-box__input"
								placeholder="搜菜名、备注或链接"
								placeholder-class="search-box__placeholder"
							/>
						</view>
						<view class="tool-button" @tap="drawTonight">
							<up-icon name="reload" size="14" color="#4d433a"></up-icon>
							<text class="tool-button__text">随机</text>
						</view>
					</view>

					<view class="meal-tabs">
						<view
							v-for="tab in mealTabs"
							:key="tab.value"
							class="meal-tab"
							:class="{ 'meal-tab--active': activeMealType === tab.value }"
							@tap="activeMealType = tab.value"
						>
							<view class="meal-tab__left">
								<view class="meal-tab__icon-shell">
									<up-icon
										:name="tab.icon"
										size="12"
										:color="activeMealType === tab.value ? tab.activeColor : '#8e8479'"
									></up-icon>
								</view>
								<text class="meal-tab__text">{{ tab.label }}</text>
							</view>
							<text class="meal-tab__count">{{ mealTypeCount(tab.value) }}</text>
						</view>
					</view>

					<scroll-view class="status-scroll" scroll-x>
						<view class="status-track">
							<view
								v-for="tab in statusTabs"
								:key="tab.value"
								class="status-pill"
								:class="{ 'status-pill--active': activeStatus === tab.value }"
								@tap="activeStatus = tab.value"
							>
								<view class="status-pill__inner">
									<up-icon
										:name="statusMap[tab.value].icon"
										size="13"
										:color="activeStatus === tab.value ? '#ffffff' : '#7a6d61'"
									></up-icon>
									<text class="status-pill__text">{{ tab.label }}</text>
								</view>
							</view>
						</view>
					</scroll-view>
				</view>

				<view class="list-caption">
					<text class="list-caption__text">{{ currentMealLabel }} · {{ filteredRecipes.length }} 道记录</text>
				</view>

				<view v-if="filteredRecipes.length" class="recipe-list">
					<view
						v-for="recipe in filteredRecipes"
						:key="recipe.id"
						class="recipe-item"
						:class="{ 'recipe-item--active': selectedRecipeId === recipe.id }"
						@tap="openRecipeDetail(recipe.id)"
					>
						<view
							class="recipe-item__marker"
							:class="'recipe-item__marker--' + recipe.status"
						></view>
						<view class="recipe-item__main">
							<view class="recipe-item__top">
								<view class="recipe-item__text">
									<text class="recipe-item__title">{{ recipe.title }}</text>
									<text class="recipe-item__meta">{{ recipeSecondaryText(recipe) }}</text>
								</view>
								<view
									class="recipe-switch"
									:class="'recipe-switch--' + recipe.status"
									@tap.stop="toggleRecipeStatus(recipe.id)"
								>
									<view class="recipe-switch__track">
										<view class="recipe-switch__slot">
											<up-icon
												name="heart-fill"
												size="12"
												color="#b8aa9b"
											></up-icon>
										</view>
										<view class="recipe-switch__slot">
											<up-icon
												name="checkmark-circle-fill"
												size="12"
												color="#b8aa9b"
											></up-icon>
										</view>
									</view>
									<view class="recipe-switch__thumb">
										<up-icon
											:name="statusMap[recipe.status].icon"
											size="12"
											:color="recipe.status === 'done' ? '#6f826d' : '#9a7b65'"
										></up-icon>
									</view>
								</view>
							</view>
						</view>
					</view>
				</view>

				<view v-else class="empty-state">
					<up-icon name="empty-search" size="40" color="#c0b3a5"></up-icon>
					<text class="empty-state__title">没有找到匹配内容</text>
					<text class="empty-state__desc">试试换个关键词，或者点中间的加号新增一道菜。</text>
				</view>
			</template>

				<template v-else>
					<view class="kitchen-hero">
					<view
						class="kitchen-card"
						:class="{ 'kitchen-card--disabled': !currentKitchenName }"
						@tap="openKitchenSelector"
					>
						<view class="kitchen-card__header">
							<view class="kitchen-card__badge">
								<up-icon name="grid-fill" size="12" color="#5b4a3b"></up-icon>
								<text class="kitchen-card__badge-text">当前厨房</text>
							</view>
							<view class="kitchen-card__switch">
								<text class="kitchen-card__switch-text">{{ canSwitchKitchen ? '切换' : kitchenConnectionLabel }}</text>
								<up-icon
									v-if="canSwitchKitchen"
									name="arrow-right"
									size="14"
									color="#7f7265"
								></up-icon>
								<view
									v-else
									class="kitchen-card__status-dot"
									:class="{ 'kitchen-card__status-dot--connected': isKitchenConnected }"
								></view>
							</view>
						</view>
						<view class="kitchen-card__name-row">
							<text class="kitchen-card__name">{{ currentKitchenDisplayName }}</text>
							<view
								v-if="currentKitchenName"
								class="kitchen-card__name-edit"
								@tap.stop="openKitchenNameSheet"
							>
								<up-icon name="edit-pen" size="15" color="#6e5f50"></up-icon>
							</view>
						</view>
						<text class="kitchen-card__meta">{{ currentKitchenMetaText }}</text>
						<view v-if="currentKitchenName" class="kitchen-card__tags">
							<view class="kitchen-card__tag">
								<text class="kitchen-card__tag-value">{{ kitchenMembers.length || 0 }}</text>
								<text class="kitchen-card__tag-label">成员</text>
							</view>
							<view v-if="currentKitchenRoleLabel" class="kitchen-card__tag">
								<text class="kitchen-card__tag-value">{{ currentKitchenRoleLabel }}</text>
								<text class="kitchen-card__tag-label">身份</text>
							</view>
							<view v-if="canSwitchKitchen" class="kitchen-card__tag">
								<text class="kitchen-card__tag-value">{{ kitchenOptions.length }}</text>
								<text class="kitchen-card__tag-label">厨房</text>
							</view>
						</view>
					</view>

					<view class="kitchen-actions">
						<view class="kitchen-actions__primary" @tap="openInviteSheet">
							<view class="kitchen-actions__primary-icon">
								<up-icon name="share" size="16" color="#ffffff"></up-icon>
							</view>
							<view class="kitchen-actions__primary-body">
								<text class="kitchen-actions__primary-title">邀请成员</text>
								<text class="kitchen-actions__primary-desc">{{ inviteActionDescription }}</text>
							</view>
						</view>
					</view>
				</view>

				<view class="member-panel">
					<view class="member-panel__header">
						<view class="member-panel__heading">
							<text class="member-panel__title">厨房成员</text>
							<text class="member-panel__desc">与你共同维护菜单的人</text>
						</view>
						<view class="member-panel__aside">
							<text class="member-panel__meta">{{ memberPanelSummary }}</text>
							<view v-if="hasMoreKitchenMembers" class="member-panel__inline-action" @tap="showAllMembers">
								<text class="member-panel__inline-action-text">查看全部</text>
							</view>
						</view>
					</view>

					<view v-if="visibleKitchenMembers.length" class="member-list">
						<view
							v-for="member in visibleKitchenMembers"
							:key="member.userId"
							class="member-card"
							:class="{ 'member-card--self': member.isCurrentUser }"
						>
							<view class="member-card__avatar">
								<image v-if="member.avatarUrl" class="member-card__avatar-image" :src="member.avatarUrl" mode="aspectFill"></image>
								<text v-else>{{ memberInitial(member) }}</text>
							</view>
							<view class="member-card__body">
								<view class="member-card__top">
									<text class="member-card__name">{{ memberDisplayName(member) }}</text>
									<view class="member-card__badges">
										<text class="member-card__badge">{{ memberRoleLabel(member.role) }}</text>
										<text v-if="member.isCurrentUser" class="member-card__badge member-card__badge--self">你</text>
									</view>
								</view>
								<text class="member-card__meta">{{ memberMemberDescription(member) }}</text>
							</view>
						</view>
					</view>

					<view v-else class="soft-empty soft-empty--inline member-panel__empty">
						<text class="soft-empty__text">
							{{ isLoadingKitchenMembers ? '正在获取成员信息...' : '这间厨房暂时只有你，邀请好友后会显示在这里。' }}
						</text>
					</view>

					<view class="member-panel__footer">
						<view class="member-panel__join-link" @tap="openInviteCodeSheet">
							<text class="member-panel__join-link-text">已有邀请码？去加入</text>
							<up-icon name="arrow-right" size="14" color="#7f7265"></up-icon>
						</view>
					</view>
				</view>

				<view
					class="app-intro"
					@tap="openAppIntro"
					@touchstart="handleAppIntroTouchStart"
					@touchend="handleAppIntroTouchEnd"
					@touchcancel="handleAppIntroTouchCancel"
				>
					<view class="app-intro__header">
						<text class="app-intro__label">应用简介</text>
						<text class="app-intro__hint">点按查看作用说明</text>
					</view>
					<text class="app-intro__text">和家人共享菜单，记录想吃和吃过，也能把视频菜谱自动整理成食材与步骤。</text>
					<view class="app-intro__progress-track">
						<view class="app-intro__progress-fill" :style="{ width: `${appIntroPressProgress}%` }"></view>
					</view>
				</view>
			</template>
		</view>

		<view class="bottom-nav">
			<view
				class="nav-item"
				:class="{ 'nav-item--active': activeSection === 'library' }"
				@tap="activeSection = 'library'"
			>
				<view class="nav-item__icon-shell">
					<up-icon
						:name="activeSection === 'library' ? 'home-fill' : 'home'"
						size="22"
						:color="activeSection === 'library' ? '#5b4a3b' : '#9a8d80'"
					></up-icon>
				</view>
				<text class="nav-item__label">美食库</text>
			</view>

			<view class="nav-center">
				<view class="nav-fab" @tap="openAddSheet">
					<up-icon name="plus" size="26" color="#ffffff"></up-icon>
				</view>
				<text class="nav-center__label">添加</text>
			</view>

			<view
				class="nav-item"
				:class="{ 'nav-item--active': activeSection === 'kitchen' }"
				@tap="activeSection = 'kitchen'"
			>
				<view class="nav-item__icon-shell">
					<up-icon
						name="grid"
						size="20"
						:color="activeSection === 'kitchen' ? '#5b4a3b' : '#9a8d80'"
					></up-icon>
				</view>
				<text class="nav-item__label">厨房</text>
			</view>
		</view>

		<up-popup
			:show="showInviteSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeInviteSheet"
		>
			<view class="invite-sheet">
				<view class="invite-sheet__header">
					<view class="invite-sheet__heading">
						<text class="invite-sheet__title">邀请成员</text>
						<text class="invite-sheet__subtitle">{{ inviteSheetSubtitle }}</text>
					</view>
					<view class="invite-sheet__close" @tap="closeInviteSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<scroll-view class="invite-sheet__body" scroll-y>
					<view v-if="isPreparingInvite" class="invite-sheet__state">
						<up-icon name="reload" size="22" color="#8d8074"></up-icon>
						<text class="invite-sheet__state-title">正在生成邀请</text>
						<text class="invite-sheet__state-desc">{{ invitePreparingText }}</text>
					</view>

					<view v-else-if="activeInvite" class="invite-sheet__stack">
						<view class="invite-sheet__code-card" @tap="copyInviteCode">
							<text class="invite-sheet__code-label">邀请码</text>
							<text class="invite-sheet__code">{{ formattedActiveInviteCode }}</text>
						</view>

						<text class="invite-sheet__meta-line">{{ inviteMetaLine }}</text>
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
							class="invite-sheet__action invite-sheet__action--primary"
							:class="{ 'invite-sheet__action--disabled': !activeInvite || isPreparingInvite }"
							:disabled="!activeInvite || isPreparingInvite"
							@tap="copyInviteCode"
						>
							<text class="invite-sheet__action-text invite-sheet__action-text--primary">
								{{ isPreparingInvite ? '生成中...' : inviteCodeCopied ? '邀请码已复制' : '复制邀请码' }}
							</text>
						</button>
						<button
							v-if="showInviteShareAction"
							class="invite-sheet__action invite-sheet__action--secondary"
							open-type="share"
							:disabled="!activeInvite || isPreparingInvite"
						>
							<view class="invite-sheet__action-inner">
								<up-icon name="share" size="16" color="#7a6d61"></up-icon>
								<text class="invite-sheet__action-text invite-sheet__action-text--secondary">发送给微信好友</text>
							</view>
						</button>
					</view>
					<view class="invite-sheet__utility">
						<view class="invite-sheet__utility-link" @tap="regenerateInviteCode">
							<up-icon name="reload" size="14" color="#7f7265"></up-icon>
							<text class="invite-sheet__utility-text">重新生成邀请码</text>
						</view>
					</view>
				</view>
			</view>
		</up-popup>

		<up-popup
			:show="showInviteCodeSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeInviteCodeSheet"
		>
			<view class="invite-code-sheet">
				<view class="invite-code-sheet__header">
					<view class="invite-code-sheet__heading">
						<text class="invite-code-sheet__title">输入邀请码</text>
						<text class="invite-code-sheet__subtitle">让朋友把邀请码发给你，输入后就能进入邀请页确认加入。</text>
					</view>
					<view class="invite-code-sheet__close" @tap="closeInviteCodeSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<view class="invite-code-sheet__body">
					<input
						:value="inviteCodeInput"
						class="invite-code-sheet__input"
						placeholder="输入邀请码，例如 AB12-CD34"
						placeholder-class="invite-code-sheet__placeholder"
						maxlength="9"
						@input="handleInviteCodeInput"
					/>
					<text class="invite-code-sheet__hint">输入后会先打开邀请页，再由你确认是否加入。</text>
				</view>

				<view class="invite-code-sheet__footer">
					<view class="sheet-action" @tap="closeInviteCodeSheet">
						<text class="sheet-action__text">取消</text>
					</view>
					<view
						class="sheet-action sheet-action--primary"
						:class="{ 'sheet-action--disabled': !canSubmitInviteCode }"
						@tap="submitInviteCode"
					>
						<text class="sheet-action__text sheet-action__text--primary">继续</text>
					</view>
				</view>
			</view>
		</up-popup>

		<up-popup
			:show="showProfileSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeProfileSheet"
		>
			<view class="profile-sheet">
				<view class="profile-sheet__header">
					<view class="profile-sheet__heading">
						<text class="profile-sheet__title">完善资料</text>
						<text class="profile-sheet__subtitle">设置头像和昵称后，厨房成员会更容易认出你。</text>
					</view>
					<view class="profile-sheet__close" @tap="closeProfileSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<form class="profile-sheet__body" @submit="submitProfile">
					<button class="profile-sheet__avatar-button" open-type="chooseAvatar" @chooseavatar="handleChooseAvatar">
						<image v-if="profileAvatarPreview" class="profile-sheet__avatar-image" :src="profileAvatarPreview" mode="aspectFill"></image>
						<view v-else class="profile-sheet__avatar-fallback">{{ profileAvatarFallback }}</view>
					</button>
					<text class="profile-sheet__avatar-tip">点击头像选择你的微信头像</text>

					<view class="profile-sheet__field">
						<text class="profile-sheet__label">昵称</text>
						<input
							:value="profileDraft.nickname"
							class="profile-sheet__input"
							type="nickname"
							name="nickname"
							placeholder="输入昵称"
							placeholder-class="profile-sheet__placeholder"
							maxlength="20"
							@input="handleProfileNicknameInput"
						/>
						<text class="profile-sheet__hint">点击输入框时，键盘上方会出现微信昵称。</text>
					</view>

					<view class="profile-sheet__footer">
						<button class="sheet-action" form-type="reset" @tap="closeProfileSheet">
							<text class="sheet-action__text">暂不设置</text>
						</button>
						<button
							class="sheet-action sheet-action--primary"
							:class="{ 'sheet-action--disabled': !canSubmitProfile || isSubmittingProfile }"
							form-type="submit"
							:disabled="!canSubmitProfile || isSubmittingProfile"
						>
							<text class="sheet-action__text sheet-action__text--primary">
								{{ isSubmittingProfile ? '保存中...' : '保存资料' }}
							</text>
						</button>
					</view>
				</form>
			</view>
		</up-popup>

		<up-popup
			:show="showAddSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeAddSheet"
		>
			<view class="sheet">
				<view class="sheet__header">
					<view class="sheet__heading">
						<text class="sheet__title">添加菜品</text>
						<text class="sheet__subtitle">先记下来，后面再慢慢补全</text>
					</view>
					<view class="sheet__close" @tap="closeAddSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<scroll-view class="sheet__body" scroll-y>
					<view class="form-field">
						<text class="form-field__label">菜名</text>
						<input
							v-model="draft.title"
							class="sheet-input sheet-input--title"
							placeholder="输入菜名"
							placeholder-class="sheet-input__placeholder"
							maxlength="40"
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">链接</text>
						<input
							v-model="draft.link"
							class="sheet-input"
							placeholder="支持直接粘贴菜谱或视频链接"
							placeholder-class="sheet-input__placeholder"
							maxlength="300"
						/>
					</view>

					<view class="form-field">
						<text class="form-field__label">成品图（可选）</text>
						<view class="upload-gallery">
							<view
								v-for="(image, index) in draft.images"
								:key="`draft-image-${index}`"
								class="upload-gallery__item"
								@tap="previewDraftImages(index)"
							>
								<image class="upload-gallery__thumb" :src="image" mode="aspectFill"></image>
								<view class="upload-gallery__badge">
									<text class="upload-gallery__badge-text">{{ index === 0 ? '封面' : index + 1 }}</text>
								</view>
								<view class="upload-gallery__remove" @tap.stop="removeDraftImage(index)">
									<up-icon name="close" size="14" color="#ffffff"></up-icon>
								</view>
							</view>
							<view
								v-if="draft.images.length < maxRecipeImages"
								class="upload-gallery__add"
								@tap="chooseDraftImages"
							>
								<view class="upload-gallery__plus">
									<up-icon name="plus" size="20" color="#8c8074"></up-icon>
								</view>
								<text class="upload-gallery__add-text">上传成品图</text>
							</view>
						</view>
						<text class="form-field__hint">
							{{ draft.images.length ? `已添加 ${draft.images.length} 张，首张会作为封面展示。` : `最多上传 ${maxRecipeImages} 张，首张会作为封面展示。` }}
						</text>
					</view>

					<view class="form-field">
						<text class="form-field__label">分类</text>
						<view class="segment">
							<view
								v-for="tab in mealTabs"
								:key="tab.value"
								class="segment__item"
								:class="{ 'segment__item--active': draft.mealType === tab.value }"
								@tap="draft.mealType = tab.value"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="form-field">
						<text class="form-field__label">状态</text>
						<view class="segment">
							<view
								v-for="tab in draftStatusOptions"
								:key="tab.value"
								class="segment__item"
								:class="{
									'segment__item--active': draft.status === tab.value,
									'segment__item--wishlist': draft.status === tab.value && tab.value === 'wishlist',
									'segment__item--done': draft.status === tab.value && tab.value === 'done'
								}"
								@tap="draft.status = tab.value"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="form-field">
						<text class="form-field__label">备注</text>
						<textarea
							v-model="draft.note"
							class="sheet-textarea"
							placeholder="比如口味、做法备注、视频亮点"
							placeholder-class="sheet-textarea__placeholder"
							maxlength="300"
						/>
					</view>
				</scroll-view>

				<view class="sheet__footer">
					<view class="sheet-action" @tap="closeAddSheet">
						<text class="sheet-action__text">取消</text>
					</view>
					<view
						class="sheet-action sheet-action--primary"
						:class="{ 'sheet-action--disabled': !canSubmitDraft }"
						@tap="submitDraft"
					>
						<text class="sheet-action__text sheet-action__text--primary">保存</text>
					</view>
				</view>
			</view>
		</up-popup>
	</view>
</template>

<script>
import { appConfig } from '../../utils/app-config'
import { getBilibiliSessionSetting } from '../../utils/app-settings-api'
import { ensureUploadedImage } from '../../utils/upload-api'
import {
	MAX_RECIPE_IMAGES,
	createRecipeFromDraft,
	getCachedRecipes,
	getRecipeSecondaryText,
	loadRecipes,
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

const statusMap = {
	all: { label: '全部', icon: 'list-dot' },
	wishlist: { label: '想吃', icon: 'heart-fill' },
	done: { label: '吃过', icon: 'checkmark-circle-fill' }
}

const createEmptyDraft = (overrides = {}) => ({
	title: '',
	link: '',
	images: [],
	mealType: 'breakfast',
	status: 'wishlist',
	note: '',
	...overrides
})

export default {
	data() {
		return {
			statusMap,
			activeSection: 'library',
			activeMealType: 'main',
			activeStatus: 'all',
			searchKeyword: '',
			selectedRecipeId: '',
			showAddSheet: false,
			showInviteSheet: false,
			showInviteCodeSheet: false,
			showProfileSheet: false,
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
			appIntroPressProgress: 0,
			hasDismissedProfilePrompt: false,
			syncErrorMessage: '',
			isSyncing: false,
			isSubmittingDraft: false,
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
	},
	onHide() {
		this.clearAppIntroPressState()
	},
	onUnload() {
		this.clearAppIntroPressState()
	},
	onShareAppMessage(res) {
		if (res?.from === 'button' && this.activeInvite?.sharePath) {
			return {
				title: `${this.currentKitchenName || '我们的厨房'} 邀请你一起维护菜单`,
				path: this.activeInvite.sharePath
			}
		}

		return {
			title: '来看看我们的数字厨房',
			path: '/pages/index/index'
		}
	},
	computed: {
		currentMealLabel() {
			return this.mealTabs.find((tab) => tab.value === this.activeMealType)?.label || '早餐'
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
			return this.currentKitchenName || (this.isSyncing ? '正在获取厨房信息' : this.syncErrorMessage || '暂时无法连接厨房')
		},
		currentKitchenRoleLabel() {
			if (this.currentKitchenRole === 'owner') return '创建者'
			if (this.currentKitchenRole === 'admin') return '管理员'
			if (this.currentKitchenRole === 'member') return '成员'
			return ''
		},
		canOpenAppSettings() {
			return !!this.currentUser?.canManageAppSettings
		},
		currentKitchenMetaText() {
			if (!this.currentKitchenName) {
				return this.isSyncing ? '正在同步厨房信息' : this.syncErrorMessage || '创建或加入一个厨房后，会显示在这里。'
			}

			if (this.canSwitchKitchen) {
				return '点击这张卡片，可以切换到其他厨房。'
			}
			return '邀请成员后，大家会看到同一份菜单。'
		},
		doneRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'done')
		},
		librarySummary() {
			if (!this.currentKitchenName && this.syncErrorMessage) {
				return this.syncErrorMessage
			}
			return this.isSyncing ? '正在同步这份菜单。' : '先按早餐和正餐整理，再看想吃和吃过。'
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
			const keyword = this.searchKeyword.trim().toLowerCase()
			return this.recipes.filter((recipe) => {
				const matchedMealType = recipe.mealType === this.activeMealType
				const matchedStatus = this.activeStatus === 'all' || recipe.status === this.activeStatus
				const matchedKeyword =
					!keyword ||
					recipe.title.toLowerCase().includes(keyword) ||
					(recipe.ingredient || '').toLowerCase().includes(keyword) ||
					(recipe.link || '').toLowerCase().includes(keyword) ||
					(recipe.note || '').toLowerCase().includes(keyword)
				return matchedMealType && matchedStatus && matchedKeyword
			})
		},
		inviteSheetSubtitle() {
			if (!this.currentKitchenName) {
				return '发给朋友后，对方输入邀请码即可加入。'
			}
			return `邀请朋友加入「${this.currentKitchenName}」`
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
		}
	},
	methods: {
		applySession(session = getSessionSnapshot()) {
			const snapshot = session || getSessionSnapshot()
			this.currentUser = snapshot?.user || null
			this.kitchenOptions = Array.isArray(snapshot?.kitchens) ? snapshot.kitchens : []
			this.currentKitchenName = snapshot?.currentKitchen?.name || ''
			this.currentKitchenRole = snapshot?.currentKitchen?.role || ''
			if (Number(snapshot?.currentKitchenId) !== this.kitchenMembersKitchenId) {
				this.kitchenMembers = []
				this.kitchenMembersKitchenId = Number(snapshot?.currentKitchenId) || 0
			}
			this.activeInvite = null
			this.inviteCodeCopied = false
			this.maybePromptProfile()
		},
		async refreshRecipes(options = {}) {
			const { silent = true } = options
			const cachedRecipes = getCachedRecipes()
			if (cachedRecipes.length) {
				this.recipes = cachedRecipes
			}

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
				this.recipes = recipes
			} catch (error) {
				this.syncErrorMessage = getFriendlySessionErrorMessage(error)
				this.applySession()
				this.recipes = getCachedRecipes()
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
		recipeSecondaryText(recipe) {
			return getRecipeSecondaryText(recipe)
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
				return '你正在维护这间厨房。'
			}
			return '已加入这间共享厨房。'
		},
		openAppIntro() {
			if (Date.now() - (this.appIntroPressTriggeredAt || 0) < 800) {
				return
			}

			uni.showModal({
				title: '应用简介',
				content: '这是一份给家庭和小团队共用的数字厨房：你可以记录想吃和吃过的菜，也可以贴上 B 站链接，让后台自动整理食材和步骤。',
				showCancel: false,
				confirmText: '知道了'
			})
		},
		handleAppIntroTouchStart() {
			this.clearAppIntroPressState()
			if (this.activeSection !== 'kitchen' || !this.canOpenAppSettings) {
				return
			}

			const startedAt = Date.now()
			this.appIntroPressStartedAt = startedAt
			this.appIntroProgressTimer = setInterval(() => {
				const elapsed = Date.now() - startedAt
				const progress = Math.max(0, Math.min(100, Math.round((elapsed / 2000) * 100)))
				this.appIntroPressProgress = progress
			}, 80)
			this.appIntroPressTimer = setTimeout(() => {
				this.appIntroPressTriggeredAt = Date.now()
				this.clearAppIntroPressState()
				this.openAppSettings()
			}, 2000)
		},
		handleAppIntroTouchEnd() {
			this.clearAppIntroPressState()
		},
		handleAppIntroTouchCancel() {
			this.clearAppIntroPressState()
		},
		clearAppIntroPressState() {
			if (this.appIntroPressTimer) {
				clearTimeout(this.appIntroPressTimer)
				this.appIntroPressTimer = null
			}
			if (this.appIntroProgressTimer) {
				clearInterval(this.appIntroProgressTimer)
				this.appIntroProgressTimer = null
			}
			this.appIntroPressStartedAt = 0
			this.appIntroPressProgress = 0
		},
		async openAppSettings() {
			if (!this.canOpenAppSettings) {
				return
			}

			try {
				await getBilibiliSessionSetting()
			} catch (error) {
				uni.showToast({
					title: error?.message || '暂时无法打开设置',
					icon: 'none'
				})
				return
			}

			uni.navigateTo({
				url: '/pages/app-settings/index'
			})
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
			if (!isProfileIncomplete(this.currentUser)) return
			this.profileDraft = {
				nickname: !isPlaceholderNickname(this.currentUser?.nickname) ? this.currentUser.nickname : '',
				avatarUrl: ''
			}
			this.showProfileSheet = true
		},
		closeProfileSheet() {
			this.showProfileSheet = false
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
				const avatarUrl = await ensureUploadedImage(this.profileDraft.avatarUrl)
				const user = await saveCurrentUserProfile({
					nickname: submittedNickname,
					avatarUrl
				})
				if (!user) {
					throw new Error('当前后端暂不支持保存资料')
				}
				this.showProfileSheet = false
				this.hasDismissedProfilePrompt = true
				this.resetProfileDraft()
				this.applySession()
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
		mealTypeCount(type) {
			return this.recipes.filter((recipe) => recipe.mealType === type).length
		},
		openRecipeDetail(recipeId) {
			this.selectedRecipeId = recipeId
			uni.navigateTo({
				url: `/pages/recipe-detail/index?id=${recipeId}`
			})
		},
		nextStatusText(status) {
			return status === 'done' ? '标记想吃' : '标记吃过'
		},
		toggleRecipeStatus(recipeId) {
			this.toggleRecipeStatusAsync(recipeId)
		},
		async toggleRecipeStatusAsync(recipeId) {
			try {
				await toggleRecipeStatusById(recipeId)
				this.recipes = getCachedRecipes()
			} catch (error) {
				uni.showToast({
					title: error?.message || '更新状态失败',
					icon: 'none'
				})
			}
		},
		drawTonight() {
			const pool = this.wishlistRecipes.length ? this.wishlistRecipes : this.recipes
			if (!pool.length) {
				uni.showToast({
					title: '先添加几道菜吧',
					icon: 'none'
				})
				return
			}
			const picked = pool[Math.floor(Math.random() * pool.length)]
			this.selectedRecipeId = picked.id
			uni.showToast({
				title: `今晚试试：${picked.title}`,
				icon: 'none'
			})
		},
		openAddSheet() {
			this.draft = this.createDraftFromContext()
			this.showAddSheet = true
		},
		closeAddSheet() {
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
			uni.showLoading({
				title: '保存中',
				mask: true
			})

			try {
				const newRecipe = await createRecipeFromDraft(this.draft)
				this.recipes = getCachedRecipes()
				this.selectedRecipeId = newRecipe.id
				this.activeSection = 'library'
				this.activeMealType = newRecipe.mealType
				this.activeStatus = 'all'
				this.searchKeyword = ''
				this.showAddSheet = false
				this.draft = this.createDraftFromContext()
				uni.showToast({
					title: '已保存',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '保存失败',
					icon: 'none'
				})
			} finally {
				this.isSubmittingDraft = false
				uni.hideLoading()
			}
		},
		async openInviteSheet() {
			if (!this.currentKitchenName) {
				await this.refreshRecipes({ silent: false })
			}

			if (!getCurrentKitchenId()) {
				uni.showToast({
					title: '还没拿到厨房信息',
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
					title: '还没拿到厨房信息',
					icon: 'none'
				})
				return
			}

			this.promptKitchenName()
		},
		promptKitchenName() {
			if (this.isSubmittingKitchenName) return

			uni.showModal({
				title: '修改厨房名',
				editable: true,
				content: this.currentKitchenName || '',
				placeholderText: '输入厨房名称',
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
					throw new Error('修改厨房名失败')
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
					title: '厨房名已更新',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '修改厨房名失败',
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
					title: '当前只有一个厨房',
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

<style lang="scss" scoped>
	.app-shell {
		min-height: 100vh;
		background: #f6f4f1;
	}

	.page-content {
		padding: 24rpx 24rpx 176rpx;
	}

	.page-header {
		display: flex;
		flex-direction: column;
		gap: 8rpx;
		padding: 6rpx 2rpx 0;
	}

	.page-header__title {
		font-size: 40rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.page-header__summary {
		font-size: 23rpx;
		line-height: 1.5;
		color: #8d847a;
	}

	.kitchen-hero {
		margin-top: 18rpx;
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.kitchen-card {
		padding: 22rpx 20rpx;
		border-radius: 26rpx;
		background: linear-gradient(135deg, rgba(255, 255, 255, 0.98) 0%, rgba(246, 240, 232, 0.98) 100%);
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow: 0 12rpx 26rpx rgba(56, 44, 30, 0.05);
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}

	.kitchen-card--disabled {
		opacity: 0.78;
	}

	.kitchen-card__header,
	.kitchen-card__switch {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.kitchen-card__badge {
		display: inline-flex;
		align-items: center;
		gap: 8rpx;
		align-self: flex-start;
		padding: 8rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(91, 74, 59, 0.08);
	}

	.kitchen-card__badge-text,
	.kitchen-card__switch-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #6a5a4b;
	}

	.kitchen-card__status-dot {
		width: 14rpx;
		height: 14rpx;
		border-radius: 999rpx;
		background: #b9b0a5;
		box-shadow: 0 0 0 6rpx rgba(185, 176, 165, 0.18);
		flex-shrink: 0;
	}

	.kitchen-card__status-dot--connected {
		background: #78b86d;
		box-shadow: 0 0 0 6rpx rgba(120, 184, 109, 0.16);
	}

	.kitchen-card__name {
		font-size: 38rpx;
		font-weight: 700;
		line-height: 1.28;
		color: #2f2923;
	}

	.kitchen-card__name-row {
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.kitchen-card__name-edit {
		width: 52rpx;
		height: 52rpx;
		border-radius: 16rpx;
		background: rgba(91, 74, 59, 0.08);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.kitchen-card__meta {
		font-size: 24rpx;
		line-height: 1.6;
		color: #8b7e72;
	}

	.kitchen-card__tags {
		display: flex;
		flex-wrap: wrap;
		gap: 12rpx;
	}

	.kitchen-card__tag {
		min-width: 132rpx;
		padding: 14rpx 16rpx;
		border-radius: 18rpx;
		background: rgba(255, 255, 255, 0.7);
		border: 1px solid rgba(91, 74, 59, 0.08);
		display: flex;
		flex-direction: column;
		gap: 4rpx;
	}

	.kitchen-card__tag-value {
		font-size: 28rpx;
		font-weight: 700;
		line-height: 1.2;
		color: #3d3128;
	}

	.kitchen-card__tag-label {
		font-size: 20rpx;
		color: #8b7e72;
	}

	.kitchen-actions {
		padding: 20rpx;
		border-radius: 24rpx;
		background: rgba(255, 255, 255, 0.92);
		border: 1px solid rgba(91, 74, 59, 0.06);
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.04);
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.kitchen-actions__primary {
		padding: 18rpx;
		border-radius: 22rpx;
		background: linear-gradient(180deg, #5c493c 0%, #46362c 100%);
		display: flex;
		align-items: center;
		gap: 16rpx;
	}

	.kitchen-actions__primary-icon {
		width: 64rpx;
		height: 64rpx;
		border-radius: 20rpx;
		background: rgba(255, 255, 255, 0.16);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.kitchen-actions__primary-body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 6rpx;
	}

	.kitchen-actions__primary-title {
		font-size: 27rpx;
		font-weight: 700;
		color: #ffffff;
	}

	.kitchen-actions__primary-desc {
		font-size: 22rpx;
		line-height: 1.5;
		color: rgba(255, 255, 255, 0.74);
	}

	.member-panel {
		margin-top: 18rpx;
		padding: 22rpx 20rpx;
		border-radius: 24rpx;
		background: rgba(255, 255, 255, 0.92);
		border: 1px solid rgba(91, 74, 59, 0.06);
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.04);
	}

	.member-panel__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
	}

	.member-panel__aside {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.member-panel__heading {
		flex: 1;
		min-width: 0;
	}

	.member-panel__title {
		display: block;
		font-size: 28rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.member-panel__desc {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		line-height: 1.55;
		color: #887b6f;
	}

	.member-panel__meta {
		flex-shrink: 0;
		font-size: 22rpx;
		font-weight: 600;
		color: #8a7d70;
	}

	.member-panel__inline-action {
		padding: 8rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(91, 74, 59, 0.08);
	}

	.member-panel__inline-action-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #6e5f50;
	}

	.member-list {
		margin-top: 18rpx;
		display: flex;
		flex-direction: column;
		gap: 12rpx;
	}

	.member-card {
		padding: 16rpx;
		border-radius: 18rpx;
		background: #f7f2ec;
		display: flex;
		align-items: center;
		gap: 14rpx;
	}

	.member-card--self {
		background: #f0e8dc;
		border: 1px solid rgba(91, 74, 59, 0.08);
	}

	.member-card__avatar {
		width: 58rpx;
		height: 58rpx;
		border-radius: 999rpx;
		background: linear-gradient(180deg, #e8d8c5 0%, #dbc4a8 100%);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 24rpx;
		font-weight: 700;
		color: #5b4a3b;
		flex-shrink: 0;
		overflow: hidden;
	}

	.member-card__avatar-image {
		width: 100%;
		height: 100%;
		display: block;
	}

	.member-card__body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4rpx;
	}

	.member-card__top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.member-card__name {
		font-size: 25rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.member-card__meta {
		font-size: 22rpx;
		color: #85796e;
	}

	.member-card__badges {
		display: flex;
		align-items: center;
		gap: 8rpx;
		flex-shrink: 0;
	}

	.member-card__badge {
		padding: 6rpx 12rpx;
		border-radius: 999rpx;
		background: rgba(91, 74, 59, 0.09);
		font-size: 20rpx;
		font-weight: 600;
		color: #6e5f50;
	}

	.member-card__badge--self {
		background: rgba(92, 73, 60, 0.14);
		color: #4c3c31;
	}

	.member-panel__empty {
		margin-top: 18rpx;
	}

	.member-panel__footer {
		margin-top: 18rpx;
		display: flex;
		justify-content: flex-end;
	}

	.member-panel__join-link {
		padding: 10rpx 2rpx;
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
	}

	.member-panel__join-link-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #6e5f50;
	}

	.app-intro {
		margin-top: 18rpx;
		padding: 22rpx 24rpx;
		border-radius: 24rpx;
		background: linear-gradient(135deg, rgba(255, 248, 240, 0.95), rgba(244, 236, 227, 0.96));
		border: 1px solid rgba(184, 166, 148, 0.26);
		box-shadow: 0 16rpx 30rpx rgba(105, 82, 61, 0.08);
		display: flex;
		flex-direction: column;
		gap: 12rpx;
	}

	.app-intro__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16rpx;
	}

	.app-intro__label {
		font-size: 24rpx;
		font-weight: 700;
		color: #5d4c3c;
	}

	.app-intro__hint {
		font-size: 20rpx;
		color: #928474;
	}

	.app-intro__text {
		font-size: 23rpx;
		line-height: 1.7;
		color: #78695b;
	}

	.app-intro__progress-track {
		width: 100%;
		height: 8rpx;
		border-radius: 999rpx;
		background: rgba(120, 105, 91, 0.12);
		overflow: hidden;
	}

	.app-intro__progress-fill {
		height: 100%;
		border-radius: inherit;
		background: linear-gradient(90deg, #d69a63, #7f6750);
		transition: width 0.08s linear;
	}

	.invite-sheet {
		padding: 26rpx 24rpx calc(env(safe-area-inset-bottom) + 24rpx);
		background: #f8f4ee;
	}

	.invite-sheet__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 18rpx;
	}

	.invite-sheet__heading {
		flex: 1;
		min-width: 0;
	}

	.invite-sheet__title {
		display: block;
		font-size: 36rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.invite-sheet__subtitle {
		display: block;
		margin-top: 10rpx;
		font-size: 24rpx;
		line-height: 1.6;
		color: #8a7d70;
	}

	.invite-sheet__close {
		width: 56rpx;
		height: 56rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.75);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.invite-sheet__body {
		max-height: 46vh;
		margin-top: 22rpx;
	}

	.invite-sheet__code-card,
	.invite-sheet__state {
		padding: 24rpx;
		border-radius: 24rpx;
		background: rgba(255, 255, 255, 0.94);
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.04);
	}

	.invite-sheet__stack {
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.invite-sheet__state-desc {
		display: block;
		font-size: 23rpx;
		line-height: 1.6;
		color: #82766b;
	}

	.invite-sheet__meta-line {
		display: block;
		font-size: 21rpx;
		line-height: 1.5;
		color: #908275;
		text-align: center;
	}

	.invite-sheet__state {
		margin-top: 16rpx;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 12rpx;
	}

	.invite-sheet__state-title {
		font-size: 30rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.invite-sheet__code-card {
		position: relative;
		padding: 18rpx 20rpx;
		border-radius: 18rpx;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}

	.invite-sheet__code-card:active {
		transform: scale(0.995);
	}

	.invite-sheet__code-label {
		font-size: 21rpx;
		font-weight: 500;
		color: #a3968a;
	}

	.invite-sheet__code {
		display: block;
		margin-top: 8rpx;
		font-size: 34rpx;
		font-weight: 700;
		letter-spacing: 3rpx;
		color: #2f2923;
		font-family: 'SF Mono', 'Menlo', monospace;
	}

	.invite-sheet__footer {
		margin-top: 22rpx;
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.invite-sheet__button-group {
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.invite-sheet__action {
		width: 100%;
		height: 92rpx;
		padding: 0;
		border-radius: 22rpx;
		background: #ece6de;
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		box-sizing: border-box;
		line-height: 1;
	}

	.invite-sheet__action::after {
		border: none;
	}

	.invite-sheet__action-inner {
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
	}

	.invite-sheet__action--primary {
		background: #3f352d;
	}

	.invite-sheet__action--secondary {
		background: rgba(255, 255, 255, 0.98);
		border: 1px solid rgba(91, 74, 59, 0.08);
	}

	.invite-sheet__action--disabled {
		background: #cfc5bb;
	}

	.invite-sheet__action[disabled] {
		opacity: 0.7;
	}

	.invite-sheet__action-text {
		font-size: 26rpx;
		font-weight: 700;
		color: #5c5146;
	}

	.invite-sheet__action-text--primary {
		color: #ffffff;
	}

	.invite-sheet__action-text--secondary {
		color: #6d6054;
	}

	.invite-sheet__action--disabled .invite-sheet__action-text--primary {
		color: rgba(255, 255, 255, 0.84);
	}

	.invite-sheet__utility {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.invite-sheet__utility-link {
		padding: 8rpx 2rpx;
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
	}

	.invite-sheet__utility-text {
		font-size: 21rpx;
		font-weight: 600;
		color: #6e5f50;
	}

	.invite-code-sheet {
		padding: 26rpx 24rpx calc(env(safe-area-inset-bottom) + 24rpx);
		background: #f8f4ee;
	}

	.invite-code-sheet__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 18rpx;
	}

	.invite-code-sheet__heading {
		flex: 1;
		min-width: 0;
	}

	.invite-code-sheet__title {
		display: block;
		font-size: 36rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.invite-code-sheet__subtitle {
		display: block;
		margin-top: 10rpx;
		font-size: 24rpx;
		line-height: 1.6;
		color: #8a7d70;
	}

	.invite-code-sheet__close {
		width: 56rpx;
		height: 56rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.75);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.invite-code-sheet__body {
		margin-top: 22rpx;
		padding: 24rpx;
		border-radius: 24rpx;
		background: rgba(255, 255, 255, 0.94);
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.04);
	}

	.invite-code-sheet__input {
		height: 96rpx;
		padding: 0 24rpx;
		border-radius: 20rpx;
		background: #f8f3ec;
		font-size: 32rpx;
		font-weight: 700;
		letter-spacing: 3rpx;
		color: #2f2923;
		font-family: 'SF Mono', 'Menlo', monospace;
	}

	.invite-code-sheet__placeholder {
		font-size: 28rpx;
		font-weight: 600;
		letter-spacing: 1rpx;
		color: #b0a59a;
	}

	.invite-code-sheet__hint {
		display: block;
		margin-top: 14rpx;
		font-size: 22rpx;
		line-height: 1.6;
		color: #82766b;
	}

	.invite-code-sheet__footer {
		margin-top: 22rpx;
		display: flex;
		gap: 12rpx;
	}

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

	.toolbar {
		margin-top: 16rpx;
		padding: 18rpx;
		border-radius: 22rpx;
		background: rgba(255, 255, 255, 0.86);
		border: 1px solid rgba(0, 0, 0, 0.03);
		box-shadow: 0 8rpx 20rpx rgba(56, 44, 30, 0.04);
	}

	.toolbar__row {
		display: flex;
		align-items: center;
		gap: 10rpx;
	}

	.meal-tabs {
		margin-top: 14rpx;
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 10rpx;
	}

	.meal-tab {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 18rpx 18rpx;
		border-radius: 18rpx;
		background: #f3efe9;
		border: 1px solid rgba(0, 0, 0, 0.02);
	}

	.meal-tab--active {
		background: #ffffff;
		border: 1px solid rgba(91, 74, 59, 0.14);
		box-shadow: 0 6rpx 16rpx rgba(56, 44, 30, 0.04);
	}

	.meal-tab__left {
		display: flex;
		align-items: center;
		gap: 10rpx;
		min-width: 0;
	}

	.meal-tab__icon-shell {
		width: 28rpx;
		height: 28rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.meal-tab__text {
		font-size: 25rpx;
		font-weight: 700;
		color: #4d433a;
	}

	.meal-tab__count {
		font-size: 22rpx;
		color: #8d847a;
	}

	.search-box {
		flex: 1;
		height: 72rpx;
		display: flex;
		align-items: center;
		gap: 10rpx;
		padding: 0 18rpx;
		border-radius: 16rpx;
		background: #fbfaf8;
		border: 1px solid rgba(0, 0, 0, 0.04);
	}

	.search-box__input {
		flex: 1;
		height: 72rpx;
		font-size: 25rpx;
		color: #2f2923;
	}

	.search-box__placeholder {
		color: #b0a59a;
	}

	.tool-button {
		width: 116rpx;
		height: 72rpx;
		border-radius: 16rpx;
		background: #ece8e2;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8rpx;
	}

	.tool-button__text {
		font-size: 22rpx;
		font-weight: 600;
		color: #4d433a;
	}

	.status-scroll {
		margin-top: 12rpx;
		white-space: nowrap;
	}

	.status-track {
		display: inline-flex;
		gap: 10rpx;
		padding-right: 20rpx;
	}

	.status-pill {
		padding: 12rpx 20rpx;
		border-radius: 999rpx;
		background: #efebe5;
	}

	.status-pill--active {
		background: #2f2923;
	}

	.status-pill__inner {
		display: flex;
		align-items: center;
		gap: 8rpx;
	}

	.status-pill__text {
		font-size: 23rpx;
		font-weight: 600;
		color: #6f655b;
	}

	.status-pill--active .status-pill__text {
		color: #ffffff;
	}

	.list-caption {
		margin-top: 16rpx;
		padding: 0 2rpx;
	}

	.list-caption__text {
		font-size: 22rpx;
		color: #84786d;
	}

	.recipe-list {
		margin-top: 12rpx;
		display: flex;
		flex-direction: column;
		gap: 12rpx;
	}

	.recipe-item {
		display: flex;
		align-items: stretch;
		gap: 14rpx;
		padding: 16rpx;
		border-radius: 20rpx;
		background: rgba(255, 255, 255, 0.9);
		border: 1px solid rgba(0, 0, 0, 0.03);
		box-shadow: 0 8rpx 18rpx rgba(56, 44, 30, 0.04);
		transform: scale(1);
		transition: transform 0.16s ease, border-color 0.16s ease, box-shadow 0.16s ease;
	}

	.recipe-item:active {
		transform: scale(0.992);
	}

	.recipe-item--active {
		border-color: rgba(91, 74, 59, 0.16);
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.06);
	}

	.recipe-item__marker {
		width: 8rpx;
		border-radius: 999rpx;
		background: #d8d0c7;
		flex-shrink: 0;
	}

	.recipe-item__marker--wishlist {
		background: #b59d87;
	}

	.recipe-item__marker--done {
		background: #879884;
	}

	.recipe-item__main {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.recipe-item__top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.recipe-item__text {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 6rpx;
	}

	.recipe-item__title {
		font-size: 28rpx;
		font-weight: 700;
		line-height: 1.34;
		color: #2f2923;
	}

	.recipe-item__meta {
		font-size: 23rpx;
		line-height: 1.55;
		color: #8d847a;
	}

	.recipe-switch {
		position: relative;
		display: flex;
		align-items: center;
		flex-shrink: 0;
		width: 96rpx;
		height: 48rpx;
		padding: 0;
		border-radius: 999rpx;
		background: #efe9e3;
	}

	.recipe-switch__track {
		width: 100%;
		display: flex;
		align-items: center;
		height: 100%;
	}

	.recipe-switch__slot {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.recipe-switch__thumb {
		position: absolute;
		top: 3rpx;
		left: 3rpx;
		width: 42rpx;
		height: 42rpx;
		border-radius: 999rpx;
		background: #ffffff;
		box-shadow: 0 4rpx 10rpx rgba(62, 50, 40, 0.08);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.18s ease;
	}

	.recipe-switch--wishlist {
		background: #f2ebe4;
	}

	.recipe-switch--done {
		background: #e7eee7;
	}

	.recipe-switch--done .recipe-switch__thumb {
		transform: translateX(48rpx);
	}

	.empty-state,
	.soft-empty {
		margin-top: 20rpx;
		padding: 56rpx 30rpx;
		border-radius: 22rpx;
		background: rgba(255, 255, 255, 0.84);
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 12rpx;
	}

	.empty-state__title {
		font-size: 30rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.empty-state__desc,
	.soft-empty__text {
		font-size: 24rpx;
		line-height: 1.6;
		color: #8d847a;
	}

	.soft-empty--inline {
		margin-top: 0;
		padding: 18rpx 16rpx;
		align-items: flex-start;
		text-align: left;
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
		color: #2f2923;
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
		color: #2f2923;
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
		color: #2f2923;
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
		color: #2f2923;
	}

	.simple-list__meta {
		display: block;
		margin-top: 6rpx;
		font-size: 22rpx;
		color: #8d847a;
	}

	.bottom-nav {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 9;
		padding: 12rpx 24rpx calc(env(safe-area-inset-bottom) + 12rpx);
		background: linear-gradient(180deg, rgba(246, 244, 241, 0), rgba(246, 244, 241, 0.85) 18%, rgba(255, 255, 255, 0.98) 34%);
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
	}

	.nav-item,
	.nav-center {
		width: 30%;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10rpx;
	}

	.nav-item__icon-shell {
		width: 80rpx;
		height: 80rpx;
		border-radius: 24rpx;
		background: rgba(255, 255, 255, 0.94);
		box-shadow: 0 10rpx 20rpx rgba(56, 44, 30, 0.05);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.nav-item__label,
	.nav-center__label {
		font-size: 22rpx;
		color: #978b80;
		font-weight: 600;
	}

	.nav-item--active .nav-item__label {
		color: #5b4a3b;
	}

	.nav-center {
		transform: translateY(-18rpx);
	}

	.nav-fab {
		width: 108rpx;
		height: 108rpx;
		border-radius: 999rpx;
		border: 8rpx solid rgba(255, 255, 255, 0.98);
		background: #5b4a3b;
		box-shadow: 0 18rpx 28rpx rgba(91, 74, 59, 0.16);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.sheet {
		height: 78vh;
		background: #ffffff;
		display: flex;
		flex-direction: column;
	}

	.sheet__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
		padding: 28rpx 28rpx 18rpx;
	}

	.sheet__heading {
		flex: 1;
		min-width: 0;
	}

	.sheet__title {
		font-size: 38rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.sheet__subtitle {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		line-height: 1.5;
		color: #9b9186;
	}

	.sheet__close {
		width: 68rpx;
		height: 68rpx;
		border-radius: 18rpx;
		background: #f4f0eb;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.sheet__body {
		flex: 1;
		min-height: 0;
		padding: 0 28rpx 28rpx;
		box-sizing: border-box;
	}

	.form-field {
		display: flex;
		flex-direction: column;
		gap: 12rpx;
		margin-top: 26rpx;
	}

	.form-field:first-child {
		margin-top: 0;
	}

	.form-field__label {
		font-size: 22rpx;
		font-weight: 500;
		color: #9b9186;
	}

	.form-field__hint {
		font-size: 22rpx;
		line-height: 1.6;
		color: #9b9186;
	}

	.sheet-input,
	.sheet-textarea {
		width: 100%;
		box-sizing: border-box;
		border-radius: 24rpx;
		background: #f7f4f0;
		border: 1px solid #ebe4db;
		color: #2f2923;
	}

	.sheet-input {
		height: 88rpx;
		padding: 0 24rpx;
		font-size: 27rpx;
	}

	.sheet-input--title {
		height: 96rpx;
		font-size: 30rpx;
		font-weight: 600;
		background: #ffffff;
		border-color: #e3dbd2;
	}

	.sheet-input__placeholder,
	.sheet-textarea__placeholder {
		color: #b7aea3;
	}

	.sheet-textarea {
		min-height: 180rpx;
		padding: 22rpx 24rpx;
		font-size: 26rpx;
		line-height: 1.6;
	}

	.upload-gallery {
		display: flex;
		flex-wrap: wrap;
		gap: 16rpx;
	}

	.upload-gallery__item,
	.upload-gallery__add {
		position: relative;
		width: calc((100% - 32rpx) / 3);
		height: 176rpx;
		border-radius: 24rpx;
		overflow: hidden;
	}

	.upload-gallery__item {
		background: #ebe4db;
	}

	.upload-gallery__thumb {
		width: 100%;
		height: 100%;
		display: block;
	}

	.upload-gallery__badge {
		position: absolute;
		left: 12rpx;
		bottom: 12rpx;
		padding: 8rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.58);
		backdrop-filter: blur(10rpx);
	}

	.upload-gallery__badge-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.upload-gallery__remove {
		position: absolute;
		top: 12rpx;
		right: 12rpx;
		width: 40rpx;
		height: 40rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.upload-gallery__add {
		border: 1px dashed #d8cec3;
		background: #faf7f3;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12rpx;
	}

	.upload-gallery__plus {
		width: 64rpx;
		height: 64rpx;
		border-radius: 20rpx;
		background: #f1ebe4;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.upload-gallery__add-text {
		font-size: 24rpx;
		font-weight: 600;
		color: #75685c;
	}

	.segment {
		display: flex;
		gap: 10rpx;
		padding: 8rpx;
		border-radius: 24rpx;
		background: #f3efea;
	}

	.segment__item {
		flex: 1;
		height: 76rpx;
		border-radius: 18rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
	}

	.segment__item--active {
		background: #ffffff;
		box-shadow: 0 8rpx 18rpx rgba(59, 47, 36, 0.06);
	}

	.segment__item--wishlist {
		background: #f3e7de;
	}

	.segment__item--done {
		background: #e8efe5;
	}

	.segment__text {
		font-size: 24rpx;
		font-weight: 600;
		color: #867a6f;
	}

	.segment__item--active .segment__text {
		color: #5b4a3b;
	}

	.sheet__footer {
		padding: 18rpx 28rpx calc(env(safe-area-inset-bottom) + 20rpx);
		border-top: 1px solid rgba(91, 74, 59, 0.08);
		background: #ffffff;
		display: flex;
		gap: 16rpx;
	}

	.sheet-action {
		flex: 1;
		width: 100%;
		height: 88rpx;
		padding: 0;
		border-radius: 24rpx;
		background: #f1ede8;
		border: none;
		display: flex;
		align-items: center;
		justify-content: center;
		box-sizing: border-box;
		line-height: 1;
	}

	.sheet-action::after {
		border: none;
	}

	.sheet-action--primary {
		background: #5b4a3b;
		box-shadow: 0 12rpx 20rpx rgba(91, 74, 59, 0.16);
	}

	.sheet-action--disabled {
		background: #d9d1c8;
		box-shadow: none;
		pointer-events: none;
	}

	.sheet-action__text {
		font-size: 28rpx;
		font-weight: 600;
		color: #675c51;
	}

	.sheet-action__text--primary {
		color: #ffffff;
	}

	.sheet-action--disabled .sheet-action__text--primary {
		opacity: 0.76;
	}
</style>
