package v2BillingUnlinkStripeConnect

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2BillingUnlinkStripeConnectRequestBody
	Response = openapi.V2BillingUnlinkStripeConnectResponseBody
)

// Handler serves POST /v2/billing.unlinkStripeConnect (U6 unlink path):
// deletes the encrypted connected-account reference so the period-close
// push skips the workspace. Recorded periods and rollups are unaffected.
type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/billing.unlinkStripeConnect"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	_, err = zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Billing,
		ResourceID:   "*",
		Action:       rbac.ManageBilling,
	}))
	if err != nil {
		return err
	}

	err = db.Query.ClearWorkspaceBillingStripeConnect(ctx, h.DB.RW(), principal.WorkspaceID)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to clear connected account reference"),
			fault.Public("We're unable to unlink the account right now."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: struct {
			Unlinked bool `json:"unlinked"`
		}{Unlinked: true},
	})
}
