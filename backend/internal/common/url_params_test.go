package common

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestPositiveInt64URLParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       string
		want        int64
		wantMessage string
	}{
		{name: "positive", value: "42", want: 42},
		{name: "trim spaces", value: " 7 ", want: 7},
		{name: "missing", wantMessage: "kitchenID is required"},
		{name: "zero", value: "0", wantMessage: "invalid kitchenID"},
		{name: "negative", value: "-1", wantMessage: "invalid kitchenID"},
		{name: "not integer", value: "abc", wantMessage: "invalid kitchenID"},
		{name: "overflow", value: "9223372036854775808", wantMessage: "invalid kitchenID"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("kitchenID", test.value)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

			value, err := PositiveInt64URLParam(request, "kitchenID")
			if test.wantMessage == "" {
				if err != nil || value != test.want {
					t.Fatalf("value=%d error=%v, want value=%d", value, err, test.want)
				}
				return
			}

			var appErr *AppError
			if !errors.As(err, &appErr) || appErr.Message != test.wantMessage || appErr.Code != CodeBadRequest || appErr.HTTPStatus != http.StatusBadRequest {
				t.Fatalf("error=%#v, want message %q", err, test.wantMessage)
			}
		})
	}
}
