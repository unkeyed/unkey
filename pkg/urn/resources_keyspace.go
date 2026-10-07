package urn

import "fmt"

const keyspacePathFormat = "projects/%s/keyspaces/%s"

var keyspacePattern = compileResourcePattern(keyspacePathFormat)

// Keyspace builds keyspace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── keyspaces/{keyspace_id}
//	        ├── logs
//	        └── keys/{key_id}
type Keyspace struct {
	WorkspaceID string
	ProjectID   string
	KeyspaceID  string
}

// String returns the complete URN for this keyspace.
//
// Subresource:
//
//	workspace
//	└── keyspaces/{keyspace_id}
func (k Keyspace) String() string {
	return V1{
		WorkspaceID: k.WorkspaceID,
		Resource:    fmt.Sprintf(keyspacePathFormat, k.ProjectID, k.KeyspaceID),
	}.String()
}

// ParseKeyspace parses:
//
//	unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123
//
// into:
//
//	Keyspace{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		KeyspaceID:  "ks_123",
//	}
//
// Resource ID positions may contain "*".
func ParseKeyspace(urn string) (Keyspace, error) {
	matches := keyspacePattern.FindStringSubmatch(urn)
	if matches == nil {
		return Keyspace{}, fmt.Errorf("%w: resource does not match keyspace", ErrInvalidResourceName)
	}
	return Keyspace{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		KeyspaceID:  matches[3],
	}, nil
}

// Logs returns the keyspace log resource path.
//
// Subresource:
//
//	keyspaces/{keyspace_id}
//	└── logs
func (k Keyspace) Logs() KeyspaceLogs {
	return KeyspaceLogs{
		WorkspaceID: k.WorkspaceID,
		ProjectID:   k.ProjectID,
		KeyspaceID:  k.KeyspaceID,
	}
}

// Key returns a key resource path.
//
// Subresource:
//
//	keyspaces/{keyspace_id}
//	└── keys/{key_id}
func (k Keyspace) Key(keyID string) Key {
	return Key{
		WorkspaceID: k.WorkspaceID,
		ProjectID:   k.ProjectID,
		KeyspaceID:  k.KeyspaceID,
		KeyID:       keyID,
	}
}
