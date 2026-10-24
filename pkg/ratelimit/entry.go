package ratelimit

import (
	"maps"

	"github.com/unkeyed/unkey/pkg/assert"
)

// entry holds the sessions of one rate limit on this pod and their reset
// times. Only its methods change the sessions, so the reset times always match
// them.
// The caller holds the shard lock.
type entry struct {
	// sessions holds every session this pod knows, by ID: its own and its
	// peers'. Sessions are never removed.
	sessions map[string]session

	// liveID is the ID of this pod's live session, or "" if it has none.
	liveID string

	// resetTimes caches `reset_at` and the two other times that `decide` needs,
	// so a check is O(1) instead of O(n log n) for `computeResetTimes`.
	resetTimes resetTimes
}

// mergeRows adds database rows and keeps the larger `consumedNs` per session,
// so a stale replica never moves a session back. It never removes a session,
// and on an error it changes nothing.
func (e *entry) mergeRows(rows []session) error {
	next := maps.Clone(e.sessions)
	if next == nil {
		next = make(map[string]session, len(rows))
	}
	for _, row := range rows {
		err := assert.NotEmpty(row.id, "session ID must not be empty")
		if err != nil {
			return err
		}
		known, ok := next[row.id]
		if ok {
			err = assert.Equal(known.anchorAtNs, row.anchorAtNs, "the anchor of a session never changes")
			if err != nil {
				return err
			}
			row.consumedNs = max(known.consumedNs, row.consumedNs)
		}
		next[row.id] = row
	}

	nextResetTimes, err := computeResetTimes(next, e.liveID)
	if err != nil {
		return err
	}
	e.sessions = next
	e.resetTimes = nextResetTimes
	return nil
}

// check decides a request without changing the entry.
func (e *entry) check(nowNs int64, rate Rate, units int64) (decision, error) {
	return decide(e.resetTimes, nowNs, rate, units)
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
		live := e.sessions[e.liveID]
		live.consumedNs += spent.consumedNs
		e.sessions[e.liveID] = live
	} else {
		_, known := e.sessions[newSessionID]
		err := assert.All(
			assert.NotEmpty(newSessionID, "new session ID must not be empty"),
			assert.False(known, "new session ID must not be known"),
		)
		if err != nil {
			return err
		}
		if e.sessions == nil {
			e.sessions = make(map[string]session)
		}
		e.sessions[newSessionID] = session{
			id:         newSessionID,
			anchorAtNs: spent.createNewSessionAtNs,
			consumedNs: spent.consumedNs,
		}
		e.liveID = newSessionID
	}
	e.resetTimes = resetTimesAfter(e.resetTimes, spent)
	return nil
}

// endLiveSession freezes the live session, for example when its lifetime
// ended. It keeps counting.
func (e *entry) endLiveSession() {
	e.liveID = ""
	// `reset_at` does not change. No session is live, so no candidate is
	// raised by one.
	e.resetTimes.maxCandidateRaisedByLiveSessionNs = 0
}
