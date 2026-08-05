package urn

// Identity builds identity resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── identities/{identity_id}
type Identity struct {
	workspaceID string
	path        string
}

// String returns this identity resource path.
func (i Identity) String() string {
	return V1{WorkspaceID: i.workspaceID, Resource: i.path}.String()
}
