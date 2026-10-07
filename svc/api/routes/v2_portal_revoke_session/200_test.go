package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	exchangeCode "github.com/unkeyed/unkey/svc/api/routes/v2_portal_exchange_code"
	listKeys "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_keys"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

const permission = "portal.*.create_portal_session"

// The first call caches the session, so the rejection proves the revoke wrote
// through the cache. The pending code stops redeeming too.
func TestRevokeSessionStopsActiveAndPendingSessions(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	endUserRoute := listKeys.New(h.DB)
	h.Register(endUserRoute, h.PortalMiddleware()...)
	exchangeRoute := &exchangeCode.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(exchangeRoute, h.PublicMiddleware()...)

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-live")
	sessionHeaders := h.CreatePortalSessionForPortal(
		stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})
	pendingCode := h.CreatePortalSessionInState(t, stored.ID, workspace.ID, "user_1", false, h.Clock.Now().Add(15*time.Minute))

	warm := testutil.CallRoute[listKeys.Request, listKeys.Response](h, endUserRoute, sessionHeaders, listKeys.Request{})
	require.Equal(t, http.StatusOK, warm.Status, "the session must work before the revoke: %s", warm.RawBody)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(2), res.Body.Data.SessionsRevoked)

	rejected := testutil.CallRoute[listKeys.Request, openapi.UnauthorizedErrorResponse](
		h, endUserRoute, sessionHeaders, listKeys.Request{})
	require.Equal(t, http.StatusUnauthorized, rejected.Status,
		"the revoked session must be rejected without waiting for the cache: %s", rejected.RawBody)
	require.Equal(t, "The portal session is invalid or has expired.", rejected.Body.Error.Detail)

	exchanged := testutil.CallRoute[exchangeCode.Request, openapi.UnauthorizedErrorResponse](
		h, exchangeRoute, http.Header{"Content-Type": {"application/json"}}, exchangeCode.Request{Code: pendingCode})
	require.Equal(t, http.StatusUnauthorized, exchanged.Status,
		"a revoked pending code must not be redeemable: %s", exchanged.RawBody)
}

// Expired rows are neither counted nor touched.
func TestRevokeSessionIgnoresExpiredSessions(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-expired")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})
	past := h.Clock.Now().Add(-time.Hour)
	h.CreatePortalSessionInState(t, stored.ID, workspace.ID, "user_1", true, past)
	h.CreatePortalSessionInState(t, stored.ID, workspace.ID, "user_1", false, past)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(1), res.Body.Data.SessionsRevoked, "only the live session counts")

	require.Equal(t, 2, h.CountLivePortalSessions(t, stored.ID, "user_1"),
		"expired rows are left untouched")
}

// Other users on the portal, and this user on other portals, keep access.
func TestRevokeSessionIsScopedToUserAndPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	target, mapping := seedPortal(t, h, workspace.ID, "revoke-target")
	other, otherMapping := seedPortal(t, h, workspace.ID, "revoke-other")
	h.CreatePortalSessionForPortal(target.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})
	h.CreatePortalSessionForPortal(target.ID, workspace.ID, "user_2", []string{mapping.ID}, []string{"keys:read"})
	h.CreatePortalSessionForPortal(other.ID, workspace.ID, "user_1", []string{otherMapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(target.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(1), res.Body.Data.SessionsRevoked)

	require.Equal(t, 0, h.CountLivePortalSessions(t, target.ID, "user_1"))
	require.Equal(t, 1, h.CountLivePortalSessions(t, target.ID, "user_2"),
		"another end user on the same portal is untouched")
	require.Equal(t, 1, h.CountLivePortalSessions(t, other.ID, "user_1"),
		"the same end user on another portal is untouched")
}

// Another workspace's session for the same external id is out of reach.
func TestRevokeSessionLeavesOtherWorkspacesAlone(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace
	foreign := h.CreateWorkspace()

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-home")
	h.CreatePortalSessionForPortal(stored.ID, foreign.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(0), res.Body.Data.SessionsRevoked)
	require.Equal(t, 1, h.CountLivePortalSessions(t, stored.ID, "user_1"),
		"the other workspace's row is untouched")
}

// A repeat call revokes nothing, writes no audit entry, and keeps the first
// revoked_at.
func TestRevokeSessionIsIdempotent(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-twice")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, first.Status, "expected 200, received: %s", first.RawBody)
	require.Equal(t, int64(1), first.Body.Data.SessionsRevoked)

	var sessionID string
	var revokedAt int64
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT id, revoked_at FROM portal_sessions WHERE portal_id = ? AND external_id = ?", stored.ID, "user_1",
	).Scan(&sessionID, &revokedAt))

	h.Clock.Tick(time.Minute)

	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, second.Status, "expected 200, received: %s", second.RawBody)
	require.Equal(t, int64(0), second.Body.Data.SessionsRevoked)

	var after int64
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT revoked_at FROM portal_sessions WHERE portal_id = ? AND external_id = ?", stored.ID, "user_1",
	).Scan(&after))
	require.Equal(t, revokedAt, after, "an already-revoked row keeps its original timestamp")

	metas := revokeAuditMetas(t, h, stored.ID)
	require.Len(t, metas, 1, "only the call that revoked something is audited")
	require.Equal(t, "user_1", metas[0]["externalId"])
	require.Equal(t, float64(1), metas[0]["sessionsRevoked"])
	require.Equal(t, []any{sessionID}, metas[0]["sessionIds"], "the audit entry names the revoked session")
}

// An end user with no sessions is not an error.
func TestRevokeSessionWithNoSessions(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, _ := seedPortal(t, h, workspace.ID, "revoke-empty")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "nobody"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(0), res.Body.Data.SessionsRevoked)
	require.Empty(t, revokeAuditMetas(t, h, stored.ID))
}

