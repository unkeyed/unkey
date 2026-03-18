package match

import (
	"github.com/bmatcuk/doublestar/v4"
)

// MatchWatchPaths returns true if any of the changedFiles match at least one of
// the given watch path patterns. Patterns use doublestar syntax (e.g. "src/**",
// "*.go", "services/api/**/*.ts").
//
// If patterns is empty the function returns true (no filtering).
// If changedFiles is empty the function returns false (nothing to match).
func MatchWatchPaths(patterns []string, changedFiles []string) bool {
	if len(patterns) == 0 {
		return true
	}
	if len(changedFiles) == 0 {
		return false
	}

	for _, file := range changedFiles {
		for _, pattern := range patterns {
			matched, err := doublestar.Match(pattern, file)
			if err != nil {
				// Treat bad patterns as non-matching rather than failing the deploy.
				continue
			}
			if matched {
				return true
			}
		}
	}
	return false
}
