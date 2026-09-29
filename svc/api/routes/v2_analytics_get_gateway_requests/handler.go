package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AnalyticsGetGatewayRequestsRequestBody
	Response = openapi.V2AnalyticsGetGatewayRequestsResponseBody
)

var (
	// This alias is the only table name that the endpoint accepts. It refuses
	// the physical name. Thus you can change the frontline table name without a
	// change to the public API.
	tableAliases = map[string]string{
		"gateway_requests_v1": "default.frontline_requests_raw_v1",
	}

	allowedTables = []string{
		"default.frontline_requests_raw_v1",
	}
)

type Handler struct {
	DB                         db.Database
	AnalyticsConnectionManager analytics.ConnectionManager
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/analytics.getGatewayRequests" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	wildcard := rbac.Tuple{ResourceType: rbac.Project, ResourceID: "*", Action: rbac.ReadGatewayRequests}
	securityScopes, authorized, err := h.gatewaySecurityScopes(ctx, p.AuthorizedWorkspaceID, p.Permissions)
	if err != nil {
		return err
	}
	if !authorized {
		return p.Authorize(rbac.Or(
			rbac.T(wildcard),
			rbac.U(gatewayLogsURN(p.AuthorizedWorkspaceID, "project", "app", "environment"), permissions.Read),
		))
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
