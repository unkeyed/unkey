package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_verify_domain"
)

func TestVerifyPortalDomainAlreadyVerified(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusVerified)

	res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, "portal.*.update_portal")
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Empty(t, f.ctrl.RetryVerificationCalls, "a verified domain must never reach ctrl")
}

// The row verified between the API's read and ctrl's re-check.
func TestVerifyPortalDomainCtrlPrecondition(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusFailed)
	f.ctrl.RetryVerificationFunc = func(_ context.Context, _ *ctrlv1.RetryPortalDomainVerificationRequest) (*ctrlv1.RetryPortalDomainVerificationResponse, error) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("internal gate wording"))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, "portal.*.update_portal")
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.RawBody, "The domain is already verified. No action is needed.")
	require.NotContains(t, res.RawBody, "internal gate wording")
}
