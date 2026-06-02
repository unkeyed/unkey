package publicerr

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

// TestWireCodes_CoversEveryCatalogEntry asserts that every public code
// in catalog has a wireCodes entry. A new catalog entry without a
// gRPC/Connect mapping would silently fall back to INTERNAL — this
// test fails CI on omission.
func TestWireCodes_CoversEveryCatalogEntry(t *testing.T) {
	t.Parallel()

	for code := range catalog {
		_, ok := wireCodes[code]
		require.Truef(t, ok,
			"public code %q exists in catalog but has no wireCodes "+
				"entry — add a {grpc, connect} mapping in "+
				"svc/frontline/internal/publicerr/wire.go",
			code)
	}
}

// TestWireCodes_NoOrphans asserts wireCodes has no entries without a
// matching catalog entry. Catches typos and dead entries.
func TestWireCodes_NoOrphans(t *testing.T) {
	t.Parallel()

	for code := range wireCodes {
		_, ok := catalog[code]
		require.Truef(t, ok,
			"wireCodes entry %q has no matching catalog entry", code)
	}
}

// TestConnectHTTPStatus_CoversEveryConnectCode asserts every Connect
// code referenced by wireCodes has an HTTP status mapping. Without
// this, an unmapped Connect code would default to 500 silently.
func TestConnectHTTPStatus_CoversEveryConnectCode(t *testing.T) {
	t.Parallel()

	for code, w := range wireCodes {
		_, ok := connectHTTPStatus[w.connect]
		require.Truef(t, ok,
			"public code %q maps to connect code %q which has no "+
				"connectHTTPStatus entry", code, w.connect)
	}
}

// TestProblem_GRPCStatusMapping locks the gRPC code that customers
// will see for representative public codes. Customer SDKs branch on
// these ints; a silent change would break integrations.
func TestProblem_GRPCStatusMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		urn      codes.URN
		wantGRPC int
	}{
		{codes.Frontline.Auth.InvalidKey.URN(), grpcUnauthenticated},
		{codes.Frontline.Auth.MissingCredentials.URN(), grpcUnauthenticated},
		{codes.Frontline.Auth.InsufficientPermissions.URN(), grpcPermissionDenied},
		{codes.Frontline.Auth.RateLimited.URN(), grpcResourceExhausted},
		{codes.Frontline.Firewall.Denied.URN(), grpcPermissionDenied},
		{codes.Frontline.OpenApi.InvalidRequest.URN(), grpcInvalidArgument},
		{codes.User.BadRequest.RequestBodyTooLarge.URN(), grpcResourceExhausted},
		{codes.User.BadRequest.ClientClosedRequest.URN(), grpcCanceled},
		{codes.Frontline.Routing.ConfigNotFoundForCustomDomain.URN(), grpcNotFound},
		{codes.Frontline.Internal.InternalServerError.URN(), grpcInternal},
		{codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(), grpcUnavailable},
		{codes.Frontline.Proxy.DialTimeout.URN(), grpcDeadlineExceeded},
	}

	for _, tc := range cases {
		t.Run(string(tc.urn), func(t *testing.T) {
			t.Parallel()
			p := ProblemFor(tc.urn)
			require.Equal(t, tc.wantGRPC, p.GRPCStatus())
		})
	}
}

// TestProblem_ConnectCodeMapping locks the Connect code string that
// customers will see for representative public codes.
func TestProblem_ConnectCodeMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		urn         codes.URN
		wantConnect string
		wantStatus  int
	}{
		{codes.Frontline.Auth.InvalidKey.URN(), "unauthenticated", 401},
		{codes.Frontline.Auth.InsufficientPermissions.URN(), "permission_denied", 403},
		{codes.Frontline.Auth.RateLimited.URN(), "resource_exhausted", 429},
		{codes.Frontline.OpenApi.InvalidRequest.URN(), "invalid_argument", 400},
		{codes.User.BadRequest.ClientClosedRequest.URN(), "canceled", 408},
		{codes.Frontline.Routing.ConfigNotFoundForCustomDomain.URN(), "not_found", 404},
		{codes.Frontline.Internal.InternalServerError.URN(), "internal", 500},
		{codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(), "unavailable", 503},
		{codes.Frontline.Proxy.DialTimeout.URN(), "deadline_exceeded", 408},
	}

	for _, tc := range cases {
		t.Run(string(tc.urn), func(t *testing.T) {
			t.Parallel()
			p := ProblemFor(tc.urn)
			require.Equal(t, tc.wantConnect, p.ConnectCode())
			require.Equal(t, tc.wantStatus, p.ConnectHTTPStatus().Int())
		})
	}
}

// TestProblem_UnknownCodeDefaults asserts the fail-safe: a Problem
// with an unknown Code defaults to gRPC INTERNAL / Connect "internal".
func TestProblem_UnknownCodeDefaults(t *testing.T) {
	t.Parallel()

	p := Problem{Code: "made_up_code"}
	require.Equal(t, grpcInternal, p.GRPCStatus())
	require.Equal(t, "internal", p.ConnectCode())
	require.Equal(t, 500, p.ConnectHTTPStatus().Int())
}
