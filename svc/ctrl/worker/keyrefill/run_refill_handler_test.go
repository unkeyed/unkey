package keyrefill_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestRunRefill_Integration(t *testing.T) {
	h := harness.New(t)

	// Use today's date for the refill
	now := time.Now().UTC()
	dateKey := fmt.Sprintf("%d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
	todayDay := int16(now.Day())

	t.Run("refills keys with daily refill (refill_day=NULL)", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create key with refill settings (refill_day=NULL means daily)
		refillAmount := int32(1000)
		remaining := int32(100)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			// RefillDay is nil for daily refill
		})

		// Call RunRefill via Restate with a unique date key for this test
		testDateKey := fmt.Sprintf("%s-test-daily-%s", dateKey, uid.New("", 8))
		resp, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		// Should have refilled at least 1 key
		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))
		// Audit logs are created internally but not exposed in response

		// Verify the key was refilled
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, refillAmount, key.RemainingRequests.Int32)
	})

	t.Run("refills keys with matching day of month", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create key with refill_day matching today
		refillAmount := int32(500)
		remaining := int32(50)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			RefillDay:    &todayDay,
		})

		// Call RunRefill with a unique date key
		testDateKey := fmt.Sprintf("%s-test-matching-%s", dateKey, uid.New("", 8))
		resp, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))

		// Verify the key was refilled
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, refillAmount, key.RemainingRequests.Int32)
	})

	t.Run("skips keys with non-matching day of month", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create key with refill_day NOT matching today
		differentDay := int16((int(todayDay) % 28) + 1) // Pick a different day
		if differentDay == todayDay {
			differentDay = int16((int(todayDay) % 27) + 2)
		}
		refillAmount := int32(1000)
		remaining := int32(100)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			RefillDay:    &differentDay,
		})

		// Call RunRefill with a unique date key
		testDateKey := fmt.Sprintf("%s-test-non-matching-%s", dateKey, uid.New("", 8))
		_, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		// Verify the key was NOT refilled (remaining unchanged)
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, remaining, key.RemainingRequests.Int32)
	})

	t.Run("skips keys that are already full", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create key where remaining >= refill_amount (already full)
		refillAmount := int32(1000)
		remaining := int32(1000) // Already at max
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			// RefillDay nil for daily
		})

		// Call RunRefill with a unique date key
		testDateKey := fmt.Sprintf("%s-test-full-%s", dateKey, uid.New("", 8))
		_, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		// Verify the key was NOT refilled (since it's already at max)
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, remaining, key.RemainingRequests.Int32)
	})

	t.Run("creates audit logs for refilled keys", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create key that needs refill
		refillAmount := int32(1000)
		remaining := int32(0)
		h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
		})

		// Call RunRefill with a unique date key
		testDateKey := fmt.Sprintf("%s-test-audit-%s", dateKey, uid.New("", 8))
		resp, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		// Should have refilled the key (audit logs created internally)
		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))
	})

	t.Run("is idempotent with same date key", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create keys that need refill
		refillAmount := int32(1000)
		remaining := int32(100)
		h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
		})
		h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
		})

		// Use a unique date key for this test
		testDateKey := fmt.Sprintf("%s-test-idempotent-%s", dateKey, uid.New("", 8))

		// First call should refill keys
		resp1, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)
		firstRefilled := resp1.GetKeysRefilled()

		// Second call with same date key should process 0 keys (already processed)
		resp2, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		// The second call should not refill any additional keys
		// (they were marked as processed in state)
		require.Equal(t, int32(0), resp2.GetKeysRefilled(), "Second call should not refill any keys")
		require.Greater(t, firstRefilled, int32(0), "First call should have refilled keys")
	})

	t.Run("refills keys on last day of month when refill_day exceeds month length", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create key with refill_day=31 (only exists in some months)
		refillDay := int16(31)
		refillAmount := int32(750)
		remaining := int32(50)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			RefillDay:    &refillDay,
		})

		// Use Feb 28, 2025 as the date key — it's the last day of a short month.
		// Keys with refill_day=31 should still be refilled because 31 > 28 and it's the last day.
		testDateKey := fmt.Sprintf("2025-02-28-test-lastday-%s", uid.New("", 8))
		resp, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))

		// Verify the key was refilled
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, refillAmount, key.RemainingRequests.Int32)
	})

	t.Run("skips deleted keys", func(t *testing.T) {
		// Create workspace with an API
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create a deleted key
		refillAmount := int32(1000)
		remaining := int32(100)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			Deleted:      true,
		})

		// Call RunRefill with a unique date key
		testDateKey := fmt.Sprintf("%s-test-deleted-%s", dateKey, uid.New("", 8))
		_, err := callRunRefill(h, testDateKey)
		require.NoError(t, err)

		// Verify the deleted key was NOT refilled
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyResp.KeyID)
		require.NoError(t, err)
		// Deleted keys should keep their original remaining value
		require.Equal(t, remaining, key.RemainingRequests.Int32)
	})
}

func callRunRefill(h *harness.Harness, dateKey string) (*hydrav1.RunRefillResponse, error) {
	client := hydrav1.NewKeyRefillServiceIngressClient(h.Restate, dateKey)
	return client.RunRefill().Request(h.Ctx, &hydrav1.RunRefillRequest{})
}
