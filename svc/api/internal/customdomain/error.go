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

// ctrlCodes maps the connect codes ctrl returns for a domaingate outcome onto the
// error code that decides the HTTP status. The status is all this layer decides;
// the message travels with the error.
var ctrlCodes = map[connect.Code]codes.URN{
	connect.CodeAlreadyExists:      codes.Data.Domain.Duplicate.URN(),
	connect.CodeResourceExhausted:  codes.Limits.CustomDomain.Exceeded.URN(),
	connect.CodeInvalidArgument:    codes.App.Validation.InvalidInput.URN(),
	connect.CodeFailedPrecondition: codes.App.Internal.ServiceUnavailable.URN(),
	connect.CodeNotFound:           codes.Data.Environment.NotFound.URN(),
}

// MapCtrlError converts a ctrl custom domain error into a fault. The handler runs
// the same domaingate checks first, so what lands here is the race: state changed
// between the handler's read and ctrl's authoritative re-check.
//
// The message is reflected, not rewritten. ctrl builds these errors with gatefault,
// whose message is exactly the gate's public message, so the wording has one source
// and cannot drift from what the handler would have said. Codes ctrl does not use
// for a gate outcome carry no such guarantee and fall through to
// [ctrlclient.HandleError].
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
