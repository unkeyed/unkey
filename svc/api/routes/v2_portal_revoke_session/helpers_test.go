package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
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
	return route, headersFor(rootKey)
}

func headersFor(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

func request(target, externalID string) handler.Request {
	return handler.Request{Portal: target, ExternalId: externalID}
}

// keyspaceMapping seeds an api and returns its keyspace mapping and project.
func keyspaceMapping(t *testing.T, h *testutil.Harness, workspaceID string) (portal.Mapping, string) {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}, api.ProjectID
}

// seedPortal seeds a keyspace-backed portal with the given slug.
func seedPortal(t *testing.T, h *testutil.Harness, workspaceID, slug string) (db.Portal, portal.Mapping) {
	t.Helper()

	mapping, _ := keyspaceMapping(t, h, workspaceID)
	return h.SeedPortal(t, workspaceID, slug, slug, mapping, nil, nil), mapping
}

// insertSession writes a pending or exchanged session with the given expiry and
// returns its exchange code. The harness helper only makes active sessions.
func insertSession(t *testing.T, h *testutil.Harness, portalID, workspaceID, externalID string, exchanged bool, expiresAt time.Time) string {
	t.Helper()

	exchangeCode := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
	now := h.Clock.Now()
	ctx := context.Background()

	err := db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspaceID,
		PortalID:              portalID,
		ExternalID:            externalID,
		Scopes:                []byte(`{"keyspaceIds":[],"scopes":["keys:read"]}`),
		ExchangeCodeHash:      hash.Sha256(exchangeCode),
		ExchangeCodeExpiresAt: expiresAt.UnixMilli(),
		ReturnUrl:             sql.NullString{Valid: false, String: ""},
		CreatedAt:             now.UnixMilli(),
	})
	require.NoError(t, err)

	if exchanged {
		accessToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()
		_, err = h.DB.RW().ExecContext(ctx,
			"UPDATE portal_sessions SET access_token_hash = ?, access_token_created_at = ?, access_token_expires_at = ? WHERE exchange_code_hash = ?",
			hash.Sha256(accessToken), now.UnixMilli(), expiresAt.UnixMilli(), hash.Sha256(exchangeCode),
		)
		require.NoError(t, err)
	}

	return exchangeCode
}

// sessionsFor counts an end user's session rows on a portal matching predicate.
// Scoped by portal id because external ids repeat across tests.
func sessionsFor(t *testing.T, h *testutil.Harness, portalID, externalID, predicate string) int {
	t.Helper()

	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM portal_sessions WHERE portal_id = ? AND external_id = ? AND "+predicate,
		portalID, externalID,
	).Scan(&count))
	return count
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
