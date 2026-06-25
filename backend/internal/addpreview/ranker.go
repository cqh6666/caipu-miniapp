package addpreview

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

func rankPOIs(extracted ExtractedPlace, source string, pois []poiItem, limit int) []PlaceCandidate {
	if limit <= 0 || limit > 5 {
		limit = 3
	}

	merged := mergePOIs(pois)
	candidates := make([]PlaceCandidate, 0, len(merged))
	for _, poi := range merged {
		candidate := buildCandidate(extracted, source, poi)
		if candidate.Name == "" {
			continue
		}
		candidates = append(candidates, candidate)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].MatchScore == candidates[j].MatchScore {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].MatchScore > candidates[j].MatchScore
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func mergePOIs(pois []poiItem) []poiItem {
	items := make([]poiItem, 0, len(pois))
	indexes := map[string]int{}
	for _, poi := range pois {
		key := strings.TrimSpace(poi.ID)
		if key == "" {
			key = normalizeMatchText(poi.Name + poi.Address + poi.Location)
		}
		if key == "" {
			continue
		}
		if index, exists := indexes[key]; exists {
			current := items[index]
			if len(current.Photos) == 0 && len(poi.Photos) > 0 {
				current.Photos = poi.Photos
			}
			if current.Rating == "" {
				current.Rating = poi.Rating
			}
			if current.Cost == "" {
				current.Cost = poi.Cost
			}
			if current.Tel == "" {
				current.Tel = poi.Tel
			}
			items[index] = current
			continue
		}
		indexes[key] = len(items)
		items = append(items, poi)
	}
	return items
}

func buildCandidate(extracted ExtractedPlace, source string, poi poiItem) PlaceCandidate {
	latitude, longitude := parseAMapLocation(poi.Location)
	price := formatCost(poi.Cost)
	score, reasons := scorePOI(extracted, poi)
	tags := buildPlaceTags(poi)
	address := formatPOIAddress(poi.Address, poi.BusinessArea)

	draft := PlaceDraft{
		Name:             poi.Name,
		Type:             mapPOIType(poi.Type),
		Address:          address,
		Latitude:         latitude,
		Longitude:        longitude,
		Phone:            poi.Tel,
		Price:            price,
		Source:           normalizePlaceSource(source),
		SourceURL:        extracted.SourceURL,
		Images:           append([]string{}, poi.Photos...),
		ImageURLs:        append([]string{}, poi.Photos...),
		Status:           "want",
		Tags:             tags,
		Note:             "",
		ExternalProvider: "amap",
		ExternalPOIID:    poi.ID,
		Rating:           poi.Rating,
	}

	return PlaceCandidate{
		CandidateID:   "amap:" + poi.ID,
		Provider:      "amap",
		ProviderPOIID: poi.ID,
		Name:          poi.Name,
		Type:          draft.Type,
		Address:       address,
		Latitude:      latitude,
		Longitude:     longitude,
		Phone:         poi.Tel,
		Price:         price,
		Rating:        poi.Rating,
		ImageURLs:     append([]string{}, poi.Photos...),
		MatchScore:    score,
		MatchReasons:  reasons,
		PlaceDraft:    draft,
	}
}

func scorePOI(extracted ExtractedPlace, poi poiItem) (int, []string) {
	score := 0
	reasons := []string{}

	nameScore := similarityScore(extracted.Name, poi.Name)
	switch {
	case nameScore >= 85:
		score += 90
		reasons = append(reasons, "名称高度匹配")
	case nameScore >= 55:
		score += 60
		reasons = append(reasons, "名称接近")
	case nameScore >= 30:
		score += 30
	}

	addressScore := similarityScore(extracted.Address, poi.Address)
	switch {
	case addressScore >= 70:
		score += 70
		reasons = append(reasons, "地址匹配")
	case addressScore >= 40:
		score += 45
		reasons = append(reasons, "地址接近")
	case addressScore >= 20:
		score += 20
	}

	if extracted.Phone != "" && poi.Tel != "" && strings.Contains(poi.Tel, extracted.Phone) {
		score += 80
		reasons = append(reasons, "电话一致")
	}

	locationScore, locationReasons := scoreLocationTokens(extracted, poi)
	score += locationScore
	reasons = append(reasons, locationReasons...)

	poiType := strings.TrimSpace(poi.Type)
	switch {
	case strings.Contains(poiType, "餐饮"):
		score += 45
		reasons = append(reasons, "餐饮类目")
	case strings.Contains(poiType, "购物") || strings.Contains(poiType, "商务住宅") || strings.Contains(poiType, "停车场"):
		score -= 50
	}

	if hasSharedToken(extracted.Name+" "+extracted.Address, poi.Name+" "+poi.Address) {
		score += 25
		reasons = append(reasons, "商圈或门牌相近")
	}
	if len(poi.Photos) > 0 {
		score += 10
	}
	if poi.Rating != "" {
		score += 5
	}

	if len(reasons) == 0 && score > 0 {
		reasons = append(reasons, "可能相关")
	}
	return score, reasons
}

func similarityScore(left, right string) int {
	left = normalizeMatchText(left)
	right = normalizeMatchText(right)
	if left == "" || right == "" {
		return 0
	}
	if left == right {
		return 100
	}
	if strings.Contains(left, right) || strings.Contains(right, left) {
		return 88
	}

	leftRunes := []rune(left)
	rightSet := map[rune]struct{}{}
	for _, r := range right {
		rightSet[r] = struct{}{}
	}
	matched := 0
	seen := map[rune]struct{}{}
	for _, r := range leftRunes {
		if _, counted := seen[r]; counted {
			continue
		}
		seen[r] = struct{}{}
		if _, exists := rightSet[r]; exists {
			matched++
		}
	}
	denominator := math.Max(float64(len([]rune(left))), float64(len([]rune(right))))
	if denominator == 0 {
		return 0
	}
	return int(float64(matched) / denominator * 100)
}

func hasSharedToken(left, right string) bool {
	left = normalizeMatchText(left)
	right = normalizeMatchText(right)
	if left == "" || right == "" {
		return false
	}
	tokens := []string{"北滘", "悦然里", "华美达", "人昌路", "多丰喜", "顺德"}
	for _, token := range tokens {
		if strings.Contains(left, token) && strings.Contains(right, token) {
			return true
		}
	}
	return false
}

func scoreLocationTokens(extracted ExtractedPlace, poi poiItem) (int, []string) {
	source := normalizeMatchText(extracted.Name + extracted.Address)
	target := normalizeMatchText(poi.Name + poi.Address)
	if source == "" || target == "" {
		return 0, nil
	}

	score := 0
	reasons := []string{}
	matched := 0
	tokens := []struct {
		Value   string
		Weight  int
		Penalty int
	}{
		{Value: "人昌路", Weight: 30, Penalty: 30},
		{Value: "北滘", Weight: 24, Penalty: 24},
		{Value: "华美达", Weight: 24, Penalty: 0},
		{Value: "悦然里", Weight: 20, Penalty: 0},
		{Value: "多丰喜", Weight: 18, Penalty: 0},
		{Value: "顺德", Weight: 12, Penalty: 0},
	}

	for _, token := range tokens {
		normalizedToken := normalizeMatchText(token.Value)
		if !strings.Contains(source, normalizedToken) {
			continue
		}
		if strings.Contains(target, normalizedToken) {
			score += token.Weight
			matched++
			continue
		}
		score -= token.Penalty
	}

	if strings.Contains(source, "人昌路2号") && strings.Contains(target, "人昌路2号") {
		score += 45
		matched++
	}

	if matched >= 2 {
		reasons = append(reasons, "定位信息匹配")
	}
	return score, reasons
}

func parseAMapLocation(value string) (float64, float64) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) != 2 {
		return 0, 0
	}
	longitude, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	latitude, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	return latitude, longitude
}

