package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

func TestListDomains(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	first := attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusVerified
		req.CnameVerified = true
	})
	second := attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)
	require.Len(t, res.Body.Data, 2, "received: %s", res.RawBody)
	require.False(t, res.Body.Pagination.HasMore)
	require.Nil(t, res.Body.Pagination.Cursor)

	byID := map[string]openapi.Domain{}
	for _, d := range res.Body.Data {
		byID[d.Id] = d
	}

	got := byID[first.ID]
	require.Equal(t, first.Domain, got.Domain)
	require.Equal(t, env.projectID, got.ProjectId)
	require.Equal(t, env.appID, got.AppId)
	require.Equal(t, env.environmentID, got.EnvironmentId)
	require.Equal(t, openapi.DomainStatusVerified, got.Status)
	require.NotZero(t, got.CreatedAt)
	require.True(t, got.DnsRecords[0].Verified, "cname_verified was seeded true, received: %s", res.RawBody)
	require.False(t, got.DnsRecords[1].Verified)

	pending := byID[second.ID]
	require.Equal(t, openapi.DomainStatusPending, pending.Status)
	require.False(t, pending.DnsRecords[0].Verified)
	require.False(t, pending.DnsRecords[1].Verified)
	require.Nil(t, pending.VerificationError)
}

// TestListDomainsStableOrder pins that every seeded domain comes back, in the same
// order every time. Asserting a specific order in Go would be the wrong oracle: ORDER BY
// runs under the column's MySQL collation, which sorts alphabetically with case as a
// tiebreaker rather than by byte. Stability is what callers can rely on.
func TestListDomainsStableOrder(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	seeded := map[string]struct{}{}
	for range 5 {
		seeded[attachDomain(t, h, env, nil).ID] = struct{}{}
	}
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")
	headers := authHeaders(rootKey)

	ids := func() []string {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, makeRequest(env))
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		out := make([]string, 0, len(res.Body.Data))
		for _, d := range res.Body.Data {
			out = append(out, d.Id)
		}
		return out
	}

	first := ids()
	require.Len(t, first, 5, "every seeded domain must be returned")

	returned := map[string]struct{}{}
	for _, id := range first {
		_, duplicate := returned[id]
		require.False(t, duplicate, "domain %s returned twice", id)
		returned[id] = struct{}{}
	}
	require.Equal(t, seeded, returned)

	require.Equal(t, first, ids(), "repeated calls must return the same order")
}

// TestListDomainsPagination walks all pages with a small limit. Each domain must
// appear on exactly one page, and the walk must stop when hasMore is false.
func TestListDomainsPagination(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	total := 5
	for range total {
		attachDomain(t, h, env, nil)
	}
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")
	headers := authHeaders(rootKey)

	seen := map[string]struct{}{}
	cursor := (*string)(nil)
	pages := 0
	for {
		req := makeRequest(env)
		req.Limit = ptr.P(2)
		req.Cursor = cursor

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.LessOrEqual(t, len(res.Body.Data), 2)

		for _, d := range res.Body.Data {
			_, dup := seen[d.Id]
			require.False(t, dup, "domain %s returned on more than one page", d.Id)
			seen[d.Id] = struct{}{}
		}

		pages++
		require.LessOrEqual(t, pages, total+1, "pagination did not terminate")

		if !res.Body.Pagination.HasMore {
			require.Nil(t, res.Body.Pagination.Cursor)
			break
		}
		require.NotNil(t, res.Body.Pagination.Cursor)
		cursor = res.Body.Pagination.Cursor
	}

	require.Len(t, seen, total)
}

// A cursor that does not match a domain is not an error. The query continues
// from the given value and returns the domains with a larger or equal id.
func TestListDomainsUnknownCursor(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	req := makeRequest(env)
	req.Cursor = ptr.P("dom_doesnotexist")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.False(t, res.Body.Pagination.HasMore)
}

