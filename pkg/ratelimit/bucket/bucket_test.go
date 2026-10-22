package bucket_test

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
	"github.com/unkeyed/unkey/pkg/ratelimit/bucket"
)

// TestBucketChargesRequestsWhenAPeerSessionIsNewer guarantees that a request
// is charged in full when a peer started a session after this pod's live
// session. Adding the request to the older live session would leave reset_at
// unchanged, because the peer's newer session holds the largest candidate.
func TestBucketChargesRequestsWhenAPeerSessionIsNewer(t *testing.T) {
	rate := bucket.Rate{Limit: 60, Duration: 60}
	sessions := []bucket.Session{
		{ID: "own", AnchorAtNs: 100, ConsumedNs: 1},
		{ID: "peer", AnchorAtNs: 150, ConsumedNs: 60},
	}
	b, err := bucket.New(sessions, "own")
	require.NoError(t, err)
	require.Equal(t, int64(210), b.ResetAtNs())

	decision, err := b.Check(155, rate, 1)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, int64(211), decision.ResetAtNs)
	require.Equal(t, int64(155), decision.CreateNewSessionAtNs)
}

// TestBucketNeverWrapsAround guarantees that sums beyond int64 return an
// error. A wrapped reset_at would lie far in the past and open the limit.
func TestBucketNeverWrapsAround(t *testing.T) {
	_, err := bucket.New([]bucket.Session{
		{ID: "a", AnchorAtNs: math.MaxInt64 - 5, ConsumedNs: 3},
		{ID: "b", AnchorAtNs: math.MaxInt64 - 5, ConsumedNs: 3},
	}, "")
	require.Error(t, err)

	b, err := bucket.New([]bucket.Session{
		{ID: "a", AnchorAtNs: math.MaxInt64 - 5, ConsumedNs: 3},
	}, "a")
	require.NoError(t, err)
	_, err = b.Check(1, bucket.Rate{Limit: 1, Duration: 10}, 1)
	require.Error(t, err)
}

// FuzzBucketMatchesSingleMachineGCRA guarantees the core property of the
// limiter math. For any history of peer sessions, paused or ended sessions,
// and requests, every decision equals single-machine GCRA over the same
// sessions: reset_at moves to max(reset_at, now) + units × Duration / Limit, and the request fits
// if that stays within one Duration of now.
//
// It also guarantees that the sessions the caller writes reproduce that
// reset_at by the formula over all rows, that Apply gives the same Bucket as
// a rebuild from those rows, and that a request starts a new session only when
// the live session cannot take it exactly.
func FuzzBucketMatchesSingleMachineGCRA(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		rate := bucket.Rate{
			Limit:    int64(c.Uint8()%20) + 1,
			Duration: time.Duration(c.Uint8()%200) + 1,
		}
		var sessions []bucket.Session
		liveSessionID := ""
		var b bucket.Bucket
		var nowNs int64 = 1_000

		for range 100 {
			switch c.Uint8() % 5 {
			case 0:
				nowNs += int64(c.Uint8())
			case 1:
				sessions = append(sessions, bucket.Session{
					ID:         fmt.Sprintf("peer_%d", len(sessions)),
					AnchorAtNs: max(0, nowNs+int64(c.Int8())),
					ConsumedNs: int64(c.Uint8()),
				})
				b = rebuild(t, sessions, liveSessionID)
			case 2:
				if len(sessions) == 0 {
					continue
				}
				index := int(c.Uint8()) % len(sessions)
				if sessions[index].ID == liveSessionID {
					continue
				}
				sessions[index].ConsumedNs += int64(c.Uint8())
				b = rebuild(t, sessions, liveSessionID)
			case 3:
				b.EndLiveSession()
				liveSessionID = ""
				require.Equal(t, rebuild(t, sessions, liveSessionID), b)
			case 4:
				units := int64(c.Uint8()) % (rate.Limit + 2)
				sessions, liveSessionID = requestMatchesSingleMachine(t, &b, sessions, liveSessionID, rate, nowNs, units)
				require.Equal(t, rebuild(t, sessions, liveSessionID), b)
			}
		}
	})
}

// requestMatchesSingleMachine checks one request against single-machine GCRA,
// applies it to the bucket, and writes it into the sessions the way the
// limiter writes its rows. It returns the sessions and the live session ID.
func requestMatchesSingleMachine(t *testing.T, b *bucket.Bucket, sessions []bucket.Session, liveSessionID string, rate bucket.Rate, nowNs, units int64) ([]bucket.Session, string) {
	t.Helper()
	before := resetAtByFormula(sessions)
	require.Equal(t, before, b.ResetAtNs())

	decision, err := b.Check(nowNs, rate, units)
	require.NoError(t, err)

	consumedNs := ceilDiv(units*rate.Duration.Nanoseconds(), rate.Limit)
	after := max(before, nowNs) + consumedNs
	allowed := units <= rate.Limit && after-nowNs <= rate.Duration.Nanoseconds()
	require.Equal(t, allowed, decision.Allowed)
	if !allowed || units == 0 {
		require.Equal(t, before, decision.ResetAtNs)
		require.Zero(t, decision.CreateNewSessionAtNs)
		require.Equal(t, remainingBySearch(rate, nowNs, before), decision.Remaining)
		b.Apply(decision)
		return sessions, liveSessionID
	}

	require.Equal(t, after, decision.ResetAtNs)
	require.Equal(t, consumedNs, decision.ConsumedNs)
	require.Equal(t, remainingBySearch(rate, nowNs, after), decision.Remaining)

	withLive := slices.Clone(sessions)
	liveCanTakeIt := false
	for i := range withLive {
		if liveSessionID != "" && withLive[i].ID == liveSessionID {
			withLive[i].ConsumedNs += consumedNs
			liveCanTakeIt = resetAtByFormula(withLive) == after
		}
	}
	require.Equal(t, !liveCanTakeIt, decision.CreateNewSessionAtNs != 0, "a new session starts exactly when the live session cannot take the request")

	b.Apply(decision)
	if decision.CreateNewSessionAtNs != 0 {
		liveSessionID = fmt.Sprintf("own_%d", len(sessions))
		sessions = append(sessions, bucket.Session{
			ID:         liveSessionID,
			AnchorAtNs: decision.CreateNewSessionAtNs,
			ConsumedNs: decision.ConsumedNs,
		})
	} else {
		sessions = withLive
	}
	require.Equal(t, after, resetAtByFormula(sessions))
	return sessions, liveSessionID
}

func rebuild(t *testing.T, sessions []bucket.Session, liveSessionID string) bucket.Bucket {
	t.Helper()
	b, err := bucket.New(sessions, liveSessionID)
	require.NoError(t, err)
	return b
}

// resetAtByFormula computes reset_at straight from its definition, in O(n²).
func resetAtByFormula(sessions []bucket.Session) int64 {
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
func remainingBySearch(rate bucket.Rate, nowNs, resetAtNs int64) int64 {
	for units := rate.Limit; units > 0; units-- {
		if max(resetAtNs, nowNs)+ceilDiv(units*rate.Duration.Nanoseconds(), rate.Limit)-nowNs <= rate.Duration.Nanoseconds() {
			return units
		}
	}
	return 0
}

func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}
