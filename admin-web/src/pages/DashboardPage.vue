<template>
  <AppShell>
    <div class="page-header">
      <div>
        <h2 class="page-title">AI 概览</h2>
        <div class="page-subtitle">最近 24 小时任务与调用的整体健康度。</div>
      </div>
      <el-segmented v-model="trendRange" :options="trendOptions" @change="loadTrends" />
    </div>

    <div class="metric-grid">
      <div class="metric-card">
        <div class="metric-label">任务总数</div>
        <div class="metric-value">{{ overview?.taskTotal ?? 0 }}</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">任务成功率</div>
        <div class="metric-value">{{ formatPercent(overview?.taskSuccessRate) }}</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">API 成功率</div>
        <div class="metric-value">{{ formatPercent(overview?.apiSuccessRate) }}</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">超时率</div>
        <div class="metric-value">{{ formatPercent(overview?.timeoutRate) }}</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">P95 耗时</div>
        <div class="metric-value">{{ overview?.p95DurationMs ?? 0 }} ms</div>
      </div>
    </div>

    <div class="section-grid">
      <div class="page-card chart-card">
        <div class="page-header" style="margin-bottom: 12px">
          <div>
            <div class="page-title" style="font-size: 20px">趋势</div>
            <div class="page-subtitle">按时间窗口查看任务与调用成功率。</div>
          </div>
        </div>
        <div ref="chartRef" class="chart-box"></div>
      </div>

      <div class="page-card table-card">
        <div class="page-header" style="margin-bottom: 12px">
          <div>
            <div class="page-title" style="font-size: 20px">最近失败</div>
            <div class="page-subtitle">失败和超时任务的最新记录。</div>
          </div>
        </div>
        <el-table :data="overview?.recentFailures || []" size="small" max-height="360">
          <el-table-column prop="scene" label="场景" min-width="120" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="errorMessage" label="错误摘要" min-width="220" show-overflow-tooltip />
        </el-table>
      </div>
    </div>

    <div class="section-grid" style="margin-top: 20px">
      <div class="page-card table-card">
        <div class="page-header" style="margin-bottom: 12px">
          <div>
            <div class="page-title" style="font-size: 20px">按场景分布</div>
            <div class="page-subtitle">任务量与成功率分布。</div>
          </div>
        </div>
        <el-table :data="overview?.byScene || []" size="small">
          <el-table-column prop="name" label="场景" min-width="140" />
          <el-table-column prop="total" label="总数" width="100" />
          <el-table-column label="成功率" width="120">
            <template #default="{ row }">{{ formatPercent(row.successRate) }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="page-card table-card">
        <div class="page-header" style="margin-bottom: 12px">
          <div>
            <div class="page-title" style="font-size: 20px">按 Provider / Model 分布</div>
            <div class="page-subtitle">调用侧热点模型与 Provider。</div>
          </div>
        </div>
        <el-table :data="[...(overview?.byProvider || []), ...(overview?.byModel || [])]" size="small">
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="total" label="总数" width="90" />
          <el-table-column label="成功率" width="110">
            <template #default="{ row }">{{ formatPercent(row.successRate) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </AppShell>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { ElMessage } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import * as adminApi from '@/api/admin'
import type { DashboardOverview, TrendBucket } from '@/types'

const chartRef = ref<HTMLDivElement | null>(null)
const chart = ref<echarts.ECharts | null>(null)
const overview = ref<DashboardOverview | null>(null)
const trends = ref<TrendBucket[]>([])
const trendRange = ref('24h')
const trendOptions = [
  { label: '24 小时', value: '24h' },
  { label: '7 天', value: '7d' },
  { label: '30 天', value: '30d' }
]

async function loadOverview() {
  try {
    const data = await adminApi.getDashboardOverview()
    overview.value = data.overview
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载概览失败')
  }
}

async function loadTrends() {
  try {
    const data = await adminApi.getDashboardTrends(trendRange.value)
    trends.value = data.items
    await nextTick()
    renderChart()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载趋势失败')
  }
}

function formatPercent(value?: number) {
  return `${((value || 0) * 100).toFixed(1)}%`
}

function renderChart() {
  if (!chartRef.value) return
  if (!chart.value) {
    chart.value = echarts.init(chartRef.value)
  }

  chart.value.setOption({
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
    grid: { left: 28, right: 24, top: 48, bottom: 28, containLabel: true },
    xAxis: {
      type: 'category',
      data: trends.value.map((item) => item.label)
    },
    yAxis: [
      { type: 'value', min: 0, max: 100, axisLabel: { formatter: '{value}%' } },
      { type: 'value', min: 0 }
    ],
    series: [
      {
        name: '任务成功率',
        type: 'line',
        smooth: true,
        yAxisIndex: 0,
        data: trends.value.map((item) => Number(((item.taskSuccessRate || 0) * 100).toFixed(1))),
        lineStyle: { width: 3, color: '#2563eb' },
        areaStyle: { color: 'rgba(37, 99, 235, 0.12)' }
      },
      {
        name: 'API 成功率',
        type: 'line',
        smooth: true,
        yAxisIndex: 0,
        data: trends.value.map((item) => Number(((item.apiSuccessRate || 0) * 100).toFixed(1))),
        lineStyle: { width: 3, color: '#f97316' },
        areaStyle: { color: 'rgba(249, 115, 22, 0.08)' }
      },
      {
        name: '任务总数',
        type: 'bar',
        yAxisIndex: 1,
        data: trends.value.map((item) => item.taskTotal),
        itemStyle: { color: 'rgba(15, 23, 42, 0.16)', borderRadius: [8, 8, 0, 0] }
      }
    ]
  })
}

function handleResize() {
  chart.value?.resize()
}

watch(trends, () => renderChart())

onMounted(async () => {
  window.addEventListener('resize', handleResize)
  await Promise.all([loadOverview(), loadTrends()])
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chart.value?.dispose()
})
</script>
