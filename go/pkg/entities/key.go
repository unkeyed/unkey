package entities

import "time"

// Key represents an API key in the system
type Key struct {
	// ID is the unique identifier for the key
	ID string

	// KeyringID represents the key authorization space this key belongs to
	KeyringID string

	// WorkspaceID is the ID of the workspace that owns this key
	WorkspaceID string

	// Hash is the secure hash of the key used for verification
	Hash string

	// Start is the prefix of the key shown to users for identification
	Start string

	// ForWorkspaceID is used only for internal keys to indicate which workspace the key is for
	// This is primarily used for managing the Unkey app itself and is not used for user keys
	ForWorkspaceID string

	// Name is an optional human-readable identifier for the key
	Name string

	// Meta contains arbitrary metadata associated with the key as key-value pairs
	Meta map[string]any

	// CreatedAt is the timestamp when the key was created
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the key was last modified
	UpdatedAt time.Time

	// DeletedAt indicates when the key was revoked
	// A zero time value means the key is not deleted.
	DeletedAt time.Time

	// Enabled indicates whether the key is currently active (true) or disabled (false)
	// Keys are enabled by default.
	Enabled bool

	// Environment is an optional flag used to segment keys (e.g., "test" vs "production")
	// This is a user-defined value with no system-level restrictions
	Environment string

	Expires time.Time

	Identity *Identity

	// All transient permissions, directly attached or via roles
	Permissions []string

	RemainingRequests *int64
}
