package rbac

import (
	"errors"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

var errInvalidURNPermission = errors.New("invalid urn permission")

// UnkeyPermission represents an RBAC permission scope for Unkey resources.
//
// The resource name itself belongs to [urn.V1]. RBAC only adds the action
// suffix. Its resource may identify one concrete resource, a collection scope,
// or a descendant scope.
type UnkeyPermission struct {
	// Resource is the canonical v1 resource name being protected.
	Resource urn.V1

	// Action is the operation required on the resource.
	Action ActionType
}

// String returns the full RBAC permission string for this resource and action.
func (u UnkeyPermission) String() string {
	return fmt.Sprintf("%s#%s", u.Resource.String(), u.Action)
}

// U creates a leaf query for a typed action on a canonical resource name.
//
// The resource may be concrete or may contain canonical "*" and trailing "**"
// scopes. Evaluation succeeds only when one granted permission covers every
// resource and action represented by this requirement.
func U[R fmt.Stringer, A permissions.Action[R]](resource R, action A) PermissionQuery {
	return PermissionQuery{
		Operation:            OperatorNil,
		Value:                fmt.Sprintf("%s#%s", resource.String(), action.String()),
		Children:             []PermissionQuery{},
		matchUnkeyPermission: true,
	}
}

// isUnkeyPermission reports whether a granted string is a canonical Unkey
// permission URN, so the evaluator never applies wildcard semantics to legacy
// or customer-defined permission strings.
func isUnkeyPermission(value string) bool {
	_, err := parseUrnPermission(value)
	return err == nil
}

// evaluateUnkeyPermission evaluates Unkey resource permission URNs only.
//
// The required permission is already parsed by the leaf evaluator. Legacy tuple
// permissions and customer-defined permission strings never reach this path, so
// wildcard characters only expand scope for canonical Unkey permission URNs.
func evaluateUnkeyPermission(required UnkeyPermission, granted []string) bool {
	for _, permission := range granted {
		grantedPermission, err := parseUrnPermission(permission)
		if err != nil {
			continue
		}
		if permissionCovers(required, grantedPermission) {
			return true
		}
	}

	return false
}

// parseUrnPermission parses the RBAC action suffix around a resource URN.
//
// Resource-name parsing is delegated to pkg/urn so RBAC does not duplicate the
// Unkey URN grammar. RBAC only validates the action component after "#".
//
// Accepted:
//
//	unkey:v1:ws_1:ratelimits/namespaces/ns_1/overrides/ov_1#read_override
//	unkey:v1:ws_1:keyspaces/*/keys/*#read_key    wildcard grant
//	unkey:v1:ws_1:**#*                           admin grant (translated from admin:*)
//
// Rejected with errInvalidURNPermission:
//
//	unkey:v1:ws_1:keyspaces/ks_1                 missing "#action"
//	unkey:v1:ws_1:keyspaces/ks_1#read#key        more than one "#"
//	unkey:v1:ws_1:keyspaces/ks_1#*               action wildcard off the global resource
//	api.api_1.read_api                           legacy tuple, not a URN permission
func parseUrnPermission(value string) (UnkeyPermission, error) {
	var zero UnkeyPermission

	urnStr, action, ok := strings.Cut(value, "#")
	if !ok || strings.Contains(action, "#") {
		return zero, fmt.Errorf("%w: expected exactly one action separator", errInvalidURNPermission)
	}

	resource, err := urn.ParseV1(urnStr)
	if err != nil {
		return zero, fmt.Errorf("%w: %w", errInvalidURNPermission, err)
	}

	if err := validatePermissionAction(action); err != nil {
		return zero, fmt.Errorf("%w: invalid action: %v", errInvalidURNPermission, err)
	}
	if action == "*" && resource.Resource != "**" {
		return zero, fmt.Errorf("%w: action wildcard requires the global resource pattern %q", errInvalidURNPermission, "**")
	}

	return UnkeyPermission{
		Resource: resource,
		Action:   ActionType(action),
	}, nil
}

// permissionCovers reports whether a granted permission satisfies a required
// permission. Containment is directional: one grant must cover every resource
// and action represented by the requirement. Resource containment, including
// workspace equality, is delegated to [urn.V1.Covers].
func permissionCovers(required UnkeyPermission, granted UnkeyPermission) bool {
	if granted.Action != "*" && granted.Action != required.Action {
		return false
	}

	return granted.Resource.Covers(required.Resource)
}

// validatePermissionAction enforces the action grammar after "#": either the
// "*" wildcard or a word that cannot collide with URN separators.
func validatePermissionAction(action string) error {
	if action == "*" {
		return nil
	}
	if action == "" {
		return errors.New("must not be empty")
	}
	if strings.ContainsAny(action, ":#/*") {
		return errors.New(`must not contain ":", "#", "/", or "*"`)
	}
	if strings.HasPrefix(action, "_") || strings.HasSuffix(action, "_") {
		return errors.New(`must not start or end with "_"`)
	}
	return nil
}
