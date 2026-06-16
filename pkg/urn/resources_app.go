package urn

import "fmt"

// app builds app resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
type app struct {
	workspaceID string
	path        string
}

// Environment returns builders for environment resource paths.
//
// Subresource:
//
//	apps/{app_id}
//	└── environments/{environment_id}
func (a app) Environment(environmentID string) environment {
	return environment{workspaceID: a.workspaceID, path: fmt.Sprintf("%s/environments/%s", a.path, environmentID)}
}

// Any returns a descendant pattern below this app.
func (a app) Any() V1 {
	return V1{
		WorkspaceID: a.workspaceID,
		Resource:    a.path + "/**",
	}
}