// Revoking still works on a disabled portal.
func TestRevokeSessionOnDisabledPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-disabled")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE portals SET enabled = false WHERE id = ?", stored.ID)
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(1), res.Body.Data.SessionsRevoked)
}

// The portal may be named by slug as well as by id.
func TestRevokeSessionBySlug(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-by-slug")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.Slug, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(1), res.Body.Data.SessionsRevoked)
	require.Equal(t, 0, h.CountLivePortalSessions(t, stored.ID, "user_1"))
}

// The cache holds the revoked row, not an unrevoked copy.
func TestRevokeSessionWritesRevokedStateToCache(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-cache")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	var tokenHash string
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT access_token_hash FROM portal_sessions WHERE portal_id = ? AND external_id = ?", stored.ID, "user_1",
	).Scan(&tokenHash))

	cached, hit := h.Caches.PortalSession.Get(context.Background(), tokenHash)
	require.Equal(t, cache.Hit, hit, "the revoked row must be cached")
	require.True(t, cached.RevokedAt.Valid, "the cached row carries the revocation")
}

// Two revokes in the same millisecond each report only the sessions they
// revoked, not rows an earlier revoke already stamped with the same time.
func TestRevokeSessionAtTheSameClockTick(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-same-tick")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, first.Status, "expected 200, received: %s", first.RawBody)
	require.Equal(t, int64(1), first.Body.Data.SessionsRevoked)

	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})
	var newSessionID string
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT id FROM portal_sessions WHERE portal_id = ? AND external_id = ? AND revoked_at IS NULL", stored.ID, "user_1",
	).Scan(&newSessionID))

	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, second.Status, "expected 200, received: %s", second.RawBody)
	require.Equal(t, int64(1), second.Body.Data.SessionsRevoked, "only the new session was revoked")

	metas := revokeAuditMetas(t, h, stored.ID)
	require.Len(t, metas, 2)
	var latest map[string]any
	for _, meta := range metas {
		if ids, ok := meta["sessionIds"].([]any); ok && len(ids) == 1 && ids[0] == newSessionID {
			latest = meta
		}
	}
	require.NotNil(t, latest, "the second audit entry must name only the new session")
	require.Equal(t, float64(1), latest["sessionsRevoked"])
}

// More sessions than one batch holds are all revoked, one audit entry per batch.
func TestRevokeSessionInBatches(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, _ := seedPortal(t, h, workspace.ID, "revoke-batches")
	expiresAt := h.Clock.Now().Add(time.Hour)
	const sessions = 1001
	for range sessions {
		h.CreatePortalSessionInState(t, stored.ID, workspace.ID, "user_1", true, expiresAt)
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(sessions), res.Body.Data.SessionsRevoked)
	require.Equal(t, 0, h.CountLivePortalSessions(t, stored.ID, "user_1"))

	metas := revokeAuditMetas(t, h, stored.ID)
	require.Len(t, metas, 2, "one audit entry per batch")
	revoked := 0.0
	for _, meta := range metas {
		count, ok := meta["sessionsRevoked"].(float64)
		require.True(t, ok, "sessionsRevoked must be a number")
		revoked += count
	}
	require.Equal(t, float64(sessions), revoked)
}

// created_at comes from the minting instance's clock, which can run ahead of
// the revoking instance. A session already in the database is revoked even when
// its created_at is later than the revoke's clock.
func TestRevokeSessionTakesSessionsFromAnAheadClock(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, _ := seedPortal(t, h, workspace.ID, "revoke-ahead-clock")
	expiresAt := h.Clock.Now().Add(time.Hour)
	h.CreatePortalSessionInState(t, stored.ID, workspace.ID, "user_1", true, expiresAt)
	_, err := h.DB.RW().ExecContext(context.Background(),
		"UPDATE portal_sessions SET created_at = ? WHERE portal_id = ?",
		h.Clock.Now().Add(time.Minute).UnixMilli(), stored.ID,
	)
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(1), res.Body.Data.SessionsRevoked)
	require.Equal(t, 0, h.CountLivePortalSessions(t, stored.ID, "user_1"))
}

// The batches only take rows at or below the pk read before they start, so a
// session minted while they run is left alone and the loop ends.
func TestLockLiveSessionsStopsAtTheMaxPk(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace

	stored, _ := seedPortal(t, h, workspace.ID, "revoke-max-pk")
	expiresAt := h.Clock.Now().Add(time.Hour)
	h.CreatePortalSessionInState(t, stored.ID, workspace.ID, "user_1", true, expiresAt)

	maxPk, err := db.Query.MaxPortalSessionPkByExternalID(ctx, h.DB.RW(), db.MaxPortalSessionPkByExternalIDParams{
		WorkspaceID: workspace.ID,
		PortalID:    stored.ID,
		ExternalID:  "user_1",
	})
	require.NoError(t, err)
	require.Positive(t, maxPk)

	h.CreatePortalSessionInState(t, stored.ID, workspace.ID, "user_1", true, expiresAt)

	now := h.Clock.Now().UnixMilli()
	locked, err := db.Query.LockLivePortalSessionsByExternalID(ctx, h.DB.RW(), db.LockLivePortalSessionsByExternalIDParams{
		WorkspaceID:              workspace.ID,
		PortalID:                 stored.ID,
		ExternalID:               "user_1",
		MaxPk:                    uint64(maxPk),
		AccessTokenExpiresAfter:  sql.NullInt64{Valid: true, Int64: now},
		ExchangeCodeExpiresAfter: now,
		Limit:                    10,
	})
	require.NoError(t, err)
	require.Len(t, locked, 1, "the session created after the mark is not taken")
	require.Equal(t, uint64(maxPk), locked[0].Pk)
}
