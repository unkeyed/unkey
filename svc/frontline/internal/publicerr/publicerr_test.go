package publicerr

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

// TestProblemFor_EveryFrontlineURNResolvesToCatalogEntry walks all
// codes.Frontline URNs and asserts each resolves to a catalog entry.
// A new frontline URN whose category isn't in the catalog would
// silently fall through to internal_server_error — this test makes
// the omission fail at CI time.
func TestProblemFor_EveryFrontlineURNResolvesToCatalogEntry(t *testing.T) {
	t.Parallel()

	for _, urn := range codes.CollectURNs(codes.Frontline) {
		code := publicCodeFor(urn)
		_, ok := catalog[code]
		require.Truef(t, ok,
			"frontline URN %q maps to public code %q which is not in "+
				"the publicerr catalog — add an entry to "+
				"svc/frontline/internal/publicerr/publicerr.go",
			urn, code)
	}
}

// TestProblemFor_CrossSystemURNsResolveToCatalogEntry covers the
// non-frontline URNs that frontline surfaces (cross-system errors
// like User.BadRequest.*, Auth.Authentication.*, App.Validation.*).
// Their categories aren't always HTTP-like, so they rely on explicit
// remaps in publicCodeFor.
func TestProblemFor_CrossSystemURNsResolveToCatalogEntry(t *testing.T) {
	t.Parallel()

	crossSystem := []codes.URN{
		codes.User.BadRequest.ClientClosedRequest.URN(),
		codes.User.BadRequest.RequestTimeout.URN(),
		codes.User.BadRequest.RequestBodyTooLarge.URN(),
		codes.User.BadRequest.RequestBodyUnreadable.URN(),
		codes.Auth.Authentication.Missing.URN(),
		codes.Auth.Authentication.Malformed.URN(),
		codes.App.Validation.InvalidInput.URN(),
	}

	for _, urn := range crossSystem {
		code := publicCodeFor(urn)
		_, ok := catalog[code]
		require.Truef(t, ok,
			"cross-system URN %q maps to public code %q which is not "+
				"in the publicerr catalog", urn, code)
	}
}

// TestCatalog_NoForbiddenSubstrings asserts the customer-facing
// surface (Title, Detail, TypeURL) never contains words that would
// leak internal frontline topology or mechanism. The forbidden list
// is shared with the proxy classification test via
// ForbiddenInPublicMessages so both paths enforce the same contract.
func TestCatalog_NoForbiddenSubstrings(t *testing.T) {
	t.Parallel()

	for code, e := range catalog {
		fields := map[string]string{
			"Code":    code,
			"Title":   e.title,
			"Detail":  e.detail,
			"TypeURL": e.typeURL,
		}
		for fieldName, value := range fields {
			if bad := ContainsForbidden(value); bad != "" {
				t.Errorf("catalog[%q].%s contains forbidden substring %q "+
					"(leaks internal frontline topology/mechanism)",
					code, fieldName, bad)
			}
		}
	}
}

// TestProblemFor_UnknownURNDefaultsToInternalServerError locks in
// the fail-safe: an unmapped URN must surface as a generic 500
// rather than leaking the URN string or crashing.
func TestProblemFor_UnknownURNDefaultsToInternalServerError(t *testing.T) {
	t.Parallel()

	p := ProblemFor("err:made:up:nonexistent")
	require.Equal(t, "internal_server_error", p.Code)
	require.Equal(t, 500, p.Status.Int())
}

// TestProblemFor_KnownActionable4xxPassesThroughSpecific verifies the
// allowlist actually exposes the specific URN part as a distinct
// public code, not the collapsed category.
func TestProblemFor_KnownActionable4xxPassesThroughSpecific(t *testing.T) {
	t.Parallel()

	cases := []struct {
		urn      codes.URN
		wantCode string
	}{
		{codes.Frontline.Auth.MissingCredentials.URN(), "missing_credentials"},
		{codes.Frontline.Auth.InvalidKey.URN(), "invalid_key"},
		{codes.Frontline.Auth.InsufficientPermissions.URN(), "insufficient_permissions"},
		{codes.Frontline.Auth.RateLimited.URN(), "rate_limited"},
		{codes.Frontline.Firewall.Denied.URN(), "firewall_denied"},
		{codes.Frontline.OpenApi.InvalidRequest.URN(), "openapi_invalid_request"},
		{codes.User.BadRequest.RequestBodyTooLarge.URN(), "request_body_too_large"},
	}

	for _, tc := range cases {
		t.Run(string(tc.urn), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.wantCode, ProblemFor(tc.urn).Code)
		})
	}
}

