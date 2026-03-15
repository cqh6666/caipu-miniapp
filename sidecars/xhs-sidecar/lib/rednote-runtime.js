const fs = require("fs");
const os = require("os");
const path = require("path");

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

function loadCookies(cookiePath) {
  if (!fs.existsSync(cookiePath)) {
    return {
      cookies: [],
      exists: false,
      updatedAt: "",
      parseError: ""
    };
  }

  try {
    const raw = fs.readFileSync(cookiePath, "utf8");
    const cookies = JSON.parse(raw);
    return {
      cookies: Array.isArray(cookies) ? cookies : [],
      exists: true,
      updatedAt: getCookieUpdatedAt(cookiePath),
      parseError: ""
    };
  } catch (error) {
    return {
      cookies: [],
      exists: true,
      updatedAt: getCookieUpdatedAt(cookiePath),
      parseError: error instanceof Error ? error.message : String(error || "invalid cookie file")
    };
  }
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
  const cookieState = loadCookies(cookiePath);
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

  const loggedIn = cookieState.exists && cookieState.cookies.length > 0 && !cookieState.parseError;
  let lastError = "";

  if (cookieState.parseError) {
    lastError = `invalid cookie file: ${cookieState.parseError}`;
  } else if (!cookieState.exists || cookieState.cookies.length === 0) {
    lastError = `rednote cookie file not found or empty: ${cookiePath}`;
  } else if (!playwrightAvailable) {
    lastError = "playwright is not installed";
  } else if (!browserInstalled) {
    lastError = "playwright browser is not installed";
  }

  return {
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
  loadCookies,
  summarizeRuntime,
  writeCookies
};
