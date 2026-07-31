const test = require("node:test");
const assert = require("node:assert/strict");
const { Readable } = require("node:stream");

const { safeFetch } = require("../lib/safe-fetch");
const { BILIBILI_INPUT_DOMAINS, createPinnedLookup } = require("../lib/url-policy");

const PUBLIC_LOOKUP = async () => [{ address: "8.8.8.8", family: 4 }];

function redirectResponse(url, location) {
  return {
    status: 302,
    url,
    headers: {
      get(name) {
        return name.toLowerCase() === "location" ? location : "";
      }
    },
    async text() {
      return "";
    }
  };
}

test("safeFetch follows a validated relative redirect manually", async () => {
  const calls = [];
  const lookups = [];
  const result = await safeFetch("https://b23.tv/start", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: async (hostname) => {
      lookups.push(hostname);
      return PUBLIC_LOOKUP();
    },
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      assert.deepEqual(init.approvedAddresses, [{ address: "8.8.8.8", family: 4 }]);
      if (calls.length === 1) {
        return redirectResponse(url, "/video/BV1xx411c7mD");
      }
      return { status: 200, ok: true, url, async text() { return "ok"; } };
    },
    requireHTTPS: true
  });

  assert.equal(result.finalURL, "https://b23.tv/video/BV1xx411c7mD");
  assert.equal(calls.length, 2);
  assert.equal(calls[0].init.redirect, "manual");
  assert.equal(calls[1].init.redirect, "manual");
  assert.deepEqual(lookups, ["b23.tv", "b23.tv"]);
  await result.response.text();
});

test("safeFetch enforces HTTPS/default-port policy on every hop", async () => {
  await assert.rejects(
    safeFetch("http://b23.tv/start", {
      allowedDomains: BILIBILI_INPUT_DOMAINS,
      lookupImpl: PUBLIC_LOOKUP,
      requestImpl: async () => ({ status: 200 }),
      requireHTTPS: true
    }),
    /must use HTTPS and the default port/
  );

  await assert.rejects(
    safeFetch("https://b23.tv/start", {
      allowedDomains: BILIBILI_INPUT_DOMAINS,
      lookupImpl: PUBLIC_LOOKUP,
      requestImpl: async (url) => redirectResponse(url, "https://b23.tv:8443/next"),
      requireHTTPS: true
    }),
    /must use HTTPS and the default port/
  );
});

test("safeFetch strips credentials when a redirect changes origin", async () => {
  const calls = [];
  const result = await safeFetch("https://b23.tv/start", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    headers: { Cookie: "SECRET", Authorization: "Bearer SECRET" },
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (url, init) => {
      calls.push({ url, init });
      if (calls.length === 1) {
        return redirectResponse(url, "https://www.bilibili.com/video/BV1xx411c7mD");
      }
      return { status: 200, ok: true, url, async text() { return "ok"; } };
    },
    requireHTTPS: true
  });

  assert.equal(calls[0].init.headers.Cookie, "SECRET");
  assert.equal(calls[0].init.headers.Authorization, "Bearer SECRET");
  assert.equal(calls[1].init.headers.Cookie, undefined);
  assert.equal(calls[1].init.headers.Authorization, undefined);
  await result.response.text();
});

test("safeFetch rejects untrusted, downgraded and excessive redirects", async (t) => {
  await t.test("untrusted host", async () => {
    await assert.rejects(
      safeFetch("https://b23.tv/start", {
        allowedDomains: BILIBILI_INPUT_DOMAINS,
        lookupImpl: PUBLIC_LOOKUP,
        requestImpl: async (url) => redirectResponse(url, "https://attacker.example/internal"),
        requireHTTPS: true
      }),
      /URL host is not allowed/
    );
  });

  await t.test("HTTPS downgrade", async () => {
    await assert.rejects(
      safeFetch("https://b23.tv/start", {
        allowedDomains: BILIBILI_INPUT_DOMAINS,
        lookupImpl: PUBLIC_LOOKUP,
        requestImpl: async (url) => redirectResponse(url, "http://b23.tv/next"),
        requireHTTPS: true
      }),
      /HTTPS redirect downgrade/
    );
  });

  await t.test("redirect limit", async () => {
    await assert.rejects(
      safeFetch("https://b23.tv/start", {
        allowedDomains: BILIBILI_INPUT_DOMAINS,
        lookupImpl: PUBLIC_LOOKUP,
        maxRedirects: 1,
        requestImpl: async (url) => redirectResponse(url, "/next"),
        requireHTTPS: true
      }),
      /too many redirects/
    );
  });
});

test("safeFetch applies its deadline while discarding a redirect response", async () => {
  await assert.rejects(
    safeFetch("https://b23.tv/slow-redirect", {
      allowedDomains: BILIBILI_INPUT_DOMAINS,
      lookupImpl: PUBLIC_LOOKUP,
      requestImpl: async (url) => ({
        ...redirectResponse(url, "/next"),
        async text() {
          return new Promise(() => {});
        }
      }),
      requireHTTPS: true,
      timeoutMS: 10
    }),
    /request timed out after 10ms/
  );
});

