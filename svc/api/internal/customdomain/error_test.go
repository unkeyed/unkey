package customdomain_test

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/dns/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
)

// asCtrlError mirrors svc/ctrl/internal/gatefault.ConnectWith, which this package
// cannot import across the internal boundary. Keep the two in step: the whole
// reflection contract rests on ctrl sending exactly the gate's public message.
func asCtrlError(code connect.Code, gateErr error) error {
	return connect.NewError(code, errors.New(fault.UserFacingMessage(gateErr)))
}

// Feed each gate outcome through the same conversion ctrl performs, then back
// through the mapper, and assert the caller sees what the gate said. This is the
// round trip that has to hold: it is what lets the handler's local check and
// ctrl's authoritative re-check report a rejection identically.
func TestMapCtrlErrorRoundTripsGateOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		connectCode connect.Code
		gateErr     error
		wantCode    codes.URN
	}{
		{
			name:        "duplicate",
			connectCode: connect.CodeAlreadyExists,
			gateErr:     domaingate.AlreadyAttached("api.acme.com"),
			wantCode:    codes.Data.Domain.Duplicate.URN(),
		},
		{
			name:        "allowance exhausted",
			connectCode: connect.CodeResourceExhausted,
			gateErr:     domaingate.CheckAllowance(1, 1),
			wantCode:    codes.Limits.CustomDomain.Exceeded.URN(),
		},
		{
			name:        "invalid domain",
			connectCode: connect.CodeInvalidArgument,
			gateErr:     domaingate.CheckDomain("not a domain"),
			wantCode:    codes.App.Validation.InvalidInput.URN(),
		},
		{
			name:        "limits not configured",
			connectCode: connect.CodeFailedPrecondition,
			gateErr:     domaingate.LimitsNotConfigured("ws_1234abcd"),
			wantCode:    codes.App.Internal.ServiceUnavailable.URN(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, tc.gateErr, "the gate must reject this input")

			mapped := customdomain.MapCtrlError(
				asCtrlError(tc.connectCode, tc.gateErr),
				"create custom domain",
			)
			require.Error(t, mapped)

			got, ok := fault.GetCode(mapped)
			require.True(t, ok, "expected a coded fault")
			require.Equal(t, tc.wantCode, got)

			// The wording survives the round trip, so the handler's direct return and
			// ctrl's rejection are indistinguishable to the caller.
			require.Equal(t, fault.UserFacingMessage(tc.gateErr), fault.UserFacingMessage(mapped))

			// gatefault carries only the public message, so the gate's internal detail
			// must not have travelled with it.
			require.NotContains(t, mapped.Error(), "does not match the FQDN pattern")
			require.NotContains(t, mapped.Error(), "has no limits row")
		})
	}
}

// Codes ctrl does not raise through a gate carry no public-message guarantee, so
// they must not be reflected. ctrl's own assert failures are the case that matters:
// they name internal request fields, and they are why those asserts return Internal
// rather than InvalidArgument.
func TestMapCtrlErrorDoesNotReflectUnmappedCodes(t *testing.T) {
	t.Parallel()

	for _, err := range []error{
		connect.NewError(connect.CodeInternal, errors.New("workspace_id is required")),
		connect.NewError(connect.CodeUnavailable, errors.New("dial tcp 10.0.0.1:8080: connect: connection refused")),
		connect.NewError(connect.CodeInternal, errors.New("failed to count custom domains: table is gone")),
		fmt.Errorf("not a connect error at all"),
	} {
		mapped := customdomain.MapCtrlError(err, "create custom domain")
		require.Error(t, mapped)

		got, ok := fault.GetCode(mapped)
		require.True(t, ok)
		require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), got)
		require.Equal(t, "Failed to create custom domain.", fault.UserFacingMessage(mapped))
	}
}
