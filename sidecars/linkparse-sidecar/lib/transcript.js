const fs = require("node:fs");
const fsp = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { Blob } = require("node:buffer");
const { spawn } = require("node:child_process");
const { performance } = require("node:perf_hooks");
const { Readable, Transform } = require("node:stream");
const { pipeline } = require("node:stream/promises");

const DEFAULT_PROVIDER = "siliconflow";
const DEFAULT_ENDPOINT = "https://api.siliconflow.cn/v1/audio/transcriptions";
const DEFAULT_SILICONFLOW_MODEL = "TeleAI/TeleSpeechASR";
const INFINITEAI_ENDPOINT = "https://api.infiniteai.cc/v1/audio/transcriptions";
const DEFAULT_TIMEOUT_MS = 120000;
const DEFAULT_MAX_VIDEO_MB = 80;
const DEFAULT_FFMPEG_PATH = "ffmpeg";
const TRANSCRIPT_USER_AGENT =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36";

function safeTrim(value) {
  return String(value || "").trim();
}

function normalizeText(value) {
  return String(value || "")
    .replace(/\r\n/g, "\n")
    .replace(/\u0000/g, "")
    .replace(/[ \t]+\n/g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function normalizeErrorMessage(error) {
  const message = error instanceof Error ? error.message : String(error || "unknown error");
  return safeTrim(message).slice(0, 240);
}

function formatMB(bytes) {
  return `${(Number(bytes || 0) / (1024 * 1024)).toFixed(1)}MB`;
}

function createTimeoutSignal(timeoutMS) {
  if (!(Number(timeoutMS) > 0)) {
    return {
      signal: undefined,
      cancel() {}
    };
  }

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(new Error(`timeout after ${timeoutMS}ms`)), timeoutMS);
  return {
    signal: controller.signal,
    cancel() {
      clearTimeout(timer);
    }
  };
}

function resolveTranscriptTimeoutMS(config) {
  const timeoutMS = Number(config && config.transcriptTimeoutMS);
  if (timeoutMS > 0) {
    return timeoutMS;
  }
  return DEFAULT_TIMEOUT_MS;
}

function remainingTranscriptTimeout(config, deadlineAt) {
  const remaining = Math.max(0, deadlineAt - Date.now());
  if (remaining <= 0) {
    throw new Error(`transcript timed out after ${resolveTranscriptTimeoutMS(config)}ms`);
  }
  return {
    ...(config || {}),
    transcriptTimeoutMS: remaining
  };
}

function withTranscriptFields(note, overrides) {
  if (!note || typeof note !== "object") {
    return note;
  }

  return {
    ...note,
    transcript: "",
    transcriptStatus: "skipped",
    transcriptError: "",
    ...(overrides || {})
  };
}

function logTranscript(stage, fields) {
  const parts = Object.entries(fields || {})
    .filter((entry) => entry[1] !== undefined && entry[1] !== null && String(entry[1]).trim() !== "")
    .map(([key, value]) => `${key}=${String(value).replace(/\s+/g, " ").trim()}`);
  process.stdout.write(`[linkparse-transcript] ${stage}${parts.length ? ` ${parts.join(" ")}` : ""}\n`);
}

function summarizeFFmpegError(stderr, code, signal) {
  const lines = safeTrim(stderr)
    .split(/\r?\n/)
    .map((line) => safeTrim(line))
    .filter(Boolean);
  if (lines.length > 0) {
    return `ffmpeg failed: ${lines[lines.length - 1]}`;
  }
  if (signal) {
    return `ffmpeg terminated by signal ${signal}`;
  }
  return `ffmpeg exited with code ${code}`;
}

async function runFFmpeg(inputPath, outputPath, config) {
  const ffmpegPath = safeTrim(config.ffmpegPath) || DEFAULT_FFMPEG_PATH;
  const timeoutMS = resolveTranscriptTimeoutMS(config);

  await new Promise((resolve, reject) => {
    let timer = null;
    const child = spawn(ffmpegPath, ["-y", "-i", inputPath, "-vn", "-acodec", "libmp3lame", "-q:a", "2", outputPath], {
      stdio: ["ignore", "ignore", "pipe"]
    });

    let stderr = "";
    let settled = false;
    const finish = (error) => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timer);
      if (error) {
        reject(error);
        return;
      }
      resolve();
    };

    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString("utf8");
      if (stderr.length > 4000) {
        stderr = stderr.slice(-4000);
      }
    });

    child.on("error", (error) => finish(error));
    child.on("close", (code, signal) => {
      if (code === 0) {
        finish();
        return;
      }
      finish(new Error(summarizeFFmpegError(stderr, code, signal)));
    });

    timer = setTimeout(() => {
      child.kill("SIGKILL");
      finish(new Error(`ffmpeg timed out after ${timeoutMS}ms`));
    }, timeoutMS);
  });
}

