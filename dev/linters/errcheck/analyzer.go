// Package errcheck provides an errcheck analyzer for use with nogo.
//
// Errcheck detects when a function that returns an error has its error return
// value ignored. Unchecked errors are a common source of silent failures in
// production—the program continues executing with invalid state instead of
// handling the problem.
//
// This wrapper skips files ending in _generated.go because generated code
// (protobuf, sqlc, etc.) frequently ignores errors intentionally and cannot
// be modified.
//
// # Why This Matters
//
// Consider code like:
//
//	file.Close()           // errcheck: error ignored
//	json.Unmarshal(b, &v)  // errcheck: error ignored
//
// Both calls can fail, and ignoring the error means the program continues with
// incomplete cleanup or corrupt data. The linter catches these before they
// reach production.
package errcheck

import (
	"golang.org/x/tools/go/analysis"

	"github.com/kisielk/errcheck/errcheck"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the errcheck analyzer wrapped with nolint support.
// Skips generated files since they are auto-generated and may have unchecked errors.
var Analyzer *analysis.Analyzer = nolint.WrapSkipPatterns(errcheck.Analyzer, "_generated.go")
