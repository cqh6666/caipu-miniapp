<template>
	<view class="detail-page">
		<template v-if="recipe">
			<scroll-view class="detail-scroll" scroll-y>
				<view
					class="hero-card"
					:class="{ 'hero-card--empty': !recipe.image }"
					@tap="handleHeroCardTap"
				>
					<image v-if="recipe.image" class="hero-card__image" :src="recipe.image" mode="aspectFill"></image>
					<view v-if="recipe.image" class="hero-card__preview-tip">
						<up-icon name="photo" size="14" color="#ffffff"></up-icon>
						<text class="hero-card__preview-tip-text">查看大图</text>
					</view>
					<view v-else class="hero-card__placeholder">
						<view class="hero-card__placeholder-mask"></view>
						<view class="hero-card__upload-action" :class="{ 'hero-card__upload-action--loading': isUploadingHeroImage }">
							<up-icon :name="isUploadingHeroImage ? 'reload' : 'plus'" size="18" color="#5b4a3b"></up-icon>
							<text class="hero-card__upload-action-text">{{ isUploadingHeroImage ? '上传中...' : '上传成品图' }}</text>
						</view>
					</view>
				</view>

				<view class="detail-head">
					<text class="detail-meta">{{ mealLabel }} · {{ statusLabel }}</text>
					<text class="detail-title">{{ recipe.title }}</text>
				</view>

				<view class="detail-card">
					<view class="detail-card__header">
						<view class="detail-card__heading">
							<text class="detail-card__title">来源链接</text>
						</view>
						<view v-if="recipe.link" class="detail-card__action" @tap="copyLink">
							<text class="detail-card__action-text">复制</text>
						</view>
					</view>
					<view v-if="recipe.link" class="link-panel">
						<view class="detail-link-box">
							<text class="detail-link-text" selectable>{{ recipe.link }}</text>
						</view>
					</view>
					<view
						v-if="parseStatusMeta"
						class="detail-parse"
						:class="`detail-parse--${parseStatusMeta.tone}`"
					>
						<view class="detail-parse__body">
							<view class="detail-parse__badge">
								<text class="detail-parse__badge-text">{{ parseStatusMeta.label }}</text>
							</view>
							<text class="detail-parse__desc">{{ parseStatusDescription }}</text>
							<text v-if="parseStatusSourceLabel" class="detail-parse__meta">{{ parseStatusSourceLabel }}</text>
						</view>
						<view
							v-if="canRetryParse"
							class="detail-parse__action"
							:class="{ 'detail-parse__action--disabled': isReparseSubmitting }"
							@tap="retryAutoParse"
						>
							<text class="detail-parse__action-text">{{ isReparseSubmitting ? '重新加入中...' : '重新解析' }}</text>
						</view>
					</view>
					<text v-else class="detail-empty">暂无链接。</text>
				</view>

				<view class="detail-card">
					<view class="detail-card__header detail-card__header--stack">
						<text class="detail-card__title">做法整理</text>
					</view>

					<view class="parsed-section">
						<text class="parsed-section__title">需要的食材</text>
						<view
							v-for="(ingredient, index) in parsedIngredients"
							:key="`ingredient-${index}`"
							class="parsed-item"
						>
							<view class="parsed-item__index">
								<text class="parsed-item__index-text">{{ index + 1 }}</text>
							</view>
							<text class="parsed-item__text">{{ ingredient }}</text>
						</view>
					</view>

					<view class="parsed-section parsed-section--steps">
						<text class="parsed-section__title">制作步骤</text>
						<view
							v-for="(step, index) in parsedSteps"
							:key="`step-${index}`"
							class="step-item"
						>
							<view class="step-item__index">
								<text class="step-item__index-text">Step {{ index + 1 }}</text>
							</view>
							<text class="step-item__text">{{ step }}</text>
						</view>
					</view>
				</view>

				<view class="detail-card detail-card--note">
					<view class="detail-card__header detail-card__header--stack">
						<text class="detail-card__title">备注</text>
					</view>
					<text v-if="recipe.note" class="detail-note">{{ recipe.note }}</text>
					<text v-else class="detail-empty">暂无备注。</text>
				</view>
			</scroll-view>

			<view class="detail-footer">
				<view class="detail-footer__action detail-footer__action--ghost" @tap="confirmDeleteRecipe">
					<text class="detail-footer__text detail-footer__text--danger">删除</text>
				</view>
				<view class="detail-footer__action detail-footer__action--primary" @tap="openEditSheet">
					<text class="detail-footer__text detail-footer__text--primary">编辑</text>
				</view>
			</view>
		</template>

		<template v-else>
			<view class="missing-state">
				<up-icon name="info-circle" size="42" color="#b8aa9b"></up-icon>
				<text class="missing-state__title">没找到这道菜</text>
				<text class="missing-state__desc">可能已删除或未保存。</text>
				<view class="missing-state__action" @tap="goBack">
					<text class="missing-state__action-text">返回列表</text>
				</view>
			</view>
		</template>

		<up-popup
			:show="showEditSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:safeAreaInsetBottom="false"
			@close="closeEditSheet"
		>
			<view class="editor-sheet">
				<view class="editor-sheet__header">
					<view class="editor-sheet__heading">
						<text class="editor-sheet__title">编辑菜品</text>
						<text class="editor-sheet__subtitle">把这道菜补充完整。</text>
					</view>
					<view class="editor-sheet__close" @tap="closeEditSheet">
						<up-icon name="close" size="18" color="#8a7d70"></up-icon>
					</view>
				</view>

				<scroll-view class="editor-sheet__body" scroll-y>
					<view class="editor-field">
						<text class="editor-field__label">菜名</text>
						<input
							v-model="editDraft.title"
							class="editor-input editor-input--title"
							placeholder="输入菜名"
							placeholder-class="editor-input__placeholder"
							maxlength="40"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">主要食材</text>
						<input
							v-model="editDraft.ingredient"
							class="editor-input"
							placeholder="例如：牛肉"
							placeholder-class="editor-input__placeholder"
							maxlength="60"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">链接</text>
						<input
							v-model="editDraft.link"
							class="editor-input"
							placeholder="粘贴菜谱或视频链接"
							placeholder-class="editor-input__placeholder"
							maxlength="300"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">成品图</text>
						<view
							class="editor-upload"
							:class="{ 'editor-upload--filled': !!editDraft.image }"
							@tap="chooseEditImage"
						>
							<template v-if="editDraft.image">
								<image class="editor-upload__thumb" :src="editDraft.image" mode="aspectFill"></image>
								<view class="editor-upload__content">
									<text class="editor-upload__title">已上传成品图</text>
									<view class="editor-upload__actions">
										<view class="editor-upload__action" @tap.stop="chooseEditImage">
											<text class="editor-upload__action-text">替换</text>
										</view>
										<view class="editor-upload__action editor-upload__action--danger" @tap.stop="removeEditImage">
											<text class="editor-upload__action-text editor-upload__action-text--danger">删除</text>
										</view>
									</view>
								</view>
							</template>
							<template v-else>
								<view class="editor-upload__empty">
									<view class="editor-upload__plus">
										<up-icon name="plus" size="20" color="#8c8074"></up-icon>
									</view>
									<text class="editor-upload__empty-text">上传成品图</text>
								</view>
							</template>
						</view>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">分类</text>
						<view class="segment">
							<view
								v-for="tab in mealTabs"
								:key="tab.value"
								class="segment__item"
								:class="{ 'segment__item--active': editDraft.mealType === tab.value }"
								@tap="editDraft.mealType = tab.value"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">状态</text>
						<view class="segment">
							<view
								v-for="tab in statusTabs"
								:key="tab.value"
								class="segment__item"
								:class="{
									'segment__item--active': editDraft.status === tab.value,
									'segment__item--wishlist': editDraft.status === tab.value && tab.value === 'wishlist',
									'segment__item--done': editDraft.status === tab.value && tab.value === 'done'
								}"
								@tap="editDraft.status = tab.value"
							>
								<text class="segment__text">{{ tab.label }}</text>
							</view>
						</view>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">食材清单</text>
						<textarea
							v-model="editDraft.ingredientsText"
							class="editor-textarea"
							placeholder="一行一个食材"
							placeholder-class="editor-textarea__placeholder"
							maxlength="500"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">制作步骤</text>
						<textarea
							v-model="editDraft.stepsText"
							class="editor-textarea editor-textarea--large"
							placeholder="一行一步"
							placeholder-class="editor-textarea__placeholder"
							maxlength="800"
						/>
					</view>

					<view class="editor-field">
						<text class="editor-field__label">备注</text>
						<textarea
							v-model="editDraft.note"
							class="editor-textarea"
							placeholder="口味、火候或视频亮点"
							placeholder-class="editor-textarea__placeholder"
							maxlength="300"
						/>
					</view>
				</scroll-view>

				<view class="editor-sheet__footer">
					<view class="editor-sheet__action" @tap="closeEditSheet">
						<text class="editor-sheet__action-text">取消</text>
					</view>
					<view
						class="editor-sheet__action editor-sheet__action--primary"
						:class="{ 'editor-sheet__action--disabled': !canSaveEditDraft }"
						@tap="saveEditDraft"
					>
						<text class="editor-sheet__action-text editor-sheet__action-text--primary">保存</text>
					</view>
				</view>
			</view>
		</up-popup>
	</view>
