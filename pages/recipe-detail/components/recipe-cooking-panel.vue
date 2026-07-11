<template>
<view class="detail-card detail-card--cooking">
	<view class="detail-card__header">
		<text class="detail-card__title">做法</text>
		<!-- 后台正在处理时显示状态 chip，避免 dead-click -->
		<view v-if="isCookingActive" class="detail-card__status-chip">
			<text class="detail-card__status-chip-text">{{ cookingActiveLabel }}</text>
		</view>
		<view
			v-else-if="hasCookingMenuItems && !isPublicView"
			class="detail-card__icon-action"
			hover-class="detail-card__icon-action--active"
			hover-stay-time="80"
			@tap="openCookingMenu"
		>
			<up-icon name="more-dot-fill" size="16" color="#7b6d62"></up-icon>
		</view>
		<view
			v-else-if="!hasFlowchart && canRequestFlowchart && !isPublicView"
			class="detail-card__action detail-card__action--accent"
			:class="{ 'detail-card__action--disabled': isGeneratingFlowchart }"
			@tap="handleGenerateFlowchart"
		>
			<text class="detail-card__action-text detail-card__action-text--accent">{{ flowchartActionText }}</text>
		</view>
	</view>

	<!-- Tab 切换：仅当有流程图时显示；无图直接展示「详细步骤」内容 -->
	<view v-if="hasFlowchart" class="cooking-tabs">
		<view
			class="cooking-tabs__item"
			:class="{ 'cooking-tabs__item--active': activeCookingTab === 'flowchart' }"
			hover-class="cooking-tabs__item--hover"
			hover-stay-time="60"
			@tap="switchCookingTab('flowchart')"
		>
			<text class="cooking-tabs__text">一图看懂</text>
		</view>
		<view
			class="cooking-tabs__item"
			:class="{ 'cooking-tabs__item--active': activeCookingTab === 'steps' }"
			hover-class="cooking-tabs__item--hover"
			hover-stay-time="60"
			@tap="switchCookingTab('steps')"
		>
			<text class="cooking-tabs__text">详细步骤</text>
		</view>
	</view>

	<!-- 顶部状态条：根据当前 Tab 显示对应的处理状态 -->
	<view
		v-if="activeCookingTab === 'flowchart' && flowchartStatusMeta"
		class="detail-parse"
		:class="`detail-parse--${flowchartStatusMeta.tone}`"
	>
		<view class="detail-parse__body">
			<view class="detail-parse__badge">
				<text class="detail-parse__badge-text">{{ flowchartStatusMeta.label }}</text>
			</view>
			<text class="detail-parse__desc">{{ flowchartStatusDescription }}</text>
		</view>
	</view>
	<view
		v-else-if="showCookingStepsView && parseStatusMeta && parseStatusValue !== 'done'"
		class="detail-parse"
		:class="`detail-parse--${parseStatusMeta.tone}`"
	>
		<view class="detail-parse__body">
			<view class="detail-parse__badge">
				<text class="detail-parse__badge-text">{{ parseStatusMeta.label }}</text>
			</view>
			<text class="detail-parse__desc">{{ parseStatusDescription }}</text>
		</view>
	</view>

	<view v-if="activeCookingTab === 'flowchart' && showFlowchartStaleHint" class="flowchart-hint">
		<up-icon name="info-circle" size="14" color="#b4664c"></up-icon>
		<text class="flowchart-hint__text">做法已更新，建议重新生成步骤图</text>
	</view>

	<RecipeFlowchartPanel
		:show-image="showCookingFlowchartView"
		:show-empty="activeCookingTab === 'flowchart' && !hasFlowchart"
		:image-url="flowchartDisplayImageUrl"
		:can-generate="canGenerateFlowchart"
		:empty-text="flowchartEmptyText"
		@image-error="handleFlowchartImageError"
		@preview="previewFlowchartImage"
		@open-viewer="openFlowchartViewer"
	/>

	<!-- Tab 内容区：详细步骤（食材 + 步骤） -->
	<template v-if="showCookingStepsView">
		<view v-if="!hasFlowchart && canRequestFlowchart && hasMeaningfulParsedContent && !isPublicView" class="cooking-flowchart-cta" @tap="handleGenerateFlowchart">
			<up-icon name="photo" size="14" color="#9a7343"></up-icon>
			<text class="cooking-flowchart-cta__text">生成「一图看懂」流程图</text>
			<text class="cooking-flowchart-cta__arrow">›</text>
		</view>

		<view class="parsed-section">
			<view class="parsed-section__head">
				<text class="parsed-section__title">主料</text>
				<!-- B1-2：「复制清单」覆盖主料 + 辅料，方便用户一次性带走购物清单 -->
				<view
					v-if="canCopyIngredientList"
					class="parsed-section__copy"
					hover-class="parsed-section__copy--hover"
					hover-stay-time="80"
					@tap="copyIngredientList"
				>
					<up-icon name="file-text" size="12" color="#9a7343"></up-icon>
					<text class="parsed-section__copy-text">复制清单</text>
				</view>
			</view>
			<!-- B1-1：主料 ≥ 3 项时上序号胶囊以强化数量感；< 3 项时改为紧凑点状列表，避免单条「1」的视觉冗余 -->
			<template v-if="parsedMainIngredients.length >= 3">
				<view
					v-for="(ingredient, index) in parsedMainIngredients"
					:key="`main-ingredient-${index}`"
					class="parsed-item"
				>
					<view class="parsed-item__index">
						<text class="parsed-item__index-text">{{ index + 1 }}</text>
					</view>
					<text class="parsed-item__text">{{ ingredient }}</text>
				</view>
			</template>
			<view v-else class="parsed-main-compact">
				<view
					v-for="(ingredient, index) in parsedMainIngredients"
					:key="`main-compact-${index}`"
					class="parsed-main-compact__item"
				>
					<view class="parsed-main-compact__dot"></view>
					<text class="parsed-main-compact__text">{{ ingredient }}</text>
				</view>
			</view>
		</view>

		<view v-if="parsedSecondaryGroups.length" class="parsed-section">
			<text class="parsed-section__title">辅料</text>
			<view
				v-for="group in parsedSecondaryGroups"
				:key="group.key"
				class="parsed-group"
			>
				<text class="parsed-group__label">{{ group.label }}</text>
				<text class="parsed-group__text">{{ group.text }}</text>
			</view>
		</view>

		<view class="parsed-section parsed-section--steps">
			<view class="parsed-section__head">
				<text class="parsed-section__title">制作步骤</text>
				<!-- B2-6：完成进度提示，仅当至少勾过 1 步时显示 -->
				<view v-if="completedStepCount > 0" class="parsed-section__progress">
					<text class="parsed-section__progress-text">{{ completedStepCount }} / {{ parsedSteps.length }}</text>
					<view
						class="parsed-section__progress-reset"
						hover-class="parsed-section__progress-reset--hover"
						hover-stay-time="80"
						@tap="resetCompletedSteps"
					>
						<text class="parsed-section__progress-reset-text">重置</text>
					</view>
				</view>
			</view>
			<view
				v-for="(step, index) in parsedSteps"
				:key="`step-${index}`"
				class="step-item"
				:class="{ 'step-item--done': isStepCompleted(index) }"
				hover-class="step-item--hover"
				hover-stay-time="60"
				@tap="toggleStepCompleted(index)"
			>
				<view class="step-item__index" :class="{ 'step-item__index--done': isStepCompleted(index) }">
					<!-- B1-4：去掉「Step」英文前缀，中文用户不需要，纯数字更聚焦 -->
					<!-- B2-6：已完成时显示对勾图标替代序号 -->
					<up-icon v-if="isStepCompleted(index)" name="checkmark" size="14" color="#6f8266"></up-icon>
					<text v-else class="step-item__index-text">{{ index + 1 }}</text>
				</view>
				<view class="step-item__body">
					<text class="step-item__title">{{ step.title }}</text>
					<!-- B2-5：把详情切成 segments，关键参数（时间/克数/火候）加粗高亮 -->
					<view class="step-item__text">
						<text
							v-for="(seg, segIndex) in highlightStepDetail(step.detail)"
							:key="`step-${index}-seg-${segIndex}`"
							:class="seg.highlight ? 'step-item__text--highlight' : 'step-item__text--normal'"
						>{{ seg.text }}</text>
					</view>
				</view>
			</view>
		</view>
	</template>


	<!-- 底部合并元信息行 -->
	<view v-if="cookingFooterText" class="cooking-footer">
		<text class="cooking-footer__text">{{ cookingFooterText }}</text>
	</view>
