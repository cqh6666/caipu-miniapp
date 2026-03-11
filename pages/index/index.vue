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
								placeholder="搜菜名或食材"
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
						@tap="selectRecipe(recipe.id)"
					>
						<view
							class="recipe-item__marker"
							:class="'recipe-item__marker--' + recipe.status"
						></view>
						<view class="recipe-item__main">
							<view class="recipe-item__top">
								<view class="recipe-item__text">
									<text class="recipe-item__title">{{ recipe.title }}</text>
									<text class="recipe-item__meta">{{ recipe.ingredient }}<text v-if="recipe.note"> · {{ recipe.note }}</text></text>
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
								<view v-for="recipe in section.wishlist" :key="recipe.id" class="simple-list__item">
									<text class="simple-list__title">{{ recipe.title }}</text>
									<text class="simple-list__meta">{{ recipe.ingredient }}</text>
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
								<view v-for="recipe in section.done" :key="recipe.id" class="simple-list__item">
									<text class="simple-list__title">{{ recipe.title }}</text>
									<text class="simple-list__meta">{{ recipe.ingredient }}</text>
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
			round="24"
			overlayOpacity="0.28"
			:safeAreaInsetBottom="true"
			@close="closeAddSheet"
		>
			<view class="sheet">
				<view class="sheet__header">
					<text class="sheet__title">新增一道菜</text>
					<view class="sheet__close" @tap="closeAddSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<view class="sheet__body">
					<view class="form-field">
						<text class="form-field__label">菜名</text>
						<up-input
							v-model="draft.title"
							placeholder="例如：番茄炒蛋"
							border="surround"
							clearable
						></up-input>
					</view>

					<view class="form-field">
						<text class="form-field__label">食材</text>
						<up-input
							v-model="draft.ingredient"
							placeholder="例如：鸡蛋"
							border="surround"
							clearable
						></up-input>
					</view>

					<view class="form-field">
						<text class="form-field__label">分类</text>
						<view class="choice-row">
							<view
								v-for="tab in mealTabs"
								:key="tab.value"
								class="choice-chip"
								:class="{ 'choice-chip--active': draft.mealType === tab.value }"
								@tap="draft.mealType = tab.value"
							>
								<text class="choice-chip__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="form-field">
						<text class="form-field__label">状态</text>
						<view class="choice-row">
							<view
								v-for="tab in draftStatusOptions"
								:key="tab.value"
								class="choice-chip"
								:class="{ 'choice-chip--active': draft.status === tab.value }"
								@tap="draft.status = tab.value"
							>
								<text class="choice-chip__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="form-field">
						<text class="form-field__label">备注</text>
						<textarea
							v-model="draft.note"
							class="form-field__textarea"
							placeholder="可选"
							auto-height
						/>
					</view>
				</view>

				<view class="sheet__footer">
					<up-button
						type="primary"
						text="保存"
						shape="circle"
						:customStyle="sheetButtonStyle"
						@tap="submitDraft"
					></up-button>
				</view>
			</view>
		</up-popup>
	</view>
</template>

