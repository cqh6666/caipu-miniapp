const { buildNormalized, extractTags, guessTitle, stripUrlFromInput } = require("../lib/normalize");
const {
  buildBrowserLaunchOptions,
  canUsePlaywright,
  defaultCookiePath,
  loadCookies,
  summarizeRuntime
} = require("../lib/rednote-runtime");

function unique(items) {
  return Array.from(new Set((items || []).map((item) => String(item || "").trim()).filter(Boolean)));
}

function normalizeMediaUrl(value) {
  const raw = String(value || "").trim();
  if (!raw) {
    return "";
  }
  if (raw.startsWith("//")) {
    return `https:${raw}`;
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
      author: { name: "rednote-fallback" },
      noteType: "unknown",
      likes: 0,
      comments: 0,
      favorites: 0
    },
    warnings: [`RedNote 抓取失败，已回退到分享文本可见部分。原因：${reason}`]
  };
}

function buildDemoNote(input, normalized) {
  const title = guessTitle(input) || "小红书菜谱演示草稿";
  return {
    normalized,
    quality: "full",
    note: {
      title,
      content: "牛腩 500克\n番茄 3个\n洋葱 半个\n先焯水，再翻炒番茄，最后小火慢炖到入味。",
      tags: ["stub", "rednote", "演示数据"],
      images: [],
      videos: [],
      coverUrl: "",
      author: { name: "rednote-stub" },
      noteType: "image",
      likes: 0,
      comments: 0,
      favorites: 0
    },
    warnings: ["当前返回来自 RedNote 演示数据，不代表真实小红书解析结果。"]
  };
}

function parseChineseCounter(raw) {
  const value = String(raw || "").trim();
  if (!value) {
    return 0;
  }
  if (value.includes("万")) {
    return Math.round(Number(value.replace("万", "").trim()) * 10000);
  }
  return Number(value) || 0;
}

