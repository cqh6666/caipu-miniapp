package securehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidURL     = errors.New("outbound URL is invalid")
	ErrBlockedAddress = errors.New("outbound address is blocked")
)

const defaultMaxRedirects = 5

type LookupIPFunc func(context.Context, string) ([]netip.Addr, error)
type DialContextFunc func(context.Context, string, string) (net.Conn, error)

type Options struct {
	Timeout      time.Duration
	MaxRedirects int
	LookupIP     LookupIPFunc
	DialContext  DialContextFunc
}

// NewClient creates an HTTP client for user-controlled destinations. It pins
// each connection to an IP that was checked immediately before dialing.
func NewClient(timeout time.Duration) *http.Client {
	return NewClientWithOptions(Options{Timeout: timeout})
}

func NewClientWithOptions(opts Options) *http.Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	maxRedirects := opts.MaxRedirects
	if maxRedirects <= 0 {
		maxRedirects = defaultMaxRedirects
	}
	lookupIP := opts.LookupIP
	if lookupIP == nil {
		lookupIP = func(ctx context.Context, host string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		}
	}
	dialContext := opts.DialContext
	if dialContext == nil {
		dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		dialContext = dialer.DialContext
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = guardedDialContext(lookupIP, dialContext)
	transport.DialTLSContext = nil
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = 10 * time.Second
	transport.ExpectContinueTimeout = time.Second
	transport.MaxResponseHeaderBytes = 1 << 20

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("%w: too many redirects", ErrInvalidURL)
			}
			if err := ValidateURL(req.URL); err != nil {
				return err
			}
			if len(via) > 0 && strings.EqualFold(via[len(via)-1].URL.Scheme, "https") && !strings.EqualFold(req.URL.Scheme, "https") {
				return fmt.Errorf("%w: HTTPS redirect downgrade is forbidden", ErrInvalidURL)
			}
			return nil
		},
	}
}

func ValidateURL(target *url.URL) error {
	if target == nil || !target.IsAbs() || strings.TrimSpace(target.Hostname()) == "" {
		return ErrInvalidURL
	}
	if target.User != nil {
		return fmt.Errorf("%w: userinfo is forbidden", ErrInvalidURL)
	}
	switch strings.ToLower(strings.TrimSpace(target.Scheme)) {
	case "http", "https":
	default:
		return fmt.Errorf("%w: unsupported scheme", ErrInvalidURL)
	}
	if port := target.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return fmt.Errorf("%w: invalid port", ErrInvalidURL)
		}
	}
	return nil
}

func HostMatches(host string, domains ...string) bool {
	host = normalizeHost(host)
	if host == "" {
		return false
	}
	for _, domain := range domains {
		domain = normalizeHost(domain)
		if domain != "" && (host == domain || strings.HasSuffix(host, "."+domain)) {
			return true
		}
	}
	return false
}

func guardedDialContext(lookupIP LookupIPFunc, dialContext DialContextFunc) DialContextFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("%w: parse destination: %v", ErrInvalidURL, err)
		}
		addresses, err := resolveAddresses(ctx, host, lookupIP)
		if err != nil {
			return nil, err
		}

		var lastErr error
		for _, address := range addresses {
			if !isPublicAddress(address) {
				return nil, fmt.Errorf("%w: destination is not public", ErrBlockedAddress)
			}
			conn, dialErr := dialContext(ctx, network, net.JoinHostPort(address.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("%w: destination has no usable address", ErrBlockedAddress)
	}
}

func resolveAddresses(ctx context.Context, host string, lookupIP LookupIPFunc) ([]netip.Addr, error) {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if address, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{address.Unmap()}, nil
	}
	addresses, err := lookupIP(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve outbound destination: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("%w: destination has no address", ErrBlockedAddress)
	}
	for index := range addresses {
		addresses[index] = addresses[index].Unmap()
		if !isPublicAddress(addresses[index]) {
			return nil, fmt.Errorf("%w: destination resolves to a non-public address", ErrBlockedAddress)
		}
	}
	return addresses, nil
}

func isPublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	return address.IsValid() && address.IsGlobalUnicast() && !address.IsPrivate() &&
		!address.IsLoopback() && !address.IsLinkLocalUnicast() &&
		!address.IsLinkLocalMulticast() && !address.IsMulticast() &&
		!address.IsUnspecified()
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimSuffix(host, ".")
	return host
}
