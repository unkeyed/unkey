package root_keys

import (
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/integration"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TestIPAllowlistLifecycle checks persisted network restrictions through the full
// HTTP authentication and root-key management stack, including warm-cache requests.
func TestIPAllowlistLifecycle(t *testing.T) {
	h := integration.New(t, integration.Config{
		NumNodes:          1,
		TrustedProxyCIDRs: []string{"127.0.0.0/8", "::1/128"},
	})
	workspace := h.Resources().UserWorkspace
	require.NoError(t, db.Query.UpdateIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateIPAllowlistEnabledParams{
		WorkspaceID: workspace.ID,
		Enabled:     true,
	}))
	permissions := []string{
		"unkey:v1:" + workspace.ID + ":rootKeys/*#write",
		"unkey:v1:" + workspace.ID + ":rootKeys/*#read",
	}
	admin := h.Seed.CreateUnkeyRootKey(t.Context(), seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: permissions,
	})
	addr := h.GetClusterAddrs()[0]
	policy := []string{"192.0.2.128/25", "2001:db8:1234::/48", "203.0.113.7"}
	parent := callIPAllowlist[openapi.V2RootKeysCreateKeyRequestBody, openapi.V2RootKeysCreateKeyResponseBody](t, addr, admin.Key, "192.0.2.129", "createKey", openapi.V2RootKeysCreateKeyRequestBody{
		Permissions: permissions,
		IpAllowlist: &policy,
	}, http.StatusOK)
	assertIPAllowlist(t, addr, admin.Key, parent.Data.KeyId, policy)
	for _, tc := range []struct {
		ip     string
		status int
	}{
		{"192.0.2.128", http.StatusOK},
		{"192.0.2.255", http.StatusOK},
		{"192.0.2.127", http.StatusForbidden},
		{"192.0.3.0", http.StatusForbidden},
		{"::ffff:192.0.2.129", http.StatusOK},
		{"::ffff:192.0.2.127", http.StatusForbidden},
		{"2001:db8:1234:ffff::1", http.StatusOK},
		{"2001:db8:1235::1", http.StatusForbidden},
		{"203.0.113.7", http.StatusOK},
		{"203.0.113.8", http.StatusForbidden},
		{"192.0.2.129, 198.51.100.1", http.StatusForbidden},
		{"198.51.100.1, 192.0.2.129", http.StatusOK},
		{"invalid", http.StatusForbidden},
		{"", http.StatusForbidden},
	} {
		t.Run(tc.ip, func(t *testing.T) {
			for range 2 {
				callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, parent.Data.Key, tc.ip, "listKeys", openapi.V2RootKeysListKeysRequestBody{}, tc.status)
			}
		})
	}

	child := callIPAllowlist[openapi.V2RootKeysCreateKeyRequestBody, openapi.V2RootKeysCreateKeyResponseBody](t, addr, parent.Data.Key, "192.0.2.129", "createKey", openapi.V2RootKeysCreateKeyRequestBody{
		Permissions: permissions,
	}, http.StatusOK)
	assertIPAllowlist(t, addr, admin.Key, child.Data.KeyId, policy)
	for _, broader := range [][]string{{}, {"192.0.2.0/24"}} {
		callIPAllowlist[openapi.V2RootKeysCreateKeyRequestBody, openapi.ForbiddenErrorResponse](t, addr, parent.Data.Key, "192.0.2.129", "createKey", openapi.V2RootKeysCreateKeyRequestBody{
			Permissions: permissions,
			IpAllowlist: &broader,
		}, http.StatusForbidden)
		callIPAllowlist[openapi.V2RootKeysUpdateKeyRequestBody, openapi.ForbiddenErrorResponse](t, addr, parent.Data.Key, "192.0.2.129", "updateKey", openapi.V2RootKeysUpdateKeyRequestBody{
			KeyId:       child.Data.KeyId,
			IpAllowlist: &broader,
		}, http.StatusForbidden)
	}
	assertIPAllowlist(t, addr, admin.Key, child.Data.KeyId, policy)
	callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, child.Data.Key, "192.0.2.200", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusOK)
	narrow := []string{"192.0.2.128/26"}
	callIPAllowlist[openapi.V2RootKeysUpdateKeyRequestBody, openapi.V2RootKeysUpdateKeyResponseBody](t, addr, parent.Data.Key, "192.0.2.129", "updateKey", openapi.V2RootKeysUpdateKeyRequestBody{
		KeyId:       child.Data.KeyId,
		IpAllowlist: &narrow,
	}, http.StatusOK)
	assertIPAllowlist(t, addr, admin.Key, child.Data.KeyId, narrow)
	callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, child.Data.Key, "192.0.2.200", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusForbidden)
	callIPAllowlist[openapi.V2RootKeysUpdateKeyRequestBody, openapi.V2RootKeysUpdateKeyResponseBody](t, addr, parent.Data.Key, "192.0.2.129", "updateKey", openapi.V2RootKeysUpdateKeyRequestBody{
		KeyId: child.Data.KeyId,
		Name:  nullable.NewNullableWithValue("renamed"),
	}, http.StatusOK)
	assertIPAllowlist(t, addr, admin.Key, child.Data.KeyId, narrow)
	rotated := callIPAllowlist[openapi.V2RootKeysRerollKeyRequestBody, openapi.V2RootKeysRerollKeyResponseBody](t, addr, parent.Data.Key, "192.0.2.129", "rerollKey", openapi.V2RootKeysRerollKeyRequestBody{
		KeyId:      child.Data.KeyId,
		Expiration: nullable.NewNullNullable[int64](),
	}, http.StatusOK)
	assertIPAllowlist(t, addr, admin.Key, rotated.Data.KeyId, narrow)
	callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, rotated.Data.Key, "192.0.2.129", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusOK)
	callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, rotated.Data.Key, "192.0.2.200", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusForbidden)
	callIPAllowlist[openapi.V2RootKeysUpdateKeyRequestBody, openapi.V2RootKeysUpdateKeyResponseBody](t, addr, admin.Key, "198.51.100.1", "updateKey", openapi.V2RootKeysUpdateKeyRequestBody{
		KeyId:       rotated.Data.KeyId,
		IpAllowlist: &[]string{},
	}, http.StatusOK)
	assertIPAllowlist(t, addr, admin.Key, rotated.Data.KeyId, []string{})
	callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, rotated.Data.Key, "198.51.100.1", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusOK)
}