// TestListDomainsCursorStaysScoped pins that a cursor borrowed from a different
// environment does not widen the results. The query filters by environment_id,
// so the sibling domain can never appear.
func TestListDomainsCursorStaysScoped(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attachDomain(t, h, env, nil)

	sibling := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: env.workspaceID,
		ProjectID:   env.projectID,
		AppID:       env.appID,
		Slug:        "staging",
		Description: "Staging environment",
	})
	siblingDomain := h.CreateCustomDomain(seed.CreateCustomDomainRequest{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        env.workspaceID,
		ProjectID:          env.projectID,
		AppID:              env.appID,
		EnvironmentID:      sibling.ID,
		Domain:             randomDomain(),
		VerificationStatus: db.CustomDomainsVerificationStatusVerified,
		VerificationToken:  "",
		TargetCname:        "",
		OwnershipVerified:  true,
		CnameVerified:      true,
		VerificationError:  "",
		LastCheckedAt:      0,
	})
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	req := makeRequest(env)
	req.Cursor = ptr.P(siblingDomain.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, siblingDomain.ID, "the sibling environment's domain leaked: %s", res.RawBody)
}

// TestListDomainsEmpty pins that an environment with no domains is a 200 with an
// empty array, not a 404. Callers poll this after createDomain and must be able to
// tell "no domains yet" from "environment does not exist".
func TestListDomainsEmpty(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Empty(t, res.Body.Data, "received: %s", res.RawBody)
}

func TestListDomainsBySlugs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attached := attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project:     env.projectSlug,
		App:         env.appSlug,
		Environment: "production",
		Search:      nil,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1, "received: %s", res.RawBody)
	require.Equal(t, attached.ID, res.Body.Data[0].Id)
}

// TestListDomainsScopedToEnvironment pins that a sibling environment's domains do not
// leak in. Domains are unique per workspace, not per environment, so the filter is the
// only thing keeping them apart.
func TestListDomainsScopedToEnvironment(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	mine := attachDomain(t, h, env, nil)

	sibling := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: env.workspaceID,
		ProjectID:   env.projectID,
		AppID:       env.appID,
		Slug:        "staging",
		Description: "Staging environment",
	})
	siblingDomain := h.CreateCustomDomain(seed.CreateCustomDomainRequest{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        env.workspaceID,
		ProjectID:          env.projectID,
		AppID:              env.appID,
		EnvironmentID:      sibling.ID,
		Domain:             randomDomain(),
		VerificationStatus: db.CustomDomainsVerificationStatusVerified,
		VerificationToken:  "",
		TargetCname:        "",
		OwnershipVerified:  true,
		CnameVerified:      true,
		VerificationError:  "",
		LastCheckedAt:      0,
	})
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1, "received: %s", res.RawBody)
	require.Equal(t, mine.ID, res.Body.Data[0].Id)
	require.NotContains(t, res.RawBody, siblingDomain.ID, "a sibling environment's domain leaked: %s", res.RawBody)
}

// TestListDomainsDnsRecordsPerEntry pins that each entry carries its own records built
// from its own token and target, not another row's.
func TestListDomainsDnsRecordsPerEntry(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	subdomain := attachDomain(t, h, env, nil)
	apexName := "d" + uid.DNS1035(12) + ".com"
	apex := attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.Domain = apexName
	})
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 2, "received: %s", res.RawBody)

	byID := map[string]openapi.Domain{}
	for _, d := range res.Body.Data {
		byID[d.Id] = d
	}

	sub := byID[subdomain.ID]
	require.Len(t, sub.DnsRecords, 2)
	require.Equal(t, openapi.CNAME, sub.DnsRecords[0].Type)
	require.Equal(t, subdomain.Domain, sub.DnsRecords[0].Name)
	require.Equal(t, subdomain.TargetCname, sub.DnsRecords[0].Value)
	require.Equal(t, "_unkey."+subdomain.Domain, sub.DnsRecords[1].Name)
	require.Equal(t, "unkey-domain-verify="+subdomain.VerificationToken, sub.DnsRecords[1].Value)

	// The apex shares the page with a subdomain, so a mapper that resolved apex-ness
	// once for the whole list would hand it a CNAME it cannot hold.
	ap := byID[apex.ID]
	require.Equal(t, openapi.ALIAS, ap.DnsRecords[0].Type, "apex cannot hold a CNAME, received: %s", res.RawBody)
	require.Equal(t, apexName, ap.DnsRecords[0].Name)
	require.Equal(t, apex.TargetCname, ap.DnsRecords[0].Value)
	require.Equal(t, "unkey-domain-verify="+apex.VerificationToken, ap.DnsRecords[1].Value)
}

