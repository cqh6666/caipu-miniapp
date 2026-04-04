package recipe

import "strings"

const (
	RecipeImageSourceUser   = "user"
	RecipeImageSourceParsed = "parsed"
	RecipeImageSourceLegacy = "legacy"
)

type RecipeImageMeta struct {
	URL         string `json:"url"`
	ContentHash string `json:"contentHash,omitempty"`
	SourceType  string `json:"sourceType,omitempty"`
	OriginURL   string `json:"originUrl,omitempty"`
	SourceLink  string `json:"sourceLink,omitempty"`
}

type managedImageHashResolver interface {
	IsManagedImageURL(raw string) bool
	ManagedImageContentHash(raw string) (string, error)
}

func recipeImageURLsFromItem(item Recipe) []string {
	if len(item.ImageMetas) > 0 {
		return recipeImageURLsFromMetas(item.ImageMetas)
	}

	imageURLs := cleanRecipeImageURLs(item.ImageURLs)
	if len(imageURLs) == 0 && strings.TrimSpace(item.ImageURL) != "" {
		imageURLs = []string{strings.TrimSpace(item.ImageURL)}
	}
	return imageURLs
}

func recipeImageMetasFromItem(item Recipe) []RecipeImageMeta {
	return normalizeRecipeImageMetas(recipeImageURLsFromItem(item), item.ImageMetas)
}

func recipeImageURLsFromMetas(metas []RecipeImageMeta) []string {
	imageURLs := make([]string, 0, len(metas))
	for _, meta := range metas {
		url := strings.TrimSpace(meta.URL)
		if url == "" {
			continue
		}
		imageURLs = append(imageURLs, url)
	}
	return cleanRecipeImageURLs(imageURLs)
}

func normalizeRecipeImageMetas(imageURLs []string, metas []RecipeImageMeta) []RecipeImageMeta {
	orderedURLs := cleanRecipeImageURLs(imageURLs)
	if len(orderedURLs) == 0 {
		orderedURLs = recipeImageURLsFromMetas(metas)
	}

	metaOrder := make([]string, 0, len(metas))
	metaByURL := make(map[string]RecipeImageMeta, len(metas))
	for _, raw := range metas {
		meta, ok := normalizeRecipeImageMeta(raw)
		if !ok {
			continue
		}

		if existing, exists := metaByURL[meta.URL]; exists {
			metaByURL[meta.URL] = choosePreferredRecipeImageMeta(existing, meta)
			continue
		}

		metaOrder = append(metaOrder, meta.URL)
		metaByURL[meta.URL] = meta
	}

	items := make([]RecipeImageMeta, 0, len(orderedURLs)+len(metaOrder))
	seen := make(map[string]struct{}, len(orderedURLs)+len(metaOrder))
	for _, url := range orderedURLs {
		meta, exists := metaByURL[url]
		if !exists {
			meta = RecipeImageMeta{
				URL:        url,
				SourceType: RecipeImageSourceLegacy,
				OriginURL:  url,
			}
		}

		items = append(items, meta)
		seen[url] = struct{}{}
	}

	for _, url := range metaOrder {
		if _, exists := seen[url]; exists {
			continue
		}
		items = append(items, metaByURL[url])
		seen[url] = struct{}{}
	}

	return dedupeRecipeImageMetas(items)
}

func fillManagedImageHashes(metas []RecipeImageMeta, resolver managedImageHashResolver) []RecipeImageMeta {
	items := normalizeRecipeImageMetas(recipeImageURLsFromMetas(metas), metas)
	if resolver == nil {
		return items
	}

	for index := range items {
		if items[index].ContentHash != "" || !resolver.IsManagedImageURL(items[index].URL) {
			continue
		}

		hash, err := resolver.ManagedImageContentHash(items[index].URL)
		if err != nil {
			continue
		}
		items[index].ContentHash = normalizeRecipeImageContentHash(hash)
	}

	return dedupeRecipeImageMetas(items)
}

func buildSubmittedImageMetas(nextImageURLs []string, current Recipe, resolver managedImageHashResolver) []RecipeImageMeta {
	imageURLs := cleanRecipeImageURLs(nextImageURLs)
	currentMetas := recipeImageMetasFromItem(current)
	metaByURL := make(map[string]RecipeImageMeta, len(currentMetas))
	for _, meta := range currentMetas {
		metaByURL[meta.URL] = meta
	}

	items := make([]RecipeImageMeta, 0, len(imageURLs))
	for _, url := range imageURLs {
		meta, exists := metaByURL[url]
		if !exists {
			meta = RecipeImageMeta{
				URL:        url,
				SourceType: RecipeImageSourceUser,
				OriginURL:  url,
			}
		}
		items = append(items, meta)
	}

	return fillManagedImageHashes(items, resolver)
}

