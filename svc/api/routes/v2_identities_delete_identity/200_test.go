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
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_delete_identity"
)

func TestDeleteIdentitySuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("delete identity by external ID", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		_ = h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
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
			IdentityID:  identity.ID,
			Deleted:     true,
		})
		require.NoError(t, err, "identity should still exist with deleted=true")
		require.Equal(t, externalID, deletedIdentity.ExternalID)
		require.True(t, deletedIdentity.Deleted)
	})

	t.Run("delete identity with rate limits", func(t *testing.T) {
		numberOfRatelimits := 3
		externalID := uid.New(uid.TestPrefix)

		ratelimits := array.Fill(
			numberOfRatelimits,
			func() seed.CreateRatelimitRequest {
				return seed.CreateRatelimitRequest{
					Name:        fmt.Sprintf("ratelimit_%s", uid.New("test", 3)),
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       100,
					Duration:    uint64(time.Minute.Milliseconds()),
				}
			},
		)

		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Ratelimits:  ratelimits,
		})

		// Verify rate limits exist
		rateLimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
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
			IdentityID:  identity.ID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err)

		// Verify rate limits still exist (they should remain for audit purposes)
		rateLimitsAfterDeletion, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimitsAfterDeletion, numberOfRatelimits, "rate limits should still exist after soft deletion")
	})

	t.Run("delete identity with wildcard permission", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
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
			IdentityID:  identity.ID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("verify audit logs are created", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "ratelimit_1",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       100,
					Duration:    uint64(time.Minute.Milliseconds()),
				},
				{
					Name:        "ratelimit_2",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       200,
					Duration:    uint64(time.Hour.Milliseconds()),
				},
			},
		})

		// Delete the identity
		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)

		// Verify audit logs were queued in clickhouse_outbox.
		auditLogs := h.FindAuditLogsByTargetID(ctx, t, identity.ID)
		require.GreaterOrEqual(t, len(auditLogs), 1, "should have audit logs for identity deletion")
		var foundDeleteEvent bool
		for _, ev := range auditLogs {
			if ev.Event == "identity.delete" {
				foundDeleteEvent = true
				require.Equal(t, h.Resources().UserWorkspace.ID, ev.WorkspaceID)
				break
			}
		}
		require.True(t, foundDeleteEvent, "should have identity deletion audit log")
	})

	t.Run("delete identity twice (duplicate key error handling)", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)

		// Create first identity
		identity1 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
		})

		// Delete the identity once
		req := handler.Request{Identity: externalID}
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status, "first deletion should succeed")

		// Create a new identity with the same external ID (this will trigger the duplicate key scenario)
		identity2 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
		})

		// Delete the new identity (this should trigger duplicate key error handling)
		req2 := handler.Request{Identity: externalID}
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req2)
		require.Equal(t, 200, res2.Status, "second deletion should succeed despite duplicate key scenario")

		// Verify the new identity is soft deleted
		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identity2.ID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err)

		// Verify the old identity was hard deleted (should not be found even with deleted=true)
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identity1.ID,
			Deleted:     true,
		})
		require.Equal(t, sql.ErrNoRows, err, "old identity should be hard deleted")
		oldEvents := h.FindAuditLogsByTargetID(ctx, t, identity1.ID)
		require.Len(t, oldEvents, 2, "soft deletion and cleanup each audit the identity even without ratelimits")
		currentEvents := h.FindAuditLogsByTargetID(ctx, t, identity2.ID)
		require.Len(t, currentEvents, 1)
		var cleanupEvents int
		for _, event := range oldEvents {
			if event.CorrelationID == currentEvents[0].CorrelationID {
				cleanupEvents++
				require.Equal(t, "identity.delete", event.Event)
				require.Len(t, event.Targets, 1)
				require.Equal(t, identity1.ID, event.Targets[0].ID)
			}
		}
		require.Equal(t, 1, cleanupEvents)
	})

	t.Run("delete->create->delete with ratelimits", func(t *testing.T) {
		// This test simulates a Stripe tier change workflow:
		// 1. User starts with "Advanced" tier (300k requests/month)
		// 2. User downgrades to "Starter" tier (20k requests/month)
		// 3. Implementation: delete old identity, create new identity with new limits
		externalID := uid.New(uid.TestPrefix)

		// Step 1: Create initial identity with "Advanced" tier ratelimit (300k/month)
		advancedMeta, err := json.Marshal(map[string]string{"tier": "advanced"})
		require.NoError(t, err)
		identity1 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
			Meta:        advancedMeta,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "per_month",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       300000, // 300k requests
					Duration:    uint64((time.Hour * 24 * 30).Milliseconds()),
				},
			},
		})

		// Step 2: Delete old identity (tier downgrade starts)
		oldRatelimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identity1.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, oldRatelimits, 1)
		req1 := handler.Request{Identity: externalID}
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req1)
		require.Equal(t, 200, res1.Status, "first deletion should succeed")
		oldEventsBefore := h.FindAuditLogsByTargetID(ctx, t, identity1.ID)

		// Step 3: Create new identity with "Starter" tier ratelimit (20k/month)
		starterMeta, err := json.Marshal(map[string]string{"tier": "starter"})
		require.NoError(t, err)
		identity2 := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID, // Same externalId
			Meta:        starterMeta,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "per_month",
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Limit:       20000, // 20k requests
					Duration:    uint64((time.Hour * 24 * 30).Milliseconds()),
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
			IdentityID:  identity2.ID,
			Deleted:     true,
		})
		require.NoError(t, err, "new identity should exist as soft-deleted")
		require.Equal(t, identity2.ID, deletedIdentity.ID)
		require.True(t, deletedIdentity.Deleted)

		// Verify the new identity cannot be found as active
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identity2.ID,
			Deleted:     false,
		})
		require.Equal(t, sql.ErrNoRows, err, "new identity should not be active")

		// Verify the old identity was hard deleted (cleanup)
		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  identity1.ID,
			Deleted:     true,
		})
		require.Equal(t, sql.ErrNoRows, err, "old identity should be hard deleted as cleanup")

		remainingRatelimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identity1.ID, Valid: true})
		require.NoError(t, err)
		require.Empty(t, remainingRatelimits)
		oldEventsAfter := h.FindAuditLogsByTargetID(ctx, t, identity1.ID)
		require.Len(t, oldEventsAfter, len(oldEventsBefore)+2, "cleanup must audit the old identity and its ratelimit")
		currentEvents := h.FindAuditLogsByTargetID(ctx, t, identity2.ID)
		require.Len(t, currentEvents, 2)
		var cleanupIdentityEvents, cleanupRatelimitEvents int
		for _, event := range oldEventsAfter {
			if event.CorrelationID != currentEvents[0].CorrelationID {
				continue
			}
			switch event.Event {
			case "identity.delete":
				cleanupIdentityEvents++
				require.Len(t, event.Targets, 1)
				require.Equal(t, identity1.ID, event.Targets[0].ID)
				require.Equal(t, "identity", event.Targets[0].Type)
			case "ratelimit.delete":
				cleanupRatelimitEvents++
				require.Len(t, event.Targets, 2)
				require.Equal(t, identity1.ID, event.Targets[0].ID)
				require.Equal(t, "identity", event.Targets[0].Type)
				require.Equal(t, oldRatelimits[0].ID, event.Targets[1].ID)
				require.Equal(t, "ratelimit", event.Targets[1].Type)
			}
		}
		require.Equal(t, 1, cleanupIdentityEvents)
		require.Equal(t, 1, cleanupRatelimitEvents)
	})
}

