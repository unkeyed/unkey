package urn

import "fmt"

const environmentVariablePathFormat = "projects/%s/apps/%s/environments/%s/variables/%s"

var environmentVariablePattern = compileResourcePattern(environmentVariablePathFormat)

// EnvironmentVariable builds environment variable resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── variables/{variable_id}
type EnvironmentVariable struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
	VariableID    string
}

// String returns the complete URN for this environment variable.
func (e EnvironmentVariable) String() string {
	return V1{
		WorkspaceID: e.WorkspaceID,
		Resource:    fmt.Sprintf(environmentVariablePathFormat, e.ProjectID, e.AppID, e.EnvironmentID, e.VariableID),
	}.String()
}

// ParseEnvironmentVariable parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123
//
// into:
//
//	EnvironmentVariable{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//		VariableID:    "var_123",
//	}
//
// Resource ID positions may contain "*".
func ParseEnvironmentVariable(urn string) (EnvironmentVariable, error) {
	matches := environmentVariablePattern.FindStringSubmatch(urn)
	if matches == nil {
		return EnvironmentVariable{}, fmt.Errorf("%w: resource does not match environment variable", ErrInvalidResourceName)
	}
	return EnvironmentVariable{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
		VariableID:    matches[5],
	}, nil
}
