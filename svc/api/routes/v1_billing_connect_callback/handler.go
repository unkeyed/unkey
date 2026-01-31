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
	"github.com/unkeyed/unkey/pkg/zen"
)

type CallbackRequest struct {
	Code  string `query:"code" validate:"required"`
	State string `query:"state" validate:"required"`
}

type CallbackResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	} `json:"data"`
}

type Handler struct {
	Logger        logging.Logger
	DB            db.Database
	Keys          keys.KeyService
	StripeConnect billing.StripeConnectService
}

func (h *Handler) Method() string {
	return "GET"
}

func (h *Handler) Path() string {
	return "/v1/billing/connect/callback"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/connect/callback")

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

	// 3. Parse query parameters
	var req CallbackRequest
	if err := s.BindQuery(&req); err != nil {
		return err
	}

	// 4. Verify state parameter (CSRF protection)
	// In production, this should be validated against a stored state value
	// For now, we just check it's not empty
	if req.State == "" {
		return fault.New("invalid state parameter",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("state parameter is empty"),
			fault.Public("Invalid OAuth state parameter"),
		)
	}

	// 5. Exchange authorization code for access token
	_, err = h.StripeConnect.ExchangeCode(ctx, req.Code)
	if err != nil {
		resp := CallbackResponse{}
		resp.Meta.RequestID = s.RequestID()
		resp.Data.Success = false
		resp.Data.Error = "Failed to connect Stripe account"
		return s.JSON(http.StatusOK, resp)
	}

	// 6. Return success response
	resp := CallbackResponse{}
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