func TestListDomainsDomainConnectPerEntry(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	withConnect := attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.DomainConnectProvider = "Cloudflare"
		req.DomainConnectURL = "https://dash.cloudflare.com/domainconnect?domain=acme.com"
	})
	withoutConnect := attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 2, "received: %s", res.RawBody)

	byID := map[string]openapi.Domain{}
	for _, d := range res.Body.Data {
		byID[d.Id] = d
	}

	connected := byID[withConnect.ID]
	require.NotNil(t, connected.DomainConnect, "received: %s", res.RawBody)
	require.Equal(t, "Cloudflare", connected.DomainConnect.Provider)
	require.Equal(t, withConnect.DomainConnectURL, connected.DomainConnect.Url)

	require.Nil(t, byID[withoutConnect.ID].DomainConnect, "received: %s", res.RawBody)
}

// A list mixes domains that verified through their routing record with ones that verified
// through TXT because the record could not be read back, so the flag has to be per entry
// and cannot be folded into status.
func TestListDomainsVerifiedWithUnreadableRouting(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	routing := attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusVerified
		req.CnameVerified = true
	})
	ownershipOnly := attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusVerified
		req.OwnershipVerified = true
	})
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	byID := map[string]openapi.Domain{}
	for _, d := range res.Body.Data {
		byID[d.Id] = d
	}

	require.Equal(t, openapi.DomainStatusVerified, byID[routing.ID].Status)
	require.True(t, byID[routing.ID].DnsRecords[0].Verified)

	require.Equal(t, openapi.DomainStatusVerified, byID[ownershipOnly.ID].Status, "received: %s", res.RawBody)
	require.False(t, byID[ownershipOnly.ID].DnsRecords[0].Verified,
		"the routing record could not be read back, received: %s", res.RawBody)
	require.True(t, byID[ownershipOnly.ID].DnsRecords[1].Verified,
		"ownership was proven through TXT, received: %s", res.RawBody)
}

// TestListDomainsSearch covers both halves of the search clause, which matches on id
// as well as name, and the case-insensitivity the spec promises.
func TestListDomainsSearch(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	match := attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.Domain = "d" + uid.DNS1035(10) + ".searchme.example.com"
	})
	other := attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")
	headers := authHeaders(rootKey)

	testCases := []struct {
		name      string
		search    string
		wantMatch bool
	}{
		{name: "by name fragment", search: "searchme", wantMatch: true},
		{name: "by name fragment uppercased", search: "SEARCHME", wantMatch: true},
		{name: "by id", search: match.ID, wantMatch: true},
		{name: "no match", search: "nothing-has-this-substring", wantMatch: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := makeRequest(env)
			req.Search = ptr.P(tc.search)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

			if !tc.wantMatch {
				require.Empty(t, res.Body.Data, "expected no matches, received: %s", res.RawBody)
				return
			}
			require.Len(t, res.Body.Data, 1, "received: %s", res.RawBody)
			require.Equal(t, match.ID, res.Body.Data[0].Id)
			require.NotContains(t, res.RawBody, other.ID, "search returned a non-matching domain: %s", res.RawBody)
		})
	}
}

func TestListDomainsFailedReportsError(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	const verificationError = "domain verification timed out after 24 hours"
	env := seedEnvironment(t, h)
	attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusFailed
		req.VerificationError = verificationError
		req.LastCheckedAt = time.Now().UnixMilli()
	})
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1, "received: %s", res.RawBody)
	require.Equal(t, openapi.DomainStatusFailed, res.Body.Data[0].Status)
	require.NotNil(t, res.Body.Data[0].VerificationError, "a failed domain must say why, received: %s", res.RawBody)
	require.Equal(t, verificationError, *res.Body.Data[0].VerificationError)
}

func TestListDomainsWithSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID, "environment."+env.environmentID+".read_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1, "received: %s", res.RawBody)
}
