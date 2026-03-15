const FIRST_URL_PATTERN = /https?:\/\/[^\s]+/i;
const XHS_HOST_PATTERN = /(xiaohongshu\.com|xhslink\.com)/i;

function extractInputUrl(input) {
  const raw = String(input || "").trim();
  if (!raw) {
    return { ok: false, error: "url is required" };
  }

  let value = raw;
  const match = raw.match(FIRST_URL_PATTERN);
  if (match) {
    value = match[0].replace(/[。；;，,）)\]】>]+$/g, "");
  }

  if (!/^https?:\/\//i.test(value)) {
    value = `https://${value}`;
  }

  try {
    const parsed = new URL(value);
    if (!parsed.host) {
      return { ok: false, error: "invalid url" };
    }
    return { ok: true, url: parsed.toString() };
  } catch (error) {
    return { ok: false, error: "invalid url" };
  }
}

function isSupportedXHSUrl(input) {
  const result = extractInputUrl(input);
  if (!result.ok) {
    return false;
  }
  return XHS_HOST_PATTERN.test(new URL(result.url).host);
}

function stripUrlFromInput(input) {
  return String(input || "")
    .replace(FIRST_URL_PATTERN, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function extractTags(text) {
  const matches = String(text || "").match(/#([^\s#]+)/g) || [];
  return Array.from(new Set(matches.map((item) => item.replace(/^#/, "").trim()).filter(Boolean)));
}

function guessTitle(input) {
  const text = stripUrlFromInput(input)
    .replace(/发布了一篇小红书笔记.*$/g, "")
    .replace(/快来看吧.*$/g, "")
    .trim();
  if (!text) {
    return "";
  }

  const bracketMatch = text.match(/[【\[](.*?)[】\]]/);
  if (bracketMatch && bracketMatch[1]) {
    return bracketMatch[1].trim();
  }

  const lines = text.split(/[。\n]/).map((line) => line.trim()).filter(Boolean);
  return lines[0] || "";
}

function detectNoteId(inputUrl) {
  const result = extractInputUrl(inputUrl);
  if (!result.ok) {
    return "";
  }

  try {
    const parsed = new URL(result.url);
    const redirected = extractRedirectPath(parsed);
    if (parsed.pathname === "/404" && redirected) {
      return detectNoteId(redirected);
    }
    const segments = parsed.pathname.split("/").filter(Boolean);
    const last = segments[segments.length - 1] || "";
    return /^[A-Za-z0-9]+$/.test(last) ? last : "";
  } catch (error) {
    return "";
  }
}

function buildNormalized(input) {
  const result = extractInputUrl(input);
  if (!result.ok) {
    return null;
  }

  let resolvedUrl = result.url;
  try {
    const parsed = new URL(result.url);
    const redirectPath = extractRedirectPath(parsed);
    if (parsed.pathname === "/404" && redirectPath) {
      resolvedUrl = redirectPath;
    }
  } catch (error) {
    resolvedUrl = result.url;
  }

  const noteId = detectNoteId(resolvedUrl);
  let canonicalUrl = resolvedUrl;
  if (noteId && /xiaohongshu\.com/i.test(canonicalUrl)) {
    canonicalUrl = `https://www.xiaohongshu.com/explore/${noteId}`;
  }

  let xsecToken = "";
  try {
    xsecToken = new URL(resolvedUrl).searchParams.get("xsec_token") || "";
  } catch (error) {
    xsecToken = "";
  }

  return {
    shareUrl: result.url,
    canonicalUrl,
    noteId,
    xsecToken
  };
}

function extractRedirectPath(parsed) {
  if (!parsed) {
    return "";
  }

  const direct = parsed.searchParams.get("redirectPath");
  if (direct) {
    return direct;
  }

  const source = parsed.searchParams.get("source");
  if (!source || !source.includes("redirectPath=")) {
    return "";
  }

  const idx = source.indexOf("redirectPath=");
  if (idx < 0) {
    return "";
  }

  const tail = source.slice(idx + "redirectPath=".length);
  const cutIndex = tail.search(/&(error_code|error_msg|uuid)=/);
  const encoded = cutIndex >= 0 ? tail.slice(0, cutIndex) : tail;
  try {
    return decodeURIComponent(encoded);
  } catch (error) {
    return encoded;
  }
}

module.exports = {
  extractInputUrl,
  isSupportedXHSUrl,
  stripUrlFromInput,
  extractTags,
  guessTitle,
  detectNoteId,
  buildNormalized,
  extractRedirectPath
};
