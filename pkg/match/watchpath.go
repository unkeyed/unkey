package match

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

var errInvalidWatchPath = errors.New("invalid watch path")

// MatchWatchPaths reports whether any changed file matches at least one watch
// path pattern.
//
// Patterns and files must be repository-relative paths that use '/' as the path
// separator. A leading "./" prefix is accepted and ignored. Paths with leading
// slashes, empty path segments, "." segments, parent directory segments, and
// backslashes are invalid.
//
// Watch path patterns support '*' and '?' within a single path segment. A "**"
// segment matches zero or more complete path segments. Wildcards match dotfiles,
// so "*" matches ".env". Character classes such as "[ab]" are not supported.
//
// If patterns is empty, MatchWatchPaths returns true when changedFiles contains
// at least one valid file path. If changedFiles is empty, it returns false. This
// function ignores invalid pattern and file pairs because it returns only a
// boolean. Callers that need to report invalid input must validate patterns
// before calling MatchWatchPaths.
func MatchWatchPaths(patterns []string, changedFiles []string) bool {
	if len(changedFiles) == 0 {
		return false
	}
	if len(patterns) == 0 {
		return hasValidWatchPathFile(changedFiles)
	}

	validPatterns := make([][]string, 0, len(patterns))
	for _, pattern := range patterns {
		segments := splitWatchPath(pattern)
		if len(segments) == 0 {
			continue
		}
		if validateWatchPathPattern(segments) != nil {
			continue
		}
		validPatterns = append(validPatterns, segments)
	}
	if len(validPatterns) == 0 {
		return false
	}

	validFiles := make([][]string, 0, len(changedFiles))
	for _, file := range changedFiles {
		segments := splitWatchPath(file)
		if len(segments) == 0 {
			continue
		}
		if validateWatchPathFile(segments) != nil {
			continue
		}
		validFiles = append(validFiles, segments)
	}
	if len(validFiles) == 0 {
		return false
	}

	for _, fileSegments := range validFiles {
		for _, patternSegments := range validPatterns {
			seen := map[segmentState]bool{}
			if matchWatchPathSegments(patternSegments, fileSegments, 0, 0, seen) {
				return true
			}
		}
	}
	return false
}

type segmentState struct {
	pattern int
	file    int
}

// matchWatchPath matches one validated watch path pattern against one changed
// file path.
//
// It returns errInvalidWatchPath for invalid patterns or file paths rather than
// silently treating invalid input as a non-match. This lets tests and future
// validation paths distinguish invalid input from a valid non-matching change
// set.
func matchWatchPath(pattern string, file string) (bool, error) {
	patternSegments := splitWatchPath(pattern)
	fileSegments := splitWatchPath(file)
	if len(patternSegments) == 0 || len(fileSegments) == 0 {
		return false, errInvalidWatchPath
	}
	if err := validateWatchPathPattern(patternSegments); err != nil {
		return false, err
	}
	if err := validateWatchPathFile(fileSegments); err != nil {
		return false, err
	}
	seen := map[segmentState]bool{}

	return matchWatchPathSegments(patternSegments, fileSegments, 0, 0, seen), nil
}

func splitWatchPath(value string) []string {
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	if value == "" || value == "." {
		return nil
	}

	return strings.Split(value, "/")
}

func validateWatchPathFile(segments []string) error {
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("%w: invalid file segment %q", errInvalidWatchPath, segment)
		}
		if strings.Contains(segment, "\\") {
			return fmt.Errorf("%w: backslashes are not supported", errInvalidWatchPath)
		}
	}
	return nil
}

func hasValidWatchPathFile(files []string) bool {
	for _, file := range files {
		segments := splitWatchPath(file)
		if len(segments) == 0 {
			continue
		}
		if validateWatchPathFile(segments) == nil {
			return true
		}
	}
	return false
}

func validateWatchPathPattern(segments []string) error {
	for i, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("%w: invalid pattern segment %q", errInvalidWatchPath, segment)
		}
		if strings.Contains(segment, "\\") {
			return fmt.Errorf("%w: backslashes are not supported", errInvalidWatchPath)
		}
		if i == 0 && strings.HasPrefix(segment, "!") && (len(segments) > 1 || strings.ContainsAny(segment, "*?")) {
			return fmt.Errorf("%w: negative patterns are not supported", errInvalidWatchPath)
		}
		if strings.ContainsAny(segment, "[]") {
			return fmt.Errorf("%w: character classes are not supported", errInvalidWatchPath)
		}
		if hasBraceExpansion(segment) {
			return fmt.Errorf("%w: brace expansion is not supported", errInvalidWatchPath)
		}
		if hasExtglobPrefix(segment) {
			return fmt.Errorf("%w: extglob patterns are not supported", errInvalidWatchPath)
		}
		if strings.Contains(segment, "**") && segment != "**" {
			return fmt.Errorf("%w: globstar must be its own segment", errInvalidWatchPath)
		}
		if segment == "**" {
			continue
		}
		if _, err := path.Match(segment, ""); err != nil {
			return fmt.Errorf("%w: %v", errInvalidWatchPath, err)
		}
	}
	return nil
}

func hasBraceExpansion(segment string) bool {
	open := strings.Index(segment, "{")
	if open < 0 {
		return false
	}
	close := strings.Index(segment[open+1:], "}")
	if close < 0 {
		return false
	}

	return strings.Contains(segment[open+1:open+1+close], ",")
}

func hasExtglobPrefix(segment string) bool {
	return strings.HasPrefix(segment, "@(") ||
		strings.HasPrefix(segment, "+(") ||
		strings.HasPrefix(segment, "!(") ||
		strings.HasPrefix(segment, "*(") ||
		strings.HasPrefix(segment, "?(")
}

// matchWatchPathSegments walks the pattern and file segments with bounded
// backtracking for "**".
//
// A globstar can consume any number of file segments, including zero. The seen
// set prevents revisiting the same pattern and file indexes when multiple
// globstars appear in one pattern.
func matchWatchPathSegments(patternSegments []string, fileSegments []string, patternIndex int, fileIndex int, seen map[segmentState]bool) bool {
	state := segmentState{pattern: patternIndex, file: fileIndex}
	if seen[state] {
		return false
	}
	seen[state] = true

	if patternIndex == len(patternSegments) {
		return fileIndex == len(fileSegments)
	}

	patternSegment := patternSegments[patternIndex]
	if patternSegment == "**" {
		if matchWatchPathSegments(patternSegments, fileSegments, patternIndex+1, fileIndex, seen) {
			return true
		}

		return fileIndex < len(fileSegments) &&
			matchWatchPathSegments(patternSegments, fileSegments, patternIndex, fileIndex+1, seen)
	}

	if fileIndex == len(fileSegments) {
		return false
	}

	fileSegment := fileSegments[fileIndex]
	matched, err := path.Match(patternSegment, fileSegment)
	if err != nil {
		return false
	}
	if !matched {
		return false
	}

	return matchWatchPathSegments(patternSegments, fileSegments, patternIndex+1, fileIndex+1, seen)
}
