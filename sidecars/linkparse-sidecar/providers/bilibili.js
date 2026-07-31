const { discardResponse, safeFetch } = require("../lib/safe-fetch");
const {
  BILIBILI_ASSET_DOMAINS,
  BILIBILI_INPUT_DOMAINS,
  extractInputURL,
  hostMatches,
  isTrustedBilibiliAPIURL,
  normalizeToHTTPSURL
} = require("../lib/url-policy");

const BVID_PATTERN = /(BV[0-9A-Za-z]{10})/i;
const AVID_PATTERN = /(?:^|\/|[?&])av([0-9]+)/i;
const PREFERRED_SUBTITLE_LANGS = ["zh-CN", "zh-Hans", "zh-Hant", "zh", "ai-zh"];
const USER_AGENT =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36";

function safeTrim(value) {
  return String(value || "").trim();
}

function extractInputUrl(input) {
  return extractInputURL(input);
}

function isResolvableBilibiliHost(host) {
  return hostMatches(host, BILIBILI_INPUT_DOMAINS);
}

function isSupportedBilibiliUrl(input) {
  const extracted = extractInputUrl(input);
  if (!extracted.ok) {
    return false;
  }

  try {
    normalizeToHTTPSURL(extracted.url, BILIBILI_INPUT_DOMAINS);
    return true;
  } catch (error) {
    return false;
  }
}

function parseVideoRefFromURL(rawURL) {
  try {
    const parsed = normalizeToHTTPSURL(rawURL, BILIBILI_INPUT_DOMAINS);

    let page = Number(parsed.searchParams.get("p") || 1);
    if (!(page > 0)) {
      page = 1;
    }

    const normalizedURL = parsed.toString();
    const bvidMatch = normalizedURL.match(BVID_PATTERN);
    if (bvidMatch && bvidMatch[1]) {
      return {
        bvid: bvidMatch[1],
        aid: 0,
        page,
        url: normalizedURL
      };
    }

    const avidMatch = normalizedURL.match(AVID_PATTERN);
    if (avidMatch && avidMatch[1]) {
      return {
        bvid: "",
        aid: Number(avidMatch[1]) || 0,
        page,
        url: normalizedURL
      };
    }

    return null;
  } catch (error) {
    return null;
  }
}

function buildHeaders(sessdata, targetURL) {
  const headers = {
    "User-Agent": USER_AGENT,
    Referer: "https://www.bilibili.com/"
  };
  if (safeTrim(sessdata) && isTrustedBilibiliAPIURL(targetURL)) {
    headers.Cookie = `SESSDATA=${safeTrim(sessdata)}`;
  }
  return headers;
}

async function resolveFinalURL(rawURL, config) {
  const { response, finalURL } = await safeFetch(rawURL, {
    allowedDomains: BILIBILI_INPUT_DOMAINS,
    lookupImpl: config.lookupImpl,
    maxResponseBytes: 1024 * 1024,
    method: "GET",
    headers: buildHeaders("", rawURL),
    requestImpl: config.requestImpl,
    requireHTTPS: true
  });

  // B 站短链有时会在最终视频页返回 412，但 fetch 仍然已经跟到了可解析
  // 的 canonical URL。这里优先使用最终 URL，而不是把 412 直接视为失败。
  if (!response) {
    throw new Error("failed to resolve bilibili url: HTTP 0");
  }
  await discardResponse(response);
  return finalURL;
}

