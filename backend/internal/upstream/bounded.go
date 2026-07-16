package upstream

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrResponseTooLarge = errors.New("upstream response exceeds size limit")

type ResponseTooLargeError struct {
	Limit int64
}

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("%s (%d bytes)", ErrResponseTooLarge.Error(), e.Limit)
}

func (e *ResponseTooLargeError) Unwrap() error {
	return ErrResponseTooLarge
}

func ReadAll(reader io.Reader, maxBytes int64) ([]byte, error) {
	if reader == nil {
		return nil, errors.New("upstream response reader is required")
	}
	if maxBytes <= 0 || maxBytes == int64(^uint64(0)>>1) {
		return nil, errors.New("upstream response size limit must be positive")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, &ResponseTooLargeError{Limit: maxBytes}
	}
	return data, nil
}

func DecodeJSON(reader io.Reader, maxBytes int64, dst any) error {
	data, err := ReadAll(reader, maxBytes)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("upstream response contains multiple JSON values")
		}
		return err
	}
	return nil
}

func IsResponseTooLarge(err error) bool {
	return errors.Is(err, ErrResponseTooLarge)
}
