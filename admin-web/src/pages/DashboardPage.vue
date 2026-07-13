<template>
  <AppShell>
    <template #toolbar>
      <el-radio-group v-model="overviewWindow" size="small" @change="handleWindowChange">
        <el-radio-button :value="24">24h</el-radio-button>
        <el-radio-button :value="168">7d</el-radio-button>
        <el-radio-button :value="720">30d</el-radio-button>
      </el-radio-group>
      <span v-if="lastRefreshed" class="topbar-refreshed">更新于 {{ lastRefreshed }}</span>
      <el-button :loading="refreshing" @click="refreshPage">
        <el-icon><Refresh /></el-icon>
        <span style="margin-left: 6px">刷新数据</span>
      </el-button>
    </template>

    <el-alert
      v-if="overviewError && overview"
      class="settings-summary"
      type="warning"
      :closable="false"
      :title="overviewError"
    />

    <div v-if="overviewLoading && !overview" class="metric-grid">
      <div v-for="item in 5" :key="item" class="metric-card metric-card--neutral">
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
      <div
        v-for="metric in metricItems"
        :key="metric.label"
        class="metric-card"
        :class="`metric-card--${metric.tone}`"
      >
        <div class="metric-card__header">
          <div class="metric-label">{{ metric.label }}</div>
          <StatusTag :tone="metric.tone" :text="metric.statusText" />
        </div>
        <div class="metric-value" :class="{ 'metric-value--pending': metric.pending }">
          {{ metric.value }}
        </div>
        <div class="metric-note">{{ metric.note }}</div>
      </div>
    </div>

    <div class="dashboard-health-block">
      <div v-if="serverHealthLoading && !serverHealth" class="page-card server-health-summary-card">
        <el-skeleton animated :rows="4" />
      </div>
      <div v-else-if="serverHealthError && !serverHealth" class="page-card server-health-summary-card">
        <PageState
          mode="error"
          title="服务健康摘要加载失败"
          :description="serverHealthError"
          compact
          @retry="loadServerHealth"
        />
      </div>
      <button
        v-else-if="serverHealth"
        type="button"
        class="page-card server-health-summary-card server-health-summary-card--interactive"
        @click="openServerHealthPage"
      >
        <div class="subsection-header">
          <div>
            <h3 class="subsection-title">服务健康摘要</h3>
            <div class="subsection-subtitle">主机资源与核心链路状态的一页式快照。</div>
          </div>
          <StatusTag
            :tone="toneForHealthStatus(serverHealth.summary.status)"
            :text="displayHealthStatus(serverHealth.summary.status)"
          />
        </div>

        <div class="server-health-summary-card__body">
          <HealthRing :summary="serverHealth.summary" />
          <div class="server-health-summary-card__stats">
            <div
              v-for="stat in healthStats"
              :key="stat.label"
              class="server-health-summary-stat"
              :class="{ 'server-health-summary-stat--pending': stat.pending }"
            >
              <span>{{ stat.label }}</span>
              <strong>{{ stat.value }}</strong>
              <span v-if="stat.pending" class="server-health-summary-stat__hint">采集中</span>
            </div>
          </div>
        </div>

        <div v-if="serverHealthError" class="server-health-summary-card__warning">
          {{ serverHealthError }}
        </div>
        <div class="server-health-summary-card__footer">
          <span>查看服务健康详情</span>
          <el-icon><ArrowRight /></el-icon>
        </div>
      </button>
    </div>

    <div class="dashboard-grid dashboard-grid--primary">
      <div class="page-card chart-card">
        <div class="subsection-header">
          <div>
            <h3 class="subsection-title">趋势</h3>
            <div class="subsection-subtitle">按时间窗口观察任务成功率、API 成功率与任务总量（跟随顶部窗口切换）。</div>
          </div>
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
          <div class="subsection-actions">
            <StatusTag
              :tone="overview?.recentFailures?.length ? 'warning' : 'success'"
              :text="overview?.recentFailures?.length ? `${overview.recentFailures.length} 条记录` : '暂无异常'"
            />
            <el-button link type="primary" @click="openJobsPage()">
              查看 AI 任务
            </el-button>
          </div>
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
            <el-table-column label="场景" min-width="220">
              <template #default="{ row }">
                <div class="recent-failure-entry">
                  <div class="recent-failure-entry__row">
                    <span>{{ displayScene(row.scene) }}</span>
                    <el-button
                      link
                      type="primary"
                      size="small"
                      @click="openJobDetailPage(row)"
                    >
                      排障详情
                    </el-button>
                  </div>
                  <div class="recent-failure-entry__meta">
                    开始于 {{ formatDateTime(row.startedAt) }}
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <StatusTag :tone="toneForStatus(row.status)" :text="displayJobStatus(row.status)" />
              </template>
            </el-table-column>
            <el-table-column label="错误摘要" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.errorMessage || '-' }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>

    <div class="dashboard-grid dashboard-grid--triple" style="margin-top: 20px">
      <DistributionPanel
        ref="sceneDistributionPanelRef"
        v-model="distViewMode.scene"
        title="按场景分布"
        subtitle="任务热点与成功率排行。"
        count-unit="个场景"
        empty-title="暂无场景分布"
        empty-description="当前时间窗口内还没有场景级任务数据。"
        name-column="场景"
        total-column="任务量"
        :chart-height="distChartHeight(sceneRankItems.length)"
        :items="sceneRankItems"
      />
      <DistributionPanel
        ref="providerDistributionPanelRef"
        v-model="distViewMode.provider"
        title="Provider 热点"
        subtitle="调用热点与成功率排行。"
        count-unit="个节点"
        empty-title="暂无 Provider 数据"
        empty-description="当前时间窗口内还没有调用侧热点分布。"
        name-column="Provider"
        total-column="调用量"
        :chart-height="distChartHeight(providerRankItems.length)"
        :items="providerRankItems"
      />
      <DistributionPanel
        ref="modelDistributionPanelRef"
        v-model="distViewMode.model"
        title="Model 热点"
        subtitle="模型热点与成功率排行。"
        count-unit="个模型"
        empty-title="暂无 Model 数据"
        empty-description="当前时间窗口内还没有模型侧热点分布。"
        name-column="Model"
        total-column="调用量"
        :chart-height="distChartHeight(modelRankItems.length)"
        :items="modelRankItems"
      />
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
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowRight, Refresh } from '@element-plus/icons-vue'
import { BarChart, LineChart } from 'echarts/charts'
import {
  AxisPointerComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
  type GridComponentOption,
  type LegendComponentOption,
  type TooltipComponentOption
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { use, type ComposeOption } from 'echarts/core'
import type { BarSeriesOption, LineSeriesOption } from 'echarts/charts'
import AppShell from '@/components/AppShell.vue'
import { useLastRefreshed } from '@/composables/useLastRefreshed'
import { useDashboardCharts, type DashboardChartKey } from '@/composables/useDashboardCharts'
import HealthRing from '@/components/HealthRing.vue'
import StatusTag from '@/components/StatusTag.vue'
import PageState from '@/components/PageState.vue'
import JobDetailDrawer from '@/components/JobDetailDrawer.vue'
import CallDetailDrawer from '@/components/CallDetailDrawer.vue'
import DistributionPanel from '@/components/dashboard/DistributionPanel.vue'
import * as adminApi from '@/api/admin'
import type { CallLogRecord, DashboardOverview, ServerHealthOverview, TrendBucket } from '@/types'
import {
  displayHealthStatus,
  displayJobStatus,
  displayScene,
  formatDateTime,
  formatPercent,
  formatUsagePercent,
  toneForLatency,
  toneForHealthStatus,
  toneForStatus,
  toneForSuccessRate,
  toneForTimeoutRate,
  type StatusTone
} from '@/utils/admin-display'
import { buildRouteQuery } from '@/utils/route-query'
import {
  buildDistributionRankItems,
  middleEllipsis,
  normalizeDistributionItems,
  type DistributionItem,
  type DistViewMode
} from '@/utils/dashboard-distribution'

type DashboardChartOption = ComposeOption<
  GridComponentOption | TooltipComponentOption | LegendComponentOption | LineSeriesOption | BarSeriesOption
>

use([LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, AxisPointerComponent, CanvasRenderer])

const { display: lastRefreshed, mark: markRefreshed } = useLastRefreshed('dashboard')

const overviewWindow = ref<number>(168)

const {
  chartRef,
  dispose: disposeDashboardChart,
  render: renderDashboardChart
} = useDashboardCharts()
const distViewMode = reactive<{ scene: DistViewMode; provider: DistViewMode; model: DistViewMode }>({
  scene: 'rank',
  provider: 'rank',
  model: 'rank'
})
type DistributionPanelInstance = { chartElement: HTMLDivElement | null }
const sceneDistributionPanelRef = ref<DistributionPanelInstance | null>(null)
const providerDistributionPanelRef = ref<DistributionPanelInstance | null>(null)
const modelDistributionPanelRef = ref<DistributionPanelInstance | null>(null)
const router = useRouter()
const overview = ref<DashboardOverview | null>(null)
const serverHealth = ref<ServerHealthOverview | null>(null)
const trends = ref<TrendBucket[]>([])
const refreshing = ref(false)
const overviewLoading = ref(false)
const serverHealthLoading = ref(false)
const trendsLoading = ref(false)
const overviewError = ref('')
const serverHealthError = ref('')
const trendError = ref('')

const jobDrawerVisible = ref(false)
const jobDetailLoading = ref(false)
const jobDetail = ref<{ job: DashboardOverview['recentFailures'][number]; calls: CallLogRecord[] } | null>(null)
const callDrawerVisible = ref(false)
const selectedCall = ref<CallLogRecord | null>(null)

const trendRange = computed(() => {
  if (overviewWindow.value >= 720) return '30d'
  if (overviewWindow.value >= 168) return '7d'
  return '24h'
})

function hasNumericValue(value: number | null | undefined): boolean {
  return value !== null && value !== undefined && !Number.isNaN(value)
}

interface MetricItem {
  label: string
  value: string
  tone: StatusTone
  statusText: string
  note: string
  pending: boolean
}

const sceneRankItems = computed(() => buildDistributionRankItems(overview.value?.byScene, {
  labelFormatter: (value) => displayScene(value),
  showAlert: false
}))
const providerRankItems = computed(() => buildDistributionRankItems(overview.value?.byProvider))
const modelRankItems = computed(() => buildDistributionRankItems(overview.value?.byModel))
const providerChartItems = computed(() => normalizeDistributionItems(overview.value?.byProvider))
const modelChartItems = computed(() => normalizeDistributionItems(overview.value?.byModel))

const metricItems = computed<MetricItem[]>(() => {
  const data = overview.value
  const taskTotal = data?.taskTotal ?? 0
  const apiTotal = data?.apiTotal ?? 0
  const pendingTasks = taskTotal === 0
  const pendingApi = apiTotal === 0

  const taskSuccessTone = toneForSuccessRate(data?.taskSuccessRate)
  const apiSuccessTone = toneForSuccessRate(data?.apiSuccessRate)
  const timeoutTone = toneForTimeoutRate(data?.timeoutRate)
  const latencyTone = toneForLatency(data?.p95DurationMs)

  return [
    {
      label: '任务总数',
      value: `${taskTotal}`,
      tone: taskTotal > 0 ? 'primary' : 'neutral',
      statusText: taskTotal > 0 ? '运行中' : '暂无任务',
      note: `统计窗口：最近 ${data?.windowHours ?? 24} 小时`,
      pending: pendingTasks
    },
    {
      label: '任务成功率',
      value: pendingTasks ? '—' : formatPercent(data?.taskSuccessRate),
      tone: pendingTasks ? 'neutral' : taskSuccessTone,
      statusText: pendingTasks
        ? '待数据'
        : taskSuccessTone === 'success'
          ? '稳定'
          : taskSuccessTone === 'warning'
            ? '关注'
            : '异常',
      note: pendingTasks ? '首次任务完成后展示真实成功率' : '目标建议保持在 95% 以上',
      pending: pendingTasks
    },
    {
      label: 'API 成功率',
      value: pendingApi ? '—' : formatPercent(data?.apiSuccessRate),
      tone: pendingApi ? 'neutral' : apiSuccessTone,
      statusText: pendingApi
        ? '待数据'
        : apiSuccessTone === 'success'
          ? '稳定'
          : apiSuccessTone === 'warning'
            ? '关注'
            : '异常',
      note: pendingApi ? '累计调用后展示 API 侧成功率' : `调用总数：${apiTotal}`,
      pending: pendingApi
    },
    {
      label: '超时率',
      value: pendingApi ? '—' : formatPercent(data?.timeoutRate),
      tone: pendingApi ? 'neutral' : timeoutTone,
      statusText: pendingApi
        ? '待数据'
        : timeoutTone === 'success'
          ? '正常'
          : timeoutTone === 'warning'
            ? '升高'
            : '偏高',
      note: pendingApi ? '依赖 API 调用样本统计' : '超时率越低越利于任务闭环稳定',
      pending: pendingApi
    },
    {
      label: 'P95 耗时',
      value: pendingApi ? '—' : `${data?.p95DurationMs ?? 0} ms`,
      tone: pendingApi ? 'neutral' : latencyTone,
      statusText: pendingApi
        ? '待数据'
        : latencyTone === 'success'
          ? '快速'
          : latencyTone === 'warning'
            ? '偏慢'
            : '过高',
      note: pendingApi ? '暂无耗时样本' : `平均耗时：${data?.avgDurationMs ?? 0} ms`,
      pending: pendingApi
    }
  ]
})

const healthStats = computed(() => {
  const host = serverHealth.value?.host
  const summary = serverHealth.value?.summary
  const warningTotal = summary ? summary.warningCount + summary.criticalCount : 0
  const hasCpu = hasNumericValue(host?.cpuUsagePercent)
  const hasMemory = hasNumericValue(host?.memoryUsagePercent)
  const hasDisk = hasNumericValue(host?.diskUsagePercent)

  return [
    {
      label: 'CPU 占用',
      pending: !hasCpu,
      value: hasCpu ? formatUsagePercent(host?.cpuUsagePercent) : '—'
    },
    {
      label: '内存占用',
      pending: !hasMemory,
      value: hasMemory ? formatUsagePercent(host?.memoryUsagePercent) : '—'
    },
    {
      label: '磁盘占用',
      pending: !hasDisk,
      value: hasDisk ? formatUsagePercent(host?.diskUsagePercent) : '—'
    },
    {
      label: '异常信号',
      pending: false,
      value: `${warningTotal}`
    }
  ]
})

async function loadOverview(showToast = false) {
  overviewLoading.value = true
  overviewError.value = ''
  try {
    const data = await adminApi.getDashboardOverview(overviewWindow.value)
    overview.value = data.overview
    markRefreshed()
    void renderDistributionCharts()
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

async function loadServerHealth(showToast = false) {
  serverHealthLoading.value = true
  serverHealthError.value = ''
  try {
    const data = await adminApi.getServerHealthOverview()
    serverHealth.value = data.overview
  } catch (error) {
    const message = error instanceof Error ? error.message : '加载服务健康摘要失败'
    serverHealthError.value = message
    if (showToast) {
      ElMessage.error(message)
    }
  } finally {
    serverHealthLoading.value = false
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
  await Promise.all([loadOverview(true), loadServerHealth(true), loadTrends(true)])
  refreshing.value = false
}

function renderChart() {
  if (!chartRef.value || !trends.value.length) {
    return
  }

  const option: DashboardChartOption = {
    tooltip: { trigger: 'axis' },
    legend: { top: 0, textStyle: { color: '#334155' } },
    grid: { left: 32, right: 24, top: 52, bottom: 28, containLabel: true },
    xAxis: {
      type: 'category',
      data: trends.value.map((item) => item.label),
      axisLine: { lineStyle: { color: 'rgba(148, 163, 184, 0.4)' } },
      axisLabel: { color: '#64748b' }
    },
    yAxis: [
      {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { formatter: '{value}%', color: '#64748b' },
        splitLine: { lineStyle: { color: 'rgba(148, 163, 184, 0.18)' } }
      },
      {
        type: 'value',
        min: 0,
        axisLabel: { color: '#64748b' },
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
        itemStyle: { color: '#2563eb' },
        areaStyle: { color: 'rgba(37, 99, 235, 0.1)' },
        data: trends.value.map((item) => Number(((item.taskSuccessRate || 0) * 100).toFixed(1)))
      },
      {
        name: 'API 成功率',
        type: 'line',
        smooth: true,
        yAxisIndex: 0,
        lineStyle: { width: 3, color: '#0f766e' },
        itemStyle: { color: '#0f766e' },
        areaStyle: { color: 'rgba(15, 118, 110, 0.08)' },
        data: trends.value.map((item) => Number(((item.apiSuccessRate || 0) * 100).toFixed(1)))
      },
      {
        name: '任务总量',
        type: 'bar',
        yAxisIndex: 1,
        itemStyle: {
          color: 'rgba(148, 163, 184, 0.55)',
          borderRadius: [6, 6, 0, 0]
        },
        data: trends.value.map((item) => item.taskTotal)
      }
    ]
  }

  renderDashboardChart('trend', option, chartRef.value)
}

function distChartHeight(count: number): string {
  const rows = Math.max(count, 1)
  return `${Math.max(rows * 36 + 48, 160)}px`
}

function renderDistributionChart(
  key: DashboardChartKey,
  container: HTMLDivElement | null,
  items: DistributionItem[] | undefined | null,
  nameFormatter?: (name: string) => string
) {
  if (!container || !items || !items.length) return
  const sorted = [...items].sort((a, b) => {
    const aUnknown = isUnknownDistributionName(a.name) || a.name === '未指定'
    const bUnknown = isUnknownDistributionName(b.name) || b.name === '未指定'
    if (aUnknown !== bUnknown) {
      return aUnknown ? 1 : -1
    }
    if ((a.total ?? 0) !== (b.total ?? 0)) {
      return (a.total ?? 0) - (b.total ?? 0)
    }
    return String(a.name || '').localeCompare(String(b.name || ''), 'zh-CN')
  })
  const labels = sorted.map((item) => (nameFormatter ? nameFormatter(item.name) : item.name))
  const successData = sorted.map((item) => {
    const total = item.total ?? 0
    const rate = item.successRate ?? 0
    return Math.round(total * rate)
  })
  const failData = sorted.map((item, idx) => Math.max((item.total ?? 0) - successData[idx], 0))

  const option: DashboardChartOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const list = params as Array<{ dataIndex: number; name: string }>
        if (!list?.length) return ''
        const idx = list[0].dataIndex
        const item = sorted[idx]
        const total = item.total ?? 0
        const rate = item.successRate ?? 0
        const rateTone = rate >= 0.9 ? '#16a34a' : rate >= 0.7 ? '#d97706' : '#dc2626'
        return `<div style="font-weight:600;margin-bottom:6px">${list[0].name}</div>`
          + `<div style="display:flex;justify-content:space-between;gap:16px"><span style="color:#64748b">总数</span><strong>${total}</strong></div>`
          + `<div style="display:flex;justify-content:space-between;gap:16px"><span style="color:#64748b">成功率</span><strong style="color:${rateTone}">${(rate * 100).toFixed(1)}%</strong></div>`
          + `<div style="display:flex;justify-content:space-between;gap:16px"><span style="color:#64748b">成功 / 失败</span><strong>${successData[idx]} / ${failData[idx]}</strong></div>`
      }
    },
    grid: { left: 112, right: 20, top: 12, bottom: 8, containLabel: false },
    xAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: '#94a3b8' },
      splitLine: { lineStyle: { color: 'rgba(148, 163, 184, 0.18)' } }
    },
    yAxis: {
      type: 'category',
      data: labels,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: '#334155',
        fontWeight: 500,
        width: 108,
        overflow: 'truncate',
        formatter: (value: string) => middleEllipsis(value, 10, 8)
      }
    },
    series: [
      {
        name: '成功',
        type: 'bar',
        stack: 'total',
        barMaxWidth: 18,
        itemStyle: { color: '#22c55e', borderRadius: [4, 0, 0, 4] },
        data: successData
      },
      {
        name: '失败',
        type: 'bar',
        stack: 'total',
        barMaxWidth: 18,
        itemStyle: { color: '#f87171', borderRadius: [0, 4, 4, 0] },
        data: failData
      }
    ]
  }

  renderDashboardChart(key, option, container)
}

