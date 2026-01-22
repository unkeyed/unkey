package s3

import (
	"context"
	"time"
)

// Storage defines the interface for object storage operations.
type Storage interface {
	GenerateDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
	GenerateUploadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}

var _ Storage = (*S3)(nil)
