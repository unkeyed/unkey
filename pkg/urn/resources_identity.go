package urn

// Identity is an identity resource path.
type Identity struct {
	workspaceID string
	path        string
}

// String returns this identity resource path.
func (i Identity) String() string {
	return V1{WorkspaceID: i.workspaceID, Resource: i.path}.String()
}
