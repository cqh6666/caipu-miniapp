import type { AIRoutingSceneKey, SettingAuditRecord } from "@/types";
import { displayAuditAction } from "@/utils/admin-display";

export type AuditTone =
  | "neutral"
  | "primary"
  | "success"
  | "warning"
  | "danger";
export type AuditChangeKind = "added" | "removed" | "changed";
export type AuditChangeGroupName =
  | "基础信息"
  | "请求配置"
  | "调度策略"
  | "敏感信息"
  | "其他变化";

export type AuditBusinessAction = {
  label: string;
  tone: AuditTone;
  description: string;
};

export type AuditChangeItem = {
  key: string;
  field: string;
  from: string;
  to: string;
  kind: AuditChangeKind;
  group: AuditChangeGroupName;
  priority: number;
};

export type AIProviderAuditContext = {
  currentScene: () => AIRoutingSceneKey;
  providerName: (providerId: string) => string;
};

export function formatDiffValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "空";
  if (typeof value === "object") return JSON.stringify(value, null, 2);
  return String(value);
}

export function formatAuditValue(value: string) {
  if (!value) return "空";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

export function parseAuditKeyValue(value: string) {
  const result: Record<string, string> = {};
  const raw = String(value || "").trim();
  if (!raw) return result;
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    Object.entries(parsed || {}).forEach(([key, val]) => {
      result[key] = String(val ?? "");
    });
    return result;
  } catch {
    // 兼容旧审计记录的 key=value 文本格式。
  }
  const matches = raw.matchAll(
    /([A-Za-z][\w.-]*)=([^=]+?)(?=\s+[A-Za-z][\w.-]*=|$)/g,
  );
  for (const match of matches) {
    result[match[1]] = match[2].trim();
  }
  return result;
}

export function isAuditEmptyValue(value?: string | null) {
  return (
    value === undefined ||
    value === null ||
    value === "" ||
    value === "空" ||
    value === "未设置" ||
    value === "-"
  );
}

export function auditChangeKind(
  before?: string,
  after?: string,
): AuditChangeKind {
  const oldEmpty = isAuditEmptyValue(before);
  const newEmpty = isAuditEmptyValue(after);
  if (!oldEmpty && newEmpty) return "removed";
  if (oldEmpty && !newEmpty) return "added";
  return "changed";
}

export function auditChangeKindText(kind: string) {
  if (kind === "removed") return "删除";
  if (kind === "added") return "新增";
  return "修改";
}

export function displayAuditField(field: string) {
  const map: Record<string, string> = {
    enabled: "启用状态",
    strategy: "调度策略",
    maxAttempts: "最大尝试次数",
    retryOn: "重试条件",
    breaker: "熔断策略",
    providers: "节点数量",
    requestOptions: "请求参数",
    name: "名称",
    model: "Model",
    baseURL: "Base URL",
    timeout: "超时",
    timeoutSeconds: "超时",
    endpoint: "Endpoint",
    endpointMode: "Endpoint",
    responseFormat: "返回格式",
    priority: "顺序",
    adapter: "适配器",
    apiKey: "密钥状态",
    hasAPIKey: "密钥状态",
    clearApiKey: "清空密钥",
  };
  return map[field] || field;
}

export function auditChangeGroup(field: string): AuditChangeGroupName {
  if (["name", "enabled", "model"].includes(field)) return "基础信息";
  if (
    [
      "baseURL",
      "timeout",
      "timeoutSeconds",
      "endpoint",
      "endpointMode",
      "responseFormat",
      "adapter",
      "requestOptions",
    ].includes(field)
  )
    return "请求配置";
  if (
    [
      "priority",
      "strategy",
      "maxAttempts",
      "retryOn",
      "breaker",
      "providers",
    ].includes(field)
  )
    return "调度策略";
  if (/apiKey|token|secret|password|clearApiKey|hasAPIKey/i.test(field))
    return "敏感信息";
  return "其他变化";
}

