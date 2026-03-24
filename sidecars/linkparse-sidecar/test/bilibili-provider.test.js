const test = require("node:test");
const assert = require("node:assert/strict");

const {
  buildSubtitleText,
  createBilibiliProvider,
  parseVideoRefFromURL,
  selectSubtitle
} = require("../providers/bilibili");

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

test("bilibili provider returns normalized transcript content", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    fetchImpl: async (url, init) => {
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
});

test("bilibili provider accepts resolved short url even when final page responds 412", async () => {
  const calls = [];
  const provider = createBilibiliProvider({
    bilibiliOpenAPIEnabled: true,
    fetchImpl: async (url, init) => {
      calls.push({ url, init });
      if (url === "https://b23.tv/demo123") {
        return {
          ok: false,
          status: 412,
          url: "https://www.bilibili.com/video/BV1xx411c7mD?p=2",
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
});
