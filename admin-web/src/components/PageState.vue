<template>
  <div class="page-state" :class="`page-state--${mode}`" :role="mode === 'error' ? 'alert' : undefined">
    <template v-if="mode === 'loading'">
      <div class="page-state__loading">
        <el-skeleton animated :rows="compact ? 4 : 6" />
      </div>
    </template>

    <template v-else>
      <div class="page-state__symbol" aria-hidden="true">
        {{ mode === 'error' ? '!' : '·' }}
      </div>
      <div class="page-state__title">{{ resolvedTitle }}</div>
      <div class="page-state__description">{{ resolvedDescription }}</div>
      <el-button v-if="mode === 'error'" type="primary" plain @click="$emit('retry')">
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
