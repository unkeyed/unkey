package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// targetReadGrants is what re-pointing a portal costs beyond update_portal: the
// caller must be able to read the resource it is exposing. Carried by default so
// each test exercises its own subject; TestUpdatePortalRequiresPermissionOnTheRemapTarget
// withholds them deliberately.
var targetReadGrants = []string{"api.*.read_api", "app.*.read_app"}

// newRoute registers the handler and returns it with the caller's headers.
func newRoute(t *testing.T, h *testutil.Harness, permissions ...string) (*handler.Handler, http.Header) {
	t.Helper()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
		append(append([]string{}, permissions...), targetReadGrants...)...)
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

// appMapping seeds a project and app in the workspace and maps to the app.
func appMapping(t *testing.T, h *testutil.Harness, workspaceID, slug string) openapi.PortalMapping {
	t.Helper()

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspaceID,
		Name:             slug,
		Slug:             slug,
		DeleteProtection: false,
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		Name:             slug,
		Slug:             slug,
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	return openapi.PortalMapping{Id: app.ID, Type: openapi.PortalMappingTypeApp}
}

// seedPortal writes a portal carrying the given mapping and branding.
func seedPortal(
	t *testing.T,
	h *testutil.Harness,
	workspaceID, slug string,
	mapping openapi.PortalMapping,
	logoURL, primaryColor sql.NullString,
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
		LogoUrl:      logoURL,
		PrimaryColor: primaryColor,
	})
}

// fetchPortal reads a row back so a response can be checked against what was
// actually stored.
func fetchPortal(t *testing.T, h *testutil.Harness, workspaceID, portalID string) db.Portal {
	t.Helper()

	stored, err := db.Query.FindPortalByIdOrSlug(context.Background(), h.DB.RO(),
		db.FindPortalByIdOrSlugParams{Portal: portalID, WorkspaceID: workspaceID})
	require.NoError(t, err)
	return stored
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

// countPortals counts portals in a workspace, so a rejected call can be shown to
// have written nothing.
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

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func nullStringAbsent() sql.NullString {
	return sql.NullString{String: "", Valid: false}
}

func ptr[T any](v T) *T { return &v }

// unspecified is the tri-state zero: the field was not named at all.
func unspecified() nullable.Nullable[string] {
	return nullable.Nullable[string]{}
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

// baseRequest names nothing but the target, so each test names exactly the fields
// it is about and every other field is provably omitted.
func baseRequest(target string) handler.Request {
	return handler.Request{
		Portal:       target,
		Slug:         nil,
		DisplayName:  nil,
		Enabled:      nil,
		Mapping:      nil,
		LogoUrl:      unspecified(),
		PrimaryColor: unspecified(),
	}
}

// An omitted field keeps its stored value. This is the whole point of the
// `_specified` flags, and getting it wrong would silently blank a portal's
// branding on every toggle.
func TestUpdatePortalOnlyEnabled(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "acme-portal", mapping,
		nullString("https://cdn.example.com/logo.svg"), nullString("#6366f1"))

	req := baseRequest(stored.ID)
	req.Enabled = ptr(false)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.False(t, res.Body.Data.Enabled)
	require.Equal(t, "acme-portal", res.Body.Data.Slug)
	require.Equal(t, mapping, res.Body.Data.Mapping)
	require.NotNil(t, res.Body.Data.Branding)
	require.Equal(t, "https://cdn.example.com/logo.svg", res.Body.Data.Branding.LogoUrl)
	require.Equal(t, "#6366f1", res.Body.Data.Branding.PrimaryColor)

	row := fetchPortal(t, h, workspace.ID, stored.ID)
	require.False(t, row.Enabled)
	require.Equal(t, "acme-portal", row.Slug)
	require.Equal(t, mapping.Id, row.KeyAuthID.String)
	require.False(t, row.AppID.Valid)
	require.Equal(t, "https://cdn.example.com/logo.svg", row.LogoUrl.String)
	require.Equal(t, "#6366f1", row.PrimaryColor.String)
}

func TestUpdatePortalOnlySlug(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "old-slug", mapping,
		nullString("https://cdn.example.com/logo.svg"), nullStringAbsent())

	req := baseRequest(stored.ID)
	req.Slug = ptr("new-slug")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, "new-slug", res.Body.Data.Slug)
	require.True(t, res.Body.Data.Enabled, "enabled was not named and must not change")

	row := fetchPortal(t, h, workspace.ID, stored.ID)
	require.Equal(t, "new-slug", row.Slug)
	require.True(t, row.Enabled)
	require.Equal(t, "https://cdn.example.com/logo.svg", row.LogoUrl.String)
	require.Equal(t, mapping.Id, row.KeyAuthID.String)
}

