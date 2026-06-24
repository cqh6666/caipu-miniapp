package addpreview

import "testing"

func TestParseShareText(t *testing.T) {
	text := "【旺记碳烤肥牛·烤肉大排档（北滘悦然里店）】快来试试这家餐厅吧！【地址：顺德区人昌路2号（华美达和悦然里中间停车场）】【电话：17303028852】@美团 http://dpurl.cn/4zWiEohz"

	extracted := parseShareText(text)
	if got, want := extracted.Name, "旺记碳烤肥牛·烤肉大排档（北滘悦然里店）"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
	if got, want := extracted.Address, "顺德区人昌路2号（华美达和悦然里中间停车场）"; got != want {
		t.Fatalf("Address = %q, want %q", got, want)
	}
	if got, want := extracted.Phone, "17303028852"; got != want {
		t.Fatalf("Phone = %q, want %q", got, want)
	}
	if got, want := extracted.SourceURL, "http://dpurl.cn/4zWiEohz"; got != want {
		t.Fatalf("SourceURL = %q, want %q", got, want)
	}
}

func TestRankPOIsPrefersRestaurantCandidate(t *testing.T) {
	extracted := ExtractedPlace{
		Name:      "旺记碳烤肥牛·烤肉大排档（北滘悦然里店）",
		Address:   "顺德区人昌路2号（华美达和悦然里中间停车场）",
		Phone:     "17303028852",
		SourceURL: "http://dpurl.cn/4zWiEohz",
	}

	candidates := rankPOIs(extracted, SourceMeituan, []poiItem{
		{
			ID:       "mall",
			Name:     "悦然里停车场",
			Type:     "交通设施服务;停车场;公共停车场",
			Address:  "人昌路2号",
			Location: "113.218100,22.927100",
		},
		{
			ID:       "B0JUN7FVJK",
			Name:     "旺记碳烤肥牛(多丰喜市园区北滘店)",
			Type:     "餐饮服务;中餐厅;中餐厅",
			Address:  "人昌路2号(华美达广场旁)",
			Location: "113.218424,22.927688",
			Rating:   "4.7",
			Cost:     "79.00",
			Photos:   []string{"https://example.com/1.jpg"},
		},
	}, 3)

	if len(candidates) == 0 {
		t.Fatal("expected at least one candidate")
	}
	if got, want := candidates[0].ProviderPOIID, "B0JUN7FVJK"; got != want {
		t.Fatalf("top ProviderPOIID = %q, want %q; candidates = %#v", got, want, candidates)
	}
	if candidates[0].PlaceDraft.Source != SourceMeituan {
		t.Fatalf("PlaceDraft.Source = %q, want %q", candidates[0].PlaceDraft.Source, SourceMeituan)
	}
	if len(candidates[0].MatchReasons) == 0 {
		t.Fatalf("expected match reasons for top candidate")
	}
}
