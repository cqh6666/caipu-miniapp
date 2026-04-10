export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface DashboardMetric {
  name: string
  total: number
  successRate: number
}

export interface JobRunRecord {
  id: number
  scene: string
  targetType: string
  targetId: string
  triggerSource: string
  status: string
  finalProvider: string
  finalModel: string
  fallbackUsed: boolean
  errorMessage: string
  requestId: string
  startedAt: string
  finishedAt: string
  durationMs: number
  metaJson: string
}

export interface CallLogRecord {
  id: number
  jobRunId: number
  scene: string
  provider: string
  endpoint: string
  model: string
  status: string
  httpStatus: number
  latencyMs: number
  errorType: string
  errorMessage: string
  requestId: string
  metaJson: string
  createdAt: string
}

export interface PaginationResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export interface DashboardOverview {
  windowHours: number
  taskTotal: number
  taskSuccessRate: number
  apiTotal: number
  apiSuccessRate: number
  timeoutRate: number
  avgDurationMs: number
  p95DurationMs: number
  byScene: DashboardMetric[]
  byModel: DashboardMetric[]
  byProvider: DashboardMetric[]
  recentFailures: JobRunRecord[]
}

export interface TrendBucket {
  bucket: string
  label: string
  taskTotal: number
  taskSuccessRate: number
  apiTotal: number
  apiSuccessRate: number
  avgDurationMs: number
}

export type ServerHealthStatus = 'healthy' | 'warning' | 'critical' | 'unknown'

export interface ServerHealthSummary {
  status: ServerHealthStatus
  healthyCount: number
  warningCount: number
  criticalCount: number
  unknownCount: number
}

export interface ServerHealthHost {
  hostname: string
  platform: string
  uptimeSeconds: number
  cpuUsagePercent: number | null
  memoryUsagePercent: number | null
  diskUsagePercent: number | null
  load1: number | null
  load5: number | null
  load15: number | null
}

export interface ServerHealthCheck {
  key: string
  label: string
  category: 'systemd' | 'http'
  status: ServerHealthStatus
  target: string
  detail: string
  latencyMs: number | null
  checkedAt: string
}

export interface ServerHealthOverview {
  generatedAt: string
  summary: ServerHealthSummary
  host: ServerHealthHost
  checks: ServerHealthCheck[]
}

export interface RuntimeSettingFieldView {
  key: string
  label: string
  description: string
  valueType: 'string' | 'int' | 'bool' | 'float'
  isSecret: boolean
  isRestartRequired: boolean
  hasValue: boolean
  value: string
  maskedValue: string
  source: string
  updatedAt: string
  updatedBySubject: string
}

export interface RuntimeSettingGroupView {
  name: string
  title: string
  description: string
  fields: RuntimeSettingFieldView[]
}

export interface SettingAuditRecord {
  id: number
  groupName: string
  settingKey: string
  action: string
  oldValueMasked: string
  newValueMasked: string
  operatorSubject: string
  requestId: string
  createdAt: string
}

export interface GroupTestResult {
  ok: boolean
  latencyMs: number
  message: string
}

export type AIRoutingSceneKey = 'summary' | 'title' | 'flowchart'
export type AIRoutingStrategy = 'priority_failover' | 'round_robin_failover'

export interface AIRoutingBreakerConfig {
  failureThreshold: number
  cooldownSeconds: number
}

export interface AIRoutingRequestOptions {
  stream: boolean
  temperature: number
  maxTokens: number
}

export interface AIRoutingProviderConfig {
  id: string
  scene?: AIRoutingSceneKey
  name: string
  adapter: string
  enabled: boolean
  priority: number
  weight?: number
  baseURL: string
  apiKey?: string
  apiKeyMasked?: string
  hasAPIKey: boolean
  clearApiKey?: boolean
  model: string
  timeoutSeconds: number
  updatedBySubject?: string
  updatedAt?: string
}

export interface AIRoutingSceneConfig {
  scene: AIRoutingSceneKey
  enabled: boolean
  strategy: AIRoutingStrategy
  maxAttempts: number
  retryOn: string[]
  breaker: AIRoutingBreakerConfig
  requestOptions: AIRoutingRequestOptions
  providers: AIRoutingProviderConfig[]
  updatedBySubject?: string
  updatedAt?: string
  source?: string
  compatibilityMode?: boolean
}

export interface AIRoutingSceneSummary {
  scene: AIRoutingSceneKey
  enabled: boolean
  strategy: AIRoutingStrategy
  providerCount: number
  activeProviderCount: number
  updatedBySubject?: string
  updatedAt?: string
  source?: string
  compatibilityMode?: boolean
}

export interface AIRoutingAttemptResult {
  providerId: string
  providerName: string
  model: string
  status: string
  httpStatus: number
  errorType?: string
  errorMessage?: string
  latencyMs: number
  skippedByBreaker?: boolean
  breakerOpenUntil?: string
}

export interface AIRoutingTestResult {
  ok: boolean
  message: string
  finalProvider?: string
  finalModel?: string
  attempts: AIRoutingAttemptResult[]
}
