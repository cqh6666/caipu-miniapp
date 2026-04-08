<template>
  <AppShell>
    <div class="page-header">
      <div>
        <h2 class="page-title">AI 任务</h2>
        <div class="page-subtitle">查看每次业务任务的最终结果与关联调用。</div>
      </div>
    </div>

    <div class="page-card table-card">
      <div class="filter-bar">
        <el-select v-model="filters.scene" clearable placeholder="场景">
          <el-option label="parse_summary" value="parse_summary" />
          <el-option label="flowchart" value="flowchart" />
          <el-option label="title_refine" value="title_refine" />
        </el-select>
        <el-select v-model="filters.status" clearable placeholder="状态">
          <el-option label="success" value="success" />
          <el-option label="failed" value="failed" />
          <el-option label="timeout" value="timeout" />
          <el-option label="fallback" value="fallback" />
        </el-select>
        <el-select v-model="filters.triggerSource" clearable placeholder="触发来源">
          <el-option label="worker" value="worker" />
          <el-option label="manual" value="manual" />
          <el-option label="preview" value="preview" />
        </el-select>
        <el-input v-model="filters.targetId" placeholder="target_id" clearable />
        <el-button type="primary" @click="loadJobs">筛选</el-button>
      </div>

      <el-table :data="result.items" @row-click="openJobDetail">
        <el-table-column prop="scene" label="场景" min-width="130" />
        <el-table-column prop="triggerSource" label="来源" width="100" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="finalProvider" label="最终 Provider" min-width="150" />
        <el-table-column prop="finalModel" label="最终 Model" min-width="180" show-overflow-tooltip />
        <el-table-column prop="durationMs" label="耗时" width="100" />
        <el-table-column prop="errorMessage" label="错误摘要" min-width="220" show-overflow-tooltip />
        <el-table-column prop="startedAt" label="开始时间" width="180" />
      </el-table>

      <div style="display: flex; justify-content: flex-end; margin-top: 16px">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          layout="total, prev, pager, next"
          :total="result.total"
          @current-change="loadJobs"
        />
      </div>
    </div>

    <el-drawer v-model="drawerVisible" size="720px" title="任务详情">
      <template v-if="jobDetail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="场景">{{ jobDetail.job.scene }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ jobDetail.job.status }}</el-descriptions-item>
          <el-descriptions-item label="目标">{{ jobDetail.job.targetType }} / {{ jobDetail.job.targetId }}</el-descriptions-item>
          <el-descriptions-item label="触发来源">{{ jobDetail.job.triggerSource }}</el-descriptions-item>
          <el-descriptions-item label="Provider">{{ jobDetail.job.finalProvider || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Model">{{ jobDetail.job.finalModel || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Request ID" :span="2">{{ jobDetail.job.requestId || '-' }}</el-descriptions-item>
          <el-descriptions-item label="错误摘要" :span="2">{{ jobDetail.job.errorMessage || '-' }}</el-descriptions-item>
        </el-descriptions>

        <h3 style="margin: 24px 0 12px">Meta</h3>
        <pre class="meta-block">{{ formatMeta(jobDetail.job.metaJson) }}</pre>

        <h3 style="margin: 24px 0 12px">关联调用</h3>
        <el-table :data="jobDetail.calls" size="small">
          <el-table-column prop="provider" label="Provider" min-width="150" />
          <el-table-column prop="endpoint" label="Endpoint" min-width="150" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="httpStatus" label="HTTP" width="80" />
          <el-table-column prop="latencyMs" label="耗时" width="90" />
          <el-table-column prop="errorType" label="错误类型" width="110" />
          <el-table-column prop="errorMessage" label="错误摘要" min-width="180" show-overflow-tooltip />
        </el-table>
      </template>
    </el-drawer>
  </AppShell>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import * as adminApi from '@/api/admin'
import type { CallLogRecord, JobRunRecord, PaginationResult } from '@/types'

const page = ref(1)
const pageSize = ref(20)
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

const drawerVisible = ref(false)
const jobDetail = ref<{ job: JobRunRecord; calls: CallLogRecord[] } | null>(null)

function buildQuery() {
  const query = new URLSearchParams()
  query.set('page', String(page.value))
  query.set('pageSize', String(pageSize.value))
  if (filters.scene) query.set('scene', filters.scene)
  if (filters.status) query.set('status', filters.status)
  if (filters.triggerSource) query.set('triggerSource', filters.triggerSource)
  if (filters.targetId) query.set('targetId', filters.targetId)
  return query
}

async function loadJobs() {
  try {
    const data = await adminApi.listJobs(buildQuery())
    result.value = data.result
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载任务失败')
  }
}

async function openJobDetail(row: JobRunRecord) {
  try {
    const data = await adminApi.getJobDetail(row.id)
    jobDetail.value = data
    drawerVisible.value = true
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载详情失败')
  }
}

function formatMeta(raw: string) {
  try {
    return JSON.stringify(JSON.parse(raw || '{}'), null, 2)
  } catch {
    return raw || '{}'
  }
}

loadJobs()
</script>