// TestDeleteIdentityCleanupRollsBackWithAuditFailure protects atomic cleanup,
// current identity deletion and outbox persistence when audit insertion fails.
func TestDeleteIdentityCleanupRollsBackWithAuditFailure(t *testing.T) {
	ctx := t.Context()
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	externalID := uid.New(uid.TestPrefix)
	old := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  externalID,
		Ratelimits: []seed.CreateRatelimitRequest{
			{Name: "old_limit", WorkspaceID: workspaceID, Limit: 37, Duration: 1000},
		},
	})
	require.NoError(t, db.Query.SoftDeleteIdentity(ctx, h.DB.RW(), db.SoftDeleteIdentityParams{
		WorkspaceID: workspaceID,
		IdentityID:  old.ID,
	}))
	current := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: externalID})
	route := &handler.Handler{DB: h.DB, Auditlogs: failingAuditlogs{AuditLogService: h.Auditlogs}}
	h.Register(route)
	rootKey := h.CreateRootKey(workspaceID, "identity.*.delete_identity")
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + rootKey}}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Identity: current.ID})
	require.Equal(t, http.StatusInternalServerError, res.Status, "%s", res.RawBody)
	_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{WorkspaceID: workspaceID, IdentityID: old.ID, Deleted: true})
	require.NoError(t, err)
	_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{WorkspaceID: workspaceID, IdentityID: current.ID, Deleted: false})
	require.NoError(t, err)
	ratelimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: old.ID, Valid: true})
	require.NoError(t, err)
	require.Len(t, ratelimits, 1)
	require.Empty(t, h.FindAuditLogsByTargetID(ctx, t, old.ID))
	require.Empty(t, h.FindAuditLogsByTargetID(ctx, t, current.ID))
}

