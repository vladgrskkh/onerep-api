package testutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/vladgrskkh/onerep-api/internal/handler"
)

// NewRequest builds an HTTP request with the given method, target and body.
func NewRequest(method, target, body string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// AsUser returns a copy of req carrying the given user in its context.
func AsUser(req *http.Request, userID uuid.UUID) *http.Request {
	return req.WithContext(handler.WithUserID(req.Context(), userID))
}

// DecodeError unmarshals the error response body from w.
func DecodeError(t *testing.T, w *httptest.ResponseRecorder) handler.ErrorResponse {
	t.Helper()
	var resp handler.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}
