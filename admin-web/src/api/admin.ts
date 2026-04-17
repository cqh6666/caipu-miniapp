import { request } from './http'
import type {
  AIRoutingSceneConfig,
  AIRoutingSceneSummary,
  AIRoutingTestResult,
  CallLogRecord,
  DashboardOverview,
  GroupTestResult,
  JobRunRecord,
  PaginationResult,
  RuntimeSettingGroupView,
  ServerHealthOverview,
  SettingAuditRecord,
  TrendBucket
} from '@/types'

export function login(username: string, password: string) {
  return request<{ username: string }>('/admin/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  })
}

export function logout() {
  return request<{ ok: boolean }>('/admin/auth/logout', {
    method: 'POST'
  })
}

export function getMe() {
  return request<{ username: string }>('/admin/auth/me')
}

export function getDashboardOverview(windowHours?: number) {
  const suffix = windowHours && windowHours > 0 ? `?windowHours=${windowHours}` : ''
  return request<{ overview: DashboardOverview }>(`/admin/dashboard/overview${suffix}`)
}

export function getDashboardTrends(range: string) {
  return request<{ items: TrendBucket[] }>(`/admin/dashboard/trends?range=${encodeURIComponent(range)}`)
}

export function getServerHealthOverview() {
  return request<{ overview: ServerHealthOverview }>('/admin/server-health/overview')
}

export function listJobs(query: URLSearchParams) {
  return request<{ result: PaginationResult<JobRunRecord> }>(`/admin/ai/jobs?${query.toString()}`)
}

export function getJobDetail(id: number) {
  return request<{ job: JobRunRecord; calls: CallLogRecord[] }>(`/admin/ai/jobs/${id}`)
}

export function listCalls(query: URLSearchParams) {
  return request<{ result: PaginationResult<CallLogRecord> }>(`/admin/ai/calls?${query.toString()}`)
}

export function getRuntimeSettings() {
  return request<{ groups: RuntimeSettingGroupView[] }>('/admin/runtime-settings')
}

export function updateRuntimeGroup(group: string, payload: { values?: Record<string, unknown>; clearKeys?: string[] }) {
  return request<{ group: RuntimeSettingGroupView }>(`/admin/runtime-settings/groups/${encodeURIComponent(group)}`, {
    method: 'PUT',
    body: JSON.stringify(payload)
  })
}

export function testRuntimeGroup(group: string, payload: { values?: Record<string, unknown>; clearKeys?: string[] }) {
  return request<{ result: GroupTestResult }>(`/admin/runtime-settings/groups/${encodeURIComponent(group)}/test`, {
    method: 'POST',
    body: JSON.stringify(payload)
  })
}

export function listSettingAudits(query: URLSearchParams) {
  return request<{ result: PaginationResult<SettingAuditRecord> }>(
    `/admin/runtime-settings/audits?${query.toString()}`
  )
}

export function listAIRoutingScenes() {
  return request<{ items: AIRoutingSceneSummary[] }>('/admin/ai-routing/scenes')
}

export function getAIRoutingScene(scene: string) {
  return request<{ scene: AIRoutingSceneConfig }>(`/admin/ai-routing/scenes/${encodeURIComponent(scene)}`)
}

export function updateAIRoutingScene(scene: string, payload: AIRoutingSceneConfig) {
  return request<{ scene: AIRoutingSceneConfig }>(`/admin/ai-routing/scenes/${encodeURIComponent(scene)}`, {
    method: 'PUT',
    body: JSON.stringify(payload)
  })
}

export function testAIRoutingScene(scene: string, payload: AIRoutingSceneConfig) {
  return request<{ result: AIRoutingTestResult }>(`/admin/ai-routing/scenes/${encodeURIComponent(scene)}/test`, {
    method: 'POST',
    body: JSON.stringify(payload)
  })
}
