package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2LogsListApiRequestsRequestBody
	Response = openapi.V2LogsListApiRequestsResponseBody
)

const (
	millisPerDay = 24 * 60 * 60 * 1000

	limitDefault = 50
)

type Handler struct {
	ClickHouse  clickhouse.ClickHouse
	DB          db.Database
	LimitsCache cache.Cache[string, keysdb.Limit]
	Clock       clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/logs.listApiRequests" }

type cursor struct {
	Time      int64  `json:"t"`
	RequestID string `json:"r"`
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// A logged response would copy up to a page of other requests into a new
	// row, and a polling dashboard would fill the listing with its own reads
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

	var after *clickhouse.ApiRequestCursor
	if req.Cursor != nil {
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(*req.Cursor)
		var c cursor
		if decodeErr != nil || json.Unmarshal(decoded, &c) != nil || c.Time <= 0 || c.RequestID == "" {
			return fault.New("invalid cursor",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("cursor is not a base64 encoded time and request id"),
				fault.Public("The cursor is not valid. Send the `pagination.cursor` of the previous response, or omit it to get the first page."),
			)
		}
		after = &clickhouse.ApiRequestCursor{
			Time:      c.Time,
			RequestID: c.RequestID,
		}
	}

	limits, hit, err := h.LimitsCache.SWR(ctx, principal.AuthorizedWorkspaceID, func(ctx context.Context) (keysdb.Limit, error) {
		return keysdb.Query.FindLimitsByWorkspaceID(ctx, h.DB.RO(), principal.AuthorizedWorkspaceID)
	}, caches.DefaultFindFirstOp)
	if db.IsNotFound(err) || hit == cache.Null {
		return domaingate.LimitsNotConfigured(principal.AuthorizedWorkspaceID)
	}
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load workspace limits"),
			fault.Public("Failed to load the log retention of the workspace. Try again."),
		)
	}

	var retentionStart int64
	if limits.LogsRetentionDaysMax > 0 {
		retentionStart = h.Clock.Now().UnixMilli() - int64(limits.LogsRetentionDaysMax)*millisPerDay
	}

	filters := ptr.SafeDeref(req.Filters)
	timeFilter := ptr.SafeDeref(filters.Time)
	path := ptr.SafeDeref(filters.Path)
	host := ptr.SafeDeref(filters.Host)
	method := ptr.SafeDeref(filters.Method)
	requestID := ptr.SafeDeref(filters.RequestId)
	status := ptr.SafeDeref(filters.Status)
	limit := ptr.PositiveOr(req.Limit, limitDefault)

	page, err := h.ClickHouse.ListApiRequests(ctx, clickhouse.ListApiRequestsRequest{
		WorkspaceID:    principal.AuthorizedWorkspaceID,
		RetentionStart: retentionStart,
		After:          after,
		Limit:          limit,
		Filters: clickhouse.ApiRequestFilters{
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
				Is:    methodStrings(ptr.SafeDeref(method.Is)),
				IsNot: methodStrings(ptr.SafeDeref(method.IsNot)),
			},
			RequestID: logfilter.Exact{
				Is:    ptr.SafeDeref(requestID.Is),
				IsNot: ptr.SafeDeref(requestID.IsNot),
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
		// A fresh fault, because the read limit's own public message tells the
		// caller to simplify a query they did not write
		if code, coded := fault.GetCode(err); coded {
			return fault.New("api request listing exceeds a read limit",
				fault.Code(code),
				fault.Internal(err.Error()),
				fault.Public("The query exceeds a read limit. Send a narrower `filters.time` window or more filters, then try again."),
			)
		}
		if _, isException := errors.AsType[*ch.Exception](err); isException {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("clickhouse rejected the api request listing"),
				fault.Public("An internal error occurred."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.Data.Analytics.ConnectionFailed.URN()),
			fault.Internal("failed to list api requests"),
			fault.Public("The log store is not available. Try again."),
		)
	}

	data := make([]openapi.ApiRequestLog, len(page.Requests))
	for i, r := range page.Requests {
		data[i] = openapi.ApiRequestLog{
			RequestId:       r.RequestID,
			Time:            r.Time,
			Host:            r.Host,
			Method:          r.Method,
			Path:            r.Path,
			Status:          r.ResponseStatus,
			LatencyMs:       r.ServiceLatency,
			Error:           r.Error,
			RequestHeaders:  r.RequestHeaders,
			RequestBody:     r.RequestBody,
			ResponseHeaders: r.ResponseHeaders,
			ResponseBody:    r.ResponseBody,
		}
	}

	var nextCursor *string
	if page.HasMore {
		last := page.Requests[len(page.Requests)-1]
		encoded, err := json.Marshal(cursor{
			Time:      last.Time,
			RequestID: last.RequestID,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("failed to encode cursor"),
				fault.Public("An internal error occurred."),
			)
		}
		nextCursor = new(base64.RawURLEncoding.EncodeToString(encoded))
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
		Pagination: openapi.Pagination{
			Cursor:  nextCursor,
			HasMore: page.HasMore,
		},
	})
}

func methodStrings(methods []openapi.LogHttpMethod) []string {
	values := make([]string, len(methods))
	for i, m := range methods {
		values[i] = string(m)
	}

	return values
}
