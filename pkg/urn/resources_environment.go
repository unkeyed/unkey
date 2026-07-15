package urn

import "fmt"

// Environment builds environment resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
type Environment struct {
	workspaceID string
	path        string
}

// String returns this environment resource path.
//
// Subresource:
//
//	apps/{app_id}
//	└── environments/{environment_id}
func (e Environment) String() string {
	return V1{WorkspaceID: e.workspaceID, Resource: e.path}.String()
}

// Deployment returns builders for deployment resource paths.
//
// Subresource:
//
//	environments/{environment_id}
//	└── deployments/{deployment_id}
func (e Environment) Deployment(deploymentID string) Deployment {
	return Deployment{workspaceID: e.workspaceID, path: fmt.Sprintf("%s/deployments/%s", e.path, deploymentID)}
}

// Domain returns a domain resource path.
//
// Subresource:
//
//	environments/{environment_id}
//	└── domains/{domain_id}
func (e Environment) Domain(domainID string) V1 {
	return V1{
		WorkspaceID: e.workspaceID,
		Resource:    fmt.Sprintf("%s/domains/%s", e.path, domainID),
	}
}

// Variable returns a variable resource path.
//
// Subresource:
//
//	environments/{environment_id}
//	└── variables/{variable_id}
func (e Environment) Variable(variableID string) V1 {
	return V1{
		WorkspaceID: e.workspaceID,
		Resource:    fmt.Sprintf("%s/variables/%s", e.path, variableID),
	}
}

// Any returns a descendant pattern below this environment.
func (e Environment) Any() V1 {
	return V1{
		WorkspaceID: e.workspaceID,
		Resource:    e.path + "/**",
	}
}
