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

type GetInvoiceResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		ID                 string `json:"id"`
		WorkspaceID        string `json:"workspaceId"`
		EndUserID          string `json:"endUserId"`
		StripeInvoiceID    string `json:"stripeInvoiceId"`
		BillingPeriodStart int64  `json:"billingPeriodStart"`
		BillingPeriodEnd   int64  `json:"billingPeriodEnd"`
		VerificationCount  int64  `json:"verificationCount"`
		RatelimitCount     int64  `json:"ratelimitCount"`
		TotalAmount        int64  `json:"totalAmount"`
		Currency           string `json:"currency"`
		Status             string `json:"status"`
		CreatedAt          int64  `json:"createdAt"`
		UpdatedAt          int64  `json:"updatedAt"`
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
	return "/v1/billing/invoices/:id"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/invoices/:id")

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
		return fault.New("invoice ID is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invoice ID is required"),
		)
	}

	// 5. Get invoice
	invoice, err := h.BillingService.GetInvoice(ctx, id)
	if err != nil {
		return err
	}

	// 6. Verify invoice belongs to workspace
	if invoice.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("invoice not found",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("invoice belongs to different workspace"),
			fault.Public("Invoice not found"),
		)
	}

	// 7. Return response
	resp := GetInvoiceResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data.ID = invoice.ID
	resp.Data.WorkspaceID = invoice.WorkspaceID
	resp.Data.EndUserID = invoice.EndUserID
	resp.Data.StripeInvoiceID = invoice.StripeInvoiceID
	resp.Data.BillingPeriodStart = invoice.BillingPeriodStart.UnixMilli()
	resp.Data.BillingPeriodEnd = invoice.BillingPeriodEnd.UnixMilli()
	resp.Data.VerificationCount = invoice.VerificationCount
	resp.Data.RatelimitCount = invoice.RatelimitCount
	resp.Data.TotalAmount = invoice.TotalAmount
	resp.Data.Currency = invoice.Currency
	resp.Data.Status = string(invoice.Status)
	resp.Data.CreatedAt = invoice.CreatedAt.UnixMilli()
	resp.Data.UpdatedAt = invoice.UpdatedAt.UnixMilli()

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
