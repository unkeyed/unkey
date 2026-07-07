package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

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
