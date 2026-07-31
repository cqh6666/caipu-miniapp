const http = require("node:http");
const https = require("node:https");
const { Transform } = require("node:stream");

const {
  assertURLAllowed,
  createPinnedLookup,
  resolvePublicAddresses
} = require("./url-policy");

const REDIRECT_STATUSES = new Set([301, 302, 303, 307, 308]);
const DEFAULT_MAX_REDIRECTS = 5;
const DEFAULT_MAX_RESPONSE_BYTES = 5 * 1024 * 1024;
const DEFAULT_TIMEOUT_MS = 20000;

function headerValue(headers, name) {
  if (!headers) {
    return "";
  }
  if (typeof headers.get === "function") {
    return String(headers.get(name) || "").trim();
  }
  const value = headers[String(name || "").toLowerCase()];
  return Array.isArray(value) ? value.join(", ") : String(value || "").trim();
}

function chunkBytes(chunk) {
  if (Buffer.isBuffer(chunk) || chunk instanceof Uint8Array) {
    return chunk.byteLength;
  }
  return Buffer.byteLength(String(chunk || ""), "utf8");
}

function limitNodeStream(source, maxResponseBytes, signal) {
  if (!source || !(maxResponseBytes > 0)) {
    return source;
  }

  let total = 0;
  const limiter = new Transform({
    transform(chunk, _encoding, callback) {
      total += chunkBytes(chunk);
      if (total > maxResponseBytes) {
        source.destroy();
        callback(new Error("response body exceeded size limit"));
        return;
      }
      callback(null, chunk);
    }
  });
  const abort = () => {
    const reason = signal.reason instanceof Error ? signal.reason : new Error("request aborted");
    source.destroy(reason);
    limiter.destroy(reason);
  };
  source.on("error", (error) => limiter.destroy(error));
  limiter.on("close", () => {
    if (signal) {
      signal.removeEventListener("abort", abort);
    }
    if (!source.destroyed) {
      source.destroy();
    }
  });
  if (signal) {
    if (signal.aborted) {
      abort();
    } else {
      signal.addEventListener("abort", abort, { once: true });
    }
  }
  source.pipe(limiter);
  return limiter;
}

function limitWebStream(source, maxResponseBytes, signal) {
  if (!source || typeof source.pipeThrough !== "function" || typeof TransformStream !== "function" || !(maxResponseBytes > 0)) {
    return source;
  }

  let total = 0;
  return source.pipeThrough(new TransformStream({
    transform(chunk, controller) {
      total += chunkBytes(chunk);
      if (total > maxResponseBytes) {
        controller.error(new Error("response body exceeded size limit"));
        return;
      }
      controller.enqueue(chunk);
    }
  }), signal ? { signal } : undefined);
}

function observeWebStream(source, signal, onSettled) {
  if (!source || typeof source.getReader !== "function" || typeof ReadableStream !== "function") {
    return source;
  }

  const reader = source.getReader();
  return new ReadableStream({
    async pull(controller) {
      try {
        const { done, value } = await waitWithSignal(reader.read(), signal);
        if (done) {
          onSettled();
          controller.close();
          return;
        }
        controller.enqueue(value);
      } catch (error) {
        onSettled();
        controller.error(error);
      }
    },
    async cancel(reason) {
      try {
        await waitWithSignal(reader.cancel(reason), signal);
      } finally {
        onSettled();
      }
    }
  });
}

function buildResponse(message, targetURL, maxResponseBytes, signal) {
  let consumed = false;
  let buffered;
  const body = limitNodeStream(message, maxResponseBytes, signal);

  async function readBuffer() {
    if (consumed) {
      throw new Error("response body has already been consumed");
    }
    consumed = true;
    const chunks = [];
    let total = 0;
    for await (const chunk of body) {
      const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
      total += buffer.length;
      if (maxResponseBytes > 0 && total > maxResponseBytes) {
        body.destroy();
        throw new Error("response body exceeded size limit");
      }
      chunks.push(buffer);
    }
    buffered = Buffer.concat(chunks);
    return buffered;
  }

  return {
    ok: Number(message.statusCode) >= 200 && Number(message.statusCode) < 300,
    status: Number(message.statusCode) || 0,
    url: targetURL.toString(),
    headers: {
      get(name) {
        return headerValue(message.headers, name);
      }
    },
    body,
    async text() {
      return (buffered || (await readBuffer())).toString("utf8");
    },
    async json() {
      return JSON.parse((buffered || (await readBuffer())).toString("utf8"));
    }
  };
}

