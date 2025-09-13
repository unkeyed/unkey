package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/unkeyed/unkey/go/apps/assetmanagerd/internal/config"
)

// Backend defines the interface for asset storage backends
type Backend interface {
	// Store stores an asset and returns its location
	Store(ctx context.Context, id string, reader io.Reader, size int64) (string, error)

	// Retrieve retrieves an asset by its location
	Retrieve(ctx context.Context, location string) (io.ReadCloser, error)

	// Delete deletes an asset by its location
	Delete(ctx context.Context, location string) error

	// Exists checks if an asset exists at the given location
	Exists(ctx context.Context, location string) (bool, error)

	// GetSize returns the size of an asset in bytes
	GetSize(ctx context.Context, location string) (int64, error)

	// GetChecksum returns the SHA256 checksum of an asset
	GetChecksum(ctx context.Context, location string) (string, error)

	// EnsureLocal ensures an asset is available locally and returns the local path
	// For local backend, this just returns the location
	// For remote backends, this downloads to cache if needed
	EnsureLocal(ctx context.Context, location string, cacheDir string) (string, error)

	// Type returns the backend type
	Type() string
}

// NewBackend creates a new storage backend based on configuration
func NewBackend(cfg *config.Config, logger *slog.Logger) (Backend, error) {
	switch cfg.StorageBackend {
	case "local":
		return NewLocalBackend(cfg.LocalStoragePath, logger)
	case "s3":
		return nil, fmt.Errorf("S3 backend not yet implemented")
	case "nfs":
		return nil, fmt.Errorf("NFS backend not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", cfg.StorageBackend)
	}
}
