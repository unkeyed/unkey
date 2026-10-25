package restateutil

import (
	restate "github.com/restatedev/sdk-go"
)

const notFoundCode = 404

// TerminalNotFound converts a missing-row error into a terminal 404 and returns
// any other error unchanged
//
// Call it inside a [restate.Run] closure. Run retries its closure on a
// non-terminal error, so returning a raw sql.ErrNoRows retries the lookup until
// the invocation budget runs out instead of finishing the workflow
func TerminalNotFound(err error, isNotFound func(error) bool) error {
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return err
	}
	return restate.ToTerminalError(err, restate.WithErrorCode(notFoundCode))
}

// IsNotFoundTerminal reports whether err is the terminal 404 [TerminalNotFound]
// builds, which separates a row that is gone from a real failure after Run
// returns
func IsNotFoundTerminal(err error) bool {
	te := restate.AsTerminalError(err)
	return te != nil && te.Code() == notFoundCode
}
