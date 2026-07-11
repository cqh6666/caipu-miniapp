import {
  createAIProviderAuditPresenter,
} from "../src/utils/ai-provider-audit";
import {
  buildScenePayload,
  hydrateScene,
} from "../src/utils/ai-provider-draft";
import {
  buildProviderValidationErrors,
  buildSceneBlockingValidationMessages,
  providerHasUsableSecret,
} from "../src/utils/ai-provider-validation";
import { resolveAlertStatus } from "../src/utils/ai-provider-alerts";
import type {
  AIRoutingProviderConfig,
  AIRoutingSceneConfig,
  SettingAuditRecord,
} from "../src/types";

function assertEqual<T>(actual: T, expected: T, label: string) {
  if (actual !== expected) {
    throw new Error(`${label}: expected ${String(expected)}, got ${String(actual)}`);
  }
}

function auditRecord(
  oldValueMasked: string,
  newValueMasked: string,
  action = "update",
): SettingAuditRecord {
  return {
    id: 1,
    settingGroup: "ai.routing.summary",
    settingKey: "ai.routing.summary.provider.provider-a",
    action,
    oldValueMasked,
    newValueMasked,
    operatorSubject: "test",
    requestId: "req-test",
    createdAt: "2026-07-11T00:00:00Z",
  };
}

const presenter = createAIProviderAuditPresenter({
  currentScene: () => "summary",
  providerName: (id) => (id === "provider-a" ? "主节点" : id),
});

assertEqual(
  presenter.auditBusinessAction(
    auditRecord("", "name=主节点 enabled=true model=gpt-test"),
  ).label,
  "新增节点",
  "新增 Provider 审计",
);
assertEqual(
  presenter.auditBusinessAction(
    auditRecord("name=主节点 enabled=true model=gpt-test", ""),
  ).label,
  "删除节点",
  "删除 Provider 审计",
);
assertEqual(
  presenter.auditBusinessAction(
    auditRecord("model=old", "model=new"),
  ).label,
  "修改 Model",
  "修改 Provider 审计",
);
assertEqual(
  presenter.displayRouteTestMessage("ok via provider-a"),
  "测试通过，经由 主节点",
  "路由测试结果中文化",
);

assertEqual(
  resolveAlertStatus({ thresholdReached: true } as never),
  "active",
  "旧告警字段兼容",
);

const provider: AIRoutingProviderConfig = {
  id: "provider-a",
  scene: "summary",
  name: "主节点",
  adapter: "openai-compatible",
  enabled: true,
  priority: 10,
  baseURL: "https://api.example.com/v1",
  apiKey: "",
  apiKeyMasked: "***",
  hasAPIKey: true,
  clearApiKey: false,
  model: "gpt-test",
  timeoutSeconds: 30,
  endpointMode: "chat_completions",
  responseFormat: "auto",
  extra: {},
};
const scene: AIRoutingSceneConfig = {
  scene: "summary",
  enabled: true,
  strategy: "priority_failover",
  maxAttempts: 3,
  retryOn: ["timeout"],
  breaker: { failureThreshold: 3, cooldownSeconds: 60 },
  requestOptions: { stream: false, temperature: 0, maxTokens: 0 },
  providers: [provider],
};
assertEqual(providerHasUsableSecret(provider), true, "保留已保存密钥");
const payload = buildScenePayload(hydrateScene(scene));
assertEqual(payload.maxAttempts, 1, "最大尝试次数按启用节点数截断");
assertEqual(payload.providers[0].apiKey, "", "未输入新密钥时不回传明文");
assertEqual(payload.providers[0].hasAPIKey, true, "保留已保存密钥状态");

const invalidScene = {
  ...scene,
  maxAttempts: 1,
  providers: [provider, { ...provider, id: "provider-b", name: "备用节点" }],
};
assertEqual(
  buildSceneBlockingValidationMessages(invalidScene).length,
  1,
  "多节点失败切换校验",
);
assertEqual(
  Object.keys(
    buildProviderValidationErrors(
      { ...scene, providers: [{ ...provider, baseURL: "not-a-url" }] },
      (item) => item.id,
    ),
  ).length,
  1,
  "Provider 地址校验",
);

console.log("AI Provider utils checks passed");
