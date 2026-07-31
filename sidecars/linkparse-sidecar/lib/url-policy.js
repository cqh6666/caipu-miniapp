const dns = require("node:dns").promises;
const net = require("node:net");

const FIRST_URL_PATTERN = /https?:\/\/[^\s]+/i;

const BILIBILI_INPUT_DOMAINS = Object.freeze(["bilibili.com", "b23.tv", "bili2233.cn"]);
const BILIBILI_ASSET_DOMAINS = Object.freeze(["bilibili.com", "hdslb.com", "bilivideo.com", "biliapi.net"]);
const XIAOHONGSHU_INPUT_DOMAINS = Object.freeze(["xiaohongshu.com", "xhslink.com"]);
const XIAOHONGSHU_MEDIA_DOMAINS = Object.freeze(["xiaohongshu.com", "xhscdn.com"]);

const blockedAddresses = new net.BlockList();

for (const [network, prefix] of [
  ["0.0.0.0", 8],
  ["10.0.0.0", 8],
  ["100.64.0.0", 10],
  ["127.0.0.0", 8],
  ["169.254.0.0", 16],
  ["172.16.0.0", 12],
  ["192.0.0.0", 24],
  ["192.0.2.0", 24],
  ["192.168.0.0", 16],
  ["198.18.0.0", 15],
  ["198.51.100.0", 24],
  ["203.0.113.0", 24],
  ["224.0.0.0", 4],
  ["240.0.0.0", 4]
]) {
  blockedAddresses.addSubnet(network, prefix, "ipv4");
}

for (const [network, prefix] of [
  ["::", 128],
  ["::1", 128],
  ["64:ff9b:1::", 48],
  ["100::", 64],
  ["2001::", 23],
  ["2001:db8::", 32],
  ["fc00::", 7],
  ["fec0::", 10],
  ["fe80::", 10],
  ["ff00::", 8]
]) {
  blockedAddresses.addSubnet(network, prefix, "ipv6");
}

function safeTrim(value) {
  return String(value || "").trim();
}

function normalizeHostname(hostname) {
  return safeTrim(hostname).toLowerCase().replace(/\.$/, "");
}

function hostMatches(hostname, domains) {
  const normalizedHost = normalizeHostname(hostname);
  if (!normalizedHost) {
    return false;
  }

  return (domains || []).some((domain) => {
    const normalizedDomain = normalizeHostname(domain);
    return normalizedDomain && (normalizedHost === normalizedDomain || normalizedHost.endsWith(`.${normalizedDomain}`));
  });
}

function parseHTTPURL(input) {
  let parsed;
  try {
    parsed = input instanceof URL ? new URL(input.toString()) : new URL(safeTrim(input));
  } catch (error) {
    throw new Error("invalid URL");
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("URL scheme is not allowed");
  }
  if (!parsed.hostname) {
    throw new Error("URL host is required");
  }
  if (parsed.username || parsed.password) {
    throw new Error("URL credentials are not allowed");
  }

  const normalizedHost = normalizeHostname(parsed.hostname);
  if (!normalizedHost || normalizedHost.endsWith(".")) {
    throw new Error("invalid URL host");
  }
  parsed.hostname = normalizedHost;
  return parsed;
}

function extractInputURL(input) {
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
    return { ok: true, url: parseHTTPURL(value).toString() };
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : "invalid URL" };
  }
}

function assertURLAllowed(input, domains) {
  const parsed = parseHTTPURL(input);
  if (Array.isArray(domains) && domains.length > 0 && !hostMatches(parsed.hostname, domains)) {
    throw new Error("URL host is not allowed");
  }
  return parsed;
}

function isURLAllowed(input, domains) {
  try {
    assertURLAllowed(input, domains);
    return true;
  } catch (error) {
    return false;
  }
}

function isHTTPSURLAllowed(input, domains) {
  try {
    const parsed = assertURLAllowed(input, domains);
    return parsed.protocol === "https:" && parsed.port === "";
  } catch (error) {
    return false;
  }
}

function normalizeToHTTPSURL(input, domains) {
  const parsed = assertURLAllowed(input, domains);
  if (parsed.port !== "") {
    throw new Error("URL must use the default HTTP or HTTPS port");
  }
  parsed.protocol = "https:";
  return parsed;
}