function buildRequestHeaders(headers, targetURL, initialOrigin) {
  const source = typeof headers === "function" ? headers(targetURL) : headers;
  const entries = source && typeof source.entries === "function"
    ? Array.from(source.entries())
    : Object.entries(source || {});
  const crossOrigin = targetURL.origin !== initialOrigin;

  return Object.fromEntries(
    entries.filter(([name]) => {
      const normalized = String(name || "").toLowerCase();
      return !crossOrigin || (normalized !== "authorization" && normalized !== "cookie" && normalized !== "proxy-authorization");
    })
  );
}

function waitWithSignal(promise, signal) {
  if (!signal) {
    return promise;
  }
  if (signal.aborted) {
    Promise.resolve(promise).catch(() => {});
    return Promise.reject(signal.reason || new Error("request aborted"));
  }

  return new Promise((resolve, reject) => {
    const onAbort = () => reject(signal.reason || new Error("request aborted"));
    signal.addEventListener("abort", onAbort, { once: true });
    promise.then(
      (value) => {
        signal.removeEventListener("abort", onAbort);
        resolve(value);
      },
      (error) => {
        signal.removeEventListener("abort", onAbort);
        reject(error);
      }
    );
  });
}

function cloneWebResponseWithBody(response, body) {
  const wrapped = new Response(body, {
    headers: response.headers,
    status: response.status,
    statusText: response.statusText
  });
  for (const property of ["redirected", "type", "url"]) {
    Object.defineProperty(wrapped, property, {
      configurable: true,
      value: response[property]
    });
  }
  return wrapped;
}

function enforceInjectedResponseLimit(response, maxResponseBytes, signal) {
  if (!response || !(maxResponseBytes > 0)) {
    return response;
  }

  if (response.body && typeof response.body.pipe === "function") {
    response.body = limitNodeStream(response.body, maxResponseBytes, signal);
  } else if (response.body && typeof response.body.pipeThrough === "function") {
    const body = limitWebStream(response.body, maxResponseBytes, signal);
    response = typeof Response === "function" && response instanceof Response
      ? cloneWebResponseWithBody(response, body)
      : { ...response, body };
  }

  if (typeof response.text === "function") {
    const readText = response.text.bind(response);
    response.text = async () => {
      const value = await readText();
      if (Buffer.byteLength(String(value || ""), "utf8") > maxResponseBytes) {
        throw new Error("response body exceeded size limit");
      }
      return value;
    };
  }
  if (typeof response.json === "function") {
    const readJSON = response.json.bind(response);
    response.json = async () => {
      const value = await readJSON();
      if (Buffer.byteLength(JSON.stringify(value) || "", "utf8") > maxResponseBytes) {
        throw new Error("response body exceeded size limit");
      }
      return value;
    };
  }
  return response;
}

