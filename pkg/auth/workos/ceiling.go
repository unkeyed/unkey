package workos

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/urn"
)

// ComputePermissionCeiling is the Compute MCP permission ceiling. It is the
// developer role's Compute resources: projects, apps, environments,
// deployments (including logs and build logs), domains, variables, gateway
// logs and policies, and GitHub apps. It excludes keyspaces, identities,
// ratelimits, rbac, portals, limits, and usage.
var ComputePermissionCeiling = []string{
	"github/apps/*#read",
	"github/apps/*#write",
	"github/apps/*#delete",
	"projects/*#read",
	"projects/*#write",
	"projects/*#delete",
	"projects/*/apps/*#read",
	"projects/*/apps/*#write",
	"projects/*/apps/*#delete",
	"projects/*/apps/*/environments/*#read",
	"projects/*/apps/*/environments/*#write",
	"projects/*/apps/*/environments/*#delete",
	"projects/*/apps/*/environments/*/deployments/*#read",
	"projects/*/apps/*/environments/*/deployments/*#write",
	"projects/*/apps/*/environments/*/deployments/*#delete",
	"projects/*/apps/*/environments/*/deployments/*/logs#read",
	"projects/*/apps/*/environments/*/deployments/*/buildLogs#read",
	"projects/*/apps/*/environments/*/domains/*#read",
	"projects/*/apps/*/environments/*/domains/*#write",
	"projects/*/apps/*/environments/*/domains/*#delete",
	"projects/*/apps/*/environments/*/variables/*#read",
	"projects/*/apps/*/environments/*/variables/*#write",
	"projects/*/apps/*/environments/*/variables/*#delete",
	"projects/*/apps/*/environments/*/gateway/logs#read",
	"projects/*/apps/*/environments/*/gateway/policies/*#read",
	"projects/*/apps/*/environments/*/gateway/policies/*#write",
	"projects/*/apps/*/environments/*/gateway/policies/*#delete",
}

// ValidatePermissionCeiling rejects a ceiling pattern the RBAC parser would
// not accept once it is bound to a workspace. A nil ceiling is no cap.
func ValidatePermissionCeiling(patterns []string) error {
	for _, pattern := range patterns {
		if _, err := parseCeilingPattern("ws_ceiling", pattern); err != nil {
			return err
		}
	}
	return nil
}

// applyPermissionCeiling narrows granted permissions to the ceiling. A nil
// ceiling returns granted unchanged. A grant is kept when the ceiling covers
// it. A broader grant, such as admin's **, contributes the narrower ceiling
// entry instead. The ceiling never adds a permission the grant does not cover.
func applyPermissionCeiling(workspaceID string, granted []string, ceiling []string) ([]string, error) {
	if ceiling == nil {
		return granted, nil
	}

	caps := make([]scopedPermission, 0, len(ceiling))
	for _, pattern := range ceiling {
		perm, err := parseCeilingPattern(workspaceID, pattern)
		if err != nil {
			return nil, err
		}
		caps = append(caps, perm)
	}

	seen := make(map[string]struct{})
	var narrowed []string
	for _, raw := range granted {
		grant, ok := parseGranted(raw)
		if !ok {
			continue
		}
		for _, cap := range caps {
			switch {
			case covers(grant, cap):
				narrowed = appendPermission(narrowed, seen, cap.value)
			case covers(cap, grant):
				narrowed = appendPermission(narrowed, seen, grant.value)
			}
		}
	}
	return narrowed, nil
}

type scopedPermission struct {
	resource urn.V1
	action   string
	value    string
}

func parseCeilingPattern(workspaceID string, pattern string) (scopedPermission, error) {
	resourcePath, action, ok := strings.Cut(pattern, "#")
	if !ok || resourcePath == "" || action == "" || strings.Contains(action, "#") {
		return zeroScopedPermission(), fmt.Errorf("permission_ceiling pattern %q must be resource#action", pattern)
	}
	resource, err := urn.ParseV1("unkey:v1:" + workspaceID + ":" + resourcePath)
	if err != nil {
		return zeroScopedPermission(), fmt.Errorf("permission_ceiling pattern %q: %w", pattern, err)
	}
	if err := validateCeilingAction(action, resource.Resource); err != nil {
		return zeroScopedPermission(), fmt.Errorf("permission_ceiling pattern %q: %w", pattern, err)
	}
	return scopedPermission{
		resource: resource,
		action:   action,
		value:    resource.String() + "#" + action,
	}, nil
}

func parseGranted(value string) (scopedPermission, bool) {
	resourceName, action, ok := strings.Cut(value, "#")
	if !ok || action == "" || strings.Contains(action, "#") {
		return zeroScopedPermission(), false
	}
	resource, err := urn.ParseV1(resourceName)
	if err != nil {
		return zeroScopedPermission(), false
	}
	return scopedPermission{resource: resource, action: action, value: value}, true
}

func zeroScopedPermission() scopedPermission {
	return scopedPermission{
		resource: urn.V1{WorkspaceID: "", Resource: ""},
		action:   "",
		value:    "",
	}
}

func validateCeilingAction(action string, resource string) error {
	switch action {
	case "read", "write", "delete", "decrypt", "verify", "limit":
		return nil
	case "*":
		if resource != "**" {
			return fmt.Errorf("action * is only valid on **")
		}
		return nil
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}

func covers(granted scopedPermission, required scopedPermission) bool {
	if granted.action != "*" && granted.action != required.action {
		return false
	}
	return granted.resource.Covers(required.resource)
}

func appendPermission(permissions []string, seen map[string]struct{}, value string) []string {
	if _, ok := seen[value]; ok {
		return permissions
	}
	seen[value] = struct{}{}
	return append(permissions, value)
}
