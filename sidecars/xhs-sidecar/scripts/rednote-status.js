const { defaultCookiePath, summarizeRuntime } = require("../lib/rednote-runtime");

function readConfig() {
  return {
    rednoteCookiePath: String(process.env.XHS_REDNOTE_COOKIE_PATH || "").trim() || defaultCookiePath(),
    rednoteBrowserHeadless: String(process.env.XHS_BROWSER_HEADLESS || "").trim().toLowerCase() !== "false",
    rednoteBrowserPath: String(process.env.XHS_REDNOTE_BROWSER_PATH || "").trim()
  };
}

process.stdout.write(`${JSON.stringify(summarizeRuntime(readConfig()), null, 2)}\n`);