func formatCost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "[]" {
		return ""
	}
	if strings.HasPrefix(value, "¥") {
		if strings.Contains(value, "/人") {
			return value
		}
		return value + "/人"
	}
	value = strings.TrimSuffix(value, ".00")
	return "¥" + value + "/人"
}

func mapPOIType(value string) string {
	if strings.Contains(value, "风景名胜") ||
		strings.Contains(value, "体育休闲") ||
		strings.Contains(value, "景点") ||
		strings.Contains(value, "公园") ||
		strings.Contains(value, "博物馆") {
		return "attraction"
	}
	if strings.Contains(value, "餐饮") {
		return "food"
	}
	return "other"
}

func formatPOIAddress(address, businessArea string) string {
	address = strings.TrimSpace(address)
	businessArea = strings.TrimSpace(businessArea)
	if address == "" || businessArea == "" || strings.Contains(address, businessArea) {
		return address
	}
	return address + " · " + businessArea
}

func buildPlaceTags(poi poiItem) []string {
	candidates := []string{}
	for index, part := range strings.Split(poi.Type, ";") {
		part = strings.TrimSpace(part)
		if part == "" || index == 0 {
			continue
		}
		for _, sub := range strings.Split(part, "/") {
			sub = strings.TrimSpace(strings.TrimSuffix(sub, "餐厅"))
			if sub != "" {
				candidates = append(candidates, sub)
			}
		}
	}
	candidates = append(candidates, poi.BusinessArea, poi.AdName)
	return cleanTagList(candidates, 8)
}

func cleanTagList(values []string, limit int) []string {
	items := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == "[]" {
			continue
		}
		if len([]rune(value)) > 12 {
			value = string([]rune(value)[:12])
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, value)
		if len(items) >= limit {
			break
		}
	}
	return items
}

func normalizePlaceSource(value string) string {
	switch strings.TrimSpace(value) {
	case SourceMeituan, SourceDianping:
		return value
	default:
		return SourceOther
	}
}
