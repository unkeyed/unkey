// Package ineffassign provides an ineffassign analyzer for use with nogo.
//
// Ineffassign detects assignments to variables that are never subsequently read.
// These are either dead code that should be removed or bugs where the developer
// intended to use the assigned value but made a typo or logic error.
//
// # Why This Matters
//
// Consider:
//
//	err = validateInput(req)
//	err = processRequest(req)  // shadows previous error without checking it
//	if err != nil { ... }
//
// The first error is assigned but never checked—if validateInput fails, the
// program silently proceeds to processRequest. Ineffassign catches this pattern.
package ineffassign

import (
	"golang.org/x/tools/go/analysis"

	"github.com/gordonklaus/ineffassign/pkg/ineffassign"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the ineffassign analyzer wrapped with nolint support.
var Analyzer *analysis.Analyzer = nolint.Wrap(ineffassign.Analyzer)
