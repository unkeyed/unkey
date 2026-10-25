package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_domains"
)

func TestListPortalDomainsAuthorization(t *testing.T) {
	f := setup(t)
	portalURN := fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s", f.workspaceID, f.projectID, f.portal.ID)

	testCases := []struct {
		name       string
		permission string
		shouldPass bool
	}{
		{name: "legacy read_portal", permission: "portal.*.read_portal", shouldPass: true},
		{name: "legacy update_portal", permission: "portal.*.update_portal", shouldPass: true},
		{name: "urn read", permission: portalURN + "#read", shouldPass: true},
		{name: "urn write", permission: portalURN + "#write", shouldPass: true},
		{name: "urn admin", permission: fmt.Sprintf("unkey:v1:%s:**#*", f.workspaceID), shouldPass: true},
		{name: "urn read on another project", permission: fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s#read", f.workspaceID, uid.New(uid.ProjectPrefix), f.portal.ID), shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: f.portal.ID}, tc.permission)
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "%s must authorize: %s", tc.permission, res.RawBody)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "%s must be a masked 404: %s", tc.permission, res.RawBody)
		})
	}
}