// TestIPAllowlistDisabledAndUntrustedProxy ensures feature disablement does not
// remove existing restrictions and direct clients cannot forge their source IP.
func TestIPAllowlistDisabledAndUntrustedProxy(t *testing.T) {
	h := integration.New(t, integration.Config{NumNodes: 1})
	workspace := h.Resources().UserWorkspace
	permissions := []string{
		"unkey:v1:" + workspace.ID + ":rootKeys/*#write",
		"unkey:v1:" + workspace.ID + ":rootKeys/*#read",
	}
	admin := h.Seed.CreateUnkeyRootKey(t.Context(), seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: permissions,
	})
	remote := h.Seed.CreateUnkeyRootKey(t.Context(), seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: permissions,
		IpAllowlist: []string{"192.0.2.0/24"},
	})
	local := h.Seed.CreateUnkeyRootKey(t.Context(), seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: permissions,
		IpAllowlist: []string{"127.0.0.0/8", "::1/128"},
	})
	addr := h.GetClusterAddrs()[0]
	for range 2 {
		callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.ForbiddenErrorResponse](t, addr, remote.Key, "192.0.2.129", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusForbidden)
		callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, local.Key, "192.0.2.129", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusOK)
	}
	callIPAllowlist[openapi.V2RootKeysCreateKeyRequestBody, openapi.ForbiddenErrorResponse](t, addr, admin.Key, "", "createKey", openapi.V2RootKeysCreateKeyRequestBody{
		Permissions: permissions,
		IpAllowlist: &[]string{"192.0.2.0/24"},
	}, http.StatusForbidden)
	callIPAllowlist[openapi.V2RootKeysUpdateKeyRequestBody, openapi.ForbiddenErrorResponse](t, addr, admin.Key, "", "updateKey", openapi.V2RootKeysUpdateKeyRequestBody{
		KeyId:       remote.KeyID,
		IpAllowlist: &[]string{},
	}, http.StatusForbidden)
	assertIPAllowlist(t, addr, admin.Key, remote.KeyID, []string{"192.0.2.0/24"})
	rotated := callIPAllowlist[openapi.V2RootKeysRerollKeyRequestBody, openapi.V2RootKeysRerollKeyResponseBody](t, addr, admin.Key, "", "rerollKey", openapi.V2RootKeysRerollKeyRequestBody{
		KeyId:      remote.KeyID,
		Expiration: nullable.NewNullNullable[int64](),
	}, http.StatusOK)
	assertIPAllowlist(t, addr, admin.Key, rotated.Data.KeyId, []string{"192.0.2.0/24"})
	callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.ForbiddenErrorResponse](t, addr, rotated.Data.Key, "192.0.2.129", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusForbidden)
}

func callIPAllowlist[Req, Res any](t *testing.T, addr, key, ip, operation string, request Req, status int) Res {
	t.Helper()
	res, err := integration.CallNode[Req, Res](t, addr, http.MethodPost, "/v2/rootKeys."+operation, http.Header{
		"Authorization":   {"Bearer " + key},
		"Content-Type":    {"application/json"},
		"X-Forwarded-For": {ip},
	}, request)
	require.NoError(t, err)
	require.Equal(t, status, res.Status, "%s: %s", operation, res.RawBody)
	return res.Body
}

func assertIPAllowlist(t *testing.T, addr, admin, keyID string, expected []string) {
	t.Helper()
	res := callIPAllowlist[openapi.V2RootKeysListKeysRequestBody, openapi.V2RootKeysListKeysResponseBody](t, addr, admin, "192.0.2.129", "listKeys", openapi.V2RootKeysListKeysRequestBody{}, http.StatusOK)
	for _, key := range res.Data {
		if key.KeyId == keyID {
			require.Equal(t, expected, key.IpAllowlist)
			return
		}
	}
	require.Fail(t, "root key missing from list", "%s", keyID)
}
