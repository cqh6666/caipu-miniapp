const test = require("node:test");
const assert = require("node:assert/strict");

const { createRednoteProvider } = require("../providers/rednote");

test("rednote provider does not downgrade invalid cookie header into echo fallback", async () => {
  const provider = createRednoteProvider({
    rednoteEnabled: true,
    stubMode: "echo",
    rednoteCookieHeader: "foo=bar",
    rednoteCookieDomain: ".xiaohongshu.com",
    rednoteBrowserHeadless: true,
    rednoteBrowserPath: __filename,
    rednoteTimeoutMS: 1000
  });

  const result = await provider.parse("https://www.xiaohongshu.com/explore/68b6e4f3000000001f03379f");

  assert.equal(result.ok, false);
  assert.equal(result.errorCode, "login_required");
  assert.match(result.errorMessage, /expected login cookies/i);
});
