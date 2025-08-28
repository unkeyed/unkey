// Package kv provides a key-value store abstraction with TTL support and workspace isolation.
//
// The package defines a Store interface that can be implemented by different backends.
// Currently, a MySQL-based implementation is provided in the stores/mysql subpackage.
//
// Key features:
//   - Automatic TTL expiration on read operations
//   - Workspace-based isolation
//   - Cursor-based pagination for listing operations
//   - Primary/read-replica database connection support
//   - Simple key-value model optimized for performance
//
// Example usage:
//
//	import (
//		"github.com/unkeyed/unkey/go/pkg/kv"
//		"github.com/unkeyed/unkey/go/pkg/kv/stores/mysql"
//	)
//
//	store, err := mysql.NewStore(mysql.Config{
//		PrimaryDSN: "user:pass@tcp(localhost:3306)/db?parseTime=true",
//		Logger:     logger,
//	})
//	if err != nil {
//		// handle error
//	}
//
//	// Set a key with TTL
//	ttl := 5 * time.Minute
//	err = store.Set(ctx, "user:123", "workspace1", []byte("data"), &ttl)
//
//	// Get a key
//	data, found, err := store.Get(ctx, "user:123")
//
//	// List keys by workspace with cursor pagination
//	entries, err := store.ListByWorkspace(ctx, "workspace1", 0, 10)
package kv
