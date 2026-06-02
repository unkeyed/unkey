package publicerr

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

// TestCatalog_Golden is a full snapshot of the public catalog. It
// locks every customer-visible field of every public code so a change
// to any of them shows up as a deliberate test diff in review.
//
// Adding a new public code: extend wantCatalog. Removing one: same.
// Tweaking wording: the test will fail with a diff that the reviewer
// must approve. This is deliberately friction — these strings are part
// of the public contract.
//
// Do NOT relax this test to "ignore" fields. If a field becomes
// genuinely optional or owned by another file, remove it from the
// snapshot explicitly with a comment.
func TestCatalog_Golden(t *testing.T) {
	t.Parallel()

	wantCatalog := map[string]catalogEntry{
		// ── Actionable 4xx ────────────────────────────────────────
		"missing_credentials": {
			status:     codes.StatusUnauthorized,
			title:      "Unauthorized",
			detail:     "Authentication required. Provide a valid API key.",
			typeURL:    "https://unkey.com/docs/errors/missing-credentials",
			retryAfter: nil,
		},
		"invalid_key": {
			status:     codes.StatusUnauthorized,
			title:      "Unauthorized",
			detail:     "The provided API key is invalid, disabled, or expired.",
			typeURL:    "https://unkey.com/docs/errors/invalid-key",
			retryAfter: nil,
		},
		"insufficient_permissions": {
			status:     codes.StatusForbidden,
			title:      "Forbidden",
			detail:     "The API key does not have the required permissions.",
			typeURL:    "https://unkey.com/docs/errors/insufficient-permissions",
			retryAfter: nil,
		},
		"rate_limited": {
			status:     codes.StatusTooManyRequests,
			title:      "Too Many Requests",
			detail:     "Rate limit exceeded. Retry later.",
			typeURL:    "https://unkey.com/docs/errors/rate-limited",
			retryAfter: intPtr(30),
		},
		"firewall_denied": {
			status:     codes.StatusForbidden,
			title:      "Forbidden",
			detail:     "The request was blocked by a firewall rule.",
			typeURL:    "https://unkey.com/docs/errors/firewall-denied",
			retryAfter: nil,
		},
		"openapi_invalid_request": {
			status:     codes.StatusBadRequest,
			title:      "Bad Request",
			detail:     "The request does not match the API specification.",
			typeURL:    "https://unkey.com/docs/errors/openapi-invalid-request",
			retryAfter: nil,
		},
		"request_body_too_large": {
			status:     codes.StatusRequestEntityTooLarge,
			title:      "Payload Too Large",
			detail:     "The request body exceeds the maximum allowed size.",
			typeURL:    "https://unkey.com/docs/errors/request-body-too-large",
			retryAfter: nil,
		},
		"client_closed_request": {
			status:     codes.StatusClientClosedRequest,
			title:      "Client Closed Request",
			detail:     "The client closed the connection before the request completed.",
			typeURL:    "https://unkey.com/docs/errors/client-closed-request",
			retryAfter: nil,
		},

		// ── Generic 4xx classes ───────────────────────────────────
		"bad_request": {
			status:     codes.StatusBadRequest,
			title:      "Bad Request",
			detail:     "The request is invalid.",
			typeURL:    "https://unkey.com/docs/errors/bad-request",
			retryAfter: nil,
		},
		"unauthorized": {
			status:     codes.StatusUnauthorized,
			title:      "Unauthorized",
			detail:     "Authentication is required.",
			typeURL:    "https://unkey.com/docs/errors/unauthorized",
			retryAfter: nil,
		},
		"forbidden": {
			status:     codes.StatusForbidden,
			title:      "Forbidden",
			detail:     "Access denied.",
			typeURL:    "https://unkey.com/docs/errors/forbidden",
			retryAfter: nil,
		},
		"not_found": {
			status:     codes.StatusNotFound,
			title:      "Not Found",
			detail:     "No deployment is configured for this request.",
			typeURL:    "https://unkey.com/docs/errors/not-found",
			retryAfter: nil,
		},
		"too_many_requests": {
			status:     codes.StatusTooManyRequests,
			title:      "Too Many Requests",
			detail:     "Rate limit exceeded. Retry later.",
			typeURL:    "https://unkey.com/docs/errors/too-many-requests",
			retryAfter: intPtr(30),
		},

		// ── 5xx classes ───────────────────────────────────────────
		"internal_server_error": {
			status:     codes.StatusInternalServerError,
			title:      "Internal Server Error",
			detail:     "An unexpected error occurred. Retry later.",
			typeURL:    "https://unkey.com/docs/errors/internal-server-error",
			retryAfter: nil,
		},
		"bad_gateway": {
			status:     codes.StatusBadGateway,
			title:      "Bad Gateway",
			detail:     "Unable to connect. Retry in a few moments.",
			typeURL:    "https://unkey.com/docs/errors/bad-gateway",
			retryAfter: intPtr(15),
		},
		"service_unavailable": {
			status:     codes.StatusServiceUnavailable,
			title:      "Service Unavailable",
			detail:     "The service is temporarily unavailable. Retry later.",
			typeURL:    "https://unkey.com/docs/errors/service-unavailable",
			retryAfter: intPtr(30),
		},
		"gateway_timeout": {
			status:     codes.StatusGatewayTimeout,
			title:      "Gateway Timeout",
			detail:     "The request took too long to process. Retry later.",
			typeURL:    "https://unkey.com/docs/errors/gateway-timeout",
			retryAfter: intPtr(15),
		},
	}

	require.Equal(t, len(wantCatalog), len(catalog),
		"catalog size changed — update wantCatalog")

	for code, want := range wantCatalog {
		got, ok := catalog[code]
		require.Truef(t, ok, "catalog missing public code %q", code)
		require.Equalf(t, want, got,
			"catalog[%q] has drifted from golden snapshot", code)
	}
}

