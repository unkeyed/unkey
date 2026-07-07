package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// millisPerDay is the width of one retention day in unix milliseconds.
const millisPerDay = 24 * 60 * 60 * 1000

type (
	Request  = openapi.V2PortalGetVerificationsRequestBody
	Response = openapi.V2PortalGetVerificationsResponseBody
)

// Handler serves portal.getVerifications. Unlike the protected
// analytics.getVerifications, it is a dedicated endpoint: it runs a fixed,
// server-side query on the shared ClickHouse connection scoped to the portal
// session's external identity. It deliberately does not reuse the analytics
// handler, which requires a per-workspace ClickHouse user and a query-language
// parser that are inappropriate for an end user.
type Handler struct {
	ClickHouse clickhouse.ClickHouse
	DB         db.Database
	QuotaCache cache.Cache[string, keysdb.Quotas]
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string { return "POST" }

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.getVerifications" }

// Handle returns a verification timeseries scoped to the portal session's
// external identity.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	// The workspace owner controls whether a portal session may read analytics by
	// including a read_analytics grant in the session permissions. Identity scoping
	// already restricts *what* is returned to the session's own events; this gates
	// whether analytics is exposed to this end user at all. The query spans all of
	// the identity's keys across every API, so require the wildcard grant.
	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.ReadAnalytics,
	}))
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if req.EndTime <= req.StartTime {
		return fault.New("invalid time window",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("endTime must be greater than startTime"),
			fault.Public("`endTime` must be greater than `startTime`."),
		)
	}

	// Bound the window to the workspace's log retention. This runs on the shared
	// ClickHouse connection, so an unbounded window (e.g. the unix epoch to a far
	// future) would let an end user force an arbitrarily large scan and
	// zero-filled series. We reuse LogsRetentionDays, the same quota the protected
	// analytics.getVerifications enforces as MaxQueryRangeDays, so the portal can
	// never query a wider range than the workspace itself.
	quota, _, err := h.QuotaCache.SWR(ctx, principal.WorkspaceID, func(ctx context.Context) (keysdb.Quotas, error) {
		return keysdb.Query.FindQuotaByWorkspaceID(ctx, h.DB.RO(), principal.WorkspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load workspace quota"),
			fault.Public("Failed to validate the requested time window."),
		)
	}

	if quota.LogsRetentionDays > 0 && req.EndTime-req.StartTime > int64(quota.LogsRetentionDays)*millisPerDay {
		return fault.New("time window too large",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("requested window exceeds workspace log retention"),
			fault.Public(fmt.Sprintf("The requested time window is too large. The maximum window is %d days.", quota.LogsRetentionDays)),
		)
	}

	points, err := h.ClickHouse.GetVerificationsByExternalID(ctx, clickhouse.VerificationTimeseriesRequest{
		WorkspaceID: principal.WorkspaceID,
		ExternalID:  externalID,
		KeyID:       ptr.SafeDeref(req.KeyId),
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	})
	if err != nil {
		return err
	}

	data := make([]openapi.V2PortalGetVerificationsDataPoint, len(points))
	for i, p := range points {
		data[i] = openapi.V2PortalGetVerificationsDataPoint{
			Time:                    p.Time,
			Total:                   p.Total,
			Valid:                   p.Valid,
			RateLimited:             p.RateLimited,
			InsufficientPermissions: p.InsufficientPermissions,
			Forbidden:               p.Forbidden,
			Disabled:                p.Disabled,
			Expired:                 p.Expired,
			UsageExceeded:           p.UsageExceeded,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
