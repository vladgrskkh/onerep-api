// Package jwt validates access tokens issued by the auth service against its
// JSON Web Key Set.
package jwt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const (
	// jwksPath is the auth service's JWKS route.
	jwksPath = "/v1/.well-known/jwks.json"

	// jwksFetchTimeout bounds JWKS fetches: the default client timeout and the
	// retry context used by GetUserIDFromToken.
	jwksFetchTimeout = 10 * time.Second
)

// Validator caches the auth service's JWKS and validates access tokens.
type Validator struct {
	jwksURL string
	client  *http.Client

	mu     sync.RWMutex
	keySet jwk.Set
}

// NewValidator creates a Validator that fetches the JWKS from the auth
// service at authBaseURL. A nil httpClient defaults to a client with a
// bounded timeout.
func NewValidator(authBaseURL string, httpClient *http.Client) *Validator {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: jwksFetchTimeout}
	}
	return &Validator{
		jwksURL: strings.TrimSuffix(authBaseURL, "/") + jwksPath,
		client:  httpClient,
	}
}

// Refresh fetches the auth service's JWKS and replaces the cached key set.
func (v *Validator) Refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("build jwks request: %w", err)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: unexpected status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read jwks response: %w", err)
	}

	set, err := jwk.Parse(body)
	if err != nil {
		return fmt.Errorf("parse jwks: %w", err)
	}
	if set.Len() == 0 {
		return fmt.Errorf("parse jwks: empty key set")
	}

	v.mu.Lock()
	v.keySet = set
	v.mu.Unlock()
	return nil
}

// GetUserIDFromToken validates tokenString against the cached JWKS and
// returns the subject user ID. Tokens must be RS256-signed and carry an exp
// claim. A stale cached key set is recovered from automatically by a single
// refresh, so key rotation takes effect without an explicit Refresh call.
func (v *Validator) GetUserIDFromToken(tokenString string) (uuid.UUID, error) {
	userID, err := v.parseToken(tokenString)
	if err == nil {
		return userID, nil
	}

	// Auth access tokens carry no kid header, so a rotated key can only be
	// detected as a signature failure; refresh once and retry. The retry is
	// bounded to a single fetch per request.
	if !isSignatureFailure(err) {
		return uuid.Nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), jwksFetchTimeout)
	defer cancel()
	if err := v.Refresh(ctx); err != nil {
		return uuid.Nil, err
	}

	return v.parseToken(tokenString)
}

func (v *Validator) parseToken(tokenString string) (uuid.UUID, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &jwtlib.RegisteredClaims{}, func(t *jwtlib.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		key, err := v.resolveKey(kid)
		if err != nil {
			return nil, err
		}
		var raw any
		if err := key.Raw(&raw); err != nil {
			return nil, fmt.Errorf("extract public key: %w", err)
		}
		return raw, nil
	}, jwtlib.WithValidMethods([]string{jwtlib.SigningMethodRS256.Alg()}), jwtlib.WithExpirationRequired())
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(*jwtlib.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid claims")
	}

	return uuid.Parse(claims.Subject)
}

// resolveKey returns the cached key for kid. Tokens signed without a kid are
// matched against the sole cached key. A miss triggers a single refresh
// before failing.
func (v *Validator) resolveKey(kid string) (jwk.Key, error) {
	if key, ok := v.lookup(kid); ok {
		return key, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), jwksFetchTimeout)
	defer cancel()
	if err := v.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("refresh jwks: %w", err)
	}

	if key, ok := v.lookup(kid); ok {
		return key, nil
	}
	return nil, fmt.Errorf("unknown signing key %q", kid)
}

func (v *Validator) lookup(kid string) (jwk.Key, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.keySet == nil {
		return nil, false
	}
	if kid == "" {
		if v.keySet.Len() == 1 {
			key, ok := v.keySet.Key(0)
			return key, ok
		}
		return nil, false
	}
	for i := range v.keySet.Len() {
		key, ok := v.keySet.Key(i)
		if ok && key.KeyID() == kid {
			return key, true
		}
	}
	return nil, false
}

func isSignatureFailure(err error) bool {
	return errors.Is(err, jwtlib.ErrSignatureInvalid) || errors.Is(err, jwtlib.ErrTokenSignatureInvalid)
}
