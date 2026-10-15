package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_verify_domain"
)

// ctrl trusts the portal and domain ids it is sent, so every miss must stop in
// the API.
func TestVerifyPortalDomainMissesNeverReachCtrl(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusFailed)

	sibling, _ := seedPortal(t, f.h, f.workspaceID)
	siblingDomain := f.h.SeedPortalDomain(t, f.workspaceID, sibling.ID, randomDomain(), db.PortalDomainsVerificationStatusFailed)

	other := f.h.CreateWorkspace()
	foreignPortal, _ := seedPortal(t, f.h, other.ID)
	foreignDomain := f.h.SeedPortalDomain(t, other.ID, foreignPortal.ID, randomDomain(), db.PortalDomainsVerificationStatusFailed)

	testCases := []struct {
		name        string
		req         handler.Request
		permissions []string
		message     string
	}{
		{name: "unknown domain", req: handler.Request{Portal: f.portal.ID, DomainId: uid.New(uid.PortalDomainPrefix)}, permissions: []string{"portal.*.update_portal"}, message: "The requested domain does not exist."},
		{name: "domain on another portal in this workspace", req: handler.Request{Portal: f.portal.ID, DomainId: siblingDomain.ID}, permissions: []string{"portal.*.update_portal"}, message: "The requested domain does not exist."},
		{name: "domain in another workspace", req: handler.Request{Portal: f.portal.ID, DomainId: foreignDomain.ID}, permissions: []string{"portal.*.update_portal"}, message: "The requested domain does not exist."},
		{name: "portal in another workspace", req: handler.Request{Portal: foreignPortal.ID, DomainId: foreignDomain.ID}, permissions: []string{"portal.*.update_portal"}, message: "Portal not found."},
		{name: "no permissions", req: handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, permissions: []string{}, message: "Portal not found."},
		{name: "read only", req: handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, permissions: []string{"portal.*.read_portal"}, message: "Portal not found."},
		{name: "update on another portal", req: handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, permissions: []string{fmt.Sprintf("portal.%s.update_portal", sibling.ID)}, message: "Portal not found."},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := f.call(t, tc.req, tc.permissions...)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Contains(t, res.RawBody, tc.message)
		})
	}

	require.Empty(t, f.ctrl.RetryVerificationCalls)
}

// The row was removed between the API's read and ctrl's.
func TestVerifyPortalDomainCtrlNotFound(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusFailed)
	f.ctrl.RetryVerificationFunc = func(_ context.Context, _ *ctrlv1.RetryPortalDomainVerificationRequest) (*ctrlv1.RetryPortalDomainVerificationResponse, error) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("portal domain not found: "+stored.ID))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, "portal.*.update_portal")
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Contains(t, res.RawBody, "The requested domain does not exist.")
}
