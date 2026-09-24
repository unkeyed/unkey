package urn

import "fmt"

const deploymentLogsPathFormat = "projects/%s/apps/%s/environments/%s/deployments/%s/logs"

var deploymentLogsPattern = compileResourcePattern(deploymentLogsPathFormat)

// DeploymentLogs builds deployment log resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── deployments/{deployment_id}
//	                └── logs
type DeploymentLogs struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
	DeploymentID  string
}

// String returns the complete URN for these deployment logs.
func (d DeploymentLogs) String() string {
	return V1{
		WorkspaceID: d.WorkspaceID,
		Resource:    fmt.Sprintf(deploymentLogsPathFormat, d.ProjectID, d.AppID, d.EnvironmentID, d.DeploymentID),
	}.String()
}

// ParseDeploymentLogs parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123/logs
//
// into:
//
//	DeploymentLogs{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//		DeploymentID:  "dep_123",
//	}
//
// Resource ID positions may contain "*".
func ParseDeploymentLogs(urn string) (DeploymentLogs, error) {
	matches := deploymentLogsPattern.FindStringSubmatch(urn)
	if matches == nil {
		return DeploymentLogs{}, fmt.Errorf("%w: resource does not match deployment logs", ErrInvalidResourceName)
	}
	return DeploymentLogs{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
		DeploymentID:  matches[5],
	}, nil
}
