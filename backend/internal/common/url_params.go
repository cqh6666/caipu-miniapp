package common

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func PositiveInt64URLParam(r *http.Request, name string) (int64, error) {
	raw := strings.TrimSpace(chi.URLParam(r, name))
	if raw == "" {
		return 0, NewAppError(CodeBadRequest, name+" is required", http.StatusBadRequest)
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, NewAppError(CodeBadRequest, "invalid "+name, http.StatusBadRequest)
	}

	return value, nil
}
