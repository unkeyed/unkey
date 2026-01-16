// Package unused provides an unused analyzer for use with nogo.
//
// Unused identifies code that is never referenced: unexported functions,
// types, constants, and variables that have no callers. This dead code adds
// maintenance burden, confuses readers, and bloats binaries.
//
// This analyzer comes from the staticcheck suite (honnef.co/go/tools/unused)
// and performs whole-program analysis to find truly unreachable code, unlike
// the compiler's basic unused variable checks.
//
// # Why This Matters
//
// Dead code accumulates during refactoring—a function is replaced but the old
// version is never deleted. Over time, these remnants make the codebase harder
// to navigate and understand. Unused catches what the compiler misses.
package unused

import (
	"golang.org/x/tools/go/analysis"

	"honnef.co/go/tools/unused"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the unused analyzer wrapped with nolint support.
// unused.Analyzer is a *lint.Analyzer, so we access its inner Analyzer field.
var Analyzer *analysis.Analyzer = nolint.Wrap(unused.Analyzer.Analyzer)
