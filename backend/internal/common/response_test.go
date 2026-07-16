package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestWriteJSONEncodesBeforeWritingHeaders(t *testing.T) {
	t.Parallel()

	writer := &observedResponseWriter{header: make(http.Header)}
	WriteData(writer, http.StatusOK, map[string]any{"invalid": make(chan int)})

	if len(writer.statuses) != 1 || writer.statuses[0] != http.StatusInternalServerError {
		t.Fatalf("statuses=%v, want one 500", writer.statuses)
	}
	if got := writer.header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type=%q", got)
	}
	if body := writer.body.String(); body != `{"code":50000,"message":"internal server error","data":null}`+"\n" {
		t.Fatalf("body=%q", body)
	}
	if len(writer.errors) != 1 || !strings.Contains(writer.errors[0].Error(), "encode JSON response") {
		t.Fatalf("observed errors=%v", writer.errors)
	}
}

func TestObserveErrorTraversesWrappedResponseWriter(t *testing.T) {
	t.Parallel()

	base := &observedResponseWriter{header: make(http.Header)}
	wrapper := unwrapResponseWriter{ResponseWriter: base}
	want := errors.New("test error")
	ObserveError(wrapper, want)
	if len(base.errors) != 1 || !errors.Is(base.errors[0], want) {
		t.Fatalf("observed errors=%v", base.errors)
	}
}

func TestWriteErrorKeepsStableStatusEnvelopeAndObservedCause(t *testing.T) {
	t.Parallel()

	internalCause := errors.New("database password=must-not-reach-client")
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    int
		wantMessage string
	}{
		{name: "bad request", err: ErrBadRequest, wantStatus: http.StatusBadRequest, wantCode: CodeBadRequest, wantMessage: "bad request"},
		{name: "unauthorized", err: ErrUnauthorized, wantStatus: http.StatusUnauthorized, wantCode: CodeUnauthorized, wantMessage: "unauthorized"},
		{name: "forbidden", err: ErrForbidden, wantStatus: http.StatusForbidden, wantCode: CodeForbidden, wantMessage: "forbidden"},
		{name: "not found", err: ErrNotFound, wantStatus: http.StatusNotFound, wantCode: CodeNotFound, wantMessage: "not found"},
		{name: "method not allowed", err: ErrMethodNotAllowed, wantStatus: http.StatusMethodNotAllowed, wantCode: CodeMethodNotAllowed, wantMessage: "method not allowed"},
		{name: "conflict", err: ErrConflict, wantStatus: http.StatusConflict, wantCode: CodeConflict, wantMessage: "conflict"},
		{name: "wrapped internal", err: ErrInternal.WithErr(internalCause), wantStatus: http.StatusInternalServerError, wantCode: CodeInternalServer, wantMessage: "internal server error"},
		{name: "raw internal", err: internalCause, wantStatus: http.StatusInternalServerError, wantCode: CodeInternalServer, wantMessage: "internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := &observedResponseWriter{header: make(http.Header)}
			WriteError(writer, test.err)
			if len(writer.statuses) != 1 || writer.statuses[0] != test.wantStatus {
				t.Fatalf("statuses=%v, want one %d", writer.statuses, test.wantStatus)
			}
			var response Response
			if err := json.Unmarshal(writer.body.Bytes(), &response); err != nil {
				t.Fatalf("decode body: %v; body=%s", err, writer.body.String())
			}
			if response.Code != test.wantCode || response.Message != test.wantMessage || response.Data != nil {
				t.Fatalf("response=%#v", response)
			}
			if len(writer.errors) != 1 || !errors.Is(writer.errors[0], test.err) {
				t.Fatalf("observed errors=%v", writer.errors)
			}
			if strings.Contains(writer.body.String(), "must-not-reach-client") {
				t.Fatalf("response leaked internal cause: %s", writer.body.String())
			}
		})
	}
}

type observedResponseWriter struct {
	header   http.Header
	statuses []int
	body     bytes.Buffer
	errors   []error
}

func (w *observedResponseWriter) Header() http.Header { return w.header }

func (w *observedResponseWriter) WriteHeader(status int) {
	w.statuses = append(w.statuses, status)
}

func (w *observedResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *observedResponseWriter) ObserveError(err error) {
	w.errors = append(w.errors, err)
}

type unwrapResponseWriter struct {
	http.ResponseWriter
}

func (w unwrapResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
