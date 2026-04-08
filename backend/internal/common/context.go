package common

import (
	"context"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const (
	currentUserIDKey       contextKey = "currentUserID"
	currentAdminSubjectKey contextKey = "currentAdminSubject"
)

func WithCurrentUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, currentUserIDKey, userID)
}

func CurrentUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(currentUserIDKey).(int64)
	return userID, ok
}

func WithCurrentAdminSubject(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, currentAdminSubjectKey, subject)
}

func CurrentAdminSubject(ctx context.Context) (string, bool) {
	subject, ok := ctx.Value(currentAdminSubjectKey).(string)
	return subject, ok
}

func RequestID(ctx context.Context) string {
	return chimiddleware.GetReqID(ctx)
}