// The two branding columns carry their own flag, so naming one must not disturb
// the other.
// The display name is the only field an end user reads, and it is independent of
// the slug: renaming the portal must not move its URL, and vice versa.
func TestUpdatePortalOnlyDisplayName(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "acme-portal", mapping,
		nullStringAbsent(), nullStringAbsent())

	req := baseRequest(stored.ID)
	req.DisplayName = ptr("Acme Payments")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, "Acme Payments", res.Body.Data.DisplayName)
	require.Equal(t, "acme-portal", res.Body.Data.Slug, "the slug is not the display name")

	row := fetchPortal(t, h, workspace.ID, stored.ID)
	require.Equal(t, "Acme Payments", row.DisplayName)
	require.Equal(t, "acme-portal", row.Slug)
}

func TestUpdatePortalOneBrandingFieldLeavesTheOther(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	stored := seedPortal(t, h, workspace.ID, "branded", keyspaceMapping(t, h, workspace.ID),
		nullString("https://cdn.example.com/logo.svg"), nullString("#6366f1"))

	req := baseRequest(stored.ID)
	req.PrimaryColor = nullable.NewNullableWithValue("#000000")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, "#000000", res.Body.Data.Branding.PrimaryColor)
	require.Equal(t, "https://cdn.example.com/logo.svg", res.Body.Data.Branding.LogoUrl)

	row := fetchPortal(t, h, workspace.ID, stored.ID)
	require.Equal(t, "#000000", row.PrimaryColor.String)
	require.Equal(t, "https://cdn.example.com/logo.svg", row.LogoUrl.String)
}

// An explicit null clears, an omitted field keeps. If these two collapsed
// into one, either clearing branding would be impossible or every partial update
// would wipe it.
func TestUpdatePortalDistinguishesNullFromOmitted(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	cleared := seedPortal(t, h, workspace.ID, "cleared", keyspaceMapping(t, h, workspace.ID),
		nullString("https://cdn.example.com/logo.svg"), nullString("#6366f1"))
	kept := seedPortal(t, h, workspace.ID, "kept", keyspaceMapping(t, h, workspace.ID),
		nullString("https://cdn.example.com/logo.svg"), nullString("#6366f1"))

	nullReq := baseRequest(cleared.ID)
	nullReq.LogoUrl = nullable.NewNullNullable[string]()
	nullRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, nullReq)
	require.Equal(t, http.StatusOK, nullRes.Status, "expected 200, received: %s", nullRes.RawBody)
	require.Equal(t, "", nullRes.Body.Data.Branding.LogoUrl, "an explicit null clears the logo")
	require.Equal(t, "#6366f1", nullRes.Body.Data.Branding.PrimaryColor)

	clearedRow := fetchPortal(t, h, workspace.ID, cleared.ID)
	require.False(t, clearedRow.LogoUrl.Valid, "the column is null, not an empty string")
	require.Equal(t, "#6366f1", clearedRow.PrimaryColor.String)

	// The same field omitted, against an identically seeded portal.
	omittedRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, baseRequest(kept.ID))
	require.Equal(t, http.StatusOK, omittedRes.Status, "expected 200, received: %s", omittedRes.RawBody)
	require.Equal(t, "https://cdn.example.com/logo.svg", omittedRes.Body.Data.Branding.LogoUrl)

	keptRow := fetchPortal(t, h, workspace.ID, kept.ID)
	require.True(t, keptRow.LogoUrl.Valid, "an omitted field must keep its stored value")
	require.Equal(t, "https://cdn.example.com/logo.svg", keptRow.LogoUrl.String)
}

// Clearing both branding columns drops the object from the response rather than
// returning two empty strings.
func TestUpdatePortalClearingAllBrandingOmitsTheObject(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	stored := seedPortal(t, h, workspace.ID, "unbranded", keyspaceMapping(t, h, workspace.ID),
		nullString("https://cdn.example.com/logo.svg"), nullString("#6366f1"))

	req := baseRequest(stored.ID)
	req.LogoUrl = nullable.NewNullNullable[string]()
	req.PrimaryColor = nullable.NewNullNullable[string]()

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Nil(t, res.Body.Data.Branding)

	row := fetchPortal(t, h, workspace.ID, stored.ID)
	require.False(t, row.LogoUrl.Valid)
	require.False(t, row.PrimaryColor.Valid)
}

