//nolint:exhaustruct // Response structs initialized incrementally
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/billing"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

type ListInvoicesRequest struct {
	EndUserID string `query:"endUserId"`
	Status    string `query:"status"`
	Limit     string `query:"limit"`
	Offset    string `query:"offset"`
}

type InvoiceData struct {
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
}

type ListInvoicesResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data []InvoiceData `json:"data"`
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
	return "/v1/billing/invoices"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/invoices")

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
	var req ListInvoicesRequest
	if err := s.BindQuery(&req); err != nil {
		return err
	}

	// Parse limit and offset
	limit := 50
	if req.Limit != "" {
		if l, err := strconv.Atoi(req.Limit); err == nil {
			limit = l
		}
	}

	offset := 0
	if req.Offset != "" {
		if o, err := strconv.Atoi(req.Offset); err == nil {
			offset = o
		}
	}

	// 5. List invoices
	invoices, err := h.BillingService.ListInvoices(ctx, &billing.ListInvoicesRequest{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		EndUserID:   req.EndUserID,
		Status:      req.Status,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return err
	}

	// 6. Build response
	resp := ListInvoicesResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data = make([]InvoiceData, 0, len(invoices))

	for _, invoice := range invoices {
		resp.Data = append(resp.Data, InvoiceData{
			ID:                 invoice.ID,
			WorkspaceID:        invoice.WorkspaceID,
			EndUserID:          invoice.EndUserID,
			StripeInvoiceID:    invoice.StripeInvoiceID,
			BillingPeriodStart: invoice.BillingPeriodStart.UnixMilli(),
			BillingPeriodEnd:   invoice.BillingPeriodEnd.UnixMilli(),
			VerificationCount:  invoice.VerificationCount,
			RatelimitCount:     invoice.RatelimitCount,
			TotalAmount:        invoice.TotalAmount,
			Currency:           invoice.Currency,
			Status:             string(invoice.Status),
			CreatedAt:          invoice.CreatedAt.UnixMilli(),
			UpdatedAt:          invoice.UpdatedAt.UnixMilli(),
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