async function renderDistributionCharts() {
  await nextTick()
  const pairs: Array<[DistViewMode, DashboardChartKey, HTMLDivElement | null, DistributionItem[] | undefined, ((n: string) => string) | undefined]> = [
    [distViewMode.scene, 'scene', sceneDistributionPanelRef.value?.chartElement ?? null, overview.value?.byScene, displayScene],
    [distViewMode.provider, 'provider', providerDistributionPanelRef.value?.chartElement ?? null, providerChartItems.value, undefined],
    [distViewMode.model, 'model', modelDistributionPanelRef.value?.chartElement ?? null, modelChartItems.value, undefined]
  ]
  for (const [mode, key, container, items, fmt] of pairs) {
    try {
      if (mode === 'chart' && items?.length && container) {
        renderDistributionChart(key, container, items, fmt)
      } else {
        disposeDashboardChart(key)
      }
    } catch (err) {
      console.warn(`[dashboard] render distribution chart "${key}" failed`, err)
    }
  }
}

watch(
  () => [overview.value, distViewMode.scene, distViewMode.provider, distViewMode.model],
  () => {
    void renderDistributionCharts()
  },
  { deep: true, flush: 'post' }
)

async function handleWindowChange() {
  await Promise.all([loadOverview(true), loadTrends(true)])
}

