package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
)

func TestPreconditionFailed(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	validHeaders := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("with identity - missing ratelimit", func(t *testing.T) {
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  "test-missing-ratelimit",
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "existing-ratelimit",
					WorkspaceID: workspace.ID,
					Duration:    60_000,
					Limit:       100,
				},
			},
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  ptr.P(identity.ID),
		})

		req := handler.Request{
			Key: key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
				{Name: "does-not-exist"},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)

		// Should contain useful error message about missing ratelimit for key and identity
		expectedMsg := fmt.Sprintf("ratelimit 'does-not-exist' was requested but does not exist for key '%s' nor identity", key.KeyID)
		require.Contains(t, res.Body.Error.Detail, expectedMsg)
		require.Contains(t, res.Body.Error.Detail, identity.ID)
		require.Contains(t, res.Body.Error.Detail, "test-missing-ratelimit")
	})

	t.Run("without identity - missing ratelimit", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "existing-ratelimit",
					WorkspaceID: workspace.ID,
					Duration:    60_000,
					Limit:       100,
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
				{Name: "does-not-exist"},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)

		// Should contain error message indicating no identity connected
		expectedMsg := fmt.Sprintf("ratelimit 'does-not-exist' was requested but does not exist for key '%s' and there is no identity connected", key.KeyID)
		require.Contains(t, res.Body.Error.Detail, expectedMsg)
	})

	t.Run("invalid ratelimit configuration", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
		})

		req := handler.Request{
			Key: key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
				{
					Name: "missing_config",
					Cost: ptr.P(4),
					// Missing limit and duration for custom ratelimit
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})
}