export function auditChangePriority(field: string) {
  const map: Record<string, number> = {
    apiKey: 1,
    enabled: 2,
    name: 3,
    model: 4,
    baseURL: 5,
    timeout: 6,
    timeoutSeconds: 6,
    endpoint: 7,
    endpointMode: 7,
    responseFormat: 8,
    priority: 9,
    strategy: 10,
    maxAttempts: 11,
    breaker: 12,
    providers: 13,
    requestOptions: 14,
  };
  return map[field] || 99;
}

export function compactAuditValue(value: unknown, field = "") {
  if (value === undefined || value === null || value === "" || value === "空")
    return "未设置";
  if (/apiKey|token|secret|password/i.test(field)) return "已设置（脱敏）";
  if (typeof value === "boolean") return value ? "启用" : "停用";
  if (typeof value === "object") return JSON.stringify(value);
  const text = String(value);
  if (field === "enabled")
    return text === "true" ? "启用" : text === "false" ? "停用" : text;
  if (/^(sk-|Bearer\s+)/i.test(text)) return "已设置（脱敏）";
  if (text === "true") return "是";
  if (text === "false") return "否";
  return text.length > 72 ? `${text.slice(0, 69)}...` : text;
}

function auditValuesEqual(before?: string, after?: string) {
  if (isAuditEmptyValue(before) && isAuditEmptyValue(after)) return true;
  return before === after;
}

export function auditChangeSummary(
  record: SettingAuditRecord,
): AuditChangeItem[] {
  if (record.action === "test") return [];
  const before = parseAuditKeyValue(record.oldValueMasked);
  const after = parseAuditKeyValue(record.newValueMasked);
  const keys = Array.from(
    new Set([...Object.keys(before), ...Object.keys(after)]),
  );
  return keys
    .filter((key) => !auditValuesEqual(before[key], after[key]))
    .map((key) => ({
      key,
      field: displayAuditField(key),
      from: compactAuditValue(before[key], key),
      to: compactAuditValue(after[key], key),
      kind: auditChangeKind(before[key], after[key]),
      group: auditChangeGroup(key),
      priority: auditChangePriority(key),
    }))
    .sort(
      (left, right) =>
        left.priority - right.priority || left.field.localeCompare(right.field),
    );
}

export function groupedAuditChanges(record: SettingAuditRecord) {
  const groups: AuditChangeGroupName[] = [
    "基础信息",
    "请求配置",
    "调度策略",
    "敏感信息",
    "其他变化",
  ];
  const changes = auditChangeSummary(record);
  return groups
    .map((name) => ({
      name,
      changes: changes.filter((item) => item.group === name),
    }))
    .filter((group) => group.changes.length);
}

export function auditChangeStats(record: SettingAuditRecord) {
  const changes = auditChangeSummary(record);
  return [
    { kind: "added" as const, label: "新增" },
    { kind: "changed" as const, label: "修改" },
    { kind: "removed" as const, label: "删除" },
  ]
    .map((config) => ({
      ...config,
      count: changes.filter((item) => item.kind === config.kind).length,
    }))
    .filter((item) => item.count > 0);
}

function isProviderAuditRecord(record: SettingAuditRecord) {
  return record.settingKey.includes(".provider.");
}

function providerNameFromAuditRecord(record: SettingAuditRecord) {
  const after = parseAuditKeyValue(record.newValueMasked);
  const before = parseAuditKeyValue(record.oldValueMasked);
  return after.name || before.name || "";
}

function providerAddedAuditDescription(after: Record<string, string>) {
  return after.enabled === "false"
    ? "已新增停用节点，暂不参与当前场景调度。"
    : "已新增节点，并参与当前场景调度。";
}

