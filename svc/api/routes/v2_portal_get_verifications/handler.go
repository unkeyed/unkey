package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/auth/portalrbac"
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

// DefaultMaxPerKeySeries bounds how many keys a single per-key breakout may
// return. It exists to cap what one session can pull over the shared ClickHouse
// connection, not to express a product limit: a portal end user holding a
// thousand keys with traffic in one window is far outside real usage.
const DefaultMaxPerKeySeries = 1000

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
	ClickHouse      clickhouse.ClickHouse
	DB              db.Database
	LimitsCache     cache.Cache[string, keysdb.Limit]
	MaxPerKeySeries int
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string { return "POST" }

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.getVerifications" }

// Handle returns a verification timeseries scoped to the portal session's
// external identity, plus a per-key breakout of the same window when the
// request asks for one.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	keySpaceIDs, err := portalscope.KeyspaceIDs(s)
	if err != nil {
		return err
	}

	// Capability and identity scope are separate: this gates the action while the
	// ClickHouse query below fixes the visible data to the session external ID.
	err = principal.Authorize(rbac.S(portalrbac.CapAnalyticsRead))
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
	// zero-filled series. We use the same log retention limit that the protected
	// analytics.getVerifications path uses as MaxQueryRangeDays, so the portal
	// cannot query a wider range than the workspace itself.
	limits, _, err := h.LimitsCache.SWR(ctx, principal.AuthorizedWorkspaceID, func(ctx context.Context) (keysdb.Limit, error) {
		return keysdb.Query.FindLimitsByWorkspaceID(ctx, h.DB.RO(), principal.AuthorizedWorkspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load workspace limits"),
			fault.Public("Failed to validate the requested time window."),
		)
	}

	if limits.LogsRetentionDaysMax > 0 && req.EndTime-req.StartTime > int64(limits.LogsRetentionDaysMax)*millisPerDay {
		return fault.New("time window too large",
			fault.Code(codes.User.BadRequest.QueryRangeExceedsRetention.URN()),
			fault.Internal("requested window exceeds workspace log retention"),
			fault.Public(fmt.Sprintf("The requested time window is too large. The maximum window is %d days.", limits.LogsRetentionDaysMax)),
		)
	}

	// Built once and shared by both reads below: the per-key breakout must be
	// scoped identically to the account-wide series, and two literals would
	// drift the next time a scoping field is added.
	scope := clickhouse.VerificationTimeseriesRequest{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		ExternalID:  externalID,
		KeySpaceIDs: keySpaceIDs,
		KeyID:       ptr.SafeDeref(req.KeyId),
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	}

	points, err := h.ClickHouse.GetVerificationsByExternalID(ctx, scope)
	if err != nil {
		return err
	}

	response := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: toDataPoints(points),
		Keys: nil,
	}

	if ptr.SafeDeref(req.PerKey) {
		perKey, err := h.ClickHouse.GetVerificationsByExternalIDPerKey(ctx, clickhouse.VerificationTimeseriesPerKeyRequest{
			VerificationTimeseriesRequest: scope,
			MaxKeys:                       h.MaxPerKeySeries,
		})
		if errors.Is(err, clickhouse.ErrTooManyVerificationKeys) {
			return fault.Wrap(err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("per-key breakout exceeds the key cap"),
				fault.Public(fmt.Sprintf("The per-key breakout is limited to %d keys. Request a narrower window or a single `keyId`.", h.MaxPerKeySeries)),
			)
		}
		if err != nil {
			return err
		}

		keys := make([]openapi.V2PortalGetVerificationsKeySeries, len(perKey))
		for i, series := range perKey {
			keys[i] = openapi.V2PortalGetVerificationsKeySeries{
				KeyId: series.KeyID,
				Data:  toDataPoints(series.Data),
			}
		}
		response.Keys = &keys
	}

	return s.JSON(http.StatusOK, response)
}

// toDataPoints converts a ClickHouse timeseries into its API representation.
func toDataPoints(points []clickhouse.VerificationTimeseriesDataPoint) []openapi.V2PortalGetVerificationsDataPoint {
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
	return data
}
