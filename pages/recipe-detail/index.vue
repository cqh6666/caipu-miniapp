<template>
	<view class="detail-page" :class="{ 'detail-page--public': isPublicView }">
		<!-- P2-D 分享路径升级：公开只读模式顶部说明 banner -->
		<!-- 加载成功后显示，给接收者「来自 XX 的空间 · 加入后可参与编辑」语境 -->
		<view v-if="isPublicView && recipe && !publicViewLoadFailed" class="public-banner">
			<view class="public-banner__body">
				<up-icon name="eye" size="14" color="#7b6d62"></up-icon>
				<text class="public-banner__text">{{ publicBannerText }}</text>
			</view>
			<view class="public-banner__action" hover-class="public-banner__action--hover" hover-stay-time="80" @tap="openPublicReadOnlyExplain">
				<text class="public-banner__action-text">了解</text>
			</view>
		</view>
		<template v-if="recipe">
			<scroll-view class="detail-scroll" scroll-y>
				<!-- P2 修复：公开只读模式下「无成品图」菜谱整块跳过 Hero 区，
				     避免被 .hero-card 的 min-height: 380rpx 撑出大块空白；
				     无图时下方 detail-head 兜底分支接管标题与 meta 渲染。
				     私有模式保留原 placeholder（含「上传成品图」CTA），仍需要 380rpx 撑住按钮 -->
				<view
					v-if="displayRecipeImages.length || !isPublicView"
					class="hero-card"
					:class="{ 'hero-card--empty': !recipeImages.length, 'hero-card--with-overlay': displayRecipeImages.length > 0 }"
					@tap="handleHeroCardTap"
				>
					<swiper
						v-if="displayRecipeImages.length"
						class="hero-card__swiper"
						:circular="displayRecipeImages.length > 1"
						:autoplay="false"
						:duration="280"
						@change="handleHeroSwiperChange"
					>
						<swiper-item v-for="(image, index) in displayRecipeImages" :key="image.cacheKey || `hero-image-${index}`">
							<image
								class="hero-card__image"
								:src="image.displayURL"
								mode="aspectFill"
								@error="handleRecipeImageError(image)"
							></image>
						</swiper-item>
					</swiper>
					<!-- H4：底部渐变蒙层，为压图标题/分页器提供读性 -->
					<view v-if="displayRecipeImages.length" class="hero-card__overlay" @tap.stop="handleHeroCardTap"></view>
					<!-- Hero 操作菜单：右下角 ⋯ 按钮，按当前位置动态装配「设为封面 / 添加 / 删除」 -->
					<!-- P2-D：公开只读模式下隐藏，避免接收者点击触发编辑路径 -->
					<view
						v-if="canShowHeroActionMenu && !isPublicView"
						class="hero-card__action"
						hover-class="hero-card__action--hover"
						hover-stay-time="80"
						@tap.stop="openHeroActionMenu"
					>
						<up-icon name="more-dot-fill" size="14" color="#ffffff"></up-icon>
					</view>
					<!-- H5：标题 + meta chips 压图（仅在有图时） -->
					<view v-if="displayRecipeImages.length" class="hero-card__title-block" @tap.stop="handleHeroCardTap">
						<view class="hero-card__meta">
							<view class="hero-card__chip hero-card__chip--meal">
								<text class="hero-card__chip-text">{{ mealLabel }}</text>
							</view>
							<view
								class="hero-card__chip"
								:class="recipe?.status === 'done' ? 'hero-card__chip--done' : 'hero-card__chip--wishlist'"
							>
								<text class="hero-card__chip-text">{{ statusLabel }}</text>
							</view>
							<view v-if="isPinned" class="hero-card__chip hero-card__chip--pin">
								<text class="hero-card__chip-text">已置顶</text>
							</view>
						</view>
						<text class="hero-card__title">{{ recipe.title }}</text>
					</view>
					<!-- H3：底部居中分页器；≤5 张用圆点 dots，>5 张用数字 chip -->
					<view v-if="displayRecipeImages.length > 1" class="hero-card__pager">
						<template v-if="displayRecipeImages.length <= 5">
							<view
								v-for="i in displayRecipeImages.length"
								:key="`hero-dot-${i}`"
								class="hero-card__dot"
								:class="{ 'hero-card__dot--active': i - 1 === heroImageIndex }"
							></view>
						</template>
						<view v-else class="hero-card__counter">
							<text class="hero-card__counter-text">{{ heroImageIndex + 1 }} / {{ displayRecipeImages.length }}</text>
						</view>
					</view>
					<view v-if="!displayRecipeImages.length && !isPublicView" class="hero-card__placeholder">
						<view class="hero-card__placeholder-mask"></view>
						<view class="hero-card__upload-action" :class="{ 'hero-card__upload-action--loading': isUploadingHeroImage }">
							<up-icon :name="recipeImages.length ? 'photo' : (isUploadingHeroImage ? 'reload' : 'plus')" size="18" color="#5b4a3b"></up-icon>
							<text class="hero-card__upload-action-text">{{ recipeImages.length ? '封面加载失败，点查看原图' : (isUploadingHeroImage ? '上传中...' : '上传成品图') }}</text>
						</view>
					</view>
				</view>

				<!-- H5：无图时回退到外部 detail-head 渲染标题 + meta；有图时压在图上 -->
				<view v-if="!displayRecipeImages.length" class="detail-head">
					<view class="detail-head__meta">
						<view class="detail-chip detail-chip--meal">
							<text class="detail-chip__text">{{ mealLabel }}</text>
						</view>
						<view
							class="detail-chip"
							:class="recipe?.status === 'done' ? 'detail-chip--done' : 'detail-chip--wishlist'"
						>
							<text class="detail-chip__text">{{ statusLabel }}</text>
						</view>
						<view v-if="isPinned" class="detail-chip detail-chip--pin">
							<text class="detail-chip__text">已置顶</text>
						</view>
					</view>
					<text class="detail-title">{{ recipe.title }}</text>
					<view v-if="recipe.summary" class="detail-summary-card">
						<text class="detail-summary">{{ recipe.summary }}</text>
					</view>
				</view>
				<!-- H5：有图时只渲染 summary（标题已经在图上） -->
				<view v-else-if="recipe.summary" class="detail-head detail-head--summary-only">
					<view class="detail-summary-card">
						<text class="detail-summary">{{ recipe.summary }}</text>
					</view>
				</view>

				<!--
					P2-A: 「一图看懂」与「做法整理」合并为统一「做法」卡片
					- 顶部 Tab：仅当有流程图时显示，默认选中「一图看懂」
					- 右上 ⋯：合并菜单（重新生成 / 重新整理 / 查看详情）
					- 底部：合并元信息行（AI 生成 · MM-DD · 来源）
				-->
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

					<!-- Tab 内容区：一图看懂 -->
					<view v-if="showCookingFlowchartView" class="flowchart-panel">
						<view class="flowchart-panel__image-shell">
							<image
								class="flowchart-panel__image"
								:src="flowchartDisplayImageUrl"
								mode="widthFix"
								hover-class="flowchart-panel__image--active"
								hover-stay-time="80"
								@error="handleFlowchartImageError"
								@tap="previewFlowchartImage"
							></image>
							<view class="flowchart-panel__image-shadow"></view>
							<view
								class="flowchart-panel__cta"
								hover-class="flowchart-panel__cta--active"
								hover-stay-time="80"
								@tap.stop="openFlowchartViewer"
							>
								<text class="flowchart-panel__cta-text">横屏查看</text>
								<text class="flowchart-panel__cta-arrow">›</text>
							</view>
						</view>
					</view>

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

					<!-- 流程图空态：无图且选中流程图 Tab 时（理论上不会出现，因为无图 Tab 隐藏；这里保险起见保留） -->
					<view v-if="activeCookingTab === 'flowchart' && !hasFlowchart" class="flowchart-empty" :class="{ 'flowchart-empty--disabled': !canGenerateFlowchart }">
						<view class="flowchart-empty__icon">
							<up-icon name="photo" size="24" color="#b08c72"></up-icon>
						</view>
						<text class="flowchart-empty__title">还没生成步骤图</text>
						<text class="flowchart-empty__desc">{{ flowchartEmptyText }}</text>
					</view>

					<!-- 底部合并元信息行 -->
					<view v-if="cookingFooterText" class="cooking-footer">
						<text class="cooking-footer__text">{{ cookingFooterText }}</text>
					</view>
				</view>

				<!-- P2 修复：公开只读模式下后端 DTO 已剔除 link/note，
				     前端再隐藏整块卡片，避免出现「来源链接 / 暂无链接」「备注 / 暂无备注」空卡，
				     体验从「没有内容」变为「该内容不公开」 -->
				<view v-if="!isPublicView" class="detail-card detail-card--quiet">
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
								<text class="detail-link-text" selectable>{{ displayRecipeLink }}</text>
							</view>
						</view>
					<text v-else class="detail-empty">暂无链接。</text>
				</view>

				<view v-if="!isPublicView" class="detail-card detail-card--note detail-card--quiet">
					<view class="detail-card__header detail-card__header--stack">
						<text class="detail-card__title">备注</text>
					</view>
					<text v-if="recipe.note" class="detail-note">{{ recipe.note }}</text>
					<text v-else class="detail-empty">暂无备注。</text>
				</view>
			</scroll-view>

			<view v-if="!isPublicView" class="detail-footer">
				<view class="detail-footer__action detail-footer__action--ghost detail-footer__action--delete" @tap="confirmDeleteRecipe">
					<up-icon name="trash" size="18" color="#b4664c"></up-icon>
				</view>
				<view
					class="detail-footer__action detail-footer__action--soft detail-footer__action--pin"
					:class="{
						'detail-footer__action--soft-active': isPinned,
						'detail-footer__action--disabled': isPinSubmitting
					}"
					@tap="togglePinned"
				>
					<text
						class="detail-footer__text"
						:class="{ 'detail-footer__text--accent': isPinned }"
					>{{ pinActionText }}</text>
				</view>
				<view class="detail-footer__action detail-footer__action--primary detail-footer__action--edit" @tap="openEditSheet">
					<text class="detail-footer__text detail-footer__text--primary">编辑</text>
				</view>
			</view>
		</template>

		<template v-else-if="showRecipeLoadingState">
			<view class="detail-loading">
				<view class="detail-loading__hero detail-loading__pulse"></view>
				<view class="detail-loading__section">
					<view class="detail-loading__chips">
						<view class="detail-loading__chip detail-loading__pulse"></view>
						<view class="detail-loading__chip detail-loading__chip--short detail-loading__pulse"></view>
					</view>
					<view class="detail-loading__title detail-loading__pulse"></view>
					<view class="detail-loading__line detail-loading__pulse"></view>
				</view>
				<view class="detail-loading__card">
					<view class="detail-loading__card-title detail-loading__pulse"></view>
					<view class="detail-loading__line detail-loading__pulse"></view>
					<view class="detail-loading__line detail-loading__line--short detail-loading__pulse"></view>
				</view>
				<view class="detail-loading__card">
					<view class="detail-loading__card-title detail-loading__pulse"></view>
					<view class="detail-loading__row detail-loading__pulse"></view>
					<view class="detail-loading__row detail-loading__pulse"></view>
					<view class="detail-loading__row detail-loading__row--short detail-loading__pulse"></view>
				</view>
			</view>
		</template>

		<template v-else>
			<view class="missing-state">
				<up-icon name="info-circle" size="42" color="#b8aa9b"></up-icon>
				<!-- P2-D：区分公开链接失效 vs 私有未找到，给出不同文案 -->
				<text class="missing-state__title">{{ isPublicView ? '分享链接已失效' : '没找到这道菜' }}</text>
				<text class="missing-state__desc">{{ isPublicView ? '这道菜谱可能已被删除或分享已收回。' : '可能已删除或未保存。' }}</text>
				<view class="missing-state__action" @tap="handleMissingStateBack">
					<text class="missing-state__action-text">{{ isPublicView ? '返回上一页' : '返回列表' }}</text>
				</view>
			</view>
		</template>

		<action-feedback
			:visible="actionFeedbackVisible"
			:feedback-key="actionFeedbackKey"
			:tone="actionFeedbackTone"
			:title="actionFeedbackTitle"
			:description="actionFeedbackDescription"
		></action-feedback>

		<!-- P2-D 分享路径升级：只读规则说明 popup -->
		<!-- banner 「了解」按钮触发，解释为何不能编辑 + 如何获得编辑权 -->
		<up-popup
			v-if="showPublicReadOnlyExplain"
			:show="showPublicReadOnlyExplain"
			mode="center"
			round="20"
			overlayOpacity="0.32"
			:closeOnClickOverlay="true"
			@close="closePublicReadOnlyExplain"
		>
			<view class="public-explain">
				<view class="public-explain__icon">
					<up-icon name="eye" size="22" color="#7b6d62"></up-icon>
				</view>
				<text class="public-explain__title">这是一份只读分享</text>
				<text class="public-explain__desc">{{ publicExplainBody }}</text>
				<view class="public-explain__action" hover-class="public-explain__action--hover" hover-stay-time="80" @tap="closePublicReadOnlyExplain">
					<text class="public-explain__action-text">我知道了</text>
				</view>
			</view>
		</up-popup>

		<up-popup
			v-if="showEditSheet"
			:show="showEditSheet"
			mode="bottom"
			round="32"
			overlayOpacity="0.22"
			:closeOnClickOverlay="false"
			:safeAreaInsetBottom="false"
			@close="handleEditSheetPopupClose"
		>
			<view class="editor-sheet">
				<view class="editor-sheet__header">
					<view class="editor-sheet__heading">
						<text class="editor-sheet__title">编辑菜品</text>
						<text class="editor-sheet__subtitle">把这道菜补充完整。</text>
					</view>
					<view class="editor-sheet__close" @tap="requestCloseEditSheet">
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
						<view class="editor-gallery">
							<view
								v-for="(image, index) in editDraft.images"
								:key="`edit-image-${index}`"
								class="editor-gallery__item"
								@tap="previewEditImages(index)"
							>
								<image class="editor-gallery__thumb" :src="image" mode="aspectFill"></image>
								<view
									v-if="editDraft.images.length > 1"
									class="editor-gallery__sort"
									@tap.stop="openEditImageOrderActions(index)"
								>
									<text class="editor-gallery__sort-text">排序</text>
								</view>
								<view class="editor-gallery__badge">
									<text class="editor-gallery__badge-text">{{ index === 0 ? '封面' : index + 1 }}</text>
								</view>
								<view class="editor-gallery__remove" @tap.stop="removeEditImage(index)">
									<up-icon name="close" size="14" color="#ffffff"></up-icon>
								</view>
							</view>
							<view
								v-if="editDraft.images.length < maxRecipeImages"
								class="editor-gallery__add"
								@tap="chooseEditImages"
							>
								<view class="editor-gallery__plus">
									<up-icon name="plus" size="20" color="#8c8074"></up-icon>
								</view>
								<text class="editor-gallery__add-text">上传成品图</text>
							</view>
						</view>
						<text class="editor-field__hint">
							{{ editDraft.images.length ? `已添加 ${editDraft.images.length} 张，首张作封面，可调整顺序。` : `最多上传 ${maxRecipeImages} 张，首张作封面。` }}
						</text>
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
						<view class="editor-field__head">
							<text class="editor-field__label">食材清单</text>
							<text class="editor-field__meta">{{ editIngredientCount }} 项</text>
						</view>
						<view class="editor-structured">
							<view class="editor-structured__section">
								<view class="editor-structured__header">
									<view class="editor-structured__heading">
										<text class="editor-structured__title">主料</text>
										<text class="editor-structured__desc">核心食材和份量</text>
									</view>
									<view class="editor-structured__action" @tap="addEditIngredient('main')">
										<text class="editor-structured__action-text">添加</text>
									</view>
								</view>

								<view v-if="editDraft.mainIngredients.length" class="editor-ingredient-list">
									<view
										v-for="(ingredient, index) in editDraft.mainIngredients"
										:key="ingredient.id"
										class="editor-ingredient-item"
									>
										<view class="editor-ingredient-item__index">
											<text class="editor-ingredient-item__index-text">{{ index + 1 }}</text>
										</view>
										<input
											:value="ingredient.value"
											class="editor-ingredient-item__input"
											placeholder="例如：牛肉 500g"
											placeholder-class="editor-input__placeholder"
											maxlength="60"
											@input="handleEditIngredientInput('main', index, $event)"
										/>
										<view class="editor-ingredient-item__menu" @tap="openEditIngredientActions('main', index)">
											<view class="editor-ingredient-item__menu-dots">
												<view class="editor-ingredient-item__menu-dot"></view>
												<view class="editor-ingredient-item__menu-dot"></view>
												<view class="editor-ingredient-item__menu-dot"></view>
											</view>
										</view>
									</view>
								</view>
								<view v-else class="editor-structured__empty">
									<text class="editor-structured__empty-text">{{ ingredientGroupEmptyText('main') }}</text>
								</view>
							</view>

							<view class="editor-structured__section">
								<view class="editor-structured__header">
									<view class="editor-structured__heading">
										<text class="editor-structured__title">辅料 / 调味</text>
										<text class="editor-structured__desc">配菜、调味和辅助食材</text>
									</view>
									<view class="editor-structured__action" @tap="addEditIngredient('secondary')">
										<text class="editor-structured__action-text">添加</text>
									</view>
								</view>

								<view v-if="editDraft.secondaryIngredients.length" class="editor-ingredient-list">
									<view
										v-for="(ingredient, index) in editDraft.secondaryIngredients"
										:key="ingredient.id"
										class="editor-ingredient-item"
									>
										<view class="editor-ingredient-item__index">
											<text class="editor-ingredient-item__index-text">{{ index + 1 }}</text>
										</view>
										<input
											:value="ingredient.value"
											class="editor-ingredient-item__input"
											placeholder="例如：葱姜蒜、盐、生抽"
											placeholder-class="editor-input__placeholder"
											maxlength="60"
											@input="handleEditIngredientInput('secondary', index, $event)"
										/>
										<view class="editor-ingredient-item__menu" @tap="openEditIngredientActions('secondary', index)">
											<view class="editor-ingredient-item__menu-dots">
												<view class="editor-ingredient-item__menu-dot"></view>
												<view class="editor-ingredient-item__menu-dot"></view>
												<view class="editor-ingredient-item__menu-dot"></view>
											</view>
										</view>
									</view>
								</view>
								<view v-else class="editor-structured__empty">
									<text class="editor-structured__empty-text">{{ ingredientGroupEmptyText('secondary') }}</text>
								</view>
							</view>
						</view>
						<text class="editor-field__hint">
							{{ editIsUsingFallbackContent ? '还没添加食材，直接补充即可。' : '食材按主料和辅料分开显示，可调整顺序。' }}
						</text>
					</view>

					<view class="editor-field">
						<view class="editor-field__head">
							<text class="editor-field__label">制作步骤</text>
							<text class="editor-field__meta">{{ editStepCount }} 步</text>
						</view>
						<view class="editor-step-list">
							<view
								v-for="(step, index) in editDraft.steps"
								:key="step.id"
								class="editor-step-card"
							>
								<view class="editor-step-card__header">
									<view class="editor-step-card__badge">
										<text class="editor-step-card__badge-text">Step {{ index + 1 }}</text>
									</view>
									<view class="editor-step-card__actions">
										<view
											class="editor-step-card__action"
											:class="{ 'editor-step-card__action--disabled': index === 0 }"
											@tap="moveEditStep(index, index - 1)"
										>
											<text class="editor-step-card__action-text">上移</text>
										</view>
										<view
											class="editor-step-card__action"
											:class="{ 'editor-step-card__action--disabled': index === editDraft.steps.length - 1 }"
											@tap="moveEditStep(index, index + 1)"
										>
											<text class="editor-step-card__action-text">下移</text>
										</view>
										<view class="editor-step-card__action editor-step-card__action--danger" @tap="removeEditStep(index)">
											<text class="editor-step-card__action-text editor-step-card__action-text--danger">删除</text>
										</view>
									</view>
								</view>

								<view class="editor-step-card__field">
									<text class="editor-step-card__label">步骤标题</text>
									<input
										:value="step.title"
										class="editor-step-card__input"
										placeholder="例如：腌制入味"
										placeholder-class="editor-input__placeholder"
										maxlength="30"
										@input="handleEditStepFieldInput(index, 'title', $event)"
									/>
								</view>

								<view class="editor-step-card__field">
									<text class="editor-step-card__label">步骤内容</text>
									<textarea
										:value="step.detail"
										auto-height
										class="editor-step-card__textarea"
										placeholder="写清楚这一小步的动作、时间或火候"
										placeholder-class="editor-textarea__placeholder"
										maxlength="220"
										@input="handleEditStepFieldInput(index, 'detail', $event)"
									/>
								</view>
							</view>

							<view v-if="!editDraft.steps.length" class="editor-structured__empty editor-structured__empty--large">
								<text class="editor-structured__empty-text">{{ stepEmptyText() }}</text>
							</view>

							<view class="editor-step-add" @tap="addEditStep">
								<text class="editor-step-add__text">添加一步</text>
							</view>
						</view>
						<text class="editor-field__hint">
							{{ editIsUsingFallbackContent ? '还没添加步骤，直接补充即可。' : '步骤标题可选填，留空会自动补全。' }}
						</text>
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
					<view class="editor-sheet__action" @tap="requestCloseEditSheet">
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
	<canvas canvas-id="_flowchart_square_canvas" style="position:fixed;left:-9999px;top:-9999px;width:300px;height:300px;"></canvas>
