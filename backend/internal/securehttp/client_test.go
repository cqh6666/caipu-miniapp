package securehttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"
)

func TestGuardedDialRejectsNonPublicAddressesBeforeDial(t *testing.T) {
	tests := []string{
		"127.0.0.1", "::1", "10.0.0.1", "172.16.0.1", "192.168.1.1",
		"169.254.169.254", "fe80::1", "0.0.0.0", "224.0.0.1",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			address := netip.MustParseAddr(raw)
			dialed := false
			dial := guardedDialContext(
				func(context.Context, string) ([]netip.Addr, error) { return []netip.Addr{address}, nil },
				func(context.Context, string, string) (net.Conn, error) {
					dialed = true
					return nil, errors.New("unexpected dial")
				},
			)
			_, err := dial(context.Background(), "tcp", "example.test:443")
			if !errors.Is(err, ErrBlockedAddress) || dialed {
				t.Fatalf("dial error = %v, dialed = %t", err, dialed)
			}
		})
	}
}

func TestGuardedDialPinsValidatedPublicAddress(t *testing.T) {
	var dialedAddress string
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	dial := guardedDialContext(
		func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
		},
		func(_ context.Context, _, address string) (net.Conn, error) {
			dialedAddress = address
			return clientConn, nil
		},
	)
	conn, err := dial(context.Background(), "tcp", "example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if dialedAddress != "93.184.216.34:443" {
		t.Fatalf("dialed address = %q", dialedAddress)
	}
}

func TestClientRevalidatesRedirectsAndRejectsDowngrade(t *testing.T) {
	client := NewClient(time.Second)
	previous := &http.Request{URL: mustURL(t, "https://example.com/start")}

	if err := client.CheckRedirect(&http.Request{URL: mustURL(t, "http://example.com/next")}, []*http.Request{previous}); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("downgrade error = %v", err)
	}
	if err := client.CheckRedirect(&http.Request{URL: mustURL(t, "https://user@example.com/next")}, []*http.Request{previous}); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("userinfo error = %v", err)
	}
}

func TestHostMatchesUsesDNSLabelBoundary(t *testing.T) {
	if !HostMatches("api.bilibili.com", "bilibili.com") {
		t.Fatal("expected subdomain match")
	}
	if HostMatches("bilibili.com.attacker.example", "bilibili.com") {
		t.Fatal("must reject fake suffix")
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
