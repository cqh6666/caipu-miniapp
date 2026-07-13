<template>
  <AppShell>
    <template #toolbar>
      <div class="toolbar-cluster toolbar-cluster--publish">
        <StatusTag
          v-if="draftScene"
          :tone="bottomBarState.tone"
          :text="statusBarText"
        />
        <el-button
          v-if="draftScene"
          :loading="testingScene"
          :disabled="!draftScene"
          @click="handleTestScene"
        >
          测试草稿
        </el-button>
        <el-button
          v-if="draftScene && isDirty"
          type="primary"
          :loading="savingScene"
          :disabled="savingScene"
          @click="handleSaveScene"
        >
          保存并发布
        </el-button>
        <el-tooltip :content="discardTooltip" placement="bottom">
          <span v-if="draftScene && isDirty" class="toolbar-discard-wrap">
            <el-button
              type="danger"
              text
              :disabled="!isDirty"
              @click="handleDiscardDraft"
            >
              放弃草稿
            </el-button>
          </span>
        </el-tooltip>
        <el-tooltip content="重新拉取远端配置" placement="bottom">
          <el-button
            circle
            :loading="pageRefreshing"
            aria-label="刷新"
            @click="refreshPage"
          >
            <el-icon><Refresh /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </template>

    <button type="button" class="skip-link" @click="focusMainEditor">
      跳到主编辑区
    </button>

    <div
      ref="topCtaSentinelRef"
      class="routing-top-cta-sentinel"
      aria-hidden="true"
    ></div>

    <section
      id="ai-provider-scene-cards"
      :ref="(el) => setAnchorSectionRef('scene-cards', el)"
      class="routing-anchor-section"
    >
      <SceneOverviewCards
        ref="sceneOverviewRef"
        :cards="sceneCards"
        :current-scene="currentSceneKey"
        :show-shortcut="viewportWidth >= 1024"
        :shortcut-prev="shortcutTokens('scene-prev')"
        :shortcut-next="shortcutTokens('scene-next')"
        @select="handleSceneChange"
      />
    </section>

    <div v-if="sceneLoading && !draftScene" class="page-card routing-panel">
      <PageState mode="loading" title="正在加载场景配置" />
    </div>
    <div v-else-if="sceneError && !draftScene" class="page-card routing-panel">
      <PageState
        mode="error"
        title="场景配置加载失败"
        :description="sceneError"
        @retry="loadCurrentScene"
      />
    </div>
    <template v-else-if="draftScene">
      <section
        id="ai-provider-main-editor"
        ref="mainEditorRef"
        class="routing-main-editor"
        tabindex="-1"
        aria-label="AI Provider 主编辑区"
      >
        <el-alert
          v-if="draftScene.compatibilityMode"
          class="routing-alert"
          type="warning"
          :closable="false"
          title="当前仍处于兼容模式"
          :description="compatibilityHint"
        />

        <div class="routing-breadcrumb" aria-live="polite">
          <span class="routing-breadcrumb__crumbs">
            场景策略
            <el-icon class="routing-breadcrumb__sep"><ArrowRight /></el-icon>
            正在编辑：<strong>{{ currentSceneTitle }}</strong>
          </span>
          <div class="routing-breadcrumb__actions">
            <el-popover
              v-if="showAnchorDirectory"
              placement="bottom-end"
              :width="320"
              trigger="click"
            >
              <template #reference>
                <el-button
                  text
                  class="routing-breadcrumb__directory-button"
                  aria-label="打开页面目录"
                >
                  页面目录
                </el-button>
              </template>
              <div class="routing-utility-panel">
                <div class="routing-utility-panel__header">
                  <strong>页面目录</strong>
                  <span>当前高亮会随滚动位置更新</span>
                </div>
                <div class="routing-utility-list">
                  <button
                    v-for="item in anchorNavItems"
                    :key="item.key"
                    type="button"
                    class="routing-utility-list__item"
                    :class="{
                      'routing-utility-list__item--active':
                        item.key === activeAnchorKey,
                    }"
                    @click="scrollToAnchorSection(item.key)"
                  >
                    <span>{{ item.label }}</span>
                    <span
                      v-if="item.key === activeAnchorKey"
                      class="routing-utility-list__badge"
                    >
                      当前
                    </span>
                  </button>
                </div>
              </div>
            </el-popover>
            <StatusTag :tone="bottomBarState.tone" :text="statusBarText" />
          </div>
        </div>

        <div
          class="routing-status-strip"
          :class="`routing-status-strip--${bottomBarState.tone}`"
          aria-live="polite"
        >
          <div class="routing-status-strip__main">
            <div>
              <strong>{{ currentEffectHeadline }}</strong>
              <p>{{ currentEffectDescription }}</p>
            </div>
            <div class="routing-status-strip__tags">
              <StatusTag :tone="currentChannel.tone" :text="currentChannel.label" />
              <StatusTag
                :tone="alertStatusSummary.tone"
                :text="alertStatusSummary.text"
              />
              <StatusTag
                v-if="isDirty"
                tone="warning"
                :text="`${diffCount} 项未保存`"
              />
            </div>
          </div>
          <div class="routing-status-strip__actions">
            <el-popover
              placement="bottom-end"
              :width="mediumPopoverWidth"
              trigger="click"
            >
              <template #reference>
                <el-button link>查看原因</el-button>
              </template>
              <div class="channel-popover">
                <div class="channel-popover__title">
                  {{ currentSceneTitle }} 当前生效逻辑
                </div>
                <p class="channel-popover__text">
                  {{ currentChannel.reason }}
                </p>
                <table class="channel-popover__table">
                  <thead>
                    <tr>
                      <th>状态</th>
                      <th>新路由</th>
                      <th>用户可见结论</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="(row, idx) in channelMatrix"
                      :key="idx"
                      :class="{ 'is-hit': row.hit }"
                    >
                      <td>{{ row.draft }}</td>
                      <td>{{ row.toggle }}</td>
                      <td>{{ row.effect }}</td>
                    </tr>
                  </tbody>
                </table>
                <div class="channel-popover__tech">
                  技术标识：<code>{{ currentChannel.technicalLabel }}</code>
                </div>
              </div>
            </el-popover>
            <el-popover
              placement="bottom-end"
              :width="largePopoverWidth"
              trigger="click"
            >
              <template #reference>
                <el-button link>告警配置</el-button>
              </template>
              <AlertLifecyclePanel
                :scene-title="currentSceneTitle"
                :description="alertStatusDescription"
                :overview="alertOverview"
                :sections="currentSceneAlertSections"
                :has-nodes="hasCurrentSceneAlertNodes"
                :pending-actions="pendingAlertActions"
                @retest="handleAlertRetest"
                @archive="handleAlertArchive"
                @mute="handleAlertMute"
                @unmute="handleAlertUnmute"
                @batch-retest="handleBatchRetest"
                @batch-archive="handleBatchArchive"
                @logs="goAlertProviderLogs"
                @config="goAlertConfig"
                @copy-request-id="copyRequestId"
              />
            </el-popover>
          </div>
        </div>

        <div
          v-if="blockingRiskItems.length"
          class="routing-risk-strip"
          aria-live="polite"
        >
          <article
            v-for="item in blockingRiskItems"
            :key="item.key"
            class="routing-risk-strip__item"
            :class="`routing-risk-strip__item--${item.tone}`"
          >
            <StatusTag :tone="item.tone" :text="item.title" />
            <span class="routing-risk-strip__text">{{ item.description }}</span>
          </article>
        </div>

        <div class="routing-editor-grid">
          <div
            :ref="(el) => setAnchorSectionRef('scene-strategy', el)"
            class="routing-anchor-section"
          >
            <SceneStrategyPanel
              :draft-scene="draftScene"
              :current-channel="currentChannel"
              :help-tips="helpTips"
              :minimum-max-attempts="minimumMaxAttempts"
              :max-attempt-ceiling="maxAttemptCeiling"
              :numeric-warn="numericWarn"
              :timeline-segments="timelineSegments"
              :expected-first-round-seconds="expectedFirstRoundSeconds"
              :retry-options="retryOptions"
            />
          </div>

          <div
            :ref="(el) => setAnchorSectionRef('provider-nodes', el)"
            class="page-card routing-panel routing-anchor-section"
          >
            <ProviderEditor
              :draft-scene="draftScene"
              :enabled-provider-count="enabledProviderCount"
              :help-tips="helpTips"
              :provider-preset-options="providerPresetOptions"
              :single-test-provider-id="singleTestProviderId"
              :testing-scene="testingScene"
              :saving-scene="savingScene"
              :select-options="providerSelectOptions"
              :editor="providerEditorContract"
            />
          </div>
        </div>

        <div
          v-if="testResult"
          ref="testCardRef"
          :ref="(el) => setAnchorSectionRef('latest-test', el)"
          class="page-card routing-test-card routing-anchor-section"
        >
          <RouteTestResult
            :result="testResult"
            :summary="testResultSummary"
            :scope-description="testScopeDescription"
            :provider-name="providerDisplayName"
            :display-message="displayRouteTestMessage"
          />
        </div>

        <div
          :ref="(el) => setAnchorSectionRef('recent-audits', el)"
          class="page-card audit-section routing-anchor-section"
        >
          <AuditTimeline
            v-model:kind-filter="recentAuditKindFilter"
            :recent-audits="recentAudits"
            :current-audit-group="currentAuditGroup"
            :kind-options="recentAuditKindOptions"
            :audits-loading="auditsLoading"
            :recent-audits-loading="recentAuditsLoading"
            :recent-audits-error="recentAuditsError"
            :large-popover-width="largePopoverWidth"
            :help-tip="helpTips.audit"
            :tone-for-audit-action="toneForAuditAction"
            :audit-business-action="auditBusinessAction"
            :audit-target-title="auditTargetTitle"
            :audit-event-label="auditEventLabel"
            :audit-diff-title="auditDiffTitle"
            :audit-diff-status-text="auditDiffStatusText"
            :audit-fallback-summary="auditFallbackSummary"
            @open-full="auditDrawerVisible = true"
            @retry="loadRecentAudits"
          />
        </div>

        <AuditDrawer
          v-model:visible="auditDrawerVisible"
          v-model:action="auditAction"
          v-model:operator="auditOperator"
          v-model:setting-key="auditSettingKey"
          v-model:time-range="auditTimeRange"
          v-model:page-size="auditPageSize"
          v-model:page="auditPage"
          :audits="audits"
          :page-size-options="auditPageSizeOptions"
          :action-options="auditActionOptions"
          :active-filters="activeAuditFilters"
          :audits-loading="auditsLoading"
          :audit-diff-title="auditDiffTitle"
          :audit-event-label="auditEventLabel"
          :audit-business-action="auditBusinessAction"
          :audit-diff-status-text="auditDiffStatusText"
          :audit-fallback-summary="auditFallbackSummary"
          :audit-target-title="auditTargetTitle"
          @reset="resetAuditFilters"
          @apply="applyAuditFilters"
          @page-size-change="handleAuditPageSizeChange"
          @page-change="handleAuditPageChange"
        />

        <div
          class="routing-bottom-bar"
          :class="[
            `routing-bottom-bar--${bottomBarState.tone}`,
            { 'routing-bottom-bar--tucked': !floatingBarVisible },
          ]"
          :aria-hidden="floatingBarVisible ? 'false' : 'true'"
          aria-live="polite"
        >
          <div class="routing-bottom-bar__status">
            <el-icon aria-hidden="true"
              ><component :is="bottomBarState.icon"
            /></el-icon>
            <span>{{ bottomBarState.text }}</span>
          </div>
          <el-popover
            v-if="isDirty"
            placement="top"
            :width="smallPopoverWidth"
            trigger="click"
          >
            <template #reference>
              <el-button link>草稿摘要</el-button>
            </template>
            <div class="draft-summary-popover">
              <div
                v-for="(item, idx) in sceneDiff.slice(0, 6)"
                :key="idx"
                class="draft-summary-popover__item"
              >
                <strong>{{ item.scope }} · {{ item.path }}</strong>
                <span
                  >{{ formatDiffValue(item.from) }} →
                  {{ formatDiffValue(item.to) }}</span
                >
              </div>
              <div
                v-if="sceneDiff.length > 6"
                class="draft-summary-popover__more"
              >
                另有 {{ sceneDiff.length - 6 }} 项改动
              </div>
            </div>
          </el-popover>
          <div class="routing-bottom-bar__actions">
            <el-button
              :loading="testingScene"
              :disabled="!draftScene"
              @click="handleTestScene"
              >测试草稿</el-button
            >
            <el-button
              v-if="isDirty"
              text
              type="danger"
              :disabled="!isDirty"
              @click="handleDiscardDraft"
              >放弃草稿</el-button
            >
            <el-button
              type="primary"
              :loading="savingScene"
              :disabled="!isDirty || savingScene"
              @click="handleSaveScene"
              >保存并发布</el-button
            >
          </div>
        </div>
      </section>
    </template>
  </AppShell>
