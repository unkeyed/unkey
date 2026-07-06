package v2BillingLinkStripeConnect

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/stripeconnect"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2BillingLinkStripeConnectRequestBody
	Response = openapi.V2BillingLinkStripeConnectResponseBody
)

// Handler serves POST /v2/billing.linkStripeConnect (R13, U6).
//
// Access control: this endpoint writes the reference all money-movement
// dispatch targets, so it requires the dedicated billing.*.manage_billing
// permission — grant it as sparingly as key-encryption permissions. The
// account id is verified against Stripe (control of the platform-connected
// account) before it is persisted, Vault-encrypted, never plaintext (R5).
type Handler struct {
	DB       db.Database
	Vault    vault.VaultServiceClient
	Verifier stripeconnect.Verifier
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/billing.linkStripeConnect"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
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

	// Never store a caller-supplied account id without proof it is connected
	// to this platform.
	err = h.Verifier.VerifyConnectedAccount(ctx, req.ConnectedAccountId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("connected account verification failed"),
			fault.Public("This connected account could not be verified. Complete Stripe Connect onboarding for Unkey first."),
		)
	}

	encrypted, err := h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: principal.WorkspaceID,
		Data:    req.ConnectedAccountId,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to encrypt connected account reference"),
			fault.Public("We're unable to link the account right now."),
		)
	}

	err = db.Query.SetWorkspaceBillingStripeConnect(ctx, h.DB.RW(), db.SetWorkspaceBillingStripeConnectParams{
		ID:                           uid.New(uid.RateCardPrefix),
		WorkspaceID:                  principal.WorkspaceID,
		StripeConnectEncrypted:       sqlNullString(encrypted.GetEncrypted()),
		StripeConnectEncryptionKeyID: sqlNullString(encrypted.GetKeyId()),
		StripeConnectStatus: db.NullWorkspaceBillingSettingsStripeConnectStatus{
			Valid: true,
			WorkspaceBillingSettingsStripeConnectStatus: db.WorkspaceBillingSettingsStripeConnectStatusLinked,
		},
		CreatedAt: time.Now().UnixMilli(),
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to persist connected account reference"),
			fault.Public("We're unable to link the account right now."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: struct {
			ConnectedAccountId string `json:"connectedAccountId"`
		}{ConnectedAccountId: req.ConnectedAccountId},
	})
}

func sqlNullString(s string) sql.NullString {
	return sql.NullString{Valid: true, String: s}
}
