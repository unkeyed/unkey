package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestMultiLimit(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("without identities", func(t *testing.T) {
		t.Run("returns valid with multiple limits", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "10/10s",
						WorkspaceID: workspace.ID,
						Duration:    10_000,
						Limit:       10,
					},
					{
						Name:        "1/1min",
						WorkspaceID: workspace.ID,
						Duration:    60_000,
						Limit:       1,
					},
				},
			})

			req := handler.Request{
				Key: key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
					{Name: "10/10s", Cost: ptr.P(4)},
					{Name: "1/1min", Cost: ptr.P(1)},
				},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid)
		})

		t.Run("returns RATE_LIMITED when one limit exceeded", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "10/10s-test",
						WorkspaceID: workspace.ID,
						Duration:    10_000,
						Limit:       10,
					},
					{
						Name:        "1/1min-test",
						WorkspaceID: workspace.ID,
						Duration:    60_000,
						Limit:       1,
					},
				},
			})

			req := handler.Request{
				Key: key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
					{Name: "10/10s-test", Cost: ptr.P(4)},
					{Name: "1/1min-test", Cost: ptr.P(2)}, // Exceeds limit of 1
				},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid)
		})
	})

	t.Run("with identity - key precedence over identity", func(t *testing.T) {
		t.Run("key limits take precedence and pass", func(t *testing.T) {
			identity := h.CreateIdentity(seed.CreateIdentityRequest{
				WorkspaceID: workspace.ID,
				ExternalID:  "test-precedence-pass",
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "limit1",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       1, // Identity has restrictive limit
					},
					{
						Name:        "limit2",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       10,
					},
				},
			})

			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				IdentityID:  ptr.P(identity.ID),
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "limit1",
						WorkspaceID: workspace.ID,
						Duration:    10_000,
						Limit:       4, // Key has more permissive limit
					},
				},
			})

			// Should pass 3 times due to key limit being 4
			for i := 0; i < 3; i++ {
				req := handler.Request{

					Key:        key.Key,
					Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "limit1"}, {Name: "limit2"}},
				}

				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, 200, res.Status)
				require.NotNil(t, res.Body)
				require.Equal(t, openapi.VALID, res.Body.Data.Code)
				require.True(t, res.Body.Data.Valid)
			}
		})

		t.Run("key limits take precedence and reject", func(t *testing.T) {
			identity := h.CreateIdentity(seed.CreateIdentityRequest{
				WorkspaceID: workspace.ID,
				ExternalID:  "test-precedence-reject",
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "limit1-reject",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       10, // Identity has permissive limit
					},
					{
						Name:        "limit2-reject",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       10,
					},
				},
			})

			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				IdentityID:  ptr.P(identity.ID),
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "limit1-reject",
						WorkspaceID: workspace.ID,
						Duration:    10_000,
						Limit:       1, // Key has restrictive limit
					},
				},
			})

			// First request should pass
			req := handler.Request{

				Key:        key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "limit1-reject"}, {Name: "limit2-reject"}},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid)

			// Second request should be rate limited due to key limit
			res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid)
		})

		t.Run("fallback identity limits still reject", func(t *testing.T) {
			identity := h.CreateIdentity(seed.CreateIdentityRequest{
				WorkspaceID: workspace.ID,
				ExternalID:  "test-fallback",
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "limit1-fallback",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       10,
					},
					{
						Name:        "limit2-fallback",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       2, // This will be the fallback limit
					},
				},
			})

			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				IdentityID:  ptr.P(identity.ID),
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "limit1-fallback",
						WorkspaceID: workspace.ID,
						Duration:    10_000,
						Limit:       4, // Key limit for limit1
					},
				},
			})

			// Should pass twice (limit2 has limit of 2)
			for i := 0; i < 2; i++ {
				req := handler.Request{

					Key:        key.Key,
					Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "limit1-fallback"}, {Name: "limit2-fallback"}},
				}

				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, 200, res.Status)
				require.NotNil(t, res.Body)
				require.Equal(t, openapi.VALID, res.Body.Data.Code)
				require.True(t, res.Body.Data.Valid)
			}

			// Third request should be rate limited by limit2 (identity fallback)
			req := handler.Request{

				Key:        key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "limit1-fallback"}, {Name: "limit2-fallback"}},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid)
		})
	})

	t.Run("with identity - shared rate limits across keys", func(t *testing.T) {
		t.Run("rate limit is shared across multiple keys", func(t *testing.T) {
			identity := h.CreateIdentity(seed.CreateIdentityRequest{
				WorkspaceID: workspace.ID,
				ExternalID:  "test-shared",
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "100per10m",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       5, // Small limit for testing
					},
				},
			})

			// Create multiple keys with same identity
			key1 := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				IdentityID:  ptr.P(identity.ID),
			})

			key2 := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				IdentityID:  ptr.P(identity.ID),
			})

			// Use up some quota with key1
			for i := 0; i < 3; i++ {
				req := handler.Request{

					Key:        key1.Key,
					Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "100per10m"}},
				}

				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, 200, res.Status)
				require.NotNil(t, res.Body)
				require.Equal(t, openapi.VALID, res.Body.Data.Code)
				require.True(t, res.Body.Data.Valid)
			}

			// key2 should only have 2 requests left due to shared limit
			for i := 0; i < 2; i++ {
				req := handler.Request{

					Key:        key2.Key,
					Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "100per10m"}},
				}

				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, 200, res.Status)
				require.NotNil(t, res.Body)
				require.Equal(t, openapi.VALID, res.Body.Data.Code)
				require.True(t, res.Body.Data.Valid)
			}

			// Next request with key2 should be rate limited
			req := handler.Request{

				Key:        key2.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "100per10m"}},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid)
		})
	})

	t.Run("without specifying ratelimits in request", func(t *testing.T) {
		t.Run("should use auto-applied default limit", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "default-auto",
						WorkspaceID: workspace.ID,
						Duration:    20_000,
						Limit:       1,
						AutoApply:   true,
					},
				},
			})

			// First request should pass
			req := handler.Request{
				Key: key.Key,
				// No ratelimits specified - should use auto-applied
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid)

			// Second request should be rate limited
			res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid)
		})
	})

	t.Run("falls back to identity limits", func(t *testing.T) {
		t.Run("should reject after identity limit hit", func(t *testing.T) {
			identity := h.CreateIdentity(seed.CreateIdentityRequest{
				WorkspaceID: workspace.ID,
				ExternalID:  "test-identity-fallback",
				Ratelimits: []seed.CreateRatelimitRequest{
					{
						Name:        "tokens-identity",
						WorkspaceID: workspace.ID,
						Duration:    10_000,
						Limit:       10,
					},
					{
						Name:        "10_per_10m-identity",
						WorkspaceID: workspace.ID,
						Duration:    600_000,
						Limit:       10,
					},
				},
			})

			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				IdentityID:  ptr.P(identity.ID),
			})

			// First request with cost 4 should pass
			req1 := handler.Request{
				Key: key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
					{Name: "tokens-identity", Cost: ptr.P(4)},
					{Name: "10_per_10m-identity"},
				},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req1)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid)

			// Second request with cost 6 should pass (total 10)
			req2 := handler.Request{
				Key: key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
					{Name: "tokens-identity", Cost: ptr.P(6)},
					{Name: "10_per_10m-identity"},
				},
			}

			res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req2)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid)

			// Third request with cost 1 should be rate limited (would be 11 total)
			req3 := handler.Request{
				Key: key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
					{Name: "tokens-identity", Cost: ptr.P(1)},
					{Name: "10_per_10m-identity"},
				},
			}

			res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req3)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid)
		})
	})
}
