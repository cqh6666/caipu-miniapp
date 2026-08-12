const test = require("node:test");
const assert = require("node:assert/strict");

const {
  buildSubtitleText,
  createBilibiliProvider,
  isSupportedBilibiliUrl,
  parseVideoRefFromURL,
  SESSION_VERIFICATION_PROBES,
  selectSubtitle
} = require("../providers/bilibili");

const PUBLIC_LOOKUP = async () => [{ address: "8.8.8.8", family: 4 }];

test("isSupportedBilibiliUrl rejects pseudo-suffix and userinfo confusion", () => {
  assert.equal(isSupportedBilibiliUrl("https://www.bilibili.com/video/BV1xx411c7mD"), true);
  assert.equal(isSupportedBilibiliUrl("http://b23.tv/demo"), true);
  assert.equal(isSupportedBilibiliUrl("https://b23.tv:8443/demo"), false);
  assert.equal(isSupportedBilibiliUrl("https://bilibili.com.attacker.example/share"), false);
  assert.equal(isSupportedBilibiliUrl("https://bilibili.com@attacker.example/share"), false);
  assert.equal(isSupportedBilibiliUrl("https://attacker.example@www.bilibili.com/share"), false);
});

test("bilibili provider upgrades an HTTP short link before fetching", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url) => {
      calls.push(url);
      throw new Error("stop after capturing normalized URL");
    }
  });

  await assert.rejects(provider.parse("http://b23.tv/demo", { includeTranscript: false }), /stop after capturing/);
  assert.deepEqual(calls, ["https://b23.tv/demo"]);
});

test("bilibili provider rejects an unsupported host before sending credentials", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      throw new Error("fetch should not be called");
    }
  });

  await assert.rejects(
    provider.parse("https://bilibili.com.attacker.example/share", {
      includeTranscript: false,
      sessdata: "TOP_SECRET"
    }),
    /only bilibili links are supported/
  );
  assert.equal(calls.length, 0);
});

test("bilibili provider validates every redirect before following it", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      return {
        status: 302,
        url,
        headers: {
          get(name) {
            return name.toLowerCase() === "location" ? "https://bilibili.com.attacker.example/video/BV1xx411c7mD" : "";
          }
        },
        async text() {
          return "";
        }
      };
    }
  });

  await assert.rejects(
    provider.parse("https://b23.tv/demo123", {
      includeTranscript: false,
      sessdata: "TOP_SECRET"
    }),
    /URL host is not allowed/
  );
  assert.equal(calls.length, 1);
  assert.equal(calls[0].init.redirect, "manual");
  assert.equal(calls[0].init.headers.Cookie, undefined);
});

test("parseVideoRefFromURL extracts bvid and page", () => {
  assert.deepEqual(parseVideoRefFromURL("https://www.bilibili.com/video/BV1xx411c7mD?p=2"), {
    bvid: "BV1xx411c7mD",
    aid: 0,
    page: 2,
    url: "https://www.bilibili.com/video/BV1xx411c7mD?p=2"
  });
});

test("selectSubtitle prefers chinese subtitle variants", () => {
  const selected = selectSubtitle([
    { lan: "en", subtitle_url: "//i0.hdslb.com/en.json" },
    { lan: "zh-CN", subtitle_url: "//i0.hdslb.com/zh.json" }
  ]);

  assert.equal(selected.lan, "zh-CN");
});

test("buildSubtitleText joins subtitle lines", () => {
  assert.deepEqual(
    buildSubtitleText({
      body: [{ content: "先焯水" }, { content: "再慢炖" }]
    }),
    {
      text: "先焯水\n再慢炖",
      segments: 2
    }
  );
});

test("bilibili provider verifies SESSDATA against fixed subtitle probes", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      const successfulProbe = SESSION_VERIFICATION_PROBES[1];
      const isSuccessfulProbe = url.includes(`bvid=${successfulProbe.bvid}`) && url.includes(`cid=${successfulProbe.cid}`);
      return {
        ok: true,
        status: 200,
        async json() {
          return {
            code: 0,
            data: {
              subtitle: {
                subtitles: isSuccessfulProbe
                  ? [{ lan: "zh-CN", subtitle_url: "https://i0.hdslb.com/subtitle.json" }]
                  : []
              }
            }
          };
        }
      };
    }
  });

  assert.equal(await provider.verifySession("session-secret"), true);
  assert.equal(calls.length, 2);
  assert.ok(calls.every((call) => call.url.startsWith("https://api.bilibili.com/x/player/v2?")));
  assert.ok(calls.every((call) => call.init.headers.Cookie === "SESSDATA=session-secret"));
  assert.ok(calls.every((call) => call.init.redirect === "manual"));
});

test("bilibili provider separates empty credentials, invalid credentials, and upstream failures", async () => {
  let calls = 0;
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async () => {
      calls += 1;
      throw new Error("upstream unavailable");
    }
  });

  assert.equal(await provider.verifySession(""), false);
  assert.equal(calls, 0);
  await assert.rejects(provider.verifySession("session-during-outage"), /upstream unavailable/);
  assert.equal(calls, SESSION_VERIFICATION_PROBES.length);

  const invalidProvider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async () => ({
      ok: true,
      status: 200,
      async json() {
        return { code: 0, data: { subtitle: { subtitles: [] } } };
      }
    })
  });
  assert.equal(await invalidProvider.verifySession("invalid-session"), false);
});

