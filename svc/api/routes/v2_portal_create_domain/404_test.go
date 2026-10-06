package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

// ctrl trusts the portal id it is sent, so every miss must stop in the API.
func TestCreatePortalDomainUnresolvedPortalNeverReachesCtrl(t *testing.T) {
	f := setup(t)

	other := f.h.CreateWorkspace()
	otherMapping, _ := f.h.SeedKeyspaceMapping(t, other.ID)
	foreign := f.h.SeedPortal(t, other.ID, "foreign", "foreign", otherMapping, nil, nil)

	testCases := []struct {
		name        string
		portal      string
		permissions []string
	}{
		{name: "unknown portal", portal: uid.New(uid.PortalPrefix), permissions: []string{"portal.*.update_portal"}},
		{name: "portal in another workspace", portal: foreign.ID, permissions: []string{"portal.*.update_portal"}},
		{name: "no permissions", portal: f.portal.ID, permissions: []string{}},
		{name: "read only", portal: f.portal.ID, permissions: []string{"portal.*.read_portal"}},
		{name: "update on another portal", portal: f.portal.ID, permissions: []string{fmt.Sprintf("portal.%s.update_portal", uid.New(uid.PortalPrefix))}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: tc.portal, Domain: randomDomain()}, tc.permissions...)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Contains(t, res.RawBody, "Portal not found.")
			require.NotContains(t, res.RawBody, f.portal.ID, "a masked denial must not disclose the portal id")
		})
	}

	require.Empty(t, f.ctrl.AddPortalDomainCalls)
}
