<template>
  <div class="routing-panel__header">
    <div>
      <h3 class="routing-panel__title">最近测试结果</h3>
      <div class="routing-panel__subtitle">{{ scopeDescription }}</div>
    </div>
    <div class="routing-panel__tags">
      <StatusTag :tone="summary.tone" :text="summary.text" />
      <StatusTag tone="neutral" :text="`总耗时 ${formatDuration(totalLatency)}`" />
    </div>
  </div>
  <div class="routing-test-card__summary">
    <span>结果：{{ displayMessage(result.message) }}</span>
    <span>最终节点：{{ providerName(result.finalProvider || '') }}</span>
    <span>最终模型：{{ result.finalModel || "-" }}</span>
  </div>
  <div class="table-scroll">
    <el-table :data="result.attempts" size="small" style="width: 100%">
      <el-table-column label="Provider" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ providerName(row.providerId) }}</template>
      </el-table-column>
      <el-table-column prop="model" label="Model" min-width="140" show-overflow-tooltip />
      <el-table-column label="状态" width="120">
        <template #default="{ row }"><StatusTag :tone="toneForStatus(row.status)" :text="displayCallStatus(row.status)" /></template>
      </el-table-column>
      <el-table-column prop="httpStatus" label="HTTP" width="90" />
      <el-table-column prop="latencyMs" label="耗时" width="100">
        <template #default="{ row }">{{ formatDuration(row.latencyMs) }}</template>
      </el-table-column>
      <el-table-column label="错误类型" min-width="120">
        <template #default="{ row }">{{ row.errorType || "-" }}</template>
      </el-table-column>
      <el-table-column label="备注" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          {{ row.skippedByBreaker ? `熔断冷却至 ${formatDateTime(row.breakerOpenUntil)}` : displayMessage(row.errorMessage || "-") }}
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import StatusTag from "@/components/StatusTag.vue";
import type { AIRoutingTestResult } from "@/types";
import {
  displayCallStatus,
  formatDateTime,
  formatDuration,
  toneForStatus,
} from "@/utils/admin-display";

const props = defineProps<{
  result: AIRoutingTestResult;
  summary: { tone: "neutral" | "primary" | "success" | "warning" | "danger"; text: string };
  scopeDescription: string;
  providerName: (providerId: string) => string;
  displayMessage: (message?: string) => string;
}>();
const totalLatency = computed(() =>
  props.result.attempts.reduce((total, item) => total + item.latencyMs, 0),
);
</script>
