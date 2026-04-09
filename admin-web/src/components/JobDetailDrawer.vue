<template>
  <el-drawer
    :model-value="modelValue"
    :size="drawerSize"
    title="任务详情"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <PageState v-if="loading" mode="loading" title="正在加载任务详情" compact />
    <PageState
      v-else-if="!detail"
      mode="empty"
      title="暂无任务详情"
      description="请选择一条任务记录查看。"
      compact
    />
    <template v-else>
      <div class="detail-stack">
        <section class="detail-panel">
          <div class="detail-panel__header">
            <div>
              <div class="detail-panel__title">核心信息</div>
              <div class="detail-panel__subtitle">快速确认任务状态、目标对象与最终使用的模型。</div>
            </div>
            <StatusTag :tone="toneForStatus(detail.job.status)" :text="displayJobStatus(detail.job.status)" />
          </div>

          <el-descriptions :column="descriptionColumns" border>
            <el-descriptions-item label="任务 ID">
              <div class="detail-inline">
                <span class="mono-text">{{ detail.job.id }}</span>
                <CopyTextButton :text="String(detail.job.id)" label="复制" />
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <StatusTag :tone="toneForStatus(detail.job.status)" :text="displayJobStatus(detail.job.status)" />
            </el-descriptions-item>
            <el-descriptions-item label="场景">{{ displayScene(detail.job.scene) }}</el-descriptions-item>
            <el-descriptions-item label="触发来源">{{ displayTriggerSource(detail.job.triggerSource) }}</el-descriptions-item>
            <el-descriptions-item label="目标类型">{{ detail.job.targetType || '-' }}</el-descriptions-item>
            <el-descriptions-item label="目标 ID">
              <div class="detail-inline">
                <span class="mono-text">{{ detail.job.targetId || '-' }}</span>
                <CopyTextButton :text="detail.job.targetId" label="复制" />
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="最终 Provider">{{ detail.job.finalProvider || '-' }}</el-descriptions-item>
            <el-descriptions-item label="最终 Model">
              <span class="mono-text">{{ detail.job.finalModel || '-' }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="开始时间">{{ formatDateTime(detail.job.startedAt) }}</el-descriptions-item>
            <el-descriptions-item label="完成时间">{{ formatDateTime(detail.job.finishedAt) }}</el-descriptions-item>
            <el-descriptions-item label="耗时">{{ formatDuration(detail.job.durationMs) }}</el-descriptions-item>
            <el-descriptions-item label="降级处理">
              {{ detail.job.fallbackUsed ? '已触发' : '未触发' }}
            </el-descriptions-item>
            <el-descriptions-item label="Request ID" :span="2">
              <div class="detail-inline">
                <span class="mono-text">{{ detail.job.requestId || '-' }}</span>
                <CopyTextButton :text="detail.job.requestId" label="复制 Request ID" />
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="错误摘要" :span="2">
              {{ detail.job.errorMessage || '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </section>

        <JsonViewerCard
          title="任务 Meta JSON"
          description="这里保留任务入参与收尾上下文，方便核对排障线索。"
          :raw="detail.job.metaJson"
        />

        <section class="detail-panel">
          <div class="detail-panel__header">
            <div>
              <div class="detail-panel__title">关联调用</div>
              <div class="detail-panel__subtitle">按时间顺序展示该任务触发的外部调用。</div>
            </div>
            <StatusTag
              tone="neutral"
              :text="detail.calls.length ? `${detail.calls.length} 次调用` : '无关联调用'"
            />
          </div>

          <PageState
            v-if="!detail.calls.length"
            mode="empty"
            title="暂无关联调用"
            description="这条任务暂时没有记录到 AI provider 或 sidecar 调用。"
            compact
          />
          <div v-else class="table-scroll">
            <el-table :data="detail.calls" size="small" style="width: 100%">
              <el-table-column label="状态" width="96">
                <template #default="{ row }">
                  <StatusTag :tone="toneForStatus(row.status)" :text="displayCallStatus(row.status)" />
                </template>
              </el-table-column>
              <el-table-column prop="provider" label="Provider" min-width="150" />
              <el-table-column prop="endpoint" label="Endpoint" min-width="170" show-overflow-tooltip />
              <el-table-column prop="model" label="Model" min-width="180" show-overflow-tooltip />
              <el-table-column label="耗时" width="110">
                <template #default="{ row }">{{ formatDuration(row.latencyMs) }}</template>
              </el-table-column>
              <el-table-column label="时间" width="180">
                <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="120" :fixed="actionColumnFixed">
                <template #default="{ row }">
                  <el-button text size="small" @click="emit('openCall', row)">查看调用</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </section>
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
import type { CallLogRecord, JobRunRecord } from '@/types'
import {
  displayCallStatus,
  displayJobStatus,
  displayScene,
  displayTriggerSource,
  formatDateTime,
  formatDuration,
  toneForStatus
} from '@/utils/admin-display'

defineProps<{
  modelValue: boolean
  detail: { job: JobRunRecord; calls: CallLogRecord[] } | null
  loading?: boolean
}>()

const { isCompactLayout, isMobile } = useResponsive()
const drawerSize = computed(() => (isMobile.value ? '100%' : 'min(760px, 100vw)'))
const descriptionColumns = computed(() => (isMobile.value ? 1 : 2))
const actionColumnFixed = computed(() => (isCompactLayout.value ? false : 'right'))

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'openCall', call: CallLogRecord): void
}>()
</script>
