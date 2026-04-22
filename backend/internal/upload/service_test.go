package upload

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
