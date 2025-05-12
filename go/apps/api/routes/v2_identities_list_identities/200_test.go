package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_list_identities/handler"
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
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	// Setup test data
	ctx := context.Background()
	tx, err := h.Database().RW().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	workspaceID := h.DefaultWorkspaceID()

	// Create metadata
	metaMap := map[string]interface{}{
		"plan": "pro",
	}
	metaBytes, err := json.Marshal(metaMap)
	require.NoError(t, err)

	// Insert multiple test identities
	const totalIdentities = 15
	var identityIDs []string
	var externalIDs []string

	for i := 0; i < totalIdentities; i++ {
		identityID := uid.New(uid.IdentityPrefix)
		externalID := "test_user_" + uid.NewUUID().String()

		identityIDs = append(identityIDs, identityID)
		externalIDs = append(externalIDs, externalID)

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        metaBytes,
		})
		require.NoError(t, err)

		// Add a ratelimit to every other identity
		if i%2 == 0 {
			ratelimitID := uid.New(uid.RatelimitPrefix)
			err = db.Query.InsertIdentityRatelimit(ctx, tx, db.InsertIdentityRatelimitParams{
				ID:          ratelimitID,
				WorkspaceID: workspaceID,
				IdentityID:  sql.NullString{String: identityID, Valid: true},
				Name:        "api_calls",
				Limit:       100,
				Duration:    60000, // 1 minute
				CreatedAt:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}
	}

	// Create identity in different environment
	otherEnvIdentityID := uid.New(uid.IdentityPrefix)
	otherEnvExternalID := "test_user_other_env"
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          otherEnvIdentityID,
		ExternalID:  otherEnvExternalID,
		WorkspaceID: workspaceID,
		Environment: "production",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Set up permissions
	h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.read_identity", true)

	t.Run("basic listing with default settings", func(t *testing.T) {
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, got: %d, response: %s", res.Status, res.RawBody)

		// Should have totalIdentities items (only default environment)
		require.Equal(t, totalIdentities, len(res.Body.Data.Identities))
		require.Equal(t, totalIdentities, res.Body.Data.Total)

		// Should have a cursor for pagination
		require.NotNil(t, res.Body.Data.Cursor)

		// Verify first identity
		found := false
		for _, identity := range res.Body.Data.Identities {
			for i, id := range identityIDs {
				if identity.ID == id {
					assert.Equal(t, externalIDs[i], identity.ExternalID)
					found = true

					// Check if this identity should have ratelimits
					if i%2 == 0 {
						require.Equal(t, 1, len(identity.Ratelimits), "identity %s should have 1 ratelimit", id)
						assert.Equal(t, "api_calls", identity.Ratelimits[0].Name)
						assert.Equal(t, 100, identity.Ratelimits[0].Limit)
						assert.Equal(t, int64(60000), identity.Ratelimits[0].Duration)
					} else {
						assert.Empty(t, identity.Ratelimits, "identity %s should have no ratelimits", id)
					}
					break
				}
			}
		}
		require.True(t, found, "should have found at least one of our test identities")
	})

	t.Run("with custom limit", func(t *testing.T) {
		limit := 5
		req := handler.Request{
			limit: &limit,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Should have exactly 5 items
		require.Equal(t, 5, len(res.Body.Data.Identities))
		// Total should still be the total number of identities
		require.Equal(t, totalIdentities, res.Body.Data.Total)
		// Should have a cursor
		require.NotNil(t, res.Body.Data.Cursor)
	})

	t.Run("with pagination", func(t *testing.T) {
		// First page
		limit := 7
		req := handler.Request{
			limit: &limit,
		}
		firstRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, firstRes.Status)
		require.Equal(t, 7, len(firstRes.Body.Data.Identities))
		require.NotNil(t, firstRes.Body.Data.Cursor)

		// Second page
		req := handler.Request{
			limit:  &limit,
			cursor: firstRes.Body.Data.Cursor,
		}
		secondRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, secondRes.Status)

		// Should have the remaining identities
		require.Equal(t, totalIdentities-7, len(secondRes.Body.Data.Identities))

		// Ensure no overlap between pages
		firstPageIDs := make(map[string]bool)
		for _, identity := range firstRes.Body.Data.Identities {
			firstPageIDs[identity.ID] = true
		}

		for _, identity := range secondRes.Body.Data.Identities {
			_, found := firstPageIDs[identity.ID]
			assert.False(t, found, "identity %s should not appear in both pages", identity.ID)
		}
	})

	t.Run("with environment filter", func(t *testing.T) {
		env := "production"
		req := handler.Request{
			environment: &env,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Should only have 1 identity in production environment
		require.Equal(t, 1, len(res.Body.Data.Identities))
		require.Equal(t, 1, res.Body.Data.Total)

		// Verify it's the correct identity
		assert.Equal(t, otherEnvIdentityID, res.Body.Data.Identities[0].ID)
		assert.Equal(t, otherEnvExternalID, res.Body.Data.Identities[0].ExternalID)
	})

	t.Run("with min limit", func(t *testing.T) {
		// Test with limit = 0 (should be adjusted to 1)
		limit := 0
		req := handler.Request{
			limit: &limit,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Should have minimum 1 identity
		require.GreaterOrEqual(t, len(res.Body.Data.Identities), 1)
	})

	t.Run("with max limit", func(t *testing.T) {
		// Test with limit = 200 (should be capped to 100)
		limit := 200
		req := handler.Request{
			limit: &limit,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Should have no more than 100 identities
		require.LessOrEqual(t, len(res.Body.Data.Identities), 100)
	})
}
