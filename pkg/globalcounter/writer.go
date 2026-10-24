package globalcounter

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/repeat"
)

const (
	// rotationInterval is how long a writer id stays current. When it passes, the
	// Counter starts the next sequence, and the old writer id becomes the previous
	// one. A writer id is written only while it is the current or the previous
	// one, which is at most about two intervals after it started.
	rotationInterval = 15 * time.Minute
	// rotationJitterFraction spreads the rotations of a fleet by this fraction of
	// the interval, so the peer caches of all pods do not miss at the same time.
	// With it, a writer id lives at most 36 minutes: two jittered intervals.
	rotationJitterFraction = 0.2

	// compactionDelay is how long a row must stay untouched before compaction may
	// merge it, by the database clock. A writer stops writing a row when its writer
	// id is no longer the current or the previous one, at most about 36 minutes
	// after the id started and a few seconds per retry after that. The delay is the
	// margin for a write that is still in flight when its row is merged: a merged
	// row that a late write recreated would count twice.
	compactionDelay = 60 * time.Minute

	// firstSequence is the sequence a Counter starts with. The previous sequence,
	// which the Counter reads as an id too, never has rows.
	firstSequence = int64(1)
)

// writerID returns the writer id of sequence. A Counter's writer ids share its
// process id, so they never collide with another process's, and a restart starts
// a new process id instead of reusing sequences, whose revisions it has lost.
func (c *Counter) writerID(sequence int64) string {
	return fmt.Sprintf("%s_%d", c.processID, sequence)
}

// rotate starts the next sequence. Writes that began before it may still land in
// the previous one, which stays in memory.
func (c *Counter) rotate() {
	c.sequence.Add(1)
}

// runRotation calls rotate about every rotationInterval until c.done closes. The
// first call of repeat.Every runs at once, and that one only marks the start.
func (c *Counter) runRotation() {
	started := false
	repeat.Every(rotationInterval, func(context.Context) {
		if !started {
			started = true
			return
		}
		c.rotate()
	},
		repeat.WithClock(c.clock),
		repeat.WithJitter(rotationJitterFraction),
		repeat.UntilClose(c.done),
	)
}
