package handler_test

import (
	"testing"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_verify_domain"
)

type fixture struct {
	h           *testutil.Harness
	route       *handler.Handler
	ctrl        *testutil.MockPortalDomainClient
	workspaceID string
	projectID   string
	portal      db.Portal
}

func setup(t *testing.T) fixture {
	t.Helper()

	h := testutil.NewHarness(t)
	ctrl := &testutil.MockPortalDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrl}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	portal, projectID := seedPortal(t, h, workspaceID)
	return fixture{
		h:           h,
		route:       route,
		ctrl:        ctrl,
		workspaceID: workspaceID,
		projectID:   projectID,
		portal:      portal,
	}
}

// seedPortal creates a keyspace-backed portal and returns it with its project.
func seedPortal(t *testing.T, h *testutil.Harness, workspaceID string) (db.Portal, string) {
	t.Helper()
	mapping, projectID := h.SeedKeyspaceMapping(t, workspaceID)
	slug := uid.DNS1035(12)
	return h.SeedPortal(t, workspaceID, slug, slug, mapping, nil, nil), projectID
}

// seedDomain attaches a fresh hostname to the fixture's portal.
func (f fixture) seedDomain(t *testing.T, status db.PortalDomainsVerificationStatus) db.FindPortalDomainByIdRow {
	t.Helper()
	return f.h.SeedPortalDomain(t, f.workspaceID, f.portal.ID, randomDomain(), status)
}

// call sends req authenticated with a fresh root key holding permissions.
func (f fixture) call(t *testing.T, req handler.Request, permissions ...string) testutil.TestResponse[handler.Response] {
	t.Helper()
	rootKey := f.h.CreateRootKey(f.workspaceID, permissions...)
	return testutil.CallRoute[handler.Request, handler.Response](f.h, f.route, testutil.RootKeyHeaders(rootKey), req)
}

func randomDomain() string {
	return uid.DNS1035(16) + ".example.com"
}
