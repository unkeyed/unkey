package urn

import "fmt"

// deployment builds deployment resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── deployments/{deployment_id}
type deployment struct {
	workspaceID string
	path        string
}

// Instance returns a deployment instance resource path.
//
// Subresource:
//
//	deployments/{deployment_id}
//	└── instances/{instance_id}
func (d deployment) Instance(instanceID string) V1 {
	return V1{
		WorkspaceID: d.workspaceID,
		Resource:    fmt.Sprintf("%s/instances/%s", d.path, instanceID),
	}
}

// Any returns a descendant pattern below this deployment.
func (d deployment) Any() V1 {
	return V1{
		WorkspaceID: d.workspaceID,
		Resource:    d.path + "/**",
	}
}
