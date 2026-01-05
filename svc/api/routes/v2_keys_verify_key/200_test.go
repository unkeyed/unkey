package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
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
			KeySpaceID:  api.KeyAuthID.String,
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
			KeySpaceID:  api.KeyAuthID.String,
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
			KeySpaceID:  api.KeyAuthID.String,
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
				KeySpaceID:  api.KeyAuthID.String,
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
				KeySpaceID:  api.KeyAuthID.String,
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
				KeySpaceID:  api.KeyAuthID.String,
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
				KeySpaceID:  api.KeyAuthID.String,
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
			require.EqualValues(t, *res.Body.Data.Credits, int32(5), "Key should have 5 credits remaining but got %d", *res.Body.Data.Credits)
		})

		t.Run("allow credits 0 even when remaining 0", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
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
		ipWhitelistApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID, IpWhitelist: "123.123.123.123"})
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  ipWhitelistApi.KeyAuthID.String,
		})

		req := handler.Request{
			Key: key.Key,
		}

		// First request with wrong IP - should be forbidden
		headersWithWrongIP := http.Header{
			"Content-Type":    {"application/json"},
			"Authorization":   {fmt.Sprintf("Bearer %s", rootKey)},
			"X-Forwarded-For": {"192.168.1.1"},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersWithWrongIP, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.FORBIDDEN, res.Body.Data.Code, "Key should be forbidden but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)

		// Second request with correct IP - should be valid
		headersWithCorrectIP := http.Header{
			"Content-Type":    {"application/json"},
			"Authorization":   {fmt.Sprintf("Bearer %s", rootKey)},
			"X-Forwarded-For": {"123.123.123.123"},
		}
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headersWithCorrectIP, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
	})

	t.Run("key with permissions", func(t *testing.T) {
		t.Run("with role permission valid", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
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
			require.Len(t, res.Body.Data.Permissions, 1, "Key should be have a single permission attached")
		})

		t.Run("with direct permission valid", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
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
			require.Len(t, res.Body.Data.Permissions, 1, "Key should be have a single permission attached")
		})

		t.Run("missing permissions", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
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
				KeySpaceID:  api.KeyAuthID.String,
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

		t.Run("with wildcard permission query", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Permissions: []seed.CreatePermissionRequest{
					{
						Name:        "All endpoints",
						Slug:        "api.*",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
					{
						Name:        "api.edit",
						Slug:        "api.edit",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
				},
			})

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P("api.* OR api.edit"),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
		})

		t.Run("with colon namespace permissions", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Permissions: []seed.CreatePermissionRequest{
					{
						Name:        "System Admin Read",
						Slug:        "system:admin:read",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
					{
						Name:        "System Admin Write",
						Slug:        "system:admin:write",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
					{
						Name:        "User Basic Read",
						Slug:        "user:basic:read",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
				},
			})

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P("system:admin:read AND (system:admin:write OR user:basic:read)"),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
			require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
		})

		t.Run("with mixed characters including colons and asterisks", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Permissions: []seed.CreatePermissionRequest{
					{
						Name:        "All System Admin",
						Slug:        "system:admin:*",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
					{
						Name:        "API v2 Test Read",
						Slug:        "api_v2-test:read",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
				},
			})

			req := handler.Request{
				Key:         key.Key,
				Permissions: ptr.P("system:admin:* AND api_v2-test:read"),
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
				KeySpaceID:  api.KeyAuthID.String,
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
			KeySpaceID:  api.KeyAuthID.String,
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
			KeySpaceID:  api.KeyAuthID.String,
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
			KeySpaceID:  api.KeyAuthID.String,
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
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  ptr.P(identity.ID),
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
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  ptr.P(identity.ID),
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
		require.Len(t, res.Body.Data.Roles, 1, "Key should have 1 role")
		require.Len(t, res.Body.Data.Permissions, 3, "Key should have 3 permissions")
		require.EqualValues(t, openapi.Identity{Id: identity.ID, ExternalId: externalId, Meta: meta, Ratelimits: nil}, ptr.SafeDeref(res.Body.Data.Identity))
		require.Equal(t, keyName, res.Body.Data.Name, "Key should have the same name")
	})

	t.Run("root key with wrong api permissions", func(t *testing.T) {
		api2 := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api2.KeyAuthID.String,
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
}
