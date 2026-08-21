package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
	listKeys "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_keys"
)

// newRoute registers the handler and returns it with the caller's headers.
func newRoute(t *testing.T, h *testutil.Harness, permissions ...string) (*handler.Handler, http.Header) {
	t.Helper()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, permissions...)
	return route, headersFor(rootKey)
}

func headersFor(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

// keyspaceMapping seeds an api in the workspace and maps to its keyspace.
func keyspaceMapping(t *testing.T, h *testutil.Harness, workspaceID string) openapi.PortalMapping {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspaceID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	return openapi.PortalMapping{Id: api.KeyAuthID.String, Type: openapi.PortalMappingTypeKeyspace}
}

// seedPortal writes a portal carrying the given mapping.
func seedPortal(
	t *testing.T,
	h *testutil.Harness,
	workspaceID, slug string,
	mapping openapi.PortalMapping,
) db.Portal {
	t.Helper()

	appID := nullStringAbsent()
	keyAuthID := nullStringAbsent()
	switch mapping.Type {
	case openapi.PortalMappingTypeApp:
		appID = nullString(mapping.Id)
	case openapi.PortalMappingTypeKeyspace:
		keyAuthID = nullString(mapping.Id)
	default:
		t.Fatalf("unsupported mapping type %q", mapping.Type)
	}

	return h.CreatePortal(seed.CreatePortalRequest{
		ID:           "",
		WorkspaceID:  workspaceID,
		Slug:         slug,
		AppID:        appID,
		KeyAuthID:    keyAuthID,
		Enabled:      true,
		LogoUrl:      nullStringAbsent(),
		PrimaryColor: nullStringAbsent(),
	})
}

// portalExists reports whether a row is still there, which is the only thing a
// delete can be judged on.
func portalExists(t *testing.T, h *testutil.Harness, workspaceID, target string) bool {
	t.Helper()

	_, err := db.Query.FindPortalByIdOrSlug(context.Background(), h.DB.RO(),
		db.FindPortalByIdOrSlugParams{Portal: target, WorkspaceID: workspaceID})
	if err == nil {
		return true
	}
	require.True(t, db.IsNotFound(err), "unexpected error reading portal: %v", err)
	return false
}

// liveSessions counts a portal's unrevoked sessions. Revocation is asserted
// against the column rather than through the session resolver, whose cache would
// otherwise decide the result.
func liveSessions(t *testing.T, h *testutil.Harness, portalID string) int {
	t.Helper()

	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM portal_sessions WHERE portal_id = ? AND revoked_at IS NULL", portalID,
	).Scan(&count))
	return count
}

// sessionsFor counts a portal's session rows matching an extra predicate.
//
// Scoped on the portal id because the test database is shared between runs: an
// external id or a slug can appear in rows this test never wrote, while a minted
// portal id cannot.
func sessionsFor(t *testing.T, h *testutil.Harness, portalID, predicate string) int {
	t.Helper()

	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM portal_sessions WHERE portal_id = ? AND "+predicate, portalID,
	).Scan(&count))
	return count
}

// countPortals counts portals in a workspace, so a rejected call can be shown to
// have deleted nothing.
func countPortals(t *testing.T, h *testutil.Harness, workspaceID string) int {
	t.Helper()

	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM portals WHERE workspace_id = ?", workspaceID,
	).Scan(&count))
	return count
}

// countAuditEntriesMentioning counts outbox audit payloads referencing a string.
func countAuditEntriesMentioning(t *testing.T, h *testutil.Harness, workspaceID, needle string) int {
	t.Helper()

	rows, err := db.Query.ListClickhouseOutboxByWorkspace(context.Background(), h.DB.RO(), workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		if strings.Contains(string(row.Payload), needle) {
			count++
		}
	}
	return count
}

