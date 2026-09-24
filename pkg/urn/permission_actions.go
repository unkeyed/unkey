package urn

import "strings"

// PermissionAction identifies an operation supported by a resource.
type PermissionAction string

// String returns the serialized permission action.
func (a PermissionAction) String() string {
	return string(a)
}

const (
	// PermissionRead authorizes reading a resource, except root keys.
	PermissionRead PermissionAction = "read"
	// PermissionWrite authorizes creating or updating a resource.
	PermissionWrite PermissionAction = "write"
	// PermissionDelete authorizes deleting a resource.
	PermissionDelete PermissionAction = "delete"
	// PermissionDecrypt authorizes decrypting key data.
	PermissionDecrypt PermissionAction = "decrypt"
	// PermissionVerify authorizes verifying a key.
	PermissionVerify PermissionAction = "verify"
	// PermissionLimit authorizes using a rate limit namespace.
	PermissionLimit PermissionAction = "limit"
	// PermissionAll is the action used by the global administrator permission.
	PermissionAll PermissionAction = "*"
)

// permissionActionSet provides constant-time action lookups.
type permissionActionSet map[PermissionAction]bool

// These sets group actions shared by several resource types.
var (
	readWriteDelete          = newPermissionActionSet(PermissionRead, PermissionWrite, PermissionDelete)
	keyActions               = newPermissionActionSet(PermissionRead, PermissionWrite, PermissionDelete, PermissionDecrypt, PermissionVerify)
	namespaceActions         = newPermissionActionSet(PermissionRead, PermissionWrite, PermissionDelete, PermissionLimit)
	projectDescendantActions = newPermissionActionSet(PermissionRead, PermissionWrite, PermissionDelete, PermissionDecrypt, PermissionVerify, PermissionLimit)
	globalActions            = newPermissionActionSet(PermissionRead, PermissionWrite, PermissionDelete, PermissionDecrypt, PermissionVerify, PermissionLimit, PermissionAll)
)

// permissionResource supplies the actions for one catalog resource shape.
type permissionResource interface {
	permissionActions(selectsDescendants bool) permissionActionSet
}

// rootKey represents the root key path shape used by parsed resource names.
type rootKey struct{}

// portalSession represents the portal session path shape used by parsed resource names.
type portalSession struct{}

// global represents the workspace-wide ** resource pattern.
type global struct{}

// SupportsPermissionAction reports whether action is valid for this resource
// or pattern. A zero value or manually constructed invalid V1 returns false.
func (v V1) SupportsPermissionAction(action PermissionAction) bool {
	if err := validateWorkspaceID(v.WorkspaceID); err != nil {
		return false
	}
	if err := validateResourcePath(v.Resource); err != nil {
		return false
	}
	if v.Resource == "**" {
		return global{}.permissionActions(false)[action]
	}

	segments := strings.Split(v.Resource, "/")
	selectsDescendants := segments[len(segments)-1] == "**"
	if selectsDescendants {
		segments = segments[:len(segments)-1]
	}
	for _, shape := range resourcePathShapes {
		if resourcePathMatchesShape(segments, shape.segments) {
			return shape.resource.permissionActions(selectsDescendants)[action]
		}
	}
	if selectsDescendants {
		for _, shape := range resourceContainerPathShapes {
			if resourcePathMatchesShape(segments, shape.segments) {
				return shape.resource.permissionActions(true)[action]
			}
		}
	}
	return false
}

// newPermissionActionSet returns a set containing the supplied known actions.
func newPermissionActionSet(actions ...PermissionAction) permissionActionSet {
	set := make(permissionActionSet, len(actions))
	for _, action := range actions {
		set[action] = true
	}
	return set
}

// permissionActions returns the read, write, and delete actions for GitHub apps.
func (GitHubApp) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns write for root keys.
func (rootKey) permissionActions(bool) permissionActionSet {
	return newPermissionActionSet(PermissionWrite)
}

// permissionActions adds descendant-only actions to project patterns.
func (Project) permissionActions(descendants bool) permissionActionSet {
	if descendants {
		return projectDescendantActions
	}
	return readWriteDelete
}

// permissionActions returns the read, write, and delete actions for apps.
func (App) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns the read, write, and delete actions for environments.
func (Environment) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns the read, write, and delete actions for deployments.
func (Deployment) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns read for deployment logs.
func (DeploymentLogs) permissionActions(bool) permissionActionSet {
	return newPermissionActionSet(PermissionRead)
}

// permissionActions returns the read, write, and delete actions for domains.
func (Domain) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns the read, write, and delete actions for environment variables.
func (EnvironmentVariable) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns all actions selected below the gateway container.
func (gateway) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns read for gateway logs.
func (GatewayLogs) permissionActions(bool) permissionActionSet {
	return newPermissionActionSet(PermissionRead)
}

// permissionActions returns the read, write, and delete actions for gateway policies.
func (GatewayPolicy) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns the read, write, and delete actions for identities.
func (Identity) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions adds key-only actions to keyspace descendant patterns.
func (Keyspace) permissionActions(descendants bool) permissionActionSet {
	if descendants {
		return keyActions
	}
	return readWriteDelete
}

// permissionActions returns read for keyspace logs.
func (KeyspaceLogs) permissionActions(bool) permissionActionSet {
	return newPermissionActionSet(PermissionRead)
}

// permissionActions returns every action supported by keys.
func (Key) permissionActions(bool) permissionActionSet { return keyActions }

// permissionActions returns the read, write, and delete actions for portals.
func (Portal) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns write for portal sessions.
func (portalSession) permissionActions(bool) permissionActionSet {
	return newPermissionActionSet(PermissionWrite)
}

// permissionActions returns every action supported by rate limit namespaces.
func (RatelimitNamespace) permissionActions(bool) permissionActionSet { return namespaceActions }

// permissionActions returns read for rate limit logs.
func (RatelimitLogs) permissionActions(bool) permissionActionSet {
	return newPermissionActionSet(PermissionRead)
}

// permissionActions returns the read, write, and delete actions for rate limit overrides.
func (RatelimitOverride) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns all actions selected below the RBAC container.
func (rbac) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns the read, write, and delete actions for roles.
func (Role) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns the read, write, and delete actions for permission definitions.
func (Permission) permissionActions(bool) permissionActionSet { return readWriteDelete }

// permissionActions returns every supported action and the action wildcard.
func (global) permissionActions(bool) permissionActionSet { return globalActions }
