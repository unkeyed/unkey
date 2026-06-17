package urn

import "fmt"

// Keyspace builds keyspace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── keyspaces/{keyspace_id}
//
// A keyspace can also produce a descendant pattern for grants covering every
// key and future keyspace child.
type Keyspace struct {
	workspaceID string
	path        string
}

// String returns this keyspace resource path.
//
// Subresource:
//
//	workspace
//	└── keyspaces/{keyspace_id}
func (k Keyspace) String() string {
	return V1{WorkspaceID: k.workspaceID, Resource: k.path}.String()
}

// Key is a key resource path.
type Key struct {
	workspaceID string
	path        string
}

// String returns this key resource path.
func (k Key) String() string {
	return V1{WorkspaceID: k.workspaceID, Resource: k.path}.String()
}

// V1 returns this key as a parsed v1 resource name.
func (k Key) V1() V1 {
	return V1{WorkspaceID: k.workspaceID, Resource: k.path}
}

// Key returns a key resource path.
//
// Subresource:
//
//	keyspaces/{keyspace_id}
//	└── keys/{key_id}
func (k Keyspace) Key(keyID string) Key {
	return Key{workspaceID: k.workspaceID, path: fmt.Sprintf("%s/keys/%s", k.path, keyID)}
}

// Any returns a descendant pattern below this keyspace.
func (k Keyspace) Any() V1 {
	return V1{
		WorkspaceID: k.workspaceID,
		Resource:    k.path + "/**",
	}
}
