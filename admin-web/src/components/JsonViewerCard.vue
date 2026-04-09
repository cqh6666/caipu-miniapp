<template>
  <section class="json-viewer">
    <div class="json-viewer__header">
      <div>
        <h3 class="json-viewer__title">{{ title }}</h3>
        <div v-if="description" class="json-viewer__description">{{ description }}</div>
      </div>
      <div class="json-viewer__actions">
        <CopyTextButton :text="formattedValue" label="复制 JSON" />
        <el-button text size="small" @click="expanded = !expanded">
          {{ expanded ? '收起' : '展开' }}
        </el-button>
      </div>
    </div>
    <pre class="json-viewer__body" :class="{ 'json-viewer__body--collapsed': !expanded }">{{ formattedValue }}</pre>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import CopyTextButton from '@/components/CopyTextButton.vue'

const props = withDefaults(
  defineProps<{
    title: string
    raw?: string
    description?: string
    defaultExpanded?: boolean
  }>(),
  {
    raw: '',
    description: '',
    defaultExpanded: false
  }
)

const expanded = ref(props.defaultExpanded)

const formattedValue = computed(() => {
  const raw = props.raw?.trim()
  if (!raw) {
    return '{}'
  }
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
})
</script>
