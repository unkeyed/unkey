package urn

import "fmt"

// project builds project resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//
// Projects are owned by a workspace.
type project struct {
	workspaceID string
	path        string
}

// App returns builders for app resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── apps/{app_id}
func (p project) App(appID string) app {
	return app{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/apps/%s", p.path, appID)}
}

// Any returns a descendant pattern below this project.
func (p project) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    p.path + "/**",
	}
}
