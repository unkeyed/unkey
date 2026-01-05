//go:build integration

package root_keys

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/integration"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestGetRootKey_Valid tests that a valid root key works correctly
func TestGetRootKey_Valid(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 200, res.Status)
	require.True(t, res.Body.Data.Valid)
}

// TestGetRootKey_NotFound tests that a non-existent root key returns proper error
func TestGetRootKey_NotFound(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	fakeRootKey := uid.New("test_fake_root_key")

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", fakeRootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 401, res.Status, "should return 401 for non-existent root key")
}

// TestGetRootKey_Disabled tests that a disabled root key returns proper error
func TestGetRootKey_Disabled(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	// Disable the root key
	keyHash := hash.Sha256(rootKey)
	keyRow, err := db.Query.FindKeyForVerification(ctx, h.DB.RO(), keyHash)
	require.NoError(t, err)

	err = db.Query.UpdateKey(ctx, h.DB.RW(), db.UpdateKeyParams{
		ID:               keyRow.ID,
		EnabledSpecified: 1,
		Enabled:          sql.NullBool{Bool: false, Valid: true},
		Now:              sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})
	require.NoError(t, err)

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 403, res.Status, "should return 403 for disabled root key")
}

// TestGetRootKey_Expired tests that an expired root key returns proper error
func TestGetRootKey_Expired(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	rootWorkspace := h.Resources().RootWorkspace
	rootKeySpace := h.Resources().RootKeySpace

	// Create an expired root key
	expiredTime := time.Now().Add(-1 * time.Hour)
	rootKeyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID:    rootWorkspace.ID,
		KeySpaceID:     rootKeySpace.ID,
		ForWorkspaceID: &workspace.ID,
		Expires:        &expiredTime,
	})
	rootKey := rootKeyResponse.Key

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 403, res.Status, "should return 403 for expired root key")
}

// TestGetRootKey_TargetWorkspaceDisabled tests that a root key for a disabled workspace returns proper error
func TestGetRootKey_TargetWorkspaceDisabled(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	// Disable the target workspace
	_, err := db.Query.UpdateWorkspaceEnabled(ctx, h.DB.RW(), db.UpdateWorkspaceEnabledParams{
		ID:      workspace.ID,
		Enabled: false,
	})
	require.NoError(t, err)

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 403, res.Status, "should return 403 for disabled target workspace")
}

// TestGetRootKey_RootWorkspaceDisabled tests that a root key whose owning workspace is disabled returns proper error
func TestGetRootKey_RootWorkspaceDisabled(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	rootWorkspace := h.Resources().RootWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	// Disable the root workspace (the one that owns the root key)
	_, err := db.Query.UpdateWorkspaceEnabled(ctx, h.DB.RW(), db.UpdateWorkspaceEnabledParams{
		ID:      rootWorkspace.ID,
		Enabled: false,
	})
	require.NoError(t, err)

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 403, res.Status, "should return 403 for disabled root workspace")
}

// TestGetRootKey_WorkspaceNotFound tests that a root key pointing to a non-existent workspace returns proper error
func TestGetRootKey_WorkspaceNotFound(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	rootWorkspace := h.Resources().RootWorkspace
	rootKeySpace := h.Resources().RootKeySpace
	nonExistentWorkspaceID := uid.New("ws_nonexistent")

	// Create a root key pointing to a non-existent workspace
	rootKeyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID:    rootWorkspace.ID,
		KeySpaceID:     rootKeySpace.ID,
		ForWorkspaceID: &nonExistentWorkspaceID,
	})
	rootKey := rootKeyResponse.Key

	workspace := h.Resources().UserWorkspace
	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 404, res.Status, "should return 404 for non-existent target workspace")
}

// TestGetRootKey_MissingBearer tests that missing Authorization header returns proper error
func TestGetRootKey_MissingBearer(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type": {"application/json"},
		// No Authorization header
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 400, res.Status, "should return 400 for missing authorization header")
}

// TestGetRootKey_Deleted tests that a soft-deleted root key returns proper error
func TestGetRootKey_Deleted(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	// Soft delete the root key
	keyHash := hash.Sha256(rootKey)
	keyRow, err := db.Query.FindKeyForVerification(ctx, h.DB.RO(), keyHash)
	require.NoError(t, err)

	err = db.Query.SoftDeleteKeyByID(ctx, h.DB.RW(), db.SoftDeleteKeyByIDParams{
		ID:  keyRow.ID,
		Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})
	require.NoError(t, err)

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{Key: keyResponse.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	require.Equal(t, 401, res.Status, "should return 401 for non-existent root key")
}

// TestGetRootKey_InsufficientPermissions tests that a root key without required permissions returns proper error
func TestGetRootKey_InsufficientPermissions(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace
	// Create root key without the required ratelimit permission
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.create_key")

	// Try to use ratelimit endpoint which requires ratelimit.*.create_namespace permission
	type RatelimitRequest struct {
		Namespace  string `json:"namespace"`
		Identifier string `json:"identifier"`
		Limit      int    `json:"limit"`
		Duration   int64  `json:"duration"`
	}

	type RatelimitResponse struct {
		Success bool `json:"success"`
	}

	req := RatelimitRequest{
		Namespace:  "test",
		Identifier: "test-id",
		Limit:      10,
		Duration:   60000,
	}

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[RatelimitRequest, RatelimitResponse](
		lb, "POST", "/v2/ratelimit.limit", headers, req)

	require.NoError(t, err)
	require.Equal(t, 403, res.Status, "should return 403 for insufficient permissions")
}

// TestGetRootKey_RegularKeyWithSamePermissions tests that a regular API key cannot be used
// as a root key even if it has the same permissions. This is a security test to ensure
// that only keys with ForWorkspaceID set can act as root keys.
func TestGetRootKey_RegularKeyWithSamePermissions(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	ctx := context.Background()
	h := integration.New(t, integration.Config{NumNodes: 1})

	workspace := h.Resources().UserWorkspace

	// Create an API in the workspace
	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Create a regular API key (NOT a root key - no ForWorkspaceID)
	// Give it the same permission that a root key would have
	regularKey := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Permissions: []seed.CreatePermissionRequest{
			{
				WorkspaceID: workspace.ID,
				Name:        "api.*.verify_key",
				Slug:        "api.*.verify_key",
			},
		},
	})

	// Create another key to verify
	keyToVerify := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	// Try to use the regular key as a root key in the Authorization header
	req := handler.Request{Key: keyToVerify.Key}
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", regularKey.Key)},
	}

	lb := integration.NewLoadbalancer(h)
	res, err := integration.CallRandomNode[handler.Request, handler.Response](
		lb, "POST", "/v2/keys.verifyKey", headers, req)

	require.NoError(t, err)
	// Should return 401 (same as non-existent key) to avoid leaking that the key exists
	require.Equal(t, 401, res.Status, "regular key should not be usable as root key, even with same permissions")
}
