package urn

import "fmt"

// workspace builds resource paths inside one workspace.
//
// Hierarchy:
//
//	workspace
//	├── team
//	├── billing
//	└── projects/{project_id}
//
// Children with their own descendants return another typed builder. Leaf
// resources return V1 directly.
type workspace struct {
	workspaceID string

	// Team builds team resource paths in this workspace.
	Team team
}

// Billing returns builders for billing resource paths.
//
// Subresource:
//
//	workspace
//	└── billing
func (w workspace) Billing() billing {
	return billing{workspaceID: w.workspaceID, path: "billing"}
}

// Project returns builders for project resource paths.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
func (w workspace) Project(projectID string) Project {
	path := fmt.Sprintf("projects/%s", projectID)
	return Project{
		workspaceID: w.workspaceID,
		path:        path,
		RBAC:        rbac{workspaceID: w.workspaceID, path: path + "/rbac"},
	}
}
