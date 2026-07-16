package common

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRequestContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "valid", body: `{"name":"demo"}`},
		{name: "empty", body: "", wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"name":"demo","extra":true}`, wantStatus: http.StatusBadRequest},
		{name: "multiple values", body: `{"name":"demo"} {"name":"second"}`, wantStatus: http.StatusBadRequest},
		{name: "trailing garbage", body: `{"name":"demo"} trailing`, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			var payload struct {
				Name string `json:"name"`
			}
			err := DecodeJSON(request, &payload)
			if test.wantStatus == 0 {
				if err != nil || payload.Name != "demo" {
					t.Fatalf("payload=%#v error=%v", payload, err)
				}
				return
			}
			var appErr *AppError
			if !errors.As(err, &appErr) || appErr.HTTPStatus != test.wantStatus {
				t.Fatalf("error=%#v, want status %d", err, test.wantStatus)
			}
		})
	}
}

func TestDecodeJSONMapsMaxBytesErrorToPayloadTooLarge(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"too long"}`))
	request.Body = http.MaxBytesReader(nil, request.Body, 8)
	var payload struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(request, &payload)
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusRequestEntityTooLarge {
		t.Fatalf("error=%#v, want payload too large", err)
	}
}
