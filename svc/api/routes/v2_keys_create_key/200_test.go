package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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

// TestCreateKeyAppliesKeySpaceDefaults guards the keyspace `default_prefix`
// and `default_bytes` columns. The dashboard surfaces both fields, so when
// the request omits them they must fall back to what the keyspace stores.
// Regression: previously the handler ignored both columns and always used
// the hardcoded byte length 16 with no prefix.
func TestCreateKeyAppliesKeySpaceDefaults(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// base58 length is deterministic per byte length within ±1 char (leading
	// zero handling), so the floor below safely separates 16-byte from
	// 64-byte keys: 16 → ~22 chars, 64 → ~86 chars.
	const minEncodedLengthFor64Bytes = 80

	t.Run("default prefix is applied when request omits prefix", func(t *testing.T) {
		t.Parallel()

		defaultPrefix := "dash"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   h.Resources().UserWorkspace.ID,
			DefaultPrefix: &defaultPrefix,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](
			h, route, headers, handler.Request{ApiId: api.ID},
		)
		require.Equal(t, 200, res.Status)
		require.True(t, strings.HasPrefix(res.Body.Data.Key, defaultPrefix+"_"),
			"key %q should start with keyspace default_prefix %q", res.Body.Data.Key, defaultPrefix)
	})

	t.Run("default bytes is applied when request omits byteLength", func(t *testing.T) {
		t.Parallel()

		defaultBytes := int32(64)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:  h.Resources().UserWorkspace.ID,
			DefaultBytes: &defaultBytes,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](
			h, route, headers, handler.Request{ApiId: api.ID},
		)
		require.Equal(t, 200, res.Status)
		require.GreaterOrEqual(t, len(res.Body.Data.Key), minEncodedLengthFor64Bytes,
			"key %q (len=%d) should be long enough to reflect default_bytes=64",
			res.Body.Data.Key, len(res.Body.Data.Key))
	})

	t.Run("both defaults apply together", func(t *testing.T) {
		t.Parallel()

		defaultPrefix := "both"
		defaultBytes := int32(64)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   h.Resources().UserWorkspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](
			h, route, headers, handler.Request{ApiId: api.ID},
		)
		require.Equal(t, 200, res.Status)
		require.True(t, strings.HasPrefix(res.Body.Data.Key, defaultPrefix+"_"),
			"key %q should start with default_prefix %q", res.Body.Data.Key, defaultPrefix)

		encoded := strings.TrimPrefix(res.Body.Data.Key, defaultPrefix+"_")
		require.GreaterOrEqual(t, len(encoded), minEncodedLengthFor64Bytes,
			"encoded portion %q (len=%d) should reflect default_bytes=64",
			encoded, len(encoded))
	})

	t.Run("request prefix overrides keyspace default_prefix", func(t *testing.T) {
		t.Parallel()

		defaultPrefix := "dash"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   h.Resources().UserWorkspace.ID,
			DefaultPrefix: &defaultPrefix,
		})

		reqPrefix := "explicit"
		res := testutil.CallRoute[handler.Request, handler.Response](
			h, route, headers, handler.Request{ApiId: api.ID, Prefix: &reqPrefix},
		)
		require.Equal(t, 200, res.Status)
		require.True(t, strings.HasPrefix(res.Body.Data.Key, reqPrefix+"_"),
			"request prefix should win, got %q", res.Body.Data.Key)
		require.False(t, strings.HasPrefix(res.Body.Data.Key, defaultPrefix+"_"),
			"keyspace default should not apply when request supplies prefix, got %q", res.Body.Data.Key)
	})

	t.Run("request byteLength overrides keyspace default_bytes", func(t *testing.T) {
		t.Parallel()

		defaultBytes := int32(64)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:  h.Resources().UserWorkspace.ID,
			DefaultBytes: &defaultBytes,
		})

		reqByteLength := 16
		res := testutil.CallRoute[handler.Request, handler.Response](
			h, route, headers, handler.Request{ApiId: api.ID, ByteLength: &reqByteLength},
		)
		require.Equal(t, 200, res.Status)
		// 16 bytes encodes to ~22 base58 chars, well below the 64-byte floor.
		require.Less(t, len(res.Body.Data.Key), minEncodedLengthFor64Bytes,
			"request byteLength=16 should win over default_bytes=64, got %q (len=%d)",
			res.Body.Data.Key, len(res.Body.Data.Key))
	})

	t.Run("falls back to 16 bytes when neither request nor keyspace provides byteLength", func(t *testing.T) {
		t.Parallel()

		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](
			h, route, headers, handler.Request{ApiId: api.ID},
		)
		require.Equal(t, 200, res.Status)
		// 16 bytes → ~22 base58 chars; allow a small ± window for leading zeros.
		require.GreaterOrEqual(t, len(res.Body.Data.Key), 20)
		require.LessOrEqual(t, len(res.Body.Data.Key), 24)
	})
}
