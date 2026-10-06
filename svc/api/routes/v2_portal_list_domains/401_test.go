package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_domains"
)

func TestListPortalDomainsRequiresAuthentication(t *testing.T) {
	f := setup(t)

	res := testutil.CallRoute[handler.Request, handler.Response](f.h, f.route,
		testutil.RootKeyHeaders("unkey_thiskeydoesnotexist"),
		handler.Request{Portal: f.portal.ID})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
