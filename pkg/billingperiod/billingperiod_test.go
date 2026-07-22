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

func TestPeriodCloseAllowed(t *testing.T) {
	p, err := Parse("2026-07")
	require.NoError(t, err)
	roll := p.End().Unix()

	require.True(t, p.CloseAllowed(p.End(), 0))
	require.True(t, p.CloseAllowed(p.End().Add(-time.Second), roll))
	require.False(t, p.CloseAllowed(p.End().Add(-time.Second), 0))
	require.False(t, p.CloseAllowed(p.End().Add(-time.Second), roll-1))
}

func TestFromKeyPrev(t *testing.T) {
	t.Run("From uses UTC calendar month", func(t *testing.T) {
		// An instant late on the last day of June in a positive offset would be
		// July locally but is still June in UTC; From must key by UTC.
		p := From(time.Date(2026, time.June, 30, 23, 30, 0, 0, time.UTC))
		require.Equal(t, Period{Year: 2026, Month: time.June}, p)
	})

	t.Run("Key is the inverse of Parse", func(t *testing.T) {
		p, err := Parse("2026-03")
		require.NoError(t, err)
		require.Equal(t, "2026-03", p.Key())
		require.Equal(t, "2026-11", Period{Year: 2026, Month: time.November}.Key())
	})

	t.Run("Prev steps back a month across year boundary", func(t *testing.T) {
		require.Equal(t, "2026-06", Period{Year: 2026, Month: time.July}.Prev().Key())
		require.Equal(t, "2025-12", Period{Year: 2026, Month: time.January}.Prev().Key())
	})
}

func TestPeriodEnd(t *testing.T) {
	p, err := Parse("2026-06")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), p.End())

	dec, err := Parse("2026-12")
	require.NoError(t, err)
	require.Equal(t, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC), dec.End())
}
