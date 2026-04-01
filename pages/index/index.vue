<template>
	<view class="app-shell">
		<view class="page-content" :class="{ 'page-content--meal-order': showMealOrderFloatingBar }">
			<template v-if="activeSection === 'library'">
				<view class="page-header" :class="{ 'page-header--meal-order': isLibraryMealOrderMode }">
					<view class="page-header__top">
						<view class="page-header__heading">
							<view class="page-header__title-row">
								<view
									class="page-header__title-mark"
									:class="isLibraryMealOrderMode ? 'page-header__title-mark--meal-order' : 'page-header__title-mark--library'"
								>
									<up-icon
										:name="isLibraryMealOrderMode ? 'heart-fill' : 'grid-fill'"
										size="14"
										:color="isLibraryMealOrderMode ? '#bf715f' : '#7a6755'"
									></up-icon>
								</view>
								<text class="page-header__title">{{ libraryHeaderTitle }}</text>
							</view>
							<text v-if="libraryHeaderSummary" class="page-header__summary">{{ libraryHeaderSummary }}</text>
						</view>
						<view v-if="isLibraryMealOrderMode" class="meal-order-mode-bar__actions page-header__mode-actions">
							<view class="meal-order-mode-bar__chip meal-order-mode-bar__chip--accent" @tap="openMealOrderDateSheet">
								<text class="meal-order-mode-bar__chip-text">改日期</text>
							</view>
							<view class="meal-order-mode-bar__chip meal-order-mode-bar__chip--ghost" @tap="exitMealOrderMode">
								<text class="meal-order-mode-bar__chip-text">返回</text>
							</view>
						</view>
						<view v-else class="page-header__action" @tap="openMealOrderDateSheet">
							<up-icon name="calendar" size="15" color="#745742"></up-icon>
							<text class="page-header__action-text">安排菜单</text>
						</view>
					</view>
				</view>

				<view
					v-if="!isLibraryMealOrderMode"
					class="meal-order-spotlight"
					:class="{ 'meal-order-spotlight--empty': !mealOrderSpotlightRecord }"
					@tap="handleMealOrderSpotlightTap"
					@touchstart="handleMealOrderSpotlightTouchStart"
					@touchend="handleMealOrderSpotlightTouchEnd"
				>
					<view class="meal-order-spotlight__main">
						<text class="meal-order-spotlight__title">{{ mealOrderSpotlightTitle }}</text>
						<text class="meal-order-spotlight__desc">{{ mealOrderSpotlightDesc }}</text>
					</view>
					<view class="meal-order-spotlight__aside">
						<text v-if="mealOrderSpotlightMetaText" class="meal-order-spotlight__meta-text">{{ mealOrderSpotlightMetaText }}</text>
						<up-icon name="arrow-right" size="16" color="#8a7968"></up-icon>
					</view>
				</view>
				<view class="toolbar">
					<view class="toolbar__search-row">
						<view
							class="search-box"
							:class="{ 'search-box--active': isSearchFocused || trimmedSearchKeyword }"
						>
							<up-icon name="search" size="15" color="#8f8377"></up-icon>
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
								<up-icon name="close" size="14" color="#8f8377"></up-icon>
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
								:class="{ 'status-pill--active': activeStatus === tab.value }"
								@tap="activeStatus = tab.value"
							>
								<view class="status-pill__inner">
									<up-icon
										:name="statusMap[tab.value].icon"
										size="13"
										:color="activeStatus === tab.value ? '#fffaf3' : '#7a6d61'"
									></up-icon>
									<text class="status-pill__text">{{ tab.label }}</text>
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
								<up-icon name="reload" size="13" color="#6f6154"></up-icon>
								<text class="list-caption__pick-text">帮我选</text>
							</view>
						</view>
					</view>
				</view>

				<view v-if="filteredRecipes.length" class="recipe-list">
					<view
						v-for="card in recipeCards"
						:key="card.id"
						class="recipe-card"
						:class="{
							'recipe-card--active': selectedRecipeId === card.id,
							'recipe-card--pinned': card.isPinned,
							'recipe-card--meal-order-selected': isLibraryMealOrderMode && mealOrderHasRecipe(card.id)
						}"
						@tap="openRecipeDetail(card.id)"
					>
						<view class="recipe-card__media" :class="{ 'recipe-card__media--empty': !getRecipeCardDisplayCover(card) }">
							<image
								v-if="getRecipeCardDisplayCover(card)"
								class="recipe-card__image"
								:src="getRecipeCardDisplayCover(card)"
								mode="aspectFill"
								@error="handleRecipeCardImageError(card)"
							></image>
							<view v-else class="recipe-card__placeholder">
								<view class="recipe-card__placeholder-icon">
									<up-icon :name="card.placeholderIcon" size="26" color="#866d58"></up-icon>
								</view>
								<text class="recipe-card__placeholder-text">暂无图片</text>
							</view>
							<view
								v-if="isLibraryMealOrderMode && mealOrderHasRecipe(card.id)"
								class="recipe-card__selected-badge"
							>
								<up-icon name="checkmark" size="10" color="#fff9f1"></up-icon>
								<text class="recipe-card__selected-badge-text">已选</text>
							</view>
							<view v-if="card.sourceBadge && !isLibraryMealOrderMode" class="recipe-card__source-badge">
								<text class="recipe-card__source-badge-text">{{ card.sourceBadge }}</text>
							</view>
							<view v-if="card.imageCount > 1 && !isLibraryMealOrderMode" class="recipe-card__count">
								<text class="recipe-card__count-text">{{ card.imageCount }}</text>
							</view>
						</view>
						<view class="recipe-card__body">
							<view class="recipe-card__top">
								<view class="recipe-card__title-wrap">
									<view class="recipe-card__title-row">
										<view v-if="card.isPinned" class="recipe-card__pin-badge">
											<text class="recipe-card__pin-badge-text">置顶</text>
										</view>
										<text class="recipe-card__title">{{ card.title }}</text>
									</view>
								</view>
								<view
									v-if="!isLibraryMealOrderMode"
									class="recipe-switch"
									:class="'recipe-switch--' + card.status"
									@tap.stop="toggleRecipeStatus(card.id)"
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
											:name="statusMap[card.status].icon"
											size="12"
											:color="card.status === 'done' ? '#6f826d' : '#9a7b65'"
										></up-icon>
									</view>
								</view>
								<view v-else class="meal-order-control" @tap.stop>
								<view
									class="meal-order-add"
									:class="{ 'meal-order-add--active': mealOrderHasRecipe(card.id) }"
									@tap.stop="toggleMealOrderRecipe(card)"
								>
									<up-icon
										v-if="mealOrderHasRecipe(card.id)"
										class="meal-order-add__icon"
										name="checkmark"
										size="12"
										color="#fffaf3"
									></up-icon>
									<text
										class="meal-order-add__text"
										:class="{ 'meal-order-add__text--active': mealOrderHasRecipe(card.id) }"
									>
											{{ mealOrderHasRecipe(card.id) ? '已加入' : '加入菜单' }}
										</text>
									</view>
								</view>
							</view>
							<text v-if="!isLibraryMealOrderMode" class="recipe-card__info">{{ card.infoLine }}</text>
							<text v-if="!isLibraryMealOrderMode" class="recipe-card__summary">{{ card.listSummary }}</text>
						</view>
					</view>
				</view>

				<view v-else class="empty-state">
					<up-icon name="empty-search" size="40" color="#c0b3a5"></up-icon>
					<text class="empty-state__title">{{ emptyStateTitle }}</text>
					<text class="empty-state__desc">{{ emptyStateDesc }}</text>
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
							:class="{ 'member-card--self': member.isCurrentUser, 'member-card--interactive': member.isCurrentUser }"
							:hover-class="member.isCurrentUser ? 'member-card--hover' : ''"
							@tap="handleMemberCardTap(member)"
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
								<view class="member-card__meta-row">
									<text class="member-card__meta">{{ memberMemberDescription(member) }}</text>
									<view v-if="member.isCurrentUser" class="member-card__action">
										<text class="member-card__action-text">修改资料</text>
										<up-icon name="arrow-right" size="12" color="#8a7d70"></up-icon>
									</view>
								</view>
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

		<view class="bottom-nav" :class="{ 'bottom-nav--meal-order': showMealOrderFloatingBar }">
			<view
				class="nav-item"
				:class="{ 'nav-item--active': activeSection === 'library' }"
				@tap="switchSection('library')"
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
				@tap="switchSection('kitchen')"
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
			:show="showMealOrderDateSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeMealOrderDateSheet"
		>
			<view class="meal-order-sheet">
				<view class="meal-order-sheet__header">
					<view class="meal-order-sheet__heading">
						<text class="meal-order-sheet__title">哪天一起吃</text>
						<text class="meal-order-sheet__subtitle">先挑个日子，再把想吃的菜慢慢放进这天的小菜单里。</text>
					</view>
					<view class="meal-order-sheet__close" @tap="closeMealOrderDateSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<view class="meal-order-date-grid">
					<view
						v-for="option in mealOrderQuickDateOptions"
						:key="option.value"
						class="meal-order-date-card"
						:class="{ 'meal-order-date-card--active': option.value === mealOrderDatePickerValue }"
						@tap="startMealOrderMode(option.value)"
					>
						<text class="meal-order-date-card__label">{{ option.label }}</text>
						<text class="meal-order-date-card__date">{{ option.dateText }}</text>
					</view>
				</view>

				<picker mode="date" :value="mealOrderDatePickerValue" :start="mealOrderDateStart" @change="handleMealOrderDatePickerChange">
					<view class="meal-order-date-picker">
						<text class="meal-order-date-picker__text">自选日期</text>
						<up-icon name="calendar" size="16" color="#6f5f50"></up-icon>
					</view>
				</picker>
			</view>
		</up-popup>

		<up-popup
			:show="showMealOrderSpotlightSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeMealOrderSpotlightSheet"
		>
			<view
				v-if="mealOrderSpotlightRecord"
				class="meal-order-sheet"
				@touchstart="handleMealOrderSpotlightTouchStart"
				@touchend="handleMealOrderSpotlightTouchEnd"
			>
				<view class="meal-order-sheet__header">
					<view class="meal-order-sheet__heading">
						<text class="meal-order-sheet__eyebrow">{{ mealOrderSpotlightDetailEyebrow }}</text>
						<text class="meal-order-sheet__title">{{ mealOrderSpotlightTitle }}</text>
						<text class="meal-order-sheet__subtitle">{{ mealOrderSpotlightDetailSubtitle }}</text>
					</view>
					<view class="meal-order-sheet__close" @tap="closeMealOrderSpotlightSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<scroll-view class="meal-order-cart-list" scroll-y>
					<view class="meal-order-checkout-list">
						<view
							v-for="item in mealOrderSpotlightDetailItems"
							:key="`meal-order-spotlight-${item.recipeId}`"
							class="meal-order-checkout-item"
						>
							<text class="meal-order-checkout-item__title">{{ item.title }}</text>
						</view>
					</view>
				</scroll-view>

				<view v-if="mealOrderSpotlightDetailNote" class="meal-order-checkout-note">
					<text class="meal-order-checkout-note__label">备注</text>
					<text class="meal-order-checkout-note__text">{{ mealOrderSpotlightDetailNote }}</text>
				</view>

				<view class="meal-order-sheet__footer">
					<view class="sheet-action" @tap="closeMealOrderSpotlightSheet">
						<text class="sheet-action__text">关闭</text>
					</view>
					<view
						v-if="mealOrderSpotlightCanResume"
						class="sheet-action sheet-action--primary"
						@tap="resumeMealOrderSpotlightRecord"
					>
						<text class="sheet-action__text sheet-action__text--primary">继续安排</text>
					</view>
				</view>
			</view>
		</up-popup>

		<up-popup
			:show="showMealOrderCartSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeMealOrderCartSheet"
		>
			<view class="meal-order-sheet">
				<view class="meal-order-sheet__header">
					<view class="meal-order-sheet__heading">
						<text class="meal-order-sheet__title">这天的小菜单</text>
						<text class="meal-order-sheet__subtitle">{{ mealOrderDateText }} · 已选 {{ mealOrderCartDishCount }} 道</text>
					</view>
					<view class="meal-order-sheet__close" @tap="closeMealOrderCartSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<scroll-view class="meal-order-cart-list" scroll-y>
					<view v-if="mealOrderCartItems.length" class="meal-order-cart-stack">
						<view v-for="item in mealOrderCartItems" :key="`meal-order-cart-${item.recipeId}`" class="meal-order-cart-item">
							<view class="meal-order-cart-item__main">
								<text class="meal-order-cart-item__title">{{ item.title }}</text>
							</view>
							<view class="meal-order-cart-item__action" @tap="removeMealOrderRecipe(item.recipeId)">
								<text class="meal-order-cart-item__action-text">移出</text>
							</view>
						</view>
					</view>
					<view v-else class="soft-empty meal-order-cart-empty">
						<text class="soft-empty__text">还没选菜，先去美食库慢慢挑两道喜欢的吧。</text>
					</view>
				</scroll-view>

				<view class="meal-order-note">
					<text class="meal-order-note__label">想说的话</text>
					<textarea
						:value="mealOrderDraftNote"
						class="meal-order-note__input"
						placeholder="比如：周六想吃热乎一点，提前把牛肉腌上"
						placeholder-class="meal-order-note__placeholder"
						maxlength="120"
						@input="handleMealOrderNoteInput"
					/>
				</view>

				<view class="meal-order-sheet__footer">
					<view class="sheet-action" @tap="clearMealOrderCart">
						<text class="sheet-action__text">清空</text>
					</view>
					<view
						class="sheet-action sheet-action--primary"
						:class="{ 'sheet-action--disabled': !mealOrderCanCheckout }"
						@tap="openMealOrderCheckoutSheet"
					>
						<text class="sheet-action__text sheet-action__text--primary">确认菜单</text>
					</view>
				</view>
			</view>
		</up-popup>

		<up-popup
			:show="showMealOrderCheckoutSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeMealOrderCheckoutSheet"
		>
			<view class="meal-order-sheet">
				<view class="meal-order-sheet__header">
					<view class="meal-order-sheet__heading">
						<text class="meal-order-sheet__title">一起确认菜单</text>
						<text class="meal-order-sheet__subtitle">{{ mealOrderDateText }} · 共 {{ mealOrderCartDishCount }} 道</text>
					</view>
					<view class="meal-order-sheet__close" @tap="closeMealOrderCheckoutSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<scroll-view class="meal-order-cart-list" scroll-y>
					<view v-if="mealOrderCartItems.length" class="meal-order-checkout-list">
						<view v-for="item in mealOrderCartItems" :key="`meal-order-checkout-${item.recipeId}`" class="meal-order-checkout-item">
							<text class="meal-order-checkout-item__title">{{ item.title }}</text>
						</view>
					</view>
					<view v-else class="soft-empty meal-order-cart-empty">
						<text class="soft-empty__text">这天还没有安排菜，先回去挑一挑。</text>
					</view>
				</scroll-view>

				<view v-if="mealOrderDraftNote" class="meal-order-checkout-note">
					<text class="meal-order-checkout-note__label">备注</text>
					<text class="meal-order-checkout-note__text">{{ mealOrderDraftNote }}</text>
				</view>

				<view class="meal-order-sheet__footer">
					<view class="sheet-action" @tap="closeMealOrderCheckoutSheet">
						<text class="sheet-action__text">返回修改</text>
					</view>
					<view
						class="sheet-action sheet-action--primary"
						:class="{ 'sheet-action--disabled': !mealOrderCanCheckout }"
						@tap="submitMealOrder"
					>
						<text class="sheet-action__text sheet-action__text--primary">
							{{ isSubmittingMealOrder ? '安排中...' : '安排这天菜单' }}
						</text>
					</view>
				</view>
			</view>
		</up-popup>

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
						<text class="profile-sheet__title">{{ profileSheetTitle }}</text>
						<text class="profile-sheet__subtitle">{{ profileSheetSubtitle }}</text>
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
							<text class="sheet-action__text">{{ profileSheetSecondaryActionText }}</text>
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
						<text class="form-field__label">菜谱链接</text>
						<input
							:value="draft.link"
							class="sheet-input"
							placeholder="支持直接粘贴 B 站或小红书分享链接"
							placeholder-class="sheet-input__placeholder"
							maxlength="300"
							@input="handleDraftLinkInput"
						/>
						<text v-if="draftLinkAssistText" class="form-field__hint">{{ draftLinkAssistText }}</text>
					</view>

					<view class="form-field">
						<text class="form-field__label">菜名</text>
						<input
							:value="draft.title"
							class="sheet-input sheet-input--title"
							placeholder="可手动填写，或等待系统自动识别"
							placeholder-class="sheet-input__placeholder"
							maxlength="40"
							@input="handleDraftTitleInput"
						/>
						<text v-if="draftTitleAssistText" class="form-field__hint">{{ draftTitleAssistText }}</text>
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
import { listMealPlanStore, saveMealPlanDraft, submitMealPlan as submitMealPlanRequest } from '../../utils/meal-plan-api'
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

