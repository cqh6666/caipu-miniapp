package addpreview

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upstream"
)

type RecipeParser interface {
	ParseRecipeLink(ctx context.Context, rawInput string) (linkparse.RecipeParseOutcome, error)
}

type Options struct {
	AMapEnabled     bool
	AMapKey         string
	AMapDefaultCity string
	AMapTimeout     time.Duration
	AMapMaxAttempts int
	AMapQPSDelay    time.Duration
	POISearcher     poiSearcher
	HTTPClient      *http.Client
}

type Service struct {
	kitchen      *kitchen.Service
	recipeParser RecipeParser
	poiSearcher  poiSearcher
	amapEnabled  bool
	defaultCity  string
	maxAttempts  int
	qpsDelay     time.Duration
	httpClient   *http.Client
}

func NewService(kitchenService *kitchen.Service, recipeParser RecipeParser, options Options) *Service {
	timeout := options.AMapTimeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	maxAttempts := options.AMapMaxAttempts
	if maxAttempts <= 0 || maxAttempts > 6 {
		maxAttempts = 4
	}
	qpsDelay := options.AMapQPSDelay
	if qpsDelay < 0 {
		qpsDelay = 0
	}
	searcher := options.POISearcher
	if searcher == nil && strings.TrimSpace(options.AMapKey) != "" {
		searcher = NewAMapClient(options.AMapKey, timeout)
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	return &Service{
		kitchen:      kitchenService,
		recipeParser: recipeParser,
		poiSearcher:  searcher,
		amapEnabled:  options.AMapEnabled,
		defaultCity:  strings.TrimSpace(options.AMapDefaultCity),
		maxAttempts:  maxAttempts,
		qpsDelay:     qpsDelay,
		httpClient:   httpClient,
	}
}

func (s *Service) Preview(ctx context.Context, userID, kitchenID int64, req PreviewRequest) (PreviewResponse, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return PreviewResponse{}, err
	}

	text := sanitizePreviewText(req.Text)
	if text == "" {
		return PreviewResponse{}, common.NewAppError(common.CodeBadRequest, "text is required", http.StatusBadRequest)
	}

	previewID, err := common.NewPrefixedID("plp")
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("generate preview id: %w", err)
	}

	extracted := parseShareText(text)
	source := detectSource(text)
	if looksLikePlaceShare(text, extracted) {
		return s.previewPlace(ctx, previewID, text, req, source, extracted), nil
	}

	if linkparse.DetectParsePlatform(text) != "" || looksLikeRecipeShare(text) {
		return s.previewRecipe(ctx, previewID, text), nil
	}

	return PreviewResponse{
		PreviewID:   previewID,
		Status:      StatusFailed,
		ContentType: "",
		Message:     "暂时无法识别分享内容类型",
		Warnings: []Warning{
			{Code: "unsupported_content", Message: "请粘贴美团、大众点评、小红书或 B站分享链接"},
		},
	}, nil
}

func (s *Service) previewRecipe(ctx context.Context, previewID string, text string) PreviewResponse {
	if s.recipeParser == nil {
		return PreviewResponse{
			PreviewID:   previewID,
			Status:      StatusFailed,
			ContentType: ContentTypeRecipe,
			Message:     "菜谱解析服务未配置",
		}
	}

	outcome, err := s.recipeParser.ParseRecipeLink(ctx, text)
	if err != nil {
		return PreviewResponse{
			PreviewID:   previewID,
			Status:      StatusFailed,
			ContentType: ContentTypeRecipe,
			Source:      linkparse.DetectParsePlatform(text),
			Message:     "暂时无法解析菜谱链接，可手动填写",
			RecipeDraft: RecipeDraft{
				Link: extractFirstURL(text),
			},
			Warnings: []Warning{
				{Code: "recipe_parse_failed", Message: err.Error()},
			},
		}
	}

	draft := outcome.RecipeDraft
	imageURLs := cleanPreviewImageURLs(append(draft.ImageURLs, draft.ImageURL))
	return PreviewResponse{
		PreviewID:   previewID,
		Status:      StatusRecipeResult,
		ContentType: ContentTypeRecipe,
		Source:      firstNonEmpty(outcome.Source, linkparse.DetectParsePlatform(text)),
		RecipeDraft: RecipeDraft{
			Title:         draft.Title,
			Ingredient:    draft.Ingredient,
			Summary:       draft.Summary,
			Link:          firstNonEmpty(draft.Link, extractFirstURL(text)),
			ImageURL:      firstNonEmpty(draft.ImageURL, firstString(imageURLs)),
			Images:        imageURLs,
			ImageURLs:     imageURLs,
			Note:          draft.Note,
			ParsedContent: draft.ParsedContent,
		},
		Warnings: recipeWarnings(outcome.Warnings),
	}
}