</template>

<script>
import ActionFeedback from '../../components/action-feedback.vue'
import {
	MAX_RECIPE_IMAGES,
	deleteRecipeById,
	ensureRecipeShareTokenById,
	fetchPublicRecipeByShareToken,
	generateRecipeFlowchartById,
	getCachedRecipeById,
	getRecipeById,
	isFallbackParsedContent as isFallbackLikeParsedContent,
	mealTypeLabelMap,
	mealTypeOptions,
	normalizeParsedContentView,
	normalizeParsedSteps,
	normalizeTextList,
	reparseRecipeById,
	setRecipePinnedById,
	statusLabelMap,
	statusOptions,
	updateRecipeById
} from '../../utils/recipe-store'
import { buildImageCacheKey, getCachedImagePath, invalidateCachedImage, warmImageCache } from '../../utils/image-cache'

const createEmptyDraft = (overrides = {}) => ({
	title: '',
	ingredient: '',
	link: '',
	images: [],
	mealType: 'breakfast',
	status: 'wishlist',
	mainIngredients: [],
	secondaryIngredients: [],
	steps: [],
	parsedContentMode: 'empty',
	note: '',
	...overrides
})
let editDraftItemSeed = 0
const createEditDraftItemId = (prefix = 'draft') => `${prefix}-${Date.now()}-${editDraftItemSeed += 1}`
const normalizeIngredientDraftItem = (item = '') => {
	if (typeof item === 'object' && item !== null) {
		return {
			id: String(item.id || createEditDraftItemId('ingredient')),
			value: String(item.value || '')
		}
	}
	return {
		id: createEditDraftItemId('ingredient'),
		value: String(item || '')
	}
}
const createIngredientDraftList = (items = []) => (Array.isArray(items) ? items : []).map((item) => normalizeIngredientDraftItem(item))
const getIngredientDraftValues = (items = []) =>
	(Array.isArray(items) ? items : []).map((item) => (typeof item === 'object' && item !== null ? String(item.value || '') : String(item || '')))
const createStepDraftItem = (step = {}) => {
	const source = typeof step === 'object' && step !== null ? step : { detail: step }
	return {
		id: String(source.id || createEditDraftItemId('step')),
		title: String(source.title || ''),
		detail: String(source.detail || source.text || '')
	}
}
const moveListItem = (items = [], fromIndex = 0, toIndex = 0) => {
	if (!Array.isArray(items) || !items.length) return Array.isArray(items) ? items : []
	if (fromIndex < 0 || fromIndex >= items.length) return items
	if (toIndex < 0 || toIndex >= items.length || fromIndex === toIndex) return items

	const list = [...items]
	const [item] = list.splice(fromIndex, 1)
	list.splice(toIndex, 0, item)
	return list
}
const cloneStepDraftList = (steps = []) => normalizeParsedSteps(steps).map((step) => createStepDraftItem(step))
const buildComparableDraftTextList = (items = []) =>
	getIngredientDraftValues(items)
		.map((item) => item.trim())
		.filter(Boolean)
const buildComparableDraftStepList = (steps = []) =>
	(Array.isArray(steps) ? steps : [])
		.map((step) => {
			const normalized = createStepDraftItem(step)
			return {
				title: normalized.title.trim(),
				detail: normalized.detail.trim()
			}
		})
		.filter((step) => step.title || step.detail)
const serializeComparableEditDraft = (draft = {}) =>
	JSON.stringify({
		title: String(draft.title || '').trim(),
		ingredient: String(draft.ingredient || '').trim(),
		link: String(draft.link || '').trim(),
		images: (Array.isArray(draft.images) ? draft.images : []).map((item) => String(item || '').trim()).filter(Boolean),
		mealType: String(draft.mealType || '').trim(),
		status: String(draft.status || '').trim(),
		mainIngredients: buildComparableDraftTextList(draft.mainIngredients),
		secondaryIngredients: buildComparableDraftTextList(draft.secondaryIngredients),
		steps: buildComparableDraftStepList(draft.steps),
		note: String(draft.note || '').trim()
	})

const ACTIVE_PARSE_STATUSES = ['pending', 'processing']
const ACTIVE_FLOWCHART_STATUSES = ['pending', 'processing']
const FLOWCHART_VIEWER_STORAGE_KEY = 'recipe-flowchart-viewer-payload'
const parseStatusMetaMap = {
	idle: {
		label: '可自动整理',
		tone: 'pending',
		description: '支持链接自动整理，可手动开始整理当前做法。'
	},
	pending: {
		label: '等待解析',
		tone: 'pending',
		description: '已加入后台整理队列，稍后会自动补齐食材和步骤。'
	},
	processing: {
		label: '解析中',
		tone: 'processing',
		description: '后台正在整理链接内容，结果会自动更新。'
	},
	done: {
		label: '已自动整理',
		tone: 'done',
		description: '食材和步骤已自动整理完成。'
	},
	failed: {
		label: '解析失败',
		tone: 'failed',
		description: '这次自动整理没成功，可以再试一次。'
	}
}

const flowchartStatusMetaMap = {
	pending: {
		label: '等待出图',
		tone: 'pending',
		description: '已加入生成队列，稍后会自动补上步骤图。'
	},
	processing: {
		label: '正在出图',
		tone: 'processing',
		description: '后台正在整理步骤图，完成后会自动刷新。'
	},
	failed: {
		label: '生成失败',
		tone: 'failed',
		description: '这次步骤图生成没成功，可以重新再试。'
	}
}

function isAutoParseSupportedLink(link = '') {
	return /(bilibili\.com|b23\.tv|bili2233\.cn|xiaohongshu\.com|xhslink\.com)/i.test(String(link).trim())
}

function extractCopyableLink(value = '') {
	const source = String(value || '').trim()
	if (!source) return ''
	const matched = source.match(/https?:\/\/[^\s]+/i)
	const link = String(matched?.[0] || source).trim()
	return link.replace(/[)\]】》」'",，。；;!?！？]+$/g, '').trim()
}

function formatParseSourceLabel(source = '') {
	const value = String(source).trim()
	if (!value) return ''
	if (value === 'bilibili') return '来源：B 站链接自动解析'
	if (value === 'bilibili:ai') return '来源：B 站内容 + AI 总结'
	if (value === 'bilibili:heuristic') return '来源：B 站规则整理'
	if (value.startsWith('xiaohongshu')) {
		const parts = value.toLowerCase().split(':').filter(Boolean)
		const summaryMode = parts.includes('ai') ? 'ai' : parts.includes('heuristic') ? 'heuristic' : ''
		if (!summaryMode) return '来源：小红书链接自动解析'
		if (summaryMode === 'ai') return '来源：小红书 + AI 总结'
		if (summaryMode === 'heuristic') return '来源：小红书规则整理'
	}
	return `来源：${value}`
}

function buildParseResultHint(status = '', source = '') {
	const normalizedStatus = String(status || '').trim().toLowerCase()
	const normalizedSource = String(source || '').trim().toLowerCase()
	if (normalizedStatus !== 'done') return ''
	if (normalizedSource === 'bilibili:heuristic') {
		return '这次先按规则整理，通常是因为字幕不可用，或 AI 总结暂时不可用；可以稍后再试一次。'
	}
	return ''
}

function toPositiveInteger(value = 0) {
	const parsed = Number(value)
	if (!Number.isFinite(parsed) || parsed <= 0) return 0
	return Math.ceil(parsed)
}

function resolveRemainingWaitSeconds(value = 0, syncedAt = 0, now = 0) {
	const base = toPositiveInteger(value)
	if (!base) return 0
	const startedAt = Number(syncedAt) || 0
	const current = Number(now) || 0
	const elapsedSeconds = startedAt > 0 && current > startedAt ? Math.floor((current - startedAt) / 1000) : 0
	return Math.max(base - elapsedSeconds, 0)
}

function formatApproxWait(seconds = 0) {
	const totalSeconds = toPositiveInteger(seconds)
	if (!totalSeconds) return ''
	if (totalSeconds < 60) {
		const rounded = Math.max(5, Math.ceil(totalSeconds / 5) * 5)
		return `${rounded} 秒左右`
	}
	if (totalSeconds < 3600) {
		const minutes = Math.max(1, Math.ceil(totalSeconds / 60))
		return `${minutes} 分钟左右`
	}
	const hours = Math.floor(totalSeconds / 3600)
	const minutes = Math.ceil((totalSeconds % 3600) / 60)
	if (!minutes) return `${hours} 小时左右`
	return `${hours} 小时 ${minutes} 分钟左右`
}

function buildParseWaitHint(status = '', queueAhead = 0, waitSeconds = 0) {
	const normalizedStatus = String(status || '').trim().toLowerCase()
	const waitText = formatApproxWait(waitSeconds)
	if (!waitText) return ''
	if (normalizedStatus === 'pending') {
		if (queueAhead > 0) {
			return `前面还有 ${queueAhead} 个任务，预计还要 ${waitText}，整理完成后会自动刷新。`
		}
		return `已加入整理队列，预计 ${waitText} 后完成。`
	}
	if (normalizedStatus === 'processing') {
		return `后台正在整理链接内容，预计还要 ${waitText}，完成后会自动刷新。`
	}
	return ''
}

function buildFlowchartWaitHint(status = '', queueAhead = 0, waitSeconds = 0) {
	const normalizedStatus = String(status || '').trim().toLowerCase()
	const waitText = formatApproxWait(waitSeconds)
	if (!waitText) return ''
	if (normalizedStatus === 'pending') {
		if (queueAhead > 0) {
			return `前面还有 ${queueAhead} 个任务，预计还要 ${waitText}，出图完成后会自动刷新。`
		}
		return `已加入出图队列，预计 ${waitText} 后完成。`
	}
	if (normalizedStatus === 'processing') {
		return `后台正在生成步骤图，预计还要 ${waitText}，完成后会自动刷新。`
	}
	return ''
}

function formatDateTime(value = '') {
	const date = new Date(value)
	if (Number.isNaN(date.getTime())) return ''
	const year = date.getFullYear()
	const month = `${date.getMonth() + 1}`.padStart(2, '0')
	const day = `${date.getDate()}`.padStart(2, '0')
	const hours = `${date.getHours()}`.padStart(2, '0')
	const minutes = `${date.getMinutes()}`.padStart(2, '0')
	return `${year}-${month}-${day} ${hours}:${minutes}`
}

function buildRecipeImageVersion(recipe = {}) {
	return String(recipe?.updatedAt || recipe?.parseFinishedAt || '').trim()
}

// B2-5：步骤详情中需要高亮的关键参数模式
// - 数量+单位：8分钟 / 5g / 100ml / 2勺 / 3条 ...
// - 火候：大火 / 中火 / 小火 / 中小火 / 文火 / 武火 / 旺火
// - 温度：180度 / 180°C / 摄氏180度
// 注意：使用 m 与 g 标志，逐段匹配；优先匹配长 token（火候 + 温度），再匹配数量+单位
const STEP_HIGHLIGHT_REGEX = /(\d+(?:\.\d+)?\s?(?:分钟|秒|小时|分|克|斤|g|kg|毫升|ml|L|勺|匙|杯|碗|条|个|片|块|颗|粒|根|瓣|只|滴|圈)|大火|中火|小火|中小火|大小火|文火|武火|旺火|微火|小炖|\d+\s?°?C|\d+\s?度)/g

function highlightStepDetailText(detail) {
	const raw = String(detail || '').trim()
	if (!raw) return [{ text: '', highlight: false }]

	const segments = []
	let lastIndex = 0
	let match

	// 重置 lastIndex 防止全局正则状态污染（多次调用同一正则实例）
	STEP_HIGHLIGHT_REGEX.lastIndex = 0
	while ((match = STEP_HIGHLIGHT_REGEX.exec(raw)) !== null) {
		const start = match.index
		const end = start + match[0].length
		if (start > lastIndex) {
			segments.push({ text: raw.slice(lastIndex, start), highlight: false })
		}
		segments.push({ text: match[0], highlight: true })
		lastIndex = end
	}
	if (lastIndex < raw.length) {
		segments.push({ text: raw.slice(lastIndex), highlight: false })
	}
	return segments.length ? segments : [{ text: raw, highlight: false }]
}

// B2-6：步骤完成状态本地持久化 key 前缀，按 recipeId 隔离
const STEP_COMPLETED_STORAGE_PREFIX = 'recipe-step-done:'
const STEP_COMPLETED_STORAGE_VERSION = 2

function buildStepCompletedStorageKey(recipeId) {
	const id = String(recipeId || '').trim()
	return id ? `${STEP_COMPLETED_STORAGE_PREFIX}${id}` : ''
}

function buildComparableStepIdentity(step = {}) {
	const normalized = createStepDraftItem(step)
	return JSON.stringify({
		title: normalized.title.trim(),
		detail: normalized.detail.trim()
	})
}

function buildStepCompletionKeyList(steps = []) {
	const occurrenceMap = {}
	return normalizeParsedSteps(steps).map((step) => {
		const identity = buildComparableStepIdentity(step)
		const nextOccurrence = (occurrenceMap[identity] || 0) + 1
		occurrenceMap[identity] = nextOccurrence
		return `${identity}#${nextOccurrence}`
	})
}

function normalizeCompletedStepKeyMap(rawValue, currentStepKeys = []) {
	const allowedKeys = new Set(Array.isArray(currentStepKeys) ? currentStepKeys : [])
	const nextMap = {}
	const markCompleted = (value = '') => {
		const key = String(value || '').trim()
		if (!key || !allowedKeys.has(key)) return
		nextMap[key] = true
	}

	let payload = rawValue
	if (typeof payload === 'string' && payload) {
		try {
			payload = JSON.parse(payload)
		} catch (error) {
			return {}
		}
	}

	if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
		return {}
	}

	if (Number(payload.version) >= STEP_COMPLETED_STORAGE_VERSION && Array.isArray(payload.completedKeys)) {
		payload.completedKeys.forEach((value) => markCompleted(value))
		return nextMap
	}

	// 旧版「按步骤下标」存储在步骤改顺序/改内容后会把完成态错套到别的步骤上，直接丢弃更安全。
	return {}
}

function createCompletedStepStoragePayload(stepKeyMap = {}) {
	return {
		version: STEP_COMPLETED_STORAGE_VERSION,
		completedKeys: Object.keys(stepKeyMap).filter((key) => stepKeyMap[key])
	}
}

function requestImageInfo(src = '') {
	const target = String(src || '').trim()
	if (!target) return Promise.resolve(null)

	return new Promise((resolve, reject) => {
		uni.getImageInfo({
			src: target,
			success: resolve,
			fail: reject
		})
	})
}

function exportCanvasToTempFilePath(options = {}, component) {
	return new Promise((resolve, reject) => {
		uni.canvasToTempFilePath(
			{
				...options,
				success: resolve,
				fail: reject
			},
			component
		)
	})
}