async function downloadFile(url, outputPath, config) {
  const fetchImpl = config.fetchImpl || globalThis.fetch;
  if (typeof fetchImpl !== "function") {
    throw new Error("fetch is not available");
  }

  const maxVideoMB = Number(config.transcriptMaxVideoMB) > 0 ? Number(config.transcriptMaxVideoMB) : DEFAULT_MAX_VIDEO_MB;
  const maxBytes = Math.round(maxVideoMB * 1024 * 1024);
  const timeoutMS = resolveTranscriptTimeoutMS(config);
  const timeout = createTimeoutSignal(timeoutMS);

  try {
    const response = await fetchImpl(url, {
      method: "GET",
      redirect: "follow",
      signal: timeout.signal,
      headers: {
        "User-Agent": TRANSCRIPT_USER_AGENT,
        Referer: "https://www.xiaohongshu.com/"
      }
    });

    if (!response || !response.ok) {
      throw new Error(`video download failed: HTTP ${response ? response.status : 0}`);
    }

    const contentLength = Number(response.headers && typeof response.headers.get === "function" ? response.headers.get("content-length") : 0) || 0;
    if (maxBytes > 0 && contentLength > maxBytes) {
      throw new Error(`video too large: ${formatMB(contentLength)} exceeds ${maxVideoMB}MB`);
    }
    if (!response.body) {
      throw new Error("video download returned empty body");
    }

    const source =
      typeof response.body.getReader === "function"
        ? Readable.fromWeb(response.body)
        : response.body;

    let bytesWritten = 0;
    await pipeline(
      source,
      new Transform({
        transform(chunk, _encoding, callback) {
          bytesWritten += chunk.length;
          if (maxBytes > 0 && bytesWritten > maxBytes) {
            callback(new Error(`video too large: exceeded ${maxVideoMB}MB while downloading`));
            return;
          }
          callback(null, chunk);
        }
      }),
      fs.createWriteStream(outputPath)
    );

    return bytesWritten;
  } finally {
    timeout.cancel();
  }
}

async function transcribeWithInfiniteAI(mp3Path, config) {
  return transcribeMultipartAudio(mp3Path, {
    ...config,
    transcriptEndpoint: safeTrim(config.transcriptEndpoint) || INFINITEAI_ENDPOINT,
    transcriptModel: ""
  });
}

async function transcribeWithSiliconFlow(mp3Path, config) {
  const model = safeTrim(config.transcriptModel) || DEFAULT_SILICONFLOW_MODEL;
  return transcribeMultipartAudio(mp3Path, {
    ...config,
    transcriptEndpoint: safeTrim(config.transcriptEndpoint) || DEFAULT_ENDPOINT,
    transcriptModel: model
  });
}

async function transcribeMultipartAudio(mp3Path, config) {
  const fetchImpl = config.fetchImpl || globalThis.fetch;
  if (typeof fetchImpl !== "function") {
    throw new Error("fetch is not available");
  }
  if (typeof globalThis.FormData !== "function") {
    throw new Error("FormData is not available");
  }

  const apiKey = safeTrim(config.transcriptAPIKey);
  if (!apiKey) {
    throw new Error("transcript api key is not configured");
  }

  const endpoint = safeTrim(config.transcriptEndpoint) || DEFAULT_ENDPOINT;
  const timeoutMS = resolveTranscriptTimeoutMS(config);
  const timeout = createTimeoutSignal(timeoutMS);
  const buffer = await fsp.readFile(mp3Path);
  const form = new FormData();
  form.append("file", new Blob([buffer], { type: "audio/mpeg" }), path.basename(mp3Path));
  if (safeTrim(config.transcriptModel)) {
    form.append("model", safeTrim(config.transcriptModel));
  }

  try {
    const response = await fetchImpl(endpoint, {
      method: "POST",
      signal: timeout.signal,
      headers: {
        Authorization: `Bearer ${apiKey}`
      },
      body: form
    });

    if (!response || !response.ok) {
      const message = response ? normalizeErrorMessage(await response.text()) : "";
      throw new Error(`asr request failed: HTTP ${response ? response.status : 0}${message ? ` ${message}` : ""}`);
    }

    const payload = await response.json();
    const transcript = normalizeText(payload && (payload.text || payload.transcript));
    if (!transcript) {
      throw new Error("asr returned empty text");
    }
    return transcript;
  } finally {
    timeout.cancel();
  }
}

