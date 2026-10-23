package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

func TestCreatePortalDomainCapReached(t *testing.T) {
	f := setup(t)
	for range domaingate.PortalDomainsMax {
		require.NoError(t, db.Query.InsertPortalDomain(context.Background(), f.h.DB.RW(), db.InsertPortalDomainParams{
			ID:                 uid.New(uid.PortalDomainPrefix),
			WorkspaceID:        f.workspaceID,
			PortalID:           f.portal.ID,
			Domain:             randomDomain(),
			VerificationStatus: db.PortalDomainsVerificationStatusPending,
			VerificationToken:  uid.Secure(24),
			TargetCname:        randomDomain(),
			CreatedAt:          time.Now().UnixMilli(),
		}))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: randomDomain()}, "portal.*.update_portal")
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.Empty(t, f.ctrl.AddPortalDomainCalls)
}

// ctrl re-checks the cap, so a create that races past the handler's count is
// reflected as the same 403.
func TestCreatePortalDomainCapReachedInCtrl(t *testing.T) {
	f := setup(t)
	f.ctrl.AddPortalDomainFunc = func(_ context.Context, _ *ctrlv1.AddPortalDomainRequest) (*ctrlv1.AddPortalDomainResponse, error) {
		return nil, asCtrlError(connect.CodeResourceExhausted, domaingate.CheckPortalAllowance(domaingate.PortalDomainsMax))
	}

	res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: randomDomain()}, "portal.*.update_portal")
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}
