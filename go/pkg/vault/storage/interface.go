package storage

import (
	"context"
	"errors"
	"time"
)

var ErrObjectNotFound = errors.New("object not found")

type GetObjectOptions struct {
	IfUnModifiedSince time.Time
}

type Storage interface {
	// PutObject stores the object data for the given key
	PutObject(ctx context.Context, key string, object []byte) error

	// GetObject returns the object data for the given key
	GetObject(ctx context.Context, key string) ([]byte, bool, error)

	// ListObjectKeys returns a list of object keys that match the given prefix
	ListObjectKeys(ctx context.Context, prefix string) ([]string, error)

	// Key returns the object key for the given shard and version
	Key(shard string, dekID string) string

	// Latest returns the object key for the latest version of the given workspace
	Latest(shard string) string
	// GetPresignedURL generates a presigned URL for downloading an object
	GetPresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
	// PutPresignedURL generates a presigned URL for uploading an object
	PutPresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}
