package upload

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	if data, contentType, ok, err := decodeDataImageURL(sourceURL); err != nil {
		return Image{}, common.NewAppError(common.CodeBadRequest, err.Error(), http.StatusBadRequest)
	} else if ok {
		if int64(len(data)) > s.maxImageSizeBytes {
			return Image{}, common.NewAppError(common.CodeBadRequest, "remote image exceeds upload size limit", http.StatusBadRequest)
		}
		return s.saveImageReader("", nopReadSeekCloser{Reader: bytes.NewReader(data)}, contentType)
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

func (s *Service) ManagedImageContentHash(raw string) (string, error) {
	absolutePath, err := s.managedImagePath(raw)
	if err != nil {
		return "", err
	}

	file, err := os.Open(absolutePath)
	if err != nil {
		return "", fmt.Errorf("open managed image: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash managed image: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
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

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, hasher), file); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("save upload file: %w", err))
	}

	relativePath := "/" + strings.TrimLeft(filepath.ToSlash(filepath.Join(relativeDir, fileName)), "/")
	return Image{
		URL:         buildPublicURL(s.publicBaseURL, requestBaseURL, relativePath),
		ContentHash: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func buildPublicURL(publicBaseURL string, requestBaseURL string, relativePath string) string {
	base := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(requestBaseURL), "/") + "/uploads"
	}

	return base + relativePath
}

func decodeDataImageURL(raw string) ([]byte, string, bool, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return nil, "", false, nil
	}

	comma := strings.Index(raw, ",")
	if comma <= len("data:") {
		return nil, "", true, fmt.Errorf("data image url is invalid")
	}

	meta := raw[len("data:"):comma]
	payload := raw[comma+1:]
	if strings.TrimSpace(payload) == "" {
		return nil, "", true, fmt.Errorf("data image url is empty")
	}

	parts := strings.Split(meta, ";")
	contentType := strings.TrimSpace(parts[0])
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, "", true, fmt.Errorf("data image url must use image content type")
	}

	isBase64 := false
	for _, part := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(part), "base64") {
			isBase64 = true
			break
		}
	}
	if !isBase64 {
		return nil, "", true, fmt.Errorf("data image url must use base64 encoding")
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", true, fmt.Errorf("decode data image url: %w", err)
	}

	return data, contentType, true, nil
}

func (s *Service) managedImagePath(raw string) (string, error) {
	parsedPath := strings.TrimSpace(raw)
	if parsedPath == "" {
		return "", fmt.Errorf("managed image url is required")
	}

	if parsed, err := url.Parse(parsedPath); err == nil && strings.TrimSpace(parsed.Path) != "" {
		parsedPath = parsed.Path
	}

	parsedPath = strings.TrimSpace(parsedPath)
	if !strings.HasPrefix(parsedPath, "/uploads/") {
		return "", fmt.Errorf("image url is not managed by uploads")
	}

	relativePath := strings.TrimPrefix(parsedPath, "/uploads/")
	relativePath = filepath.Clean(filepath.FromSlash(relativePath))
	if relativePath == "." || relativePath == "" {
		return "", fmt.Errorf("managed image path is invalid")
	}
	if strings.HasPrefix(relativePath, "..") {
		return "", fmt.Errorf("managed image path escapes upload dir")
	}

	return filepath.Join(s.uploadDir, relativePath), nil
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