async function requestPinned(targetURL, options) {
  const addresses = await waitWithSignal(
    resolvePublicAddresses(targetURL.hostname, options.lookupImpl),
    options.signal
  );
  const pinnedLookup = createPinnedLookup(addresses);
  if (typeof options.requestImpl === "function") {
    const requestPromise = Promise.resolve().then(() => options.requestImpl(targetURL.toString(), {
        approvedAddresses: addresses.map((item) => ({ ...item })),
        body: options.body,
        headers: options.headers,
        lookup: pinnedLookup,
        method: options.method,
        redirect: "manual",
        signal: options.signal
      }));
    let response;
    try {
      response = await waitWithSignal(requestPromise, options.signal);
    } catch (error) {
      if (options.signal && options.signal.aborted) {
        requestPromise.then(
          (lateResponse) => discardResponse(lateResponse, options.signal).catch(() => {}),
          () => {}
        );
      }
      throw error;
    }
    const contentLength = Number(headerValue(response && response.headers, "content-length")) || 0;
    if (options.maxResponseBytes > 0 && contentLength > options.maxResponseBytes) {
      await discardResponse(response, options.signal);
      throw new Error("response body exceeded size limit");
    }
    return enforceInjectedResponseLimit(response, options.maxResponseBytes, options.signal);
  }

  return new Promise((resolve, reject) => {
    const client = targetURL.protocol === "https:" ? https : http;
    const request = client.request(
      targetURL,
      {
        agent: false,
        method: options.method,
        headers: options.headers,
        lookup: pinnedLookup,
        signal: options.signal
      },
      (message) => {
        const contentLength = Number(headerValue(message.headers, "content-length")) || 0;
        if (options.maxResponseBytes > 0 && contentLength > options.maxResponseBytes) {
          message.destroy();
          reject(new Error("response body exceeded size limit"));
          return;
        }
        resolve(buildResponse(message, targetURL, options.maxResponseBytes, options.signal));
      }
    );

    request.on("error", reject);
    if (options.timeoutMS > 0) {
      request.setTimeout(options.timeoutMS, () => request.destroy(new Error(`request timed out after ${options.timeoutMS}ms`)));
    }
    if (options.body == null) {
      request.end();
      return;
    }
    if (typeof options.body.pipe === "function") {
      options.body.pipe(request);
      return;
    }
    request.end(options.body);
  });
}

async function waitForCleanup(action, signal) {
  const cleanupPromise = Promise.resolve().then(action);
  cleanupPromise.catch(() => {});
  try {
    await waitWithSignal(cleanupPromise, signal);
  } catch (error) {
    if (signal && signal.aborted) {
      throw signal.reason || error;
    }
  }
}

async function discardResponse(response, signal) {
  if (!response) {
    return;
  }
  if (response.body && typeof response.body.cancel === "function") {
    await waitForCleanup(() => response.body.cancel(), signal);
    return;
  }
  if (response.body && typeof response.body.destroy === "function") {
    try {
      response.body.destroy();
    } catch {
    }
    if (signal && signal.aborted) {
      throw signal.reason || new Error("request aborted");
    }
    return;
  }
  if (typeof response.text === "function") {
    await waitForCleanup(() => response.text(), signal);
    return;
  }
  if (typeof response.json === "function") {
    await waitForCleanup(() => response.json(), signal);
  }
}

function createDeadlineSignal(externalSignal, timeoutMS) {
  const controller = new AbortController();
  const abortFromExternal = () => controller.abort(externalSignal.reason);
  if (externalSignal) {
    if (externalSignal.aborted) {
      abortFromExternal();
    } else {
      externalSignal.addEventListener("abort", abortFromExternal, { once: true });
    }
  }
  const timer = timeoutMS > 0
    ? setTimeout(() => controller.abort(new Error(`request timed out after ${timeoutMS}ms`)), timeoutMS)
    : null;

  return {
    signal: controller.signal,
    cancel() {
      if (timer) {
        clearTimeout(timer);
      }
      if (externalSignal) {
        externalSignal.removeEventListener("abort", abortFromExternal);
      }
    }
  };
}

