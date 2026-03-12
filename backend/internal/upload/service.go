package upload

import (
	"fmt"
	"io"
	"net/http"
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
}

func NewService(uploadDir, publicBaseURL string, maxImageSizeMB int64) *Service {
	if maxImageSizeMB <= 0 {
		maxImageSizeMB = 10
	}

	return &Service{
		uploadDir:         uploadDir,
		publicBaseURL:     strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		maxImageSizeBytes: maxImageSizeMB * 1024 * 1024,
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
	extension, ok := allowedImageTypes[contentType]
	if !ok {
		return Image{}, common.NewAppError(common.CodeBadRequest, "only jpg, png, webp and gif images are supported", http.StatusBadRequest)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Image{}, common.ErrInternal.WithErr(fmt.Errorf("seek upload file: %w", err))
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
