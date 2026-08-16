package testutil

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/vladgrskkh/onerep-api/internal/handler"
)

// Serve builds a request with the given body and serves it on the router.
// When userID is not the nil UUID it is injected into the request context.
func Serve(router *chi.Mux, method, target, body string, userID uuid.UUID) *httptest.ResponseRecorder {
	return serve(router, method, target, body, userID, nil)
}

// ServeJSON is like Serve but sets the request Content-Type header to
// application/json.
func ServeJSON(router *chi.Mux, method, target, body string, userID uuid.UUID) *httptest.ResponseRecorder {
	return serve(router, method, target, body, userID, map[string]string{"Content-Type": "application/json"})
}

func serve(
	router *chi.Mux,
	method, target, body string,
	userID uuid.UUID,
	headers map[string]string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
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
