package handler

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/migration"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

func expandPermissions(ctx context.Context, tx db.DBTX, p *principal.Principal, requested []string) ([]string, error) {
	callerPermissions, err := canonicalCallerPermissions(ctx, tx, p)
	if err != nil {
		return nil, err
	}

	grants := make(map[string]struct{})
	for _, permission := range requested {
		if _, exists := grants[permission]; exists {
			continue
		}

		if permission == "*" {
			if authorizeErr := rbac.Check(rbac.S(permission), p.Permissions); authorizeErr != nil {
				return nil, authorizeErr
			}
			grants[permission] = struct{}{}
			continue
		}

		if strings.HasPrefix(permission, "unkey:") {
			resource, action, parseErr := parseCanonicalPermission(permission, p.AuthorizedWorkspaceID)
			if parseErr != nil {
				return nil, parseErr
			}
			if authorizeErr := rbac.Check(rbac.U(resource, action), callerPermissions); authorizeErr != nil {
				return nil, authorizeErr
			}
			grants[permission] = struct{}{}
			continue
		}

		canonical, translateErr := translateLegacyPermission(ctx, tx, p.AuthorizedWorkspaceID, permission)
		if translateErr != nil {
			if errors.Is(translateErr, migration.ErrUnsupported) || errors.Is(translateErr, migration.ErrInvalidScope) {
				return nil, invalidPermission()
			}
			return nil, translateErr
		}
		resource, action, parseErr := parseCanonicalPermission(canonical, p.AuthorizedWorkspaceID)
		if parseErr != nil {
			return nil, parseErr
		}
		if authorizeErr := rbac.Check(rbac.U(resource, action), callerPermissions); authorizeErr != nil {
			return nil, authorizeErr
		}
		grants[permission] = struct{}{}
		grants[canonical] = struct{}{}
	}

	result := make([]string, 0, len(grants))
	for grant := range grants {
		result = append(result, grant)
	}
	slices.Sort(result)
	return result, nil
}

func canonicalCallerPermissions(ctx context.Context, tx db.DBTX, p *principal.Principal) ([]string, error) {
	canonical := make([]string, 0, len(p.Permissions))
	for _, permission := range p.Permissions {
		if strings.HasPrefix(permission, "unkey:") {
			if _, _, err := parseCanonicalPermission(permission, p.AuthorizedWorkspaceID); err == nil {
				canonical = append(canonical, permission)
			}
			continue
		}

		translated, err := translateLegacyPermission(ctx, tx, p.AuthorizedWorkspaceID, permission)
		if err == nil {
			canonical = append(canonical, translated)
			continue
		}
		if !errors.Is(err, migration.ErrUnsupported) && !errors.Is(err, migration.ErrInvalidScope) {
			return nil, err
		}
	}
	return canonical, nil
}

func translateLegacyPermission(ctx context.Context, tx db.DBTX, workspaceID, permission string) (string, error) {
	scope, err := migrationScope(ctx, tx, workspaceID, permission)
	if err != nil {
		return "", err
	}
	return migration.Translate(permission, scope)
}

