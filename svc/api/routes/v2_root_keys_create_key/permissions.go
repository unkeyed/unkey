package handler

import (
	"context"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

func authorizePermissions(ctx context.Context, p *principal.Principal, requested []string) ([]string, error) {
	index := make(map[permissions.Action]*permissionNode)
	for i, permission := range p.Permissions {
		if i%64 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		resource, action, err := parsePermission(permission, p.AuthorizedWorkspaceID)
		if err != nil {
			continue
		}
		root := index[action]
		if root == nil {
			root = new(permissionNode)
			index[action] = root
		}
		root.insert(resource.Resource)
	}

	grants := make(map[string]struct{}, len(requested))
	for i, permission := range requested {
		if i%64 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if _, exists := grants[permission]; exists {
			continue
		}
		resource, action, err := parsePermission(permission, p.AuthorizedWorkspaceID)
		if err != nil {
			return nil, err
		}
		if !index[action].covers(resource.Resource) && !index[permissions.Wildcard].covers(resource.Resource) {
			return nil, insufficientPermissions()
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

type permissionNode struct {
	children map[string]*permissionNode
	exact    bool
	subtree  bool
}

func (n *permissionNode) insert(resource string) {
	segments := strings.Split(resource, "/")
	if segments[len(segments)-1] == "**" {
		segments = segments[:len(segments)-1]
		if len(segments) == 0 {
			n.subtree = true
			return
		}
	}
	for _, segment := range segments {
		if n.children == nil {
			n.children = make(map[string]*permissionNode)
		}
		child := n.children[segment]
		if child == nil {
			child = new(permissionNode)
			n.children[segment] = child
		}
		n = child
	}
	if strings.HasSuffix(resource, "/**") {
		n.subtree = true
	} else {
		n.exact = true
	}
}

func (n *permissionNode) covers(resource string) bool {
	if n == nil {
		return false
	}
	return n.coversSegments(strings.Split(resource, "/"))
}

func (n *permissionNode) coversSegments(segments []string) bool {
	if n.subtree {
		return true
	}
	if len(segments) == 0 {
		return n.exact
	}
	if child := n.children[segments[0]]; child != nil && child.coversSegments(segments[1:]) {
		return true
	}
	return segments[0] != "*" && segments[0] != "**" && n.children["*"] != nil && n.children["*"].coversSegments(segments[1:])
}

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

func insufficientPermissions() error {
	return fault.New("insufficient permissions",
		fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
		fault.Public("Insufficient permissions to grant the requested permission."))
}
