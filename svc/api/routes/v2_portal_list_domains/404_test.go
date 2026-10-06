package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_domains"
)

func TestListPortalDomainsUnresolvedPortal(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusPending)

	other := f.h.CreateWorkspace()
	foreign, _ := seedPortal(t, f.h, other.ID)
	f.h.SeedPortalDomain(t, other.ID, foreign.ID, randomDomain(), db.PortalDomainsVerificationStatusPending)

	testCases := []struct {
		name        string
		portal      string
		permissions []string
	}{
		{name: "unknown portal", portal: uid.New(uid.PortalPrefix), permissions: []string{"portal.*.read_portal"}},
		{name: "portal in another workspace", portal: foreign.ID, permissions: []string{"portal.*.read_portal"}},
		{name: "no permissions", portal: f.portal.ID, permissions: []string{}},
		{name: "create_portal only", portal: f.portal.ID, permissions: []string{"portal.*.create_portal"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: tc.portal}, tc.permissions...)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Contains(t, res.RawBody, "Portal not found.")
			require.NotContains(t, res.RawBody, stored.Domain)
		})
	}
}
