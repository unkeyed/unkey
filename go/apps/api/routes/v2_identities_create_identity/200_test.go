package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateIdentitySuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a identity via DB
	t.Run("insert identity via DB", func(t *testing.T) {
		identityID := uid.New(uid.IdentityPrefix)
		externalTestID := uid.New("test_external_id")
		err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalTestID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Meta:        []byte("{}"),
			CreatedAt:   time.Now().UnixMilli(),
			Environment: "default",
		})
		require.NoError(t, err)

		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			IdentityID:  identityID,
			Deleted:     false,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, externalTestID)
	})

	// Create a identity with ratelimits via DB
	t.Run("insert identity via DB and add ratelimits", func(t *testing.T) {
		identityID := uid.New(uid.IdentityPrefix)
		externalTestID := uid.New("test_external_id")
		err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalTestID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Meta:        []byte("{}"),
			CreatedAt:   time.Now().UnixMilli(),
			Environment: "default",
		})
		require.NoError(t, err)

		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			IdentityID:  identityID,
			Deleted:     false,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, externalTestID)

		err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
			ID:          uid.New(uid.RatelimitPrefix),
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  sql.NullString{String: identityID, Valid: true},
			Name:        "Requests",
			Limit:       15,
			Duration:    (time.Minute * 15).Milliseconds(),
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		rateLimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identityID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimits, 1)
		require.Equal(t, rateLimits[0].Name, "Requests")
		require.Equal(t, rateLimits[0].Limit, int32(15))
		require.Equal(t, rateLimits[0].Duration, (time.Minute * 15).Milliseconds())
	})

	// Test creating a identity with no other information
	t.Run("create identity", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")
		req := handler.Request{ExternalId: externalTestID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalTestID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)
	})

	// Test creating a identity with metadata
	t.Run("create identity with metadata", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")
		meta := &map[string]any{"key": "example"}
		req := handler.Request{
			ExternalId: externalTestID,
			Meta:       meta,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalTestID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)

		var dbMeta map[string]any
		err = json.Unmarshal(identity.Meta, &dbMeta)
		require.NoError(t, err)
		require.Equal(t, *meta, dbMeta)
	})

	// Test creating a identity with ratelimits
	t.Run("create identity with ratelimits", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")
		identityRateLimits := []openapi.RatelimitRequest{
			{
				Duration:  time.Minute.Milliseconds(),
				Limit:     100,
				Name:      "test",
				AutoApply: true,
			},
			{
				Duration:  time.Minute.Milliseconds(),
				Limit:     200,
				Name:      "test2",
				AutoApply: true,
			},
		}

		req := handler.Request{
			ExternalId: externalTestID,
			Ratelimits: &identityRateLimits,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalTestID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)

		rateLimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimits, len(identityRateLimits))

		for _, ratelimit := range identityRateLimits {
			idx := -1
			for i, limit := range rateLimits {
				if limit.Name == ratelimit.Name {
					idx = i
					break
				}
			}

			require.True(t, idx >= 0 && idx < len(rateLimits), "Rate limit with name %s not found in the database", ratelimit.Name)
			require.Equal(t, rateLimits[idx].Duration, ratelimit.Duration)
			require.Equal(t, int64(rateLimits[idx].Limit), ratelimit.Limit)
			require.Equal(t, rateLimits[idx].Name, ratelimit.Name)
		}
	})

	// Test creating a identity with both metadata and rate limits
	t.Run("create identity with metadata and ratelimits", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")

		meta := &map[string]any{"userId": "user_123", "role": "admin"}

		identityRateLimits := []openapi.RatelimitRequest{
			{
				Duration:  time.Minute.Milliseconds(),
				Limit:     100,
				Name:      "requests_per_minute",
				AutoApply: true,
			},
			{
				Duration:  (time.Hour * 24).Milliseconds(),
				Limit:     1000,
				Name:      "requests_per_day",
				AutoApply: true,
			},
		}

		req := handler.Request{
			ExternalId: externalTestID,
			Meta:       meta,
			Ratelimits: &identityRateLimits,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalTestID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)

		// Verify metadata
		var dbMeta map[string]any
		err = json.Unmarshal(identity.Meta, &dbMeta)
		require.NoError(t, err)
		require.Equal(t, *meta, dbMeta)

		// Verify rate limits
		rateLimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimits, len(identityRateLimits))

		for _, ratelimit := range identityRateLimits {
			found := false
			for _, limit := range rateLimits {
				if limit.Name == ratelimit.Name {
					require.Equal(t, limit.Duration, ratelimit.Duration)
					require.Equal(t, int64(limit.Limit), ratelimit.Limit)
					found = true
					break
				}
			}
			require.True(t, found, "Rate limit with name %s not found in the database", ratelimit.Name)
		}
	})

	// Test creating an identity with complex nested metadata
	t.Run("create identity with complex nested metadata", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")

		nestedMeta := &map[string]any{
			"user": map[string]any{
				"id":     "u_12345",
				"name":   "John Doe",
				"active": true,
				"roles":  []string{"admin", "editor"},
				"preferences": map[string]any{
					"theme":         "dark",
					"notifications": true,
					"languages":     []any{"en", "fr", "de"},
				},
			},
			"metrics": map[string]any{
				"logins":     42,
				"lastActive": "2023-08-15T14:30:00Z",
				"devices": []map[string]any{
					{"type": "mobile", "os": "iOS", "lastLogin": "2023-08-15T14:30:00Z"},
					{"type": "desktop", "os": "Windows", "lastLogin": "2023-08-14T10:15:00Z"},
				},
			},
			"features": []string{"feature1", "feature2", "feature3"},
			"limits": map[string]any{
				"storage": 5000,
				"api": map[string]any{
					"rate":  100,
					"burst": 150,
				},
			},
		}

		req := handler.Request{
			ExternalId: externalTestID,
			Meta:       nestedMeta,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalTestID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)

		// Verify complex metadata is correctly stored and retrieved
		var dbMeta map[string]any
		err = json.Unmarshal(identity.Meta, &dbMeta)
		require.NoError(t, err)

		// Convert expected and actual to JSON strings for comparison to handle potential subtle differences in map types
		expectedJSON, err := json.Marshal(nestedMeta)
		require.NoError(t, err)
		actualJSON, err := json.Marshal(dbMeta)
		require.NoError(t, err)

		require.JSONEq(t, string(expectedJSON), string(actualJSON), "Complex nested metadata was not stored correctly")
	})

	// Test creating multiple identities with different externalIds
	t.Run("create multiple identities with different externalIds", func(t *testing.T) {
		externalIDs := []string{
			uid.New("test_external_id_1"),
			uid.New("test_external_id_2"),
			uid.New("test_external_id_3"),
		}
		identityIDs := make([]string, 0)

		for _, externalID := range externalIDs {
			req := handler.Request{ExternalId: externalID}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

			require.Equal(t, 200, res.Status, "expected 200 for externalId %s", externalID)
			require.NotNil(t, res.Body)
		}

		// Verify each identity was created with the correct externalId
		for i, externalID := range externalIDs {
			identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				ExternalID:  externalID,
				Deleted:     false,
			})
			identityIDs = append(identityIDs, identity.ID)
			require.NoError(t, err)
			require.Equal(t, externalIDs[i], identity.ExternalID)
		}

		// Verify identities have unique IDs
		idMap := make(map[string]bool)
		for _, id := range identityIDs {
			idMap[id] = true
		}

		require.Len(t, idMap, len(identityIDs), "Identity IDs should be unique")
	})

	// Edge case: Create identity with externalId at exactly minimum length (3 characters)
	t.Run("create identity with minimum length externalId", func(t *testing.T) {
		externalTestID := "123" // Exactly 3 characters

		req := handler.Request{ExternalId: externalTestID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify in database
		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalTestID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, externalTestID, identity.ExternalID)
	})

	// Edge case: Create identity with various metadata types
	t.Run("create identity with various metadata types", func(t *testing.T) {
		externalTestID := uid.New("ext")

		// Create metadata with various types: strings, numbers, booleans, arrays, null
		meta := &map[string]any{
			"stringValue": "test string",
			"intValue":    42,
			"floatValue":  3.14159,
			"boolValue":   true,
			"nullValue":   nil,
			"arrayValue":  []any{1, "two", true, nil, 4.5},
			"mapValue": map[string]any{
				"nestedString": "nested",
				"nestedInt":    123,
				"nestedBool":   false,
			},
			"emptyArray":  []any{},
			"emptyObject": map[string]any{},
		}

		req := handler.Request{
			ExternalId: externalTestID,
			Meta:       meta,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify in database
		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalTestID,
			Deleted:     false,
		})
		require.NoError(t, err)

		// Verify metadata
		var dbMeta map[string]any
		err = json.Unmarshal(identity.Meta, &dbMeta)
		require.NoError(t, err)

		// Convert to JSON for comparison
		expectedJSON, err := json.Marshal(meta)
		require.NoError(t, err)
		actualJSON, err := json.Marshal(dbMeta)
		require.NoError(t, err)

		require.JSONEq(t, string(expectedJSON), string(actualJSON), "Metadata with various types was not stored correctly")
	})
}
