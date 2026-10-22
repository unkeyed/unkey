package ratelimit

import (
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
)

// entry holds what this pod knows about one rate limit: its own sessions,
// the peer rows it has read, and their summary. Its methods are the only code
// that changes the sessions, and each of them updates the summary, so the
// summary always matches the sessions.
//
// An entry has no locks; the limiter holds the shard lock while it uses one.
type entry struct {
	// live is this pod's live session: the session that takes its admitted
	// units. Its `id` is empty when the pod has no live session, and then the
	// next admitted request starts a new session.
	live session

	// others are all other sessions: the sessions of peers and this pod's
	// frozen sessions, in no particular order.
	others []session

	summary summary
}

// all returns a copy of all sessions, the live session included, in no
// particular order.
func (e *entry) all() []session {
	all := slices.Clone(e.others)
	if e.live.id != "" {
		all = append(all, e.live)
	}
	return all
}

// mergeRows adds rows read from the database. For a session that is already
// known, it keeps the larger `consumedNs`: memory is never older than the
// database, so a stale replica never moves a session back, and this pod's own
// rows are never counted twice. Then it recomputes the summary.
//
// mergeRows never removes a session. It returns an error, and changes
// nothing, if a row has an empty ID, a known ID with a different anchor, or
// values that `summarize` rejects.
func (e *entry) mergeRows(rows []session) error {
	live := e.live
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
		known := &live
		if row.id != live.id {
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

	all := others
	if live.id != "" {
		all = append(slices.Clone(others), live)
	}
	next, err := summarize(all, live.id)
	if err != nil {
		return err
	}
	e.live = live
	e.others = others
	e.summary = next
	return nil
}

// check decides a request without changing the entry.
func (e *entry) check(nowNs int64, rate Rate, units int64) (decision, error) {
	return decide(e.summary, e.live.id != "", nowNs, rate, units)
}

// spend writes a decision from `check` into the sessions: it adds the
// consumed time to the live session, or freezes the live session and starts
// a new one with `newSessionID`. It does nothing for a decision with zero
// `consumedNs`.
//
// `newSessionID` is only used when a new session starts. spend returns an
// error, and changes nothing, if it is then empty or already known. The
// upsert keeps the larger `consumed_ns`, so a session that started again from
// zero under an old ID would be hidden.
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
		e.live = session{
			id:         newSessionID,
			anchorAtNs: spent.createNewSessionAtNs,
			consumedNs: spent.consumedNs,
		}
	}
	e.summary = summaryAfter(e.summary, spent)
	return nil
}

// endLiveSession freezes the live session, for example when its lifetime
// ended. It keeps counting in full, and the next admitted request starts a
// new session.
func (e *entry) endLiveSession() {
	if e.live.id == "" {
		return
	}
	e.others = append(e.others, e.live)
	e.live = session{id: "", anchorAtNs: 0, consumedNs: 0}
	e.summary = summaryWithoutLiveSession(e.summary)
}

// isKnown reports whether a session with `id` exists. It is O(n), but it
// only runs when a new session starts.
func (e *entry) isKnown(id string) bool {
	return id == e.live.id || slices.ContainsFunc(e.others, func(s session) bool {
		return s.id == id
	})
}
