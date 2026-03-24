const http = require("http");
const { createBilibiliProvider, isSupportedBilibiliUrl } = require("./providers/bilibili");
const { createImporterProvider } = require("./providers/importer");
const { createRednoteProvider } = require("./providers/rednote");
const { buildNormalized, isSupportedXHSUrl } = require("./lib/normalize");
const { enrichTranscriptIfNeeded } = require("./lib/transcript");

function getEnvBool(key, fallback) {
  const raw = String(process.env[key] || "").trim().toLowerCase();
  if (!raw) {
    return fallback;
  }
  return ["1", "true", "yes", "on"].includes(raw);
}

function readConfig() {
  return {
    port: Number(process.env.PORT || 8091),
    apiKey: String(process.env.LINKPARSE_INTERNAL_API_KEY || "").trim(),
    xiaohongshuDefaultProvider: String(process.env.XHS_PROVIDER_DEFAULT || "auto").trim().toLowerCase() || "auto",
    importerEnabled: getEnvBool("XHS_PROVIDER_IMPORTER_ENABLED", true),
    rednoteEnabled: getEnvBool("XHS_PROVIDER_REDNOTE_ENABLED", true),
    stubMode: String(process.env.XHS_SIDECAR_STUB_MODE || "echo").trim().toLowerCase() || "echo",
    rednoteCookiePath: String(process.env.XHS_REDNOTE_COOKIE_PATH || "").trim(),
    rednoteCookieHeader:
      String(process.env.XHS_REDNOTE_COOKIE_HEADER || process.env.XHS_REDNOTE_COOKIE || "").trim(),
    rednoteCookieDomain: String(process.env.XHS_REDNOTE_COOKIE_DOMAIN || ".xiaohongshu.com").trim() || ".xiaohongshu.com",
    rednoteBrowserHeadless: getEnvBool("XHS_BROWSER_HEADLESS", true),
    rednoteBrowserPath: String(process.env.XHS_REDNOTE_BROWSER_PATH || "").trim(),
    rednoteTimeoutMS: Number(process.env.XHS_REDNOTE_TIMEOUT_MS || 15000),
    transcriptEnabled: getEnvBool("XHS_TRANSCRIPT_ENABLED", false),
    transcriptProvider: String(process.env.XHS_TRANSCRIPT_PROVIDER || "siliconflow").trim().toLowerCase() || "siliconflow",
    transcriptAPIKey: String(process.env.XHS_TRANSCRIPT_API_KEY || "").trim(),
    transcriptModel: String(process.env.XHS_TRANSCRIPT_MODEL || "TeleAI/TeleSpeechASR").trim(),
    transcriptEndpoint: String(process.env.XHS_TRANSCRIPT_ENDPOINT || "").trim(),
    transcriptTimeoutMS: Number(process.env.XHS_TRANSCRIPT_TIMEOUT_MS || 120000),
    transcriptMaxVideoMB: Number(process.env.XHS_TRANSCRIPT_MAX_VIDEO_MB || 80),
    transcriptKeepTemp: getEnvBool("XHS_TRANSCRIPT_KEEP_TEMP", false),
    ffmpegPath: String(process.env.FFMPEG_PATH || "ffmpeg").trim() || "ffmpeg",
    bilibiliDefaultProvider: String(process.env.BILI_PROVIDER_DEFAULT || "auto").trim().toLowerCase() || "auto",
    bilibiliOpenAPIEnabled: getEnvBool("BILI_PROVIDER_OPENAPI_ENABLED", true)
  };
}

function sendJSON(res, status, payload) {
  res.writeHead(status, { "Content-Type": "application/json; charset=utf-8" });
  res.end(JSON.stringify(payload));
}

async function readJSON(req) {
  const chunks = [];
  for await (const chunk of req) {
    chunks.push(chunk);
  }
  const body = Buffer.concat(chunks).toString("utf8").trim();
  if (!body) {
    return {};
  }
  return JSON.parse(body);
}

function ensureAuthorized(req, config) {
  if (!config.apiKey) {
    return true;
  }
  const auth = String(req.headers.authorization || "").trim();
  return auth === `Bearer ${config.apiKey}`;
}

function buildProviders(config) {
  return {
    xiaohongshu: {
      importer: createImporterProvider(config),
      rednote: createRednoteProvider(config)
    },
    bilibili: {
      openapi: createBilibiliProvider(config)
    }
  };
}

