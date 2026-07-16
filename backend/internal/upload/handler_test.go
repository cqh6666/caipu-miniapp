package upload

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadImageReturnsPayloadTooLargeForChunkedMultipart(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "oversized.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(strings.Repeat("x", 2*1024*1024+1))); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(NewService(t.TempDir(), "", 1))
	request := httptest.NewRequest(http.MethodPost, "/api/uploads/images", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.ContentLength = -1
	response := httptest.NewRecorder()

	handler.UploadImage(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestUploadImageUsesConfiguredPublicBaseURL(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "image.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(NewService(t.TempDir(), "https://static.example.com/uploads", 1))
	request := httptest.NewRequest(http.MethodPost, "/api/uploads/images", &body)
	request.Host = "attacker.example"
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Forwarded-Host", "forwarded-attacker.example")
	request.Header.Set("X-Forwarded-Proto", "javascript")
	response := httptest.NewRecorder()

	handler.UploadImage(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "https://static.example.com/uploads/") {
		t.Fatalf("response does not use configured public base URL: %s", response.Body.String())
	}
	for _, attackerValue := range []string{"attacker.example", "forwarded-attacker.example", "javascript"} {
		if strings.Contains(response.Body.String(), attackerValue) {
			t.Fatalf("response URL contains untrusted value %q: %s", attackerValue, response.Body.String())
		}
	}
}

func TestRequestBaseURLIgnoresForwardingHeaders(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "http://backend.local/api/uploads/images", nil)
	request.Header.Set("X-Forwarded-Host", "attacker.example")
	request.Header.Set("X-Forwarded-Proto", "javascript")

	if got := requestBaseURL(request); got != "http://backend.local" {
		t.Fatalf("requestBaseURL=%q, want direct request host", got)
	}
}
