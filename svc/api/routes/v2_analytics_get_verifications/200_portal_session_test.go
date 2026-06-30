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
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_analytics_get_verifications"
)

// portalMiddleware returns a middleware stack that authenticates requests
// (including portal session cookies) but skips OpenAPI spec validation. The
// OpenAPI spec only declares rootKey security, so cookie-authenticated portal
// requests would be rejected by the validator.
func portalMiddleware(h *testutil.Harness) []zen.Middleware {
	return []zen.Middleware{
		zen.WithObservability(),
		zen.WithLogging(),
		middleware.WithErrorHandling(),
		middleware.WithAuthentication(middleware.AuthenticationConfig{
			Auth:       h.Auth,
			Database:   h.DB,
			QuotaCache: h.Caches.WorkspaceQuota,
			Ratelimit:  h.Ratelimit,
		}),
	}
}

// createPortalSession inserts a portal session row and returns a cookie header
// suitable for use in CallRoute.
func createPortalSession(
	t *testing.T,
	h *testutil.Harness,
	workspaceID string,
	externalID string,
	permissions []string,
) http.Header {
	t.Helper()
	ctx := context.Background()

	sessionID := uid.New(uid.PortalSessionPrefix)

	permsJSON, err := json.Marshal(permissions)
	require.NoError(t, err)

	err = db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:             sessionID,
		WorkspaceID:    workspaceID,
		PortalConfigID: uid.New(uid.PortalConfigPrefix),
		ExternalID:     externalID,
		Permissions:    permsJSON,
		Preview:        false,
		ExpiresAt:      time.Now().Add(24 * time.Hour).UnixMilli(),
		CreatedAt:      time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	return http.Header{
		"Content-Type": {"application/json"},
		"Cookie":       {fmt.Sprintf("portal_session=%s", sessionID)},
	}
}

// TestPortalSessionAnalyticsScopedToOwnKeys verifies that a portal session only
// sees verification events for keys belonging to its own externalId, even when
// other identities in the same workspace have their own verification events.
func TestPortalSessionAnalyticsScopedToOwnKeys(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := &handler.Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route, portalMiddleware(h)...)

	// Identity A (the portal session's identity) owns one key.
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

	// A also has a soft-deleted key. Its verification events are immutable
	// history that still belongs to A, so they must remain visible after the key
	// is deleted (matching the root-key analytics path, which is scoped by
	// key_space_id and is unaffected by key deletion).
	keyADeleted := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})
	err := db.Query.SoftDeleteKeyByID(context.Background(), h.DB.RW(), db.SoftDeleteKeyByIDParams{
		Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:  keyADeleted.KeyID,
	})
	require.NoError(t, err)

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

	// 3 events for A's live key.
	for i := range 3 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       keyA.KeyID,
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  identityA.ID,
			Tags:        []string{},
		})
	}

	// 2 events for A's soft-deleted key; these must still be counted.
	for i := range 2 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       keyADeleted.KeyID,
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  identityA.ID,
			Tags:        []string{},
		})
	}

	// 5 events for B's key.
	for i := range 5 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       keyB.KeyID,
			Region:      "us-east-1",
			Outcome:     "VALID",
			IdentityID:  identityB.ID,
			Tags:        []string{},
		})
	}

	headers := createPortalSession(t, h, workspace.ID, externalA, []string{
		"api.*.read_analytics",
	})

	req := handler.Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)
		require.Len(c, res.Body.Data, 1)

		// A's 3 live-key events plus 2 deleted-key events = 5, never B's 5.
		count, ok := res.Body.Data[0]["count"]
		require.True(c, ok, "count field should exist")
		require.Equal(c, float64(5), count, "portal session should see its own keys' events, including deleted keys, but never another identity's")
	}, 30*time.Second, time.Second)
}