function buildProviderOrder(platform, requestedProvider) {
  switch (platform) {
    case "xiaohongshu":
      switch (requestedProvider) {
        case "importer":
          return ["importer"];
        case "rednote":
          return ["rednote"];
        default:
          return ["importer", "rednote"];
      }
    case "bilibili":
      switch (requestedProvider) {
        case "openapi":
          return ["openapi"];
        default:
          return ["openapi"];
      }
    default:
      return [];
  }
}

function normalizeXiaohongshuContent(note) {
  return {
    title: String(note?.title || "").trim(),
    description: "",
    body: String(note?.content || "").trim(),
    part: "",
    transcript: String(note?.transcript || "").trim(),
    transcriptStatus: String(note?.transcriptStatus || "").trim(),
    transcriptError: String(note?.transcriptError || "").trim(),
    tags: Array.isArray(note?.tags) ? note.tags : [],
    images: Array.isArray(note?.images) ? note.images : [],
    videos: Array.isArray(note?.videos) ? note.videos : [],
    coverUrl: String(note?.coverUrl || "").trim(),
    author: {
      name: String(note?.author?.name || "").trim(),
      avatarUrl: String(note?.author?.avatarUrl || "").trim()
    },
    contentType: String(note?.noteType || "unknown").trim() || "unknown",
    likes: Number(note?.likes || 0) || 0,
    comments: Number(note?.comments || 0) || 0,
    favorites: Number(note?.favorites || 0) || 0,
    subtitleLanguage: "",
    subtitleSegments: 0
  };
}

function normalizeXiaohongshuResponse(result, input, providerRequested, providerUsed, includeTranscript, config) {
  const note = result.note || {};
  return {
    ok: true,
    platform: "xiaohongshu",
    providerRequested,
    providerUsed,
    normalized: {
      shareUrl: result.normalized?.shareUrl || buildNormalized(input)?.shareUrl || "",
      canonicalUrl: result.normalized?.canonicalUrl || buildNormalized(input)?.canonicalUrl || "",
      id: result.normalized?.noteId || "",
      xsecToken: result.normalized?.xsecToken || ""
    },
    getContent: async () => {
      const enriched = includeTranscript ? await enrichTranscriptIfNeeded(note, config) : note;
      return normalizeXiaohongshuContent(enriched);
    },
    warnings: result.warnings || [],
    quality: result.quality || "full"
  };
}

function normalizeBilibiliResponse(result, providerRequested, providerUsed) {
  return {
    ok: true,
    platform: "bilibili",
    providerRequested,
    providerUsed,
    normalized: result.normalized || {
      shareUrl: "",
      canonicalUrl: "",
      id: ""
    },
    getContent: async () => result.content || {},
    warnings: result.warnings || [],
    quality: result.quality || "full"
  };
}

async function runProviderChain({
  platform,
  input,
  providerRequested,
  providerMap,
  requestOptions,
  includeDebug,
  normalizeSuccess
}) {
  const attempts = [];
  let lastError = null;
  let fallbackResult = null;
  let fallbackProvider = "";

  for (const providerName of buildProviderOrder(platform, providerRequested)) {
    const provider = providerMap[providerName];
    if (!provider || !provider.enabled) {
      attempts.push({ provider: providerName, ok: false, reason: "provider disabled" });
      lastError = {
        code: "provider_unavailable",
        message: "provider disabled",
        retryable: false
      };
      continue;
    }

    let result;
    try {
      result = await provider.parse(input, requestOptions);
    } catch (error) {
      result = {
        ok: false,
        errorCode: "provider_unavailable",
        errorMessage: error instanceof Error ? error.message : String(error || "provider failed")
      };
    }
    if (!result.ok) {
      attempts.push({
        provider: providerName,
        ok: false,
        reason: result.errorMessage || result.errorCode || "provider failed"
      });
      lastError = {
        code: result.errorCode || "provider_unavailable",
        message: result.errorMessage || "provider failed",
        retryable: false
      };
      continue;
    }

    if (providerRequested === "auto" && result.quality === "degraded") {
      attempts.push({ provider: providerName, ok: true, degraded: true });
      if (!fallbackResult) {
        fallbackResult = result;
        fallbackProvider = providerName;
      }
      continue;
    }

    const response = await normalizeSuccess(result, providerName);
    if (includeDebug) {
      response.debug = { attempts: attempts.concat([{ provider: providerName, ok: true }]) };
    }
    return response;
  }

  if (fallbackResult) {
    const response = await normalizeSuccess(fallbackResult, fallbackProvider);
    response.quality = fallbackResult.quality || "degraded";
    if (includeDebug) {
      response.debug = { attempts };
    }
    return response;
  }

  const errorPayload = {
    ok: false,
    platform,
    providerRequested,
    providerUsed: "",
    error: lastError || {
      code: "provider_unavailable",
      message: `no available ${platform} provider succeeded`,
      retryable: false
    },
    warnings: []
  };
  if (includeDebug) {
    errorPayload.debug = { attempts };
  }
  return errorPayload;
}

