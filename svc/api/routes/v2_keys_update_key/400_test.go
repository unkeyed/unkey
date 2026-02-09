package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
)

func TestUpdateKeyInvalidRefillConfig(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create API using helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Create key using helper
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("test"),
	})

	t.Run("reject invalid refill config", func(t *testing.T) {
		t.Parallel()
		req := handler.Request{
			KeyId: keyResponse.KeyID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
				Remaining: nullable.NewNullableWithValue(int64(100)),
				Refill: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsRefill{
					Interval:  openapi.UpdateKeyCreditsRefillIntervalDaily,
					Amount:    100,
					RefillDay: ptr.P(int(4)), // Invalid: can't set refillDay for daily

				}),
			}),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "must not be provided when the refill interval is")
	})
}
