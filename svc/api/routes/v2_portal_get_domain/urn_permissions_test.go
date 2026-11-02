package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_domain"
)

func TestGetPortalDomainAuthorization(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusPending)
	portalURN := fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s", f.workspaceID, f.projectID, f.portal.ID)

	testCases := []struct {
		name       string
		permission string
		status     int
	}{
		{name: "this portal's domains", permission: f.grant("read"), status: http.StatusOK},
		{name: "this domain", permission: fmt.Sprintf("%s/domains/%s#read", portalURN, stored.ID), status: http.StatusOK},
		{name: "every portal's domains in the project", permission: fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*/domains/*#read", f.workspaceID, f.projectID), status: http.StatusOK},
		{name: "everything under this portal", permission: portalURN + "/**#read", status: http.StatusOK},
		{name: "admin", permission: fmt.Sprintf("unkey:v1:%s:**#*", f.workspaceID), status: http.StatusOK},
		{name: "portal read names the missing permission", permission: portalURN + "#read", status: http.StatusForbidden},
		{name: "domains write only", permission: f.grant("write"), status: http.StatusNotFound},
		{name: "another domain", permission: fmt.Sprintf("%s/domains/%s#read", portalURN, uid.New(uid.PortalDomainPrefix)), status: http.StatusNotFound},
		{name: "domains in another project", permission: fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/domains/*#read", f.workspaceID, uid.New(uid.ProjectPrefix), f.portal.ID), status: http.StatusNotFound},
		{name: "legacy read_portal", permission: "portal.*.read_portal", status: http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, tc.permission)
			require.Equal(t, tc.status, res.Status, "%s: %s", tc.permission, res.RawBody)
		})
	}
}
