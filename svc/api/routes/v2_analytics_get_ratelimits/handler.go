package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2AnalyticsGetRatelimitsRequestBody
type Response = openapi.V2AnalyticsGetRatelimitsResponseBody

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
	// FilterBuilder is optional because namespace policy supplies filters after parsing.
	rows, err := analytics.Execute(ctx, h.AnalyticsConnectionManager, analytics.ExecuteRequest{ //nolint:exhaustruct
		WorkspaceID:  p.WorkspaceID,
		Query:        req.Query,
		ParserConfig: rateLimitParserConfig(),
		Policy: func(parser *queryparser.Parser) ([]queryparser.SecurityFilter, error) {
			return h.authorize(ctx, p, parser.ExtractColumn("namespace_id"))
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

// rateLimitParserConfig builds aliases and allowed tables from one stable list.
// Returning fresh collections prevents one request from mutating another's policy.
func rateLimitParserConfig() queryparser.Config {
	tables := [...]struct {
		alias string
		name  string
	}{
		{alias: "ratelimits_v1", name: "default.ratelimits_raw_v2"},
		{alias: "ratelimits_per_minute_v1", name: "default.ratelimits_per_minute_v2"},
		{alias: "ratelimits_per_hour_v1", name: "default.ratelimits_per_hour_v2"},
		{alias: "ratelimits_per_day_v1", name: "default.ratelimits_per_day_v2"},
		{alias: "ratelimits_per_month_v1", name: "default.ratelimits_per_month_v2"},
	}
	// The executor supplies workspace-specific limits and security fields.
	config := queryparser.Config{ //nolint:exhaustruct
		TableAliases:           make(map[string]string, len(tables)),
		AllowedTables:          make([]string, 0, len(tables)),
		PublicTableAliasesOnly: true,
		DisallowJoins:          true,
	}
	for _, table := range tables {
		config.TableAliases[table.alias] = table.name
		config.AllowedTables = append(config.AllowedTables, table.name)
	}
	return config
}

func (h *Handler) authorize(ctx context.Context, p *principal.Principal, namespaceIDs []string) ([]queryparser.SecurityFilter, error) {
	if len(namespaceIDs) == 0 || len(namespaceIDs) > 10 {
		return nil, fault.New("rate limit analytics requires between one and ten namespace ids",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()), fault.Public("Invalid analytics query"))
	}
	namespaceIDsFound, err := db.Query.FindRatelimitNamespacesByIDs(ctx, h.DB.RO(), db.FindRatelimitNamespacesByIDsParams{
		WorkspaceID:  p.WorkspaceID,
		NamespaceIds: namespaceIDs,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to resolve ratelimit namespaces"))
	}
	found := make(map[string]struct{}, len(namespaceIDsFound))
	for _, namespaceID := range namespaceIDsFound {
		found[namespaceID] = struct{}{}
	}
	checks := make([]rbac.PermissionQuery, 0, len(namespaceIDs))
	for _, namespaceID := range namespaceIDs {
		if _, ok := found[namespaceID]; !ok {
			return nil, namespaceNotFound(namespaceID)
		}
		checks = append(checks, rbac.T(rbac.Tuple{ResourceType: rbac.Ratelimit, ResourceID: namespaceID, Action: rbac.ReadAnalytics}))
	}
	wildcard := rbac.T(rbac.Tuple{ResourceType: rbac.Ratelimit, ResourceID: "*", Action: rbac.ReadAnalytics})
	if err := p.Authorize(rbac.Or(wildcard, rbac.And(checks...))); err != nil {
		return nil, err
	}
	// Parenthesized injection guarantees an OR in caller SQL cannot escape this scope.
	return []queryparser.SecurityFilter{{Column: "namespace_id", AllowedValues: namespaceIDs}}, nil
}

func namespaceNotFound(namespaceID string) error {
	return fault.New("ratelimit namespace not found",
		fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
		fault.Public(fmt.Sprintf("Namespace '%s' was not found.", namespaceID)))
}
