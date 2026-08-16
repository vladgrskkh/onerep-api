package handler

import (
	"context"

	"github.com/google/uuid"
)

// ctxKey is the type of context keys used by handlers.
type ctxKey string

// userIDKey is the context key under which the authenticated user ID is stored.
const userIDKey ctxKey = "user_id"

// WithUserID returns a context carrying the authenticated user ID.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromContext returns the authenticated user ID from ctx, or uuid.Nil
// when the request is not authenticated.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
