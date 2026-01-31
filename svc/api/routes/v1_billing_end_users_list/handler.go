//nolint:exhaustruct // Response structs initialized incrementally
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/billing"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

type EndUserData struct {
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
}

type ListEndUsersResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data []EndUserData `json:"data"`
}

type Handler struct {
	Logger         logging.Logger
	DB             db.Database
	Keys           keys.KeyService
	EndUserService billing.EndUserService
}

func (h *Handler) Method() string {
	return "GET"
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
			Action:       rbac.ReadAPI,
		}),
	)))
	if err != nil {
		return err
	}

	// 4. List end users
	endUsers, err := h.EndUserService.ListEndUsers(ctx, auth.AuthorizedWorkspaceID)
	if err != nil {
		return err
	}

	// 5. Build response
	resp := ListEndUsersResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data = make([]EndUserData, 0, len(endUsers))

	for _, endUser := range endUsers {
		resp.Data = append(resp.Data, EndUserData{
			ID:                   endUser.ID,
			WorkspaceID:          endUser.WorkspaceID,
			ExternalID:           endUser.ExternalID,
			PricingModelID:       endUser.PricingModelID,
			StripeCustomerID:     endUser.StripeCustomerID,
			StripeSubscriptionID: endUser.StripeSubscriptionID,
			Email:                endUser.Email,
			Name:                 endUser.Name,
			Metadata:             endUser.Metadata,
			CreatedAt:            endUser.CreatedAt.UnixMilli(),
			UpdatedAt:            endUser.UpdatedAt.UnixMilli(),
		})
	}

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

	if betaFeatures == nil || betaFeatures["billing"] != true {
		return fault.New("billing feature not enabled",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("workspace does not have billing beta access"),
			fault.Public("Workspace does not have access to billing features"),
		)
	}

	return nil
}
