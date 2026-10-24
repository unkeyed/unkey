package globalcounter

import (
	"testing"
)

func Cleanup(t testing.TB, counter *Counter) {
	t.Helper()
	t.Cleanup(func() { counter.closeOnce.Do(func() { close(counter.done) }) })
	t.Cleanup(counter.peerCache.Close)
	t.Cleanup(counter.buffer.Close)
}

func RotateOnce(counter *Counter) { counter.rotate() }

func WriterID(counter *Counter) string { return counter.writerID(counter.sequence.Load()) }

func PreviousWriterID(counter *Counter) string { return counter.writerID(counter.sequence.Load() - 1) }
