const { buildNormalized, extractTags, guessTitle, stripUrlFromInput } = require("../lib/normalize");

const USER_AGENT =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36";

function unique(items) {
  return Array.from(new Set((items || []).map((item) => String(item || "").trim()).filter(Boolean)));
}

function cleanContent(value) {
  return String(value || "")
    .replace(/<[^>]+>/g, "")
    .replace(/\[话题\]/g, "")
    .replace(/\[[^\]]+\]/g, "")
    .replace(/&nbsp;/g, " ")
    .replace(/\s+\n/g, "\n")
    .replace(/\n\s+/g, "\n")
    .trim();
}

function normalizeMediaUrl(value) {
  const raw = String(value || "").trim();
  if (!raw) {
    return "";
  }
  if (raw.startsWith("//")) {
    return `https:${raw}`;
  }
  if (raw.startsWith("http://")) {
    return `https://${raw.slice("http://".length)}`;
  }
  return raw;
}

function buildEchoNote(input, normalized, reason) {
  const rawText = stripUrlFromInput(input);
  if (!rawText) {
    return null;
  }

  const title = guessTitle(input) || "小红书图文草稿";
  return {
    normalized,
    quality: "degraded",
    note: {
      title,
      content: rawText,
      tags: extractTags(rawText),
      images: [],
      videos: [],
      coverUrl: "",
      author: { name: "" },
      noteType: "unknown",
      likes: 0,
      comments: 0,
      favorites: 0
    },
    warnings: [`轻量抓取未拿到有效笔记内容，已回退到分享文本可见部分。原因：${reason}`]
  };
}

function buildDemoNote(input, normalized) {
  const title = guessTitle(input) || "小红书菜谱演示草稿";
  return {
    normalized,
    quality: "full",
    note: {
      title,
      content: "牛腩 500克\n番茄 3个\n土豆 2个\n先把牛腩焯水，再把番茄炒出沙，最后和土豆一起炖煮至软烂。",
      tags: ["stub", "演示数据"],
      images: [],
      videos: [],
      coverUrl: "",
      author: { name: "linkparse-sidecar-stub" },
      noteType: "image",
      likes: 0,
      comments: 0,
      favorites: 0
    },
    warnings: ["当前返回来自 xiaohongshu-importer 演示数据，不代表真实小红书解析结果。"]
  };
}

function extractInitialState(html) {
  const match = String(html || "").match(/window\.__INITIAL_STATE__=(.*?)<\/script>/s);
  if (!match || !match[1]) {
    return null;
  }

  const raw = match[1].trim().replace(/undefined/g, "null").replace(/;?\s*$/, "");
  try {
    return JSON.parse(raw);
  } catch (error) {
    return null;
  }
}

function pickNoteDetail(state) {
  const noteDetailMap = state?.note?.noteDetailMap;
  if (!noteDetailMap || typeof noteDetailMap !== "object") {
    return null;
  }

  for (const [noteId, detail] of Object.entries(noteDetailMap)) {
    if (!noteId || noteId === "null") {
      continue;
    }
    const note = detail?.note;
    if (note && typeof note === "object" && Object.keys(note).length > 0) {
      return { noteId, detail, note };
    }
  }

  return null;
}

function extractTitle(html, note) {
  const titleMatch = String(html || "").match(/<title>(.*?)<\/title>/s);
  const htmlTitle = titleMatch ? cleanContent(titleMatch[1]).replace(/\s*-\s*小红书$/, "") : "";
  const noteTitle = cleanContent(note?.title || note?.displayTitle || "");
  const fallbackTitle = noteTitle || htmlTitle;

  if (/页面不见了|404/i.test(fallbackTitle)) {
    return noteTitle;
  }

  return fallbackTitle;
}

function extractContent(html, note) {
  const divMatch = String(html || "").match(/<div id="detail-desc" class="desc">([\s\S]*?)<\/div>/);
  if (divMatch && divMatch[1]) {
    const cleaned = cleanContent(divMatch[1]);
    if (cleaned && cleaned !== "Content not found") {
      return cleaned;
    }
  }

  const content = cleanContent(note?.desc || note?.content || "");
  return content;
}

