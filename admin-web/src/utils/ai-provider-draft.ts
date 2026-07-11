import type {
  AIRoutingProviderConfig,
  AIRoutingProviderEndpointMode,
  AIRoutingSceneConfig,
  AIRoutingSceneKey,
} from "@/types";

export type SceneDiffItem = {
  scope: string;
  path: string;
  from: unknown;
  to: unknown;
};

const defaultRetryOn = [
  "timeout",
  "network",
  "rate_limit",
  "auth",
  "upstream",
  "invalid_response",
];

export function normalizeThinkingType(value: unknown) {
  const raw = String(value || "").trim().toLowerCase();
  return raw === "enabled" || raw === "disabled" ? raw : "";
}

export function normalizeReasoningEffort(value: unknown) {
  const raw = String(value || "").trim().toLowerCase();
  return raw === "high" || raw === "max" ? raw : "";
}

export function normalizeImageOutputFormat(value: unknown) {
  const raw = String(value || "").trim().toLowerCase();
  if (raw === "jpg") return "jpeg";
  return raw === "jpeg" || raw === "webp" || raw === "png" ? raw : "";
}

export function normalizeProviderExtra(provider: AIRoutingProviderConfig) {
  const extra = { ...(provider.extra || {}) };
  if ((provider.endpointMode || "chat_completions") !== "images_generations") {
    delete extra.size;
    delete extra.quality;
    delete extra.background;
    delete extra.output_format;
    delete extra.output_compression;
    delete extra.n;
    const thinkingType = normalizeThinkingType(extra.thinking_type);
    extra.thinking_type = thinkingType || "auto";
    extra.reasoning_effort =
      thinkingType === "disabled"
        ? ""
        : normalizeReasoningEffort(extra.reasoning_effort);
    return extra;
  }

  delete extra.thinking_type;
  delete extra.reasoning_effort;
  const outputFormat = normalizeImageOutputFormat(extra.output_format) || "png";
  extra.output_format = outputFormat;
  extra.size = String(extra.size || "").trim() || "auto";
  extra.quality = String(extra.quality || "").trim() || "auto";
  extra.background = String(extra.background || "").trim() || "auto";
  const outputCompression = extra.output_compression as unknown;
  if (outputFormat === "png") {
    delete extra.output_compression;
  } else if (
    outputCompression === undefined ||
    outputCompression === null ||
    outputCompression === ""
  ) {
    extra.output_compression = 60;
  } else {
    extra.output_compression = Math.min(
      Math.max(Number(extra.output_compression) || 0, 0),
      100,
    );
  }
  const imageCount = extra.n as unknown;
  extra.n =
    imageCount === undefined || imageCount === null || imageCount === ""
      ? 1
      : Math.min(Math.max(Number(extra.n) || 1, 1), 10);
  return extra;
}

export function hydrateScene(scene: AIRoutingSceneConfig): AIRoutingSceneConfig {
  const clone = JSON.parse(JSON.stringify(scene)) as AIRoutingSceneConfig;
  clone.requestOptions ||= {
    stream: false,
    temperature: 0,
    maxTokens: clone.scene === "title" ? 64 : 0,
  };
  clone.breaker ||= { failureThreshold: 3, cooldownSeconds: 60 };
  clone.retryOn ||= [...defaultRetryOn];
  clone.providers = (clone.providers || []).map((provider, index) => {
    const endpointMode = (provider.endpointMode ||
      "chat_completions") as AIRoutingProviderEndpointMode;
    const normalized = {
      ...provider,
      adapter: provider.adapter || "openai-compatible",
      enabled: provider.enabled ?? true,
      priority: Number(provider.priority) || (index + 1) * 10,
      timeoutSeconds: Number(provider.timeoutSeconds) || 30,
      baseURL: provider.baseURL || "",
      name: provider.name || "",
      id: provider.id || "",
      hasAPIKey: !!provider.hasAPIKey,
      apiKeyMasked: provider.apiKeyMasked || "",
      endpointMode,
      responseFormat:
        endpointMode === "images_generations"
          ? provider.responseFormat || "auto"
          : "auto",
      apiKey: "",
      clearApiKey: false,
    } as AIRoutingProviderConfig;
    normalized.extra = normalizeProviderExtra(normalized);
    return normalized;
  });
  return clone;
}

export function buildScenePayload(
  scene: AIRoutingSceneConfig,
): AIRoutingSceneConfig {
  const payload = JSON.parse(JSON.stringify(scene)) as AIRoutingSceneConfig;
  payload.providers = payload.providers.map((provider, index) => ({
    ...provider,
    id: provider.id.trim(),
    name: provider.name.trim(),
    adapter: provider.adapter || "openai-compatible",
    baseURL: provider.baseURL.trim(),
    model: provider.model.trim(),
    priority: (index + 1) * 10,
    timeoutSeconds: Number(provider.timeoutSeconds) || 30,
    endpointMode: provider.endpointMode || "chat_completions",
    responseFormat:
      provider.endpointMode === "images_generations"
        ? provider.responseFormat || "auto"
        : "auto",
    extra: normalizeProviderExtra(provider),
    apiKey: (provider.apiKey || "").trim(),
    apiKeyMasked: provider.apiKeyMasked || "",
    hasAPIKey: !!provider.hasAPIKey,
    clearApiKey: !!provider.clearApiKey,
  }));
  payload.maxAttempts = Math.min(
    Math.max(Number(payload.maxAttempts) || 1, 1),
    Math.max(payload.providers.filter((item) => item.enabled).length, 1),
  );
  payload.breaker.failureThreshold = Math.max(
    Number(payload.breaker.failureThreshold) || 1,
    1,
  );
  payload.breaker.cooldownSeconds = Math.max(
    Number(payload.breaker.cooldownSeconds) || 5,
    5,
  );
  if (payload.scene !== "title") {
    payload.requestOptions.stream = false;
    payload.requestOptions.temperature = 0;
    payload.requestOptions.maxTokens = 0;
  }
  return payload;
}

