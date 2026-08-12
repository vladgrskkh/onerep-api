package jwt_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/suite"

	validatorjwt "github.com/vladgrskkh/onerep-api/internal/infrastructure/auth/jwt"
)

const testKeyID = "onerep-auth-signing-key"

type ValidatorTestSuite struct {
	suite.Suite

	key    *rsa.PrivateKey
	userID uuid.UUID
}

func (s *ValidatorTestSuite) SetupTest() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	s.Require().NoError(err)
	s.key = key
	s.userID = uuid.Must(uuid.NewV7())
}

// testClaims mirrors the auth service's access token claims: sub, iat, exp
// and email, with no iss/aud.
type testClaims struct {
	jwtlib.RegisteredClaims

	Email string `json:"email"`
}

func (s *ValidatorTestSuite) claims(exp time.Time) testClaims {
	return testClaims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   s.userID.String(),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			ExpiresAt: jwtlib.NewNumericDate(exp),
		},
		Email: "test@example.com",
	}
}

// signToken produces an RS256 token signed with key. Like the auth service's
// TokenManager, the token carries no kid header.
func (s *ValidatorTestSuite) signToken(key *rsa.PrivateKey, claims testClaims) string {
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	s.Require().NoError(err)
	return signed
}

// signTokenWithKid is signToken with an explicit kid header, for tests that
// exercise kid-based key selection.
func (s *ValidatorTestSuite) signTokenWithKid(key *rsa.PrivateKey, kid string, claims testClaims) string {
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	s.Require().NoError(err)
	return signed
}

// jwksPayload builds the JWKS response body exactly as the auth service's
// JWKSHandler does: one RSA key with kid and RS256 alg.
func (s *ValidatorTestSuite) jwksPayload(key *rsa.PublicKey) []byte {
	jwkKey, err := jwk.FromRaw(key)
	s.Require().NoError(err)
	s.Require().NoError(jwkKey.Set(jwk.KeyIDKey, testKeyID))
	s.Require().NoError(jwkKey.Set(jwk.AlgorithmKey, "RS256"))

	set := jwk.NewSet()
	s.Require().NoError(set.AddKey(jwkKey))

	body, err := json.Marshal(set)
	s.Require().NoError(err)
	return body
}

// serveJWKS starts an httptest server serving the JWKS bytes that current
// points to; rotating the pointed-to payload simulates key rotation.
func (s *ValidatorTestSuite) serveJWKS(current *[]byte) *httptest.Server {
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		body := *current
		mu.Unlock()
		w.Write(body)
	}))
}

func (s *ValidatorTestSuite) TestRefreshAndValidate() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	token := s.signToken(s.key, s.claims(time.Now().Add(time.Hour)))
	userID, err := validator.GetUserIDFromToken(token)
	s.Require().NoError(err)
	s.Equal(s.userID, userID)
}

func (s *ValidatorTestSuite) TestTokenWithoutKidMatchesSoleKey() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	claims := s.claims(time.Now().Add(time.Hour))
	claims.Audience = []string{"gym-api"}
	token := s.signToken(s.key, claims)

	userID, err := validator.GetUserIDFromToken(token)
	s.Require().NoError(err)
	s.Equal(s.userID, userID)
}

func (s *ValidatorTestSuite) TestExpiredToken() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	token := s.signToken(s.key, s.claims(time.Now().Add(-time.Hour)))
	_, err := validator.GetUserIDFromToken(token)
	s.Error(err)
}

func (s *ValidatorTestSuite) TestTokenWithoutExpiration() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	claims := s.claims(time.Now().Add(time.Hour))
	claims.ExpiresAt = nil
	token := s.signToken(s.key, claims)

	_, err := validator.GetUserIDFromToken(token)
	s.Error(err)
}

func (s *ValidatorTestSuite) TestTamperedToken() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	parts := strings.Split(s.signToken(s.key, s.claims(time.Now().Add(time.Hour))), ".")
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	s.Require().NoError(err)
	otherID := uuid.Must(uuid.NewV7())
	payloadBytes = bytes.Replace(payloadBytes, []byte(s.userID.String()), []byte(otherID.String()), 1)
	parts[1] = base64.RawURLEncoding.EncodeToString(payloadBytes)

	_, err = validator.GetUserIDFromToken(strings.Join(parts, "."))
	s.Error(err)
}

func (s *ValidatorTestSuite) TestWrongSigningMethod() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, s.claims(time.Now().Add(time.Hour)))
	hs256, err := token.SignedString([]byte("test-secret"))
	s.Require().NoError(err)

	_, err = validator.GetUserIDFromToken(hs256)
	s.Error(err)
}

func (s *ValidatorTestSuite) TestInvalidSubject() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	claims := s.claims(time.Now().Add(time.Hour))
	claims.Subject = "not-a-uuid"
	token := s.signToken(s.key, claims)

	_, err := validator.GetUserIDFromToken(token)
	s.Error(err)
}

func (s *ValidatorTestSuite) TestValidateWithoutRefresh() {
	validator := validatorjwt.NewValidator("http://127.0.0.1:1", nil)

	token := s.signToken(s.key, s.claims(time.Now().Add(time.Hour)))
	_, err := validator.GetUserIDFromToken(token)
	s.Error(err)
}

func (s *ValidatorTestSuite) TestRefreshNonOKStatus() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Error(validator.Refresh(context.Background()))
}

func (s *ValidatorTestSuite) TestKeyRotationRecoversAutomatically() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)
	defer srv.Close()

	// Both validators cache the old key before the auth service rotates.
	kidValidator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(kidValidator.Refresh(context.Background()))
	kidlessValidator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(kidlessValidator.Refresh(context.Background()))

	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	s.Require().NoError(err)
	payload = s.jwksPayload(&newKey.PublicKey)

	// A token with an explicit kid misses the cached key set and triggers a
	// refresh retry inside the keyfunc.
	token := s.signTokenWithKid(newKey, testKeyID, s.claims(time.Now().Add(time.Hour)))
	userID, err := kidValidator.GetUserIDFromToken(token)
	s.Require().NoError(err)
	s.Equal(s.userID, userID)

	// Real auth tokens carry no kid; a rotated key surfaces as a signature
	// failure and triggers a refresh retry as well.
	token = s.signToken(newKey, s.claims(time.Now().Add(time.Hour)))
	userID, err = kidlessValidator.GetUserIDFromToken(token)
	s.Require().NoError(err)
	s.Equal(s.userID, userID)
}

func (s *ValidatorTestSuite) TestRotationWhenJWKSUnavailable() {
	payload := s.jwksPayload(&s.key.PublicKey)
	srv := s.serveJWKS(&payload)

	validator := validatorjwt.NewValidator(srv.URL, nil)
	s.Require().NoError(validator.Refresh(context.Background()))

	// A token still signed by the cached key validates even while the auth
	// service is down.
	token := s.signToken(s.key, s.claims(time.Now().Add(time.Hour)))
	userID, err := validator.GetUserIDFromToken(token)
	s.Require().NoError(err)
	s.Equal(s.userID, userID)

	// The auth service rotates its key and goes down; validation can no
	// longer recover.
	srv.Close()
	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	s.Require().NoError(err)
	_, err = validator.GetUserIDFromToken(s.signToken(newKey, s.claims(time.Now().Add(time.Hour))))
	s.Error(err)
}

func TestValidatorSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ValidatorTestSuite))
}