// TestCatalog_EveryEntryReachable asserts every catalog entry is
// reachable from at least one URN via publicCodeFor. Catches dead
// entries that would mislead the next reader: a code in the catalog
// that no URN actually maps to looks supported but never fires.
//
// The set of "known URNs" is the union of every codes.Frontline URN
// (walked via reflection) and the cross-system URNs frontline
// surfaces. If a future catalog entry is reachable only from a URN
// outside that set, add it to crossSystem below.
func TestCatalog_EveryEntryReachable(t *testing.T) {
	t.Parallel()

	urns := codes.CollectURNs(codes.Frontline)
	urns = append(urns,
		// Cross-system URNs frontline currently re-emits.
		codes.User.BadRequest.ClientClosedRequest.URN(),
		codes.User.BadRequest.RequestTimeout.URN(),
		codes.User.BadRequest.RequestBodyTooLarge.URN(),
		codes.User.BadRequest.RequestBodyUnreadable.URN(),
		codes.Auth.Authentication.Missing.URN(),
		codes.Auth.Authentication.Malformed.URN(),
		codes.App.Validation.InvalidInput.URN(),
		// Cross-system URN that proves "too_many_requests" is
		// reachable from a real URN's category default and not dead.
		codes.User.TooManyRequests.QueryQuotaExceeded.URN(),
	)
	// Synthetic unknown URN — reaches the internal_server_error
	// fallback in ProblemFor. Without this entry the catalog would
	// flag internal_server_error as unreachable if no real URN
	// happens to map to it directly.
	reached := map[string]bool{"internal_server_error": true}
	for _, u := range urns {
		reached[publicCodeFor(u)] = true
	}

	// defensiveEntries are catalog codes that exist as category
	// fallbacks rather than mappings from currently-emitted URNs.
	// They are kept around so a future cross-system URN in the
	// matching category surfaces as a sensible HTTP class instead of
	// a generic 500. Listed explicitly so an accidental new entry
	// (typo, copy-paste) is still flagged.
	defensiveEntries := map[string]string{
		// CategoryForbidden URNs all have explicit overrides today
		// (firewall_denied, insufficient_permissions), so the
		// category default never fires from frontline-emitted URNs.
		// Kept as the generic 403 for future cross-system URNs.
		"forbidden": "CategoryForbidden generic 403 fallback",
	}

	for code := range catalog {
		if reached[code] || defensiveEntries[code] != "" {
			continue
		}
		t.Errorf("catalog entry %q is unreachable — no known URN "+
			"maps to it. Either remove the entry, add a URN that "+
			"uses it, or add it to defensiveEntries with justification.",
			code)
	}
}
