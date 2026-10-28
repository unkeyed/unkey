package ratelimit

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// timedRequest is one admitted request for the reference level.
type timedRequest struct {
	atMs  int64
	units int64
	podID string
}

// referenceLevel is the level of one machine that saw `requests`, computed
// term by term without periods or scaled sums.
func referenceLevel(requests []timedRequest, nowMs int64, duration time.Duration) float64 {
	level := 0.0
	for _, r := range requests {
		level += float64(r.units) * math.Exp(-float64(nowMs-r.atMs)/float64(duration.Milliseconds()))
	}
	return level
}

// rowsOf builds the rows that pods write for `requests`, adding the requests
// of each row in the order given.
func rowsOf(requests []timedRequest, duration time.Duration) map[rowKey]row {
	periodMs := periodLengthMs(duration)
	rows := make(map[rowKey]row)
	for _, r := range requests {
		key := rowKey{periodStartMs: periodStartMs(r.atMs, periodMs), podID: r.podID}
		stored := rows[key]
		stored.scaledSum += float64(r.units) * growthSince(r.atMs, key.periodStartMs, duration)
		rows[key] = stored
	}
	return rows
}

// FuzzLevelIsTheSameInAnyOrder guarantees the property that design D rests
// on: however requests are split into pods and periods, and in whatever
// order they are added, the rows give the level of one machine that saw
// every request, up to float rounding.
func FuzzLevelIsTheSameInAnyOrder(f *testing.F) {
	f.Add(uint64(1), uint8(10), uint8(3), int64(time.Minute.Milliseconds()))
	f.Add(uint64(2), uint8(200), uint8(8), int64(time.Second.Milliseconds()))
	f.Add(uint64(3), uint8(50), uint8(1), int64(maxDuration.Milliseconds()))
	f.Fuzz(func(t *testing.T, seed uint64, requestCount uint8, podCount uint8, durationMs int64) {
		duration := time.Duration(durationMs) * time.Millisecond
		if (Rate{Limit: 1, Duration: duration}).Validate() != nil || podCount == 0 {
			t.Skip()
		}
		rng := rand.New(rand.NewPCG(seed, seed))
		startMs := time.Unix(1_700_000_000, 0).UnixMilli()
		spanMs := 3 * periodLengthMs(duration)
		requests := make([]timedRequest, requestCount)
		for i := range requests {
			requests[i] = timedRequest{
				atMs:  startMs + rng.Int64N(spanMs),
				units: 1 + rng.Int64N(1_000),
				podID: string(rune('a' + rng.IntN(int(podCount)))),
			}
		}
		nowMs := startMs + spanMs + rng.Int64N(periodLengthMs(duration))
		want := referenceLevel(requests, nowMs, duration)

		rng.Shuffle(len(requests), func(i, j int) { requests[i], requests[j] = requests[j], requests[i] })
		sum, err := computeLevelSum(rowsOf(requests, duration), periodStartMs(nowMs, periodLengthMs(duration)), duration)
		require.NoError(t, err)
		require.InDelta(t, want, sum.level(nowMs, duration), 1e-12*max(want, 1))
	})
}

func TestComputeLevelSumRejectsInvalidRows(t *testing.T) {
	for name, sum := range map[string]float64{"negative": -1, "infinite": math.Inf(1), "not a number": math.NaN()} {
		t.Run(name, func(t *testing.T) {
			_, err := computeLevelSum(map[rowKey]row{{periodStartMs: 0, podID: "a"}: {scaledSum: sum}}, 0, time.Second)
			require.Error(t, err)
		})
	}
}

