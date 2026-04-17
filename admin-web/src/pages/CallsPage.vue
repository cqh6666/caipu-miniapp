<template>
  <AppShell>
    <div class="page-card table-card">
      <FilterToolbar>
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

        <div style="display: flex; justify-content: flex-end; margin-top: 16px">
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
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import CallDetailDrawer from '@/components/CallDetailDrawer.vue'
import JobDetailDrawer from '@/components/JobDetailDrawer.vue'
import * as adminApi from '@/api/admin'
import type { CallLogRecord, JobRunRecord, PaginationResult } from '@/types'
import {
  callStatusOptions,
  displayCallStatus,
  displayScene,
  formatDateTime,
  formatDuration,
  sceneOptions,
  toneForStatus
} from '@/utils/admin-display'
import { buildRouteQuery, readDateRange, readQueryNumber, readQueryString, writeDateRange, type DateRangeValue } from '@/utils/route-query'
import { useResponsive } from '@/composables/useResponsive'

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
  provider: '',
  model: '',
  requestId: ''
})

const result = ref<PaginationResult<CallLogRecord>>({
  items: [],
  total: 0,
  page: 1,
  pageSize: 20
})

const callDrawerVisible = ref(false)
const selectedCall = ref<CallLogRecord | null>(null)
const jobDrawerVisible = ref(false)
const jobDetailLoading = ref(false)
const jobDetail = ref<{ job: JobRunRecord; calls: CallLogRecord[] } | null>(null)
const actionColumnFixed = computed(() => (isCompactLayout.value ? false : 'right'))

function syncStateFromRoute() {
  page.value = readQueryNumber(route.query, 'page', 1)
  filters.scene = readQueryString(route.query, 'scene')
  filters.status = readQueryString(route.query, 'status')
  filters.provider = readQueryString(route.query, 'provider')
  filters.model = readQueryString(route.query, 'model')
  filters.requestId = readQueryString(route.query, 'requestId')
  timeRange.value = readDateRange(route.query)
}

function buildListRouteQuery(nextPage = page.value) {
  return buildRouteQuery({
    page: nextPage > 1 ? nextPage : undefined,
    scene: filters.scene || undefined,
    status: filters.status || undefined,
    provider: filters.provider || undefined,
    model: filters.model || undefined,
    requestId: filters.requestId || undefined,
    ...writeDateRange(timeRange.value)
  })
}

function buildRequestQuery() {
  const query = new URLSearchParams()
  query.set('page', String(page.value))
  query.set('pageSize', '20')
  if (filters.scene) query.set('scene', filters.scene)
  if (filters.status) query.set('status', filters.status)
  if (filters.provider) query.set('provider', filters.provider)
  if (filters.model) query.set('model', filters.model)
  if (filters.requestId) query.set('requestId', filters.requestId)
  if (timeRange.value.length) {
    query.set('timeFrom', timeRange.value[0].toISOString())
    query.set('timeTo', timeRange.value[1].toISOString())
  }
  return query
}

async function loadCalls() {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await adminApi.listCalls(buildRequestQuery())
    result.value = data.result
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载调用失败'
  } finally {
    loading.value = false
  }
}

async function applyFilters() {
  const nextQuery = buildListRouteQuery(1)
  if (JSON.stringify(route.query) === JSON.stringify(nextQuery)) {
    page.value = 1
    await loadCalls()
    return
  }
  await router.replace({ query: nextQuery })
}

async function resetFilters() {
  filters.scene = ''
  filters.status = ''
  filters.provider = ''
  filters.model = ''
  filters.requestId = ''
  timeRange.value = []
  if (!Object.keys(route.query).length) {
    page.value = 1
    await loadCalls()
    return
  }
  await router.replace({ query: {} })
}

async function handlePageChange(nextPage: number) {
  await router.replace({ query: buildListRouteQuery(nextPage) })
}

function openCallDetail(call: CallLogRecord) {
  selectedCall.value = call
  callDrawerVisible.value = true
}

async function openJobDetail(jobId: number) {
  callDrawerVisible.value = false
  jobDrawerVisible.value = true
  jobDetailLoading.value = true
  try {
    const data = await adminApi.getJobDetail(jobId)
    jobDetail.value = data
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载任务详情失败')
    jobDetail.value = null
  } finally {
    jobDetailLoading.value = false
  }
}

watch(
  () => route.fullPath,
  () => {
    syncStateFromRoute()
    void loadCalls()
  },
  { immediate: true }
)
</script>
