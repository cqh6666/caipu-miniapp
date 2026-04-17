<template>
  <AppShell>
    <template #toolbar>
      <StatusTag
        tone="neutral"
        :text="overview ? `更新于 ${formatDateTime(overview.generatedAt)}` : '等待首次拉取'"
      />
      <el-button :loading="loading" @click="loadOverview(true)">
        <el-icon><Refresh /></el-icon>
        <span style="margin-left: 6px">刷新数据</span>
      </el-button>
    </template>

    <el-alert
      v-if="errorMessage && overview"
      class="settings-summary"
      type="warning"
      :closable="false"
      :title="errorMessage"
    />

    <div v-if="loading && !overview" class="page-card table-card">
      <PageState mode="loading" title="正在加载服务健康概览" />
    </div>
    <div v-else-if="errorMessage && !overview" class="page-card table-card">
      <PageState
        mode="error"
        title="服务健康加载失败"
        :description="errorMessage"
        @retry="loadOverview"
      />
    </div>
    <template v-else-if="overview">
      <div class="server-health-layout">
        <div class="page-card server-health-hero-card">
          <div class="subsection-header">
            <div>
              <h3 class="subsection-title">总体健康度</h3>
              <div class="subsection-subtitle">汇总主机资源、关键服务与 HTTP 健康检查的当前状态。</div>
            </div>
            <StatusTag :tone="toneForHealthStatus(overview.summary.status)" :text="displayHealthStatus(overview.summary.status)" />
          </div>

          <div class="server-health-hero-card__body">
            <HealthRing :summary="overview.summary" size="lg" />
            <div class="server-health-hero-card__meta">
              <div class="server-health-signal-list">
                <div class="server-health-signal-list__item">
                  <span>正常信号</span>
                  <strong>{{ overview.summary.healthyCount }}</strong>
                </div>
                <div class="server-health-signal-list__item">
                  <span>关注信号</span>
                  <strong>{{ overview.summary.warningCount }}</strong>
                </div>
                <div class="server-health-signal-list__item">
                  <span>异常信号</span>
                  <strong>{{ overview.summary.criticalCount }}</strong>
                </div>
                <div class="server-health-signal-list__item">
                  <span>未知信号</span>
                  <strong>{{ overview.summary.unknownCount }}</strong>
                </div>
              </div>

              <div class="server-health-host-meta">
                <div>主机：{{ overview.host.hostname || '当前主机' }}</div>
                <div>平台：{{ overview.host.platform || '-' }}</div>
                <div>Load Avg：{{ formatLoadAverage(overview.host.load1, overview.host.load5, overview.host.load15) }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="server-health-resource-grid">
          <div v-for="item in resourceCards" :key="item.label" class="metric-card">
            <div class="metric-card__header">
              <div class="metric-label">{{ item.label }}</div>
              <StatusTag :tone="item.tone" :text="item.statusText" />
            </div>
            <div class="metric-value">{{ item.value }}</div>
            <div class="metric-note">{{ item.note }}</div>
          </div>
        </div>
      </div>

      <div class="dashboard-grid dashboard-grid--primary" style="margin-top: 20px">
        <div class="page-card table-card">
          <div class="subsection-header">
            <div>
              <h3 class="subsection-title">核心服务状态</h3>
              <div class="subsection-subtitle">优先关注 systemd 关键服务是否处于 active。</div>
            </div>
            <StatusTag tone="neutral" :text="`${serviceChecks.length} 项检查`" />
          </div>

          <PageState
            v-if="!serviceChecks.length"
            mode="empty"
            title="暂无服务状态数据"
            description="当前环境还没有可展示的 systemd 服务检查。"
            compact
          />
          <div v-else class="health-check-list">
            <article v-for="check in serviceChecks" :key="check.key" class="health-check-card">
              <div class="health-check-card__header">
                <div>
                  <h4 class="health-check-card__title">{{ check.label }}</h4>
                  <div class="health-check-card__target mono-text">{{ check.target || '-' }}</div>
                </div>
                <StatusTag :tone="toneForHealthStatus(check.status)" :text="displayHealthStatus(check.status)" />
              </div>
              <p class="health-check-card__detail">{{ check.detail || '-' }}</p>
              <div class="health-check-card__meta">
                <span>检查时间：{{ formatDateTime(check.checkedAt) }}</span>
              </div>
            </article>
          </div>
        </div>

        <div class="page-card table-card">
          <div class="subsection-header">
            <div>
              <h3 class="subsection-title">HTTP 健康探测</h3>
              <div class="subsection-subtitle">确认后端与 sidecar 的内网探测地址是否可以正常响应。</div>
            </div>
            <StatusTag tone="neutral" :text="`${httpChecks.length} 项检查`" />
          </div>

          <PageState
            v-if="!httpChecks.length"
            mode="empty"
            title="暂无 HTTP 探测数据"
            description="当前环境还没有可展示的 HTTP 健康探测。"
            compact
          />
          <div v-else class="health-check-list">
            <article v-for="check in httpChecks" :key="check.key" class="health-check-card">
              <div class="health-check-card__header">
                <div>
                  <h4 class="health-check-card__title">{{ check.label }}</h4>
                  <div class="health-check-card__target mono-text">{{ check.target || '-' }}</div>
                </div>
                <StatusTag :tone="toneForHealthStatus(check.status)" :text="displayHealthStatus(check.status)" />
              </div>
              <p class="health-check-card__detail">{{ check.detail || '-' }}</p>
              <div class="health-check-card__meta">
                <span>耗时：{{ formatDuration(check.latencyMs ?? undefined) }}</span>
                <span>检查时间：{{ formatDateTime(check.checkedAt) }}</span>
              </div>
            </article>
          </div>
        </div>
      </div>
    </template>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import AppShell from '@/components/AppShell.vue'
import HealthRing from '@/components/HealthRing.vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import * as adminApi from '@/api/admin'
import type { ServerHealthOverview } from '@/types'
import {
  displayHealthStatus,
  formatDateTime,
  formatDuration,
  formatLoadAverage,
  formatUptime,
  formatUsagePercent,
  statusForResourceUsage,
  toneForHealthStatus
} from '@/utils/admin-display'

const overview = ref<ServerHealthOverview | null>(null)
const loading = ref(false)
const errorMessage = ref('')

const serviceChecks = computed(() => overview.value?.checks.filter((item) => item.category === 'systemd') || [])
const httpChecks = computed(() => overview.value?.checks.filter((item) => item.category === 'http') || [])

const resourceCards = computed(() => {
  const host = overview.value?.host
  return [
    {
      label: 'CPU 占用',
      value: formatUsagePercent(host?.cpuUsagePercent),
      note: '阈值 75% / 90%',
      statusText: displayHealthStatus(statusForResourceUsage(host?.cpuUsagePercent, 'cpu')),
      tone: toneForHealthStatus(statusForResourceUsage(host?.cpuUsagePercent, 'cpu'))
    },
    {
      label: '内存占用',
      value: formatUsagePercent(host?.memoryUsagePercent),
      note: '阈值 75% / 90%',
      statusText: displayHealthStatus(statusForResourceUsage(host?.memoryUsagePercent, 'memory')),
      tone: toneForHealthStatus(statusForResourceUsage(host?.memoryUsagePercent, 'memory'))
    },
    {
      label: '磁盘占用',
      value: formatUsagePercent(host?.diskUsagePercent),
      note: '阈值 80% / 90%',
      statusText: displayHealthStatus(statusForResourceUsage(host?.diskUsagePercent, 'disk')),
      tone: toneForHealthStatus(statusForResourceUsage(host?.diskUsagePercent, 'disk'))
    },
    {
      label: '运行时长',
      value: formatUptime(host?.uptimeSeconds),
      note: `Load Avg：${formatLoadAverage(host?.load1, host?.load5, host?.load15)}`,
      statusText: host?.platform || '未知平台',
      tone: 'primary' as const
    }
  ]
})

async function loadOverview(showToast = false) {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await adminApi.getServerHealthOverview()
    overview.value = data.overview
  } catch (error) {
    const message = error instanceof Error ? error.message : '加载服务健康失败'
    errorMessage.value = message
    if (showToast) {
      ElMessage.error(message)
    }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadOverview()
})
</script>