</template>

<script>
import {
	deleteRecipeById,
	getCachedRecipeById,
	getRecipeById,
	mealTypeLabelMap,
	mealTypeOptions,
	reparseRecipeById,
	statusLabelMap,
	statusOptions,
	updateRecipeById
} from '../../utils/recipe-store'

const createEmptyDraft = (overrides = {}) => ({
	title: '',
	ingredient: '',
	link: '',
	image: '',
	mealType: 'breakfast',
	status: 'wishlist',
	ingredientsText: '',
	stepsText: '',
	note: '',
	...overrides
})

const listToText = (items = []) => items.join('\n')
const textToList = (text = '') =>
	text
		.split('\n')
		.map((item) => item.trim())
		.filter(Boolean)

const ACTIVE_PARSE_STATUSES = ['pending', 'processing']
const parseStatusMetaMap = {
	pending: {
		label: '等待解析',
		tone: 'pending',
		description: '已加入后台队列，稍后会自动整理食材和步骤。'
	},
	processing: {
		label: '解析中',
		tone: 'processing',
		description: '后台正在解析链接内容，结果会自动更新。'
	},
	done: {
		label: '已自动整理',
		tone: 'done',
		description: '食材和步骤已由后台自动补齐。'
	},
	failed: {
		label: '解析失败',
		tone: 'failed',
		description: '这次自动整理没成功，可以重新发起一次解析。'
	}
}

