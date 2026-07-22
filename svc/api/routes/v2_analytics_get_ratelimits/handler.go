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
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2AnalyticsGetRatelimitsRequestBody
type Response = openapi.V2AnalyticsGetRatelimitsResponseBody

var (
	ratelimitTableAliases = map[string]string{
		"ratelimits_v1":            "default.ratelimits_raw_v2",
		"ratelimits_per_minute_v1": "default.ratelimits_per_minute_v2",
		"ratelimits_per_hour_v1":   "default.ratelimits_per_hour_v2",
		"ratelimits_per_day_v1":    "default.ratelimits_per_day_v2",
		"ratelimits_per_month_v1":  "default.ratelimits_per_month_v2",
	}

	ratelimitAllowedTables = []string{
		"default.ratelimits_raw_v2",
		"default.ratelimits_per_minute_v2",
		"default.ratelimits_per_hour_v2",
		"default.ratelimits_per_day_v2",
		"default.ratelimits_per_month_v2",
	}
)

type Handler struct {
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
	hasWildcard := slices.Contains(p.Permissions, "ratelimit.*.read_analytics")
	allowedNamespaceIDs := extractAllowedNamespaceIDs(p.Permissions)
	if !hasWildcard && len(allowedNamespaceIDs) == 0 {
		return p.Authorize(wildcard)
	}
	securityFilters := make([]queryparser.SecurityFilter, 0, 1)
	if !hasWildcard {
		securityFilters = append(securityFilters, queryparser.SecurityFilter{Column: "namespace_id", AllowedValues: allowedNamespaceIDs})
	}
	rows, err := analytics.Execute(ctx, h.AnalyticsConnectionManager, analytics.ExecuteRequest{
		Query:                  req.Query,
		WorkspaceID:            p.WorkspaceID,
		TableAliases:           ratelimitTableAliases,
		AllowedTables:          ratelimitAllowedTables,
		InitialSecurityFilters: securityFilters,
		Policy: func(parser *queryparser.Parser) ([]queryparser.SecurityFilter, error) {
			if hasWildcard {
				return nil, nil
			}
			namespaceIDs := parser.ExtractColumn("namespace_id")
			if len(namespaceIDs) == 0 {
				return nil, p.Authorize(wildcard)
			}
			checks := make([]rbac.PermissionQuery, 0, len(namespaceIDs))
			for _, namespaceID := range namespaceIDs {
				checks = append(checks, rbac.T(rbac.Tuple{ResourceType: rbac.Ratelimit, ResourceID: namespaceID, Action: rbac.ReadAnalytics}))
			}
			return nil, p.Authorize(rbac.And(checks...))
		},
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
