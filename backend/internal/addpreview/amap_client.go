package addpreview

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type poiSearcher interface {
	SearchPOIs(ctx context.Context, input poiSearchInput) ([]poiItem, error)
}

type AMapClient struct {
	key    string
	client *http.Client
}

func NewAMapClient(key string, timeout time.Duration) *AMapClient {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &AMapClient{
		key: strings.TrimSpace(key),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *AMapClient) SearchPOIs(ctx context.Context, input poiSearchInput) ([]poiItem, error) {
	key := strings.TrimSpace(c.key)
	keyword := strings.TrimSpace(input.Keyword)
	if key == "" {
		return nil, fmt.Errorf("amap key is empty")
	}
	if keyword == "" {
		return nil, nil
	}

	limit := input.Limit
	if limit <= 0 || limit > 10 {
		limit = 10
	}

	values := url.Values{}
	values.Set("key", key)
	values.Set("keywords", keyword)
	values.Set("city", strings.TrimSpace(input.City))
	values.Set("citylimit", "true")
	values.Set("offset", fmt.Sprintf("%d", limit))
	values.Set("page", "1")
	values.Set("extensions", "all")
	values.Set("output", "json")

	endpoint := "https://restapi.amap.com/v3/place/text?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("amap http status %d", resp.StatusCode)
	}

	var payload amapTextResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.Status) != "1" {
		return nil, fmt.Errorf("amap error: %s", strings.TrimSpace(payload.Info))
	}

	items := make([]poiItem, 0, len(payload.POIs))
	for _, poi := range payload.POIs {
		items = append(items, poiItem{
			ID:           strings.TrimSpace(poi.ID),
			Name:         strings.TrimSpace(poi.Name),
			Type:         strings.TrimSpace(poi.Type),
			TypeCode:     strings.TrimSpace(poi.TypeCode),
			Address:      strings.TrimSpace(poi.Address.String()),
			Location:     strings.TrimSpace(poi.Location.String()),
			Tel:          normalizePhone(poi.Tel.String()),
			Rating:       strings.TrimSpace(poi.BizExt.Rating.String()),
			Cost:         strings.TrimSpace(poi.BizExt.Cost.String()),
			BusinessArea: strings.TrimSpace(poi.BusinessArea),
			AdName:       strings.TrimSpace(poi.AdName),
			PName:        strings.TrimSpace(poi.PName),
			CityName:     strings.TrimSpace(poi.CityName),
			Photos:       normalizePhotoURLs(poi.Photos),
		})
	}

	return items, nil
}

type amapTextResponse struct {
	Status string       `json:"status"`
	Info   string       `json:"info"`
	POIs   []amapRawPOI `json:"pois"`
}

type amapRawPOI struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	TypeCode     string         `json:"typecode"`
	Address      flexibleString `json:"address"`
	Location     flexibleString `json:"location"`
	Tel          flexibleString `json:"tel"`
	BizExt       amapRawBizExt  `json:"biz_ext"`
	Photos       []amapRawPhoto `json:"photos"`
	BusinessArea string         `json:"business_area"`
	AdName       string         `json:"adname"`
	PName        string         `json:"pname"`
	CityName     string         `json:"cityname"`
}

type amapRawBizExt struct {
	Rating flexibleString `json:"rating"`
	Cost   flexibleString `json:"cost"`
}

type amapRawPhoto struct {
	URL flexibleString `json:"url"`
}

type flexibleString string

func (s flexibleString) String() string {
	return string(s)
}

func (s *flexibleString) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*s = flexibleString(text)
		return nil
	}

	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		if len(list) > 0 {
			*s = flexibleString(list[0])
		} else {
			*s = ""
		}
		return nil
	}

	*s = ""
	return nil
}

func normalizePhotoURLs(photos []amapRawPhoto) []string {
	items := make([]string, 0, len(photos))
	seen := map[string]struct{}{}
	for _, photo := range photos {
		value := strings.TrimSpace(photo.URL.String())
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
		if len(items) >= 3 {
			break
		}
	}
	return items
}
