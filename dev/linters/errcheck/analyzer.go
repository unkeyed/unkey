// Package errcheck provides an errcheck analyzer for use with nogo.
package errcheck

import (
	"golang.org/x/tools/go/analysis"

	"github.com/kisielk/errcheck/errcheck"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the errcheck analyzer wrapped with nolint support.
// Skips generated files since they are auto-generated and may have unchecked errors.
var Analyzer *analysis.Analyzer = nolint.WrapSkipPatterns(errcheck.Analyzer, "_generated.go")
