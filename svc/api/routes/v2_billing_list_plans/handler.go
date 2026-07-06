package v2BillingListPlans

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/billing"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2BillingListPlansRequestBody
	Response = openapi.V2BillingListPlansResponseBody
	// plan matches the generated anonymous Plans element type.
	plan = struct {
		Config   interface{} `json:"config,omitempty"`
		Currency string      `json:"currency"`
		Id       string      `json:"id"`
		Name     string      `json:"name"`
	}
)

// Handler serves POST /v2/billing.listPlans: the workspace's selectable rate
// cards plus the end-user's current selection and effective card (R17).
// Called by the customer's backend on behalf of their end-user; the identity
// is always resolved inside the authenticated workspace, never across it.
type Handler struct {
	DB       db.Database
	Resolver *billing.Resolver
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/billing.listPlans"
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
			fault.Public("We're unable to list plans right now."),
		)
	}

	cards, err := db.Query.ListSelectableRateCards(ctx, h.DB.RO(), principal.WorkspaceID)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to list rate cards"),
			fault.Public("We're unable to list plans right now."),
		)
	}

	plans := make([]plan, 0, len(cards))
	for _, card := range cards {
		var config interface{}
		if unmarshalErr := json.Unmarshal(card.Config, &config); unmarshalErr != nil {
			config = nil
		}
		plans = append(plans, plan{
			Id:       card.ID,
			Name:     card.Name,
			Currency: card.Currency,
			Config:   config,
		})
	}

	data := openapi.V2BillingListPlansResponseData{
		Plans:               plans,
		SelectedRateCardId:  nil,
		EffectiveRateCardId: nil,
	}
	if identity.SelectedRateCardID.Valid {
		data.SelectedRateCardId = ptr.P(identity.SelectedRateCardID.String)
	}

	effective, err := h.Resolver.ResolveLive(ctx, principal.WorkspaceID, identity.ID)
	switch {
	case err == nil:
		data.EffectiveRateCardId = ptr.P(effective.Card.ID)
	case errors.Is(err, billing.ErrNoRateCard):
		// No card resolves: effectiveRateCardId stays absent.
	default:
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to resolve effective rate card"),
			fault.Public("We're unable to list plans right now."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
