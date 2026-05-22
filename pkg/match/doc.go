// Package match provides small, dependency-free pattern matching utilities.
//
// The package contains two separate matchers with different contracts.
// [MatchWatchPaths] matches repository-relative file paths against the watch
// path syntax documented below. [Wildcard] matches a single string against a
// simpler '*' wildcard pattern and does not treat '/' specially.
//
// # Watch paths
//
// Watch paths are repository-relative glob patterns. The matcher accepts '/' as
// the only path separator, normalizes leading "./" prefixes, and rejects paths
// with leading slashes, empty path segments, "." segments, parent directory
// segments, and backslashes. This keeps matching scoped to relative paths.
//
// The supported watch path syntax is intentionally limited:
//
//   - "*" matches any characters inside one path segment, including a leading
//     dot.
//   - "?" matches one character inside one path segment.
//   - "**" matches zero or more complete path segments only when it is the
//     whole segment.
//
// Patterns such as "src/**/main.go" can match both "src/main.go" and
// "src/pkg/api/main.go". Patterns such as "src/**.go" and "***" are invalid
// because "**" must be its own segment. Negative patterns, brace expansion, and
// extglob syntax are not supported. Character classes such as "[ab]" are not
// supported.
//
// # Watch path filtering
//
// An empty watch path list means no watch path filter is configured. In that
// case, [MatchWatchPaths] returns true when at least one changed file is a valid
// repository-relative file path. An empty changed file list always returns
// false because there is nothing to match.
//
// [MatchWatchPaths] returns only a boolean. It ignores invalid pattern and file
// pairs, then returns true on the first valid match. Callers that need to report
// invalid input must validate that input before calling
// [MatchWatchPaths].
//
// # Usage
//
// Basic watch path filtering:
//
//	changed := []string{"docs/readme.md", "src/api/server.go"}
//	matched := match.MatchWatchPaths([]string{"src/**/*.go"}, changed)
//	if !matched {
//	    // No changed path matched the configured patterns.
//	}
package match
