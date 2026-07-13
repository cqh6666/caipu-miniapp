<template>
  <div class="page-card table-card distribution-card">
    <div class="subsection-header">
      <div class="distribution-card__header-main">
        <h3 class="subsection-title">{{ title }}</h3>
        <div class="subsection-subtitle">{{ subtitle }}</div>
      </div>
      <div class="distribution-header-actions">
        <el-radio-group v-model="mode" size="small" class="distribution-view-switch">
          <el-radio-button value="rank">排行</el-radio-button>
          <el-radio-button value="chart">图表</el-radio-button>
        </el-radio-group>
        <StatusTag tone="neutral" :text="`${items.length} ${countUnit}`" />
      </div>
    </div>

    <PageState
      v-if="!items.length"
      mode="empty"
      :title="emptyTitle"
      :description="emptyDescription"
      compact
    />
    <div
      v-else-if="mode === 'chart'"
      ref="chartElement"
      class="distribution-chart"
      :style="{ height: chartHeight }"
    ></div>
    <div v-else class="distribution-rank-list">
      <div class="distribution-rank-header">
        <span class="distribution-rank-header__index">#</span>
        <span>{{ nameColumn }}</span>
        <span>{{ totalColumn }}</span>
        <span>成功率</span>
      </div>
      <div v-for="item in items" :key="item.key" class="distribution-rank-row">
        <span class="distribution-rank-index">{{ item.rankLabel }}</span>
        <div class="distribution-rank-name" :title="item.label">
          <span class="distribution-rank-name__text">{{ item.shortLabel }}</span>
        </div>
        <div class="distribution-rank-metric distribution-rank-metric--total">
          <strong class="distribution-rank-metric__value">{{ item.totalText }}</strong>
          <div class="distribution-rank-meter distribution-rank-meter--total">
            <div class="distribution-rank-meter__fill distribution-rank-meter__fill--total" :style="{ width: `${item.totalPercent}%` }"></div>
          </div>
        </div>
        <div class="distribution-rank-metric distribution-rank-metric--rate">
          <div class="distribution-rank-meter distribution-rank-meter--rate">
            <div
              class="distribution-rank-meter__fill"
              :class="`distribution-rank-meter__fill--${item.tone}`"
              :style="{ width: `${item.ratePercent}%` }"
            ></div>
          </div>
          <div class="distribution-rank-rate-meta">
            <strong class="distribution-rank-metric__value">{{ item.rateText }}</strong>
            <el-icon
              v-if="item.showAlert"
              class="distribution-rank-alert"
              :class="`distribution-rank-alert--${item.tone}`"
              :title="item.alertText"
            >
              <Warning />
            </el-icon>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Warning } from '@element-plus/icons-vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import type { DistributionRankItem, DistViewMode } from '@/utils/dashboard-distribution'

defineProps<{
  title: string
  subtitle: string
  countUnit: string
  emptyTitle: string
  emptyDescription: string
  nameColumn: string
  totalColumn: string
  chartHeight: string
  items: DistributionRankItem[]
}>()

const mode = defineModel<DistViewMode>({ required: true })
const chartElement = ref<HTMLDivElement | null>(null)

defineExpose({ chartElement })
</script>

<style scoped>
.distribution-header-actions { display: flex; flex-shrink: 0; flex-direction: column; align-items: flex-end; gap: 8px; }
.distribution-card__header-main { min-width: 0; }
.distribution-view-switch { display: inline-flex; flex-wrap: nowrap; }
.distribution-view-switch :deep(.el-radio-button__inner) { min-width: 52px; padding-inline: 12px; }
.distribution-card { display: flex; min-width: 0; min-height: 100%; flex-direction: column; overflow: hidden; }
.distribution-chart { width: 100%; max-width: 100%; min-width: 0; min-height: 160px; overflow: hidden; }
.distribution-rank-list { display: flex; flex-direction: column; }
.distribution-rank-header, .distribution-rank-row { display: grid; grid-template-columns: 28px minmax(0, 1.25fr) minmax(92px, 0.88fr) minmax(118px, 0.95fr); align-items: center; gap: 12px; }
.distribution-rank-header { padding: 4px 0 8px; border-bottom: 1px solid rgba(148, 163, 184, 0.14); color: var(--color-text-subtle); font-size: 12px; }
.distribution-rank-header__index, .distribution-rank-index { text-align: center; }
.distribution-rank-row { padding: 12px 0; border-top: 1px solid rgba(148, 163, 184, 0.14); }
.distribution-rank-row:first-of-type { border-top: none; }
.distribution-rank-index { color: var(--color-text-subtle); font-weight: 700; font-variant-numeric: tabular-nums; }
.distribution-rank-name { display: flex; min-width: 0; align-items: center; gap: 6px; }
.distribution-rank-name__text { min-width: 0; overflow: hidden; color: var(--color-text); font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.distribution-rank-metric { display: flex; min-width: 0; flex-direction: column; gap: 6px; }
.distribution-rank-metric__value { color: var(--color-text); font-size: 12px; font-variant-numeric: tabular-nums; }
.distribution-rank-meter { position: relative; width: 100%; height: 6px; border-radius: 999px; background: #eef2f7; overflow: hidden; }
.distribution-rank-meter__fill { height: 100%; border-radius: 999px; transition: width 0.22s ease; }
.distribution-rank-meter__fill--total { background: linear-gradient(90deg, #93c5fd, #3b82f6); }
.distribution-rank-meter__fill--success { background: linear-gradient(90deg, #22c55e, #16a34a); }
.distribution-rank-meter__fill--warning { background: linear-gradient(90deg, #f59e0b, #d97706); }
.distribution-rank-meter__fill--danger { background: linear-gradient(90deg, #f87171, #dc2626); }
.distribution-rank-meter__fill--neutral, .distribution-rank-meter__fill--primary { background: #cbd5e1; }
.distribution-rank-rate-meta { display: flex; align-items: center; gap: 6px; }
.distribution-rank-alert { font-size: 14px; }
.distribution-rank-alert--warning { color: #d97706; }
.distribution-rank-alert--danger { color: #dc2626; }

@media (max-width: 1280px) {
  .distribution-rank-header, .distribution-rank-row { grid-template-columns: 24px minmax(0, 1.1fr) minmax(78px, 0.8fr) minmax(96px, 0.9fr); gap: 10px; }
}
@media (max-width: 1200px) {
  .distribution-header-actions { align-items: flex-start; }
}
</style>
