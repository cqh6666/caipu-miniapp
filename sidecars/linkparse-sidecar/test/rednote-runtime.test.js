const test = require("node:test");
const assert = require("node:assert/strict");

const {
  loadCookiesFromHeader,
  normalizeCookieHeader,
  summarizeRuntime
} = require("../lib/rednote-runtime");

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
