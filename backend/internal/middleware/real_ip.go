package middleware

import (
	"net"
	"net/http"
	"strings"
)

// TrustedRealIP only accepts nginx's X-Real-IP when the direct peer is local.
// Public clients cannot spoof forwarding headers to evade per-IP controls.
func TrustedRealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerIP := parseRemoteIP(r.RemoteAddr)
		if peerIP != nil && peerIP.IsLoopback() {
			if forwardedIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); forwardedIP != nil {
				r.RemoteAddr = forwardedIP.String()
			}
		}
		next.ServeHTTP(w, r)
	})
}

func parseRemoteIP(remoteAddr string) net.IP {
	value := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	return net.ParseIP(strings.Trim(value, "[]"))
}
