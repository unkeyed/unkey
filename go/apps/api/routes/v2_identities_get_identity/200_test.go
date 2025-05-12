package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity/handler"
	"github.com/unkeyed/unkey/go/internal/testutil"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	// Setup test data
	ctx := context.Background()
	tx, err := h.Database().RW().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	workspaceID := "ws_123"
	identityID := uid.New(uid.IdentityPrefix)
	externalID := "test_user_123"

	// Create metadata
	metaMap := map[string]interface{}{
		"name":    "Test User",
		"email":   "test@example.com",
		"plan":    "pro",
		"credits": 100,
	}
	metaBytes, err := json.Marshal(metaMap)
	require.NoError(t, err)

	// Insert test identity
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(t, err)

	// Insert test ratelimits
	ratelimitID1 := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, tx, db.InsertIdentityRatelimitParams{
		ID:          ratelimitID1,
		WorkspaceID: workspaceID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "api_calls",
		Limit:       100,
		Duration:    60000, // 1 minute
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	ratelimitID2 := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, tx, db.InsertIdentityRatelimitParams{
		ID:          ratelimitID2,
		WorkspaceID: workspaceID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "special_feature",
		Limit:       10,
		Duration:    3600000, // 1 hour
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Set up permissions
	h.SetupPermissions(t, rootKeyID, h.DefaultWorkspaceID(), "identity.*.read_identity", true)

	t.Run("get by identityId", func(t *testing.T) {
		req := handler.Request{
			identityID: &identityID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, identityID, res.Body.Data.Id)
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify metadata
		assert.Equal(t, "Test User", res.Body.Data.Meta["name"])
		assert.Equal(t, "test@example.com", res.Body.Data.Meta["email"])
		assert.Equal(t, "pro", res.Body.Data.Meta["plan"])
		assert.Equal(t, float64(100), res.Body.Data.Meta["credits"])

		// Verify ratelimits
		require.Len(t, res.Body.Data.Ratelimits, 2)

		// Ratelimits can be in any order, so we need to find the specific ones
		var apiCallsLimit, specialFeatureLimit *openapi.Ratelimit
		for i := range res.Body.Data.Ratelimits {
			if res.Body.Data.Ratelimits[i].Name == "api_calls" {
				apiCallsLimit = &res.Body.Data.Ratelimits[i]
			} else if res.Body.Data.Ratelimits[i].Name == "special_feature" {
				specialFeatureLimit = &res.Body.Data.Ratelimits[i]
			}
		}

		require.NotNil(t, apiCallsLimit, "api_calls ratelimit not found")
		require.NotNil(t, specialFeatureLimit, "special_feature ratelimit not found")

		assert.Equal(t, 100, apiCallsLimit.Limit)
		assert.Equal(t, int64(60000), apiCallsLimit.Duration)

		assert.Equal(t, 10, specialFeatureLimit.Limit)
		assert.Equal(t, int64(3600000), specialFeatureLimit.Duration)
	})

	t.Run("get by externalId", func(t *testing.T) {
		req := handler.Request{
			externalID: &externalID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, identityID, res.Body.Data.Id)
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify metadata
		assert.Equal(t, "Test User", res.Body.Data.Meta["name"])
		assert.Equal(t, "test@example.com", res.Body.Data.Meta["email"])
		assert.Equal(t, "pro", res.Body.Data.Meta["plan"])
		assert.Equal(t, float64(100), res.Body.Data.Meta["credits"])

		// Verify ratelimits
		require.Len(t, res.Body.Data.Ratelimits, 2)
	})

	t.Run("metadata is empty object when not set", func(t *testing.T) {
		// Create a new identity without metadata
		identityWithoutMetaID := uid.New(uid.IdentityPrefix)
		externalIDWithoutMeta := "test_user_no_meta"

		tx, err := h.Database().RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityWithoutMetaID,
			ExternalID:  externalIDWithoutMeta,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		req := handler.Request{
			identityID: &identityWithoutMetaID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify meta is an empty object, not null
		assert.NotNil(t, res.Body.Data.Meta)
		assert.Equal(t, 0, len(res.Body.Data.Meta))
	})

	t.Run("ratelimits is empty array when none exist", func(t *testing.T) {
		// Create a new identity without ratelimits
		identityWithoutRatelimitsID := uid.New(uid.IdentityPrefix)
		externalIDWithoutRatelimits := "test_user_no_ratelimits"

		tx, err := h.Database().RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityWithoutRatelimitsID,
			ExternalID:  externalIDWithoutRatelimits,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		req := handler.Request{
			identityID: &identityWithoutRatelimitsID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify ratelimits is an empty array
		assert.Empty(t, res.Body.Data.Ratelimits)
	})
}
