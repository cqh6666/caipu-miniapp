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
          <el-option v-for="item in jobStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="filters.triggerSource" clearable placeholder="触发来源">
          <el-option
            v-for="item in triggerSourceOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
        <el-input
          v-model.trim="filters.targetId"
          clearable
          placeholder="target_id"
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

      <PageState v-if="loading && !result.items.length" mode="loading" title="正在加载任务列表" compact />
      <PageState
        v-else-if="errorMessage && !result.items.length"
        mode="error"
        title="任务列表加载失败"
        :description="errorMessage"
        compact
        @retry="loadJobs"
      />
      <PageState
        v-else-if="!result.items.length"
        mode="empty"
        title="暂无任务记录"
        description="当前筛选条件下没有命中的任务，可以调整时间范围或状态再试。"
        compact
      />
      <template v-else>
        <div class="table-scroll">
          <el-table :data="result.items" size="small" style="width: 100%">
            <el-table-column label="场景" min-width="130">
              <template #default="{ row }">{{ displayScene(row.scene) }}</template>
            </el-table-column>
            <el-table-column label="目标" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <div>{{ row.targetType || '-' }}</div>
                <div class="mono-text" style="color: var(--color-text-subtle)">{{ row.targetId || '-' }}</div>
              </template>
            </el-table-column>
            <el-table-column label="来源" width="110">
              <template #default="{ row }">{{ displayTriggerSource(row.triggerSource) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <StatusTag :tone="toneForStatus(row.status)" :text="displayJobStatus(row.status)" />
              </template>
            </el-table-column>
            <el-table-column label="Provider / Model" min-width="220">
              <template #default="{ row }">
                <div>{{ row.finalProvider || '-' }}</div>
                <div class="mono-text" style="color: var(--color-text-subtle)">{{ row.finalModel || '-' }}</div>
              </template>
            </el-table-column>
            <el-table-column label="耗时" width="110">
              <template #default="{ row }">{{ formatDuration(row.durationMs) }}</template>
            </el-table-column>
            <el-table-column label="开始时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.startedAt) }}</template>
            </el-table-column>
            <el-table-column label="错误摘要" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.errorMessage || '-' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="118" :fixed="actionColumnFixed">
              <template #default="{ row }">
                <el-button text size="small" @click="openJobDetail(row.id)">查看详情</el-button>
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

    <JobDetailDrawer
      v-model="jobDrawerVisible"
      :detail="jobDetail"
      :loading="jobDetailLoading"
      @open-call="openCallDetail"
    />
    <CallDetailDrawer
      v-model="callDrawerVisible"
      :call="selectedCall"
      @open-job="openJobDetail"
    />
  </AppShell>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import JobDetailDrawer from '@/components/JobDetailDrawer.vue'
import CallDetailDrawer from '@/components/CallDetailDrawer.vue'
import * as adminApi from '@/api/admin'
import type { CallLogRecord, JobRunRecord, PaginationResult } from '@/types'
import {
  displayJobStatus,
  displayScene,
  displayTriggerSource,
  formatDateTime,
  formatDuration,
  jobStatusOptions,
  sceneOptions,
  toneForStatus,
  triggerSourceOptions
} from '@/utils/admin-display'
import { buildRouteQuery, readDateRange, readQueryNumber, readQueryString, writeDateRange, type DateRangeValue } from '@/utils/route-query'
import { useResponsive } from '@/composables/useResponsive'
import { useLastRefreshed } from '@/composables/useLastRefreshed'

const { display: lastRefreshed, mark: markRefreshed } = useLastRefreshed('jobs')

const route = useRoute()
const router = useRouter()
const { isCompactLayout } = useResponsive()

const page = ref(1)
const loading = ref(false)
const errorMessage = ref('')
const timeRange = ref<DateRangeValue>([])
const filters = reactive({
  scene: '',
  status: '',
  triggerSource: '',
  targetId: ''
})

const result = ref<PaginationResult<JobRunRecord>>({
  items: [],
  total: 0,
  page: 1,
  pageSize: 20
})

const activeJobId = ref(0)
const jobDrawerVisible = ref(false)
const jobDetailLoading = ref(false)
const jobDetail = ref<{ job: JobRunRecord; calls: CallLogRecord[] } | null>(null)
const callDrawerVisible = ref(false)
const selectedCall = ref<CallLogRecord | null>(null)
const actionColumnFixed = computed(() => (isCompactLayout.value ? false : 'right'))

function syncStateFromRoute() {
  page.value = readQueryNumber(route.query, 'page', 1)
  filters.scene = readQueryString(route.query, 'scene')
  filters.status = readQueryString(route.query, 'status')
  filters.triggerSource = readQueryString(route.query, 'triggerSource')
  filters.targetId = readQueryString(route.query, 'targetId')
  timeRange.value = readDateRange(route.query)
  activeJobId.value = readQueryNumber(route.query, 'jobId', 0)
}

