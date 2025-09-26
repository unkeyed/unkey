package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_whoami"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestGetKeyNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("nonexistent raw key", func(t *testing.T) {
		nonexistentKey := uid.New("api")
		req := handler.Request{
			Key: nonexistentKey,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "We could not find the requested key")
	})
}

func TestGetKeyForbidden(t *testing.T) {

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	// Create API for testing
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            keyAuthID,
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false, String: ""},
		DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
	})
	require.NoError(t, err)

	apiID := uid.New(uid.APIPrefix)
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api",
		WorkspaceID: h.Resources().UserWorkspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create another API for cross-API testing
	otherKeyAuthID := uid.New(uid.KeyAuthPrefix)
	err = db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            otherKeyAuthID,
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false, String: ""},
		DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
	})
	require.NoError(t, err)

	otherApiID := uid.New(uid.APIPrefix)
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          otherApiID,
		Name:        "other-api",
		WorkspaceID: h.Resources().UserWorkspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: otherKeyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create another Workspace for cross-API testing
	otherWorkspace := h.CreateWorkspace()

	otherWsKeyAuthID := uid.New(uid.KeyAuthPrefix)
	err = db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            otherWsKeyAuthID,
		WorkspaceID:   otherWorkspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false, String: ""},
		DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
	})
	require.NoError(t, err)

	otherWsApiID := uid.New(uid.APIPrefix)
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          otherWsApiID,
		Name:        "test-api",
		WorkspaceID: otherWorkspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: otherWsKeyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create a test key
	keyID := uid.New(uid.KeyPrefix)
	keyString := "test_" + uid.New("")
	err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
		ID:                keyID,
		KeyringID:         keyAuthID,
		Hash:              hash.Sha256(keyString),
		Start:             keyString[:4],
		WorkspaceID:       h.Resources().UserWorkspace.ID,
		ForWorkspaceID:    sql.NullString{Valid: false},
		Name:              sql.NullString{Valid: true, String: "Test Key"},
		CreatedAtM:        time.Now().UnixMilli(),
		Enabled:           true,
		IdentityID:        sql.NullString{Valid: false},
		Meta:              sql.NullString{Valid: false},
		Expires:           sql.NullTime{Valid: false},
		RemainingRequests: sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	req := handler.Request{
		Key: keyString,
	}

	t.Run("no permissions", func(t *testing.T) {
		// Create root key with no permissions
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		// We  do not want to leak the existence of a key to someone who doesn't have permissions,
		// so we return a 404 here
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("wrong permission - has create but not read", func(t *testing.T) {
		// Create root key with read permission instead of create
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		// We  do not want to leak the existence of a key to someone who doesn't have permissions,
		// so we return a 404 here
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("cross workspace access", func(t *testing.T) {
		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Create a root key for the different workspace with full permissions
		rootKey := h.CreateRootKey(differentWorkspace.ID, "api.*.read_key", "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("cross api access", func(t *testing.T) {
		// Create root key with read permission for a single api
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.read_key", otherApiID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		// We  do not want to leak the existence of a key to someone who doesn't have permissions,
		// so we return a 404 here
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
	})
}
