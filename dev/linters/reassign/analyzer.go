// Package reassign provides a reassign analyzer for use with nogo.
package reassign

import (
	"golang.org/x/tools/go/analysis"

	"github.com/curioswitch/go-reassign"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the reassign analyzer wrapped with nolint support.
var Analyzer *analysis.Analyzer = nolint.Wrap(reassign.NewAnalyzer())
