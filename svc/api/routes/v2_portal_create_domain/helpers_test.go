package handler_test

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
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
	mapping, projectID := h.SeedKeyspaceMapping(t, workspaceID)
	slug := uid.DNS1035(12)
	return fixture{
		h:           h,
		route:       route,
		ctrl:        ctrl,
		workspaceID: workspaceID,
		projectID:   projectID,
		portal:      h.SeedPortal(t, workspaceID, slug, slug, mapping, nil, nil),
	}
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

// asCtrlError mirrors svc/ctrl/internal/gatefault.ConnectWith, which sends only
// the gate's public message across the wire.
func asCtrlError(code connect.Code, gateErr error) error {
	return connect.NewError(code, errors.New(fault.UserFacingMessage(gateErr)))
}
