package restateutil

import (
	"database/sql"
	"errors"

	restate "github.com/restatedev/sdk-go"
)

const notFoundCode = 404

// TerminalNotFound converts [sql.ErrNoRows] into a terminal 404 and returns any
// other error unchanged
//
// Call it inside a [restate.Run] closure. Run retries its closure on a
// non-terminal error, so a raw sql.ErrNoRows retries the lookup until Restate's
// retry policy pauses the invocation instead of letting the handler decide
func TerminalNotFound(err error) error {
	if err == nil || !errors.Is(err, sql.ErrNoRows) {
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
