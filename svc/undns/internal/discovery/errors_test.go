package discovery

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReasonOfSurvivesWrapping guarantees that a discovery failure keeps its
// metric reason and its cause when callers wrap it, and that errors without a
// reason, such as cache index failures, report lookup_error.
func TestReasonOfSurvivesWrapping(t *testing.T) {
	for _, reason := range []Reason{
		ReasonUnknownCaller, ReasonAmbiguousCaller, ReasonIneligibleCaller,
		ReasonConnectionUnresolved, ReasonConnectionAmbiguous, ReasonConnectionInvalid,
		ReasonServiceMissing, ReasonServiceRejected, ReasonServiceRetired,
		ReasonNoReadyEndpoints, ReasonLookupError,
	} {
		t.Run(string(reason), func(t *testing.T) {
			cause := errors.New("assertion failed")
			err := fmt.Errorf("resolve payments: %w", fail(reason, "discovery failed: %w", cause))
			require.Equal(t, reason, ReasonOf(err))
			require.ErrorIs(t, err, cause)
			require.EqualError(t, err, "resolve payments: discovery failed: assertion failed")
		})
	}

	t.Run("unknown error", func(t *testing.T) {
		require.Equal(t, ReasonLookupError, ReasonOf(fmt.Errorf("index pods: %w", errors.New("index failure"))))
	})
}