test("bilibili provider returns normalized transcript content", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      if (url.includes("/x/web-interface/view")) {
        return {
          ok: true,
          async json() {
            return {
              code: 0,
              data: {
                title: "番茄牛腩",
                desc: "牛腩 500克\n番茄 3个",
                pic: "https://i0.hdslb.com/demo.jpg",
                bvid: "BV1xx411c7mD",
                aid: 10086,
                owner: { name: "厨房UP" },
                pages: [{ cid: 20086, page: 1, part: "正片" }]
              }
            };
          }
        };
      }
      if (url.includes("/x/player/v2")) {
        return {
          ok: true,
          async json() {
            return {
              code: 0,
              data: {
                need_login_subtitle: false,
                subtitle: {
                  subtitles: [{ lan: "zh-CN", lan_doc: "中文", subtitle_url: "https://i0.hdslb.com/subtitle.json" }]
                }
              }
            };
          }
        };
      }
      if (url === "https://i0.hdslb.com/subtitle.json") {
        return {
          ok: true,
          async json() {
            return {
              body: [{ content: "先焯水" }, { content: "再慢炖" }]
            };
          }
        };
      }

      throw new Error(`unexpected url ${url}`);
    }
  });

  const result = await provider.parse("https://www.bilibili.com/video/BV1xx411c7mD", {
    includeTranscript: true,
    sessdata: "sess-123"
  });

  assert.equal(result.ok, true);
  assert.equal(result.normalized.bvid, "BV1xx411c7mD");
  assert.equal(result.content.transcript, "先焯水\n再慢炖");
  assert.equal(result.content.subtitleLanguage, "中文");
  assert.equal(result.content.subtitleSegments, 2);
  assert.match(calls[0].init.headers.Cookie, /SESSDATA=sess-123/);
  assert.match(calls[1].init.headers.Cookie, /SESSDATA=sess-123/);
  assert.equal(calls[2].init.headers.Cookie, undefined);
  assert.ok(calls.every((call) => call.init.redirect === "manual"));
});

test("bilibili provider accepts resolved short url even when final page responds 412", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      if (url === "https://b23.tv/demo123") {
        return {
          status: 302,
          url,
          headers: {
            get(name) {
              return name.toLowerCase() === "location"
                ? "https://www.bilibili.com/video/BV1xx411c7mD?p=2"
                : "";
            }
          },
          async text() {
            return "";
          }
        };
      }
      if (url === "https://www.bilibili.com/video/BV1xx411c7mD?p=2") {
        return {
          ok: false,
          status: 412,
          url,
          async text() {
            return "";
          }
        };
      }
      if (url.includes("/x/web-interface/view")) {
        return {
          ok: true,
          async json() {
            return {
              code: 0,
              data: {
                title: "番茄牛腩",
                desc: "",
                pic: "https://i0.hdslb.com/demo.jpg",
                bvid: "BV1xx411c7mD",
                aid: 10086,
                owner: { name: "厨房UP" },
                pages: [{ cid: 20086, page: 1, part: "正片" }, { cid: 20087, page: 2, part: "第二段" }]
              }
            };
          }
        };
      }

      throw new Error(`unexpected url ${url}`);
    }
  });

  const result = await provider.parse("https://b23.tv/demo123", {
    includeTranscript: false
  });

  assert.equal(result.ok, true);
  assert.equal(result.normalized.shareUrl, "https://b23.tv/demo123");
  assert.equal(result.normalized.canonicalUrl, "https://www.bilibili.com/video/BV1xx411c7mD?p=2");
  assert.equal(result.normalized.page, 2);
  assert.match(result.warnings.join("\n"), /已自动展开 B 站短链接/);
  assert.equal(calls[0].url, "https://b23.tv/demo123");
  assert.equal(calls[0].init.headers.Cookie, undefined);
  assert.equal(calls[1].init.headers.Cookie, undefined);
  assert.equal(calls[2].url.includes("/x/web-interface/view"), true);
});

test("bilibili provider rejects an untrusted subtitle URL without leaking SESSDATA", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      if (url.includes("/x/web-interface/view")) {
        return {
          ok: true,
          status: 200,
          async json() {
            return {
              code: 0,
              data: {
                title: "番茄牛腩",
                desc: "",
                pic: "",
                bvid: "BV1xx411c7mD",
                aid: 10086,
                owner: { name: "厨房UP" },
                pages: [{ cid: 20086, page: 1, part: "正片" }]
              }
            };
          }
        };
      }
      if (url.includes("/x/player/v2")) {
        return {
          ok: true,
          status: 200,
          async json() {
            return {
              code: 0,
              data: {
                need_login_subtitle: false,
                subtitle: {
                  subtitles: [{ lan: "zh-CN", subtitle_url: "https://attacker.example/subtitle.json" }]
                }
              }
            };
          }
        };
      }
      throw new Error(`unexpected fetch ${url}`);
    }
  });

  await assert.rejects(
    provider.parse("https://www.bilibili.com/video/BV1xx411c7mD", {
      includeTranscript: true,
      sessdata: "TOP_SECRET"
    }),
    /URL host is not allowed/
  );

  assert.equal(calls.length, 2);
  assert.ok(calls.every((call) => call.init.headers.Cookie === "SESSDATA=TOP_SECRET"));
});