// TestProblemFor_StatusOwnedByPublicCode verifies the HTTP status
// comes from the public code's catalog entry, not the internal URN.
// This matters for remapped URNs whose native HTTP class differs
// from the customer-facing status — e.g. an internal
// User.BadRequest.RequestTimeout (408) surfaces as gateway_timeout
// (504) because frontline is a gateway.
func TestProblemFor_StatusOwnedByPublicCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		urn        codes.URN
		wantStatus int
	}{
		{codes.Frontline.Auth.InvalidKey.URN(), 401},
		{codes.Frontline.Auth.InsufficientPermissions.URN(), 403},
		{codes.Frontline.Auth.RateLimited.URN(), 429},
		{codes.Frontline.Firewall.Denied.URN(), 403},
		{codes.Frontline.OpenApi.InvalidRequest.URN(), 400},
		{codes.User.BadRequest.RequestBodyTooLarge.URN(), 413},
		{codes.User.BadRequest.ClientClosedRequest.URN(), 499},
		// Remap: internal URN is 408, public code is gateway_timeout (504).
		{codes.User.BadRequest.RequestTimeout.URN(), 504},
		{codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(), 503},
		{codes.Frontline.Proxy.DialTimeout.URN(), 504},
		{codes.Frontline.Proxy.UpstreamConnectionReset.URN(), 502},
		{codes.Frontline.Internal.ConfigLoadFailed.URN(), 500},
		{codes.Frontline.Routing.NoRunningInstances.URN(), 503},
		{codes.Frontline.Routing.ConfigNotFoundForCustomDomain.URN(), 404},
	}

	for _, tc := range cases {
		t.Run(string(tc.urn), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.wantStatus, ProblemFor(tc.urn).Status.Int())
		})
	}
}

// TestProblemFor_RetryHints verifies the catalog assigns retry_after so
// the observability middleware can emit the standard HTTP Retry-After
// header. RetryAfter is header-only — it is deliberately not in the
// response body (see publicerr package doc).
func TestProblemFor_RetryHints(t *testing.T) {
	t.Parallel()

	cases := []struct {
		urn             codes.URN
		wantHasRetryAft bool
	}{
		// 4xx caller-fixable — no backoff hint.
		{codes.Frontline.Auth.InvalidKey.URN(), false},
		{codes.Frontline.Auth.MissingCredentials.URN(), false},
		{codes.Frontline.Auth.InsufficientPermissions.URN(), false},
		{codes.Frontline.OpenApi.InvalidRequest.URN(), false},
		{codes.User.BadRequest.RequestBodyTooLarge.URN(), false},
		// Rate limits — backoff hint set.
		{codes.Frontline.Auth.RateLimited.URN(), true},
		// Owner-config faults — no backoff hint.
		{codes.Frontline.Firewall.Denied.URN(), false},
		{codes.Frontline.Routing.ConfigNotFoundForCustomDomain.URN(), false},
		// 5xx transient — backoff hint set.
		{codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(), true},
		{codes.Frontline.Proxy.DialTimeout.URN(), true},
		{codes.Frontline.Proxy.UpstreamConnectionReset.URN(), true},
		{codes.Frontline.Internal.InternalServerError.URN(), false},
	}

	for _, tc := range cases {
		t.Run(string(tc.urn), func(t *testing.T) {
			t.Parallel()

			p := ProblemFor(tc.urn)
			if tc.wantHasRetryAft {
				require.NotNil(t, p.RetryAfter, "RetryAfter must be set")
				require.Greater(t, *p.RetryAfter, 0, "RetryAfter must be positive")
			} else {
				require.Nil(t, p.RetryAfter, "RetryAfter must be nil")
			}
		})
	}
}

// TestProblemFor_5xxFrontlineURNsCollapseToCategory verifies the 5xx
// internal URNs do not leak their specific part — they collapse to the
// HTTP-class category public code.
func TestProblemFor_5xxFrontlineURNsCollapseToCategory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		urn      codes.URN
		wantCode string
	}{
		{codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(), "service_unavailable"},
		{codes.Frontline.Proxy.PeerFrontlineDNSTimeout.URN(), "gateway_timeout"},
		{codes.Frontline.Proxy.PeerFrontlineConnectionReset.URN(), "bad_gateway"},
		{codes.Frontline.Proxy.UpstreamConnectionRefused.URN(), "service_unavailable"},
		{codes.Frontline.Proxy.DialTimeout.URN(), "gateway_timeout"},
		{codes.Frontline.Proxy.ProxyErrorUnclassified.URN(), "bad_gateway"},
		{codes.Frontline.Internal.ConfigLoadFailed.URN(), "internal_server_error"},
		{codes.Frontline.Internal.InstanceLoadFailed.URN(), "internal_server_error"},
		{codes.Frontline.Internal.InvalidConfiguration.URN(), "internal_server_error"},
		{codes.Frontline.Routing.NoRunningInstances.URN(), "service_unavailable"},
		{codes.Frontline.Routing.NoReachableRegion.URN(), "service_unavailable"},
		{codes.Frontline.Routing.DeploymentSelectionFailed.URN(), "internal_server_error"},
	}

	for _, tc := range cases {
		t.Run(string(tc.urn), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.wantCode, ProblemFor(tc.urn).Code)
		})
	}
}
