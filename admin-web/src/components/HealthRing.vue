<template>
  <div class="health-ring" :class="`health-ring--${size}`">
    <div class="health-ring__dial" :style="{ background: gradient }">
      <div class="health-ring__center">
        <div class="health-ring__status">{{ statusLabel }}</div>
        <div class="health-ring__meta">{{ totalSignals }} 项信号</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ServerHealthSummary, ServerHealthStatus } from '@/types'
import { displayHealthStatus } from '@/utils/admin-display'

const props = withDefaults(
  defineProps<{
    summary: ServerHealthSummary
    status?: ServerHealthStatus
    size?: 'sm' | 'lg'
  }>(),
  {
    status: undefined,
    size: 'sm'
  }
)

const totalSignals = computed(() => {
  return (
    props.summary.healthyCount +
    props.summary.warningCount +
    props.summary.criticalCount +
    props.summary.unknownCount
  )
})

const statusLabel = computed(() => displayHealthStatus(props.status || props.summary.status))

const gradient = computed(() => {
  const total = totalSignals.value
  if (!total) {
    return 'conic-gradient(#cbd5e1 0deg 360deg)'
  }

  const segments = [
    { value: props.summary.healthyCount, color: '#0f766e' },
    { value: props.summary.warningCount, color: '#d97706' },
    { value: props.summary.criticalCount, color: '#dc2626' },
    { value: props.summary.unknownCount, color: '#64748b' }
  ]

  let degree = 0
  const stops: string[] = []
  for (const segment of segments) {
    if (!segment.value) {
      continue
    }
    const nextDegree = degree + (segment.value / total) * 360
    stops.push(`${segment.color} ${degree}deg ${nextDegree}deg`)
    degree = nextDegree
  }

  if (!stops.length) {
    return 'conic-gradient(#cbd5e1 0deg 360deg)'
  }

  if (degree < 360) {
    stops.push(`#cbd5e1 ${degree}deg 360deg`)
  }

  return `conic-gradient(${stops.join(', ')})`
})
</script>

<style scoped>
.health-ring {
  display: inline-flex;
}

.health-ring__dial {
  position: relative;
  display: grid;
  place-items: center;
  border-radius: 50%;
}

.health-ring__dial::after {
  content: '';
  position: absolute;
  inset: 12%;
  border-radius: 50%;
  background:
    radial-gradient(circle at top, rgba(255, 255, 255, 0.98), rgba(244, 247, 251, 0.94));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.75);
}

.health-ring__center {
  position: relative;
  z-index: 1;
  display: grid;
  gap: 6px;
  place-items: center;
  text-align: center;
}

.health-ring__status {
  color: var(--color-text);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.health-ring__meta {
  color: var(--color-text-subtle);
  font-size: 12px;
  line-height: 1.4;
}

.health-ring--sm .health-ring__dial {
  width: 132px;
  height: 132px;
}

.health-ring--sm .health-ring__status {
  font-size: 20px;
}

.health-ring--lg .health-ring__dial {
  width: 184px;
  height: 184px;
}

.health-ring--lg .health-ring__status {
  font-size: 28px;
}
</style>
