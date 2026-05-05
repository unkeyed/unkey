package bus

import (
	"container/list"
	"sync"
	"time"
)

// replayEntry is a single record kept in the per-topic ring.
type replayEntry struct {
	topic    string
	sender   string
	id       string
	envelope []byte
	at       time.Time
}

func (e *replayEntry) size() int {
	return len(e.envelope) + len(e.topic) + len(e.sender) + len(e.id)
}

// replayLog stores recently-published BusEnvelope bytes per topic so that a
// late-joining peer can request a replay on join (PR 3). Two caps are
// enforced: one per topic (so a noisy topic cannot starve a quiet one) and
// one across the pod (so a runaway pair of topics cannot blow up memory).
//
// Eviction:
//   - When a topic ring exceeds its per-topic cap, drop the topic's oldest
//     entries until under the cap.
//   - When the aggregate exceeds the total cap, drop the oldest entry from
//     the largest ring, repeatedly, until under the cap.
//
// The publisher writes to the log on every successful Publish; readers come
// from the replay-on-join handler in PR 3.
type replayLog struct {
	mu             sync.Mutex
	perTopicBytes  int
	totalBytes     int
	totalUsed      int
	rings          map[string]*topicRing
	dropPerTopic   func(topic string)
	dropAggregate  func(topic string)
	usedGaugeReset func(topic string, used int)
}

type topicRing struct {
	entries *list.List
	used    int
}

func newReplayLog(perTopicCap, totalCap int, hooks replayHooks) *replayLog {
	if perTopicCap < 1 {
		perTopicCap = 1
	}
	if totalCap < perTopicCap {
		totalCap = perTopicCap
	}
	return &replayLog{
		mu:             sync.Mutex{},
		perTopicBytes:  perTopicCap,
		totalBytes:     totalCap,
		totalUsed:      0,
		rings:          make(map[string]*topicRing),
		dropPerTopic:   hooks.onTopicEviction,
		dropAggregate:  hooks.onAggregateEviction,
		usedGaugeReset: hooks.onTopicSize,
	}
}

// replayHooks are optional callbacks for metrics. Each is called under the
// log's lock; implementations should not block.
type replayHooks struct {
	onTopicEviction     func(topic string)
	onAggregateEviction func(topic string)
	onTopicSize         func(topic string, bytes int)
}

func (l *replayLog) Append(entry replayEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ring, ok := l.rings[entry.topic]
	if !ok {
		ring = &topicRing{entries: list.New(), used: 0}
		l.rings[entry.topic] = ring
	}

	sz := entry.size()
	ring.entries.PushBack(entry)
	ring.used += sz
	l.totalUsed += sz

	for ring.used > l.perTopicBytes {
		front := ring.entries.Front()
		if front == nil {
			break
		}
		fe := front.Value.(replayEntry)
		ring.entries.Remove(front)
		ring.used -= fe.size()
		l.totalUsed -= fe.size()
		if l.dropPerTopic != nil {
			l.dropPerTopic(entry.topic)
		}
	}

	for l.totalUsed > l.totalBytes {
		victimTopic, victimRing := l.largestRingLocked()
		if victimRing == nil {
			break
		}
		front := victimRing.entries.Front()
		if front == nil {
			break
		}
		fe := front.Value.(replayEntry)
		victimRing.entries.Remove(front)
		victimRing.used -= fe.size()
		l.totalUsed -= fe.size()
		if l.dropAggregate != nil {
			l.dropAggregate(victimTopic)
		}
	}

	if l.usedGaugeReset != nil {
		l.usedGaugeReset(entry.topic, ring.used)
	}
}

// Snapshot returns the entries for a topic in publish order. PR 3 uses this
// to satisfy replay-on-join queries. Returns a fresh slice; callers may
// retain it.
func (l *replayLog) Snapshot(topic string) []replayEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	ring, ok := l.rings[topic]
	if !ok {
		return nil
	}
	out := make([]replayEntry, 0, ring.entries.Len())
	for e := ring.entries.Front(); e != nil; e = e.Next() {
		out = append(out, e.Value.(replayEntry))
	}
	return out
}

func (l *replayLog) TotalBytes() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.totalUsed
}

func (l *replayLog) largestRingLocked() (string, *topicRing) {
	var (
		bestTopic string
		bestRing  *topicRing
	)
	for t, r := range l.rings {
		if bestRing == nil || r.used > bestRing.used {
			bestTopic = t
			bestRing = r
		}
	}
	return bestTopic, bestRing
}
