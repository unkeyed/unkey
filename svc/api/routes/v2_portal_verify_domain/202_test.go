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

func TestVerifyPortalDomainCallsCtrlWithActor(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusFailed)

	res := f.call(t, handler.Request{Portal: f.portal.Slug, DomainId: stored.ID}, f.grant("write"))
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	require.Len(t, f.ctrl.RetryPortalDomainVerificationCalls, 1)
	call := f.ctrl.RetryPortalDomainVerificationCalls[0]
	require.Equal(t, f.workspaceID, call.GetWorkspaceId())
	require.Equal(t, f.portal.ID, call.GetPortalId())
	require.Equal(t, stored.ID, call.GetDomainId())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, call.GetActor().GetType())
	require.NotEmpty(t, call.GetActor().GetId())
}

// Verifying a verified domain is a no-op: the caller's goal already holds.
func TestVerifyPortalDomainAlreadyVerified(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusVerified)

	res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, f.grant("write"))
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
	require.Empty(t, f.ctrl.RetryPortalDomainVerificationCalls, "a verified domain must never reach ctrl")
}

// The row verified between the API's read and ctrl's re-check.
func TestVerifyPortalDomainVerifiedBeforeCtrl(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusFailed)
	f.ctrl.RetryPortalDomainVerificationFunc = func(_ context.Context, _ *ctrlv1.RetryPortalDomainVerificationRequest) (*ctrlv1.RetryPortalDomainVerificationResponse, error) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("internal gate wording"))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, f.grant("write"))
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, "internal gate wording")
}
