package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/vladgrskkh/onerep-api/internal/handler"
)

// JWTValidator extracts the authenticated user ID from a bearer token.
type JWTValidator interface {
	GetUserIDFromToken(tokenString string) (uuid.UUID, error)
}

// Authenticate guards routes behind a valid access token and injects the
// authenticated user ID into the request context via handler.WithUserID.
func Authenticate(validator JWTValidator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok {
				handler.WriteError(w, logger, http.StatusUnauthorized, handler.MissingAuthorizationDetail())
				return
			}

			userID, err := validator.GetUserIDFromToken(token)
			if err != nil {
				handler.WriteError(w, logger, http.StatusUnauthorized, handler.InvalidTokenDetail(err))
				return
			}

			next.ServeHTTP(w, r.WithContext(handler.WithUserID(r.Context(), userID)))
		})
	}
}
