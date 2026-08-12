function uniqueStrings(items) {
  return Array.from(new Set((items || []).map((item) => String(item || "").trim()).filter(Boolean)));
}

function normalizeMediaURL(value) {
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

function buildNote(fields = {}) {
  const images = uniqueStrings(fields.images);
  const videos = uniqueStrings(fields.videos);
  return {
    title: String(fields.title || "").trim(),
    content: String(fields.content || "").trim(),
    tags: uniqueStrings(fields.tags),
    images,
    videos,
    coverUrl: String(fields.coverUrl || images[0] || "").trim(),
    author: {
      name: String(fields.authorName || "").trim(),
      ...(String(fields.authorAvatarURL || "").trim()
        ? { avatarUrl: String(fields.authorAvatarURL).trim() }
        : {})
    },
    noteType: String(fields.noteType || "unknown").trim() || "unknown",
    likes: Number(fields.likes || 0) || 0,
    comments: Number(fields.comments || 0) || 0,
    favorites: Number(fields.favorites || 0) || 0
  };
}

module.exports = {
  buildNote,
  normalizeMediaURL,
  uniqueStrings
};
