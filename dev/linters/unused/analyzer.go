// Package unused provides an unused analyzer for use with nogo.
package unused

import (
	"golang.org/x/tools/go/analysis"

	"honnef.co/go/tools/unused"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the unused analyzer wrapped with nolint support.
// unused.Analyzer is a *lint.Analyzer, so we access its inner Analyzer field.
var Analyzer *analysis.Analyzer = nolint.Wrap(unused.Analyzer.Analyzer)
