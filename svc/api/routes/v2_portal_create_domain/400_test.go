package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

func TestCreatePortalDomainRejectsInvalidHostnames(t *testing.T) {
	f := setup(t)

	testCases := map[string]string{
		"scheme":        "https://portal.acme.com",
		"path":          "portal.acme.com/login",
		"ipv4 literal":  "192.168.10.1",
		"public suffix": "co.uk",
		"single label":  "localhost",
		"underscore":    "por_tal.acme.com",
		"port":          "portal.acme.com:8443",
		"trailing dot":  "portal.acme.com.",
		"too short":     "a.b",
	}

	for name, domain := range testCases {
		t.Run(name, func(t *testing.T) {
			res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: domain}, "portal.*.update_portal")
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	require.Empty(t, f.ctrl.AddPortalDomainCalls, "an invalid request must never reach ctrl")
}
