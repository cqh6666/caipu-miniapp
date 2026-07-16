package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

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

func TestSaveRemoteImageRejectsOversizedAndNonImageResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		body        []byte
	}{
		{
			name:        "oversized image",
			contentType: "image/png",
			body:        bytes.Repeat([]byte{0x89}, 1024*1024+1),
		},
		{
			name:        "non image",
			contentType: "text/plain",
			body:        []byte("not an image"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{test.contentType}},
					Body:       io.NopCloser(bytes.NewReader(test.body)),
				}, nil
			})}
			service := NewServiceWithHTTPClient(t.TempDir(), "https://static.example.com/uploads", 1, client)

			_, err := service.SaveRemoteImage(context.Background(), "https://images.example.com/photo")
			var appErr *common.AppError
			if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
				t.Fatalf("error = %#v, want bad request", err)
			}
		})
	}
}

func TestSaveRemoteImageHonorsTimeoutAndCancellation(t *testing.T) {
	t.Parallel()

	blockingTransport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})

	t.Run("client timeout", func(t *testing.T) {
		client := &http.Client{Transport: blockingTransport, Timeout: 20 * time.Millisecond}
		service := NewServiceWithHTTPClient(t.TempDir(), "https://static.example.com/uploads", 1, client)
		_, err := service.SaveRemoteImage(context.Background(), "https://images.example.com/photo.png")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want deadline exceeded", err)
		}
	})

	t.Run("request cancellation", func(t *testing.T) {
		client := &http.Client{Transport: blockingTransport}
		service := NewServiceWithHTTPClient(t.TempDir(), "https://static.example.com/uploads", 1, client)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := service.SaveRemoteImage(ctx, "https://images.example.com/photo.png")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	})
}

func TestSaveRemoteImageSecurityLogOmitsCredentialsPathAndQuery(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	service := NewServiceWithLogger(t.TempDir(), "https://static.example.com/uploads", 1, logger)
	rawURL := "https://user:password@127.0.0.1/private/image.png?token=sensitive"

	if _, err := service.SaveRemoteImage(context.Background(), rawURL); err == nil {
		t.Fatal("expected blocked remote image error")
	}

	logOutput := output.String()
	for _, expected := range []string{"outbound_request_blocked", "127.0.0.1", "invalid_url"} {
		if !strings.Contains(logOutput, expected) {
			t.Fatalf("security log %q does not contain %q", logOutput, expected)
		}
	}
	for _, secret := range []string{"user", "password", "/private/image.png", "token", "sensitive"} {
		if strings.Contains(logOutput, secret) {
			t.Fatalf("security log leaked %q: %s", secret, logOutput)
		}
	}
}

func TestSaveImageEnforcesServiceLimitAndRemovesPartialFile(t *testing.T) {
	t.Parallel()

	uploadDir := t.TempDir()
	service := NewService(uploadDir, "https://static.example.com/uploads", 1)
	payload := append(
		[]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
		bytes.Repeat([]byte{0}, 1024*1024)...,
	)

	_, err := service.SaveImage("http://ignored.example", nopReadSeekCloser{Reader: bytes.NewReader(payload)}, "oversized.png")
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusRequestEntityTooLarge {
		t.Fatalf("error=%#v, want 413", err)
	}
	assertUploadDirContainsNoFiles(t, uploadDir)
}

func TestSaveImageRemovesPartialFileAfterReadFailure(t *testing.T) {
	t.Parallel()

	uploadDir := t.TempDir()
	service := NewService(uploadDir, "https://static.example.com/uploads", 1)
	reader := &failingReadSeekCloser{
		reader:    bytes.NewReader(bytes.Repeat([]byte{0x89}, 128)),
		failAfter: 16,
	}

	if _, err := service.saveImageReader("", reader, "image/png"); err == nil {
		t.Fatal("expected injected read failure")
	}
	assertUploadDirContainsNoFiles(t, uploadDir)
}

func TestSaveImageUsesRandomizedPublicFilenameAndLeavesNoTempFile(t *testing.T) {
	t.Parallel()

	uploadDir := t.TempDir()
	service := NewService(uploadDir, "https://static.example.com/uploads", 1)
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	filenamePattern := regexp.MustCompile(`/[0-9]{4}/[0-9]{2}/img_[0-9]+_[0-9a-f]{12}\.png$`)
	urls := make(map[string]struct{}, 2)

	for i := 0; i < 2; i++ {
		image, err := service.SaveImage("", nopReadSeekCloser{Reader: bytes.NewReader(png)}, "image.png")
		if err != nil {
			t.Fatal(err)
		}
		if !filenamePattern.MatchString(image.URL) {
			t.Fatalf("public URL does not contain randomized image ID: %q", image.URL)
		}
		urls[image.URL] = struct{}{}
	}
	if len(urls) != 2 {
		t.Fatalf("generated image URLs are not unique: %#v", urls)
	}

	err := filepath.WalkDir(uploadDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".part") {
			t.Errorf("temporary upload remains after successful save: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type failingReadSeekCloser struct {
	reader    *bytes.Reader
	failAfter int64
	offset    int64
}

func (r *failingReadSeekCloser) Read(buffer []byte) (int, error) {
	if r.offset >= r.failAfter {
		return 0, errors.New("injected upload read failure")
	}
	remaining := r.failAfter - r.offset
	if int64(len(buffer)) > remaining {
		buffer = buffer[:remaining]
	}
	n, err := r.reader.Read(buffer)
	r.offset += int64(n)
	return n, err
}

func (r *failingReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	position, err := r.reader.Seek(offset, whence)
	if err == nil {
		r.offset = position
	}
	return position, err
}

func (r *failingReadSeekCloser) Close() error {
	return nil
}

func assertUploadDirContainsNoFiles(t *testing.T, uploadDir string) {
	t.Helper()
	err := filepath.WalkDir(uploadDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			t.Errorf("unexpected upload file after failed save: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
