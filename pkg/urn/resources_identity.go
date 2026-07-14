package urn

import "fmt"

// Identity is an identity resource path beneath a project.
type Identity struct {
	workspaceID string
	path        string
}

// Identity returns an identity resource path beneath this project.
func (p Project) Identity(identityID string) Identity {
	return Identity{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/identities/%s", p.path, identityID)}
}

// String returns this identity resource path.
func (i Identity) String() string {
	return V1{WorkspaceID: i.workspaceID, Resource: i.path}.String()
}
