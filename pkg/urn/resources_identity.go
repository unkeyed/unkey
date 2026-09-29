package urn

import "fmt"

const identityPathFormat = "projects/%s/identities/%s"

var identityPattern = compileResourcePattern(identityPathFormat)

// Identity builds identity resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── identities/{identity_id}
type Identity struct {
	WorkspaceID string
	ProjectID   string
	IdentityID  string
}

// String returns the complete URN for this identity.
//
// Subresource:
//
//	projects/{project_id}
//	└── identities/{identity_id}
func (i Identity) String() string {
	return V1{
		WorkspaceID: i.WorkspaceID,
		Resource:    fmt.Sprintf(identityPathFormat, i.ProjectID, i.IdentityID),
	}.String()
}

// ParseIdentity parses:
//
//	unkey:v1:ws_123:projects/proj_123/identities/identity_123
//
// into:
//
//	Identity{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		IdentityID:  "identity_123",
//	}
//
// Resource ID positions may contain "*".
func ParseIdentity(urn string) (Identity, error) {
	matches := identityPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Identity{}, fmt.Errorf("%w: resource does not match identity", ErrInvalidResourceName)
	}
	return Identity{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		IdentityID:  matches[3],
	}, nil
}
