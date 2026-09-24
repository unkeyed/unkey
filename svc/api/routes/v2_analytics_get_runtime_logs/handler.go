package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AnalyticsGetRuntimeLogsRequestBody
	Response = openapi.V2AnalyticsGetRuntimeLogsResponseBody
)

var (
	// This alias is the only table name that the endpoint accepts. It refuses
	// the physical name. Thus you can change the runtime logs table name without
	// a change to the public API.
	tableAliases = map[string]string{
		"runtime_logs_v1": "default.runtime_logs_raw_v1",
	}

	allowedTables = []string{
		"default.runtime_logs_raw_v1",
	}
)

type Handler struct {
	AnalyticsConnectionManager analytics.ConnectionManager
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/analytics.getRuntimeLogs" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	securityScopes, authorized := runtimeLogSecurityScopes(p.AuthorizedWorkspaceID, p.Permissions)
	if !authorized {
		return p.Authorize(rbac.T(rbac.Tuple{ResourceType: rbac.Project, ResourceID: "*", Action: rbac.ReadRuntimeLogs}))
	}

	rows, err := analytics.Execute(ctx, h.AnalyticsConnectionManager, analytics.ExecuteRequest{
		Query:           req.Query,
		WorkspaceID:     p.AuthorizedWorkspaceID,
		TableAliases:    tableAliases,
		AllowedTables:   allowedTables,
		SecurityFilters: nil,
		SecurityScopes:  securityScopes,
	})
	if err != nil {
		return err
	}

	responseBytes, err := json.Marshal(Response{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: rows})
	if err != nil {
		return fault.Wrap(err, fault.Public("Failed to encode query results"))
	}

	if len(responseBytes) > clickhouse.AnalyticsResultBytesMax {
		return fault.New(
			"analytics response byte limit exceeded",
			fault.Code(codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN()),
			fault.Public("Query result exceeds the maximum response size."),
		)
	}

	s.AddHeader("Content-Type", "application/json")
	return s.Send(http.StatusOK, responseBytes)
}

// runtimeLogSecurityScopes converts permissions that cover runtime logs into row
// scopes. It returns nil only when one permission covers every log in the workspace.
func runtimeLogSecurityScopes(workspaceID string, permissionsToCheck []string) ([]queryparser.SecurityScope, bool) {
	securityScopes := make([]queryparser.SecurityScope, 0)
	for _, permission := range permissionsToCheck {
		if permission == "*" || permission == "project.*.read_runtime_logs" {
			return nil, true
		}

		resourceName, action, ok := strings.Cut(permission, "#")
		if !ok {
			continue
		}
		resource, err := urn.ParseV1(resourceName)
		if err != nil || resource.WorkspaceID != workspaceID {
			continue
		}
		if resource.Resource == "**" {
			if action == "read" || action == "*" {
				return nil, true
			}
			continue
		}
		if action != "read" {
			continue
		}

		ancestry, ok := runtimeLogAncestry(resourceName)
		if !ok {
			continue
		}
		filters := make([]queryparser.SecurityFilter, 0, 4)
		columns := []string{"project_id", "app_id", "environment_id", "deployment_id"}
		for i, value := range ancestry {
			if value != "*" {
				filters = append(filters, queryparser.SecurityFilter{Column: columns[i], AllowedValues: []string{value}})
			}
		}
		if len(filters) == 0 {
			return nil, true
		}
		securityScopes = append(securityScopes, queryparser.SecurityScope{Filters: filters})
	}

	return securityScopes, len(securityScopes) > 0
}

// runtimeLogAncestry returns ordered project, app, environment, and deployment
// IDs covered by a log URN or subtree pattern. For example, projects/p/**
// returns only p; projects/p#read is not a log resource.
func runtimeLogAncestry(resource string) ([]string, bool) {
	base, descendants := strings.CutSuffix(resource, "/**")
	if logs, err := urn.ParseDeploymentLogs(base); err == nil {
		return []string{logs.ProjectID, logs.AppID, logs.EnvironmentID, logs.DeploymentID}, true
	}
	if !descendants {
		return nil, false
	}
	if deployment, err := urn.ParseDeployment(base); err == nil {
		return []string{deployment.ProjectID, deployment.AppID, deployment.EnvironmentID, deployment.DeploymentID}, true
	}
	if environment, err := urn.ParseEnvironment(base); err == nil {
		return []string{environment.ProjectID, environment.AppID, environment.EnvironmentID}, true
	}
	if app, err := urn.ParseApp(base); err == nil {
		return []string{app.ProjectID, app.AppID}, true
	}
	if project, err := urn.ParseProject(base); err == nil {
		return []string{project.ProjectID}, true
	}
	return nil, false
}