</template>

<script setup lang="ts">
import {
  computed,
  h,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowRight,
  Check,
  Clock,
  Refresh,
  Warning,
} from "@element-plus/icons-vue";
import AppShell from "@/components/AppShell.vue";
import PageState from "@/components/PageState.vue";
import StatusTag from "@/components/StatusTag.vue";
import AlertLifecyclePanel from "@/components/ai-providers/AlertLifecyclePanel.vue";
import AuditDrawer from "@/components/ai-providers/AuditDrawer.vue";
import AuditTimeline from "@/components/ai-providers/AuditTimeline.vue";
import ProviderEditor from "@/components/ai-providers/ProviderEditor.vue";
import RouteTestResult from "@/components/ai-providers/RouteTestResult.vue";
import SaveSceneConfirmContent from "@/components/ai-providers/SaveSceneConfirmContent.vue";
import SceneOverviewCards from "@/components/ai-providers/SceneOverviewCards.vue";
import SceneStrategyPanel from "@/components/ai-providers/SceneStrategyPanel.vue";
import * as adminApi from "@/api/admin";
import { useResponsive } from "@/composables/useResponsive";
import { useAIRoutingDraft } from "@/composables/useAIRoutingDraft";
import { useAIRoutingAlerts } from "@/composables/useAIRoutingAlerts";
import { useAIProviderEditor } from "@/composables/useAIProviderEditor";
import type {
  AIRoutingAlertOverview,
  AIRoutingProviderEndpointMode,
  AIRoutingProviderConfig,
  AIRoutingProviderResponseFormat,
  AIRoutingSceneConfig,
  AIRoutingSceneKey,
  AIRoutingSceneSummary,
  AIRoutingTestResult,
  PaginationResult,
  SceneCardHealthSnapshot,
  SettingAuditRecord,
} from "@/types";
import {
  buildRouteQuery,
  readQueryString,
  type DateRangeValue,
} from "@/utils/route-query";
import {
  auditActionOptions,
  displayAIRoutingScene,
  displayAIRoutingStrategy,
  displayAuditAction,
  displayCallStatus,
  displaySettingSource,
  formatDateTime,
  formatDuration,
} from "@/utils/admin-display";
import {
  sceneAggregateStatus,
  summarizeSceneAlertStatus,
  summarizeSceneIssue,
} from "@/utils/ai-provider-alerts";
import {
  createAIProviderAuditPresenter,
  formatDiffValue,
} from "@/utils/ai-provider-audit";
import {
  buildScenePayload,
  createProvider,
  hydrateScene,
  normalizeImageOutputFormat,
  normalizeProviderExtra,
  normalizeReasoningEffort,
  normalizeThinkingType,
} from "@/utils/ai-provider-draft";
import { buildSaveSceneConfirmModel } from "@/utils/ai-provider-save-confirm";
import {
  buildProviderValidationErrors,
  buildSceneBlockingValidationMessages,
  countProviderValidationErrors,
  providerHasUsableSecret,
  type ProviderValidationError,
} from "@/utils/ai-provider-validation";

const router = useRouter();
const route = useRoute();
const { width: viewportWidth } = useResponsive();

type AnchorSectionKey =
  | "scene-cards"
  | "scene-strategy"
  | "provider-nodes"
  | "latest-test"
  | "recent-audits";

const sceneKeys: AIRoutingSceneKey[] = ["summary", "title", "flowchart"];
const currentSceneKey = ref<AIRoutingSceneKey>("summary");
const sceneSummaries = ref<AIRoutingSceneSummary[]>([]);
const remoteScene = ref<AIRoutingSceneConfig | null>(null);
const draftScene = ref<AIRoutingSceneConfig | null>(null);
const testResult = ref<AIRoutingTestResult | null>(null);
const testScope = ref("");
const sceneDetailMap = ref<
  Partial<Record<AIRoutingSceneKey, AIRoutingSceneConfig>>
>({});
const sceneLatestTestAuditMap = ref<
  Partial<Record<AIRoutingSceneKey, SettingAuditRecord | null>>
>({});
const alertOverview = ref<AIRoutingAlertOverview | null>(null);
const activeAnchorKey = ref<AnchorSectionKey>("scene-cards");
const isMacLikePlatform = ref(false);
const {
  diffCount,
  isDirty,
  pendingClearKeyCount,
  pendingRemovedProviderCount,
  sceneDiff,
} = useAIRoutingDraft({ draftScene, remoteScene });

// 顶部工具栏 CTA 是否已滚出视口：只有滚出后才浮现底部悬浮操作栏，
// 避免顶部与底部同时出现两套完全相同的「测试草稿 / 保存并发布 / 放弃草稿」。
const topCtaSentinelRef = ref<HTMLElement | null>(null);
const floatingBarVisible = ref(false);
let topCtaObserver: IntersectionObserver | null = null;

const sceneLoading = ref(false);
const sceneError = ref("");
const pageRefreshing = ref(false);
const savingScene = ref(false);
const testingScene = ref(false);
const singleTestProviderId = ref("");
const testCardRef = ref<HTMLElement | null>(null);
const mainEditorRef = ref<HTMLElement | null>(null);
const sceneOverviewRef = ref<InstanceType<typeof SceneOverviewCards> | null>(
  null,
);
const routeSceneOverride = ref<AIRoutingSceneKey | null>(null);
const shouldFocusSceneAfterChange = ref(false);
const anchorSectionRefs: Partial<Record<AnchorSectionKey, HTMLElement | null>> =
  {};
