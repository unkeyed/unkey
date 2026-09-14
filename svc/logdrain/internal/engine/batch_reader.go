package engine

import (
	"context"
	"time"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// batchReader adapts source windows within one catch-up cycle. Durable cursor
// ownership remains with the engine: a page is committed only after delivery.
type batchReader struct {
	source     source.Source
	watermark  int64
	batchSize  int
	windowSize time.Duration
}

// batchPage describes both delivered events and progress through empty history.
type batchPage struct {
	events   []sink.Event
	next     source.Cursor
	caughtUp bool
}

// newBatchReader fixes the lagging watermark for one cycle and starts with a
// small window so a busy workspace does not begin with a backlog-wide scan.
func newBatchReader(src source.Source, watermark int64, batchSize int) *batchReader {
	return &batchReader{source: src, watermark: watermark, batchSize: batchSize, windowSize: time.Minute}
}

// Read grows empty windows up to an hour. Nonempty pages continue at their last
// event because a source may stop at its byte limit before reaching the row limit.
func (r *batchReader) Read(ctx context.Context, workspaceID string, from source.Cursor, config *logdrainv1.Config) (batchPage, error) {
	if from.Time >= r.watermark {
		return batchPage{events: nil, next: from, caughtUp: true}, nil
	}
	windowEnd := min(from.Time+r.windowSize.Milliseconds(), r.watermark)
	events, next, err := r.source.Read(ctx, workspaceID, from, windowEnd, r.batchSize, config)
	if err != nil {
		return batchPage{}, err
	}
	if len(events) > 0 {
		r.windowSize = time.Minute
		return batchPage{events: events, next: next, caughtUp: false}, nil
	}
	r.windowSize = min(2*r.windowSize, time.Hour)
	return batchPage{
		events: events,
		// The exclusive upper bound belongs to the next window, including its first event.
		next:     source.Cursor{Time: windowEnd, EventID: ""},
		caughtUp: windowEnd == r.watermark,
	}, nil
}
