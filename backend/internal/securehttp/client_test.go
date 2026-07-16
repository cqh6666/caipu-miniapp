package securehttp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
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

func TestGuardedDialRejectsMixedPublicAndPrivateDNSAnswers(t *testing.T) {
	t.Parallel()

	dialed := false
	dial := guardedDialContext(
		func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{
				netip.MustParseAddr("93.184.216.34"),
				netip.MustParseAddr("10.0.0.8"),
			}, nil
		},
		func(context.Context, string, string) (net.Conn, error) {
			dialed = true
			return nil, errors.New("unexpected dial")
		},
	)

	_, err := dial(context.Background(), "tcp", "images.example.com:443")
	if !errors.Is(err, ErrBlockedAddress) || dialed {
		t.Fatalf("dial error = %v, dialed = %t", err, dialed)
	}
}

func TestClientRejectsPublicRedirectToPrivateDestination(t *testing.T) {
	t.Parallel()

	dialCount := 0
	client := NewClientWithOptions(Options{
		Timeout: time.Second,
		LookupIP: func(_ context.Context, host string) ([]netip.Addr, error) {
			switch host {
			case "public.example":
				return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
			case "private.example":
				return []netip.Addr{netip.MustParseAddr("10.0.0.9")}, nil
			default:
				return nil, errors.New("unexpected host")
			}
		},
		DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialCount++
			if address != "93.184.216.34:80" {
				return nil, errors.New("unexpected dial address")
			}
			clientConn, serverConn := net.Pipe()
			go func() {
				defer serverConn.Close()
				request, err := http.ReadRequest(bufio.NewReader(serverConn))
				if err != nil {
					return
				}
				_, _ = io.Copy(io.Discard, request.Body)
				_ = request.Body.Close()
				_, _ = io.WriteString(serverConn, strings.Join([]string{
					"HTTP/1.1 302 Found",
					"Location: http://private.example/secret",
					"Content-Length: 0",
					"Connection: close",
					"",
					"",
				}, "\r\n"))
			}()
			return clientConn, nil
		},
	})

	response, err := client.Get("http://public.example/start")
	if response != nil {
		response.Body.Close()
	}
	if !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("request error = %v, want blocked address", err)
	}
	if dialCount != 1 {
		t.Fatalf("dial count = %d, want 1", dialCount)
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

func TestClientRejectsRedirectsBeyondConfiguredLimit(t *testing.T) {
	t.Parallel()

	client := NewClientWithOptions(Options{Timeout: time.Second, MaxRedirects: 2})
	via := []*http.Request{
		{URL: mustURL(t, "https://example.com/one")},
		{URL: mustURL(t, "https://example.com/two")},
	}
	err := client.CheckRedirect(&http.Request{URL: mustURL(t, "https://example.com/three")}, via)
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("redirect error = %v, want invalid URL", err)
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
