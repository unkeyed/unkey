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

	// liveSessionEndsAtNs is when the live session ends, even if the rate
	// limit has no pause. Only the limiter sets it.
	liveSessionEndsAtNs int64
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

// apply writes a decision from `check` into the sessions. A new session gets
// `newSessionID`, which must be new: the upsert keeps the larger
// `consumed_ns`, so a reused ID would hide the new session. On an error apply
// changes nothing.
func (e *entry) apply(d decision, newSessionID string) error {
	if !d.allowed || d.consumedNs == 0 {
		return nil
	}
	if d.createNewSessionAtNs == 0 {
		live := e.sessions[e.liveID]
		live.consumedNs += d.consumedNs
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
			anchorAtNs: d.createNewSessionAtNs,
			consumedNs: d.consumedNs,
		}
		e.liveID = newSessionID
	}
	e.resetTimes = resetTimesAfter(e.resetTimes, d)
	return nil
}

// endLiveSession stops new requests from going into the live session, for
// example when its lifetime ended. The session stays in `sessions`, and its
// consumed time still counts toward `reset_at`. The next allowed request
// starts a new session.
func (e *entry) endLiveSession() {
	e.liveID = ""
	// `reset_at` does not change. No session is live, so no candidate is
	// raised by one.
	e.resetTimes.maxCandidateRaisedByLiveSessionNs = 0
}
