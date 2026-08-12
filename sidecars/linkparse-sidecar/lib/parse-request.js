async function readJSON(req) {
  const chunks = [];
  for await (const chunk of req) {
    chunks.push(chunk);
  }
  const body = Buffer.concat(chunks).toString("utf8").trim();
  return body ? JSON.parse(body) : {};
}

function parseError(platform, providerRequested, code, message, status = 400) {
  return {
    status,
    payload: {
      ok: false,
      platform,
      providerRequested,
      providerUsed: "",
      error: {
        code,
        message,
        retryable: false
      },
      warnings: []
    }
  };
}

async function readParseRequest(req, platform, defaultProvider) {
  let payload;
  try {
    payload = await readJSON(req);
  } catch {
    return {
      error: parseError(platform, "", "invalid_input", "invalid json body")
    };
  }

  const input = String(payload.input || "").trim();
  const providerRequested = String(payload.provider || defaultProvider || "auto").trim().toLowerCase() || "auto";
  if (!input) {
    return {
      error: parseError(platform, providerRequested, "invalid_input", "input is required")
    };
  }

  return {
    input,
    providerRequested,
    includeDebug: !!payload.includeDebug,
    includeTranscript: !!payload.includeTranscript
  };
}

module.exports = {
  parseError,
  readParseRequest
};
