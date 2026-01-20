// Package govet provides a composite go vet analyzer for use with nogo.
//
// This package bundles all standard go vet checks into a single analyzer that
// can be referenced by a unified //nolint:govet directive. Without this wrapper,
// each vet check (printf, atomic, copylock, etc.) would need its own directive.
//
// # Included Checks
//
// The analyzer includes all default go vet passes except fieldalignment and shadow,
// which are excluded per project configuration. Notable checks include:
//
//   - printf: Validates format string/argument consistency
//   - atomic: Catches incorrect atomic value usage
//   - copylock: Detects copying of sync.Mutex and similar types
//   - loopclosure: Finds loop variable capture bugs
//   - unreachable: Identifies code after return/panic statements
//   - structtag: Validates struct field tag syntax
//
// # Why a Composite?
//
// Running 30+ individual analyzers creates noise in configuration and makes
// suppression unwieldy. This composite presents them as a single logical unit
// matching how developers think about "vet checks" while preserving nolint support.
package govet

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// analyzers contains all go vet analyzers to run.
// Excludes fieldalignment and shadow per .golangci.yaml configuration.
var analyzers = []*analysis.Analyzer{
	appends.Analyzer,
	asmdecl.Analyzer,
	assign.Analyzer,
	atomic.Analyzer,
	atomicalign.Analyzer,
	bools.Analyzer,
	buildtag.Analyzer,
	cgocall.Analyzer,
	composite.Analyzer,
	copylock.Analyzer,
	deepequalerrors.Analyzer,
	defers.Analyzer,
	directive.Analyzer,
	errorsas.Analyzer,
	framepointer.Analyzer,
	httpresponse.Analyzer,
	ifaceassert.Analyzer,
	loopclosure.Analyzer,
	lostcancel.Analyzer,
	nilfunc.Analyzer,
	printf.Analyzer,
	shift.Analyzer,
	sigchanyzer.Analyzer,
	slog.Analyzer,
	sortslice.Analyzer,
	stdmethods.Analyzer,
	stringintconv.Analyzer,
	structtag.Analyzer,
	testinggoroutine.Analyzer,
	tests.Analyzer,
	timeformat.Analyzer,
	unmarshal.Analyzer,
	unreachable.Analyzer,
	unsafeptr.Analyzer,
	unusedresult.Analyzer,
}

// Analyzer is a composite analyzer that runs all go vet analyzers.
var Analyzer *analysis.Analyzer = nolint.WrapWithName(createCompositeAnalyzer(), "govet")

func createCompositeAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "govet",
		Doc:      "composite analyzer running all go vet checks",
		Requires: analyzers,
		Run: func(pass *analysis.Pass) (any, error) {
			// All sub-analyzers run via Requires, their diagnostics are reported automatically.
			// This composite just ensures they all run.
			return nil, nil
		},
	}
}
