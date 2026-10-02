package principal

import (
	"context"
	"slices"
	"strings"

	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// ValidateDelegatedPermissions returns sorted, deduplicated permissions that the
// caller may assign to a child root key. Every request must be a supported URN in
// the caller's workspace and fit within one of the caller's permissions.
// For example, projects/*#read permits projects/proj_one#read, but not #write.
// Invalid requests fail with 400; requests beyond the caller's access fail with
// 403. Cancellation stops validation between permission checks.
func ValidateDelegatedPermissions(ctx context.Context, p *authprincipal.Principal, requestedPermissions []string) ([]string, error) {
	validatedPermissionSet := make(map[string]struct{}, len(requestedPermissions))
	for _, requestedPermission := range requestedPermissions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, exists := validatedPermissionSet[requestedPermission]; exists {
			continue
		}
		resource, action, err := parsePermission(requestedPermission, p.AuthorizedWorkspaceID)
		if err != nil {
			return nil, err
		}
		if err := p.Authorize(rbac.U(resource, action)); err != nil {
			return nil, err
		}
		validatedPermissionSet[requestedPermission] = struct{}{}
	}

	validatedPermissions := make([]string, 0, len(validatedPermissionSet))
	for validatedPermission := range validatedPermissionSet {
		validatedPermissions = append(validatedPermissions, validatedPermission)
	}
	slices.Sort(validatedPermissions)
	return validatedPermissions, nil
}

// parsePermission validates the URN, workspace, and resource/action combination.
// For example, keyspace logs accept #read, not #write. It checks syntax and the
// permission catalog, not whether the resource exists in the database.
func parsePermission(permission, workspaceID string) (urn.V1, permissions.Action, error) {
	resourceName, actionName, ok := strings.Cut(permission, "#")
	if !ok || strings.Contains(actionName, "#") {
		return urn.V1{}, "", invalidPermission()
	}
	resource, err := urn.ParseV1(resourceName)
	if err != nil || resource.WorkspaceID != workspaceID || !resource.SupportsPermissionAction(permissions.Action(actionName)) {
		return urn.V1{}, "", invalidPermission()
	}
	return resource, permissions.Action(actionName), nil
}

func invalidPermission() error {
	return fault.New("invalid permission",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public("A requested permission is not a supported URN permission in this workspace."))
}
