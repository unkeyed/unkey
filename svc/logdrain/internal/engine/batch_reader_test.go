package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// TestBatchReader_AdaptsWindows preserves boundary events and timestamp ties
// while growing empty windows, retaining partial windows, and resetting full ones.
func TestBatchReader_AdaptsWindows(t *testing.T) {
	const start int64 = 1000000
	const minute int64 = 60000
	cases := []struct {
		name string
		from source.Cursor
		end  int64
		ids  []string
		next source.Cursor
	}{
		{"empty", source.Cursor{Time: start}, start + minute, nil, source.Cursor{Time: start + minute}},
		{"partial", source.Cursor{Time: start + minute}, start + 3*minute, []string{"a"}, source.Cursor{Time: start + minute, EventID: "a"}},
		{"full", source.Cursor{Time: start + minute, EventID: "a"}, start + 2*minute, []string{"b", "c"}, source.Cursor{Time: start + minute, EventID: "c"}},
		{"timestamp tie", source.Cursor{Time: start + minute, EventID: "c"}, start + 2*minute, []string{"d"}, source.Cursor{Time: start + minute, EventID: "d"}},
		{"empty after partial", source.Cursor{Time: start + minute, EventID: "d"}, start + 2*minute, nil, source.Cursor{Time: start + 2*minute}},
		{"empty expansion", source.Cursor{Time: start + 2*minute}, start + 4*minute, nil, source.Cursor{Time: start + 4*minute}},
		{"watermark", source.Cursor{Time: start + 4*minute}, start + 6*minute, nil, source.Cursor{Time: start + 6*minute}},
	}
	var src windowSource
	reader := newBatchReader(&src, start+6*minute, 2)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var events []sink.Event
			advance := tc.from
			for _, id := range tc.ids {
				events = append(events, sink.Event{EventID: id})
				advance.EventID = id
			}
			src.read = func(_ context.Context, _ string, from source.Cursor, to int64, limit int, _ *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
				require.Equal(t, tc.from, from)
				require.Equal(t, tc.end, to)
				require.Equal(t, 2, limit)
				return events, advance, nil
			}
			page, err := reader.Read(context.Background(), "workspace", tc.from, nil)
			require.NoError(t, err)
			require.Equal(t, events, page.events)
			require.Equal(t, tc.next, page.next)
			require.Equal(t, tc.name == "watermark", page.caughtUp)
		})
	}
}

// TestBatchReader_ReadFailure retries the same window rather than treating a
// source error as empty history and skipping its events.
func TestBatchReader_ReadFailure(t *testing.T) {
	from := source.Cursor{Time: 1000000}
	failure := errors.New("source unavailable")
	calls := 0
	src := windowSource{read: func(_ context.Context, _ string, cursor source.Cursor, to int64, _ int, _ *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
		calls++
		require.Equal(t, from, cursor)
		require.Equal(t, from.Time+time.Minute.Milliseconds(), to)
		return nil, cursor, failure
	}}
	reader := newBatchReader(src, from.Time+time.Hour.Milliseconds(), 100)
	for range 2 {
		page, err := reader.Read(context.Background(), "workspace", from, nil)
		require.ErrorIs(t, err, failure)
		require.Equal(t, batchPage{}, page)
	}
	require.Equal(t, 2, calls)
}

// windowSource exposes the source boundary without a ClickHouse server.
type windowSource struct {
	read func(context.Context, string, source.Cursor, int64, int, *logdrainv1.Config) ([]sink.Event, source.Cursor, error)
}

// Read delegates bounded reads to the test fixture.
func (s windowSource) Read(ctx context.Context, workspaceID string, from source.Cursor, to int64, limit int, config *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
	return s.read(ctx, workspaceID, from, to, limit, config)
}
