package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

type TestIdentity struct {
	ID           string
	ExternalID   string
	RatelimitIds []string
}

// Helper function that creates a new identity with rate-limits and returns it
func createTestIdentity(t *testing.T, h *testutil.Harness, numberOfRatelimits int) TestIdentity {
	identityID := uid.New(uid.IdentityPrefix)
	externalID := uid.New("test_external_id")

	err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Meta:        nil,
		CreatedAt:   time.Now().UnixMilli(),
		Environment: "default",
	})
	require.NoError(t, err)

	ratelimitIds := make([]string, 0, numberOfRatelimits)
	for i := 0; i < numberOfRatelimits; i++ {
		rateLimitID := uid.New(uid.RatelimitPrefix)
		err = db.Query.InsertIdentityRatelimit(t.Context(), h.DB.RW(), db.InsertIdentityRatelimitParams{
			ID:          rateLimitID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  sql.NullString{String: identityID, Valid: true},
			Name:        fmt.Sprintf("Rate Limit %d", i+1),
			Limit:       10,
			Duration:    (time.Minute * 10).Milliseconds(),
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)
		ratelimitIds = append(ratelimitIds, rateLimitID)
	}

	return TestIdentity{
		ID:           identityID,
		ExternalID:   externalID,
		RatelimitIds: ratelimitIds,
	}
}

func TestDeleteIdentitySuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("delete identity by ID", func(t *testing.T) {
		testIdentity := createTestIdentity(t, h, 0)

		// Verify identity exists before deletion
		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      testIdentity.ID,
			Deleted: false,
		})
		require.NoError(t, err)
		require.Equal(t, testIdentity.ID, identity.ID)

		// Delete the identity via API
		req := handler.Request{IdentityId: ptr.P(testIdentity.ID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted (no longer found with deleted=false)
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      testIdentity.ID,
			Deleted: false,
		})
		require.Equal(t, sql.ErrNoRows, err, "identity should not be found with deleted=false")

		// Verify identity still exists but marked as deleted
		deletedIdentity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      testIdentity.ID,
			Deleted: true,
		})
		require.NoError(t, err, "identity should still exist with deleted=true")
		require.Equal(t, testIdentity.ID, deletedIdentity.ID)
		require.True(t, deletedIdentity.Deleted)
	})

	t.Run("delete identity by external ID", func(t *testing.T) {
		testIdentity := createTestIdentity(t, h, 0)

		// Verify identity exists before deletion
		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  testIdentity.ExternalID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, testIdentity.ExternalID, identity.ExternalID)

		// Delete the identity via API
		req := handler.Request{ExternalId: ptr.P(testIdentity.ExternalID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted
		_, err = db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  testIdentity.ExternalID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err, "identity should not be found with deleted=false")

		// Verify identity still exists but marked as deleted
		deletedIdentity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  testIdentity.ExternalID,
			Deleted:     true,
		})
		require.NoError(t, err, "identity should still exist with deleted=true")
		require.Equal(t, testIdentity.ExternalID, deletedIdentity.ExternalID)
		require.True(t, deletedIdentity.Deleted)
	})

	t.Run("delete identity with rate limits", func(t *testing.T) {
		numberOfRatelimits := 3
		testIdentity := createTestIdentity(t, h, numberOfRatelimits)

		// Verify rate limits exist
		rateLimits, err := db.Query.FindRatelimitsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: testIdentity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimits, numberOfRatelimits)

		// Delete the identity via API
		req := handler.Request{IdentityId: ptr.P(testIdentity.ID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      testIdentity.ID,
			Deleted: false,
		})
		require.Equal(t, sql.ErrNoRows, err)

		// Verify rate limits still exist (they should remain for audit purposes)
		rateLimitsAfterDeletion, err := db.Query.FindRatelimitsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: testIdentity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimitsAfterDeletion, numberOfRatelimits, "rate limits should still exist after soft deletion")
	})

	t.Run("delete identity with wildcard permission", func(t *testing.T) {
		testIdentity := createTestIdentity(t, h, 0)

		// Create root key with wildcard permission
		wildcardKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
		wildcardHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", wildcardKey)},
		}

		req := handler.Request{IdentityId: ptr.P(testIdentity.ID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, wildcardHeaders, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted
		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      testIdentity.ID,
			Deleted: false,
		})
		require.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("delete identity with specific permission", func(t *testing.T) {
		testIdentity := createTestIdentity(t, h, 0)

		// Create root key with specific identity permission
		specificKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("identity.%s.delete_identity", testIdentity.ID))
		specificHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", specificKey)},
		}

		req := handler.Request{IdentityId: ptr.P(testIdentity.ID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, specificHeaders, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted
		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      testIdentity.ID,
			Deleted: false,
		})
		require.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("verify audit logs are created", func(t *testing.T) {
		testIdentity := createTestIdentity(t, h, 2)

		// Delete the identity
		req := handler.Request{IdentityId: ptr.P(testIdentity.ID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)

		// Verify audit logs were created
		auditLogs, err := db.Query.FindAuditLogTargetById(ctx, h.DB.RO(), testIdentity.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(auditLogs), 1, "should have audit logs for identity deletion")

		// Look for identity deletion event
		var foundDeleteEvent bool
		for _, log := range auditLogs {
			if log.AuditLog.Event == "identity.delete" {
				foundDeleteEvent = true
				require.Equal(t, h.Resources().UserWorkspace.ID, log.AuditLog.WorkspaceID)
				break
			}
		}
		require.True(t, foundDeleteEvent, "should have identity deletion audit log")
	})

	t.Run("delete identity twice (duplicate key error handling)", func(t *testing.T) {
		testIdentity := createTestIdentity(t, h, 0)

		// Delete the identity once
		req := handler.Request{IdentityId: ptr.P(testIdentity.ID)}
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status, "first deletion should succeed")

		// Create a new identity with the same external ID (this will trigger the duplicate key scenario)
		newIdentityID := uid.New(uid.IdentityPrefix)
		err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
			ID:          newIdentityID,
			ExternalID:  testIdentity.ExternalID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Meta:        nil,
			CreatedAt:   time.Now().UnixMilli(),
			Environment: "default",
		})
		require.NoError(t, err)

		// Delete the new identity (this should trigger duplicate key error handling)
		req2 := handler.Request{IdentityId: ptr.P(newIdentityID)}
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req2)
		require.Equal(t, 200, res2.Status, "second deletion should succeed despite duplicate key scenario")

		// Verify the new identity is soft deleted
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      newIdentityID,
			Deleted: false,
		})
		require.Equal(t, sql.ErrNoRows, err)

		// Verify the old identity was hard deleted (should not be found even with deleted=true)
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      testIdentity.ID,
			Deleted: true,
		})
		require.Equal(t, sql.ErrNoRows, err, "old identity should be hard deleted")
	})
}