async function extractNoteFromPage(page) {
  await page.waitForLoadState("domcontentloaded");
  await page.waitForTimeout(1200);

  if (await page.locator(".login-container").count()) {
    throw new Error("login required");
  }

  await page.waitForSelector(".note-container", { timeout: 12000 });

  return page.evaluate(() => {
    const parseCounter = (value) => {
      const raw = String(value || "").trim();
      if (!raw) return 0;
      if (raw.includes("万")) {
        return Math.round(Number(raw.replace("万", "").trim()) * 10000);
      }
      return Number(raw) || 0;
    };

    const root = document.querySelector(".note-container");
    if (!root) {
      throw new Error("note container not found");
    }

    const contentBlock = root.querySelector(".note-scroller") || root;
    const title =
      root.querySelector("#detail-title")?.textContent?.trim() ||
      root.querySelector(".title")?.textContent?.trim() ||
      document.title.replace(/\s*-\s*小红书$/, "").trim();
    const content =
      contentBlock.querySelector(".note-content .note-text span")?.textContent?.trim() ||
      contentBlock.querySelector("#detail-desc .note-text")?.textContent?.trim() ||
      "";
    const tags = Array.from(contentBlock.querySelectorAll(".note-content .note-text a"))
      .map((tag) => tag.textContent?.trim().replace(/^#/, "") || "")
      .filter(Boolean);

    const authorElement = root.querySelector(".author-container .info") || root.querySelector(".author-wrapper");
    const author =
      authorElement?.querySelector(".username")?.textContent?.trim() ||
      authorElement?.querySelector(".name")?.textContent?.trim() ||
      "";
    const avatarUrl =
      authorElement?.querySelector(".avatar-item")?.getAttribute("src") ||
      authorElement?.querySelector("img")?.getAttribute("src") ||
      "";

    const interact = document.querySelector(".interact-container");
    const comments = interact?.querySelector(".chat-wrapper .count")?.textContent?.trim() || "";
    const likes = interact?.querySelector(".like-wrapper .count")?.textContent?.trim() || "";
    const favorites = interact?.querySelector(".collect-wrapper .count")?.textContent?.trim() || "";

    const images = Array.from(document.querySelectorAll(".media-container img"))
      .map((img) => img.getAttribute("src") || img.getAttribute("data-src") || "")
      .filter(Boolean);
    const videos = Array.from(document.querySelectorAll(".media-container video"))
      .map((video) => video.getAttribute("src") || "")
      .filter(Boolean);

    return {
      title,
      content,
      tags,
      author,
      avatarUrl,
      images,
      videos,
      likes: parseCounter(likes),
      comments: parseCounter(comments),
      favorites: parseCounter(favorites)
    };
  });
}

async function parseViaRednote(input, config) {
  const normalized = buildNormalized(input);
  if (!normalized) {
    return {
      ok: false,
      errorCode: "invalid_input",
      errorMessage: "invalid xiaohongshu input"
    };
  }

  const cookiePath = config.rednoteCookiePath || defaultCookiePath();
  const cookieState = loadCookies(config);
  if (!cookieState.exists || cookieState.cookies.length === 0) {
    return {
      ok: false,
      errorCode: "login_required",
      errorMessage:
        cookieState.source === "header"
          ? "rednote cookie header is empty or invalid"
          : `rednote cookie file not found or empty: ${cookiePath}`
    };
  }

  const playwright = canUsePlaywright();
  if (!playwright || !playwright.chromium) {
    return {
      ok: false,
      errorCode: "provider_unavailable",
      errorMessage: "playwright is not installed; run npm install in sidecars/xhs-sidecar and prepare a browser"
    };
  }

  let browser;
  let context;
  let page;
  try {
    browser = await playwright.chromium.launch(buildBrowserLaunchOptions(config));
    context = await browser.newContext();
    await context.addCookies(cookieState.cookies);
    page = await context.newPage();
    page.setDefaultTimeout(config.rednoteTimeoutMS);

    await page.goto(normalized.shareUrl, {
      waitUntil: "domcontentloaded",
      timeout: config.rednoteTimeoutMS
    });

    const extracted = await extractNoteFromPage(page);
    const images = unique(extracted.images.map(normalizeMediaUrl).filter((url) => /^https?:\/\//i.test(url)));
    const videos = unique(extracted.videos.map(normalizeMediaUrl).filter((url) => /^https?:\/\//i.test(url)));
    const tags = unique((extracted.tags || []).concat(extractTags(extracted.content)));

    if (!String(extracted.content || "").trim() && images.length === 0 && videos.length === 0) {
      return {
        ok: false,
        errorCode: "note_unavailable",
        errorMessage: "rednote page opened but no usable content was extracted"
      };
    }

    return {
      ok: true,
      normalized,
      quality: "full",
      note: {
        title: String(extracted.title || guessTitle(input) || "小红书图文草稿").trim(),
        content: String(extracted.content || "").trim(),
        tags,
        images,
        videos,
        coverUrl: images[0] || "",
        author: {
          name: String(extracted.author || "").trim(),
          avatarUrl: normalizeMediaUrl(extracted.avatarUrl || "")
        },
        noteType: videos.length > 0 ? "video" : "image",
        likes: parseChineseCounter(extracted.likes),
        comments: parseChineseCounter(extracted.comments),
        favorites: parseChineseCounter(extracted.favorites)
      },
      warnings: []
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error || "rednote failed");
    if (/login required/i.test(message)) {
      return {
        ok: false,
        errorCode: "login_required",
        errorMessage: "rednote page requires login"
      };
    }
    if (/Executable doesn't exist|browserType\.launch/i.test(message)) {
      return {
        ok: false,
        errorCode: "provider_unavailable",
        errorMessage: "playwright browser is not installed or executable path is invalid"
      };
    }
    return {
      ok: false,
      errorCode: "note_unavailable",
      errorMessage: message
    };
  } finally {
    if (page) {
      await page.close().catch(() => {});
    }
    if (context) {
      await context.close().catch(() => {});
    }
    if (browser) {
      await browser.close().catch(() => {});
    }
  }
}

function createRednoteProvider(config) {
  return {
    name: "rednote",
    enabled: config.rednoteEnabled,
    requiresLogin: true,
    async status() {
      return summarizeRuntime(config);
    },
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

      const live = await parseViaRednote(input, config);
      if (live.ok) {
        return live;
      }

      if (config.stubMode === "echo" && live.errorCode === "note_unavailable") {
        const echoed = buildEchoNote(input, normalized, live.errorMessage || live.errorCode || "rednote failed");
        if (echoed) {
          return { ok: true, ...echoed };
        }
      }

      return live;
    }
  };
}

module.exports = {
  createRednoteProvider,
  defaultCookiePath
};
