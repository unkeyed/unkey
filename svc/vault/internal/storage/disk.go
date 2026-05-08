package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/unkeyed/unkey/pkg/logger"
)

// disk persists objects under a single root directory. Object keys are
// interpreted as forward-slash relative paths beneath the root, so the on-disk
// layout mirrors the S3 key layout. Intended for local development as a
// drop-in replacement for the S3 backend (no minio container needed).
type disk struct {
	mu   sync.RWMutex
	root string
}

func NewDisk(root string) (Storage, error) {
	if root == "" {
		return nil, errors.New("disk storage: root path must not be empty")
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("disk storage: resolve root: %w", err)
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("disk storage: create root: %w", err)
	}

	logger.Info("using disk storage", "root", abs)
	return &disk{root: abs, mu: sync.RWMutex{}}, nil
}

func (d *disk) Key(workspaceId string, dekID string) string {
	return fmt.Sprintf("%s/%s", workspaceId, dekID)
}

func (d *disk) Latest(workspaceId string) string {
	return d.Key(workspaceId, "LATEST")
}

// resolve maps an object key to an absolute filesystem path under the root.
// It rejects keys that would escape the root via "..", absolute paths, or
// embedded null bytes.
func (d *disk) resolve(key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	if strings.ContainsRune(key, 0) {
		return "", errors.New("key contains null byte")
	}

	rel := filepath.FromSlash(key)
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("key must be relative: %q", key)
	}

	full := filepath.Join(d.root, rel)
	relCheck, err := filepath.Rel(d.root, full)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		return "", fmt.Errorf("key escapes storage root: %q", key)
	}
	return full, nil
}

func (d *disk) PutObject(ctx context.Context, key string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := d.resolve(key)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".vault-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	cleanup = false
	return nil
}

func (d *disk) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	path, err := d.resolve(key)
	if err != nil {
		return nil, false, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read object: %w", err)
	}
	return b, true, nil
}

func (d *disk) ListObjectKeys(ctx context.Context, prefix string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Match memory backend semantics: an empty prefix returns no keys.
	if prefix == "" {
		return []string{}, nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	keys := []string{}
	walkErr := filepath.WalkDir(d.root, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == d.root {
				return filepath.SkipAll
			}
			return err
		}
		if de.IsDir() {
			return nil
		}
		// Skip in-flight temp files from interrupted PutObject calls.
		if strings.HasPrefix(de.Name(), ".vault-") && strings.HasSuffix(de.Name(), ".tmp") {
			return nil
		}
		rel, err := filepath.Rel(d.root, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)
		if !strings.HasPrefix(key, prefix) {
			return nil
		}
		keys = append(keys, key)
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk: %w", walkErr)
	}
	return keys, nil
}
