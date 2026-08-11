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

// Storage uploads objects to and reads them from an S3-compatible bucket,
// such as AWS S3 or MinIO.
type Storage struct {
	client *s3.Client
	bucket string
}

// NewStorage builds an S3 client for the given endpoint, ensures the bucket
// exists, and returns the storage adapter. useSSL only applies when the
// endpoint carries no scheme.
func NewStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithBaseEndpoint(endpointWithScheme(endpoint, useSSL)),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	if err := ensureBucket(context.Background(), client, bucket); err != nil {
		return nil, err
	}

	return &Storage{client: client, bucket: bucket}, nil
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

// PresignedGetURL returns a time-limited URL for downloading the object at
// key without credentials.
func (s *Storage) PresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := s3.NewPresignClient(s.client).PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
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
