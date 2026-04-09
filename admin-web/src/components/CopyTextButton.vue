<template>
  <el-button
    text
    size="small"
    class="copy-text-button"
    :disabled="disabled || !text"
    :aria-label="ariaLabel || label"
    @click="handleCopy"
  >
    {{ copied ? successLabel : label }}
  </el-button>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { ElMessage } from 'element-plus'

const props = withDefaults(
  defineProps<{
    text?: string
    label?: string
    successLabel?: string
    ariaLabel?: string
    disabled?: boolean
  }>(),
  {
    text: '',
    label: '复制',
    successLabel: '已复制',
    ariaLabel: '',
    disabled: false
  }
)

const copied = ref(false)
let resetTimer: number | undefined

async function writeText(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'readonly')
  textarea.style.position = 'absolute'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  document.body.removeChild(textarea)
}

async function handleCopy() {
  if (!props.text) {
    return
  }

  try {
    await writeText(props.text)
    copied.value = true
    window.clearTimeout(resetTimer)
    resetTimer = window.setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '复制失败，请稍后重试')
  }
}

onBeforeUnmount(() => {
  window.clearTimeout(resetTimer)
})
</script>
