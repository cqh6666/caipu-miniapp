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
				<view class="page-header">
					<text class="page-header__title">我们的厨房</text>
					<text class="page-header__summary">按早餐和正餐分开看，哪些吃过，哪些还想吃。</text>
				</view>

				<view class="meal-panel-list">
					<view
						v-for="section in mealSections"
						:key="section.value"
						class="meal-panel"
					>
						<view class="meal-panel__header">
							<text class="meal-panel__title">{{ section.label }}</text>
							<text class="meal-panel__meta">想吃 {{ section.wishlist.length }} · 吃过 {{ section.done.length }}</text>
						</view>

						<view class="meal-panel__block">
							<view class="meal-panel__block-header">
								<up-icon name="heart-fill" size="12" color="#9a7b65"></up-icon>
								<text class="meal-panel__block-title">想吃</text>
							</view>
							<view v-if="section.wishlist.length" class="simple-list">
								<view v-for="recipe in section.wishlist" :key="recipe.id" class="simple-list__item simple-list__item--link" @tap="openRecipeDetail(recipe.id)">
									<text class="simple-list__title">{{ recipe.title }}</text>
									<text class="simple-list__meta">{{ recipeSecondaryText(recipe) }}</text>
								</view>
							</view>
							<view v-else class="soft-empty soft-empty--inline">
								<text class="soft-empty__text">这一类暂时没有想吃的菜。</text>
							</view>
						</view>

						<view class="meal-panel__block">
							<view class="meal-panel__block-header">
								<up-icon name="checkmark-circle-fill" size="12" color="#6f826d"></up-icon>
								<text class="meal-panel__block-title">吃过</text>
							</view>
							<view v-if="section.done.length" class="simple-list">
								<view v-for="recipe in section.done" :key="recipe.id" class="simple-list__item simple-list__item--link" @tap="openRecipeDetail(recipe.id)">
									<text class="simple-list__title">{{ recipe.title }}</text>
									<text class="simple-list__meta">{{ recipeSecondaryText(recipe) }}</text>
								</view>
							</view>
							<view v-else class="soft-empty soft-empty--inline">
								<text class="soft-empty__text">这一类还没有吃过的记录。</text>
							</view>
						</view>
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
						<view
							class="upload-card"
							:class="{ 'upload-card--filled': !!draft.image }"
							@tap="chooseDraftImage"
						>
							<template v-if="draft.image">
								<image class="upload-card__thumb" :src="draft.image" mode="aspectFill"></image>
								<view class="upload-card__content">
									<text class="upload-card__title">已上传成品图</text>
									<text class="upload-card__desc">点击卡片可替换，也可以直接删除。</text>
									<view class="upload-card__actions">
										<view class="upload-card__action" @tap.stop="chooseDraftImage">
											<text class="upload-card__action-text">替换</text>
										</view>
										<view class="upload-card__action upload-card__action--danger" @tap.stop="removeDraftImage">
											<text class="upload-card__action-text upload-card__action-text--danger">删除</text>
										</view>
									</view>
								</view>
							</template>
							<template v-else>
								<view class="upload-card__empty">
									<view class="upload-card__plus">
										<up-icon name="plus" size="20" color="#8c8074"></up-icon>
									</view>
									<text class="upload-card__empty-title">上传成品图</text>
								</view>
							</template>
						</view>
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
import {
	createRecipeFromDraft,
	getRecipeSecondaryText,
	loadRecipes,
	mealTypeOptions,
	saveRecipes,
	statusOptions
} from '../../utils/recipe-store'

const statusMap = {
	all: { label: '全部', icon: 'list-dot' },
	wishlist: { label: '想吃', icon: 'heart-fill' },
	done: { label: '吃过', icon: 'checkmark-circle-fill' }
}

