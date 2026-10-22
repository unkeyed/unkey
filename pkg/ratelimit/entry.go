package ratelimit

import (
	"maps"

	"github.com/unkeyed/unkey/pkg/assert"
)

// entry holds the sessions of one rate limit on this pod and their reset
// times. Only its methods change the sessions, so the reset times always match
// them.
// entry has no lock of its own: every method needs the shard lock of its
// fingerprint.
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

	// writtenNs holds the consumed time that the store acknowledged for each
	// of this pod's sessions that may still need a write: the live session,
	// and frozen sessions with units that are not written yet.
	writtenNs map[string]int64

	// rowsReadAtNs is when the last read of peer rows started, whether it
	// succeeded or not. Zero means never.
	rowsReadAtNs int64

	// readDone is set while a read of peer rows runs, and closed when it
	// has finished.
	readDone chan struct{}
}

// mergeRows adds database rows and keeps the larger `consumedNs` per session,
// so a stale replica never moves a session back. It never removes a session,
// and on an error it changes nothing.
// The caller must hold the shard lock of the entry.
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

// check decides a request without changing the entry. The caller must hold
// the shard lock of the entry.
func (e *entry) check(nowNs int64, rate Rate, units int64) (decision, error) {
	return decide(e.resetTimes, nowNs, rate, units)
}

// apply writes a decision from `check` into the sessions. A new session gets
// `newSessionID`, which must be new: the upsert keeps the larger
// `consumed_ns`, so a reused ID would hide the new session. On an error apply
// changes nothing. The caller must hold the shard lock of the entry.
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
		if e.writtenNs == nil {
			e.writtenNs = make(map[string]int64)
		}
		e.writtenNs[newSessionID] = 0
	}
	e.resetTimes = resetTimesAfter(e.resetTimes, d)
	return nil
}

// endLiveSession stops new requests from going into the live session, for
// example when its lifetime ended. The session stays in `sessions`, and its
// consumed time still counts toward `reset_at`. The next allowed request
// starts a new session. The caller must hold the shard lock of the entry.
func (e *entry) endLiveSession() {
	e.liveID = ""
	// `reset_at` does not change. No session is live, so no candidate is
	// raised by one.
	e.resetTimes.maxCandidateRaisedByLiveSessionNs = 0
}

// forgetWrittenSessions stops tracking this pod's sessions that need no
// more writes: frozen sessions that are fully written, and sessions past
// their `compact_after`. Cleanup may already have merged such a row, so a
// late write would count its units twice.
// The caller must hold the shard lock of the entry.
func (e *entry) forgetWrittenSessions(nowNs int64) {
	for id, writtenNs := range e.writtenNs {
		s := e.sessions[id]
		pastCompactAfter := nowNs >= s.anchorAtNs+compactAfter.Nanoseconds()
		if pastCompactAfter || (id != e.liveID && s.consumedNs <= writtenNs) {
			delete(e.writtenNs, id)
		}
	}
}

// unwrittenSessions returns this pod's tracked sessions whose consumed time
// the store has not acknowledged. It does not change the entry.
// The caller must hold the shard lock of the entry.
func (e *entry) unwrittenSessions() []session {
	var unwritten []session
	for id, writtenNs := range e.writtenNs {
		s := e.sessions[id]
		if s.consumedNs > writtenNs {
			unwritten = append(unwritten, s)
		}
	}
	return unwritten
}

// markWritten records that the store acknowledged `consumedNs` for a session.
// The caller must hold the shard lock of the entry.
func (e *entry) markWritten(id string, consumedNs int64) {
	writtenNs, ok := e.writtenNs[id]
	if ok {
		e.writtenNs[id] = max(writtenNs, consumedNs)
	}
}
