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
	projectID   string
	appID       string
}

// String returns this app resource path.
func (a App) String() string {
	return V1{WorkspaceID: a.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s", a.projectID, a.appID)}.String()
}

// Environment returns builders for environment resource paths.
func (a App) Environment(environmentID string) Environment {
	return Environment{workspaceID: a.workspaceID, projectID: a.projectID, appID: a.appID, environmentID: environmentID}
}

// Any returns a descendant pattern below this app.
func (a App) Any() V1 {
	return V1{
		WorkspaceID: a.workspaceID,
		Resource:    fmt.Sprintf("projects/%s/apps/%s/**", a.projectID, a.appID),
	}
}
