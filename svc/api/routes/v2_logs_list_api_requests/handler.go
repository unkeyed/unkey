package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
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
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/internal/requestlogs"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2LogsListApiRequestsRequestBody
	Response = openapi.V2LogsListApiRequestsResponseBody
)

const limitDefault = 50

type Handler struct {
	ClickHouse  clickhouse.ClickHouse
	DB          db.Database
	LimitsCache cache.Cache[string, keysdb.Limit]
	Clock       clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/logs.listApiRequests" }

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

	err = principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).ApiRequestLogs(), permissions.Read))
	if err != nil {
		return err
	}

	p := pagination.Parse(req.Limit, req.Cursor, limitDefault)
	var pageCursor *clickhouse.ApiRequestCursor
	if p.Cursor != "" {
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(p.Cursor)
		timeText, requestID, _ := strings.Cut(string(decoded), ":")
		cursorTime, parseErr := strconv.ParseInt(timeText, 10, 64)
		if decodeErr != nil || parseErr != nil || cursorTime <= 0 || requestID == "" {
			return fault.New("invalid cursor",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("cursor is not a base64 encoded time and request id"),
				fault.Public("The cursor is not valid. Send the `pagination.cursor` of the previous response, or omit it to get the first page."),
			)
		}
		pageCursor = &clickhouse.ApiRequestCursor{
			Time:      cursorTime,
			RequestID: requestID,
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
		retention := time.Duration(limits.LogsRetentionDaysMax) * 24 * time.Hour
		retentionStart = h.Clock.Now().Add(-retention).UnixMilli()
	}

	filters := ptr.SafeDeref(req.Filters)
	rows, err := h.ClickHouse.ListApiRequests(ctx, clickhouse.ListApiRequestsRequest{
		WorkspaceID:    principal.AuthorizedWorkspaceID,
		RetentionStart: retentionStart,
		Cursor:         pageCursor,
		Limit:          int(p.FetchLimit()),
		Filters: clickhouse.ApiRequestFilters{
			Time:      requestlogs.Time(filters.Time),
			Path:      requestlogs.Text(filters.Path),
			Host:      requestlogs.Exact(filters.Host),
			Method:    requestlogs.Method(filters.Method),
			RequestID: requestlogs.Exact(filters.RequestId),
			Status:    requestlogs.Int(filters.Status),
		},
	})
	if err != nil {
		// A fresh fault, because the read limit's own public message tells the
		// caller to simplify a query they did not write
		if readLimitCode, isReadLimit := fault.GetCode(err); isReadLimit {
			return fault.New("api request listing exceeds a read limit",
				fault.Code(readLimitCode),
				fault.Internal(err.Error()),
				fault.Public("The query exceeds a read limit. Send a lower `limit`, a narrower `filters.time` window, or more filters, then try again."),
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

	rows, pg := pagination.Paginate(rows, p, func(r clickhouse.ApiRequestLog) string {
		return base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "%d:%s", r.Time, r.RequestID))
	})

	data := make([]openapi.ApiRequestLog, len(rows))
	for i, r := range rows {
		data[i] = openapi.ApiRequestLog{
			RequestId:       r.RequestID,
			Time:            r.Time,
			Host:            r.Host,
			Method:          r.Method,
			Path:            r.Path,
			Status:          r.ResponseStatus,
			LatencyMs:       r.ServiceLatency,
			RequestHeaders:  r.RequestHeaders,
			RequestBody:     r.RequestBody,
			ResponseHeaders: r.ResponseHeaders,
			ResponseBody:    r.ResponseBody,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       data,
		Pagination: pg,
	})
}
