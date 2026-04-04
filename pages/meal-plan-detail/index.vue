<template>
	<view class="meal-plan-page">
		<template v-if="record">
			<scroll-view class="meal-plan-scroll" scroll-y>
				<view class="meal-plan-hero" :class="record.type === 'draft' ? 'meal-plan-hero--draft' : 'meal-plan-hero--submitted'">
					<view class="meal-plan-hero__chips">
						<view class="meal-plan-chip" :class="record.type === 'draft' ? 'meal-plan-chip--draft' : 'meal-plan-chip--submitted'">
							<text class="meal-plan-chip__text">{{ statusLabel }}</text>
						</view>
						<view class="meal-plan-chip meal-plan-chip--soft">
							<text class="meal-plan-chip__text">{{ dateText }}</text>
						</view>
						<view v-if="hasSiblingDraft" class="meal-plan-chip meal-plan-chip--editing">
							<text class="meal-plan-chip__text">待修改草稿</text>
						</view>
					</view>
					<text class="meal-plan-hero__title">{{ pageTitle }}</text>
					<text class="meal-plan-hero__summary">{{ dishSummary }}</text>
					<text v-if="timeMetaText" class="meal-plan-hero__meta">{{ timeMetaText }}</text>
				</view>

				<view class="meal-plan-notice" :class="noticeTone ? `meal-plan-notice--${noticeTone}` : ''">
					<up-icon class="meal-plan-notice__icon" name="info-circle" size="14" color="#b46d55"></up-icon>
					<text class="meal-plan-notice__text">{{ helperText }}</text>
				</view>

				<view class="meal-plan-card meal-plan-card--summary">
					<view class="meal-plan-card__header">
						<text class="meal-plan-card__title">这天的安排</text>
						<text class="meal-plan-card__meta">共 {{ dishCount }} 道</text>
					</view>
					<view class="meal-plan-summary-grid">
						<view class="meal-plan-summary-box">
							<text class="meal-plan-summary-box__value">{{ breakfastCount }}</text>
							<text class="meal-plan-summary-box__label">早餐</text>
						</view>
						<view class="meal-plan-summary-box">
							<text class="meal-plan-summary-box__value">{{ mainCount }}</text>
							<text class="meal-plan-summary-box__label">正餐</text>
						</view>
					</view>
				</view>

				<view
					v-for="section in mealSections"
					:key="section.key"
					class="meal-plan-card"
				>
					<view class="meal-plan-card__header">
						<text class="meal-plan-card__title">{{ section.label }}</text>
						<text class="meal-plan-card__meta">{{ section.items.length }} 道</text>
					</view>
					<view class="meal-plan-list">
						<view
							v-for="item in section.items"
							:key="`${section.key}-${item.recipeId}`"
							class="meal-plan-item"
							hover-class="meal-plan-item--hover"
							@tap="openRecipeDetail(item)"
						>
							<view class="meal-plan-item__thumb">
								<image
									v-if="item.imageSnapshot"
									class="meal-plan-item__thumb-image"
									:src="item.imageSnapshot"
									mode="aspectFill"
								></image>
								<view v-else class="meal-plan-item__thumb-placeholder">
									<up-icon name="photo" size="18" color="#b79e87"></up-icon>
								</view>
							</view>
							<view class="meal-plan-item__main">
								<text class="meal-plan-item__title">{{ item.title }}</text>
								<view class="meal-plan-item__meta-row">
									<text class="meal-plan-item__meta">{{ item.mealTypeLabel }}</text>
									<text v-if="item.quantity > 1" class="meal-plan-item__meta">x{{ item.quantity }}</text>
								</view>
							</view>
							<up-icon name="arrow-right" size="14" color="#9d8c7a"></up-icon>
						</view>
					</view>
				</view>

				<view v-if="noteText" class="meal-plan-card meal-plan-card--note">
					<view class="meal-plan-card__header">
						<text class="meal-plan-card__title">备注</text>
					</view>
					<text class="meal-plan-note">{{ noteText }}</text>
				</view>
			</scroll-view>

			<view class="meal-plan-footer">
				<view
					class="meal-plan-footer__action meal-plan-footer__action--danger"
					:class="{ 'meal-plan-footer__action--disabled': isActing }"
					@tap="handleDangerAction"
				>
					<text class="meal-plan-footer__text meal-plan-footer__text--danger">{{ dangerActionText }}</text>
				</view>
				<view
					class="meal-plan-footer__action meal-plan-footer__action--primary"
					:class="{ 'meal-plan-footer__action--disabled': isActing }"
					@tap="handlePrimaryAction"
				>
					<text class="meal-plan-footer__text meal-plan-footer__text--primary">{{ primaryActionText }}</text>
				</view>
			</view>
		</template>

		<template v-else>
			<view class="meal-plan-empty">
				<up-icon :name="isLoading ? 'reload' : 'info-circle'" size="40" color="#b4a392"></up-icon>
				<text class="meal-plan-empty__title">{{ isLoading ? '菜单加载中' : '没找到这份菜单' }}</text>
				<text class="meal-plan-empty__desc">{{ emptyDescription }}</text>
				<view class="meal-plan-empty__actions">
					<view class="meal-plan-empty__action" @tap="goBackToIndex">
						<text class="meal-plan-empty__action-text">返回首页</text>
					</view>
					<view v-if="!isLoading" class="meal-plan-empty__action meal-plan-empty__action--primary" @tap="loadPage">
						<text class="meal-plan-empty__action-text meal-plan-empty__action-text--primary">重新加载</text>
					</view>
				</view>
			</view>
		</template>
	</view>
