package handler

import (
	"context"
	"net/http"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/requestlogs"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2LogsGetApiRequestMetricsRequestBody
	Response = openapi.V2LogsGetApiRequestMetricsResponseBody
)

type Handler struct {
	ClickHouse  clickhouse.ClickHouse
	DB          db.Database
	LimitsCache cache.Cache[string, keysdb.Limit]
	Clock       clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/logs.getApiRequestMetrics" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// The dashboard would fill the request logs with its own reads
	s.SkipRequestLog()

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).ApiRequestLogs(), permissions.Read))
	if err != nil {
		return err
	}

	retentionStart, err := requestlogs.RetentionStart(ctx, h.LimitsCache, h.DB, h.Clock, principal.AuthorizedWorkspaceID)
	if err != nil {
		return err
	}

	filters := ptr.SafeDeref(req.Filters)
	series, err := h.ClickHouse.GetApiRequestMetrics(ctx, clickhouse.ApiRequestMetricsRequest{
		WorkspaceID:    principal.AuthorizedWorkspaceID,
		Now:            h.Clock.Now().UnixMilli(),
		RetentionStart: retentionStart,
		Filters: clickhouse.ApiRequestMetricsFilters{
			Time:   requestlogs.Time(filters.Time),
			Path:   requestlogs.Text(filters.Path),
			Method: requestlogs.Method(filters.Method),
			Status: requestlogs.Int(filters.Status),
		},
	})
	if err != nil {
		return requestlogs.ReadError(err, "failed to read the api request metrics",
			"The query exceeds a read limit. Send a narrower `filters.time` window or more filters, then try again.",
		)
	}

	points := make([]openapi.ApiRequestMetricsPoint, len(series))
	for i, p := range series {
		points[i] = openapi.ApiRequestMetricsPoint{
			Time:        p.Time,
			Success:     p.Success,
			ClientError: p.ClientError,
			ServerError: p.ServerError,
			Total:       p.Total,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: points,
	})
}
