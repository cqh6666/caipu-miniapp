const test = require("node:test");
const assert = require("node:assert/strict");

const { buildNote, normalizeMediaURL, uniqueStrings } = require("../lib/xhs-provider-common");

test("xiaohongshu provider common helpers normalize values without platform policy", () => {
  assert.deepEqual(uniqueStrings([" a ", "", "a", "b"]), ["a", "b"]);
  assert.equal(normalizeMediaURL("//sns-webpic-qc.xhscdn.com/demo.jpg"), "https://sns-webpic-qc.xhscdn.com/demo.jpg");
  assert.equal(normalizeMediaURL("http://sns-webpic-qc.xhscdn.com/demo.jpg"), "https://sns-webpic-qc.xhscdn.com/demo.jpg");
  assert.deepEqual(buildNote({
    title: " 菜谱 ",
    content: " 内容 ",
    tags: ["家常", "家常"],
    images: ["https://example.test/a.jpg", "https://example.test/a.jpg"],
    authorName: " 作者 ",
    noteType: "image"
  }), {
    title: "菜谱",
    content: "内容",
    tags: ["家常"],
    images: ["https://example.test/a.jpg"],
    videos: [],
    coverUrl: "https://example.test/a.jpg",
    author: { name: "作者" },
    noteType: "image",
    likes: 0,
    comments: 0,
    favorites: 0
  });
});