</template>

<script>
import { ensureSession, getCurrentKitchenId } from '../../utils/auth'
import {
	createMealPlanDraftFromSubmitted,
	deleteMealPlanDraft,
	deleteSubmittedMealPlan,
	listMealPlanStore
} from '../../utils/meal-plan-api'
import { mealTypeLabelMap } from '../../utils/recipe-store'
import {
	buildMealOrderDishSummary,
	formatMealOrderDateText,
	normalizeMealOrderDate,
	normalizeMealOrderDraft,
	normalizeMealOrderRecord,
	normalizeMealOrderStore,
	writePendingMealOrderAction
} from '../index/meal-order'

const mealSectionDefs = [
	{ key: 'breakfast', label: '早餐' },
	{ key: 'main', label: '正餐' }
]

function padTimeNumber(value) {
	return String(Number(value) || 0).padStart(2, '0')
}

function formatActionTimeText(value = '') {
	const date = new Date(value)
	if (Number.isNaN(date.getTime())) return ''
	return `${date.getMonth() + 1}月${date.getDate()}日 ${padTimeNumber(date.getHours())}:${padTimeNumber(date.getMinutes())}`
}

export default {
	data() {
		return {
			planDate: '',
			planType: 'submitted',
			currentKitchenId: 0,
			record: null,
			siblingDraft: null,
			siblingSubmitted: null,
			isLoading: true,
			isActing: false,
			errorMessage: ''
		}
	},
	computed: {
		dateText() {
			return formatMealOrderDateText(this.planDate)
		},
		statusLabel() {
			return this.record?.type === 'draft' ? '草稿中' : '已安排'
		},
		pageTitle() {
			return this.record?.type === 'draft' ? '这天的小菜单草稿' : '这天的菜单安排'
		},
		dishSummary() {
			return buildMealOrderDishSummary(this.record?.items || [])
		},
		dishCount() {
			return Array.isArray(this.record?.items) ? this.record.items.length : 0
		},
		breakfastCount() {
			return (Array.isArray(this.record?.items) ? this.record.items : [])
				.filter((item) => item.mealTypeSnapshot === 'breakfast').length
		},
		mainCount() {
			return (Array.isArray(this.record?.items) ? this.record.items : [])
				.filter((item) => item.mealTypeSnapshot !== 'breakfast').length
		},
		mealSections() {
			const items = Array.isArray(this.record?.items) ? this.record.items : []
			return mealSectionDefs.map((section) => ({
				...section,
				items: items
					.filter((item) => {
						if (section.key === 'breakfast') return item.mealTypeSnapshot === 'breakfast'
						return item.mealTypeSnapshot !== 'breakfast'
					})
					.map((item) => ({
						...item,
						title: String(item.titleSnapshot || '').trim() || '未命名菜品',
						imageSnapshot: String(item.imageSnapshot || '').trim(),
						mealTypeLabel: mealTypeLabelMap[String(item.mealTypeSnapshot || '').trim()] || '正餐'
					}))
			})).filter((section) => section.items.length > 0)
		},
		noteText() {
			return String(this.record?.note || '').trim()
		},
		hasSiblingDraft() {
			return !!this.siblingDraft && this.record?.type === 'submitted'
		},
		primaryActionText() {
			if (!this.record) return ''
			if (this.record.type === 'draft') return '继续安排'
			return this.hasSiblingDraft ? '继续改草稿' : '修改菜单'
		},
		dangerActionText() {
			if (!this.record) return ''
			return this.record.type === 'draft' ? '删除草稿' : '删除安排'
		},
		helperText() {
			if (!this.record) return ''
			if (this.record.type === 'draft') {
				if (this.siblingSubmitted) {
					return '这份草稿还没重新提交，当前厨房成员看到的仍然是之前已安排好的菜单。'
				}
				return '这份草稿还没同步成正式安排，改好后重新提交，这天菜单才会更新。'
			}
			if (this.hasSiblingDraft) {
				return '这天已经有一份待提交草稿。重新提交前，当前已安排菜单仍然对厨房成员生效。'
			}
			return '这份菜单已经对厨房成员可见。点“修改菜单”时，我们会先带出一份草稿给你继续改。'
		},
		noticeTone() {
			if (!this.record) return ''
			if (this.record.type === 'draft') return 'draft'
			if (this.hasSiblingDraft) return 'editing'
			return 'submitted'
		},
		timeMetaText() {
			if (!this.record) return ''
			if (this.record.type === 'draft') {
				const text = formatActionTimeText(this.record.updatedAt)
				return text ? `最近修改于 ${text}` : '草稿会在你重新提交前一直保留'
			}
			const text = formatActionTimeText(this.record.submittedAt)
			return text ? `已在 ${text} 同步到这间厨房` : '这份安排已经同步给这间厨房'
		},
		emptyDescription() {
			if (this.isLoading) {
				return '正在整理这天的菜单记录。'
			}
			return this.errorMessage || '可能这份菜单已经被删除，或者你当前不在对应厨房里。'
		}
	},
	onLoad(options) {
		this.planDate = normalizeMealOrderDate(options?.planDate || '')
		this.planType = String(options?.type || '').trim() === 'draft' ? 'draft' : 'submitted'
	},
	onShow() {
		this.loadPage()
	},
	methods: {
		async loadPage() {
			if (!this.planDate) {
				this.record = null
				this.errorMessage = '菜单日期参数不完整'
				this.isLoading = false
				return
			}

			this.isLoading = true
			this.errorMessage = ''
			try {
				await ensureSession()
				const kitchenId = Number(getCurrentKitchenId()) || 0
				if (!kitchenId) {
					throw new Error('请先进入一个厨房')
				}
				this.currentKitchenId = kitchenId
				const store = normalizeMealOrderStore(await listMealPlanStore(kitchenId))
				this.applyStore(store)
				if (!this.record) {
					this.errorMessage = '没找到这份菜单'
				}
			} catch (error) {
				this.record = null
				this.siblingDraft = null
				this.siblingSubmitted = null
				this.errorMessage = error?.message || '加载菜单失败'
			} finally {
				this.isLoading = false
			}
		},
		applyStore(store = normalizeMealOrderStore({})) {
			const drafts = store?.drafts || {}
			const submittedList = Array.isArray(store?.submitted) ? store.submitted : []
			const draft = normalizeMealOrderDraft(drafts[this.planDate], this.planDate)
			const hasDraft = draft.planDate && draft.items.length
			const submitted = submittedList.find((item) => item.planDate === this.planDate) || null

			let resolved = null
			if (this.planType === 'draft' && hasDraft) {
				resolved = { ...draft, type: 'draft' }
			} else if (this.planType === 'submitted' && submitted) {
				resolved = { ...submitted, type: 'submitted' }
			} else if (hasDraft) {
				resolved = { ...draft, type: 'draft' }
				this.planType = 'draft'
			} else if (submitted) {
				resolved = { ...submitted, type: 'submitted' }
				this.planType = 'submitted'
			}

			this.record = resolved
			this.siblingDraft = resolved?.type === 'submitted' && hasDraft ? draft : null
			this.siblingSubmitted = resolved?.type === 'draft' && submitted ? normalizeMealOrderRecord(submitted) : null
		},
		openRecipeDetail(item = {}) {
			const recipeId = String(item.recipeId || '').trim()
			if (!recipeId) return
			uni.navigateTo({
				url: `/pages/recipe-detail/index?id=${recipeId}`
			})
		},
		handlePrimaryAction() {
			if (!this.record || this.isActing) return
			if (this.record.type === 'draft' || this.hasSiblingDraft) {
				this.resumeMealOrder(this.hasSiblingDraft ? '已继续这天草稿' : '继续安排这天菜单')
				return
			}
			this.createDraftAndResume()
		},
		async createDraftAndResume() {
			if (!this.record || !this.currentKitchenId) return
			this.isActing = true
			try {
				await createMealPlanDraftFromSubmitted(this.currentKitchenId, this.planDate)
				this.resumeMealOrder('已带出这天菜单')
			} catch (error) {
				uni.showToast({
					title: error?.message || '暂时无法修改菜单',
					icon: 'none'
				})
			} finally {
				this.isActing = false
			}
		},
		handleDangerAction() {
			if (!this.record || this.isActing) return
			const isDraft = this.record.type === 'draft'
			const title = isDraft ? '删除草稿' : '删除安排'
			const content = isDraft
				? '删除这份菜单草稿后，需要重新安排。确认删除吗？'
				: '删除这天已经安排好的菜单后，厨房成员都会看不到。确认删除吗？'

			uni.showModal({
				title,
				content,
				confirmText: title,
				success: ({ confirm }) => {
					if (!confirm) return
					this.deleteCurrentRecord(isDraft)
				}
			})
		},
		async deleteCurrentRecord(isDraft = false) {
			if (!this.currentKitchenId) return
			this.isActing = true
			try {
				if (isDraft) {
					await deleteMealPlanDraft(this.currentKitchenId, this.planDate)
				} else {
					await deleteSubmittedMealPlan(this.currentKitchenId, this.planDate)
				}
				this.navigateBackWithAction({
					kind: 'reload',
					kitchenId: this.currentKitchenId,
					message: isDraft ? '草稿已删除' : '安排已删除'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || `${isDraft ? '删除草稿' : '删除安排'}失败`,
					icon: 'none'
				})
			} finally {
				this.isActing = false
			}
		},
		resumeMealOrder(message = '') {
			this.navigateBackWithAction({
				kind: 'resume',
				kitchenId: this.currentKitchenId,
				planDate: this.planDate,
				message
			})
		},
		navigateBackWithAction(action = null) {
			writePendingMealOrderAction(action || {})
			const pages = getCurrentPages()
			if (pages.length > 1) {
				uni.navigateBack({
					delta: 1
				})
				return
			}
			uni.reLaunch({
				url: '/pages/index/index'
			})
		},
		goBackToIndex() {
			this.navigateBackWithAction({
				kind: 'reload',
				kitchenId: this.currentKitchenId
			})
		}
	}
}
</script>