export default {
	components: {
		ActionFeedback
	},
	data() {
		return {
			recipeId: '',
			recipe: null,
			showEditSheet: false,
			editDraft: createEmptyDraft(),
			maxRecipeImages: MAX_RECIPE_IMAGES,
			mealTabs: mealTypeOptions,
			statusTabs: statusOptions,
			isLoadingRecipe: false,
			isUploadingHeroImage: false,
			isSavingRecipe: false,
			isDeletingRecipe: false,
			isReparseSubmitting: false,
			isGeneratingFlowchart: false,
			isPinSubmitting: false,
			heroImageIndex: 0,
			editDraftSnapshot: '',
			parsePollingTimer: null,
			statusEstimateTimer: null,
			statusEstimateSyncedAt: 0,
			statusEstimateNow: 0,
			actionFeedbackVisible: false,
			actionFeedbackTone: '',
			actionFeedbackTitle: '',
			actionFeedbackDescription: '',
			actionFeedbackTick: 0,
			actionFeedbackTimer: null,
			hasResolvedInitialRecipeLoad: false,
			cachedRecipeImageMap: {},
			recipeImageFallbackMap: {},
			recipeImageHiddenMap: {},
			recipeImageCacheRequestID: 0,
			// P2-A：「做法」卡片当前激活的 Tab；'flowchart' | 'steps'
			// 默认值在 watch hasFlowchart / 初次加载时由 ensureCookingTabValid 校正
			activeCookingTab: 'flowchart',
			// B2-6：步骤完成状态，按「步骤内容签名」持久化，避免改顺序后串位
			completedStepKeyMap: {},
			// P2-D 分享路径升级：share_token 公开只读机制
			// shareToken：当前菜谱的永久 share_token（私有模式下后台 ensure，分享时拼到 path）
			// publicViewToken：从 onLoad query 读到的 shareToken，存在则强制走公开只读
			// isPublicView：是否处于公开只读模式（隐藏所有编辑入口、显示顶部 banner）
			// publicKitchenName / publicCreatorName：公开模式下 banner 显示的上下文
			// publicViewLoadFailed：公开拉取失败（token 失效 / 菜谱已删），显示兜底空态
			// showPublicReadOnlyExplain：banner 「了解」按钮控制的 popup 开关
			shareToken: '',
			// 进行中的 ensure share_token Promise（去重 + 供分享时 await 兜底）
			_shareTokenEnsurePromise: null,
			publicViewToken: '',
			isPublicView: false,
			publicKitchenName: '',
			publicCreatorName: '',
			publicViewLoadFailed: false,
			showPublicReadOnlyExplain: false,
			cachedFlowchartImagePath: '',
			flowchartImageCacheVersion: '',
			flowchartImageCacheRequestID: 0,
			flowchartSquareImagePath: '',
			flowchartSquareImageSourceKey: '',
			flowchartShareImagePendingKey: '',
			_flowchartShareImagePromise: null
		}
	},
	computed: {
		mealLabel() {
			return mealTypeLabelMap[this.recipe?.mealType] || '早餐'
		},
		// P2-D 分享路径升级：公开只读 banner 文案
		// 优先「来自 XX 的空间」；空间名缺失时退化为「来自他人的菜谱分享」
		publicBannerText() {
			const kitchen = String(this.publicKitchenName || '').trim()
			if (kitchen) return `来自「${kitchen}」的菜谱 · 加入空间可参与编辑`
			return '来自他人的菜谱分享 · 加入空间可参与编辑'
		},
		// 只读规则 popup 正文，按是否有创建者昵称差异化
		publicExplainBody() {
			const creator = String(this.publicCreatorName || '').trim()
			const owner = creator ? `「${creator}」` : '原作者'
			return `这道菜由${owner}整理，分享出来仅供查看。如果想一起编辑、调整步骤或补充心得，可以请对方把你加入空间。`
		},
		statusLabel() {
			return statusLabelMap[this.recipe?.status] || '想吃'
		},
		isPinned() {
			return !!String(this.recipe?.pinnedAt || '').trim()
		},
		parsedContentView() {
			return normalizeParsedContentView(this.recipe?.parsedContent || {})
		},
		parsedMainIngredients() {
			return this.parsedContentView.mainIngredients
		},
		parsedSecondaryIngredients() {
			return this.parsedContentView.secondaryIngredients
		},
		parsedSecondaryGroups() {
			const groups = []
			const supportingIngredients = this.parsedContentView.supportingIngredients || []
			const seasonings = this.parsedContentView.seasonings || []

			if (supportingIngredients.length) {
				groups.push({
					key: 'supporting',
					label: '配菜',
					text: supportingIngredients.join('、')
				})
			}

			if (seasonings.length) {
				groups.push({
					key: 'seasonings',
					label: '调味',
					text: seasonings.join('、')
				})
			}

			return groups
		},
		parsedSteps() {
			return this.parsedContentView.steps
		},
		editIngredientCount() {
			const mainCount = Array.isArray(this.editDraft.mainIngredients) ? this.editDraft.mainIngredients.length : 0
			const secondaryCount = Array.isArray(this.editDraft.secondaryIngredients) ? this.editDraft.secondaryIngredients.length : 0
			return mainCount + secondaryCount
		},
		editStepCount() {
			return Array.isArray(this.editDraft.steps) ? this.editDraft.steps.length : 0
		},
		editIsUsingFallbackContent() {
			return this.editDraft.parsedContentMode === 'fallback'
		},
		hasUnsavedEditChanges() {
			if (!this.showEditSheet) return false
			return this.editDraftSnapshot !== serializeComparableEditDraft(this.editDraft)
		},
		hasMeaningfulParsedContent() {
			return !isFallbackLikeParsedContent(this.recipe || {}, {
				mainIngredients: this.parsedMainIngredients,
				secondaryIngredients: this.parsedSecondaryIngredients,
				steps: this.parsedSteps
			})
		},
		hasManualParsedContentEdits() {
			return !!this.recipe?.parsedContentEdited
		},
		recipeImageVersion() {
			return buildRecipeImageVersion(this.recipe || {})
		},
		recipeImages() {
			if (Array.isArray(this.recipe?.imageUrls) && this.recipe.imageUrls.length) {
				return this.recipe.imageUrls.filter(Boolean)
			}
			const fallbackImage = String(this.recipe?.image || this.recipe?.imageUrl || '').trim()
			return fallbackImage ? [fallbackImage] : []
		},
		displayRecipeLink() {
			const rawLink = String(this.recipe?.link || '').trim()
			return extractCopyableLink(rawLink) || rawLink
		},
		displayRecipeImages() {
			const version = this.recipeImageVersion
			return this.recipeImages
				.map((remoteURL) => {
					const cacheKey = buildImageCacheKey(remoteURL, version)
					if (this.recipeImageHiddenMap[cacheKey]) return null

					const cachedURL = String(this.cachedRecipeImageMap[cacheKey] || '').trim()
					return {
						cacheKey,
						remoteURL,
						displayURL: this.recipeImageFallbackMap[cacheKey] ? remoteURL : (cachedURL || remoteURL)
					}
				})
				.filter(Boolean)
		},
		visibleRecipeSourceImages() {
			const version = this.recipeImageVersion
			return this.recipeImages.filter((remoteURL) => {
				const cacheKey = buildImageCacheKey(remoteURL, version)
				return !this.recipeImageHiddenMap[cacheKey]
			})
		},
		flowchartImageUrl() {
			return String(this.recipe?.flowchartImageUrl || '').trim()
		},
		flowchartDisplayImageUrl() {
			return String(this.cachedFlowchartImagePath || '').trim() || this.flowchartImageUrl
		},
		flowchartStatusValue() {
			return String(this.recipe?.flowchartStatus || '').trim()
		},
		hasFlowchart() {
			return !!this.flowchartImageUrl
		},
		canGenerateFlowchart() {
			return this.hasMeaningfulParsedContent && this.parsedSteps.length >= 3
		},
		canRequestFlowchart() {
			return this.canGenerateFlowchart && !ACTIVE_FLOWCHART_STATUSES.includes(this.flowchartStatusValue)
		},
		isFlowchartActive() {
			// 后台正在生成 / 排队中
			return ACTIVE_FLOWCHART_STATUSES.includes(this.flowchartStatusValue)
		},
		flowchartActionText() {
			if (this.isGeneratingFlowchart) return '提交中...'
			if (ACTIVE_FLOWCHART_STATUSES.includes(this.flowchartStatusValue)) return '生成中...'
			return this.hasFlowchart ? '重新生成' : '生成步骤图'
		},
		flowchartEmptyText() {
			if (this.canGenerateFlowchart) {
				return '生成后会把关键步骤整理成一张图，先看懂再下厨。'
			}
			return '先补充至少 3 个关键步骤，再生成步骤图。'
		},
		flowchartStatusMeta() {
			const status = this.flowchartStatusValue
			if (!status || status === 'done') return null
			return flowchartStatusMetaMap[status] || null
		},
		flowchartStatusDescription() {
			if (!this.flowchartStatusMeta) return ''
			const errorMessage = String(this.recipe?.flowchartError || '').trim()
			if (this.flowchartStatusValue === 'failed' && errorMessage) {
				return errorMessage
			}
			const waitHint = buildFlowchartWaitHint(this.flowchartStatusValue, this.flowchartQueueAhead, this.flowchartEstimatedWaitSeconds)
			if (waitHint) {
				return waitHint
			}
			return this.flowchartStatusMeta.description
		},
		flowchartQueueAhead() {
			return toPositiveInteger(this.recipe?.flowchartQueueAhead || 0)
		},
		flowchartEstimatedWaitSeconds() {
			return resolveRemainingWaitSeconds(
				this.recipe?.flowchartEstimatedWaitSeconds || 0,
				this.statusEstimateSyncedAt,
				this.statusEstimateNow
			)
		},
		showFlowchartStaleHint() {
			return this.hasFlowchart && !!this.recipe?.flowchartStale
		},
		flowchartUpdatedAtText() {
			const value = formatDateTime(this.recipe?.flowchartUpdatedAt || '')
			return value ? `已生成：${value}` : ''
		},
		flowchartCaptionText() {
			const raw = String(this.recipe?.flowchartUpdatedAt || '').trim()
			if (!raw) return ''
			// 仅取月-日，作为卡片底部一行 caption（完整时间放到「查看生成详情」里）
			const date = new Date(raw)
			if (Number.isNaN(date.getTime())) {
				return 'AI 生成'
			}
			const mm = String(date.getMonth() + 1).padStart(2, '0')
			const dd = String(date.getDate()).padStart(2, '0')
			return `AI 生成 · ${mm}-${dd}`
		},
		flowchartModelTip() {
			const model = String(this.recipe?.flowchartModel || '').trim()
			return model ? `由 ${model} 生成` : ''
		},
		parseStatusValue() {
			return String(this.recipe?.parseStatus || '').trim()
		},
		parseStatusMeta() {
			const status = this.parseStatusValue
			if (status && parseStatusMetaMap[status]) {
				return parseStatusMetaMap[status]
			}
			if (this.isAutoParseRecipe) {
				return parseStatusMetaMap.idle
			}
			return null
		},
		parseStatusDescription() {
			if (!this.parseStatusMeta) return ''
			const errorMessage = String(this.recipe?.parseError || '').trim()
			if (this.parseStatusValue === 'failed' && errorMessage) {
				return errorMessage
			}
			if (this.parseStatusValue === 'done' && errorMessage && String(this.recipe?.parseSource || '').toLowerCase().includes('heuristic')) {
				return errorMessage
			}
			const waitHint = buildParseWaitHint(this.parseStatusValue, this.parseQueueAhead, this.parseEstimatedWaitSeconds)
			if (waitHint) {
				return waitHint
			}
			const resultHint = buildParseResultHint(this.parseStatusValue, this.recipe?.parseSource || '')
			if (resultHint) {
				return resultHint
			}
			return this.parseStatusMeta.description
		},
		parseQueueAhead() {
			return toPositiveInteger(this.recipe?.parseQueueAhead || 0)
		},
		parseEstimatedWaitSeconds() {
			return resolveRemainingWaitSeconds(
				this.recipe?.parseEstimatedWaitSeconds || 0,
				this.statusEstimateSyncedAt,
				this.statusEstimateNow
			)
		},
		parseStatusSourceLabel() {
			return formatParseSourceLabel(this.recipe?.parseSource || '')
		},
		isAutoParseRecipe() {
			return isAutoParseSupportedLink(this.recipe?.link || '')
		},
		canRequestParse() {
			return this.isAutoParseRecipe && !ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)
		},
		needsParseOverwriteConfirm() {
			// 仅在「手动改过」时才走覆盖警告；纯 AI 结果走下方的「轻确认」分支
			return this.hasManualParsedContentEdits
		},
		parseOverwriteModalContent() {
			if (this.hasManualParsedContentEdits) {
				return '你手动修改过食材或制作步骤，重新整理后可能会覆盖这些内容。'
			}
			return '将根据来源链接更新当前食材和步骤。'
		},
		parseActionText() {
			if (this.isReparseSubmitting) return '整理中...'
			if (!this.parseStatusValue) return '开始整理'
			if (this.parseStatusValue === 'failed') return '再试一次'
			return '重新整理'
		},
		pinActionText() {
			if (this.isPinSubmitting) return '处理中...'
			return this.isPinned ? '取消置顶' : '置顶'
		},
		canSaveEditDraft() {
			return !!this.editDraft.title.trim()
		},
		showRecipeLoadingState() {
			return !this.recipe && (!this.hasResolvedInitialRecipeLoad || this.isLoadingRecipe)
		},
		actionFeedbackKey() {
			return `${this.actionFeedbackTone || 'idle'}:${this.actionFeedbackTick}`
		},
		// ===== P2-A：「做法」合并卡片相关 =====
		// 是否处于后台异步进行中（一图生成 or 步骤整理），用于把右上 ⋯ 替换为 chip
		isCookingActive() {
			return this.isFlowchartActive || ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)
		},
		// chip 文案，根据当前激活的 Tab 选择对应任务的标签
		cookingActiveLabel() {
			if (this.activeCookingTab === 'flowchart' && this.isFlowchartActive) {
				return '生成中…'
			}
			if (this.activeCookingTab === 'steps' && ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)) {
				return '整理中…'
			}
			// 跨 Tab 的兜底：另一 Tab 在跑也提示用户
			if (this.isFlowchartActive) return '一图生成中…'
			if (ACTIVE_PARSE_STATUSES.includes(this.parseStatusValue)) return '步骤整理中…'
			return ''
		},
		// 统一 ⋯ 菜单是否至少有一个可执行项；为空则隐藏入口
		hasCookingMenuItems() {
			if (this.canRequestFlowchart && !this.isFlowchartActive) return true
			if (this.canRequestParse) return true
			if (this.flowchartUpdatedAtText || this.flowchartModelTip) return true
			if (this.parseStatusSourceLabel) return true
			return false
		},
		showCookingFlowchartView() {
			return this.hasFlowchart && this.activeCookingTab === 'flowchart'
		},
		showCookingStepsView() {
			return !this.hasFlowchart || this.activeCookingTab === 'steps'
		},
		// 卡片底部一行 caption：根据当前 Tab 选择对应来源
		cookingFooterText() {
			if (this.showCookingFlowchartView) {
				return this.flowchartCaptionText || ''
			}
			if (this.showCookingStepsView) {
				return this.parseStatusSourceLabel || ''
			}
			return ''
		},
		// B1-2：是否可复制食材清单（至少要有 1 项主料或 1 个辅料分组）
		canCopyIngredientList() {
			return this.parsedMainIngredients.length > 0 || this.parsedSecondaryGroups.length > 0
		},
		// B2-6：已完成步骤数（用于「2 / 4」进度提示）
		completedStepCount() {
			const stepKeys = this.buildCurrentStepCompletionKeys()
			if (!stepKeys.length) return 0
			let count = 0
			for (let i = 0; i < stepKeys.length; i += 1) {
				if (this.completedStepKeyMap[stepKeys[i]]) count += 1
			}
			return count
		},
		// ===== Hero 操作菜单：当前位置图片是否可设为封面 =====
		canSetCurrentAsCover() {
			return this.recipeImages.length > 1 && this.resolveOriginalImageIndex(this.heroImageIndex) > 0
		},
		// 是否可以再添加图片（未到上限）
		canAddMoreHeroImages() {
			return this.recipeImages.length > 0 && this.visibleRecipeSourceImages.length < this.maxRecipeImages
		},
		// 是否可以删除当前图片（至少要有 1 张存在）
		canDeleteCurrentImage() {
			return this.resolveOriginalImageIndex(this.heroImageIndex) >= 0
		},
		// 是否显示 Hero ⋯ 按钮：上传中隐藏；菜单全空（理论上 length > 0 至少有「删除」）也隐藏
		canShowHeroActionMenu() {
			if (this.isUploadingHeroImage) return false
			if (!this.displayRecipeImages.length) {
				return this.canAddMoreHeroImages
			}
			return this.canSetCurrentAsCover || this.canAddMoreHeroImages || this.canDeleteCurrentImage
		}
	},
	watch: {
		// P2-A：当流程图从「有」变「无」（如生成失败被清空）且当前正停留在「一图」Tab，
		// 自动回退到「详细步骤」Tab，避免显示空内容
		hasFlowchart() {
			this.ensureCookingTabValid()
		},
		// P2-A 修复：流程图任务终止（active -> idle/failed）后也要回退一次
		isFlowchartActive() {
			this.ensureCookingTabValid()
		}
	},
	onLoad(options) {
		this.recipeId = options?.id || ''
		// P2-D 分享路径升级：query 带 shareToken 即视为公开只读访问
		// 不论是否同空间成员，统一走公开只读，避免拉起登录、避免编辑误触
		const shareToken = String(options?.shareToken || '').trim()
		if (shareToken) {
			this.publicViewToken = shareToken
			this.isPublicView = true
			// P1 修复：把入参 shareToken 同步写到 this.shareToken
			// 否则公开模式下二次转发时 buildRecipeShareConfig 拿不到 token，发出去的链接会断在第二跳
			this.shareToken = shareToken
		}
	},
	onShow() {
		this.loadRecipe()
	},
	onHide() {
		this.stopParsePolling()
		this.clearActionFeedback()
	},
	onUnload() {
		this.stopParsePolling()
		this.clearActionFeedback()
	},
	onBackPress() {
		if (!this.showEditSheet) return false
		this.requestCloseEditSheet()
		return true
	},
	// P2-D 分享路径升级：开启微信原生右上角胶囊菜单的「转发 / 分享到朋友圈 / 收藏」三项能力
	// 只要定义这三个生命周期函数，对应菜单项就会出现，无需 UI 改动
	// P2 修复：分享窗口期兜底
	//   - 若 token 已就绪：同步返回完整 config（含 shareToken）
	//   - 若 token 还在 ensure 中：通过 promise 字段（基础库 2.12.0+）等 token 到位再返回
	//   - 微信侧 promise 超时（约 5s）会回退使用同步返回的兜底 config（不带 token，行为退化为旧版鉴权墙）
	onShareAppMessage(res) {
		const fallback = this.buildRecipeShareConfig({ from: res?.from, channel: 'message' })
		const needsShareToken = !this.shareToken && !this.isPublicView && !!this.recipeId
		const needsFlowchartShareCover =
			this.shouldPreferFlowchartShareCover('message') && this.hasFlowchart && !this.getCurrentFlowchartShareImagePath()
		if (!needsShareToken && !needsFlowchartShareCover) return fallback
		return {
			...fallback,
			promise: this.buildRecipeShareConfigAsync({ from: res?.from, channel: 'message' })
				.catch(() => fallback)
		}
	},
	onShareTimeline() {
		const fallback = this.buildRecipeShareConfig({ channel: 'timeline' })
		if (this.shareToken || this.isPublicView || !this.recipeId) return fallback
		// 朋友圈 promise 字段需基础库 3.12.0+，老版本会忽略 promise，自动回退到同步配置
		return {
			...fallback,
			promise: this.ensureShareTokenIfNeeded()
				.then(() => this.buildRecipeShareConfig({ channel: 'timeline' }))
				.catch(() => fallback)
		}
	},
	onAddToFavorites() {
		// 收藏夹接口不支持 promise 字段，只能同步返回当前最佳配置。
		// 若这次优先走流程图封面，则在真正触发分享时后台懒裁一份供后续复用。
		if (this.shouldPreferFlowchartShareCover('favorite') && this.hasFlowchart && !this.getCurrentFlowchartShareImagePath()) {
			this.ensureFlowchartShareImagePath().catch(() => {})
		}
		return this.buildRecipeShareConfig({ channel: 'favorite' })
	},
	methods: {
		clearActionFeedbackTimer() {
			if (!this.actionFeedbackTimer) return
			clearTimeout(this.actionFeedbackTimer)
			this.actionFeedbackTimer = null
		},
		clearActionFeedback() {
			this.clearActionFeedbackTimer()
			this.actionFeedbackVisible = false
			this.actionFeedbackTone = ''
			this.actionFeedbackTitle = ''
			this.actionFeedbackDescription = ''
		},
		showActionFeedback(options = {}) {
			const title = String(options?.title || '').trim()
			if (!title) return
			this.clearActionFeedbackTimer()
			this.actionFeedbackTone = String(options?.tone || 'done').trim() || 'done'
			this.actionFeedbackTitle = title
			this.actionFeedbackDescription = String(options?.description || '').trim()
			this.actionFeedbackVisible = true
			this.actionFeedbackTick += 1
			this.actionFeedbackTimer = setTimeout(() => {
				this.actionFeedbackVisible = false
				this.actionFeedbackTimer = null
			}, Math.max(1200, Number(options?.duration) || 1680))
		},
		async loadRecipe() {
			// P2-D 分享路径升级：公开只读模式优先走 share_token 公开接口
			// 不进缓存（避免污染同 id 的私有缓存）、不触发 ensureSession
			if (this.isPublicView && this.publicViewToken) {
				try {
					this.isLoadingRecipe = true
					const view = await fetchPublicRecipeByShareToken(this.publicViewToken)
					if (!view || !view.recipe) {
						this.recipe = null
						this.publicViewLoadFailed = true
						this.hasResolvedInitialRecipeLoad = true
						return
					}
					this.recipeId = view.recipe.id || this.recipeId
					this.publicKitchenName = view.kitchenName || ''
					this.publicCreatorName = view.creatorName || ''
					this.publicViewLoadFailed = false
					this.applyRecipe(view.recipe)
					this.hasResolvedInitialRecipeLoad = true
				} catch (error) {
					this.recipe = null
					this.publicViewLoadFailed = true
					this.hasResolvedInitialRecipeLoad = true
				} finally {
					this.isLoadingRecipe = false
				}
				return
			}

			if (!this.recipeId) {
				this.recipe = null
				this.hasResolvedInitialRecipeLoad = true
				return
			}

			// P2 修复：进页就立刻 fire ensure share_token（与缓存读取并行）
			// 缩短「打开详情秒分享」窗口，applyRecipe 末尾的 ensure 作为兜底
			this.ensureShareTokenIfNeeded()

			const cachedRecipe = getCachedRecipeById(this.recipeId)
			if (cachedRecipe) {
				this.applyRecipe(cachedRecipe)
				this.hasResolvedInitialRecipeLoad = true
			}

			try {
				this.isLoadingRecipe = true
				const recipe = await getRecipeById(this.recipeId, { preferCache: !cachedRecipe })
				this.applyRecipe(recipe)
				this.hasResolvedInitialRecipeLoad = true
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
				this.hasResolvedInitialRecipeLoad = true
			}
		},
		buildFlowchartImageCacheVersion(recipe = this.recipe) {
			const updatedAt = String(recipe?.flowchartUpdatedAt || '').trim()
			if (updatedAt) return updatedAt
			return String(recipe?.flowchartImageUrl || '').trim()
		},
		buildFlowchartImageCacheEntry(recipe = this.recipe) {
			const url = String(recipe?.flowchartImageUrl || '').trim()
			const version = this.buildFlowchartImageCacheVersion(recipe)
			return {
				url,
				version,
				cacheKey: buildImageCacheKey(url, version)
			}
		},
		buildFlowchartPreviewURLs() {
			const urls = []
			const appendURL = (value = '') => {
				const target = String(value || '').trim()
				if (!target || urls.includes(target)) return
				urls.push(target)
			}
			appendURL(this.cachedFlowchartImagePath)
			appendURL(this.flowchartImageUrl)
			return urls
		},
		shouldPreferFlowchartShareCover(channel = 'message') {
			return channel === 'message' || channel === 'favorite'
		},
		getCurrentFlowchartShareImagePath() {
			const currentKey = this.buildFlowchartImageCacheEntry().cacheKey
			if (!currentKey || this.flowchartSquareImageSourceKey !== currentKey) return ''
			return String(this.flowchartSquareImagePath || '').trim()
		},
		async buildRecipeShareConfigAsync({ channel = 'message' } = {}) {
			const shouldEnsureToken = !this.shareToken && !this.isPublicView && !!this.recipeId
			const flowchartShareImageTask =
				this.shouldPreferFlowchartShareCover(channel) && this.hasFlowchart
					? this.ensureFlowchartShareImagePath().catch(() => '')
					: Promise.resolve('')

			const [, flowchartShareImage] = await Promise.all([
				shouldEnsureToken ? this.ensureShareTokenIfNeeded() : Promise.resolve(this.shareToken || this.publicViewToken || ''),
				flowchartShareImageTask
			])

			return this.buildRecipeShareConfig({
				channel,
				flowchartShareImage
			})
		},
		async ensureFlowchartShareImagePath() {
			const entry = this.buildFlowchartImageCacheEntry()
			if (!entry.url) return ''

			const readyPath = this.getCurrentFlowchartShareImagePath()
			if (readyPath) return readyPath

			if (this._flowchartShareImagePromise && this.flowchartShareImagePendingKey === entry.cacheKey) {
				return this._flowchartShareImagePromise
			}

			this.flowchartShareImagePendingKey = entry.cacheKey
			const shareImagePromise = this.createFlowchartSquareImage()
				.then((tempFilePath) => {
					if (!tempFilePath) return ''
					if (this.buildFlowchartImageCacheEntry().cacheKey !== entry.cacheKey) return ''
					this.flowchartSquareImagePath = tempFilePath
					this.flowchartSquareImageSourceKey = entry.cacheKey
					return tempFilePath
				})
				.catch(() => '')
				.finally(() => {
					if (this.flowchartShareImagePendingKey === entry.cacheKey) {
						this.flowchartShareImagePendingKey = ''
					}
					if (this._flowchartShareImagePromise === shareImagePromise) {
						this._flowchartShareImagePromise = null
					}
				})

			this._flowchartShareImagePromise = shareImagePromise
			return this._flowchartShareImagePromise
		},
		async createFlowchartSquareImage() {
			if (typeof uni.createCanvasContext !== 'function') return ''

			const sourceURLs = this.buildFlowchartPreviewURLs()
			if (!sourceURLs.length) return ''

			const canvasId = '_flowchart_square_canvas'
			for (let index = 0; index < sourceURLs.length; index += 1) {
				const sourceURL = sourceURLs[index]
				try {
					const imageInfo = await requestImageInfo(sourceURL)
					const sourcePath = String(imageInfo?.path || sourceURL || '').trim()
					const width = Number(imageInfo?.width) || 0
					const height = Number(imageInfo?.height) || 0
					const size = Math.min(width, height)
					if (!sourcePath || !size) continue

					const offsetX = width > size ? -(width - size) / 2 : 0
					const offsetY = height > size ? -(height - size) / 2 : 0
					const ctx = uni.createCanvasContext(canvasId, this)
					if (typeof ctx.clearRect === 'function') {
						ctx.clearRect(0, 0, size, size)
					}
					ctx.drawImage(sourcePath, offsetX, offsetY, width, height)
					const tempFilePath = await new Promise((resolve) => {
						ctx.draw(false, async () => {
							try {
								const result = await exportCanvasToTempFilePath(
									{
										canvasId,
										width: size,
										height: size,
										destWidth: size,
										destHeight: size
									},
									this
								)
								resolve(String(result?.tempFilePath || '').trim())
							} catch (error) {
								resolve('')
							}
						})
					})
					if (tempFilePath) return tempFilePath
				} catch (error) {
					const currentLocalPath = String(this.cachedFlowchartImagePath || '').trim()
					if (sourceURL === currentLocalPath) {
						this.cachedFlowchartImagePath = ''
						try {
							await invalidateCachedImage(this.flowchartImageUrl, this.flowchartImageCacheVersion || this.buildFlowchartImageCacheVersion())
						} catch (invalidateError) {
							// Ignore stale cache cleanup failures and continue with remote fallback.
						}
					}
				}
			}

			return ''
		},
		// P2-D 分享路径升级：统一构造分享配置
		// channel: 'message' (微信好友) | 'timeline' (朋友圈) | 'favorite' (收藏)
		// 文案策略（简洁派，让封面图说话）：
		//   - message  → 「{菜名} · 完整做法」（有流程图或 ≥3 步）/「{菜名}」（兜底）
		//   - timeline → 「{菜名}」（朋友圈最克制，封面承担表达）
		//   - favorite → 「{菜名}」（收藏夹清单识别）
		// 封面策略（差异化）：
		//   - message  → 优先流程图，缺则首图：转发场景=「教你做菜」
		//   - timeline → 优先首图，缺则流程图：朋友圈是炫耀场，成品图传播力更强
		//   - favorite → 优先流程图，缺则首图：收藏=「以后要用」
		// 微信会按各渠道比例（5:4 / 1:1）自适应裁切
		buildRecipeShareConfig({ channel = 'message', flowchartShareImage = '' } = {}) {
			const recipe = this.recipe || {}
			const rawTitle = String(recipe.title || '').trim()
			const dishName = rawTitle || '一道值得做的菜'
			// 标题：朋友圈/收藏只用菜名；转发若有「完整做法」价值锚点（流程图或多步骤）则附加
			let title = dishName
			if (channel === 'message') {
				const hasFullRecipe = this.hasFlowchart || (Array.isArray(this.parsedSteps) && this.parsedSteps.length >= 3)
				if (hasFullRecipe) title = `${dishName} · 完整做法`
			}
			// path 带 from=share 留作埋点 / 拉新归因，朋友圈只取 query 部分
			// P2-D 分享路径升级：若已 ensured share_token，则附带，让接收者可公开只读
			// P1 修复：双保险——优先 shareToken，缺则回退 publicViewToken（公开模式下二次转发兜底）
			const recipeId = String(recipe.id || this.recipeId || '').trim()
			const effectiveToken = String(this.shareToken || this.publicViewToken || '').trim()
			const tokenSegment = effectiveToken ? `&shareToken=${effectiveToken}` : ''
			const path = recipeId ? `/pages/recipe-detail/index?id=${recipeId}&from=share${tokenSegment}` : '/pages/index/index'
			const query = recipeId ? `id=${recipeId}&from=share${tokenSegment}` : ''
			// 按渠道选封面：朋友圈优先成品首图，其余优先流程图
			// 注意用 visibleRecipeSourceImages（已过滤掉加载失败被 recipeImageHiddenMap 标记的坏图），
			// 避免把页面已知失效的 URL 发给微信做封面
			const flowchart = String(flowchartShareImage || this.getCurrentFlowchartShareImagePath() || this.flowchartImageUrl || '').trim()
			const timelineFlowchart = String(this.flowchartImageUrl || '').trim()
			const coverImage = String(this.visibleRecipeSourceImages?.[0] || '').trim()
			const shareImage = channel === 'timeline'
				? (coverImage || timelineFlowchart)
				: (flowchart || coverImage)

			if (channel === 'timeline') {
				const config = { title, query }
				if (shareImage) config.imageUrl = shareImage
				return config
			}
			if (channel === 'favorite') {
				const config = { title, query }
				if (shareImage) config.imageUrl = shareImage
				return config
			}
			// channel === 'message'
			const config = { title, path }
			if (shareImage) config.imageUrl = shareImage
			return config
		},
		applyRecipe(recipe) {
			const previousFlowchartCacheKey = this.buildFlowchartImageCacheEntry(this.recipe).cacheKey
			const nextFlowchartCacheKey = this.buildFlowchartImageCacheEntry(recipe).cacheKey
			if (previousFlowchartCacheKey !== nextFlowchartCacheKey) {
				this.flowchartSquareImagePath = ''
				this.flowchartSquareImageSourceKey = ''
				this.flowchartShareImagePendingKey = ''
			}
			this.recipe = recipe
			const now = Date.now()
			this.statusEstimateSyncedAt = now
			this.statusEstimateNow = now
			this.syncRecipeImageCache(recipe)
			this.syncFlowchartImageCache(recipe, {
				previousCacheKey: previousFlowchartCacheKey
			})
			// B2-6：每次加载菜谱时同步读取本地步骤完成进度
			this.loadCompletedSteps()
			if (this.heroImageIndex >= this.displayRecipeImages.length) {
				this.heroImageIndex = 0
			}
			// P2-A 修复：菜谱数据落定后再校正一次「做法」Tab，避免无图时初次加载落到空态
			this.ensureCookingTabValid()
			if (this.recipe?.title) {
				uni.setNavigationBarTitle({
					title: this.recipe.title
				})
			}
			this.syncParsePolling()
			// P2-D 分享路径升级：私有模式下后台静默 ensure share_token
			// 不阻塞渲染、失败静默；分享时直接用 this.shareToken 拼 path
			this.ensureShareTokenIfNeeded()
		},
		// 后台幂等获取菜谱永久 share_token，仅私有模式且尚未拿到时触发
		// 失败不打扰用户，分享路径会兜底为不带 token 的链接（行为退化为旧版）
		// P2 修复：返回 Promise<string|null>，供 onShareAppMessage 在 token 未就绪时 await 兜底
		ensureShareTokenIfNeeded() {
			if (this.isPublicView) return Promise.resolve(this.shareToken || null)
			if (this.shareToken) return Promise.resolve(this.shareToken)
			const recipeId = String(this.recipe?.id || this.recipeId || '').trim()
			if (!recipeId) return Promise.resolve(null)
			// 复用进行中的 ensure 任务，避免重复请求
			if (this._shareTokenEnsurePromise) return this._shareTokenEnsurePromise
			this._shareTokenEnsurePromise = ensureRecipeShareTokenById(recipeId)
				.then((token) => {
					if (token) this.shareToken = token
					return token || null
				})
				.catch(() => null)
				.finally(() => {
					this._shareTokenEnsurePromise = null
				})
			return this._shareTokenEnsurePromise
		},
		// P2-A 修复：当流程图既不可用也未在生成时，强制回退到「详细步骤」Tab
		ensureCookingTabValid() {
			if (this.activeCookingTab !== 'flowchart' && this.activeCookingTab !== 'steps') {
				this.activeCookingTab = this.hasFlowchart ? 'flowchart' : 'steps'
				return
			}
			if (!this.hasFlowchart && this.activeCookingTab !== 'steps') {
				this.activeCookingTab = 'steps'
			}
		},
		buildRecipeImageCacheEntries(recipe = this.recipe) {
			const source = recipe || {}
			const images =
				Array.isArray(source.imageUrls) && source.imageUrls.length
					? source.imageUrls.filter(Boolean)
					: [source.image, source.imageUrl].filter(Boolean)
			const version = buildRecipeImageVersion(source)
			return images.map((url) => ({
				url: String(url || '').trim(),
				version,
				cacheKey: buildImageCacheKey(url, version)
			})).filter((entry) => entry.url)
		},
		async syncFlowchartImageCache(recipe = this.recipe, options = {}) {
			const entry = this.buildFlowchartImageCacheEntry(recipe)
			const requestID = this.flowchartImageCacheRequestID + 1
			const previousCacheKey = String(options.previousCacheKey || '').trim()

			this.flowchartImageCacheRequestID = requestID
			this.flowchartImageCacheVersion = entry.version

			if (entry.cacheKey !== previousCacheKey) {
				this.cachedFlowchartImagePath = ''
			}

			if (!entry.url) {
				this.cachedFlowchartImagePath = ''
				this.flowchartImageCacheVersion = ''
				return
			}

			const localPath = await getCachedImagePath(entry.url, entry.version)
			if (requestID !== this.flowchartImageCacheRequestID) return

			if (localPath) {
				this.cachedFlowchartImagePath = localPath
				return
			}

			this.cachedFlowchartImagePath = ''
			warmImageCache([entry], {
				concurrency: 1,
				onResolved: ({ localPath: resolvedPath }) => {
					if (requestID !== this.flowchartImageCacheRequestID || !resolvedPath) return
					if (this.buildFlowchartImageCacheEntry().cacheKey !== entry.cacheKey) return
					this.cachedFlowchartImagePath = resolvedPath
				}
			})
		},
		async syncRecipeImageCache(recipe = this.recipe) {
			const entries = this.buildRecipeImageCacheEntries(recipe)
			const requestID = this.recipeImageCacheRequestID + 1
			this.recipeImageCacheRequestID = requestID
			this.cachedRecipeImageMap = {}
			this.recipeImageFallbackMap = {}
			this.recipeImageHiddenMap = {}

			if (!entries.length) {
				return
			}

			const cachedEntries = await Promise.all(
				entries.map(async (entry) => ({
					cacheKey: entry.cacheKey,
					localPath: await getCachedImagePath(entry.url, entry.version)
				}))
			)

			if (requestID !== this.recipeImageCacheRequestID) return

			const nextImageMap = {}
			cachedEntries.forEach((entry) => {
				if (!entry.localPath) return
				nextImageMap[entry.cacheKey] = entry.localPath
			})
			this.cachedRecipeImageMap = nextImageMap

			warmImageCache(entries, {
				concurrency: 2,
				onResolved: ({ cacheKey, localPath }) => {
					if (requestID !== this.recipeImageCacheRequestID || !localPath) return
					if (this.cachedRecipeImageMap[cacheKey] === localPath) return
					this.cachedRecipeImageMap = {
						...this.cachedRecipeImageMap,
						[cacheKey]: localPath
					}
				}
			})
		},
		syncParsePolling() {
			const parseStatus = String(this.recipe?.parseStatus || '').trim()
			const flowchartStatus = String(this.recipe?.flowchartStatus || '').trim()
			if (!ACTIVE_PARSE_STATUSES.includes(parseStatus) && !ACTIVE_FLOWCHART_STATUSES.includes(flowchartStatus)) {
				this.stopParsePolling()
				return
			}

			this.syncStatusEstimateTimer()

			if (this.parsePollingTimer) return

			this.parsePollingTimer = setInterval(() => {
				this.refreshParseStatus()
			}, 4000)
		},
		syncStatusEstimateTimer() {
			if (!this.parseEstimatedWaitSeconds && !this.flowchartEstimatedWaitSeconds) {
				this.stopStatusEstimateTimer()
				return
			}
			if (this.statusEstimateTimer) return
			this.statusEstimateTimer = setInterval(() => {
				this.statusEstimateNow = Date.now()
			}, 1000)
		},
		stopParsePolling() {
			if (this.parsePollingTimer) {
				clearInterval(this.parsePollingTimer)
				this.parsePollingTimer = null
			}
			this.stopStatusEstimateTimer()
		},
		stopStatusEstimateTimer() {
			if (!this.statusEstimateTimer) return
			clearInterval(this.statusEstimateTimer)
			this.statusEstimateTimer = null
		},
		async refreshParseStatus() {
			if (!this.recipeId || this.isLoadingRecipe || this.isSavingRecipe || this.isDeletingRecipe || this.isReparseSubmitting || this.isGeneratingFlowchart || this.isPinSubmitting) {
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
			const parsedContentView = normalizeParsedContentView(recipe.parsedContent || {})
			const hasStructuredContent = !isFallbackLikeParsedContent(recipe, recipe.parsedContent || {})
			return createEmptyDraft({
				title: recipe.title || '',
				ingredient: recipe.ingredient || '',
				link: recipe.link || '',
				images:
					Array.isArray(recipe.imageUrls) && recipe.imageUrls.length
						? [...recipe.imageUrls]
						: recipe.image
							? [recipe.image]
							: [],
				mealType: recipe.mealType || 'breakfast',
				status: recipe.status || 'wishlist',
				mainIngredients: hasStructuredContent ? createIngredientDraftList(parsedContentView.mainIngredients) : [],
				secondaryIngredients: hasStructuredContent ? createIngredientDraftList(parsedContentView.secondaryIngredients) : [],
				steps: hasStructuredContent ? cloneStepDraftList(parsedContentView.steps) : [],
				parsedContentMode: hasStructuredContent ? 'existing' : 'fallback',
				note: recipe.note || ''
			})
		},
		openEditSheet() {
			// P1 修复：公开只读模式禁止打开编辑面板
			if (this.isPublicView) return
			if (!this.recipe) return
			const draft = this.createDraftFromRecipe(this.recipe)
			this.editDraft = draft
			this.editDraftSnapshot = serializeComparableEditDraft(draft)
			this.showEditSheet = true
		},
		handleHeroCardTap() {
			if (!this.recipe) return
			if (this.displayRecipeImages.length) {
				this.previewRecipeImage()
				return
			}
			this.chooseHeroImages()
		},
		handleHeroSwiperChange(event) {
			this.heroImageIndex = Number(event?.detail?.current) || 0
		},
		async handleRecipeImageError(image = {}) {
			const remoteURL = String(image?.remoteURL || '').trim()
			if (!remoteURL) return

			const version = this.recipeImageVersion
			const cacheKey = String(image?.cacheKey || buildImageCacheKey(remoteURL, version)).trim()
			const displayedURL = String(image?.displayURL || '').trim()
			const cachedURL = String(this.cachedRecipeImageMap[cacheKey] || '').trim()

			if (
				cachedURL &&
				displayedURL === cachedURL &&
				cachedURL !== remoteURL &&
				!this.recipeImageFallbackMap[cacheKey]
			) {
				this.recipeImageFallbackMap = {
					...this.recipeImageFallbackMap,
					[cacheKey]: true
				}

				if (this.cachedRecipeImageMap[cacheKey]) {
					const nextImageMap = { ...this.cachedRecipeImageMap }
					delete nextImageMap[cacheKey]
					this.cachedRecipeImageMap = nextImageMap
				}

				try {
					await invalidateCachedImage(remoteURL, version)
				} catch (error) {
					// Ignore cache cleanup failures and keep the remote fallback usable.
				}
				return
			}

			if (this.recipeImageHiddenMap[cacheKey]) return
			this.recipeImageHiddenMap = {
				...this.recipeImageHiddenMap,
				[cacheKey]: true
			}
			this.heroImageIndex = 0
		},
		handleEditSheetPopupClose() {
			if (!this.showEditSheet) return
			this.requestCloseEditSheet()
		},
		resetEditDraftState() {
			this.editDraft = createEmptyDraft()
			this.editDraftSnapshot = ''
		},
		closeEditSheet() {
			this.showEditSheet = false
			this.resetEditDraftState()
		},
		requestCloseEditSheet() {
			if (!this.showEditSheet || this.isSavingRecipe) return
			if (!this.hasUnsavedEditChanges) {
				this.closeEditSheet()
				return
			}

			uni.showModal({
				title: '放弃当前修改？',
				content: '未保存的食材、步骤和备注改动会丢失。',
				cancelText: '继续编辑',
				confirmText: '放弃修改',
				confirmColor: '#b4664c',
				success: ({ confirm }) => {
					if (!confirm) return
					this.closeEditSheet()
				}
			})
		},
		chooseHeroImages() {
			// P1 修复：公开只读模式禁止上传成品图（模板已 v-if 隐藏入口，此为防御性双重保险）
			if (this.isPublicView) return
			if (!this.recipe || this.isUploadingHeroImage) return
			const remaining = Math.max(this.maxRecipeImages - this.visibleRecipeSourceImages.length, 0)
			if (!remaining) return

			uni.chooseImage({
				count: remaining,
				sizeType: ['compressed'],
				sourceType: ['album', 'camera'],
				success: ({ tempFilePaths }) => {
					if (!tempFilePaths || !tempFilePaths.length) return
					this.saveHeroImages(tempFilePaths)
				}
			})
		},
		async saveHeroImages(imagePaths = []) {
			const incoming = Array.isArray(imagePaths) ? imagePaths.filter(Boolean) : [imagePaths].filter(Boolean)
			if (!incoming.length || !this.recipeId || this.isUploadingHeroImage) return

			this.isUploadingHeroImage = true
			uni.showLoading({
				title: '上传中',
				mask: true
			})

			try {
				// 隐藏态图片代表缓存与远端都已加载失败；用户下次补图时顺手把这些失效图清掉，避免卡住上限。
				const nextImages = [...this.visibleRecipeSourceImages]
				incoming.forEach((path) => {
					if (path && !nextImages.includes(path) && nextImages.length < this.maxRecipeImages) {
						nextImages.push(path)
					}
				})
				const recipe = await updateRecipeById(this.recipeId, {
					images: nextImages
				})
				this.applyRecipe(recipe)
				uni.showToast({
					title: `已添加 ${incoming.length} 张`,
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
		chooseEditImages() {
			const remaining = Math.max(this.maxRecipeImages - this.editDraft.images.length, 0)
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
					const nextImages = [...this.editDraft.images]
					tempFilePaths.forEach((path) => {
						if (path && !nextImages.includes(path) && nextImages.length < this.maxRecipeImages) {
							nextImages.push(path)
						}
					})
					this.editDraft.images = nextImages
				}
			})
		},
		removeEditImage(index) {
			if (typeof index !== 'number') return
			this.editDraft.images = this.editDraft.images.filter((_, currentIndex) => currentIndex !== index)
		},
		openEditImageOrderActions(index) {
			const images = Array.isArray(this.editDraft.images) ? this.editDraft.images.filter(Boolean) : []
			if (typeof index !== 'number' || images.length < 2 || index < 0 || index >= images.length) return

			const actions = []
			if (index > 0) {
				actions.push({
					label: '设为封面',
					handler: () => {
						this.moveEditImage(index, 0)
					}
				})
				actions.push({
					label: '左移一位',
					handler: () => {
						this.moveEditImage(index, index - 1)
					}
				})
			}
			if (index < images.length - 1) {
				actions.push({
					label: '右移一位',
					handler: () => {
						this.moveEditImage(index, index + 1)
					}
				})
			}
			if (!actions.length) return

			uni.showActionSheet({
				itemList: actions.map((item) => item.label),
				success: ({ tapIndex }) => {
					actions[tapIndex]?.handler?.()
				}
			})
		},
		moveEditImage(fromIndex, toIndex) {
			const nextImages = moveListItem(this.editDraft.images, fromIndex, toIndex)
			if (nextImages === this.editDraft.images) return
			this.editDraft.images = nextImages
		},
		getEditIngredientFieldKey(group = 'main') {
			return group === 'secondary' ? 'secondaryIngredients' : 'mainIngredients'
		},
		markEditParsedContentEdited() {
			if (!this.editDraft || this.editDraft.parsedContentMode === 'manual') return
			this.editDraft.parsedContentMode = 'manual'
		},
		ingredientGroupEmptyText(group = 'main') {
			if (this.editIsUsingFallbackContent) {
				return group === 'secondary'
					? '还没添加辅料或调味。'
					: '还没添加主料。'
			}
			return group === 'secondary'
				? '还没添加辅料或调味，比如葱姜蒜、盐、生抽。'
				: '还没添加主料，比如牛肉 500g。'
		},
		stepEmptyText() {
			if (this.editIsUsingFallbackContent) {
				return '还没添加步骤。'
			}
			return '还没添加步骤，可先补 3 到 6 步。'
		},
		addEditIngredient(group = 'main') {
			const fieldKey = this.getEditIngredientFieldKey(group)
			const nextIngredients = Array.isArray(this.editDraft[fieldKey]) ? [...this.editDraft[fieldKey]] : []
			nextIngredients.push(normalizeIngredientDraftItem())
			this.editDraft[fieldKey] = nextIngredients
			this.markEditParsedContentEdited()
		},
		handleEditIngredientInput(group = 'main', index = 0, event) {
			const fieldKey = this.getEditIngredientFieldKey(group)
			const nextIngredients = Array.isArray(this.editDraft[fieldKey]) ? [...this.editDraft[fieldKey]] : []
			if (index < 0 || index >= nextIngredients.length) return
			nextIngredients[index] = {
				...normalizeIngredientDraftItem(nextIngredients[index]),
				value: String(event?.detail?.value || '')
			}
			this.editDraft[fieldKey] = nextIngredients
			this.markEditParsedContentEdited()
		},
		removeEditIngredient(group = 'main', index = 0) {
			const fieldKey = this.getEditIngredientFieldKey(group)
			const nextIngredients = Array.isArray(this.editDraft[fieldKey])
				? this.editDraft[fieldKey].filter((_, currentIndex) => currentIndex !== index)
				: []
			this.editDraft[fieldKey] = nextIngredients
			this.markEditParsedContentEdited()
		},
		moveEditIngredient(group = 'main', fromIndex = 0, toIndex = 0) {
			const fieldKey = this.getEditIngredientFieldKey(group)
			const nextIngredients = moveListItem(this.editDraft[fieldKey], fromIndex, toIndex)
			if (nextIngredients === this.editDraft[fieldKey]) return
			this.editDraft[fieldKey] = nextIngredients
			this.markEditParsedContentEdited()
		},
		moveEditIngredientToGroup(fromGroup = 'main', index = 0, toGroup = 'secondary') {
			const fromFieldKey = this.getEditIngredientFieldKey(fromGroup)
			const toFieldKey = this.getEditIngredientFieldKey(toGroup)
			const currentIngredients = Array.isArray(this.editDraft[fromFieldKey]) ? [...this.editDraft[fromFieldKey]] : []
			if (index < 0 || index >= currentIngredients.length) return

			const [item] = currentIngredients.splice(index, 1)
			const nextTargetIngredients = Array.isArray(this.editDraft[toFieldKey]) ? [...this.editDraft[toFieldKey]] : []
			nextTargetIngredients.push(item)

			this.editDraft[fromFieldKey] = currentIngredients
			this.editDraft[toFieldKey] = nextTargetIngredients
			this.markEditParsedContentEdited()
		},
		openEditIngredientActions(group = 'main', index = 0) {
			const fieldKey = this.getEditIngredientFieldKey(group)
			const ingredients = Array.isArray(this.editDraft[fieldKey]) ? this.editDraft[fieldKey] : []
			if (index < 0 || index >= ingredients.length) return

			const actions = []
			if (index > 0) {
				actions.push({
					label: '上移一位',
					handler: () => {
						this.moveEditIngredient(group, index, index - 1)
					}
				})
			}
			if (index < ingredients.length - 1) {
				actions.push({
					label: '下移一位',
					handler: () => {
						this.moveEditIngredient(group, index, index + 1)
					}
				})
			}
			actions.push({
				label: group === 'secondary' ? '移到主料' : '移到辅料 / 调味',
				handler: () => {
					this.moveEditIngredientToGroup(group, index, group === 'secondary' ? 'main' : 'secondary')
				}
			})
			actions.push({
				label: '删除',
				handler: () => {
					this.removeEditIngredient(group, index)
				}
			})

			uni.showActionSheet({
				itemList: actions.map((item) => item.label),
				success: ({ tapIndex }) => {
					actions[tapIndex]?.handler?.()
				}
			})
		},
		addEditStep() {
			const nextSteps = Array.isArray(this.editDraft.steps) ? [...this.editDraft.steps] : []
			nextSteps.push(createStepDraftItem())
			this.editDraft.steps = nextSteps
			this.markEditParsedContentEdited()
		},
		handleEditStepFieldInput(index = 0, field = 'title', event) {
			const nextSteps = Array.isArray(this.editDraft.steps) ? [...this.editDraft.steps] : []
			if (index < 0 || index >= nextSteps.length) return
			nextSteps[index] = {
				...createStepDraftItem(nextSteps[index]),
				[field]: String(event?.detail?.value || '')
			}
			this.editDraft.steps = nextSteps
			this.markEditParsedContentEdited()
		},
		moveEditStep(fromIndex = 0, toIndex = 0) {
			const nextSteps = moveListItem(this.editDraft.steps, fromIndex, toIndex)
			if (nextSteps === this.editDraft.steps) return
			this.editDraft.steps = nextSteps
			this.markEditParsedContentEdited()
		},
		removeEditStep(index = 0) {
			const nextSteps = Array.isArray(this.editDraft.steps)
				? this.editDraft.steps.filter((_, currentIndex) => currentIndex !== index)
				: []
			this.editDraft.steps = nextSteps
			this.markEditParsedContentEdited()
		},
		previewEditImages(index = 0) {
			const urls = Array.isArray(this.editDraft.images) ? this.editDraft.images.filter(Boolean) : []
			if (!urls.length) return
			uni.previewImage({
				current: urls[index] || urls[0],
				urls
			})
		},
		async saveEditDraft() {
			if (!this.canSaveEditDraft || this.isSavingRecipe) return

			this.isSavingRecipe = true
			uni.showLoading({
				title: '保存中',
				mask: true
			})

			try {
				const mainIngredients = normalizeTextList(getIngredientDraftValues(this.editDraft.mainIngredients))
				const secondaryIngredients = normalizeTextList(getIngredientDraftValues(this.editDraft.secondaryIngredients))
				const steps = normalizeParsedSteps(this.editDraft.steps)
				const recipe = await updateRecipeById(this.recipeId, {
					title: this.editDraft.title.trim(),
					ingredient: this.editDraft.ingredient.trim(),
					link: this.editDraft.link.trim(),
					images: this.editDraft.images,
					mealType: this.editDraft.mealType,
					status: this.editDraft.status,
					parsedContent: {
						mainIngredients,
						secondaryIngredients,
						steps
					},
					parsedContentEdited: this.editDraft.parsedContentMode === 'manual',
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
		handleParseAction() {
			// P1 修复：公开只读模式禁止触发 AI 整理（消耗额度的写入口）
			if (this.isPublicView) return
			if (!this.canRequestParse || this.isReparseSubmitting) return
			if (this.needsParseOverwriteConfirm) {
				uni.showModal({
					title: '更新做法整理',
					content: this.parseOverwriteModalContent,
					confirmText: '继续整理',
					confirmColor: '#b4664c',
					success: ({ confirm }) => {
						if (!confirm) return
						this.requestAutoParse()
					}
				})
				return
			}

			// 已有整理结果时，重新整理也会消耗一次 AI 额度，做轻确认
			if (this.hasMeaningfulParsedContent) {
				uni.showModal({
					title: '重新整理？',
					content: '将再次调用 AI 整理食材与步骤，消耗 1 次额度。',
					confirmText: '继续整理',
					confirmColor: '#b4664c',
					success: ({ confirm }) => {
						if (!confirm) return
						this.requestAutoParse()
					}
				})
				return
			}

			this.requestAutoParse()
		},
		async requestAutoParse() {
			if (!this.canRequestParse || this.isReparseSubmitting) return

			this.isReparseSubmitting = true
			uni.showLoading({
				title: '整理中',
				mask: true
			})

			try {
				const recipe = await reparseRecipeById(this.recipeId)
				this.applyRecipe(recipe)
				this.showActionFeedback({
					tone: 'pending',
					title: '已加入整理队列',
					description:
						buildParseWaitHint(recipe?.parseStatus, recipe?.parseQueueAhead, recipe?.parseEstimatedWaitSeconds) ||
						parseStatusMetaMap.pending.description,
					duration: 1800
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '发起整理失败',
					icon: 'none'
				})
			} finally {
				this.isReparseSubmitting = false
				uni.hideLoading()
			}
		},
		// ===== P2-A：合并卡片 Tab 切换 + 统一 ⋯ 菜单 =====
		// 备注：原 openFlowchartMenu / openParseMenu 已被 openCookingMenu 取代并移除
		switchCookingTab(tab) {
			if (tab !== 'flowchart' && tab !== 'steps') return
			if (this.activeCookingTab === tab) return
			// 「一图看懂」Tab 仅在有图时可用；无图时静默忽略以防误触
			if (tab === 'flowchart' && !this.hasFlowchart) return
			this.activeCookingTab = tab
			// 轻触觉反馈，与底部「横屏查看」胶囊一致
			if (typeof uni !== 'undefined' && typeof uni.vibrateShort === 'function') {
				uni.vibrateShort({ type: 'light' })
			}
		},
		openCookingMenu() {
			// P1 修复：公开只读模式禁止打开做法菜单（含重新生成 / 重整理 / 查看详情写入口）
			if (this.isPublicView) return
			// 任一异步任务进行中时，⋯ 已被替换为 chip，此处只是双保险
			if (this.isCookingActive) return
			if (this.isGeneratingFlowchart || this.isReparseSubmitting) return

			const items = []
			const actions = []

			// 1) 重新生成一图（仅可执行时暴露）
			if (this.canRequestFlowchart && !this.isFlowchartActive) {
				items.push(this.hasFlowchart ? '重新生成一图看懂' : '生成一图看懂')
				actions.push('regen-flowchart')
			}

			// 2) 重新整理步骤
			if (this.canRequestParse) {
				items.push('重新整理步骤')
				actions.push('reparse')
			}

			// 3) 查看详情（合并 flowchart + parse 来源信息）
			const detailLines = [
				this.flowchartUpdatedAtText,
				this.flowchartModelTip,
				this.parseStatusSourceLabel
			].filter(Boolean)
			if (detailLines.length) {
				items.push('查看生成详情')
				actions.push('detail')
			}

			if (!items.length) {
				uni.showToast({ title: '当前无可执行操作', icon: 'none' })
				return
			}

			uni.showActionSheet({
				itemList: items,
				success: ({ tapIndex }) => {
					const action = actions[tapIndex]
					if (action === 'regen-flowchart') {
						this.handleGenerateFlowchart()
					} else if (action === 'reparse') {
						this.handleParseAction()
					} else if (action === 'detail') {
						uni.showModal({
							title: '生成详情',
							content: detailLines.join('\n'),
							showCancel: false,
							confirmText: '知道了',
							confirmColor: '#5b4a3b'
						})
					}
				}
			})
		},
		// ===== B1-2：复制食材清单到剪贴板 =====
		copyIngredientList() {
			if (!this.canCopyIngredientList) return

			const lines = []
			const title = String(this.recipe?.title || '').trim()
			if (title) {
				lines.push(`${title} · 食材清单`)
				lines.push('')
			}

			if (this.parsedMainIngredients.length) {
				lines.push(`主料：${this.parsedMainIngredients.join('、')}`)
			}

			this.parsedSecondaryGroups.forEach((group) => {
				if (group?.text) {
					lines.push(`${group.label}：${group.text}`)
				}
			})

			const text = lines.join('\n').trim()
			if (!text) {
				uni.showToast({ title: '清单为空', icon: 'none' })
				return
			}

			uni.setClipboardData({
				data: text,
				success: () => {
					if (typeof uni.vibrateShort === 'function') {
						uni.vibrateShort({ type: 'light' })
					}
					// uni.setClipboardData 默认会弹「已复制」toast，这里再补一句更具体的提示
					uni.showToast({
						title: '已复制，可去备忘录粘贴',
						icon: 'none',
						duration: 1600
					})
				},
				fail: () => {
					uni.showToast({ title: '复制失败，请重试', icon: 'none' })
				}
			})
		},
		// ===== B2-5：步骤详情高亮切片代理（透传到外部纯函数）=====
		highlightStepDetail(detail) {
			return highlightStepDetailText(detail)
		},
		buildCurrentStepCompletionKeys() {
			return buildStepCompletionKeyList(this.parsedSteps)
		},
		getStepCompletionKey(index) {
			const stepKeys = this.buildCurrentStepCompletionKeys()
			return stepKeys[index] || ''
		},
		// ===== B2-6：步骤完成状态管理 =====
		isStepCompleted(index) {
			const stepKey = this.getStepCompletionKey(index)
			return !!(stepKey && this.completedStepKeyMap[stepKey])
		},
		toggleStepCompleted(index) {
			if (typeof index !== 'number' || index < 0) return
			const stepKey = this.getStepCompletionKey(index)
			if (!stepKey) return
			// 触发 Vue 响应式更新：使用 $set 或对象重建（这里直接重建以兼容 Vue 3 / 2.x）
			const next = { ...this.completedStepKeyMap }
			if (next[stepKey]) {
				delete next[stepKey]
			} else {
				next[stepKey] = true
			}
			this.completedStepKeyMap = next
			this.persistCompletedSteps()
			if (typeof uni.vibrateShort === 'function') {
				uni.vibrateShort({ type: 'light' })
			}
		},
		resetCompletedSteps() {
			if (!this.completedStepCount) return
			uni.showModal({
				title: '重置完成进度？',
				content: '将清除当前菜谱所有步骤的「已完成」标记。',
				confirmText: '重置',
				confirmColor: '#b4664c',
				success: ({ confirm }) => {
					if (!confirm) return
					this.completedStepKeyMap = {}
					this.persistCompletedSteps()
				}
			})
		},
		loadCompletedSteps() {
			const key = buildStepCompletedStorageKey(this.recipeId)
			if (!key) {
				this.completedStepKeyMap = {}
				return
			}
			const currentStepKeys = this.buildCurrentStepCompletionKeys()
			try {
				const raw = uni.getStorageSync(key)
				this.completedStepKeyMap = normalizeCompletedStepKeyMap(raw, currentStepKeys)
			} catch (error) {
				// 存储读失败不致命，回退到空状态
				this.completedStepKeyMap = {}
			}
		},
		persistCompletedSteps() {
			const key = buildStepCompletedStorageKey(this.recipeId)
			if (!key) return
			try {
				if (Object.keys(this.completedStepKeyMap).length === 0) {
					uni.removeStorageSync(key)
				} else {
					uni.setStorageSync(key, createCompletedStepStoragePayload(this.completedStepKeyMap))
				}
			} catch (error) {
				// 存储写失败不致命，仅记录到 console（生产环境无影响）
				// eslint-disable-next-line no-console
				console.warn('[recipe-detail] persistCompletedSteps failed:', error)
			}
		},
		async handleGenerateFlowchart() {
			// P1 修复：公开只读模式禁止生成流程图（防御性兜底，模板已 v-if 隐藏 CTA）
			if (this.isPublicView) return
			if (!this.recipeId || this.isGeneratingFlowchart || !this.canRequestFlowchart) return
			if (!this.canGenerateFlowchart) {
				uni.showToast({
					title: '先补充至少 3 个关键步骤',
					icon: 'none'
				})
				return
			}

			// 已有流程图时，重新生成会消耗一次 AI 额度，做二次确认
			if (this.hasFlowchart) {
				const confirmed = await new Promise((resolve) => {
					uni.showModal({
						title: '重新生成步骤图？',
						content: '将再次调用 AI 生成，消耗 1 次额度，约需 15 秒。',
						confirmText: '继续生成',
						confirmColor: '#b4664c',
						success: ({ confirm }) => resolve(!!confirm),
						fail: () => resolve(false)
					})
				})
				if (!confirmed) return
			}

			await this.submitFlowchartGeneration()
		},
		async submitFlowchartGeneration() {
			this.isGeneratingFlowchart = true
			uni.showLoading({
				title: '提交中',
				mask: true
			})

			try {
				const recipe = await generateRecipeFlowchartById(this.recipeId)
				this.applyRecipe(recipe)
				this.showActionFeedback({
					tone: 'pending',
					title: '已加入生成队列',
					description:
						buildFlowchartWaitHint(recipe?.flowchartStatus, recipe?.flowchartQueueAhead, recipe?.flowchartEstimatedWaitSeconds) ||
						flowchartStatusMetaMap.pending.description,
					duration: 1800
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '生成流程图失败',
					icon: 'none'
				})
			} finally {
				this.isGeneratingFlowchart = false
				uni.hideLoading()
			}
		},
		async togglePinned() {
			if (!this.recipeId || !this.recipe || this.isPinSubmitting) return

			const nextPinned = !this.isPinned
			this.isPinSubmitting = true
			uni.showLoading({
				title: nextPinned ? '置顶中' : '更新中',
				mask: true
			})

			try {
				const recipe = await setRecipePinnedById(this.recipeId, nextPinned)
				this.applyRecipe(recipe)
				uni.showToast({
					title: nextPinned ? '已置顶' : '已取消置顶',
					icon: 'none'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '更新置顶失败',
					icon: 'none'
				})
			} finally {
				this.isPinSubmitting = false
				uni.hideLoading()
			}
		},
		confirmDeleteRecipe() {
			// P1 修复：公开只读模式禁止删除菜谱
			if (this.isPublicView) return
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
			const link = this.displayRecipeLink
			if (!link) {
				uni.showToast({
					title: '暂无链接',
					icon: 'none'
				})
				return
			}
			uni.setClipboardData({
				data: link,
				success: () => {
					uni.showToast({
						title: '已复制链接',
						icon: 'none'
					})
				}
			})
		},
		previewRecipeImage() {
			const urls = this.displayRecipeImages.map((item) => item.displayURL).filter(Boolean)
			if (!urls.length) return

			uni.previewImage({
				current: urls[this.heroImageIndex] || urls[0],
				urls
			})
		},
		// ===== Hero 操作菜单：⋯ 按钮入口 =====
		openHeroActionMenu() {
			// P1 修复：公开只读模式禁止 Hero 操作菜单（替换/重排/删除封面图入口）
			if (this.isPublicView) return
			if (!this.canShowHeroActionMenu) return

			const items = []
			const actions = []

			if (this.canSetCurrentAsCover) {
				items.push('设为封面')
				actions.push('set-cover')
			}
			if (this.canAddMoreHeroImages) {
				items.push('添加更多图片')
				actions.push('add-more')
			}
			if (this.canDeleteCurrentImage) {
				items.push('删除这张图')
				actions.push('delete')
			}

			if (!items.length) return

			uni.showActionSheet({
				itemList: items,
				success: ({ tapIndex }) => {
					const action = actions[tapIndex]
					if (action === 'set-cover') {
						this.setCurrentImageAsCover()
					} else if (action === 'add-more') {
						this.chooseHeroImages()
					} else if (action === 'delete') {
						this.confirmDeleteCurrentImage()
					}
				}
			})
		},
		// Hero 修复：把「可见列表」(displayRecipeImages) 索引映射回 recipeImages 原始索引
		// 当某些图加载失败被 recipeImageHiddenMap 标记时，两个数组长度/顺序会错位
		// 返回 -1 表示无法映射（越界或可见列表为空）
		resolveOriginalImageIndex(visibleIndex) {
			const visibleList = this.displayRecipeImages
			if (!Array.isArray(visibleList) || visibleIndex < 0 || visibleIndex >= visibleList.length) {
				return -1
			}
			const target = visibleList[visibleIndex]
			const targetKey = target && target.cacheKey
			if (!targetKey) return -1
			const version = this.recipeImageVersion
			for (let i = 0; i < this.recipeImages.length; i += 1) {
				const cacheKey = buildImageCacheKey(this.recipeImages[i], version)
				if (cacheKey === targetKey) return i
			}
			return -1
		},
		async setCurrentImageAsCover() {
			if (!this.canSetCurrentAsCover || !this.recipeId) return
			if (this.isUploadingHeroImage) return

			// Hero 修复：heroImageIndex 是「可见列表」索引，必须映射回原始 recipeImages 索引，
			// 否则前面有图加载失败被隐藏时会操作到错图
			const fromIndex = this.resolveOriginalImageIndex(this.heroImageIndex)
			const list = [...this.recipeImages]
			if (fromIndex <= 0 || fromIndex >= list.length) return
			// 把当前位置图片移到第 0 位（其余顺序保持不变）
			const [picked] = list.splice(fromIndex, 1)
			list.unshift(picked)

			this.isUploadingHeroImage = true
			uni.showLoading({ title: '设置中', mask: true })
			try {
				const recipe = await updateRecipeById(this.recipeId, { images: list })
				this.applyRecipe(recipe)
				// 把视图切回第 0 位，让用户立刻看到「新封面」生效
				this.heroImageIndex = 0
				if (typeof uni.vibrateShort === 'function') {
					uni.vibrateShort({ type: 'medium' })
				}
				this.showActionFeedback({
					tone: 'done',
					title: '已设为封面'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '设置失败，请重试',
					icon: 'none'
				})
			} finally {
				this.isUploadingHeroImage = false
				uni.hideLoading()
			}
		},
		confirmDeleteCurrentImage() {
			if (!this.canDeleteCurrentImage) return
			uni.showModal({
				title: '删除这张图？',
				content: '删除后无法恢复，仍可重新上传。',
				confirmText: '删除',
				confirmColor: '#b4664c',
				success: ({ confirm }) => {
					if (!confirm) return
					this.deleteCurrentImage()
				}
			})
		},
		async deleteCurrentImage() {
			if (!this.canDeleteCurrentImage || !this.recipeId) return
			if (this.isUploadingHeroImage) return

			// Hero 修复：同样需要把可见索引映射回原始数组索引，避免删错图
			const removeIndex = this.resolveOriginalImageIndex(this.heroImageIndex)
			const list = [...this.recipeImages]
			if (removeIndex < 0 || removeIndex >= list.length) return
			list.splice(removeIndex, 1)

			this.isUploadingHeroImage = true
			uni.showLoading({ title: '删除中', mask: true })
			try {
				const recipe = await updateRecipeById(this.recipeId, { images: list })
				this.applyRecipe(recipe)
				// 调整 heroImageIndex 防止越界（applyRecipe 也有兜底，此处再保险一次）
				if (this.heroImageIndex >= this.displayRecipeImages.length) {
					this.heroImageIndex = Math.max(this.displayRecipeImages.length - 1, 0)
				}
				if (typeof uni.vibrateShort === 'function') {
					uni.vibrateShort({ type: 'light' })
				}
				this.showActionFeedback({
					tone: 'done',
					title: '已删除'
				})
			} catch (error) {
				uni.showToast({
					title: error?.message || '删除失败，请重试',
					icon: 'none'
				})
			} finally {
				this.isUploadingHeroImage = false
				uni.hideLoading()
			}
		},
		openFlowchartViewer() {
			if (!this.flowchartDisplayImageUrl) return
			const key = `${this.recipeId || 'recipe'}-${Date.now()}`
			uni.setStorageSync(FLOWCHART_VIEWER_STORAGE_KEY, {
				key,
				imageUrl: this.flowchartImageUrl,
				localImagePath: String(this.cachedFlowchartImagePath || '').trim(),
				title: String(this.recipe?.title || '').trim(),
				updatedAtText: this.flowchartUpdatedAtText
			})
			uni.navigateTo({
				url: `/pages/flowchart-viewer/index?key=${encodeURIComponent(key)}`
			})
		},
		async handleFlowchartImageError() {
			const localPath = String(this.cachedFlowchartImagePath || '').trim()
			if (!localPath || this.flowchartDisplayImageUrl !== localPath) return
			this.cachedFlowchartImagePath = ''
			try {
				await invalidateCachedImage(this.flowchartImageUrl, this.flowchartImageCacheVersion || this.buildFlowchartImageCacheVersion())
			} catch (error) {
				// Ignore stale cache cleanup failures and keep remote fallback usable.
			}
		},
		previewFlowchartImage() {
			// 轻点图片：用系统原生 previewImage 做快速预览（双指缩放、保存、长按菜单）
			// 与右下「横屏查看 ›」胶囊的横屏沉浸模式区分：轻 = 快看，重 = 横屏沉浸
			const urls = this.buildFlowchartPreviewURLs()
			if (!urls.length) return
			uni.vibrateShort && uni.vibrateShort({ type: 'light' })
			uni.previewImage({
				urls,
				current: urls[0]
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
		},
		// P2-D 分享路径升级：失效空态 CTA
		// 公开模式下优先 navigateBack 关闭当前页（用户期望「关掉这个失效页面」），
		// 私有路径走 goBack 兜底（reLaunch 到首页）
		handleMissingStateBack() {
			if (this.isPublicView) {
				if (getCurrentPages().length > 1) {
					uni.navigateBack()
					return
				}
				uni.reLaunch({ url: '/pages/index/index' })
				return
			}
			this.goBack()
		},
		// P2-D 分享路径升级：banner 「了解」按钮，弹出只读规则说明
		openPublicReadOnlyExplain() {
			this.showPublicReadOnlyExplain = true
		},
		closePublicReadOnlyExplain() {
			this.showPublicReadOnlyExplain = false
		}
	}
}
</script>

<style lang="scss" scoped>
	.detail-page {
		min-height: 100vh;
		background:
			radial-gradient(circle at top right, rgba(255, 235, 214, 0.3) 0%, rgba(255, 235, 214, 0) 32%),
			linear-gradient(180deg, #f7f4ef 0%, #f4f1ec 100%);
	}

	/* P2-D 分享路径升级：公开只读模式专属样式 */
	/* 公开模式下 detail-scroll 顶部留出 banner 高度 */
	.detail-page--public .detail-scroll {
		padding-top: 76rpx;
	}

	/* 顶部 banner：薄薄一条暖灰背景 + 「了解」轻量按钮 */
	/* 固定在顶部不滚动，z-index 高于 hero-card 但低于 popup */
	.public-banner {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 8;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16rpx 28rpx;
		padding-top: calc(env(safe-area-inset-top) + 12rpx);
		background: rgba(247, 240, 230, 0.94);
		backdrop-filter: blur(12px);
		border-bottom: 1rpx solid rgba(180, 156, 130, 0.18);
	}
	.public-banner__body {
		display: flex;
		align-items: center;
		gap: 10rpx;
		flex: 1;
		min-width: 0;
	}
	.public-banner__text {
		font-size: 24rpx;
		color: #6b5b4d;
		line-height: 1.4;
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.public-banner__action {
		flex: none;
		padding: 8rpx 20rpx;
		border-radius: 24rpx;
		background: rgba(180, 156, 130, 0.16);
		transition: background 0.16s ease;
	}
	.public-banner__action--hover {
		background: rgba(180, 156, 130, 0.28);
	}
	.public-banner__action-text {
		font-size: 24rpx;
		color: #7b6d62;
		font-weight: 500;
	}

	/* 只读规则 popup 内部布局 */
	.public-explain {
		width: 560rpx;
		padding: 48rpx 40rpx 32rpx;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}
	.public-explain__icon {
		width: 72rpx;
		height: 72rpx;
		border-radius: 50%;
		background: rgba(180, 156, 130, 0.14);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 20rpx;
	}
	.public-explain__title {
		font-size: 32rpx;
		font-weight: 600;
		color: #3f342b;
		margin-bottom: 16rpx;
	}
	.public-explain__desc {
		font-size: 26rpx;
		color: #6b5b4d;
		line-height: 1.6;
		margin-bottom: 32rpx;
	}
	.public-explain__action {
		width: 100%;
		padding: 22rpx 0;
		border-radius: 16rpx;
		background: linear-gradient(135deg, #c79a72 0%, #b4856a 100%);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: opacity 0.16s ease;
	}
	.public-explain__action--hover {
		opacity: 0.86;
	}
	.public-explain__action-text {
		font-size: 28rpx;
		color: #ffffff;
		font-weight: 600;
	}

	.detail-scroll {
		height: 100vh;
		box-sizing: border-box;
		padding: 28rpx 24rpx calc(env(safe-area-inset-bottom) + 200rpx);
	}

	.hero-card,
	.detail-card,
	.missing-state {
		border-radius: 30rpx;
		background: rgba(255, 253, 249, 0.96);
		border: 1px solid rgba(100, 78, 58, 0.05);
		box-shadow:
			0 14rpx 30rpx rgba(70, 54, 40, 0.045),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.68);
	}

	.hero-card {
		position: relative;
		overflow: hidden;
		min-height: 380rpx;
		box-shadow:
			0 18rpx 34rpx rgba(70, 54, 40, 0.06),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.34);
	}

	/* H5：有图时增高 Hero，为压图标题预留 ~140rpx 空间 */
	.hero-card--with-overlay {
		min-height: 520rpx;
	}

	.hero-card::before,
	.hero-card::after {
		content: '';
		position: absolute;
		inset: 0;
		pointer-events: none;
		z-index: 1;
	}

	.hero-card::before {
		border-radius: inherit;
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.18);
	}

	.hero-card::after {
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.18) 0%, rgba(255, 255, 255, 0) 30%),
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.22) 0%, rgba(255, 255, 255, 0) 34%);
	}

	.hero-card--empty {
		min-height: 380rpx;
	}

	.hero-card__swiper {
		width: 100%;
		height: 380rpx;
	}

	.hero-card--with-overlay .hero-card__swiper {
		height: 520rpx;
	}

	.hero-card__image {
		position: relative;
		z-index: 0;
		width: 100%;
		height: 380rpx;
		display: block;
	}

	.hero-card--with-overlay .hero-card__image {
		height: 520rpx;
	}

	/* H4：底部渐变蒙层，从 0 → 0.82 黑色，为压图文字提供读性 */
	.hero-card__overlay {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		height: 280rpx;
		background:
			linear-gradient(180deg,
				rgba(20, 14, 10, 0) 0%,
				rgba(20, 14, 10, 0.32) 38%,
				rgba(20, 14, 10, 0.72) 78%,
				rgba(20, 14, 10, 0.85) 100%
			);
		pointer-events: none;
		z-index: 2;
	}

	/* Hero 操作菜单按钮：右上角圆形 ⋯，与压图标题/分页器同层 z-index 3 */
	.hero-card__action {
		position: absolute;
		top: 22rpx;
		right: 22rpx;
		z-index: 3;
		width: 56rpx;
		height: 56rpx;
		border-radius: 50%;
		background: rgba(20, 14, 10, 0.42);
		border: 1px solid rgba(255, 255, 255, 0.18);
		backdrop-filter: blur(10rpx);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.16s ease, background 0.16s ease;
	}

	.hero-card__action--hover {
		transform: scale(0.92);
		background: rgba(15, 10, 7, 0.62);
	}

	/* H5：标题压图块；定位在 Hero 底部、分页器上方 */
	.hero-card__title-block {
		position: absolute;
		left: 28rpx;
		right: 28rpx;
		bottom: 60rpx;
		z-index: 3;
		display: flex;
		flex-direction: column;
		gap: 14rpx;
	}

	.hero-card__meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10rpx;
	}

	/* 压图 chip：半透明深底 + 白字，确保在任意配色背景上的可读性 */
	.hero-card__chip {
		min-height: 40rpx;
		padding: 0 14rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.22);
		border: 1px solid rgba(255, 255, 255, 0.28);
		backdrop-filter: blur(8rpx);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.hero-card__chip--meal {
		background: rgba(180, 102, 76, 0.78);
		border-color: rgba(255, 255, 255, 0.32);
	}

	.hero-card__chip--done {
		background: rgba(111, 130, 102, 0.78);
		border-color: rgba(255, 255, 255, 0.32);
	}

	.hero-card__chip--wishlist {
		background: rgba(154, 115, 67, 0.78);
		border-color: rgba(255, 255, 255, 0.32);
	}

	.hero-card__chip--pin {
		background: rgba(186, 145, 81, 0.82);
		border-color: rgba(255, 255, 255, 0.32);
	}

	.hero-card__chip-text {
		font-size: 22rpx;
		font-weight: 700;
		line-height: 1;
		color: #fff;
		letter-spacing: 0.4rpx;
	}

	.hero-card__title {
		display: block;
		font-family: "Songti SC", "STKaiti", "DejaVu Serif", serif;
		font-size: 44rpx;
		font-weight: 800;
		line-height: 1.25;
		color: #ffffff;
		letter-spacing: 0.5rpx;
		text-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.36);
	}

	/* H3：分页器 —— 底部居中圆点 dots / 数字 chip，与压图标题分层（下方 ~22rpx） */
	.hero-card__pager {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 22rpx;
		z-index: 3;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 10rpx;
		pointer-events: none;
	}

	.hero-card__dot {
		width: 10rpx;
		height: 10rpx;
		border-radius: 50%;
		background: rgba(255, 255, 255, 0.42);
		transition: width 0.2s ease, background 0.2s ease;
	}

	.hero-card__dot--active {
		width: 24rpx;
		border-radius: 5rpx;
		background: rgba(255, 255, 255, 0.95);
	}

	.hero-card__counter {
		padding: 6rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(20, 14, 10, 0.42);
		border: 1px solid rgba(255, 255, 255, 0.18);
		backdrop-filter: blur(10rpx);
	}

	.hero-card__counter-text {
		font-size: 21rpx;
		font-weight: 600;
		color: #ffffff;
		letter-spacing: 0.4rpx;
	}

	.hero-card__placeholder {
		position: relative;
		min-height: 380rpx;
		box-sizing: border-box;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.28), rgba(255, 255, 255, 0) 36%),
			linear-gradient(135deg, #e6dbcf 0%, #d6c6b5 100%);
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
		z-index: 2;
		padding: 18rpx 30rpx;
		border-radius: 999rpx;
		border: 1px solid rgba(255, 255, 255, 0.58);
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.88), rgba(255, 255, 255, 0.74));
		box-shadow:
			0 12rpx 22rpx rgba(91, 74, 59, 0.08),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.56);
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
		padding: 28rpx 6rpx 10rpx;
	}

	/* H5：有图时只渲染 summary，把上 padding 收紧避免空隙过大 */
	.detail-head--summary-only {
		padding: 18rpx 6rpx 10rpx;
	}

	.detail-head__meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12rpx;
	}

	.detail-chip {
		min-height: 48rpx;
		padding: 0 18rpx;
		border-radius: 999rpx;
		border: 1px solid rgba(99, 79, 60, 0.08);
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.88), rgba(249, 245, 240, 0.92));
		box-shadow:
			0 8rpx 18rpx rgba(68, 52, 38, 0.04),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.66);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.detail-chip--meal {
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(246, 241, 235, 0.94));
	}

	.detail-chip--wishlist {
		background:
			linear-gradient(180deg, rgba(255, 246, 239, 0.98), rgba(247, 235, 225, 0.96));
		border-color: rgba(187, 127, 88, 0.12);
	}

	.detail-chip--done {
		background:
			linear-gradient(180deg, rgba(244, 250, 243, 0.98), rgba(232, 240, 231, 0.96));
		border-color: rgba(103, 132, 104, 0.14);
	}

	.detail-chip--pin {
		background:
			linear-gradient(180deg, rgba(255, 249, 237, 0.98), rgba(247, 237, 211, 0.96));
		border-color: rgba(186, 145, 81, 0.14);
	}

	.detail-chip__text {
		display: block;
		font-size: 22rpx;
		font-weight: 700;
		line-height: 1;
		color: #746558;
	}

	.detail-chip--wishlist .detail-chip__text {
		color: #a06b49;
	}

	.detail-chip--done .detail-chip__text {
		color: #668264;
	}

	.detail-chip--pin .detail-chip__text {
		color: #9a7343;
	}

	.detail-title {
		display: block;
		margin-top: 20rpx;
		font-family: "Songti SC", "STKaiti", "DejaVu Serif", serif;
		font-size: 44rpx;
		font-weight: 700;
		line-height: 1.2;
		letter-spacing: 0.5rpx;
		color: #2f2923;
	}

	.detail-summary-card {
		position: relative;
		margin-top: 18rpx;
		padding: 18rpx 22rpx 18rpx 24rpx;
		border-radius: 26rpx;
		background:
			radial-gradient(circle at top right, rgba(255, 239, 226, 0.38) 0%, rgba(255, 239, 226, 0) 36%),
			linear-gradient(180deg, rgba(255, 255, 255, 0.76), rgba(248, 243, 237, 0.9));
		border: 1px solid rgba(111, 86, 64, 0.06);
		box-shadow:
			0 8rpx 18rpx rgba(70, 54, 40, 0.032),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.58);
		overflow: hidden;
	}

	.detail-summary-card::before {
		content: '';
		position: absolute;
		left: 0;
		top: 14rpx;
		bottom: 14rpx;
		width: 6rpx;
		border-radius: 999rpx;
		background: linear-gradient(180deg, #d1894f 0%, rgba(209, 137, 79, 0.42) 100%);
	}

	/* P1-4: 移除原右上角装饰引号 ::after（截图反馈像孤立 bug），亮点语义改由加深的左色条承载 */

	.detail-summary {
		position: relative;
		z-index: 1;
		display: block;
		margin-top: 0;
		padding-right: 8rpx;
		font-size: 26rpx;
		line-height: 1.74;
		color: #5e544b;
	}

	.detail-card {
		/* P3-1: 卡片间距规范化为 24rpx，对齐 8 网格 */
		margin-top: 24rpx;
		padding: 28rpx;
		background:
			linear-gradient(180deg, rgba(255, 254, 252, 0.98), rgba(255, 251, 246, 0.95));
	}

	.detail-card--flowchart {
		background:
			radial-gradient(circle at top right, rgba(255, 233, 211, 0.52) 0%, rgba(255, 233, 211, 0) 34%),
			linear-gradient(180deg, rgba(255, 254, 252, 0.98), rgba(255, 250, 245, 0.95));
		border-color: rgba(181, 136, 94, 0.08);
		box-shadow:
			0 18rpx 32rpx rgba(70, 54, 40, 0.05),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.72);
	}

	.detail-card--content {
		background:
			radial-gradient(circle at top right, rgba(236, 243, 232, 0.42) 0%, rgba(236, 243, 232, 0) 34%),
			linear-gradient(180deg, rgba(255, 254, 252, 0.98), rgba(251, 251, 248, 0.95));
		border-color: rgba(103, 132, 104, 0.06);
	}

	.detail-card--quiet {
		background:
			linear-gradient(180deg, rgba(255, 254, 252, 0.96), rgba(250, 246, 241, 0.92));
		box-shadow:
			0 12rpx 24rpx rgba(70, 54, 40, 0.038),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.66);
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
		font-size: 32rpx;
		font-weight: 600;
		line-height: 1.25;
		color: #1e293b;
		letter-spacing: 0.2rpx;
	}

	.detail-card__action {
		min-height: 56rpx;
		padding: 0 20rpx;
		box-sizing: border-box;
		border-radius: 999rpx;
		border: 1px solid rgba(99, 79, 60, 0.08);
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(245, 239, 232, 0.94));
		box-shadow:
			0 8rpx 16rpx rgba(68, 52, 38, 0.04),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transform: scale(1);
		transition: transform 0.16s ease, box-shadow 0.16s ease, border-color 0.16s ease, background 0.16s ease;
	}

	.detail-card__action:active {
		transform: scale(0.985);
	}

	.detail-card__action--accent {
		background:
			linear-gradient(180deg, rgba(255, 245, 239, 0.98), rgba(248, 233, 223, 0.96));
		border: 1px solid rgba(180, 102, 76, 0.14);
	}

	.detail-card__action--disabled {
		opacity: 0.6;
		pointer-events: none;
	}

	.detail-card__action-text {
		display: block;
		font-size: 22rpx;
		font-weight: 600;
		line-height: 1;
		color: #6d6155;
		text-align: center;
	}

	.detail-card__action-text--accent {
		color: #b4664c;
	}

	/* P1: 卡片右上角折叠菜单的图标按钮（⋯），降低视觉权重 */
	.detail-card__icon-action {
		width: 56rpx;
		height: 56rpx;
		border-radius: 999rpx;
		border: 1px solid rgba(99, 79, 60, 0.08);
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(245, 239, 232, 0.94));
		box-shadow:
			0 6rpx 14rpx rgba(68, 52, 38, 0.04),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: transform 0.16s ease, background 0.16s ease;
	}

	.detail-card__icon-action--active {
		transform: scale(0.94);
		background: rgba(245, 239, 232, 0.96);
	}

	.detail-card__icon-action--disabled {
		opacity: 0.5;
		pointer-events: none;
	}

	/* 后台生成中：非交互 chip，明确告知用户「正在生成」，避免点击 ⋯ 后跳出无意义的 dead-click 菜单 */
	.detail-card__status-chip {
		min-height: 56rpx;
		padding: 0 18rpx;
		border-radius: 999rpx;
		background: rgba(244, 233, 218, 0.92);
		border: 1px solid rgba(186, 145, 81, 0.18);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.detail-card__status-chip-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #9a7343;
		letter-spacing: 0.2rpx;
	}

	/* ===== P2-A：「做法」合并卡片 ===== */
	.detail-card--cooking {
		overflow: hidden;
	}

	/* 顶部 Tab 栏：分段控制器风格，与卡片视觉融为一体 */
	.cooking-tabs {
		margin-top: 18rpx;
		padding: 6rpx;
		border-radius: 18rpx;
		background: rgba(244, 233, 218, 0.62);
		border: 1px solid rgba(180, 102, 76, 0.1);
		display: flex;
		align-items: stretch;
		gap: 4rpx;
	}

	.cooking-tabs__item {
		flex: 1;
		min-height: 64rpx;
		border-radius: 14rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		transition: background 0.16s ease, transform 0.16s ease;
	}

	.cooking-tabs__item--active {
		background: #ffffff;
		box-shadow:
			0 4rpx 12rpx rgba(70, 54, 40, 0.06),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.8);
	}

	.cooking-tabs__item--hover {
		transform: scale(0.98);
		background: rgba(255, 255, 255, 0.5);
	}

	.cooking-tabs__text {
		font-size: 26rpx;
		font-weight: 600;
		color: #8c7a66;
		letter-spacing: 0.4rpx;
	}

	.cooking-tabs__item--active .cooking-tabs__text {
		color: #5b4a3b;
		font-weight: 700;
	}

	/* 「详细步骤」Tab 内的「生成一图看懂」CTA：弱主色，引导但不抢戏 */
	.cooking-flowchart-cta {
		margin-top: 18rpx;
		padding: 18rpx 22rpx;
		border-radius: 18rpx;
		background:
			linear-gradient(135deg, rgba(255, 244, 232, 0.96), rgba(249, 232, 215, 0.92));
		border: 1px dashed rgba(180, 102, 76, 0.28);
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
		transition: transform 0.16s ease, background 0.16s ease;
	}

	.cooking-flowchart-cta:active {
		transform: scale(0.985);
		background:
			linear-gradient(135deg, rgba(255, 240, 224, 0.98), rgba(247, 226, 207, 0.96));
	}

	.cooking-flowchart-cta__text {
		font-size: 25rpx;
		font-weight: 600;
		color: #b4664c;
		letter-spacing: 0.4rpx;
	}

	.cooking-flowchart-cta__arrow {
		font-size: 26rpx;
		font-weight: 700;
		color: #b4664c;
		line-height: 1;
	}

	/* 卡片底部统一 caption 行 */
	.cooking-footer {
		margin-top: 18rpx;
		padding-top: 14rpx;
		border-top: 1px solid rgba(91, 74, 59, 0.06);
		display: flex;
		align-items: center;
		justify-content: flex-start;
	}

	.cooking-footer__text {
		font-size: 22rpx;
		line-height: 1.4;
		color: #94a3b8;
		letter-spacing: 0.2rpx;
	}

	.link-panel {
		margin-top: 18rpx;
	}

	.detail-link-box {
		width: 100%;
		box-sizing: border-box;
		padding: 20rpx 22rpx;
		border-radius: 20rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.82), rgba(247, 243, 237, 0.92));
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.58);
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

	.detail-card--flowchart {
		overflow: hidden;
	}

	.flowchart-hint {
		margin-top: 18rpx;
		padding: 14rpx 18rpx;
		border-radius: 18rpx;
		background:
			linear-gradient(180deg, rgba(255, 244, 238, 0.98), rgba(249, 236, 227, 0.96));
		border: 1px solid rgba(193, 120, 87, 0.12);
		display: flex;
		align-items: center;
		gap: 10rpx;
	}

	.flowchart-hint__text {
		font-size: 22rpx;
		line-height: 1.5;
		color: #b4664c;
	}

	.flowchart-panel {
		margin-top: 18rpx;
		position: relative;
		border-radius: 24rpx;
		overflow: hidden;
		background:
			linear-gradient(180deg, rgba(250, 246, 240, 0.98), rgba(246, 241, 234, 0.94));
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow:
			0 12rpx 24rpx rgba(70, 54, 40, 0.045),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.42);
		transform: scale(1);
		transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
	}

	.flowchart-panel--active {
		border-color: rgba(123, 96, 72, 0.12);
		box-shadow:
			0 10rpx 20rpx rgba(70, 54, 40, 0.052),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.42);
		transform: scale(0.994);
	}

	.flowchart-panel__image-shell {
		position: relative;
	}

	.flowchart-panel__image {
		width: 100%;
		display: block;
		background: #f6f2ed;
		transition: opacity 0.18s ease;
	}

	/* 轻点图片：调系统原生预览，按下时给一个克制的透明度反馈 */
	.flowchart-panel__image--active {
		opacity: 0.92;
	}

	.flowchart-panel__image-shadow {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		height: 120rpx;
		background: linear-gradient(180deg, rgba(38, 28, 21, 0), rgba(38, 28, 21, 0.42));
		pointer-events: none;
	}

	.flowchart-panel__cta {
		position: absolute;
		right: 20rpx;
		bottom: 20rpx;
		z-index: 1;
		min-height: 64rpx;
		padding: 0 18rpx 0 22rpx;
		border-radius: 999rpx;
		background: rgba(29, 22, 17, 0.82);
		border: 1px solid rgba(255, 255, 255, 0.1);
		backdrop-filter: blur(14rpx);
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
		box-shadow:
			0 12rpx 26rpx rgba(0, 0, 0, 0.16),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.08);
		transition: transform 0.16s ease, background 0.16s ease;
	}

	/* 胶囊按下：轻微 scale + 加深背景，明确「这是个独立按钮」 */
	.flowchart-panel__cta--active {
		transform: scale(0.96);
		background: rgba(15, 10, 7, 0.92);
	}

	.flowchart-panel__cta-text,
	.flowchart-panel__cta-arrow {
		display: block;
		line-height: 1;
		color: #fff9f2;
	}

	.flowchart-panel__cta-text {
		font-size: 24rpx;
		font-weight: 700;
		letter-spacing: 0.5rpx;
	}

	.flowchart-panel__cta-arrow {
		font-size: 24rpx;
		font-weight: 700;
		color: rgba(255, 249, 242, 0.84);
	}

	.flowchart-panel__footer {
		padding: 14rpx 20rpx 18rpx;
		display: flex;
		align-items: center;
		justify-content: flex-start;
	}

	.flowchart-panel__caption {
		font-size: 22rpx;
		line-height: 1.4;
		color: #94a3b8;
		letter-spacing: 0.2rpx;
	}

	.flowchart-empty {
		margin-top: 18rpx;
		padding: 34rpx 28rpx;
		border-radius: 24rpx;
		background:
			radial-gradient(circle at top left, rgba(255, 255, 255, 0.34), rgba(255, 255, 255, 0) 44%),
			linear-gradient(135deg, #f9f4ee, #f2e8dd);
		border: 1px dashed rgba(180, 102, 76, 0.2);
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}

	.flowchart-empty--disabled {
		opacity: 0.72;
	}

	.flowchart-empty__icon {
		width: 78rpx;
		height: 78rpx;
		border-radius: 22rpx;
		background: rgba(255, 255, 255, 0.72);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.flowchart-empty__title {
		margin-top: 18rpx;
		font-size: 27rpx;
		font-weight: 700;
		color: #4e4339;
	}

	.flowchart-empty__desc {
		margin-top: 10rpx;
		font-size: 23rpx;
		line-height: 1.6;
		color: #8f8275;
	}

	.detail-parse {
		margin-top: 18rpx;
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18rpx;
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.4);
	}

	.detail-parse--pending,
	.detail-parse--processing {
		background:
			linear-gradient(180deg, rgba(249, 244, 233, 0.98), rgba(245, 237, 221, 0.94));
		border: 1px solid rgba(195, 150, 89, 0.16);
	}

	.detail-parse--done {
		background:
			linear-gradient(180deg, rgba(242, 248, 241, 0.98), rgba(233, 241, 232, 0.94));
		border: 1px solid rgba(111, 130, 109, 0.16);
	}

	.detail-parse--failed {
		background:
			linear-gradient(180deg, rgba(252, 241, 237, 0.98), rgba(248, 232, 226, 0.94));
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
		padding: 8rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.28);
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

	.parsed-section {
		margin-top: 24rpx;
	}

	.parsed-section--steps {
		margin-top: 32rpx;
	}

	/* B1-2 / B2-6：分组标题行支持「标题 + 右侧操作」布局 */
	.parsed-section__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12rpx;
	}

	.parsed-section__title {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-height: 42rpx;
		padding: 0 16rpx;
		border-radius: 999rpx;
		background: rgba(242, 235, 226, 0.88);
		border: 1px solid rgba(122, 98, 74, 0.08);
		font-size: 24rpx;
		font-weight: 600;
		color: #475569;
		letter-spacing: 0.2rpx;
	}

	/* B1-2：复制清单按钮 —— 与「来源链接 复制」一致的轻量文字按钮 */
	.parsed-section__copy {
		display: inline-flex;
		align-items: center;
		gap: 6rpx;
		min-height: 40rpx;
		padding: 0 14rpx;
		border-radius: 999rpx;
		background: rgba(244, 233, 218, 0.62);
		border: 1px solid rgba(180, 102, 76, 0.12);
		transition: transform 0.16s ease, background 0.16s ease;
	}

	.parsed-section__copy--hover {
		transform: scale(0.96);
		background: rgba(244, 233, 218, 0.88);
	}

	.parsed-section__copy-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #9a7343;
		letter-spacing: 0.2rpx;
	}

	/* B2-6：步骤进度提示 + 重置 */
	.parsed-section__progress {
		display: inline-flex;
		align-items: center;
		gap: 10rpx;
	}

	.parsed-section__progress-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #6f8266;
		letter-spacing: 0.4rpx;
	}

	.parsed-section__progress-reset {
		min-height: 36rpx;
		padding: 0 12rpx;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.6);
		border: 1px solid rgba(91, 74, 59, 0.12);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.16s ease, background 0.16s ease;
	}

	.parsed-section__progress-reset--hover {
		transform: scale(0.94);
		background: rgba(244, 233, 218, 0.88);
	}

	.parsed-section__progress-reset-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #94837a;
	}

	/* B1-1：主料紧凑列表（< 3 项时使用，避免单条「1」的视觉冗余） */
	.parsed-main-compact {
		margin-top: 14rpx;
		padding: 16rpx 20rpx;
		border-radius: 20rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.82), rgba(247, 243, 237, 0.92));
		border: 1px solid rgba(91, 74, 59, 0.08);
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}

	.parsed-main-compact__item {
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.parsed-main-compact__dot {
		width: 8rpx;
		height: 8rpx;
		border-radius: 50%;
		background: #b08c72;
		flex-shrink: 0;
	}

	.parsed-main-compact__text {
		flex: 1;
		min-width: 0;
		font-size: 26rpx;
		line-height: 1.6;
		color: #4d433a;
		font-weight: 500;
	}

	.parsed-item,
	.step-item {
		margin-top: 14rpx;
		display: flex;
		align-items: flex-start;
		gap: 14rpx;
	}

	.parsed-item {
		padding: 16rpx 18rpx;
		border-radius: 22rpx;
		background: rgba(255, 255, 255, 0.82);
		border: 1px solid rgba(91, 74, 59, 0.07);
	}

	.parsed-group {
		margin-top: 14rpx;
		padding: 18rpx 20rpx;
		border-radius: 20rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.82), rgba(247, 243, 237, 0.92));
		border: 1px solid rgba(91, 74, 59, 0.08);
	}

	.parsed-group__label {
		display: inline-flex;
		padding: 6rpx 14rpx;
		border-radius: 999rpx;
		background:
			linear-gradient(180deg, rgba(245, 239, 231, 0.98), rgba(236, 227, 216, 0.96));
		border: 1px solid rgba(122, 98, 74, 0.08);
		font-size: 20rpx;
		font-weight: 700;
		line-height: 1.2;
		color: #7a6c60;
	}

	.parsed-group__text {
		display: block;
		margin-top: 12rpx;
		font-size: 25rpx;
		line-height: 1.7;
		color: #4d433a;
	}

	.parsed-item__index {
		width: 40rpx;
		height: 40rpx;
		border-radius: 12rpx;
		background:
			linear-gradient(180deg, rgba(244, 238, 230, 0.98), rgba(235, 226, 215, 0.96));
		border: 1px solid rgba(122, 98, 74, 0.08);
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

	.parsed-item__text {
		flex: 1;
		min-width: 0;
		font-size: 25rpx;
		line-height: 1.7;
		color: #4d433a;
	}

	.step-item__body {
		flex: 1;
		min-width: 0;
	}

	.step-item {
		padding: 18rpx;
		border-radius: 24rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.86), rgba(248, 245, 239, 0.94));
		border: 1px solid rgba(91, 74, 59, 0.08);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.5);
		transition: opacity 0.2s ease, transform 0.16s ease, background 0.2s ease;
	}

	/* B2-6：步骤已完成态 —— 整体淡出，标题加灰，让用户的视觉自然跳过已做完的步骤 */
	.step-item--done {
		opacity: 0.55;
		background:
			linear-gradient(180deg, rgba(247, 246, 243, 0.86), rgba(243, 240, 234, 0.94));
	}

	.step-item--done .step-item__title {
		color: #8a8278;
		text-decoration: line-through;
		text-decoration-color: rgba(138, 130, 120, 0.5);
	}

	.step-item--hover {
		transform: scale(0.992);
	}

	.step-item__title {
		display: block;
		font-size: 26rpx;
		font-weight: 700;
		line-height: 1.4;
		color: #2f2923;
	}

	.step-item__text {
		display: block;
		margin-top: 8rpx;
		font-size: 25rpx;
		line-height: 1.7;
		color: #4d433a;
	}

	/* B2-5：步骤详情中关键参数高亮（时间/克数/火候） */
	.step-item__text--normal {
		color: #4d433a;
		font-weight: 400;
	}

	.step-item__text--highlight {
		color: #b4664c;
		font-weight: 700;
	}

	.step-item--done .step-item__text--highlight {
		color: #a89a8a;
		font-weight: 600;
	}

	/* B1-4：序号去掉「Step」前缀后改为正方形小徽章；B2-6：完成时变绿底承载对勾 */
	.step-item__index {
		flex-shrink: 0;
		width: 48rpx;
		height: 48rpx;
		padding: 0;
		box-sizing: border-box;
		border-radius: 14rpx;
		background:
			linear-gradient(180deg, rgba(245, 239, 231, 0.98), rgba(236, 226, 214, 0.96));
		border: 1px solid rgba(124, 104, 84, 0.08);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: background 0.2s ease, border-color 0.2s ease;
	}

	.step-item__index--done {
		background:
			linear-gradient(180deg, rgba(232, 240, 230, 0.98), rgba(218, 230, 215, 0.96));
		border-color: rgba(111, 130, 102, 0.24);
	}

	.step-item__index-text {
		display: block;
		font-size: 24rpx;
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
		background:
			linear-gradient(180deg, rgba(246, 244, 241, 0), rgba(246, 244, 241, 0.9) 18%, rgba(255, 252, 248, 0.98) 42%);
		display: flex;
		gap: 16rpx;
	}

	.detail-footer__action {
		flex: 1;
		height: 88rpx;
		border-radius: 24rpx;
		border: 1px solid rgba(100, 78, 58, 0.06);
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.94), rgba(243, 237, 230, 0.96));
		box-shadow:
			0 12rpx 22rpx rgba(70, 54, 40, 0.045),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
		display: flex;
		align-items: center;
		justify-content: center;
		transform: scale(1);
		transition: transform 0.16s ease, box-shadow 0.16s ease, background 0.16s ease, border-color 0.16s ease;
	}

	.detail-footer__action:active {
		transform: scale(0.988);
	}

	.detail-footer__action--ghost {
		background:
			linear-gradient(180deg, rgba(255, 247, 244, 0.98), rgba(246, 237, 232, 0.96));
		border-color: rgba(180, 102, 76, 0.1);
	}

	.detail-footer__action--primary {
		background:
			linear-gradient(180deg, #6d5846, #594736);
		border-color: rgba(89, 71, 54, 0.8);
		box-shadow:
			0 16rpx 24rpx rgba(91, 74, 59, 0.18),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.12);
	}

	.detail-footer__action--soft {
		background:
			linear-gradient(180deg, rgba(255, 252, 246, 0.98), rgba(243, 235, 222, 0.96));
	}

	.detail-footer__action--soft-active {
		background:
			linear-gradient(180deg, rgba(255, 248, 236, 0.98), rgba(246, 235, 211, 0.96));
		border-color: rgba(186, 145, 81, 0.16);
		box-shadow:
			0 12rpx 22rpx rgba(111, 85, 45, 0.06),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.48);
	}

	.detail-footer__action--delete {
		flex: 0 0 96rpx;
		background: rgba(255, 255, 255, 0.86);
		border-color: rgba(180, 102, 76, 0.18);
		box-shadow:
			0 8rpx 16rpx rgba(180, 102, 76, 0.05),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
	}

	.detail-footer__action--pin {
		flex: 1;
	}

	.detail-footer__action--edit {
		flex: 1.6;
	}

	.detail-footer__action--disabled {
		opacity: 0.62;
		pointer-events: none;
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

	.detail-footer__text--accent {
		color: #9a7343;
	}

	.editor-sheet {
		height: 78vh;
		background:
			radial-gradient(circle at top right, rgba(255, 236, 214, 0.26) 0%, rgba(255, 236, 214, 0) 26%),
			linear-gradient(180deg, #fbf8f4 0%, #f7f3ee 100%);
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
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(242, 235, 227, 0.94));
		box-shadow:
			0 8rpx 18rpx rgba(70, 54, 40, 0.04),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.66);
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

	.editor-field__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16rpx;
	}

	.editor-field__label {
		font-size: 22rpx;
		font-weight: 500;
		color: #9b9186;
	}

	.editor-field__meta {
		flex-shrink: 0;
		font-size: 20rpx;
		font-weight: 600;
		color: #8f8377;
	}

	.editor-field__hint {
		font-size: 22rpx;
		line-height: 1.6;
		color: #9b9186;
	}

	.editor-input,
	.editor-textarea {
		width: 100%;
		box-sizing: border-box;
		border-radius: 24rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.84), rgba(247, 243, 237, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.08);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.58);
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
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(252, 248, 243, 0.96));
		border-color: rgba(170, 134, 103, 0.12);
		box-shadow:
			0 10rpx 18rpx rgba(70, 54, 40, 0.03),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.66);
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

	.editor-structured {
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}

	.editor-structured__section {
		padding: 22rpx;
		border-radius: 28rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(248, 243, 236, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.06);
		box-shadow:
			0 10rpx 18rpx rgba(70, 54, 40, 0.03),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.58);
	}

	.editor-structured__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
	}

	.editor-structured__heading {
		flex: 1;
		min-width: 0;
	}

	.editor-structured__title {
		display: block;
		font-size: 26rpx;
		font-weight: 700;
		color: #312b24;
	}

	.editor-structured__desc {
		display: block;
		margin-top: 6rpx;
		font-size: 21rpx;
		line-height: 1.5;
		color: #9b9186;
	}

	.editor-structured__action {
		flex-shrink: 0;
		height: 56rpx;
		padding: 0 18rpx;
		border-radius: 999rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(247, 242, 236, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.08);
		box-shadow:
			0 8rpx 14rpx rgba(68, 52, 38, 0.035),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.6);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.editor-structured__action-text {
		font-size: 22rpx;
		font-weight: 600;
		color: #675b4f;
	}

	.editor-structured__empty {
		margin-top: 16rpx;
		padding: 24rpx 20rpx;
		border-radius: 24rpx;
		background: rgba(255, 255, 255, 0.74);
		border: 1px dashed #ddd3c7;
	}

	.editor-structured__empty--large {
		padding: 36rpx 24rpx;
		text-align: center;
	}

	.editor-structured__empty-text {
		font-size: 23rpx;
		line-height: 1.6;
		color: #8f8377;
	}

	.editor-ingredient-list {
		margin-top: 16rpx;
		display: flex;
		flex-direction: column;
		gap: 12rpx;
	}

	.editor-ingredient-item {
		min-height: 84rpx;
		padding: 0 16rpx;
		border-radius: 24rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(249, 245, 240, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.06);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.52);
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.editor-ingredient-item__index {
		width: 52rpx;
		height: 52rpx;
		border-radius: 16rpx;
		background: #f3ece4;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.editor-ingredient-item__index-text {
		font-size: 21rpx;
		font-weight: 700;
		color: #6f6153;
	}

	.editor-ingredient-item__input {
		flex: 1;
		min-width: 0;
		height: 100%;
		font-size: 26rpx;
		color: #2f2923;
	}

	.editor-ingredient-item__menu {
		width: 52rpx;
		height: 52rpx;
		border-radius: 16rpx;
		background:
			linear-gradient(180deg, rgba(244, 238, 230, 0.98), rgba(235, 226, 215, 0.96));
		border: 1px solid rgba(122, 98, 74, 0.08);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.editor-ingredient-item__menu-dots {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 5rpx;
	}

	.editor-ingredient-item__menu-dot {
		width: 6rpx;
		height: 6rpx;
		border-radius: 999rpx;
		background: #74685c;
		flex-shrink: 0;
	}

	.editor-step-list {
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}

	.editor-step-card {
		padding: 22rpx;
		border-radius: 28rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(248, 243, 236, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.06);
		box-shadow:
			0 10rpx 18rpx rgba(70, 54, 40, 0.03),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.58);
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}

	.editor-step-card__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16rpx;
	}

	.editor-step-card__badge {
		height: 52rpx;
		padding: 0 16rpx;
		border-radius: 999rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(247, 242, 236, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.08);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.56);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.editor-step-card__badge-text {
		font-size: 21rpx;
		font-weight: 700;
		color: #685c50;
	}

	.editor-step-card__actions {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		flex-wrap: wrap;
		gap: 10rpx;
	}

	.editor-step-card__action {
		height: 52rpx;
		padding: 0 18rpx;
		border-radius: 999rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(247, 242, 236, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.08);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.56);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.editor-step-card__action--disabled {
		opacity: 0.42;
		pointer-events: none;
	}

	.editor-step-card__action--danger {
		background:
			linear-gradient(180deg, rgba(252, 241, 237, 0.98), rgba(248, 232, 226, 0.94));
		border-color: rgba(193, 106, 81, 0.14);
	}

	.editor-step-card__action-text {
		font-size: 21rpx;
		font-weight: 600;
		color: #6b5e52;
	}

	.editor-step-card__action-text--danger {
		color: #b4664c;
	}

	.editor-step-card__field {
		display: flex;
		flex-direction: column;
		gap: 10rpx;
	}

	.editor-step-card__label {
		font-size: 21rpx;
		font-weight: 500;
		color: #988d81;
	}

	.editor-step-card__input,
	.editor-step-card__textarea {
		width: 100%;
		box-sizing: border-box;
		border-radius: 22rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(249, 245, 240, 0.94));
		border: 1px solid rgba(111, 86, 64, 0.06);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.52);
		color: #2f2923;
	}

	.editor-step-card__input {
		height: 82rpx;
		padding: 0 22rpx;
		font-size: 26rpx;
	}

	.editor-step-card__textarea {
		min-height: 144rpx;
		padding: 20rpx 22rpx;
		font-size: 25rpx;
		line-height: 1.65;
	}

	.editor-step-add {
		height: 84rpx;
		border-radius: 24rpx;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(248, 243, 236, 0.94));
		border: 1px dashed rgba(150, 126, 104, 0.26);
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.52);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.editor-step-add__text {
		font-size: 24rpx;
		font-weight: 600;
		color: #75685c;
	}

	.editor-gallery {
		display: flex;
		flex-wrap: wrap;
		gap: 16rpx;
	}

	.editor-gallery__item,
	.editor-gallery__add {
		position: relative;
		width: calc((100% - 32rpx) / 3);
		height: 176rpx;
		box-sizing: border-box;
		border-radius: 24rpx;
		overflow: hidden;
	}

	.editor-gallery__item {
		background: linear-gradient(145deg, #ebdfd3 0%, #e1d3c4 100%);
		box-shadow:
			0 10rpx 18rpx rgba(66, 51, 37, 0.07),
			inset 0 0 0 1px rgba(255, 255, 255, 0.24);
	}

	.editor-gallery__thumb {
		width: 100%;
		height: 100%;
		display: block;
	}

	.editor-gallery__badge {
		position: absolute;
		left: 12rpx;
		bottom: 12rpx;
		padding: 8rpx 14rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.58);
		backdrop-filter: blur(10rpx);
	}

	.editor-gallery__badge-text {
		font-size: 20rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.editor-gallery__sort {
		position: absolute;
		top: 12rpx;
		left: 12rpx;
		height: 40rpx;
		padding: 0 14rpx;
		border-radius: 999rpx;
		background: rgba(47, 41, 35, 0.56);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.editor-gallery__sort-text {
		font-size: 19rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.editor-gallery__remove {
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

	.editor-gallery__add {
		border: 1px dashed rgba(150, 126, 104, 0.26);
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(248, 243, 236, 0.94));
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.58);
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12rpx;
	}

	.editor-gallery__plus {
		width: 64rpx;
		height: 64rpx;
		border-radius: 20rpx;
		background:
			linear-gradient(180deg, rgba(244, 238, 230, 0.98), rgba(235, 226, 215, 0.96));
		border: 1px solid rgba(122, 98, 74, 0.08);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.editor-gallery__add-text {
		font-size: 24rpx;
		font-weight: 600;
		color: #75685c;
	}

	.segment {
		display: flex;
		gap: 10rpx;
		padding: 8rpx;
		border-radius: 24rpx;
		background:
			linear-gradient(180deg, rgba(243, 239, 234, 0.96), rgba(238, 232, 224, 0.92));
		box-shadow: inset 0 1rpx 0 rgba(255, 255, 255, 0.46);
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
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(249, 245, 240, 0.96));
		box-shadow:
			0 10rpx 18rpx rgba(59, 47, 36, 0.055),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
	}

	.segment__item--wishlist {
		background:
			linear-gradient(180deg, rgba(250, 240, 233, 0.98), rgba(243, 231, 222, 0.96));
	}

	.segment__item--done {
		background:
			linear-gradient(180deg, rgba(240, 247, 239, 0.98), rgba(232, 239, 229, 0.96));
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
		background:
			linear-gradient(180deg, rgba(248, 244, 239, 0.4), rgba(255, 255, 255, 0.96) 32%);
		display: flex;
		gap: 16rpx;
	}

	.editor-sheet__action {
		flex: 1;
		height: 88rpx;
		border-radius: 24rpx;
		border: 1px solid rgba(100, 78, 58, 0.06);
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.94), rgba(243, 237, 230, 0.96));
		box-shadow:
			0 12rpx 22rpx rgba(70, 54, 40, 0.045),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.62);
		display: flex;
		align-items: center;
		justify-content: center;
		transform: scale(1);
		transition: transform 0.16s ease, box-shadow 0.16s ease, background 0.16s ease, border-color 0.16s ease;
	}

	.editor-sheet__action:active {
		transform: scale(0.988);
	}

	.editor-sheet__action--primary {
		background:
			linear-gradient(180deg, #6d5846, #594736);
		border-color: rgba(89, 71, 54, 0.8);
		box-shadow:
			0 16rpx 24rpx rgba(91, 74, 59, 0.18),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.12);
	}

	.editor-sheet__action--disabled {
		background:
			linear-gradient(180deg, #ddd4ca, #d5cbc0);
		border-color: rgba(174, 159, 143, 0.68);
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

	.detail-loading {
		padding: 28rpx 24rpx calc(env(safe-area-inset-bottom) + 188rpx);
		display: flex;
		flex-direction: column;
		gap: 20rpx;
	}

	.detail-loading__hero,
	.detail-loading__card,
	.detail-loading__section {
		border-radius: 30rpx;
		background: rgba(255, 253, 249, 0.96);
		border: 1px solid rgba(100, 78, 58, 0.05);
		box-shadow:
			0 14rpx 30rpx rgba(70, 54, 40, 0.045),
			inset 0 1rpx 0 rgba(255, 255, 255, 0.68);
	}

	.detail-loading__hero {
		height: 380rpx;
	}

	.detail-loading__section,
	.detail-loading__card {
		padding: 28rpx;
	}

	.detail-loading__chips {
		display: flex;
		align-items: center;
		gap: 12rpx;
	}

	.detail-loading__chip {
		width: 116rpx;
		height: 48rpx;
		border-radius: 999rpx;
	}

	.detail-loading__chip--short {
		width: 92rpx;
	}

	.detail-loading__title {
		margin-top: 22rpx;
		width: 58%;
		height: 44rpx;
		border-radius: 18rpx;
	}

	.detail-loading__card-title,
	.detail-loading__line,
	.detail-loading__row {
		border-radius: 18rpx;
	}

	.detail-loading__card-title {
		width: 32%;
		height: 32rpx;
	}

	.detail-loading__line {
		margin-top: 18rpx;
		width: 100%;
		height: 26rpx;
	}

	.detail-loading__line--short {
		width: 72%;
	}

	.detail-loading__row {
		margin-top: 18rpx;
		width: 100%;
		height: 86rpx;
	}

	.detail-loading__row--short {
		width: 84%;
	}

	.detail-loading__pulse {
		position: relative;
		overflow: hidden;
		background:
			linear-gradient(90deg, rgba(240, 233, 225, 0.88) 0%, rgba(255, 249, 243, 0.96) 48%, rgba(240, 233, 225, 0.88) 100%);
		background-size: 220% 100%;
		animation: detail-loading-shimmer 1.22s ease-in-out infinite;
	}

	@keyframes detail-loading-shimmer {
		0% {
			background-position: 100% 50%;
		}
		100% {
			background-position: 0 50%;
		}
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
