package sink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestParseRetryAfter guarantees support for both standard wire formats and
// rejects values that cannot produce a valid future delay.
func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{name: "delta seconds", value: "120", want: 2 * time.Minute, ok: true},
		{name: "zero delta", value: "0", ok: true},
		{name: "HTTP date", value: "Thu, 27 Aug 2026 12:05:00 GMT", want: 5 * time.Minute, ok: true},
		{name: "expired HTTP date", value: "Thu, 27 Aug 2026 11:59:59 GMT"},
		{name: "negative delta", value: "-1"},
		{name: "invalid", value: "later"},
		{name: "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseRetryAfter(tt.value, now)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}
