package s3_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	medias3 "github.com/vladgrskkh/onerep-api/internal/infrastructure/media/s3"
)

const (
	unitPublicEndpoint = "https://onerep.mooo.com"
	unitSecretKey      = "test-secret-key"
	unitObjectKey      = "exercises/photo.jpg"
)

type StorageSuite struct {
	suite.Suite
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) TestPresignedPutURLUsesPublicEndpoint() {
	storage, _ := s.newTestStorage(unitPublicEndpoint, false)
	signedURL, err := storage.PresignedPutURL(s.T().Context(), unitObjectKey, "image/jpeg", time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(signedURL, http.MethodPut, "https", "onerep.mooo.com")
}

func (s *StorageSuite) TestPresignedGetURLUsesPublicEndpoint() {
	storage, _ := s.newTestStorage(unitPublicEndpoint, false)
	signedURL, err := storage.PresignedGetURL(s.T().Context(), unitObjectKey, time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(signedURL, http.MethodGet, "https", "onerep.mooo.com")
}

func (s *StorageSuite) TestPresignedURLsDefaultToInternalEndpoint() {
	storage, internalEndpoint := s.newTestStorage("", false)
	putURL, err := storage.PresignedPutURL(s.T().Context(), unitObjectKey, "image/jpeg", time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(putURL, http.MethodPut, "http", endpointHost(internalEndpoint))
	getURL, err := storage.PresignedGetURL(s.T().Context(), unitObjectKey, time.Minute)
	s.Require().NoError(err)
	s.assertSignedURL(getURL, http.MethodGet, "http", endpointHost(internalEndpoint))
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
			storage, _ := s.newTestStorage(tc.endpoint, tc.useSSL)
			getURL, err := storage.PresignedGetURL(s.T().Context(), unitObjectKey, time.Minute)
			s.Require().NoError(err)
			s.assertSignedURL(getURL, http.MethodGet, tc.scheme, "media.invalid:9443")
		})
	}
}

func (s *StorageSuite) newTestStorage(publicEndpoint string, useSSL bool) (*medias3.Storage, string) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/gym-media" {
			http.Error(w, "unexpected S3 request", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	s.T().Cleanup(server.Close)

	storage, err := medias3.NewStorage(context.Background(), server.URL, publicEndpoint,
		"test-access-key", unitSecretKey, "gym-media", useSSL)
	s.Require().NoError(err)
	return storage, server.URL
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

func endpointHost(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		panic(err)
	}
	return parsed.Host
}

func signatureHMAC(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return mac.Sum(nil)
}
