package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_domain"
)

func TestGetPortalDomainMasksDeniedPortal(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusPending)

	for name, permissions := range map[string][]string{
		"no permissions":       {},
		"create_portal only":   {"portal.*.create_portal"},
		"session minting only": {"portal.*.create_portal_session"},
	} {
		t.Run(name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, permissions...)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Contains(t, res.RawBody, "Portal not found.")
			require.NotContains(t, res.RawBody, stored.Domain)
		})
	}
}

// The domain id is resolved under the requested portal, so naming a domain that
// exists elsewhere reads the same as one that exists nowhere.
func TestGetPortalDomainOutsideThePortal(t *testing.T) {
	f := setup(t)

	sibling, _ := seedPortal(t, f.h, f.workspaceID)
	siblingDomain := f.h.SeedPortalDomain(t, f.workspaceID, sibling.ID, randomDomain(), db.PortalDomainsVerificationStatusPending)

	other := f.h.CreateWorkspace()
	foreignPortal, _ := seedPortal(t, f.h, other.ID)
	foreignDomain := f.h.SeedPortalDomain(t, other.ID, foreignPortal.ID, randomDomain(), db.PortalDomainsVerificationStatusPending)

	for name, domainID := range map[string]string{
		"unknown id":                 uid.New(uid.PortalDomainPrefix),
		"another portal, same space": siblingDomain.ID,
		"another workspace":          foreignDomain.ID,
	} {
		t.Run(name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: domainID}, "portal.*.read_portal")
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Contains(t, res.RawBody, "The requested domain does not exist.")
		})
	}
}

func TestGetPortalDomainForeignPortal(t *testing.T) {
	f := setup(t)

	other := f.h.CreateWorkspace()
	foreignPortal, _ := seedPortal(t, f.h, other.ID)
	foreignDomain := f.h.SeedPortalDomain(t, other.ID, foreignPortal.ID, randomDomain(), db.PortalDomainsVerificationStatusPending)

	res := f.call(t, handler.Request{Portal: foreignPortal.ID, DomainId: foreignDomain.ID}, "portal.*.read_portal")
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Contains(t, res.RawBody, "Portal not found.")
}
