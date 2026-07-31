const test = require("node:test");
const assert = require("node:assert/strict");

const { createRednoteProvider, isAllowedRednoteRequest } = require("../providers/rednote");

test("rednote request policy only allows trusted HTTPS page and asset domains", () => {
  assert.equal(isAllowedRednoteRequest("https://www.xiaohongshu.com/explore/123", true), true);
  assert.equal(isAllowedRednoteRequest("https://xiaohongshu.com.attacker.example/explore/123", true), false);
  assert.equal(isAllowedRednoteRequest("http://www.xiaohongshu.com/explore/123", true), false);
  assert.equal(isAllowedRednoteRequest("http://127.0.0.1/internal", false), false);
  assert.equal(isAllowedRednoteRequest("https://sns-webpic-qc.xhscdn.com/resource.js", false), true);
  assert.equal(isAllowedRednoteRequest("https://xhslink.com/resource.js", false), false);
  assert.equal(isAllowedRednoteRequest("https://cdn.example.com/resource.js", false), false);
  assert.equal(isAllowedRednoteRequest("https://sns-webpic-qc.xhscdn.com:8443/resource.js", false), false);
});

test("rednote provider rejects a pseudo-suffix before browser startup", async () => {
  const provider = createRednoteProvider({
    rednoteEnabled: true,
    stubMode: "off"
  });

  const result = await provider.parse("https://xiaohongshu.com.attacker.example/explore/123");

  assert.equal(result.ok, false);
  assert.equal(result.errorCode, "invalid_input");
});

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