const firstUrlPattern = /https?:\/\/[^\s]+/i
const draftLinkTrailingPunctuationPattern = /[).,，。！!？?】）'"”’]+$/
const draftPlatformPattern = /\s*-\s*(哔哩哔哩|小红书)\s*$/i
const draftShareSuffixPattern = /复制后打开【小红书】查看笔记!?/g
const draftWhitespacePattern = /\s+/g
const draftSplitPattern = /[!！?？~～|｜/·•,:，。；;、\s]+/
const draftLowConfidencePattern = /教程|做法|分享|来咯|来啦|来了|最好吃|就是这个味|超级软烂|超软烂|入口即化|香迷糊|巨好吃|真的绝了|一学就会|零失败|保姆级|超下饭|超级入味/i
const draftNarrativePattern = /我做了|我家|我们家|拿手菜|私房菜|祖传|开店|饭店|餐馆|摆摊|多年|[0-9一二三四五六七八九十两]+年/i
const draftDishPattern = /炖|炒|烧|煮|蒸|焖|拌|炸|卤|煎|烤|焗|煲|炝|凉拌|清蒸|红烧|糖醋|牛腩|牛肉|排骨|鸡翅|鸡腿|五花肉|里脊|番茄|西红柿|土豆|茄子|豆腐|虾|鱼|面|饭|粥|汤|蛋/i
const draftDescriptorPattern = /鲜香|入味|浓稠|软烂|下饭|香辣|酸甜|麻辣|清爽|酥脆|嫩滑|家常|科学/i
const draftNoisePatterns = [
	/(.+?)(?:最好吃的做法|家常做法|详细做法|做法分享|做法教程|做法来了|做法来咯|做法来啦|教程来咯|教程来啦|教程来了|教程分享|教程|做法).*$/i,
	/(.+?)(?:就是这个味|超级软烂|超软烂|入口即化|香迷糊了?|巨好吃|好吃到哭|一学就会|零失败|保姆级|超下饭|真的绝了?|超级入味).*$/i
]
const RECENT_SEARCH_STORAGE_KEY = 'caipu-miniapp-recent-searches'
const LAST_DRAFT_LINK_PREFILL_STORAGE_KEY = 'caipu-miniapp-last-draft-link-prefill'
const MAX_RECENT_SEARCHES = 6
const mealOrderWeekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const searchSuggestionKeywordsByMeal = {
	breakfast: ['鸡蛋', '面包', '粥', '快手'],
	main: ['下饭', '牛肉', '鸡翅', '汤']
}

