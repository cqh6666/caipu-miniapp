package recipe

import (
	"context"
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00})
	}))
	defer server.Close()

	uploadDir := t.TempDir()
	uploadService := upload.NewService(uploadDir, "", 10)

	mirrored, changed, err := mirrorRecipeImages(context.Background(), []string{
		server.URL + "/cover.png",
		"/uploads/2026/03/existing.jpg",
	}, uploadService)
	if err != nil {
		t.Fatalf("mirrorRecipeImages returned error: %v", err)
	}
	if !changed {
		t.Fatal("mirrorRecipeImages should report changes")
	}
	if got, want := len(mirrored), 2; got != want {
		t.Fatalf("len(mirrored) = %d, want %d", got, want)
	}
	if !strings.HasPrefix(mirrored[0], "/uploads/") {
		t.Fatalf("mirrored[0] = %q, want local uploads url", mirrored[0])
	}
	if mirrored[1] != "/uploads/2026/03/existing.jpg" {
		t.Fatalf("mirrored[1] = %q", mirrored[1])
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
