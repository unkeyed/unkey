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

type DeletePricingModelResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		Success bool `json:"success"`
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
	return "DELETE"
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

	// 5. Verify pricing model belongs to workspace
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

	// 6. Delete pricing model
	err = h.PricingService.DeletePricingModel(ctx, id)
	if err != nil {
		return err
	}

	// 7. Audit log
	err = h.Auditlogs.Insert(ctx, h.DB.RW(), []auditlog.AuditLog{
		{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       "billing.pricing_model.deleted",
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.Key.ID,
			ActorName:   "root key",
			ActorMeta:   map[string]any{},
			Display:     fmt.Sprintf("Deleted pricing model %s", existing.Name),
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
			Resources: []auditlog.AuditLogResource{
				{
					Type:        "pricing_model",
					ID:          existing.ID,
					DisplayName: existing.Name,
					Name:        existing.Name,
					Meta:        map[string]any{},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// 8. Return success response
	resp := DeletePricingModelResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data.Success = true

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
