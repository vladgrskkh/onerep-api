package s3 //nolint:testpackage // Exercise the unexported constructor without networked bucket bootstrap.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const (
	unitInternalEndpoint = "http://rustfs:9000"
	unitPublicEndpoint   = "https://onerep.mooo.com"
	unitSecretKey        = "test-secret-key"
	unitObjectKey        = "exercises/photo.jpg"
)

type StorageSuite struct {
	suite.Suite
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) TestPresignedPutURLUsesPublicEndpoint() {
	storage := s.newStorage(unitPublicEndpoint, false)
	signedURL, err := storage.PresignedPutURL(s.T().Context(), unitObjectKey, "image/jpeg", time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(signedURL, http.MethodPut, "https", "onerep.mooo.com")
}

func (s *StorageSuite) TestPresignedGetURLUsesPublicEndpoint() {
	storage := s.newStorage(unitPublicEndpoint, false)
	signedURL, err := storage.PresignedGetURL(s.T().Context(), unitObjectKey, time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(signedURL, http.MethodGet, "https", "onerep.mooo.com")
}

func (s *StorageSuite) TestPresignedURLsDefaultToInternalEndpoint() {
	storage := s.newStorage("", false)
	putURL, err := storage.PresignedPutURL(s.T().Context(), unitObjectKey, "image/jpeg", time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(putURL, http.MethodPut, "http", "rustfs:9000")
	getURL, err := storage.PresignedGetURL(s.T().Context(), unitObjectKey, time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(getURL, http.MethodGet, "http", "rustfs:9000")
}

func (s *StorageSuite) TestPresignedURLScheme() {
	for _, tc := range []struct {
		name     string
		endpoint string
		useSSL   bool
		scheme   string
	}{
		{name: "bare host with SSL", endpoint: "media.invalid:9443", useSSL: true, scheme: "https"},
		{name: "bare host without SSL", endpoint: "media.invalid:9443", scheme: "http"},
		{name: "explicit HTTPS", endpoint: "https://media.invalid:9443", scheme: "https"},
		{name: "explicit HTTP", endpoint: "http://media.invalid:9443", useSSL: true, scheme: "http"},
	} {
		s.Run(tc.name, func() {
			storage := s.newStorage(tc.endpoint, tc.useSSL)
			getURL, err := storage.PresignedGetURL(s.T().Context(), unitObjectKey, time.Minute)
			s.Require().NoError(err)
			s.assertSignedURL(getURL, http.MethodGet, tc.scheme, "media.invalid:9443")
		})
	}
}

func (s *StorageSuite) newStorage(publicEndpoint string, useSSL bool) *Storage {
	storage, err := newStorage(context.Background(), unitInternalEndpoint, publicEndpoint,
		"test-access-key", unitSecretKey, "gym-media", useSSL)
	s.Require().NoError(err)
	return storage
}

func (s *StorageSuite) assertSignedURL(rawURL, method, scheme, host string) {
	parsed, err := url.Parse(rawURL)
	s.Require().NoError(err)
	s.Equal(scheme, parsed.Scheme)
	s.Equal(host, parsed.Host)
	s.Equal("/gym-media/exercises/photo.jpg", parsed.Path)
	query := parsed.Query()
	s.Require().NotEmpty(query.Get("X-Amz-Signature"))
	s.Equal("60", query.Get("X-Amz-Expires"))
	s.Equal("AWS4-HMAC-SHA256", query.Get("X-Amz-Algorithm"))
	s.Require().Equal("host", query.Get("X-Amz-SignedHeaders"))

	// Verify independently that the returned host participated in signing.
	// Replacing the URL host after signing must not satisfy this assertion.
	signature := query.Get("X-Amz-Signature")
	query.Del("X-Amz-Signature")
	canonicalQuery := strings.ReplaceAll(query.Encode(), "+", "%20")
	canonicalRequest := method + "\n" + parsed.EscapedPath() + "\n" + canonicalQuery +
		"\nhost:" + parsed.Host + "\n\nhost\nUNSIGNED-PAYLOAD"
	requestHash := sha256.Sum256([]byte(canonicalRequest))
	credential := strings.Split(query.Get("X-Amz-Credential"), "/")
	s.Require().Len(credential, 5)
	s.Equal("test-access-key", credential[0])
	s.Equal("us-east-1", credential[2])
	s.Equal("s3", credential[3])
	s.Equal("aws4_request", credential[4])
	stringToSign := "AWS4-HMAC-SHA256\n" + query.Get("X-Amz-Date") + "\n" +
		strings.Join(credential[1:], "/") + "\n" + hex.EncodeToString(requestHash[:])
	signingKey := []byte("AWS4" + unitSecretKey)
	for _, part := range credential[1:] {
		signingKey = signatureHMAC(signingKey, part)
	}
	s.Equal(hex.EncodeToString(signatureHMAC(signingKey, stringToSign)), signature)
}

func signatureHMAC(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return mac.Sum(nil)
}
