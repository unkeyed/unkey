package handler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_domain"
)

func TestCreatePortalDomainReturnsDNSRecords(t *testing.T) {
	f := setup(t)

	domainID := uid.New(uid.PortalDomainPrefix)
	f.ctrl.AddPortalDomainFunc = func(_ context.Context, _ *ctrlv1.AddPortalDomainRequest) (*ctrlv1.AddPortalDomainResponse, error) {
		return &ctrlv1.AddPortalDomainResponse{
			DomainId:          domainID,
			TargetCname:       "a1b2c3d4e5f6g7h8.portal.unkey.com",
			Status:            ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
			VerificationToken: "3ZQ8xK1mP7vT5nR2wY6bJ4hL",
		}, nil
	}

	domain := randomDomain()
	res := f.call(t, handler.Request{Portal: f.portal.Slug, Domain: domain}, "portal.*.update_portal")
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, domainID, res.Body.Data.DomainId)

	require.Len(t, res.Body.Data.DnsRecords, 2)
	routing := res.Body.Data.DnsRecords[0]
	require.Equal(t, openapi.CNAME, routing.Type)
	require.Equal(t, domain, routing.Name)
	require.Equal(t, "a1b2c3d4e5f6g7h8.portal.unkey.com", routing.Value)
	require.False(t, routing.Verified)

	txt := res.Body.Data.DnsRecords[1]
	require.Equal(t, openapi.TXT, txt.Type)
	require.Equal(t, "_unkey."+domain, txt.Name)
	require.Equal(t, "unkey-domain-verify=3ZQ8xK1mP7vT5nR2wY6bJ4hL", txt.Value)
	require.False(t, txt.Verified)

	// The portal was addressed by slug; ctrl must receive the resolved id.
	require.Len(t, f.ctrl.AddPortalDomainCalls, 1)
	call := f.ctrl.AddPortalDomainCalls[0]
	require.Equal(t, f.workspaceID, call.GetWorkspaceId())
	require.Equal(t, f.portal.ID, call.GetPortalId())
	require.Equal(t, domain, call.GetDomain())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, call.GetActor().GetType())
}

// ctrl receives the canonical form, so the stored name and the records agree.
func TestCreatePortalDomainCanonicalizesTheName(t *testing.T) {
	f := setup(t)

	domain := randomDomain()
	res := f.call(t, handler.Request{Portal: f.portal.ID, Domain: strings.ToUpper(domain)}, "portal.*.update_portal")
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, f.ctrl.AddPortalDomainCalls, 1)
	require.Equal(t, domain, f.ctrl.AddPortalDomainCalls[0].GetDomain())
	require.Equal(t, domain, res.Body.Data.DnsRecords[0].Name)
}