let sceneHealthLoadToken = 0;

const audits = ref<PaginationResult<SettingAuditRecord>>({
  items: [],
  total: 0,
  page: 1,
  pageSize: 20,
});
const recentAuditItems = ref<SettingAuditRecord[]>([]);
const auditsLoading = ref(false);
const recentAuditsLoading = ref(false);
const auditsError = ref("");
const recentAuditsError = ref("");
const recentAuditKindFilter = ref<"all" | "changes" | "tests">("all");
const auditAction = ref("");
const auditOperator = ref("");
const auditSettingKey = ref("");
const auditTimeRange = ref<DateRangeValue>([]);
const auditPageSize = ref(20);
const auditPage = ref(1);
const auditDrawerVisible = ref(false);
const testingStartedAt = ref<number | null>(null);
const testingElapsedSeconds = ref(0);
let testingTimer: ReturnType<typeof window.setInterval> | null = null;

type BlockingRiskItem = {
  key: string;
  tone: "warning" | "danger";
  title: string;
  description: string;
};

type ProviderPresetKey = "openai-text" | "openai-image";

const helpTips = {
  sceneStrategy:
    "调度策略决定多节点失败后如何切换；详细规则可在当前生效逻辑中查看。",
  maxAttempts: "建议 2-5 次。尝试次数越大，异常降级链路等待越久。",
  breaker: "连续失败触发熔断后，节点会在冷却时间内跳过。",
  requestOptions:
    "标题清洗场景使用这些请求参数；其他场景由业务层固定 prompt 控制。",
  providerName:
    "用于页面展示、测试结果和审计记录；同一场景内不可重复。内部 ID 由系统自动生成并隐藏。",
  baseURL: "填写兼容 OpenAI 协议的服务地址，例如 https://api.example.com/v1。",
  endpoint:
    "图片生成节点使用 images/generations；普通文本节点使用 chat/completions。",
  responseFormat:
    "DALL-E 或三方兼容节点会随请求发送；GPT image 节点默认返回 b64_json，本字段只作为解码偏好。",
  thinkingType:
    "DeepSeek 等兼容接口可用；auto 不发送字段，disabled 会关闭思考模式。",
  reasoningEffort:
    "仅 thinking 未关闭时发送；DeepSeek 当前支持 high / max。",
  imageSize: "支持 auto 或 1024x1024、1536x1024、1024x1536 等 OpenAI 图片尺寸。",
  imageBackground:
    "控制生成图的背景透明度；JPEG 建议选择 opaque，透明背景请配合 png 或 webp。",
  imageCompression: "仅 jpeg / webp 生效；数值越低体积越小，画质损失越明显。",
  apiKey: "已保存密钥留空会继续保留旧值；点击清空并保存后才会移除。",
  audit: "默认只展示最近 5 条，完整审计可在抽屉中筛选和分页。",
};

const retryOptions = [
  { label: "超时 timeout", value: "timeout" },
  { label: "网络 network", value: "network" },
  { label: "限流 rate_limit", value: "rate_limit" },
  { label: "鉴权 auth", value: "auth" },
  { label: "上游 upstream", value: "upstream" },
  { label: "响应异常 invalid_response", value: "invalid_response" },
];

const providerEndpointModeOptions: Array<{
  label: string;
  value: AIRoutingProviderEndpointMode;
}> = [
  { label: "chat/completions", value: "chat_completions" },
  { label: "images/generations", value: "images_generations" },
];

const providerResponseFormatOptions: Array<{
  label: string;
  value: AIRoutingProviderResponseFormat;
}> = [
  { label: "auto", value: "auto" },
  { label: "image_url", value: "image_url" },
  { label: "b64_json", value: "b64_json" },
];

const thinkingTypeOptions = [
  { label: "auto", value: "auto" },
  { label: "enabled", value: "enabled" },
  { label: "disabled", value: "disabled" },
];

const reasoningEffortOptions = [
  { label: "auto", value: "" },
  { label: "high", value: "high" },
  { label: "max", value: "max" },
];
const imageSizeOptions = [
  { label: "auto", value: "auto" },
  { label: "1024x1024", value: "1024x1024" },
  { label: "1536x1024", value: "1536x1024" },
  { label: "1024x1536", value: "1024x1536" },
];
const imageQualityOptions = [
  { label: "auto", value: "auto" },
  { label: "low", value: "low" },
  { label: "medium", value: "medium" },
  { label: "high", value: "high" },
];
const imageBackgroundOptions = [
  { label: "auto", value: "auto", tip: "交给模型或上游默认处理，兼容性最好。" },
  { label: "opaque", value: "opaque", tip: "强制不透明背景，适合 jpeg 和小程序展示。" },
  {
    label: "transparent",
    value: "transparent",
    tip: "透明背景，建议搭配 png 或 webp；jpeg 通常不适用。",
  },
];
const imageOutputFormatOptions = [
  { label: "png", value: "png" },
  { label: "jpeg", value: "jpeg" },
  { label: "webp", value: "webp" },
];
const providerSelectOptions = {
  endpointModes: providerEndpointModeOptions,
  responseFormats: providerResponseFormatOptions,
  thinkingTypes: thinkingTypeOptions,
  reasoningEfforts: reasoningEffortOptions,
  imageSizes: imageSizeOptions,
  imageQualities: imageQualityOptions,
  imageBackgrounds: imageBackgroundOptions,
  imageOutputFormats: imageOutputFormatOptions,
};
const auditPageSizeOptions = [20, 50, 100];
const recentAuditKindOptions = [
  { label: "全部", value: "all" },
  { label: "配置变更", value: "changes" },
  { label: "测试执行", value: "tests" },
];
const providerPresetOptions = computed(() => {
  const options: Array<{
    key: ProviderPresetKey;
    title: string;
    description: string;
  }> = [
    {
      key: "openai-text",
      title: "OpenAI 兼容文本节点",
      description: "chat/completions、普通 JSON 响应、适合总结和标题清洗。",
    },
  ];
  if (currentSceneKey.value === "flowchart") {
    options.push({
      key: "openai-image",
      title: "OpenAI 图片生成节点",
      description: "images/generations、b64_json 解析、适合步骤图生成。",
    });
  }
  return options;
});

const currentAuditGroup = computed(() => `ai.routing.${currentSceneKey.value}`);
const enabledProviderCount = computed(
  () => draftScene.value?.providers.filter((item) => item.enabled).length || 0,
);
const minimumMaxAttempts = computed(() =>
  enabledProviderCount.value > 1 ? 2 : 1,
);
const maxAttemptCeiling = computed(() =>
  Math.max(enabledProviderCount.value || 1, minimumMaxAttempts.value),
);
const smallPopoverWidth = computed(() => (viewportWidth.value < 768 ? "90vw" : 360));
const mediumPopoverWidth = computed(() =>
  viewportWidth.value < 768 ? "90vw" : 380,
);
const largePopoverWidth = computed(() =>
  viewportWidth.value < 768 ? "90vw" : 680,
);
const compatibilityHint = computed(() => {
  if (!draftScene.value?.compatibilityMode) {
    return "";
  }
  return "当前运行时仍优先走旧单 Provider 配置；保存并启用本场景后，summary / title / flowchart 才会正式切到新的多节点路由。";
});

function emptySceneCardHealthSnapshot(): SceneCardHealthSnapshot {
  return {
    recentTest: {
      tone: "neutral",
      text: "加载中",
      testedAt: "",
    },
    configRisk: {
      tone: "neutral",
      text: "加载中",
    },
    alertStatus: {
      tone: "neutral",
      text: "加载中",
    },
  };
}

function summarizeSceneRecentTest(
  record?: SettingAuditRecord | null,
): SceneCardHealthSnapshot["recentTest"] {
  if (!record) {
    return {
      tone: "neutral",
      text: "未测试",
      testedAt: "",
    };
  }
  const summary =
    `${record.newValueMasked || ""} ${record.oldValueMasked || ""}`
      .trim()
      .toLowerCase();
  return {
    tone: summary.startsWith("ok") ? "success" : "warning",
    text: summary.startsWith("ok") ? "成功" : "异常",
    testedAt: record.createdAt,
  };
}

function summarizeSceneConfigRisk(
  scene?: AIRoutingSceneConfig,
): SceneCardHealthSnapshot["configRisk"] {
  if (!scene) {
    return {
      tone: "neutral",
      text: "加载中",
    };
  }
  const enabledProviders = scene.providers.filter((item) => item.enabled);
  if (!enabledProviders.length) {
    return {
      tone: "danger",
      text: "无启用节点",
    };
  }
  const missingSecretCount = enabledProviders.filter(
    (item) => !providerHasUsableSecret(item),
  ).length;
  if (missingSecretCount > 0) {
    return {
      tone: "warning",
      text: "启用节点缺密钥",
    };
  }
  return {
    tone: "success",
    text: "正常",
  };
}

