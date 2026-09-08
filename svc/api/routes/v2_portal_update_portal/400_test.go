package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// Only what the request names is validated, so each case here names exactly one
// bad field. Note what is absent: there is no case for "both an app and a
// keyspace" or "neither", because the mapping names its kind and id together and
// those states cannot be expressed.
func TestUpdatePortalRejectsInvalidInput(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := h.SeedPortal(t, workspace.ID, "valid-portal", "valid-portal", mapping,
		ptr.P("https://cdn.example.com/logo.svg"), ptr.P("#6366f1"))

	blankKeyspaceID := openapi.PortalKeyspaceId("")
	someAppID := openapi.PortalAppId("app_1234abcd")

	testCases := []struct {
		name   string
		mutate func(handler.Request) handler.Request
	}{
		{name: "slug too short", mutate: func(r handler.Request) handler.Request { r.Slug = ptr.P("ab"); return r }},
		{name: "slug uppercase", mutate: func(r handler.Request) handler.Request { r.Slug = ptr.P("Acme-Portal"); return r }},
		{name: "slug consecutive hyphens", mutate: func(r handler.Request) handler.Request { r.Slug = ptr.P("acme--portal"); return r }},
		{name: "slug leading hyphen", mutate: func(r handler.Request) handler.Request { r.Slug = ptr.P("-acme"); return r }},
		{name: "slug too long", mutate: func(r handler.Request) handler.Request { r.Slug = ptr.P(strings.Repeat("a", 65)); return r }},
		{
			// The id-or-slug resolver reads one argument as either form, so an
			// id-shaped slug would make resolution ambiguous.
			name:   "id-shaped slug",
			mutate: func(r handler.Request) handler.Request { r.Slug = ptr.P("pc_1234abcd"); return r },
		},
		{
			name: "blank keyspace id",
			mutate: func(r handler.Request) handler.Request {
				r.KeyspaceId = &blankKeyspaceID
				return r
			},
		},
		{
			// The flat pair can name two resources, which the nested object could
			// not. A portal serves one, so this is a contradiction rather than a
			// merge.
			name: "both keyspace and app id",
			mutate: func(r handler.Request) handler.Request {
				r.KeyspaceId = ksOf(mapping)
				r.AppId = &someAppID
				return r
			},
		},
		{
			name: "logo url with http scheme",
			mutate: func(r handler.Request) handler.Request {
				r.LogoUrl = nullable.NewNullableWithValue("http://cdn.example.com/logo.svg")
				return r
			},
		},
		{
			name: "logo url not a url",
			mutate: func(r handler.Request) handler.Request {
				r.LogoUrl = nullable.NewNullableWithValue("not a url at all")
				return r
			},
		},
		{
			// The column is varchar(500). Rejecting here keeps this from becoming a
			// silent truncation, which changes which host the end user's browser
			// contacts, or a driver error surfacing as a 500.
			name: "logo url over the column width",
			mutate: func(r handler.Request) handler.Request {
				r.LogoUrl = nullable.NewNullableWithValue("https://example.com/" + strings.Repeat("a", 500))
				return r
			},
		},
		{
			name: "logo url set to empty string",
			mutate: func(r handler.Request) handler.Request {
				r.LogoUrl = nullable.NewNullableWithValue("")
				return r
			},
		},
		{
			name: "primary color without hash",
			mutate: func(r handler.Request) handler.Request {
				r.PrimaryColor = nullable.NewNullableWithValue("6366f1")
				return r
			},
		},
		{
			name: "primary color shorthand",
			mutate: func(r handler.Request) handler.Request {
				r.PrimaryColor = nullable.NewNullableWithValue("#fff")
				return r
			},
		},
		{
			name:   "blank target",
			mutate: func(r handler.Request) handler.Request { r.Portal = "   "; return r },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, tc.mutate(baseRequest(stored.ID)))
			require.Equal(t, http.StatusBadRequest, res.Status,
				"expected 400, received: %s", res.RawBody)

			row := fetchPortal(t, h, workspace.ID, stored.ID)
			require.Equal(t, "valid-portal", row.Slug, "a rejected request must not write")
			require.True(t, row.Enabled)
			require.Equal(t, mapping.ID, row.KeyAuthID.String)
			require.Equal(t, "https://cdn.example.com/logo.svg", row.LogoUrl.String)
			require.Equal(t, "#6366f1", row.PrimaryColor.String)
		})
	}
}

// `slug` and `enabled` are NOT NULL columns, and the generated request models
// them as pointers, so the handler itself cannot tell a JSON null from an omitted
// field. What closes that gap is the request-schema middleware, which rejects a
// null for a non-nullable property before the handler runs -- so a null never
// reaches MySQL as a NOT NULL violation. Pinned with raw bodies, because the
// typed request cannot express the state.
func TestUpdatePortalRejectsNullForNotNullFields(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	stored := h.SeedPortal(t, workspace.ID, "not-null", "not-null", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	testCases := map[string]string{
		"null slug":    fmt.Sprintf(`{"portal":%q,"slug":null}`, stored.ID),
		"null enabled": fmt.Sprintf(`{"portal":%q,"enabled":null}`, stored.ID),
		"both null":    fmt.Sprintf(`{"portal":%q,"slug":null,"enabled":null}`, stored.ID),
		// The mapping is a required-when-present choice, so clearing it -- which
		// would leave a row with neither association -- is rejected the same way.
		"null mapping": fmt.Sprintf(`{"portal":%q,"mapping":null}`, stored.ID),
	}

	for name, body := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[json.RawMessage, handler.Response](h, route, headers, json.RawMessage(body))
			require.Equal(t, http.StatusBadRequest, res.Status,
				"a null for a non-nullable field must be rejected before the write: %s", res.RawBody)

			row := fetchPortal(t, h, workspace.ID, stored.ID)
			require.Equal(t, "not-null", row.Slug, "the stored slug is kept, never cleared")
			require.True(t, row.Enabled, "the stored enabled flag is kept, never cleared")
			require.True(t, row.KeyAuthID.Valid, "the association is kept, never cleared")
		})
	}
}
