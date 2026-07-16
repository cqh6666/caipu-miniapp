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

	decoder := newRequestJSONDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return NewAppError(CodeBadRequest, "request body is required", http.StatusBadRequest)
		}
		return requestDecodeError(err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return err
	}
	return nil
}

// DecodeJSONAllowEmpty 与 DecodeJSON 相同，但把空 body 视为“全部字段取默认值”而非报错，
// 适用于所有字段可选的动作接口（如归档/静默）。
func DecodeJSONAllowEmpty(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}

	decoder := newRequestJSONDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return requestDecodeError(err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return err
	}
	return nil
}

func newRequestJSONDecoder(body io.Reader) *json.Decoder {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	return decoder
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		if isRequestBodyTooLarge(err) {
			return ErrPayloadTooLarge.WithErr(err)
		}
		return NewAppError(
			CodeBadRequest,
			"request body must contain a single JSON object",
			http.StatusBadRequest,
		).WithErr(err)
	}
	return NewAppError(
		CodeBadRequest,
		"request body must contain a single JSON object",
		http.StatusBadRequest,
	)
}

func requestDecodeError(err error) error {
	if isRequestBodyTooLarge(err) {
		return ErrPayloadTooLarge.WithErr(err)
	}
	return NewAppError(CodeBadRequest, "invalid request body", http.StatusBadRequest).WithErr(err)
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}