async function finalizeResponse(response) {
  if (!response.ok) {
    return response;
  }

  return {
    ok: true,
    platform: response.platform,
    providerRequested: response.providerRequested,
    providerUsed: response.providerUsed,
    normalized: response.normalized,
    content: await response.getContent(),
    warnings: response.warnings,
    quality: response.quality,
    ...(response.debug ? { debug: response.debug } : {})
  };
}

async function handleParseXiaohongshu(req, res, config, providers) {
  let payload;
  try {
    payload = await readJSON(req);
  } catch (error) {
    return sendJSON(res, 400, {
      ok: false,
      platform: "xiaohongshu",
      providerRequested: "",
      providerUsed: "",
      error: {
        code: "invalid_input",
        message: "invalid json body",
        retryable: false
      },
      warnings: []
    });
  }

  const input = String(payload.input || "").trim();
  const providerRequested = String(payload.provider || config.xiaohongshuDefaultProvider || "auto").trim().toLowerCase() || "auto";
  const includeDebug = !!payload.includeDebug;
  const includeTranscript = !!payload.includeTranscript;

  if (!input) {
    return sendJSON(res, 400, {
      ok: false,
      platform: "xiaohongshu",
      providerRequested,
      providerUsed: "",
      error: {
        code: "invalid_input",
        message: "input is required",
        retryable: false
      },
      warnings: []
    });
  }

  if (!isSupportedXHSUrl(input)) {
    return sendJSON(res, 400, {
      ok: false,
      platform: "xiaohongshu",
      providerRequested,
      providerUsed: "",
      error: {
        code: "unsupported_url",
        message: "unsupported xiaohongshu url",
        retryable: false
      },
      warnings: []
    });
  }

  const response = await runProviderChain({
    platform: "xiaohongshu",
    input,
    providerRequested,
    providerMap: providers.xiaohongshu,
    requestOptions: {},
    includeDebug,
    normalizeSuccess: async (result, providerName) =>
      normalizeXiaohongshuResponse(result, input, providerRequested, providerName, includeTranscript, config)
  });

  return sendJSON(res, 200, await finalizeResponse(response));
}

async function handleParseBilibili(req, res, config, providers) {
  let payload;
  try {
    payload = await readJSON(req);
  } catch (error) {
    return sendJSON(res, 400, {
      ok: false,
      platform: "bilibili",
      providerRequested: "",
      providerUsed: "",
      error: {
        code: "invalid_input",
        message: "invalid json body",
        retryable: false
      },
      warnings: []
    });
  }

  const input = String(payload.input || "").trim();
  const providerRequested = String(payload.provider || config.bilibiliDefaultProvider || "auto").trim().toLowerCase() || "auto";
  const includeDebug = !!payload.includeDebug;
  const includeTranscript = !!payload.includeTranscript;
  const sessdata = String(req.headers["x-bilibili-sessdata"] || "").trim();

  if (!input) {
    return sendJSON(res, 400, {
      ok: false,
      platform: "bilibili",
      providerRequested,
      providerUsed: "",
      error: {
        code: "invalid_input",
        message: "input is required",
        retryable: false
      },
      warnings: []
    });
  }

  if (!isSupportedBilibiliUrl(input)) {
    return sendJSON(res, 400, {
      ok: false,
      platform: "bilibili",
      providerRequested,
      providerUsed: "",
      error: {
        code: "unsupported_url",
        message: "unsupported bilibili url",
        retryable: false
      },
      warnings: []
    });
  }

  const response = await runProviderChain({
    platform: "bilibili",
    input,
    providerRequested,
    providerMap: providers.bilibili,
    requestOptions: {
      includeTranscript,
      sessdata
    },
    includeDebug,
    normalizeSuccess: async (result, providerName) =>
      normalizeBilibiliResponse(result, providerRequested, providerName)
  });

  return sendJSON(res, 200, await finalizeResponse(response));
}

async function getRednoteStatus(providers) {
  return providers.xiaohongshu.rednote && typeof providers.xiaohongshu.rednote.status === "function"
    ? providers.xiaohongshu.rednote.status()
    : { loggedIn: false, cookieCount: 0, playwrightAvailable: false };
}

