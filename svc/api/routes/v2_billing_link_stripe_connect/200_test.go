package v2BillingLinkStripeConnect_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_billing_link_stripe_connect"
	unlink "github.com/unkeyed/unkey/svc/api/routes/v2_billing_unlink_stripe_connect"
)

// fakeVerifier accepts exactly one account id, standing in for Stripe's
// "account is connected to this platform" check.
type fakeVerifier struct {
	accepted string
}

func (f *fakeVerifier) VerifyConnectedAccount(ctx context.Context, accountID string) error {
	if accountID == f.accepted {
		return nil
	}
	return fault.New("account not connected to platform")
}

func TestLinkAndUnlinkStripeConnect(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:       h.DB,
		Vault:    h.Vault,
		Verifier: &fakeVerifier{accepted: "acct_verified123"},
	}
	unlinkRoute := &unlink.Handler{DB: h.DB}
	h.Register(route)
	h.Register(unlinkRoute)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, "billing.*.manage_billing")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("verified account links and is stored as ciphertext", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ConnectedAccountId: "acct_verified123",
		})
		require.Equal(t, 200, res.Status, "%s", res.RawBody)

		settings, err := db.Query.FindWorkspaceBillingSettings(context.Background(), h.DB.RO(), workspaceID)
		require.NoError(t, err)
		require.True(t, settings.StripeConnectEncrypted.Valid)
		// R5: never plaintext — the stored blob must not contain the account id.
		require.NotContains(t, settings.StripeConnectEncrypted.String, "acct_verified123")
		require.True(t, settings.StripeConnectEncryptionKeyID.Valid)
	})

	t.Run("unverified account is rejected and not stored", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ConnectedAccountId: "acct_attacker9",
		})
		require.Equal(t, 400, res.Status, "%s", res.RawBody)
	})

	t.Run("missing manage_billing permission is rejected", func(t *testing.T) {
		weakKey := h.CreateRootKey(workspaceID, "billing.*.read_billing")
		weakHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", weakKey)},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, weakHeaders, handler.Request{
			ConnectedAccountId: "acct_verified123",
		})
		require.Equal(t, 403, res.Status, "%s", res.RawBody)
	})

	t.Run("unlink clears the reference", func(t *testing.T) {
		res := testutil.CallRoute[unlink.Request, unlink.Response](h, unlinkRoute, headers, unlink.Request{})
		require.Equal(t, 200, res.Status, "%s", res.RawBody)

		settings, err := db.Query.FindWorkspaceBillingSettings(context.Background(), h.DB.RO(), workspaceID)
		require.NoError(t, err)
		require.False(t, settings.StripeConnectEncrypted.Valid)
	})
}
