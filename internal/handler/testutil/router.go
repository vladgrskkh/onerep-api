package testutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/vladgrskkh/onerep-api/internal/handler"
)

// Route pairs an HTTP method and path pattern with a handler function.
type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}

// NewRouter builds a chi router with the given routes registered.
func NewRouter(routes ...Route) *chi.Mux {
	router := chi.NewRouter()
	for _, r := range routes {
		router.Method(r.Method, r.Pattern, r.Handler)
	}
	return router
}

// Serve builds a request with the given body and serves it on the router.
// When userID is not the nil UUID it is injected into the request context.
func Serve(router *chi.Mux, method, target, body string, userID uuid.UUID) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
	if userID != uuid.Nil {
		req = req.WithContext(handler.WithUserID(req.Context(), userID))
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// DecodeError unmarshals the error response body from w.
func DecodeError(t *testing.T, w *httptest.ResponseRecorder) handler.ErrorResponse {
	t.Helper()
	var resp handler.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}
