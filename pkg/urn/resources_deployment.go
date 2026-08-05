package urn

import "fmt"

// Deployment builds deployment resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── deployments/{deployment_id}
type Deployment struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	deploymentID  string
}

// Instance is a deployment instance resource path.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── deployments/{deployment_id}
//	                └── instances/{instance_id}
type Instance struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	deploymentID  string
	instanceID    string
}

// String returns this deployment instance resource path.
func (i Instance) String() string {
	return V1{WorkspaceID: i.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s/deployments/%s/instances/%s", i.projectID, i.appID, i.environmentID, i.deploymentID, i.instanceID)}.String()
}

// String returns this deployment resource path.
func (d Deployment) String() string {
	return V1{WorkspaceID: d.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s/deployments/%s", d.projectID, d.appID, d.environmentID, d.deploymentID)}.String()
}

// Instance returns a deployment instance resource path.
func (d Deployment) Instance(instanceID string) Instance {
	return Instance{workspaceID: d.workspaceID, projectID: d.projectID, appID: d.appID, environmentID: d.environmentID, deploymentID: d.deploymentID, instanceID: instanceID}
}

// Any returns a descendant pattern below this deployment.
func (d Deployment) Any() V1 {
	return V1{
		WorkspaceID: d.workspaceID,
		Resource:    fmt.Sprintf("projects/%s/apps/%s/environments/%s/deployments/%s/**", d.projectID, d.appID, d.environmentID, d.deploymentID),
	}
}
