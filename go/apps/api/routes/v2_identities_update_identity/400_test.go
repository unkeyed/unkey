package handler_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity/handler"
	"github.com/unkeyed/unkey/go/internal/testutil"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
		Auditlogs:   h.Auditlogs(),
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	// Set up permissions
	h.SetupPermissions(t, rootKeyID, h.DefaultWorkspaceID(), "identity.*.update_identity", true)

	t.Run("missing both identityId and externalId", func(t *testing.T) {
		req := handler.Request{
			meta: map[string]interface{}{
				"test": "value",
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "missing required field", res.Body.Error.Detail)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty identityId", func(t *testing.T) {
		emptyStr := ""
		req := handler.Request{
			identityID: &emptyStr,
			meta: map[string]interface{}{
				"test": "value",
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.updateIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty externalId", func(t *testing.T) {
		emptyStr := ""
		req := handler.Request{
			externalID: &emptyStr,
			meta: map[string]interface{}{
				"test": "value",
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.updateIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("duplicate ratelimit names", func(t *testing.T) {
		identityID := "identity_123"
		ratelimits := []openapi.Ratelimit{
			{
				Name:     "api_calls",
				Limit:    100,
				Duration: 60000,
			},
			{
				Name:     "api_calls", // Duplicate name
				Limit:    200,
				Duration: 120000,
			},
		}

		req := handler.Request{
			identityID: &identityID,
			ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/ratelimit/duplicate", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "duplicate ratelimit name")
		require.Equal(t, res.Body.Error.Detail, "api_calls")
	})

	t.Run("metadata too large", func(t *testing.T) {
		identityID := "identity_123"

		// Create a large metadata object (over 1MB)
		largeString := strings.Repeat("a", 1024*1024)
		largeMeta := map[string]interface{}{
			"large_field": largeString,
		}

		req := handler.Request{
			identityID: &identityID,
			meta:       largeMeta,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "metadata is too large")
	})
}
