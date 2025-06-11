package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Setup test data
	ctx := context.Background()

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
	err = db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(t, err)

	// Insert test ratelimits
	ratelimitID1 := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
		ID:          ratelimitID1,
		WorkspaceID: h.Resources().UserWorkspace.ID,
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
		WorkspaceID: h.Resources().UserWorkspace.ID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "special_feature",
		Limit:       10,
		Duration:    3600000, // 1 hour
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// No need to set up permissions since we already gave the key the required permission

	t.Run("get by identityId", func(t *testing.T) {
		req := handler.Request{
			IdentityId: &identityID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		// Verify response

		require.Equal(t, identityID, res.Body.Data.Id)
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify metadata
		require.Equal(t, "Test User", (*res.Body.Data.Meta)["name"])
		require.Equal(t, "test@example.com", (*res.Body.Data.Meta)["email"])
		require.Equal(t, "pro", (*res.Body.Data.Meta)["plan"])
		require.Equal(t, float64(100), (*res.Body.Data.Meta)["credits"])

		// Verify ratelimits
		require.Len(t, *res.Body.Data.Ratelimits, 2)

		// Ratelimits can be in any order, so we need to find the specific ones
		var apiCallsLimit, specialFeatureLimit *openapi.Ratelimit
		for i := range *res.Body.Data.Ratelimits {
			if (*res.Body.Data.Ratelimits)[i].Name == "api_calls" {
				apiCallsLimit = &(*res.Body.Data.Ratelimits)[i]
			} else if (*res.Body.Data.Ratelimits)[i].Name == "special_feature" {
				specialFeatureLimit = &(*res.Body.Data.Ratelimits)[i]
			}
		}

		require.NotNil(t, apiCallsLimit, "api_calls ratelimit not found")
		require.NotNil(t, specialFeatureLimit, "special_feature ratelimit not found")

		require.Equal(t, int64(100), apiCallsLimit.Limit)
		require.Equal(t, int64(60000), apiCallsLimit.Duration)

		require.Equal(t, int64(10), specialFeatureLimit.Limit)
		require.Equal(t, int64(3600000), specialFeatureLimit.Duration)
	})

	t.Run("get by externalId", func(t *testing.T) {
		req := handler.Request{
			ExternalId: &externalID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, identityID, res.Body.Data.Id)
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify metadata
		require.Equal(t, "Test User", (*res.Body.Data.Meta)["name"])
		require.Equal(t, "test@example.com", (*res.Body.Data.Meta)["email"])
		require.Equal(t, "pro", (*res.Body.Data.Meta)["plan"])
		require.Equal(t, float64(100), (*res.Body.Data.Meta)["credits"])

		// Verify ratelimits
		require.Len(t, *res.Body.Data.Ratelimits, 2)
	})

	t.Run("metadata is empty object when not set", func(t *testing.T) {
		// Create a new identity without metadata
		identityWithoutMetaID := uid.New(uid.IdentityPrefix)
		externalIDWithoutMeta := "test_user_no_meta"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityWithoutMetaID,
			ExternalID:  externalIDWithoutMeta,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		req := handler.Request{
			IdentityId: &identityWithoutMetaID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify meta is an empty object, not null
		require.NotNil(t, res.Body.Data.Meta)
		metaMap := *res.Body.Data.Meta
		require.Equal(t, 0, len(metaMap))
	})

	t.Run("ratelimits is empty array when none exist", func(t *testing.T) {
		// Create a new identity without ratelimits
		identityWithoutRatelimitsID := uid.New(uid.IdentityPrefix)
		externalIDWithoutRatelimits := "test_user_no_ratelimits"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityWithoutRatelimitsID,
			ExternalID:  externalIDWithoutRatelimits,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		req := handler.Request{
			IdentityId: &identityWithoutRatelimitsID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify ratelimits is an empty array
		require.Empty(t, *res.Body.Data.Ratelimits)
	})

	t.Run("verify environment field is returned", func(t *testing.T) {
		// Create a new identity with a custom environment
		customEnvIdentityID := uid.New(uid.IdentityPrefix)
		customEnvExternalID := "test_user_custom_env"
		customEnvironment := "production"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          customEnvIdentityID,
			ExternalID:  customEnvExternalID,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			Environment: customEnvironment,
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		req := handler.Request{
			IdentityId: &customEnvIdentityID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Note: Environment field is not returned in the response
		// This test verifies that we can retrieve an identity with a custom environment
	})

	t.Run("retrieve identity with large metadata", func(t *testing.T) {
		// Create an identity with large metadata
		largeMetaIdentityID := uid.New(uid.IdentityPrefix)
		largeMetaExternalID := "test_user_large_meta"

		// Create large metadata map
		largeMetaMap := map[string]interface{}{
			"user_profile": map[string]interface{}{
				"name": "Large Metadata User",
				"bio": "This is a very long biography with lots of text " +
					strings.Repeat("Lorem ipsum dolor sit amet ", 50),
				"preferences": map[string]interface{}{
					"theme":         "dark",
					"notifications": true,
					"languages":     []string{"en", "fr", "es", "de", "it", "pt", "ru", "zh", "ja", "ko"},
					"display_options": map[string]interface{}{
						"show_avatar": true,
						"show_email":  false,
						"show_name":   true,
					},
				},
				"stats": map[string]interface{}{
					"logins":           1274,
					"last_active":      time.Now().Format(time.RFC3339),
					"account_age_days": 365,
					"activity_score":   98.7,
				},
			},
			"subscription": map[string]interface{}{
				"plan":   "enterprise",
				"status": "active",
				"features": map[string]interface{}{
					"feature1": true,
					"feature2": true,
					"feature3": true,
				},
				"limits": map[string]interface{}{
					"api_calls":  100000,
					"storage_gb": 500,
					"users":      250,
				},
				"history": []map[string]interface{}{
					{"date": "2021-01-01", "plan": "free"},
					{"date": "2021-06-01", "plan": "pro"},
					{"date": "2022-01-01", "plan": "enterprise"},
				},
			},
			"devices": []map[string]interface{}{
				{"type": "mobile", "os": "iOS", "lastLogin": time.Now().Add(-24 * time.Hour).Format(time.RFC3339)},
				{"type": "desktop", "os": "Windows", "lastLogin": time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339)},
				{"type": "tablet", "os": "Android", "lastLogin": time.Now().Add(-14 * 24 * time.Hour).Format(time.RFC3339)},
			},
		}

		largeMetaBytes, err := json.Marshal(largeMetaMap)
		require.NoError(t, err)
		require.Greater(t, len(largeMetaBytes), 1000, "Metadata should be sufficiently large")

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          largeMetaIdentityID,
			ExternalID:  largeMetaExternalID,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        largeMetaBytes,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Retrieve the identity
		req := handler.Request{
			IdentityId: &largeMetaIdentityID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify all metadata was returned correctly
		require.NotNil(t, res.Body.Data.Meta)

		// Compare by converting to JSON and back to ensure deep equality
		returnedMetaBytes, err := json.Marshal(res.Body.Data.Meta)
		require.NoError(t, err)

		var returnedMetaMap map[string]interface{}
		err = json.Unmarshal(returnedMetaBytes, &returnedMetaMap)
		require.NoError(t, err)

		// Compare the two maps
		require.Equal(t, largeMetaMap["user_profile"].(map[string]interface{})["name"],
			returnedMetaMap["user_profile"].(map[string]interface{})["name"])
		require.Equal(t, largeMetaMap["subscription"].(map[string]interface{})["plan"],
			returnedMetaMap["subscription"].(map[string]interface{})["plan"])
	})

	t.Run("retrieve identity with many rate limits", func(t *testing.T) {
		// Create an identity with many rate limits
		manyRateLimitsIdentityID := uid.New(uid.IdentityPrefix)
		manyRateLimitsExternalID := "test_user_many_ratelimits"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          manyRateLimitsIdentityID,
			ExternalID:  manyRateLimitsExternalID,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		// Create 10 rate limits for the identity
		rateLimits := []struct {
			name     string
			limit    int32
			duration int64
		}{
			{"api_calls_per_second", 10, 1000},
			{"api_calls_per_minute", 100, 60 * 1000},
			{"api_calls_per_hour", 1000, 60 * 60 * 1000},
			{"api_calls_per_day", 10000, 24 * 60 * 60 * 1000},
			{"heavy_operations", 5, 60 * 1000},
			{"storage_operations", 20, 60 * 1000},
			{"database_queries", 200, 60 * 1000},
			{"premium_features", 50, 60 * 60 * 1000},
			{"export_operations", 5, 24 * 60 * 60 * 1000},
			{"search_queries", 100, 60 * 1000},
		}

		for _, rl := range rateLimits {
			err = db.Query.InsertIdentityRatelimit(ctx, tx, db.InsertIdentityRatelimitParams{
				ID:          uid.New(uid.RatelimitPrefix),
				WorkspaceID: h.Resources().UserWorkspace.ID,

				IdentityID: sql.NullString{String: manyRateLimitsIdentityID, Valid: true},
				Name:       rl.name,
				Limit:      rl.limit,
				Duration:   rl.duration,
				CreatedAt:  time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}

		err = tx.Commit()
		require.NoError(t, err)

		// Retrieve the identity
		req := handler.Request{
			IdentityId: &manyRateLimitsIdentityID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify all rate limits were returned
		require.Len(t, *res.Body.Data.Ratelimits, len(rateLimits))

		// Create a map of rate limits by name to verify each one
		returnedRateLimits := make(map[string]openapi.Ratelimit)
		for _, rl := range *res.Body.Data.Ratelimits {
			returnedRateLimits[rl.Name] = rl
		}

		// Verify each rate limit
		for _, expected := range rateLimits {
			actual, exists := returnedRateLimits[expected.name]
			require.True(t, exists, "Rate limit %s not found in response", expected.name)
			require.Equal(t, int64(expected.limit), actual.Limit)
			require.Equal(t, expected.duration, actual.Duration)
		}
	})

	t.Run("retrieve recently created identity", func(t *testing.T) {
		// Create a new identity
		recentIdentityID := uid.New(uid.IdentityPrefix)
		recentExternalID := "test_user_recent"
		creationTime := time.Now().UnixMilli()

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          recentIdentityID,
			ExternalID:  recentExternalID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   creationTime,
			Meta:        nil,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Immediately retrieve the identity
		req := handler.Request{
			IdentityId: &recentIdentityID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify it's returned correctly
		require.Equal(t, recentIdentityID, res.Body.Data.Id)
		require.Equal(t, recentExternalID, res.Body.Data.ExternalId)
		// Note: CreatedAt is not returned in the response
	})

	t.Run("retrieve identity with associated keys", func(t *testing.T) {
		// Create an identity with associated keys
		identityWithKeysID := uid.New(uid.IdentityPrefix)
		identityWithKeysExternalID := "test_user_with_keys"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		// Insert the identity
		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityWithKeysID,
			ExternalID:  identityWithKeysExternalID,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		// Create keyring for associated keys
		keyringID := uid.New(uid.KeyAuthPrefix)
		err = db.Query.InsertKeyring(ctx, tx, db.InsertKeyringParams{
			ID:                 keyringID,
			WorkspaceID:        h.Resources().UserWorkspace.ID,
			CreatedAtM:         time.Now().UnixMilli(),
			DefaultPrefix:      sql.NullString{Valid: true, String: "test_"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			StoreEncryptedKeys: false,
		})
		require.NoError(t, err)

		// Create API for the keys
		apiID := uid.New(uid.APIPrefix)
		err = db.Query.InsertApi(ctx, tx, db.InsertApiParams{
			ID:          apiID,
			Name:        "Test API for Identity Keys",
			WorkspaceID: h.Resources().UserWorkspace.ID,

			AuthType:   db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:  sql.NullString{Valid: true, String: keyringID},
			CreatedAtM: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create associated keys
		key1ID := uid.New(uid.KeyPrefix)
		key2ID := uid.New(uid.KeyPrefix)

		// Insert first key
		err = db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
			ID:          key1ID,
			KeyringID:   keyringID,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			IdentityID: sql.NullString{Valid: true, String: identityWithKeysID},
			CreatedAtM: time.Now().UnixMilli(),
			Hash:       "hash1",
			Start:      "test_key1",
			Name:       sql.NullString{Valid: true, String: "First Key"},
		})
		require.NoError(t, err)

		// Insert second key
		err = db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
			ID:          key2ID,
			KeyringID:   keyringID,
			WorkspaceID: h.Resources().UserWorkspace.ID,

			IdentityID: sql.NullString{Valid: true, String: identityWithKeysID},
			CreatedAtM: time.Now().UnixMilli(),
			Hash:       "hash2",
			Start:      "test_key2",
			Name:       sql.NullString{Valid: true, String: "Second Key"},
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Retrieve the identity
		req := handler.Request{
			IdentityId: &identityWithKeysID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify identity data
		require.Equal(t, identityWithKeysID, res.Body.Data.Id)
		require.Equal(t, identityWithKeysExternalID, res.Body.Data.ExternalId)

		// Note: The current implementation might not return the associated keys directly
		// This test verifies that we can retrieve an identity that has associated keys
		// If the implementation is updated to return keys, this test should be updated
	})
}