function bindResponseDeadline(response, deadline) {
  if (!response) {
    deadline.cancel();
    return response;
  }

  let released = false;
  const release = () => {
    if (!released) {
      released = true;
      deadline.cancel();
    }
  };
  let body = response.body;
  if (body && typeof body.getReader === "function") {
    body = observeWebStream(body, deadline.signal, release);
    response = typeof Response === "function" && response instanceof Response
      ? cloneWebResponseWithBody(response, body)
      : { ...response, body };
  }
  if (body && typeof body.once === "function") {
    body.once("end", release);
    body.once("close", release);
    body.once("error", release);
    if (body.readableEnded || body.destroyed) {
      release();
    }
  }
  if (body && typeof body.cancel === "function") {
    const cancelBody = body.cancel.bind(body);
    body.cancel = async (...args) => {
      try {
        return await waitWithSignal(cancelBody(...args), deadline.signal);
      } finally {
        release();
      }
    };
  }
  for (const method of ["text", "json"]) {
    if (typeof response[method] !== "function") {
      continue;
    }
    const original = response[method].bind(response);
    response[method] = async (...args) => {
      try {
        return await waitWithSignal(original(...args), deadline.signal);
      } finally {
        release();
      }
    };
  }
  if (!body && typeof response.text !== "function" && typeof response.json !== "function") {
    release();
  }
  return response;
}

function assertFetchURL(input, allowedDomains, requireHTTPS) {
  const parsed = assertURLAllowed(input, allowedDomains);
  if (requireHTTPS && (parsed.protocol !== "https:" || parsed.port !== "")) {
    throw new Error("outbound URL must use HTTPS and the default port");
  }
  return parsed;
}

async function safeFetch(rawURL, options = {}) {
  const allowedDomains = Array.isArray(options.allowedDomains) ? options.allowedDomains : [];
  const maxRedirects = Number.isInteger(options.maxRedirects) && options.maxRedirects >= 0
    ? options.maxRedirects
    : DEFAULT_MAX_REDIRECTS;
  const requestOptions = {
    body: options.body,
    headers: options.headers || {},
    lookupImpl: options.lookupImpl,
    maxResponseBytes: Number(options.maxResponseBytes) > 0
      ? Number(options.maxResponseBytes)
      : DEFAULT_MAX_RESPONSE_BYTES,
    method: String(options.method || "GET").toUpperCase(),
    requestImpl: options.requestImpl,
    signal: undefined,
    timeoutMS: Number(options.timeoutMS) > 0 ? Number(options.timeoutMS) : DEFAULT_TIMEOUT_MS
  };
  const redirectMode = options.redirectMode === "error" ? "error" : "follow";
  const requireHTTPS = !!options.requireHTTPS;
  let currentURL = assertFetchURL(rawURL, allowedDomains, requireHTTPS);
  const initialOrigin = currentURL.origin;
  let redirectCount = 0;
  const deadline = createDeadlineSignal(options.signal, requestOptions.timeoutMS);
  requestOptions.signal = deadline.signal;
  let responseOwnsDeadline = false;

  try {
    for (;;) {
      const currentRequestOptions = {
        ...requestOptions,
        headers: buildRequestHeaders(requestOptions.headers, currentURL, initialOrigin)
      };
      const response = await requestPinned(currentURL, currentRequestOptions);
      const status = Number(response && response.status) || 0;
      if (!REDIRECT_STATUSES.has(status)) {
        responseOwnsDeadline = true;
        return { response: bindResponseDeadline(response, deadline), finalURL: currentURL.toString() };
      }

      const location = headerValue(response && response.headers, "location");
      await discardResponse(response, deadline.signal);
      if (redirectMode === "error") {
        throw new Error("redirect is not allowed for this request");
      }
      if (!location) {
        throw new Error("redirect response is missing Location");
      }
      if (redirectCount >= maxRedirects) {
        throw new Error("too many redirects");
      }

      const candidateURL = new URL(location, currentURL);
      if (currentURL.protocol === "https:" && candidateURL.protocol !== "https:") {
        throw new Error("HTTPS redirect downgrade is not allowed");
      }
      currentURL = assertFetchURL(candidateURL, allowedDomains, requireHTTPS);
      redirectCount += 1;
    }
  } finally {
    if (!responseOwnsDeadline) {
      deadline.cancel();
    }
  }
}

module.exports = {
  buildRequestHeaders,
  discardResponse,
  headerValue,
  safeFetch
};
