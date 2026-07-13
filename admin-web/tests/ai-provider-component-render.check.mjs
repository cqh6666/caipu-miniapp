import { fileURLToPath } from "node:url";
import { createSSRApp, defineComponent, h, ref } from "vue";
import { renderToString } from "@vue/server-renderer";
import { createServer } from "vite";

const root = fileURLToPath(new URL("..", import.meta.url));
const server = await createServer({
  root,
  appType: "custom",
  logLevel: "error",
  server: { middlewareMode: true },
  ssr: { noExternal: ["element-plus"] },
});

function assertIncludes(html, expected, label) {
  if (!html.includes(expected)) {
    throw new Error(`${label}: SSR output did not include ${expected}`);
  }
}

try {
  const [
    { default: ProviderEditor },
    { default: SceneStrategyPanel },
    { default: SaveSceneConfirmContent },
    { default: AlertLifecyclePanel },
    { useAIRoutingAlerts },
    { buildSaveSceneConfirmModel },
    { ID_INJECTION_KEY, ZINDEX_INJECTION_KEY },
  ] = await Promise.all([
    server.ssrLoadModule("/src/components/ai-providers/ProviderEditor.vue"),
    server.ssrLoadModule("/src/components/ai-providers/SceneStrategyPanel.vue"),
    server.ssrLoadModule(
      "/src/components/ai-providers/SaveSceneConfirmContent.vue",
    ),
    server.ssrLoadModule(
      "/src/components/ai-providers/AlertLifecyclePanel.vue",
    ),
    server.ssrLoadModule("/src/composables/useAIRoutingAlerts.ts"),
    server.ssrLoadModule("/src/utils/ai-provider-save-confirm.ts"),
    server.ssrLoadModule("element-plus"),
  ]);

  async function render(component, props = {}) {
    const app = createSSRApp({ render: () => h(component, props) });
    app.provide(ID_INJECTION_KEY, { prefix: 1024, current: 0 });
    app.provide(ZINDEX_INJECTION_KEY, { current: 0 });
    return renderToString(app);
  }

  const provider = {
    id: "provider-a",
    scene: "summary",
    name: "主节点",
    adapter: "openai-compatible",
    enabled: true,
    priority: 10,
    weight: 100,
    baseURL: "https://api.example.com/v1",
    apiKey: "",
    apiKeyMasked: "***",
    hasAPIKey: true,
    clearApiKey: false,
    model: "gpt-test",
    timeoutSeconds: 30,
    endpointMode: "chat_completions",
    responseFormat: "auto",
    extra: { thinking_type: "auto", reasoning_effort: "" },
  };
  const draftScene = {
    scene: "summary",
    enabled: true,
    strategy: "round_robin_failover",
    maxAttempts: 2,
    retryOn: ["timeout"],
    breaker: { failureThreshold: 3, cooldownSeconds: 30 },
    requestOptions: { stream: false, temperature: 0, maxTokens: 0 },
    providers: [provider],
    updatedAt: "2026-07-14T00:00:00Z",
    updatedBySubject: "tester",
  };
  const providerHelpTips = {
    providerName: "节点名称说明",
    baseURL: "Base URL 说明",
    endpoint: "Endpoint 说明",
    responseFormat: "响应格式说明",
    thinkingType: "Thinking 说明",
    reasoningEffort: "Reasoning Effort 说明",
    imageSize: "图片尺寸说明",
    imageBackground: "图片背景说明",
    imageCompression: "图片压缩说明",
    apiKey: "API Key 说明",
  };
  const noOp = () => undefined;
  const providerHtml = await render(ProviderEditor, {
    draftScene,
    enabledProviderCount: 1,
    helpTips: providerHelpTips,
    providerPresetOptions: [],
    singleTestProviderId: "",
    testingScene: false,
    savingScene: false,
    selectOptions: {
      endpointModes: [],
      responseFormats: [],
      thinkingTypes: [],
      reasoningEfforts: [],
      imageSizes: [],
      imageQualities: [],
      imageBackgrounds: [],
      imageOutputFormats: [],
    },
    editor: {
      draggingProviderIndex: null,
      dragOverProviderIndex: null,
      getProviderLocalKey: (item) => item.id,
      isProviderCollapsed: () => false,
      firstProviderError: () => "",
      getProviderTestState: () => undefined,
      providerFieldError: () => "",
      isImageGenerationProvider: () => false,
      providerThinkingLabel: () => "",
      shouldShowProviderSecretEditor: () => false,
      add: noOp,
      addPreset: noOp,
      dragOver: noOp,
      drop: noOp,
      dragStart: noOp,
      dragEnd: noOp,
      toggleCollapse: noOp,
      testSingle: noOp,
      menu: noOp,
      touchField: noOp,
      endpointChange: noOp,
      thinkingChange: noOp,
      toggleSecret: noOp,
      clearSecret: noOp,
      apiKeyInput: noOp,
    },
  });
  assertIncludes(providerHtml, "Provider 节点", "ProviderEditor contract");
  assertIncludes(providerHtml, "主节点", "ProviderEditor first provider");
  assertIncludes(providerHtml, "Base URL", "ProviderEditor first provider expanded");

  const sceneStrategyHtml = await render(SceneStrategyPanel, {
    draftScene,
    currentChannel: { tone: "success", label: "新路由" },
    helpTips: {
      sceneStrategy: "策略说明",
      maxAttempts: "尝试次数说明",
      breaker: "熔断说明",
      requestOptions: "请求参数说明",
    },
    minimumMaxAttempts: 1,
    maxAttemptCeiling: 3,
    numericWarn: {
      maxAttempts: "",
      failureThreshold: "",
      cooldownSeconds: "",
    },
    timelineSegments: { attempts: 2 },
    expectedFirstRoundSeconds: 60,
    retryOptions: [{ label: "超时 timeout", value: "timeout" }],
  });
  assertIncludes(sceneStrategyHtml, "场景策略", "SceneStrategyPanel title");
  assertIncludes(sceneStrategyHtml, "预计首轮最长", "SceneStrategyPanel timeline");

  const saveConfirmHtml = await render(SaveSceneConfirmContent, {
    model: buildSaveSceneConfirmModel({
      sceneTitle: "正文总结",
      diffItems: [
        {
          scope: "provider:provider-a",
          path: "clearApiKey",
          from: false,
          to: true,
        },
      ],
      testing: false,
      testPassed: false,
      providerName: () => "主节点",
    }),
  });
  assertIncludes(saveConfirmHtml, "保存后立即更新线上", "Save confirm lead");
  assertIncludes(saveConfirmHtml, "已标记清空", "Save confirm secret state");
  assertIncludes(saveConfirmHtml, "测试异常", "Save confirm test state");

  const generatedAt = "2026-07-14T00:00:00Z";
  const overview = {
    generatedAt,
    enabled: true,
    failureThreshold: 3,
    activeWindowHours: 24,
    hasDeliveryConfig: true,
    activeAlertCount: 1,
    staleAlertCount: 0,
    pendingVerifyCount: 0,
    mutedAlertCount: 0,
    archivedAlertCount: 0,
    recoveredCount: 0,
    reviewAlertCount: 0,
    latestAlertedAt: generatedAt,
    items: [
      {
        providerId: "provider-a",
        providerName: "主节点",
        scene: "summary",
        model: "gpt-test",
        consecutiveFailures: 3,
        lastStatus: "failed",
        lastErrorType: "timeout",
        lastErrorMessage: "upstream timeout",
        lastRequestId: "req-1",
        lastFailedAt: generatedAt,
        lastRecoveredAt: "",
        lastAlertedAt: generatedAt,
        updatedAt: generatedAt,
        alertStatus: "active",
        statusReason: "连续失败达到阈值",
        isProviderEnabled: true,
        isInEffectiveRoute: true,
        canRetest: true,
        canArchive: true,
        canMute: true,
        canUnmute: false,
        thresholdReached: true,
      },
    ],
  };
  const AlertHarness = defineComponent({
    setup() {
      const alertOverview = ref(overview);
      const controller = useAIRoutingAlerts({
        currentSceneKey: ref("summary"),
        alertOverview,
        router: { push: async () => undefined },
        extractMessage: (error) => String(error),
        confirm: async () => "confirm",
        messages: {
          success: noOp,
          warning: noOp,
          error: noOp,
          info: noOp,
        },
        clipboard: { writeText: async () => undefined },
        api: {
          retestAIRoutingAlert: async () => ({
            result: { ok: true, message: "ok", overview },
          }),
          archiveAIRoutingAlert: async () => ({
            result: { ok: true, message: "ok", overview },
          }),
          muteAIRoutingAlert: async () => ({
            result: { ok: true, message: "ok", overview },
          }),
          unmuteAIRoutingAlert: async () => ({
            result: { ok: true, message: "ok", overview },
          }),
        },
      });
      return () =>
        h(AlertLifecyclePanel, {
          sceneTitle: "正文总结",
          description: controller.alertStatusDescription.value,
          overview: alertOverview.value,
          sections: controller.currentSceneAlertSections.value,
          hasNodes: controller.hasCurrentSceneAlertNodes.value,
          pendingActions: controller.pendingAlertActions.value,
          onRetest: controller.handleAlertRetest,
          onArchive: controller.handleAlertArchive,
          onMute: controller.handleAlertMute,
          onUnmute: controller.handleAlertUnmute,
          onBatchRetest: controller.handleBatchRetest,
          onBatchArchive: controller.handleBatchArchive,
          onLogs: controller.goAlertProviderLogs,
          onConfig: controller.goAlertConfig,
          onCopyRequestId: controller.copyRequestId,
        });
    },
  });
  const alertHtml = await render(AlertHarness);
  assertIncludes(alertHtml, "连续异常告警", "Alert composable assembly");
  assertIncludes(alertHtml, "主节点", "Alert composable active provider");

  console.log("AI Provider component render checks passed");
} finally {
  await server.close();
}
