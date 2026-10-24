package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

// An environment without a portal app answers FailedPrecondition. The shared
// domain mapper reports it as a server-side failure, reflecting ctrl's message.
func TestCreatePortalDomainNotConfigured(t *testing.T) {
	f := setup(t)
	const message = "Portal domains are not available in this environment. Contact support@unkey.com."
	f.ctrl.AddPortalDomainFunc = func(_ context.Context, _ *ctrlv1.AddPortalDomainRequest) (*ctrlv1.AddPortalDomainResponse, error) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(message))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: randomDomain()}, "portal.*.update_portal")
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Contains(t, res.RawBody, message)
}
