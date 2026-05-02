package coordinator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

func TestGroupMinCursor(t *testing.T) {
	t.Parallel()

	t.Run("returns lex-min across non-blocked drains", func(t *testing.T) {
		t.Parallel()
		drains := map[string]drainCursor{
			"d1": {cursor: cursor{timeMs: 1500, lastID: "log_0000000000000007"}},
			"d2": {cursor: cursor{timeMs: 1000, lastID: "log_0000000000000099"}},
			"d3": {cursor: cursor{timeMs: 1000, lastID: "log_0000000000000050"}},
		}
		got, ok := groupMinCursor(drains)
		require.True(t, ok)
		require.Equal(t, cursor{timeMs: 1000, lastID: "log_0000000000000050"}, got,
			"min must use lex order on (timeMs, lastID)")
	})

	t.Run("ignores blocked drains", func(t *testing.T) {
		t.Parallel()
		drains := map[string]drainCursor{
			// d1 is the lex-min but it's blocked — it must not pin the
			// group's read watermark.
			"d1": {cursor: cursor{timeMs: 100, lastID: ""}, blocked: true},
			"d2": {cursor: cursor{timeMs: 500, lastID: ""}},
			"d3": {cursor: cursor{timeMs: 800, lastID: ""}},
		}
		got, ok := groupMinCursor(drains)
		require.True(t, ok)
		require.Equal(t, cursor{timeMs: 500, lastID: ""}, got)
	})

	t.Run("all blocked returns ok=false", func(t *testing.T) {
		t.Parallel()
		drains := map[string]drainCursor{
			"d1": {cursor: cursor{timeMs: 100}, blocked: true},
			"d2": {cursor: cursor{timeMs: 200}, blocked: true},
		}
		_, ok := groupMinCursor(drains)
		require.False(t, ok, "ok=false signals 'no readable cursor'")
	})

	t.Run("empty map returns ok=false", func(t *testing.T) {
		t.Parallel()
		_, ok := groupMinCursor(map[string]drainCursor{})
		require.False(t, ok)
	})
}

func TestCursorLess(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b cursor
		want bool
	}{
		{"a<b on time", cursor{timeMs: 100}, cursor{timeMs: 200}, true},
		{"a>b on time", cursor{timeMs: 200}, cursor{timeMs: 100}, false},
		{"a<b on lastID with same time",
			cursor{timeMs: 100, lastID: "log_0000000000000001"},
			cursor{timeMs: 100, lastID: "log_0000000000000002"}, true},
		{"a==b is not less",
			cursor{timeMs: 100, lastID: "log_0000000000000001"},
			cursor{timeMs: 100, lastID: "log_0000000000000001"}, false},
		// Empty lastID sorts before every real id, matching the
		// bootstrap cursor behaviour where a freshly-created drain has
		// lastID="" and must read every row inside its BatchWindow.
		{"empty lastID is min",
			cursor{timeMs: 1, lastID: ""},
			cursor{timeMs: 1, lastID: "log_0000000000000001"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, cursorLess(tc.a, tc.b))
		})
	}
}

func TestRecordsPastCursor(t *testing.T) {
	t.Parallel()

	batch := []sinks.Record{
		{CursorTimeMs: 100, LastID: "log_0000000000000001"},
		{CursorTimeMs: 100, LastID: "log_0000000000000005"},
		{CursorTimeMs: 200, LastID: "log_0000000000000000"},
		{CursorTimeMs: 300, LastID: "log_0000000000000009"},
	}

	t.Run("cursor at start returns full slice", func(t *testing.T) {
		t.Parallel()
		out := recordsPastCursor(batch, cursor{timeMs: 0, lastID: ""})
		require.Len(t, out, 4)
	})

	t.Run("cursor mid-batch returns suffix", func(t *testing.T) {
		t.Parallel()
		// Cursor at (100, log_..05) — the second record. Drain has
		// already delivered up to and including it; expect records 3
		// and 4.
		out := recordsPastCursor(batch, cursor{timeMs: 100, lastID: "log_0000000000000005"})
		require.Len(t, out, 2)
		require.EqualValues(t, 200, out[0].CursorTimeMs)
	})

	t.Run("cursor past end returns nil", func(t *testing.T) {
		t.Parallel()
		// Drain is ahead of every record in the batch — nothing to send.
		out := recordsPastCursor(batch, cursor{timeMs: 1000, lastID: ""})
		require.Empty(t, out)
	})

	t.Run("equal cursor excludes that record", func(t *testing.T) {
		t.Parallel()
		// Strictly past — a record at exactly the cursor's tuple has
		// already been delivered, so it must not be re-sent.
		out := recordsPastCursor(batch, cursor{timeMs: 100, lastID: "log_0000000000000001"})
		require.Len(t, out, 3)
		require.Equal(t, "log_0000000000000005", out[0].LastID)
	})
}
