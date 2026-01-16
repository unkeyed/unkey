// Package nolint provides utilities to wrap go/analysis analyzers with
// support for //nolint directives, which nogo does not handle by default.
package nolint

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Options configures the nolint wrapper behavior.
type Options struct {
	// SkipPatterns excludes files matching these suffix patterns from analysis.
	// Example: []string{"_test.go", "_generated.go"}
	SkipPatterns []string
}

// Wrap returns a new analyzer that filters out diagnostics suppressed by
// //nolint comments. It supports:
//   - //nolint - suppresses all linters
//   - //nolint:name - suppresses the specific linter
//   - //nolint:name1,name2 - suppresses multiple linters
//
// The suppression applies to the AST node following the comment, similar to golangci-lint.
func Wrap(a *analysis.Analyzer) *analysis.Analyzer {
	return WrapWithOptions(a, a.Name, Options{SkipPatterns: nil})
}

// WrapSkipPatterns is like Wrap but excludes files matching any of the given suffix patterns.
func WrapSkipPatterns(a *analysis.Analyzer, patterns ...string) *analysis.Analyzer {
	return WrapWithOptions(a, a.Name, Options{SkipPatterns: patterns})
}

// WrapWithName is like Wrap but allows overriding the linter name used for
// matching //nolint:name directives.
func WrapWithName(a *analysis.Analyzer, name string) *analysis.Analyzer {
	return WrapWithOptions(a, name, Options{SkipPatterns: nil})
}

// WrapWithOptions is the most flexible wrapper that accepts all options.
func WrapWithOptions(a *analysis.Analyzer, name string, opts Options) *analysis.Analyzer {
	wrapped := *a
	origRun := a.Run

	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		// Build suppression ranges before running the analyzer
		ranges := buildSuppressionRanges(pass, name, opts)

		// Create a wrapped pass that filters Report calls
		wrappedPass := *pass
		origReport := pass.Report
		wrappedPass.Report = func(d analysis.Diagnostic) {
			pos := pass.Fset.Position(d.Pos)
			if !isSuppressed(ranges, pos.Filename, pos.Line) {
				origReport(d)
			}
		}

		return origRun(&wrappedPass)
	}

	return &wrapped
}

// matchesAnyPattern checks if the filename matches any of the given suffix patterns.
func matchesAnyPattern(filename string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(filename, pattern) {
			return true
		}
	}
	return false
}

// ignoredRange represents a range of lines to suppress.
type ignoredRange struct {
	from int
	to   int
}

// fileRanges holds suppression data for a file.
type fileRanges struct {
	suppressAll bool
	ranges      []ignoredRange
}

// suppressionRanges maps filename -> suppression data.
type suppressionRanges map[string]*fileRanges

// buildSuppressionRanges scans all files and builds AST-aware suppression ranges.
func buildSuppressionRanges(pass *analysis.Pass, linterName string, opts Options) suppressionRanges {
	result := make(suppressionRanges)

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		// Skip files matching any of the configured patterns
		if matchesAnyPattern(filename, opts.SkipPatterns) {
			result[filename] = &fileRanges{suppressAll: true, ranges: nil}
			continue
		}

		fr := &fileRanges{
			suppressAll: false,
			ranges:      make([]ignoredRange, 0),
		}

		// Build a map of comment end line -> comment info for quick lookup
		type commentInfo struct {
			col     int
			endLine int
		}
		nolintComments := make(map[int]commentInfo) // line -> info

		for _, cg := range file.Comments {
			for _, c := range cg.List {
				if shouldSuppress(c.Text, linterName) {
					pos := pass.Fset.Position(c.Slash)
					endPos := pass.Fset.Position(c.End())

					// Check for file-level suppression (before package decl or first 2 lines)
					pkgLine := pass.Fset.Position(file.Package).Line
					if pos.Line < pkgLine || pos.Line <= 2 {
						fr.suppressAll = true
						break
					}

					nolintComments[endPos.Line] = commentInfo{
						col:     pos.Column,
						endLine: endPos.Line,
					}
				}
			}
			if fr.suppressAll {
				break
			}
		}

		if fr.suppressAll {
			result[filename] = fr
			continue
		}

		// Walk the AST to expand ranges based on node boundaries
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}

			nodeStart := pass.Fset.Position(node.Pos())
			nodeEnd := pass.Fset.Position(node.End())

			// Check if there's a nolint comment ending on the line before this node
			if info, ok := nolintComments[nodeStart.Line-1]; ok {
				// Expand range to cover the entire node
				fr.ranges = append(fr.ranges, ignoredRange{
					from: info.endLine,
					to:   nodeEnd.Line,
				})
				// Remove so we don't match again (first match wins)
				delete(nolintComments, nodeStart.Line-1)
			}

			// Also check for inline comments (same line as node start)
			if inlineInfo, ok := nolintComments[nodeStart.Line]; ok {
				fr.ranges = append(fr.ranges, ignoredRange{
					from: nodeStart.Line,
					to:   nodeEnd.Line,
				})
				_ = inlineInfo // used for lookup only
				delete(nolintComments, nodeStart.Line)
			}

			return true
		})

		// Any remaining comments without matching nodes: apply to comment line + next line
		for line, info := range nolintComments {
			fr.ranges = append(fr.ranges, ignoredRange{
				from: info.endLine,
				to:   line + 1,
			})
		}

		if len(fr.ranges) > 0 || fr.suppressAll {
			result[filename] = fr
		}
	}

	return result
}

// shouldSuppress checks if a comment text suppresses the given linter.
func shouldSuppress(commentText, linterName string) bool {
	text := strings.TrimSpace(strings.TrimPrefix(commentText, "//"))

	if !strings.Contains(text, "nolint") {
		return false
	}

	colonIdx := strings.Index(text, ":")
	if colonIdx == -1 {
		// Plain //nolint suppresses all
		return strings.TrimSpace(text) == "nolint"
	}

	prefix := strings.TrimSpace(text[:colonIdx])
	if prefix != "nolint" {
		return false
	}

	linters := strings.TrimSpace(text[colonIdx+1:])
	for _, l := range strings.Split(linters, ",") {
		l = strings.TrimSpace(l)
		// Handle trailing comments like "nolint:exhaustruct // reason"
		if spaceIdx := strings.Index(l, " "); spaceIdx != -1 {
			l = l[:spaceIdx]
		}
		// "all" suppresses all linters
		if l == "all" || l == linterName {
			return true
		}
	}

	return false
}

// isSuppressed checks if a diagnostic at the given file:line is suppressed.
func isSuppressed(sr suppressionRanges, filename string, line int) bool {
	fr, ok := sr[filename]
	if !ok {
		return false
	}

	if fr.suppressAll {
		return true
	}

	for _, r := range fr.ranges {
		if line >= r.from && line <= r.to {
			return true
		}
	}

	return false
}
