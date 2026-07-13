import { formatDiffValue } from "@/utils/ai-provider-audit";
import type { SceneDiffItem } from "@/utils/ai-provider-draft";

export type SaveConfirmDiffKind =
  | "added"
  | "removed"
  | "changed"
  | "secret"
  | "warning";

export type SaveConfirmDiffRow = {
  kind: SaveConfirmDiffKind;
  tag: string;
  title: string;
  description: string;
  before?: string;
  after?: string;
  beforeLabel?: string;
  afterLabel?: string;
};

export type SaveSceneConfirmModel = {
  sceneTitle: string;
  diffCount: number;
  rows: SaveConfirmDiffRow[];
  moreCount: number;
  test: {
    kind: SaveConfirmDiffKind;
    text: string;
    notice: string;
    role: "status" | "alert";
  };
};

type BuildSaveSceneConfirmOptions = {
  sceneTitle: string;
  diffItems: SceneDiffItem[];
  testing: boolean;
  testingElapsedSeconds?: number;
  testPassed?: boolean | null;
  providerName?: (providerId: string) => string;
  maxRows?: number;
};

const fieldLabels: Record<string, string> = {
  enabled: "启用状态",
  strategy: "调度策略",
  maxAttempts: "最大尝试次数",
  retryOn: "重试条件",
  breaker: "熔断策略",
  requestOptions: "请求参数",
  order: "Provider 顺序",
  name: "展示名称",
  adapter: "适配器",
  baseURL: "Base URL",
  model: "Model",
  timeoutSeconds: "超时时间",
  endpointMode: "接口模式",
  responseFormat: "响应格式",
  apiKey: "密钥",
  clearApiKey: "密钥清空标记",
};

const kindTags: Record<SaveConfirmDiffKind, string> = {
  added: "新增",
  removed: "删除",
  changed: "修改",
  secret: "密钥",
  warning: "顺序",
};

function diffKind(item: SceneDiffItem): SaveConfirmDiffKind {
  if (item.path === "added") return "added";
  if (item.path === "removed") return "removed";
  if (item.path === "apiKey" || item.path === "clearApiKey") return "secret";
  if (item.scope === "providers" && item.path === "order") return "warning";
  return "changed";
}

function providerSnapshotTitle(
  value: unknown,
  fallback: string,
  providerName: (providerId: string) => string,
) {
  if (!value || typeof value !== "object") return providerName(fallback);
  const item = value as Record<string, unknown>;
  const name = String(item.name || "").trim();
  const id = String(item.id || fallback).trim();
  return name || providerName(id);
}

function secretStateLabel(path: string, value: unknown) {
  if (path === "clearApiKey") {
    return value === true || String(value) === "true"
      ? "已标记清空"
      : "保留原密钥";
  }
  if (value === null || value === undefined || value === "" || value === false)
    return "未录入新密钥";
  const text = String(value).replace(/^\[|\]$/g, "");
  return text === "已录入" || text === "true" ? "已录入新密钥" : text;
}

export function formatSaveConfirmDiffItem(
  item: SceneDiffItem,
  providerName: (providerId: string) => string = (providerId) =>
    providerId || "未命名节点",
): SaveConfirmDiffRow {
  const kind = diffKind(item);
  const providerId = item.scope.startsWith("provider:")
    ? item.scope.replace("provider:", "")
    : "";
  if (kind === "added" || kind === "removed") {
    const title = providerSnapshotTitle(
      kind === "added" ? item.to : item.from,
      providerId,
      providerName,
    );
    return {
      kind,
      tag: kindTags[kind],
      title,
      description: kind === "added" ? "Provider 将新增。" : "Provider 将移除。",
    };
  }
  if (item.scope === "providers" && item.path === "order") {
    return {
      kind,
      tag: kindTags[kind],
      title: "Provider 顺序",
      description: "调度顺序将调整。",
      before: formatDiffValue(item.from),
      after: formatDiffValue(item.to),
    };
  }
  const field = fieldLabels[item.path] || item.path;
  if (providerId) {
    const title = providerSnapshotTitle(item.to || item.from, providerId, providerName);
    if (kind === "secret") {
      return {
        kind,
        tag: kindTags[kind],
        title,
        description:
          item.path === "clearApiKey"
            ? "密钥清空状态将更新，内容已脱敏。"
            : "密钥将更新，内容已脱敏。",
        before: secretStateLabel(item.path, item.from),
        after: secretStateLabel(item.path, item.to),
        beforeLabel: "当前",
        afterLabel: "发布后",
      };
    }
    return {
      kind,
      tag: kindTags[kind],
      title,
      description: `${field} 将更新。`,
      before: formatDiffValue(item.from),
      after: formatDiffValue(item.to),
      beforeLabel: "当前",
      afterLabel: "发布后",
    };
  }
  return {
    kind,
    tag: kindTags[kind],
    title: `场景${field}`,
    description: "配置将更新。",
    before: formatDiffValue(item.from),
    after: formatDiffValue(item.to),
    beforeLabel: "当前",
    afterLabel: "发布后",
  };
}

function describeTestState(
  testing: boolean,
  elapsedSeconds: number,
  testPassed: boolean | null | undefined,
): SaveSceneConfirmModel["test"] {
  if (testing) {
    return {
      kind: "changed",
      text: `测试中 · ${elapsedSeconds}s`,
      notice: "测试仍在进行，建议等待结果后发布。",
      role: "alert",
    };
  }
  if (testPassed === null || testPassed === undefined) {
    return {
      kind: "warning",
      text: "未测试",
      notice: "当前草稿未测试，建议先测试。",
      role: "alert",
    };
  }
  if (testPassed) {
    return {
      kind: "added",
      text: "测试通过",
      notice: "最近测试通过，可以发布。",
      role: "status",
    };
  }
  return {
    kind: "removed",
    text: "测试异常",
    notice: "最近测试异常，建议先排查后发布。",
    role: "alert",
  };
}

export function buildSaveSceneConfirmModel(
  options: BuildSaveSceneConfirmOptions,
): SaveSceneConfirmModel {
  const maxRows = Math.max(options.maxRows ?? 5, 0);
  const providerName = options.providerName || ((providerId: string) => providerId);
  const rows = options.diffItems
    .slice(0, maxRows)
    .map((item) => formatSaveConfirmDiffItem(item, providerName));
  return {
    sceneTitle: options.sceneTitle,
    diffCount: options.diffItems.length,
    rows,
    moreCount: Math.max(options.diffItems.length - rows.length, 0),
    test: describeTestState(
      options.testing,
      options.testingElapsedSeconds || 0,
      options.testPassed,
    ),
  };
}
