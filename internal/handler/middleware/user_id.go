package middleware

// CtxKey is the type of context keys stored by middleware.
type CtxKey string

// UserIDKey is the context key under which the authenticated user ID is stored.
const UserIDKey CtxKey = "user_id"