func TestDecideAtTheEdges(t *testing.T) {
	nowMs := time.Unix(1_700_000_000, 0).UnixMilli()
	cases := []struct {
		name    string
		rate    Rate
		level   float64
		units   int64
		allowed bool
	}{
		{"zero units on a full limit", Rate{Limit: 10, Duration: time.Second}, 10, 0, true},
		{"zero units over the limit", Rate{Limit: 10, Duration: time.Second}, 12, 0, false},
		{"a level below one unit counts as zero", Rate{Limit: 10, Duration: time.Second}, 0.999, 10, true},
		{"a level of one unit counts", Rate{Limit: 10, Duration: time.Second}, 1, 10, false},
		{"exactly the limit", Rate{Limit: 10, Duration: time.Second}, 4, 6, true},
		{"one over the limit", Rate{Limit: 10, Duration: time.Second}, 4, 7, false},
		{"one over a limit beyond float precision", Rate{Limit: 1 << 60, Duration: time.Second}, 0, 1<<60 + 1, false},
		{"the largest limit", Rate{Limit: math.MaxInt64, Duration: maxDuration}, 0, math.MaxInt64, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			referenceMs := periodStartMs(nowMs, periodLengthMs(tc.rate.Duration))
			current := levelSum{referenceMs: referenceMs, sum: tc.level * growthSince(nowMs, referenceMs, tc.rate.Duration)}
			d, err := decide(current, nowMs, tc.rate, tc.units)
			require.NoError(t, err)
			require.Equal(t, tc.allowed, d.allowed)
			require.InEpsilon(t, max(tc.level, 1e-300), max(d.levelBefore, 1e-300), 1e-12)
		})
	}

	rate := Rate{Limit: 10, Duration: time.Second}
	referenceMs := periodStartMs(nowMs, periodLengthMs(rate.Duration))
	_, err := decide(levelSum{referenceMs: referenceMs - periodLengthMs(rate.Duration), sum: 0}, nowMs, rate, 1)
	require.Error(t, err, "the level must be referenced to the current period")
	_, err = decide(levelSum{referenceMs: referenceMs, sum: 0}, nowMs, rate, -1)
	require.Error(t, err)
	d, err := decide(levelSum{referenceMs: referenceMs, sum: 0}, nowMs, Rate{Limit: math.MaxInt64, Duration: time.Second}, math.MaxInt64)
	require.NoError(t, err)
	require.True(t, d.allowed)
}

// TestPeriodsKeepFloatPrecision guarantees the choice of `periodDurations`:
// a full limit admitted at the very end of a period, after the most growth,
// still decides to the unit. Above 2^53 units float64 cannot add one unit,
// so the largest limits decide to about 1e-16 of the limit instead.
func TestPeriodsKeepFloatPrecision(t *testing.T) {
	for _, rate := range []Rate{{Limit: 2, Duration: minDuration}, {Limit: 1 << 40, Duration: maxDuration}, {Limit: 1 << 50, Duration: time.Minute}} {
		periodMs := periodLengthMs(rate.Duration)
		atMs := periodStartMs(time.Unix(1_700_000_000, 0).UnixMilli(), periodMs) + periodMs - 1
		e := newEntry(rate.Duration)
		d, err := e.check(atMs, rate, rate.Limit)
		require.NoError(t, err)
		require.True(t, d.allowed)
		e.apply(d, "pod")
		require.Less(t, e.rows[rowKey{periodStartMs: periodStartMs(atMs, periodMs), podID: "pod"}].scaledSum, float64(rate.Limit)*math.Exp(periodDurations)*1.000001)

		d, err = e.check(atMs, rate, 1)
		require.NoError(t, err)
		require.False(t, d.allowed, "%+v", rate)
		require.InEpsilon(t, float64(rate.Limit), d.levelBefore, 1e-15)
	}
}

func TestRateReportsRemainingAndTheNextUnit(t *testing.T) {
	rate := Rate{Limit: 100, Duration: time.Minute}
	require.Equal(t, int64(100), rate.remaining(0.99, 0))
	require.Equal(t, int64(99), rate.remaining(0.5, 1), "the cut applies before the cost")
	require.Equal(t, int64(98), rate.remaining(1.5, 0))
	require.Equal(t, int64(0), rate.remaining(150, 0))
	require.Equal(t, int64(0), rate.remaining(99.5, 1))

	require.Zero(t, rate.nextUnitAfterMs(99))
	require.Equal(t, afterMs(rate.Duration, 100, 99).Milliseconds(), rate.nextUnitAfterMs(100))
	require.Equal(t, afterMs(rate.Duration, 150, 99).Milliseconds(), rate.nextUnitAfterMs(150))

	one := Rate{Limit: 1, Duration: time.Minute}
	require.Zero(t, one.nextUnitAfterMs(0.999))
	require.Equal(t, int64(1), one.nextUnitAfterMs(1), "a limit of 1 waits until the level is below one unit")
	require.Equal(t, afterMs(one.Duration, 2, 1).Milliseconds()+1, one.nextUnitAfterMs(2))
}

func TestRateValidate(t *testing.T) {
	require.NoError(t, Rate{Limit: 1, Duration: time.Second}.Validate())
	require.NoError(t, Rate{Limit: math.MaxInt64, Duration: maxDuration}.Validate())
	require.Error(t, Rate{Limit: 0, Duration: time.Second}.Validate())
	require.Error(t, Rate{Limit: 1, Duration: time.Second - 1}.Validate())
	require.Error(t, Rate{Limit: 1, Duration: maxDuration + 1}.Validate())
}
