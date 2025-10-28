package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_credits"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestKeyUpdateCreditsBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  uid.New(""),
	})

	// Create root key with read permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.update_identity")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing identity", func(t *testing.T) {
		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/identities.updateCredits' failed to validate schema")
	})

	t.Run("empty identity string", func(t *testing.T) {
		req := handler.Request{
			Identity: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("increment with null throws error", func(t *testing.T) {
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Increment,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("can't increment unlimited key", func(t *testing.T) {
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(int64(1)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("decrement with null throws error", func(t *testing.T) {
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Decrement,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("can't decrement unlimited key", func(t *testing.T) {
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(int64(1)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("negative increment value", func(t *testing.T) {
		// Create identity with credits
		identityWithCredits := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

		req := handler.Request{
			Identity:  identityWithCredits.ID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(int64(-10)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/identities.updateCredits' failed to validate schema")
	})

	t.Run("negative decrement value", func(t *testing.T) {
		// Create identity with credits
		identityWithCredits := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

		req := handler.Request{
			Identity:  identityWithCredits.ID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(int64(-10)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/identities.updateCredits' failed to validate schema")
	})

	t.Run("value too large for int32", func(t *testing.T) {
		// Create identity with credits
		identityWithCredits := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

		// Try to set value larger than max int32
		req := handler.Request{
			Identity:  identityWithCredits.ID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(int64(2147483648)), // max int32 + 1
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		// This should either return 400 or handle gracefully
		// The actual behavior depends on implementation
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("set with null value", func(t *testing.T) {
		// Create identity with credits
		identityWithCredits := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

		// Setting to null is actually valid (means unlimited)
		// This test verifies it doesn't return 400
		nullValue := nullable.Nullable[int64]{}
		nullValue.SetNull()
		req := handler.Request{
			Identity:  identityWithCredits.ID,
			Operation: openapi.Set,
			Value:     nullValue,
		}

		// This should succeed (not return 400)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
	})
}
