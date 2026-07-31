const test = require("node:test");
const assert = require("node:assert/strict");

const {
  BILIBILI_INPUT_DOMAINS,
  extractInputURL,
  hostMatches,
  isHTTPSURLAllowed,
  isPublicIPAddress,
  isURLAllowed,
  normalizeToHTTPSURL,
  resolvePublicAddresses
} = require("../lib/url-policy");

test("hostMatches only accepts exact domains and DNS-label subdomains", () => {
  assert.equal(hostMatches("bilibili.com", BILIBILI_INPUT_DOMAINS), true);
  assert.equal(hostMatches("www.bilibili.com", BILIBILI_INPUT_DOMAINS), true);
  assert.equal(hostMatches("bilibili.com.", BILIBILI_INPUT_DOMAINS), true);
  assert.equal(hostMatches("bilibili.com.attacker.example", BILIBILI_INPUT_DOMAINS), false);
  assert.equal(hostMatches("notbilibili.com", BILIBILI_INPUT_DOMAINS), false);
});

test("extractInputURL rejects URL credentials and malformed ports", () => {
  assert.equal(extractInputURL("https://user:pass@www.bilibili.com/video/demo").ok, false);
  assert.equal(extractInputURL("https://www.bilibili.com:bad/video/demo").ok, false);
  assert.equal(extractInputURL("看看 https://www.bilibili.com:443/video/demo。 ").ok, true);
});

test("isURLAllowed rejects hostname and userinfo confusion", () => {
  assert.equal(isURLAllowed("https://bilibili.com@attacker.example/video/demo", BILIBILI_INPUT_DOMAINS), false);
  assert.equal(isURLAllowed("https://attacker.example@www.bilibili.com/video/demo", BILIBILI_INPUT_DOMAINS), false);
  assert.equal(isURLAllowed("https://www.bilibili.com:443/video/demo", BILIBILI_INPUT_DOMAINS), true);
});

test("isHTTPSURLAllowed requires HTTPS and the default port", () => {
  assert.equal(isHTTPSURLAllowed("https://www.bilibili.com/video/demo", BILIBILI_INPUT_DOMAINS), true);
  assert.equal(isHTTPSURLAllowed("https://www.bilibili.com:443/video/demo", BILIBILI_INPUT_DOMAINS), true);
  assert.equal(isHTTPSURLAllowed("http://www.bilibili.com/video/demo", BILIBILI_INPUT_DOMAINS), false);
  assert.equal(isHTTPSURLAllowed("https://www.bilibili.com:8443/video/demo", BILIBILI_INPUT_DOMAINS), false);
});

test("normalizeToHTTPSURL upgrades default-port HTTP and rejects custom ports", () => {
  assert.equal(normalizeToHTTPSURL("http://b23.tv/demo", BILIBILI_INPUT_DOMAINS).toString(), "https://b23.tv/demo");
  assert.throws(
    () => normalizeToHTTPSURL("https://b23.tv:8443/demo", BILIBILI_INPUT_DOMAINS),
    /default HTTP or HTTPS port/
  );
});

test("isPublicIPAddress rejects private, loopback, link-local and metadata addresses", () => {
  for (const address of [
    "127.0.0.1",
    "10.0.0.8",
    "172.16.0.1",
    "192.168.1.1",
    "169.254.169.254",
    "::1",
    "::a9fe:a9fe",
    "64:ff9b::a9fe:a9fe",
    "2002:a9fe:a9fe::",
    "fc00::1",
    "fec0::1",
    "fe80::1"
  ]) {
    assert.equal(isPublicIPAddress(address), false, address);
  }
  assert.equal(isPublicIPAddress("8.8.8.8"), true);
  assert.equal(isPublicIPAddress("2606:4700:4700::1111"), true);
});

test("resolvePublicAddresses rejects a hostname when any DNS answer is private", async () => {
  await assert.rejects(
    resolvePublicAddresses("www.bilibili.com", async () => [
      { address: "8.8.8.8", family: 4 },
      { address: "127.0.0.1", family: 4 }
    ]),
    /non-public address/
  );
});
