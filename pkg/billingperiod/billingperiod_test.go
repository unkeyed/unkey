package billingperiod

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p, err := Parse("2026-03")
		require.NoError(t, err)
		require.Equal(t, 2026, p.Year)
		require.Equal(t, time.March, p.Month)
	})

	t.Run("start is first of month UTC", func(t *testing.T) {
		p, err := Parse("2026-03")
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), p.Start())
	})

	for _, key := range []string{"", "2026", "2026-13", "2026-00", "2026-3-1", "abc-03", "2026-ab", "2026-5", "26-05"} {
		t.Run("rejects "+key, func(t *testing.T) {
			_, err := Parse(key)
			require.Error(t, err)
		})
	}
}

func TestPeriodEnd(t *testing.T) {
	p, err := Parse("2026-06")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), p.End())

	dec, err := Parse("2026-12")
	require.NoError(t, err)
	require.Equal(t, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC), dec.End())
}