async function resolveVideoRef(input, config) {
  const extracted = extractInputUrl(input);
  if (!extracted.ok) {
    throw new Error("invalid bilibili url");
  }
  if (!isSupportedBilibiliUrl(extracted.url)) {
    throw new Error("only bilibili links are supported");
  }
  const normalizedInputURL = normalizeToHTTPSURL(extracted.url, BILIBILI_INPUT_DOMAINS).toString();

  const directRef = parseVideoRefFromURL(normalizedInputURL);
  if (directRef) {
    return { ref: directRef, shareUrl: normalizedInputURL, warnings: [] };
  }

  let parsed;
  try {
    parsed = new URL(normalizedInputURL);
  } catch (error) {
    throw new Error("invalid bilibili url");
  }
  if (!isResolvableBilibiliHost(parsed.hostname)) {
    throw new Error("only bilibili links are supported");
  }

  const resolvedURL = await resolveFinalURL(normalizedInputURL, config);
  const resolvedRef = parseVideoRefFromURL(resolvedURL);
  if (!resolvedRef) {
    throw new Error("could not extract BV/AV id from bilibili url");
  }

  return {
    ref: resolvedRef,
    shareUrl: normalizedInputURL,
    warnings: ["已自动展开 B 站短链接。"]
  };
}

async function fetchJSON(url, options, config) {
  const credentialed = !!options.credentialed;
  if (credentialed && !isTrustedBilibiliAPIURL(url)) {
    throw new Error("credentials are not allowed for this bilibili URL");
  }
  const { response } = await safeFetch(url, {
    allowedDomains: BILIBILI_ASSET_DOMAINS,
    lookupImpl: config.lookupImpl,
    maxResponseBytes: 4 * 1024 * 1024,
    method: "GET",
    redirectMode: credentialed ? "error" : "follow",
    headers: (targetURL) => buildHeaders(credentialed ? options.sessdata : "", targetURL),
    requestImpl: config.requestImpl,
    requireHTTPS: true
  });

  if (!response || !response.ok) {
    await discardResponse(response);
    throw new Error(`bilibili request failed: HTTP ${response ? response.status : 0}`);
  }

  return response.json();
}

async function fetchView(ref, sessdata, config) {
  const params = new URLSearchParams();
  if (ref.bvid) {
    params.set("bvid", ref.bvid);
  }
  if (ref.aid > 0) {
    params.set("aid", String(ref.aid));
  }

  const payload = await fetchJSON(
    `https://api.bilibili.com/x/web-interface/view?${params.toString()}`,
    { credentialed: true, sessdata },
    config
  );
  if (Number(payload.code) !== 0) {
    throw new Error(safeTrim(payload.message) || "failed to fetch bilibili video info");
  }
  if (!payload.data || !payload.data.bvid || !Array.isArray(payload.data.pages) || payload.data.pages.length === 0) {
    throw new Error("bilibili video info is incomplete");
  }

  return payload;
}

function pickPage(pages, requestedPage) {
  const list = Array.isArray(pages) ? pages : [];
  const normalizedPage = requestedPage > 0 ? requestedPage : 1;
  for (const page of list) {
    if (Number(page?.page) === normalizedPage) {
      return { page, warnings: [] };
    }
  }

  return {
    page: list[0] || { cid: 0, page: 1, part: "" },
    warnings: list.length > 0 ? ["请求的分 P 不存在，已回退到第一页。"] : []
  };
}

async function fetchSubtitles(bvid, cid, sessdata, config) {
  const payload = await fetchJSON(
    `https://api.bilibili.com/x/player/v2?bvid=${encodeURIComponent(bvid)}&cid=${encodeURIComponent(cid)}`,
    { credentialed: true, sessdata },
    config
  );
  if (Number(payload.code) !== 0) {
    throw new Error(safeTrim(payload.message) || "failed to fetch bilibili subtitles");
  }

  return {
    needLoginSubtitle: !!payload?.data?.need_login_subtitle,
    subtitles: Array.isArray(payload?.data?.subtitle?.subtitles) ? payload.data.subtitle.subtitles : []
  };
}

function selectSubtitle(items) {
  for (const lang of PREFERRED_SUBTITLE_LANGS) {
    for (const item of items || []) {
      if (safeTrim(item?.lan).toLowerCase() === lang.toLowerCase() && safeTrim(item?.subtitle_url)) {
        return item;
      }
    }
  }

  for (const item of items || []) {
    if (safeTrim(item?.subtitle_url)) {
      return item;
    }
  }

  return null;
}