// deleteAuditMeta reads the meta object off the single portal.delete audit entry.
//
// Parsed rather than substring-matched: the payload is a MySQL JSON column, so
// its whitespace and key order are the database's to choose, and a needle that
// depended on either would pass or fail for reasons unrelated to the handler.
func deleteAuditMeta(t *testing.T, h *testutil.Harness, workspaceID string) map[string]any {
	t.Helper()

	rows, err := db.Query.ListClickhouseOutboxByWorkspace(context.Background(), h.DB.RO(), workspaceID)
	require.NoError(t, err)

	var metas []map[string]any
	for _, row := range rows {
		var payload struct {
			Event   string `json:"event"`
			Targets []struct {
				Meta map[string]any `json:"meta"`
			} `json:"targets"`
		}
		require.NoError(t, json.Unmarshal(row.Payload, &payload))
		if payload.Event != string(auditlog.PortalDeleteEvent) {
			continue
		}
		require.Len(t, payload.Targets, 1, "a delete names exactly one portal target")
		metas = append(metas, payload.Targets[0].Meta)
	}

	require.Len(t, metas, 1, "exactly one portal.delete entry")
	return metas[0]
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func nullStringAbsent() sql.NullString {
	return sql.NullString{String: "", Valid: false}
}

// normalizeRequestID strips the per-request id so two error bodies can be
// compared for the parity the masking depends on.
func normalizeRequestID(body string) string {
	const key = `"requestId":"`
	start := strings.Index(body, key)
	if start == -1 {
		return body
	}
	rest := body[start+len(key):]
	end := strings.Index(rest, `"`)
	if end == -1 {
		return body
	}
	return body[:start+len(key)] + "REQ" + rest[end:]
}

func request(target string) handler.Request {
	return handler.Request{Portal: target}
}

// The row goes away and the payload is empty. The empty data object is the
// contract, so an accidental entity body would be a breaking change the moment
// callers relied on it.
func TestDeletePortalByID(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	stored := seedPortal(t, h, workspace.ID, "acme-portal", keyspaceMapping(t, h, workspace.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Empty(t, res.Body.Data, "the data payload is empty by design")

	require.False(t, portalExists(t, h, workspace.ID, stored.ID), "the row must be gone")
	require.Equal(t, 0, countPortals(t, h, workspace.ID))
}

// The target is an id or a slug, and a slug must reach its own row
// rather than the only row.
func TestDeletePortalBySlug(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	stored := seedPortal(t, h, workspace.ID, "by-slug", keyspaceMapping(t, h, workspace.ID))
	sibling := seedPortal(t, h, workspace.ID, "sibling", keyspaceMapping(t, h, workspace.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.Slug))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.False(t, portalExists(t, h, workspace.ID, stored.ID))
	require.True(t, portalExists(t, h, workspace.ID, sibling.ID), "the sibling must survive")
}

// Unlike update, revocation here is unconditional. A deleted
// portal has no state left for a live session to be consistent with, and the
// session resolver never reads `portals`, so without this every end user would
// keep their access until the token expired.
func TestDeletePortalRevokesItsSessions(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "revoked", mapping)

	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.Id}, []string{"keys:read"})
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_2", []string{mapping.Id}, []string{"keys:read"})
	// Guards the fixture: without a live session to lose, the assertion below
	// would pass against a handler that revokes nothing.
	require.Equal(t, 2, liveSessions(t, h, stored.ID), "the fixture must have live sessions to lose")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Equal(t, 0, liveSessions(t, h, stored.ID), "deleting a portal revokes its sessions")

	var revokedAt sql.NullInt64
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT revoked_at FROM portal_sessions WHERE portal_id = ? AND external_id = ?",
		stored.ID, "user_1",
	).Scan(&revokedAt))
	require.True(t, revokedAt.Valid, "revoked_at is the durable record of the revocation")
	require.Positive(t, revokedAt.Int64)
}

