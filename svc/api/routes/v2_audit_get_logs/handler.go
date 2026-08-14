package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AuditGetLogsRequestBody
	Response = openapi.V2AuditGetLogsResponseBody
)

const (
	// defaultLimit and maxLimit bound how many events one page returns. The
	// OpenAPI schema also declares these; we clamp defensively.
	defaultLimit = 100
	maxLimit     = 500

	// defaultBucket is the only audit-log stream today. Per-bucket selection is
	// deferred, so the bucket is fixed here rather than exposed as a param.
	defaultBucket = "unkey_mutations"

	// eventVersion is the schema version stamped on every returned event.
	eventVersion = "v1"

	// millisPerDay is the width of one retention day in unix milliseconds.
	millisPerDay = 24 * 60 * 60 * 1000
)

// Handler serves /v2/audit.getLogs: a stable, SIEM-friendly, cursor-paginated
// view of a workspace's audit logs. It reads the internal
// default.audit_logs_raw_v1 table via the shared platform ClickHouse
// connection (not the per-workspace analytics user), always scoping to the
// authenticated principal's workspace.
type Handler struct {
	ClickHouse  clickhouse.ClickHouse
	DB          db.Database
	LimitsCache cache.Cache[string, keysdb.Limit]
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/audit.getLogs" }

// auditCursor is the opaque keyset cursor. It advances on inserted_at (the
// drain-stamped visibility timestamp), NOT event time, so late-drained rows are
// never skipped by an advancing watermark.
type auditCursor struct {
	InsertedAt int64  `json:"ia"`
	EventID    string `json:"e"`
}

func encodeCursor(c auditCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (auditCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return auditCursor{}, err
	}
	var c auditCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return auditCursor{}, err
	}
	if c.EventID == "" {
		return auditCursor{}, errInvalidCursor
	}
	return c, nil
}

var errInvalidCursor = fault.New("invalid cursor",
	fault.Code(codes.App.Validation.InvalidInput.URN()),
	fault.Internal("cursor missing event id"),
	fault.Public("The provided cursor is invalid."),
)

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// The workspace scope is derived from the authenticated principal and is
	// the sole tenant-isolation boundary; it is never taken from the request.
	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Audit,
		ResourceID:   "*",
		Action:       rbac.ReadAuditLog,
	}))
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	limit := defaultLimit
	if req.Limit != nil {
		limit = *req.Limit
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	// Resolve the workspace's audit-log retention. This fails closed: if the
	// limit cannot be resolved we deny rather than expose the full ClickHouse
	// TTL window to a lower-tier workspace.
	limits, _, err := h.LimitsCache.SWR(ctx, principal.WorkspaceID, func(ctx context.Context) (keysdb.Limit, error) {
		return keysdb.Query.FindLimitsByWorkspaceID(ctx, h.DB.RO(), principal.WorkspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load workspace audit-log retention"),
			fault.Public("Failed to resolve audit-log retention for this workspace."),
		)
	}
	if limits.LogsAuditRetentionDaysMax == 0 {
		return fault.New("audit retention unresolved",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("workspace has no audit-log retention configured"),
			fault.Public("Audit-log retention is not configured for this workspace."),
		)
	}

	now := time.Now()
	// The window operates on inserted_at (visibility time), matching the cursor.
	retentionFloorMs := now.UnixMilli() - int64(limits.LogsAuditRetentionDaysMax)*millisPerDay
	startMs := retentionFloorMs
	if req.Start != nil {
		if reqStart := req.Start.UnixMilli(); reqStart > startMs {
			startMs = reqStart
		}
	}
	endMs := now.UnixMilli()
	if req.End != nil {
		endMs = req.End.UnixMilli()
	}

	chReq := clickhouse.ListAuditLogsRequest{ //nolint:exhaustruct // optional filters/cursor set conditionally below
		WorkspaceID: principal.WorkspaceID,
		Bucket:      defaultBucket,
		StartMs:     startMs,
		EndMs:       endMs,
		Limit:       limit + 1, // fetch one extra to detect a next page
	}
	if req.Cursor != nil && *req.Cursor != "" {
		cur, decErr := decodeCursor(*req.Cursor)
		if decErr != nil {
			return errInvalidCursor
		}
		chReq.AfterInsertedAtMs = cur.InsertedAt
		chReq.AfterEventID = cur.EventID
	}
	if req.Event != nil {
		chReq.Events = *req.Event
	}
	if req.ActorId != nil {
		chReq.ActorID = *req.ActorId
	}
	if req.ResourceType != nil {
		chReq.ResourceType = *req.ResourceType
	}

	rows, err := h.ClickHouse.ListAuditLogs(ctx, chReq)
	if err != nil {
		return err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	data := make([]openapi.AuditLog, len(rows))
	for i, r := range rows {
		data[i] = mapAuditLog(r)
	}

	pagination := openapi.Pagination{HasMore: hasMore} //nolint:exhaustruct // Cursor set only when there is a next page
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		cursor := encodeCursor(auditCursor{InsertedAt: last.InsertedAt, EventID: last.EventID})
		pagination.Cursor = &cursor
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pagination,
	})
}
