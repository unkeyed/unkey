package urn

import "fmt"

// Keyspace builds keyspace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── keyspaces/{keyspace_id}
//
// A keyspace can also produce a descendant pattern for grants covering every
// key and future keyspace child.
type Keyspace struct {
	workspaceID string
	projectID   string
	keyspaceID  string
}

// String returns this keyspace resource path.
func (k Keyspace) String() string {
	return V1{WorkspaceID: k.workspaceID, Resource: fmt.Sprintf("projects/%s/keyspaces/%s", k.projectID, k.keyspaceID)}.String()
}

// Key is a key resource path.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── keyspaces/{keyspace_id}
//	        └── keys/{key_id}
type Key struct {
	workspaceID string
	projectID   string
	keyspaceID  string
	keyID       string
}

// String returns this key resource path.
func (k Key) String() string {
	return k.V1().String()
}

// V1 returns this key as a parsed v1 resource name.
func (k Key) V1() V1 {
	return V1{WorkspaceID: k.workspaceID, Resource: fmt.Sprintf("projects/%s/keyspaces/%s/keys/%s", k.projectID, k.keyspaceID, k.keyID)}
}

// Key returns a key resource path.
func (k Keyspace) Key(keyID string) Key {
	return Key{workspaceID: k.workspaceID, projectID: k.projectID, keyspaceID: k.keyspaceID, keyID: keyID}
}

// Any returns a descendant pattern below this keyspace.
func (k Keyspace) Any() V1 {
	return V1{
		WorkspaceID: k.workspaceID,
		Resource:    fmt.Sprintf("projects/%s/keyspaces/%s/**", k.projectID, k.keyspaceID),
	}
}
