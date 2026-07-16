package airouter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/upstream"
)

func TestOpenAICompatibleRejectsOversizedResponses(t *testing.T) {
	tests := []struct {
		name  string
		mode  ProviderEndpointMode
		limit int64
	}{
		{name: "chat JSON", mode: EndpointModeChatCompletions, limit: maxAIChatResponseBytes},
		{name: "image base64 JSON", mode: EndpointModeImagesGenerations, limit: maxAIImageGenerationResponseBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeRepeatedResponse(t, w, tt.limit+1)
			}))
			defer server.Close()

			service := NewService(nil, "test-secret", nil, nil, nil)
			_, _, status, _, err := service.callOpenAICompatible(
				context.Background(),
				SceneConfig{Scene: SceneFlowchart},
				orderedProvider{ProviderConfig: ProviderConfig{
					BaseURL:        server.URL,
					Model:          "test-model",
					TimeoutSeconds: 5,
					Scene:          SceneFlowchart,
					EndpointMode:   tt.mode,
				}},
				ChatCompletionInput{Messages: []ChatMessage{{Role: "user", Content: "ping"}}},
			)
			if status != http.StatusOK {
				t.Fatalf("status = %d, want %d", status, http.StatusOK)
			}
			if routeErrorType(err) != ErrorTypeResponseTooLarge {
				t.Fatalf("error type = %q, error = %v", routeErrorType(err), err)
			}
			if !errors.Is(err, upstream.ErrResponseTooLarge) {
				t.Fatalf("error does not retain size cause: %v", err)
			}
			if err == nil || err.Error() != "upstream response exceeded size limit" {
				t.Fatalf("client error = %v", err)
			}
		})
	}
}

func writeRepeatedResponse(t *testing.T, w http.ResponseWriter, size int64) {
	t.Helper()
	chunk := make([]byte, 32*1024)
	for index := range chunk {
		chunk[index] = 'x'
	}
	for remaining := size; remaining > 0; {
		count := int64(len(chunk))
		if count > remaining {
			count = remaining
		}
		if _, err := w.Write(chunk[:count]); err != nil {
			return
		}
		remaining -= count
	}
}