function extractImages(note) {
  const imageList = Array.isArray(note?.imageList) ? note.imageList : [];
  return unique(
    imageList
      .map((img) => normalizeMediaUrl(img?.urlDefault || img?.urlPre || img?.url || ""))
      .filter((url) => /^https?:\/\//i.test(url))
  );
}

function extractVideo(note) {
  const stream = note?.video?.media?.stream;
  if (!stream) {
    return [];
  }

  const candidates = [];
  for (const item of Array.isArray(stream.h264) ? stream.h264 : []) {
    candidates.push(normalizeMediaUrl(item?.masterUrl || item?.backupUrl || ""));
  }
  for (const item of Array.isArray(stream.h265) ? stream.h265 : []) {
    candidates.push(normalizeMediaUrl(item?.masterUrl || item?.backupUrl || ""));
  }

  return unique(candidates.filter((url) => /^https?:\/\//i.test(url)));
}

function extractAuthor(note, detail) {
  return (
    cleanContent(note?.user?.nickname || "") ||
    cleanContent(detail?.user?.nickname || "") ||
    cleanContent(note?.author?.nickname || "") ||
    ""
  );
}

function extractCounters(note) {
  return {
    likes: Number(note?.interactInfo?.likedCount || note?.interactInfo?.likeCount || 0) || 0,
    comments: Number(note?.interactInfo?.commentCount || 0) || 0,
    favorites: Number(note?.interactInfo?.collectedCount || note?.interactInfo?.collectCount || 0) || 0
  };
}

async function fetchHTML(inputUrl) {
  const response = await fetch(inputUrl, {
    headers: {
      "User-Agent": USER_AGENT,
      Referer: "https://www.xiaohongshu.com/"
    },
    redirect: "follow"
  });

  const html = await response.text();
  return {
    status: response.status,
    finalUrl: response.url || inputUrl,
    html
  };
}

async function parseViaImporter(input) {
  const normalizedInput = buildNormalized(input);
  if (!normalizedInput) {
    return {
      ok: false,
      errorCode: "invalid_input",
      errorMessage: "invalid xiaohongshu input"
    };
  }

  const fetched = await fetchHTML(normalizedInput.shareUrl);
  const fetchedNormalized = buildNormalized(fetched.finalUrl) || normalizedInput;
  const normalized = {
    shareUrl: normalizedInput.shareUrl,
    canonicalUrl: fetchedNormalized.canonicalUrl || normalizedInput.canonicalUrl,
    noteId: fetchedNormalized.noteId || normalizedInput.noteId,
    xsecToken: fetchedNormalized.xsecToken || normalizedInput.xsecToken
  };

  const state = extractInitialState(fetched.html);
  const detailEntry = pickNoteDetail(state);
  const note = detailEntry?.note || null;
  const title = extractTitle(fetched.html, note);
  const content = extractContent(fetched.html, note);
  const images = extractImages(note);
  const videos = extractVideo(note);
  const tags = unique(
    (Array.isArray(note?.tagList) ? note.tagList.map((tag) => tag?.name || "") : []).concat(extractTags(content))
  );
  const noteType = cleanContent(note?.type || (videos.length > 0 ? "video" : images.length > 0 ? "image" : "unknown")) || "unknown";
  const counters = extractCounters(note);
  const normalizedWithDetail = {
    ...normalized,
    noteId: normalized.noteId || detailEntry?.noteId || "",
    canonicalUrl: normalized.noteId || detailEntry?.noteId
      ? `https://www.xiaohongshu.com/explore/${normalized.noteId || detailEntry?.noteId}`
      : normalized.canonicalUrl
  };

  if (!content && images.length === 0 && videos.length === 0) {
    let reason = "note content not found";
    if (/\/404\b/.test(fetched.finalUrl) || /页面不见了/.test(title)) {
      reason = "redirected to xiaohongshu guard page";
    }

    return {
      ok: false,
      errorCode: "note_unavailable",
      errorMessage: reason,
      normalized: normalizedWithDetail
    };
  }

  const warnings = [];
  if (/\/404\b/.test(fetched.finalUrl) || /页面不见了/.test(title)) {
    warnings.push("页面命中了小红书守卫页，当前结果可能不完整。");
  }
  if (!detailEntry) {
    warnings.push("未从 __INITIAL_STATE__ 中提取到完整 noteDetailMap，结果可能不完整。");
  }

  return {
    ok: true,
    normalized: normalizedWithDetail,
    quality: "full",
    note: {
      title: title || guessTitle(input) || "小红书图文草稿",
      content,
      tags,
      images,
      videos,
      coverUrl: images[0] || "",
      author: {
        name: extractAuthor(note, detailEntry?.detail)
      },
      noteType,
      likes: counters.likes,
      comments: counters.comments,
      favorites: counters.favorites
    },
    warnings
  };
}

function createImporterProvider(config) {
  return {
    name: "importer",
    enabled: config.importerEnabled,
    requiresLogin: false,
    async parse(input) {
      const normalized = buildNormalized(input);
      if (!normalized) {
        return {
          ok: false,
          errorCode: "invalid_input",
          errorMessage: "invalid xiaohongshu input"
        };
      }

      if (config.stubMode === "demo") {
        return { ok: true, ...buildDemoNote(input, normalized) };
      }

      const live = await parseViaImporter(input).catch((error) => ({
        ok: false,
        errorCode: "internal_error",
        errorMessage: error instanceof Error ? error.message : String(error || "importer failed"),
        normalized
      }));

      if (live.ok) {
        return live;
      }

      if (config.stubMode === "echo") {
        const echoed = buildEchoNote(input, live.normalized || normalized, live.errorMessage || "note content not found");
        if (echoed) {
          return { ok: true, ...echoed };
        }
      }

      if (config.stubMode === "off") {
        return {
          ok: false,
          errorCode: live.errorCode || "note_unavailable",
          errorMessage: live.errorMessage || "xiaohongshu-importer provider failed"
        };
      }

      return {
        ok: false,
        errorCode: live.errorCode || "note_unavailable",
        errorMessage: live.errorMessage || "xiaohongshu-importer provider failed"
      };
    }
  };
}

module.exports = {
  createImporterProvider,
  normalizeMediaUrl
};
