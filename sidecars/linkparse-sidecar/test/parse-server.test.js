const test = require("node:test");
const assert = require("node:assert/strict");

const { createServer } = require("../server");

async function withServer(action) {
  const server = createServer({
    apiKey: "",
    bilibiliDefaultProvider: "auto",
    bilibiliOpenAPIEnabled: false,
    importerEnabled: false,
    rednoteEnabled: false,
    xiaohongshuDefaultProvider: "auto"
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  try {
    await action(`http://127.0.0.1:${address.port}`);
  } finally {
    await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
}

for (const platform of ["xiaohongshu", "bilibili"]) {
  test(`${platform} parse endpoint shares invalid request envelopes`, async () => {
    await withServer(async (baseURL) => {
      const invalidJSON = await fetch(`${baseURL}/v1/parse/${platform}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: "{"
      });
      assert.equal(invalidJSON.status, 400);
      assert.deepEqual(await invalidJSON.json(), {
        ok: false,
        platform,
        providerRequested: "",
        providerUsed: "",
        error: { code: "invalid_input", message: "invalid json body", retryable: false },
        warnings: []
      });

      const empty = await fetch(`${baseURL}/v1/parse/${platform}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: "OPENAPI" })
      });
      assert.equal(empty.status, 400);
      const emptyBody = await empty.json();
      assert.equal(emptyBody.providerRequested, "openapi");
      assert.equal(emptyBody.error.message, "input is required");
    });
  });
}

test("bilibili parse endpoint distinguishes provider absence from upstream failure", async () => {
  await withServer(async (baseURL) => {
    const unavailable = await fetch(`${baseURL}/v1/parse/bilibili`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ input: "https://www.bilibili.com/video/BV1xx411c7mD" })
    });
    assert.equal(unavailable.status, 503);
    assert.equal((await unavailable.json()).error.code, "provider_unavailable");
  });

  const server = createServer({
    apiKey: "",
    bilibiliDefaultProvider: "auto",
    bilibiliOpenAPIEnabled: true,
    importerEnabled: false,
    rednoteEnabled: false,
    xiaohongshuDefaultProvider: "auto",
    lookupImpl: async () => [{ address: "8.8.8.8", family: 4 }],
    requestImpl: async () => {
      throw new Error("private upstream outage details");
    }
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  try {
    const failure = await fetch(`http://127.0.0.1:${address.port}/v1/parse/bilibili`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ input: "https://www.bilibili.com/video/BV1xx411c7mD" })
    });
    assert.equal(failure.status, 502);
    const body = await failure.json();
    assert.equal(body.error.code, "upstream_failure");
    assert.equal(body.error.retryable, true);
    assert.equal(JSON.stringify(body).includes("private upstream"), false);
  } finally {
    await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
});
