import { nextTick, ref, type ComputedRef, type Ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { AIRoutingProviderConfig, AIRoutingSceneConfig, AIRoutingTestResult } from '@/types'
import { normalizeProviderExtra } from '@/utils/ai-provider-draft'
import type { ProviderValidationError } from '@/utils/ai-provider-validation'

export type ProviderTestState = {
  ok: boolean
  text: string
  latencyMs?: number
  errorType?: string
  testedAt: string
}

export function useAIProviderEditor(options: {
  draftScene: Ref<AIRoutingSceneConfig | null>
  validationErrors: ComputedRef<Record<string, ProviderValidationError>>
  formatDuration: (value?: number | null) => string
  formatTestMessage: (value?: string) => string
  displayCallStatus: (value?: string) => string
  confirm?: typeof ElMessageBox.confirm
  messages?: Pick<typeof ElMessage, 'info' | 'warning'>
}) {
  const {
    draftScene,
    validationErrors,
    formatDuration,
    formatTestMessage,
    displayCallStatus,
    confirm = ElMessageBox.confirm,
    messages = ElMessage
  } = options
  const draggingProviderIndex = ref<number | null>(null)
  const dragOverProviderIndex = ref<number | null>(null)
  const providerSecretEditorState = ref<Record<string, boolean>>({})
  const collapsedProviderKeys = ref<Set<string>>(new Set())
  const providerTouchedState = ref<Record<string, boolean>>({})
  const providerLastTestState = ref<Record<string, ProviderTestState>>({})
  const providerLocalKeys = new WeakMap<AIRoutingProviderConfig, string>()
  let providerLocalKeyCounter = 0

  function getProviderLocalKey(provider: AIRoutingProviderConfig) {
    const existing = providerLocalKeys.get(provider)
    if (existing) return existing
    const key = `provider-local-${(providerLocalKeyCounter += 1)}`
    providerLocalKeys.set(provider, key)
    return key
  }

  function preserveViewportAfterUpdate(update: () => void) {
    const scrollX = window.scrollX
    const scrollY = window.scrollY
    update()
    nextTick(() => window.scrollTo(scrollX, scrollY))
  }

  function setProviderSecretEditor(provider: AIRoutingProviderConfig, open: boolean) {
    providerSecretEditorState.value = {
      ...providerSecretEditorState.value,
      [getProviderLocalKey(provider)]: open
    }
  }

  function shouldShowProviderSecretEditor(provider: AIRoutingProviderConfig) {
    return !provider.hasAPIKey || !!provider.apiKey?.trim() || !!providerSecretEditorState.value[getProviderLocalKey(provider)]
  }

  function toggleProviderSecretEditor(provider: AIRoutingProviderConfig) {
    preserveViewportAfterUpdate(() => {
      if (provider.clearApiKey) provider.clearApiKey = false
      setProviderSecretEditor(provider, !shouldShowProviderSecretEditor(provider))
    })
  }

  async function handleClearProviderApiKey(provider: AIRoutingProviderConfig) {
    if (provider.clearApiKey) {
      provider.clearApiKey = false
      messages.info('已撤销清空密钥')
      return
    }
    try {
      await confirm('清空后需要保存当前场景才会真正移除旧密钥，是否继续？', '确认清空密钥', { type: 'warning' })
    } catch { return }
    provider.apiKey = ''
    provider.clearApiKey = true
    setProviderSecretEditor(provider, false)
    messages.warning('已标记为清空密钥，保存后生效')
  }

  function handleProviderApiKeyInput(provider: AIRoutingProviderConfig, value: string) {
    provider.apiKey = value
    if (value.trim()) {
      provider.clearApiKey = false
      setProviderSecretEditor(provider, true)
    }
  }

  function isProviderCollapsed(provider: AIRoutingProviderConfig) {
    return collapsedProviderKeys.value.has(getProviderLocalKey(provider))
  }

  function toggleProviderCollapsed(provider: AIRoutingProviderConfig) {
    preserveViewportAfterUpdate(() => {
      const key = getProviderLocalKey(provider)
      const next = new Set(collapsedProviderKeys.value)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      collapsedProviderKeys.value = next
    })
  }

  function handleProviderDragStart(index: number, event: DragEvent) {
    draggingProviderIndex.value = index
    dragOverProviderIndex.value = index
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move'
      event.dataTransfer.setData('text/plain', String(index))
    }
  }

  function handleProviderDragOver(index: number) {
    if (draggingProviderIndex.value === null || draggingProviderIndex.value === index) return
    dragOverProviderIndex.value = index
  }

  function handleProviderDragEnd() {
    draggingProviderIndex.value = null
    dragOverProviderIndex.value = null
  }

  function handleProviderDrop(index: number) {
    if (!draftScene.value || draggingProviderIndex.value === null) {
      handleProviderDragEnd()
      return
    }
    const sourceIndex = draggingProviderIndex.value
    if (sourceIndex !== index) {
      const items = draftScene.value.providers
      const [current] = items.splice(sourceIndex, 1)
      items.splice(index, 0, current)
    }
    handleProviderDragEnd()
  }

  function touchProviderField(provider: AIRoutingProviderConfig, field: keyof ProviderValidationError) {
    providerTouchedState.value = {
      ...providerTouchedState.value,
      [`${getProviderLocalKey(provider)}:${field}`]: true
    }
  }

  function touchAllProviderFields() {
    const next = { ...providerTouchedState.value }
    draftScene.value?.providers.forEach((provider) => {
      next[`${getProviderLocalKey(provider)}:__all`] = true
    })
    providerTouchedState.value = next
  }

  function providerFieldError(provider: AIRoutingProviderConfig, field: keyof ProviderValidationError) {
    const key = getProviderLocalKey(provider)
    if (!providerTouchedState.value[`${key}:${field}`] && !providerTouchedState.value[`${key}:__all`]) return ''
    return validationErrors.value[key]?.[field] || ''
  }

  function firstProviderError(provider: AIRoutingProviderConfig) {
    const errors = validationErrors.value[getProviderLocalKey(provider)]
    return errors ? Object.values(errors)[0] || '' : ''
  }

  function getProviderTestState(provider: AIRoutingProviderConfig) {
    return providerLastTestState.value[provider.id.trim()]
  }

  function recordProviderTestState(result: AIRoutingTestResult) {
    const testedAt = new Date().toISOString()
    const next = { ...providerLastTestState.value }
    result.attempts.forEach((attempt) => {
      const ok = attempt.status === 'success'
      next[attempt.providerId] = {
        ok,
        text: ok
          ? `通过 · ${formatDuration(attempt.latencyMs)}`
          : `${formatTestMessage(attempt.errorType || attempt.errorMessage || displayCallStatus(attempt.status))} · ${formatDuration(attempt.latencyMs)}`,
        latencyMs: attempt.latencyMs,
        errorType: attempt.errorType,
        testedAt
      }
    })
    providerLastTestState.value = next
  }

  function isImageGenerationProvider(provider: AIRoutingProviderConfig) {
    return (provider.endpointMode || 'chat_completions') === 'images_generations'
  }

  function handleEndpointModeChange(provider: AIRoutingProviderConfig) {
    if (!isImageGenerationProvider(provider) || !provider.responseFormat) provider.responseFormat = 'auto'
    provider.extra = normalizeProviderExtra(provider)
  }

  function handleThinkingTypeChange(provider: AIRoutingProviderConfig) {
    if (provider.extra.thinking_type === 'disabled') provider.extra.reasoning_effort = ''
    provider.extra = normalizeProviderExtra(provider)
  }

  function providerThinkingLabel(provider: AIRoutingProviderConfig) {
    if (isImageGenerationProvider(provider)) return ''
    const thinkingType = String(provider.extra?.thinking_type || '').trim()
    return !thinkingType || thinkingType === 'auto' ? '' : `thinking ${thinkingType}`
  }

  function resetProviderUIState() {
    providerSecretEditorState.value = {}
    providerTouchedState.value = {}
    handleProviderDragEnd()
    if (draftScene.value && draftScene.value.providers.length > 3) {
      collapsedProviderKeys.value = new Set(
        draftScene.value.providers.slice(1).map((provider) => getProviderLocalKey(provider))
      )
    } else {
      collapsedProviderKeys.value = new Set()
    }
  }

  return {
    dragOverProviderIndex,
    draggingProviderIndex,
    firstProviderError,
    getProviderLocalKey,
    getProviderTestState,
    handleClearProviderApiKey,
    handleEndpointModeChange,
    handleProviderApiKeyInput,
    handleProviderDragEnd,
    handleProviderDragOver,
    handleProviderDragStart,
    handleProviderDrop,
    handleThinkingTypeChange,
    isImageGenerationProvider,
    isProviderCollapsed,
    providerFieldError,
    providerThinkingLabel,
    recordProviderTestState,
    resetProviderUIState,
    setProviderSecretEditor,
    shouldShowProviderSecretEditor,
    toggleProviderCollapsed,
    toggleProviderSecretEditor,
    touchAllProviderFields,
    touchProviderField
  }
}