func (s *Service) previewPlace(ctx context.Context, previewID string, text string, req PreviewRequest, source string, extracted ExtractedPlace) PreviewResponse {
	warnings := []Warning{
		{Code: "candidate_requires_confirmation", Message: "识别结果可能存在同商圈或同品牌门店偏差，请确认后保存"},
	}
	if expanded, err := s.expandShareURL(ctx, extracted.SourceURL); err == nil && expanded != "" {
		poiID, poiIDEncrypt := extractPOIIDs(expanded)
		if extracted.POIID == "" {
			extracted.POIID = poiID
		}
		if extracted.POIIDEncrypt == "" {
			extracted.POIIDEncrypt = poiIDEncrypt
		}
	}

	draft := buildExtractedPlaceDraft(extracted, source)
	if !s.amapEnabled || s.poiSearcher == nil {
		warnings = append(warnings, Warning{Code: "poi_lookup_disabled", Message: "地图 POI 补全未启用，已返回基础字段"})
		return PreviewResponse{
			PreviewID:   previewID,
			Status:      StatusPartial,
			ContentType: ContentTypePlace,
			Source:      normalizePlaceSource(source),
			Extracted:   extracted,
			Draft:       draft,
			Warnings:    warnings,
		}
	}

	city := strings.TrimSpace(req.City)
	if city == "" {
		city = s.defaultCity
	}
	keywords := buildPOIKeywords(extracted)
	pois := make([]poiItem, 0)
	for index, keyword := range keywords {
		if index >= s.maxAttempts {
			break
		}
		if index > 0 && s.qpsDelay > 0 {
			timer := time.NewTimer(s.qpsDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				warnings = append(warnings, Warning{Code: "poi_lookup_timeout", Message: "地图 POI 查询超时，已返回基础字段"})
				return PreviewResponse{
					PreviewID:   previewID,
					Status:      StatusPartial,
					ContentType: ContentTypePlace,
					Source:      normalizePlaceSource(source),
					Extracted:   extracted,
					Draft:       draft,
					Warnings:    warnings,
				}
			case <-timer.C:
			}
		}
		items, err := s.poiSearcher.SearchPOIs(ctx, poiSearchInput{
			Keyword: keyword,
			City:    city,
			Limit:   10,
		})
		if err != nil {
			warnings = append(warnings, Warning{Code: "poi_lookup_failed", Message: "地图 POI 查询失败，已返回基础字段"})
			continue
		}
		pois = append(pois, items...)
	}

	limit := req.Limit
	if limit <= 0 || limit > 5 {
		limit = 3
	}
	candidates := rankPOIs(extracted, source, pois, limit)
	if len(candidates) == 0 {
		warnings = append(warnings, Warning{Code: "no_poi_candidates", Message: "暂时没有匹配到可信地点候选，可手动填写"})
		return PreviewResponse{
			PreviewID:   previewID,
			Status:      StatusPartial,
			ContentType: ContentTypePlace,
			Source:      normalizePlaceSource(source),
			Extracted:   extracted,
			Draft:       draft,
			Warnings:    warnings,
		}
	}

	return PreviewResponse{
		PreviewID:   previewID,
		Status:      StatusPlaceCandidates,
		ContentType: ContentTypePlace,
		Source:      normalizePlaceSource(source),
		Extracted:   extracted,
		Draft:       draft,
		Candidates:  candidates,
		Warnings:    warnings,
	}
}

func looksLikeRecipeShare(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(text, "小红书") ||
		strings.Contains(text, "B站") ||
		strings.Contains(lower, "bilibili") ||
		strings.Contains(lower, "b23.tv") ||
		strings.Contains(lower, "xhslink") ||
		strings.Contains(lower, "xiaohongshu")
}

func buildExtractedPlaceDraft(extracted ExtractedPlace, source string) PlaceDraft {
	return PlaceDraft{
		Name:      extracted.Name,
		Type:      "food",
		Address:   extracted.Address,
		Phone:     extracted.Phone,
		Source:    normalizePlaceSource(source),
		SourceURL: extracted.SourceURL,
		Images:    []string{},
		ImageURLs: []string{},
		Status:    "want",
		Tags:      []string{},
		Note:      "",
	}
}

func buildPOIKeywords(extracted ExtractedPlace) []string {
	candidates := []string{
		extracted.Name,
		cleanStoreName(extracted.Name),
		cleanStoreName(extracted.Name) + " " + extracted.Address,
		extracted.Phone,
		extracted.Address,
	}
	items := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, item := range candidates {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		key := normalizeMatchText(value)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, value)
	}
	return items
}

func (s *Service) expandShareURL(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || s.httpClient == nil {
		return "", nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", nil
	}
	if !isAllowedExpandHost(parsed.Host) {
		return "", nil
	}
	if isPrivateHost(parsed.Hostname()) {
		return "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 caipu-miniapp add-preview")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	location := strings.TrimSpace(resp.Header.Get("Location"))
	if location != "" {
		return location, nil
	}
	body, err := upstream.ReadAll(resp.Body, 256*1024)
	if err != nil {
		if upstream.IsResponseTooLarge(err) {
			return "", common.NewAppError(common.CodeInternalServer, "share page response exceeded size limit", http.StatusBadGateway).WithErr(err)
		}
		return "", err
	}
	if match := firstURLPattern.FindString(string(body)); match != "" {
		return match, nil
	}
	return "", nil
}

func isAllowedExpandHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "dpurl.cn" || strings.HasSuffix(host, ".dpurl.cn")
}

func isPrivateHost(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}
	return false
}

func cleanPreviewImageURLs(values []string) []string {
	items := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
		if len(items) >= 9 {
			break
		}
	}
	return items
}

func recipeWarnings(values []string) []Warning {
	warnings := make([]Warning, 0, len(values))
	for _, value := range values {
		message := strings.TrimSpace(value)
		if message == "" {
			continue
		}
		warnings = append(warnings, Warning{Code: "recipe_parse_warning", Message: message})
	}
	return warnings
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
