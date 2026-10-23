package gcra

import (
	"math"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// burstToleranceDivisor sets the burst tolerance to one tenth of Limit.
const burstToleranceDivisor = 10

// Rate allows Limit units per Duration. At most a tenth of Limit fits at
// once, except that a single request of up to Limit units fits when nothing is
// outstanding.
type Rate struct {
	Limit    int64
	Duration time.Duration
}

// State is one writer's values for a name, or the combination of all its peers'
// values: SpentNs summed and FreshStartNs maxed. Both only grow.
type State struct {
	// SpentNs is the total time admitted, in nanoseconds.
	SpentNs int64

	// FreshStartNs is the highest proposed baseline, in Unix nanoseconds.
	FreshStartNs int64
}

// Result is the outcome of one decision.
type Result struct {
	// Allowed reports whether the request fits.
	Allowed bool

	// Remaining is how many more units fit right now.
	Remaining int64

	// Current is how many units are in use, including the request's cost.
	Current int64

	// ResetAt is when every unit is free again, never before the decision time.
	ResetAt time.Time
}

// Decision is a [Result] together with the writer's state to commit for it.
type Decision struct {
	Result

	// Writer is the writer's state after the request. It equals the input state
	// unless the request was allowed and had a cost.
	Writer State

	// Clamped reports that Writer raised the fresh start, because the name had
	// been idle.
	Clamped bool
}

// Decide is the GCRA decision for one request of cost units at now, given the
// writer's own state and its peers' combined state. It does not change
// anything; the caller commits Decision.Writer.
//
// reset_at is MAX(fresh start) + SUM(spent). A name whose reset_at lies in the
// past is idle, and the writer raises its fresh start to now - SUM(spent), so
// idle time is forgotten instead of banked. A request fits if reset_at, moved
// by its cost, stays within the burst tolerance of now. A request that costs
// more than the burst tolerance fits only when nothing is outstanding, and
// its cost beyond the burst tolerance delays later requests.
//
// Time is priced per unit: a unit costs ceil(Duration / Limit) nanoseconds and
// the burst tolerance is ceil(Limit / 10) units. Rounding per unit rather than
// per request keeps a request of Limit units admissible for every Limit.
func Decide(now time.Time, rate Rate, cost int64, writer, peers State) (Decision, error) {
	if err := assert.All(
		assert.Greater(rate.Limit, 0, "gcra limit must be greater than zero"),
		assert.Greater(rate.Duration, 0, "gcra duration must be greater than zero"),
		assert.GreaterOrEqual(cost, 0, "gcra cost must not be negative"),
		assert.GreaterOrEqual(writer.SpentNs, 0, "gcra spent must not be negative"),
		assert.GreaterOrEqual(peers.SpentNs, 0, "gcra spent must not be negative"),
	); err != nil {
		return Decision{}, err
	}

	nowNs := now.UnixNano()
	durationNs := rate.Duration.Nanoseconds()
	unitNs := durationNs / rate.Limit
	if durationNs%rate.Limit != 0 {
		unitNs++
	}
	if _, err := checkedMul(rate.Limit, unitNs); err != nil {
		return Decision{}, err
	}
	burstToleranceNs := ceilDiv(rate.Limit, burstToleranceDivisor) * unitNs
	totalSpentNs, err := checkedAdd(peers.SpentNs, writer.SpentNs)
	if err != nil {
		return Decision{}, err
	}
	resetAtNs, err := checkedAdd(max(peers.FreshStartNs, writer.FreshStartNs), totalSpentNs)
	if err != nil {
		return Decision{}, err
	}

	proposed := writer
	clamped := false
	if resetAtNs < nowNs {
		// baseline + totalSpent < now, so now - totalSpent cannot underflow.
		proposed.FreshStartNs = nowNs - totalSpentNs
		resetAtNs = nowNs
		clamped = true
	}
	aheadNs := resetAtNs - nowNs

	current := saturatingAdd(ceilDiv(aheadNs, unitNs), cost)
	allowed := false
	if cost <= rate.Limit {
		// cost <= Limit keeps cost * unitNs <= Limit * unitNs, so it cannot overflow.
		allowed = saturatingAdd(aheadNs, cost*unitNs) <= max(burstToleranceNs, cost*unitNs)
	}
	if !allowed {
		return Decision{
			Result: Result{
				Allowed:   false,
				Remaining: max(0, burstToleranceNs-aheadNs) / unitNs,
				Current:   current,
				ResetAt:   time.Unix(0, resetAtNs),
			},
			Writer:  writer,
			Clamped: false,
		}, nil
	}

	costNs := cost * unitNs
	if cost > 0 {
		if proposed.SpentNs, err = checkedAdd(proposed.SpentNs, costNs); err != nil {
			return Decision{}, err
		}
	} else {
		proposed, clamped = writer, false
	}
	aheadNs += costNs
	resetAfterNs, err := checkedAdd(nowNs, aheadNs)
	if err != nil {
		return Decision{}, err
	}
	return Decision{
		Result: Result{
			Allowed:   true,
			Remaining: max(0, burstToleranceNs-aheadNs) / unitNs,
			Current:   current,
			ResetAt:   time.Unix(0, resetAfterNs),
		},
		Writer:  proposed,
		Clamped: clamped,
	}, nil
}

func checkedAdd(a, b int64) (int64, error) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, fault.New("gcra arithmetic overflow", fault.Internal("adding nanosecond values overflows int64"))
	}
	return a + b, nil
}

func checkedMul(a, b int64) (int64, error) {
	if a > 0 && b > 0 && a > math.MaxInt64/b {
		return 0, fault.New("gcra arithmetic overflow", fault.Internal("multiplying nanosecond values overflows int64"))
	}
	return a * b, nil
}

func saturatingAdd(a, b int64) int64 {
	if sum, err := checkedAdd(a, b); err == nil {
		return sum
	}
	return math.MaxInt64
}

// ceilDiv divides a non-negative a by a positive b, rounding up.
func ceilDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 {
		q++
	}
	return q
}