const sceneCardHealthMap = computed<
  Partial<Record<AIRoutingSceneKey, SceneCardHealthSnapshot>>
>(() => {
  return sceneKeys.reduce(
    (acc, scene) => {
      const detail =
        scene === currentSceneKey.value && draftScene.value
          ? draftScene.value
          : sceneDetailMap.value[scene];
      acc[scene] = {
        recentTest: summarizeSceneRecentTest(
          sceneLatestTestAuditMap.value[scene],
        ),
        configRisk: summarizeSceneConfigRisk(detail),
        alertStatus: summarizeSceneAlertStatus(scene, alertOverview.value),
      };
      return acc;
    },
    {} as Partial<Record<AIRoutingSceneKey, SceneCardHealthSnapshot>>,
  );
});

const sceneCards = computed(() => {
  return sceneKeys.map((scene) => {
    const summary = sceneSummaries.value.find((item) => item.scene === scene);
    const health =
      sceneCardHealthMap.value[scene] || emptySceneCardHealthSnapshot();
    return {
      scene,
      title:
        scene === "summary"
          ? "正文总结"
          : scene === "title"
            ? "标题清洗"
            : "步骤图生成",
      strategy:
        summary?.strategy ||
        (scene === "title" ? "round_robin_failover" : "priority_failover"),
      providerCount: summary?.providerCount || 0,
      activeProviderCount: summary?.activeProviderCount || 0,
      updatedAt: summary?.updatedAt || "",
      source: summary?.source || "empty",
      savedChannelLabel: savedSceneChannel(scene).label,
      compatibilityMode: summary?.compatibilityMode ?? true,
      health,
      aggregate: sceneAggregateStatus(summary, health),
      issue: summarizeSceneIssue(
        scene,
        health,
        summary,
        alertOverview.value,
      ),
    };
  });
});

const currentSceneTitle = computed(() => {
  const card = sceneCards.value.find(
    (item) => item.scene === currentSceneKey.value,
  );
  return card?.title || displayAIRoutingScene(currentSceneKey.value);
});

const {
  alertStatusDescription,
  alertStatusSummary,
  copyRequestId,
  currentSceneAlertSections,
  goAlertConfig,
  goAlertProviderLogs,
  handleAlertArchive,
  handleAlertMute,
  handleAlertRetest,
  handleAlertUnmute,
  handleBatchArchive,
  handleBatchRetest,
  hasCurrentSceneAlertNodes,
  pendingAlertActions,
} = useAIRoutingAlerts({
  currentSceneKey,
  alertOverview,
  router,
  extractMessage,
  api: adminApi,
});

const showAnchorDirectory = computed(() => {
  const currentCard = sceneCards.value.find(
    (item) => item.scene === currentSceneKey.value,
  );
  return (
    viewportWidth.value >= 1280 &&
    ((draftScene.value?.providers.length || currentCard?.providerCount || 0) >=
      4 ||
      !!testResult.value ||
      recentAudits.value.length > 0)
  );
});

const anchorNavItems = computed(() => {
  const items: Array<{ key: AnchorSectionKey; label: string }> = [
    { key: "scene-cards", label: "场景卡" },
    { key: "scene-strategy", label: "场景策略" },
    { key: "provider-nodes", label: "Provider 节点" },
  ];
  if (testResult.value) {
    items.push({ key: "latest-test", label: "最近测试结果" });
  }
  items.push({ key: "recent-audits", label: "最近审计" });
  return items;
});

function shortcutTokens(kind: "save" | "test" | "scene-prev" | "scene-next") {
  const modKey = isMacLikePlatform.value ? "⌘" : "Ctrl";
  if (kind === "save") {
    return [modKey, "S"];
  }
  if (kind === "test") {
    return [modKey, "Enter"];
  }
  const altKey = isMacLikePlatform.value ? "⌥" : "Alt";
  return [altKey, kind === "scene-prev" ? "←" : "→"];
}

function shortcutText(kind: "save" | "test" | "scene-prev" | "scene-next") {
  return shortcutTokens(kind).join(" + ");
}

const saveActionTooltip = computed(() => `保存场景 · ${shortcutText("save")}`);
const testActionTooltip = computed(
  () => `测试当前草稿 · ${shortcutText("test")}`,
);

function effectiveChannel(params: {
  scene: AIRoutingSceneKey;
  enabled: boolean;
  compatibilityMode: boolean;
  isDraftDirty?: boolean;
}) {
  const { scene, enabled, compatibilityMode, isDraftDirty } = params;
  if (isDraftDirty) {
    return {
      label: "草稿待发布",
      technicalLabel: `${scene}-draft`,
      tone: "primary" as const,
      reason: "当前存在未保存草稿。测试入口会使用草稿，线上仍保持上次保存的配置。",
    };
  }
  if (compatibilityMode) {
    return {
      label: "旧单 Provider 兼容链路",
      technicalLabel: `${scene}-compat`,
      tone: "warning" as const,
      reason: "运行时仍优先走旧单 Provider 配置。启用新路由并保存后才会切换。",
    };
  }
  if (!enabled) {
    return {
      label: "旧单 Provider 兼容链路",
      technicalLabel: `${scene}-compat`,
      tone: "neutral" as const,
      reason: "当前场景的新多节点路由未启用，线上回退旧单 Provider 配置。",
    };
  }
  return {
    label: "新多节点路由",
    technicalLabel: `${scene}-v2`,
    tone: "success" as const,
    reason: "线上已使用保存后的多节点路由配置。",
  };
}

function sceneEffectiveChannel(scene: AIRoutingSceneKey) {
  const summary = sceneSummaries.value.find((item) => item.scene === scene);
  const isCurrent = scene === currentSceneKey.value && !!draftScene.value;
  const enabled =
    isCurrent && draftScene.value
      ? draftScene.value.enabled
      : (summary?.enabled ?? false);
  const compatibilityMode =
    isCurrent && draftScene.value
      ? draftScene.value.compatibilityMode
      : (summary?.compatibilityMode ?? true);
  const isDraftDirty = isCurrent && isDirty.value;
  return effectiveChannel({ scene, enabled, compatibilityMode, isDraftDirty });
}

function savedSceneChannel(scene: AIRoutingSceneKey) {
  const summary = sceneSummaries.value.find((item) => item.scene === scene);
  const savedScene =
    scene === currentSceneKey.value ? remoteScene.value : sceneDetailMap.value[scene];
  const enabled = savedScene?.enabled ?? summary?.enabled ?? false;
  const compatibilityMode =
    savedScene?.compatibilityMode ?? summary?.compatibilityMode ?? true;
  return effectiveChannel({ scene, enabled, compatibilityMode });
}

const currentChannel = computed(() =>
  sceneEffectiveChannel(currentSceneKey.value),
);

const currentEffectHeadline = computed(() => {
  if (isDirty.value) {
    return `线上仍在使用：${savedSceneChannel(currentSceneKey.value).label}`;
  }
  return `线上正在使用：${currentChannel.value.label}`;
});

const currentEffectDescription = computed(() => {
  if (isDirty.value) {
    return "你的草稿尚未发布。测试会读取当前草稿，保存并发布后才会影响线上。";
  }
  return currentChannel.value.reason;
});

const channelMatrix = computed(() => [
  {
    draft: "已保存",
    toggle: "开",
    effect: "线上使用新多节点路由",
    hit:
      !isDirty.value &&
      draftScene.value?.enabled &&
      !draftScene.value?.compatibilityMode,
  },
  {
    draft: "已保存",
    toggle: "关",
    effect: "线上使用旧单 Provider 兼容链路",
    hit:
      !isDirty.value &&
      (!draftScene.value?.enabled || draftScene.value?.compatibilityMode),
  },
  {
    draft: "草稿未发布",
    toggle: "不改变线上",
    effect: "仅测试入口使用草稿",
    hit: isDirty.value,
  },
]);

const numericWarn = computed(() => {
  const scene = draftScene.value;
  const warn: {
    maxAttempts: string;
    failureThreshold: string;
    cooldownSeconds: string;
  } = {
    maxAttempts: "",
    failureThreshold: "",
    cooldownSeconds: "",
  };
  if (!scene) return warn;
  const ma = Number(scene.maxAttempts) || 0;
  if (ma > 5) warn.maxAttempts = `当前 ${ma} 次偏大，失败降级会变慢`;
  else if (enabledProviderCount.value > 1 && ma < 2)
    warn.maxAttempts = "至少 2 次才能触发节点切换";
  const ft = Number(scene.breaker?.failureThreshold) || 0;
  if (ft < 3) warn.failureThreshold = "阈值过低容易误熔断";
  else if (ft > 10) warn.failureThreshold = "阈值过高将延迟熔断保护";
  const cs = Number(scene.breaker?.cooldownSeconds) || 0;
  if (cs < 30) warn.cooldownSeconds = "低于 30s 容易抖动";
  else if (cs > 300) warn.cooldownSeconds = "超过 5 分钟可能影响恢复";
  return warn;
});

