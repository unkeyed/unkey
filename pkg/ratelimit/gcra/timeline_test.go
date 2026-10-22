package gcra_test

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
	"github.com/unkeyed/unkey/pkg/ratelimit/gcra"
)

// TestTimelineChargesRequestsWhenAPeerSessionIsNewer guarantees that a request
// is charged in full when a peer started a session after this pod's live
// session. Adding the request to the older live session would leave reset_at
// unchanged, because the peer's newer session holds the largest candidate.
func TestTimelineChargesRequestsWhenAPeerSessionIsNewer(t *testing.T) {
	rate := gcra.Rate{Limit: 60, Duration: 60}
	sessions := []gcra.Session{
		{ID: "own", AnchorAtNs: 0, ConsumedNs: 1},
		{ID: "peer", AnchorAtNs: 50, ConsumedNs: 60},
	}
	timeline, err := gcra.NewTimeline(sessions, "own")
	require.NoError(t, err)
	require.Equal(t, int64(110), timeline.ResetAtNs())

	decision, err := timeline.Check(55, rate, 1)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, int64(111), decision.ResetAtNs)
	require.True(t, decision.StartsSession)
	require.Equal(t, int64(55), decision.SessionAnchorAtNs)
}

// TestTimelineNeverWrapsAround guarantees that sums beyond int64 return an
// error. A wrapped reset_at would lie far in the past and open the limit.
func TestTimelineNeverWrapsAround(t *testing.T) {
	_, err := gcra.NewTimeline([]gcra.Session{
		{ID: "a", AnchorAtNs: math.MaxInt64 - 5, ConsumedNs: 3},
		{ID: "b", AnchorAtNs: math.MaxInt64 - 5, ConsumedNs: 3},
	}, "")
	require.Error(t, err)

	timeline, err := gcra.NewTimeline([]gcra.Session{
		{ID: "a", AnchorAtNs: math.MaxInt64 - 5, ConsumedNs: 3},
	}, "a")
	require.NoError(t, err)
	_, err = timeline.Check(0, gcra.Rate{Limit: 1, Duration: 10}, 1)
	require.Error(t, err)
}

// FuzzTimelineMatchesSingleMachineGCRA guarantees the core property of the
// limiter math. For any history of peer sessions, paused or ended sessions,
// and requests, every decision equals single-machine GCRA over the same
// sessions: reset_at moves to max(reset_at, now) + cost, and the request fits
// if that stays within one Duration of now.
//
// It also guarantees that the sessions the caller writes reproduce that
// reset_at by the formula over all rows, that Apply gives the same Timeline as
// a rebuild from those rows, and that a request starts a new session only when
// the live session cannot take it exactly.
func FuzzTimelineMatchesSingleMachineGCRA(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		rate := gcra.Rate{
			Limit:    int64(c.Uint8()%20) + 1,
			Duration: time.Duration(c.Uint8()%200) + 1,
		}
		var sessions []gcra.Session
		liveSessionID := ""
		var timeline gcra.Timeline
		var nowNs int64 = 1_000

		for range 100 {
			switch c.Uint8() % 5 {
			case 0:
				nowNs += int64(c.Uint8())
			case 1:
				sessions = append(sessions, gcra.Session{
					ID:         fmt.Sprintf("peer_%d", len(sessions)),
					AnchorAtNs: max(0, nowNs+int64(c.Int8())),
					ConsumedNs: int64(c.Uint8()),
				})
				timeline = rebuild(t, sessions, liveSessionID)
			case 2:
				if len(sessions) == 0 {
					continue
				}
				index := int(c.Uint8()) % len(sessions)
				if sessions[index].ID == liveSessionID {
					continue
				}
				sessions[index].ConsumedNs += int64(c.Uint8())
				timeline = rebuild(t, sessions, liveSessionID)
			case 3:
				timeline.EndLiveSession()
				liveSessionID = ""
				require.Equal(t, rebuild(t, sessions, liveSessionID), timeline)
			case 4:
				cost := int64(c.Uint8()) % (rate.Limit + 2)
				sessions, liveSessionID = requestMatchesSingleMachine(t, &timeline, sessions, liveSessionID, rate, nowNs, cost)
				require.Equal(t, rebuild(t, sessions, liveSessionID), timeline)
			}
		}
	})
}

