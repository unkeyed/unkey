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
	result, err := New().EvaluatePermissions(query, granted)
	if err != nil || result.Valid || result.Message == "" {
		return denialHint("")
	}
	return denialHint(result.Message)
}

func denialHint(message string) string {
	if message == "" {
		return "Insufficient permissions to access this resource. Log in again with the missing permission, or grant it."
	}
	hint := "Log in again with that permission, or grant it."
	if strings.Contains(message, "one of these permissions") {
		hint = "Log in again with one of those permissions, or grant it."
	}
	return message + ". " + hint
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

// readLeaf rewrites one leaf as a read. Legacy tuple leaves are kept as
// written: no read route accepts a legacy read action, so holding one must not
// turn a 404 into a 403.
func readLeaf(query PermissionQuery) PermissionQuery {
	if query.matchUnkeyPermission || isUnkeyPermission(query.Value) {
		return readURNLeaf(query.Value)
	}
	return PermissionQuery{
		Operation:            OperatorNil,
		Value:                query.Value,
		Children:             []PermissionQuery{},
		matchUnkeyPermission: false,
	}
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
