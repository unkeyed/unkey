package s3

import (
	"context"
	"time"
)

// Storage defines the interface for generating pre-signed URLs for object storage.
// Implementations must be safe for concurrent use.
type Storage interface {
	// GenerateDownloadURL returns a pre-signed URL that allows downloading the
	// object at key. The URL expires after expiresIn.
	GenerateDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)

	// GenerateUploadURL returns a pre-signed URL that allows uploading an object
	// to key using HTTP PUT. The URL expires after expiresIn.
	GenerateUploadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}

var _ Storage = (*S3)(nil)
