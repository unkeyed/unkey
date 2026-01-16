// Package exhaustive provides an exhaustive analyzer for use with nogo.
package exhaustive

import (
	"golang.org/x/tools/go/analysis"

	"github.com/nishanths/exhaustive"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the exhaustive analyzer wrapped with nolint support.
var Analyzer *analysis.Analyzer = nolint.Wrap(exhaustive.Analyzer)
