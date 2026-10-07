package handler

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"

	chquery "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// legacyGatewayRequestsWildcard preserves the original workspace-wide permission.
const legacyGatewayRequestsWildcard = "project.*.read_gateway_requests"

// gatewayScope is one project, app, or environment branch of a permission union.
type gatewayScope struct {
	projectID     string
	appID         string
	environmentID string
}

// gatewaySecurityScopes resolves URN permissions into OR scopes. A nil result
// is unrestricted within the workspace, while a non-nil empty result denies all rows.
func (h *Handler) gatewaySecurityScopes(ctx context.Context, workspaceID string, permissionsToCheck []string) ([]chquery.SecurityScope, bool, error) {
	if slices.Contains(permissionsToCheck, legacyGatewayRequestsWildcard) {
		return nil, true, nil
	}

	securityScopes := make([]chquery.SecurityScope, 0)
	authorized := false
	for _, permission := range permissionsToCheck {
		scope, ok := parseGatewayScope(workspaceID, permission)
		if !ok {
			continue
		}
		authorized = true
		if scope.projectID == "*" {
			return nil, true, nil
		}

		valid, err := h.validateGatewayScope(ctx, workspaceID, scope)
		if err != nil {
			return nil, false, err
		}
		if !valid {
			continue
		}

		filters := []chquery.SecurityFilter{{Column: "project_id", AllowedValues: []string{scope.projectID}}}
		if scope.appID != "*" {
			filters = append(filters, chquery.SecurityFilter{Column: "app_id", AllowedValues: []string{scope.appID}})
		}
		if scope.environmentID != "*" {
			filters = append(filters, chquery.SecurityFilter{Column: "environment_id", AllowedValues: []string{scope.environmentID}})
		}
		securityScopes = append(securityScopes, chquery.SecurityScope{Filters: filters})
	}

	return securityScopes, authorized, nil
}

// parseGatewayScope accepts only URN read permissions that cover gateway logs
// in the authorized workspace.
func parseGatewayScope(workspaceID, permission string) (gatewayScope, bool) {
	var zero gatewayScope

	resourceValue, _, ok := strings.Cut(permission, "#")
	if !ok {
		return zero, false
	}
	resource, err := urn.ParseV1(resourceValue)
	if err != nil {
		return zero, false
	}

	if resource.Resource == "**" {
		target := gatewayLogsURN(workspaceID, "project", "app", "environment")
		return gatewayScope{projectID: "*", appID: "*", environmentID: "*"}, rbac.Check(rbac.U(target, permissions.Read), []string{permission}) == nil
	}

	base, descendants := strings.CutSuffix(resourceValue, "/**")
	scope := gatewayScope{
		projectID:     "*",
		appID:         "*",
		environmentID: "*",
	}
	if logs, err := urn.ParseGatewayLogs(base); err == nil {
		scope.projectID, scope.appID, scope.environmentID = logs.ProjectID, logs.AppID, logs.EnvironmentID
	} else if gateway, err := urn.ParseGateway(base); descendants && err == nil {
		scope.projectID, scope.appID, scope.environmentID = gateway.ProjectID, gateway.AppID, gateway.EnvironmentID
	} else if environment, err := urn.ParseEnvironment(base); descendants && err == nil {
		scope.projectID, scope.appID, scope.environmentID = environment.ProjectID, environment.AppID, environment.EnvironmentID
	} else if app, err := urn.ParseApp(base); descendants && err == nil {
		scope.projectID, scope.appID = app.ProjectID, app.AppID
	} else if project, err := urn.ParseProject(base); descendants && err == nil {
		scope.projectID = project.ProjectID
	} else {
		return zero, false
	}
	target := gatewayLogsURN(workspaceID, concreteID(scope.projectID, "project"), concreteID(scope.appID, "app"), concreteID(scope.environmentID, "environment"))
	if rbac.Check(rbac.U(target, permissions.Read), []string{permission}) != nil {
		return zero, false
	}
	return scope, true
}

// validateGatewayScope confirms that concrete IDs describe resources owned by
// the authorized workspace and by each preceding ancestor in the permission path.
func (h *Handler) validateGatewayScope(ctx context.Context, workspaceID string, scope gatewayScope) (bool, error) {
	if scope.environmentID != "*" {
		environment, err := db.Query.FindEnvironmentById(ctx, h.DB.RO(), scope.environmentID)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fault.Wrap(err, fault.Internal("failed to resolve gateway log environment"))
		}
		return environment.WorkspaceID == workspaceID && environment.ProjectID == scope.projectID && environment.AppID == scope.appID, nil
	}

	if scope.appID != "*" {
		app, err := db.Query.FindAppById(ctx, h.DB.RO(), scope.appID)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fault.Wrap(err, fault.Internal("failed to resolve gateway log app"))
		}
		return app.WorkspaceID == workspaceID && app.ProjectID == scope.projectID, nil
	}

	project, err := db.Query.FindProjectById(ctx, h.DB.RO(), scope.projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fault.Wrap(err, fault.Internal("failed to resolve gateway log project"))
	}
	return project.WorkspaceID == workspaceID, nil
}

// gatewayLogsURN builds one concrete gateway log resource for authorization.
func gatewayLogsURN(workspaceID, projectID, appID, environmentID string) urn.GatewayLogs {
	return urn.New().Workspace(workspaceID).Project(projectID).App(appID).Environment(environmentID).Gateway().Logs()
}

// concreteID substitutes a concrete value when a permission segment is a wildcard.
func concreteID(id, fallback string) string {
	if id == "*" {
		return fallback
	}
	return id
}
