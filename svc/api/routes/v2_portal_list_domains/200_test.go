package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_domains"
)

func TestListPortalDomainsOnlyThisPortal(t *testing.T) {
	f := setup(t)
	pending := f.seedDomain(t, db.PortalDomainsVerificationStatusPending)
	verified := f.seedDomain(t, db.PortalDomainsVerificationStatusVerified)

	sibling, _ := seedPortal(t, f.h, f.workspaceID)
	f.h.SeedPortalDomain(t, f.workspaceID, sibling.ID, randomDomain(), db.PortalDomainsVerificationStatusVerified)

	res := f.call(t, handler.Request{Portal: f.portal.Slug}, "portal.*.read_portal")
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	statuses := map[string]openapi.PortalDomainStatus{}
	for _, d := range res.Body.Data {
		require.Equal(t, f.portal.ID, d.PortalId, "a sibling portal's domain leaked into the list")
		require.Len(t, d.DnsRecords, 2)
		statuses[d.Id] = d.Status
	}
	require.Equal(t, map[string]openapi.PortalDomainStatus{
		pending.ID:  openapi.PortalDomainStatusPending,
		verified.ID: openapi.PortalDomainStatusVerified,
	}, statuses)
}

func TestListPortalDomainsEmpty(t *testing.T) {
	f := setup(t)

	res := f.call(t, handler.Request{Portal: f.portal.ID}, "portal.*.read_portal")
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body.Data)
	require.Empty(t, res.Body.Data)
	require.Contains(t, res.RawBody, `"data":[]`)
}
