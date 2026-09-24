package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2AnalyticsGetRatelimitsRequestBody
type Response = openapi.V2AnalyticsGetRatelimitsResponseBody

var (
	tableAliases = map[string]string{
		"ratelimits_v1":            "default.ratelimits_raw_v2",
		"ratelimits_per_minute_v1": "default.ratelimits_per_minute_v2",
		"ratelimits_per_hour_v1":   "default.ratelimits_per_hour_v2",
		"ratelimits_per_day_v1":    "default.ratelimits_per_day_v2",
		"ratelimits_per_month_v1":  "default.ratelimits_per_month_v2",
	}

	allowedTables = []string{
		"default.ratelimits_raw_v2",
		"default.ratelimits_per_minute_v2",
		"default.ratelimits_per_hour_v2",
		"default.ratelimits_per_day_v2",
		"default.ratelimits_per_month_v2",
	}
)

type Handler struct {
	DB                         db.Database
	AnalyticsConnectionManager analytics.ConnectionManager
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/analytics.getRatelimits" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	wildcard := rbac.T(rbac.Tuple{ResourceType: rbac.Ratelimit, ResourceID: "*", Action: rbac.ReadAnalytics})
	hasLegacyWildcard := slices.Contains(p.Permissions, "ratelimit.*.read_analytics")
	allowedNamespaceIDs := extractAllowedNamespaceIDs(p.Permissions)
	logPermissions, hasWorkspaceWidePermission := extractLogPermissions(p.Permissions, p.AuthorizedWorkspaceID)
	if !hasLegacyWildcard && len(allowedNamespaceIDs) == 0 && len(logPermissions) == 0 {
		return p.Authorize(wildcard)
	}
	if !hasLegacyWildcard && !hasWorkspaceWidePermission && len(logPermissions) > 0 {
		namespaceRows, queryErr := db.Query.ListRatelimitNamespaceOwnershipByWorkspace(ctx, h.DB.RO(), p.AuthorizedWorkspaceID)
		if queryErr != nil {
			return fault.Wrap(queryErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Public("An unexpected error occurred while loading your rate limit namespaces."),
			)
		}
		allowedNamespaceIDs = append(allowedNamespaceIDs, authorizedNamespaceIDs(namespaceRows, logPermissions, p.AuthorizedWorkspaceID)...)
	}
	securityFilters := make([]queryparser.SecurityFilter, 0, 1)
	if !hasLegacyWildcard && !hasWorkspaceWidePermission {
		securityFilters = append(securityFilters, queryparser.SecurityFilter{Column: "namespace_id", AllowedValues: allowedNamespaceIDs})
	}
	rows, err := analytics.Execute(ctx, h.AnalyticsConnectionManager, analytics.ExecuteRequest{
		Query:           req.Query,
		WorkspaceID:     p.AuthorizedWorkspaceID,
		TableAliases:    tableAliases,
		AllowedTables:   allowedTables,
		SecurityFilters: securityFilters,
		SecurityScopes:  nil,
	})
	if err != nil {
		return err
	}
	responseBytes, err := json.Marshal(Response{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: rows})
	if err != nil {
		return fault.Wrap(err, fault.Public("Failed to encode query results"))
	}
	if len(responseBytes) > clickhouse.AnalyticsResultBytesMax {
		return fault.New("analytics response byte limit exceeded",
			fault.Code(codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN()),
			fault.Public("Query result exceeds the maximum response size."),
		)
	}
	s.AddHeader("Content-Type", "application/json")
	return s.Send(http.StatusOK, responseBytes)
}

func extractAllowedNamespaceIDs(permissions []string) []string {
	namespaceIDs := make([]string, 0)
	for _, permission := range permissions {
		pattern := strings.Split(permission, ".")
		if len(pattern) != 3 || pattern[0] != "ratelimit" || pattern[2] != "read_analytics" {
			continue
		}
		namespaceIDs = append(namespaceIDs, pattern[1])
	}
	return namespaceIDs
}

func extractLogPermissions(permissionsToCheck []string, workspaceID string) ([]urn.V1, bool) {
	permissions := make([]urn.V1, 0)
	workspaceLogs := ratelimitLogResource(workspaceID, "*", "*")
	workspaceWide := false
	for _, permission := range permissionsToCheck {
		resourceValue, action, ok := strings.Cut(permission, "#")
		if !ok || strings.Contains(action, "#") {
			continue
		}
		resource, err := urn.ParseV1(resourceValue)
		if err != nil || resource.WorkspaceID != workspaceID || !isLogReadAction(resource, action) || !canCoverRatelimitLogs(resource) {
			continue
		}
		permissions = append(permissions, resource)
		workspaceWide = workspaceWide || resource.Covers(workspaceLogs)
	}
	return permissions, workspaceWide
}

func isLogReadAction(resource urn.V1, action string) bool {
	return action == permissions.Read.String() || action == permissions.Wildcard && resource.Resource == "**"
}

func canCoverRatelimitLogs(resource urn.V1) bool {
	segments := strings.Split(resource.Resource, "/")
	projectID, namespaceID := "project", "namespace"
	if len(segments) > 1 && segments[0] == "projects" && segments[1] != "*" {
		projectID = segments[1]
	}
	if len(segments) > 4 && segments[3] == "namespaces" && segments[4] != "*" {
		namespaceID = segments[4]
	}
	return resource.Covers(ratelimitLogResource(resource.WorkspaceID, projectID, namespaceID))
}

func authorizedNamespaceIDs(rows []db.ListRatelimitNamespaceOwnershipByWorkspaceRow, permissions []urn.V1, workspaceID string) []string {
	allowed := make([]string, 0, len(rows))
	for _, row := range rows {
		target := ratelimitLogResource(workspaceID, row.ProjectID, row.ID)
		if slices.ContainsFunc(permissions, func(permission urn.V1) bool { return permission.Covers(target) }) {
			allowed = append(allowed, row.ID)
		}
	}
	return allowed
}

func ratelimitLogResource(workspaceID, projectID, namespaceID string) urn.V1 {
	return urn.V1{
		WorkspaceID: workspaceID,
		Resource:    "projects/" + projectID + "/ratelimits/namespaces/" + namespaceID + "/logs",
	}
}