async function fetchSubtitleFile(subtitleURL, config) {
  let resolvedURL = safeTrim(subtitleURL);
  if (resolvedURL.startsWith("//")) {
    resolvedURL = `https:${resolvedURL}`;
  } else if (resolvedURL.startsWith("/")) {
    resolvedURL = `https://api.bilibili.com${resolvedURL}`;
  }
  resolvedURL = normalizeToHTTPSURL(resolvedURL, BILIBILI_ASSET_DOMAINS).toString();

  return fetchJSON(resolvedURL, { credentialed: false, sessdata: "" }, config);
}

function buildSubtitleText(file) {
  const lines = [];
  for (const item of Array.isArray(file?.body) ? file.body : []) {
    const line = safeTrim(item?.content);
    if (line) {
      lines.push(line);
    }
  }

  return {
    text: lines.join("\n"),
    segments: lines.length
  };
}

function createBilibiliProvider(config) {
  return {
    name: "openapi",
    enabled: config.bilibiliOpenAPIEnabled,
    requiresLogin: false,
    async parse(input, options = {}) {
      const sessdata = safeTrim(options.sessdata);
      const includeTranscript = !!options.includeTranscript;

      const { ref, shareUrl, warnings: resolveWarnings } = await resolveVideoRef(input, config);
      const view = await fetchView(ref, sessdata, config);
      const { page, warnings: pageWarnings } = pickPage(view.data.pages, ref.page);
      const warnings = resolveWarnings.concat(pageWarnings);

      let transcript = "";
      let transcriptStatus = includeTranscript ? "skipped" : "disabled";
      let transcriptError = "";
      let subtitleLanguage = "";
      let subtitleSegments = 0;
      let quality = "full";

      if (includeTranscript) {
        const subtitleResult = await fetchSubtitles(view.data.bvid, page.cid, sessdata, config);
        const selectedSubtitle = selectSubtitle(subtitleResult.subtitles);
        if (!selectedSubtitle) {
          quality = "degraded";
          if (subtitleResult.needLoginSubtitle && !sessdata) {
            warnings.push("当前字幕需要登录态，未提供 B 站 SESSDATA。");
          } else {
            warnings.push("当前视频没有可直接访问的字幕。");
          }
        } else {
          const subtitleFile = await fetchSubtitleFile(selectedSubtitle.subtitle_url, config);
          const built = buildSubtitleText(subtitleFile);
          transcript = built.text;
          subtitleSegments = built.segments;
          subtitleLanguage = safeTrim(selectedSubtitle.lan_doc) || safeTrim(selectedSubtitle.lan);
          transcriptStatus = transcript ? "success" : "skipped";
          if (!transcript) {
            quality = "degraded";
            warnings.push("字幕列表存在，但未提取到可用文本。");
          }
        }
      }

      return {
        ok: true,
        normalized: {
          shareUrl,
          canonicalUrl: ref.url,
          id: safeTrim(view.data.bvid) || (view.data.aid ? String(view.data.aid) : ""),
          bvid: safeTrim(view.data.bvid),
          aid: Number(view.data.aid) || 0,
          cid: Number(page.cid) || 0,
          page: Number(page.page) || 1
        },
        content: {
          title: safeTrim(view.data.title) || safeTrim(page.part) || "B站视频菜谱草稿",
          description: safeTrim(view.data.desc),
          body: "",
          part: safeTrim(page.part),
          transcript,
          transcriptStatus,
          transcriptError,
          tags: [],
          images: [],
          videos: [],
          coverUrl: safeTrim(view.data.pic),
          author: {
            name: safeTrim(view?.data?.owner?.name),
            avatarUrl: ""
          },
          contentType: "video",
          likes: 0,
          comments: 0,
          favorites: 0,
          subtitleLanguage,
          subtitleSegments
        },
        warnings,
        quality
      };
    }
  };
}

module.exports = {
  buildSubtitleText,
  createBilibiliProvider,
  isSupportedBilibiliUrl,
  parseVideoRefFromURL,
  selectSubtitle
};
