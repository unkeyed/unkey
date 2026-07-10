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
// verification events attributed to its own externalId and configured
// keyspaces, even when the same identity has keys elsewhere. Events for a
// soft-deleted key still count because both scope fields are written to the
// event before deletion.
func TestPortalSessionAnalyticsScopedToOwnKeys(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	otherApi := h.CreateApi(seed.CreateApiRequest{
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
	keyAOtherKeyspace := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  otherApi.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})

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

	buffer := func(keyID, keyspaceID, externalID, identityID string, n int) {
		for i := range n {
			h.KeyVerifications.Buffer(schema.KeyVerification{
				RequestID:   uid.New(uid.RequestPrefix),
				Time:        now - int64(i*1000),
				WorkspaceID: workspace.ID,
				KeySpaceID:  keyspaceID,
				KeyID:       keyID,
				Region:      "us-west-1",
				Outcome:     "VALID",
				IdentityID:  identityID,
				ExternalID:  externalID,
				Tags:        []string{},
			})
		}
	}

	buffer(keyA.KeyID, api.KeyAuthID.String, externalA, identityA.ID, 3)                   // A live key
	buffer(keyADeleted.KeyID, api.KeyAuthID.String, externalA, identityA.ID, 2)            // A deleted key
	buffer(keyAOtherKeyspace.KeyID, otherApi.KeyAuthID.String, externalA, identityA.ID, 7) // A key outside the session keyspaces
	buffer(keyB.KeyID, api.KeyAuthID.String, identityB.ExternalID, identityB.ID, 5)        // B key (must not leak)

	// Prove all same-identity events reached the aggregate before testing that the
	// route excludes the seven events outside the session keyspace.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var total int64
		err := h.ClickHouse.Conn().QueryRow(
			context.Background(),
			"SELECT SUM(count) FROM default.key_verifications_per_minute_v3 WHERE workspace_id = ? AND external_id = ?",
			workspace.ID,
			externalA,
		).Scan(&total)
		assert.NoError(c, err)
		assert.Equal(c, int64(12), total)
	}, 30*time.Second, time.Second)

	headers := h.CreatePortalSession(workspace.ID, externalA, []string{api.KeyAuthID.String}, []string{"analytics:read"})

	// Window of ~1h ending just after now -> minute-bucket granularity.
	req := Request{
		StartTime: now - int64(time.Hour/time.Millisecond),
		EndTime:   now + int64(time.Minute/time.Millisecond),
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)

	// Only A's events in the session keyspace are visible: 3 live-key events plus
	// 2 deleted-key events. Neither B's events nor A's other keyspace leak.
	require.Equal(t, int64(5), sumTotals(res.Body.Data),
		"portal session analytics must remain within its external identity and configured keyspaces")
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

// TestPortalSessionAnalyticsRequiresAnalyticsRead verifies that reading keys
// does not implicitly grant access to analytics.
func TestPortalSessionAnalyticsRequiresAnalyticsRead(t *testing.T) {
	// No ClickHouse needed: the handler rejects on the permission check before it
	// ever queries analytics.
	h := testutil.NewHarness(t, testutil.HarnessConfig{})

	workspace := h.CreateWorkspace()
	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{"ks_none"}, []string{"keys:read"})

	now := time.Now().UnixMilli()
	res := testutil.CallRoute[Request, openapi.ForbiddenErrorResponse](h, route, headers, Request{
		StartTime: now - int64(time.Hour/time.Millisecond),
		EndTime:   now + int64(time.Minute/time.Millisecond),
	})

	require.Equal(t, 403, res.Status,
		"portal session without analytics:read must be forbidden from reading analytics")
	require.NotContains(t, res.RawBody, workspace.ID)
	require.NotContains(t, res.RawBody, "portal_user_A")
	require.NotContains(t, res.RawBody, "ks_none")
}
