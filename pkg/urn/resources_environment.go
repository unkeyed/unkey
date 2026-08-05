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
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
func (e Environment) String() string {
	return V1{WorkspaceID: e.workspaceID, Resource: e.path}.String()
}

// Deployment returns builders for deployment resource paths.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── deployments/{deployment_id}
func (e Environment) Deployment(deploymentID string) Deployment {
	return Deployment{workspaceID: e.workspaceID, path: fmt.Sprintf("%s/deployments/%s", e.path, deploymentID)}
}

// Domain is a domain resource path.
type Domain struct {
	workspaceID string
	path        string
}

// String returns this domain resource path.
func (d Domain) String() string {
	return V1{WorkspaceID: d.workspaceID, Resource: d.path}.String()
}

// Domain returns a domain resource path.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── domains/{domain_id}
func (e Environment) Domain(domainID string) Domain {
	return Domain{workspaceID: e.workspaceID, path: fmt.Sprintf("%s/domains/%s", e.path, domainID)}
}

// Variable is an environment variable resource path.
type Variable struct {
	workspaceID string
	path        string
}

// String returns this environment variable resource path.
func (v Variable) String() string {
	return V1{WorkspaceID: v.workspaceID, Resource: v.path}.String()
}

// Variable returns a variable resource path.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── variables/{variable_id}
func (e Environment) Variable(variableID string) Variable {
	return Variable{workspaceID: e.workspaceID, path: fmt.Sprintf("%s/variables/%s", e.path, variableID)}
}

// Gateway returns the environment's gateway resource namespace.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
func (e Environment) Gateway() Gateway {
	return Gateway{workspaceID: e.workspaceID, path: fmt.Sprintf("%s/gateway", e.path)}
}

// Any returns a descendant pattern below this environment.
func (e Environment) Any() V1 {
	return V1{
		WorkspaceID: e.workspaceID,
		Resource:    fmt.Sprintf("%s/**", e.path),
	}
}
