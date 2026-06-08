// Package restateutil holds small helpers shared by Restate handlers.
package restateutil

import (
	"time"

	restate "github.com/restatedev/sdk-go"
)

// Now reads the wall clock inside a journaled step so the value is recorded on
// the first execution and reused verbatim on every replay. Reading time.Now()
// directly in handler code would re-evaluate on each replay and drift any
// downstream step that depends on it. Callers convert the result to whatever
// unit they need (Unix, UnixMilli, Add, ...); those conversions are
// deterministic and safe outside the journaled step.
func Now(ctx restate.Context) (time.Time, error) {
	return restate.Run(ctx, func(restate.RunContext) (time.Time, error) {
		return time.Now(), nil
	}, restate.WithName("time.Now()"))
}
