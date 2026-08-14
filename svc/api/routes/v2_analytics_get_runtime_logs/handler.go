package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
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

	wildcard := rbac.Tuple{ResourceType: rbac.Project, ResourceID: "*", Action: rbac.ReadRuntimeLogs}
	if !slices.Contains(p.Permissions, wildcard.String()) {
		return p.Authorize(rbac.T(wildcard))
	}

	rows, err := analytics.Execute(ctx, h.AnalyticsConnectionManager, analytics.ExecuteRequest{
		Query:           req.Query,
		WorkspaceID:     p.WorkspaceID,
		TableAliases:    tableAliases,
		AllowedTables:   allowedTables,
		SecurityFilters: nil,
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