func choosePreferredRecipeImageMeta(existing, candidate RecipeImageMeta) RecipeImageMeta {
	normalizedExisting, existingOK := normalizeRecipeImageMeta(existing)
	normalizedCandidate, candidateOK := normalizeRecipeImageMeta(candidate)
	switch {
	case !existingOK:
		return normalizedCandidate
	case !candidateOK:
		return normalizedExisting
	}

	preferred := normalizedExisting
	fallback := normalizedCandidate
	if recipeImageSourcePriority(normalizedCandidate.SourceType) > recipeImageSourcePriority(normalizedExisting.SourceType) {
		preferred = normalizedCandidate
		fallback = normalizedExisting
	}

	if preferred.URL == "" {
		preferred.URL = fallback.URL
	}
	if preferred.ContentHash == "" {
		preferred.ContentHash = fallback.ContentHash
	}
	if preferred.OriginURL == "" {
		preferred.OriginURL = fallback.OriginURL
	}
	if preferred.SourceLink == "" {
		preferred.SourceLink = fallback.SourceLink
	}
	if preferred.OriginURL == "" {
		preferred.OriginURL = preferred.URL
	}

	return preferred
}

func dedupeRecipeImageMetas(metas []RecipeImageMeta) []RecipeImageMeta {
	items := dedupeRecipeImageMetasByURL(metas)
	items = dedupeRecipeImageMetasByHash(items)
	items = dedupeRecipeImageMetasByURL(items)
	if len(items) > maxRecipeImages {
		items = items[:maxRecipeImages]
	}
	return items
}

func dedupeRecipeImageMetasByURL(metas []RecipeImageMeta) []RecipeImageMeta {
	items := make([]RecipeImageMeta, 0, len(metas))
	indexByURL := make(map[string]int, len(metas))
	for _, raw := range metas {
		meta, ok := normalizeRecipeImageMeta(raw)
		if !ok {
			continue
		}

		if index, exists := indexByURL[meta.URL]; exists {
			items[index] = choosePreferredRecipeImageMeta(items[index], meta)
			continue
		}

		indexByURL[meta.URL] = len(items)
		items = append(items, meta)
	}
	return items
}

func dedupeRecipeImageMetasByHash(metas []RecipeImageMeta) []RecipeImageMeta {
	items := make([]RecipeImageMeta, 0, len(metas))
	indexByHash := make(map[string]int, len(metas))
	for _, raw := range metas {
		meta, ok := normalizeRecipeImageMeta(raw)
		if !ok {
			continue
		}

		if meta.ContentHash == "" {
			items = append(items, meta)
			continue
		}

		if index, exists := indexByHash[meta.ContentHash]; exists {
			items[index] = choosePreferredRecipeImageMeta(items[index], meta)
			continue
		}

		indexByHash[meta.ContentHash] = len(items)
		items = append(items, meta)
	}
	return items
}

func normalizeRecipeImageMeta(meta RecipeImageMeta) (RecipeImageMeta, bool) {
	meta.URL = strings.TrimSpace(meta.URL)
	if meta.URL == "" {
		return RecipeImageMeta{}, false
	}

	meta.ContentHash = normalizeRecipeImageContentHash(meta.ContentHash)
	meta.SourceType = normalizeRecipeImageSource(meta.SourceType)
	meta.OriginURL = strings.TrimSpace(meta.OriginURL)
	meta.SourceLink = strings.TrimSpace(meta.SourceLink)
	if meta.OriginURL == "" {
		meta.OriginURL = meta.URL
	}

	return meta, true
}

func normalizeRecipeImageSource(sourceType string) string {
	switch strings.TrimSpace(strings.ToLower(sourceType)) {
	case RecipeImageSourceUser:
		return RecipeImageSourceUser
	case RecipeImageSourceParsed:
		return RecipeImageSourceParsed
	default:
		return RecipeImageSourceLegacy
	}
}

func normalizeRecipeImageContentHash(hash string) string {
	return strings.TrimSpace(strings.ToLower(hash))
}

func recipeImageSourcePriority(sourceType string) int {
	switch normalizeRecipeImageSource(sourceType) {
	case RecipeImageSourceUser:
		return 3
	case RecipeImageSourceLegacy:
		return 2
	case RecipeImageSourceParsed:
		return 1
	default:
		return 0
	}
}

func recipeImageMetasEqual(left, right []RecipeImageMeta) bool {
	left = normalizeRecipeImageMetas(recipeImageURLsFromMetas(left), left)
	right = normalizeRecipeImageMetas(recipeImageURLsFromMetas(right), right)
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
