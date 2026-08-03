// Package objectstore is a provider-neutral adapter for S3-compatible object
// storage. Swapping providers (Cloudflare R2 in production, a local Garage
// container in dev/CI) changes only configuration, so it lives in core. The
// application issues presigned URLs and never hands bucket keys to clients.
package objectstore

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config holds the connection settings for one bucket. When any required field
// is empty the store is considered unconfigured and New returns (nil, nil).
type Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// Configured reports whether all required settings are present.
func (c Config) Configured() bool {
	return c.Endpoint != "" && c.Bucket != "" && c.AccessKeyID != "" && c.SecretAccessKey != ""
}

// Store is an S3-compatible object store scoped to one bucket.
type Store struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// New constructs a store from config, or returns (nil, nil) when unconfigured so
// callers can gate features that require object storage. Path-style addressing
// and an endpoint override make it work against any S3-compatible service; the
// checksum calculation is limited to when required so S3-compatible servers that
// reject aws-chunked checksum trailers still accept uploads.
func New(cfg Config) (*Store, error) {
	if !cfg.Configured() {
		return nil, nil
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	client := s3.New(s3.Options{
		Region:                     region,
		BaseEndpoint:               aws.String(cfg.Endpoint),
		Credentials:                credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle:               true,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
	})
	return &Store{client: client, presign: s3.NewPresignClient(client), bucket: cfg.Bucket}, nil
}

// Put uploads an object. Pass a seekable body (for example *bytes.Reader) so the
// client can set Content-Length rather than stream chunked.
func (s *Store) Put(ctx context.Context, key, contentType string, body io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

// Delete removes an object. Deleting a missing key is not an error.
func (s *Store) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}); err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

// PresignGet returns a time-limited URL that grants read access to one object
// without exposing credentials or the bucket key to the client.
func (s *Store) PresignGet(ctx context.Context, key string, expires time.Duration) (string, error) {
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("presign get object %q: %w", key, err)
	}
	return request.URL, nil
}