function parseIPv6Words(address) {
  const value = safeTrim(address).replace(/^\[|\]$/g, "").split("%")[0].toLowerCase();
  if (!value.includes(":")) {
    return null;
  }

  const halves = value.split("::");
  if (halves.length > 2) {
    return null;
  }
  const parseHalf = (half) => {
    const parts = half ? half.split(":") : [];
    const words = [];
    for (const part of parts) {
      if (part.includes(".")) {
        const bytes = part.split(".").map(Number);
        if (bytes.length !== 4 || bytes.some((byte) => !Number.isInteger(byte) || byte < 0 || byte > 255)) {
          return null;
        }
        words.push((bytes[0] << 8) | bytes[1], (bytes[2] << 8) | bytes[3]);
        continue;
      }
      if (!/^[0-9a-f]{1,4}$/.test(part)) {
        return null;
      }
      words.push(Number.parseInt(part, 16));
    }
    return words;
  };

  const head = parseHalf(halves[0]);
  const tail = parseHalf(halves[1] || "");
  if (!head || !tail) {
    return null;
  }
  const missing = 8 - head.length - tail.length;
  if ((halves.length === 1 && missing !== 0) || (halves.length === 2 && missing < 1)) {
    return null;
  }
  return head.concat(Array(Math.max(0, missing)).fill(0), tail);
}

function embeddedIPv4Address(address) {
  const words = parseIPv6Words(address);
  if (!words || words.length !== 8) {
    return "";
  }

  let firstWordIndex = -1;
  const firstFiveZero = words.slice(0, 5).every((word) => word === 0);
  if (words.slice(0, 6).every((word) => word === 0) || (firstFiveZero && words[5] === 0xffff)) {
    firstWordIndex = 6;
  } else if (words[0] === 0x64 && words[1] === 0xff9b && words.slice(2, 6).every((word) => word === 0)) {
    firstWordIndex = 6;
  } else if (words[0] === 0x2002) {
    firstWordIndex = 1;
  }
  if (firstWordIndex < 0) {
    return "";
  }

  const high = words[firstWordIndex];
  const low = words[firstWordIndex + 1];
  return `${high >> 8}.${high & 0xff}.${low >> 8}.${low & 0xff}`;
}

function isPublicIPAddress(address) {
  const value = safeTrim(address).replace(/^\[|\]$/g, "");
  const family = net.isIP(value);
  if (family === 0) {
    return false;
  }
  if (blockedAddresses.check(value, family === 6 ? "ipv6" : "ipv4")) {
    return false;
  }
  const embedded = family === 6 ? embeddedIPv4Address(value) : "";
  return !embedded || isPublicIPAddress(embedded);
}

function normalizeLookupResults(result) {
  const values = Array.isArray(result) ? result : result ? [result] : [];
  return values
    .map((item) => {
      if (typeof item === "string") {
        return { address: item, family: net.isIP(item) };
      }
      return {
        address: safeTrim(item && item.address),
        family: Number(item && item.family) || net.isIP(safeTrim(item && item.address))
      };
    })
    .filter((item) => item.address && (item.family === 4 || item.family === 6));
}

async function resolvePublicAddresses(hostname, lookupImpl = dns.lookup) {
  const normalizedHost = normalizeHostname(hostname).replace(/^\[|\]$/g, "");
  const literalFamily = net.isIP(normalizedHost);
  const addresses = literalFamily
    ? [{ address: normalizedHost, family: literalFamily }]
    : normalizeLookupResults(await lookupImpl(normalizedHost, { all: true, verbatim: true }));

  if (addresses.length === 0) {
    throw new Error("URL host has no DNS address");
  }
  for (const item of addresses) {
    if (!isPublicIPAddress(item.address)) {
      throw new Error(`URL host resolves to a non-public address: ${item.address}`);
    }
  }
  return addresses;
}

function createPinnedLookup(addresses) {
  const approved = normalizeLookupResults(addresses);
  if (approved.length === 0 || approved.some((item) => !isPublicIPAddress(item.address))) {
    throw new Error("cannot create lookup without approved public addresses");
  }

  return (_hostname, options, callback) => {
    if (options && options.all) {
      callback(null, approved.map((item) => ({ ...item })));
      return;
    }
    callback(null, approved[0].address, approved[0].family);
  };
}

function isTrustedBilibiliAPIURL(input) {
  try {
    const parsed = parseHTTPURL(input);
    return (
      parsed.protocol === "https:" &&
      parsed.hostname === "api.bilibili.com" &&
      (parsed.port === "" || parsed.port === "443") &&
      (parsed.pathname === "/x/web-interface/view" || parsed.pathname === "/x/player/v2")
    );
  } catch (error) {
    return false;
  }
}

module.exports = {
  BILIBILI_ASSET_DOMAINS,
  BILIBILI_INPUT_DOMAINS,
  XIAOHONGSHU_INPUT_DOMAINS,
  XIAOHONGSHU_MEDIA_DOMAINS,
  assertURLAllowed,
  createPinnedLookup,
  extractInputURL,
  hostMatches,
  isPublicIPAddress,
  isHTTPSURLAllowed,
  isTrustedBilibiliAPIURL,
  isURLAllowed,
  normalizeToHTTPSURL,
  normalizeHostname,
  parseHTTPURL,
  resolvePublicAddresses
};
