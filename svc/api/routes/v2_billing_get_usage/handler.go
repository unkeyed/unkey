package v2BillingGetUsage

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/billing"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2BillingGetUsageRequestBody
	Response = openapi.V2BillingGetUsageResponseBody
)

// Handler serves POST /v2/billing.getUsage: per-end-user billable quantities
// for one period, read from the pre-aggregated per-identity rollup (R8).
type Handler struct {
	ClickHouse clickhouse.ClickHouse
	Resolver   *billing.Resolver
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/billing.getUsage"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Billing,
		ResourceID:   "*",
		Action:       rbac.ReadBilling,
	}))
	if err != nil {
		return err
	}

	enabledKeyspaces, enabledNamespaces, err := h.Resolver.LoadEnabledResources(ctx, principal.WorkspaceID)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to load enabled billable resources"),
			fault.Public("We're unable to load billable usage right now."),
		)
	}

	usage, err := h.ClickHouse.GetBillableUsagePerIdentity(ctx, principal.WorkspaceID, req.Year, req.Month, enabledKeyspaces, enabledNamespaces)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to query per-identity usage"),
			fault.Public("We're unable to load billable usage right now."),
		)
	}

	data := make([]openapi.V2BillingGetUsageResponseData, 0, len(usage))
	for _, row := range usage {
		if req.ExternalId != nil && row.ExternalID != ptr.SafeDeref(req.ExternalId) {
			continue
		}
		data = append(data, openapi.V2BillingGetUsageResponseData{
			IdentityId:       row.IdentityID,
			ExternalId:       row.ExternalID,
			Verifications:    row.Verifications,
			SpentCredits:     row.SpentCredits,
			RatelimitsPassed: row.RatelimitsPassed,
		})
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
