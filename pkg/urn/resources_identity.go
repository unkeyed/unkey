package urn

import "fmt"

// Identity builds identity resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── identities/{identity_id}
type Identity struct {
	workspaceID string
	projectID   string
	identityID  string
}

// String returns this identity resource path.
func (i Identity) String() string {
	return V1{WorkspaceID: i.workspaceID, Resource: fmt.Sprintf("projects/%s/identities/%s", i.projectID, i.identityID)}.String()
}
