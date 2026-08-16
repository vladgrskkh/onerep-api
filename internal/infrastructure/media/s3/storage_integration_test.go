//go:build integration

package s3_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

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
		integrationEndpoint,
		integrationAccessKey,
		integrationSecretKey,
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
		integrationEndpoint,
		integrationAccessKey,
		integrationSecretKey,
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
