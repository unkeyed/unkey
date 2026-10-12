package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_get_identity"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	externalID := uid.New(uid.TestPrefix)
	// Create metadata
	metaMap := map[string]any{
		"name":    "Test User",
		"email":   "test@example.com",
		"plan":    "pro",
		"credits": 100,
	}
	metaBytes, err := json.Marshal(metaMap)
	require.NoError(t, err)

	// Create identity with ratelimits using testutil helper
	h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		ExternalID:  externalID,
		Meta:        metaBytes,
		Ratelimits: []seed.CreateRatelimitRequest{
			{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				Name:        "api_calls",
				Limit:       100,
				Duration:    60000,
			},
			{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				Name:        "special_feature",
				Limit:       10,
				Duration:    3600000,
			},
		},
	})

	// No need to set up permissions since we already gave the key the required permission
	t.Run("get by externalId", func(t *testing.T) {
		req := handler.Request{
			Identity: externalID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify metadata
		require.Equal(t, "Test User", (res.Body.Data.Meta)["name"])
		require.Equal(t, "test@example.com", (res.Body.Data.Meta)["email"])
		require.Equal(t, "pro", (res.Body.Data.Meta)["plan"])
		require.Equal(t, float64(100), (res.Body.Data.Meta)["credits"])

		// Verify ratelimits
		require.Len(t, res.Body.Data.Ratelimits, 2)
	})

	t.Run("metadata is empty object when not set", func(t *testing.T) {
		// Create a new identity without metadata
		externalIDWithoutMeta := uid.New(uid.TestPrefix)
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			ExternalID:  externalIDWithoutMeta,
		})

		req := handler.Request{
			Identity: externalIDWithoutMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify meta is an empty object, not null
		require.Empty(t, res.Body.Data.Meta)
		metaMap := res.Body.Data.Meta
		require.Equal(t, 0, len(metaMap))
	})

	t.Run("ratelimits is empty array when none exist", func(t *testing.T) {
		// Create a new identity without ratelimits
		externalIDWithoutRatelimits := uid.New(uid.TestPrefix)
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			ExternalID:  externalIDWithoutRatelimits,
		})

		req := handler.Request{
			Identity: externalIDWithoutRatelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify ratelimits is an empty array
		require.Empty(t, res.Body.Data.Ratelimits)
	})

	t.Run("retrieve identity with large metadata", func(t *testing.T) {
		// Create an identity with large metadata
		largeMetaExternalID := uid.New(uid.TestPrefix)

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

		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			ExternalID:  largeMetaExternalID,
			Meta:        largeMetaBytes,
		})

		// Retrieve the identity
		req := handler.Request{
			Identity: largeMetaExternalID,
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
		manyRateLimitsExternalID := uid.New(uid.TestPrefix)

		// Create 10 rate limits for the identity
		rateLimits := []struct {
			name     string
			limit    uint64
			duration uint64
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

		ratelimitRequests := make([]seed.CreateRatelimitRequest, 0, len(rateLimits))
		for _, rl := range rateLimits {
			ratelimitRequests = append(ratelimitRequests, seed.CreateRatelimitRequest{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				Name:        rl.name,
				Limit:       rl.limit,
				Duration:    rl.duration,
			})
		}
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			ExternalID:  manyRateLimitsExternalID,
			Ratelimits:  ratelimitRequests,
		})

		// Retrieve the identity
		req := handler.Request{
			Identity: manyRateLimitsExternalID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify all rate limits were returned
		require.Len(t, res.Body.Data.Ratelimits, len(rateLimits))

		// Create a map of rate limits by name to verify each one
		returnedRateLimits := make(map[string]openapi.RatelimitResponse)
		for _, rl := range res.Body.Data.Ratelimits {
			returnedRateLimits[rl.Name] = rl
		}

		// Verify each rate limit
		for _, expected := range rateLimits {
			actual, exists := returnedRateLimits[expected.name]
			require.True(t, exists, "Rate limit %s not found in response", expected.name)
			require.Equal(t, int64(expected.limit), actual.Limit)
			require.Equal(t, int64(expected.duration), actual.Duration)
		}
	})

	t.Run("retrieve recently created identity", func(t *testing.T) {
		// Create a new identity
		recentExternalID := uid.New(uid.TestPrefix)
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			ExternalID:  recentExternalID,
		})

		// Immediately retrieve the identity
		req := handler.Request{
			Identity: recentExternalID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify it's returned correctly
		require.Equal(t, recentExternalID, res.Body.Data.ExternalId)
	})

	t.Run("retrieve identity with identityId", func(t *testing.T) {
		// Create identity and capture the internal ID
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  uid.New(uid.TestPrefix),
		})

		req := handler.Request{
			Identity: identity.ID,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status)
		require.Equal(t, identity.ExternalID, res.Body.Data.ExternalId)
	})

	t.Run("prefer internal ID over matching external ID", func(t *testing.T) {
		internalIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  uid.New(uid.TestPrefix),
		})
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  internalIdentity.ID,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Identity: internalIdentity.ID,
		})

		require.Equal(t, http.StatusOK, res.Status)
		require.Equal(t, internalIdentity.ID, res.Body.Data.Id)
		require.Equal(t, internalIdentity.ExternalID, res.Body.Data.ExternalId)
	})
}
