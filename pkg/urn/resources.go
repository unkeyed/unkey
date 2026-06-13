package urn

import "fmt"

// Build returns builders for canonical Unkey resource names.
func Build() Builder {
	return Builder{}
}

// Builder starts typed URN construction.
type Builder struct{}

// Workspace returns builders for canonical v1 resource names in a workspace.
func (Builder) Workspace(workspaceID string) workspaceResource {
	return workspaceResource{workspaceID: workspaceID}
}

// workspaceResource builds resource paths inside one workspace.
type workspaceResource struct {
	workspaceID string
}

// Settings returns the workspace settings resource path.
func (r workspaceResource) Settings() V1 {
	return r.v1("settings")
}

// Membership returns a workspace membership resource path.
func (r workspaceResource) Membership(membershipID string) V1 {
	return r.v1(fmt.Sprintf("memberships/%s", membershipID))
}

// Invitation returns a workspace invitation resource path.
func (r workspaceResource) Invitation(invitationID string) V1 {
	return r.v1(fmt.Sprintf("invitations/%s", invitationID))
}

// Billing returns builders for billing resource paths.
func (r workspaceResource) Billing() billingResource {
	return billingResource{workspaceID: r.workspaceID, path: "billing"}
}

// Keyspace returns builders for keyspace resource paths.
func (r workspaceResource) Keyspace(keyspaceID string) keyspaceResource {
	return keyspaceResource{workspaceID: r.workspaceID, path: fmt.Sprintf("keyspaces/%s", keyspaceID)}
}

// Identity returns an identity resource path.
func (r workspaceResource) Identity(identityID string) V1 {
	return r.v1(fmt.Sprintf("identities/%s", identityID))
}

// RatelimitNamespace returns builders for rate limit namespace resource paths.
func (r workspaceResource) RatelimitNamespace(namespaceID string) ratelimitNamespaceResource {
	return ratelimitNamespaceResource{workspaceID: r.workspaceID, path: fmt.Sprintf("ratelimits/namespaces/%s", namespaceID)}
}

// Role returns an RBAC role resource path.
func (r workspaceResource) Role(roleID string) V1 {
	return r.v1(fmt.Sprintf("rbac/roles/%s", roleID))
}

// Permission returns an RBAC permission resource path.
func (r workspaceResource) Permission(permissionID string) V1 {
	return r.v1(fmt.Sprintf("rbac/permissions/%s", permissionID))
}

// Project returns builders for project resource paths.
func (r workspaceResource) Project(projectID string) projectResource {
	return projectResource{workspaceID: r.workspaceID, path: fmt.Sprintf("projects/%s", projectID)}
}

// Portal returns builders for portal resource paths.
func (r workspaceResource) Portal(portalID string) portalResource {
	return portalResource{workspaceID: r.workspaceID, path: fmt.Sprintf("portals/%s", portalID)}
}

// v1 wraps a resource path in a [V1] for this workspace.
func (r workspaceResource) v1(path string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: path}
}

// billingResource builds billing resource paths.
type billingResource struct {
	workspaceID string
	path        string
}

// V1 returns the billing resource name.
func (r billingResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r billingResource) String() string {
	return r.V1().String()
}

// Invoice returns a billing invoice resource path.
func (r billingResource) Invoice(invoiceID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/invoices/%s", r.path, invoiceID)}
}

// Quotas returns the workspace billing quotas resource path.
func (r billingResource) Quotas() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/quotas"}
}

// keyspaceResource builds keyspace resource paths.
type keyspaceResource struct {
	workspaceID string
	path        string
}

// V1 returns the keyspace resource name.
func (r keyspaceResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r keyspaceResource) String() string {
	return r.V1().String()
}

// Key returns a key resource path.
func (r keyspaceResource) Key(keyID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/keys/%s", r.path, keyID)}
}

// Any returns a descendant pattern below this keyspace.
func (r keyspaceResource) Any() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/**"}
}

// ratelimitNamespaceResource builds rate limit namespace resource paths.
type ratelimitNamespaceResource struct {
	workspaceID string
	path        string
}

// V1 returns the rate limit namespace resource name.
func (r ratelimitNamespaceResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r ratelimitNamespaceResource) String() string {
	return r.V1().String()
}

// Override returns a rate limit override resource path.
func (r ratelimitNamespaceResource) Override(overrideID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/overrides/%s", r.path, overrideID)}
}

// Any returns a descendant pattern below this rate limit namespace.
func (r ratelimitNamespaceResource) Any() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/**"}
}

// projectResource builds project resource paths.
type projectResource struct {
	workspaceID string
	path        string
}

// V1 returns the project resource name.
func (r projectResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r projectResource) String() string {
	return r.V1().String()
}

// App returns builders for app resource paths.
func (r projectResource) App(appID string) appResource {
	return appResource{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/apps/%s", r.path, appID)}
}

// Any returns a descendant pattern below this project.
func (r projectResource) Any() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/**"}
}

// appResource builds app resource paths.
type appResource struct {
	workspaceID string
	path        string
}

// V1 returns the app resource name.
func (r appResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r appResource) String() string {
	return r.V1().String()
}

// Environment returns builders for environment resource paths.
func (r appResource) Environment(environmentID string) environmentResource {
	return environmentResource{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/environments/%s", r.path, environmentID)}
}

// Any returns a descendant pattern below this app.
func (r appResource) Any() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/**"}
}

// environmentResource builds environment resource paths.
type environmentResource struct {
	workspaceID string
	path        string
}

// V1 returns the environment resource name.
func (r environmentResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r environmentResource) String() string {
	return r.V1().String()
}

// Deployment returns builders for deployment resource paths.
func (r environmentResource) Deployment(deploymentID string) deploymentResource {
	return deploymentResource{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/deployments/%s", r.path, deploymentID)}
}

// Domain returns a domain resource path.
func (r environmentResource) Domain(domainID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/domains/%s", r.path, domainID)}
}

// Variable returns a variable resource path.
func (r environmentResource) Variable(variableID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/variables/%s", r.path, variableID)}
}

// Any returns a descendant pattern below this environment.
func (r environmentResource) Any() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/**"}
}

// deploymentResource builds deployment resource paths.
type deploymentResource struct {
	workspaceID string
	path        string
}

// V1 returns the deployment resource name.
func (r deploymentResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r deploymentResource) String() string {
	return r.V1().String()
}

// Instance returns a deployment instance resource path.
func (r deploymentResource) Instance(instanceID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/instances/%s", r.path, instanceID)}
}

// Any returns a descendant pattern below this deployment.
func (r deploymentResource) Any() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/**"}
}

// portalResource builds portal resource paths.
type portalResource struct {
	workspaceID string
	path        string
}

// V1 returns the portal resource name.
func (r portalResource) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// String returns the canonical resource-name string.
func (r portalResource) String() string {
	return r.V1().String()
}

// SessionToken returns a portal session token resource path.
func (r portalResource) SessionToken(tokenID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/session_tokens/%s", r.path, tokenID)}
}

// Session returns a portal session resource path.
func (r portalResource) Session(sessionID string) V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("%s/sessions/%s", r.path, sessionID)}
}

// Branding returns a portal branding resource path.
func (r portalResource) Branding() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/branding"}
}

// Any returns a descendant pattern below this portal.
func (r portalResource) Any() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path + "/**"}
}
