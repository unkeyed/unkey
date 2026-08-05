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
	workspaceID string
	path        string
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
	workspaceID string
	path        string
}

// String returns this deployment instance resource path.
func (i Instance) String() string {
	return V1{WorkspaceID: i.workspaceID, Resource: i.path}.String()
}

// String returns this deployment resource path.
func (d Deployment) String() string {
	return V1{WorkspaceID: d.workspaceID, Resource: d.path}.String()
}

// Instance returns a deployment instance resource path.
func (d Deployment) Instance(instanceID string) Instance {
	return Instance{workspaceID: d.workspaceID, path: fmt.Sprintf("%s/instances/%s", d.path, instanceID)}
}

// Any returns a descendant pattern below this deployment.
func (d Deployment) Any() V1 {
	return V1{
		WorkspaceID: d.workspaceID,
		Resource:    fmt.Sprintf("%s/**", d.path),
	}
}
