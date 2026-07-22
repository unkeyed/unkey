// Package gatefault converts a deploygate precondition fault into ctrl's error
// surfaces. It surfaces only fault.UserFacingMessage — never err.Error(), which
// carries internal detail — so callers can't leak it by mistake.
package gatefault

import (
	"errors"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Connect converts a deploygate fault into a connect FailedPrecondition error,
// or nil when err is nil. For ctrl RPC services.
func Connect(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition, errors.New(fault.UserFacingMessage(err)))
}

// Terminal converts a deploygate fault into a restate terminal error (400), or
// nil when err is nil. For ctrl Restate workers.
func Terminal(err error) error {
	if err == nil {
		return nil
	}
	return restate.TerminalError(errors.New(fault.UserFacingMessage(err)), 400)
}
