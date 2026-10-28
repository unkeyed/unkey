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
	atNs  int64
	units int64
	podID string
}

// referenceLevel is the level of one machine that saw `requests`, computed
// term by term without periods or scaled sums.
func referenceLevel(requests []timedRequest, nowNs int64, duration time.Duration) float64 {
	level := 0.0
	for _, r := range requests {
		level += float64(r.units) * math.Exp(-float64(nowNs-r.atNs)/float64(duration.Nanoseconds()))
	}
	return level
}

// rowsOf builds the rows that pods write for `requests`, adding the requests
// of each row in the order given.
func rowsOf(requests []timedRequest, duration time.Duration) map[rowKey]row {
	periodNs := periodLengthNs(duration)
	rows := make(map[rowKey]row)
	for _, r := range requests {
		key := rowKey{periodStartNs: periodStartNs(r.atNs, periodNs), podID: r.podID}
		stored := rows[key]
		stored.scaledSum += float64(r.units) * growthSince(r.atNs, key.periodStartNs, duration)
		rows[key] = stored
	}
	return rows
}

// FuzzLevelIsTheSameInAnyOrder guarantees the property that design D rests
// on: however requests are split into pods and periods, and in whatever
// order they are added, the rows give the level of one machine that saw
// every request, up to float rounding.
func FuzzLevelIsTheSameInAnyOrder(f *testing.F) {
	f.Add(uint64(1), uint8(10), uint8(3), int64(time.Minute))
	f.Add(uint64(2), uint8(200), uint8(8), int64(time.Second))
	f.Add(uint64(3), uint8(50), uint8(1), int64(maxDuration))
	f.Fuzz(func(t *testing.T, seed uint64, requestCount uint8, podCount uint8, durationNs int64) {
		duration := time.Duration(durationNs)
		if (Rate{Limit: 1, Duration: duration}).Validate() != nil || podCount == 0 {
			t.Skip()
		}
		rng := rand.New(rand.NewPCG(seed, seed))
		startNs := time.Unix(1_700_000_000, 0).UnixNano()
		spanNs := 3 * periodLengthNs(duration)
		requests := make([]timedRequest, requestCount)
		for i := range requests {
			requests[i] = timedRequest{
				atNs:  startNs + rng.Int64N(spanNs),
				units: 1 + rng.Int64N(1_000),
				podID: string(rune('a' + rng.IntN(int(podCount)))),
			}
		}
		nowNs := startNs + spanNs + rng.Int64N(periodLengthNs(duration))
		want := referenceLevel(requests, nowNs, duration)

		rng.Shuffle(len(requests), func(i, j int) { requests[i], requests[j] = requests[j], requests[i] })
		sum, err := computeLevelSum(rowsOf(requests, duration), periodStartNs(nowNs, periodLengthNs(duration)), duration)
		require.NoError(t, err)
		require.InDelta(t, want, sum.level(nowNs, duration), 1e-12*max(want, 1))
	})
}

func TestComputeLevelSumRejectsInvalidRows(t *testing.T) {
	for name, sum := range map[string]float64{"negative": -1, "infinite": math.Inf(1), "not a number": math.NaN()} {
		t.Run(name, func(t *testing.T) {
			_, err := computeLevelSum(map[rowKey]row{{periodStartNs: 0, podID: "a"}: {scaledSum: sum}}, 0, time.Second)
			require.Error(t, err)
		})
	}
}

func TestDecideAtTheEdges(t *testing.T) {
	nowNs := time.Unix(1_700_000_000, 0).UnixNano()
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
			referenceNs := periodStartNs(nowNs, periodLengthNs(tc.rate.Duration))
			current := levelSum{referenceNs: referenceNs, sum: tc.level * growthSince(nowNs, referenceNs, tc.rate.Duration)}
			d, err := decide(current, nowNs, tc.rate, tc.units)
			require.NoError(t, err)
			require.Equal(t, tc.allowed, d.allowed)
			require.InEpsilon(t, max(tc.level, 1e-300), max(d.levelBefore, 1e-300), 1e-12)
		})
	}

	rate := Rate{Limit: 10, Duration: time.Second}
	referenceNs := periodStartNs(nowNs, periodLengthNs(rate.Duration))
	_, err := decide(levelSum{referenceNs: referenceNs - periodLengthNs(rate.Duration), sum: 0}, nowNs, rate, 1)
	require.Error(t, err, "the level must be referenced to the current period")
	_, err = decide(levelSum{referenceNs: referenceNs, sum: 0}, nowNs, rate, -1)
	require.Error(t, err)
	d, err := decide(levelSum{referenceNs: referenceNs, sum: 0}, nowNs, Rate{Limit: math.MaxInt64, Duration: time.Second}, math.MaxInt64)
	require.NoError(t, err)
	require.True(t, d.allowed)
}

// TestPeriodsKeepFloatPrecision guarantees the choice of `periodDurations`:
// a full limit admitted at the very end of a period, after the most growth,
// still decides to the unit. Above 2^53 units float64 cannot add one unit,
// so the largest limits decide to about 1e-16 of the limit instead.
func TestPeriodsKeepFloatPrecision(t *testing.T) {
	for _, rate := range []Rate{{Limit: 2, Duration: minDuration}, {Limit: 1 << 40, Duration: maxDuration}, {Limit: 1 << 50, Duration: time.Minute}} {
		periodNs := periodLengthNs(rate.Duration)
		atNs := periodStartNs(time.Unix(1_700_000_000, 0).UnixNano(), periodNs) + periodNs - 1
		e := newEntry(rate.Duration)
		d, err := e.check(atNs, rate, rate.Limit)
		require.NoError(t, err)
		require.True(t, d.allowed)
		e.apply(d, "pod")
		require.Less(t, e.rows[rowKey{periodStartNs: periodStartNs(atNs, periodNs), podID: "pod"}].scaledSum, float64(rate.Limit)*math.Exp(periodDurations)*1.000001)

		d, err = e.check(atNs, rate, 1)
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

	require.Zero(t, rate.nextUnitAfterNs(99))
	require.Equal(t, afterNs(rate.Duration, 100, 99).Nanoseconds(), rate.nextUnitAfterNs(100))
	require.Equal(t, afterNs(rate.Duration, 150, 99).Nanoseconds(), rate.nextUnitAfterNs(150))

	one := Rate{Limit: 1, Duration: time.Minute}
	require.Zero(t, one.nextUnitAfterNs(0.999))
	require.Equal(t, int64(1), one.nextUnitAfterNs(1), "a limit of 1 waits until the level is below one unit")
	require.Equal(t, afterNs(one.Duration, 2, 1).Nanoseconds()+1, one.nextUnitAfterNs(2))
}

func TestRateValidate(t *testing.T) {
	require.NoError(t, Rate{Limit: 1, Duration: time.Second}.Validate())
	require.NoError(t, Rate{Limit: math.MaxInt64, Duration: maxDuration}.Validate())
	require.Error(t, Rate{Limit: 0, Duration: time.Second}.Validate())
	require.Error(t, Rate{Limit: 1, Duration: time.Second - 1}.Validate())
	require.Error(t, Rate{Limit: 1, Duration: maxDuration + 1}.Validate())
}