function isAutoParseSupportedLink(link = '') {
	return /(bilibili\.com|b23\.tv|bili2233\.cn|xiaohongshu\.com|xhslink\.com)/i.test(String(link).trim())
}

function formatParseSourceLabel(source = '') {
	const value = String(source).trim()
	if (!value) return ''
	if (value === 'bilibili') return '来源：B 站链接自动解析'
	if (value === 'bilibili:ai') return '来源：B 站内容 + AI 总结'
	if (value === 'bilibili:heuristic') return '来源：B 站简介规则整理'
	if (value === 'xiaohongshu') return '来源：小红书链接自动解析'
	if (value === 'xiaohongshu:ai') return '来源：小红书图文 + AI 总结'
	if (value === 'xiaohongshu:heuristic') return '来源：小红书正文规则整理'
	return `来源：${value}`
}

export default {
	data() {
		return {
			recipeId: '',
			recipe: null,
			showEditSheet: false,
			editDraft: createEmptyDraft(),
			mealTabs: mealTypeOptions,
			statusTabs: statusOptions,
			isLoadingRecipe: false,
			isUploadingHeroImage: false,
			isSavingRecipe: false,
			isDeletingRecipe: false,
			isReparseSubmitting: false,
			parsePollingTimer: null
		}
	},
	computed: {
		mealLabel() {
			return mealTypeLabelMap[this.recipe?.mealType] || '早餐'
		},
		statusLabel() {
			return statusLabelMap[this.recipe?.status] || '想吃'
		},
		parsedIngredients() {
			return this.recipe?.parsedContent?.ingredients || []
		},
		parsedSteps() {
			return this.recipe?.parsedContent?.steps || []
		},
		parseStatusMeta() {
			const status = String(this.recipe?.parseStatus || '').trim()
			if (status && parseStatusMetaMap[status]) {
				return parseStatusMetaMap[status]
			}
			if (this.isAutoParseRecipe) {
				return parseStatusMetaMap.pending
			}
			return null
		},
		parseStatusDescription() {
			if (!this.parseStatusMeta) return ''
			const errorMessage = String(this.recipe?.parseError || '').trim()
			if ((this.recipe?.parseStatus || '') === 'failed' && errorMessage) {
				return errorMessage
			}
			return this.parseStatusMeta.description
		},
		parseStatusSourceLabel() {
			return formatParseSourceLabel(this.recipe?.parseSource || '')
		},
		isAutoParseRecipe() {
			return isAutoParseSupportedLink(this.recipe?.link || '')
		},
		canRetryParse() {
			return this.isAutoParseRecipe && this.recipe?.parseStatus === 'failed'
		},
		canSaveEditDraft() {
			return !!this.editDraft.title.trim()
		}
	},
	onLoad(options) {
		this.recipeId = options?.id || ''
	},
	onShow() {
		this.loadRecipe()
	},
	onHide() {
		this.stopParsePolling()
	},
	onUnload() {
		this.stopParsePolling()
	},
	methods: {
		async loadRecipe() {
			if (!this.recipeId) {
				this.recipe = null
				return
			}

			const cachedRecipe = getCachedRecipeById(this.recipeId)
			if (cachedRecipe) {
				this.applyRecipe(cachedRecipe)
			}

			try {
				this.isLoadingRecipe = true
				const recipe = await getRecipeById(this.recipeId, { preferCache: !cachedRecipe })
				this.applyRecipe(recipe)
			} catch (error) {
				if (!cachedRecipe) {
					this.recipe = null
					uni.showToast({
						title: error?.message || '加载失败',
						icon: 'none'
					})
				}
			} finally {
				this.isLoadingRecipe = false
			}
		},
		applyRecipe(recipe) {
			this.recipe = recipe
			if (this.recipe?.title) {
				uni.setNavigationBarTitle({
					title: this.recipe.title
				})
			}
			this.syncParsePolling()
		},
		syncParsePolling() {
			const status = String(this.recipe?.parseStatus || '').trim()
			if (!ACTIVE_PARSE_STATUSES.includes(status)) {
				this.stopParsePolling()
				return
			}

			if (this.parsePollingTimer) return

			this.parsePollingTimer = setInterval(() => {
				this.refreshParseStatus()
			}, 4000)
		},
		stopParsePolling() {
			if (!this.parsePollingTimer) return
			clearInterval(this.parsePollingTimer)
			this.parsePollingTimer = null
		},
		async refreshParseStatus() {
			if (!this.recipeId || this.isLoadingRecipe || this.isSavingRecipe || this.isDeletingRecipe || this.isReparseSubmitting) {
				return
			}

			try {
				const recipe = await getRecipeById(this.recipeId, { preferCache: false })
				this.applyRecipe(recipe)
			} catch (error) {
				// Ignore transient polling errors and keep the last known state on screen.
			}
		},
		createDraftFromRecipe(recipe = {}) {
			return createEmptyDraft({
				title: recipe.title || '',
				ingredient: recipe.ingredient || '',
				link: recipe.link || '',
				image: recipe.image || '',
				mealType: recipe.mealType || 'breakfast',
				status: recipe.status || 'wishlist',
				ingredientsText: listToText(recipe.parsedContent?.ingredients || []),
				stepsText: listToText(recipe.parsedContent?.steps || []),
				note: recipe.note || ''
			})
		},
		openEditSheet() {
			if (!this.recipe) return
			this.editDraft = this.createDraftFromRecipe(this.recipe)
			this.showEditSheet = true
		},
		handleHeroCardTap() {
			if (!this.recipe) return
			if (this.recipe.image) {
				this.previewRecipeImage()
				return
			}
			this.chooseHeroImage()
		},
		closeEditSheet() {
			this.showEditSheet = false
			this.editDraft = createEmptyDraft()
		},
		chooseHeroImage() {
			if (!this.recipe || this.isUploadingHeroImage) return

			uni.chooseImage({
				count: 1,
				sizeType: ['compressed'],
				sourceType: ['album', 'camera'],
				success: ({ tempFilePaths }) => {
					if (!tempFilePaths || !tempFilePaths.length) return
					this.saveHeroImage(tempFilePaths[0])
				}
			})
		},
		async saveHeroImage(imagePath = '') {
			if (!imagePath || !this.recipeId || this.isUploadingHeroImage) return

			this.isUploadingHeroImage = true
			uni.showLoading({
				title: '上传中',
				mask: true
			})

			try {
				const recipe = await updateRecipeById(this.recipeId, {
					image: imagePath
				})
				this.applyRecipe(recipe)
				uni.showToast({
					title: '已添加成品图',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '上传失败',
					icon: 'none'
				})
			} finally {
				this.isUploadingHeroImage = false
				uni.hideLoading()
			}
		},
		chooseEditImage() {
			uni.chooseImage({
				count: 1,
				sizeType: ['compressed'],
				sourceType: ['album', 'camera'],
				success: ({ tempFilePaths }) => {
					if (!tempFilePaths || !tempFilePaths.length) return
					this.editDraft.image = tempFilePaths[0]
				}
			})
		},
		removeEditImage() {
			this.editDraft.image = ''
		},
		async saveEditDraft() {
			if (!this.canSaveEditDraft || this.isSavingRecipe) return

			this.isSavingRecipe = true
			uni.showLoading({
				title: '保存中',
				mask: true
			})

			try {
				const recipe = await updateRecipeById(this.recipeId, {
					title: this.editDraft.title.trim(),
					ingredient: this.editDraft.ingredient.trim(),
					link: this.editDraft.link.trim(),
					image: this.editDraft.image,
					mealType: this.editDraft.mealType,
					status: this.editDraft.status,
					parsedContent: {
						ingredients: textToList(this.editDraft.ingredientsText),
						steps: textToList(this.editDraft.stepsText)
					},
					note: this.editDraft.note.trim()
				})
				this.closeEditSheet()
				this.applyRecipe(recipe)
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
				this.isSavingRecipe = false
				uni.hideLoading()
			}
		},
		async retryAutoParse() {
			if (!this.canRetryParse || this.isReparseSubmitting) return

			this.isReparseSubmitting = true
			uni.showLoading({
				title: '重新加入中',
				mask: true
			})

			try {
				const recipe = await reparseRecipeById(this.recipeId)
				this.applyRecipe(recipe)
				uni.showToast({
					title: '已重新加入解析队列',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '重新解析失败',
					icon: 'none'
				})
			} finally {
				this.isReparseSubmitting = false
				uni.hideLoading()
			}
		},
		confirmDeleteRecipe() {
			if (!this.recipe) return
			uni.showModal({
				title: '删除菜品',
				content: '删除后会从列表和详情页移除。',
				confirmColor: '#c16a51',
				success: async ({ confirm }) => {
					if (!confirm) return
					await this.deleteCurrentRecipe()
				}
			})
		},
		async deleteCurrentRecipe() {
			if (this.isDeletingRecipe) return

			this.isDeletingRecipe = true
			uni.showLoading({
				title: '删除中',
				mask: true
			})

			try {
				await deleteRecipeById(this.recipeId)
				uni.showToast({
					title: '已删除',
					icon: 'none'
				})
				setTimeout(() => {
					this.goBack()
				}, 280)
			} catch (error) {
				uni.showToast({
					title: error?.message || '删除失败',
					icon: 'none'
				})
			} finally {
				this.isDeletingRecipe = false
				uni.hideLoading()
			}
		},
		copyLink() {
			if (!this.recipe?.link) {
				uni.showToast({
					title: '暂无链接',
					icon: 'none'
				})
				return
			}
			uni.setClipboardData({
				data: this.recipe.link,
				success: () => {
					uni.showToast({
						title: '已复制链接',
						icon: 'none'
					})
				}
			})
		},
		previewRecipeImage() {
			const imageUrl = String(this.recipe?.image || '').trim()
			if (!imageUrl) return

			uni.previewImage({
				current: imageUrl,
				urls: [imageUrl]
			})
		},
		goBack() {
			if (getCurrentPages().length > 1) {
				uni.navigateBack()
				return
			}
			uni.reLaunch({
				url: '/pages/index/index'
			})
		}
	}
}
</script>

