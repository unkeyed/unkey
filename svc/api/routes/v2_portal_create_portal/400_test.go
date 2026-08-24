package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// Note what is absent: there is no case for "both an app and a keyspace" or for
// "neither". The request names the mapping kind and id together, so those states
// cannot be expressed rather than being rejected at runtime. What remains
// reachable is a kind outside the enum and an empty id.
func TestCreatePortalRejectsInvalidInput(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.create_portal")
	workspace := h.Resources().UserWorkspace
	mapping := keyspaceMapping(t, h, workspace.ID)

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{
			name: "slug too short",
			req: handler.Request{Slug: "ab", DisplayName: "Acme", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true)},
		},
		{
			name: "slug uppercase",
			req: handler.Request{Slug: "Acme-Portal", DisplayName: "Acme", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true)},
		},
		{
			name: "slug with consecutive hyphens",
			req: handler.Request{Slug: "acme--portal", DisplayName: "Acme", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true)},
		},
		{
			name: "slug leading hyphen",
			req: handler.Request{Slug: "-acme", DisplayName: "Acme", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true)},
		},
		{
			name: "slug too long",
			req: handler.Request{
				Slug:        strings.Repeat("a", 65),
				DisplayName: "Acme",
				KeyspaceId:  ksOf(mapping),
				AppId:       appOf(mapping),
				Enabled:     ptr(true),
			},
		},
		{
			name: "display name omitted",
			req: handler.Request{Slug: "valid-slug", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true)},
		},
		{
			name: "display name is whitespace only",
			req: handler.Request{Slug: "valid-slug", DisplayName: "   ", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true)},
		},
		{
			name: "display name too long",
			req: handler.Request{
				Slug:        "valid-slug",
				DisplayName: strings.Repeat("a", 65),
				KeyspaceId:  ksOf(mapping),
				AppId:       appOf(mapping),
				Enabled:     ptr(true),
			},
		},
		{
			// The id-or-slug resolver treats one argument as either form, so a
			// slug that could pass for an id would make resolution ambiguous. The
			// slug charset forbids underscores, which is what keeps them apart.
			name: "id-shaped slug",
			req: handler.Request{Slug: "pc_1234abcd", DisplayName: "Acme", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true)},
		},
		{
			name: "empty mapping id",
			req: handler.Request{
				Slug:        "valid-slug",
				DisplayName: "Acme",
				KeyspaceId:  ksOf(portal.Mapping{Type: portal.MappingTypeKeyspace, ID: ""}),
				AppId:       appOf(portal.Mapping{Type: portal.MappingTypeKeyspace, ID: ""}),
				Enabled:     ptr(true),
			},
		},
		{
			name: "unknown mapping type",
			req: handler.Request{
				Slug:        "valid-slug",
				DisplayName: "Acme",
				KeyspaceId:  ksOf(portal.Mapping{Type: portal.MappingType("project"), ID: mapping.ID}),
				AppId:       appOf(portal.Mapping{Type: portal.MappingType("project"), ID: mapping.ID}),
				Enabled:     ptr(true),
			},
		},
		{
			name: "logo url with http scheme",
			req: handler.Request{
				Slug: "valid-slug", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true),
				DisplayName: "Acme",
				LogoUrl:     ptr("http://cdn.example.com/logo.svg"),
			},
		},
		{
			name: "logo url not a url",
			req: handler.Request{
				Slug: "valid-slug", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true),
				DisplayName: "Acme",
				LogoUrl:     ptr("not a url at all"),
			},
		},
		{
			// The column is varchar(500). Rejecting here is what keeps this from
			// becoming either a silent truncation, which changes which host is
			// contacted, or a driver error surfacing as a 500.
			name: "logo url over the column width",
			req: handler.Request{
				Slug: "valid-slug", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true),
				DisplayName: "Acme",
				LogoUrl:     ptr("https://example.com/" + strings.Repeat("a", 500)),
			},
		},
		{
			name: "primary color without hash",
			req: handler.Request{
				Slug: "valid-slug", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true),
				DisplayName:  "Acme",
				PrimaryColor: ptr("6366f1"),
			},
		},
		{
			name: "primary color shorthand",
			req: handler.Request{
				Slug: "valid-slug", KeyspaceId: ksOf(mapping),
				AppId: appOf(mapping), Enabled: ptr(true),
				DisplayName:  "Acme",
				PrimaryColor: ptr("#fff"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			before := countPortals(t, h, workspace.ID)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status,
				"expected 400, received: %s", res.RawBody)
			require.Equal(t, before, countPortals(t, h, workspace.ID),
				"a rejected request must not write a portal")
		})
	}
}