<script>
const statusMap = {
	all: { label: '全部', icon: 'list-dot' },
	wishlist: { label: '想吃', icon: 'heart-fill' },
	done: { label: '吃过', icon: 'checkmark-circle-fill' }
}

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
			mealTabs: [
				{ label: '早餐', value: 'breakfast', icon: 'clock-fill', activeColor: '#a06a3f' },
				{ label: '正餐', value: 'main', icon: 'grid-fill', activeColor: '#5b4a3b' }
			],
			statusTabs: [
				{ label: '全部', value: 'all' },
				{ label: '想吃', value: 'wishlist' },
				{ label: '吃过', value: 'done' }
			],
			draftStatusOptions: [
				{ label: '想吃', value: 'wishlist' },
				{ label: '吃过', value: 'done' }
			],
			draft: {
				title: '',
				ingredient: '',
				mealType: 'breakfast',
				status: 'wishlist',
				note: ''
			},
			recipes: [
				{ id: 'r1', title: '番茄滑蛋牛肉', ingredient: '牛肉', mealType: 'main', status: 'done', note: '下饭' },
				{ id: 'r2', title: '神仙糖醋排骨', ingredient: '排骨', mealType: 'main', status: 'wishlist', note: '周末吃' },
				{ id: 'r3', title: '蒜香虾仁意面', ingredient: '虾仁', mealType: 'main', status: 'wishlist', note: '' },
				{ id: 'r4', title: '低卡燕麦松饼', ingredient: '燕麦', mealType: 'breakfast', status: 'done', note: '简单快手' },
				{ id: 'r5', title: '周末寿喜锅', ingredient: '牛肉', mealType: 'main', status: 'wishlist', note: '' },
				{ id: 'r6', title: '牛油果煎蛋吐司', ingredient: '牛油果', mealType: 'breakfast', status: 'wishlist', note: '十分钟内' }
			]
		}
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
					recipe.ingredient.toLowerCase().includes(keyword) ||
					recipe.note.toLowerCase().includes(keyword)
				return matchedMealType && matchedStatus && matchedKeyword
			})
		},
		sheetButtonStyle() {
			return {
				width: '100%',
				height: '84rpx',
				fontWeight: '600'
			}
		}
	},
	methods: {
		mealTypeCount(type) {
			return this.recipes.filter((recipe) => recipe.mealType === type).length
		},
		selectRecipe(recipeId) {
			this.selectedRecipeId = recipeId
		},
		nextStatusText(status) {
			return status === 'done' ? '标记想吃' : '标记吃过'
		},
		toggleRecipeStatus(recipeId) {
			this.recipes = this.recipes.map((recipe) => {
				if (recipe.id !== recipeId) return recipe
				return {
					...recipe,
					status: recipe.status === 'done' ? 'wishlist' : 'done'
				}
			})
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
			this.showAddSheet = true
		},
		closeAddSheet() {
			this.showAddSheet = false
		},
		submitDraft() {
			const title = this.draft.title.trim()
			if (!title) {
				uni.showToast({
					title: '先输入菜名',
					icon: 'none'
				})
				return
			}
			const newRecipe = {
				id: `recipe-${Date.now()}`,
				title,
				ingredient: this.draft.ingredient.trim() || '未分类',
				mealType: this.draft.mealType,
				status: this.draft.status,
				note: this.draft.note.trim()
			}
			this.recipes = [newRecipe, ...this.recipes]
			this.selectedRecipeId = newRecipe.id
			this.activeSection = 'library'
			this.activeMealType = newRecipe.mealType
			this.activeStatus = 'all'
			this.searchKeyword = ''
			this.draft = {
				title: '',
				ingredient: '',
				mealType: this.activeMealType,
				status: 'wishlist',
				note: ''
			}
			this.showAddSheet = false
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
		padding: 22rpx 22rpx 32rpx;
		background: #fffdf9;
	}

	.sheet__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16rpx;
	}

	.sheet__title {
		font-size: 30rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.sheet__close {
		width: 68rpx;
		height: 68rpx;
		border-radius: 18rpx;
		background: #f3eee8;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.sheet__body {
		margin-top: 22rpx;
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}

	.form-field {
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.form-field__label {
		font-size: 24rpx;
		font-weight: 600;
		color: #6f655b;
	}

	.choice-row {
		display: flex;
		gap: 10rpx;
	}

	.choice-chip {
		padding: 12rpx 18rpx;
		border-radius: 999rpx;
		background: #efebe5;
	}

	.choice-chip--active {
		background: #2f2923;
	}

	.choice-chip__text {
		font-size: 23rpx;
		font-weight: 600;
		color: #6f655b;
	}

	.choice-chip--active .choice-chip__text {
		color: #ffffff;
	}

	.form-field__textarea {
		width: 100%;
		min-height: 120rpx;
		padding: 20rpx;
		box-sizing: border-box;
		border-radius: 20rpx;
		background: #fbfaf8;
		border: 1px solid rgba(0, 0, 0, 0.04);
		font-size: 26rpx;
		line-height: 1.6;
		color: #2f2923;
	}

	.sheet__footer {
		margin-top: 24rpx;
	}
</style>
