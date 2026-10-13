//go:build integration

package s3_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	medias3 "github.com/vladgrskkh/onerep-api/internal/infrastructure/media/s3"
)

const (
	integrationEndpoint    = "http://localhost:9000"
	integrationAccessKey   = "minioadmin"
	integrationSecretKey   = "minioadmin"
	integrationBucket      = "gym-media-test"
	integrationHTTPTimeout = 5 * time.Second
)

type StorageIntegrationSuite struct {
	suite.Suite

	storage *medias3.Storage
	ctx     context.Context
}

func (s *StorageIntegrationSuite) SetupTest() {
	storage, err := medias3.NewStorage(
		context.Background(),
		integrationEnv("S3_TEST_ENDPOINT", integrationEndpoint),
		"",
		integrationEnv("S3_TEST_ACCESS_KEY", integrationAccessKey),
		integrationEnv("S3_TEST_SECRET_KEY", integrationSecretKey),
		integrationBucket,
		false,
	)
	s.Require().NoError(err)
	s.storage = storage
	s.ctx = context.Background()
}

func (s *StorageIntegrationSuite) TestUploadAndPresignedGetURL_RoundTrip() {
	key := medias3.GenerateKey("tests")
	payload := []byte("hello from the media storage")

	s.Require().NoError(s.storage.Upload(s.ctx, key, bytes.NewReader(payload), "text/plain"))

	url, err := s.storage.PresignedGetURL(s.ctx, key, time.Minute)
	s.Require().NoError(err)

	client := &http.Client{Timeout: integrationHTTPTimeout}
	resp, err := client.Get(url)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	got, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Equal(payload, got)
}

func (s *StorageIntegrationSuite) TestPresignedPutURL_RoundTrip() {
	key := medias3.GenerateKey("tests")
	payload := []byte("uploaded via a presigned put")

	url, err := s.storage.PresignedPutURL(s.ctx, key, "text/plain", time.Minute)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(s.ctx, http.MethodPut, url, bytes.NewReader(payload))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "text/plain")
	client := &http.Client{Timeout: integrationHTTPTimeout}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	getURL, err := s.storage.PresignedGetURL(s.ctx, key, time.Minute)
	s.Require().NoError(err)
	resp, err = client.Get(getURL)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	got, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Equal(payload, got)
}

func (s *StorageIntegrationSuite) TestDelete_RemovesObject() {
	key := medias3.GenerateKey("tests")
	payload := []byte("to be deleted")
	s.Require().NoError(s.storage.Upload(s.ctx, key, bytes.NewReader(payload), "text/plain"))

	s.Require().NoError(s.storage.Delete(s.ctx, key))

	url, err := s.storage.PresignedGetURL(s.ctx, key, time.Minute)
	s.Require().NoError(err)
	client := &http.Client{Timeout: integrationHTTPTimeout}
	resp, err := client.Get(url)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *StorageIntegrationSuite) TestNewStorage_IsIdempotent() {
	// Rebuilding the storage against the same bucket must not fail.
	_, err := medias3.NewStorage(
		context.Background(),
		integrationEnv("S3_TEST_ENDPOINT", integrationEndpoint),
		"",
		integrationEnv("S3_TEST_ACCESS_KEY", integrationAccessKey),
		integrationEnv("S3_TEST_SECRET_KEY", integrationSecretKey),
		integrationBucket,
		false,
	)
	s.Require().NoError(err)
}

func (s *StorageIntegrationSuite) TestGenerateKey_UniquePerCall() {
	key1 := medias3.GenerateKey("tests")
	key2 := medias3.GenerateKey("tests")
	s.NotEqual(key1, key2)
	s.Equal("tests/", key1[:len("tests/")])
}

func TestStorageIntegrationSuite(t *testing.T) {
	suite.Run(t, new(StorageIntegrationSuite))
}

func (s *StorageIntegrationSuite) TestPublicEndpointKeepsObjectOperationsInternal() {
	ctx, cancel := context.WithTimeout(s.T().Context(), integrationHTTPTimeout)
	defer cancel()
	bucket := "gym-media-test-" + uuid.NewString()
	publicEndpoint := integrationEnv("S3_TEST_PUBLIC_ENDPOINT", "https://media.invalid")
	if publicEndpoint == "" {
		publicEndpoint = "https://media.invalid"
	}
	internalURL, err := url.Parse(integrationEnv("S3_TEST_ENDPOINT", integrationEndpoint))
	s.Require().NoError(err)
	publicURL, err := url.Parse(publicEndpoint)
	s.Require().NoError(err)
	s.Require().NotEqual(internalURL.Host, publicURL.Host)
	storage, err := medias3.NewStorage(ctx,
		integrationEnv("S3_TEST_ENDPOINT", integrationEndpoint), publicEndpoint,
		integrationEnv("S3_TEST_ACCESS_KEY", integrationAccessKey),
		integrationEnv("S3_TEST_SECRET_KEY", integrationSecretKey), bucket, false)
	s.Require().NoError(err, "bucket bootstrap must use the internal endpoint")

	key := medias3.GenerateKey("tests")
	payload := []byte("private object operations with public signing")
	s.Require().NoError(storage.Upload(ctx, key, bytes.NewReader(payload), "text/plain"))
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		var signedURL string
		if method == http.MethodGet {
			signedURL, err = storage.PresignedGetURL(ctx, key, time.Minute)
		} else {
			signedURL, err = storage.PresignedPutURL(ctx, key, "text/plain", time.Minute)
		}
		s.Require().NoError(err)
		parsed, parseErr := url.Parse(signedURL)
		s.Require().NoError(parseErr)
		s.Equal(publicURL.Scheme, parsed.Scheme)
		s.Equal(publicURL.Host, parsed.Host)
		s.Equal("/"+bucket+"/"+key, parsed.Path)
		s.NotEmpty(parsed.Query().Get("X-Amz-Signature"))
	}

	// Read only through a separate internal signer; never dereference public URLs.
	internal, err := medias3.NewStorage(ctx,
		integrationEnv("S3_TEST_ENDPOINT", integrationEndpoint), "",
		integrationEnv("S3_TEST_ACCESS_KEY", integrationAccessKey),
		integrationEnv("S3_TEST_SECRET_KEY", integrationSecretKey), bucket, false)
	s.Require().NoError(err)
	getURL, err := internal.PresignedGetURL(ctx, key, time.Minute)
	s.Require().NoError(err)
	client := &http.Client{Timeout: integrationHTTPTimeout}
	resp, err := client.Get(getURL)
	s.Require().NoError(err)
	got, err := io.ReadAll(resp.Body)
	s.Require().NoError(resp.Body.Close())
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(payload, got)
	s.Require().NoError(storage.Delete(ctx, key))
	resp, err = client.Get(getURL)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func integrationEnv(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}
