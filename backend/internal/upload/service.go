package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type Service struct {
	uploadDir         string
	publicBaseURL     string
	maxImageSizeBytes int64
	httpClient        *http.Client
}

func NewService(uploadDir, publicBaseURL string, maxImageSizeMB int64) *Service {
	if maxImageSizeMB <= 0 {
		maxImageSizeMB = 10
	}

	return &Service{
		uploadDir:         uploadDir,
		publicBaseURL:     strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		maxImageSizeBytes: maxImageSizeMB * 1024 * 1024,
		httpClient:        &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *Service) MaxImageSizeBytes() int64 {
	return s.maxImageSizeBytes
}

func (s *Service) SaveImage(requestBaseURL string, file multipartFile, _ string) (Image, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("read upload header: %w", err))
	}

	contentType := http.DetectContentType(buffer[:n])

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("seek upload file: %w", err))
	}

	return s.saveImageReader(requestBaseURL, file, contentType)
}

func (s *Service) SaveRemoteImage(ctx context.Context, sourceURL string) (Image, error) {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return Image{}, common.NewAppError(common.CodeBadRequest, "remote image url is required", http.StatusBadRequest)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("build remote image request: %w", err))
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; caipu-miniapp/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("download remote image: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Image{}, common.NewAppError(common.CodeBadRequest, fmt.Sprintf("remote image request failed with status %d", resp.StatusCode), http.StatusBadRequest)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, s.maxImageSizeBytes+1))
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("read remote image body: %w", err))
	}
	if int64(len(data)) > s.maxImageSizeBytes {
		return Image{}, common.NewAppError(common.CodeBadRequest, "remote image exceeds upload size limit", http.StatusBadRequest)
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, ";") {
		contentType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return s.saveImageReader("", nopReadSeekCloser{Reader: bytes.NewReader(data)}, contentType)
}

func (s *Service) IsManagedImageURL(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/uploads/") {
		return true
	}
	if s.publicBaseURL != "" && strings.HasPrefix(value, s.publicBaseURL+"/") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(parsed.Path), "/uploads/")
}

func (s *Service) saveImageReader(requestBaseURL string, file readSeekCloser, contentType string) (Image, error) {
	extension, ok := allowedImageTypes[strings.TrimSpace(contentType)]
	if !ok {
		return Image{}, common.NewAppError(common.CodeBadRequest, "only jpg, png, webp and gif images are supported", http.StatusBadRequest)
	}

	now := time.Now()
	relativeDir := filepath.Join(now.Format("2006"), now.Format("01"))
	absoluteDir := filepath.Join(s.uploadDir, relativeDir)
	if err := os.MkdirAll(absoluteDir, 0o755); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("create upload directory: %w", err))
	}

	fileID, err := common.NewPrefixedID("img")
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("generate upload id: %w", err))
	}

	fileName := fileID + extension
	absolutePath := filepath.Join(absoluteDir, fileName)
	dst, err := os.Create(absolutePath)
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("create upload file: %w", err))
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("save upload file: %w", err))
	}

	relativePath := "/" + strings.TrimLeft(filepath.ToSlash(filepath.Join(relativeDir, fileName)), "/")
	return Image{
		URL: buildPublicURL(s.publicBaseURL, requestBaseURL, relativePath),
	}, nil
}

func buildPublicURL(publicBaseURL string, requestBaseURL string, relativePath string) string {
	base := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(requestBaseURL), "/") + "/uploads"
	}

	return base + relativePath
}

type multipartFile interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type nopReadSeekCloser struct {
	*bytes.Reader
}

func (n nopReadSeekCloser) Close() error {
	return nil
}
