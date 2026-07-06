package v2BillingSelectPlan

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2BillingSelectPlanRequestBody
	Response = openapi.V2BillingSelectPlanResponseBody
)

// Handler serves POST /v2/billing.selectPlan: records an end-user's pick
// from the workspace's selectable rate cards (R17). Selections apply to
// periods not yet billed; a recorded period keeps its card (R18).
type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/billing.selectPlan"
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
		Action:       rbac.UpdateBilling,
	}))
	if err != nil {
		return err
	}

	identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
		WorkspaceID: principal.WorkspaceID,
		ExternalID:  req.ExternalId,
		Deleted:     false,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("identity not found",
				fault.Code(codes.Data.Identity.NotFound.URN()),
				fault.Internal("identity not found"), fault.Public("This identity does not exist."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to load identity"),
			fault.Public("We're unable to record the selection right now."),
		)
	}

	card, err := db.Query.FindRateCardByID(ctx, h.DB.RO(), db.FindRateCardByIDParams{
		WorkspaceID: principal.WorkspaceID,
		RateCardID:  req.RateCardId,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("rate card not found",
				fault.Code(codes.Data.RateCard.NotFound.URN()),
				fault.Internal("rate card not found"), fault.Public("This rate card does not exist."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to load rate card"),
			fault.Public("We're unable to record the selection right now."),
		)
	}

	// Only cards the workspace marked selectable are in the end-user's
	// allowed set; assignments of private cards remain owner-only (R16).
	if !card.Selectable || card.Archived {
		return fault.New("rate card is not selectable",
			fault.Code(codes.Data.RateCard.NotSelectable.URN()),
			fault.Internal("rate card not selectable"),
			fault.Public("This rate card is not available for selection."),
		)
	}

	err = db.Query.UpdateIdentitySelectedRateCard(ctx, h.DB.RW(), db.UpdateIdentitySelectedRateCardParams{
		SelectedRateCardID: sql.NullString{Valid: true, String: card.ID},
		WorkspaceID:        principal.WorkspaceID,
		IdentityID:         identity.ID,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to record selection"),
			fault.Public("We're unable to record the selection right now."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: struct {
			RateCardId string `json:"rateCardId"`
		}{RateCardId: card.ID},
	})
}
