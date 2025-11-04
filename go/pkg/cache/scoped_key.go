package cache

import (
	"fmt"
	"strings"
)

// ScopedKey represents a cache key that is scoped to a specific workspace.
//
// This type is designed for caching data where keys are only unique within
// a workspace context, rather than being globally unique. For example, a user
// might create a ratelimit namespace called "api-calls" which is unique within
// their workspace but could exist in multiple workspaces.
//
// The ScopedKey ensures cache isolation between workspaces by combining the
// workspace ID with the resource key, preventing cache collisions and data
// leakage between different workspaces.
//
// # Usage
//
// Use ScopedKey when caching data that is workspace-specific:
//
//	// Cache a ratelimit namespace by name
//	key := cache.ScopedKey{
//		WorkspaceID: "ws_123",
//		Key:         "api-calls",
//	}
//
//	// Cache by ID (still workspace-scoped for consistency)
//	key := cache.ScopedKey{
//		WorkspaceID: "ws_123",
//		Key:         "ns_456",
//	}
//
//	// Cache any workspace-scoped resource
//	key := cache.ScopedKey{
//		WorkspaceID: "ws_123",
//		Key:         "some-resource-identifier",
//	}
//
// # Design Rationale
//
// We chose this approach over concatenating strings because it provides type
// safety and makes the workspace scoping explicit in the API. It also allows
// for future extension if additional scoping dimensions are needed.
//
// The generic Key field can hold any string identifier (names, IDs, slugs, etc.)
// while maintaining consistent workspace isolation across all cache usage patterns.
type ScopedKey struct {
	// WorkspaceID is the unique identifier for the workspace that owns this resource.
	// This ensures that cache keys are isolated between different workspaces,
	// preventing accidental data leakage or cache collisions.
	WorkspaceID string

	// Key is the identifier for the resource within the workspace.
	// This can be a user-provided name, system-generated ID, slug, or any other
	// string identifier that uniquely identifies the resource within the workspace.
	//
	// The key is only guaranteed to be unique within the workspace context.
	// Different workspaces may have resources with the same key value.
	Key string
}

func (k ScopedKey) String() string {
	return k.WorkspaceID + ":" + k.Key
}

// ParseScopedKey parses a string in the format "workspace_id:key" into a ScopedKey.
// Returns an error if the string is not in the expected format.
func ParseScopedKey(s string) (ScopedKey, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return ScopedKey{}, fmt.Errorf("invalid scoped key format: expected 'workspace_id:key', got %q", s)
	}
	return ScopedKey{
		WorkspaceID: parts[0],
		Key:         parts[1],
	}, nil
}

var ScopedKeyToString = func(k ScopedKey) string { return k.String() }
var ScopedKeyFromString = func(s string) (ScopedKey, error) { return ParseScopedKey(s) }
