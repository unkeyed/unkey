// Package gatefault converts a fault raised by the shared precondition and
// validation packages into ctrl's error surfaces.
package gatefault

import (
	"errors"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Connect converts a fault into a connect FailedPrecondition error,
// or nil when err is nil. For ctrl RPC services.
func Connect(err error) error {
	return ConnectWith(connect.CodeFailedPrecondition, err)
}

// ConnectWith is [Connect] for faults that are not all preconditions, so the
// caller picks the matching connect code. Returns nil when err is nil.
func ConnectWith(code connect.Code, err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(code, errors.New(fault.UserFacingMessage(err)))
}

// Terminal converts a fault into a restate terminal error (400), or
// nil when err is nil. For ctrl Restate workers.
func Terminal(err error) error {
	if err == nil {
		return nil
	}
	return restate.TerminalError(errors.New(fault.UserFacingMessage(err)), 400)
}
