package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/logging"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	err         error
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(data)
}

func (r *statusRecorder) Flush() {
	flusher, ok := r.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	flusher.Flush()
}

func (r *statusRecorder) ObserveError(err error) {
	if err == nil {
		return
	}
	if r.err == nil {
		r.err = err
		return
	}
	r.err = errors.Join(r.err, err)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if chi.RouteContext(r.Context()) == nil {
				routeContext := chi.NewRouteContext()
				r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext))
			}
			start := time.Now()
			requestID := chimiddleware.GetReqID(r.Context())
			if requestID != "" {
				w.Header().Set("X-Request-ID", requestID)
			}
			recorder := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			attrs := []any{
				"request_id", chimiddleware.GetReqID(r.Context()),
				"method", r.Method,
				"route", requestRoutePattern(r),
				"status", recorder.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_ip", r.RemoteAddr,
			}
			attrs = append(attrs, safeBusinessIDAttrs(r)...)
			errorHTTPStatus := 0
			if recorder.err != nil {
				typeChain := logging.ErrorTypeChain(recorder.err)
				attrs = append(attrs,
					"error_type", lastString(typeChain),
					"error_chain", strings.Join(typeChain, " -> "),
					"error_message", logging.SafeErrorSummary(recorder.err),
				)
				var appErr *common.AppError
				if errors.As(recorder.err, &appErr) {
					errorHTTPStatus = appErr.HTTPStatus
					attrs = append(attrs, "error_code", appErr.Code, "error_http_status", appErr.HTTPStatus)
				}
			} else if recorder.status >= http.StatusBadRequest {
				attrs = append(attrs, "error_type", "http_status", "error_chain", "http_status")
			}

			level := slog.LevelInfo
			if recorder.status >= http.StatusInternalServerError || errorHTTPStatus >= http.StatusInternalServerError {
				level = slog.LevelError
			} else if recorder.status >= http.StatusBadRequest || errorHTTPStatus >= http.StatusBadRequest {
				level = slog.LevelWarn
			}
			logger.Log(r.Context(), level, "request completed", attrs...)
		})
	}
}

func requestRoutePattern(r *http.Request) string {
	routeContext := chi.RouteContext(r.Context())
	if routeContext == nil {
		return "unmatched"
	}
	if pattern := routeContext.RoutePattern(); pattern != "" {
		return pattern
	}
	return "unmatched"
}

func safeBusinessIDAttrs(r *http.Request) []any {
	params := []struct {
		name string
		key  string
	}{
		{name: "kitchenID", key: "kitchen_id"},
		{name: "recipeID", key: "recipe_id"},
		{name: "placeID", key: "place_id"},
		{name: "providerId", key: "provider_id"},
		{name: "scene", key: "scene"},
		{name: "group", key: "settings_group"},
		{name: "planDate", key: "plan_date"},
		{name: "id", key: "object_id"},
	}
	attrs := make([]any, 0, len(params)*2)
	for _, param := range params {
		value := strings.TrimSpace(chi.URLParam(r, param.name))
		if !isSafeBusinessID(value) {
			continue
		}
		attrs = append(attrs, param.key, value)
	}
	return attrs
}

func isSafeBusinessID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || strings.ContainsRune("_.:-", char) {
			continue
		}
		return false
	}
	return true
}

func lastString(values []string) string {
	if len(values) == 0 {
		return "unknown"
	}
	return values[len(values)-1]
}
