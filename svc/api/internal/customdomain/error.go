// Package customdomain translates the control plane's custom domain errors into
// the API's fault surface.
package customdomain

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
)

// Every code here is one ctrl raises only through gatefault, so its message is the
// gate's public message and is reflected as-is. Adding a code ctrl can raise some
// other way would publish that producer's internal wording.
var ctrlCodes = map[connect.Code]codes.URN{
	connect.CodeAlreadyExists:      codes.Data.Domain.Duplicate.URN(),
	connect.CodeResourceExhausted:  codes.Limits.CustomDomain.Exceeded.URN(),
	connect.CodeFailedPrecondition: codes.App.Internal.ServiceUnavailable.URN(),
	connect.CodeInvalidArgument:    codes.App.Validation.InvalidInput.URN(),
}

// MapCtrlError converts a ctrl custom domain error into a fault. The handler runs
// the same domaingate checks first, so what lands here is the race: state changed
// between that read and ctrl's re-check. Anything ctrl does not raise through a gate
// falls through to [ctrlclient.HandleError].
func MapCtrlError(err error, action string) error {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return ctrlclient.HandleError(err, action)
	}

	urn, ok := ctrlCodes[connectErr.Code()]
	if !ok {
		return ctrlclient.HandleError(err, action)
	}

	return fault.Wrap(
		err,
		fault.Code(urn),
		fault.Internal("ctrl rejected the request after the handler's checks passed: "+connectErr.Message()),
		fault.Public(connectErr.Message()),
	)
}
