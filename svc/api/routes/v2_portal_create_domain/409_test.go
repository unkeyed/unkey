package handler_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

// ctrl rejects a name already held by this workspace, as a portal domain or an
// environment domain, with AlreadyExists.
func TestCreatePortalDomainDuplicate(t *testing.T) {
	f := setup(t)
	domain := randomDomain()
	f.ctrl.AddPortalDomainFunc = func(_ context.Context, req *ctrlv1.AddPortalDomainRequest) (*ctrlv1.AddPortalDomainResponse, error) {
		return nil, asCtrlError(connect.CodeAlreadyExists, domaingate.AlreadyExists(req.GetDomain()))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: domain}, "portal.*.update_portal")
	require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
	require.Contains(t, res.RawBody, domain)
}
