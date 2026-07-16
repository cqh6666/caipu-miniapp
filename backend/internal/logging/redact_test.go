package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactingHandlerRemovesSecretsAndStructuredPayloads(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(NewRedactingHandler(slog.NewJSONHandler(&output, nil)))
	logger.Error(
		"upstream failed with Authorization: Bearer header-secret",
		"api_key", "attribute-secret",
		"error", errors.New(`POST https://user:password@example.com/private/path?token=query-secret failed: {"messages":[{"content":"private prompt"}]}`),
		"safe_id", "recipe-42",
	)

	logOutput := output.String()
	for _, secret := range []string{"header-secret", "attribute-secret", "user:password", "/private/path", "query-secret", "private prompt"} {
		if strings.Contains(logOutput, secret) {
			t.Fatalf("log leaked %q: %s", secret, logOutput)
		}
	}
	for _, expected := range []string{"[REDACTED]", "[structured payload redacted]", "recipe-42", "https://example.com"} {
		if !strings.Contains(logOutput, expected) {
			t.Fatalf("log does not contain %q: %s", expected, logOutput)
		}
	}
}

func TestSafeErrorSummaryPreservesContextAndTypeChain(t *testing.T) {
	t.Parallel()

	err := errors.New("database unavailable")
	wrapped := &testWrappedError{message: "load user", err: err}
	if got := SafeErrorSummary(wrapped); !strings.Contains(got, "load user") || !strings.Contains(got, "database unavailable") {
		t.Fatalf("summary=%q", got)
	}
	if got := ErrorTypeChain(wrapped); len(got) != 2 {
		t.Fatalf("type chain=%#v", got)
	}
}

type testWrappedError struct {
	message string
	err     error
}

func (e *testWrappedError) Error() string { return e.message }
func (e *testWrappedError) Unwrap() error { return e.err }