export function comparableScene(scene: AIRoutingSceneConfig) {
  return JSON.stringify(buildScenePayload(scene));
}

export function createProvider(
  scene: AIRoutingSceneKey,
): AIRoutingProviderConfig {
  const seed = `${scene}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`;
  return {
    id: seed,
    scene,
    name: "",
    adapter: "openai-compatible",
    enabled: true,
    priority: 10,
    weight: 100,
    baseURL: "",
    apiKey: "",
    apiKeyMasked: "",
    hasAPIKey: false,
    clearApiKey: false,
    model: "",
    timeoutSeconds: scene === "flowchart" ? 120 : scene === "title" ? 5 : 30,
    endpointMode:
      scene === "flowchart" ? "images_generations" : "chat_completions",
    responseFormat: scene === "flowchart" ? "b64_json" : "auto",
    extra:
      scene === "flowchart"
        ? {
            size: "1536x1024",
            quality: "high",
            background: "opaque",
            output_format: "jpeg",
            output_compression: 60,
            n: 1,
          }
        : {},
  };
}

function diffProviderKey(provider: Record<string, unknown>, index: number) {
  return String(provider.id || "").trim() || `__index_${index}`;
}

function providerDiffSnapshot(provider: Record<string, unknown>) {
  return {
    id: String(provider.id || "").trim(),
    name: String(provider.name || "").trim(),
    adapter: String(provider.adapter || "").trim(),
    enabled: Boolean(provider.enabled),
    baseURL: String(provider.baseURL || "").trim(),
    model: String(provider.model || "").trim(),
    timeoutSeconds: Number(provider.timeoutSeconds) || 0,
    endpointMode: String(provider.endpointMode || "chat_completions").trim(),
    responseFormat: String(provider.responseFormat || "auto").trim(),
    clearApiKey: Boolean(provider.clearApiKey),
    apiKey:
      typeof provider.apiKey === "string" && provider.apiKey.trim()
        ? "[已录入]"
        : "",
  };
}

export function buildSceneDiff(
  draft: AIRoutingSceneConfig | null,
  remote: AIRoutingSceneConfig | null,
): SceneDiffItem[] {
  if (!draft || !remote) return [];
  const next = buildScenePayload(draft) as unknown as Record<string, unknown>;
  const prev = buildScenePayload(remote) as unknown as Record<string, unknown>;
  const results: SceneDiffItem[] = [];
  const sceneFieldKeys = [
    "enabled",
    "strategy",
    "maxAttempts",
    "retryOn",
    "breaker",
    "requestOptions",
  ] as const;
  for (const key of sceneFieldKeys) {
    if (JSON.stringify(next[key]) !== JSON.stringify(prev[key]))
      results.push({ scope: "scene", path: key, from: prev[key], to: next[key] });
  }
  const nextProviders =
    (next.providers as Array<Record<string, unknown>>) || [];
  const prevProviders =
    (prev.providers as Array<Record<string, unknown>>) || [];
  const nextEntries = nextProviders.map(
    (provider, index) => [diffProviderKey(provider, index), provider] as const,
  );
  const prevEntries = prevProviders.map(
    (provider, index) => [diffProviderKey(provider, index), provider] as const,
  );
  const nextMap = new Map(nextEntries);
  const prevMap = new Map(prevEntries);
  const nextKeys = nextEntries.map(([key]) => key);
  const prevKeys = prevEntries.map(([key]) => key);
  const sameProviderSet =
    nextKeys.length === prevKeys.length &&
    nextKeys.every((key) => prevMap.has(key));
  if (sameProviderSet && JSON.stringify(nextKeys) !== JSON.stringify(prevKeys))
    results.push({
      scope: "providers",
      path: "order",
      from: prevKeys,
      to: nextKeys,
    });
  for (const key of Array.from(new Set([...prevKeys, ...nextKeys]))) {
    const nextProvider = nextMap.get(key);
    const prevProvider = prevMap.get(key);
    if (!nextProvider || !prevProvider) {
      results.push({
        scope: `provider:${key}`,
        path: nextProvider ? "added" : "removed",
        from: prevProvider ? providerDiffSnapshot(prevProvider) : null,
        to: nextProvider ? providerDiffSnapshot(nextProvider) : null,
      });
      continue;
    }
    const nextSnapshot = providerDiffSnapshot(nextProvider);
    const prevSnapshot = providerDiffSnapshot(prevProvider);
    for (const field of Object.keys({ ...prevSnapshot, ...nextSnapshot })) {
      const keyName = field as keyof typeof nextSnapshot;
      if (
        JSON.stringify(nextSnapshot[keyName]) !==
        JSON.stringify(prevSnapshot[keyName])
      )
        results.push({
          scope: `provider:${key}`,
          path: field,
          from: prevSnapshot[keyName],
          to: nextSnapshot[keyName],
        });
    }
  }
  return results;
}
