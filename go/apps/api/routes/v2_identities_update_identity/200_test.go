package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.update_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
	}

	// Setup test data
	ctx := context.Background()

	workspaceID := h.Resources().UserWorkspace.ID
	identityID := uid.New(uid.IdentityPrefix)
	otherIdentityID := uid.New(uid.IdentityPrefix)
	externalID := "test_user_123"
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
	err = db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(t, err)

	err = db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          otherIdentityID,
		ExternalID:  otherExternalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)

	// Insert test ratelimits for the first identity
	ratelimitID1 := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
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
	err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
		ID:          ratelimitID2,
		WorkspaceID: workspaceID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "special_feature",
		Limit:       10,
		Duration:    3600000, // 1 hour
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	t.Run("update metadata", func(t *testing.T) {
		newMeta := map[string]interface{}{
			"joined": "2023-01-01",
			"active": true,
		}

		req := handler.Request{
			Identity: otherExternalID,
			Meta:     &newMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, otherExternalID, res.Body.Data.ExternalId)

		// Verify metadata
		require.NotNil(t, res.Body.Data.Meta)
		meta := *res.Body.Data.Meta
		assert.Equal(t, "2023-01-01", meta["joined"])
		assert.Equal(t, true, meta["active"])

		// Verify no ratelimits
		require.Nil(t, res.Body.Data.Ratelimits)
	})

	t.Run("update ratelimits - add new, update existing, delete one", func(t *testing.T) {
		// This will:
		// 1. Update 'api_calls' limit from 100 to 200
		// 2. Add a new 'new_feature' limit
		// 3. Delete 'special_feature' limit (by not including it)
		ratelimits := []openapi.RatelimitRequest{
			{
				Name:      "api_calls",
				Limit:     200,
				Duration:  60000,
				AutoApply: true,
			},
			{
				Name:      "new_feature",
				Limit:     5,
				Duration:  86400000, // 1 day
				AutoApply: false,
			},
		}

		req := handler.Request{
			Identity:   externalID,
			Ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify exactly 2 ratelimits (should have removed 'special_feature')
		require.NotNil(t, res.Body.Data.Ratelimits)
		require.Len(t, *res.Body.Data.Ratelimits, 2)

		// Check ratelimit values
		var apiCallsLimit, newFeatureLimit *openapi.RatelimitResponse
		for i := range *res.Body.Data.Ratelimits {
			switch (*res.Body.Data.Ratelimits)[i].Name {
			case "api_calls":
				apiCallsLimit = &(*res.Body.Data.Ratelimits)[i]
			case "new_feature":
				newFeatureLimit = &(*res.Body.Data.Ratelimits)[i]
			}
		}

		require.NotNil(t, apiCallsLimit, "api_calls ratelimit not found")
		require.NotNil(t, newFeatureLimit, "new_feature ratelimit not found")

		// Verify updated limit
		assert.Equal(t, int64(200), apiCallsLimit.Limit)
		assert.Equal(t, int64(60000), apiCallsLimit.Duration)

		// Verify new limit
		assert.Equal(t, int64(5), newFeatureLimit.Limit)
		assert.Equal(t, int64(86400000), newFeatureLimit.Duration)

		// Verify 'special_feature' was removed
		for _, rl := range *res.Body.Data.Ratelimits {
			assert.NotEqual(t, "special_feature", rl.Name, "special_feature should have been removed")
		}
	})

	t.Run("remove all ratelimits", func(t *testing.T) {
		// Empty array should remove all ratelimits
		emptyRatelimits := []openapi.RatelimitRequest{}

		req := handler.Request{
			Identity:   externalID,
			Ratelimits: &emptyRatelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify no ratelimits
		require.Nil(t, res.Body.Data.Ratelimits)
	})

	t.Run("clear metadata", func(t *testing.T) {
		// Empty map should clear metadata
		emptyMeta := map[string]interface{}{}

		req := handler.Request{
			Identity: externalID,
			Meta:     &emptyMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify empty metadata
		require.NotNil(t, res.Body.Data.Meta)
		assert.Empty(t, *res.Body.Data.Meta)
	})

	t.Run("update both metadata and ratelimits", func(t *testing.T) {
		newMeta := map[string]interface{}{
			"plan":    "enterprise",
			"credits": 1000,
		}

		ratelimits := []openapi.RatelimitRequest{
			{
				Name:      "enterprise_feature",
				Limit:     50,
				Duration:  3600000,
				AutoApply: true,
			},
		}

		req := handler.Request{
			Identity:   externalID,
			Meta:       &newMeta,
			Ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify metadata
		require.NotNil(t, res.Body.Data.Meta)
		meta := *res.Body.Data.Meta
		assert.Equal(t, "enterprise", meta["plan"])
		assert.Equal(t, float64(1000), meta["credits"])

		// Verify ratelimits
		require.NotNil(t, res.Body.Data.Ratelimits)
		require.Len(t, *res.Body.Data.Ratelimits, 1)
		rlimits := *res.Body.Data.Ratelimits
		assert.Equal(t, "enterprise_feature", rlimits[0].Name)
		assert.Equal(t, int64(50), rlimits[0].Limit)
		assert.Equal(t, int64(3600000), rlimits[0].Duration)
	})
}
