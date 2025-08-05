package kv

import (
	"context"
	"time"
)

// Store defines the interface for a key-value store with TTL support
type Store interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, workspaceID string, value []byte, ttl *time.Duration) error
	Delete(ctx context.Context, key string) error
	ListByWorkspace(ctx context.Context, workspaceID string, cursor int64, limit int) ([]KvEntry, error)
}

// KvEntry represents a key-value entry in the store
type KvEntry struct {
	ID          int64
	Key         string
	WorkspaceID string
	Value       []byte
	TTL         *int64
	CreatedAt   int64
}
