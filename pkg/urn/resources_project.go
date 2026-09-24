package urn

import "fmt"

const projectPathFormat = "projects/%s"

var projectPattern = compileResourcePattern(projectPathFormat)

// Project builds project resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    ├── apps/{app_id}
//	    ├── identities/{identity_id}
//	    ├── keyspaces/{keyspace_id}
//	    ├── portals/{portal_id}
//	    ├── ratelimits/namespaces/{namespace_id}
//	    └── rbac
type Project struct {
	WorkspaceID string
	ProjectID   string
}

// String returns the complete URN for this project.
func (p Project) String() string {
	return V1{
		WorkspaceID: p.WorkspaceID,
		Resource:    fmt.Sprintf(projectPathFormat, p.ProjectID),
	}.String()
}

// ParseProject parses:
//
//	unkey:v1:ws_123:projects/proj_123
//
// into:
//
//	Project{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//	}
//
// Resource ID positions may contain "*".
func ParseProject(urn string) (Project, error) {
	matches := projectPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Project{}, fmt.Errorf("%w: resource does not match project", ErrInvalidResourceName)
	}
	return Project{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
	}, nil
}

// App returns builders for app resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── apps/{app_id}
func (p Project) App(appID string) App {
	return App{
		WorkspaceID: p.WorkspaceID,
		ProjectID:   p.ProjectID,
		AppID:       appID,
	}
}

// Identity returns an identity resource path.
//
// Subresource:
//
//	projects/{project_id}
//	└── identities/{identity_id}
func (p Project) Identity(identityID string) Identity {
	return Identity{
		WorkspaceID: p.WorkspaceID,
		ProjectID:   p.ProjectID,
		IdentityID:  identityID,
	}
}

// Keyspace returns builders for keyspace resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── keyspaces/{keyspace_id}
func (p Project) Keyspace(keyspaceID string) Keyspace {
	return Keyspace{
		WorkspaceID: p.WorkspaceID,
		ProjectID:   p.ProjectID,
		KeyspaceID:  keyspaceID,
	}
}

// Portal returns builders for portal resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── portals/{portal_id}
func (p Project) Portal(portalID string) Portal {
	return Portal{
		WorkspaceID: p.WorkspaceID,
		ProjectID:   p.ProjectID,
		PortalID:    portalID,
	}
}

// RatelimitNamespace returns builders for rate limit namespace resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── ratelimits/namespaces/{namespace_id}
func (p Project) RatelimitNamespace(namespaceID string) RatelimitNamespace {
	return RatelimitNamespace{
		WorkspaceID: p.WorkspaceID,
		ProjectID:   p.ProjectID,
		NamespaceID: namespaceID,
	}
}

// RBAC returns builders for RBAC resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── rbac
func (p Project) RBAC() RBAC {
	return RBAC{
		WorkspaceID: p.WorkspaceID,
		ProjectID:   p.ProjectID,
	}
}
