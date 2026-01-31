//nolint:exhaustruct // Response structs initialized incrementally
package handler

import (
	"context"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/billing"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/zen"
)

type WebhookResponse struct {
	Meta struct {
		RequestID string `json:"requestId"`
	} `json:"meta"`
	Data struct {
		Received bool `json:"received"`
	} `json:"data"`
}

type Handler struct {
	Logger         logging.Logger
	BillingService billing.BillingService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v1/billing/webhooks/stripe"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v1/billing/webhooks/stripe")

	// 1. Get Stripe signature from header
	signature := s.Request().Header.Get("Stripe-Signature")
	if signature == "" {
		return fault.New("missing Stripe signature",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("Stripe-Signature header is missing"),
			fault.Public("Invalid webhook request"),
		)
	}

	// 2. Read raw body
	body, err := io.ReadAll(s.Request().Body)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to read request body"),
			fault.Public("Failed to process webhook"),
		)
	}

	// 3. Process webhook
	err = h.BillingService.ProcessWebhook(ctx, body, signature)
	if err != nil {
		// Return 500 to trigger Stripe retry
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to process webhook"),
			fault.Public("Failed to process webhook"),
		)
	}

	// 4. Return success response
	resp := WebhookResponse{}
	resp.Meta.RequestID = s.RequestID()
	resp.Data.Received = true

	return s.JSON(http.StatusOK, resp)
}
