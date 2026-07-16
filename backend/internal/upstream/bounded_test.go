package upstream

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestReadAllDistinguishesExactLimitFromOversize(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"ok":true}`)
	data, err := ReadAll(bytes.NewReader(payload), int64(len(payload)))
	if err != nil || !bytes.Equal(data, payload) {
		t.Fatalf("exact limit data=%q err=%v", data, err)
	}
	if _, err := ReadAll(bytes.NewReader(append(payload, ' ')), int64(len(payload))); !IsResponseTooLarge(err) {
		t.Fatalf("oversize error=%v", err)
	}
}

func TestDecodeJSONStopsInfiniteReaderAtLimit(t *testing.T) {
	t.Parallel()

	reader := repeatReader{value: 'x'}
	var target map[string]any
	err := DecodeJSON(reader, 1024, &target)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("error=%v", err)
	}
}

func TestDecodeJSONAcceptsWhitespaceAtExactLimitAndRejectsMultipleValues(t *testing.T) {
	t.Parallel()

	payload := []byte("{\"ok\":true}   ")
	var target map[string]any
	if err := DecodeJSON(bytes.NewReader(payload), int64(len(payload)), &target); err != nil {
		t.Fatal(err)
	}
	if err := DecodeJSON(bytes.NewBufferString(`{"ok":true}{"other":true}`), 128, &target); err == nil {
		t.Fatal("expected multiple JSON values to be rejected")
	}
}

func BenchmarkReadAllBoundedOversize(b *testing.B) {
	payload := bytes.Repeat([]byte{'x'}, 2*1024*1024)
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		_, _ = ReadAll(bytes.NewReader(payload), 1024*1024)
	}
}

type repeatReader struct {
	value byte
}

func (r repeatReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = r.value
	}
	return len(buffer), nil
}

var _ io.Reader = repeatReader{}
