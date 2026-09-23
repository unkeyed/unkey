package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestVerifyKey_KeyspaceRejectionsDoNotConsumeQuota(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, DirectAuditLogs: h.DirectAuditLogs, KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	otherAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	otherAPIRootKey := h.CreateRootKey(workspace.ID, "api."+otherAPI.ID+".verify_key")
	otherWorkspace := h.CreateWorkspace()
	otherWorkspaceRootKey := h.CreateRootKey(otherWorkspace.ID, "api.*.verify_key")
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Remaining:   ptr.P(int64(7)),
		Ratelimits: []seed.CreateRatelimitRequest{{
			Name: "requests", WorkspaceID: workspace.ID, AutoApply: true,
			Duration: uint64(time.Minute.Milliseconds()), Limit: 5,
		}},
	})

	for _, tt := range []struct {
		name        string
		keyspaceIDs []string
		rootKey     string
	}{
		{name: "different keyspace", keyspaceIDs: []string{otherAPI.KeyAuthID.String}, rootKey: rootKey},
		{name: "exact match required", keyspaceIDs: []string{api.KeyAuthID.String + "_suffix"}, rootKey: rootKey},
		{name: "allowlist does not grant permission", keyspaceIDs: []string{api.KeyAuthID.String}, rootKey: otherAPIRootKey},
		{name: "allowlist does not cross workspaces", keyspaceIDs: []string{api.KeyAuthID.String}, rootKey: otherWorkspaceRootKey},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Content-Type": {"application/json"}, "Authorization": {"Bearer " + tt.rootKey},
			}, handler.Request{
				Key: key.Key, Keyspaces: &tt.keyspaceIDs, Credits: &openapi.KeysVerifyKeyCredits{Cost: 2},
			})
			require.Equal(t, http.StatusOK, res.Status, res.RawBody)
			require.Equal(t, openapi.V2KeysVerifyKeyResponseData{Code: openapi.NOTFOUND, Valid: false}, res.Body.Data)
		})
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type": {"application/json"}, "Authorization": {"Bearer " + rootKey},
	}, handler.Request{
		Key: key.Key, Credits: &openapi.KeysVerifyKeyCredits{Cost: 2},
		Keyspaces: ptr.P([]string{
			otherAPI.KeyAuthID.String, "ks_second", "ks_third", strings.Repeat("x", 100), api.KeyAuthID.String,
		}),
	})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.VALID, res.Body.Data.Code)
	require.True(t, res.Body.Data.Valid)
	require.Equal(t, api.KeyAuthID.String, res.Body.Data.KeyspaceId)
	require.Equal(t, ptr.P(int64(5)), res.Body.Data.Credits)
	require.Len(t, res.Body.Data.Ratelimits, 1)
	require.Equal(t, int64(4), res.Body.Data.Ratelimits[0].Remaining)
}

func TestVerifyKey_KeyspaceAllowlistHidesInvalidKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, DirectAuditLogs: h.DirectAuditLogs, KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	otherAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	headers := http.Header{
		"Content-Type": {"application/json"}, "Authorization": {"Bearer " + rootKey},
	}

	for _, state := range []struct {
		name     string
		disabled bool
		expires  *time.Time
		code     openapi.V2KeysVerifyKeyResponseDataCode
	}{
		{name: "disabled", disabled: true, code: openapi.DISABLED},
		{name: "expired", expires: ptr.P(time.UnixMilli(1)), code: openapi.EXPIRED},
	} {
		t.Run(state.name, func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String,
				Disabled: state.disabled, Expires: state.expires,
				Name: ptr.P("private key"), Meta: ptr.P(`{"private":"metadata"}`), Remaining: ptr.P(int64(7)),
			})
			for _, tt := range []struct {
				name      string
				keyspaces *[]string
				hidden    bool
			}{
				{name: "mismatch", keyspaces: ptr.P([]string{otherAPI.KeyAuthID.String}), hidden: true},
				{name: "matching", keyspaces: ptr.P([]string{api.KeyAuthID.String})},
				{name: "omitted"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
						Key: key.Key, Keyspaces: tt.keyspaces, Credits: &openapi.KeysVerifyKeyCredits{Cost: 2},
					})
					require.Equal(t, http.StatusOK, res.Status, res.RawBody)
					if tt.hidden {
						require.Equal(t, openapi.V2KeysVerifyKeyResponseData{Code: openapi.NOTFOUND, Valid: false}, res.Body.Data)
						return
					}
					require.False(t, res.Body.Data.Valid)
					require.Equal(t, state.code, res.Body.Data.Code)
					require.Equal(t, key.KeyID, res.Body.Data.KeyId)
					require.Equal(t, api.KeyAuthID.String, res.Body.Data.KeyspaceId)
					require.Equal(t, "private key", res.Body.Data.Name)
					require.Equal(t, map[string]any{"private": "metadata"}, res.Body.Data.Meta)
					require.Equal(t, ptr.P(int64(7)), res.Body.Data.Credits)
				})
			}
		})
	}
}

func TestVerifyKey_RejectsInvalidKeyspaceAllowlist(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, DirectAuditLogs: h.DirectAuditLogs, KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	headers := http.Header{
		"Content-Type": {"application/json"}, "Authorization": {"Bearer " + rootKey},
	}

	for _, tt := range []struct {
		name  string
		value string
	}{
		{name: "null is not omission", value: `null`},
		{name: "not an array", value: `"ks_test"`},
		{name: "non-string entry", value: `[123]`},
		{name: "empty allowlist", value: `[]`},
		{name: "empty ID", value: `[""]`},
		{name: "ID exceeds 100 characters", value: fmt.Sprintf(`[%q]`, strings.Repeat("x", 101))},
		{name: "more than five IDs", value: `["a","b","c","d","e","f"]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := json.RawMessage(fmt.Sprintf(`{"key":%q,"keyspaces":%s}`, key.Key, tt.value))
			res := testutil.CallRoute[json.RawMessage, openapi.BadRequestErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
			require.NotNil(t, res.Body.Error)
		})
	}
}
