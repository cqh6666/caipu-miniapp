package common

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"code":50000,"message":"internal server error","data":null}`, http.StatusInternalServerError)
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
