package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	// ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace
	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("verifies key as valid", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
		})

		req := handler.Request{
			Key: key.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
	})

	t.Run("verifies expired key as valid and then invalid", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Expires:     ptr.P(time.Now().Add(time.Second * 3)),
		})

		req := handler.Request{
			Key: key.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)

		time.Sleep(time.Second * 3)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.EXPIRED, res.Body.Data.Code, "Key should be expired but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("disabled key", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Disabled:    true,
		})

		req := handler.Request{
			Key: key.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.DISABLED, res.Body.Data.Code, "Key should be disabled but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("key with credits", func(t *testing.T) {
		t.Run("allowed default credit cost", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Remaining:   ptr.P(int32(5)),
			})

			req := handler.Request{
				Key: key.Key,
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
			require.EqualValues(t, *res.Body.Data.Credits, int32(4), "Key should have 4 credits but got %d", *res.Body.Data.Credits)
		})

		t.Run("exceeding with default credit cost", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Remaining:   ptr.P(int32(0)),
			})

			req := handler.Request{
				Key: key.Key,
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.USAGEEXCEEDED, res.Body.Data.Code, "Key should show usage exceeded but got %s", res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
			require.EqualValues(t, *res.Body.Data.Credits, int32(0), "Key should have 0 credits but got %d", *res.Body.Data.Credits)
		})

		t.Run("allowed custom credit cost", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Remaining:   ptr.P(int32(5)),
			})

			req := handler.Request{
				Key: key.Key,
				Credits: &openapi.KeysVerifyKeyCredits{
					Cost: 5,
				},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
			require.EqualValues(t, *res.Body.Data.Credits, int32(0), "Key should have 0 credits remaining but got %d", *res.Body.Data.Credits)
		})

		t.Run("exceeding with custom credit cost", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Remaining:   ptr.P(int32(5)),
			})

			req := handler.Request{
				Key: key.Key,
				Credits: &openapi.KeysVerifyKeyCredits{
					Cost: 15,
				},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.USAGEEXCEEDED, res.Body.Data.Code, "Key should be usage exceeded but got %s", res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
			require.EqualValues(t, *res.Body.Data.Credits, int32(0), "Key should have 0 credits remaining but got %d", *res.Body.Data.Credits)
		})

		t.Run("allow credits 0 even when remaining 0", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Remaining:   ptr.P(int32(0)),
			})

			req := handler.Request{
				Key: key.Key,
				Credits: &openapi.KeysVerifyKeyCredits{
					Cost: 0,
				},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be code valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
			require.EqualValues(t, *res.Body.Data.Credits, int32(0), "Key should have 0 credits remaining but got %d", *res.Body.Data.Credits)
		})
	})

	t.Run("with ip whitelist", func(t *testing.T) {
		ipWhitelistApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID, IpWhitelist: "127.0.0.1"})
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   ipWhitelistApi.KeyAuthID.String,
		})

		req := handler.Request{
			Key: key.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.FORBIDDEN, res.Body.Data.Code, "Key should be forbidden but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("key with permissions", func(t *testing.T) {
		t.Run("with role permission valid", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Roles: []seed.CreateRoleRequest{{
					Name:        "test-role",
					Description: nil,
					WorkspaceID: workspace.ID,
					Permissions: []seed.CreatePermissionRequest{{
						Name:        "domain.write",
						Slug:        "domain.write",
						Description: nil,
						WorkspaceID: workspace.ID,
					}},
				}},
			})

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P("domain.write"),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
			require.Len(t, ptr.SafeDeref(res.Body.Data.Permissions), 1, "Key should be have a single permission attached")
		})

		t.Run("with direct permission valid", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Permissions: []seed.CreatePermissionRequest{{
					Name:        "domain.read",
					Slug:        "domain.read",
					Description: nil,
					WorkspaceID: workspace.ID,
				}},
			})

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P("domain.read"),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
			require.Len(t, ptr.SafeDeref(res.Body.Data.Permissions), 1, "Key should be have a single permission attached")
		})

		t.Run("missing permissions", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
			})

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P("domain.write"),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.INSUFFICIENTPERMISSIONS, res.Body.Data.Code, "Key should be no perms but got %s", res.Body.Data.Code)
			require.False(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
		})

		t.Run("with complex permissions query", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Permissions: []seed.CreatePermissionRequest{
					{
						Name:        "api.read",
						Slug:        "api.read",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
					{
						Name:        "api.write",
						Slug:        "api.write",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
				},
			})

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P("api.read AND api.write"),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
		})

		t.Run("with large permissions query (20+ permissions)", func(t *testing.T) {
			// Create a key with 25 permissions
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Permissions: []seed.CreatePermissionRequest{
					{Name: "read.users", Slug: "read.users", WorkspaceID: workspace.ID},
					{Name: "write.users", Slug: "write.users", WorkspaceID: workspace.ID},
					{Name: "delete.users", Slug: "delete.users", WorkspaceID: workspace.ID},
					{Name: "read.posts", Slug: "read.posts", WorkspaceID: workspace.ID},
					{Name: "write.posts", Slug: "write.posts", WorkspaceID: workspace.ID},
					{Name: "delete.posts", Slug: "delete.posts", WorkspaceID: workspace.ID},
					{Name: "read.comments", Slug: "read.comments", WorkspaceID: workspace.ID},
					{Name: "write.comments", Slug: "write.comments", WorkspaceID: workspace.ID},
					{Name: "delete.comments", Slug: "delete.comments", WorkspaceID: workspace.ID},
					{Name: "read.files", Slug: "read.files", WorkspaceID: workspace.ID},
					{Name: "write.files", Slug: "write.files", WorkspaceID: workspace.ID},
					{Name: "delete.files", Slug: "delete.files", WorkspaceID: workspace.ID},
					{Name: "read.settings", Slug: "read.settings", WorkspaceID: workspace.ID},
					{Name: "write.settings", Slug: "write.settings", WorkspaceID: workspace.ID},
					{Name: "admin.users", Slug: "admin.users", WorkspaceID: workspace.ID},
					{Name: "admin.posts", Slug: "admin.posts", WorkspaceID: workspace.ID},
					{Name: "admin.system", Slug: "admin.system", WorkspaceID: workspace.ID},
					{Name: "moderate.comments", Slug: "moderate.comments", WorkspaceID: workspace.ID},
					{Name: "backup.create", Slug: "backup.create", WorkspaceID: workspace.ID},
					{Name: "backup.restore", Slug: "backup.restore", WorkspaceID: workspace.ID},
					{Name: "analytics.view", Slug: "analytics.view", WorkspaceID: workspace.ID},
					{Name: "analytics.export", Slug: "analytics.export", WorkspaceID: workspace.ID},
					{Name: "billing.view", Slug: "billing.view", WorkspaceID: workspace.ID},
					{Name: "billing.manage", Slug: "billing.manage", WorkspaceID: workspace.ID},
					{Name: "audit.view", Slug: "audit.view", WorkspaceID: workspace.ID},
				},
			})

			// Complex query with 25 permissions using AND and OR operators
			largeQuery := "(read.users OR write.users) AND (read.posts OR write.posts OR delete.posts) AND " +
				"(read.comments AND write.comments) AND (read.files OR write.files) AND " +
				"(read.settings AND write.settings) AND (admin.users OR admin.posts) AND " +
				"admin.system AND moderate.comments AND (backup.create OR backup.restore) AND " +
				"(analytics.view AND analytics.export) AND (billing.view OR billing.manage) AND " +
				"audit.view AND delete.users AND delete.comments AND delete.files"

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P(largeQuery),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
		})
	})

	t.Run("key with auto applied ratelimit", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "auto-apply",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    time.Minute.Milliseconds(),
					Limit:       1,
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "Key should be ratelimited but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("key with specified ratelimit", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "requests",
					WorkspaceID: workspace.ID,
					AutoApply:   false,
					Duration:    time.Minute.Milliseconds(),
					Limit:       1,
				},
			},
		})

		req := handler.Request{

			Key:        key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "requests"}},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "Key should be ratelimited but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("key with custom ratelimit", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
		})

		req := handler.Request{

			Key:        key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "requests", Cost: ptr.P(15), Duration: ptr.P(int(time.Minute.Milliseconds())), Limit: ptr.P(20)}},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "Key should be ratelimited but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("key with identity ratelimit", func(t *testing.T) {
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  "test-123",
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "tokens",
					WorkspaceID: workspace.ID,
					AutoApply:   false,
					Duration:    (time.Minute * 30).Milliseconds(),
					Limit:       4,
					// Will be set later
					IdentityID: nil,
					KeyID:      nil,
				},
			},
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			IdentityID:  ptr.P(identity),
		})

		req := handler.Request{

			Key:        key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "tokens", Cost: ptr.P(4)}},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "Key should be ratelimited but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("returns correct information", func(t *testing.T) {
		meta := map[string]interface{}{"key": "value"}

		raw, err := json.Marshal(meta)
		require.NoError(t, err)

		externalId := uid.New("ext")
		identity := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspace.ID, ExternalID: externalId, Meta: raw, Ratelimits: nil})
		keyName := "valid-info"

		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			IdentityID:  ptr.P(identity),
			Name:        ptr.P(keyName),
			Roles: []seed.CreateRoleRequest{{
				Name:        "read-writer",
				Description: nil,
				WorkspaceID: workspace.ID,
				Permissions: []seed.CreatePermissionRequest{{
					Name:        "domain.delete",
					Slug:        "domain.delete",
					Description: nil,
					WorkspaceID: workspace.ID,
				}, {
					Name:        "domain.edit",
					Slug:        "domain.edit",
					Description: nil,
					WorkspaceID: workspace.ID,
				}},
			}},
			Permissions: []seed.CreatePermissionRequest{{
				Name:        "domain.create",
				Slug:        "domain.create",
				Description: nil,
				WorkspaceID: workspace.ID,
			}},
		})

		req := handler.Request{
			Key: key.Key,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
		require.Len(t, ptr.SafeDeref(res.Body.Data.Roles), 1, "Key should have 1 role")
		require.Len(t, ptr.SafeDeref(res.Body.Data.Permissions), 3, "Key should have 3 permissions")
		require.EqualValues(t, openapi.Identity{ExternalId: externalId, Meta: &meta, Ratelimits: nil}, ptr.SafeDeref(res.Body.Data.Identity))
		require.Equal(t, keyName, ptr.SafeDeref(res.Body.Data.Name), "Key should have the same name")
	})

	t.Run("root key with wrong api permissions", func(t *testing.T) {
		api2 := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api2.KeyAuthID.String,
		})
		rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("api.%s.verify_key", api.ID))

		req := handler.Request{
			Key: key.Key,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.NOTFOUND, res.Body.Data.Code, "Key should be not found but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
	})

	t.Run("root key without sufficient permissions", func(t *testing.T) {
		// Create root key with insufficient permissions
		limitedRootKey := h.CreateRootKey(workspace.ID, "api.*.read") // Wrong permission

		req := handler.Request{
			Key: key.Key,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", limitedRootKey)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.NOTFOUND, res.Body.Data.Code, "Key should be not found but got %s", res.Body.Data.Code)
	})
}
