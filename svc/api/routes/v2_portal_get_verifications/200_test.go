package handler_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_verifications"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

type (
	Request  = openapi.V2PortalGetVerificationsRequestBody
	Response = openapi.V2PortalGetVerificationsResponseBody
)

// newHandler builds the standalone portal.getVerifications handler backed by the
// harness's shared ClickHouse client.
func newHandler(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		ClickHouse: h.ClickHouse,
		DB:         h.DB,
		QuotaCache: h.Caches.WorkspaceQuota,
	}
}

// sumTotals adds up the Total across every bucket in the timeseries.
func sumTotals(points []openapi.V2PortalGetVerificationsDataPoint) int64 {
	var total int64
	for _, p := range points {
		total += p.Total
	}
	return total
}

// TestPortalSessionAnalyticsScopedToOwnKeys verifies a portal session only sees
// verification events attributed to its own externalId, even when another
// identity in the same workspace has its own events, and that events for a
// soft-deleted key still count (scoping is by external_id at write time, not by
// current key ownership).
func TestPortalSessionAnalyticsScopedToOwnKeys(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	// Identity A (the portal session's identity) owns one live key.
	externalA := "portal_user_A"
	identityA := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalA,
	})
	keyA := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})

	// A also has a soft-deleted key; its events carry A's external_id and must
	// still be counted.
	keyADeleted := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})
	require.NoError(t, db.Query.SoftDeleteKeyByID(context.Background(), h.DB.RW(), db.SoftDeleteKeyByIDParams{
		Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:  keyADeleted.KeyID,
	}))

	// Identity B owns a different key whose events must never be visible to A.
	identityB := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "portal_user_B",
	})
	keyB := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identityB.ID),
	})

	now := time.Now().UnixMilli()

	buffer := func(keyID, externalID, identityID string, n int) {
		for i := range n {
			h.KeyVerifications.Buffer(schema.KeyVerification{
				RequestID:   uid.New(uid.RequestPrefix),
				Time:        now - int64(i*1000),
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				KeyID:       keyID,
				Region:      "us-west-1",
				Outcome:     "VALID",
				IdentityID:  identityID,
				ExternalID:  externalID,
				Tags:        []string{},
			})
		}
	}

	buffer(keyA.KeyID, externalA, identityA.ID, 3)            // A live key
	buffer(keyADeleted.KeyID, externalA, identityA.ID, 2)     // A deleted key
	buffer(keyB.KeyID, identityB.ExternalID, identityB.ID, 5) // B key (must not leak)

	headers := h.CreatePortalSession(workspace.ID, externalA, []string{api.KeyAuthID.String}, []string{"analytics:read"})

	// Window of ~1h ending just after now -> minute-bucket granularity.
	req := Request{
		StartTime: now - int64(time.Hour/time.Millisecond),
		EndTime:   now + int64(time.Minute/time.Millisecond),
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)

		// A's 3 live-key events + 2 deleted-key events = 5, never B's 5.
		require.Equal(c, int64(5), sumTotals(res.Body.Data),
			"portal session should see its own keys' events (including deleted keys) but never another identity's")
	}, 30*time.Second, time.Second)
}

// TestPortalSessionAnalyticsKeyIdFilter verifies the optional keyId narrows the
// timeseries to a single key while staying scoped to the session identity.
func TestPortalSessionAnalyticsKeyIdFilter(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	externalA := "portal_user_A"
	identityA := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalA,
	})
	targetKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})
	otherKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})

	now := time.Now().UnixMilli()
	for i := range 4 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       targetKey.KeyID,
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  identityA.ID,
			ExternalID:  externalA,
			Tags:        []string{},
		})
	}
	for i := range 6 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       otherKey.KeyID,
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  identityA.ID,
			ExternalID:  externalA,
			Tags:        []string{},
		})
	}

	headers := h.CreatePortalSession(workspace.ID, externalA, []string{api.KeyAuthID.String}, []string{"analytics:read"})

	req := Request{
		StartTime: now - int64(time.Hour/time.Millisecond),
		EndTime:   now + int64(time.Minute/time.Millisecond),
		KeyId:     ptr.P(targetKey.KeyID),
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)
		require.Equal(c, int64(4), sumTotals(res.Body.Data),
			"keyId filter should return only the target key's events")
	}, 30*time.Second, time.Second)
}

// TestPortalSessionAnalyticsRequiresReadAnalytics verifies that a portal session
// whose permissions do not include a read_analytics grant is rejected, even
// though the endpoint is otherwise scoped to its own identity. The workspace
// owner gates analytics access via the session permissions.
func TestPortalSessionAnalyticsRequiresReadAnalytics(t *testing.T) {
	// No ClickHouse needed: the handler rejects on the permission check before it
	// ever queries analytics.
	h := testutil.NewHarness(t, testutil.HarnessConfig{})

	workspace := h.CreateWorkspace()
	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	// Session granted key access but not analytics.
	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{"ks_none"}, []string{"keys:read"})

	now := time.Now().UnixMilli()
	res := testutil.CallRoute[Request, Response](h, route, headers, Request{
		StartTime: now - int64(time.Hour/time.Millisecond),
		EndTime:   now + int64(time.Minute/time.Millisecond),
	})

	require.Equal(t, 403, res.Status,
		"portal session without read_analytics must be forbidden from reading analytics")
}
