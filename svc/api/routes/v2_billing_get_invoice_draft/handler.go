package v2BillingGetInvoiceDraft

import (
	"context"
	"errors"
	"math/big"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/billing"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2BillingGetInvoiceDraftRequestBody
	Response = openapi.V2BillingGetInvoiceDraftResponseBody
	// lineItem matches the generated anonymous LineItems element type.
	lineItem = struct {
		AmountCents        *string                                                        `json:"amountCents,omitempty"`
		AmountCentsRounded *int64                                                         `json:"amountCentsRounded,omitempty"`
		Dimension          openapi.V2BillingGetInvoiceDraftResponseDataLineItemsDimension `json:"dimension"`
		Quantity           int64                                                          `json:"quantity"`
	}
)

// Handler serves POST /v2/billing.getInvoiceDraft: per-end-user line items
// priced by the rate card resolved for each identity and period (R9, R19).
// Drafting records the resolved card so re-drafting and the period-close
// push price identically (R18).
type Handler struct {
	ClickHouse clickhouse.ClickHouse
	Resolver   *billing.Resolver
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/billing.getInvoiceDraft"
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

	// Preview must scope usage exactly like the period-close push (R19 parity).
	usage, err := h.ClickHouse.GetBillableUsagePerIdentity(ctx, principal.WorkspaceID, req.Year, req.Month, enabledKeyspaces, enabledNamespaces)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to query per-identity usage"),
			fault.Public("We're unable to load billable usage right now."),
		)
	}

	data := make([]openapi.V2BillingGetInvoiceDraftResponseData, 0, len(usage))
	for _, row := range usage {
		if req.ExternalId != nil && row.ExternalID != ptr.SafeDeref(req.ExternalId) {
			continue
		}

		entry := openapi.V2BillingGetInvoiceDraftResponseData{
			IdentityId: row.IdentityID,
			ExternalId: row.ExternalID,
			Priced:     false,
			LineItems: []lineItem{
				{Dimension: openapi.Verifications, Quantity: row.Verifications, AmountCents: nil, AmountCentsRounded: nil},
				{Dimension: openapi.Credits, Quantity: row.SpentCredits, AmountCents: nil, AmountCentsRounded: nil},
				{Dimension: openapi.Ratelimits, Quantity: row.RatelimitsPassed, AmountCents: nil, AmountCentsRounded: nil},
			},
			RateCardId:        nil,
			RateCardName:      nil,
			Currency:          nil,
			ResolvedFrom:      nil,
			TotalCents:        nil,
			TotalCentsRounded: nil,
		}

		// A draft is a read/preview: it must NOT pin the period's rate card.
		// Pinning happens at billing time in the period-close push. Using the
		// non-persisting resolver keeps previewing an open month from freezing
		// the card for a period still accruing usage.
		resolved, resolveErr := h.Resolver.Resolve(ctx, principal.WorkspaceID, row.IdentityID, req.Year, req.Month)
		switch {
		case resolveErr == nil:
			amounts, priceErr := resolved.Price(row.Verifications, row.SpentCredits, row.RatelimitsPassed)
			if priceErr != nil {
				return fault.Wrap(priceErr,
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Internal("failed to price usage"),
					fault.Public("We're unable to price this invoice draft right now."),
				)
			}
			entry.Priced = true
			entry.RateCardId = ptr.P(resolved.Card.ID)
			entry.RateCardName = ptr.P(resolved.Card.Name)
			entry.Currency = ptr.P(resolved.Card.Currency)
			entry.ResolvedFrom = ptr.P(openapi.V2BillingGetInvoiceDraftResponseDataResolvedFrom(resolved.ResolvedFrom))
			entry.LineItems = []lineItem{
				priced(openapi.Verifications, row.Verifications, amounts.VerificationsCents),
				priced(openapi.Credits, row.SpentCredits, amounts.CreditsCents),
				priced(openapi.Ratelimits, row.RatelimitsPassed, amounts.RatelimitsCents),
			}
			entry.TotalCents = ptr.P(ratecard.CentsString(amounts.TotalCents))
			// Stripe bills the sum of the per-line-item rounded amounts, so the
			// draft total must be that same sum — not RoundedCents of the exact
			// total, which can differ by a cent when two dimensions each round up.
			entry.TotalCentsRounded = ptr.P(
				ratecard.RoundedCents(amounts.VerificationsCents) +
					ratecard.RoundedCents(amounts.CreditsCents) +
					ratecard.RoundedCents(amounts.RatelimitsCents),
			)
		case errors.Is(resolveErr, billing.ErrNoRateCard):
			// Quantity-only entry: nothing to price against, surfaced via
			// priced=false rather than dropping the identity from the draft.
		default:
			return fault.Wrap(resolveErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("failed to resolve rate card"),
				fault.Public("We're unable to price this invoice draft right now."),
			)
		}

		data = append(data, entry)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}

func priced(dimension openapi.V2BillingGetInvoiceDraftResponseDataLineItemsDimension, quantity int64, cents *big.Rat) lineItem {
	return lineItem{
		Dimension:          dimension,
		Quantity:           quantity,
		AmountCents:        ptr.P(ratecard.CentsString(cents)),
		AmountCentsRounded: ptr.P(ratecard.RoundedCents(cents)),
	}
}
