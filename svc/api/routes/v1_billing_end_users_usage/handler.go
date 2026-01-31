//nolint:exhaustruct // Response structs initialized incrementally
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/billing"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

type GetUsageRequest struct {
	StartDate string `query:"startDate" validate:"required"`
	EndDate   string `query:"endDate" validate:"required"`
}

type GetUsageResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		ExternalID    string `json:"externalId"`
		Verifications int64  `json:"verifications"`
		RateLimits    int64  `json:"rateLimits"`
	} `json:"data"`
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
	return "/v1/billing/end-users/:id/usage"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/end-users/:id/usage")

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

	// 4. Get ID from path parameter
	id := s.Request().PathValue("id")
	if id == "" {
		return fault.New("end user ID is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("End user ID is required"),
		)
	}

	// 5. Parse query parameters
	var req GetUsageRequest
	if err := s.BindQuery(&req); err != nil {
		return err
	}

	// 6. Parse dates
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invalid start date format. Use RFC3339 format."),
		)
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invalid end date format. Use RFC3339 format."),
		)
	}

	// 7. Verify end user belongs to workspace
	endUser, err := h.EndUserService.GetEndUser(ctx, id)
	if err != nil {
		return err
	}

	if endUser.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("end user not found",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("end user belongs to different workspace"),
			fault.Public("End user not found"),
		)
	}

	// 8. Get usage
	usage, err := h.EndUserService.GetUsage(ctx, id, startDate, endDate)
	if err != nil {
		return err
	}

	// 9. Return response
	resp := GetUsageResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data.ExternalID = usage.ExternalID
	resp.Data.Verifications = usage.Verifications
	resp.Data.RateLimits = usage.RateLimits

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
