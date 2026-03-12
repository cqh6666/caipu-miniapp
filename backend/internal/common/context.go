package common

import "context"

type contextKey string

const currentUserIDKey contextKey = "currentUserID"

func WithCurrentUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, currentUserIDKey, userID)
}

func CurrentUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(currentUserIDKey).(int64)
	return userID, ok
}