// failingAuditlogs forces rollback after the real service writes the outbox.
type failingAuditlogs struct {
	auditlogs.AuditLogService
}

func (a failingAuditlogs) Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	if err := a.AuditLogService.Insert(ctx, tx, logs); err != nil {
		return err
	}
	return errors.New("audit persistence failed")
}

// TestConcurrentIdentityCleanupAuditsOnce protects the committed resource set
// from duplicate audit events when deletion requests overlap.
func TestConcurrentIdentityCleanupAuditsOnce(t *testing.T) {
	ctx := t.Context()
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	externalID := uid.New(uid.TestPrefix)
	old := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  externalID,
		Ratelimits: []seed.CreateRatelimitRequest{
			{Name: "burst", WorkspaceID: workspaceID, Limit: 13, Duration: 1000},
			{Name: "daily", WorkspaceID: workspaceID, Limit: 197, Duration: 86400000},
		},
	})
	oldRatelimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: old.ID, Valid: true})
	require.NoError(t, err)
	require.Len(t, oldRatelimits, 2)
	require.NoError(t, db.Query.SoftDeleteIdentity(ctx, h.DB.RW(), db.SoftDeleteIdentityParams{WorkspaceID: workspaceID, IdentityID: old.ID}))
	current := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: externalID})
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	rootKey := h.CreateRootKey(workspaceID, "identity.*.delete_identity")
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + rootKey}}
	start := make(chan struct{})
	statuses := make(chan int, 16)
	for range cap(statuses) {
		go func() {
			<-start
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Identity: current.ID})
			statuses <- res.Status
		}()
	}
	close(start)
	var successful int
	for range cap(statuses) {
		status := <-statuses
		require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, status)
		if status == http.StatusOK {
			successful++
		}
	}
	require.Positive(t, successful)
	oldEvents := h.FindAuditLogsByTargetID(ctx, t, old.ID)
	require.Len(t, oldEvents, 3)
	currentEvents := h.FindAuditLogsByTargetID(ctx, t, current.ID)
	require.Len(t, currentEvents, 1)
	require.NotEmpty(t, currentEvents[0].CorrelationID)
	for _, event := range oldEvents {
		require.Equal(t, currentEvents[0].CorrelationID, event.CorrelationID)
	}
	for _, ratelimit := range oldRatelimits {
		events := h.FindAuditLogsByTargetID(ctx, t, ratelimit.ID)
		require.Len(t, events, 1)
		require.Equal(t, "ratelimit.delete", events[0].Event)
		require.Equal(t, currentEvents[0].CorrelationID, events[0].CorrelationID)
	}
}
