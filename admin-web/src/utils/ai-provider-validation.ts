import type { AIRoutingProviderConfig, AIRoutingSceneConfig } from "@/types";

export type ProviderValidationError = {
  name?: string;
  id?: string;
  baseURL?: string;
  model?: string;
  timeoutSeconds?: string;
};

export function isValidHttpUrl(value: string) {
  const raw = String(value || "").trim();
  if (!raw) return false;
  try {
    const parsed = new URL(raw);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

export function getProviderSecretStatus(provider: AIRoutingProviderConfig) {
  if (provider.clearApiKey) return { tone: "warning" as const, text: "待清空" };
  if (provider.apiKey?.trim())
    return { tone: "primary" as const, text: "新密钥草稿" };
  if (provider.hasAPIKey)
    return { tone: "success" as const, text: "已保存密钥" };
  return { tone: "neutral" as const, text: "未配置密钥" };
}

export function providerHasUsableSecret(provider: AIRoutingProviderConfig) {
  if (provider.apiKey?.trim()) return true;
  if (provider.clearApiKey) return false;
  return !!provider.hasAPIKey;
}

export function buildProviderValidationErrors(
  scene: AIRoutingSceneConfig | null,
  providerKey: (provider: AIRoutingProviderConfig) => string,
) {
  const result: Record<string, ProviderValidationError> = {};
  if (!scene) return result;
  const idCounts = new Map<string, number>();
  const nameCounts = new Map<string, number>();
  scene.providers.forEach((provider) => {
    const id = provider.id.trim();
    const name = provider.name.trim();
    if (id) idCounts.set(id, (idCounts.get(id) || 0) + 1);
    if (name) nameCounts.set(name, (nameCounts.get(name) || 0) + 1);
  });
  scene.providers.forEach((provider) => {
    const errors: ProviderValidationError = {};
    const id = provider.id.trim();
    const name = provider.name.trim();
    if (!id) errors.id = "内部 Provider ID 缺失，请重新新增节点";
    else if ((idCounts.get(id) || 0) > 1)
      errors.id = "内部 Provider ID 重复，请重新新增节点";
    if (!name) errors.name = "节点名称不能为空";
    else if ((nameCounts.get(name) || 0) > 1)
      errors.name = "节点名称在当前场景内重复";
    if (!isValidHttpUrl(provider.baseURL))
      errors.baseURL = "Base URL 必须是合法的 http(s) 地址";
    if (!provider.model.trim()) errors.model = "Model 不能为空";
    const timeout = Number(provider.timeoutSeconds);
    if (!Number.isFinite(timeout) || timeout < 1 || timeout > 600)
      errors.timeoutSeconds = "超时范围必须为 1-600 秒";
    if (Object.keys(errors).length) result[providerKey(provider)] = errors;
  });
  return result;
}

export function countProviderValidationErrors(
  errors: Record<string, ProviderValidationError>,
) {
  return Object.values(errors).reduce(
    (total, item) => total + Object.keys(item).length,
    0,
  );
}

export function buildSceneBlockingValidationMessages(
  scene: AIRoutingSceneConfig | null,
) {
  if (!scene) return [];
  const enabledProviderCount = scene.providers.filter(
    (provider) => provider.enabled,
  ).length;
  if (
    scene.enabled &&
    enabledProviderCount > 1 &&
    Number(scene.maxAttempts || 0) < 2
  )
    return ["当前启用多个节点，最大尝试次数至少为 2 才能触发失败切换。"];
  return [];
}
