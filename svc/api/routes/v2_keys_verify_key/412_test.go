package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestPreconditionFailed(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		DirectAuditLogs:  h.DirectAuditLogs,
		KeyVerifications: h.KeyVerifications,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, rootKeyGrant(workspace.ID, "*", "*", "*", "verify"))
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	validHeaders := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("with identity - missing ratelimit", func(t *testing.T) {
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(uid.TestPrefix),
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
			IdentityID:  new(identity.ID),
		})

		req := handler.Request{
			Key: key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
				{Name: "does-not-exist"},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, validHeaders, req)
		require.Equal(t, 412, res.Status)
		require.Equal(t, 412, res.Body.Error.Status)
		require.Equal(t, codes.App.Precondition.PreconditionFailed.DocsURL(), res.Body.Error.Type)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		expectedMsg := fmt.Sprintf("ratelimit 'does-not-exist' was requested but does not exist for key '%s' nor identity: '%s' external ID: '%s'", key.KeyID, identity.ID, identity.ExternalID)
		require.Equal(t, expectedMsg, res.Body.Error.Detail)
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

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, validHeaders, req)
		require.Equal(t, 412, res.Status)
		require.Equal(t, 412, res.Body.Error.Status)
		require.Equal(t, codes.App.Precondition.PreconditionFailed.DocsURL(), res.Body.Error.Type)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		expectedMsg := fmt.Sprintf("ratelimit 'does-not-exist' was requested but does not exist for key '%s' and there is no identity connected.", key.KeyID)
		require.Equal(t, expectedMsg, res.Body.Error.Detail)
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
					Cost: new(4),
					// Missing limit and duration for custom ratelimit
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, validHeaders, req)
		require.Equal(t, 412, res.Status)
		require.Equal(t, 412, res.Body.Error.Status)
		require.Equal(t, codes.App.Precondition.PreconditionFailed.DocsURL(), res.Body.Error.Type)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
