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
	"github.com/unkeyed/unkey/go/pkg/array"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteIdentitySuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("delete identity by external ID", func(t *testing.T) {
		externalID := "test_user_1"
		identityID := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        []byte("{}"),
		})

		// Verify identity exists before deletion
		identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Equal(t, externalID, identity.ExternalID)

		// Delete the identity via API
		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted
		_, err = db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err, "identity should not be found with deleted=false")

		// Verify identity still exists but marked as deleted
		deletedIdentity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID,
			Deleted:     true,
		})
		require.NoError(t, err, "identity should still exist with deleted=true")
		require.Equal(t, externalID, deletedIdentity.ExternalID)
		require.True(t, deletedIdentity.Deleted)
	})

	t.Run("delete identity with rate limits", func(t *testing.T) {
		numberOfRatelimits := 3
		externalID := "test_user_with_ratelimits"

		ratelimits := array.Fill(
			numberOfRatelimits,
			func() seed.CreateRatelimitRequest {
				return seed.CreateRatelimitRequest{
					Name:        fmt.Sprintf("ratelimit_%s", uid.New("test", 3)),
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       100,
					Duration:    time.Minute.Milliseconds(),
				}
			},
		)

		identityID := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        []byte("{}"),
			Ratelimits:  ratelimits,
		})

		// Verify rate limits exist
		rateLimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identityID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimits, numberOfRatelimits)

		// Delete the identity via API
		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err)

		// Verify rate limits still exist (they should remain for audit purposes)
		rateLimitsAfterDeletion, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identityID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimitsAfterDeletion, numberOfRatelimits, "rate limits should still exist after soft deletion")
	})

	t.Run("delete identity with wildcard permission", func(t *testing.T) {
		externalID := "test_user_wildcard"
		identityID := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        []byte("{}"),
		})

		// Create root key with wildcard permission
		wildcardKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
		wildcardHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", wildcardKey)},
		}

		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, wildcardHeaders, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		// Verify identity is soft deleted
		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("verify audit logs are created", func(t *testing.T) {
		externalID := "test_user_audit_logs"
		identityID := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        []byte("{}"),
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "ratelimit_1",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       100,
					Duration:    time.Minute.Milliseconds(),
				},
				{
					Name:        "ratelimit_2",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       200,
					Duration:    time.Hour.Milliseconds(),
				},
			},
		})

		// Delete the identity
		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)

		// Verify audit logs were created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), identityID)
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
		externalID := "test_user_duplicate"

		// Create first identity
		identityID1 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        []byte("{}"),
		})

		// Delete the identity once
		req := handler.Request{Identity: externalID}
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status, "first deletion should succeed")

		// Create a new identity with the same external ID (this will trigger the duplicate key scenario)
		identityID2 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        []byte("{}"),
		})

		// Delete the new identity (this should trigger duplicate key error handling)
		req2 := handler.Request{Identity: externalID}
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req2)
		require.Equal(t, 200, res2.Status, "second deletion should succeed despite duplicate key scenario")

		// Verify the new identity is soft deleted
		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID2,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err)

		// Verify the old identity was hard deleted (should not be found even with deleted=true)
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID1,
			Deleted:     true,
		})
		require.Equal(t, sql.ErrNoRows, err, "old identity should be hard deleted")
	})

	t.Run("delete->create->delete with ratelimits", func(t *testing.T) {
		// This test simulates a Stripe tier change workflow:
		// 1. User starts with "Advanced" tier (300k requests/month)
		// 2. User downgrades to "Starter" tier (20k requests/month)
		// 3. Implementation: delete old identity, create new identity with new limits
		externalID := "stripe_user_12345"

		// Step 1: Create initial identity with "Advanced" tier ratelimit (300k/month)
		identityID1 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        []byte(`{"tier":"advanced"}`),
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "per_month",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       300000, // 300k requests
					Duration:    (time.Hour * 24 * 30).Milliseconds(),
				},
			},
		})

		// Step 2: Delete old identity (tier downgrade starts)
		req1 := handler.Request{Identity: externalID}
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req1)
		require.Equal(t, 200, res1.Status, "first deletion should succeed")

		// Step 3: Create new identity with "Starter" tier ratelimit (20k/month)
		identityID2 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID, // Same externalId
			Meta:        []byte(`{"tier":"starter"}`),
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "per_month",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       20000, // 20k requests
					Duration:    (time.Hour * 24 * 30).Milliseconds(),
				},
			},
		})

		// Step 4: Delete new identity (this is where the bug used to happen)
		req2 := handler.Request{Identity: externalID}
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req2)
		require.Equal(t, 200, res2.Status, "second deletion should succeed without 500 error")

		// Verify the new identity is soft deleted (not hard deleted)
		deletedIdentity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID2,
			Deleted:     true,
		})
		require.NoError(t, err, "new identity should exist as soft-deleted")
		require.Equal(t, identityID2, deletedIdentity.ID)
		require.True(t, deletedIdentity.Deleted)

		// Verify the new identity cannot be found as active
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID2,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err, "new identity should not be active")

		// Verify the old identity was hard deleted (cleanup)
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identityID1,
			Deleted:     true,
		})
		require.Equal(t, sql.ErrNoRows, err, "old identity should be hard deleted as cleanup")
	})
}
