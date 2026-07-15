package upload

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const tinyPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aF9sAAAAASUVORK5CYII="

func TestSaveRemoteImageSupportsBase64DataURL(t *testing.T) {
	t.Parallel()

	uploadDir := t.TempDir()
	service := NewService(uploadDir, "https://static.example.com/uploads", 10)

	image, err := service.SaveRemoteImage(context.Background(), tinyPNGDataURL)
	if err != nil {
		t.Fatalf("SaveRemoteImage(dataURL) error = %v", err)
	}
	if got := image.URL; !strings.HasPrefix(got, "https://static.example.com/uploads/") {
		t.Fatalf("SaveRemoteImage(dataURL) url = %q, want uploads base url", got)
	}
	if image.ContentHash == "" {
		t.Fatal("SaveRemoteImage(dataURL) content hash = empty")
	}

	var savedFile string
	err = filepath.WalkDir(uploadDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		savedFile = path
		return nil
	})
	if err != nil {
		t.Fatalf("walk upload dir error = %v", err)
	}
	if savedFile == "" {
		t.Fatal("SaveRemoteImage(dataURL) did not persist a file")
	}
}

func TestSaveRemoteImageRejectsPrivateAndUnsupportedDestinations(t *testing.T) {
	service := NewService(t.TempDir(), "https://static.example.com/uploads", 10)
	for _, raw := range []string{
		"http://127.0.0.1/image.png",
		"http://[::1]/image.png",
		"http://169.254.169.254/latest/meta-data",
		"file:///etc/passwd",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := service.SaveRemoteImage(context.Background(), raw)
			if err == nil {
				t.Fatal("expected destination rejection")
			}
			var appErr *common.AppError
			if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
				t.Fatalf("error = %#v, want bad request", err)
			}
		})
	}
}

func TestSaveRemoteImageInjectedClientSupportsDeterministicPublicAssetTest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00})
	}))
	defer server.Close()

	service := NewServiceWithHTTPClient(t.TempDir(), "https://static.example.com/uploads", 10, server.Client())
	if _, err := service.SaveRemoteImage(context.Background(), server.URL+"/image.png"); err != nil {
		t.Fatalf("SaveRemoteImage() error = %v", err)
	}
}
