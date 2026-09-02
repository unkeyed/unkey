package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_keys_create_key"
)

// TestCreateKeyStoresVersion1Format guarantees that v3 creates the key in the
// requested keyspace and stores each searchable plaintext field.
func TestCreateKeyStoresVersion1Format(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	prefix := "abcdefghijklmnop"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		DefaultPrefix: &prefix,
	})
	rootKey := h.CreateRootKey(
		h.Resources().UserWorkspace.ID,
		createKeyPermission(api.ID),
	)

	res := testutil.CallRoute[handler.Request, handler.Response](
		h,
		route,
		authorizedHeaders(rootKey),
		handler.Request{Keyspace: api.KeyAuthID.String},
	)
	require.Equal(t, http.StatusOK, res.Status, "response: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.Regexp(
		t,
		regexp.MustCompile(`^abcdefghijklmnop_[1-9A-HJ-NP-Za-km-z]{8}unkeyv1[1-9A-HJ-NP-Za-km-z]{42}$`),
		res.Body.Data.Key,
	)

	key, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, api.KeyAuthID.String, key.KeyAuthID)
	require.Equal(t, hash.Sha256(res.Body.Data.Key), key.Hash)
	require.Equal(t, prefix, key.Prefix)
	require.Equal(t, res.Body.Data.Key[len(prefix)+1:len(prefix)+5], key.Start)
	require.Equal(t, res.Body.Data.Key[len(res.Body.Data.Key)-4:], key.End)

	auditLogs := h.FindAuditLogsByTargetID(t.Context(), t, api.KeyAuthID.String)
	require.Len(t, auditLogs, 1)
	require.Contains(t, auditLogs[0].Targets, auditlog.EventTarget{
		Type: string(auditlog.KeySpaceResourceType),
		ID:   api.KeyAuthID.String,
	})
}

// TestCreateKeyUsesKeyspacePrefix guarantees that the route uses the keyspace default prefix.
func TestCreateKeyUsesKeyspacePrefix(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	defaultPrefix := "prod"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		DefaultPrefix: &defaultPrefix,
	})
	rootKey := h.CreateRootKey(
		h.Resources().UserWorkspace.ID,
		createAnyKeyPermission(),
	)

	res := testutil.CallRoute[handler.Request, handler.Response](
		h,
		route,
		authorizedHeaders(rootKey),
		handler.Request{Keyspace: api.KeyAuthID.String},
	)
	require.Equal(t, http.StatusOK, res.Status, "response: %s", res.RawBody)
	require.True(t, strings.HasPrefix(res.Body.Data.Key, defaultPrefix+"_"))
}

