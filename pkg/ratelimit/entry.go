package ratelimit

import (
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
)

// entry holds the sessions of one rate limit on this pod and their reset
// times. Only its methods change the sessions, so the reset times always match
// them.
// The caller holds the shard lock.
type entry struct {
	// live is nil if the pod has no live session.
	live *session

	// others are peer sessions and this pod's frozen sessions, unordered.
	others []session

	// resetTimes caches `reset_at` and the two other times that `decide` needs,
	// so a check is O(1) instead of O(n log n) for `computeResetTimes`.
	resetTimes resetTimes
}

// all returns a copy of all sessions.
func (e *entry) all() []session {
	all := slices.Clone(e.others)
	if e.live != nil {
		all = append(all, *e.live)
	}
	return all
}

// mergeRows adds database rows and keeps the larger `consumedNs` per session,
// so a stale replica never moves a session back. It never removes a session,
// and on an error it changes nothing.
func (e *entry) mergeRows(rows []session) error {
	var live *session
	if e.live != nil {
		liveCopy := *e.live
		live = &liveCopy
	}
	others := slices.Clone(e.others)

	indexByID := make(map[string]int, len(others))
	for index, s := range others {
		indexByID[s.id] = index
	}
	for _, row := range rows {
		err := assert.NotEmpty(row.id, "session ID must not be empty")
		if err != nil {
			return err
		}
		known := live
		if live == nil || row.id != live.id {
			index, ok := indexByID[row.id]
			if !ok {
				indexByID[row.id] = len(others)
				others = append(others, row)
				continue
			}
			known = &others[index]
		}
		err = assert.Equal(known.anchorAtNs, row.anchorAtNs, "the anchor of a session never changes")
		if err != nil {
			return err
		}
		known.consumedNs = max(known.consumedNs, row.consumedNs)
	}

	next, err := computeResetTimes(others, live)
	if err != nil {
		return err
	}
	e.live = live
	e.others = others
	e.resetTimes = next
	return nil
}

// check decides a request without changing the entry.
func (e *entry) check(nowNs int64, rate Rate, units int64) (decision, error) {
	return decide(e.resetTimes, e.live != nil, nowNs, rate, units)
}

// spend writes a decision from `check` into the sessions. A new session gets
// `newSessionID`, which must be new: the upsert keeps the larger
// `consumed_ns`, so a reused ID would hide the new session. On an error spend
// changes nothing.
func (e *entry) spend(spent decision, newSessionID string) error {
	if !spent.allowed || spent.consumedNs == 0 {
		return nil
	}
	if spent.createNewSessionAtNs == 0 {
		e.live.consumedNs += spent.consumedNs
	} else {
		err := assert.All(
			assert.NotEmpty(newSessionID, "new session ID must not be empty"),
			assert.False(e.isKnown(newSessionID), "new session ID must not be known"),
		)
		if err != nil {
			return err
		}
		e.endLiveSession()
		e.live = &session{
			id:         newSessionID,
			anchorAtNs: spent.createNewSessionAtNs,
			consumedNs: spent.consumedNs,
		}
	}
	e.resetTimes = resetTimesAfter(e.resetTimes, spent)
	return nil
}

// endLiveSession freezes the live session, for example when its lifetime
// ended. It keeps counting.
func (e *entry) endLiveSession() {
	if e.live == nil {
		return
	}
	e.others = append(e.others, *e.live)
	e.live = nil
	// `reset_at` does not change. No session is live, so no candidate is
	// raised by one.
	e.resetTimes.maxCandidateRaisedByLiveSessionNs = 0
}

// isKnown is O(n), but it only runs when a new session starts.
func (e *entry) isKnown(id string) bool {
	return (e.live != nil && id == e.live.id) || slices.ContainsFunc(e.others, func(s session) bool {
		return s.id == id
	})
}