const createEmptyDraft = (overrides = {}) => ({
	title: '',
	link: '',
	image: '',
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
			mealTabs: mealTypeOptions,
			statusTabs: [
				{ label: '全部', value: 'all' },
				{ label: '想吃', value: 'wishlist' },
				{ label: '吃过', value: 'done' }
			],
			draftStatusOptions: statusOptions,
			draft: createEmptyDraft(),
			recipes: []
		}
	},
	onShow() {
		this.refreshRecipes()
	},
	computed: {
		currentMealLabel() {
			return this.mealTabs.find((tab) => tab.value === this.activeMealType)?.label || '早餐'
		},
		wishlistRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'wishlist')
		},
		doneRecipes() {
			return this.recipes.filter((recipe) => recipe.status === 'done')
		},
		mealSections() {
			return this.mealTabs.map((tab) => ({
				value: tab.value,
				label: tab.label,
				wishlist: this.recipes.filter((recipe) => recipe.mealType === tab.value && recipe.status === 'wishlist'),
				done: this.recipes.filter((recipe) => recipe.mealType === tab.value && recipe.status === 'done')
			}))
		},
		librarySummary() {
			return `先按早餐和正餐整理，再看想吃和吃过。`
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
		canSubmitDraft() {
			return !!this.draft.title.trim()
		}
	},
	methods: {
		refreshRecipes() {
			this.recipes = loadRecipes()
		},
		recipeSecondaryText(recipe) {
			return getRecipeSecondaryText(recipe)
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
			const nextRecipes = this.recipes.map((recipe) => {
				if (recipe.id !== recipeId) return recipe
				return {
					...recipe,
					status: recipe.status === 'done' ? 'wishlist' : 'done'
				}
			})
			this.recipes = saveRecipes(nextRecipes)
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
		chooseDraftImage() {
			uni.chooseImage({
				count: 1,
				sizeType: ['compressed'],
				sourceType: ['album', 'camera'],
				success: ({ tempFilePaths }) => {
					if (!tempFilePaths || !tempFilePaths.length) return
					this.draft.image = tempFilePaths[0]
				}
			})
		},
		removeDraftImage() {
			this.draft.image = ''
		},
		submitDraft() {
			if (!this.canSubmitDraft) return
			const newRecipe = createRecipeFromDraft(this.draft)
			this.recipes = saveRecipes([newRecipe, ...this.recipes])
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

	.upload-card {
		min-height: 168rpx;
		padding: 20rpx;
		box-sizing: border-box;
		border-radius: 24rpx;
		border: 1px dashed #d8cec3;
		background: #faf7f3;
		display: flex;
		align-items: center;
	}

	.upload-card--filled {
		border-style: solid;
		background: #ffffff;
	}

	.upload-card__empty {
		width: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 14rpx;
	}

	.upload-card__plus {
		width: 68rpx;
		height: 68rpx;
		border-radius: 20rpx;
		background: #f1ebe4;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.upload-card__empty-title {
		font-size: 25rpx;
		font-weight: 600;
		color: #75685c;
	}

	.upload-card__thumb {
		width: 148rpx;
		height: 148rpx;
		border-radius: 20rpx;
		background: #f1ebe4;
		flex-shrink: 0;
	}

	.upload-card__content {
		flex: 1;
		min-width: 0;
		margin-left: 20rpx;
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.upload-card__title {
		font-size: 28rpx;
		font-weight: 600;
		color: #2f2923;
	}

	.upload-card__desc {
		font-size: 22rpx;
		line-height: 1.5;
		color: #95897e;
	}

	.upload-card__actions {
		display: flex;
		gap: 12rpx;
		margin-top: 4rpx;
	}

	.upload-card__action {
		padding: 10rpx 18rpx;
		border-radius: 999rpx;
		background: #f1ebe4;
	}

	.upload-card__action--danger {
		background: #f8eeea;
	}

	.upload-card__action-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #6c6156;
	}

	.upload-card__action-text--danger {
		color: #b4664c;
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
		height: 88rpx;
		border-radius: 24rpx;
		background: #f1ede8;
		display: flex;
		align-items: center;
		justify-content: center;
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
