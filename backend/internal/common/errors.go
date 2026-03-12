package common

import "net/http"

const (
	CodeOK             = 0
	CodeBadRequest     = 40000
	CodeUnauthorized   = 40001
	CodeForbidden      = 40003
	CodeNotFound       = 40400
	CodeConflict       = 40900
	CodeUnprocessable  = 42200
	CodeInternalServer = 50000
)

type AppError struct {
	Code       int
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithErr(err error) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		HTTPStatus: e.HTTPStatus,
		Err:        err,
	}
}

func NewAppError(code int, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

var (
	ErrBadRequest   = NewAppError(CodeBadRequest, "bad request", http.StatusBadRequest)
	ErrUnauthorized = NewAppError(CodeUnauthorized, "unauthorized", http.StatusUnauthorized)
	ErrForbidden    = NewAppError(CodeForbidden, "forbidden", http.StatusForbidden)
	ErrNotFound     = NewAppError(CodeNotFound, "not found", http.StatusNotFound)
	ErrConflict     = NewAppError(CodeConflict, "conflict", http.StatusConflict)
	ErrInternal     = NewAppError(CodeInternalServer, "internal server error", http.StatusInternalServerError)
)
