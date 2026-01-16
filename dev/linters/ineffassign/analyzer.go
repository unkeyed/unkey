// Package ineffassign provides an ineffassign analyzer for use with nogo.
package ineffassign

import (
	"golang.org/x/tools/go/analysis"

	"github.com/gordonklaus/ineffassign/pkg/ineffassign"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the ineffassign analyzer wrapped with nolint support.
var Analyzer *analysis.Analyzer = nolint.Wrap(ineffassign.Analyzer)
