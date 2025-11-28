package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_create_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestCreateKeySuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:    h.Logger,
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
		Logger:    h.Logger,
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
		Logger:    h.Logger,
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

func TestCreateKeyConcurrentWithSameExternalId(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:    h.Logger,
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

	// Use same externalId for concurrent requests
	externalID := "user_concurrent_test"

	// Create multiple keys concurrently with the same externalId
	// This simulates the race condition where:
	// 1. Request A checks if identity exists - doesn't find it
	// 2. Request B checks if identity exists - doesn't find it
	// 3. Request A tries to insert identity - succeeds
	// 4. Request B tries to insert identity - gets duplicate key error
	// 5. Request B handles the error by finding the existing identity
	numConcurrent := 5
	results := make(chan testutil.TestResponse[handler.Response], numConcurrent)
	errors := make(chan error, numConcurrent)

	for range numConcurrent {
		go func() {
			req := handler.Request{
				ApiId:      api.ID,
				ExternalId: &externalID,
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if res.Status != 200 {
				errors <- fmt.Errorf("unexpected status code: %d", res.Status)
				return
			}
			results <- res
		}()
	}

	// Collect all results
	keyIDs := make([]string, 0, numConcurrent)
	for i := 0; i < numConcurrent; i++ {
		select {
		case res := <-results:
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.NotEmpty(t, res.Body.Data.KeyId)
			keyIDs = append(keyIDs, res.Body.Data.KeyId)
		case err := <-errors:
			t.Fatal(err)
		}
	}

	// Verify all keys were created
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
	require.Equal(t, externalID, identity.ExternalID)
}

func TestCreateKeyWithCreditsRemainingNull(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
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
