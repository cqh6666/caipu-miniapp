import { computed, ref, type Ref } from 'vue'
import type { Router } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type {
  AIRoutingAlertMutationResult,
  AIRoutingAlertOverview,
  AIRoutingAlertOverviewItem,
  AIRoutingSceneKey
} from '@/types'
import {
  findSceneActiveAlertItems,
  findSceneReviewAlertItems,
  isWithinLast24Hours,
  resolveAlertStatus
} from '@/utils/ai-provider-alerts'

interface AlertApi {
  retestAIRoutingAlert: (providerId: string) => Promise<{ result: AIRoutingAlertMutationResult }>
  archiveAIRoutingAlert: (providerId: string, reason: string) => Promise<{ result: AIRoutingAlertMutationResult }>
  muteAIRoutingAlert: (providerId: string, hours: number, reason: string) => Promise<{ result: AIRoutingAlertMutationResult }>
  unmuteAIRoutingAlert: (providerId: string) => Promise<{ result: AIRoutingAlertMutationResult }>
}

export function useAIRoutingAlerts(options: {
  currentSceneKey: Ref<AIRoutingSceneKey>
  alertOverview: Ref<AIRoutingAlertOverview | null>
  router: Router
  extractMessage: (error: unknown) => string
  api: AlertApi
  confirm?: typeof ElMessageBox.confirm
  messages?: Pick<typeof ElMessage, 'success' | 'warning' | 'error' | 'info'>
  clipboard?: Pick<Clipboard, 'writeText'>
}) {
  const {
    currentSceneKey,
    alertOverview,
    router,
    extractMessage,
    api,
    confirm = ElMessageBox.confirm,
    messages = ElMessage,
    clipboard = navigator.clipboard
  } = options
  const pendingAlertActions = ref<Record<string, boolean>>({})

  const currentSceneActiveAlertItems = computed(() =>
    findSceneActiveAlertItems(currentSceneKey.value, alertOverview.value)
  )
  const currentSceneReviewAlertItems = computed(() =>
    findSceneReviewAlertItems(currentSceneKey.value, alertOverview.value)
  )
  const currentSceneRecoveredItems = computed(() => {
    const overview = alertOverview.value
    if (!overview) return []
    return overview.items
      .filter((item) =>
        item.scene === currentSceneKey.value &&
        resolveAlertStatus(item) === 'recovered' &&
        isWithinLast24Hours(item.lastRecoveredAt, overview.generatedAt)
      )
      .sort((left, right) => Date.parse(right.lastRecoveredAt || '') - Date.parse(left.lastRecoveredAt || ''))
  })

  const alertStatusSummary = computed(() => {
    const overview = alertOverview.value
    if (!overview) return { tone: 'neutral' as const, text: '当前场景告警加载中' }
    if (!overview.enabled) return { tone: 'neutral' as const, text: '告警未启用' }
    if (!overview.hasDeliveryConfig) return { tone: 'warning' as const, text: '告警配置不完整' }
    const activeCount = currentSceneActiveAlertItems.value.length
    const reviewCount = currentSceneReviewAlertItems.value.length
    if (activeCount > 0) {
      return {
        tone: 'danger' as const,
        text: reviewCount > 0 ? `告警中 ${activeCount} · 待复核 ${reviewCount}` : `当前场景告警 ${activeCount} 项`
      }
    }
    if (reviewCount > 0) return { tone: 'warning' as const, text: `当前场景待复核 ${reviewCount} 项` }
    return { tone: 'success' as const, text: '当前场景无告警' }
  })

  const alertStatusDescription = computed(() => {
    const overview = alertOverview.value
    if (!overview) return '正在拉取最近告警概览。'
    if (!overview.enabled) return '当前未启用连续异常邮件告警，可在配置中心开启。'
    if (!overview.hasDeliveryConfig) return '告警已启用，但 SMTP 或收件人配置不完整，当前不会形成有效投递。'
    const activeCount = currentSceneActiveAlertItems.value.length
    const reviewCount = currentSceneReviewAlertItems.value.length
    if (activeCount > 0) {
      const suffix = reviewCount > 0 ? `，另有 ${reviewCount} 个待复核（黄色）` : ''
      return `当前场景有 ${activeCount} 个 Provider 处于告警中（红色）${suffix}。`
    }
    if (reviewCount > 0) return `当前场景有 ${reviewCount} 个 Provider 待复核（历史过期 / 配置变更 / 已静默），不计入红色告警。`
    return '阈值、活跃窗口、SMTP 和收件人统一在配置中心维护，当前场景最近无告警。'
  })

  const currentSceneAlertSections = computed(() => [
    { key: 'active' as const, title: '当前告警', hint: '红色：仍在失败且在线上路由中，需立即处理', items: currentSceneActiveAlertItems.value },
    { key: 'review' as const, title: '待复核', hint: '黄色：历史过期 / 配置变更 / 已静默，不计入红色', items: currentSceneReviewAlertItems.value },
    { key: 'recovered' as const, title: '最近恢复', hint: '24 小时内复测或真实调用已恢复', items: currentSceneRecoveredItems.value }
  ])
  const hasCurrentSceneAlertNodes = computed(() => currentSceneAlertSections.value.some((section) => section.items.length > 0))

  function isPending(providerId: string) {
    return !!pendingAlertActions.value[providerId]
  }

  function setPending(providerId: string, pending: boolean) {
    const next = { ...pendingAlertActions.value }
    if (pending) next[providerId] = true
    else delete next[providerId]
    pendingAlertActions.value = next
  }

  async function runAction(providerId: string, action: () => ReturnType<AlertApi['retestAIRoutingAlert']>) {
    setPending(providerId, true)
    try {
      const { result } = await action()
      if (result.overview) alertOverview.value = result.overview
      if (result.ok) messages.success(result.message || '操作成功')
      else messages.warning(result.message || '操作已完成')
      return result
    } catch (error) {
      messages.error(extractMessage(error))
      return null
    } finally {
      setPending(providerId, false)
    }
  }

  async function handleAlertRetest(item: AIRoutingAlertOverviewItem) {
    if (!item.canRetest || isPending(item.providerId)) return
    try {
      await confirm(`将对「${item.providerName}」发起一次真实上游调用以复测，可能产生额度/费用。确认继续？`, '复测并恢复', { confirmButtonText: '复测', cancelButtonText: '取消', type: 'warning' })
    } catch { return }
    await runAction(item.providerId, () => api.retestAIRoutingAlert(item.providerId))
  }

  async function handleAlertArchive(item: AIRoutingAlertOverviewItem) {
    if (!item.canArchive || isPending(item.providerId)) return
    try {
      await confirm(`确认归档「${item.providerName}」的告警？归档后不再计入当前状态，新失败会自动重新触发。`, '确认归档', { confirmButtonText: '归档', cancelButtonText: '取消', type: 'warning' })
    } catch { return }
    await runAction(item.providerId, () => api.archiveAIRoutingAlert(item.providerId, '后台手动归档'))
  }

  async function handleAlertMute(item: AIRoutingAlertOverviewItem) {
    if (!item.canMute || isPending(item.providerId)) return
    await runAction(item.providerId, () => api.muteAIRoutingAlert(item.providerId, 24, '后台手动静默'))
  }

  async function handleAlertUnmute(item: AIRoutingAlertOverviewItem) {
    if (!item.canUnmute || isPending(item.providerId)) return
    await runAction(item.providerId, () => api.unmuteAIRoutingAlert(item.providerId))
  }

  async function handleBatchRetest(items: AIRoutingAlertOverviewItem[]) {
    const targets = items.filter((item) => item.canRetest)
    if (!targets.length) return messages.info('没有可复测的节点')
    try {
      await confirm(`将对 ${targets.length} 个节点各发起一次真实调用，可能产生额度/费用。确认继续？`, '批量复测', { confirmButtonText: '复测', cancelButtonText: '取消', type: 'warning' })
    } catch { return }
    for (const item of targets) await runAction(item.providerId, () => api.retestAIRoutingAlert(item.providerId))
  }

  async function handleBatchArchive(items: AIRoutingAlertOverviewItem[]) {
    const targets = items.filter((item) => item.canArchive)
    if (!targets.length) return messages.info('没有可归档的节点')
    try {
      await confirm(`确认归档 ${targets.length} 个待复核节点？新失败仍会自动重新触发告警。`, '批量归档', { confirmButtonText: '归档', cancelButtonText: '取消', type: 'warning' })
    } catch { return }
    for (const item of targets) await runAction(item.providerId, () => api.archiveAIRoutingAlert(item.providerId, '批量归档'))
  }

  function goAlertConfig() {
    void router.push({ path: '/settings', query: { group: 'ai.provider_alert' }, hash: '#ai-provider-alert' })
  }

  function goAlertProviderLogs(item: AIRoutingAlertOverviewItem) {
    void router.push({ path: '/ai-calls', query: { provider: item.providerId } })
  }

  async function copyRequestId(requestId: string) {
    if (!requestId) return
    try {
      await clipboard.writeText(requestId)
      messages.success('已复制 requestId')
    } catch {
      messages.warning('复制失败，请手动选择复制')
    }
  }

  return {
    alertStatusDescription,
    alertStatusSummary,
    copyRequestId,
    currentSceneActiveAlertItems,
    currentSceneAlertSections,
    currentSceneRecoveredItems,
    currentSceneReviewAlertItems,
    goAlertConfig,
    goAlertProviderLogs,
    handleAlertArchive,
    handleAlertMute,
    handleAlertRetest,
    handleAlertUnmute,
    handleBatchArchive,
    handleBatchRetest,
    hasCurrentSceneAlertNodes,
    pendingAlertActions
  }
}