function buildOverviewTimeRange() {
  const timeTo = new Date()
  const timeFrom = new Date(timeTo.getTime() - overviewWindow.value * 60 * 60 * 1000)
  return {
    timeFrom: timeFrom.toISOString(),
    timeTo: timeTo.toISOString()
  }
}

function buildJobsPageQuery(job?: DashboardOverview['recentFailures'][number]) {
  const range = buildOverviewTimeRange()
  return buildRouteQuery({
    ...range,
    scene: job?.scene || undefined,
    status: job?.status || undefined,
    jobId: job?.id || undefined
  })
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

function openJobDetailPage(job: DashboardOverview['recentFailures'][number]) {
  void router.push({
    path: '/ai-jobs',
    query: buildJobsPageQuery(job)
  })
}

function openJobsPage() {
  void router.push({
    path: '/ai-jobs',
    query: buildJobsPageQuery()
  })
}

function openServerHealthPage() {
  void router.push('/server-health')
}

onMounted(async () => {
  await Promise.all([loadOverview(), loadServerHealth(), loadTrends()])
})
</script>

<style scoped>
.subsection-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.recent-failure-entry {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

.recent-failure-entry__row {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.recent-failure-entry__row > span:first-child {
  min-width: 0;
  font-weight: 600;
  color: var(--color-text);
}

.recent-failure-entry__meta {
  font-size: 12px;
  color: var(--color-text-subtle);
}

</style>
