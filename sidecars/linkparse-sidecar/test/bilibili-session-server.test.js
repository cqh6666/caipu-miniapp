const test = require("node:test");
const assert = require("node:assert/strict");

const { createServer } = require("../server");

const PUBLIC_LOOKUP = async () => [{ address: "8.8.8.8", family: 4 }];

async function withServer(overrides, action) {
  const server = createServer({
    apiKey: "internal-secret",
    bilibiliDefaultProvider: "auto",
    bilibiliOpenAPIEnabled: true,
    importerEnabled: false,
    rednoteEnabled: false,
    xiaohongshuDefaultProvider: "auto",
    lookupImpl: PUBLIC_LOOKUP,
    ...overrides
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  try {
    await action(`http://127.0.0.1:${address.port}`);
  } finally {
    await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
}

test("bilibili session verification endpoint is authenticated and does not echo credentials", async () => {
  const requests = [];
  await withServer({
    requestImpl: async (url, init) => {
      requests.push({ url, init });
      return {
        ok: true,
        status: 200,
        async json() {
          return {
            code: 0,
            data: {
              subtitle: {
                subtitles: [{ lan: "zh-CN", subtitle_url: "https://i0.hdslb.com/subtitle.json" }]
              }
            }
          };
        }
      };
    }
  }, async (baseURL) => {
    const unauthorized = await fetch(`${baseURL}/v1/verify/bilibili-session`, { method: "POST" });
    assert.equal(unauthorized.status, 401);

    const response = await fetch(`${baseURL}/v1/verify/bilibili-session`, {
      method: "POST",
      headers: {
        Authorization: "Bearer internal-secret",
        "X-Bilibili-SESSDATA": "session-secret"
      }
    });
    assert.equal(response.status, 200);
    const body = await response.json();
    assert.deepEqual(body, {
      ok: true,
      platform: "bilibili",
      providerUsed: "openapi",
      valid: true
    });
    assert.equal(JSON.stringify(body).includes("session-secret"), false);
  });

  assert.equal(requests.length, 1);
  assert.equal(requests[0].init.headers.Cookie, "SESSDATA=session-secret");
});

test("bilibili session verification endpoint preserves invalid and unavailable errors", async () => {
  await withServer({
    requestImpl: async () => ({
      ok: true,
      status: 200,
      async json() {
        return { code: 0, data: { subtitle: { subtitles: [] } } };
      }
    })
  }, async (baseURL) => {
    const empty = await fetch(`${baseURL}/v1/verify/bilibili-session`, {
      method: "POST",
      headers: { Authorization: "Bearer internal-secret" }
    });
    assert.equal(empty.status, 400);
    assert.equal((await empty.json()).error.code, "invalid_input");

    const invalid = await fetch(`${baseURL}/v1/verify/bilibili-session`, {
      method: "POST",
      headers: {
        Authorization: "Bearer internal-secret",
        "X-Bilibili-SESSDATA": "invalid-session"
      }
    });
    assert.equal(invalid.status, 400);
    const invalidBody = await invalid.json();
    assert.equal(invalidBody.error.code, "invalid_credentials");
    assert.equal(JSON.stringify(invalidBody).includes("invalid-session"), false);
  });

  await withServer({ bilibiliOpenAPIEnabled: false }, async (baseURL) => {
    const unavailable = await fetch(`${baseURL}/v1/verify/bilibili-session`, {
      method: "POST",
      headers: {
        Authorization: "Bearer internal-secret",
        "X-Bilibili-SESSDATA": "session-secret"
      }
    });
    assert.equal(unavailable.status, 503);
    assert.equal((await unavailable.json()).error.code, "provider_unavailable");
  });

  await withServer({
    requestImpl: async () => {
      throw new Error("private upstream outage details");
    }
  }, async (baseURL) => {
    const unavailable = await fetch(`${baseURL}/v1/verify/bilibili-session`, {
      method: "POST",
      headers: {
        Authorization: "Bearer internal-secret",
        "X-Bilibili-SESSDATA": "session-secret"
      }
    });
    assert.equal(unavailable.status, 502);
    const body = await unavailable.json();
    assert.equal(body.error.code, "upstream_failure");
    assert.equal(body.error.retryable, true);
    assert.equal(JSON.stringify(body).includes("private upstream"), false);
    assert.equal(JSON.stringify(body).includes("session-secret"), false);
  });
});
