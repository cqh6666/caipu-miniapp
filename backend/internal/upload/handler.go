package upload

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	bodyLimit := h.service.MaxImageSizeBytes() + (1 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)

	if err := r.ParseMultipartForm(bodyLimit); err != nil {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "invalid upload payload", http.StatusBadRequest).WithErr(err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "file is required", http.StatusBadRequest).WithErr(err))
		return
	}
	defer file.Close()

	image, err := h.service.SaveImage(requestBaseURL(r), file, header.Filename)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusCreated, image)
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}

	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}

	return scheme + "://" + host
}
