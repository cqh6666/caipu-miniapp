const test = require("node:test");
const assert = require("node:assert/strict");

const { normalizeMediaUrl } = require("../providers/importer");

test("normalizeMediaUrl upgrades protocol-relative urls to https", () => {
  assert.equal(
    normalizeMediaUrl("//sns-webpic-qc.xhscdn.com/demo.jpg"),
    "https://sns-webpic-qc.xhscdn.com/demo.jpg"
  );
});

test("normalizeMediaUrl upgrades http urls to https", () => {
  assert.equal(
    normalizeMediaUrl("http://sns-webpic-qc.xhscdn.com/demo.jpg"),
    "https://sns-webpic-qc.xhscdn.com/demo.jpg"
  );
});
