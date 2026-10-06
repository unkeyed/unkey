package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_domain"
)

func TestGetPortalDomainRejectsMalformedRequests(t *testing.T) {
	f := setup(t)

	testCases := map[string]handler.Request{
		"missing domain id":   {Portal: f.portal.ID, DomainId: ""},
		"malformed domain id": {Portal: f.portal.ID, DomainId: "pdom 1234"},
		"missing portal":      {Portal: "", DomainId: "pdom_1234abcd"},
	}

	for name, req := range testCases {
		t.Run(name, func(t *testing.T) {
			res := f.call(t, req, "portal.*.read_portal")
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
