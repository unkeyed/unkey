package handler_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// A mapping the caller does not own reports the same not-found as one that
// exists nowhere.
//
// This is the check that keeps a portal from claiming another tenant's app or
// keyspace: those two unique keys span the whole table, so an unvalidated mapping
// would be a permanent global claim the victim could never clear.
func TestCreatePortalRejectsMappingsItDoesNotOwn(t *testing.T) {
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
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      other.ID,
		Name:             "theirs",
		Slug:             "theirs",
		DeleteProtection: false,
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      other.ID,
		ProjectID:        otherProject.ID,
		Name:             "theirs",
		Slug:             "theirs",
		DeleteProtection: false,
	})

	testCases := map[string]portal.Mapping{
		"keyspace owned by another workspace": {ID: otherApi.KeyAuthID.String, Type: portal.MappingTypeKeyspace},
		"app owned by another workspace":      {ID: otherApp.ID, Type: portal.MappingTypeApp},
		"keyspace that exists nowhere":        {ID: "ks_doesnotexist", Type: portal.MappingTypeKeyspace},
		"app that exists nowhere":             {ID: "app_doesnotexist", Type: portal.MappingTypeApp},
	}

	bodies := map[string]string{}
	for name, mapping := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Slug:        "not-mine",
				DisplayName: "Acme",
				KeyspaceId:  ksOf(mapping),
				AppId:       appOf(mapping),
				Enabled:     ptr.P(true),
			})
			require.Equal(t, http.StatusNotFound, res.Status,
				"expected 404, received: %s", res.RawBody)
			require.Equal(t, 0, countPortals(t, h, workspace.ID),
				"a rejected mapping must not write a portal")
			bodies[name] = res.RawBody
		})
	}

	// A foreign resource and an absent one must be indistinguishable. Otherwise
	// the difference answers "does this id exist in some other workspace", which
	// is exactly what the ownership check is meant to keep private.
	require.Equal(t,
		normalizeRequestID(bodies["keyspace owned by another workspace"]),
		normalizeRequestID(bodies["keyspace that exists nowhere"]),
		"a foreign keyspace must look identical to an absent one")
	require.Equal(t,
		normalizeRequestID(bodies["app owned by another workspace"]),
		normalizeRequestID(bodies["app that exists nowhere"]),
		"a foreign app must look identical to an absent one")

	// The victim workspace can still wire up its own portal for the resource the
	// caller tried to claim, which is the consequence that would break if the
	// squat had been allowed.
	// The same registered route, called with the other workspace's key: the
	// principal comes from the key, so one registration serves both callers.
	victimKey := h.CreateRootKey(other.ID,
		append([]string{"portal.*.create_portal"}, targetReadGrants...)...)
	victimHeaders := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + victimKey},
	}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, victimHeaders, handler.Request{
		Slug:        "mine-after-all",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(portal.Mapping{Type: portal.MappingTypeKeyspace, ID: otherApi.KeyAuthID.String}),
		AppId:       appOf(portal.Mapping{Type: portal.MappingTypeKeyspace, ID: otherApi.KeyAuthID.String}),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusOK, res.Status,
		"the owning workspace must still be able to claim its own keyspace: %s", res.RawBody)

	// Guard against the fixture silently drifting into a no-op.
	require.NotEqual(t, sql.NullString{}, otherApi.KeyAuthID)
}