test("createPinnedLookup returns only the already-approved DNS answers", async () => {
  const lookup = createPinnedLookup([
    { address: "8.8.8.8", family: 4 },
    { address: "2606:4700:4700::1111", family: 6 }
  ]);

  const all = await new Promise((resolve, reject) => {
    lookup("ignored.example", { all: true }, (error, addresses) => error ? reject(error) : resolve(addresses));
  });
  assert.deepEqual(all, [
    { address: "8.8.8.8", family: 4 },
    { address: "2606:4700:4700::1111", family: 6 }
  ]);
});

test("safeFetch rejects a private DNS answer before opening a socket", async () => {
  let requested = false;
  await assert.rejects(
    safeFetch("https://www.bilibili.com/video/demo", {
      allowedDomains: BILIBILI_INPUT_DOMAINS,
      lookupImpl: async () => [{ address: "169.254.169.254", family: 4 }],
      requestImpl: async () => {
        requested = true;
      },
      requireHTTPS: true
    }),
    /non-public address/
  );
  assert.equal(requested, false);
});

test("safeFetch applies its deadline while waiting for response headers", async () => {
  await assert.rejects(
    safeFetch("https://b23.tv/slow", {
      allowedDomains: BILIBILI_INPUT_DOMAINS,
      lookupImpl: PUBLIC_LOOKUP,
      requestImpl: async () => new Promise(() => {}),
      requireHTTPS: true,
      timeoutMS: 10
    }),
    /request timed out after 10ms/
  );
});

test("safeFetch applies its deadline while waiting for DNS", async () => {
  await assert.rejects(
    safeFetch("https://b23.tv/slow-dns", {
      allowedDomains: BILIBILI_INPUT_DOMAINS,
      lookupImpl: async () => new Promise(() => {}),
      requireHTTPS: true,
      timeoutMS: 10
    }),
    /request timed out after 10ms/
  );
});

test("safeFetch keeps the deadline active until the response body closes", async () => {
  const external = new AbortController();
  const body = new Readable({ read() {} });
  let transportSignal;
  const { response } = await safeFetch("https://b23.tv/slow-body", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (_url, init) => {
      transportSignal = init.signal;
      return { status: 200, ok: true, body };
    },
    requireHTTPS: true,
    signal: external.signal,
    timeoutMS: 1000
  });

  external.abort(new Error("caller cancelled"));
  assert.equal(transportSignal.aborted, true);
  response.body.destroy();
});

test("safeFetch enforces response size without Content-Length", async () => {
  const { response } = await safeFetch("https://b23.tv/large", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: PUBLIC_LOOKUP,
    maxResponseBytes: 10,
    requestImpl: async () => ({
      status: 200,
      ok: true,
      async text() {
        return "12345678901";
      }
    }),
    requireHTTPS: true
  });

  await assert.rejects(response.text(), /response body exceeded size limit/);
});

test("safeFetch enforces response size for direct stream consumers", async () => {
  const { response } = await safeFetch("https://b23.tv/large-stream", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: PUBLIC_LOOKUP,
    maxResponseBytes: 10,
    requestImpl: async () => ({
      status: 200,
      ok: true,
      body: Readable.from([Buffer.from("12345678901")])
    }),
    requireHTTPS: true
  });

  await assert.rejects(
    async () => {
      for await (const _chunk of response.body) {
      }
    },
    /response body exceeded size limit/
  );
});

test("safeFetch wraps a native Response without locking its exposed body", async () => {
  const { response } = await safeFetch("https://b23.tv/large-web-stream", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: PUBLIC_LOOKUP,
    maxResponseBytes: 10,
    requestImpl: async () => new Response("12345678901"),
    requireHTTPS: true
  });

  assert.equal(response.body.locked, false);
  await assert.rejects(response.text(), /response body exceeded size limit/);
});

test("safeFetch aborts an injected response stream when the deadline expires", async () => {
  const source = new Readable({ read() {} });
  const { response } = await safeFetch("https://b23.tv/slow-stream", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async () => ({ status: 200, ok: true, body: source }),
    requireHTTPS: true,
    timeoutMS: 10
  });

  await assert.rejects(
    async () => {
      for await (const _chunk of response.body) {
      }
    },
    /request timed out after 10ms/
  );
  assert.equal(source.destroyed, true);
});

test("safeFetch releases its deadline after a Web Stream reaches EOF", async () => {
  let transportSignal;
  const { response } = await safeFetch("https://b23.tv/web-stream", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async (_url, init) => {
      transportSignal = init.signal;
      return new Response("ok");
    },
    requireHTTPS: true,
    timeoutMS: 100
  });

  const reader = response.body.getReader();
  while (!(await reader.read()).done) {
  }
  await new Promise((resolve) => setTimeout(resolve, 125));

  assert.equal(transportSignal.aborted, false);
});

test("safeFetch discards an injected response that arrives after timeout", async () => {
  let resolveRequest;
  const request = safeFetch("https://b23.tv/late-response", {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: PUBLIC_LOOKUP,
    requestImpl: async () => new Promise((resolve) => {
      resolveRequest = resolve;
    }),
    requireHTTPS: true,
    timeoutMS: 10
  });

  await assert.rejects(request, /request timed out after 10ms/);

  const source = new Readable({ read() {} });
  resolveRequest({ status: 200, ok: true, body: source });
  await new Promise((resolve) => setImmediate(resolve));

  assert.equal(source.destroyed, true);
});
