<template>
  <div class="filter-toolbar">
    <div class="filter-toolbar__content">
      <slot />
    </div>
    <div v-if="$slots.actions" class="filter-toolbar__actions">
      <slot name="actions" />
    </div>
  </div>
</template>
