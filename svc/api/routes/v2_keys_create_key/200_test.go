package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_create_key"
	"golang.org/x/sync/errgroup"
)

func TestCreateKeySuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	// Create API using testutil helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test basic key creation
	req := handler.Request{
		ApiId: api.ID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)

	require.NotEmpty(t, res.Body.Data.KeyId)
	require.NotEmpty(t, res.Body.Data.Key)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	// Verify key was created in database
	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)

	require.Equal(t, res.Body.Data.KeyId, key.ID)
	require.NotEmpty(t, key.Hash)
	require.NotEmpty(t, key.Start)
	require.True(t, key.Enabled)
}

func TestCreateKeyWithOptionalFields(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	// Create API using testutil helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test key creation with optional fields
	name := "Test Key"
	prefix := "test"
	externalID := "user_123"
	byteLength := 24
	expires := int64(1704067200000) // Jan 1, 2024
	enabled := true

	req := handler.Request{
		ApiId:      api.ID,
		Name:       &name,
		Prefix:     &prefix,
		ExternalId: &externalID,
		ByteLength: &byteLength,
		Expires:    &expires,
		Enabled:    &enabled,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)

	require.NotEmpty(t, res.Body.Data.KeyId)
	require.NotEmpty(t, res.Body.Data.Key)
	require.Contains(t, res.Body.Data.Key, prefix+"_")

	// Verify key fields in database
	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)

	require.True(t, key.Name.Valid)
	require.Equal(t, name, key.Name.String)
	require.True(t, key.Enabled)
}

func TestCreateKeyWithEncryption(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	// Create API with encrypted keys using testutil helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		EncryptedKeys: true,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key", "api.*.encrypt_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test key creation with optional fields
	name := "Test Key"

	req := handler.Request{
		ApiId:       api.ID,
		Name:        &name,
		ExternalId:  ptr.P("user_123"),
		Enabled:     ptr.P(true),
		Recoverable: ptr.P(true),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)

	require.NotEmpty(t, res.Body.Data.KeyId)
	require.NotEmpty(t, res.Body.Data.Key)

	// Verify key fields in database
	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)

	require.True(t, key.Name.Valid)
	require.Equal(t, name, key.Name.String)
	require.True(t, key.Enabled)

	// Verify key fields in database
	keyEncryption, err := db.Query.FindKeyEncryptionByKeyID(ctx, h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, keyEncryption.KeyID, res.Body.Data.KeyId)
	require.Equal(t, keyEncryption.WorkspaceID, h.Resources().UserWorkspace.ID)
}

// TestCreateKeyConcurrentWithSameExternalId tests that concurrent key creation
// with the same externalId doesn't deadlock. This was previously possible due to
// gap locks when inserting identities. The fix uses INSERT ... ON DUPLICATE KEY
// UPDATE (upsert) to avoid gap lock deadlocks.
func TestCreateKeyConcurrentWithSameExternalId(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	numConcurrent := 20
	externalID := "user_concurrent_test"

	var mu sync.Mutex
	keyIDs := make([]string, 0, numConcurrent)

	g := errgroup.Group{}
	for range numConcurrent {
		g.Go(func() error {
			req := handler.Request{
				ApiId:      api.ID,
				ExternalId: &externalID,
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if res.Status != 200 {
				return fmt.Errorf("unexpected status code: %d", res.Status)
			}
			mu.Lock()
			keyIDs = append(keyIDs, res.Body.Data.KeyId)
			mu.Unlock()
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err, "All concurrent creates should succeed without deadlock")

	require.Len(t, keyIDs, numConcurrent)

	// Verify all keys reference the same identity
	var sharedIdentityID string
	for i, keyID := range keyIDs {
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.True(t, key.IdentityID.Valid)

		if i == 0 {
			sharedIdentityID = key.IdentityID.String
		} else {
			require.Equal(t, sharedIdentityID, key.IdentityID.String,
				"All concurrent keys should reference the same identity")
		}
	}

	// Verify only one identity was created
	identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		ExternalID:  externalID,
		Deleted:     false,
	})
	require.NoError(t, err)
	require.Equal(t, sharedIdentityID, identity.ID)
}

func TestCreateKeyWithCreditsRemainingNull(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	// Create API using testutil helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("credits.remaining null without refill should succeed", func(t *testing.T) {
		t.Parallel()
		req := handler.Request{
			ApiId: api.ID,
			Credits: &openapi.KeyCreditsData{
				Remaining: nullable.NewNullNullable[int64](),
			},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.KeyId)
		require.NotEmpty(t, res.Body.Data.Key)
	})
}
