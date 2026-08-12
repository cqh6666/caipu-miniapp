<template>
  <AppShell>
    <template #toolbar>
      <span v-if="lastRefreshed" class="topbar-refreshed">更新于 {{ lastRefreshed }}</span>
    </template>
    <div class="page-card table-card">
      <FilterToolbar :active-filters="activeFilters" :on-clear-all="hasActiveFilters ? resetFilters : undefined">
        <el-select v-model="filters.scene" clearable placeholder="场景">
          <el-option v-for="item in sceneOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="filters.status" clearable placeholder="状态">
          <el-option v-for="item in callStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-input
          v-model.trim="filters.provider"
          clearable
          placeholder="provider"
          @keyup.enter="applyFilters"
        />
        <el-input v-model.trim="filters.model" clearable placeholder="model" @keyup.enter="applyFilters" />
        <el-input
          v-model.trim="filters.requestId"
          clearable
          placeholder="request_id"
          @keyup.enter="applyFilters"
        />
        <el-date-picker
          v-model="timeRange"
          type="datetimerange"
          unlink-panels
          range-separator="至"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
        />
        <template #actions>
          <el-button @click="resetFilters">重置</el-button>
          <el-button type="primary" :loading="loading" @click="applyFilters">筛选</el-button>
        </template>
      </FilterToolbar>

      <el-alert
        v-if="errorMessage && result.items.length"
        class="setting-alert"
        type="warning"
        :closable="false"
        :title="errorMessage"
      />

      <PageState v-if="loading && !result.items.length" mode="loading" title="正在加载调用列表" compact />
      <PageState
        v-else-if="errorMessage && !result.items.length"
        mode="error"
        title="调用列表加载失败"
        :description="errorMessage"
        compact
        @retry="loadCalls"
      />
      <PageState
        v-else-if="!result.items.length"
        mode="empty"
        title="暂无调用记录"
        description="当前筛选条件下没有命中的调用记录，可以扩大时间范围再试。"
        compact
      />
      <template v-else>
        <div class="table-scroll">
          <el-table :data="result.items" size="small" style="width: 100%">
            <el-table-column label="场景" width="120">
              <template #default="{ row }">{{ displayScene(row.scene) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <StatusTag :tone="toneForStatus(row.status)" :text="displayCallStatus(row.status)" />
              </template>
            </el-table-column>
            <el-table-column prop="provider" label="Provider" min-width="150" />
            <el-table-column label="Endpoint / Model" min-width="220">
              <template #default="{ row }">
                <div class="mono-text">{{ row.endpoint || '-' }}</div>
                <div class="mono-text" style="color: var(--color-text-subtle)">{{ row.model || '-' }}</div>
              </template>
            </el-table-column>
            <el-table-column label="耗时" width="110">
              <template #default="{ row }">{{ formatDuration(row.latencyMs) }}</template>
            </el-table-column>
            <el-table-column prop="httpStatus" label="HTTP" width="80" />
            <el-table-column label="Request ID" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <span class="mono-text">{{ row.requestId || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="错误摘要" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.errorMessage || '-' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="118" :fixed="actionColumnFixed">
              <template #default="{ row }">
                <el-button text size="small" @click="openCallDetail(row)">查看详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="pagination-row">
          <el-pagination
            v-model:current-page="page"
            layout="total, prev, pager, next"
            background
            :total="result.total"
            @current-change="handlePageChange"
          />
        </div>
      </template>
    </div>

    <CallDetailDrawer
      v-model="callDrawerVisible"
      :call="selectedCall"
      @open-job="openJobDetail"
    />
    <JobDetailDrawer
      v-model="jobDrawerVisible"
      :detail="jobDetail"
      :loading="jobDetailLoading"
      @open-call="openCallDetail"
    />
  </AppShell>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import CallDetailDrawer from '@/components/CallDetailDrawer.vue'
import JobDetailDrawer from '@/components/JobDetailDrawer.vue'
import * as adminApi from '@/api/admin'
import type { CallLogRecord } from '@/types'
import {
  callStatusOptions,
  displayCallStatus,
  displayScene,
  formatDateTime,
  formatDuration,
  sceneOptions,
  toneForStatus
} from '@/utils/admin-display'
import { useJobDetailDrawer } from '@/composables/useJobDetailDrawer'
import { useRoutedAdminList } from '@/composables/useRoutedAdminList'
import { useResponsive } from '@/composables/useResponsive'
import { useLastRefreshed } from '@/composables/useLastRefreshed'

const { display: lastRefreshed, mark: markRefreshed } = useLastRefreshed('calls')

const route = useRoute()
const router = useRouter()
const { isCompactLayout } = useResponsive()

const {
  applyFilters,
  errorMessage,
  filters,
  handlePageChange,
  loadList: loadCalls,
  loading,
  page,
  removeFilter,
  resetFilters,
  result,
  syncStateFromRoute,
  timeRange
} = useRoutedAdminList<CallLogRecord, {
  scene: string
  status: string
  provider: string
  model: string
  requestId: string
}>({
  route,
  router,
  initialFilters: { scene: '', status: '', provider: '', model: '', requestId: '' },
  fields: [
    { key: 'scene' },
    { key: 'status' },
    { key: 'provider' },
    { key: 'model' },
    { key: 'requestId' }
  ],
  fetchList: async (query) => (await adminApi.listCalls(query)).result,
  loadErrorMessage: '加载调用失败',
  onLoaded: markRefreshed
})
const {
  callDrawerVisible,
  jobDetail,
  jobDetailLoading,
  jobDrawerVisible,
  openCallDetail,
  openJobDetail,
  selectedCall
} = useJobDetailDrawer({
  loadDetail: adminApi.getJobDetail,
  onError: (error) => ElMessage.error(error instanceof Error ? error.message : '加载任务详情失败')
})
const actionColumnFixed = computed(() => (isCompactLayout.value ? false : 'right'))

function labelFor(options: { label: string; value: string }[], value: string) {
  return options.find((item) => item.value === value)?.label || value
}

const activeFilters = computed(() => {
  const chips: { key: string; label: string; onRemove?: () => void }[] = []
  if (filters.scene) {
    chips.push({ key: 'scene', label: `场景：${labelFor(sceneOptions, filters.scene)}`, onRemove: () => removeFilter('scene') })
  }
  if (filters.status) {
    chips.push({ key: 'status', label: `状态：${labelFor(callStatusOptions, filters.status)}`, onRemove: () => removeFilter('status') })
  }
  if (filters.provider) {
    chips.push({ key: 'provider', label: `provider：${filters.provider}`, onRemove: () => removeFilter('provider') })
  }
  if (filters.model) {
    chips.push({ key: 'model', label: `model：${filters.model}`, onRemove: () => removeFilter('model') })
  }
  if (filters.requestId) {
    chips.push({ key: 'requestId', label: `request_id：${filters.requestId}`, onRemove: () => removeFilter('requestId') })
  }
  if (timeRange.value.length === 2) {
    chips.push({
      key: 'timeRange',
      label: `时间：${formatDateTime(timeRange.value[0].toISOString())} ~ ${formatDateTime(timeRange.value[1].toISOString())}`,
      onRemove: () => removeFilter('timeRange')
    })
  }
  return chips
})

const hasActiveFilters = computed(() => activeFilters.value.length > 0)

watch(
  () => route.fullPath,
  () => {
    syncStateFromRoute()
    void loadCalls()
  },
  { immediate: true }
)
</script>
