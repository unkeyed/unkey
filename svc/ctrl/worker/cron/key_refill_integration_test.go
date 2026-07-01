package cron_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestRunKeyRefill_Integration(t *testing.T) {
	h := harness.New(t)

	now := time.Now().UTC()
	dateKey := fmt.Sprintf("%d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
	todayDay := int16(now.Day())

	t.Run("refills keys with daily refill (refill_day=NULL)", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		refillAmount := int64(1000)
		remaining := int64(100)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
		})

		testDateKey := fmt.Sprintf("%s-test-daily-%s", dateKey, uid.New("", 8))
		resp, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))

		key, err := h.DB.FindKeyByID(h.Ctx, keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, refillAmount, key.RemainingRequests.Int64)
	})

	t.Run("refills keys with matching day of month", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		refillAmount := int64(500)
		remaining := int64(50)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			RefillDay:    &todayDay,
		})

		testDateKey := fmt.Sprintf("%s-test-matching-%s", dateKey, uid.New("", 8))
		resp, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))

		key, err := h.DB.FindKeyByID(h.Ctx, keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, refillAmount, key.RemainingRequests.Int64)
	})

	t.Run("skips keys with non-matching day of month", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		differentDay := int16((int(todayDay) % 28) + 1)
		if differentDay == todayDay {
			differentDay = int16((int(todayDay) % 27) + 2)
		}
		refillAmount := int64(1000)
		remaining := int64(100)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			RefillDay:    &differentDay,
		})

		testDateKey := fmt.Sprintf("%s-test-non-matching-%s", dateKey, uid.New("", 8))
		_, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		key, err := h.DB.FindKeyByID(h.Ctx, keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, remaining, key.RemainingRequests.Int64)
	})

	t.Run("skips keys that are already full", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		refillAmount := int64(1000)
		remaining := int64(1000)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
		})

		testDateKey := fmt.Sprintf("%s-test-full-%s", dateKey, uid.New("", 8))
		_, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		key, err := h.DB.FindKeyByID(h.Ctx, keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, remaining, key.RemainingRequests.Int64)
	})

	t.Run("creates audit logs for refilled keys", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		refillAmount := int64(1000)
		remaining := int64(0)
		h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
		})

		testDateKey := fmt.Sprintf("%s-test-audit-%s", dateKey, uid.New("", 8))
		resp, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))
	})

	t.Run("is idempotent with same date key", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		refillAmount := int64(1000)
		remaining := int64(100)
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

		testDateKey := fmt.Sprintf("%s-test-idempotent-%s", dateKey, uid.New("", 8))

		resp1, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)
		firstRefilled := resp1.GetKeysRefilled()

		resp2, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		require.Equal(t, int32(0), resp2.GetKeysRefilled(), "Second call should not refill any keys")
		require.Greater(t, firstRefilled, int32(0), "First call should have refilled keys")
	})

	t.Run("refills keys on last day of month when refill_day exceeds month length", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		refillDay := int16(31)
		refillAmount := int64(750)
		remaining := int64(50)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			RefillDay:    &refillDay,
		})

		testDateKey := fmt.Sprintf("2025-02-28-test-lastday-%s", uid.New("", 8))
		resp, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetKeysRefilled(), int32(1))

		key, err := h.DB.FindKeyByID(h.Ctx, keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, refillAmount, key.RemainingRequests.Int64)
	})

	t.Run("skips deleted keys", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		refillAmount := int64(1000)
		remaining := int64(100)
		keyResp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID:  ws.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    &remaining,
			RefillAmount: &refillAmount,
			Deleted:      true,
		})

		testDateKey := fmt.Sprintf("%s-test-deleted-%s", dateKey, uid.New("", 8))
		_, err := callRunKeyRefill(h, testDateKey)
		require.NoError(t, err)

		key, err := h.DB.FindKeyByID(h.Ctx, keyResp.KeyID)
		require.NoError(t, err)
		require.Equal(t, remaining, key.RemainingRequests.Int64)
	})
}

func callRunKeyRefill(h *harness.Harness, dateKey string) (*hydrav1.RunKeyRefillResponse, error) {
	client := hydrav1.NewCronServiceIngressClient(h.Restate, dateKey)
	return client.RunKeyRefill().Request(h.Ctx, &hydrav1.RunKeyRefillRequest{})
}