// TestPortalSessionAnalyticsNonWildcardPermission verifies the combined filter
// path: a portal session scoped to a single API via a non-wildcard
// api.<id>.read_analytics permission gets both a key_space_id filter (from the
// permission) and a key_id filter (from identity scoping). It must see only its
// own keys within the permitted API, never its keys in other APIs nor other
// identities' keys in the permitted API.
func TestPortalSessionAnalyticsNonWildcardPermission(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api1 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	api2 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := &handler.Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route, portalMiddleware(h)...)

	// Identity A owns a key in each API.
	externalA := "portal_user_A"
	identityA := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalA,
	})
	keyA1 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api1.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})
	keyA2 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api2.KeyAuthID.String,
		IdentityID:  ptr.P(identityA.ID),
	})

	// Identity B owns a key in api1 whose events must never be visible to A.
	identityB := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "portal_user_B",
	})
	keyB1 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api1.KeyAuthID.String,
		IdentityID:  ptr.P(identityB.ID),
	})

	now := time.Now().UnixMilli()

	// 4 events for A's api1 key (the only events that should be counted).
	for i := range 4 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api1.KeyAuthID.String,
			KeyID:       keyA1.KeyID,
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  identityA.ID,
			Tags:        []string{},
		})
	}

	// 3 events for A's api2 key; excluded by the key_space_id filter.
	for i := range 3 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api2.KeyAuthID.String,
			KeyID:       keyA2.KeyID,
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  identityA.ID,
			Tags:        []string{},
		})
	}

	// 5 events for B's api1 key; excluded by the key_id filter.
	for i := range 5 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api1.KeyAuthID.String,
			KeyID:       keyB1.KeyID,
			Region:      "us-east-1",
			Outcome:     "VALID",
			IdentityID:  identityB.ID,
			Tags:        []string{},
		})
	}

	// Permission is scoped to api1 only (non-wildcard); the query must reference
	// api1's key_space_id so authorization resolves to the api1 permission.
	headers := createPortalSession(t, h, workspace.ID, externalA, []string{
		fmt.Sprintf("api.%s.read_analytics", api1.ID),
	})

	req := handler.Request{
		Query: fmt.Sprintf("SELECT COUNT(*) as count FROM key_verifications_v1 WHERE key_space_id = '%s'", api1.KeyAuthID.String),
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)
		require.Len(c, res.Body.Data, 1)

		// Only A's 4 api1 events: not A's 3 api2 events (key_space_id filter),
		// not B's 5 api1 events (key_id filter).
		count, ok := res.Body.Data[0]["count"]
		require.True(c, ok, "count field should exist")
		require.Equal(c, float64(4), count, "combined key_space_id + key_id filters should yield only the session's keys in the permitted API")
	}, 30*time.Second, time.Second)
}

// TestPortalSessionAnalyticsNoKeysReturnsEmpty verifies that a portal session
// whose identity owns no keys receives empty analytics rather than an error or
// another identity's data.
func TestPortalSessionAnalyticsNoKeysReturnsEmpty(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := &handler.Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route, portalMiddleware(h)...)

	// Another identity has events; the portal session's identity has zero keys.
	otherIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "portal_user_B",
	})
	otherKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(otherIdentity.ID),
	})

	now := time.Now().UnixMilli()
	for i := range 5 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       otherKey.KeyID,
			Region:      "us-east-1",
			Outcome:     "VALID",
			IdentityID:  otherIdentity.ID,
			Tags:        []string{},
		})
	}

	headers := createPortalSession(t, h, workspace.ID, "portal_user_zero_keys", []string{
		"api.*.read_analytics",
	})

	req := handler.Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	// The handler short-circuits before touching ClickHouse, so this is
	// deterministic and does not need to wait for buffered data.
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
	require.Empty(t, res.Body.Data, "session with no keys should receive empty analytics")
}

// TestPortalSessionAnalyticsWithoutPermissionForbidden verifies that RBAC
// authorization is enforced even when the session's identity owns no keys. The
// zero-keys path must not short-circuit before Authorize, otherwise a session
// lacking read_analytics would receive an empty 200 instead of a 403.
func TestPortalSessionAnalyticsWithoutPermissionForbidden(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	_ = h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := &handler.Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route, portalMiddleware(h)...)

	// Session owns no keys AND has no read_analytics permission.
	headers := createPortalSession(t, h, workspace.ID, "portal_user_no_perm", []string{
		"api.*.read_api",
	})

	req := handler.Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 403, res.Status, "session without read_analytics must be forbidden even with zero keys")
}
