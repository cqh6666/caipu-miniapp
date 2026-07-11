import type {
  AIRoutingAlertOverview,
  AIRoutingAlertOverviewItem,
  AIRoutingAlertStatus,
  AIRoutingSceneKey,
  AIRoutingSceneSummary,
  SceneCardHealthSnapshot,
} from "@/types";
import { formatDateTime } from "@/utils/admin-display";

export type SceneCardIssue = {
  tone: "danger" | "warning";
  title: string;
  detail: string;
};

const alertErrorTypeLabelMap: Record<string, string> = {
  timeout: "请求超时",
  failed: "调用失败",
  http_status: "网关状态异常",
  http_error: "网关错误",
  connection: "连接失败",
  network: "网络异常",
  rate_limit: "触发限流",
  invalid_response: "响应异常",
};

const alertStatusLabelMap: Record<AIRoutingAlertStatus, string> = {
  normal: "正常",
  active: "告警中",
  stale: "待复测（已过期）",
  pending_verify: "待复测（配置变更）",
  muted: "已静默",
  archived: "已归档",
  recovered: "已恢复",
};

function parseTimeValue(value?: string) {
  const timestamp = Date.parse(String(value || "").trim());
  return Number.isFinite(timestamp) ? timestamp : null;
}

export function displayAlertErrorType(value?: string) {
  const key = String(value || "")
    .trim()
    .toLowerCase();
  if (!key) {
    return "";
  }
  return alertErrorTypeLabelMap[key] || value || "";
}

