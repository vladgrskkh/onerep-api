package middleware_test

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/middleware"
	"github.com/vladgrskkh/onerep-api/internal/handler/middleware/mocks"
	"github.com/vladgrskkh/onerep-api/internal/handler/testutil"
)

type AuthenticateTestSuite struct {
	suite.Suite

	validator *mocks.MockJWTValidator
	router    *chi.Mux
	userID    uuid.UUID
}

func (s *AuthenticateTestSuite) SetupTest() {
	s.validator = mocks.NewMockJWTValidator(s.T())
	s.userID = uuid.Must(uuid.NewV7())

	logger := slog.New(slog.DiscardHandler)
	s.router = chi.NewRouter()
	s.router.Use(middleware.Authenticate(s.validator, logger))
	s.router.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, logger, http.StatusOK, map[string]string{
			"user_id": handler.UserIDFromContext(r.Context()).String(),
		})
	})
}

func (s *AuthenticateTestSuite) serve(token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *AuthenticateTestSuite) TestMissingAuthorizationHeader() {
	w := s.serve("")
	s.Equal(http.StatusUnauthorized, w.Code)
	s.Equal(handler.ErrorCode("MISSING_AUTHORIZATION_HEADER"), testutil.DecodeError(s.T(), w).Error.Code)
}

func (s *AuthenticateTestSuite) TestMalformedAuthorizationHeader() {
	w := s.serve("Bearer")
	s.Equal(http.StatusUnauthorized, w.Code)

	w = s.serve("Basic dXNlcjpwYXNz")
	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *AuthenticateTestSuite) TestValidToken() {
	s.validator.EXPECT().GetUserIDFromToken("valid-token").Return(s.userID, nil)

	w := s.serve("Bearer valid-token")
	s.Equal(http.StatusOK, w.Code)
	s.JSONEq(`{"user_id":"`+s.userID.String()+`"}`, w.Body.String())
}

func (s *AuthenticateTestSuite) TestInvalidToken() {
	s.validator.EXPECT().GetUserIDFromToken("invalid-token").Return(uuid.Nil, errors.New("token is expired"))

	w := s.serve("Bearer invalid-token")
	s.Equal(http.StatusUnauthorized, w.Code)
	s.Equal(handler.ErrorCode("INVALID_TOKEN"), testutil.DecodeError(s.T(), w).Error.Code)
}

func TestAuthenticateSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(AuthenticateTestSuite))
}