async function cleanupTempFiles(tempDir, config) {
  if (!tempDir) {
    return;
  }
  if (config && config.transcriptKeepTemp) {
    logTranscript("keep-temp", { dir: tempDir });
    return;
  }
  await fsp.rm(tempDir, { recursive: true, force: true }).catch(() => {});
}

async function enrichTranscriptIfNeeded(note, config) {
  const baseNote = withTranscriptFields(note);
  if (!baseNote) {
    return note;
  }

  if (!config || !config.transcriptEnabled) {
    return withTranscriptFields(baseNote, { transcriptStatus: "disabled" });
  }

  const videos = Array.isArray(baseNote.videos) ? baseNote.videos.map((item) => safeTrim(item)).filter(Boolean) : [];
  if (videos.length === 0) {
    return withTranscriptFields(baseNote, { transcriptStatus: "skipped" });
  }

  const provider = safeTrim(config.transcriptProvider || DEFAULT_PROVIDER).toLowerCase();
  let transcribeAudio;
  switch (provider) {
    case "siliconflow":
      transcribeAudio = transcribeWithSiliconFlow;
      break;
    case "infiniteai":
      transcribeAudio = transcribeWithInfiniteAI;
      break;
    default:
      return withTranscriptFields(baseNote, {
        transcriptStatus: "failed",
        transcriptError: `unsupported transcript provider: ${provider}`
      });
  }

  let tempDir = "";
  try {
    const deadlineAt = Date.now() + resolveTranscriptTimeoutMS(config);
    tempDir = await fsp.mkdtemp(path.join(os.tmpdir(), "xhs-transcript-"));
    const videoPath = path.join(tempDir, "input.mp4");
    const audioPath = path.join(tempDir, "audio.mp3");

    const downloadStartedAt = performance.now();
    const bytesWritten = await downloadFile(videos[0], videoPath, remainingTranscriptTimeout(config, deadlineAt));
    logTranscript("download", {
      status: "success",
      ms: Math.round(performance.now() - downloadStartedAt),
      size: formatMB(bytesWritten)
    });

    const ffmpegStartedAt = performance.now();
    await runFFmpeg(videoPath, audioPath, remainingTranscriptTimeout(config, deadlineAt));
    logTranscript("ffmpeg", {
      status: "success",
      ms: Math.round(performance.now() - ffmpegStartedAt)
    });

    const asrStartedAt = performance.now();
    const transcript = await transcribeAudio(audioPath, remainingTranscriptTimeout(config, deadlineAt));
    logTranscript("asr", {
      status: "success",
      ms: Math.round(performance.now() - asrStartedAt),
      chars: transcript.length
    });

    return withTranscriptFields(baseNote, {
      transcript,
      transcriptStatus: "success",
      transcriptError: ""
    });
  } catch (error) {
    const message = normalizeErrorMessage(error);
    logTranscript("failed", { reason: message });
    return withTranscriptFields(baseNote, {
      transcript: "",
      transcriptStatus: "failed",
      transcriptError: message
    });
  } finally {
    await cleanupTempFiles(tempDir, config);
  }
}

module.exports = {
  enrichTranscriptIfNeeded,
  downloadFile,
  transcribeWithSiliconFlow,
  transcribeWithInfiniteAI,
  cleanupTempFiles,
  withTranscriptFields,
  DEFAULT_SILICONFLOW_MODEL,
  DEFAULT_PROVIDER
};
