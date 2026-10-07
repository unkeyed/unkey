package urn

import "fmt"

const deploymentBuildLogsPathFormat = "projects/%s/apps/%s/environments/%s/deployments/%s/buildLogs"

var deploymentBuildLogsPattern = compileResourcePattern(deploymentBuildLogsPathFormat)

// DeploymentBuildLogs builds deployment build log resource paths
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── deployments/{deployment_id}
//	                └── buildLogs
type DeploymentBuildLogs struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
	DeploymentID  string
}

// String returns the complete URN for these deployment build logs
func (d DeploymentBuildLogs) String() string {
	return V1{
		WorkspaceID: d.WorkspaceID,
		Resource:    fmt.Sprintf(deploymentBuildLogsPathFormat, d.ProjectID, d.AppID, d.EnvironmentID, d.DeploymentID),
	}.String()
}

// ParseDeploymentBuildLogs parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123/buildLogs
//
// into:
//
//	DeploymentBuildLogs{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//		DeploymentID:  "dep_123",
//	}
//
// Resource ID positions may contain "*"
func ParseDeploymentBuildLogs(urn string) (DeploymentBuildLogs, error) {
	matches := deploymentBuildLogsPattern.FindStringSubmatch(urn)
	if matches == nil {
		return DeploymentBuildLogs{}, fmt.Errorf("%w: resource does not match deployment build logs", ErrInvalidResourceName)
	}
	return DeploymentBuildLogs{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
		DeploymentID:  matches[5],
	}, nil
}
