// Package reassign provides a reassign analyzer for use with nogo.
//
// Reassign detects reassignment of package-level variables, particularly those
// from other packages like os.Stdout, http.DefaultClient, or sql.ErrNoRows.
// Reassigning these global variables affects all code in the process and leads
// to hard-to-debug issues in concurrent programs.
//
// # Why This Matters
//
// Code like:
//
//	http.DefaultClient = &http.Client{Timeout: 5 * time.Second}
//
// Looks reasonable but modifies a global that other packages depend on. Library
// code expecting the default client's behavior now gets unexpected timeouts.
// The linter pushes developers toward explicit clients passed as dependencies.
package reassign

import (
	"golang.org/x/tools/go/analysis"

	"github.com/curioswitch/go-reassign"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the reassign analyzer wrapped with nolint support.
var Analyzer *analysis.Analyzer = nolint.Wrap(reassign.NewAnalyzer())
