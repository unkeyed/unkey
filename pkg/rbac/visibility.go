package rbac

import "strings"

// CanRead reports whether granted permissions let the caller read the resources
// in query, ignoring the action query actually requires.
//
// A denial stays hidden as not-found only when this is false. A caller who can
// already read the resource learns nothing new from a 403, so handlers use that
// status and name the missing permission instead.
func CanRead(query PermissionQuery, granted []string) bool {
	result, err := New().EvaluatePermissions(toReadQuery(query), granted)
	return err == nil && result.Valid
}

// PermissionDeniedDetail is the public explanation for a query the caller
// cannot satisfy. It names the missing permission or permissions and tells the
// caller to log in again or grant them.
func PermissionDeniedDetail(query PermissionQuery, granted []string) string {
	const fallback = "Insufficient permissions to access this resource. Log in again with the missing permission, or grant it."

	result, err := New().EvaluatePermissions(query, granted)
	if err != nil || result.Valid || result.Message == "" {
		return fallback
	}

	hint := "Log in again with that permission, or grant it."
	if strings.Contains(result.Message, "one of these permissions") {
		hint = "Log in again with one of those permissions, or grant it."
	}
	return result.Message + ". " + hint
}

// toReadQuery rewrites query so every leaf asks for read access to the same
// resource. AND and OR structure is preserved: the caller can read an OR when
// any alternative is readable, and an AND only when every branch is.
func toReadQuery(query PermissionQuery) PermissionQuery {
	if query.Value != "" {
		return readLeaf(query)
	}

	children := make([]PermissionQuery, len(query.Children))
	for i, child := range query.Children {
		children[i] = toReadQuery(child)
	}
	return PermissionQuery{
		Operation:            query.Operation,
		Value:                "",
		Children:             children,
		matchUnkeyPermission: false,
	}
}

func readLeaf(query PermissionQuery) PermissionQuery {
	if query.matchUnkeyPermission || isUnkeyPermission(query.Value) {
		return readURNLeaf(query.Value)
	}

	tuple, err := TupleFromString(query.Value)
	if err != nil {
		return PermissionQuery{
			Operation:            OperatorNil,
			Value:                query.Value,
			Children:             []PermissionQuery{},
			matchUnkeyPermission: false,
		}
	}

	actions := readActionsFor(tuple.ResourceType, tuple.Action)
	if len(actions) == 0 {
		return PermissionQuery{
			Operation:            OperatorNil,
			Value:                query.Value,
			Children:             []PermissionQuery{},
			matchUnkeyPermission: false,
		}
	}
	if len(actions) == 1 {
		tuple.Action = actions[0]
		return T(tuple)
	}

	children := make([]PermissionQuery, len(actions))
	for i, action := range actions {
		tuple.Action = action
		children[i] = T(tuple)
	}
	return Or(children...)
}

func readURNLeaf(value string) PermissionQuery {
	resource, action, ok := strings.Cut(value, "#")
	if ok && action != "read" && action != "*" {
		value = resource + "#read"
	}
	leaf := urnReadQuery(value)
	parent, ok := singletonParent(resource)
	if !ok {
		return leaf
	}
	// buildLogs and logs are children of the resource a not-found response
	// names. Reading that parent already reveals it, so a missing child read
	// is a 403 rather than a 404. Id segments are not stripped: a project read
	// must not reveal a keyspace, and an environment read must not reveal a
	// deployment.
	return Or(leaf, urnReadQuery(parent+"#read"))
}

func urnReadQuery(value string) PermissionQuery {
	return PermissionQuery{
		Operation:            OperatorNil,
		Value:                value,
		Children:             []PermissionQuery{},
		matchUnkeyPermission: true,
	}
}

func singletonParent(resource string) (string, bool) {
	for _, suffix := range []string{"/buildLogs", "/logs"} {
		if strings.HasSuffix(resource, suffix) {
			return strings.TrimSuffix(resource, suffix), true
		}
	}
	return "", false
}

// readActionsFor returns the legacy read actions that reveal resource.
//
// Permissions and roles share the rbac resource type but are different
// resources, so a role read must not reveal a permission. Every other resource
// type treats any of its read actions as visibility: the caller already knows
// that resource exists.
func readActionsFor(resource ResourceType, action ActionType) []ActionType {
	switch resource {
	case Api:
		return []ActionType{ReadAPI, ReadKey, ReadAnalytics}
	case Ratelimit:
		return []ActionType{ReadNamespace, ReadOverride, ListOverrides}
	case Identity:
		return []ActionType{ReadIdentity}
	case Project:
		return []ActionType{ReadProject, ReadGatewayRequests, ReadRuntimeLogs}
	case App:
		return []ActionType{ReadApp}
	case Environment:
		return []ActionType{ReadEnvironment, ReadDeployment, ReadEnvironmentVariables, ReadPolicies, ReadDomain}
	case Portal:
		return []ActionType{ReadPortal}
	case Workspace:
		// Workspace permissions are create_root_key and install_github. Neither
		// is a read, so holding one does not reveal the workspace.
		return nil
	case Rbac:
		switch action {
		case ReadRole, CreateRole, UpdateRole, DeleteRole, AddRoleToKey, RemoveRoleFromKey:
			return []ActionType{ReadRole}
		case ReadPermission, CreatePermission, UpdatePermission, DeletePermission,
			AddPermissionToKey, RemovePermissionFromKey, AddPermissionToRole, RemovePermissionFromRole:
			return []ActionType{ReadPermission}
		case ReadAPI, CreateAPI, DeleteAPI, UpdateAPI, CreateKey, UpdateKey, DeleteKey,
			EncryptKey, DecryptKey, ReadKey, VerifyKey, ReadAnalytics, CreateRootKey, InstallGithub,
			Limit, CreateNamespace, ReadNamespace, UpdateNamespace, DeleteNamespace, SetOverride,
			ReadOverride, DeleteOverride, ListOverrides, CreateIdentity, ReadIdentity, UpdateIdentity,
			DeleteIdentity, CreateDeployment, ReadDeployment, StopDeployment, StartDeployment,
			PromoteDeployment, RollbackDeployment, GenerateUploadURL, CreateProject, ReadProject,
			UpdateProject, DeleteProject, ReadGatewayRequests, ReadRuntimeLogs, CreateApp, ReadApp,
			UpdateApp, DeleteApp, ConnectRepository, ReadEnvironment, UpdateEnvironment,
			SetEnvironmentVariables, RemoveEnvironmentVariables, ReadEnvironmentVariables, SetPolicies,
			UpdatePolicy, ReadPolicies, CreateDomain, ReadDomain, DeleteDomain, VerifyDomain,
			CreatePortal, ReadPortal, UpdatePortal, DeletePortal, CreatePortalSession:
			return []ActionType{ReadPermission, ReadRole}
		default:
			return []ActionType{ReadPermission, ReadRole}
		}
	default:
		return nil
	}
}
