package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_domain"
)

func TestDeletePortalDomainAuthorization(t *testing.T) {
	f := setup(t)
	portalURN := fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s", f.workspaceID, f.projectID, f.portal.ID)

	testCases := []struct {
		name       string
		permission string
		status     int
	}{
		{name: "this portal's domains", permission: f.grant("delete"), status: http.StatusOK},
		{name: "every portal's domains in the project", permission: fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*/domains/*#delete", f.workspaceID, f.projectID), status: http.StatusOK},
		{name: "everything under this portal", permission: portalURN + "/**#delete", status: http.StatusOK},
		{name: "admin", permission: fmt.Sprintf("unkey:v1:%s:**#*", f.workspaceID), status: http.StatusOK},
		{name: "domains read names the missing permission", permission: f.grant("read"), status: http.StatusForbidden},
		{name: "portal read names the missing permission", permission: portalURN + "#read", status: http.StatusForbidden},
		{name: "domains write only", permission: f.grant("write"), status: http.StatusNotFound},
		{name: "portal delete without domains", permission: portalURN + "#delete", status: http.StatusNotFound},
		{name: "domains in another project", permission: fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/domains/*#delete", f.workspaceID, uid.New(uid.ProjectPrefix), f.portal.ID), status: http.StatusNotFound},
		{name: "legacy update_portal", permission: "portal.*.update_portal", status: http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stored := f.seedDomain(t, db.PortalDomainsVerificationStatusFailed)
			res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, tc.permission)
			require.Equal(t, tc.status, res.Status, "%s: %s", tc.permission, res.RawBody)
		})
	}
}
