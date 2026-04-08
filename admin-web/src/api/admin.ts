import { request } from './http'
import type {
  CallLogRecord,
  DashboardOverview,
  GroupTestResult,
  JobRunRecord,
  PaginationResult,
  RuntimeSettingGroupView,
  SettingAuditRecord,
  TrendBucket
} from '@/types'

export function login(username: string, password: string) {
  return request<{ username: string }>('/api/admin/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  })
}

export function logout() {
  return request<{ ok: boolean }>('/api/admin/auth/logout', {
    method: 'POST'
  })
}

export function getMe() {
  return request<{ username: string }>('/api/admin/auth/me')
}

export function getDashboardOverview() {
  return request<{ overview: DashboardOverview }>('/api/admin/dashboard/overview')
}

export function getDashboardTrends(range: string) {
  return request<{ items: TrendBucket[] }>(`/api/admin/dashboard/trends?range=${encodeURIComponent(range)}`)
}

export function listJobs(query: URLSearchParams) {
  return request<{ result: PaginationResult<JobRunRecord> }>(`/api/admin/ai/jobs?${query.toString()}`)
}

export function getJobDetail(id: number) {
  return request<{ job: JobRunRecord; calls: CallLogRecord[] }>(`/api/admin/ai/jobs/${id}`)
}

export function listCalls(query: URLSearchParams) {
  return request<{ result: PaginationResult<CallLogRecord> }>(`/api/admin/ai/calls?${query.toString()}`)
}

export function getRuntimeSettings() {
  return request<{ groups: RuntimeSettingGroupView[] }>('/api/admin/runtime-settings')
}

export function updateRuntimeGroup(group: string, payload: { values?: Record<string, unknown>; clearKeys?: string[] }) {
  return request<{ group: RuntimeSettingGroupView }>(`/api/admin/runtime-settings/groups/${encodeURIComponent(group)}`, {
    method: 'PUT',
    body: JSON.stringify(payload)
  })
}

export function testRuntimeGroup(group: string, payload: { values?: Record<string, unknown>; clearKeys?: string[] }) {
  return request<{ result: GroupTestResult }>(`/api/admin/runtime-settings/groups/${encodeURIComponent(group)}/test`, {
    method: 'POST',
    body: JSON.stringify(payload)
  })
}

export function listSettingAudits(query: URLSearchParams) {
  return request<{ result: PaginationResult<SettingAuditRecord> }>(
    `/api/admin/runtime-settings/audits?${query.toString()}`
  )
}
