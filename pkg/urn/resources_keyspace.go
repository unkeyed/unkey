package urn

import "fmt"

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

// Logs returns the keyspace log resource path.
//
// Subresource:
//
//	keyspaces/{keyspace_id}
//	└── logs
func (k Keyspace) Logs() KeyspaceLogs {
	return KeyspaceLogs{workspaceID: k.workspaceID, path: k.path + "/logs"}
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

// KeyspaceLogs builds keyspace log resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── keyspaces/{keyspace_id}
//	        └── logs
type KeyspaceLogs struct {
	workspaceID string
	path        string
}

// String returns this keyspace log resource path.
func (k KeyspaceLogs) String() string {
	return V1{WorkspaceID: k.workspaceID, Resource: k.path}.String()
}