function buildListRouteQuery(nextPage = page.value, jobId?: number | null) {
  return buildRouteQuery({
    page: nextPage > 1 ? nextPage : undefined,
    scene: filters.scene || undefined,
    status: filters.status || undefined,
    triggerSource: filters.triggerSource || undefined,
    targetId: filters.targetId || undefined,
    jobId: jobId || undefined,
    ...writeDateRange(timeRange.value)
  })
}

function buildRequestQuery() {
  const query = new URLSearchParams()
  query.set('page', String(page.value))
  query.set('pageSize', '20')
  if (filters.scene) query.set('scene', filters.scene)
  if (filters.status) query.set('status', filters.status)
  if (filters.triggerSource) query.set('triggerSource', filters.triggerSource)
  if (filters.targetId) query.set('targetId', filters.targetId)
  if (timeRange.value.length) {
    query.set('timeFrom', timeRange.value[0].toISOString())
    query.set('timeTo', timeRange.value[1].toISOString())
  }
  return query
}

async function loadJobs() {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await adminApi.listJobs(buildRequestQuery())
    result.value = data.result
    markRefreshed()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载任务失败'
  } finally {
    loading.value = false
  }
}

async function applyFilters() {
  const nextQuery = buildListRouteQuery(1)
  if (JSON.stringify(route.query) === JSON.stringify(nextQuery)) {
    page.value = 1
    await loadJobs()
    return
  }
  await router.replace({ query: nextQuery })
}

async function resetFilters() {
  filters.scene = ''
  filters.status = ''
  filters.triggerSource = ''
  filters.targetId = ''
  timeRange.value = []
  if (!Object.keys(route.query).length) {
    page.value = 1
    await loadJobs()
    return
  }
  await router.replace({ query: {} })
}

async function handlePageChange(nextPage: number) {
  await router.replace({ query: buildListRouteQuery(nextPage) })
}

function labelFor(options: { label: string; value: string }[], value: string) {
  return options.find((item) => item.value === value)?.label || value
}

function removeFilter(key: 'scene' | 'status' | 'triggerSource' | 'targetId' | 'timeRange') {
  if (key === 'timeRange') {
    timeRange.value = []
  } else {
    filters[key] = ''
  }
  void applyFilters()
}

const activeFilters = computed(() => {
  const chips: { key: string; label: string; onRemove?: () => void }[] = []
  if (filters.scene) {
    chips.push({ key: 'scene', label: `场景：${labelFor(sceneOptions, filters.scene)}`, onRemove: () => removeFilter('scene') })
  }
  if (filters.status) {
    chips.push({ key: 'status', label: `状态：${labelFor(jobStatusOptions, filters.status)}`, onRemove: () => removeFilter('status') })
  }
  if (filters.triggerSource) {
    chips.push({ key: 'triggerSource', label: `来源：${labelFor(triggerSourceOptions, filters.triggerSource)}`, onRemove: () => removeFilter('triggerSource') })
  }
  if (filters.targetId) {
    chips.push({ key: 'targetId', label: `target_id：${filters.targetId}`, onRemove: () => removeFilter('targetId') })
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

async function fetchJobDetail(jobId: number) {
  callDrawerVisible.value = false
  jobDrawerVisible.value = true
  jobDetailLoading.value = true
  try {
    const data = await adminApi.getJobDetail(jobId)
    jobDetail.value = data
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载详情失败')
    jobDetail.value = null
  } finally {
    jobDetailLoading.value = false
  }
}

async function openJobDetail(jobId: number, options: { syncRoute?: boolean } = {}) {
  const { syncRoute = true } = options
  if (syncRoute) {
    const nextQuery = buildListRouteQuery(page.value, jobId)
    if (JSON.stringify(route.query) !== JSON.stringify(nextQuery)) {
      await router.replace({ query: nextQuery })
      return
    }
  }
  await fetchJobDetail(jobId)
}

function openCallDetail(call: CallLogRecord) {
  selectedCall.value = call
  callDrawerVisible.value = true
}

watch(
  () => route.fullPath,
  async () => {
    syncStateFromRoute()
    await loadJobs()
    if (activeJobId.value > 0) {
      await openJobDetail(activeJobId.value, { syncRoute: false })
      return
    }
    jobDrawerVisible.value = false
    jobDetail.value = null
  },
  { immediate: true }
)

watch(
  () => jobDrawerVisible.value,
  (visible) => {
    if (visible || activeJobId.value <= 0) {
      return
    }
    const nextQuery = buildListRouteQuery(page.value)
    if (JSON.stringify(route.query) === JSON.stringify(nextQuery)) {
      return
    }
    void router.replace({ query: nextQuery })
  }
)
</script>