export function createAIProviderAuditPresenter(
  context: AIProviderAuditContext,
) {
  function providerDisplayName(providerId: string) {
    const id = String(providerId || "").trim();
    return id ? context.providerName(id) || id : "-";
  }

  function displayRouteTestMessage(message?: string) {
    const text = String(message || "").trim();
    if (!text || text === "-") return "-";
    const lower = text.toLowerCase();
    const errorTypeMap: Record<string, string> = {
      timeout: "请求超时",
      network: "网络异常",
      rate_limit: "上游限流",
      auth: "鉴权失败",
      upstream: "上游异常",
      invalid_response: "响应格式异常",
    };
    if (errorTypeMap[lower]) return errorTypeMap[lower];
    if (lower.includes("all providers are cooling down"))
      return "所有启用节点都在熔断冷却中";
    if (lower.includes("route test succeeded")) return "路由测试通过";
    const okVia = text.match(/^ok via\s+(.+)$/i);
    if (okVia) return `测试通过，经由 ${providerDisplayName(okVia[1])}`;
    if (lower.includes("context deadline exceeded"))
      return "请求超时，请检查节点响应速度或超时配置";
    if (lower.includes("connection refused"))
      return "连接被拒绝，请检查 Base URL 或网络连通性";
    if (lower.includes("unauthorized") || lower.includes("invalid api key"))
      return "鉴权失败，请检查 API Key";
    if (lower.includes("cooling down")) return "节点处于熔断冷却中";
    return text;
  }

  function auditTargetTitle(target: SettingAuditRecord | string) {
    const settingKey = typeof target === "string" ? target : target.settingKey;
    const prefix = `ai.routing.${context.currentScene()}.`;
    const key = settingKey.startsWith(prefix)
      ? settingKey.slice(prefix.length)
      : settingKey;
    if (key === "scene") return "场景策略";
    if (key.startsWith("provider.")) {
      const providerId = key.slice("provider.".length);
      const auditName =
        typeof target === "string" ? "" : providerNameFromAuditRecord(target);
      return auditName || context.providerName(providerId) || "Provider 节点";
    }
    return key || settingKey;
  }

  function auditFallbackSummary(record: SettingAuditRecord) {
    if (record.action === "test") {
      const summary = displayRouteTestMessage(
        record.newValueMasked || record.oldValueMasked,
      );
      return summary === "-" ? "测试记录已写入" : summary;
    }
    if (record.action === "save" || record.action === "update")
      return "配置已更新，点击查看变化获取完整对比。";
    return record.newValueMasked || record.oldValueMasked || "暂无摘要";
  }

  function auditBusinessAction(record: SettingAuditRecord): AuditBusinessAction {
    if (record.action === "test") {
      const text =
        `${record.newValueMasked} ${record.oldValueMasked}`.toLowerCase();
      const message = displayRouteTestMessage(
        record.newValueMasked || record.oldValueMasked,
      );
      const timeout = text.includes("timeout") || text.includes("deadline");
      const failed =
        timeout ||
        text.includes("fail") ||
        text.includes("error") ||
        text.includes("refused") ||
        text.includes("cooling down");
      if (timeout)
        return {
          label: "测试超时",
          tone: "warning",
          description: "测试请求超时，请检查节点连通性、模型响应速度或超时配置。",
        };
      if (failed)
        return {
          label: "测试异常",
          tone: "warning",
          description: message || "测试未通过，请查看错误摘要。",
        };
      return {
        label: "测试通过",
        tone: "success",
        description: message || "当前草稿路由测试通过。",
      };
    }

    const changes = auditChangeSummary(record);
    const keys = changes.map((item) => item.key);
    const before = parseAuditKeyValue(record.oldValueMasked);
    const after = parseAuditKeyValue(record.newValueMasked);
    const allAdded =
      changes.length > 0 && changes.every((item) => item.kind === "added");
    const allRemoved =
      changes.length > 0 && changes.every((item) => item.kind === "removed");
    const providerRecord = isProviderAuditRecord(record);
    if (providerRecord && allAdded)
      return {
        label: "新增节点",
        tone: "primary",
        description: providerAddedAuditDescription(after),
      };
    if (providerRecord && allRemoved)
      return {
        label: "删除节点",
        tone: "danger",
        description: "该 Provider 节点已从当前场景移除。",
      };
    if (providerRecord && keys.length === 1) {
      const key = keys[0];
      if (key === "apiKey") {
        if (isAuditEmptyValue(before.apiKey) && !isAuditEmptyValue(after.apiKey))
          return {
            label: "新增密钥",
            tone: "warning",
            description: "该节点已新增密钥，密钥内容已脱敏。",
          };
        if (!isAuditEmptyValue(before.apiKey) && isAuditEmptyValue(after.apiKey))
          return {
            label: "清空密钥",
            tone: "danger",
            description: "该节点旧密钥已清空。",
          };
        return {
          label: "更新密钥",
          tone: "warning",
          description: "该节点密钥已更新，密钥内容已脱敏。",
        };
      }
      const providerActions: Record<
        string,
        (afterValue: string) => AuditBusinessAction
      > = {
        enabled: (value) =>
          value === "true"
            ? {
                label: "启用节点",
                tone: "success",
                description: "该 Provider 节点已启用，并参与当前场景调度。",
              }
            : {
                label: "停用节点",
                tone: "neutral",
                description: "该 Provider 节点已停用，不再参与当前场景调度。",
              },
        priority: () => ({
          label: "调整顺序",
          tone: "neutral",
          description: "节点调度顺序已更新。",
        }),
        name: () => ({
          label: "重命名节点",
          tone: "primary",
          description: "节点名称已更新，页面展示、测试结果和审计标识会使用新名称。",
        }),
        model: () => ({
          label: "修改 Model",
          tone: "primary",
          description: "该节点请求模型已更新。",
        }),
        baseURL: () => ({
          label: "修改地址",
          tone: "warning",
          description: "该节点 Base URL 已更新。",
        }),
        timeout: () => ({
          label: "修改超时",
          tone: "neutral",
          description: "该节点请求超时时间已更新。",
        }),
      };
      if (providerActions[key]) return providerActions[key](after[key]);
    }
    if (providerRecord)
      return {
        label: "修改配置",
        tone: "primary",
        description: `该节点配置已更新，以下 ${changes.length} 项字段发生变化。`,
      };

    if (keys.length === 1) {
      const key = keys[0];
      if (key === "enabled")
        return after.enabled === "true"
          ? {
              label: "启用场景",
              tone: "success",
              description: "当前场景已启用新路由配置。",
            }
          : {
              label: "停用场景",
              tone: "neutral",
              description: "当前场景已停用新路由配置。",
            };
      const sceneActions: Record<string, AuditBusinessAction> = {
        strategy: {
          label: "修改调度策略",
          tone: "primary",
          description: "当前场景已使用新的 Provider 调度策略。",
        },
        maxAttempts: {
          label: "修改重试次数",
          tone: "primary",
          description: "当前场景最大尝试次数已更新。",
        },
        providers: {
          label: "节点数量变化",
          tone:
            Number(after.providers) > Number(before.providers)
              ? "primary"
              : "danger",
          description: "当前场景的 Provider 节点数量已变化。",
        },
        breaker: {
          label: "修改熔断策略",
          tone: "warning",
          description: "当前场景熔断阈值或冷却时间已更新。",
        },
      };
      if (sceneActions[key]) return sceneActions[key];
    }
    if (keys.includes("providers"))
      return {
        label: "调整场景策略",
        tone: "primary",
        description: "场景策略和 Provider 数量已同步更新。",
      };
    if (changes.length)
      return {
        label: "调整场景策略",
        tone: "primary",
        description: `场景策略已更新，以下 ${changes.length} 项配置发生变化。`,
      };
    return {
      label: displayAuditAction(record.action),
      tone: "neutral",
      description: auditFallbackSummary(record),
    };
  }

  function toneForAuditAction(record: SettingAuditRecord) {
    return auditBusinessAction(record).tone;
  }

  function auditEventLabel(record: SettingAuditRecord) {
    if (record.action === "test") return "测试执行";
    if (record.action === "update" || record.action === "save")
      return "保存发布";
    return displayAuditAction(record.action);
  }

  function auditDiffTitle(record: SettingAuditRecord) {
    return `${auditTargetTitle(record)} · ${auditBusinessAction(record).label}`;
  }

  function auditDiffStatusText(record: SettingAuditRecord) {
    const changes = auditChangeSummary(record);
    if (!changes.length) return auditBusinessAction(record).label;
    return auditChangeStats(record)
      .map((item) => `${item.label} ${item.count}`)
      .join(" · ");
  }

  return {
    auditBusinessAction,
    auditDiffStatusText,
    auditDiffTitle,
    auditEventLabel,
    auditFallbackSummary,
    auditTargetTitle,
    displayRouteTestMessage,
    toneForAuditAction,
  };
}
