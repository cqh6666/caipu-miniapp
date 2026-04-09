<template>
  <AppShell>
    <div class="page-header">
      <div>
        <h2 class="page-title">AI 概览</h2>
        <div class="page-subtitle">最近 {{ overview?.windowHours ?? 24 }} 小时任务与调用的整体健康度。</div>
      </div>
      <div class="page-header__actions">
        <el-segmented v-model="trendRange" :options="trendOptions" @change="handleTrendRangeChange" />
        <el-button :loading="refreshing" @click="refreshPage">刷新数据</el-button>
      </div>
    </div>

    <el-alert
      v-if="overviewError && overview"
      class="settings-summary"
      type="warning"
      :closable="false"
      :title="overviewError"
    />

    <div v-if="overviewLoading && !overview" class="metric-grid">
      <div v-for="item in 5" :key="item" class="metric-card">
        <el-skeleton animated :rows="3" />
      </div>
    </div>
    <div v-else-if="overviewError && !overview" class="page-card table-card">
      <PageState
        mode="error"
        title="概览数据加载失败"
        :description="overviewError"
        @retry="refreshPage"
      />
    </div>
    <div v-else class="metric-grid">
      <div v-for="metric in metricItems" :key="metric.label" class="metric-card">
        <div class="metric-card__header">
          <div class="metric-label">{{ metric.label }}</div>
          <StatusTag :tone="metric.tone" :text="metric.statusText" />
        </div>
        <div class="metric-value">{{ metric.value }}</div>
        <div class="metric-note">{{ metric.note }}</div>
      </div>
    </div>

    <div class="dashboard-grid dashboard-grid--primary">
      <div class="page-card chart-card">
        <div class="subsection-header">
          <div>
            <h3 class="subsection-title">趋势</h3>
            <div class="subsection-subtitle">按时间窗口观察任务成功率、API 成功率与任务总量。</div>
          </div>
          <StatusTag tone="neutral" :text="currentTrendLabel" />
        </div>

        <PageState
          v-if="trendsLoading && !trends.length"
          mode="loading"
          title="正在加载趋势图"
          compact
        />
        <PageState
          v-else-if="trendError && !trends.length"
          mode="error"
          title="趋势数据加载失败"
          :description="trendError"
          compact
          @retry="loadTrends"
        />
        <PageState
          v-else-if="!trends.length"
          mode="empty"
          title="暂无趋势数据"
          description="当前时间窗口内还没有足够的数据生成趋势图。"
          compact
        />
        <div v-else ref="chartRef" class="chart-box"></div>
      </div>

      <div class="page-card table-card">
        <div class="subsection-header">
          <div>
            <h3 class="subsection-title">最近失败</h3>
            <div class="subsection-subtitle">失败与超时任务优先暴露，支持直接进入详情排障。</div>
          </div>
          <StatusTag
            tone="neutral"
            :text="overview?.recentFailures?.length ? `${overview.recentFailures.length} 条记录` : '暂无异常'"
          />
        </div>

        <PageState
          v-if="overviewLoading && !overview"
          mode="loading"
          title="正在加载失败任务"
          compact
        />
        <PageState
          v-else-if="!(overview?.recentFailures?.length)"
          mode="empty"
          title="暂无失败任务"
          description="当前窗口内没有失败或超时任务，整体状态较稳定。"
          compact
        />
        <div v-else class="table-scroll">
          <el-table :data="overview?.recentFailures || []" size="small" style="width: 100%">
            <el-table-column label="场景" min-width="130">
              <template #default="{ row }">{{ displayScene(row.scene) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="96">
              <template #default="{ row }">
                <StatusTag :tone="toneForStatus(row.status)" :text="displayJobStatus(row.status)" />
              </template>
            </el-table-column>
            <el-table-column label="错误摘要" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.errorMessage || '-' }}</template>
            </el-table-column>
            <el-table-column label="开始时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.startedAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="108" fixed="right">
              <template #default="{ row }">
                <el-button text size="small" @click="openJobDetail(row.id)">查看详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>

    <div class="dashboard-grid dashboard-grid--triple" style="margin-top: 20px">
      <div class="page-card table-card">
        <div class="subsection-header">
          <div>
            <h3 class="subsection-title">按场景分布</h3>
            <div class="subsection-subtitle">任务量与成功率分布。</div>
          </div>
        </div>
        <PageState
          v-if="!(overview?.byScene?.length)"
          mode="empty"
          title="暂无场景分布"
          description="当前时间窗口内还没有场景级任务数据。"
          compact
        />
        <div v-else class="table-scroll">
          <el-table :data="overview?.byScene || []" size="small" style="width: 100%">
            <el-table-column label="场景" min-width="130">
              <template #default="{ row }">{{ displayScene(row.name) }}</template>
            </el-table-column>
            <el-table-column prop="total" label="总数" width="90" />
            <el-table-column label="成功率" width="110">
              <template #default="{ row }">{{ formatPercent(row.successRate) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <div class="page-card table-card">
        <div class="subsection-header">
          <div>
            <h3 class="subsection-title">Provider 热点</h3>
            <div class="subsection-subtitle">调用量最高的 Provider 与成功率。</div>
          </div>
        </div>
        <PageState
          v-if="!(overview?.byProvider?.length)"
          mode="empty"
          title="暂无 Provider 数据"
          description="当前时间窗口内还没有调用侧热点分布。"
          compact
        />
        <div v-else class="table-scroll">
          <el-table :data="overview?.byProvider || []" size="small" style="width: 100%">
            <el-table-column prop="name" label="Provider" min-width="160" show-overflow-tooltip />
            <el-table-column prop="total" label="总数" width="90" />
            <el-table-column label="成功率" width="110">
              <template #default="{ row }">{{ formatPercent(row.successRate) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <div class="page-card table-card">
        <div class="subsection-header">
          <div>
            <h3 class="subsection-title">Model 热点</h3>
            <div class="subsection-subtitle">调用量最高的模型分布与成功率。</div>
          </div>
        </div>
        <PageState
          v-if="!(overview?.byModel?.length)"
          mode="empty"
          title="暂无 Model 数据"
          description="当前时间窗口内还没有模型侧热点分布。"
          compact
        />
        <div v-else class="table-scroll">
          <el-table :data="overview?.byModel || []" size="small" style="width: 100%">
            <el-table-column prop="name" label="Model" min-width="180" show-overflow-tooltip />
            <el-table-column prop="total" label="总数" width="90" />
            <el-table-column label="成功率" width="110">
              <template #default="{ row }">{{ formatPercent(row.successRate) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>
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
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { BarChart, LineChart } from 'echarts/charts'
import {
  GridComponent,
  LegendComponent,
  TooltipComponent,
  type GridComponentOption,
  type LegendComponentOption,
  type TooltipComponentOption
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { use, init, type ComposeOption, type ECharts } from 'echarts/core'
import type { BarSeriesOption, LineSeriesOption } from 'echarts/charts'
import AppShell from '@/components/AppShell.vue'
import StatusTag from '@/components/StatusTag.vue'
import PageState from '@/components/PageState.vue'
import JobDetailDrawer from '@/components/JobDetailDrawer.vue'
import CallDetailDrawer from '@/components/CallDetailDrawer.vue'
import * as adminApi from '@/api/admin'
import type { CallLogRecord, DashboardOverview, TrendBucket } from '@/types'
import {
  displayJobStatus,
  displayScene,
  formatDateTime,
  formatPercent,
  toneForLatency,
  toneForStatus,
  toneForSuccessRate,
  toneForTimeoutRate
} from '@/utils/admin-display'

type DashboardChartOption = ComposeOption<
  GridComponentOption | TooltipComponentOption | LegendComponentOption | LineSeriesOption | BarSeriesOption
>

use([LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const chartRef = ref<HTMLDivElement | null>(null)
const chart = ref<ECharts | null>(null)
const overview = ref<DashboardOverview | null>(null)
const trends = ref<TrendBucket[]>([])
const trendRange = ref('24h')
const refreshing = ref(false)
const overviewLoading = ref(false)
const trendsLoading = ref(false)
const overviewError = ref('')
const trendError = ref('')

const jobDrawerVisible = ref(false)
const jobDetailLoading = ref(false)
const jobDetail = ref<{ job: DashboardOverview['recentFailures'][number]; calls: CallLogRecord[] } | null>(null)
const callDrawerVisible = ref(false)
const selectedCall = ref<CallLogRecord | null>(null)

const trendOptions = [
  { label: '24 小时', value: '24h' },
  { label: '7 天', value: '7d' },
  { label: '30 天', value: '30d' }
]

const currentTrendLabel = computed(() => {
  return trendOptions.find((item) => item.value === trendRange.value)?.label || '24 小时'
})

const metricItems = computed(() => {
  const data = overview.value
  return [
    {
      label: '任务总数',
      value: `${data?.taskTotal ?? 0}`,
      tone: (data?.taskTotal || 0) > 0 ? 'primary' : 'neutral',
      statusText: (data?.taskTotal || 0) > 0 ? '运行中' : '暂无任务',
      note: `统计窗口：最近 ${data?.windowHours ?? 24} 小时`
    },
    {
      label: '任务成功率',
      value: formatPercent(data?.taskSuccessRate),
      tone: toneForSuccessRate(data?.taskSuccessRate),
      statusText:
        toneForSuccessRate(data?.taskSuccessRate) === 'success'
          ? '稳定'
          : toneForSuccessRate(data?.taskSuccessRate) === 'warning'
            ? '关注'
            : '异常',
      note: '目标建议保持在 95% 以上'
    },
    {
      label: 'API 成功率',
      value: formatPercent(data?.apiSuccessRate),
      tone: toneForSuccessRate(data?.apiSuccessRate),
      statusText:
        toneForSuccessRate(data?.apiSuccessRate) === 'success'
          ? '稳定'
          : toneForSuccessRate(data?.apiSuccessRate) === 'warning'
            ? '关注'
            : '异常',
      note: `调用总数：${data?.apiTotal ?? 0}`
    },
    {
      label: '超时率',
      value: formatPercent(data?.timeoutRate),
      tone: toneForTimeoutRate(data?.timeoutRate),
      statusText:
        toneForTimeoutRate(data?.timeoutRate) === 'success'
          ? '正常'
          : toneForTimeoutRate(data?.timeoutRate) === 'warning'
            ? '升高'
            : '偏高',
      note: '超时率越低越利于任务闭环稳定'
    },
    {
      label: 'P95 耗时',
      value: `${data?.p95DurationMs ?? 0} ms`,
      tone: toneForLatency(data?.p95DurationMs),
      statusText:
        toneForLatency(data?.p95DurationMs) === 'success'
          ? '快速'
          : toneForLatency(data?.p95DurationMs) === 'warning'
            ? '偏慢'
            : '过高',
      note: `平均耗时：${data?.avgDurationMs ?? 0} ms`
    }
  ]
})

async function loadOverview(showToast = false) {
  overviewLoading.value = true
  overviewError.value = ''
  try {
    const data = await adminApi.getDashboardOverview()
    overview.value = data.overview
  } catch (error) {
    const message = error instanceof Error ? error.message : '加载概览失败'
    overviewError.value = message
    if (showToast) {
      ElMessage.error(message)
    }
  } finally {
    overviewLoading.value = false
  }
}

async function loadTrends(showToast = false) {
  trendsLoading.value = true
  trendError.value = ''
  try {
    const data = await adminApi.getDashboardTrends(trendRange.value)
    trends.value = data.items
    await nextTick()
    renderChart()
  } catch (error) {
    const message = error instanceof Error ? error.message : '加载趋势失败'
    trendError.value = message
    if (showToast) {
      ElMessage.error(message)
    }
  } finally {
    trendsLoading.value = false
  }
}

async function refreshPage() {
  refreshing.value = true
  await Promise.all([loadOverview(true), loadTrends(true)])
  refreshing.value = false
}

function renderChart() {
  if (!chartRef.value || !trends.value.length) {
    return
  }
  if (!chart.value) {
    chart.value = init(chartRef.value)
  }

  const option: DashboardChartOption = {
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
    grid: { left: 32, right: 24, top: 52, bottom: 28, containLabel: true },
    xAxis: {
      type: 'category',
      data: trends.value.map((item) => item.label),
      axisLine: {
        lineStyle: {
          color: 'rgba(148, 163, 184, 0.4)'
        }
      }
    },
    yAxis: [
      {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { formatter: '{value}%' },
        splitLine: {
          lineStyle: {
            color: 'rgba(148, 163, 184, 0.18)'
          }
        }
      },
      {
        type: 'value',
        min: 0,
        splitLine: { show: false }
      }
    ],
    series: [
      {
        name: '任务成功率',
        type: 'line',
        smooth: true,
        yAxisIndex: 0,
        lineStyle: { width: 3, color: '#2563eb' },
        areaStyle: { color: 'rgba(37, 99, 235, 0.1)' },
        data: trends.value.map((item) => Number(((item.taskSuccessRate || 0) * 100).toFixed(1)))
      },
      {
        name: 'API 成功率',
        type: 'line',
        smooth: true,
        yAxisIndex: 0,
        lineStyle: { width: 3, color: '#0f766e' },
        areaStyle: { color: 'rgba(15, 118, 110, 0.08)' },
        data: trends.value.map((item) => Number(((item.apiSuccessRate || 0) * 100).toFixed(1)))
      },
      {
        name: '任务总量',
        type: 'bar',
        yAxisIndex: 1,
        itemStyle: {
          color: 'rgba(148, 163, 184, 0.55)',
          borderRadius: [8, 8, 0, 0]
        },
        data: trends.value.map((item) => item.taskTotal)
      }
    ]
  }

  chart.value.setOption(option, true)
}

function handleResize() {
  chart.value?.resize()
}

async function handleTrendRangeChange() {
  await loadTrends(true)
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

function openCallDetail(call: CallLogRecord) {
  selectedCall.value = call
  callDrawerVisible.value = true
}

onMounted(async () => {
  window.addEventListener('resize', handleResize)
  await Promise.all([loadOverview(), loadTrends()])
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chart.value?.dispose()
})
</script>
