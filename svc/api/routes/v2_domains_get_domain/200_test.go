package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_get_domain"
)

func TestGetDomainByName(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	checkedAt := time.Now().UnixMilli()
	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusVerified
		req.CnameVerified = true
		req.LastCheckedAt = checkedAt
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	data := res.Body.Data
	require.Equal(t, seeded.domainID, data.Id)
	require.Equal(t, seeded.domain, data.Domain)
	require.Equal(t, seeded.projectID, data.ProjectId)
	require.Equal(t, seeded.appID, data.AppId)
	require.Equal(t, seeded.environmentID, data.EnvironmentId)
	require.Equal(t, openapi.DomainStatusVerified, data.Status)
	require.True(t, data.RoutingVerified, "cname_verified was seeded true, received: %s", res.RawBody)
	require.False(t, data.OwnershipVerified)
	require.Nil(t, data.VerificationError, "no error was seeded, received: %s", res.RawBody)
	require.NotNil(t, data.LastCheckedAt)
	require.Equal(t, checkedAt, *data.LastCheckedAt)
	require.NotZero(t, data.CreatedAt)
}

// TestGetDomainById pins that the same row is reachable by id, since that is what
// domains.createDomain hands back.
func TestGetDomainById(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domainID,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, seeded.domainID, res.Body.Data.Id)
	require.Equal(t, seeded.domain, res.Body.Data.Domain)
}

// TestGetDomainDnsRecordsMatchCreate pins that the returned records are rebuilt from
// stored state with the same values domains.createDomain produced. A caller that lost
// the create response recovers them here, so a divergence would send them to records
// the verification worker does not look for.
func TestGetDomainDnsRecordsMatchCreate(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data.DnsRecords, 2, "expected routing + ownership records, received: %s", res.RawBody)

	routing := res.Body.Data.DnsRecords[0]
	require.Equal(t, openapi.CNAME, routing.Type)
	require.Equal(t, seeded.domain, routing.Name)
	require.Equal(t, seeded.targetCname, routing.Value)
	require.NotNil(t, routing.Note)

	txt := res.Body.Data.DnsRecords[1]
	require.Equal(t, openapi.TXT, txt.Type)
	require.Equal(t, "_unkey."+seeded.domain, txt.Name)
	require.Equal(t, "unkey-domain-verify="+seeded.token, txt.Value)
	require.NotNil(t, txt.Note)
}

// TestGetDomainApexRecords pins that an apex domain reports an apex-compatible alias
// rather than a CNAME it cannot hold.
func TestGetDomainApexRecords(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	apex := randomApexDomain()
	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.Domain = apex
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: apex,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data.DnsRecords, 2, "received: %s", res.RawBody)
	require.Equal(t, openapi.ALIAS, res.Body.Data.DnsRecords[0].Type, "apex cannot hold a CNAME, received: %s", res.RawBody)
	require.Equal(t, apex, res.Body.Data.DnsRecords[0].Name)
	require.Equal(t, "_unkey."+apex, res.Body.Data.DnsRecords[1].Name)
}

// TestGetDomainVerifiedWithoutRouting pins the case the response exists to expose:
// verification can pass on proof of ownership alone, so a domain reports verified
// while nothing routes. Collapsing this into status alone would tell the caller
// traffic is live when it is not.
func TestGetDomainVerifiedWithoutRouting(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusVerified
		req.OwnershipVerified = true
		req.CnameVerified = false
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, openapi.DomainStatusVerified, res.Body.Data.Status)
	require.True(t, res.Body.Data.OwnershipVerified)
	require.False(t, res.Body.Data.RoutingVerified, "routing must not be reported as verified, received: %s", res.RawBody)
}

func TestGetDomainFailedReportsError(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	const verificationError = "domain verification timed out after 24 hours"
	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusFailed
		req.VerificationError = verificationError
		req.LastCheckedAt = time.Now().UnixMilli()
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, openapi.DomainStatusFailed, res.Body.Data.Status)
	require.NotNil(t, res.Body.Data.VerificationError, "a failed domain must say why, received: %s", res.RawBody)
	require.Equal(t, verificationError, *res.Body.Data.VerificationError)
}

// TestGetDomainStatusMapping walks every stored state so a new database enum value
// cannot silently fall through to pending.
func TestGetDomainStatusMapping(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	testCases := []struct {
		stored   db.CustomDomainsVerificationStatus
		expected openapi.DomainStatus
	}{
		{stored: db.CustomDomainsVerificationStatusPending, expected: openapi.DomainStatusPending},
		{stored: db.CustomDomainsVerificationStatusVerifying, expected: openapi.DomainStatusVerifying},
		{stored: db.CustomDomainsVerificationStatusVerified, expected: openapi.DomainStatusVerified},
		{stored: db.CustomDomainsVerificationStatusFailed, expected: openapi.DomainStatusFailed},
	}

	for _, tc := range testCases {
		t.Run(string(tc.stored), func(t *testing.T) {
			seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
				req.VerificationStatus = tc.stored
			})
			rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
				Domain: seeded.domain,
			})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, tc.expected, res.Body.Data.Status, "received: %s", res.RawBody)
		})
	}
}

// TestGetDomainOmitsUnsetOptionalFields pins that a never-checked domain omits
// lastCheckedAt, verificationError, and updatedAt rather than serializing zero values
// a caller would read as real timestamps.
func TestGetDomainOmitsUnsetOptionalFields(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Nil(t, res.Body.Data.LastCheckedAt)
	require.Nil(t, res.Body.Data.VerificationError)
	require.Nil(t, res.Body.Data.UpdatedAt)
	require.NotContains(t, res.RawBody, "lastCheckedAt", "unset optional fields must be absent, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, "verificationError", "unset optional fields must be absent, received: %s", res.RawBody)
}

func TestGetDomainWithSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment."+seeded.environmentID+".read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
}
