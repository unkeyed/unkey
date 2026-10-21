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

// ctrl's internal failures carry no public-message guarantee, so they must not
// be reflected.
func TestDeletePortalDomainCtrlFailureIsNotReflected(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusVerified)
	f.ctrl.DeletePortalDomainFunc = func(_ context.Context, _ *ctrlv1.DeletePortalDomainRequest) (*ctrlv1.DeletePortalDomainResponse, error) {
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to delete frontline route: table is gone"))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, DomainId: stored.ID}, "portal.*.update_portal")
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, "frontline")
}
