<template>
  <el-tooltip :content="content" :placement="placement" :show-after="160" effect="dark">
    <button type="button" class="help-tip" :aria-label="ariaLabel">
      <el-icon aria-hidden="true"><InfoFilled /></el-icon>
    </button>
  </el-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { InfoFilled } from '@element-plus/icons-vue'

const props = withDefaults(
  defineProps<{
    content: string
    label?: string
    placement?: 'top' | 'top-start' | 'top-end' | 'bottom' | 'bottom-start' | 'bottom-end' | 'left' | 'right'
  }>(),
  {
    label: '查看说明',
    placement: 'top'
  }
)

const ariaLabel = computed(() => props.label || '查看说明')
</script>

<style scoped>
.help-tip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: #64748b;
  cursor: help;
  transition:
    background 0.16s ease,
    color 0.16s ease,
    box-shadow 0.16s ease;
}

.help-tip:hover,
.help-tip:focus-visible {
  background: rgba(37, 99, 235, 0.1);
  color: #2563eb;
  outline: none;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.14);
}

.help-tip :deep(.el-icon) {
  font-size: 15px;
}
</style>
