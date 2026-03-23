const test = require("node:test");
const assert = require("node:assert/strict");
const os = require("node:os");
const path = require("node:path");
const { Readable } = require("node:stream");
const fsp = require("node:fs/promises");

const {
  downloadFile,
  enrichTranscriptIfNeeded,
  transcribeWithSiliconFlow,
  transcribeWithInfiniteAI
} = require("../lib/transcript");

test("enrichTranscriptIfNeeded marks transcript as disabled when feature flag is off", async () => {
  const note = await enrichTranscriptIfNeeded(
    {
      title: "视频笔记",
      videos: ["https://cdn.example.com/demo.mp4"]
    },
    {
      transcriptEnabled: false
    }
  );

  assert.equal(note.transcript, "");
  assert.equal(note.transcriptStatus, "disabled");
  assert.equal(note.transcriptError, "");
});

test("enrichTranscriptIfNeeded skips non-video notes", async () => {
  const note = await enrichTranscriptIfNeeded(
    {
      title: "图文笔记",
      videos: []
    },
    {
      transcriptEnabled: true,
      transcriptAPIKey: "secret"
    }
  );

  assert.equal(note.transcript, "");
  assert.equal(note.transcriptStatus, "skipped");
  assert.equal(note.transcriptError, "");
});

test("downloadFile rejects oversized video before writing to disk", async () => {
  const tempDir = await fsp.mkdtemp(path.join(os.tmpdir(), "xhs-transcript-test-"));
  const outputPath = path.join(tempDir, "input.mp4");

  await assert.rejects(
    () =>
      downloadFile("https://cdn.example.com/demo.mp4", outputPath, {
        transcriptMaxVideoMB: 0.0001,
        fetchImpl: async () => ({
          ok: true,
          status: 200,
          headers: {
            get(name) {
              return String(name).toLowerCase() === "content-length" ? "1024" : "";
            }
          },
          body: Readable.from([Buffer.alloc(1024)])
        })
      }),
    /video too large/i
  );

  await fsp.rm(tempDir, { recursive: true, force: true });
});

test("transcribeWithInfiniteAI returns normalized transcript text", async () => {
  const tempDir = await fsp.mkdtemp(path.join(os.tmpdir(), "xhs-transcript-test-"));
  const audioPath = path.join(tempDir, "audio.mp3");
  await fsp.writeFile(audioPath, Buffer.from("fake mp3"));

  let called = false;
  const transcript = await transcribeWithInfiniteAI(audioPath, {
    transcriptAPIKey: "secret",
    fetchImpl: async (url, init) => {
      called = true;
      assert.equal(url, "https://api.infiniteai.cc/v1/audio/transcriptions");
      assert.equal(init.method, "POST");
      assert.equal(init.headers.Authorization, "Bearer secret");
      assert.ok(init.body instanceof FormData);
      return {
        ok: true,
        status: 200,
        async json() {
          return { text: " 先焯水 \n再慢炖  " };
        }
      };
    }
  });

  assert.equal(called, true);
  assert.equal(transcript, "先焯水\n再慢炖");

  await fsp.rm(tempDir, { recursive: true, force: true });
});

test("transcribeWithSiliconFlow posts model and returns normalized transcript text", async () => {
  const tempDir = await fsp.mkdtemp(path.join(os.tmpdir(), "xhs-transcript-test-"));
  const audioPath = path.join(tempDir, "audio.mp3");
  await fsp.writeFile(audioPath, Buffer.from("fake mp3"));

  let called = false;
  const transcript = await transcribeWithSiliconFlow(audioPath, {
    transcriptAPIKey: "secret",
    transcriptModel: "TeleAI/TeleSpeechASR",
    fetchImpl: async (url, init) => {
      called = true;
      assert.equal(url, "https://api.siliconflow.cn/v1/audio/transcriptions");
      assert.equal(init.method, "POST");
      assert.equal(init.headers.Authorization, "Bearer secret");
      assert.ok(init.body instanceof FormData);
      return {
        ok: true,
        status: 200,
        async json() {
          return { text: " 先煎鱼 \n再焖煮  " };
        }
      };
    }
  });

  assert.equal(called, true);
  assert.equal(transcript, "先煎鱼\n再焖煮");

  await fsp.rm(tempDir, { recursive: true, force: true });
});

test("enrichTranscriptIfNeeded degrades to failed when ffmpeg is unavailable", async () => {
  const note = await enrichTranscriptIfNeeded(
    {
      title: "视频笔记",
      videos: ["https://cdn.example.com/demo.mp4"]
    },
    {
      transcriptEnabled: true,
      transcriptProvider: "siliconflow",
      transcriptAPIKey: "secret",
      ffmpegPath: path.join(os.tmpdir(), "missing-ffmpeg-binary"),
      fetchImpl: async () => ({
        ok: true,
        status: 200,
        headers: {
          get() {
            return "4";
          }
        },
        body: Readable.from([Buffer.from("fake")])
      })
    }
  );

  assert.equal(note.transcript, "");
  assert.equal(note.transcriptStatus, "failed");
  assert.match(note.transcriptError, /missing-ffmpeg-binary|enoent/i);
});
