//nolint:exhaustruct // Response structs initialized incrementally
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/billing"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

type UpdatePricingModelRequest struct {
	ID                    string                 `json:"id" validate:"required"`
	Name                  string                 `json:"name" validate:"required"`
	VerificationUnitPrice float64                `json:"verificationUnitPrice" validate:"gte=0"`
	TieredPricing         *billing.TieredPricing `json:"tieredPricing,omitempty"`
}

type UpdatePricingModelResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		ID                    string                 `json:"id"`
		WorkspaceID           string                 `json:"workspaceId"`
		Name                  string                 `json:"name"`
		Currency              string                 `json:"currency"`
		VerificationUnitPrice float64                `json:"verificationUnitPrice"`
		TieredPricing         *billing.TieredPricing `json:"tieredPricing,omitempty"`
		Version               int32                  `json:"version"`
		Active                bool                   `json:"active"`
		CreatedAt             int64                  `json:"createdAt"`
		UpdatedAt             int64                  `json:"updatedAt"`
	} `json:"data"`
}

type Handler struct {
	Logger         logging.Logger
	DB             db.Database
	Keys           keys.KeyService
	PricingService billing.PricingModelService
	Auditlogs      auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "PUT"
}

func (h *Handler) Path() string {
	return "/v1/billing/pricing-models/:id"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/pricing-models/:id")

	// 1. Authentication
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// 2. Check beta access
	if err := h.checkBetaAccess(ctx, auth.AuthorizedWorkspaceID); err != nil {
		return err
	}

	// 3. Permission check
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   auth.AuthorizedWorkspaceID,
			Action:       rbac.UpdateAPI,
		}),
	)))
	if err != nil {
		return err
	}

	// 4. Get ID from path parameter
	id := s.Request().PathValue("id")
	if id == "" {
		return fault.New("pricing model ID is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Pricing model ID is required"),
		)
	}

	// 5. Request validation
	req, err := zen.BindBody[UpdatePricingModelRequest](s)
	if err != nil {
		return err
	}

	// 6. Verify pricing model belongs to workspace
	existing, err := h.PricingService.GetPricingModel(ctx, id)
	if err != nil {
		return err
	}

	if existing.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("pricing model not found",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("pricing model belongs to different workspace"),
			fault.Public("Pricing model not found"),
		)
	}

	// 7. Update pricing model
	model, err := h.PricingService.UpdatePricingModel(ctx, id, &billing.UpdatePricingModelRequest{
		Name:                  req.Name,
		VerificationUnitPrice: req.VerificationUnitPrice,
		TieredPricing:         req.TieredPricing,
	})
	if err != nil {
		return err
	}

	// 8. Audit log
	err = h.Auditlogs.Insert(ctx, h.DB.RW(), []auditlog.AuditLog{
		{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       "billing.pricing_model.updated",
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.Key.ID,
			ActorName:   "root key",
			ActorMeta:   map[string]any{},
			Display:     fmt.Sprintf("Updated pricing model %s to version %d", model.Name, model.Version),
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
			Resources: []auditlog.AuditLogResource{
				{
					Type:        "pricing_model",
					ID:          model.ID,
					DisplayName: model.Name,
					Name:        model.Name,
					Meta:        map[string]any{},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// 9. Return response
	resp := UpdatePricingModelResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data.ID = model.ID
	resp.Data.WorkspaceID = model.WorkspaceID
	resp.Data.Name = model.Name
	resp.Data.Currency = model.Currency
	resp.Data.VerificationUnitPrice = model.VerificationUnitPrice
	resp.Data.TieredPricing = model.TieredPricing
	resp.Data.Version = model.Version
	resp.Data.Active = model.Active
	resp.Data.CreatedAt = model.CreatedAt.UnixMilli()
	resp.Data.UpdatedAt = model.UpdatedAt.UnixMilli()

	return s.JSON(http.StatusOK, resp)
}

// checkBetaAccess verifies the workspace has billing beta access
func (h *Handler) checkBetaAccess(ctx context.Context, workspaceID string) error {
	workspace, err := db.Query.FindWorkspaceByID(ctx, h.DB.RO(), workspaceID)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to find workspace"),
			fault.Public("Failed to verify workspace access"),
		)
	}

	// Check if billing is in beta features
	var betaFeatures map[string]interface{}
	if len(workspace.BetaFeatures) > 0 {
		if err := json.Unmarshal(workspace.BetaFeatures, &betaFeatures); err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to parse beta features"),
				fault.Public("Failed to verify workspace access"),
			)
		}
	}

	// Check if billing feature is enabled
	if betaFeatures == nil || betaFeatures["billing"] != true {
		return fault.New("billing feature not enabled",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("workspace does not have billing beta access"),
			fault.Public("Workspace does not have access to billing features"),
		)
	}

	return nil
}
