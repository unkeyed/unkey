package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	portalservice "github.com/unkeyed/unkey/internal/services/portal"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

const permission = "portal.*.create_portal_session"

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

// grantedKeyspaceID is the keyspace every seeded session is scoped to, so a
// response can be checked for leaking it.
var grantedKeyspaceID = uid.New(uid.KeySpacePrefix)

// session describes one portal_sessions row to seed. A nil tokenExpiresAt leaves
// the session pending; otherwise it was exchanged and expires then. A non-nil
// rawScopes is stored verbatim in place of grant.
type session struct {
	externalID     string
	codeExpiresAt  time.Time
	tokenExpiresAt *time.Time
	revoked        bool
	grant          portalservice.Grant
	rawScopes      []byte
}

// pending is an unopened session whose portal URL is still valid.
func pending(h *testutil.Harness, externalID string) session {
	return session{
		externalID:     externalID,
		codeExpiresAt:  h.Clock.Now().Add(15 * time.Minute),
		tokenExpiresAt: nil,
		revoked:        false,
		grant:          portalservice.Grant{KeyspaceIDs: []string{grantedKeyspaceID}, Scopes: []string{"keys:read"}},
		rawScopes:      nil,
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
		grant:          portalservice.Grant{KeyspaceIDs: []string{grantedKeyspaceID}, Scopes: []string{"keys:read", "keys:reroll"}},
		rawScopes:      nil,
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

	scopes := s.rawScopes
	if scopes == nil {
		var err error
		scopes, err = json.Marshal(s.grant)
		require.NoError(t, err)
	}

	err := db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    id,
		WorkspaceID:           workspaceID,
		PortalID:              portalID,
		ExternalID:            s.externalID,
		Scopes:                scopes,
		ExchangeCodeHash:      hash.Sha256(exchangeCode),
		ExchangeCodeExpiresAt: s.codeExpiresAt.UnixMilli(),
		ReturnUrl:             sql.NullString{Valid: false, String: ""},
		CreatedAt:             now.UnixMilli(),
	})
	require.NoError(t, err)

	if s.tokenExpiresAt != nil {
		accessToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()
		res, err := db.Query.ExchangePortalSessionCode(ctx, h.DB.RW(), db.ExchangePortalSessionCodeParams{
			AccessTokenHash:      sql.NullString{String: hash.Sha256(accessToken), Valid: true},
			AccessTokenCreatedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			AccessTokenExpiresAt: sql.NullInt64{Int64: s.tokenExpiresAt.UnixMilli(), Valid: true},
			ExchangeCodeHash:     hash.Sha256(exchangeCode),
			Now:                  s.codeExpiresAt.Add(-time.Millisecond).UnixMilli(),
		})
		require.NoError(t, err)
		exchanged, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), exchanged)
	}

	if s.revoked {
		revoked, err := db.Query.RevokePortalSessionsByIDs(ctx, h.DB.RW(), db.RevokePortalSessionsByIDsParams{
			RevokedAt:   sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			WorkspaceID: workspaceID,
			Ids:         []string{id},
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), revoked)
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
