package handler_test

import (
	"context"
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

	// 3 events for A's key.
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

		// Only A's 3 events, never B's 5.
		count, ok := res.Body.Data[0]["count"]
		require.True(c, ok, "count field should exist")
		require.Equal(c, float64(3), count, "portal session should only see its own keys' events")
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
