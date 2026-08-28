package match

import (
	"github.com/bmatcuk/doublestar/v4"
)

// MatchWatchPaths reports whether any of the changedFiles match at least one of
// the given watch path patterns, and returns the patterns that are not valid
// doublestar globs. Patterns use doublestar syntax (e.g. "src/**", "*.go",
// "services/api/**/*.ts").
//
// The two results are independent. An invalid pattern never matches, but it does
// not stop a valid sibling from matching, so a caller can see a match alongside a
// non-empty invalid list and should report both.
//
// If patterns is empty, matched is true (no filtering).
// If changedFiles is empty, matched is false (nothing to match).
func MatchWatchPaths(patterns []string, changedFiles []string) (matched bool, invalid []string) {
	// Validating up front rather than inside the match loop keeps the invalid list
	// complete. The loop returns on the first hit and both empty cases return
	// early, so lazy validation would make the list depend on the pushed files.
	invalid = InvalidWatchPaths(patterns)

	if len(patterns) == 0 {
		return true, invalid
	}
	if len(changedFiles) == 0 {
		return false, invalid
	}

	for _, file := range changedFiles {
		for _, pattern := range patterns {
			ok, err := doublestar.Match(pattern, file)
			if err != nil {
				continue
			}
			if ok {
				return true, invalid
			}
		}
	}
	return false, invalid
}

// InvalidWatchPaths returns the patterns that are not valid doublestar globs, in
// input order. Callers that only accept watch paths, such as the settings API,
// have no files to match against and use this alone; MatchWatchPaths builds on it
// for the push path.
func InvalidWatchPaths(patterns []string) []string {
	var invalid []string
	for _, pattern := range patterns {
		if !doublestar.ValidatePattern(pattern) {
			invalid = append(invalid, pattern)
		}
	}
	return invalid
}
