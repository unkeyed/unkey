package urn

import "fmt"

const deploymentPathFormat = "projects/%s/apps/%s/environments/%s/deployments/%s"

var deploymentPattern = compileResourcePattern(deploymentPathFormat)

// Deployment builds deployment resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── deployments/{deployment_id}
//	                └── logs
type Deployment struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
	DeploymentID  string
}

// String returns the complete URN for this deployment.
//
// Subresource:
//
//	environments/{environment_id}
//	└── deployments/{deployment_id}
func (d Deployment) String() string {
	return V1{
		WorkspaceID: d.WorkspaceID,
		Resource:    fmt.Sprintf(deploymentPathFormat, d.ProjectID, d.AppID, d.EnvironmentID, d.DeploymentID),
	}.String()
}

// ParseDeployment parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123
//
// into:
//
//	Deployment{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//		DeploymentID:  "dep_123",
//	}
//
// Resource ID positions may contain "*".
func ParseDeployment(urn string) (Deployment, error) {
	matches := deploymentPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Deployment{}, fmt.Errorf("%w: resource does not match deployment", ErrInvalidResourceName)
	}
	return Deployment{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
		DeploymentID:  matches[5],
	}, nil
}

// Logs returns the deployment log resource path.
//
// Subresource:
//
//	deployments/{deployment_id}
//	└── logs
func (d Deployment) Logs() DeploymentLogs {
	return DeploymentLogs{
		WorkspaceID:   d.WorkspaceID,
		ProjectID:     d.ProjectID,
		AppID:         d.AppID,
		EnvironmentID: d.EnvironmentID,
		DeploymentID:  d.DeploymentID,
	}
}
