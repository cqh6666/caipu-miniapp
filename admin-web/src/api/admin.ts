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

export function getDashboardOverview() {
  return request<{ overview: DashboardOverview }>('/admin/dashboard/overview')
}

export function getDashboardTrends(range: string) {
  return request<{ items: TrendBucket[] }>(`/admin/dashboard/trends?range=${encodeURIComponent(range)}`)
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
