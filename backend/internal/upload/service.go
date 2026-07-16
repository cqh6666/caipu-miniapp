package upload

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/securehttp"
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
	logger            *slog.Logger
}

func NewService(uploadDir, publicBaseURL string, maxImageSizeMB int64) *Service {
	return newService(uploadDir, publicBaseURL, maxImageSizeMB, nil, nil)
}

func NewServiceWithLogger(uploadDir, publicBaseURL string, maxImageSizeMB int64, logger *slog.Logger) *Service {
	return newService(uploadDir, publicBaseURL, maxImageSizeMB, nil, logger)
}

func newService(uploadDir, publicBaseURL string, maxImageSizeMB int64, client *http.Client, logger *slog.Logger) *Service {
	if maxImageSizeMB <= 0 {
		maxImageSizeMB = 10
	}
	if client == nil {
		client = securehttp.NewClient(20 * time.Second)
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		uploadDir:         uploadDir,
		publicBaseURL:     strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		maxImageSizeBytes: maxImageSizeMB * 1024 * 1024,
		httpClient:        client,
		logger:            logger,
	}
}

// NewServiceWithHTTPClient is intended for trusted adapters and deterministic
// tests. Production code should use NewService so untrusted destinations pass
// through the guarded DNS and redirect policy.
func NewServiceWithHTTPClient(uploadDir, publicBaseURL string, maxImageSizeMB int64, client *http.Client) *Service {
	return newService(uploadDir, publicBaseURL, maxImageSizeMB, client, nil)
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

	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		s.logRemoteImageRejection(nil, err)
		return Image{}, common.NewAppError(common.CodeBadRequest, "remote image URL is not allowed", http.StatusBadRequest).WithErr(err)
	}
	if err := securehttp.ValidateURL(parsedURL); err != nil {
		s.logRemoteImageRejection(parsedURL, err)
		return Image{}, common.NewAppError(common.CodeBadRequest, "remote image URL is not allowed", http.StatusBadRequest).WithErr(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("build remote image request: %w", err))
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; caipu-miniapp/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, securehttp.ErrBlockedAddress) || errors.Is(err, securehttp.ErrInvalidURL) {
			s.logRemoteImageRejection(parsedURL, err)
			return Image{}, common.NewAppError(common.CodeBadRequest, "remote image URL is not allowed", http.StatusBadRequest).WithErr(err)
		}
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

func (s *Service) logRemoteImageRejection(target *url.URL, err error) {
	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	reason := "invalid_url"
	if errors.Is(err, securehttp.ErrBlockedAddress) {
		reason = "blocked_address"
	}
	fields := []any{
		"securityEvent", "outbound_request_blocked",
		"reason", reason,
	}
	if target != nil {
		fields = append(fields,
			"scheme", strings.ToLower(strings.TrimSpace(target.Scheme)),
			"host", truncateLogValue(strings.ToLower(strings.TrimSpace(target.Hostname())), 255),
		)
	}
	logger.Warn("blocked remote image download", fields...)
}

func truncateLogValue(value string, maxRunes int) string {
	runes := []rune(value)
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func (s *Service) IsManagedImageURL(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/uploads/") || strings.HasPrefix(value, "/caipu-uploads/") {
		return true
	}
	if s.publicBaseURL != "" && strings.HasPrefix(value, s.publicBaseURL+"/") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	path := strings.TrimSpace(parsed.Path)
	return strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/caipu-uploads/")
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
	dst, err := os.CreateTemp(absoluteDir, "."+fileName+".*.part")
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("create upload file: %w", err))
	}
	temporaryPath := dst.Name()
	committed := false
	defer func() {
		_ = dst.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(dst, hasher), io.LimitReader(file, s.maxImageSizeBytes+1))
	if err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("save upload file: %w", err))
	}
	if written > s.maxImageSizeBytes {
		return Image{}, common.NewAppError(
			common.CodePayloadTooLarge,
			"image exceeds upload size limit",
			http.StatusRequestEntityTooLarge,
		)
	}
	if err := dst.Chmod(0o644); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("set upload file permissions: %w", err))
	}
	if err := dst.Close(); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("close upload file: %w", err))
	}
	if err := os.Rename(temporaryPath, absolutePath); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("commit upload file: %w", err))
	}
	committed = true

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
