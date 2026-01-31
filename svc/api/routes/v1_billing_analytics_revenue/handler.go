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

type RevenueAnalyticsRequest struct {
	StartDate   string `query:"startDate" validate:"required"`
	EndDate     string `query:"endDate" validate:"required"`
	Granularity string `query:"granularity" validate:"required,oneof=day week month"`
}

type RevenueDataPoint struct {
	Timestamp int64 `json:"timestamp"`
	Revenue   int64 `json:"revenue"`
	Count     int64 `json:"count"`
}

type RevenueAnalyticsResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		TotalRevenue int64              `json:"totalRevenue"`
		DataPoints   []RevenueDataPoint `json:"dataPoints"`
	} `json:"data"`
}

type Handler struct {
	Logger         logging.Logger
	DB             db.Database
	Keys           keys.KeyService
	BillingService billing.BillingService
}

func (h *Handler) Method() string {
	return "GET"
}

func (h *Handler) Path() string {
	return "/v1/billing/analytics/revenue"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/analytics/revenue")

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

	// 4. Parse query parameters
	var req RevenueAnalyticsRequest
	if err := s.BindQuery(&req); err != nil {
		return err
	}

	// 5. Parse dates
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

	// 6. Get revenue analytics
	analytics, err := h.BillingService.GetRevenueAnalytics(ctx, &billing.RevenueAnalyticsRequest{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		StartDate:   startDate,
		EndDate:     endDate,
		Granularity: req.Granularity,
	})
	if err != nil {
		return err
	}

	// 7. Build response
	resp := RevenueAnalyticsResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data.TotalRevenue = analytics.TotalRevenue
	resp.Data.DataPoints = make([]RevenueDataPoint, 0, len(analytics.DataPoints))

	for _, dp := range analytics.DataPoints {
		resp.Data.DataPoints = append(resp.Data.DataPoints, RevenueDataPoint{
			Timestamp: dp.Timestamp.UnixMilli(),
			Revenue:   dp.Revenue,
			Count:     dp.Count,
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
