package urn

import "fmt"

// Gateway builds resources beneath an environment's gateway namespace.
type Gateway struct {
	workspaceID string
	path        string
}

// String returns this gateway resource path.
func (g Gateway) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: g.path}.String()
}

// GatewayPolicy is a gateway policy resource path.
type GatewayPolicy struct {
	workspaceID string
	path        string
}

// String returns this gateway policy resource path.
func (p GatewayPolicy) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: p.path}.String()
}

// Policy returns a policy beneath this gateway.
func (g Gateway) Policy(policyID string) GatewayPolicy {
	return GatewayPolicy{workspaceID: g.workspaceID, path: fmt.Sprintf("%s/policies/%s", g.path, policyID)}
}
