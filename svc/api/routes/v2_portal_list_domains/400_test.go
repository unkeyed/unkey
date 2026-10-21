package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_domains"
)

func TestListPortalDomainsRejectsMalformedPortal(t *testing.T) {
	f := setup(t)

	for name, portal := range map[string]string{
		"missing":   "",
		"malformed": "not a portal",
	} {
		t.Run(name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: portal}, "portal.*.read_portal")
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