// TestCreateKeyWithoutDefaultPrefix guarantees that a keyspace does not need a prefix.
func TestCreateKeyWithoutDefaultPrefix(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})
	rootKey := h.CreateRootKey(
		h.Resources().UserWorkspace.ID,
		createKeyPermission(api.ID),
	)

	res := testutil.CallRoute[handler.Request, handler.Response](
		h,
		route,
		authorizedHeaders(rootKey),
		handler.Request{Keyspace: api.KeyAuthID.String},
	)
	require.Equal(t, http.StatusOK, res.Status, "response: %s", res.RawBody)
	require.Regexp(
		t,
		regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{8}unkeyv1[1-9A-HJ-NP-Za-km-z]{42}$`),
		res.Body.Data.Key,
	)

	key, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Empty(t, key.Prefix)
	require.Equal(t, res.Body.Data.Key[:4], key.Start)
}

// TestCreateRecoverableKeyUsesLegacyPermissions guarantees that API-scoped
// create and encrypt permissions authorize recoverable V3 key creation.
func TestCreateRecoverableKeyUsesLegacyPermissions(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		EncryptedKeys: true,
	})
	rootKey := h.CreateRootKey(
		h.Resources().UserWorkspace.ID,
		createKeyPermission(api.ID),
		encryptKeyPermission(api.ID),
	)

	res := testutil.CallRoute[handler.Request, handler.Response](
		h,
		route,
		authorizedHeaders(rootKey),
		handler.Request{
			Keyspace:    api.KeyAuthID.String,
			Recoverable: ptr.P(true),
		},
	)
	require.Equal(t, http.StatusOK, res.Status, "response: %s", res.RawBody)

	encryption, err := db.Query.FindKeyEncryptionByKeyID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, res.Body.Data.KeyId, encryption.KeyID)
}

// TestCreateKeyRejectsInvalidFormatOptions guarantees that v3 requires a
// keyspace and does not accept per-key format options.
func TestCreateKeyRejectsInvalidFormatOptions(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})
	rootKey := h.CreateRootKey(
		h.Resources().UserWorkspace.ID,
		createKeyPermission(api.ID),
	)

	testCases := []struct {
		name string
		req  map[string]any
	}{
		{name: "missing keyspace", req: map[string]any{}},
		{name: "keyspaceId", req: map[string]any{"keyspaceId": api.KeyAuthID.String}},
		{name: "prefix", req: map[string]any{"keyspace": api.KeyAuthID.String, "prefix": "prod"}},
		{name: "byte length", req: map[string]any{"keyspace": api.KeyAuthID.String, "byteLength": 32}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res := testutil.CallRoute[map[string]any, openapi.BadRequestErrorResponse](
				h,
				route,
				authorizedHeaders(rootKey),
				testCase.req,
			)
			require.Equal(t, http.StatusBadRequest, res.Status, "response: %s", res.RawBody)
		})
	}
}

// TestCreateKeyRequiresAuthentication guarantees that invalid credentials
// cannot create a key.
func TestCreateKeyRequiresAuthentication(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
		h,
		route,
		http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid"},
		},
		handler.Request{Keyspace: "ks_123"},
	)
	require.Equal(t, http.StatusUnauthorized, res.Status, "response: %s", res.RawBody)
}

// TestCreateKeyMasksUnauthorizedKeyspaces guarantees that the route returns the
// same 404 for a missing grant and a keyspace from another workspace.
func TestCreateKeyMasksUnauthorizedKeyspaces(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	otherWorkspace := h.CreateWorkspace()
	otherAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: otherWorkspace.ID})

	testCases := []struct {
		name       string
		keyspaceID string
		rootKey    string
	}{
		{
			name:       "missing grant",
			keyspaceID: api.KeyAuthID.String,
			rootKey:    h.CreateRootKey(workspaceID),
		},
		{
			name:       "different workspace",
			keyspaceID: otherAPI.KeyAuthID.String,
			rootKey:    h.CreateRootKey(workspaceID, createAnyKeyPermission()),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h,
				route,
				authorizedHeaders(testCase.rootKey),
				handler.Request{Keyspace: testCase.keyspaceID},
			)
			require.Equal(t, http.StatusNotFound, res.Status, "response: %s", res.RawBody)
			require.NotContains(t, res.RawBody, testCase.keyspaceID)
		})
	}
}

// TestCreateKeyRejectsDeletedResources prevents the route from returning keys
// that verification will reject.
func TestCreateKeyRejectsDeletedResources(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := newRoute(t, h)
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	deletedAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	deletedKeyspace := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	now := sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}

	err := db.Query.SoftDeleteApi(t.Context(), h.DB.RW(), db.SoftDeleteApiParams{
		ApiID: deletedAPI.ID,
		Now:   now,
	})
	require.NoError(t, err)

	_, err = h.DB.RW().ExecContext(
		t.Context(),
		"UPDATE key_auth SET deleted_at_m = ? WHERE id = ?",
		now.Int64,
		deletedKeyspace.KeyAuthID.String,
	)
	require.NoError(t, err)

	rootKey := h.CreateRootKey(workspaceID, createAnyKeyPermission())
	testCases := []struct {
		name       string
		keyspaceID string
	}{
		{name: "deleted API", keyspaceID: deletedAPI.KeyAuthID.String},
		{name: "deleted keyspace", keyspaceID: deletedKeyspace.KeyAuthID.String},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h,
				route,
				authorizedHeaders(rootKey),
				handler.Request{Keyspace: testCase.keyspaceID},
			)
			require.Equal(t, http.StatusNotFound, res.Status, "response: %s", res.RawBody)
		})
	}
}

func newRoute(t *testing.T, h *testutil.Harness) *handler.Handler {
	t.Helper()

	return &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}
}

func authorizedHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

func createKeyPermission(apiID string) string {
	return fmt.Sprintf("api.%s.create_key", apiID)
}

func createAnyKeyPermission() string {
	return "api.*.create_key"
}

func encryptKeyPermission(apiID string) string {
	return fmt.Sprintf("api.%s.encrypt_key", apiID)
}
