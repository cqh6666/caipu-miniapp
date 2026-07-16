package upload

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicImageHandlerServesImagesWithSecurityAndCacheHeaders(t *testing.T) {
	t.Parallel()

	uploadDir := t.TempDir()
	imagePath := filepath.Join(uploadDir, "2026", "07", "img_1_0123456789ab.png")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o755); err != nil {
		t.Fatal(err)
	}
	imageData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(imagePath, imageData, 0o644); err != nil {
		t.Fatal(err)
	}

	handler := NewPublicImageHandler(uploadDir)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/2026/07/img_1_0123456789ab.png", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Body.String() != string(imageData) {
		t.Fatalf("body=%q, want image bytes", response.Body.Bytes())
	}
	expectedHeaders := map[string]string{
		"Cache-Control":                publicImageCacheControl,
		"Content-Security-Policy":      "default-src 'none'; sandbox",
		"Cross-Origin-Resource-Policy": "cross-origin",
		"Referrer-Policy":              "no-referrer",
		"X-Content-Type-Options":       "nosniff",
	}
	for name, want := range expectedHeaders {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s=%q, want=%q", name, got, want)
		}
	}
}

func TestPublicImageHandlerDoesNotExposeDirectoriesOrUnrelatedFiles(t *testing.T) {
	t.Parallel()

	uploadDir := t.TempDir()
	directory := filepath.Join(uploadDir, "2026", "07")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"img_secret.png":         "predictable image name must not be public",
		"img_1_0123456789ab.png": "public image",
		"private.txt":            "must not be public",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	handler := NewPublicImageHandler(uploadDir)
	for _, path := range []string{
		"/",
		"/2026/",
		"/2026/07/",
		"/2026/07/img_secret.png",
		"/2026/07/private.txt",
		"/2026/07/img_guessed_ffffffffffff.png",
	} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
			}
			if got := response.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("GET %s Cache-Control=%q, want no-store", path, got)
			}
			for _, hidden := range []string{"img_secret.png", "private.txt", "must not be public", "public image"} {
				if strings.Contains(response.Body.String(), hidden) {
					t.Fatalf("GET %s leaked directory/file data %q: %s", path, hidden, response.Body.String())
				}
			}
		})
	}
}

func TestPublicImageHandlerIsReadOnly(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	NewPublicImageHandler(t.TempDir()).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, "/2026/07/img_1_0123456789ab.png", strings.NewReader("data")),
	)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d body=%s", response.Code, response.Body.String())
	}
}
