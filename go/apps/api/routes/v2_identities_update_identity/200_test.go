package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity/handler"
	"github.com/unkeyed/unkey/go/internal/testutil"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
		Auditlogs:   h.Auditlogs(),
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	// Setup test data
	ctx := context.Background()
	tx, err := h.Database().RW().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	workspaceID := h.DefaultWorkspaceID()
	identityID := uid.New(uid.IdentityPrefix)
	externalID := "test_user_123"
	otherIdentityID := uid.New(uid.IdentityPrefix)
	otherExternalID := "test_user_456"

	// Create initial metadata
	metaMap := map[string]interface{}{
		"name":    "Test User",
		"email":   "test@example.com",
		"plan":    "free",
		"credits": 50,
	}
	metaBytes, err := json.Marshal(metaMap)
	require.NoError(t, err)

	// Insert test identities
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(t, err)

	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          otherIdentityID,
		ExternalID:  otherExternalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        nil,
	})
	require.NoError(t, err)

	// Insert test ratelimits for the first identity
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
	h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.update_identity", true)

	t.Run("update metadata by identityId", func(t *testing.T) {
		newMeta := map[string]interface{}{
			"name":    "Updated User",
			"email":   "updated@example.com",
			"plan":    "pro",
			"credits": 100,
		}

		req := handler.Request{
			identityID: &identityID,
			meta:       newMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, identityID, res.Body.Data.ID)
		require.Equal(t, externalID, res.Body.Data.ExternalID)

		// Verify metadata
		assert.Equal(t, "Updated User", res.Body.Data.Meta["name"])
		assert.Equal(t, "updated@example.com", res.Body.Data.Meta["email"])
		assert.Equal(t, "pro", res.Body.Data.Meta["plan"])
		assert.Equal(t, float64(100), res.Body.Data.Meta["credits"])

		// Verify ratelimits remain unchanged
		require.Len(t, res.Body.Data.Ratelimits, 2)
	})

	t.Run("update metadata by externalId", func(t *testing.T) {
		newMeta := map[string]interface{}{
			"joined": "2023-01-01",
			"active": true,
		}

		req := handler.Request{
			externalID: &otherExternalID,
			meta:       newMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, otherIdentityID, res.Body.Data.ID)
		require.Equal(t, otherExternalID, res.Body.Data.ExternalID)

		// Verify metadata
		assert.Equal(t, "2023-01-01", res.Body.Data.Meta["joined"])
		assert.Equal(t, true, res.Body.Data.Meta["active"])

		// Verify no ratelimits
		assert.Empty(t, res.Body.Data.Ratelimits)
	})

	t.Run("update ratelimits - add new, update existing, delete one", func(t *testing.T) {
		// This will:
		// 1. Update 'api_calls' limit from 100 to 200
		// 2. Add a new 'new_feature' limit
		// 3. Delete 'special_feature' limit (by not including it)

		ratelimits := []openapi.Ratelimit{
			{
				Name:     "api_calls",
				Limit:    200,
				Duration: 60000,
			},
			{
				Name:     "new_feature",
				Limit:    5,
				Duration: 86400000, // 1 day
			},
		}

		req := handler.Request{
			identityID: &identityID,
			ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, identityID, res.Body.Data.ID)

		// Verify exactly 2 ratelimits (should have removed 'special_feature')
		require.Len(t, res.Body.Data.Ratelimits, 2)

		// Check ratelimit values
		var apiCallsLimit, newFeatureLimit *openapi.Ratelimit
		for i := range res.Body.Data.Ratelimits {
			if res.Body.Data.Ratelimits[i].Name == "api_calls" {
				apiCallsLimit = &res.Body.Data.Ratelimits[i]
			} else if res.Body.Data.Ratelimits[i].Name == "new_feature" {
				newFeatureLimit = &res.Body.Data.Ratelimits[i]
			}
		}

		require.NotNil(t, apiCallsLimit, "api_calls ratelimit not found")
		require.NotNil(t, newFeatureLimit, "new_feature ratelimit not found")

		// Verify updated limit
		assert.Equal(t, 200, apiCallsLimit.Limit)
		assert.Equal(t, int64(60000), apiCallsLimit.Duration)

		// Verify new limit
		assert.Equal(t, 5, newFeatureLimit.Limit)
		assert.Equal(t, int64(86400000), newFeatureLimit.Duration)

		// Verify 'special_feature' was removed
		for _, rl := range res.Body.Data.Ratelimits {
			assert.NotEqual(t, "special_feature", rl.Name, "special_feature should have been removed")
		}
	})

	t.Run("remove all ratelimits", func(t *testing.T) {
		// Empty array should remove all ratelimits
		emptyRatelimits := []openapi.Ratelimit{}

		req := handler.Request{
			identityID: &identityID,
			ratelimits: &emptyRatelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, identityID, res.Body.Data.ID)

		// Verify no ratelimits
		assert.Empty(t, res.Body.Data.Ratelimits)
	})

	t.Run("clear metadata", func(t *testing.T) {
		// Empty map should clear metadata
		emptyMeta := map[string]interface{}{}

		req := handler.Request{
			identityID: &identityID,
			meta:       emptyMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, identityID, res.Body.Data.ID)

		// Verify empty metadata
		assert.Empty(t, res.Body.Data.Meta)
	})

	t.Run("update both metadata and ratelimits", func(t *testing.T) {
		newMeta := map[string]interface{}{
			"plan":    "enterprise",
			"credits": 1000,
		}

		ratelimits := []openapi.Ratelimit{
			{
				Name:     "enterprise_feature",
				Limit:    50,
				Duration: 3600000,
			},
		}

		req := handler.Request{
			identityID: &identityID,
			meta:       newMeta,
			ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify metadata
		assert.Equal(t, "enterprise", res.Body.Data.Meta["plan"])
		assert.Equal(t, float64(1000), res.Body.Data.Meta["credits"])

		// Verify ratelimits
		require.Len(t, res.Body.Data.Ratelimits, 1)
		assert.Equal(t, "enterprise_feature", res.Body.Data.Ratelimits[0].Name)
		assert.Equal(t, 50, res.Body.Data.Ratelimits[0].Limit)
		assert.Equal(t, int64(3600000), res.Body.Data.Ratelimits[0].Duration)
	})
}