const timelineSegments = computed(() => {
  const attempts = Math.max(Number(draftScene.value?.maxAttempts) || 1, 1);
  return { attempts: Math.min(attempts, 6) };
});

const expectedFirstRoundSeconds = computed(() => {
  const scene = draftScene.value;
  if (!scene) return 0;
  const attempts = Math.max(Number(scene.maxAttempts) || 1, 1);
  const enabled = scene.providers.filter((p) => p.enabled);
  const maxTimeout =
    enabled.reduce(
      (acc, p) => Math.max(acc, Number(p.timeoutSeconds) || 0),
      0,
    ) || 30;
  return attempts * maxTimeout;
});

const latestTestSummary = computed(() => {
  if (testingScene.value) {
    return {
      tone: "primary" as const,
      text: `测试中 · ${testingElapsedSeconds.value}s`,
    };
  }
  if (!testResult.value) {
    return null;
  }
  const scopePrefix = testScope.value.includes("单节点") ? "单节点" : "草稿";
  const target = testResult.value.finalProvider
    ? providerDisplayName(testResult.value.finalProvider)
    : "查看详情";
  const latency = testResult.value.attempts.reduce(
    (acc, item) => acc + (Number(item.latencyMs) || 0),
    0,
  );
  return {
    tone: testResultSummary.value.tone,
    text: `${scopePrefix}${testResultSummary.value.shortText} · ${target}${latency ? ` · ${formatDuration(latency)}` : ""}`,
  };
});

const testResultSummary = computed(() => {
  if (!testResult.value) {
    return {
      tone: "neutral" as const,
      text: "未测试",
      shortText: "未测试",
    };
  }
  if (allEnabledProvidersCooling.value) {
    return {
      tone: "warning" as const,
      text: "所有节点冷却中",
      shortText: "测试需关注",
    };
  }
  if (testResult.value.ok) {
    return {
      tone: "success" as const,
      text: "测试成功",
      shortText: "测试通过",
    };
  }
  return {
    tone: "warning" as const,
    text: "需要关注",
    shortText: "测试异常",
  };
});

const testScopeDescription = computed(() => {
  if (testScope.value.includes("单节点")) {
    return `${testScope.value}。结果只代表当前节点草稿，不代表线上当前行为。`;
  }
  return "测试对象：当前草稿。该结果不代表线上当前行为，保存并发布后才会影响线上。";
});

const providerValidationErrors = computed<
  Record<string, ProviderValidationError>
>(() => buildProviderValidationErrors(draftScene.value, getProviderLocalKey));

const blockingValidationCount = computed(() =>
  countProviderValidationErrors(providerValidationErrors.value),
);

const sceneBlockingValidationMessages = computed(() =>
  buildSceneBlockingValidationMessages(draftScene.value),
);

const recentAudits = computed(() => {
  if (recentAuditKindFilter.value === "tests") {
    return recentAuditItems.value.filter((item) => item.action === "test");
  }
  if (recentAuditKindFilter.value === "changes") {
    return recentAuditItems.value.filter((item) => item.action !== "test");
  }
  return recentAuditItems.value;
});

const providerNameById = computed(() => {
  const map = new Map<string, string>();
  const collect = (scene?: AIRoutingSceneConfig | null) => {
    scene?.providers?.forEach((provider) => {
      const id = provider.id.trim();
      const name = provider.name.trim();
      if (id && name) map.set(id, name);
    });
  };
  collect(remoteScene.value);
  collect(draftScene.value);
  return map;
});

const {
  auditBusinessAction,
  auditDiffStatusText,
  auditDiffTitle,
  auditEventLabel,
  auditFallbackSummary,
  auditTargetTitle,
  displayRouteTestMessage,
  toneForAuditAction,
} = createAIProviderAuditPresenter({
  currentScene: () => currentSceneKey.value,
  providerName: (providerId) => providerNameById.value.get(providerId) || "",
});

const {
  dragOverProviderIndex,
  draggingProviderIndex,
  firstProviderError,
  getProviderLocalKey,
  getProviderTestState,
  handleClearProviderApiKey,
  handleEndpointModeChange,
  handleProviderApiKeyInput,
  handleProviderDragEnd,
  handleProviderDragOver,
  handleProviderDragStart,
  handleProviderDrop,
  handleThinkingTypeChange,
  isImageGenerationProvider,
  isProviderCollapsed,
  providerFieldError,
  providerThinkingLabel,
  recordProviderTestState,
  resetProviderUIState,
  setProviderSecretEditor,
  shouldShowProviderSecretEditor,
  toggleProviderCollapsed,
  toggleProviderSecretEditor,
  touchAllProviderFields,
  touchProviderField,
} = useAIProviderEditor({
  draftScene,
  validationErrors: providerValidationErrors,
  formatDuration,
  formatTestMessage: displayRouteTestMessage,
  displayCallStatus,
});

const providerEditorContract = computed(() => ({
  draggingProviderIndex: draggingProviderIndex.value,
  dragOverProviderIndex: dragOverProviderIndex.value,
  getProviderLocalKey,
  isProviderCollapsed,
  firstProviderError,
  getProviderTestState,
  providerFieldError,
  isImageGenerationProvider,
  providerThinkingLabel,
  shouldShowProviderSecretEditor,
  add: handleAddProvider,
  addPreset: handleAddProviderPreset,
  dragOver: handleProviderDragOver,
  drop: handleProviderDrop,
  dragStart: handleProviderDragStart,
  dragEnd: handleProviderDragEnd,
  toggleCollapse: toggleProviderCollapsed,
  testSingle: handleTestSingleProvider,
  menu: handleProviderMenuCommand,
  touchField: touchProviderField,
  endpointChange: handleEndpointModeChange,
  thinkingChange: handleThinkingTypeChange,
  toggleSecret: toggleProviderSecretEditor,
  clearSecret: handleClearProviderApiKey,
  apiKeyInput: handleProviderApiKeyInput,
}));

const bottomBarState = computed(() => {
  if (testingScene.value) {
    return {
      tone: "primary" as const,
      text: `测试中 · ${testingElapsedSeconds.value}s`,
      icon: Clock,
    };
  }
  if (testResult.value && !testResult.value.ok) {
    const errorType =
      testResult.value.attempts.find((item) => item.errorType)?.errorType ||
      "查看详情";
    return {
      tone: "warning" as const,
      text: `测试异常 · ${errorType}`,
      icon: Warning,
    };
  }
  if (isDirty.value) {
    return {
      tone: "warning" as const,
      text: `有未保存的更改 · ${diffCount.value} 项`,
      icon: Warning,
    };
  }
  if (testResult.value?.ok) {
    return {
      tone: "success" as const,
      text: latestTestSummary.value?.text || "测试通过",
      icon: Check,
    };
  }
  return { tone: "success" as const, text: "已同步", icon: Check };
});

const statusBarText = computed(() => {
  if (testingScene.value) return `测试中 · ${testingElapsedSeconds.value}s`;
  if (isDirty.value) return `草稿 ${diffCount.value} 项 · 保存后影响线上`;
  if (latestTestSummary.value) return latestTestSummary.value.text;
  return "已同步";
});

const enabledProvidersMissingSecretCount = computed(() => {
  return (
    draftScene.value?.providers.filter(
      (provider) => provider.enabled && !providerHasUsableSecret(provider),
    ).length || 0
  );
});

const allEnabledProvidersCooling = computed(() => {
  if (!draftScene.value || !testResult.value) {
    return false;
  }
  const enabledProviderIDs = draftScene.value.providers
    .filter((provider) => provider.enabled)
    .map((provider) => provider.id.trim())
    .filter(Boolean);
  if (!enabledProviderIDs.length) {
    return false;
  }
  const attemptByProviderID = new Map(
    testResult.value.attempts.map((attempt) => [attempt.providerId, attempt]),
  );
  const attempts = enabledProviderIDs
    .map((providerID) => attemptByProviderID.get(providerID))
    .filter(Boolean);
  if (attempts.length !== enabledProviderIDs.length) {
    return false;
  }
  return attempts.every((attempt) => attempt?.skippedByBreaker);
});

