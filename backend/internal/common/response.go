package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		ObserveError(w, fmt.Errorf("encode JSON response: %w", err))
		status = http.StatusInternalServerError
		encoded = []byte(`{"code":50000,"message":"internal server error","data":null}`)
	}
	encoded = append(encoded, '\n')

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(encoded); err != nil {
		ObserveError(w, fmt.Errorf("write JSON response: %w", err))
	}
}

func WriteData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, Response{
		Code:    CodeOK,
		Message: "ok",
		Data:    data,
	})
}

func WriteError(w http.ResponseWriter, err error) {
	ObserveError(w, err)
	var appErr *AppError
	if errors.As(err, &appErr) {
		WriteJSON(w, appErr.HTTPStatus, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
			Data:    nil,
		})
		return
	}

	WriteJSON(w, http.StatusInternalServerError, Response{
		Code:    CodeInternalServer,
		Message: "internal server error",
		Data:    nil,
	})
}

type responseErrorObserver interface {
	ObserveError(error)
}

type responseWriterUnwrapper interface {
	Unwrap() http.ResponseWriter
}

func ObserveError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	for current := w; current != nil; {
		if observer, ok := current.(responseErrorObserver); ok {
			observer.ObserveError(err)
			return
		}
		unwrapper, ok := current.(responseWriterUnwrapper)
		if !ok {
			return
		}
		current = unwrapper.Unwrap()
	}
}
