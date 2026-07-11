<template>
  <div class="channel-popover routing-alert-panel" aria-live="polite">
    <div class="channel-popover__title">连续异常告警 · {{ sceneTitle }}</div>
    <p class="channel-popover__text">{{ description }}</p>
    <div v-if="overview" class="routing-alert-overview-meta">
      <span>阈值：{{ overview.failureThreshold }} 次</span>
      <span>活跃窗口：{{ overview.activeWindowHours }}h</span>
      <span>投递：{{ overview.hasDeliveryConfig ? "可用" : "不完整" }}</span>
    </div>

    <template v-if="hasNodes">
      <div
        v-for="section in sections"
        v-show="section.items.length"
        :key="section.key"
        class="routing-alert-section"
      >
        <div class="routing-alert-section__head">
          <span class="routing-alert-section__title">{{ section.title }} <em>{{ section.items.length }}</em></span>
          <div v-if="section.key !== 'recovered'" class="routing-alert-section__bulk">
            <el-button link size="small" @click="emit('batch-retest', section.items)">批量复测</el-button>
            <el-button v-if="section.key === 'review'" link size="small" @click="emit('batch-archive', section.items)">批量归档</el-button>
          </div>
        </div>
        <div class="routing-alert-section__hint">{{ section.hint }}</div>
        <div
          v-for="node in section.items"
          :key="node.providerId"
          class="routing-alert-node"
          :class="`routing-alert-node--${resolveAlertStatus(node)}`"
        >
          <div class="routing-alert-node__head">
            <span class="routing-alert-node__name">{{ node.providerName }}</span>
            <StatusTag :tone="alertStatusTone(resolveAlertStatus(node))" :text="alertStatusLabel(resolveAlertStatus(node))" />
            <span class="routing-alert-node__model">{{ node.model || "未指定模型" }}</span>
          </div>
          <div class="routing-alert-node__meta">
            <template v-if="node.consecutiveFailures">连续失败 {{ node.consecutiveFailures }} 次</template>
            <template v-if="displayAlertErrorType(node.lastErrorType)"> · 类型：{{ displayAlertErrorType(node.lastErrorType) }}</template>
            <template v-if="node.lastFailedAt"> · <span :title="formatDateTime(node.lastFailedAt)">{{ relativeTime(node.lastFailedAt) }}</span></template>
          </div>
          <div v-if="node.statusReason" class="routing-alert-node__reason">{{ node.statusReason }}</div>
          <div v-if="resolveAlertStatus(node) === 'active' && activeWindowCountdown(node)" class="routing-alert-node__countdown">{{ activeWindowCountdown(node) }}</div>
          <div v-if="node.lastRecoveredAt && resolveAlertStatus(node) === 'recovered'" class="routing-alert-node__recovered">恢复于 {{ formatDateTime(node.lastRecoveredAt) }}</div>
          <div v-if="node.lastErrorMessage" class="routing-alert-node__error">最后错误：{{ node.lastErrorMessage }}</div>
          <div v-if="node.lastRequestId" class="routing-alert-node__reqid">
            reqId：{{ node.lastRequestId }}
            <el-button link size="small" @click="emit('copy-request-id', node.lastRequestId)">复制</el-button>
          </div>
          <div class="routing-alert-node__actions">
            <el-button v-if="node.canRetest" type="primary" size="small" :loading="isPending(node.providerId)" @click="emit('retest', node)">复测并恢复</el-button>
            <el-button v-if="node.canUnmute" size="small" :disabled="isPending(node.providerId)" @click="emit('unmute', node)">解除静默</el-button>
            <el-button v-if="node.canMute" size="small" :disabled="isPending(node.providerId)" @click="emit('mute', node)">静默 24h</el-button>
            <el-button v-if="node.canArchive" size="small" :disabled="isPending(node.providerId)" @click="emit('archive', node)">归档</el-button>
            <el-button link size="small" @click="emit('logs', node)">查看日志</el-button>
          </div>
        </div>
      </div>
      <p class="routing-alert-retest-hint">复测会对该节点发起一次真实上游调用，可能产生额度/费用。</p>
    </template>
    <p v-else class="routing-alert-empty">当前场景无告警，运行正常。</p>
    <el-button type="primary" link @click="emit('config')">前往配置 <el-icon><ArrowRight /></el-icon></el-button>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight } from "@element-plus/icons-vue";
import StatusTag from "@/components/StatusTag.vue";
import type { AIRoutingAlertOverview, AIRoutingAlertOverviewItem } from "@/types";
import { formatDateTime } from "@/utils/admin-display";
import {
  alertStatusLabel,
  alertStatusTone,
  displayAlertErrorType,
  formatAlertRelativeTime,
  resolveAlertStatus,
} from "@/utils/ai-provider-alerts";

type AlertSection = {
  key: string;
  title: string;
  hint: string;
  items: AIRoutingAlertOverviewItem[];
};

const props = defineProps<{
  sceneTitle: string;
  description: string;
  overview: AIRoutingAlertOverview | null;
  sections: AlertSection[];
  hasNodes: boolean;
  pendingActions: Record<string, boolean>;
}>();
const emit = defineEmits<{
  retest: [item: AIRoutingAlertOverviewItem];
  archive: [item: AIRoutingAlertOverviewItem];
  mute: [item: AIRoutingAlertOverviewItem];
  unmute: [item: AIRoutingAlertOverviewItem];
  "batch-retest": [items: AIRoutingAlertOverviewItem[]];
  "batch-archive": [items: AIRoutingAlertOverviewItem[]];
  logs: [item: AIRoutingAlertOverviewItem];
  config: [];
  "copy-request-id": [requestId: string];
}>();

function isPending(providerId: string) {
  return !!props.pendingActions[providerId];
}

function relativeTime(value: string) {
  return formatAlertRelativeTime(value, props.overview?.generatedAt);
}

function activeWindowCountdown(item: AIRoutingAlertOverviewItem) {
  if (!item.activeUntil) return "";
  const end = Date.parse(item.activeUntil);
  const reference = Date.parse(props.overview?.generatedAt || "");
  if (!Number.isFinite(end) || !Number.isFinite(reference) || end <= reference)
    return "";
  const minutes = Math.ceil((end - reference) / 60000);
  if (minutes < 60) return `活跃窗口剩余约 ${minutes} 分钟`;
  const hours = Math.ceil(minutes / 60);
  return `活跃窗口剩余约 ${hours} 小时`;
}
</script>
