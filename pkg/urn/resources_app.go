package urn

import "fmt"

// App builds app resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
type App struct {
	workspaceID string
	path        string
}

// String returns this app resource path.
//
// Subresource:
//
//	projects/{project_id}
//	└── apps/{app_id}
func (a App) String() string {
	return V1{WorkspaceID: a.workspaceID, Resource: a.path}.String()
}

// Environment returns builders for environment resource paths.
//
// Subresource:
//
//	apps/{app_id}
//	└── environments/{environment_id}
func (a App) Environment(environmentID string) Environment {
	return Environment{workspaceID: a.workspaceID, path: fmt.Sprintf("%s/environments/%s", a.path, environmentID)}
}

// Any returns a descendant pattern below this app.
func (a App) Any() V1 {
	return V1{
		WorkspaceID: a.workspaceID,
		Resource:    a.path + "/**",
	}
}