const blockingRiskItems = computed<BlockingRiskItem[]>(() => {
  const scene = draftScene.value;
  if (!scene) {
    return [];
  }
  const items: BlockingRiskItem[] = [];
  if (!enabledProviderCount.value) {
    items.push({
      key: "no-enabled-provider",
      tone: scene.enabled ? "danger" : "warning",
      title: "当前没有启用节点",
      description: scene.enabled
        ? "保存后新路由将没有可调度节点。"
        : "若后续启用新路由，需要先至少启用一个节点。",
    });
  }
  if (enabledProvidersMissingSecretCount.value > 0) {
    items.push({
      key: "missing-secret",
      tone: "danger",
      title: `${enabledProvidersMissingSecretCount.value} 个启用节点缺可用密钥`,
      description: "这些节点当前测试会失败，保存后仍会保持不可用。",
    });
  }
  if (
    enabledProviderCount.value > 0 &&
    Number(draftScene.value?.maxAttempts || 0) > enabledProviderCount.value
  ) {
    items.push({
      key: "max-attempts-overflow",
      tone: "warning",
      title: "最大尝试次数高于启用节点数",
      description: `当前仅有 ${enabledProviderCount.value} 个启用节点，测试和保存时会按该值截断。`,
    });
  }
  if (
    scene.enabled &&
    enabledProviderCount.value > 1 &&
    Number(scene.maxAttempts || 0) < 2
  ) {
    items.push({
      key: "max-attempts-too-low",
      tone: "danger",
      title: "最大尝试次数无法触发切换",
      description: "当前启用多个节点，至少需要 2 次尝试才能在失败后切到备用节点。",
    });
  }
  if (allEnabledProvidersCooling.value) {
    items.push({
      key: "all-provider-cooling",
      tone: "danger",
      title: "最近一次测试中所有启用节点都在冷却",
      description:
        "本轮请求已被 breaker 全部跳过，需要等待冷却结束或调整配置。",
    });
  } else if (testResult.value && !testResult.value.ok) {
    const failedAttempt = testResult.value.attempts.find(
      (attempt) => attempt.errorType || attempt.errorMessage,
    );
    items.push({
      key: "latest-test-failed",
      tone: "warning",
      title: "最近一次测试异常",
      description:
        failedAttempt?.errorType ||
        failedAttempt?.errorMessage ||
        testResult.value.message ||
        "请先查看测试详情并排查后再保存。",
    });
  }
  if (pendingClearKeyCount.value > 0) {
    items.push({
      key: "clear-secret",
      tone: "danger",
      title: `保存将清空 ${pendingClearKeyCount.value} 个密钥`,
      description: "相关节点保存后会彻底失去旧密钥。",
    });
  }
  if (pendingRemovedProviderCount.value > 0) {
    items.push({
      key: "remove-provider",
      tone: "warning",
      title: `保存将删除 ${pendingRemovedProviderCount.value} 个节点`,
      description: "删除后线上只保留当前草稿中的 Provider 列表。",
    });
  }
  return items;
});

const activeAuditFilters = computed(() => {
  const items: Array<{ key: string; label: string; onRemove: () => void }> = [];
  if (auditAction.value) {
    items.push({
      key: "action",
      label: `动作：${displayAuditAction(auditAction.value)}`,
      onRemove: () => {
        auditAction.value = "";
        applyAuditFilters();
      },
    });
  }
  if (auditOperator.value) {
    items.push({
      key: "operator",
      label: `操作人：${auditOperator.value}`,
      onRemove: () => {
        auditOperator.value = "";
        applyAuditFilters();
      },
    });
  }
  if (auditSettingKey.value) {
    items.push({
      key: "settingKey",
      label: `配置键：${auditSettingKey.value}`,
      onRemove: () => {
        auditSettingKey.value = "";
        applyAuditFilters();
      },
    });
  }
  if (auditTimeRange.value.length) {
    items.push({
      key: "timeRange",
      label: `时间：${formatDateTime(auditTimeRange.value[0].toISOString())} 至 ${formatDateTime(auditTimeRange.value[1].toISOString())}`,
      onRemove: () => {
        auditTimeRange.value = [];
        applyAuditFilters();
      },
    });
  }
  if (auditPageSize.value !== 20) {
    items.push({
      key: "pageSize",
      label: `每页：${auditPageSize.value} 条`,
      onRemove: () => {
        auditPageSize.value = 20;
        applyAuditFilters();
      },
    });
  }
  return items;
});

const discardTooltip = computed(() => {
  if (!isDirty.value) return "当前没有未保存改动";
  const n = diffCount.value;
  return n > 0 ? `将丢弃 ${n} 项未保存改动` : "将丢弃当前未保存改动";
});

function setAnchorSectionRef(key: AnchorSectionKey, element: Element | null) {
  anchorSectionRefs[key] = element instanceof HTMLElement ? element : null;
}

function scrollElementToViewport(element: HTMLElement, offset = 96) {
  const top = element.getBoundingClientRect().top + window.scrollY - offset;
  window.scrollTo({ top: Math.max(top, 0), behavior: "smooth" });
}

function scrollToAnchorSection(key: AnchorSectionKey) {
  const element = anchorSectionRefs[key];
  if (!element) {
    return;
  }
  scrollElementToViewport(element);
  activeAnchorKey.value = key;
}

function updateActiveAnchorSection() {
  if (!showAnchorDirectory.value) {
    activeAnchorKey.value = "scene-cards";
    return;
  }
  const visibleItems = anchorNavItems.value.filter(
    (item) => anchorSectionRefs[item.key],
  );
  if (!visibleItems.length) {
    return;
  }
  let currentKey = visibleItems[0].key;
  for (const item of visibleItems) {
    const top = anchorSectionRefs[item.key]?.getBoundingClientRect().top ?? 0;
    if (top <= 180) {
      currentKey = item.key;
      continue;
    }
    break;
  }
  activeAnchorKey.value = currentKey;
}

function activeInteractiveElement(event?: KeyboardEvent) {
  const target = event?.target;
  if (target instanceof HTMLElement) {
    return target;
  }
  return document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null;
}

function isEditingElement(element: HTMLElement | null) {
  if (!element) {
    return false;
  }
  if (element.matches("input, textarea, select")) {
    return true;
  }
  if (element.isContentEditable) {
    return true;
  }
  return !!element.closest(
    'input, textarea, select, [contenteditable]:not([contenteditable="false"])',
  );
}

function hasOpenMessageBox() {
  return !!document.body.querySelector(
    ".el-message-box, .el-message-box__wrapper",
  );
}

function shouldDisablePageShortcuts(event?: KeyboardEvent) {
  if (auditDrawerVisible.value || savingScene.value || testingScene.value) {
    return true;
  }
  if (hasOpenMessageBox()) {
    return true;
  }
  return isEditingElement(activeInteractiveElement(event));
}

function handleGlobalKeydown(event: KeyboardEvent) {
  if (event.repeat || shouldDisablePageShortcuts(event)) {
    return;
  }

  const isMod = event.metaKey || event.ctrlKey;
  const key = event.key.toLowerCase();
  if (isMod && !event.altKey && !event.shiftKey && key === "s") {
    event.preventDefault();
    void handleSaveScene();
    return;
  }
  if (isMod && !event.altKey && !event.shiftKey && event.key === "Enter") {
    event.preventDefault();
    void handleTestScene();
    return;
  }
  if (!isMod && event.altKey && !event.shiftKey && event.key === "ArrowLeft") {
    event.preventDefault();
    handleSceneArrowKey(-1);
    return;
  }
  if (!isMod && event.altKey && !event.shiftKey && event.key === "ArrowRight") {
    event.preventDefault();
    handleSceneArrowKey(1);
  }
}

function updatePlatformShortcutLabels() {
  if (typeof navigator === "undefined") {
    isMacLikePlatform.value = false;
    return;
  }
  const platform =
    navigator.userAgentData?.platform || navigator.platform || "";
  isMacLikePlatform.value = /mac|iphone|ipad|ipod/i.test(platform);
}

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (isDirty.value) {
    e.preventDefault();
    e.returnValue = "";
  }
}

onBeforeRouteLeave(async () => {
  return await guardUnsavedChanges("离开页面");
});

onMounted(async () => {
  updatePlatformShortcutLabels();
  window.addEventListener("beforeunload", onBeforeUnload);
  window.addEventListener("keydown", handleGlobalKeydown);
  window.addEventListener("scroll", updateActiveAnchorSection, {
    passive: true,
  });
  window.addEventListener("resize", updateActiveAnchorSection, {
    passive: true,
  });
  if (typeof IntersectionObserver !== "undefined" && topCtaSentinelRef.value) {
    topCtaObserver = new IntersectionObserver(
      ([entry]) => {
        // 哨兵位于顶部工具栏正下方：一旦它离开视口，说明顶部 CTA 也已滚走。
        floatingBarVisible.value = !entry.isIntersecting;
      },
      { threshold: 0 },
    );
    topCtaObserver.observe(topCtaSentinelRef.value);
  }
  const queryScene = readQueryString(route.query, "scene");
  if (sceneKeys.includes(queryScene as AIRoutingSceneKey)) {
    currentSceneKey.value = queryScene as AIRoutingSceneKey;
  }
  await refreshPage();
});

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", onBeforeUnload);
  window.removeEventListener("keydown", handleGlobalKeydown);
  window.removeEventListener("scroll", updateActiveAnchorSection);
  window.removeEventListener("resize", updateActiveAnchorSection);
  topCtaObserver?.disconnect();
  topCtaObserver = null;
  stopTestingTimer();
});

watch(
  () => route.query.scene,
  async (value) => {
    const nextScene = String(value || "").trim();
    if (!sceneKeys.includes(nextScene as AIRoutingSceneKey)) {
      return;
    }
    if (nextScene === currentSceneKey.value) {
      return;
    }
    if (routeSceneOverride.value === nextScene) {
      routeSceneOverride.value = null;
      await applySceneChange(nextScene as AIRoutingSceneKey);
      return;
    }
    const allowed = await guardUnsavedChanges(
      `切换到「${displayAIRoutingScene(nextScene)}」`,
    );
    if (!allowed) {
      await router.replace({
        query: buildRouteQuery({ scene: currentSceneKey.value }),
      });
      return;
    }
    await applySceneChange(nextScene as AIRoutingSceneKey);
  },
);

