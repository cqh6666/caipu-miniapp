import { fileURLToPath } from "node:url";
import { createSSRApp, h } from "vue";
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

try {
  const { default: ProviderEditor } = await server.ssrLoadModule(
    "/src/components/ai-providers/ProviderEditor.vue",
  );
  const { ID_INJECTION_KEY, ZINDEX_INJECTION_KEY } =
    await server.ssrLoadModule("element-plus");
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
  };
  const helpTips = {
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
  const app = createSSRApp({
    render: () =>
      h(ProviderEditor, {
        draftScene,
        enabledProviderCount: 1,
        helpTips,
        providerPresetOptions: [],
        draggingProviderIndex: null,
        dragOverProviderIndex: null,
        singleTestProviderId: "",
        testingScene: false,
        savingScene: false,
        providerEndpointModeOptions: [],
        providerResponseFormatOptions: [],
        thinkingTypeOptions: [],
        reasoningEffortOptions: [],
        imageSizeOptions: [],
        imageQualityOptions: [],
        imageBackgroundOptions: [],
        imageOutputFormatOptions: [],
        getProviderLocalKey: (item) => item.id,
        isProviderCollapsed: () => false,
        firstProviderError: () => "",
        getProviderTestState: () => undefined,
        providerFieldError: () => "",
        isImageGenerationProvider: () => false,
        providerThinkingLabel: () => "",
        shouldShowProviderSecretEditor: () => false,
      }),
  });
  app.provide(ID_INJECTION_KEY, { prefix: 1024, current: 0 });
  app.provide(ZINDEX_INJECTION_KEY, { current: 0 });
  const html = await renderToString(app);
  if (!html.includes("Provider 节点") || !html.includes("主节点")) {
    throw new Error("ProviderEditor SSR render did not include provider content");
  }
  console.log("AI Provider component render check passed");
} finally {
  await server.close();
}
