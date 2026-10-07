package urn

import "fmt"

const appPathFormat = "projects/%s/apps/%s"

var appPattern = compileResourcePattern(appPathFormat)

// App builds app resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
type App struct {
	WorkspaceID string
	ProjectID   string
	AppID       string
}

// String returns the complete URN for this app.
//
// Subresource:
//
//	projects/{project_id}
//	└── apps/{app_id}
func (a App) String() string {
	return V1{
		WorkspaceID: a.WorkspaceID,
		Resource:    fmt.Sprintf(appPathFormat, a.ProjectID, a.AppID),
	}.String()
}

// ParseApp parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123
//
// into:
//
//	App{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		AppID:       "app_123",
//	}
//
// Resource ID positions may contain "*".
func ParseApp(urn string) (App, error) {
	matches := appPattern.FindStringSubmatch(urn)
	if matches == nil {
		return App{}, fmt.Errorf("%w: resource does not match app", ErrInvalidResourceName)
	}
	return App{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		AppID:       matches[3],
	}, nil
}

// Environment returns builders for environment resource paths.
//
// Subresource:
//
//	apps/{app_id}
//	└── environments/{environment_id}
func (a App) Environment(environmentID string) Environment {
	return Environment{
		WorkspaceID:   a.WorkspaceID,
		ProjectID:     a.ProjectID,
		AppID:         a.AppID,
		EnvironmentID: environmentID,
	}
}
