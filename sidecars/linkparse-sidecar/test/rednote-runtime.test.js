const test = require("node:test");
const assert = require("node:assert/strict");

const {
  filterXiaohongshuCookies,
  loadCookiesFromHeader,
  normalizeCookieHeader,
  summarizeRuntime
} = require("../lib/rednote-runtime");

test("filterXiaohongshuCookies removes cookies for unrelated domains", () => {
  const cookies = filterXiaohongshuCookies([
    { name: "a1", domain: ".xiaohongshu.com" },
    { name: "web_session", domain: "www.xiaohongshu.com" },
    { name: "secret", domain: "xiaohongshu.com.attacker.example" }
  ]);

  assert.deepEqual(cookies.map((cookie) => cookie.name), ["a1", "web_session"]);
});

test("loadCookiesFromHeader rejects an unrelated configured cookie domain", () => {
  const state = loadCookiesFromHeader("a1=secret", ".attacker.example");

  assert.equal(state.cookies.length, 0);
  assert.match(state.parseError, /cookie domain must be xiaohongshu\.com/);
});

test("normalizeCookieHeader strips leading Cookie prefix", () => {
  assert.equal(
    normalizeCookieHeader("Cookie: a1=abc; webId=def; gid=ghi"),
    "a1=abc; webId=def; gid=ghi"
  );
});

test("loadCookiesFromHeader parses cookie names after Cookie prefix", () => {
  const state = loadCookiesFromHeader("Cookie: a1=abc; webId=def; gid=ghi", ".xiaohongshu.com");

  assert.equal(state.source, "header");
  assert.equal(state.cookies.length, 3);
  assert.deepEqual(
    state.cookies.map((cookie) => cookie.name),
    ["a1", "webId", "gid"]
  );
});

test("summarizeRuntime keeps junk header from reporting logged-in", () => {
  const status = summarizeRuntime({
    rednoteCookieHeader: "foo=bar",
    rednoteCookieDomain: ".xiaohongshu.com",
    rednoteBrowserPath: __filename
  });

  assert.equal(status.cookieSource, "header");
  assert.equal(status.loggedIn, false);
  assert.equal(status.ready, false);
  assert.match(status.lastError, /expected login cookies/i);
});
