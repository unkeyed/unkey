package handler

import (
	"context"
	"net/http"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
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
	Request  = openapi.V2LogsGetApiRequestsTimeseriesRequestBody
	Response = openapi.V2LogsGetApiRequestsTimeseriesResponseBody
)

type Handler struct {
	ClickHouse  clickhouse.ClickHouse
	DB          db.Database
	LimitsCache cache.Cache[string, keysdb.Limit]
	Clock       clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/logs.getApiRequestsTimeseries" }

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

	err = principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).RequestLogs(), permissions.Read))
	if err != nil {
		return err
	}

	retentionStart, err := requestlogs.RetentionStart(ctx, h.LimitsCache, h.DB, h.Clock, principal.AuthorizedWorkspaceID)
	if err != nil {
		return err
	}

	filters := ptr.SafeDeref(req.Filters)
	timeFilter := ptr.SafeDeref(filters.Time)
	path := ptr.SafeDeref(filters.Path)
	host := ptr.SafeDeref(filters.Host)
	method := ptr.SafeDeref(filters.Method)
	status := ptr.SafeDeref(filters.Status)

	series, err := h.ClickHouse.GetApiRequestsTimeseries(ctx, clickhouse.ApiRequestsTimeseriesRequest{
		WorkspaceID:    principal.AuthorizedWorkspaceID,
		Now:            h.Clock.Now().UnixMilli(),
		RetentionStart: retentionStart,
		Filters: clickhouse.ApiRequestsTimeseriesFilters{
			Time: logfilter.Int{
				Is:    nil,
				IsNot: nil,
				Gt:    timeFilter.Gt,
				Gte:   timeFilter.Gte,
				Lt:    timeFilter.Lt,
				Lte:   timeFilter.Lte,
			},
			Path: logfilter.Text{
				Is:          ptr.SafeDeref(path.Is),
				IsNot:       ptr.SafeDeref(path.IsNot),
				Contains:    ptr.SafeDeref(path.Contains),
				NotContains: ptr.SafeDeref(path.NotContains),
				StartsWith:  ptr.SafeDeref(path.StartsWith),
				EndsWith:    ptr.SafeDeref(path.EndsWith),
			},
			Host: logfilter.Exact{
				Is:    ptr.SafeDeref(host.Is),
				IsNot: ptr.SafeDeref(host.IsNot),
			},
			Method: logfilter.Exact{
				Is:    requestlogs.Methods(ptr.SafeDeref(method.Is)),
				IsNot: requestlogs.Methods(ptr.SafeDeref(method.IsNot)),
			},
			Status: logfilter.Int{
				Is:    ptr.SafeDeref(status.Is),
				IsNot: ptr.SafeDeref(status.IsNot),
				Gt:    status.Gt,
				Gte:   status.Gte,
				Lt:    status.Lt,
				Lte:   status.Lte,
			},
		},
	})
	if err != nil {
		return requestlogs.ReadError(err, "failed to read the api requests timeseries")
	}

	points := make([]openapi.ApiRequestsTimeseriesPoint, len(series.Points))
	for i, p := range series.Points {
		points[i] = openapi.ApiRequestsTimeseriesPoint{
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
		Data: openapi.V2LogsGetApiRequestsTimeseriesResponseData{
			BucketMs: series.BucketMs,
			Points:   points,
		},
	})
}
