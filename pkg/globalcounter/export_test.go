package globalcounter

import (
	"context"
	"testing"
)

func Cleanup(t testing.TB, counter *Counter) {
	t.Helper()
	t.Cleanup(func() { counter.closeOnce.Do(func() { close(counter.done) }) })
	t.Cleanup(counter.peerCache.Close)
	t.Cleanup(counter.buffer.Close)
}

func EvictOnce(ctx context.Context, counter *Counter) { counter.evict(ctx) }

func RotateOnce(counter *Counter) { counter.rotate() }

func Sequence(counter *Counter) int64 { return counter.sequence.Load() }

// DecideAt runs the decision step of ConditionalAdd for sequence, which may be
// stale, and reports whether it refused because a rotation had happened.
func DecideAt(counter *Counter, name string, sequence int64) bool {
	_, _, _, _, rotated, _ := counter.addDecided(ConditionalAddParams{Name: name, Delta: func(int64) int64 { return 1 }}, entryKey{name: name, sequence: sequence}, 0, counter.clock.Now())
	return rotated
}

// DecideManyAt is DecideAt for ConditionalAddMany.
func DecideManyAt(counter *Counter, name string, sequence int64) bool {
	_, _, rotated, _ := counter.decideMany([]ConditionalAddCheck{{Name: name, Delta: func(int64) (int64, bool) { return 1, false }}}, sequence, []int64{0}, counter.clock.Now())
	return rotated
}

func WriterID(counter *Counter) string { return counter.writerID(counter.sequence.Load()) }

func PreviousWriterID(counter *Counter) string { return counter.writerID(counter.sequence.Load() - 1) }
