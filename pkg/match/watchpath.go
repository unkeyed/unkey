package match

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// MatchWatchPaths reports whether any changedFile matches at least one doublestar
// pattern (e.g. "src/**", "**/*.go"). A broken pattern is a config error,
// so it returns the fault from ValidateWatchPaths instead of reporting no match.
// Empty patterns match everything; empty changedFiles match nothing.
func MatchWatchPaths(patterns []string, changedFiles []string) (bool, error) {
	// We catch broken patterns below at `doublestar.Match`, but this is just a fail-fast version
	if err := ValidateWatchPaths(patterns); err != nil {
		return false, err
	}
	if len(patterns) == 0 {
		return true, nil
	}

	for _, file := range changedFiles {
		for _, pattern := range patterns {
			ok, err := doublestar.Match(pattern, file)
			if err != nil {
				return false, fault.Wrap(
					err,
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal(fmt.Sprintf("watch path %q failed to match", pattern)),
					fault.Public(invalidWatchPathMessage(pattern)),
				)
			}
			if ok {
				return true, nil
			}
		}
	}
	return false, nil
}

// ValidateWatchPaths returns a fault for the first pattern that is not a valid
// doublestar glob.
func ValidateWatchPaths(patterns []string) error {
	for _, pattern := range patterns {
		if !doublestar.ValidatePattern(pattern) {
			return fault.New(
				"invalid watch path",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(fmt.Sprintf("watch path %q is not a valid doublestar pattern", pattern)),
				fault.Public(invalidWatchPathMessage(pattern)),
			)
		}
	}
	return nil
}

func invalidWatchPathMessage(pattern string) string {
	return fmt.Sprintf("Watch path '%s' is not a valid glob pattern. Fix it in your build settings using glob syntax like 'src/**' or '**/*.go'.", pattern)
}
