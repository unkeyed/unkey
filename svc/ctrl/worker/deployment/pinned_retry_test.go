package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestPinnedRetryDelayBacksOffToCap guarantees that a stop deferred by a
// pinning binding is rechecked less often the longer the binding stays, and
// never less often than pinnedRetryDelayMax.
func TestPinnedRetryDelayBacksOffToCap(t *testing.T) {
	for _, tt := range []struct {
		retries int
		want    time.Duration
	}{
		{retries: 0, want: time.Minute},
		{retries: 1, want: 2 * time.Minute},
		{retries: 3, want: 8 * time.Minute},
		{retries: 4, want: 15 * time.Minute},
		{retries: 1000, want: 15 * time.Minute},
	} {
		require.Equal(t, tt.want, pinnedRetryDelay(tt.retries), "pinnedRetryDelay(%d)", tt.retries)
	}
}
