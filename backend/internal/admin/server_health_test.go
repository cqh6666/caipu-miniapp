package admin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
)

func TestServerHealthOverviewHealthy(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Host: ServerHealthHost{
				Hostname:           "srv-a",
				Platform:           "linux",
				UptimeSeconds:      7200,
				CPUUsagePercent:    float64Ptr(28.5),
				MemoryUsagePercent: float64Ptr(42.1),
				DiskUsagePercent:   float64Ptr(33.6),
				Load1:              float64Ptr(0.13),
				Load5:              float64Ptr(0.12),
				Load15:             float64Ptr(0.08),
			},
			Signals: []ServerHealthStatus{
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
			},
		}
	}
	service.execCommand = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("active\n"), nil
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return newTestResponse(r, http.StatusOK), nil
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	if overview.Summary.Status != ServerHealthStatusHealthy {
		t.Fatalf("summary status = %q, want %q", overview.Summary.Status, ServerHealthStatusHealthy)
	}
	if len(overview.Checks) != 6 {
		t.Fatalf("len(checks) = %d, want 6", len(overview.Checks))
	}
}

func TestServerHealthOverviewMarksNonActiveSystemdAsCritical(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Signals: []ServerHealthStatus{
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
			},
		}
	}
	service.execCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if len(args) > 1 && args[1] == "caipu-backend" {
			return []byte("inactive\n"), errors.New("exit status 3")
		}
		return []byte("active\n"), nil
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return newTestResponse(r, http.StatusOK), nil
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	backendCheck := findServerHealthCheck(overview.Checks, "caipu-backend")
	if backendCheck == nil {
		t.Fatal("expected caipu-backend check")
	}
	if backendCheck.Status != ServerHealthStatusCritical {
		t.Fatalf("backend check status = %q, want %q", backendCheck.Status, ServerHealthStatusCritical)
	}
	if overview.Summary.Status != ServerHealthStatusCritical {
		t.Fatalf("summary status = %q, want %q", overview.Summary.Status, ServerHealthStatusCritical)
	}
}

func TestServerHealthOverviewMarksDisabledSidecarAsUnknown(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.cfg.LinkparseSidecarEnabled = false
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Signals: []ServerHealthStatus{
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
			},
		}
	}
	service.execCommand = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("active\n"), nil
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return newTestResponse(r, http.StatusOK), nil
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	systemdCheck := findServerHealthCheck(overview.Checks, "caipu-linkparse-sidecar")
	if systemdCheck == nil {
		t.Fatal("expected sidecar systemd check")
	}
	if systemdCheck.Status != ServerHealthStatusUnknown {
		t.Fatalf("sidecar systemd status = %q, want %q", systemdCheck.Status, ServerHealthStatusUnknown)
	}

	httpCheck := findServerHealthCheck(overview.Checks, "sidecar-health")
	if httpCheck == nil {
		t.Fatal("expected sidecar http check")
	}
	if httpCheck.Status != ServerHealthStatusUnknown {
		t.Fatalf("sidecar http status = %q, want %q", httpCheck.Status, ServerHealthStatusUnknown)
	}
}

func TestServerHealthOverviewMarksMissingSystemdAsUnknown(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Signals: []ServerHealthStatus{
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
			},
		}
	}
	service.execCommand = func(context.Context, string, ...string) ([]byte, error) {
		return nil, osexecErrNotFound()
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return newTestResponse(r, http.StatusOK), nil
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	for _, key := range []string{"nginx", "caipu-backend"} {
		check := findServerHealthCheck(overview.Checks, key)
		if check == nil {
			t.Fatalf("expected %s check", key)
		}
		if check.Status != ServerHealthStatusUnknown {
			t.Fatalf("%s status = %q, want %q", key, check.Status, ServerHealthStatusUnknown)
		}
	}
}

func TestServerHealthOverviewMarksHTTPFailureAsCritical(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Signals: []ServerHealthStatus{
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
			},
		}
	}
	service.execCommand = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("active\n"), nil
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/api/healthz") {
			return nil, context.DeadlineExceeded
		}
		return newTestResponse(r, http.StatusOK), nil
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	check := findServerHealthCheck(overview.Checks, "backend-api-healthz")
	if check == nil {
		t.Fatal("expected backend api healthz check")
	}
	if check.Status != ServerHealthStatusCritical {
		t.Fatalf("check status = %q, want %q", check.Status, ServerHealthStatusCritical)
	}
}

