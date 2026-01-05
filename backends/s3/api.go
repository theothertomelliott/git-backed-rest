package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"runtime/trace"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

var _ gitbackedrest.APIBackend = (*Backend)(nil)

// Config holds configuration for S3-compatible storage
type Config struct {
	// Endpoint is the S3-compatible endpoint URL (e.g., https://<account-id>.r2.cloudflarestorage.com)
	Endpoint string
	// AccessKeyID is the access key for authentication
	AccessKeyID string
	// SecretAccessKey is the secret key for authentication
	SecretAccessKey string
	// Bucket is the bucket name to use
	Bucket string
	// Prefix is an optional path prefix within the bucket (e.g., "test/store1")
	// This allows multiple stores to coexist in the same bucket
	Prefix string
	// Region is the AWS region (can be "auto" for R2)
	Region string
}

// Backend implements APIBackend using S3-compatible storage
type Backend struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewBackend creates a new S3-compatible backend
func NewBackend(cfg Config) (*Backend, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("access key ID is required")
	}
	if cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("secret access key is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}

	client := s3.NewFromConfig(aws.Config{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		BaseEndpoint: aws.String(cfg.Endpoint),
	})

	return &Backend{
		client: client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

// buildKey constructs the full S3 key from the path and prefix
func (b *Backend) buildKey(p string) string {
	if b.prefix == "" {
		return p
	}
	return path.Join(b.prefix, p)
}

// GET implements gitbackedrest.APIBackend.
func (b *Backend) GET(ctx context.Context, p string) (context.Context, []byte, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "GET").End()

	key := b.buildKey(p)

	reg := trace.StartRegion(ctx, "GetObject")
	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	reg.End()
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return ctx, nil, gitbackedrest.ErrNotFound
		}
		return ctx, nil, gitbackedrest.ErrInternalServerError
	}
	defer result.Body.Close()

	reg = trace.StartRegion(ctx, "ReadBody")
	body, err := io.ReadAll(result.Body)
	reg.End()
	if err != nil {
		return ctx, nil, gitbackedrest.ErrInternalServerError
	}

	return ctx, body, nil
}

// POST implements gitbackedrest.APIBackend.
func (b *Backend) POST(ctx context.Context, p string, body []byte) (context.Context, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "POST").End()

	key := b.buildKey(p)

	// Check if object already exists
	reg := trace.StartRegion(ctx, "HeadObject")
	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	reg.End()
	if err == nil {
		return ctx, gitbackedrest.ErrConflict
	}
	var notFound *types.NotFound
	if !errors.As(err, &notFound) {
		return ctx, gitbackedrest.ErrInternalServerError
	}

	// Object doesn't exist, create it
	reg = trace.StartRegion(ctx, "PutObject")
	_, err = b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	reg.End()
	if err != nil {
		return ctx, gitbackedrest.ErrInternalServerError
	}

	return ctx, nil
}

// PUT implements gitbackedrest.APIBackend.
func (b *Backend) PUT(ctx context.Context, p string, body []byte) (context.Context, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "PUT").End()

	key := b.buildKey(p)

	// Check if object exists
	reg := trace.StartRegion(ctx, "HeadObject")
	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	reg.End()
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return ctx, gitbackedrest.ErrNotFound
		}
		return ctx, gitbackedrest.ErrInternalServerError
	}

	// Object exists, update it
	reg = trace.StartRegion(ctx, "PutObject")
	_, err = b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	reg.End()
	if err != nil {
		return ctx, gitbackedrest.ErrInternalServerError
	}

	return ctx, nil
}

// DELETE implements gitbackedrest.APIBackend.
func (b *Backend) DELETE(ctx context.Context, p string) (context.Context, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "DELETE").End()

	key := b.buildKey(p)

	// Check if object exists
	reg := trace.StartRegion(ctx, "HeadObject")
	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	reg.End()
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return ctx, gitbackedrest.ErrNotFound
		}
		return ctx, gitbackedrest.ErrInternalServerError
	}

	// Object exists, delete it
	reg = trace.StartRegion(ctx, "DeleteObject")
	_, err = b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	reg.End()
	if err != nil {
		return ctx, gitbackedrest.ErrInternalServerError
	}

	return ctx, nil
}

// Close cleans up resources (currently no-op but allows for future cleanup)
func (b *Backend) Close() error {
	return nil
}

// CleanupPrefix deletes all objects under the backend's prefix
// Useful for test cleanup
func (b *Backend) CleanupPrefix(ctx context.Context) error {
	defer trace.StartRegion(ctx, "CleanupPrefix").End()

	if b.prefix == "" {
		return fmt.Errorf("refusing to cleanup empty prefix (would delete entire bucket)")
	}

	// List all objects under the prefix
	reg := trace.StartRegion(ctx, "ListObjectsV2")
	listOutput, err := b.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(b.prefix),
	})
	reg.End()
	if err != nil {
		return fmt.Errorf("listing objects: %w", err)
	}

	// Delete each object
	reg = trace.StartRegion(ctx, "DeleteObjects")
	for _, obj := range listOutput.Contents {
		_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(b.bucket),
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("deleting object %s: %w", *obj.Key, err)
		}
	}
	reg.End()

	return nil
}
