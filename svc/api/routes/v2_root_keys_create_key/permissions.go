package handler

import (
	"context"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// validateDelegatedPermissions returns sorted, deduplicated permissions that the
// caller may grant to a child root key. Every request must be a supported URN in
// the caller's workspace and fit within one of the caller's permissions.
// For example, projects/*#read permits projects/proj_one#read, but not #write.
// Invalid requests fail with 400; requests beyond the caller's access fail with
// 403. Cancellation stops validation between permission checks.
func validateDelegatedPermissions(ctx context.Context, p *principal.Principal, requested []string) ([]string, error) {
	grants := make(map[string]struct{}, len(requested))
	for _, permission := range requested {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, exists := grants[permission]; exists {
			continue
		}
		resource, action, err := parsePermission(permission, p.AuthorizedWorkspaceID)
		if err != nil {
			return nil, err
		}
		if err := p.Authorize(rbac.U(resource, action)); err != nil {
			return nil, err
		}
		grants[permission] = struct{}{}
	}

	result := make([]string, 0, len(grants))
	for grant := range grants {
		result = append(result, grant)
	}
	slices.Sort(result)
	return result, nil
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
	if err != nil || resource.WorkspaceID != workspaceID || !actionAllowed(resource.Resource, actionName) {
		return urn.V1{}, "", invalidPermission()
	}
	return resource, permissions.Action(actionName), nil
}

// actionAllowed checks actions for a resource path already validated by ParseV1.
// Resource kinds come from fixed path positions, never ID text: a project named
// "keys" does not gain #decrypt. Subtree grants include descendant actions;
// for example, projects/p/** may grant #decrypt for keys below that project.
func actionAllowed(resource, action string) bool {
	if resource == "**" {
		return slices.Contains([]string{"read", "write", "delete", "decrypt", "verify", "limit", permissions.Wildcard}, action)
	}
	if action == permissions.Wildcard {
		return false
	}

	parts := strings.Split(resource, "/")
	recursive := parts[len(parts)-1] == "**"
	if recursive {
		parts = parts[:len(parts)-1]
	}
	allowed := []string{"read", "write", "delete"}
	switch {
	case parts[0] == "rootKeys":
		allowed = []string{"write"}
	case len(parts) == 6 && parts[0] == "projects" && parts[2] == "portals" && parts[4] == "sessions":
		allowed = []string{"write"}
	case isLog(parts):
		allowed = []string{"read"}
	case isKey(parts):
		allowed = append(allowed, "decrypt", "verify")
	case isNamespace(parts):
		allowed = append(allowed, "limit")
	case parts[0] == "projects" && recursive && len(parts) == 2:
		allowed = append(allowed, "decrypt", "verify", "limit")
	case recursive && len(parts) == 4 && parts[0] == "projects" && parts[2] == "keyspaces":
		allowed = append(allowed, "decrypt", "verify")
	}
	return slices.Contains(allowed, action)
}

func isLog(parts []string) bool {
	return len(parts) == 5 && parts[0] == "projects" && parts[2] == "keyspaces" && parts[4] == "logs" ||
		len(parts) == 6 && parts[0] == "projects" && parts[2] == "ratelimits" && parts[3] == "namespaces" && parts[5] == "logs" ||
		len(parts) == 8 && parts[0] == "projects" && parts[2] == "apps" && parts[4] == "environments" && parts[6] == "gateway" && parts[7] == "logs" ||
		len(parts) == 9 && parts[0] == "projects" && parts[2] == "apps" && parts[4] == "environments" && parts[6] == "deployments" && parts[8] == "logs"
}

func isKey(parts []string) bool {
	return len(parts) == 6 && parts[0] == "projects" && parts[2] == "keyspaces" && parts[4] == "keys"
}

func isNamespace(parts []string) bool {
	return len(parts) == 5 && parts[0] == "projects" && parts[2] == "ratelimits" && parts[3] == "namespaces"
}

func invalidPermission() error {
	return fault.New("invalid permission",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public("A requested permission is not a supported URN permission in this workspace."))
}