watch(
  () => [
    showAnchorDirectory.value,
    anchorNavItems.value.map((item) => item.key).join(","),
    recentAudits.value.length,
    testResult.value ? 1 : 0,
    draftScene.value?.providers.length || 0,
  ],
  async () => {
    await nextTick();
    updateActiveAnchorSection();
  },
);

async function refreshPage() {
  const allowed = await guardUnsavedChanges("刷新页面");
  if (!allowed) {
    return;
  }
  pageRefreshing.value = true;
  resetSceneEditor();
  try {
    await Promise.all([loadSceneSummaries(), loadCurrentScene()]);
  } finally {
    pageRefreshing.value = false;
  }
}

async function handleDiscardDraft() {
  if (!remoteScene.value || !isDirty.value) return;
  try {
    await ElMessageBox.confirm("确定放弃所有未保存的改动？", "放弃草稿", {
      type: "warning",
    });
    discardDraftChanges("已恢复到上次保存的状态");
  } catch {
    return;
  }
}

async function loadSceneSummaries() {
  const response = await adminApi.listAIRoutingScenes();
  sceneSummaries.value = response.items;
}

async function loadLatestTestAuditForScene(scene: AIRoutingSceneKey) {
  const query = new URLSearchParams();
  query.set("group", `ai.routing.${scene}`);
  query.set("action", "test");
  query.set("page", "1");
  query.set("pageSize", "1");
  const response = await adminApi.listSettingAudits(query);
  return response.result.items[0] || null;
}

async function loadSceneCardHealthData(
  currentScene: AIRoutingSceneKey,
  currentSceneConfig: AIRoutingSceneConfig,
) {
  const requestToken = ++sceneHealthLoadToken;
  const nextDetailMap: Partial<
    Record<AIRoutingSceneKey, AIRoutingSceneConfig>
  > = {
    ...sceneDetailMap.value,
    [currentScene]: hydrateScene(currentSceneConfig),
  };
  const nextTestAuditMap: Partial<
    Record<AIRoutingSceneKey, SettingAuditRecord | null>
  > = {
    ...sceneLatestTestAuditMap.value,
  };

  const [detailResults, testAuditResults, alertResult] = await Promise.all([
    Promise.allSettled(
      sceneKeys
        .filter((scene) => scene !== currentScene)
        .map(async (scene) => {
          const response = await adminApi.getAIRoutingScene(scene);
          return {
            scene,
            config: hydrateScene(response.scene),
          };
        }),
    ),
    Promise.allSettled(
      sceneKeys.map(async (scene) => ({
        scene,
        audit: await loadLatestTestAuditForScene(scene),
      })),
    ),
    adminApi.getAIRoutingAlertsOverview().catch(() => null),
  ]);

  if (requestToken !== sceneHealthLoadToken) {
    return;
  }

  detailResults.forEach((result) => {
    if (result.status === "fulfilled") {
      nextDetailMap[result.value.scene] = result.value.config;
    }
  });

  testAuditResults.forEach((result) => {
    if (result.status === "fulfilled") {
      nextTestAuditMap[result.value.scene] = result.value.audit;
    }
  });

  sceneDetailMap.value = nextDetailMap;
  sceneLatestTestAuditMap.value = nextTestAuditMap;
  if (alertResult?.overview) {
    alertOverview.value = alertResult.overview;
  }
}

async function loadCurrentScene() {
  sceneLoading.value = true;
  sceneError.value = "";
  try {
    const response = await adminApi.getAIRoutingScene(currentSceneKey.value);
    remoteScene.value = hydrateScene(response.scene);
    draftScene.value = hydrateScene(response.scene);
    resetProviderUIState();
    testResult.value = null;
    testScope.value = "";
    auditPage.value = 1;
    await Promise.all([
      loadRecentAudits(),
      loadAudits(),
      loadSceneCardHealthData(currentSceneKey.value, response.scene),
    ]);
    await nextTick();
    updateActiveAnchorSection();
  } catch (error) {
    resetSceneEditor();
    sceneError.value = extractMessage(error);
  } finally {
    sceneLoading.value = false;
  }
}

async function loadAudits() {
  auditsLoading.value = true;
  auditsError.value = "";
  try {
    const query = new URLSearchParams();
    query.set("group", currentAuditGroup.value);
    query.set("page", String(auditPage.value));
    query.set("pageSize", String(auditPageSize.value));
    if (auditAction.value) {
      query.set("action", auditAction.value);
    }
    if (auditOperator.value) {
      query.set("operator", auditOperator.value);
    }
    if (auditSettingKey.value) {
      query.set("settingKey", auditSettingKey.value);
    }
    if (auditTimeRange.value.length) {
      query.set("timeFrom", auditTimeRange.value[0].toISOString());
      query.set("timeTo", auditTimeRange.value[1].toISOString());
    }
    const response = await adminApi.listSettingAudits(query);
    audits.value = response.result;
  } catch (error) {
    auditsError.value = extractMessage(error);
  } finally {
    auditsLoading.value = false;
  }
}

async function loadRecentAudits() {
  recentAuditsLoading.value = true;
  recentAuditsError.value = "";
  try {
    const query = new URLSearchParams();
    query.set("group", currentAuditGroup.value);
    query.set("page", "1");
    query.set("pageSize", "12");
    const response = await adminApi.listSettingAudits(query);
    recentAuditItems.value = response.result.items;
  } catch (error) {
    recentAuditsError.value = extractMessage(error);
  } finally {
    recentAuditsLoading.value = false;
  }
}

async function handleSceneChange(scene: AIRoutingSceneKey) {
  if (scene === currentSceneKey.value) {
    return;
  }
  const allowed = await guardUnsavedChanges(
    `切换到「${displayAIRoutingScene(scene)}」`,
  );
  if (!allowed) {
    return;
  }
  routeSceneOverride.value = scene;
  await router.replace({ query: buildRouteQuery({ scene }) });
}

async function applySceneChange(scene: AIRoutingSceneKey) {
  currentSceneKey.value = scene;
  resetSceneEditor();
  await loadCurrentScene();
  if (shouldFocusSceneAfterChange.value) {
    shouldFocusSceneAfterChange.value = false;
    focusSceneCard(scene);
  }
}

function resetSceneEditor() {
  remoteScene.value = null;
  draftScene.value = null;
  testResult.value = null;
  testScope.value = "";
  resetProviderUIState();
}

function handleAddProvider() {
  if (!draftScene.value) {
    return;
  }
  const provider = createProvider(draftScene.value.scene);
  provider.name = buildUniqueProviderName("新节点");
  draftScene.value.providers.push(provider);
  setProviderSecretEditor(provider, true);
}

function handleAddProviderPreset(key: ProviderPresetKey) {
  if (!draftScene.value) {
    return;
  }
  const provider = createProvider(draftScene.value.scene);
  if (key === "openai-image") {
    provider.name = buildUniqueProviderName("图片生成节点");
    provider.model = "gpt-image-1";
    provider.endpointMode = "images_generations";
    provider.responseFormat = "b64_json";
    provider.extra = normalizeProviderExtra(provider);
  } else {
    provider.name = buildUniqueProviderName("OpenAI 兼容节点");
    provider.model = draftScene.value.scene === "title" ? "gpt-4.1-mini" : "gpt-4.1";
    provider.endpointMode = "chat_completions";
    provider.responseFormat = "auto";
    provider.extra = normalizeProviderExtra(provider);
  }
  draftScene.value.providers.push(provider);
  setProviderSecretEditor(provider, true);
}

async function handleRemoveProvider(index: number) {
  if (!draftScene.value) {
    return;
  }
  try {
    await ElMessageBox.confirm(
      "删除后该节点会从当前草稿里移除，保存后才会真正生效。",
      "确认删除节点",
      {
        type: "warning",
      },
    );
    draftScene.value.providers.splice(index, 1);
  } catch {
    return;
  }
}

function handleProviderMenuCommand(command: string, index: number) {
  if (command === "move-up") {
    moveProvider(index, -1);
    return;
  }
  if (command === "move-down") {
    moveProvider(index, 1);
    return;
  }
  if (command === "duplicate") {
    handleDuplicateProvider(index);
    return;
  }
  if (command === "delete") {
    handleRemoveProvider(index);
  }
}

function handleDuplicateProvider(index: number) {
  if (!draftScene.value) {
    return;
  }
  const source = draftScene.value.providers[index];
  const copied = JSON.parse(JSON.stringify(source)) as AIRoutingProviderConfig;
  copied.id = buildDuplicateProviderId(source.id || `provider-${index + 1}`);
  copied.name = buildUniqueProviderName(
    source.name ? `${source.name} 副本` : "新节点",
  );
  copied.apiKey = "";
  copied.apiKeyMasked = "";
  copied.hasAPIKey = false;
  copied.clearApiKey = false;
  draftScene.value.providers.splice(index + 1, 0, copied);
  setProviderSecretEditor(copied, true);
}

