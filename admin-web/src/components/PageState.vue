<template>
  <div class="page-state" :class="`page-state--${mode}`" :role="mode === 'error' ? 'alert' : undefined">
    <template v-if="mode === 'loading'">
      <div class="page-state__loading">
        <el-skeleton animated :rows="compact ? 4 : 6" />
      </div>
    </template>

    <template v-else>
      <div class="page-state__illustration" aria-hidden="true">
        <!-- Empty illustration -->
        <svg v-if="mode === 'empty'" viewBox="0 0 80 80" fill="none" xmlns="http://www.w3.org/2000/svg">
          <rect x="12" y="16" width="56" height="48" rx="8" fill="#eef3fb" stroke="#c7d2e3" stroke-width="1.5" />
          <path d="M12 28 H68" stroke="#c7d2e3" stroke-width="1.5" />
          <circle cx="20" cy="22" r="1.5" fill="#94a3b8" />
          <circle cx="26" cy="22" r="1.5" fill="#94a3b8" />
          <circle cx="32" cy="22" r="1.5" fill="#94a3b8" />
          <rect x="22" y="38" width="36" height="4" rx="2" fill="#cbd5e1" />
          <rect x="22" y="46" width="26" height="4" rx="2" fill="#dbe3ef" />
          <rect x="22" y="54" width="20" height="4" rx="2" fill="#dbe3ef" />
        </svg>

        <!-- Error illustration -->
        <svg v-else-if="mode === 'error'" viewBox="0 0 80 80" fill="none" xmlns="http://www.w3.org/2000/svg">
          <circle cx="40" cy="40" r="30" fill="#fee2e2" />
          <circle cx="40" cy="40" r="22" fill="#fecaca" />
          <path d="M40 28 V44" stroke="#dc2626" stroke-width="3.5" stroke-linecap="round" />
          <circle cx="40" cy="52" r="2.5" fill="#dc2626" />
        </svg>
      </div>
      <div class="page-state__title">{{ resolvedTitle }}</div>
      <div class="page-state__description">{{ resolvedDescription }}</div>
      <el-button
        v-if="mode === 'error'"
        type="primary"
        plain
        class="page-state__action"
        @click="$emit('retry')"
      >
        {{ retryText }}
      </el-button>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    mode: 'loading' | 'empty' | 'error'
    title?: string
    description?: string
    retryText?: string
    compact?: boolean
  }>(),
  {
    title: '',
    description: '',
    retryText: '重新加载',
    compact: false
  }
)

defineEmits<{
  (event: 'retry'): void
}>()

const resolvedTitle = computed(() => {
  if (props.title) {
    return props.title
  }
  return props.mode === 'error' ? '加载失败' : '暂无数据'
})

const resolvedDescription = computed(() => {
  if (props.description) {
    return props.description
  }
  if (props.mode === 'error') {
    return '请稍后重试，或检查接口与登录态是否正常。'
  }
  return '当前时间窗口内还没有可展示的数据。'
})
</script>