<style lang="scss" scoped>
.meal-plan-page {
	min-height: 100vh;
	background:
		radial-gradient(circle at top right, rgba(255, 233, 206, 0.6) 0%, rgba(255, 233, 206, 0) 28%),
		#f6f1ea;
}

.meal-plan-scroll {
	height: calc(100vh - env(safe-area-inset-bottom) - 124rpx);
	padding: 28rpx 24rpx 36rpx;
	box-sizing: border-box;
}

.meal-plan-hero {
	padding: 26rpx 24rpx;
	border-radius: 30rpx;
	background:
		radial-gradient(circle at top left, rgba(255, 255, 255, 0.84) 0%, rgba(255, 255, 255, 0) 42%),
		linear-gradient(145deg, #f3e2d1 0%, #e7d1bb 100%);
	border: 1px solid rgba(177, 138, 101, 0.14);
	box-shadow:
		0 18rpx 34rpx rgba(76, 57, 39, 0.08),
		inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
}

.meal-plan-hero--draft {
	background:
		radial-gradient(circle at top left, rgba(255, 255, 255, 0.84) 0%, rgba(255, 255, 255, 0) 42%),
		linear-gradient(145deg, #f5e8d9 0%, #ecd8bf 100%);
}

.meal-plan-hero--submitted {
	background:
		radial-gradient(circle at top left, rgba(255, 255, 255, 0.84) 0%, rgba(255, 255, 255, 0) 42%),
		linear-gradient(145deg, #f0dcc8 0%, #e6ccb3 100%);
}

.meal-plan-hero__chips {
	display: flex;
	flex-wrap: wrap;
	gap: 10rpx;
}

.meal-plan-chip {
	min-height: 42rpx;
	padding: 0 14rpx;
	border-radius: 999rpx;
	display: inline-flex;
	align-items: center;
	justify-content: center;
}

.meal-plan-chip--draft {
	background: rgba(255, 248, 238, 0.88);
}

.meal-plan-chip--submitted {
	background: rgba(255, 244, 233, 0.9);
}

.meal-plan-chip--editing {
	background: rgba(255, 236, 227, 0.92);
}

.meal-plan-chip--soft {
	background: rgba(255, 255, 255, 0.58);
}

.meal-plan-chip__text {
	font-size: 21rpx;
	font-weight: 700;
	line-height: 1;
	color: #7a604d;
}

.meal-plan-hero__title {
	display: block;
	margin-top: 18rpx;
	font-size: 42rpx;
	font-weight: 700;
	line-height: 1.18;
	color: #2f2923;
}

.meal-plan-hero__summary {
	display: block;
	margin-top: 14rpx;
	font-size: 24rpx;
	line-height: 1.7;
	color: #6f5a48;
}

.meal-plan-hero__meta {
	display: block;
	margin-top: 14rpx;
	font-size: 22rpx;
	line-height: 1.5;
	color: #8c7461;
}

.meal-plan-notice {
	margin-top: 18rpx;
	padding: 18rpx;
	border-radius: 22rpx;
	background: rgba(255, 250, 244, 0.94);
	border: 1px solid rgba(202, 165, 133, 0.16);
	display: flex;
	align-items: flex-start;
	gap: 10rpx;
}

.meal-plan-notice--draft {
	background: rgba(255, 249, 241, 0.96);
}

.meal-plan-notice--editing {
	background: rgba(255, 241, 235, 0.96);
}

.meal-plan-notice--submitted {
	background: rgba(252, 247, 241, 0.96);
}

.meal-plan-notice__icon {
	margin-top: 2rpx;
	flex-shrink: 0;
}

.meal-plan-notice__text {
	font-size: 23rpx;
	line-height: 1.65;
	color: #765f4f;
}

.meal-plan-card {
	margin-top: 18rpx;
	padding: 20rpx 18rpx;
	border-radius: 24rpx;
	background: rgba(255, 255, 255, 0.92);
	border: 1px solid rgba(91, 74, 59, 0.06);
	box-shadow: 0 12rpx 22rpx rgba(56, 44, 30, 0.04);
}

.meal-plan-card__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12rpx;
}

.meal-plan-card__title {
	font-size: 28rpx;
	font-weight: 700;
	line-height: 1.2;
	color: #2f2923;
}

.meal-plan-card__meta {
	font-size: 21rpx;
	line-height: 1.2;
	color: #8f8173;
}

.meal-plan-summary-grid {
	margin-top: 16rpx;
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 12rpx;
}

.meal-plan-summary-box {
	padding: 18rpx 16rpx;
	border-radius: 20rpx;
	background: linear-gradient(180deg, #fbf6f0 0%, #f5ede4 100%);
	border: 1px solid rgba(91, 74, 59, 0.05);
}

.meal-plan-summary-box__value {
	display: block;
	font-size: 34rpx;
	font-weight: 700;
	line-height: 1.1;
	color: #302821;
}

.meal-plan-summary-box__label {
	display: block;
	margin-top: 8rpx;
	font-size: 21rpx;
	line-height: 1.2;
	color: #8b7b6f;
}

.meal-plan-list {
	margin-top: 16rpx;
	display: flex;
	flex-direction: column;
	gap: 12rpx;
}

.meal-plan-item {
	padding: 14rpx;
	border-radius: 20rpx;
	background: rgba(250, 247, 243, 0.96);
	border: 1px solid rgba(91, 74, 59, 0.05);
	display: flex;
	align-items: center;
	gap: 14rpx;
}

.meal-plan-item--hover {
	background: #fffaf4;
}

.meal-plan-item__thumb {
	width: 92rpx;
	height: 92rpx;
	border-radius: 20rpx;
	background: rgba(244, 235, 225, 0.94);
	border: 1px solid rgba(91, 74, 59, 0.05);
	overflow: hidden;
	flex-shrink: 0;
}

.meal-plan-item__thumb-image,
.meal-plan-item__thumb-placeholder {
	width: 100%;
	height: 100%;
}

.meal-plan-item__thumb-image {
	display: block;
}

.meal-plan-item__thumb-placeholder {
	display: flex;
	align-items: center;
	justify-content: center;
}

.meal-plan-item__main {
	flex: 1;
	min-width: 0;
}

.meal-plan-item__title {
	display: block;
	font-size: 26rpx;
	font-weight: 700;
	line-height: 1.45;
	color: #2f2923;
}

.meal-plan-item__meta-row {
	margin-top: 8rpx;
	display: flex;
	flex-wrap: wrap;
	gap: 10rpx;
}

.meal-plan-item__meta {
	font-size: 20rpx;
	line-height: 1.2;
	color: #8e7d70;
}

.meal-plan-note {
	display: block;
	margin-top: 14rpx;
	font-size: 24rpx;
	line-height: 1.75;
	color: #4f443a;
}

.meal-plan-footer {
	position: fixed;
	left: 0;
	right: 0;
	bottom: 0;
	z-index: 5;
	padding: 14rpx 24rpx calc(env(safe-area-inset-bottom) + 16rpx);
	background:
		linear-gradient(180deg, rgba(246, 241, 234, 0) 0%, rgba(246, 241, 234, 0.82) 18%, rgba(246, 241, 234, 0.98) 34%),
		rgba(246, 241, 234, 0.94);
	display: flex;
	gap: 12rpx;
}

.meal-plan-footer__action {
	flex: 1;
	height: 92rpx;
	border-radius: 26rpx;
	display: flex;
	align-items: center;
	justify-content: center;
}

.meal-plan-footer__action--danger {
	background: #fff6f2;
	border: 1px solid rgba(181, 91, 74, 0.14);
}

.meal-plan-footer__action--primary {
	background: #5b4a3b;
	box-shadow: 0 12rpx 22rpx rgba(91, 74, 59, 0.14);
}

.meal-plan-footer__action--disabled {
	opacity: 0.6;
	pointer-events: none;
}

.meal-plan-footer__text {
	font-size: 28rpx;
	font-weight: 700;
	line-height: 1;
}

.meal-plan-footer__text--danger {
	color: #b55b4a;
}

.meal-plan-footer__text--primary {
	color: #fffaf3;
}

.meal-plan-empty {
	min-height: 100vh;
	padding: 0 36rpx;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	text-align: center;
	gap: 16rpx;
}

.meal-plan-empty__title {
	font-size: 34rpx;
	font-weight: 700;
	line-height: 1.25;
	color: #3a322b;
}

.meal-plan-empty__desc {
	font-size: 24rpx;
	line-height: 1.7;
	color: #8b7f74;
}

.meal-plan-empty__actions {
	margin-top: 8rpx;
	display: flex;
	gap: 12rpx;
}

.meal-plan-empty__action {
	min-width: 180rpx;
	height: 84rpx;
	padding: 0 20rpx;
	border-radius: 24rpx;
	background: rgba(255, 255, 255, 0.92);
	border: 1px solid rgba(91, 74, 59, 0.08);
	display: inline-flex;
	align-items: center;
	justify-content: center;
}

.meal-plan-empty__action--primary {
	background: #5b4a3b;
	border-color: transparent;
}

.meal-plan-empty__action-text {
	font-size: 26rpx;
	font-weight: 700;
	line-height: 1;
	color: #6a5a4a;
}

.meal-plan-empty__action-text--primary {
	color: #fffaf3;
}
</style>
