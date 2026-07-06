package common

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return NewAppError(CodeBadRequest, "request body is required", http.StatusBadRequest)
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return NewAppError(CodeBadRequest, "request body is required", http.StatusBadRequest)
		}
		return NewAppError(CodeBadRequest, "invalid request body", http.StatusBadRequest).WithErr(err)
	}

	if decoder.More() {
		return NewAppError(CodeBadRequest, "request body must contain a single JSON object", http.StatusBadRequest)
	}

	return nil
}

// DecodeJSONAllowEmpty 与 DecodeJSON 相同，但把空 body 视为“全部字段取默认值”而非报错，
// 适用于所有字段可选的动作接口（如归档/静默）。
func DecodeJSONAllowEmpty(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return NewAppError(CodeBadRequest, "invalid request body", http.StatusBadRequest).WithErr(err)
	}

	if decoder.More() {
		return NewAppError(CodeBadRequest, "request body must contain a single JSON object", http.StatusBadRequest)
	}

	return nil
}
