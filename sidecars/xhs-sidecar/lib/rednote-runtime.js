const fs = require("fs");
const os = require("os");
const path = require("path");

const SET_COOKIE_ATTRIBUTES = new Set([
  "path",
  "domain",
  "expires",
  "max-age",
  "samesite",
  "secure",
  "httponly",
  "priority",
  "partitioned"
]);

const LIKELY_LOGIN_COOKIE_NAMES = new Set([
  "a1",
  "webid",
  "gid",
  "web_session",
  "web_session_id",
  "websectiga",
  "websectiga_v2"
]);

function defaultCookiePath() {
  return path.join(os.homedir(), ".mcp", "rednote", "cookies.json");
}

function canUsePlaywright() {
  try {
    return require("playwright");
  } catch (error) {
    try {
      return require("playwright-core");
    } catch (innerError) {
      return null;
    }
  }
}

function getCookieUpdatedAt(cookiePath) {
  try {
    return fs.statSync(cookiePath).mtime.toISOString();
  } catch (error) {
    return "";
  }
}

function loadCookiesFromFile(cookiePath) {
  if (!fs.existsSync(cookiePath)) {
    return {
      cookies: [],
      exists: false,
      updatedAt: "",
      parseError: "",
      source: "file"
    };
  }

  try {
    const raw = fs.readFileSync(cookiePath, "utf8");
    const cookies = JSON.parse(raw);
    return {
      cookies: Array.isArray(cookies) ? cookies : [],
      exists: true,
      updatedAt: getCookieUpdatedAt(cookiePath),
      parseError: "",
      source: "file"
    };
  } catch (error) {
    return {
      cookies: [],
      exists: true,
      updatedAt: getCookieUpdatedAt(cookiePath),
      parseError: error instanceof Error ? error.message : String(error || "invalid cookie file"),
      source: "file"
    };
  }
}

function buildCookieFromHeaderPair(name, value, domain) {
  return {
    name,
    value,
    domain,
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax"
  };
}

function normalizeCookieHeader(rawHeader) {
  return String(rawHeader || "")
    .trim()
    .replace(/^cookie\s*:\s*/i, "")
    .trim();
}

function hasLikelyLoginCookies(cookies) {
  const names = new Set((cookies || []).map((cookie) => String(cookie?.name || "").trim().toLowerCase()).filter(Boolean));
  if (names.size === 0) {
    return false;
  }

  if (names.has("a1") || names.has("web_session") || names.has("web_session_id")) {
    return true;
  }

  return names.has("webid") && names.has("gid");
}

function validateCookieState(cookieState) {
  if (cookieState.parseError) {
    return {
      ok: false,
      errorMessage:
        cookieState.source === "header"
          ? `invalid cookie header: ${cookieState.parseError}`
          : `invalid cookie file: ${cookieState.parseError}`
    };
  }

  if (!cookieState.exists || cookieState.cookies.length === 0) {
    return {
      ok: false,
      errorMessage:
        cookieState.source === "header"
          ? "rednote cookie header is empty or invalid"
          : "rednote cookie file not found or empty"
    };
  }

  if (!hasLikelyLoginCookies(cookieState.cookies)) {
    return {
      ok: false,
      errorMessage:
        cookieState.source === "header"
          ? "rednote cookie header does not include expected login cookies"
          : "rednote cookie file does not include expected login cookies"
    };
  }

  return { ok: true, errorMessage: "" };
}

function parseCookieHeader(rawHeader, domain) {
  const pairs = normalizeCookieHeader(rawHeader)
    .split(";")
    .map((part) => part.trim())
    .filter(Boolean);

  const cookies = [];
  for (const pair of pairs) {
    const equalIndex = pair.indexOf("=");
    if (equalIndex <= 0) {
      continue;
    }

    const name = pair.slice(0, equalIndex).trim();
    const value = pair.slice(equalIndex + 1).trim();
    if (!name || !value) {
      continue;
    }
    if (SET_COOKIE_ATTRIBUTES.has(name.toLowerCase())) {
      continue;
    }

    cookies.push(buildCookieFromHeaderPair(name, value, domain));
  }

  return cookies;
}

function loadCookiesFromHeader(rawHeader, cookieDomain) {
  const cookies = parseCookieHeader(rawHeader, cookieDomain || ".xiaohongshu.com");
  return {
    cookies,
    exists: cookies.length > 0,
    updatedAt: "",
    parseError: cookies.length > 0 ? "" : "no valid cookies found in cookie header",
    source: "header"
  };
}

function loadCookies(configOrPath) {
  if (typeof configOrPath === "string") {
    return loadCookiesFromFile(configOrPath);
  }

  const config = configOrPath || {};
  const rawHeader = String(config.rednoteCookieHeader || "").trim();
  if (rawHeader) {
    return loadCookiesFromHeader(rawHeader, String(config.rednoteCookieDomain || "").trim() || ".xiaohongshu.com");
  }

  const cookiePath = String(config.rednoteCookiePath || "").trim() || defaultCookiePath();
  return loadCookiesFromFile(cookiePath);
}

function ensureCookieDir(cookiePath) {
  fs.mkdirSync(path.dirname(cookiePath), { recursive: true });
}

function writeCookies(cookiePath, cookies) {
  ensureCookieDir(cookiePath);
  fs.writeFileSync(cookiePath, `${JSON.stringify(cookies, null, 2)}\n`, "utf8");
}

function buildBrowserLaunchOptions(config, overrides = {}) {
  const launchOptions = {
    headless: config.rednoteBrowserHeadless,
    ...overrides
  };
  if (config.rednoteBrowserPath) {
    launchOptions.executablePath = config.rednoteBrowserPath;
  }
  return launchOptions;
}

function summarizeRuntime(config) {
  const cookiePath = config.rednoteCookiePath || defaultCookiePath();
  const cookieState = loadCookies(config);
  const cookieValidation = validateCookieState(cookieState);
  const playwright = canUsePlaywright();
  const playwrightAvailable = !!(playwright && playwright.chromium);
  let browserInstalled = false;

  if (config.rednoteBrowserPath) {
    browserInstalled = fs.existsSync(config.rednoteBrowserPath);
  } else if (playwrightAvailable) {
    try {
      browserInstalled = fs.existsSync(playwright.chromium.executablePath());
    } catch (error) {
      browserInstalled = false;
    }
  }

  const loggedIn = cookieValidation.ok;
  let lastError = "";

  if (!cookieValidation.ok) {
    lastError = cookieValidation.errorMessage;
  } else if (!playwrightAvailable) {
    lastError = "playwright is not installed";
  } else if (!browserInstalled) {
    lastError = "playwright browser is not installed";
  }

  return {
    cookieSource: cookieState.source || "file",
    cookiePath,
    cookieCount: cookieState.cookies.length,
    cookieUpdatedAt: cookieState.updatedAt,
    playwrightAvailable,
    browserInstalled,
    loggedIn,
    ready: loggedIn && playwrightAvailable && browserInstalled,
    lastError
  };
}

module.exports = {
  buildBrowserLaunchOptions,
  canUsePlaywright,
  defaultCookiePath,
  ensureCookieDir,
  hasLikelyLoginCookies,
  loadCookies,
  loadCookiesFromHeader,
  normalizeCookieHeader,
  summarizeRuntime,
  validateCookieState,
  writeCookies
};
