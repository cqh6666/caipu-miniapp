package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedRealIPOnlyTrustsLocalProxy(t *testing.T) {
	t.Parallel()

	handler := TrustedRealIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.RemoteAddr))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{
			name:       "local nginx",
			remoteAddr: "127.0.0.1:43210",
			forwarded:  "203.0.113.8",
			want:       "203.0.113.8",
		},
		{
			name:       "public peer cannot spoof",
			remoteAddr: "198.51.100.20:43210",
			forwarded:  "203.0.113.8",
			want:       "198.51.100.20:43210",
		},
		{
			name:       "invalid forwarded address",
			remoteAddr: "127.0.0.1:43210",
			forwarded:  "not-an-ip",
			want:       "127.0.0.1:43210",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = test.remoteAddr
			request.Header.Set("X-Real-IP", test.forwarded)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Body.String() != test.want {
				t.Fatalf("RemoteAddr=%q, want=%q", response.Body.String(), test.want)
			}
		})
	}
}
