// Package exhaustive provides an exhaustive analyzer for use with nogo.
//
// Exhaustive ensures that switch statements on enum types handle all possible
// values. When a new enum value is added, the compiler won't catch missing cases,
// but exhaustive will—preventing runtime panics or incorrect default behavior.
//
// # Why This Matters
//
// Adding a new status like StatusPending to an existing enum can silently break
// every switch that handles that type. Without exhaustive checks, the new value
// falls through to a default case (if present) or causes undefined behavior.
// This linter makes enum additions a compile-time error rather than a runtime bug.
package exhaustive

import (
	"golang.org/x/tools/go/analysis"

	"github.com/nishanths/exhaustive"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the exhaustive analyzer wrapped with nolint support.
var Analyzer *analysis.Analyzer = nolint.Wrap(exhaustive.Analyzer)
