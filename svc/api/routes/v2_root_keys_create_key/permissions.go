package handler

import (
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

func authorizePermissions(p *principal.Principal, requested []string) ([]string, error) {
	grants := make(map[string]struct{}, len(requested))
	for _, permission := range requested {
		if _, exists := grants[permission]; exists {
			continue
		}
		resource, action, err := parseCanonicalPermission(permission, p.AuthorizedWorkspaceID)
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

func parseCanonicalPermission(permission, workspaceID string) (urn.V1, permissions.Action, error) {
	resourceName, actionName, ok := strings.Cut(permission, "#")
	if !ok || strings.Contains(actionName, "#") {
		return urn.V1{}, "", invalidPermission()
	}
	resource, err := urn.ParseV1(resourceName)
	if err != nil || resource.WorkspaceID != workspaceID || !canonicalActionAllowed(resource.Resource, actionName) {
		return urn.V1{}, "", invalidPermission()
	}
	return resource, permissions.Action(actionName), nil
}

func canonicalActionAllowed(resource, action string) bool {
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
	case isCanonicalLog(parts):
		allowed = []string{"read"}
	case isCanonicalKey(parts):
		allowed = append(allowed, "decrypt", "verify")
	case isCanonicalNamespace(parts):
		allowed = append(allowed, "limit")
	case parts[0] == "projects" && recursive && len(parts) == 2:
		allowed = append(allowed, "decrypt", "verify", "limit")
	case recursive && len(parts) == 4 && parts[0] == "projects" && parts[2] == "keyspaces":
		allowed = append(allowed, "decrypt", "verify")
	}
	return slices.Contains(allowed, action)
}

func isCanonicalLog(parts []string) bool {
	return len(parts) == 5 && parts[0] == "projects" && parts[2] == "keyspaces" && parts[4] == "logs" ||
		len(parts) == 6 && parts[0] == "projects" && parts[2] == "ratelimits" && parts[3] == "namespaces" && parts[5] == "logs" ||
		len(parts) == 8 && parts[0] == "projects" && parts[2] == "apps" && parts[4] == "environments" && parts[6] == "gateway" && parts[7] == "logs" ||
		len(parts) == 9 && parts[0] == "projects" && parts[2] == "apps" && parts[4] == "environments" && parts[6] == "deployments" && parts[8] == "logs"
}

func isCanonicalKey(parts []string) bool {
	return len(parts) == 6 && parts[0] == "projects" && parts[2] == "keyspaces" && parts[4] == "keys"
}

func isCanonicalNamespace(parts []string) bool {
	return len(parts) == 5 && parts[0] == "projects" && parts[2] == "ratelimits" && parts[3] == "namespaces"
}

func invalidPermission() error {
	return fault.New("invalid canonical permission",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public("A requested permission is not a supported URN permission in this workspace."))
}
