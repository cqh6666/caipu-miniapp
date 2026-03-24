const FIRST_URL_PATTERN = /https?:\/\/[^\s]+/i;
const BVID_PATTERN = /(BV[0-9A-Za-z]{10})/i;
const AVID_PATTERN = /(?:^|\/|[?&])av([0-9]+)/i;
const PREFERRED_SUBTITLE_LANGS = ["zh-CN", "zh-Hans", "zh-Hant", "zh", "ai-zh"];
const USER_AGENT =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36";

function safeTrim(value) {
  return String(value || "").trim();
}

function extractInputUrl(input) {
  const raw = safeTrim(input);
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

function isResolvableBilibiliHost(host) {
  const value = safeTrim(host).toLowerCase();
  return value.includes("bilibili.com") || value.includes("b23.tv") || value.includes("bili2233.cn");
}

function isSupportedBilibiliUrl(input) {
  const extracted = extractInputUrl(input);
  if (!extracted.ok) {
    return false;
  }

  try {
    return isResolvableBilibiliHost(new URL(extracted.url).host);
  } catch (error) {
    return false;
  }
}

function parseVideoRefFromURL(rawURL) {
  try {
    const parsed = new URL(rawURL);
    if (!isResolvableBilibiliHost(parsed.host)) {
      return null;
    }

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

function buildHeaders(sessdata) {
  const headers = {
    "User-Agent": USER_AGENT,
    Referer: "https://www.bilibili.com/"
  };
  if (safeTrim(sessdata)) {
    headers.Cookie = `SESSDATA=${safeTrim(sessdata)}`;
  }
  return headers;
}

async function resolveFinalURL(rawURL, sessdata, config) {
  const fetchImpl = config.fetchImpl || globalThis.fetch;
  if (typeof fetchImpl !== "function") {
    throw new Error("fetch is not available");
  }

  const response = await fetchImpl(rawURL, {
    method: "GET",
    redirect: "follow",
    headers: buildHeaders(sessdata)
  });

  if (!response || !response.ok) {
    throw new Error(`failed to resolve bilibili url: HTTP ${response ? response.status : 0}`);
  }

  if (typeof response.text === "function") {
    await response.text().catch(() => {});
  }

  return safeTrim(response.url) || rawURL;
}

async function resolveVideoRef(input, sessdata, config) {
  const extracted = extractInputUrl(input);
  if (!extracted.ok) {
    throw new Error("invalid bilibili url");
  }

  const directRef = parseVideoRefFromURL(extracted.url);
  if (directRef) {
    return { ref: directRef, shareUrl: extracted.url, warnings: [] };
  }

  let parsed;
  try {
    parsed = new URL(extracted.url);
  } catch (error) {
    throw new Error("invalid bilibili url");
  }
  if (!isResolvableBilibiliHost(parsed.host)) {
    throw new Error("only bilibili links are supported");
  }

  const resolvedURL = await resolveFinalURL(extracted.url, sessdata, config);
  const resolvedRef = parseVideoRefFromURL(resolvedURL);
  if (!resolvedRef) {
    throw new Error("could not extract BV/AV id from bilibili url");
  }

  return {
    ref: resolvedRef,
    shareUrl: extracted.url,
    warnings: ["已自动展开 B 站短链接。"]
  };
}

async function fetchJSON(url, sessdata, config) {
  const fetchImpl = config.fetchImpl || globalThis.fetch;
  if (typeof fetchImpl !== "function") {
    throw new Error("fetch is not available");
  }

  const response = await fetchImpl(url, {
    method: "GET",
    redirect: "follow",
    headers: buildHeaders(sessdata)
  });

  if (!response || !response.ok) {
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

  const payload = await fetchJSON(`https://api.bilibili.com/x/web-interface/view?${params.toString()}`, sessdata, config);
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
    sessdata,
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

async function fetchSubtitleFile(subtitleURL, sessdata, config) {
  let resolvedURL = safeTrim(subtitleURL);
  if (resolvedURL.startsWith("//")) {
    resolvedURL = `https:${resolvedURL}`;
  } else if (resolvedURL.startsWith("/")) {
    resolvedURL = `https://api.bilibili.com${resolvedURL}`;
  }

  return fetchJSON(resolvedURL, sessdata, config);
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

      const { ref, shareUrl, warnings: resolveWarnings } = await resolveVideoRef(input, sessdata, config);
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
          const subtitleFile = await fetchSubtitleFile(selectedSubtitle.subtitle_url, sessdata, config);
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
