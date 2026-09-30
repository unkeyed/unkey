package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

const permission = "portal.*.read_portal"

func registerRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{
		DB:    h.DB,
		Clock: h.Clock,
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

func request(target string) handler.Request {
	return handler.Request{Portal: target, Limit: nil, Cursor: nil, Search: nil}
}

// keyspaceMapping seeds an api and returns its keyspace mapping and project.
func keyspaceMapping(t *testing.T, h *testutil.Harness, workspaceID string) (portal.Mapping, string) {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}, api.ProjectID
}

// seedPortal seeds a keyspace-backed portal with the given slug.
func seedPortal(t *testing.T, h *testutil.Harness, workspaceID, slug string) db.Portal {
	t.Helper()

	mapping, _ := keyspaceMapping(t, h, workspaceID)
	return h.SeedPortal(t, workspaceID, slug, slug, mapping, nil, nil)
}

// session describes one portal_sessions row to seed. A nil tokenExpiresAt leaves
// the session pending; otherwise it was exchanged and expires then.
type session struct {
	externalID     string
	codeExpiresAt  time.Time
	tokenExpiresAt *time.Time
	revoked        bool
	scopes         string
}

// pending is an unopened session whose portal URL is still valid.
func pending(h *testutil.Harness, externalID string) session {
	return session{
		externalID:     externalID,
		codeExpiresAt:  h.Clock.Now().Add(15 * time.Minute),
		tokenExpiresAt: nil,
		revoked:        false,
		scopes:         `{"keyspaceIds":["ks_1"],"scopes":["keys:read"]}`,
	}
}

// active is an exchanged session whose access token is still valid. Its code has
// already expired, so a handler reading the code expiry would be caught.
func active(h *testutil.Harness, externalID string) session {
	return session{
		externalID:     externalID,
		codeExpiresAt:  h.Clock.Now().Add(-time.Minute),
		tokenExpiresAt: new(h.Clock.Now().Add(24 * time.Hour)),
		revoked:        false,
		scopes:         `{"keyspaceIds":["ks_1"],"scopes":["keys:read","keys:reroll"]}`,
	}
}

// insertSession writes s and returns its session id, created at the harness
// clock's current time.
func insertSession(t *testing.T, h *testutil.Harness, portalID, workspaceID string, s session) string {
	t.Helper()

	id := uid.New(uid.PortalSessionPrefix)
	exchangeCode := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
	now := h.Clock.Now()
	ctx := context.Background()

	err := db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    id,
		WorkspaceID:           workspaceID,
		PortalID:              portalID,
		ExternalID:            s.externalID,
		Scopes:                []byte(s.scopes),
		ExchangeCodeHash:      hash.Sha256(exchangeCode),
		ExchangeCodeExpiresAt: s.codeExpiresAt.UnixMilli(),
		ReturnUrl:             sql.NullString{Valid: false, String: ""},
		CreatedAt:             now.UnixMilli(),
	})
	require.NoError(t, err)

	if s.tokenExpiresAt != nil {
		accessToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()
		_, err = h.DB.RW().ExecContext(ctx,
			"UPDATE portal_sessions SET access_token_hash = ?, access_token_created_at = ?, access_token_expires_at = ? WHERE id = ?",
			hash.Sha256(accessToken), now.UnixMilli(), s.tokenExpiresAt.UnixMilli(), id,
		)
		require.NoError(t, err)
	}

	if s.revoked {
		_, err = h.DB.RW().ExecContext(ctx, "UPDATE portal_sessions SET revoked_at = ? WHERE id = ?", now.UnixMilli(), id)
		require.NoError(t, err)
	}

	return id
}

// externalIDs returns the end users a response lists, in order.
func externalIDs(res *handler.Response) []string {
	ids := make([]string, 0, len(res.Data))
	for _, group := range res.Data {
		ids = append(ids, group.ExternalId)
	}
	return ids
}
