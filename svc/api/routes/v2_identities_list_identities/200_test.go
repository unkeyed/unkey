package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
	}

	// Register the route with the harness
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Setup test data
	ctx := context.Background()
	tx, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	defer func() {
		err := tx.Rollback()
		require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
	}()

	workspaceID := h.Resources().UserWorkspace.ID

	// Create metadata
	metaMap := map[string]interface{}{
		"plan": "pro",
	}
	metaBytes, err := json.Marshal(metaMap)
	require.NoError(t, err)

	// Insert multiple test identities
	const totalIdentities = 15
	var externalIDs []string
	for i := range totalIdentities {
		identityID := uid.New(uid.IdentityPrefix)
		externalID := "test_user_" + uid.New("") // Generate a unique ID

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

	err = tx.Commit()
	require.NoError(t, err)

	t.Run("basic listing with default settings", func(t *testing.T) {
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, got: %d, response: %s", res.Status, res.RawBody)

		// Number may be different from totalIdentities due to other test identities
		require.GreaterOrEqual(t, len(res.Body.Data), 1)

		// Verify first identity
		found := false
		for _, identity := range res.Body.Data {
			for i, id := range externalIDs {
				if identity.ExternalId == id {
					require.Equal(t, externalIDs[i], identity.ExternalId)
					found = true

					// Check if this identity should have ratelimits
					if i%2 == 0 {
						require.GreaterOrEqual(t, len(identity.Ratelimits), 1, "identity %s should have at least 1 ratelimit", id)
						hasApiCallsLimit := false
						for _, rl := range identity.Ratelimits {
							if rl.Name == "api_calls" {
								hasApiCallsLimit = true
								require.Equal(t, int64(100), rl.Limit)
								require.Equal(t, int64(60000), rl.Duration)
								break
							}
						}
						require.True(t, hasApiCallsLimit, "identity should have api_calls ratelimit")
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
			Limit: &limit,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Should have exactly 5 items
		require.Equal(t, 5, len(res.Body.Data))
		// Should have a cursor
		require.NotNil(t, res.Body.Pagination.Cursor)
	})

	t.Run("with pagination", func(t *testing.T) {
		// First page
		limit := 7
		req := handler.Request{
			Limit: &limit,
		}
		firstRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, firstRes.Status)
		require.Equal(t, 7, len(firstRes.Body.Data))
		require.NotNil(t, firstRes.Body.Pagination.Cursor)
		require.True(t, firstRes.Body.Pagination.HasMore)

		// Second page
		req = handler.Request{
			Limit:  &limit,
			Cursor: firstRes.Body.Pagination.Cursor,
		}
		secondRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, secondRes.Status)

		// Should have some of the remaining identities
		require.Greater(t, len(secondRes.Body.Data), 0)

		// Ensure no overlap between pages
		firstPageExternalIDs := make(map[string]bool)
		for _, identity := range firstRes.Body.Data {
			firstPageExternalIDs[identity.ExternalId] = true
		}

		// Check a sample identity from second page to ensure no overlap
		if len(secondRes.Body.Data) > 0 {
			sampleExternalID := secondRes.Body.Data[0].ExternalId
			_, found := firstPageExternalIDs[sampleExternalID]
			require.False(t, found, "identity %s should not appear in both pages", sampleExternalID)
		}
	})

	// Test for deleted identities
	t.Run("deleted identities are excluded", func(t *testing.T) {
		// Create a new identity
		deletedIdentityID := uid.New(uid.IdentityPrefix)
		deletedExternalID := "test_deleted_user"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer func() {
			err := tx.Rollback()
			require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
		}()

		// Insert the identity
		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          deletedIdentityID,
			ExternalID:  deletedExternalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        metaBytes,
		})
		require.NoError(t, err)

		// Soft delete the identity
		err = db.Query.SoftDeleteIdentity(ctx, tx, db.SoftDeleteIdentityParams{
			IdentityID:  deletedIdentityID,
			WorkspaceID: workspaceID,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Try to retrieve the identity in a listing
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// The deleted identity should not be in the results
		for _, identity := range res.Body.Data {
			require.NotEqual(t, deletedExternalID, identity.ExternalId, "Deleted identity should not be returned")
		}
	})

	// Test with Unicode characters
	t.Run("identities with Unicode characters", func(t *testing.T) {
		// Create an identity with Unicode characters
		unicodeIdentityID := uid.New(uid.IdentityPrefix)
		unicodeExternalID := "unicode-user-测试-🔑"

		// Create metadata with Unicode characters
		unicodeMetaMap := map[string]interface{}{
			"name":        "名字",
			"description": "这是一个测试用户 with 英文 and emoji 👍👨‍👩‍👧‍👦🇯🇵",
			"tags":        []string{"测试", "官方", "认证✓"},
		}
		unicodeMetaBytes, err := json.Marshal(unicodeMetaMap)
		require.NoError(t, err)

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer func() {
			err := tx.Rollback()
			require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
		}()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          unicodeIdentityID,
			ExternalID:  unicodeExternalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        unicodeMetaBytes,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Query for all identities
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Should find the identity with Unicode
		var foundUnicode bool
		for _, identity := range res.Body.Data {
			if identity.ExternalId == unicodeExternalID {
				foundUnicode = true

				// Verify the Unicode external ID was preserved
				require.Equal(t, unicodeExternalID, identity.ExternalId)

				// Verify Unicode in metadata was preserved
				require.NotNil(t, identity.Meta)
				metaMap := identity.Meta
				require.Equal(t, "名字", metaMap["name"])
				require.Equal(t, "这是一个测试用户 with 英文 and emoji 👍👨‍👩‍👧‍👦🇯🇵", metaMap["description"])

				break
			}
		}

		require.True(t, foundUnicode, "The Unicode identity should be found in the results")
	})

	// Test with exactly one identity
	t.Run("list with exactly one identity", func(t *testing.T) {
		// Create a new workspace with exactly one identity
		singleWorkspaceID := uid.New(uid.WorkspacePrefix)
		singleIdentityID := uid.New(uid.IdentityPrefix)
		singleExternalID := "test_single_user"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer func() {
			err := tx.Rollback()
			require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
		}()

		// Create the workspace
		err = db.Query.InsertWorkspace(ctx, tx, db.InsertWorkspaceParams{
			ID:        singleWorkspaceID,
			Name:      "Single Identity Workspace",
			Slug:      uid.New("slug"),
			OrgID:     uid.New(uid.OrgPrefix),
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a single identity in this workspace
		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          singleIdentityID,
			ExternalID:  singleExternalID,
			WorkspaceID: singleWorkspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        metaBytes,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Create a root key for this workspace
		singleWorkspaceKey := h.CreateRootKey(singleWorkspaceID, "identity.*.read_identity")
		singleHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", singleWorkspaceKey)},
		}

		// Query for identities in this specific workspace
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, singleHeaders, req)
		require.Equal(t, 200, res.Status)

		// Should return exactly one identity
		require.Equal(t, 1, len(res.Body.Data))
		require.Equal(t, singleExternalID, res.Body.Data[0].ExternalId)

		// Pagination should indicate no more results
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})

	// Test for verifying the complete response structure
	t.Run("verify complete response structure", func(t *testing.T) {
		// Make a basic request
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify top-level structure
		require.NotNil(t, res.Body.Data, "Data field should be present")
		require.NotNil(t, res.Body.Meta, "Meta field should be present")
		require.NotEmpty(t, res.Body.Meta.RequestId, "RequestId should be present")

		// Pagination structure should always be present, even if empty
		require.NotNil(t, res.Body.Pagination, "Pagination field should be present")

		// If we have results, verify the structure of an identity
		if len(res.Body.Data) > 0 {
			// Check all identities
			for _, identity := range res.Body.Data {
				// ID fields should never be empty
				require.NotEmpty(t, identity.ExternalId, "External ID should not be empty")

				dbIdentity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RW(), db.FindIdentityByExternalIDParams{WorkspaceID: workspaceID, ExternalID: identity.ExternalId, Deleted: false})
				require.NoError(t, err)
				require.NotNil(t, dbIdentity, "Identity should be found in the database")

				identityRatelimits, err := db.Query.ListIdentityRatelimits(ctx, h.DB.RW(), sql.NullString{Valid: true, String: dbIdentity.ID})
				require.NoError(t, err)

				if len(identityRatelimits) == 0 {
					require.Nil(t, identity.Ratelimits, "Ratelimits should be nil")
				} else {
					require.NotNil(t, identity.Ratelimits, "Ratelimits should be set")
					require.Len(t, identity.Ratelimits, len(identityRatelimits), "Ratelimits should match the database")
				}

				if len(dbIdentity.Meta) > 0 {
					raw, err := json.Marshal(identity.Meta)
					require.NoError(t, err)
					require.NotNil(t, identity.Meta, "Meta should be set")
					require.JSONEq(t, string(raw), string(dbIdentity.Meta), "Meta should match the database")
				}
			}
		}
	})
}