</view>
</template>

<script setup>
import RecipeFlowchartPanel from './recipe-flowchart-panel.vue'

defineProps({
	isCookingActive: Boolean,
	cookingActiveLabel: String,
	hasCookingMenuItems: Boolean,
	hasFlowchart: Boolean,
	activeCookingTab: String,
	flowchartStatusMeta: Object,
	flowchartStatusValue: String,
	flowchartStatusDescription: String,
	parseStatusMeta: Object,
	parseStatusValue: String,
	parseStatusDescription: String,
	showCookingFlowchartView: Boolean,
	flowchartDisplayImageUrl: String,
	canRequestFlowchart: Boolean,
	canGenerateFlowchart: Boolean,
	isFlowchartActive: Boolean,
	isGeneratingFlowchart: Boolean,
	flowchartActionText: String,
	showFlowchartStaleHint: Boolean,
	hasMeaningfulParsedContent: Boolean,
	isPublicView: Boolean,
	showCookingStepsView: Boolean,
	parsedMainIngredients: Array,
	canCopyIngredientList: Boolean,
	parsedSecondaryGroups: Array,
	parsedSteps: Array,
	completedStepCount: Number,
	canRequestParse: Boolean,
	flowchartEmptyText: String,
	cookingFooterText: String,
	isStepCompleted: Function,
	highlightStepDetail: Function
})
const emit = defineEmits([
	'open-menu', 'switch-tab', 'flowchart-image-error', 'preview-flowchart',
	'open-flowchart-viewer', 'generate-flowchart', 'copy-ingredients', 'toggle-step',
	'reset-steps', 'parse'
])
function openCookingMenu() { emit('open-menu') }
function switchCookingTab(tab) { emit('switch-tab', tab) }
function handleFlowchartImageError(event) { emit('flowchart-image-error', event) }
function previewFlowchartImage() { emit('preview-flowchart') }
function openFlowchartViewer() { emit('open-flowchart-viewer') }
function handleGenerateFlowchart() { emit('generate-flowchart') }
function copyIngredientList() { emit('copy-ingredients') }
function toggleStepCompleted(index) { emit('toggle-step', index) }
function resetCompletedSteps() { emit('reset-steps') }
function handleParseAction() { emit('parse') }
</script>
