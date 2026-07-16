package airouter

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type airouterHTTPDoerFunc func(*http.Request) (*http.Response, error)

func (f airouterHTTPDoerFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestOpenAICompatibleUsesInjectedDoerAndRequestContextDeadline(t *testing.T) {
	var calls atomic.Int32
	service := NewServiceWithOptions(nil, "test-secret", nil, nil, nil, ServiceOptions{
		HTTPDoer: airouterHTTPDoerFunc(func(request *http.Request) (*http.Response, error) {
			calls.Add(1)
			if request.Method != http.MethodPost || request.URL.String() != "https://provider.example/v1/chat/completions" {
				t.Fatalf("request=%s %s", request.Method, request.URL.String())
			}
			if got := request.Header.Get("Authorization"); got != "Bearer plain-secret" {
				t.Fatalf("Authorization=%q", got)
			}
			if got := request.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type=%q", got)
			}
			deadline, ok := request.Context().Deadline()
			if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 6*time.Second {
				t.Fatalf("request deadline=%v ok=%t", deadline, ok)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), `"model":"test-model"`) || !strings.Contains(string(body), `"stream":false`) {
				t.Fatalf("request body=%s", body)
			}
			return airouterResponse(http.StatusOK, `{"choices":[{"message":{"content":"pong"}}]}`), nil
		}),
	})

	content, _, status, _, err := service.callOpenAICompatible(
		context.Background(),
		SceneConfig{Scene: SceneSummary},
		orderedProvider{ProviderConfig: ProviderConfig{
			BaseURL:        "https://provider.example/v1",
			APIKey:         "plain-secret",
			Model:          "test-model",
			TimeoutSeconds: 5,
			Scene:          SceneSummary,
		}},
		ChatCompletionInput{Messages: []ChatMessage{{Role: "user", Content: "ping"}}},
	)
	if err != nil || status != http.StatusOK || content != "pong" || calls.Load() != 1 {
		t.Fatalf("content=%q status=%d calls=%d error=%v", content, status, calls.Load(), err)
	}
}

func TestOpenAICompatibleInjectedDoerClassifiesProtocolFailures(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		doErr      error
		wantType   string
		wantStatus int
	}{
		{name: "rate limit", status: http.StatusTooManyRequests, body: `{"error":"slow down"}`, wantType: ErrorTypeRateLimit, wantStatus: http.StatusTooManyRequests},
		{name: "auth", status: http.StatusUnauthorized, body: `{"error":"bad key"}`, wantType: ErrorTypeAuth, wantStatus: http.StatusUnauthorized},
		{name: "bad request", status: http.StatusBadRequest, body: `{"error":"bad payload"}`, wantType: ErrorTypeBadRequest, wantStatus: http.StatusBadRequest},
		{name: "upstream", status: http.StatusBadGateway, body: `{"error":"down"}`, wantType: ErrorTypeUpstream, wantStatus: http.StatusBadGateway},
		{name: "malformed JSON", status: http.StatusOK, body: `{broken`, wantType: ErrorTypeInvalidResponse, wantStatus: http.StatusOK},
		{name: "network", doErr: &net.DNSError{Err: "unreachable", Name: "provider.example"}, wantType: ErrorTypeNetwork},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewServiceWithOptions(nil, "test-secret", nil, nil, nil, ServiceOptions{
				HTTPDoer: airouterHTTPDoerFunc(func(*http.Request) (*http.Response, error) {
					if test.doErr != nil {
						return nil, test.doErr
					}
					return airouterResponse(test.status, test.body), nil
				}),
			})
			_, _, status, _, err := service.callOpenAICompatible(
				context.Background(),
				SceneConfig{Scene: SceneSummary},
				orderedProvider{ProviderConfig: ProviderConfig{
					BaseURL: "https://provider.example/v1", Model: "model", TimeoutSeconds: 5,
				}},
				ChatCompletionInput{Messages: []ChatMessage{{Role: "user", Content: "ping"}}},
			)
			if got := routeErrorType(err); got != test.wantType {
				t.Fatalf("error type=%q status=%d error=%v", got, status, err)
			}
			if status != test.wantStatus {
				t.Fatalf("status=%d, want=%d", status, test.wantStatus)
			}
		})
	}
}

func TestOpenAICompatibleInjectedDoerPreservesTimeoutAndCancellation(t *testing.T) {
	tests := []struct {
		name     string
		ctx      func() (context.Context, context.CancelFunc)
		wantType string
		wantErr  error
	}{
		{
			name: "deadline",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 20*time.Millisecond)
			},
			wantType: ErrorTypeTimeout,
			wantErr:  context.DeadlineExceeded,
		},
		{
			name: "cancellation",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantType: ErrorTypeUnknown,
			wantErr:  context.Canceled,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewServiceWithOptions(nil, "test-secret", nil, nil, nil, ServiceOptions{
				HTTPDoer: airouterHTTPDoerFunc(func(request *http.Request) (*http.Response, error) {
					<-request.Context().Done()
					return nil, request.Context().Err()
				}),
			})
			ctx, cancel := test.ctx()
			defer cancel()
			_, _, _, _, err := service.callOpenAICompatible(
				ctx,
				SceneConfig{Scene: SceneSummary},
				orderedProvider{ProviderConfig: ProviderConfig{
					BaseURL: "https://provider.example/v1", Model: "model", TimeoutSeconds: 5,
				}},
				ChatCompletionInput{Messages: []ChatMessage{{Role: "user", Content: "ping"}}},
			)
			if routeErrorType(err) != test.wantType || !errors.Is(err, test.wantErr) {
				t.Fatalf("type=%q error=%v", routeErrorType(err), err)
			}
		})
	}
}

func TestOpenAICompatibleFailsClosedOnEncryptedCredentialDecryptionError(t *testing.T) {
	var calls atomic.Int32
	service := NewServiceWithOptions(nil, "test-secret", nil, nil, nil, ServiceOptions{
		HTTPDoer: airouterHTTPDoerFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return airouterResponse(http.StatusOK, `{}`), nil
		}),
	})
	_, _, _, _, err := service.callOpenAICompatible(
		context.Background(),
		SceneConfig{Scene: SceneSummary},
		orderedProvider{ProviderConfig: ProviderConfig{
			BaseURL:         "https://provider.example/v1",
			APIKey:          "enc:v1:missing:not-valid",
			Model:           "model",
			TimeoutSeconds:  5,
			apiKeyEncrypted: true,
		}},
		ChatCompletionInput{Messages: []ChatMessage{{Role: "user", Content: "ping"}}},
	)
	if routeErrorType(err) != ErrorTypeAuth || calls.Load() != 0 {
		t.Fatalf("type=%q calls=%d error=%v", routeErrorType(err), calls.Load(), err)
	}
	if err == nil || strings.Contains(err.Error(), "missing") || strings.Contains(err.Error(), "not-valid") {
		t.Fatalf("credential error leaked details: %v", err)
	}
}

func airouterResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