function padDateNumber(value) {
	return String(Number(value) || 0).padStart(2, '0')
}

function toISODate(value = new Date()) {
	const date = value instanceof Date ? value : new Date(value)
	if (Number.isNaN(date.getTime())) return ''
	return `${date.getFullYear()}-${padDateNumber(date.getMonth() + 1)}-${padDateNumber(date.getDate())}`
}

function parseISODate(value = '') {
	const match = String(value || '').trim().match(/^(\d{4})-(\d{2})-(\d{2})$/)
	if (!match) return null
	const year = Number(match[1])
	const month = Number(match[2]) - 1
	const day = Number(match[3])
	const date = new Date(year, month, day)
	if (Number.isNaN(date.getTime())) return null
	if (date.getFullYear() !== year || date.getMonth() !== month || date.getDate() !== day) return null
	return date
}

function normalizeMealOrderDate(value = '') {
	const date = parseISODate(value)
	return date ? toISODate(date) : ''
}

function addDaysFromISODate(baseDate = '', offset = 0) {
	const date = parseISODate(baseDate) || new Date()
	date.setDate(date.getDate() + Number(offset || 0))
	return toISODate(date)
}

function nextWeekendISODate(baseDate = '') {
	const seed = parseISODate(baseDate) || new Date()
	for (let index = 0; index < 8; index += 1) {
		const candidate = new Date(seed)
		candidate.setDate(seed.getDate() + index)
		const day = candidate.getDay()
		if (day === 0 || day === 6) {
			return toISODate(candidate)
		}
	}
	return toISODate(seed)
}

function formatMealOrderDateText(value = '') {
	const date = parseISODate(value)
	if (!date) return '--'
	const month = padDateNumber(date.getMonth() + 1)
	const day = padDateNumber(date.getDate())
	const weekday = mealOrderWeekdays[date.getDay()] || ''
	return `${month}月${day}日 ${weekday}`
}

function formatMealOrderHeaderTitle(value = '') {
	const date = parseISODate(value)
	if (!date) return '这天的小菜单'
	return `${date.getMonth() + 1}月${date.getDate()}日的小菜单`
}

function createEmptyMealOrderStore() {
	return {
		drafts: {},
		submitted: []
	}
}

function normalizeMealOrderItem(raw = {}) {
	const quantity = Math.max(1, Math.min(9, Number(raw.quantity) || 1))
	const recipeId = String(raw.recipeId || '').trim()
	if (!recipeId) return null
	const titleSnapshot = String(raw.titleSnapshot || raw.title || '').trim() || '未命名菜品'
	const imageSnapshot = String(raw.imageSnapshot || raw.image || '').trim()
	const mealTypeSnapshot = String(raw.mealTypeSnapshot || raw.mealType || '').trim() || 'main'

	return {
		recipeId,
		quantity,
		titleSnapshot,
		imageSnapshot,
		mealTypeSnapshot
	}
}

function normalizeMealOrderDraft(raw = {}, planDate = '') {
	const normalizedPlanDate = normalizeMealOrderDate(planDate || raw.planDate || '')
	const items = (Array.isArray(raw.items) ? raw.items : [])
		.map((item) => normalizeMealOrderItem(item))
		.filter(Boolean)
	const note = String(raw.note || '').trim()
	const updatedAt = String(raw.updatedAt || '').trim()

	return {
		planDate: normalizedPlanDate,
		items,
		note,
		updatedAt
	}
}

function normalizeMealOrderRecord(raw = {}) {
	const planDate = normalizeMealOrderDate(raw.planDate || '')
	const items = (Array.isArray(raw.items) ? raw.items : [])
		.map((item) => normalizeMealOrderItem(item))
		.filter(Boolean)
	const note = String(raw.note || '').trim()
	const submittedAt = String(raw.submittedAt || '').trim()
	if (!planDate || !items.length) return null

	return {
		id: String(raw.id || '').trim() || `mord_${Date.now()}`,
		planDate,
		items,
		note,
		submittedAt
	}
}

function normalizeMealOrderStore(raw = {}) {
	const source = raw && typeof raw === 'object' ? raw : {}
	const draftSource = source.drafts && typeof source.drafts === 'object' ? source.drafts : {}
	const drafts = {}
	Object.keys(draftSource).forEach((dateKey) => {
		const normalizedDate = normalizeMealOrderDate(dateKey)
		if (!normalizedDate) return
		const draft = normalizeMealOrderDraft(draftSource[dateKey], normalizedDate)
		if (!draft.items.length && !draft.note) return
		drafts[normalizedDate] = draft
	})

	const submitted = (Array.isArray(source.submitted) ? source.submitted : [])
		.map((record) => normalizeMealOrderRecord(record))
		.filter(Boolean)
		.sort((left, right) => String(right.submittedAt || '').localeCompare(String(left.submittedAt || '')))
		.slice(0, 60)

	return {
		drafts,
		submitted
	}
}

function buildMealPlanPayload(raw = {}) {
	const draft = normalizeMealOrderDraft(raw, raw?.planDate)
	return {
		items: draft.items.map((item) => ({
			recipeId: item.recipeId,
			quantity: item.quantity,
			titleSnapshot: item.titleSnapshot,
			imageSnapshot: item.imageSnapshot,
			mealTypeSnapshot: item.mealTypeSnapshot
		})),
		note: draft.note
	}
}

function buildMealOrderDishSummary(items = []) {
	const names = (Array.isArray(items) ? items : [])
		.map((item) => String(item?.titleSnapshot || '').trim())
		.filter(Boolean)
		.slice(0, 3)
	if (!names.length) return '还没有菜'
	return names.join(' / ')
}

function detectDraftLinkPlatform(input = '') {
	const value = String(input || '').toLowerCase()
	if (!value) return ''
	if (value.includes('bilibili.com') || value.includes('b23.tv') || value.includes('bili2233.cn')) return 'bilibili'
	if (value.includes('xiaohongshu.com') || value.includes('xhslink.com')) return 'xiaohongshu'
	return ''
}

function extractSupportedDraftLink(input = '') {
	const raw = String(input || '').trim()
	if (!raw) return ''

	const match = raw.match(firstUrlPattern)
	let candidate = String(match?.[0] || '').trim()
	if (!candidate && detectDraftLinkPlatform(raw)) {
		candidate = raw
	}
	if (!candidate) return ''

	candidate = candidate.replace(draftLinkTrailingPunctuationPattern, '').trim()
	return detectDraftLinkPlatform(candidate) ? candidate : ''
}

function normalizeDraftAutoTitle(input = '') {
	let title = String(input || '').trim()
	if (!title) return ''
	const bracketMatch = title.match(/[【\[]([^】\]]+)[】\]]/)
	if (bracketMatch && bracketMatch[1]) {
		title = bracketMatch[1].trim()
	}
	title = title.replace(draftPlatformPattern, '').replace(draftShareSuffixPattern, '').trim()
	title = trimTrailingDraftTag(title)
	title = title.replace(draftWhitespacePattern, ' ').trim()
	title = trimDraftTitleNoise(title)
	title = chooseDraftTitleCandidate(title)
	title = title.replace(/[。！!~～\s]+$/g, '').trim()
	return title
}

function trimDraftTitleNoise(title = '') {
	let value = String(title || '').trim()
	if (!value) return ''

	draftNoisePatterns.some((pattern) => {
		const match = value.match(pattern)
		if (!match || !match[1]) return false
		const candidate = String(match[1] || '').trim()
		if ([...candidate].length < 2) return false
		value = candidate
		return true
	})

	return value.replace(/[。！!~～\s]+$/g, '').trim()
}

function isLowConfidenceDraftTitle(title = '') {
	return scoreDraftTitleCandidate(title) < 5
}

function scoreDraftTitleCandidate(title = '') {
	const value = String(title || '').trim()
	if (!value) return -100
	const length = [...value].length
	let score = 0

	if (length < 2) {
		score -= 8
	} else if (length <= 12) {
		score += 4
	} else if (length <= 16) {
		score += 2
	} else if (length <= 20) {
		score -= 1
	} else {
		score -= 5
	}

	if (draftDishPattern.test(value)) score += 5
	if (draftLowConfidencePattern.test(value)) score -= 3
	if (draftNarrativePattern.test(value)) score -= 4
	if (value.includes('的') && draftDescriptorPattern.test(value)) score -= 1

	return score
}

