package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
