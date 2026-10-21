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

// The workspace cap is enforced in ctrl; the API reflects it as the same 403 a
// plan allowance produces for environment domains.
func TestCreatePortalDomainCapReached(t *testing.T) {
	f := setup(t)
	f.ctrl.AddPortalDomainFunc = func(_ context.Context, _ *ctrlv1.AddPortalDomainRequest) (*ctrlv1.AddPortalDomainResponse, error) {
		return nil, asCtrlError(connect.CodeResourceExhausted, domaingate.CheckAllowance(10, 10))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: randomDomain()}, "portal.*.update_portal")
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}