function chooseDraftTitleCandidate(title = '') {
	const value = String(title || '').trim()
	if (!value) return ''

	const candidates = collectDraftTitleCandidates(value)
	let best = value
	let bestScore = scoreDraftTitleCandidate(value)
	let bestLength = [...value].length

	candidates.forEach((candidate) => {
		const score = scoreDraftTitleCandidate(candidate)
		const length = [...candidate].length
		if (score > bestScore || (score === bestScore && length < bestLength)) {
			best = candidate
			bestScore = score
			bestLength = length
		}
	})

	return best
}

function collectDraftTitleCandidates(title = '') {
	const candidates = []
	const appendCandidate = (raw = '') => {
		const candidate = trimDraftTitleNoise(trimTrailingDraftTag(String(raw || '').trim()))
			.replace(/[。！!~～\s]+$/g, '')
			.trim()
			.replace(/^[【\[]+|[】\]]+$/g, '')
			.trim()
		if ([...candidate].length < 2) return
		if (candidates.includes(candidate)) return
		candidates.push(candidate)
	}

	appendCandidate(title)
	String(title || '')
		.split(draftSplitPattern)
		.forEach((segment) => appendCandidate(segment))

	;[...candidates].forEach((candidate) => {
		const index = candidate.lastIndexOf('的')
		if (index >= 0 && index < candidate.length - 1) {
			appendCandidate(candidate.slice(index + 1))
		}
	})

	return candidates
}

function trimTrailingDraftTag(title = '') {
	const value = String(title || '').trim()
	const lastBracket = Math.max(value.lastIndexOf('【'), value.lastIndexOf('['))
	if (lastBracket > 0) {
		return value.slice(0, lastBracket).trim()
	}
	return value
}

function guessDraftTitleFromShareText(input = '') {
	const raw = String(input || '').trim()
	if (!raw) return ''

	const text = raw
		.replace(firstUrlPattern, ' ')
		.replace(/复制后打开【小红书】查看笔记!?/g, ' ')
		.replace(/发布了一篇小红书笔记.*$/g, ' ')
		.replace(/快来看吧.*$/g, ' ')
		.replace(/\s+/g, ' ')
		.trim()

	if (!text) return ''

	const bracketMatch = text.match(/[【\[]([^】\]]+)[】\]]/)
	if (bracketMatch && bracketMatch[1]) {
		return normalizeDraftAutoTitle(bracketMatch[1])
	}

	const line = text.split(/[。\n]/).map((item) => item.trim()).filter(Boolean)[0] || ''
	return normalizeDraftAutoTitle(line)
}

function buildRecipeInfoLine(recipe = {}) {
	const mealLabel = mealTypeLabelMap[recipe.mealType] || '早餐'
	const parseStatus = String(recipe.parseStatus || '').trim()

	let parseLabel = '手动整理'
	if (parseStatus === 'done') {
		parseLabel = '已整理'
	} else if (parseStatus === 'pending' || parseStatus === 'processing') {
		parseLabel = '整理中'
	} else if (parseStatus === 'failed') {
		parseLabel = '待重试'
	} else if (String(recipe.link || '').trim()) {
		parseLabel = '可整理'
	}

	return `${mealLabel} · ${parseLabel}`
}

function detectRecipeSource(recipe = {}) {
	const parseSource = String(recipe.parseSource || '').trim().toLowerCase()
	const link = String(recipe.link || '').trim().toLowerCase()
	if (parseSource.includes('bilibili') || link.includes('bilibili.com') || link.includes('b23.tv') || link.includes('bili2233.cn')) {
		return 'B站'
	}
	if (parseSource.includes('xiaohongshu') || link.includes('xiaohongshu.com') || link.includes('xhslink.com')) {
		if (parseSource.includes(':video')) return '小红书视频'
		if (parseSource.includes(':image')) return '小红书图文'
		return '小红书'
	}
	if (link) return '链接'
	return ''
}

function pickRecipePlaceholderIcon(recipe = {}) {
	return recipe.mealType === 'main' ? 'grid-fill' : 'clock-fill'
}

function extractRecipeImages(recipe = {}) {
	if (Array.isArray(recipe.imageUrls) && recipe.imageUrls.length) {
		return recipe.imageUrls.filter(Boolean)
	}
	if (Array.isArray(recipe.images) && recipe.images.length) {
		return recipe.images.filter(Boolean)
	}
	return [recipe.image, recipe.imageUrl].filter(Boolean)
}

function truncateTextByRune(value = '', maxLength = 15) {
	const items = Array.from(String(value || '').trim())
	if (items.length <= maxLength) return items.join('')
	return items.slice(0, maxLength).join('')
}

function buildRecipeListSummary(recipe = {}) {
	return truncateTextByRune(String(recipe.summary || '').trim(), 24)
}

function buildRecipeCoverVersion(recipe = {}) {
	return String(recipe.updatedAt || recipe.parseFinishedAt || '').trim()
}

function buildRecipeSearchText(recipe = {}) {
	const parsedContent = recipe.parsedContent || {}
	const ingredientLines = [
		...(Array.isArray(parsedContent.ingredients) ? parsedContent.ingredients : []),
		...(Array.isArray(parsedContent.mainIngredients) ? parsedContent.mainIngredients : []),
		...(Array.isArray(parsedContent.secondaryIngredients) ? parsedContent.secondaryIngredients : [])
	]
	const stepLines = (Array.isArray(parsedContent.steps) ? parsedContent.steps : []).reduce((result, step) => {
		if (typeof step === 'string') {
			return result.concat(step)
		}
		return result.concat([step?.title, step?.detail, step?.text].filter(Boolean))
	}, [])
		.filter(Boolean)

	return [
		recipe.title,
		recipe.summary,
		recipe.ingredient,
		recipe.note,
		recipe.link,
		...ingredientLines,
		...stepLines
	]
		.filter(Boolean)
		.join('\n')
		.toLowerCase()
}

function buildRecipeCard(recipe = {}, cachedCoverMap = {}) {
	const images = extractRecipeImages(recipe)
	const remoteCover = images[0] || ''
	const cachedCover = cachedCoverMap[recipe.id] || ''
	return {
		...recipe,
		cover: cachedCover || remoteCover,
		cachedCover,
		remoteCover,
		coverVersion: buildRecipeCoverVersion(recipe),
		isPinned: !!String(recipe.pinnedAt || '').trim(),
		imageCount: images.length,
		sourceBadge: detectRecipeSource(recipe),
		placeholderIcon: pickRecipePlaceholderIcon(recipe),
		mealTypeLabel: mealTypeLabelMap[recipe.mealType] || '正餐',
		infoLine: buildRecipeInfoLine(recipe),
		listSummary: buildRecipeListSummary(recipe)
	}
}

function readRecentSearches() {
	try {
		const stored = uni.getStorageSync(RECENT_SEARCH_STORAGE_KEY)
		if (!Array.isArray(stored)) return []
		return stored
			.map((item) => String(item || '').trim())
			.filter(Boolean)
			.slice(0, MAX_RECENT_SEARCHES)
	} catch (error) {
		return []
	}
}

function writeRecentSearches(items = []) {
	try {
		uni.setStorageSync(RECENT_SEARCH_STORAGE_KEY, items)
	} catch (error) {
		// Ignore storage write failures and keep search usable.
	}
}

function readLastDraftLinkPrefill() {
	try {
		return String(uni.getStorageSync(LAST_DRAFT_LINK_PREFILL_STORAGE_KEY) || '').trim()
	} catch (error) {
		return ''
	}
}

function writeLastDraftLinkPrefill(value = '') {
	try {
		uni.setStorageSync(LAST_DRAFT_LINK_PREFILL_STORAGE_KEY, String(value || '').trim())
	} catch (error) {
		// Ignore storage write failures and keep prefill usable.
	}
}

