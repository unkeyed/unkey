package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// LocalBackend implements Backend for local filesystem storage
type LocalBackend struct {
	basePath string
	logger   *slog.Logger
}

// NewLocalBackend creates a new local storage backend
func NewLocalBackend(basePath string, logger *slog.Logger) (*LocalBackend, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalBackend{
		basePath: basePath,
		logger:   logger.With("backend", "local"),
	}, nil
}

// Store stores an asset locally
func (b *LocalBackend) Store(ctx context.Context, id string, reader io.Reader, size int64) (string, error) {
	// Create subdirectory based on first 2 chars of ID for better filesystem performance
	// AIDEV-NOTE: Sharding prevents too many files in a single directory
	subdir := id[:2]
	dirPath := filepath.Join(b.basePath, subdir)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(dirPath, id)

	// Create temporary file first
	tmpPath := filePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpPath) // Clean up on any error

	// Copy data
	written, err := io.Copy(tmpFile, reader)
	if err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write asset: %w", err)
	}
	tmpFile.Close()

	// Verify size if provided
	if size > 0 && written != size {
		return "", fmt.Errorf("size mismatch: expected %d, got %d", size, written)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, filePath); err != nil {
		return "", fmt.Errorf("failed to finalize asset: %w", err)
	}

	b.logger.LogAttrs(ctx, slog.LevelInfo, "stored asset",
		slog.String("id", id),
		slog.String("path", filePath),
		slog.Int64("size", written),
	)

	// Return relative path from base
	return filepath.Join(subdir, id), nil
}

// Retrieve retrieves an asset from local storage
func (b *LocalBackend) Retrieve(ctx context.Context, location string) (io.ReadCloser, error) {
	fullPath := filepath.Join(b.basePath, location)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("asset not found: %s", location)
		}
		return nil, fmt.Errorf("failed to open asset: %w", err)
	}

	return file, nil
}

// Delete deletes an asset from local storage
func (b *LocalBackend) Delete(ctx context.Context, location string) error {
	fullPath := filepath.Join(b.basePath, location)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	b.logger.LogAttrs(ctx, slog.LevelInfo, "deleted asset",
		slog.String("location", location),
		slog.String("path", fullPath),
	)

	return nil
}

// Exists checks if an asset exists
func (b *LocalBackend) Exists(ctx context.Context, location string) (bool, error) {
	fullPath := filepath.Join(b.basePath, location)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat asset: %w", err)
	}

	return true, nil
}

// GetSize returns the size of an asset
func (b *LocalBackend) GetSize(ctx context.Context, location string) (int64, error) {
	fullPath := filepath.Join(b.basePath, location)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("asset not found: %s", location)
		}
		return 0, fmt.Errorf("failed to stat asset: %w", err)
	}

	return info.Size(), nil
}

// GetChecksum calculates and returns the SHA256 checksum
func (b *LocalBackend) GetChecksum(ctx context.Context, location string) (string, error) {
	fullPath := filepath.Join(b.basePath, location)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("asset not found: %s", location)
		}
		return "", fmt.Errorf("failed to open asset: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// EnsureLocal returns the full path for local assets
func (b *LocalBackend) EnsureLocal(ctx context.Context, location string, cacheDir string) (string, error) {
	fullPath := filepath.Join(b.basePath, location)

	// Verify it exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("asset not found: %s", location)
		}
		return "", fmt.Errorf("failed to stat asset: %w", err)
	}

	return fullPath, nil
}

// Type returns the backend type
func (b *LocalBackend) Type() string {
	return "local"
}
