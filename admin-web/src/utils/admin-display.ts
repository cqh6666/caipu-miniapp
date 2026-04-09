export const sceneOptions = [
  { label: '解析总结', value: 'parse_summary' },
  { label: '流程图生成', value: 'flowchart' },
  { label: '标题精修', value: 'title_refine' }
]

export const jobStatusOptions = [
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
  { label: '超时', value: 'timeout' },
  { label: '降级', value: 'fallback' }
]

export const callStatusOptions = [
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
  { label: '超时', value: 'timeout' }
]

export const triggerSourceOptions = [
  { label: 'Worker', value: 'worker' },
  { label: '手动触发', value: 'manual' },
  { label: '预览请求', value: 'preview' }
]

export const auditActionOptions = [
  { label: '保存', value: 'update' },
  { label: '测试', value: 'test' }
]

export type StatusTone = 'neutral' | 'primary' | 'success' | 'warning' | 'danger'
export type HealthResourceKind = 'cpu' | 'memory' | 'disk'

const sceneLabelMap: Record<string, string> = Object.fromEntries(sceneOptions.map((item) => [item.value, item.label]))
const jobStatusLabelMap: Record<string, string> = Object.fromEntries(jobStatusOptions.map((item) => [item.value, item.label]))
const callStatusLabelMap: Record<string, string> = Object.fromEntries(callStatusOptions.map((item) => [item.value, item.label]))
const triggerSourceLabelMap: Record<string, string> = Object.fromEntries(
  triggerSourceOptions.map((item) => [item.value, item.label])
)
const auditActionLabelMap: Record<string, string> = Object.fromEntries(
  auditActionOptions.map((item) => [item.value, item.label])
)

const dateTimeFormatter = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false
})

export function formatPercent(value?: number, digits = 1) {
  return `${((value || 0) * 100).toFixed(digits)}%`
}

export function formatUsagePercent(value?: number | null, digits = 1) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '-'
  }
  return `${value.toFixed(digits)}%`
}

export function formatDateTime(value?: string) {
  if (!value) {
    return '-'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return dateTimeFormatter.format(date).replace(/\//g, '-')
}

export function formatDuration(ms?: number) {
  if (!ms && ms !== 0) {
    return '-'
  }
  if (ms < 1000) {
    return `${ms} ms`
  }
  if (ms < 60_000) {
    return `${(ms / 1000).toFixed(ms >= 10_000 ? 0 : 1)} s`
  }
  return `${(ms / 60_000).toFixed(1)} min`
}

export function formatUptime(seconds?: number | null) {
  if (!seconds || seconds <= 0) {
    return '-'
  }

  const days = Math.floor(seconds / 86_400)
  const hours = Math.floor((seconds % 86_400) / 3_600)
  const minutes = Math.floor((seconds % 3_600) / 60)

  if (days > 0) {
    return `${days} 天 ${hours} 小时`
  }
  if (hours > 0) {
    return `${hours} 小时 ${minutes} 分`
  }
  if (minutes > 0) {
    return `${minutes} 分`
  }
  return `${seconds} 秒`
}

export function formatLoadAverage(...values: Array<number | null | undefined>) {
  const visible = values.filter((value) => value !== null && value !== undefined && !Number.isNaN(value))
  if (!visible.length) {
    return '-'
  }
  return visible.map((value) => Number(value).toFixed(2)).join(' / ')
}

export function displayScene(value?: string) {
  if (!value) {
    return '-'
  }
  return sceneLabelMap[value] || value
}

export function displayJobStatus(value?: string) {
  if (!value) {
    return '未知'
  }
  return jobStatusLabelMap[value] || value
}

export function displayCallStatus(value?: string) {
  if (!value) {
    return '未知'
  }
  return callStatusLabelMap[value] || value
}

export function displayTriggerSource(value?: string) {
  if (!value) {
    return '-'
  }
  return triggerSourceLabelMap[value] || value
}

export function displayAuditAction(value?: string) {
  if (!value) {
    return '-'
  }
  return auditActionLabelMap[value] || value
}

export function displayHealthStatus(value?: string) {
  switch (value) {
    case 'healthy':
      return '正常'
    case 'warning':
      return '关注'
    case 'critical':
      return '异常'
    default:
      return '未知'
  }
}

export function displaySettingSource(value?: string) {
  switch (value) {
    case 'db':
      return '数据库覆盖'
    case 'env':
      return '环境变量'
    case 'none':
      return '未配置'
    default:
      return value || '-'
  }
}

export function displayValueType(value?: string) {
  switch (value) {
    case 'bool':
      return '布尔'
    case 'int':
      return '整数'
    case 'float':
      return '数值'
    default:
      return '文本'
  }
}

export function toneForStatus(status?: string): StatusTone {
  switch (status) {
    case 'success':
      return 'success'
    case 'timeout':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'fallback':
      return 'primary'
    default:
      return 'neutral'
  }
}

export function toneForHealthStatus(status?: string): StatusTone {
  switch (status) {
    case 'healthy':
      return 'success'
    case 'warning':
      return 'warning'
    case 'critical':
      return 'danger'
    default:
      return 'neutral'
  }
}

export function statusForResourceUsage(
  value: number | null | undefined,
  kind: HealthResourceKind
): 'healthy' | 'warning' | 'critical' | 'unknown' {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'unknown'
  }

  const warningThreshold = kind === 'disk' ? 80 : 75
  const criticalThreshold = 90

  if (value >= criticalThreshold) {
    return 'critical'
  }
  if (value >= warningThreshold) {
    return 'warning'
  }
  return 'healthy'
}

export function toneForSuccessRate(value?: number): StatusTone {
  const rate = value || 0
  if (rate >= 0.95) {
    return 'success'
  }
  if (rate >= 0.85) {
    return 'warning'
  }
  return 'danger'
}

export function toneForTimeoutRate(value?: number): StatusTone {
  const rate = value || 0
  if (rate <= 0.02) {
    return 'success'
  }
  if (rate <= 0.08) {
    return 'warning'
  }
  return 'danger'
}

export function toneForLatency(value?: number): StatusTone {
  const latency = value || 0
  if (latency <= 4_000) {
    return 'success'
  }
  if (latency <= 8_000) {
    return 'warning'
  }
  return 'danger'
}
