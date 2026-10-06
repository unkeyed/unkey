package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_domain"
)

func TestGetPortalDomain(t *testing.T) {
	f := setup(t)
	stored := f.seedDomain(t, db.PortalDomainsVerificationStatusVerifying)

	res := f.call(t, handler.Request{Portal: f.portal.Slug, DomainId: stored.ID}, "portal.*.read_portal")
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	data := res.Body.Data
	require.Equal(t, stored.ID, data.Id)
	require.Equal(t, f.portal.ID, data.PortalId)
	require.Equal(t, stored.Domain, data.Domain)
	require.Equal(t, openapi.PortalDomainStatusVerifying, data.Status)
	require.Equal(t, stored.CreatedAt, data.CreatedAt)
	require.Nil(t, data.VerificationError)

	require.Len(t, data.DnsRecords, 2)
	require.Equal(t, openapi.CNAME, data.DnsRecords[0].Type)
	require.Equal(t, stored.TargetCname, data.DnsRecords[0].Value)
	require.Equal(t, openapi.TXT, data.DnsRecords[1].Type)
	require.Equal(t, "_unkey."+stored.Domain, data.DnsRecords[1].Name)
	require.Equal(t, "unkey-domain-verify="+stored.VerificationToken, data.DnsRecords[1].Value)
}