export function formatAlertRelativeTime(value: string, reference?: string) {
  const target = parseTimeValue(value);
  if (target === null) {
    return "";
  }
  const referenceTime = parseTimeValue(reference) ?? Date.now();
  const diff = referenceTime - target;
  if (diff < 0) {
    return formatDateTime(value);
  }
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) {
    return "刚刚";
  }
  if (minutes < 60) {
    return `${minutes} 分钟前`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours} 小时前`;
  }
  const days = Math.floor(hours / 24);
  if (days < 30) {
    return `${days} 天前`;
  }
  return formatDateTime(value);
}

export function isWithinLast24Hours(value: string, reference?: string) {
  const targetTime = parseTimeValue(value);
  if (targetTime === null) {
    return false;
  }
  const referenceTime = parseTimeValue(reference) ?? Date.now();
  return (
    referenceTime >= targetTime &&
    referenceTime - targetTime <= 24 * 60 * 60 * 1000
  );
}

// 兼容旧版 overview：后端尚未返回 alertStatus 时继续使用 thresholdReached。
export function resolveAlertStatus(
  item: AIRoutingAlertOverviewItem,
): AIRoutingAlertStatus {
  return item.alertStatus || (item.thresholdReached ? "active" : "normal");
}

export function alertStatusLabel(status: AIRoutingAlertStatus) {
  return alertStatusLabelMap[status] || status;
}

export function alertStatusTone(
  status: AIRoutingAlertStatus,
): "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case "active":
      return "danger";
    case "stale":
    case "pending_verify":
    case "muted":
      return "warning";
    case "archived":
      return "neutral";
    case "recovered":
      return "success";
    default:
      return "success";
  }
}

export function isReviewStatus(status: AIRoutingAlertStatus) {
  return (
    status === "stale" || status === "pending_verify" || status === "muted"
  );
}

export function summarizeSceneAlertStatus(
  scene: AIRoutingSceneKey,
  overview?: AIRoutingAlertOverview | null,
): SceneCardHealthSnapshot["alertStatus"] {
  if (!overview) {
    return {
      tone: "neutral",
      text: "加载中",
    };
  }
  const sceneItems = overview.items.filter((item) => item.scene === scene);
  const activeCount = sceneItems.filter(
    (item) => resolveAlertStatus(item) === "active",
  ).length;
  const reviewCount = sceneItems.filter((item) =>
    isReviewStatus(resolveAlertStatus(item)),
  ).length;
  if (activeCount > 0) {
    return {
      tone: "danger",
      text:
        reviewCount > 0
          ? `告警中 ${activeCount} · 待复核 ${reviewCount}`
          : `告警中 ${activeCount}`,
    };
  }
  if (reviewCount > 0) {
    return {
      tone: "warning",
      text: `待复核 ${reviewCount}`,
    };
  }
  return {
    tone: "success",
    text: "无告警",
  };
}

export function findSceneActiveAlertItems(
  scene: AIRoutingSceneKey,
  overview?: AIRoutingAlertOverview | null,
) {
  if (!overview) {
    return [];
  }
  return overview.items
    .filter(
      (item) => item.scene === scene && resolveAlertStatus(item) === "active",
    )
    .sort((left, right) => right.consecutiveFailures - left.consecutiveFailures);
}

export function findSceneReviewAlertItems(
  scene: AIRoutingSceneKey,
  overview?: AIRoutingAlertOverview | null,
) {
  if (!overview) {
    return [];
  }
  return overview.items
    .filter(
      (item) =>
        item.scene === scene && isReviewStatus(resolveAlertStatus(item)),
    )
    .sort((left, right) => right.consecutiveFailures - left.consecutiveFailures);
}

export function summarizeSceneIssue(
  scene: AIRoutingSceneKey,
  health: SceneCardHealthSnapshot,
  summary?: AIRoutingSceneSummary,
  overview?: AIRoutingAlertOverview | null,
): SceneCardIssue | null {
  if (!summary || summary.providerCount === 0) {
    return null;
  }
  if (health.alertStatus.tone === "danger") {
    const items = findSceneActiveAlertItems(scene, overview);
    const reviewCount = findSceneReviewAlertItems(scene, overview).length;
    const top = items[0];
    if (top) {
      const parts: string[] = [];
      const errorType = displayAlertErrorType(top.lastErrorType);
      if (errorType) {
        parts.push(errorType);
      }
      const relative = formatAlertRelativeTime(
        top.lastFailedAt,
        overview?.generatedAt,
      );
      if (relative) {
        parts.push(relative);
      }
      if (items.length > 1) {
        parts.push(`共 ${items.length} 个节点告警`);
      }
      if (reviewCount > 0) {
        parts.push(`另有 ${reviewCount} 个待复核`);
      }
      return {
        tone: "danger",
        title: `${top.providerName} 连续失败 ${top.consecutiveFailures} 次`,
        detail: parts.join(" · "),
      };
    }
    return {
      tone: "danger",
      title: "节点告警中",
      detail: "有启用节点连续失败超阈值",
    };
  }
  if (health.configRisk.tone === "danger") {
    return {
      tone: "danger",
      title: "无启用节点",
      detail: "至少启用 1 个节点才能发布",
    };
  }
  if (health.configRisk.tone === "warning") {
    return {
      tone: "warning",
      title: "启用节点缺密钥",
      detail: "有启用节点未配置可用密钥",
    };
  }
  if (health.alertStatus.tone === "warning") {
    const reviewItems = findSceneReviewAlertItems(scene, overview);
    const top = reviewItems[0];
    const relative = top
      ? formatAlertRelativeTime(top.lastFailedAt, overview?.generatedAt)
      : "";
    return {
      tone: "warning",
      title: `${reviewItems.length} 个待复核`,
      detail: relative
        ? `最近失败 ${relative} · 建议复测或归档`
        : "建议复测确认或归档，历史告警不计入红色",
    };
  }
  if (summary.compatibilityMode) {
    return {
      tone: "warning",
      title: "兼容模式待切换",
      detail: "线上仍走旧单 Provider，保存并启用后切换",
    };
  }
  return null;
}

export function sceneAggregateStatus(
  summary: AIRoutingSceneSummary | undefined,
  health: SceneCardHealthSnapshot,
) {
  if (!summary || summary.providerCount === 0) {
    return {
      tone: "neutral" as const,
      text: "未配置",
    };
  }
  if (health.alertStatus.tone === "danger") {
    return {
      tone: "danger" as const,
      text: "告警中",
    };
  }
  if (health.configRisk.tone === "danger") {
    return {
      tone: "danger" as const,
      text: "不可发布",
    };
  }
  if (health.alertStatus.tone === "warning") {
    return {
      tone: "warning" as const,
      text: "待复核",
    };
  }
  if (health.configRisk.tone === "warning") {
    return {
      tone: "warning" as const,
      text: "需关注",
    };
  }
  if (summary.compatibilityMode) {
    return {
      tone: "warning" as const,
      text: "兼容模式",
    };
  }
  if (!summary.enabled) {
    return {
      tone: "neutral" as const,
      text: "未启用",
    };
  }
  return {
    tone: "success" as const,
    text: "运行正常",
  };
}
