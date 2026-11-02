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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_domain"
)

func TestDeletePortalDomainCallsCtrlWithActor(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusVerified)

	res := f.call(t, handler.Request{Portal: f.portal.Slug, DomainId: stored.ID}, f.grant("delete"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Len(t, f.ctrl.DeletePortalDomainCalls, 1)
	call := f.ctrl.DeletePortalDomainCalls[0]
	require.Equal(t, f.workspaceID, call.GetWorkspaceId())
	require.Equal(t, f.portal.ID, call.GetPortalId())
	require.Equal(t, stored.ID, call.GetDomainId())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, call.GetActor().GetType())
	require.NotEmpty(t, call.GetActor().GetId())
}

// The row was removed between the API's read and ctrl's, which leaves what the
// caller asked for.
func TestDeletePortalDomainAlreadyDeletedInCtrl(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusFailed)
	f.ctrl.DeletePortalDomainFunc = func(_ context.Context, _ *ctrlv1.DeletePortalDomainRequest) (*ctrlv1.DeletePortalDomainResponse, error) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("portal domain not found: "+stored.ID))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, f.grant("delete"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
}