<style lang="scss" scoped>
	.detail-page {
		min-height: 100vh;
		background: #f6f4f1;
	}

	.detail-scroll {
		height: 100vh;
		box-sizing: border-box;
		padding: 24rpx 24rpx calc(env(safe-area-inset-bottom) + 188rpx);
	}

	.hero-card,
	.detail-card,
	.missing-state {
		border-radius: 28rpx;
		background: #ffffff;
		box-shadow: 0 10rpx 24rpx rgba(56, 44, 30, 0.05);
	}

	.hero-card {
		position: relative;
		overflow: hidden;
		min-height: 380rpx;
	}

	.hero-card--empty {
		min-height: 380rpx;
	}

	.hero-card__image {
		width: 100%;
		height: 380rpx;
		display: block;
	}

	.hero-card__preview-tip {
		position: absolute;
		right: 22rpx;
		bottom: 22rpx;
		padding: 10rpx 16rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.46);
		display: flex;
		align-items: center;
		gap: 8rpx;
	}

	.hero-card__preview-tip-text {
		font-size: 21rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.hero-card__placeholder {
		position: relative;
		min-height: 380rpx;
		box-sizing: border-box;
		background:
			linear-gradient(135deg, rgba(255, 255, 255, 0.22), rgba(255, 255, 255, 0.08)),
			linear-gradient(135deg, #ddd2c4 0%, #cfbfae 100%);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.hero-card__placeholder-mask {
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		left: 0;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.16), rgba(255, 255, 255, 0.04)),
			radial-gradient(circle at center, rgba(255, 255, 255, 0.2), transparent 60%);
	}

	.hero-card__upload-action {
		position: relative;
		z-index: 1;
		padding: 16rpx 28rpx;
		border-radius: 999rpx;
		border: 1px solid rgba(255, 255, 255, 0.58);
		background: rgba(255, 255, 255, 0.74);
		box-shadow: 0 8rpx 18rpx rgba(91, 74, 59, 0.08);
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
	}

	.hero-card__upload-action--loading {
		background: rgba(246, 242, 237, 0.9);
	}

	.hero-card__upload-action-text {
		font-size: 25rpx;
		font-weight: 600;
		line-height: 1;
		color: #5b4a3b;
	}

	.detail-head {
		padding: 24rpx 6rpx 8rpx;
	}

	.detail-meta {
		display: block;
		font-size: 22rpx;
		font-weight: 600;
		color: #8c8176;
	}

	.detail-title {
		display: block;
		margin-top: 18rpx;
		font-size: 40rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.detail-card {
		margin-top: 18rpx;
		padding: 26rpx;
	}

	.detail-card__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
	}

	.detail-card__heading {
		flex: 1;
		min-width: 0;
	}

	.detail-card__header--stack {
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.detail-card__title {
		font-size: 30rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.detail-card__action {
		padding: 12rpx 20rpx;
		border-radius: 999rpx;
		background: #f2ece5;
	}

	.detail-card__action-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #6d6155;
	}

	.link-panel {
		margin-top: 18rpx;
	}

	.detail-link-box {
		width: 100%;
		box-sizing: border-box;
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		background: #f8f5f1;
		border: 1px solid rgba(91, 74, 59, 0.08);
	}

	.detail-link-text {
		display: block;
		font-size: 24rpx;
		line-height: 1.7;
		color: #5e544b;
		white-space: normal;
		word-break: break-all;
	}

	.detail-note,
	.detail-empty {
		display: block;
		margin-top: 16rpx;
		font-size: 25rpx;
		line-height: 1.7;
		color: #5e544b;
	}

	.detail-empty {
		color: #9e9387;
	}

	.detail-parse {
		margin-top: 18rpx;
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18rpx;
	}

	.detail-parse--pending,
	.detail-parse--processing {
		background: #f7f1e7;
		border: 1px solid rgba(195, 150, 89, 0.16);
	}

	.detail-parse--done {
		background: #eef5ee;
		border: 1px solid rgba(111, 130, 109, 0.16);
	}

	.detail-parse--failed {
		background: #fbefec;
		border: 1px solid rgba(193, 106, 81, 0.14);
	}

	.detail-parse__body {
		flex: 1;
		min-width: 0;
	}

	.detail-parse__badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 6rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.72);
	}

	.detail-parse__badge-text {
		font-size: 20rpx;
		font-weight: 700;
		color: #6e5f50;
	}

	.detail-parse__desc,
	.detail-parse__meta {
		display: block;
		line-height: 1.6;
	}

	.detail-parse__desc {
		margin-top: 10rpx;
		font-size: 23rpx;
		color: #5e544b;
		word-break: break-all;
	}

	.detail-parse__meta {
		margin-top: 6rpx;
		font-size: 21rpx;
		color: #978b80;
	}

	.detail-parse__action {
		flex-shrink: 0;
		padding: 14rpx 18rpx;
		border-radius: 999rpx;
		background: #ffffff;
		box-shadow: 0 8rpx 16rpx rgba(91, 74, 59, 0.06);
	}

	.detail-parse__action--disabled {
		opacity: 0.6;
	}

	.detail-parse__action-text {
		font-size: 22rpx;
		font-weight: 700;
		color: #b4664c;
	}

	.parsed-section {
		margin-top: 24rpx;
	}

	.parsed-section--steps {
		margin-top: 30rpx;
	}

	.parsed-section__title {
		display: block;
		font-size: 24rpx;
		font-weight: 700;
		color: #76695d;
	}

	.parsed-item,
	.step-item {
		margin-top: 14rpx;
		display: flex;
		align-items: flex-start;
		gap: 14rpx;
	}

	.parsed-item__index {
		width: 40rpx;
		height: 40rpx;
		border-radius: 12rpx;
		background: #f1ebe4;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.parsed-item__index-text {
		font-size: 21rpx;
		font-weight: 700;
		color: #7d7064;
	}

	.parsed-item__text,
	.step-item__text {
		flex: 1;
		min-width: 0;
		font-size: 25rpx;
		line-height: 1.7;
		color: #4d433a;
	}

	.step-item__index {
		flex-shrink: 0;
		min-height: 52rpx;
		padding: 0 14rpx;
		box-sizing: border-box;
		border-radius: 999rpx;
		background: #efe8df;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.step-item__index-text {
		display: block;
		font-size: 20rpx;
		line-height: 1;
		font-weight: 700;
		color: #786b5f;
	}

	.detail-card--note {
		margin-bottom: 6rpx;
	}

	.detail-footer {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 10;
		padding: 18rpx 24rpx calc(env(safe-area-inset-bottom) + 20rpx);
		background: linear-gradient(180deg, rgba(246, 244, 241, 0), rgba(246, 244, 241, 0.92) 20%, rgba(255, 255, 255, 0.98) 42%);
		display: flex;
		gap: 16rpx;
	}

	.detail-footer__action {
		flex: 1;
		height: 88rpx;
		border-radius: 24rpx;
		background: #f1ede8;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.detail-footer__action--ghost {
		background: #f7efec;
	}

	.detail-footer__action--primary {
		background: #5b4a3b;
		box-shadow: 0 12rpx 20rpx rgba(91, 74, 59, 0.16);
	}

	.detail-footer__text {
		font-size: 28rpx;
		font-weight: 600;
		color: #675c51;
	}

	.detail-footer__text--danger {
		color: #b4664c;
	}

	.detail-footer__text--primary {
		color: #ffffff;
	}

	.editor-sheet {
		height: 78vh;
		background: #ffffff;
		display: flex;
		flex-direction: column;
	}

	.editor-sheet__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
		padding: 28rpx 28rpx 18rpx;
	}

	.editor-sheet__heading {
		flex: 1;
		min-width: 0;
	}

	.editor-sheet__title {
		font-size: 38rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.editor-sheet__subtitle {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		line-height: 1.5;
		color: #9b9186;
	}

	.editor-sheet__close {
		width: 68rpx;
		height: 68rpx;
		border-radius: 18rpx;
		background: #f4f0eb;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.editor-sheet__body {
		flex: 1;
		min-height: 0;
		padding: 0 28rpx 28rpx;
		box-sizing: border-box;
	}

	.editor-field {
		display: flex;
		flex-direction: column;
		gap: 12rpx;
		margin-top: 26rpx;
	}

	.editor-field:first-child {
		margin-top: 0;
	}

	.editor-field__label {
		font-size: 22rpx;
		font-weight: 500;
		color: #9b9186;
	}

	.editor-input,
	.editor-textarea {
		width: 100%;
		box-sizing: border-box;
		border-radius: 24rpx;
		background: #f7f4f0;
		border: 1px solid #ebe4db;
		color: #2f2923;
	}

	.editor-input {
		height: 88rpx;
		padding: 0 24rpx;
		font-size: 27rpx;
	}

	.editor-input--title {
		height: 96rpx;
		font-size: 30rpx;
		font-weight: 600;
		background: #ffffff;
		border-color: #e3dbd2;
	}

	.editor-input__placeholder,
	.editor-textarea__placeholder {
		color: #b7aea3;
	}

	.editor-textarea {
		min-height: 180rpx;
		padding: 22rpx 24rpx;
		font-size: 26rpx;
		line-height: 1.6;
	}

	.editor-textarea--large {
		min-height: 220rpx;
	}

	.editor-upload {
		min-height: 168rpx;
		padding: 20rpx;
		box-sizing: border-box;
		border-radius: 24rpx;
		border: 1px dashed #d8cec3;
		background: #faf7f3;
		display: flex;
		align-items: center;
	}

	.editor-upload--filled {
		border-style: solid;
		background: #ffffff;
	}

	.editor-upload__empty {
		width: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 14rpx;
	}

	.editor-upload__plus {
		width: 68rpx;
		height: 68rpx;
		border-radius: 20rpx;
		background: #f1ebe4;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.editor-upload__empty-text {
		font-size: 25rpx;
		font-weight: 600;
		color: #75685c;
	}

	.editor-upload__thumb {
		width: 148rpx;
		height: 148rpx;
		border-radius: 20rpx;
		background: #f1ebe4;
		flex-shrink: 0;
	}

	.editor-upload__content {
		flex: 1;
		min-width: 0;
		margin-left: 20rpx;
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.editor-upload__title {
		font-size: 28rpx;
		font-weight: 600;
		color: #2f2923;
	}

	.editor-upload__actions {
		display: flex;
		gap: 12rpx;
	}

	.editor-upload__action {
		padding: 10rpx 18rpx;
		border-radius: 999rpx;
		background: #f1ebe4;
	}

	.editor-upload__action--danger {
		background: #f8eeea;
	}

	.editor-upload__action-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #6c6156;
	}

	.editor-upload__action-text--danger {
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

	.editor-sheet__footer {
		padding: 18rpx 28rpx calc(env(safe-area-inset-bottom) + 20rpx);
		border-top: 1px solid rgba(91, 74, 59, 0.08);
		background: #ffffff;
		display: flex;
		gap: 16rpx;
	}

	.editor-sheet__action {
		flex: 1;
		height: 88rpx;
		border-radius: 24rpx;
		background: #f1ede8;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.editor-sheet__action--primary {
		background: #5b4a3b;
		box-shadow: 0 12rpx 20rpx rgba(91, 74, 59, 0.16);
	}

	.editor-sheet__action--disabled {
		background: #d9d1c8;
		box-shadow: none;
		pointer-events: none;
	}

	.editor-sheet__action-text {
		font-size: 28rpx;
		font-weight: 600;
		color: #675c51;
	}

	.editor-sheet__action-text--primary {
		color: #ffffff;
	}

	.missing-state {
		margin: 180rpx 24rpx 0;
		padding: 52rpx 32rpx;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}

	.missing-state__title {
		margin-top: 18rpx;
		font-size: 32rpx;
		font-weight: 700;
		color: #2f2923;
	}

	.missing-state__desc {
		margin-top: 12rpx;
		font-size: 24rpx;
		line-height: 1.6;
		color: #8d847a;
	}

	.missing-state__action {
		margin-top: 24rpx;
		padding: 16rpx 28rpx;
		border-radius: 999rpx;
		background: #5b4a3b;
	}

	.missing-state__action-text {
		font-size: 24rpx;
		font-weight: 600;
		color: #ffffff;
	}
</style>
