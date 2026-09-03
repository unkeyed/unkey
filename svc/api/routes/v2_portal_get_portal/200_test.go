package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
)

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

	route := &handler.Handler{DB: h.DB}
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
// requireServes asserts the response names exactly the given resource, and that
// the other id is absent rather than empty.
func requireServes(t *testing.T, want portal.Mapping, got openapi.Portal) {
	t.Helper()

	switch want.Type {
	case portal.MappingTypeKeyspace:
		require.NotNil(t, got.KeyspaceId, "the keyspace id must be present")
		require.Equal(t, want.ID, string(*got.KeyspaceId))
		require.Nil(t, got.AppId, "the app id must be absent for a keyspace portal")
	case portal.MappingTypeApp:
		require.NotNil(t, got.AppId, "the app id must be present")
		require.Equal(t, want.ID, string(*got.AppId))
		require.Nil(t, got.KeyspaceId, "the keyspace id must be absent for an app portal")
	default:
		t.Fatalf("unsupported mapping type %q", want.Type)
	}
}

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

// appMapping seeds a project and app in the workspace and maps to the app.
func appMapping(t *testing.T, h *testutil.Harness, workspaceID, slug string) portal.Mapping {
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
	return portal.Mapping{Type: portal.MappingTypeApp, ID: app.ID}
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

func TestGetPortalByIdAndSlug(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.read_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := h.SeedPortal(t, workspace.ID, "acme-portal", "acme-portal", mapping,
		ptr.P("https://cdn.example.com/logo.svg"), ptr.P("#6366f1"))

	for name, target := range map[string]string{
		"by id":   stored.ID,
		"by slug": stored.Slug,
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Portal:     ptr.P(target),
				KeyspaceId: nil,
				AppId:      nil,
			})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.NotNil(t, res.Body)
			require.Equal(t, stored.ID, res.Body.Data.Id)
			require.Equal(t, "acme-portal", res.Body.Data.Slug)
			require.True(t, res.Body.Data.Enabled)
			requireServes(t, mapping, res.Body.Data)
			require.NotNil(t, res.Body.Data.Branding)
			require.Equal(t, "https://cdn.example.com/logo.svg", res.Body.Data.Branding.LogoUrl)
			require.Equal(t, "#6366f1", res.Body.Data.Branding.PrimaryColor)
		})
	}
}

// TestGetPortalWithMissingMappingTarget guarantees a legacy permission can read
// a portal after its mapping target is removed.
func TestGetPortalWithMissingMappingTarget(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.read_portal")
	workspace := h.Resources().UserWorkspace

	mapping := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: uid.New(uid.KeySpacePrefix)}
	projectID := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID}).ProjectID
	stored := h.CreatePortal(seed.CreatePortalRequest{
		WorkspaceID: workspace.ID,
		ProjectID:   projectID,
		Slug:        "orphan",
		DisplayName: "orphan",
		KeyAuthID:   sql.NullString{String: mapping.ID, Valid: true},
		Enabled:     true,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal: ptr.P(stored.ID),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, stored.ID, res.Body.Data.Id)
	requireServes(t, mapping, res.Body.Data)
}

// The dashboard reaches a portal through the app or keyspace it serves and never
// holds a portal id, so this arm is the one that unblocks it.
func TestGetPortalByMapping(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.read_portal")
	workspace := h.Resources().UserWorkspace

	keyspace := keyspaceMapping(t, h, workspace.ID)
	keyspacePortal := h.SeedPortal(t, workspace.ID, "keyspace-portal", "keyspace-portal", keyspace,
		nil, nil)

	app := appMapping(t, h, workspace.ID, "payments")
	appPortal := h.SeedPortal(t, workspace.ID, "app-portal", "app-portal", app,
		nil, nil)

	for name, tc := range map[string]struct {
		mapping  portal.Mapping
		expectID string
	}{
		"keyspace mapping": {mapping: keyspace, expectID: keyspacePortal.ID},
		"app mapping":      {mapping: app, expectID: appPortal.ID},
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Portal:     nil,
				KeyspaceId: ksOf(tc.mapping),
				AppId:      appOf(tc.mapping),
			})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, tc.expectID, res.Body.Data.Id)
			requireServes(t, tc.mapping, res.Body.Data)
		})
	}
}

// Branding is omitted rather than returned as two empty strings, so a client can
// tell "no branding set" from "branding set to empty".
func TestGetPortalOmitsAbsentBranding(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.read_portal")
	workspace := h.Resources().UserWorkspace

	stored := h.SeedPortal(t, workspace.ID, "plain", "plain", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     ptr.P(stored.ID),
		KeyspaceId: nil,
		AppId:      nil,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Nil(t, res.Body.Data.Branding, "branding must be absent, not empty strings")
	require.NotContains(t, res.RawBody, `"branding"`,
		"an omitempty branding object must not appear in the wire body")
}

// The portal carries its own display name, while a return URL belongs to a
// session rather than to a portal.
func TestGetPortalCarriesDisplayNameButNoReturnURL(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.read_portal")
	workspace := h.Resources().UserWorkspace

	stored := h.SeedPortal(t, workspace.ID, "no-extras", "no-extras", keyspaceMapping(t, h, workspace.ID),
		ptr.P("https://cdn.example.com/logo.svg"), nil)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     ptr.P(stored.ID),
		KeyspaceId: nil,
		AppId:      nil,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Equal(t, stored.DisplayName, res.Body.Data.DisplayName)

	for _, forbidden := range []string{"returnUrl", "return_url"} {
		require.NotContains(t, res.RawBody, forbidden,
			"the portal response must not carry %q", forbidden)
	}
}
