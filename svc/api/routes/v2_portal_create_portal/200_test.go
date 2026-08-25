package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// targetReadGrants is what pointing a portal at a resource costs beyond the
// portal action itself: the caller must be able to read what it is exposing.
// Carried by every test whose request names a mapping, so the cases below
// exercise their own subject rather than this check.
var targetReadGrants = []string{"api.*.read_api", "app.*.read_app"}

// newRoute registers the handler and returns it with the caller's headers.
// ksOf and appOf render a mapping as the flat request pair. Each returns nil
// unless the mapping names its kind, so a call site can set both fields
// unconditionally and still send exactly one id.
func ksOf(m portal.Mapping) *openapi.PortalKeyspaceId {
	if m.Type != portal.MappingTypeKeyspace {
		return nil
	}
	id := openapi.PortalKeyspaceId(m.ID)
	return &id
}

func appOf(m portal.Mapping) *openapi.PortalAppId {
	if m.Type != portal.MappingTypeApp {
		return nil
	}
	id := openapi.PortalAppId(m.ID)
	return &id
}

func newRoute(t *testing.T, h *testutil.Harness, permissions ...string) (*handler.Handler, http.Header) {
	t.Helper()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
		append(append([]string{}, permissions...), targetReadGrants...)...)
	return route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

// keyspaceMapping seeds an api in the workspace and maps to its keyspace.
func keyspaceMapping(t *testing.T, h *testutil.Harness, workspaceID string) portal.Mapping {
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
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}
}

// countPortals counts portals in a workspace. A rejected call returns no id, so
// counting rows is the only way to show it wrote nothing.
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

func TestCreatePortalWithKeyspaceMapping(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.create_portal")
	workspace := h.Resources().UserWorkspace
	mapping := keyspaceMapping(t, h, workspace.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:        "acme-portal",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.True(t, strings.HasPrefix(res.Body.Data.PortalId, "pc_"),
		"expected a pc_-prefixed id, got %q", res.Body.Data.PortalId)

	stored, err := db.Query.FindPortalByIdOrSlug(context.Background(), h.DB.RO(),
		db.FindPortalByIdOrSlugParams{Portal: res.Body.Data.PortalId, WorkspaceID: workspace.ID})
	require.NoError(t, err)
	require.Equal(t, "acme-portal", stored.Slug)
	require.Equal(t, "Acme", stored.DisplayName)
	require.True(t, stored.Enabled)
	require.Equal(t, mapping.ID, stored.KeyAuthID.String)
	require.False(t, stored.AppID.Valid, "the app column stays null for a keyspace mapping")
	require.False(t, stored.LogoUrl.Valid, "branding is absent when not supplied")
	require.False(t, stored.PrimaryColor.Valid)
}

func TestCreatePortalDefaultsToEnabled(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.create_portal")
	workspace := h.Resources().UserWorkspace
	mapping := keyspaceMapping(t, h, workspace.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:        "acme-portal",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	stored, err := db.Query.FindPortalByIdOrSlug(context.Background(), h.DB.RO(),
		db.FindPortalByIdOrSlugParams{Portal: res.Body.Data.PortalId, WorkspaceID: workspace.ID})
	require.NoError(t, err)
	require.True(t, stored.Enabled, "an omitted enabled creates the portal live")
}

func TestCreatePortalWithAppMappingAndBranding(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.create_portal")
	workspace := h.Resources().UserWorkspace

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "payments",
		Slug:             "payments",
		DeleteProtection: false,
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		Name:             "payments",
		Slug:             "payments",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:         "branded",
		DisplayName:  "Acme",
		KeyspaceId:   ksOf(portal.Mapping{Type: portal.MappingTypeApp, ID: app.ID}),
		AppId:        appOf(portal.Mapping{Type: portal.MappingTypeApp, ID: app.ID}),
		Enabled:      ptr.P(false),
		LogoUrl:      ptr.P("https://cdn.example.com/logo.svg"),
		PrimaryColor: ptr.P("#6366f1"),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	stored, err := db.Query.FindPortalByIdOrSlug(context.Background(), h.DB.RO(),
		db.FindPortalByIdOrSlugParams{Portal: res.Body.Data.PortalId, WorkspaceID: workspace.ID})
	require.NoError(t, err)
	require.Equal(t, app.ID, stored.AppID.String)
	require.False(t, stored.KeyAuthID.Valid, "the keyspace column stays null for an app mapping")
	require.False(t, stored.Enabled, "enabled:false creates the portal dormant")
	require.Equal(t, "https://cdn.example.com/logo.svg", stored.LogoUrl.String)
	require.Equal(t, "#6366f1", stored.PrimaryColor.String)
}

// The slug unique key is (workspace_id, slug), so the same slug in a different
// workspace is not a collision. Without this the pre-check could be written
// unscoped and nothing would notice.
func TestCreatePortalAllowsSameSlugInAnotherWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.create_portal")
	workspace := h.Resources().UserWorkspace

	other := h.CreateWorkspace()
	otherApi := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   other.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	h.CreatePortal(seed.CreatePortalRequest{
		WorkspaceID: other.ID,
		Slug:        "shared-slug",
		KeyAuthID:   sql.NullString{String: otherApi.KeyAuthID.String, Valid: true},
		Enabled:     true,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:        "shared-slug",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(keyspaceMapping(t, h, workspace.ID)),
		AppId:       appOf(keyspaceMapping(t, h, workspace.ID)),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusOK, res.Status,
		"a slug held by another workspace must not block this one: %s", res.RawBody)
}

func TestCreatePortalWritesOneAuditEntry(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.create_portal")
	workspace := h.Resources().UserWorkspace
	mapping := keyspaceMapping(t, h, workspace.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:        "audited",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	portalID := res.Body.Data.PortalId
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, portalID),
		"exactly one audit entry names the new portal")
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, "portal.create"))

	// The mapping decides which keyspaces every session from this portal can
	// reach, so an incident reviewer must be able to read it off the entry rather
	// than infer it from the row's later state.
	require.Equal(t, 1, countAuditEntriesMentioning(t, h, workspace.ID, mapping.ID),
		"the audit entry records the mapped resource")
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
