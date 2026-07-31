const test = require("node:test");
const assert = require("node:assert/strict");

const { createImporterProvider, normalizeMediaUrl } = require("../providers/importer");
const { buildNormalized, isSupportedXHSUrl } = require("../lib/normalize");

const PUBLIC_LOOKUP = async () => [{ address: "8.8.8.8", family: 4 }];

test("isSupportedXHSUrl rejects pseudo-suffix and userinfo confusion", () => {
  assert.equal(isSupportedXHSUrl("https://www.xiaohongshu.com/explore/123"), true);
  assert.equal(isSupportedXHSUrl("http://xhslink.com/demo"), true);
  assert.equal(isSupportedXHSUrl("https://xhslink.com:8443/demo"), false);
  assert.equal(isSupportedXHSUrl("https://xiaohongshu.com.attacker.example/explore/123"), false);
  assert.equal(isSupportedXHSUrl("https://xhslink.com.attacker.example/a/123"), false);
  assert.equal(isSupportedXHSUrl("https://xiaohongshu.com@attacker.example/explore/123"), false);
  assert.equal(isSupportedXHSUrl("https://attacker.example@www.xiaohongshu.com/explore/123"), false);
});

test("buildNormalized upgrades default-port HTTP input to HTTPS", () => {
  const normalized = buildNormalized("http://xhslink.com/demo");

  assert.equal(normalized.shareUrl, "https://xhslink.com/demo");
});

test("buildNormalized ignores an untrusted redirectPath", () => {
  const normalized = buildNormalized(
    "https://www.xiaohongshu.com/404?redirectPath=http%3A%2F%2F169.254.169.254%2Flatest%2Fmeta-data"
  );

  assert.ok(normalized);
  assert.equal(new URL(normalized.canonicalUrl).hostname, "www.xiaohongshu.com");
  assert.equal(new URL(normalized.canonicalUrl).pathname, "/404");
});

test("importer validates redirect targets before following them", async () => {
  const calls = [];
  const provider = createImporterProvider({
    importerEnabled: true,
    stubMode: "off",
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      return {
        status: 302,
        url,
        headers: {
          get(name) {
            return name.toLowerCase() === "location" ? "https://attacker.example/internal" : "";
          }
        },
        async text() {
          return "";
        }
      };
    }
  });

  const result = await provider.parse("https://xhslink.com/demo123");

  assert.equal(result.ok, false);
  assert.match(result.errorMessage, /URL host is not allowed/);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].init.redirect, "manual");
});

test("importer filters media URLs outside the xiaohongshu asset domains", async () => {
  const provider = createImporterProvider({
    importerEnabled: true,
    stubMode: "off",
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async () => ({
      ok: true,
      status: 200,
      async text() {
        return `<script>window.__INITIAL_STATE__=${JSON.stringify({
          note: {
            noteDetailMap: {
              demo123: {
                note: {
                  title: "番茄牛腩",
                  desc: "牛腩焯水后炖煮",
                  imageList: [
                    { urlDefault: "https://sns-webpic-qc.xhscdn.com/trusted.jpg" },
                    { urlDefault: "https://attacker.example/leak.jpg" }
                  ],
                  video: {
                    media: {
                      stream: {
                        h264: [
                          { masterUrl: "https://sns-video-hw.xhscdn.com/trusted.mp4" },
                          { masterUrl: "https://attacker.example/leak.mp4" }
                        ]
                      }
                    }
                  }
                }
              }
            }
          }
        })}</script>`;
      }
    })
  });

  const result = await provider.parse("https://www.xiaohongshu.com/explore/demo123");

  assert.equal(result.ok, true);
  assert.deepEqual(result.note.images, ["https://sns-webpic-qc.xhscdn.com/trusted.jpg"]);
  assert.deepEqual(result.note.videos, ["https://sns-video-hw.xhscdn.com/trusted.mp4"]);
  assert.equal(result.note.coverUrl, "https://sns-webpic-qc.xhscdn.com/trusted.jpg");
});

test("normalizeMediaUrl upgrades protocol-relative urls to https", () => {
  assert.equal(
    normalizeMediaUrl("//sns-webpic-qc.xhscdn.com/demo.jpg"),
    "https://sns-webpic-qc.xhscdn.com/demo.jpg"
  );
});

test("normalizeMediaUrl upgrades http urls to https", () => {
  assert.equal(
    normalizeMediaUrl("http://sns-webpic-qc.xhscdn.com/demo.jpg"),
    "https://sns-webpic-qc.xhscdn.com/demo.jpg"
  );
});
