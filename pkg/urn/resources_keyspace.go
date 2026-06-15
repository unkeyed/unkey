package urn

import "fmt"

// keyspace builds keyspace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── keyspaces/{keyspace_id}
//
// A keyspace can also produce a descendant pattern for grants covering every
// key and future keyspace child.
type keyspace struct {
	workspaceID string
	path        string
}

// Key returns a key resource path.
//
// Subresource:
//
//	keyspaces/{keyspace_id}
//	└── keys/{key_id}
func (k keyspace) Key(keyID string) V1 {
	return V1{
		WorkspaceID: k.workspaceID,
		Resource:    fmt.Sprintf("%s/keys/%s", k.path, keyID),
	}
}

// Any returns a descendant pattern below this keyspace.
func (k keyspace) Any() V1 {
	return V1{
		WorkspaceID: k.workspaceID,
		Resource:    k.path + "/**",
	}
}