func TestServerHealthOverviewUsesSidecarAPIKeyForHealthProbe(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.cfg.LinkparseSidecarAPIKey = "sidecar-secret"
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Signals: []ServerHealthStatus{
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
			},
		}
	}
	service.execCommand = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("active\n"), nil
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case "http://127.0.0.1:8091/v1/health":
			if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer sidecar-secret" {
				t.Fatalf("sidecar Authorization = %q, want %q", got, "Bearer sidecar-secret")
			}
		case "http://127.0.0.1:8080/healthz", "http://127.0.0.1:8080/api/healthz":
			if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "" {
				t.Fatalf("backend Authorization = %q, want empty", got)
			}
		default:
			t.Fatalf("unexpected probe target: %s", r.URL.String())
		}
		return newTestResponse(r, http.StatusOK), nil
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	check := findServerHealthCheck(overview.Checks, "sidecar-health")
	if check == nil {
		t.Fatal("expected sidecar health check")
	}
	if check.Status != ServerHealthStatusHealthy {
		t.Fatalf("sidecar health status = %q, want %q", check.Status, ServerHealthStatusHealthy)
	}
}

func TestServerHealthOverviewCountsResourceThresholds(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Host: ServerHealthHost{
				CPUUsagePercent:    float64Ptr(76),
				MemoryUsagePercent: float64Ptr(91),
				DiskUsagePercent:   float64Ptr(44),
			},
			Signals: []ServerHealthStatus{
				ServerHealthStatusWarning,
				ServerHealthStatusCritical,
				ServerHealthStatusHealthy,
			},
		}
	}
	service.execCommand = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("active\n"), nil
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return newTestResponse(r, http.StatusOK), nil
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	if overview.Summary.WarningCount == 0 {
		t.Fatal("expected warning signals to be counted")
	}
	if overview.Summary.CriticalCount == 0 {
		t.Fatal("expected critical signals to be counted")
	}
	if overview.Summary.Status != ServerHealthStatusCritical {
		t.Fatalf("summary status = %q, want %q", overview.Summary.Status, ServerHealthStatusCritical)
	}
}

func TestServerHealthOverviewAppliesSystemdProbeTimeout(t *testing.T) {
	t.Parallel()

	service := newTestServerHealthService()
	service.collectHostSnapshot = func(context.Context, string) serverHealthHostSnapshot {
		return serverHealthHostSnapshot{
			Signals: []ServerHealthStatus{
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
				ServerHealthStatusHealthy,
			},
		}
	}

	checked := false
	service.execCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if len(args) > 1 && args[1] == "nginx" {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("expected systemd probe deadline")
			}
			remaining := time.Until(deadline)
			if remaining <= 0 || remaining > 2*time.Second+250*time.Millisecond {
				t.Fatalf("unexpected systemd probe timeout: %v", remaining)
			}
			checked = true
		}
		return []byte("active\n"), nil
	}
	service.httpClient = newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return newTestResponse(r, http.StatusOK), nil
	})

	_, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}
	if !checked {
		t.Fatal("expected nginx systemd probe to be checked")
	}
}

func newTestServerHealthService() *ServerHealthService {
	service := NewServerHealthService(config.Config{
		UploadDir:                  ".",
		LinkparseSidecarEnabled:    true,
		LinkparseSidecarBaseURL:    "http://127.0.0.1:8091",
		LinkparseSidecarTimeoutSec: 5,
	}, nil)
	service.now = func() time.Time {
		return time.Date(2026, 4, 9, 16, 0, 0, 0, time.UTC)
	}
	return service
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func newTestResponse(r *http.Request, statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader("{}")),
		Request:    r,
		Header:     make(http.Header),
	}
}

func findServerHealthCheck(items []ServerHealthCheck, key string) *ServerHealthCheck {
	for idx := range items {
		if items[idx].Key == key {
			return &items[idx]
		}
	}
	return nil
}

func osexecErrNotFound() error {
	return errors.New("exec: \"systemctl\": executable file not found in $PATH")
}
