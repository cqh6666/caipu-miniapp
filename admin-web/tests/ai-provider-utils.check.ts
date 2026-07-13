import { computed, ref } from "vue";
import { useAIRoutingAlerts } from "../src/composables/useAIRoutingAlerts";
import { useAIProviderEditor } from "../src/composables/useAIProviderEditor";
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
import {
  buildSaveSceneConfirmModel,
  formatSaveConfirmDiffItem,
} from "../src/utils/ai-provider-save-confirm";
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

const alertItem = {
  providerId: "provider-1",
  providerName: "节点一",
  scene: "summary",
  alertStatus: "active",
  canRetest: true,
  canArchive: true,
  canMute: true,
  canUnmute: false,
};
const alertOverview = ref<any>({
  enabled: true,
  hasDeliveryConfig: true,
  generatedAt: "2026-07-14T00:00:00Z",
  items: [alertItem],
});
const routed: any[] = [];
const alertMessages: string[] = [];
let muteCallCount = 0;
const alertController = useAIRoutingAlerts({
  currentSceneKey: ref<any>("summary"),
  alertOverview,
  router: { push: async (target: any) => { routed.push(target); } } as any,
  extractMessage: (error) => String((error as Error)?.message || error),
  confirm: async () => "confirm" as any,
  messages: {
    success: ((message: any) => { alertMessages.push(`success:${message}`); }) as any,
    warning: ((message: any) => { alertMessages.push(`warning:${message}`); }) as any,
    error: ((message: any) => { alertMessages.push(`error:${message}`); }) as any,
    info: ((message: any) => { alertMessages.push(`info:${message}`); }) as any,
  },
  clipboard: { writeText: async () => undefined },
  api: {
    retestAIRoutingAlert: async () => ({ result: { ok: true, message: "复测完成", overview: alertOverview.value } }) as any,
    archiveAIRoutingAlert: async () => ({ result: { ok: true, message: "归档完成", overview: alertOverview.value } }) as any,
    muteAIRoutingAlert: async () => {
      muteCallCount += 1;
      return { result: { ok: true, message: "静默完成", overview: alertOverview.value } } as any;
    },
    unmuteAIRoutingAlert: async () => ({ result: { ok: true, message: "解除静默", overview: alertOverview.value } }) as any,
  },
});
assertEqual(alertController.alertStatusSummary.value.tone, "danger", "告警 composable 场景聚合");
assertEqual(alertController.currentSceneAlertSections.value[0]?.items.length, 1, "告警 composable 分段");
await alertController.handleAlertMute(alertItem as any);
assertEqual(muteCallCount, 1, "告警 composable 静默动作");
assertEqual(Object.keys(alertController.pendingAlertActions.value).length, 0, "告警动作完成后清理 pending");
alertController.goAlertProviderLogs(alertItem as any);
assertEqual(routed[0]?.path, "/ai-calls", "告警日志跳转");
await alertController.copyRequestId("request-1");
assertEqual(alertMessages.at(-1), "success:已复制 requestId", "告警 requestId 复制反馈");

const editorScene = ref<any>({
  scene: "summary",
  providers: [
    { id: "a", name: "A", endpointMode: "chat_completions", responseFormat: "auto", apiKey: "", clearApiKey: false, hasAPIKey: false, extra: {} },
    { id: "b", name: "B", endpointMode: "chat_completions", responseFormat: "auto", apiKey: "", clearApiKey: false, hasAPIKey: false, extra: {} },
  ],
});
const editorErrors = ref<Record<string, any>>({});
const providerEditor = useAIProviderEditor({
  draftScene: editorScene,
  validationErrors: computed(() => editorErrors.value),
  formatDuration: (value) => `${value || 0} ms`,
  formatTestMessage: (value) => String(value || ""),
  displayCallStatus: (value) => String(value || ""),
  confirm: async () => "confirm" as any,
  messages: { info: (() => undefined) as any, warning: (() => undefined) as any },
});
const firstProviderKey = providerEditor.getProviderLocalKey(editorScene.value.providers[0]);
assertEqual(providerEditor.getProviderLocalKey(editorScene.value.providers[0]), firstProviderKey, "Provider 本地 key 稳定");
editorErrors.value = { [firstProviderKey]: { name: "请填写名称" } };
assertEqual(providerEditor.providerFieldError(editorScene.value.providers[0], "name"), "", "Provider 未触碰不显示错误");
providerEditor.touchProviderField(editorScene.value.providers[0], "name");
assertEqual(providerEditor.providerFieldError(editorScene.value.providers[0], "name"), "请填写名称", "Provider 触碰后显示错误");
providerEditor.handleProviderDragStart(0, { dataTransfer: { effectAllowed: "", setData: () => undefined } } as any);
providerEditor.handleProviderDrop(1);
assertEqual(editorScene.value.providers[0].id, "b", "Provider 拖拽排序");
providerEditor.handleProviderApiKeyInput(editorScene.value.providers[0], "secret");
assertEqual(editorScene.value.providers[0].clearApiKey, false, "Provider 录入密钥撤销清空标记");
providerEditor.recordProviderTestState({
  attempts: [{ providerId: "b", status: "success", latencyMs: 12 }],
} as any);
assertEqual(providerEditor.getProviderTestState(editorScene.value.providers[0])?.ok, true, "Provider 最近测试状态");

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

const providerName = (providerId: string) =>
  providerId === "provider-a" ? "主节点" : providerId;
assertEqual(
  formatSaveConfirmDiffItem(
    {
      scope: "provider:provider-new",
      path: "added",
      from: null,
      to: { id: "provider-new", name: "新增节点" },
    },
    providerName,
  ).title,
  "新增节点",
  "保存确认新增节点标题",
);
assertEqual(
  formatSaveConfirmDiffItem(
    {
      scope: "provider:provider-a",
      path: "removed",
      from: { id: "provider-a", name: "主节点" },
      to: null,
    },
    providerName,
  ).kind,
  "removed",
  "保存确认删除节点类型",
);
assertEqual(
  formatSaveConfirmDiffItem(
    {
      scope: "provider:provider-a",
      path: "clearApiKey",
      from: false,
      to: true,
    },
    providerName,
  ).after,
  "已标记清空",
  "保存确认密钥清空状态",
);
assertEqual(
  formatSaveConfirmDiffItem({
    scope: "providers",
    path: "order",
    from: ["provider-a", "provider-b"],
    to: ["provider-b", "provider-a"],
  }).kind,
  "warning",
  "保存确认节点顺序类型",
);
const saveConfirmModel = buildSaveSceneConfirmModel({
  sceneTitle: "正文总结",
  diffItems: [
    { scope: "scene", path: "enabled", from: false, to: true },
    { scope: "scene", path: "maxAttempts", from: 1, to: 2 },
  ],
  testing: false,
  testPassed: true,
  maxRows: 1,
});
assertEqual(saveConfirmModel.test.text, "测试通过", "保存确认测试通过状态");
assertEqual(saveConfirmModel.test.role, "status", "保存确认测试状态可访问性");
assertEqual(saveConfirmModel.moreCount, 1, "保存确认折叠剩余变更数");
assertEqual(
  buildSaveSceneConfirmModel({
    sceneTitle: "正文总结",
    diffItems: [],
    testing: false,
    testPassed: false,
  }).test.text,
  "测试异常",
  "保存确认测试异常状态",
);

console.log("AI Provider utils checks passed");