function createServer(config) {
  const providers = buildProviders(config);

  return http.createServer(async (req, res) => {
    if (!ensureAuthorized(req, config)) {
      return sendJSON(res, 401, {
        ok: false,
        error: {
          code: "unauthorized",
          message: "invalid api key",
          retryable: false
        }
      });
    }

    if (req.method === "GET" && req.url === "/v1/health") {
      const rednoteStatus = await getRednoteStatus(providers);
      return sendJSON(res, 200, {
        ok: true,
        service: "linkparse-sidecar",
        version: "0.2.0",
        platforms: {
          xiaohongshu: {
            defaultProvider: config.xiaohongshuDefaultProvider,
            providers: {
              importer: { enabled: config.importerEnabled },
              rednote: {
                enabled: config.rednoteEnabled,
                loggedIn: !!rednoteStatus.loggedIn,
                ready: !!rednoteStatus.ready,
                cookieSource: rednoteStatus.cookieSource || "file",
                cookieCount: rednoteStatus.cookieCount || 0,
                cookieUpdatedAt: rednoteStatus.cookieUpdatedAt || "",
                playwrightAvailable: !!rednoteStatus.playwrightAvailable,
                browserInstalled: !!rednoteStatus.browserInstalled,
                lastError: rednoteStatus.lastError || ""
              }
            }
          },
          bilibili: {
            defaultProvider: config.bilibiliDefaultProvider,
            providers: {
              openapi: {
                enabled: config.bilibiliOpenAPIEnabled
              }
            }
          }
        }
      });
    }

    if (req.method === "GET" && req.url === "/v1/providers") {
      const rednoteStatus = await getRednoteStatus(providers);
      return sendJSON(res, 200, {
        platforms: [
          {
            name: "xiaohongshu",
            defaultProvider: config.xiaohongshuDefaultProvider,
            providers: [
              {
                name: "importer",
                enabled: config.importerEnabled,
                requiresLogin: false,
                supportsImageNotes: true,
                supportsVideoNotes: false
              },
              {
                name: "rednote",
                enabled: config.rednoteEnabled,
                requiresLogin: true,
                supportsImageNotes: true,
                supportsVideoNotes: true,
                loggedIn: !!rednoteStatus.loggedIn,
                ready: !!rednoteStatus.ready,
                cookieSource: rednoteStatus.cookieSource || "file",
                cookieCount: rednoteStatus.cookieCount || 0,
                cookieUpdatedAt: rednoteStatus.cookieUpdatedAt || "",
                playwrightAvailable: !!rednoteStatus.playwrightAvailable,
                browserInstalled: !!rednoteStatus.browserInstalled,
                lastError: rednoteStatus.lastError || ""
              }
            ]
          },
          {
            name: "bilibili",
            defaultProvider: config.bilibiliDefaultProvider,
            providers: [
              {
                name: "openapi",
                enabled: config.bilibiliOpenAPIEnabled,
                requiresLogin: false,
                supportsImageNotes: false,
                supportsVideoNotes: true
              }
            ]
          }
        ]
      });
    }

    if (req.method === "GET" && req.url === "/v1/auth/rednote/status") {
      const rednoteStatus = await getRednoteStatus(providers);
      return sendJSON(res, 200, {
        ok: true,
        provider: "rednote",
        loggedIn: !!rednoteStatus.loggedIn,
        ready: !!rednoteStatus.ready,
        cookieSource: rednoteStatus.cookieSource || "file",
        cookiePath: rednoteStatus.cookiePath || "",
        cookieCount: rednoteStatus.cookieCount || 0,
        playwrightAvailable: !!rednoteStatus.playwrightAvailable,
        browserInstalled: !!rednoteStatus.browserInstalled,
        cookieUpdatedAt: rednoteStatus.cookieUpdatedAt || "",
        lastCheckAt: new Date().toISOString(),
        lastError: rednoteStatus.lastError || ""
      });
    }

    if (req.method === "POST" && req.url === "/v1/parse/xiaohongshu") {
      return handleParseXiaohongshu(req, res, config, providers);
    }

    if (req.method === "POST" && req.url === "/v1/parse/bilibili") {
      return handleParseBilibili(req, res, config, providers);
    }

    return sendJSON(res, 404, {
      ok: false,
      error: {
        code: "not_found",
        message: "route not found",
        retryable: false
      }
    });
  });
}

if (require.main === module) {
  const config = readConfig();
  const server = createServer(config);
  server.listen(config.port, "127.0.0.1", () => {
    process.stdout.write(`linkparse-sidecar listening on http://127.0.0.1:${config.port}\n`);
  });
}

module.exports = {
  buildProviderOrder,
  createServer,
  readConfig
};