// Naming only a keyspace on an app-mapped portal writes both
// association columns, so the row can never hold both. It also revokes the
// portal's live sessions, and only that portal's.
func TestUpdatePortalRepointsMappingAndRevokesSessions(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	app := appMapping(t, h, workspace.ID, "payments")
	stored := seedPortal(t, h, workspace.ID, "repointed", app, nullStringAbsent(), nullStringAbsent())
	keyspace := keyspaceMapping(t, h, workspace.ID)

	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{keyspace.Id}, []string{"keys.read"})
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_2", []string{keyspace.Id}, []string{"keys.read"})
	require.Equal(t, 2, liveSessions(t, h, stored.ID))

	// A different portal in the same workspace, whose sessions must survive.
	bystander := seedPortal(t, h, workspace.ID, "bystander", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())
	h.CreatePortalSessionForPortal(bystander.ID, workspace.ID, "user_3", []string{keyspace.Id}, []string{"keys.read"})

	req := baseRequest(stored.ID)
	req.Mapping = &keyspace

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, keyspace, res.Body.Data.Mapping)

	row := fetchPortal(t, h, workspace.ID, stored.ID)
	require.Equal(t, keyspace.Id, row.KeyAuthID.String)
	require.False(t, row.AppID.Valid, "the app column must be cleared in the same write")

	require.Equal(t, 0, liveSessions(t, h, stored.ID),
		"re-pointing the mapping revokes the portal's sessions")
	require.Equal(t, 1, liveSessions(t, h, bystander.ID),
		"another portal's sessions must be untouched")
}

// Revocation is tied to the mapping changing, not to the request touching
// the row. Disabling a portal in particular must not cut live sessions.
func TestUpdatePortalWithoutMappingChangeKeepsSessions(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "steady", mapping, nullStringAbsent(), nullStringAbsent())

	testCases := map[string]func(handler.Request) handler.Request{
		"disable only": func(r handler.Request) handler.Request {
			r.Enabled = ptr(false)
			return r
		},
		"slug only": func(r handler.Request) handler.Request {
			r.Slug = ptr("steady-renamed")
			return r
		},
		// Re-sending the mapping it already has is not a change, so it must not
		// revoke either.
		"same mapping resent": func(r handler.Request) handler.Request {
			r.Mapping = &mapping
			return r
		},
	}

	for name, mutate := range testCases {
		t.Run(name, func(t *testing.T) {
			h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_"+name, []string{mapping.Id}, []string{"keys.read"})
			before := liveSessions(t, h, stored.ID)
			require.Positive(t, before, "the fixture must have a live session to lose")

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, mutate(baseRequest(stored.ID)))
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, before, liveSessions(t, h, stored.ID),
				"sessions must survive an update that leaves the mapping alone")
		})
	}
}

// The target is an id or a slug, and both must reach the same row.
func TestUpdatePortalAddressedBySlug(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	stored := seedPortal(t, h, workspace.ID, "by-slug", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())
	// A sibling, so addressing by slug can be shown to pick the right row rather
	// than the only row.
	sibling := seedPortal(t, h, workspace.ID, "sibling", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())

	req := baseRequest(stored.Slug)
	req.Enabled = ptr(false)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, stored.ID, res.Body.Data.Id)

	require.False(t, fetchPortal(t, h, workspace.ID, stored.ID).Enabled)
	require.True(t, fetchPortal(t, h, workspace.ID, sibling.ID).Enabled,
		"the sibling must be untouched")
}

// One entry, carrying enough of the before and after state that an incident
// reviewer does not have to infer the previous mapping from a row that no longer
// holds it.
func TestUpdatePortalWritesOneAuditEntry(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	app := appMapping(t, h, workspace.ID, "audited")
	stored := seedPortal(t, h, workspace.ID, "audited-portal", app, nullStringAbsent(), nullStringAbsent())
	keyspace := keyspaceMapping(t, h, workspace.ID)
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{keyspace.Id}, []string{"keys.read"})

	req := baseRequest(stored.ID)
	req.Mapping = &keyspace
	req.Slug = ptr("audited-renamed")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, "portal.update"))
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, stored.ID))
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, app.Id),
		"the previous mapping is recorded, not just the new one")
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, keyspace.Id))
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, "audited-portal"))
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, "sessionsRevoked"),
		"the revoked-session count is part of the record")
}
