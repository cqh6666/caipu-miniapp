<template>
  <div class="filter-toolbar" :class="{ 'filter-toolbar--stacked': hasChips }">
    <div class="filter-toolbar__row">
      <div class="filter-toolbar__content">
        <slot />
      </div>
      <div v-if="$slots.actions" class="filter-toolbar__actions">
        <slot name="actions" />
      </div>
    </div>
    <div v-if="hasChips" class="filter-toolbar__chips">
      <span class="filter-toolbar__chips-label">已应用筛选</span>
      <el-tag
        v-for="chip in activeFilters"
        :key="chip.key"
        type="info"
        closable
        @close="chip.onRemove?.()"
      >{{ chip.label }}</el-tag>
      <el-button v-if="onClearAll" text size="small" @click="onClearAll">清除全部</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface FilterChip {
  key: string
  label: string
  onRemove?: () => void
}

const props = defineProps<{
  activeFilters?: FilterChip[]
  onClearAll?: () => void
}>()

const hasChips = computed(() => (props.activeFilters?.length ?? 0) > 0)
</script>
