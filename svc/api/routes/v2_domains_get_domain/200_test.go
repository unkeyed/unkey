package handler_test

import (
	"net/http"
	"strings"
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
	require.Nil(t, data.VerificationError, "no error was seeded, received: %s", res.RawBody)
	require.NotZero(t, data.CreatedAt)

	// cname_verified was seeded true and ownership_verified false, and each flag belongs to
	// the record it was checked against.
	require.Len(t, data.DnsRecords, 2, "received: %s", res.RawBody)
	require.Equal(t, openapi.CNAME, data.DnsRecords[0].Type)
	require.True(t, data.DnsRecords[0].Verified, "received: %s", res.RawBody)
	require.Equal(t, openapi.TXT, data.DnsRecords[1].Type)
	require.False(t, data.DnsRecords[1].Verified, "received: %s", res.RawBody)
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

// TestGetDomainByNameIsCaseInsensitive pins that a name reaches its row whatever case
// the caller sends, since domains.createDomain lowercases before storing and a caller
// echoing a name from their own config should not have to match that.
func TestGetDomainByNameIsCaseInsensitive(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	for _, identifier := range []string{strings.ToUpper(seeded.domain), strings.ToTitle(seeded.domain)} {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
			Domain: identifier,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200 for %q, received: %s", identifier, res.RawBody)
		require.Equal(t, seeded.domainID, res.Body.Data.Id)
		require.Equal(t, seeded.domain, res.Body.Data.Domain, "the canonical stored name is returned, received: %s", res.RawBody)
	}
}

// TestGetDomainByUnicodeName pins that the Unicode form addresses a domain stored in
// its canonical Punycode form, the same canonicalization createDomain applies. Without
// it a customer could create 'münchen.example.com' and never fetch it back by the name
// they typed.
func TestGetDomainByUnicodeName(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	// Unique parent keeps parallel runs from colliding on the shared database.
	parent := randomDomain()
	canonical := "xn--mnchen-3ya." + parent

	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.Domain = canonical
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: "münchen." + parent,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, seeded.domainID, res.Body.Data.Id)
	require.Equal(t, canonical, res.Body.Data.Domain, "the canonical stored name is returned, received: %s", res.RawBody)
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

// TestGetDomainVerifiedWithUnreadableRouting pins the shape a proxied, flattened, or apex
// domain reports: it routes through a record the worker cannot read back, so only its TXT
// record reports verified while it serves. Per record rather than per domain, since that is
// what tells a caller which one to go fix.
func TestGetDomainVerifiedWithUnreadableRouting(t *testing.T) {
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
	require.Len(t, res.Body.Data.DnsRecords, 2, "received: %s", res.RawBody)
	require.False(t, res.Body.Data.DnsRecords[0].Verified,
		"the routing record could not be read back, received: %s", res.RawBody)
	require.True(t, res.Body.Data.DnsRecords[1].Verified,
		"ownership was proven through TXT, received: %s", res.RawBody)
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
	require.Nil(t, res.Body.Data.VerificationError)
	require.Nil(t, res.Body.Data.UpdatedAt)
	require.Nil(t, res.Body.Data.DomainConnect)
	require.NotContains(t, res.RawBody, "lastCheckedAt", "unset optional fields must be absent, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, "verificationError", "unset optional fields must be absent, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, "domainConnect", "unset optional fields must be absent, received: %s", res.RawBody)
}

func TestGetDomainReportsDomainConnect(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.DomainConnectProvider = "Cloudflare"
		req.DomainConnectURL = "https://dash.cloudflare.com/domainconnect?domain=acme.com"
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body.Data.DomainConnect, "received: %s", res.RawBody)
	require.Equal(t, "Cloudflare", res.Body.Data.DomainConnect.Provider)
	require.Equal(t, "https://dash.cloudflare.com/domainconnect?domain=acme.com", res.Body.Data.DomainConnect.Url)
}

func TestGetDomainOmitsHalfFilledDomainConnect(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	testCases := []struct {
		name     string
		provider string
		url      string
	}{
		{name: "provider without url", provider: "Cloudflare", url: ""},
		{name: "url without provider", provider: "", url: "https://dash.cloudflare.com/domainconnect"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
				req.DomainConnectProvider = tc.provider
				req.DomainConnectURL = tc.url
			})
			rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
				Domain: seeded.domain,
			})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Nil(t, res.Body.Data.DomainConnect, "received: %s", res.RawBody)
		})
	}
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
