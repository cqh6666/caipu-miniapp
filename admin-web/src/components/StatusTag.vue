<template>
  <span class="status-tag" :class="`status-tag--${tone}`">
    <span class="status-tag__dot" aria-hidden="true"></span>
    <span>{{ text }}</span>
  </span>
</template>

<script setup lang="ts">
import type { StatusTone } from '@/utils/admin-display'

withDefaults(
  defineProps<{
    text: string
    tone?: StatusTone
  }>(),
  {
    tone: 'neutral'
  }
)
</script>

<style scoped>
.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1px solid var(--status-tag-border, rgba(148, 163, 184, 0.2));
  background: var(--status-tag-bg, rgba(248, 250, 252, 0.92));
  color: var(--status-tag-color, #334155);
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
}

.status-tag__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.9;
}

.status-tag--neutral {
  --status-tag-bg: rgba(241, 245, 249, 0.96);
  --status-tag-border: rgba(148, 163, 184, 0.28);
  --status-tag-color: #475569;
}

.status-tag--primary {
  --status-tag-bg: rgba(37, 99, 235, 0.12);
  --status-tag-border: rgba(37, 99, 235, 0.22);
  --status-tag-color: #1d4ed8;
}

.status-tag--success {
  --status-tag-bg: rgba(22, 163, 74, 0.12);
  --status-tag-border: rgba(22, 163, 74, 0.22);
  --status-tag-color: #15803d;
}

.status-tag--warning {
  --status-tag-bg: rgba(217, 119, 6, 0.12);
  --status-tag-border: rgba(217, 119, 6, 0.22);
  --status-tag-color: #b45309;
}

.status-tag--danger {
  --status-tag-bg: rgba(220, 38, 38, 0.12);
  --status-tag-border: rgba(220, 38, 38, 0.22);
  --status-tag-color: #b91c1c;
}
</style>
