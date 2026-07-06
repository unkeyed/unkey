package enduserbilling

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingItems struct {
	requests []InvoiceItemRequest
}

func (r *recordingItems) CreateInvoiceItem(ctx context.Context, req InvoiceItemRequest) error {
	r.requests = append(r.requests, req)
	return nil
}

func record(identity, customer string, verifications, verificationsCents int64) UsageRecord {
	return UsageRecord{
		IdentityID:         identity,
		ExternalID:         identity + "_ext",
		ProviderCustomerID: customer,
		RateCardID:         "rc_1",
		Verifications:      verifications,
		SpentCredits:       0,
		RatelimitsPassed:   0,
		VerificationsCents: verificationsCents,
		CreditsCents:       0,
		RatelimitsCents:    0,
		Currency:           "usd",
	}
}

func TestStripeConnectPusher(t *testing.T) {
	items := &recordingItems{requests: nil}
	pusher := NewStripeConnectPusher(items)

	t.Run("posts one item per non-zero dimension on the connected account", func(t *testing.T) {
		items.requests = nil
		pushed, err := pusher.Push(context.Background(), PushRequest{
			WorkspaceID:        "ws_1",
			ConnectedAccountID: "acct_123",
			Year:               2026,
			Month:              6,
			Records:            []UsageRecord{record("id_1", "cus_1", 100, 250)},
		})
		require.NoError(t, err)
		require.Equal(t, 1, pushed)
		require.Len(t, items.requests, 1)
		item := items.requests[0]
		require.Equal(t, "acct_123", item.ConnectedAccountID)
		require.Equal(t, "cus_1", item.CustomerID)
		require.Equal(t, int64(250), item.AmountCents)
		require.Equal(t, "usd", item.Currency)
		require.Equal(t, "unkey-eub-ws_1-id_1-2026-06-verifications", item.IdempotencyKey)
	})

	t.Run("re-pushing the same period reuses the same idempotency key", func(t *testing.T) {
		items.requests = nil
		req := PushRequest{
			WorkspaceID:        "ws_1",
			ConnectedAccountID: "acct_123",
			Year:               2026,
			Month:              6,
			Records:            []UsageRecord{record("id_1", "cus_1", 100, 250)},
		}
		_, err := pusher.Push(context.Background(), req)
		require.NoError(t, err)
		_, err = pusher.Push(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, items.requests, 2)
		require.Equal(t, items.requests[0].IdempotencyKey, items.requests[1].IdempotencyKey,
			"same closed period must produce the same key so Stripe dedups")
	})

	t.Run("missing provider customer surfaces as an error, not a silent skip", func(t *testing.T) {
		items.requests = nil
		pushed, err := pusher.Push(context.Background(), PushRequest{
			WorkspaceID:        "ws_1",
			ConnectedAccountID: "acct_123",
			Year:               2026,
			Month:              6,
			Records: []UsageRecord{
				record("id_nocus", "", 10, 25),
				record("id_ok", "cus_2", 10, 25),
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "id_nocus")
		require.Equal(t, 1, pushed, "healthy records still push")
	})

	t.Run("zero-usage and zero-amount records are skipped", func(t *testing.T) {
		items.requests = nil
		pushed, err := pusher.Push(context.Background(), PushRequest{
			WorkspaceID:        "ws_1",
			ConnectedAccountID: "acct_123",
			Year:               2026,
			Month:              6,
			Records: []UsageRecord{
				record("id_zero", "cus_3", 0, 0),
				// Usage within the free tier: quantity > 0, amount 0.
				record("id_free", "cus_4", 5, 0),
			},
		})
		require.NoError(t, err)
		require.Equal(t, 0, pushed)
		require.Empty(t, items.requests)
	})

	t.Run("missing connected account fails cleanly", func(t *testing.T) {
		_, err := pusher.Push(context.Background(), PushRequest{
			WorkspaceID:        "ws_1",
			ConnectedAccountID: "",
			Year:               2026,
			Month:              6,
			Records:            []UsageRecord{record("id_1", "cus_1", 1, 1)},
		})
		require.Error(t, err)
	})
}
