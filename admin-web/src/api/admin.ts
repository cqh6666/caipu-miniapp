import { request, setAdminCSRFToken } from "./http";
import type {
  AIRoutingAlertMutationResult,
  AIRoutingAlertOverview,
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
  TrendBucket,
} from "@/types";

type AdminSession = { username: string; csrfToken: string };

export async function login(username: string, password: string) {
  const session = await request<AdminSession>("/admin/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  setAdminCSRFToken(session.csrfToken);
  return session;
}

export async function logout() {
	const result = await request<{ ok: boolean }>("/admin/auth/logout", {
		method: "POST",
	});
	setAdminCSRFToken("");
	return result;
}

export async function getMe() {
  const session = await request<AdminSession>("/admin/auth/me");
  setAdminCSRFToken(session.csrfToken);
  return session;
}

export function getDashboardOverview(windowHours?: number) {
  const suffix =
    windowHours && windowHours > 0 ? `?windowHours=${windowHours}` : "";
  return request<{ overview: DashboardOverview }>(
    `/admin/dashboard/overview${suffix}`,
  );
}

export function getDashboardTrends(range: string) {
  return request<{ items: TrendBucket[] }>(
    `/admin/dashboard/trends?range=${encodeURIComponent(range)}`,
  );
}

export function getServerHealthOverview() {
  return request<{ overview: ServerHealthOverview }>(
    "/admin/server-health/overview",
  );
}

export function listJobs(query: URLSearchParams) {
  return request<{ result: PaginationResult<JobRunRecord> }>(
    `/admin/ai/jobs?${query.toString()}`,
  );
}

export function getJobDetail(id: number) {
  return request<{ job: JobRunRecord; calls: CallLogRecord[] }>(
    `/admin/ai/jobs/${id}`,
  );
}

export function listCalls(query: URLSearchParams) {
  return request<{ result: PaginationResult<CallLogRecord> }>(
    `/admin/ai/calls?${query.toString()}`,
  );
}

export function getRuntimeSettings() {
  return request<{ groups: RuntimeSettingGroupView[] }>(
    "/admin/runtime-settings",
  );
}

export function updateRuntimeGroup(
  group: string,
  payload: { expectedVersion: number; values?: Record<string, unknown>; clearKeys?: string[] },
) {
  return request<{ group: RuntimeSettingGroupView }>(
    `/admin/runtime-settings/groups/${encodeURIComponent(group)}`,
    {
      method: "PUT",
      body: JSON.stringify(payload),
    },
  );
}

export function testRuntimeGroup(
  group: string,
  payload: { values?: Record<string, unknown>; clearKeys?: string[] },
) {
  return request<{ result: GroupTestResult }>(
    `/admin/runtime-settings/groups/${encodeURIComponent(group)}/test`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}

export function listSettingAudits(query: URLSearchParams) {
  return request<{ result: PaginationResult<SettingAuditRecord> }>(
    `/admin/runtime-settings/audits?${query.toString()}`,
  );
}

export function listAIRoutingScenes() {
  return request<{ items: AIRoutingSceneSummary[] }>(
    "/admin/ai-routing/scenes",
  );
}

export function getAIRoutingScene(scene: string) {
  return request<{ scene: AIRoutingSceneConfig }>(
    `/admin/ai-routing/scenes/${encodeURIComponent(scene)}`,
  );
}

export function updateAIRoutingScene(
  scene: string,
  payload: AIRoutingSceneConfig,
) {
  return request<{ scene: AIRoutingSceneConfig }>(
    `/admin/ai-routing/scenes/${encodeURIComponent(scene)}`,
    {
      method: "PUT",
      body: JSON.stringify(payload),
    },
  );
}

export function testAIRoutingScene(
  scene: string,
  payload: AIRoutingSceneConfig,
) {
  return request<{ result: AIRoutingTestResult }>(
    `/admin/ai-routing/scenes/${encodeURIComponent(scene)}/test`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}

export function getAIRoutingAlertsOverview() {
  return request<{ overview: AIRoutingAlertOverview }>(
    "/admin/ai-routing/alerts/overview",
  );
}

export function retestAIRoutingAlert(providerId: string) {
  return request<{ result: AIRoutingAlertMutationResult }>(
    `/admin/ai-routing/alerts/${encodeURIComponent(providerId)}/retest`,
    { method: "POST" },
  );
}

export function archiveAIRoutingAlert(providerId: string, reason?: string) {
  return request<{ result: AIRoutingAlertMutationResult }>(
    `/admin/ai-routing/alerts/${encodeURIComponent(providerId)}/archive`,
    {
      method: "POST",
      body: JSON.stringify({ reason: reason || "" }),
    },
  );
}

export function muteAIRoutingAlert(
  providerId: string,
  durationHours = 24,
  reason?: string,
) {
  return request<{ result: AIRoutingAlertMutationResult }>(
    `/admin/ai-routing/alerts/${encodeURIComponent(providerId)}/mute`,
    {
      method: "POST",
      body: JSON.stringify({ durationHours, reason: reason || "" }),
    },
  );
}

export function unmuteAIRoutingAlert(providerId: string) {
  return request<{ result: AIRoutingAlertMutationResult }>(
    `/admin/ai-routing/alerts/${encodeURIComponent(providerId)}/unmute`,
    { method: "POST" },
  );
}
