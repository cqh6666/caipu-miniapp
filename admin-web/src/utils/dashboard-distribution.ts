import { formatPercent, toneForSuccessRate, type StatusTone } from '@/utils/admin-display'

export type DistViewMode = 'chart' | 'rank'

export interface DistributionItem {
  name: string
  total?: number | null
  successRate?: number | null
}

export interface DistributionRankItem {
  key: string
  label: string
  shortLabel: string
  total: number
  totalText: string
  totalPercent: number
  successRate: number
  rateText: string
  ratePercent: number
  tone: StatusTone
  rankLabel: string
  showAlert: boolean
  alertText: string
  isUnknown: boolean
}

interface DistributionRankBuildOptions {
  labelFormatter?: (value: string) => string
  showAlert?: boolean
}

const countFormatter = new Intl.NumberFormat('zh-CN')

export function isUnknownDistributionName(value?: string) {
  const normalized = String(value || '').trim()
  return !normalized || normalized === '(empty)'
}

export function normalizeDistributionName(value?: string) {
  return isUnknownDistributionName(value) ? '未指定' : String(value || '').trim()
}

export function middleEllipsis(value: string, head = 10, tail = 8) {
  if (value.length <= head + tail + 1) return value
  return `${value.slice(0, head)}…${value.slice(-tail)}`
}

export function buildDistributionRankItems(
  items: DistributionItem[] | undefined | null,
  options: DistributionRankBuildOptions = {}
): DistributionRankItem[] {
  if (!items?.length) return []
  const { labelFormatter, showAlert = true } = options
  const normalized = items.map((item) => {
    const total = Math.max(item.total ?? 0, 0)
    const successRate = Math.max(Math.min(item.successRate ?? 0, 1), 0)
    const normalizedName = normalizeDistributionName(item.name)
    const label = labelFormatter ? labelFormatter(normalizedName) : normalizedName
    return { label, total, successRate, isUnknown: isUnknownDistributionName(item.name) }
  }).sort((left, right) => {
    if (left.isUnknown !== right.isUnknown) return left.isUnknown ? 1 : -1
    if (right.total !== left.total) return right.total - left.total
    return left.label.localeCompare(right.label, 'zh-CN')
  })

  const rankedMax = normalized.filter((item) => !item.isUnknown).reduce((max, item) => Math.max(max, item.total), 0)
  const max = rankedMax || normalized.reduce((value, item) => Math.max(value, item.total), 0) || 1
  let rank = 0

  return normalized.map((item) => {
    const tone = toneForSuccessRate(item.successRate)
    if (!item.isUnknown) rank += 1
    return {
      key: `${item.label}:${item.total}:${item.successRate}`,
      label: item.label,
      shortLabel: middleEllipsis(item.label),
      total: item.total,
      totalText: countFormatter.format(Math.max(Math.round(item.total), 0)),
      totalPercent: Math.max(Math.min((item.total / max) * 100, 100), 0),
      successRate: item.successRate,
      rateText: formatPercent(item.successRate, 0),
      ratePercent: Math.max(Math.min(item.successRate * 100, 100), 0),
      tone,
      rankLabel: item.isUnknown ? '—' : String(rank),
      showAlert: showAlert && tone !== 'success',
      alertText: tone === 'warning' ? '成功率低于 95%，建议关注。' : '成功率低于 85%，建议优先排查。',
      isUnknown: item.isUnknown
    }
  })
}

export function normalizeDistributionItems(items: DistributionItem[] | undefined | null): DistributionItem[] {
  return (items || []).map((item) => ({ ...item, name: normalizeDistributionName(item.name) }))
}
