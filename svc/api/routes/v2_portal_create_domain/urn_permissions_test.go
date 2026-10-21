package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

func TestCreatePortalDomainAuthorizesCanonicalPortalURNs(t *testing.T) {
	f := setup(t)

	testCases := []struct {
		name       string
		resource   string
		action     string
		shouldPass bool
	}{
		{name: "this portal", resource: fmt.Sprintf("projects/%s/portals/%s", f.projectID, f.portal.ID), action: "write", shouldPass: true},
		{name: "every portal in the project", resource: fmt.Sprintf("projects/%s/portals/*", f.projectID), action: "write", shouldPass: true},
		{name: "workspace-wide admin", resource: "**", action: "*", shouldPass: true},
		{name: "this portal read", resource: fmt.Sprintf("projects/%s/portals/%s", f.projectID, f.portal.ID), action: "read", shouldPass: false},
		{name: "this portal in another project", resource: fmt.Sprintf("projects/%s/portals/%s", uid.New(uid.ProjectPrefix), f.portal.ID), action: "write", shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: randomDomain()},
				fmt.Sprintf("unkey:v1:%s:%s#%s", f.workspaceID, tc.resource, tc.action))
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "%s#%s must authorize: %s", tc.resource, tc.action, res.RawBody)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "%s#%s must be a masked 404: %s", tc.resource, tc.action, res.RawBody)
		})
	}
}