export default {
	data() {
		return {
			statusMap,
			activeSection: 'library',
			activeMealType: 'main',
			activeStatus: 'all',
			searchKeyword: '',
			recentSearches: readRecentSearches(),
			lastDraftLinkPrefill: readLastDraftLinkPrefill(),
			isSearchFocused: false,
			searchBlurTimer: null,
			selectedRecipeId: '',
			showAddSheet: false,
			draftLinkPrefillSource: '',
			showInviteSheet: false,
			showInviteCodeSheet: false,
			showProfileSheet: false,
			showMealOrderDateSheet: false,
			showMealOrderSpotlightSheet: false,
			showMealOrderCartSheet: false,
			showMealOrderCheckoutSheet: false,
			isMealOrderMode: false,
			currentKitchenId: 0,
			mealOrderDate: '',
			mealOrderStore: createEmptyMealOrderStore(),
			mealOrderStoreLoadedKitchenId: 0,
			mealOrderSpotlightIndex: 0,
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
			draftLinkPreviewError: '',
			draftLinkPreviewTimer: null,
			draftLinkPreviewRequestID: 0,
			isDraftLinkPreviewing: false,
			hasDismissedProfilePrompt: false,
			cachedRecipeCoverMap: {},
			recipeCardCoverFallbackMap: {},
			recipeCardHiddenMap: {},
			recipeCoverCacheRequestID: 0,
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
	},
	onHide() {
		if (!this.isSubmittingMealOrder) {
			this.syncMealOrderDraft({ silent: true })
		}
		this.clearMealOrderDraftSyncTimer()
		this.clearDraftLinkPreviewState()
		this.clearSearchBlurTimer()
		this.recipeCoverCacheRequestID += 1
	},
	onUnload() {
		if (!this.isSubmittingMealOrder) {
			this.syncMealOrderDraft({ silent: true })
		}
		this.clearMealOrderDraftSyncTimer()
		this.clearDraftLinkPreviewState()
		this.clearSearchBlurTimer()
		this.recipeCoverCacheRequestID += 1
	},
	onShareAppMessage(res) {
		if (res?.from === 'button' && this.activeInvite?.sharePath) {
			return {
				title: `${this.currentKitchenName || '这间厨房'} 邀请你一起维护菜单`,
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
			return this.currentKitchenName || (this.isSyncing ? '正在获取厨房信息' : this.syncErrorMessage || '暂时无法连接厨房')
		},
		currentKitchenRoleLabel() {
			if (this.currentKitchenRole === 'owner') return '创建者'
			if (this.currentKitchenRole === 'admin') return '管理员'
			if (this.currentKitchenRole === 'member') return '成员'
			return ''
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
					dateText: formatMealOrderDateText(option.value)
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
					mealTypeLabel
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
		mealOrderSpotlightTitle() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return '还没有安排菜单'
			return formatMealOrderDateText(record.planDate)
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
		mealOrderSpotlightDetailEyebrow() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return ''
			const prefix = record.type === 'draft' ? '草稿中' : '已安排'
			const total = this.mealOrderSpotlightRecords.length
			if (total < 2) return prefix
			return `${prefix} · ${this.mealOrderSpotlightRecordIndex + 1}/${total}`
		},
		mealOrderSpotlightDetailSubtitle() {
			const record = this.mealOrderSpotlightRecord
			if (!record) return ''
			const dishCount = Array.isArray(record.items) ? record.items.length : 0
			return `共 ${dishCount} 道菜`
		},
		mealOrderSpotlightDetailItems() {
			const record = this.mealOrderSpotlightRecord
			return (Array.isArray(record?.items) ? record.items : []).map((item) => ({
				...item,
				title: String(item?.titleSnapshot || '').trim() || '未命名菜品'
			}))
		},
		mealOrderSpotlightDetailNote() {
			return String(this.mealOrderSpotlightRecord?.note || '').trim()
		},
		mealOrderSpotlightCanResume() {
			const record = this.mealOrderSpotlightRecord
			if (!record || record.type !== 'draft') return false
			return record.planDate >= this.mealOrderDateStart
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
				return '试试换个关键词，或者点中间的加号新增一道菜。'
			}
			if (this.activeStatus === 'all') {
				return `试试切换到另一类餐别，或者点中间的加号新增一道${this.currentMealLabel}。`
			}
			return `可以先把${this.currentMealLabel}里的菜标记为${this.currentStatusLabel}，或者切换到全部看看。`
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
		profileSheetTitle() {
			return this.profileSheetMode === 'edit' ? '个人资料' : '完善资料'
		},
		profileSheetSubtitle() {
			return this.profileSheetMode === 'edit'
				? '修改头像和昵称后，厨房成员会更容易认出你。'
				: '设置头像和昵称后，厨房成员会更容易认出你。'
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
		draftTitleAssistText() {
			if (!this.draftAutoTitle) return ''
			if (this.draftTitleTouched) {
				return `已识别出原始标题，当前保留你手动填写的菜名。`
			}
			return `已从${this.draftLinkPlatformLabel}链接识别菜名，可直接保存。`
		},
		draftLinkAssistText() {
			if (this.isDraftLinkPreviewing) {
				return `正在从${this.draftLinkPlatformLabel}链接识别菜名...`
			}
			if (this.draftLinkPreviewError) {
				return this.draftLinkPreviewError
			}
			if (this.draft.link.trim()) {
				if (this.draftLinkPrefillSource === 'clipboard') {
					return '已从剪贴板填入分享内容，可直接保存或继续修改。'
				}
				return '支持直接粘贴 B 站或小红书分享链接，系统会自动帮你补标题。'
			}
			return ''
		}
	},
	methods: {
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
				this.showMealOrderSpotlightSheet = false
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
					this.showMealOrderSpotlightSheet = false
					this.activeSection = targetSection
				}
			})
		},
		applyMealOrderStore(store = createEmptyMealOrderStore()) {
			const normalizedStore = normalizeMealOrderStore(store)
			this.mealOrderStore = normalizedStore

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
			this.mealOrderStore = createEmptyMealOrderStore()
			this.mealOrderDate = ''
			this.isMealOrderMode = false
			this.showMealOrderDateSheet = false
			this.showMealOrderSpotlightSheet = false
			this.showMealOrderCartSheet = false
			this.showMealOrderCheckoutSheet = false
			this.mealOrderSpotlightIndex = 0
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
			this.showMealOrderSpotlightSheet = true
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
		},
		closeMealOrderSpotlightSheet() {
			this.showMealOrderSpotlightSheet = false
		},
		resumeMealOrderSpotlightRecord() {
			const record = this.mealOrderSpotlightRecord
			if (!record || !this.mealOrderSpotlightCanResume) return
			this.showMealOrderSpotlightSheet = false
			this.startMealOrderMode(record.planDate)
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
				title: `帮你选了：${picked.title}`,
				icon: 'none'
			})
		},
		openMealOrderDateSheet() {
			if (!getCurrentKitchenId()) {
				uni.showToast({
					title: '请先完成厨房同步',
					icon: 'none'
				})
				return
			}
			this.showMealOrderDateSheet = true
		},
		closeMealOrderDateSheet() {
			this.showMealOrderDateSheet = false
		},
		handleMealOrderDatePickerChange(event) {
			const value = normalizeMealOrderDate(event?.detail?.value || '')
			if (!value) return
			this.startMealOrderMode(value)
		},
		startMealOrderMode(planDate = '') {
			const normalizedDate = normalizeMealOrderDate(planDate)
			if (!normalizedDate) return
			this.mealOrderDate = normalizedDate
			this.activeSection = 'library'
			this.isMealOrderMode = true
			this.showMealOrderDateSheet = false
			this.showMealOrderSpotlightSheet = false
		},
		exitMealOrderMode() {
			this.syncMealOrderDraft({ silent: true })
			this.isMealOrderMode = false
			this.showMealOrderSpotlightSheet = false
			this.showMealOrderCartSheet = false
			this.showMealOrderCheckoutSheet = false
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
				uni.showToast({
					title: '菜单已提交',
					icon: 'none'
				})
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
		},
		clearSearchKeyword() {
			this.searchKeyword = ''
			this.clearSearchBlurTimer()
			this.isSearchFocused = true
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
			cachedEntries.forEach((entry) => {
				if (!entry.localPath) return
				nextCoverMap[entry.recipeId] = entry.localPath
			})
			this.cachedRecipeCoverMap = nextCoverMap

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
					recipeIds.forEach((recipeId) => {
						if (updatedCoverMap[recipeId] === localPath) return
						updatedCoverMap[recipeId] = localPath
						changed = true
					})

					if (changed) {
						this.cachedRecipeCoverMap = updatedCoverMap
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
		async tryPrefillDraftLinkFromClipboard() {
			if (!this.showAddSheet || String(this.draft.link || '').trim()) return

			const clipboardText = await this.readClipboardText()
			if (!this.showAddSheet || String(this.draft.link || '').trim()) return
			if (!clipboardText || clipboardText === this.lastDraftLinkPrefill) return

			const link = extractSupportedDraftLink(clipboardText)
			if (!link) return

			this.draft.link = clipboardText
			this.draftLinkPrefillSource = 'clipboard'
			this.lastDraftLinkPrefill = clipboardText
			writeLastDraftLinkPrefill(clipboardText)
			const guessedTitle = guessDraftTitleFromShareText(clipboardText)
			if (guessedTitle) {
				this.applyDraftAutoTitle(guessedTitle)
			}
			this.scheduleDraftLinkPreview(clipboardText)
		},
		clearDraftLinkPreviewState() {
			if (this.draftLinkPreviewTimer) {
				clearTimeout(this.draftLinkPreviewTimer)
				this.draftLinkPreviewTimer = null
			}
			this.draftLinkPreviewRequestID += 1
			this.isDraftLinkPreviewing = false
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
				return
			}

			const platform = detectDraftLinkPlatform(value)
			this.draftLinkPreviewPlatform = platform

			const guessedTitle = guessDraftTitleFromShareText(value)
			if (guessedTitle) {
				this.applyDraftAutoTitle(guessedTitle)
			}

			if (!platform) {
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
					this.draftLinkPreviewPlatform = detectDraftLinkPlatform(result?.canonicalUrl || result?.link || value) || platform

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
		resetLibraryFilters() {
			this.activeStatus = 'all'
			this.searchKeyword = ''
			this.clearSearchBlurTimer()
			this.isSearchFocused = false
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
				this.applyRecipes(getCachedRecipes())
			} catch (error) {
				uni.showToast({
					title: error?.message || '更新状态失败',
					icon: 'none'
				})
			}
		},
		async openAddSheet() {
			this.resetDraftAssistState()
			this.draft = this.createDraftFromContext()
			this.showAddSheet = true
			await this.tryPrefillDraftLinkFromClipboard()
		},
		closeAddSheet() {
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
			uni.showLoading({
				title: '保存中',
				mask: true
			})

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

	.page-content--meal-order {
		padding-bottom: 294rpx;
	}

	.page-header {
		padding: 6rpx 2rpx 0;
	}

	.page-header--meal-order {
		padding-top: 0;
	}

	.page-header--meal-order .page-header__top {
		align-items: center;
	}

	.page-header__top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
	}

	.page-header__heading {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.page-header__title-row {
		display: flex;
		align-items: center;
		gap: 10rpx;
		min-width: 0;
	}

	.page-header__title-mark {
		position: relative;
		width: 44rpx;
		height: 44rpx;
		border-radius: 14rpx;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.page-header__title-mark--library {
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.78) 0%, rgba(255, 255, 255, 0) 46%),
			linear-gradient(145deg, #f3ece3 0%, #e7dccc 100%);
		border: 1px solid rgba(122, 103, 85, 0.12);
		box-shadow:
			0 8rpx 16rpx rgba(97, 70, 47, 0.06),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
	}

	.page-header__title-mark--meal-order {
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.78) 0%, rgba(255, 255, 255, 0) 46%),
			linear-gradient(145deg, #f7e3d2 0%, #edd2ba 100%);
		border: 1px solid rgba(191, 113, 95, 0.12);
		box-shadow:
			0 8rpx 16rpx rgba(97, 70, 47, 0.08),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
	}

	.page-header__title-mark::after {
		content: '';
		position: absolute;
		right: -4rpx;
		top: -4rpx;
		width: 14rpx;
		height: 14rpx;
		border-radius: 999rpx;
		background: rgba(255, 244, 233, 0.92);
		border: 1px solid rgba(122, 103, 85, 0.12);
	}

	.page-header__title-mark--meal-order::after {
		border-color: rgba(191, 113, 95, 0.12);
	}

	.page-header__title {
		font-size: 40rpx;
		font-weight: 700;
		line-height: 1.18;
		color: #2f2923;
	}

	.page-header__summary {
		font-size: 23rpx;
		line-height: 1.5;
		color: #8d847a;
	}

	.page-header--meal-order .page-header__heading {
		gap: 0;
	}

	.page-header--meal-order .page-header__title {
		font-size: 36rpx;
		letter-spacing: 0.6rpx;
	}

	.page-header--meal-order .page-header__title-row {
		gap: 6rpx;
	}

	.page-header--meal-order .page-header__title-mark {
		width: 34rpx;
		height: 34rpx;
		border-radius: 11rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.7) 0%, rgba(255, 255, 255, 0) 46%),
			linear-gradient(145deg, rgba(247, 227, 210, 0.82) 0%, rgba(237, 210, 186, 0.72) 100%);
		border-color: rgba(191, 113, 95, 0.08);
		box-shadow:
			0 4rpx 10rpx rgba(97, 70, 47, 0.05),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
	}

	.page-header--meal-order .page-header__title-mark::after {
		right: -3rpx;
		top: -3rpx;
		width: 10rpx;
		height: 10rpx;
		background: rgba(255, 244, 233, 0.88);
		border-color: rgba(191, 113, 95, 0.1);
	}

	.page-header__mode-actions {
		padding: 0;
		background: transparent;
		border: 0;
		box-shadow: none;
		backdrop-filter: none;
	}

	.page-header__action {
		min-height: 56rpx;
		padding: 0 18rpx;
		box-sizing: border-box;
		border-radius: 999rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.74) 0%, rgba(255, 255, 255, 0) 48%),
			linear-gradient(145deg, #f1e3d5 0%, #ead8c6 100%);
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow:
			0 10rpx 18rpx rgba(56, 44, 30, 0.05),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.6);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8rpx;
		flex-shrink: 0;
	}

	.page-header__action-text {
		font-size: 23rpx;
		font-weight: 700;
		line-height: 1;
		color: #5f4736;
	}

	.meal-order-spotlight {
		margin-top: 12rpx;
		padding: 18rpx 20rpx;
		border-radius: 22rpx;
		background:
			radial-gradient(circle at top right, rgba(255, 233, 205, 0.5) 0%, rgba(255, 233, 205, 0) 34%),
			rgba(255, 250, 244, 0.94);
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow:
			0 10rpx 22rpx rgba(56, 44, 30, 0.05),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.68);
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18rpx;
	}

	.meal-order-spotlight--empty {
		background: rgba(255, 252, 247, 0.9);
	}

	.meal-order-spotlight__main {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.meal-order-spotlight__title {
		font-size: 28rpx;
		font-weight: 700;
		line-height: 1.28;
		color: #2f2923;
	}

	.meal-order-spotlight__desc {
		font-size: 22rpx;
		line-height: 1.5;
		color: #7d6f63;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.meal-order-spotlight__aside {
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
		flex-shrink: 0;
	}

	.meal-order-spotlight__meta-text {
		font-size: 19rpx;
		font-weight: 700;
		line-height: 1;
		color: #9a8b7c;
	}

	.meal-order-mode-bar__actions {
		display: inline-flex;
		align-items: center;
		gap: 8rpx;
		flex-shrink: 0;
	}

	.meal-order-mode-bar__chip {
		min-height: 42rpx;
		padding: 0 14rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.88);
		border: 1px solid rgba(91, 74, 59, 0.07);
		box-shadow:
			0 6rpx 12rpx rgba(56, 44, 30, 0.03),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.82);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 6rpx;
	}

	.meal-order-mode-bar__chip--accent {
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.74) 0%, rgba(255, 255, 255, 0) 42%),
			linear-gradient(145deg, rgba(247, 227, 210, 0.84) 0%, rgba(237, 210, 186, 0.76) 100%);
		border-color: rgba(191, 113, 95, 0.08);
		box-shadow:
			0 8rpx 14rpx rgba(97, 70, 47, 0.05),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.76);
	}

	.meal-order-mode-bar__chip--ghost {
		background: rgba(255, 255, 255, 0.72);
		border-color: rgba(91, 74, 59, 0.05);
		padding-left: 12rpx;
		padding-right: 12rpx;
	}

	.meal-order-mode-bar__chip-text {
		font-size: 17rpx;
		font-weight: 600;
		color: #7a6655;
		line-height: 1;
	}

	.meal-order-mode-bar__chip--accent .meal-order-mode-bar__chip-text {
		color: #765948;
	}

	.meal-order-mode-bar__chip--ghost .meal-order-mode-bar__chip-text {
		color: #948476;
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

	.member-card--interactive {
		transition: transform 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease;
	}

	.member-card--hover {
		transform: translateY(1rpx);
		box-shadow: 0 8rpx 18rpx rgba(56, 44, 30, 0.06);
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
		flex: 1;
		min-width: 0;
		font-size: 25rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.member-card__meta {
		flex: 1;
		min-width: 0;
		font-size: 22rpx;
		color: #85796e;
	}

	.member-card__meta-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.member-card__action {
		display: inline-flex;
		align-items: center;
		gap: 6rpx;
		flex-shrink: 0;
	}

	.member-card__action-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #6e5f50;
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
		margin-top: 16rpx;
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.filter-group--compact {
		margin-top: 12rpx;
	}

	.meal-tabs {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 6rpx;
		padding: 6rpx;
		border-radius: 20rpx;
		background: #f7f3ee;
		border: 1px solid rgba(91, 74, 59, 0.04);
	}

	.meal-tab {
		display: flex;
		align-items: center;
		justify-content: space-between;
		min-height: 84rpx;
		padding: 0 18rpx;
		box-sizing: border-box;
		border-radius: 16rpx;
		background: rgba(255, 255, 255, 0.24);
		border: 1px solid transparent;
	}

	.meal-tab--active {
		background: #eadfd2;
		border: 1px solid rgba(91, 74, 59, 0.12);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.22);
	}

	.meal-tab__left {
		display: flex;
		align-items: center;
		gap: 10rpx;
		min-width: 0;
	}

	.meal-tab__icon-shell {
		width: 34rpx;
		height: 34rpx;
		border-radius: 999rpx;
		background: rgba(91, 74, 59, 0.05);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.meal-tab__text {
		font-size: 25rpx;
		font-weight: 700;
		color: #81756a;
	}

	.meal-tab__count {
		min-height: 36rpx;
		padding: 0 12rpx;
		border-radius: 999rpx;
		background: rgba(91, 74, 59, 0.04);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.meal-tab__count-text {
		font-size: 18rpx;
		font-weight: 600;
		line-height: 1;
		color: #998d82;
	}

	.meal-tab--active .meal-tab__icon-shell {
		background: rgba(91, 74, 59, 0.12);
	}

	.meal-tab--active .meal-tab__text {
		color: #3f342a;
	}

	.meal-tab--active .meal-tab__count {
		background: rgba(91, 74, 59, 0.14);
	}

	.meal-tab--active .meal-tab__count-text {
		color: #5f5144;
	}

	.search-box {
		flex: 1;
		min-width: 0;
		height: 68rpx;
		display: flex;
		align-items: center;
		gap: 10rpx;
		padding: 0 18rpx;
		border-radius: 18rpx;
		background: #fcfbf8;
		border: 1px solid rgba(91, 74, 59, 0.07);
		transition: all 0.2s ease;
	}

	.search-box--active {
		background: #ffffff;
		border-color: rgba(91, 74, 59, 0.16);
		box-shadow: 0 12rpx 20rpx rgba(56, 44, 30, 0.05);
	}

	.search-box__input {
		flex: 1;
		height: 68rpx;
		font-size: 25rpx;
		color: #2f2923;
	}

	.search-box__placeholder {
		color: #b0a59a;
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
	}

	.page-content--meal-order .search-box {
		height: 72rpx;
		padding: 0 22rpx;
		border-radius: 22rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.84) 0%, rgba(255, 255, 255, 0) 44%),
			rgba(255, 255, 255, 0.98);
		border-color: rgba(91, 74, 59, 0.07);
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
	}

	.search-assist__chip-text {
		font-size: 22rpx;
		line-height: 1;
		color: #6e6155;
	}

	.status-track {
		display: flex;
		gap: 10rpx;
	}

	.status-pill {
		flex: 1;
		min-width: 0;
		min-height: 68rpx;
		padding: 0 16rpx;
		box-sizing: border-box;
		border-radius: 18rpx;
		background: #f6f2ec;
		border: 1px solid rgba(91, 74, 59, 0.05);
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.status-pill--active {
		background: #5b4a3b;
		border-color: #5b4a3b;
		box-shadow: 0 10rpx 20rpx rgba(56, 44, 30, 0.12);
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
		color: #fffaf3;
	}

	.list-caption {
		margin-top: 16rpx;
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

	.list-caption__clear-text {
		font-size: 21rpx;
		font-weight: 600;
		line-height: 1;
		color: #8a7b6e;
	}

	.list-caption__pick {
		min-height: 48rpx;
		padding: 0 18rpx;
		border-radius: 999rpx;
		background: #f2ebe3;
		border: 1px solid rgba(91, 74, 59, 0.06);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.list-caption__pick-text {
		font-size: 21rpx;
		font-weight: 600;
		line-height: 1;
		color: #6f6154;
	}

	.recipe-list {
		margin-top: 14rpx;
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.recipe-card {
		display: flex;
		align-items: stretch;
		gap: 18rpx;
		padding: 16rpx;
		border-radius: 26rpx;
		background: rgba(255, 253, 249, 0.96);
		border: 1px solid rgba(100, 78, 58, 0.05);
		box-shadow: 0 12rpx 24rpx rgba(70, 54, 40, 0.045);
		transform: scale(1);
		transition: transform 0.16s ease, border-color 0.16s ease, box-shadow 0.16s ease;
	}

	.recipe-card:active {
		transform: scale(0.992);
	}

	.recipe-card--active {
		border-color: rgba(91, 74, 59, 0.16);
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.06);
	}

	.recipe-card--pinned {
		border-color: rgba(186, 145, 81, 0.16);
		box-shadow: 0 10rpx 24rpx rgba(111, 85, 45, 0.08);
	}

	.recipe-card__media {
		position: relative;
		width: 128rpx;
		height: 128rpx;
		border-radius: 20rpx;
		overflow: hidden;
		flex-shrink: 0;
		background: linear-gradient(145deg, #ebdfd3 0%, #e1d3c4 100%);
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.26);
	}

	.recipe-card__media--empty {
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.52) 0%, rgba(255, 255, 255, 0) 42%),
			linear-gradient(135deg, #f0e6db 0%, #dfcfbd 100%);
	}

	.recipe-card__image {
		width: 100%;
		height: 100%;
		display: block;
	}

	.recipe-card__placeholder {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8rpx;
	}

	.recipe-card__placeholder-icon {
		width: 58rpx;
		height: 58rpx;
		border-radius: 18rpx;
		background: rgba(255, 255, 255, 0.5);
		border: 1px solid rgba(255, 255, 255, 0.26);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.recipe-card__placeholder-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #8b725f;
	}

	.recipe-card__selected-badge {
		position: absolute;
		top: 10rpx;
		left: 10rpx;
		padding: 8rpx 12rpx;
		border-radius: 999rpx;
		background: rgba(55, 44, 35, 0.68);
		backdrop-filter: blur(10rpx);
		display: inline-flex;
		align-items: center;
		gap: 6rpx;
	}

	.recipe-card__selected-badge-text {
		font-size: 18rpx;
		font-weight: 700;
		line-height: 1;
		color: #fff9f1;
	}

	.recipe-card__source-badge {
		position: absolute;
		top: 10rpx;
		left: 10rpx;
		padding: 8rpx 12rpx;
		border-radius: 999rpx;
		background: rgba(39, 31, 25, 0.52);
		backdrop-filter: blur(10rpx);
	}

	.recipe-card__source-badge-text {
		display: block;
		font-size: 18rpx;
		font-weight: 700;
		line-height: 1;
		color: #fffdf8;
	}

	.recipe-card__count {
		position: absolute;
		right: 6rpx;
		bottom: 6rpx;
		padding: 4rpx 6rpx;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.recipe-card__count::before {
		content: '';
		position: absolute;
		inset: -8rpx -10rpx -6rpx;
		border-radius: 999rpx;
		background: radial-gradient(circle at center, rgba(24, 18, 14, 0.34), rgba(24, 18, 14, 0.14) 58%, rgba(24, 18, 14, 0) 82%);
		filter: blur(6rpx);
	}

	.recipe-card__count-text {
		position: relative;
		z-index: 1;
		font-size: 18rpx;
		font-weight: 700;
		line-height: 1;
		color: #ffffff;
	}

	.recipe-card__body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 8rpx;
	}

	.page-content--meal-order .recipe-card {
		padding: 14rpx;
		gap: 16rpx;
		border-radius: 24rpx;
	}

	.page-content--meal-order .recipe-card__media {
		width: 118rpx;
		height: 118rpx;
		border-radius: 20rpx;
	}

	.page-content--meal-order .recipe-card__body {
		gap: 8rpx;
	}

	.page-content--meal-order .recipe-card__title {
		font-size: 28rpx;
		line-height: 1.34;
		-webkit-line-clamp: 1;
	}

	.page-content--meal-order .recipe-card__top {
		align-items: center;
	}

	.recipe-card--meal-order-selected {
		border-color: rgba(103, 79, 58, 0.14);
		background:
			radial-gradient(circle at top right, rgba(255, 231, 205, 0.56) 0%, rgba(255, 231, 205, 0) 34%),
			rgba(255, 252, 247, 0.98);
		box-shadow: 0 16rpx 26rpx rgba(70, 54, 40, 0.06);
	}

	.recipe-card__top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 12rpx;
	}

	.recipe-card__title-wrap {
		flex: 1;
		min-width: 0;
	}

	.recipe-card__title-row {
		display: flex;
		align-items: flex-start;
		gap: 10rpx;
		min-width: 0;
	}

	.recipe-card__pin-badge {
		flex-shrink: 0;
		margin-top: 4rpx;
		padding: 5rpx 10rpx;
		border-radius: 999rpx;
		background: rgba(201, 157, 91, 0.13);
		border: 1px solid rgba(201, 157, 91, 0.18);
	}

	.recipe-card__pin-badge-text {
		display: block;
		font-size: 18rpx;
		font-weight: 700;
		line-height: 1;
		color: #9a7343;
	}

	.recipe-card__title {
		flex: 1;
		min-width: 0;
		display: -webkit-box;
		font-size: 30rpx;
		font-weight: 700;
		line-height: 1.34;
		color: #2f2923;
		overflow: hidden;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
	}

	.recipe-card__info {
		display: block;
		font-size: 21rpx;
		font-weight: 700;
		line-height: 1.5;
		color: #a08773;
	}

	.recipe-card__summary {
		display: -webkit-box;
		font-size: 24rpx;
		line-height: 1.56;
		color: #5f544a;
		overflow: hidden;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 1;
	}

	.recipe-card__meta-compact {
		display: inline-flex;
		align-self: flex-start;
		padding: 6rpx 12rpx;
		border-radius: 999rpx;
		background: #f3ede5;
		font-size: 20rpx;
		font-weight: 600;
		line-height: 1;
		color: #7a6a5d;
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

	.meal-order-control {
		flex-shrink: 0;
		display: flex;
		align-items: center;
	}

	.meal-order-add {
		min-width: 140rpx;
		height: 52rpx;
		padding: 0 14rpx;
		border-radius: 999rpx;
		border: 1px solid rgba(121, 95, 73, 0.18);
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.86) 0%, rgba(255, 255, 255, 0) 44%),
			linear-gradient(180deg, #fff2e5 0%, #f3e2d0 100%);
		box-shadow:
			inset 0 1rpx 0 rgba(255, 255, 255, 0.82),
			0 8rpx 16rpx rgba(63, 52, 42, 0.08);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 5rpx;
		transition: transform 0.16s ease, box-shadow 0.16s ease, background 0.16s ease, border-color 0.16s ease;
	}

	.meal-order-add:active {
		transform: scale(0.988);
	}

	.meal-order-add--active {
		border-color: rgba(91, 74, 59, 0.04);
		background:
			radial-gradient(circle at top left, rgba(255, 245, 233, 0.18) 0%, rgba(255, 245, 233, 0) 34%),
			linear-gradient(180deg, #725d4a 0%, #5b4738 100%);
		box-shadow:
			inset 0 1rpx 0 rgba(255, 255, 255, 0.14),
			0 10rpx 18rpx rgba(63, 52, 42, 0.14);
	}

	.meal-order-add__icon {
		flex-shrink: 0;
		opacity: 0.92;
	}

	.meal-order-add__text {
		font-size: 20rpx;
		font-weight: 600;
		line-height: 1;
		color: #5b4a3b;
	}

	.meal-order-add__text--active {
		color: #fffaf3;
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

	.meal-order-floating {
		position: fixed;
		left: 24rpx;
		right: 24rpx;
		bottom: calc(env(safe-area-inset-bottom) + 128rpx);
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
	}

	.meal-order-floating__action--disabled {
		background: rgba(191, 180, 168, 0.84);
		border-color: rgba(255, 255, 255, 0.08);
		pointer-events: none;
		box-shadow: none;
	}

	.meal-order-floating__action-text {
		font-size: 23rpx;
		font-weight: 700;
		line-height: 1;
		color: #4b3728;
	}

	.meal-order-sheet {
		padding: 28rpx 24rpx calc(env(safe-area-inset-bottom) + 24rpx);
		background:
			radial-gradient(circle at top right, rgba(255, 236, 214, 0.7) 0%, rgba(255, 236, 214, 0) 32%),
			#f8f4ee;
	}

	.meal-order-sheet__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 18rpx;
	}

	.meal-order-sheet__heading {
		flex: 1;
		min-width: 0;
	}

	.meal-order-sheet__eyebrow {
		display: block;
		margin-bottom: 8rpx;
		font-size: 21rpx;
		font-weight: 700;
		line-height: 1.2;
		color: #9b826d;
		letter-spacing: 0.4rpx;
	}

	.meal-order-sheet__title {
		display: block;
		font-size: 36rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.meal-order-sheet__subtitle {
		display: block;
		margin-top: 10rpx;
		font-size: 24rpx;
		line-height: 1.6;
		color: #8a7d70;
	}

	.meal-order-sheet__close {
		width: 56rpx;
		height: 56rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.86);
		box-shadow: 0 6rpx 12rpx rgba(56, 44, 30, 0.05);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.meal-order-date-grid {
		margin-top: 22rpx;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 10rpx;
	}

	.meal-order-date-card {
		padding: 18rpx 12rpx;
		border-radius: 20rpx;
		background: rgba(255, 255, 255, 0.96);
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow: 0 10rpx 18rpx rgba(56, 44, 30, 0.04);
		display: flex;
		flex-direction: column;
		gap: 8rpx;
		text-align: center;
	}

	.meal-order-date-card--active {
		background: linear-gradient(180deg, #6d5441 0%, #584233 100%);
		border-color: rgba(109, 84, 65, 0.5);
		box-shadow: 0 14rpx 24rpx rgba(74, 56, 42, 0.16);
	}

	.meal-order-date-card__label {
		font-size: 24rpx;
		font-weight: 700;
		color: #4c3e31;
	}

	.meal-order-date-card__date {
		font-size: 21rpx;
		color: #7d6f63;
	}

	.meal-order-date-card--active .meal-order-date-card__label,
	.meal-order-date-card--active .meal-order-date-card__date {
		color: #fff7ef;
	}

	.meal-order-date-picker {
		margin-top: 14rpx;
		height: 88rpx;
		padding: 0 22rpx;
		border-radius: 22rpx;
		background: rgba(255, 255, 255, 0.96);
		border: 1px dashed rgba(91, 74, 59, 0.18);
		box-shadow: 0 8rpx 16rpx rgba(56, 44, 30, 0.04);
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.meal-order-date-picker__text {
		font-size: 25rpx;
		font-weight: 600;
		color: #5b4a3b;
	}

	.meal-order-cart-list {
		max-height: 46vh;
		margin-top: 20rpx;
	}

	.meal-order-cart-stack,
	.meal-order-checkout-list {
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.meal-order-cart-item,
	.meal-order-checkout-item {
		padding: 16rpx;
		border-radius: 18rpx;
		background: rgba(255, 255, 255, 0.96);
		border: 1px solid rgba(91, 74, 59, 0.06);
		box-shadow: 0 10rpx 18rpx rgba(56, 44, 30, 0.04);
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.meal-order-cart-item__main {
		flex: 1;
		min-width: 0;
	}

	.meal-order-cart-item__title,
	.meal-order-checkout-item__title {
		display: block;
		font-size: 25rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.meal-order-cart-item__action {
		flex-shrink: 0;
		min-width: 88rpx;
		height: 50rpx;
		padding: 0 16rpx;
		border-radius: 999rpx;
		background: #f4eee7;
		border: 1px solid rgba(91, 74, 59, 0.06);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.meal-order-cart-item__action-text {
		font-size: 20rpx;
		font-weight: 600;
		line-height: 1;
		color: #7d6857;
	}

	.meal-order-cart-empty {
		margin-top: 0;
	}

	.meal-order-note {
		margin-top: 14rpx;
	}

	.meal-order-note__label {
		display: block;
		font-size: 23rpx;
		font-weight: 600;
		color: #5f5144;
	}

	.meal-order-note__input {
		margin-top: 10rpx;
		width: 100%;
		min-height: 128rpx;
		padding: 18rpx;
		box-sizing: border-box;
		border-radius: 20rpx;
		background: rgba(255, 255, 255, 0.94);
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.75);
		font-size: 24rpx;
		line-height: 1.5;
		color: #2f2923;
	}

	.meal-order-note__placeholder {
		color: #b0a59a;
	}

	.meal-order-checkout-note {
		margin-top: 12rpx;
		padding: 16rpx;
		border-radius: 16rpx;
		background: rgba(255, 255, 255, 0.9);
		border: 1px solid rgba(91, 74, 59, 0.06);
	}

	.meal-order-checkout-note__label {
		display: block;
		font-size: 22rpx;
		font-weight: 600;
		color: #6e5f50;
	}

	.meal-order-checkout-note__text {
		display: block;
		margin-top: 8rpx;
		font-size: 23rpx;
		line-height: 1.6;
		color: #4f443a;
	}

	.meal-order-sheet__footer {
		margin-top: 18rpx;
		display: flex;
		gap: 12rpx;
	}

	.bottom-nav {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 9;
		padding: 14rpx 24rpx calc(env(safe-area-inset-bottom) + 14rpx);
		background:
			linear-gradient(180deg, rgba(246, 244, 241, 0) 0%, rgba(246, 244, 241, 0.82) 18%, rgba(255, 255, 255, 0.98) 34%),
			rgba(255, 255, 255, 0.92);
		border-top: 1px solid rgba(91, 74, 59, 0.04);
		box-shadow: 0 -8rpx 24rpx rgba(56, 44, 30, 0.03);
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
	}

	.bottom-nav--meal-order .nav-center {
		transform: translateY(-8rpx);
	}

	.bottom-nav--meal-order .nav-fab {
		width: 98rpx;
		height: 98rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 248, 237, 0.2) 0%, rgba(255, 248, 237, 0) 34%),
			linear-gradient(180deg, #6b594b 0%, #5a4739 100%);
		box-shadow:
			0 14rpx 22rpx rgba(91, 74, 59, 0.12),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.14);
	}

	.bottom-nav--meal-order .nav-item__icon-shell {
		box-shadow: 0 8rpx 16rpx rgba(56, 44, 30, 0.04);
	}

	.nav-item,
	.nav-center {
		width: 30%;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8rpx;
	}

	.nav-item__icon-shell {
		width: 82rpx;
		height: 82rpx;
		border-radius: 26rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.82) 0%, rgba(255, 255, 255, 0) 46%),
			linear-gradient(145deg, rgba(255, 255, 255, 0.98) 0%, rgba(245, 240, 234, 0.96) 100%);
		border: 1px solid rgba(91, 74, 59, 0.06);
		box-shadow:
			0 10rpx 18rpx rgba(56, 44, 30, 0.045),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.86);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease, background 0.18s ease;
	}

	.nav-item__label,
	.nav-center__label {
		font-size: 22rpx;
		line-height: 1;
		color: #978b80;
		font-weight: 600;
		transition: color 0.18s ease;
	}

	.nav-item--active .nav-item__icon-shell {
		transform: translateY(-2rpx);
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.8) 0%, rgba(255, 255, 255, 0) 44%),
			linear-gradient(145deg, #f3ece3 0%, #e8dbc9 100%);
		border-color: rgba(122, 103, 85, 0.12);
		box-shadow:
			0 14rpx 22rpx rgba(56, 44, 30, 0.08),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.84);
	}

	.nav-item--active .nav-item__label {
		color: #5b4a3b;
		font-weight: 700;
	}

	.nav-center {
		transform: translateY(-16rpx);
	}

	.nav-fab {
		width: 108rpx;
		height: 108rpx;
		border-radius: 999rpx;
		border: 10rpx solid rgba(255, 255, 255, 0.98);
		background:
			radial-gradient(circle at top left, rgba(255, 248, 237, 0.22) 0%, rgba(255, 248, 237, 0) 34%),
			linear-gradient(180deg, #6a5849 0%, #534133 100%);
		box-shadow:
			0 18rpx 28rpx rgba(91, 74, 59, 0.16),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.14);
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
