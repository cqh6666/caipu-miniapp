const http = require("http");
const { createImporterProvider } = require("./providers/importer");
const { createRednoteProvider } = require("./providers/rednote");
const { buildNormalized, isSupportedXHSUrl } = require("./lib/normalize");

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
    defaultProvider: String(process.env.XHS_PROVIDER_DEFAULT || "auto").trim().toLowerCase() || "auto",
    importerEnabled: getEnvBool("XHS_PROVIDER_IMPORTER_ENABLED", true),
    rednoteEnabled: getEnvBool("XHS_PROVIDER_REDNOTE_ENABLED", true),
    stubMode: String(process.env.XHS_SIDECAR_STUB_MODE || "echo").trim().toLowerCase() || "echo",
    apiKey: String(process.env.XHS_INTERNAL_API_KEY || "").trim(),
    rednoteCookiePath: String(process.env.XHS_REDNOTE_COOKIE_PATH || "").trim(),
    rednoteBrowserHeadless: getEnvBool("XHS_BROWSER_HEADLESS", true),
    rednoteBrowserPath: String(process.env.XHS_REDNOTE_BROWSER_PATH || "").trim(),
    rednoteTimeoutMS: Number(process.env.XHS_REDNOTE_TIMEOUT_MS || 15000)
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
    importer: createImporterProvider(config),
    rednote: createRednoteProvider(config)
  };
}

function buildProviderOrder(requestedProvider, config) {
  switch (requestedProvider) {
    case "importer":
      return ["importer"];
    case "rednote":
      return ["rednote"];
    default:
      return ["importer", "rednote"];
  }
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
  const providerRequested = String(payload.provider || config.defaultProvider || "auto").trim().toLowerCase() || "auto";
  const includeDebug = !!payload.includeDebug;

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

  const attempts = [];
  let lastError = null;
  let fallbackResult = null;
  let fallbackProvider = "";
  for (const providerName of buildProviderOrder(providerRequested, config)) {
    const provider = providers[providerName];
    if (!provider || !provider.enabled) {
      attempts.push({
        provider: providerName,
        ok: false,
        reason: "provider disabled"
      });
      lastError = {
        code: "provider_unavailable",
        message: "provider disabled",
        retryable: false
      };
      continue;
    }

    const result = await provider.parse(input);
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
      attempts.push({
        provider: providerName,
        ok: true,
        degraded: true
      });
      if (!fallbackResult) {
        fallbackResult = result;
        fallbackProvider = providerName;
      }
      continue;
    }

    const response = {
      ok: true,
      platform: "xiaohongshu",
      providerRequested,
      providerUsed: providerName,
      normalized: result.normalized || buildNormalized(input),
      note: result.note,
      warnings: result.warnings || [],
      quality: result.quality || "full"
    };
    if (includeDebug) {
      response.debug = { attempts: attempts.concat([{ provider: providerName, ok: true }]) };
    }
    return sendJSON(res, 200, response);
  }

  if (fallbackResult) {
    const response = {
      ok: true,
      platform: "xiaohongshu",
      providerRequested,
      providerUsed: fallbackProvider,
      normalized: fallbackResult.normalized || buildNormalized(input),
      note: fallbackResult.note,
      warnings: fallbackResult.warnings || [],
      quality: fallbackResult.quality || "degraded"
    };
    if (includeDebug) {
      response.debug = { attempts };
    }
    return sendJSON(res, 200, response);
  }

  const errorPayload = {
    ok: false,
    platform: "xiaohongshu",
    providerRequested,
    providerUsed: "",
    error: lastError || {
      code: "provider_unavailable",
      message: "no available xiaohongshu provider succeeded",
      retryable: false
    },
    warnings: []
  };
  if (includeDebug) {
    errorPayload.debug = { attempts };
  }
  return sendJSON(res, 200, errorPayload);
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
      const rednoteStatus = providers.rednote && typeof providers.rednote.status === "function"
        ? await providers.rednote.status()
        : { loggedIn: false, cookieCount: 0, playwrightAvailable: false };
      return sendJSON(res, 200, {
        ok: true,
        service: "xhs-sidecar",
        version: "0.1.0",
        stubMode: config.stubMode,
        providers: {
          importer: { enabled: config.importerEnabled },
          rednote: {
            enabled: config.rednoteEnabled,
            loggedIn: !!rednoteStatus.loggedIn,
            ready: !!rednoteStatus.ready,
            cookieCount: rednoteStatus.cookieCount || 0,
            cookieUpdatedAt: rednoteStatus.cookieUpdatedAt || "",
            playwrightAvailable: !!rednoteStatus.playwrightAvailable,
            browserInstalled: !!rednoteStatus.browserInstalled,
            lastError: rednoteStatus.lastError || ""
          }
        }
      });
    }

    if (req.method === "GET" && req.url === "/v1/providers") {
      const rednoteStatus = providers.rednote && typeof providers.rednote.status === "function"
        ? await providers.rednote.status()
        : { loggedIn: false, cookieCount: 0, playwrightAvailable: false };
      return sendJSON(res, 200, {
        defaultProvider: config.defaultProvider,
        stubMode: config.stubMode,
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
            cookieCount: rednoteStatus.cookieCount || 0,
            cookieUpdatedAt: rednoteStatus.cookieUpdatedAt || "",
            playwrightAvailable: !!rednoteStatus.playwrightAvailable,
            browserInstalled: !!rednoteStatus.browserInstalled,
            lastError: rednoteStatus.lastError || ""
          }
        ]
      });
    }

    if (req.method === "GET" && req.url === "/v1/auth/rednote/status") {
      const rednoteStatus = providers.rednote && typeof providers.rednote.status === "function"
        ? await providers.rednote.status()
        : { loggedIn: false, cookieCount: 0, playwrightAvailable: false, cookiePath: "" };
      return sendJSON(res, 200, {
        ok: true,
        provider: "rednote",
        loggedIn: !!rednoteStatus.loggedIn,
        ready: !!rednoteStatus.ready,
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

const config = readConfig();
const server = createServer(config);
server.listen(config.port, "127.0.0.1", () => {
  process.stdout.write(`xhs-sidecar listening on http://127.0.0.1:${config.port}\n`);
});
