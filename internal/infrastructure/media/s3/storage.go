package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// Storage is the S3 media bucket adapter.
type Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

// NewStorage uses internalEndpoint for bucket and object operations and
// publicEndpoint for presigned URLs. An empty publicEndpoint falls back to
// internalEndpoint. useSSL only applies to endpoints without a scheme.
func NewStorage(
	ctx context.Context,
	internalEndpoint, publicEndpoint, accessKey, secretKey, bucket string,
	useSSL bool,
) (*Storage, error) {
	storage, err := newStorage(ctx, internalEndpoint, publicEndpoint, accessKey, secretKey, bucket, useSSL)
	if err != nil {
		return nil, err
	}
	if err := ensureBucket(ctx, storage.client, bucket); err != nil {
		return nil, err
	}
	return storage, nil
}

// newStorage constructs the adapter without networked bucket bootstrap.
func newStorage(
	ctx context.Context,
	internalEndpoint, publicEndpoint, accessKey, secretKey, bucket string,
	useSSL bool,
) (*Storage, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithBaseEndpoint(endpointWithScheme(internalEndpoint, useSSL)),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	if publicEndpoint == "" {
		publicEndpoint = internalEndpoint
	}
	presigner := s3.NewPresignClient(client, s3.WithPresignClientFromClientOptions(func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpointWithScheme(publicEndpoint, useSSL))
	}))
	return &Storage{client: client, presigner: presigner, bucket: bucket}, nil
}

// Upload stores body under key with the given content type.
func (s *Storage) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	return err
}

// Delete removes the object at key. Deleting a missing object is not an
// error.
func (s *Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// PresignedGetURL returns a time-limited URL for downloading the object at
// key without credentials.
func (s *Storage) PresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// PresignedPutURL returns a time-limited URL for uploading an object with
// the given content type to key without credentials.
func (s *Storage) PresignedPutURL(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// GenerateKey builds a unique object key under the given prefix.
func GenerateKey(prefix string) string {
	return prefix + "/" + uuid.Must(uuid.NewV7()).String()
}

// GenerateKey builds a unique object key under the given prefix.
func (s *Storage) GenerateKey(prefix string) string {
	return GenerateKey(prefix)
}

// endpointWithScheme prepends a scheme to the endpoint when it has none.
func endpointWithScheme(endpoint string, useSSL bool) string {
	if strings.Contains(endpoint, "://") {
		return endpoint
	}
	if useSSL {
		return "https://" + endpoint
	}
	return "http://" + endpoint
}

// ensureBucket creates the bucket when missing. A bucket that already exists
// (owned by us or by someone else) is not an error, matching the behavior of
// the default AWS us-east-1 CreateBucket with no location constraint.
func ensureBucket(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}
	var alreadyExists *types.BucketAlreadyExists
	var alreadyOwned *types.BucketAlreadyOwnedByYou
	if errors.As(err, &alreadyExists) || errors.As(err, &alreadyOwned) {
		return nil
	}
	return fmt.Errorf("ensure bucket %q: %w", bucket, err)
}
