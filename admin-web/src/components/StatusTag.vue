<template>
  <span class="status-tag" :class="`status-tag--${tone}`">
    <el-icon v-if="showIcon" class="status-tag__icon" aria-hidden="true">
      <component :is="resolvedIcon" />
    </el-icon>
    <span v-else class="status-tag__dot" aria-hidden="true"></span>
    <span>{{ text }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CircleCheck, CircleClose, InfoFilled, Loading, Warning } from '@element-plus/icons-vue'
import type { StatusTone } from '@/utils/admin-display'

const props = withDefaults(
  defineProps<{
    text: string
    tone?: StatusTone
    icon?: boolean
  }>(),
  {
    tone: 'neutral',
    icon: true
  }
)

const showIcon = computed(() => props.icon && props.tone !== 'neutral')

const resolvedIcon = computed(() => {
  switch (props.tone) {
    case 'success':
      return CircleCheck
    case 'warning':
      return Warning
    case 'danger':
      return CircleClose
    case 'primary':
      return InfoFilled
    default:
      return Loading
  }
})
</script>

<style scoped>
.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid var(--status-tag-border, rgba(148, 163, 184, 0.22));
  background: var(--status-tag-bg, #f1f5f9);
  color: var(--status-tag-color, #334155);
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
}

.status-tag__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.8;
}

.status-tag__icon {
  font-size: 13px;
  line-height: 1;
}

.status-tag--neutral {
  --status-tag-bg: #f1f5f9;
  --status-tag-border: #e2e8f0;
  --status-tag-color: #475569;
}

.status-tag--primary {
  --status-tag-bg: rgba(37, 99, 235, 0.10);
  --status-tag-border: rgba(37, 99, 235, 0.22);
  --status-tag-color: #1d4ed8;
}

.status-tag--success {
  --status-tag-bg: rgba(22, 163, 74, 0.12);
  --status-tag-border: rgba(22, 163, 74, 0.24);
  --status-tag-color: #15803d;
}

.status-tag--warning {
  --status-tag-bg: rgba(217, 119, 6, 0.12);
  --status-tag-border: rgba(217, 119, 6, 0.26);
  --status-tag-color: #b45309;
}

.status-tag--danger {
  --status-tag-bg: rgba(220, 38, 38, 0.12);
  --status-tag-border: rgba(220, 38, 38, 0.26);
  --status-tag-color: #b91c1c;
}
</style>