// The revocation is scoped to the portal being deleted. A workspace-wide UPDATE
// would cut every end user of every portal the workspace runs.
func TestDeletePortalLeavesOtherPortalsSessionsAlone(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	doomed := seedPortal(t, h, workspace.ID, "doomed", mapping)
	bystander := seedPortal(t, h, workspace.ID, "bystander", keyspaceMapping(t, h, workspace.ID))

	h.CreatePortalSessionForPortal(doomed.ID, workspace.ID, "user_1", []string{mapping.Id}, []string{"keys:read"})
	h.CreatePortalSessionForPortal(bystander.ID, workspace.ID, "user_2", []string{mapping.Id}, []string{"keys:read"})
	require.Equal(t, 1, liveSessions(t, h, doomed.ID))
	require.Equal(t, 1, liveSessions(t, h, bystander.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(doomed.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Equal(t, 0, liveSessions(t, h, doomed.ID))
	require.Equal(t, 1, liveSessions(t, h, bystander.ID),
		"another portal's sessions must be untouched")
	require.True(t, portalExists(t, h, workspace.ID, bystander.ID))
}

// The keyspace the portal served is not the portal. Deleting the portal must not
// cascade to the resource it pointed at, which other portals and the whole key
// API depend on.
func TestDeletePortalDoesNotCascadeToItsMapping(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "mapped", mapping)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	rows, err := db.Query.FindKeyAuthsByIdsAndWorkspace(context.Background(), h.DB.RO(),
		db.FindKeyAuthsByIdsAndWorkspaceParams{WorkspaceID: workspace.ID, KeyAuthIds: []string{mapping.Id}})
	require.NoError(t, err)
	require.Len(t, rows, 1, "the keyspace the portal served must survive")
}

// A session names the portal it was minted from, and that id is not reused. So
// recreating the same slug against the same mapping produces a different portal,
// and the old session belongs to neither it nor anything else.
func TestDeletePortalThenRecreateSameSlug(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	original := seedPortal(t, h, workspace.ID, "recycled", mapping)
	h.CreatePortalSessionForPortal(original.ID, workspace.ID, "user_1", []string{mapping.Id}, []string{"keys:read"})
	require.Equal(t, 1, liveSessions(t, h, original.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(original.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	// Seeded rather than created through the API: this test is about what happens
	// to the old session, and the seeder keeps the replacement independent of the
	// create route's own behaviour.
	replacement := seedPortal(t, h, workspace.ID, "recycled", mapping)
	require.NotEqual(t, original.ID, replacement.ID, "the id is not reused")

	// Queried by portal id, not by external id: the test database is shared, so an
	// external id is not unique to this test while a minted portal id is.
	require.Equal(t, 1, sessionsFor(t, h, original.ID, "revoked_at IS NOT NULL"),
		"the old session still names the deleted portal, and stays revoked")
	require.Equal(t, 0, sessionsFor(t, h, replacement.ID, "1 = 1"),
		"the replacement inherits no sessions")
	require.Equal(t, 0, liveSessions(t, h, replacement.ID))
}

// The durable revocation has to actually stop an end user. The portal route
// is registered alongside the delete route so the same session token can be seen
// working, then failing. The clock is advanced past the session cache's stale
// window because revocation is bounded by that cache, not instant: within the
// fresh window the cached row is served and the end user keeps working, which is
// the documented behaviour rather than a bug.
func TestDeletePortalStopsTheEndUserOnceTheCacheTurnsOver(t *testing.T) {
	h := testutil.NewHarness(t)
	deleteRoute, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	endUserRoute := listKeys.New(h.DB)
	h.Register(endUserRoute, h.PortalMiddleware()...)

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "live", mapping)
	sessionHeaders := h.CreatePortalSessionForPortal(
		stored.ID, workspace.ID, "portal_user_A", []string{mapping.Id}, []string{"keys:read"})

	warm := testutil.CallRoute[listKeys.Request, listKeys.Response](h, endUserRoute, sessionHeaders, listKeys.Request{})
	require.Equal(t, http.StatusOK, warm.Status,
		"the session must work before the delete, received: %s", warm.RawBody)

	res := testutil.CallRoute[handler.Request, handler.Response](h, deleteRoute, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	// Past the 5 minute stale window, so the resolver refetches rather than
	// serving the cached pre-revocation row.
	h.Clock.Tick(6 * time.Minute)

	rejected := testutil.CallRoute[listKeys.Request, openapi.UnauthorizedErrorResponse](
		h, endUserRoute, sessionHeaders, listKeys.Request{})
	require.Equal(t, http.StatusUnauthorized, rejected.Status,
		"a revoked session must be rejected once the cache turns over, received: %s", rejected.RawBody)
	require.Equal(t, "The portal session is invalid or has expired.", rejected.Body.Error.Detail)
	require.NotContains(t, rejected.RawBody, stored.ID,
		"the rejection must not name the deleted portal")
}

// One entry, carrying the whole deleted state. The row is gone, so nothing
// else can answer which resource the portal served or how many end users lost
// access.
func TestDeletePortalWritesOneAuditEntry(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "audited-portal", mapping)
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.Id}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, "portal.delete"))
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, stored.ID))

	meta := deleteAuditMeta(t, h, workspace.ID)
	require.Equal(t, "audited-portal", meta["slug"])
	require.Equal(t, string(openapi.PortalMappingTypeKeyspace), meta["mappingType"])
	require.Equal(t, mapping.Id, meta["mappingId"],
		"the mapping is recorded, because the row that held it is gone")
	require.Equal(t, true, meta["enabled"])
	require.Equal(t, float64(1), meta["sessionsRevoked"],
		"the revoked-session count is part of the record")
}

// A misconfigured row -- both mapping columns set, which predates these routes --
// must still be deletable. Refusing would leave the invariant violation with no
// way out.
func TestDeletePortalWithInvalidMapping(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	keyspace := keyspaceMapping(t, h, workspace.ID)
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "both",
		Slug:             "both",
		DeleteProtection: false,
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		Name:             "both",
		Slug:             "both",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	stored := h.CreatePortal(seed.CreatePortalRequest{
		ID:           "",
		WorkspaceID:  workspace.ID,
		Slug:         "broken",
		AppID:        nullString(app.ID),
		KeyAuthID:    nullString(keyspace.Id),
		Enabled:      true,
		LogoUrl:      nullStringAbsent(),
		PrimaryColor: nullStringAbsent(),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.False(t, portalExists(t, h, workspace.ID, stored.ID))
	require.Equal(t, "invalid", deleteAuditMeta(t, h, workspace.ID)["mappingType"],
		"the audit entry records that the mapping could not be read")
}
