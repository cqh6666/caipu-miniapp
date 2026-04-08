<template>
  <AppShell>
    <div class="page-header">
      <div>
        <h2 class="page-title">API 调用</h2>
        <div class="page-subtitle">统一查看 AI provider / sidecar 的调用明细。</div>
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
        </el-select>
        <el-input v-model="filters.provider" clearable placeholder="provider" />
        <el-input v-model="filters.model" clearable placeholder="model" />
        <el-input v-model="filters.requestId" clearable placeholder="request_id" />
        <el-button type="primary" @click="loadCalls">筛选</el-button>
      </div>

      <el-table :data="result.items">
        <el-table-column prop="scene" label="场景" width="120" />
        <el-table-column prop="provider" label="Provider" min-width="150" />
        <el-table-column prop="endpoint" label="Endpoint" min-width="150" />
        <el-table-column prop="model" label="Model" min-width="180" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="httpStatus" label="HTTP" width="80" />
        <el-table-column prop="latencyMs" label="耗时(ms)" width="100" />
        <el-table-column prop="errorType" label="错误类型" width="110" />
        <el-table-column prop="errorMessage" label="错误摘要" min-width="220" show-overflow-tooltip />
        <el-table-column prop="requestId" label="Request ID" min-width="180" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="时间" width="180" />
      </el-table>

      <div style="display: flex; justify-content: flex-end; margin-top: 16px">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          layout="total, prev, pager, next"
          :total="result.total"
          @current-change="loadCalls"
        />
      </div>
    </div>
  </AppShell>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import * as adminApi from '@/api/admin'
import type { CallLogRecord, PaginationResult } from '@/types'

const page = ref(1)
const pageSize = ref(20)
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

function buildQuery() {
  const query = new URLSearchParams()
  query.set('page', String(page.value))
  query.set('pageSize', String(pageSize.value))
  if (filters.scene) query.set('scene', filters.scene)
  if (filters.status) query.set('status', filters.status)
  if (filters.provider) query.set('provider', filters.provider)
  if (filters.model) query.set('model', filters.model)
  if (filters.requestId) query.set('requestId', filters.requestId)
  return query
}

async function loadCalls() {
  try {
    const data = await adminApi.listCalls(buildQuery())
    result.value = data.result
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载调用失败')
  }
}

loadCalls()
</script>
