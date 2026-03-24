const readline = require("readline");
const {
  buildBrowserLaunchOptions,
  canUsePlaywright,
  defaultCookiePath,
  writeCookies
} = require("../lib/rednote-runtime");

function readConfig() {
  return {
    rednoteCookiePath: String(process.env.XHS_REDNOTE_COOKIE_PATH || "").trim() || defaultCookiePath(),
    rednoteBrowserHeadless: false,
    rednoteBrowserPath: String(process.env.XHS_REDNOTE_BROWSER_PATH || "").trim(),
    rednoteLoginURL: String(process.env.XHS_REDNOTE_LOGIN_URL || "https://www.xiaohongshu.com/").trim(),
    rednoteTimeoutMS: Number(process.env.XHS_REDNOTE_TIMEOUT_MS || 30000)
  };
}

function waitForEnter(promptText) {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  return new Promise((resolve) => {
    rl.question(promptText, () => {
      rl.close();
      resolve();
    });
  });
}

async function main() {
  const config = readConfig();
  const playwright = canUsePlaywright();
  if (!playwright || !playwright.chromium) {
    process.stderr.write("playwright is not installed. Run `npm install` and `npx playwright install` first.\n");
    process.exitCode = 1;
    return;
  }

  const browser = await playwright.chromium.launch(buildBrowserLaunchOptions(config, { headless: false }));
  const context = await browser.newContext();
  const page = await context.newPage();
  page.setDefaultTimeout(config.rednoteTimeoutMS);

  try {
    await page.goto(config.rednoteLoginURL, {
      waitUntil: "domcontentloaded",
      timeout: config.rednoteTimeoutMS
    });

    process.stdout.write(`已打开小红书登录页：${config.rednoteLoginURL}\n`);
    process.stdout.write("请在浏览器中完成登录，然后回到终端按 Enter 保存 Cookie。\n");
    await waitForEnter("登录完成后按 Enter 继续...");

    const cookies = await context.cookies();
    if (!cookies || cookies.length === 0) {
      process.stderr.write("未读取到任何 Cookie，请确认已完成登录后重试。\n");
      process.exitCode = 1;
      return;
    }

    writeCookies(config.rednoteCookiePath, cookies);
    process.stdout.write(`已保存 ${cookies.length} 条 Cookie 到 ${config.rednoteCookiePath}\n`);
  } finally {
    await context.close().catch(() => {});
    await browser.close().catch(() => {});
  }
}

main().catch((error) => {
  process.stderr.write(`${error instanceof Error ? error.stack || error.message : String(error)}\n`);
  process.exitCode = 1;
});
