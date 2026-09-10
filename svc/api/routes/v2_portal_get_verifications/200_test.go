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
	return newHandlerWithLimits(h, handler.DefaultMaxPerKeySeries, handler.DefaultMaxResponseBytes)
}

// newHandlerWithKeyCap builds the handler with an explicit per-key breakout cap
// so a test can reach it without seeding the production number of keys.
func newHandlerWithKeyCap(h *testutil.Harness, maxPerKeySeries int) *handler.Handler {
	return newHandlerWithLimits(h, maxPerKeySeries, handler.DefaultMaxResponseBytes)
}

// newHandlerWithLimits builds the handler with explicit ceilings so a test can
// reach either one without seeding production-scale data.
func newHandlerWithLimits(h *testutil.Harness, maxPerKeySeries, maxResponseBytes int) *handler.Handler {
	return &handler.Handler{
		ClickHouse:       h.ClickHouse,
		DB:               h.DB,
		LimitsCache:      h.Caches.WorkspaceLimits,
		MaxPerKeySeries:  maxPerKeySeries,
		MaxResponseBytes: maxResponseBytes,
	}
}

// sumKeyTotals maps each per-key entry to the total across its buckets.
func sumKeyTotals(keys []openapi.V2PortalGetVerificationsKeySeries) map[string]int64 {
	totals := make(map[string]int64, len(keys))
	for _, k := range keys {
		for _, p := range k.Data {
			totals[k.KeyId] += p.Total
		}
	}
	return totals
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
// verification events attributed to its own externalId and to a keyspace the
// session is scoped to, even when another identity in the same workspace has its
// own events, and that events for a soft-deleted key still count (scoping is by
// external_id at write time, not by current key ownership).
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

	// A also owns a key in a keyspace the session is not scoped to.
	keyAOutOfScope := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  otherApi.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})

	now := time.Now().UnixMilli()

	buffer := func(keySpaceID, keyID, externalID, identityID string, n int) {
		for i := range n {
			h.KeyVerifications.Buffer(schema.KeyVerification{
				RequestID:   uid.New(uid.RequestPrefix),
				Time:        now - int64(i*1000),
				WorkspaceID: workspace.ID,
				KeySpaceID:  keySpaceID,
				KeyID:       keyID,
				Region:      "us-west-1",
				Outcome:     "VALID",
				IdentityID:  identityID,
				ExternalID:  externalID,
				Tags:        []string{},
			})
		}
	}

	inScope := api.KeyAuthID.String
	buffer(inScope, keyA.KeyID, externalA, identityA.ID, 3)                             // A live key
	buffer(inScope, keyADeleted.KeyID, externalA, identityA.ID, 2)                      // A deleted key
	buffer(inScope, keyB.KeyID, identityB.ExternalID, identityB.ID, 5)                  // B key (must not leak)
	buffer(otherApi.KeyAuthID.String, keyAOutOfScope.KeyID, externalA, identityA.ID, 4) // A, out-of-scope keyspace

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
			"portal session should see its own keys' events (including deleted keys) but never another identity's or another keyspace's")
	}, 30*time.Second, time.Second)

	// A session scoped to both keyspaces sees both keyspaces' events.
	bothHeaders := h.CreatePortalSession(workspace.ID, externalA, []string{inScope, otherApi.KeyAuthID.String}, []string{"analytics:read"})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, bothHeaders, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)
		require.Equal(c, int64(9), sumTotals(res.Body.Data),
			"a session scoped to both keyspaces should see the sum of both")
	}, 30*time.Second, time.Second)

	// Without the flag the response is unchanged: no per-key array at all.
	plain := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 200, plain.Status)
	require.Nil(t, plain.Body.Keys, "the per-key breakout is opt-in")

	// With the flag the same window is broken out per key, under the same
	// identity and keyspace scoping as the account-wide series.
	perKeyReq := req
	perKeyReq.PerKey = ptr.P(true)

	res := testutil.CallRoute[Request, Response](h, route, headers, perKeyReq)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.Keys)

	totals := sumKeyTotals(*res.Body.Keys)
	require.Equal(t, int64(3), totals[keyA.KeyID])
	require.Equal(t, int64(2), totals[keyADeleted.KeyID])
	require.NotContains(t, totals, keyB.KeyID, "another identity's key must not appear")
	require.NotContains(t, totals, keyAOutOfScope.KeyID, "an out-of-scope keyspace's key must not appear")
	require.Len(t, totals, 2, "keys with no traffic in the window are omitted")

	// Naming the out-of-scope key directly is the direct form of the leak the
	// keyspace bound closes: the key belongs to this identity, so only the
	// keyspace predicate keeps its events out.
	namedReq := req
	namedReq.KeyId = ptr.P(keyAOutOfScope.KeyID)
	namedReq.PerKey = ptr.P(true)

	named := testutil.CallRoute[Request, Response](h, route, headers, namedReq)
	require.Equal(t, 200, named.Status)
	require.Zero(t, sumTotals(named.Body.Data), "an out-of-scope key must return no events even when named")
	require.Empty(t, sumKeyTotals(*named.Body.Keys), "an out-of-scope key must produce no per-key series")

	var perKeyGrand int64
	for _, total := range totals {
		perKeyGrand += total
	}
	require.Equal(t, sumTotals(res.Body.Data), perKeyGrand,
		"per-key totals must sum to the account-wide total")
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

	perKeyReq := req
	perKeyReq.PerKey = ptr.P(true)

	res := testutil.CallRoute[Request, Response](h, route, headers, perKeyReq)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.Keys)
	require.Equal(t, map[string]int64{targetKey.KeyID: 4}, sumKeyTotals(*res.Body.Keys),
		"keyId narrows the per-key breakout as well as the account-wide series")
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
