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
//	            ├── deployments/{deployment_id}
//	            ├── domains/{domain_id}
//	            ├── variables/{variable_id}
//	            └── gateway
type Environment struct {
	workspaceID string
	path        string
}

// String returns this environment resource path.
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
func (e Environment) Domain(domainID string) Domain {
	return Domain{workspaceID: e.workspaceID, path: fmt.Sprintf("%s/domains/%s", e.path, domainID)}
}

// Variable returns an environment variable resource path.
//
// Subresource:
//
//	environments/{environment_id}
//	└── variables/{variable_id}
func (e Environment) Variable(variableID string) EnvironmentVariable {
	return EnvironmentVariable{
		workspaceID: e.workspaceID,
		path:        fmt.Sprintf("%s/variables/%s", e.path, variableID),
	}
}

// Gateway returns builders for gateway resource paths.
//
// Subresource:
//
//	environments/{environment_id}
//	└── gateway
func (e Environment) Gateway() gateway {
	return gateway{workspaceID: e.workspaceID, path: e.path + "/gateway"}
}

// Any returns a descendant pattern below this environment.
func (e Environment) Any() V1 {
	return V1{
		WorkspaceID: e.workspaceID,
		Resource:    e.path + "/**",
	}
}

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
	workspaceID string
	path        string
}

// String returns this environment variable resource path.
func (e EnvironmentVariable) String() string {
	return V1{WorkspaceID: e.workspaceID, Resource: e.path}.String()
}