// requestMatchesSingleMachine checks one request against single-machine GCRA,
// applies it to the timeline, and writes it into the sessions the way the
// limiter writes its rows. It returns the sessions and the live session ID.
func requestMatchesSingleMachine(t *testing.T, timeline *gcra.Timeline, sessions []gcra.Session, liveSessionID string, rate gcra.Rate, nowNs, cost int64) ([]gcra.Session, string) {
	t.Helper()
	before := resetAtByFormula(sessions)
	require.Equal(t, before, timeline.ResetAtNs())

	decision, err := timeline.Check(nowNs, rate, cost)
	require.NoError(t, err)

	costNs := ceilDiv(cost*int64(rate.Duration), rate.Limit)
	after := max(before, nowNs) + costNs
	allowed := cost <= rate.Limit && after-nowNs <= int64(rate.Duration)
	require.Equal(t, allowed, decision.Allowed)
	if !allowed || cost == 0 {
		require.Equal(t, before, decision.ResetAtNs)
		require.False(t, decision.StartsSession)
		require.Equal(t, remainingBySearch(rate, nowNs, before), decision.Remaining)
		timeline.Apply(decision)
		return sessions, liveSessionID
	}

	require.Equal(t, after, decision.ResetAtNs)
	require.Equal(t, costNs, decision.CostNs)
	require.Equal(t, remainingBySearch(rate, nowNs, after), decision.Remaining)

	withLive := slices.Clone(sessions)
	liveCanTakeIt := false
	for i := range withLive {
		if liveSessionID != "" && withLive[i].ID == liveSessionID {
			withLive[i].ConsumedNs += costNs
			liveCanTakeIt = resetAtByFormula(withLive) == after
		}
	}
	require.Equal(t, !liveCanTakeIt, decision.StartsSession, "a new session starts exactly when the live session cannot take the request")

	timeline.Apply(decision)
	if decision.StartsSession {
		liveSessionID = fmt.Sprintf("own_%d", len(sessions))
		sessions = append(sessions, gcra.Session{
			ID:         liveSessionID,
			AnchorAtNs: decision.SessionAnchorAtNs,
			ConsumedNs: decision.CostNs,
		})
	} else {
		sessions = withLive
	}
	require.Equal(t, after, resetAtByFormula(sessions))
	return sessions, liveSessionID
}

func rebuild(t *testing.T, sessions []gcra.Session, liveSessionID string) gcra.Timeline {
	t.Helper()
	timeline, err := gcra.NewTimeline(sessions, liveSessionID)
	require.NoError(t, err)
	return timeline
}

// resetAtByFormula computes reset_at straight from its definition, in O(n²).
func resetAtByFormula(sessions []gcra.Session) int64 {
	var resetAtNs int64
	for _, r := range sessions {
		candidateNs := r.AnchorAtNs
		for _, s := range sessions {
			if s.AnchorAtNs >= r.AnchorAtNs {
				candidateNs += s.ConsumedNs
			}
		}
		resetAtNs = max(resetAtNs, candidateNs)
	}
	return resetAtNs
}

// remainingBySearch returns the largest number of units that fit at nowNs.
func remainingBySearch(rate gcra.Rate, nowNs, resetAtNs int64) int64 {
	for units := rate.Limit; units > 0; units-- {
		if max(resetAtNs, nowNs)+ceilDiv(units*int64(rate.Duration), rate.Limit)-nowNs <= int64(rate.Duration) {
			return units
		}
	}
	return 0
}

func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}
