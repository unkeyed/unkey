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

type CreateEndUserRequest struct {
	ExternalID     string            `json:"externalId" validate:"required"`
	PricingModelID string            `json:"pricingModelId" validate:"required"`
	Email          string            `json:"email,omitempty" validate:"omitempty,email"`
	Name           string            `json:"name,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type CreateEndUserResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		ID                   string            `json:"id"`
		WorkspaceID          string            `json:"workspaceId"`
		ExternalID           string            `json:"externalId"`
		PricingModelID       string            `json:"pricingModelId"`
		StripeCustomerID     string            `json:"stripeCustomerId"`
		StripeSubscriptionID string            `json:"stripeSubscriptionId,omitempty"`
		Email                string            `json:"email,omitempty"`
		Name                 string            `json:"name,omitempty"`
		Metadata             map[string]string `json:"metadata,omitempty"`
		CreatedAt            int64             `json:"createdAt"`
		UpdatedAt            int64             `json:"updatedAt"`
	} `json:"data"`
}

type Handler struct {
	Logger         logging.Logger
	DB             db.Database
	Keys           keys.KeyService
	EndUserService billing.EndUserService
	Auditlogs      auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v1/billing/end-users"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/end-users")

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

	// 4. Request validation
	req, err := zen.BindBody[CreateEndUserRequest](s)
	if err != nil {
		return err
	}

	// 5. Create end user
	endUser, err := h.EndUserService.CreateEndUser(ctx, &billing.CreateEndUserRequest{
		WorkspaceID:    auth.AuthorizedWorkspaceID,
		ExternalID:     req.ExternalID,
		PricingModelID: req.PricingModelID,
		Email:          req.Email,
		Name:           req.Name,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return err
	}

	// 6. Audit log
	err = h.Auditlogs.Insert(ctx, h.DB.RW(), []auditlog.AuditLog{
		{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       "billing.end_user.created",
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.Key.ID,
			ActorName:   "root key",
			ActorMeta:   map[string]any{},
			Display:     fmt.Sprintf("Created end user %s", endUser.ExternalID),
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
			Resources: []auditlog.AuditLogResource{
				{
					Type:        "end_user",
					ID:          endUser.ID,
					DisplayName: endUser.ExternalID,
					Name:        endUser.ExternalID,
					Meta:        map[string]any{},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// 7. Return response
	resp := CreateEndUserResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data.ID = endUser.ID
	resp.Data.WorkspaceID = endUser.WorkspaceID
	resp.Data.ExternalID = endUser.ExternalID
	resp.Data.PricingModelID = endUser.PricingModelID
	resp.Data.StripeCustomerID = endUser.StripeCustomerID
	resp.Data.StripeSubscriptionID = endUser.StripeSubscriptionID
	resp.Data.Email = endUser.Email
	resp.Data.Name = endUser.Name
	resp.Data.Metadata = endUser.Metadata
	resp.Data.CreatedAt = endUser.CreatedAt.UnixMilli()
	resp.Data.UpdatedAt = endUser.UpdatedAt.UnixMilli()

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
