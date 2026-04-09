<template>
  <el-drawer
    :model-value="modelValue"
    :size="drawerSize"
    title="调用详情"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <PageState v-if="loading" mode="loading" title="正在加载调用详情" compact />
    <PageState
      v-else-if="!call"
      mode="empty"
      title="暂无调用详情"
      description="请选择一条调用记录查看。"
      compact
    />
    <template v-else>
      <div class="detail-stack">
        <section class="detail-panel">
          <div class="detail-panel__header">
            <div>
              <div class="detail-panel__title">核心信息</div>
              <div class="detail-panel__subtitle">统一查看请求链路、错误摘要与关联任务。</div>
            </div>
            <StatusTag :tone="toneForStatus(call.status)" :text="displayCallStatus(call.status)" />
          </div>

          <el-descriptions :column="descriptionColumns" border>
            <el-descriptions-item label="场景">{{ displayScene(call.scene) }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <StatusTag :tone="toneForStatus(call.status)" :text="displayCallStatus(call.status)" />
            </el-descriptions-item>
            <el-descriptions-item label="Provider">{{ call.provider || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Model">
              <span class="mono-text">{{ call.model || '-' }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="Endpoint">
              <span class="mono-text">{{ call.endpoint || '-' }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="耗时">{{ formatDuration(call.latencyMs) }}</el-descriptions-item>
            <el-descriptions-item label="HTTP">{{ call.httpStatus || '-' }}</el-descriptions-item>
            <el-descriptions-item label="错误类型">{{ call.errorType || '-' }}</el-descriptions-item>
            <el-descriptions-item label="请求时间">{{ formatDateTime(call.createdAt) }}</el-descriptions-item>
            <el-descriptions-item label="关联任务">
              <div class="detail-inline">
                <span class="mono-text">{{ call.jobRunId || '-' }}</span>
                <el-button
                  v-if="call.jobRunId"
                  text
                  size="small"
                  @click="emit('openJob', call.jobRunId)"
                >
                  查看任务
                </el-button>
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="Request ID" :span="2">
              <div class="detail-inline">
                <span class="mono-text">{{ call.requestId || '-' }}</span>
                <CopyTextButton :text="call.requestId" label="复制 Request ID" />
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="错误摘要" :span="2">
              {{ call.errorMessage || '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </section>

        <JsonViewerCard
          title="Meta JSON"
          description="保留原始调用上下文，便于回放和排障。"
          :raw="call.metaJson"
        />
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import JsonViewerCard from '@/components/JsonViewerCard.vue'
import CopyTextButton from '@/components/CopyTextButton.vue'
import { useResponsive } from '@/composables/useResponsive'
import type { CallLogRecord } from '@/types'
import {
  displayCallStatus,
  displayScene,
  formatDateTime,
  formatDuration,
  toneForStatus
} from '@/utils/admin-display'

defineProps<{
  modelValue: boolean
  call: CallLogRecord | null
  loading?: boolean
}>()

const { isMobile } = useResponsive()
const drawerSize = computed(() => (isMobile.value ? '100%' : 'min(720px, 100vw)'))
const descriptionColumns = computed(() => (isMobile.value ? 1 : 2))

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'openJob', jobRunId: number): void
}>()
</script>
