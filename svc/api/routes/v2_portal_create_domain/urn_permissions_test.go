package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

func TestCreatePortalDomainAuthorizesCanonicalPortalDomainURNs(t *testing.T) {
	f := setup(t)

	testCases := []struct {
		name     string
		resource string
		action   string
		status   int
	}{
		{name: "this portal's domains", resource: fmt.Sprintf("projects/%s/portals/%s/domains/*", f.projectID, f.portal.ID), action: "write", status: http.StatusOK},
		{name: "every portal's domains in the project", resource: fmt.Sprintf("projects/%s/portals/*/domains/*", f.projectID), action: "write", status: http.StatusOK},
		{name: "everything under this portal", resource: fmt.Sprintf("projects/%s/portals/%s/**", f.projectID, f.portal.ID), action: "write", status: http.StatusOK},
		{name: "workspace-wide admin", resource: "**", action: "*", status: http.StatusOK},
		{name: "this portal's domains read names the missing permission", resource: fmt.Sprintf("projects/%s/portals/%s/domains/*", f.projectID, f.portal.ID), action: "read", status: http.StatusForbidden},
		{name: "this portal itself", resource: fmt.Sprintf("projects/%s/portals/%s", f.projectID, f.portal.ID), action: "write", status: http.StatusNotFound},
		{name: "this portal's domains in another project", resource: fmt.Sprintf("projects/%s/portals/%s/domains/*", uid.New(uid.ProjectPrefix), f.portal.ID), action: "write", status: http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: randomDomain()},
				fmt.Sprintf("unkey:v1:%s:%s#%s", f.workspaceID, tc.resource, tc.action))
			require.Equal(t, tc.status, res.Status, "%s#%s: %s", tc.resource, tc.action, res.RawBody)
		})
	}
}
