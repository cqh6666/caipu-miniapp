package recipe

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

func TestNeedsImageMirroring(t *testing.T) {
	t.Parallel()

	uploadService := upload.NewService(t.TempDir(), "", 10)
	if !needsImageMirroring(Recipe{
		ImageURLs: []string{"https://cdn.example.com/cover.jpg"},
	}, uploadService) {
		t.Fatal("expected remote image to require mirroring")
	}

	if needsImageMirroring(Recipe{
		ImageURLs: []string{"/uploads/2026/03/demo.jpg"},
	}, uploadService) {
		t.Fatal("managed upload url should not require mirroring")
	}
}

func TestMirrorRecipeImagesDownloadsRemoteAssets(t *testing.T) {
	t.Parallel()

	payload := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	uploadDir := t.TempDir()
	uploadService := upload.NewService(uploadDir, "", 10)

	existing, err := uploadService.SaveRemoteImage(context.Background(), server.URL+"/existing.png")
	if err != nil {
		t.Fatalf("SaveRemoteImage(existing) error = %v", err)
	}

	mirrored, changed, err := mirrorRecipeImages(context.Background(), []RecipeImageMeta{
		{
			URL:        server.URL + "/cover.png",
			SourceType: RecipeImageSourceParsed,
			OriginURL:  server.URL + "/cover.png",
		},
		{
			URL:         existing.URL,
			SourceType:  RecipeImageSourceUser,
			OriginURL:   existing.URL,
			ContentHash: existing.ContentHash,
		},
	}, uploadService)
	if err != nil {
		t.Fatalf("mirrorRecipeImages returned error: %v", err)
	}
	if !changed {
		t.Fatal("mirrorRecipeImages should report changes")
	}
	if got, want := len(mirrored), 1; got != want {
		t.Fatalf("len(mirrored) = %d, want %d", got, want)
	}
	if !strings.HasPrefix(mirrored[0].URL, "/uploads/") {
		t.Fatalf("mirrored[0].URL = %q, want local uploads url", mirrored[0].URL)
	}
	if got, want := mirrored[0].SourceType, RecipeImageSourceUser; got != want {
		t.Fatalf("mirrored[0].SourceType = %q, want %q", got, want)
	}
	if got, want := mirrored[0].ContentHash, existing.ContentHash; got != want {
		t.Fatalf("mirrored[0].ContentHash = %q, want %q", got, want)
	}

	foundFile := false
	_ = filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			foundFile = true
		}
		return nil
	})
	if !foundFile {
		t.Fatal("expected mirrored image file to be written to upload dir")
	}
}

func TestRepositoryListImageMirrorCandidatesSkipsManagedUploads(t *testing.T) {
	t.Parallel()

	db := openFlowchartTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, image_url, image_urls_json, created_at, updated_at
) VALUES
  ('managed-cover', '已转存封面', 'https://www.gxm1227.top/uploads/2026/04/managed.jpg', '["https://www.gxm1227.top/uploads/2026/04/managed.jpg"]', '2026-04-02T10:00:00+08:00', '2026-04-02T10:00:00+08:00'),
  ('mixed-images', '仍有外链', 'https://www.gxm1227.top/uploads/2026/04/cover.jpg', '["https://www.gxm1227.top/uploads/2026/04/cover.jpg","https://sns-webpic-qc.xhscdn.com/demo-1.jpg"]', '2026-04-02T10:01:00+08:00', '2026-04-02T10:01:00+08:00'),
  ('remote-cover', '纯外链封面', 'https://sns-webpic-qc.xhscdn.com/demo-cover.jpg', '["https://sns-webpic-qc.xhscdn.com/demo-cover.jpg"]', '2026-04-02T10:02:00+08:00', '2026-04-02T10:02:00+08:00'),
  ('relative-upload', '相对路径已转存', '/uploads/2026/04/relative.jpg', '["/uploads/2026/04/relative.jpg"]', '2026-04-02T10:03:00+08:00', '2026-04-02T10:03:00+08:00');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	repo := NewRepository(db)
	items, err := repo.ListImageMirrorCandidates(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListImageMirrorCandidates() error = %v", err)
	}

	if got, want := len(items), 2; got != want {
		t.Fatalf("len(items) = %d, want %d", got, want)
	}
	if got, want := items[0].ID, "mixed-images"; got != want {
		t.Fatalf("items[0].ID = %q, want %q", got, want)
	}
	if got, want := items[1].ID, "remote-cover"; got != want {
		t.Fatalf("items[1].ID = %q, want %q", got, want)
	}
}

func TestOpenFlowchartTestDBSupportsJSONEach(t *testing.T) {
	t.Parallel()

	db := openFlowchartTestDB(t)
	defer db.Close()

	var count sql.NullInt64
	if err := db.QueryRow(`SELECT COUNT(1) FROM json_each('["a","b"]')`).Scan(&count); err != nil {
		t.Fatalf("json_each query error = %v", err)
	}
	if got, want := count.Int64, int64(2); got != want {
		t.Fatalf("json_each count = %d, want %d", got, want)
	}
}