func migrationScope(ctx context.Context, tx db.DBTX, workspaceID, permission string) (migration.Scope, error) {
	scope := migration.Scope{
		WorkspaceID:  workspaceID,
		APIs:         nil,
		Namespaces:   nil,
		Apps:         nil,
		Environments: nil,
		Identities:   nil,
	}
	parts := strings.Split(permission, ".")
	if len(parts) != 3 || parts[1] == "*" {
		return scope, nil
	}

	id := parts[1]
	switch parts[0] {
	case "api":
		api, err := db.Query.FindApiByID(ctx, tx, id)
		if err != nil {
			return scope, migrationResourceError(err)
		}
		if api.WorkspaceID != workspaceID || api.DeletedAtM.Valid || !api.KeyAuthID.Valid {
			return scope, migration.ErrInvalidScope
		}
		keyspace, err := db.Query.FindKeySpaceByID(ctx, tx, api.KeyAuthID.String)
		if err != nil {
			return scope, migrationResourceError(err)
		}
		if keyspace.WorkspaceID != workspaceID || keyspace.DeletedAtM.Valid {
			return scope, migration.ErrInvalidScope
		}
		if err := validateProject(ctx, tx, workspaceID, keyspace.ProjectID); err != nil {
			return scope, err
		}
		scope.APIs = map[string]urn.V1{id: {WorkspaceID: workspaceID, Resource: "projects/" + keyspace.ProjectID + "/keyspaces/" + keyspace.ID}}
	case "ratelimit":
		namespace, err := db.Query.FindRatelimitNamespace(ctx, tx, db.FindRatelimitNamespaceParams{WorkspaceID: workspaceID, Namespace: id})
		if err != nil {
			return scope, migrationResourceError(err)
		}
		if namespace.ID != id || namespace.WorkspaceID != workspaceID || namespace.DeletedAtM.Valid {
			return scope, migration.ErrInvalidScope
		}
		if err := validateProject(ctx, tx, workspaceID, namespace.ProjectID); err != nil {
			return scope, err
		}
		scope.Namespaces = map[string]urn.V1{id: {WorkspaceID: workspaceID, Resource: "projects/" + namespace.ProjectID + "/ratelimits/namespaces/" + id}}
	case "project":
		if err := validateProject(ctx, tx, workspaceID, id); err != nil {
			return scope, err
		}
	case "app":
		app, err := db.Query.FindAppById(ctx, tx, id)
		if err != nil {
			return scope, migrationResourceError(err)
		}
		if app.WorkspaceID != workspaceID {
			return scope, migration.ErrInvalidScope
		}
		if err := validateProject(ctx, tx, workspaceID, app.ProjectID); err != nil {
			return scope, err
		}
		scope.Apps = map[string]urn.V1{id: {WorkspaceID: workspaceID, Resource: "projects/" + app.ProjectID + "/apps/" + id}}
	case "environment":
		environment, err := db.Query.FindEnvironmentById(ctx, tx, id)
		if err != nil {
			return scope, migrationResourceError(err)
		}
		if environment.WorkspaceID != workspaceID {
			return scope, migration.ErrInvalidScope
		}
		app, err := db.Query.FindAppById(ctx, tx, environment.AppID)
		if err != nil {
			return scope, migrationResourceError(err)
		}
		if app.WorkspaceID != workspaceID || app.ProjectID != environment.ProjectID {
			return scope, migration.ErrInvalidScope
		}
		if err := validateProject(ctx, tx, workspaceID, environment.ProjectID); err != nil {
			return scope, err
		}
		scope.Environments = map[string]urn.V1{id: {WorkspaceID: workspaceID, Resource: "projects/" + environment.ProjectID + "/apps/" + app.ID + "/environments/" + id}}
	case "identity":
		identity, err := db.Query.FindIdentityByID(ctx, tx, db.FindIdentityByIDParams{WorkspaceID: workspaceID, IdentityID: id, Deleted: false})
		if err != nil {
			return scope, migrationResourceError(err)
		}
		if identity.WorkspaceID != workspaceID || identity.Deleted {
			return scope, migration.ErrInvalidScope
		}
		if err := validateProject(ctx, tx, workspaceID, identity.ProjectID); err != nil {
			return scope, err
		}
		scope.Identities = map[string]urn.V1{id: {WorkspaceID: workspaceID, Resource: "projects/" + identity.ProjectID + "/identities/" + id}}
	}
	return scope, nil
}

func validateProject(ctx context.Context, tx db.DBTX, workspaceID, projectID string) error {
	project, err := db.Query.FindProjectById(ctx, tx, projectID)
	if err != nil {
		return migrationResourceError(err)
	}
	if project.WorkspaceID != workspaceID {
		return migration.ErrInvalidScope
	}
	return nil
}

func migrationResourceError(err error) error {
	if db.IsNotFound(err) {
		return migration.ErrInvalidScope
	}
	return err
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
	return fault.New("invalid canonical or legacy permission",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public("A requested permission is not supported in this workspace."))
}
