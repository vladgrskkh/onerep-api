//go:build integration

package application_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/application"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise/dto"
	validatorjwt "github.com/vladgrskkh/onerep-api/internal/infrastructure/auth/jwt"
)

const (
	testJWKSKeyID   = "onerep-auth-signing-key"
	testDBURL       = "postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable"
	testRedisURL    = "redis://localhost:6379/0"
	testS3Endpoint  = "http://localhost:9000"
	testS3AccessKey = "minioadmin"
	testS3SecretKey = "minioadmin"
	testS3Bucket    = "gym-media-test"
)

// ApplicationIntegrationSuite boots the full application wiring — real
// postgres, redis and minio, a fake JWKS server — and drives the chi router
// through the real auth middleware with locally-signed RS256 tokens.
type ApplicationIntegrationSuite struct {
	suite.Suite

	router http.Handler
	userID uuid.UUID
	token  string
	pool   *pgxpool.Pool
}

func TestApplicationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ApplicationIntegrationSuite))
}

func (s *ApplicationIntegrationSuite) SetupTest() {
	s.T().Setenv("DATABASE_URL", testDBURL)
	s.T().Setenv("REDIS_URL", testRedisURL)
	s.T().Setenv("S3_ENDPOINT", testS3Endpoint)
	s.T().Setenv("S3_ACCESS_KEY", testS3AccessKey)
	s.T().Setenv("S3_SECRET_KEY", testS3SecretKey)
	s.T().Setenv("S3_BUCKET", testS3Bucket)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	s.Require().NoError(err)

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(jwksPayload(s.T(), &key.PublicKey))
	}))
	s.T().Cleanup(jwksSrv.Close)

	app, err := application.New(
		application.WithLogger(logger),
		application.WithDatabase(ctx),
		application.WithRedis(ctx),
		application.WithS3(),
		application.WithServices(),
		application.WithHandlers(),
	)
	s.Require().NoError(err)

	validator := validatorjwt.NewValidator(jwksSrv.URL, nil)
	s.Require().NoError(validator.Refresh(ctx))
	s.router = app.RegisterRoutes(validator)

	s.userID = uuid.Must(uuid.NewV7())
	s.token = signToken(s.T(), key, s.userID)

	pool, err := pgxpool.New(ctx, testDBURL)
	s.Require().NoError(err)
	s.pool = pool
}

func (s *ApplicationIntegrationSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *ApplicationIntegrationSuite) TestHealth() {
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/health", nil))

	s.Equal(http.StatusOK, w.Code)
	s.JSONEq(`{"status":"ok"}`, w.Body.String())
}

func (s *ApplicationIntegrationSuite) TestProtectedRoutesRequireToken() {
	for _, tc := range []struct {
		name       string
		target     string
		wantCode   string
		authorized bool
	}{
		{name: "no token", target: "/v1/exercises", wantCode: "MISSING_AUTHORIZATION_HEADER"},
		{name: "garbage token", target: "/v1/exercises", wantCode: "INVALID_TOKEN", authorized: true},
		{name: "workouts", target: "/v1/workouts", wantCode: "MISSING_AUTHORIZATION_HEADER"},
		{name: "progress", target: "/v1/progress/1rm", wantCode: "MISSING_AUTHORIZATION_HEADER"},
	} {
		s.Run(tc.name, func() {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			if tc.authorized {
				req.Header.Set("Authorization", "Bearer not-a-jwt")
			}
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)

			s.Equal(http.StatusUnauthorized, w.Code)
			var resp handler.ErrorResponse
			s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
			s.Equal(handler.ErrorCode(tc.wantCode), resp.Error.Code)
		})
	}
}

func (s *ApplicationIntegrationSuite) TestExerciseCRUDFlow() {
	exerciseID := s.createExercise("Front Squat", "Barbell squat held in the front rack")
	exercise := s.getExercise(exerciseID)
	s.Equal(
		s.userID,
		exercise.CreatedByUserID,
		"created_by_user_id must come from the token subject",
	)

	updated := s.updateExercise(exerciseID, "Front Squat (High Bar)")
	s.Equal("Front Squat (High Bar)", updated.Name)

	reloaded := s.getExercise(exerciseID)
	s.Equal("Front Squat (High Bar)", reloaded.Name)
	s.Equal(2, reloaded.Version)

	s.deleteExercise(exerciseID)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(
		w,
		s.authorizedRequest(http.MethodGet, "/v1/exercises/"+exerciseID.String(), nil),
	)
	s.Equal(http.StatusNotFound, w.Code, "soft-deleted exercise must not be readable")
}

func (s *ApplicationIntegrationSuite) createExercise(name, description string) uuid.UUID {
	body, err := json.Marshal(dto.ExerciseCreateRequest{Name: name, Description: description})
	s.Require().NoError(err)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, s.authorizedRequest(http.MethodPost, "/v1/exercises", body))

	s.Require().Equal(http.StatusCreated, w.Code, w.Body.String())
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.NotEqual(uuid.Nil, resp.ID)

	s.T().Cleanup(func() {
		_, _ = s.pool.Exec(context.Background(),
			`DELETE FROM gym.exercises WHERE id = $1`, resp.ID)
	})

	return resp.ID
}

func (s *ApplicationIntegrationSuite) getExercise(id uuid.UUID) dto.ExerciseResponse {
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, s.authorizedRequest(http.MethodGet, "/v1/exercises/"+id.String(), nil))

	s.Require().Equal(http.StatusOK, w.Code, w.Body.String())
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func (s *ApplicationIntegrationSuite) updateExercise(
	id uuid.UUID,
	name string,
) dto.ExerciseResponse {
	body, err := json.Marshal(dto.ExerciseUpdateRequest{Name: &name})
	s.Require().NoError(err)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, s.authorizedRequest(http.MethodPatch, "/v1/exercises/"+id.String(), body))

	s.Require().Equal(http.StatusOK, w.Code, w.Body.String())
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func (s *ApplicationIntegrationSuite) deleteExercise(id uuid.UUID) {
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, s.authorizedRequest(http.MethodDelete, "/v1/exercises/"+id.String(), nil))

	s.Equal(http.StatusNoContent, w.Code, w.Body.String())
}

func (s *ApplicationIntegrationSuite) authorizedRequest(
	method, target string,
	body []byte,
) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+s.token)
	return req
}

// jwksPayload builds the JWKS response body exactly as the auth service's
// JWKSHandler does: one RSA key with kid and RS256 alg.
func jwksPayload(t *testing.T, publicKey *rsa.PublicKey) []byte {
	t.Helper()
	jwkKey, err := jwk.FromRaw(publicKey)
	if err != nil {
		t.Fatalf("build jwk key: %v", err)
	}
	if err := jwkKey.Set(jwk.KeyIDKey, testJWKSKeyID); err != nil {
		t.Fatalf("set kid: %v", err)
	}
	if err := jwkKey.Set(jwk.AlgorithmKey, "RS256"); err != nil {
		t.Fatalf("set alg: %v", err)
	}

	set := jwk.NewSet()
	if err := set.AddKey(jwkKey); err != nil {
		t.Fatalf("add key to set: %v", err)
	}

	body, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	return body
}

// signToken produces an RS256 token carrying the user ID as subject, like the
// auth service's access tokens (no kid header, no iss/aud).
func signToken(t *testing.T, key *rsa.PrivateKey, userID uuid.UUID) string {
	t.Helper()
	now := time.Now()
	claims := jwtlib.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwtlib.NewNumericDate(now),
		ExpiresAt: jwtlib.NewNumericDate(now.Add(time.Hour)),
	}
	signed, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