function moveProvider(index: number, offset: number) {
  if (!draftScene.value) {
    return;
  }
  const target = index + offset;
  if (target < 0 || target >= draftScene.value.providers.length) {
    return;
  }
  const items = draftScene.value.providers;
  const [current] = items.splice(index, 1);
  items.splice(target, 0, current);
}

function startTestingTimer() {
  testingStartedAt.value = Date.now();
  testingElapsedSeconds.value = 0;
  stopTestingTimer();
  testingTimer = window.setInterval(() => {
    if (testingStartedAt.value) {
      testingElapsedSeconds.value = Math.floor(
        (Date.now() - testingStartedAt.value) / 1000,
      );
    }
  }, 1000);
}

function stopTestingTimer() {
  if (testingTimer) {
    window.clearInterval(testingTimer);
    testingTimer = null;
  }
}

function validateBeforeAction(actionLabel: string) {
  if (!blockingValidationCount.value && !sceneBlockingValidationMessages.value.length) {
    return true;
  }
  if (blockingValidationCount.value) {
    touchAllProviderFields();
  }
  const total =
    blockingValidationCount.value + sceneBlockingValidationMessages.value.length;
  ElMessage.warning(`${actionLabel}前请先修正 ${total} 个配置问题`);
  return false;
}

async function handleSaveScene() {
  await saveCurrentScene();
}

async function saveCurrentScene(successMessage = "场景配置已保存") {
  if (!draftScene.value) {
    return false;
  }
  if (!validateBeforeAction("保存")) {
    return false;
  }
  if (isDirty.value) {
    try {
      await ElMessageBox.confirm(
        h(SaveSceneConfirmContent, { model: saveConfirmModel.value }),
        `发布「${currentSceneTitle.value}」配置变更`,
        {
          type: "warning",
          customClass: "save-confirm-message-box",
          confirmButtonText: "保存并发布",
          cancelButtonText: "再检查一下",
          dangerouslyUseHTMLString: false,
        },
      );
    } catch {
      return false;
    }
  }
  savingScene.value = true;
  try {
    const response = await adminApi.updateAIRoutingScene(
      currentSceneKey.value,
      buildScenePayload(draftScene.value),
    );
    remoteScene.value = hydrateScene(response.scene);
    draftScene.value = hydrateScene(response.scene);
    resetProviderUIState();
    await loadSceneSummaries();
    await Promise.all([
      loadRecentAudits(),
      loadAudits(),
      loadSceneCardHealthData(currentSceneKey.value, response.scene),
    ]);
    ElMessage.success(successMessage);
    return true;
  } catch (error) {
    ElMessage.error(extractMessage(error));
    return false;
  } finally {
    savingScene.value = false;
  }
}

const saveConfirmModel = computed(() =>
  buildSaveSceneConfirmModel({
    sceneTitle: currentSceneTitle.value,
    diffItems: sceneDiff.value,
    testing: testingScene.value,
    testingElapsedSeconds: testingElapsedSeconds.value,
    testPassed: testResult.value?.ok,
    providerName: providerDisplayName,
  }),
);

async function handleTestScene() {
  if (testingScene.value) {
    return;
  }
  if (!validateBeforeAction("测试")) {
    return;
  }
  await runSceneTest("当前草稿测试", draftScene.value);
}

async function handleTestSingleProvider(index: number) {
  if (!draftScene.value || testingScene.value) {
    return;
  }
  const provider = draftScene.value.providers[index];
  if (!validateBeforeAction("测试")) {
    return;
  }
  const payload = buildScenePayload(draftScene.value);
  payload.enabled = true;
  payload.maxAttempts = 1;
  payload.providers = payload.providers.map((item, itemIndex) => ({
    ...item,
    enabled: itemIndex === index,
  }));
  singleTestProviderId.value = provider.id;
  await runSceneTest(`单节点测试：${provider.name || provider.id}`, payload);
  singleTestProviderId.value = "";
}

async function runSceneTest(scope: string, scene: AIRoutingSceneConfig | null) {
  if (!scene || testingScene.value) {
    return;
  }
  testingScene.value = true;
  startTestingTimer();
  try {
    const response = await adminApi.testAIRoutingScene(
      currentSceneKey.value,
      buildScenePayload(scene),
    );
    testResult.value = response.result;
    testScope.value = scope;
    recordProviderTestState(response.result);
    const [latestTestAudit, latestAlertOverview] = await Promise.all([
      loadLatestTestAuditForScene(currentSceneKey.value),
      adminApi.getAIRoutingAlertsOverview().catch(() => null),
      loadRecentAudits(),
      loadAudits(),
    ]);
    sceneLatestTestAuditMap.value = {
      ...sceneLatestTestAuditMap.value,
      [currentSceneKey.value]: latestTestAudit,
    };
    if (latestAlertOverview?.overview) {
      alertOverview.value = latestAlertOverview.overview;
    }
    await nextTick();
    updateActiveAnchorSection();
    if (response.result.ok) {
      ElMessage.success("路由测试通过");
    } else {
      ElMessage.warning(
        displayRouteTestMessage(response.result.message) || "路由测试失败",
      );
    }
  } catch (error) {
    ElMessage.error(extractMessage(error));
  } finally {
    testingScene.value = false;
    stopTestingTimer();
  }
}

function discardDraftChanges(message?: string) {
  if (!remoteScene.value) {
    return;
  }
  draftScene.value = hydrateScene(remoteScene.value);
  testResult.value = null;
  testScope.value = "";
  resetProviderUIState();
  if (message) {
    ElMessage.info(message);
  }
}

async function guardUnsavedChanges(actionLabel: string) {
  if (!isDirty.value) {
    return true;
  }
  const decision = await resolveUnsavedDraftAction(actionLabel);
  if (decision === "cancel") {
    return false;
  }
  if (decision === "save") {
    return await saveCurrentScene(`${actionLabel}前已保存当前场景`);
  }
  discardDraftChanges("已放弃未保存草稿");
  return true;
}

async function resolveUnsavedDraftAction(
  actionLabel: string,
): Promise<"save" | "discard" | "cancel"> {
  try {
    await ElMessageBox.confirm(
      `当前场景有未保存的改动。${actionLabel}前，请先处理这些草稿。`,
      "处理未保存草稿",
      {
        type: "warning",
        confirmButtonText: "保存并继续",
        cancelButtonText: "放弃更改",
        distinguishCancelAndClose: true,
        closeOnClickModal: false,
        closeOnPressEscape: false,
      },
    );
    return "save";
  } catch (error) {
    return error === "cancel" ? "discard" : "cancel";
  }
}

function buildUniqueProviderName(seed: string) {
  const base = seed.trim() || "新节点";
  const used = new Set(
    draftScene.value?.providers
      .map((item) => item.name.trim())
      .filter(Boolean) || [],
  );
  if (!used.has(base)) return base;
  let index = 2;
  let candidate = `${base} ${index}`;
  while (used.has(candidate)) {
    index += 1;
    candidate = `${base} ${index}`;
  }
  return candidate;
}

function buildDuplicateProviderId(seed: string) {
  if (!draftScene.value) {
    return `${seed}-copy`;
  }
  const used = new Set(draftScene.value.providers.map((item) => item.id));
  let candidate = `${seed}-copy`;
  let index = 2;
  while (used.has(candidate)) {
    candidate = `${seed}-copy-${index}`;
    index += 1;
  }
  return candidate;
}

function scrollToTestCard() {
  nextTick(() => {
    if (testCardRef.value) {
      scrollElementToViewport(testCardRef.value);
    }
  });
}

function focusMainEditor() {
  nextTick(() => {
    if (mainEditorRef.value) {
      scrollElementToViewport(mainEditorRef.value);
    }
    mainEditorRef.value?.focus();
  });
}

function handleSceneArrowKey(offset: number) {
  const currentIndex = sceneKeys.indexOf(currentSceneKey.value);
  if (currentIndex < 0) {
    return;
  }
  const nextIndex =
    (currentIndex + offset + sceneKeys.length) % sceneKeys.length;
  shouldFocusSceneAfterChange.value = true;
  handleSceneChange(sceneKeys[nextIndex]);
}

function focusSceneCard(scene: AIRoutingSceneKey) {
  nextTick(() => {
    sceneOverviewRef.value?.focusScene(scene);
  });
}

function resetAuditFilters() {
  auditAction.value = "";
  auditOperator.value = "";
  auditSettingKey.value = "";
  auditTimeRange.value = [];
  auditPageSize.value = 20;
  auditPage.value = 1;
  loadAudits();
}

function applyAuditFilters() {
  auditPage.value = 1;
  loadAudits();
}

function handleAuditPageSizeChange(value: number | string) {
  auditPageSize.value = Number(value) || 20;
  auditPage.value = 1;
  loadAudits();
}

function handleAuditPageChange(page: number) {
  auditPage.value = page;
  loadAudits();
}

function extractMessage(error: unknown) {
  return error instanceof Error ? error.message : "请求失败";
}
</script>

<style src="./ai-providers-page.css"></style>
