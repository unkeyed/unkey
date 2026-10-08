package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

// registerRoute registers the handler with the cache the portal authenticator
// reads.
func registerRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{
		DB:           h.DB,
		Auditlogs:    h.Auditlogs,
		Clock:        h.Clock,
		SessionCache: h.Caches.PortalSession,
	}
	h.Register(route)
	return route
}

// newRoute registers the handler and returns it with a root key's headers.
func newRoute(t *testing.T, h *testutil.Harness, permissions ...string) (*handler.Handler, http.Header) {
	t.Helper()

	route := registerRoute(h)
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, permissions...)
	return route, testutil.RootKeyHeaders(rootKey)
}

func request(target, externalID string) handler.Request {
	return handler.Request{Portal: target, ExternalId: externalID}
}

// seedPortal seeds a keyspace-backed portal with the given slug.
func seedPortal(t *testing.T, h *testutil.Harness, workspaceID, slug string) (db.Portal, portal.Mapping) {
	t.Helper()

	mapping, _ := h.SeedKeyspaceMapping(t, workspaceID)
	return h.SeedPortal(t, workspaceID, slug, slug, mapping, nil, nil), mapping
}

// sessionByCookie reads the session row behind headers returned by
// [testutil.Harness.CreatePortalSessionForPortal].
func sessionByCookie(t *testing.T, h *testutil.Harness, headers http.Header) db.PortalSession {
	t.Helper()

	accessToken, ok := strings.CutPrefix(headers.Get("Cookie"), "portal_session=")
	require.True(t, ok, "the headers must carry a portal_session cookie")
	session, err := db.Query.FindPortalSessionByAccessTokenHash(context.Background(), h.DB.RW(),
		sql.NullString{String: hash.Sha256(accessToken), Valid: true})
	require.NoError(t, err)
	return session
}

// revokeAuditMetas returns the portal target's meta from every
// portal.session.revoke entry naming the portal.
func revokeAuditMetas(t *testing.T, h *testutil.Harness, portalID string) []map[string]any {
	t.Helper()

	var metas []map[string]any
	for _, ev := range h.FindAuditLogsByTargetID(context.Background(), t, portalID) {
		if ev.Event != string(auditlog.PortalSessionRevokeEvent) {
			continue
		}
		require.Len(t, ev.Targets, 1, "a revoke names exactly one portal target")
		metas = append(metas, ev.Targets[0].Meta)
	}
	return metas
}
