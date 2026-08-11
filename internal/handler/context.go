package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/vladgrskkh/onerep-api/internal/handler/middleware"
)

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, id)
}

func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
