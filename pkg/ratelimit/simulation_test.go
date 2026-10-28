package ratelimit

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/sim"
)

var (
	simRandomSeeds = flag.Int("sim.random-seeds", 20, "random seeds that TestSimulation runs after the 100 fixed ones")
	simSeed        = flag.String("sim.seed", "", "run TestSimulation with only this seed, as printed in a subtest name")
)

// simTolerance is how far float rounding may move a level in the
// simulation: 1e-9 of the level, and at least 1e-9 units. Sums of up to a
// few thousand products of exponentials round to about 1e-13 of their value,
// so the tolerance has a wide margin and still catches any lost or doubled
// unit.
func simTolerance(level float64) float64 {
	return 1e-9 * max(1, level)
}

// TestSimulation runs 1 to 8 pods against one shared store that delays,
// reorders, fails and loses writes, and serves stale or failed reads from a
// lagging replica. Pods run cleanup passes that delete faded rows, and flush
// forgets entries. Each seed picks its own failure rates. Pod clocks differ
// from the store clock by less than `maxClockSkew`, some pods get few
// requests, and half the seeds send everything to one limit.
//
// The reference is one machine that saw every admitted request: the level
// `Σ cost × e^(−(now − t) / Duration)`, term by term. For every request we
// check:
//
//	local ≤ reference      a pod never charges more than the machine would
//	local ≥ own            a pod never forgets units of a row it still writes
//	allowed = units ≤ Limit and effective(local) + units ≤ Limit
//
// `local` is the level that the pod decides on and `own` the level of its
// admitted requests in its current period, all within `simTolerance`.
//
// On every quiet moment, and at the end, every write is delivered and every
// pod reads again. Then the level of the stored rows plus what cleanup
// deleted must equal the reference, what cleanup deleted must be below
// `negligibleUnits` per deleted row, and every pod must decide on the stored
// level.
//
// Seeds 0 to 99 run on every run and catch regressions. The random seeds
// search new scenarios. Replay a failing seed with `-sim.seed=<seed>`, and
// add it to the fixed seeds once the bug is fixed. The test stops starting
// seeds a minute before the `go test` deadline, 10 minutes by default.
func TestSimulation(t *testing.T) {
	var seeds []sim.Seed
	if *simSeed != "" {
		decoded, err := base64.StdEncoding.DecodeString(*simSeed)
		require.NoError(t, err)
		require.Len(t, decoded, len(sim.Seed{}))
		seeds = append(seeds, sim.Seed(decoded))
	} else {
		for i := range 100 {
			seeds = append(seeds, sim.NewSeedFromInt(i))
		}
		for range *simRandomSeeds {
			seeds = append(seeds, sim.NewSeed())
		}
	}
	deadline, hasDeadline := t.Deadline()
	for i, seed := range seeds {
		if hasDeadline && time.Until(deadline) < time.Minute {
			t.Logf("stopped after %d of %d seeds because the test deadline is near; pass -timeout to run more", i, len(seeds))
			return
		}
		t.Run(seed.String(), func(t *testing.T) {
			runSimulation(t, rand.New(rand.NewChaCha8(seed)))
		})
	}
}

// FuzzSimulation runs the same simulation, but the first random choices
// come from the fuzzer's bytes: the pods, limits, failure rates and the
// first events. Run it with
//
//	mise exec -- go test -run '^$' -fuzz=FuzzSimulation -fuzztime=10m ./pkg/ratelimit/
func FuzzSimulation(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		runSimulation(t, rand.New(newBytesSource(data)))
	})
}

func runSimulation(t *testing.T, rng *rand.Rand) {
	t.Helper()
	var state *simState
	simulation := sim.NewWithRand(rng,
		sim.WithTicks[simState](2_000+rng.Int64N(6_000)),
		sim.WithState(func(rng *rand.Rand) *simState {
			state = newSimState(t, rng)
			return state
		}),
	)
	require.NoError(t, simulation.Run([]sim.Event[simState]{
		requestEvent{}, requestEvent{}, requestEvent{}, flushEvent{}, deliverWriteEvent{}, replicateEvent{}, backgroundEvent{}, advanceTimeEvent{}, settleEvent{}, cleanupEvent{},
	}))
	if len(simulation.Errors) > 0 {
		t.Fatalf("%d failed checks, the first: %v", len(simulation.Errors), simulation.Errors[0])
	}

	state.store.reliable = true
	require.NoError(t, state.settleAndCheck())
	require.Positive(t, state.admittedCount, "the seed admitted no request")
	require.Positive(t, state.exactChecks)
}

// bytesSource returns the fuzzer's bytes as random numbers, then continues
// with a stream seeded from them.
type bytesSource struct {
	data []byte
	rest *rand.ChaCha8
}

func newBytesSource(data []byte) *bytesSource {
	return &bytesSource{data: data, rest: rand.NewChaCha8(sha256.Sum256(data))}
}

func (s *bytesSource) Uint64() uint64 {
	if len(s.data) < 8 {
		return s.rest.Uint64()
	}
	value := binary.LittleEndian.Uint64(s.data)
	s.data = s.data[8:]
	return value
}

type simLimit struct {
	fingerprint [32]byte
	rate        Rate
}

type simPod struct {
	limiter *limiter
	clock   *clock.TestClock
	// background holds the background reads that have not run yet.
	background []func()
	weight     float64
}

type simState struct {
	rng    *rand.Rand
	store  *simStore
	pods   []*simPod
	limits []simLimit

	// admitted holds every admitted request per fingerprint, the input of
	// the reference.
	admitted map[[32]byte][]timedRequest

	admittedCount int
	exactChecks   int
}

func newSimState(t *testing.T, rng *rand.Rand) *simState {
	t.Helper()
	start := time.Unix(1_700_000_000, 0)
	state := &simState{
		rng:      rng,
		store:    newSimStore(rng, clock.NewTestClock(start)),
		admitted: make(map[[32]byte][]timedRequest),
	}
	skews := []time.Duration{0, 50 * time.Millisecond, 2 * time.Second, maxClockSkew - time.Millisecond}
	maxSkew := skews[rng.IntN(len(skews))]
	for range 1 + rng.IntN(8) {
		pod := &simPod{
			clock: clock.NewTestClock(start.Add(time.Duration(rng.Int64N(int64(2*maxSkew+1))) - maxSkew)),
			// Cubed, so some pods get few requests.
			weight: max(rng.Float64()*rng.Float64()*rng.Float64(), 0.001),
		}
		l, err := newLimiter(Config{Clock: pod.clock, Store: state.store, Random: rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))})
		require.NoError(t, err)
		l.runInBackground = func(f func()) { pod.background = append(pod.background, f) }
		pod.limiter = l
		state.pods = append(state.pods, pod)
	}
	durations := []time.Duration{time.Second, 2 * time.Second, 10 * time.Second, time.Minute, time.Hour}
	// Half the seeds put every request on one limit, so pods contend.
	limitCount := 1
	if rng.IntN(2) == 0 {
		limitCount += rng.IntN(3)
	}
	for i := range limitCount {
		state.limits = append(state.limits, simLimit{
			fingerprint: [32]byte{byte(i)},
			rate:        Rate{Limit: 1 + rng.Int64N(20), Duration: durations[rng.IntN(len(durations))]},
		})
	}
	return state
}

// request sends one request and checks it against the reference.
func (st *simState) request(pod *simPod, limit simLimit, units int64) error {
	ctx := context.Background()
	request := Request{Fingerprint: limit.fingerprint, Rate: limit.rate, Units: units}
	nowNs := pod.clock.Now().UnixNano()
	// Refreshed and checked here so the check sees what the decision sees;
	// Allow then finds the rows fresh and changes nothing before deciding.
	pod.limiter.refreshRows(ctx, []Request{request})
	e := pod.limiter.entryOf(limit.fingerprint, limit.rate)
	before, err := e.check(nowNs, limit.rate, 0)
	if err != nil {
		return err
	}
	local := before.levelBefore
	reference := referenceLevel(st.admitted[limit.fingerprint], nowNs, limit.rate.Duration)
	own := st.ownLevel(pod, limit, nowNs)

	result, err := pod.limiter.Allow(ctx, request)
	if err != nil {
		return err
	}
	fits := units <= limit.rate.Limit && limit.rate.fits(local, units)
	describe := fmt.Sprintf("units %d, rate %+v, local %g, reference %g, own %g, now %d", units, limit.rate, local, reference, own, nowNs)
	switch {
	case local > reference+simTolerance(reference):
		return fault.New("the pod charges more than one machine would: " + describe)
	case local < own-simTolerance(own):
		return fault.New("the pod forgot its own units: " + describe)
	case result.Allowed != fits:
		return fault.New("the pod did not follow the rule on what it knows: " + describe)
	}
	if result.Allowed && units > 0 {
		st.admitted[limit.fingerprint] = append(st.admitted[limit.fingerprint], timedRequest{atNs: nowNs, units: units, podID: pod.limiter.podID})
		st.admittedCount++
	}
	return nil
}

// ownLevel is the reference level of the requests that `pod` admitted in
// its current period, the row that it still writes.
func (st *simState) ownLevel(pod *simPod, limit simLimit, nowNs int64) float64 {
	periodNs := periodLengthNs(limit.rate.Duration)
	var own []timedRequest
	for _, r := range st.admitted[limit.fingerprint] {
		if r.podID == pod.limiter.podID && periodStartNs(r.atNs, periodNs) == periodStartNs(nowNs, periodNs) {
			own = append(own, r)
		}
	}
	return referenceLevel(own, nowNs, limit.rate.Duration)
}

// settleAndCheck makes every write visible, lets every pod read again, and
// checks that the stored rows, and each pod, match the reference. The store
// must be reliable.
func (st *simState) settleAndCheck() error {
	for _, pod := range st.pods {
		st.runBackground(pod)
		err := pod.limiter.flush(context.Background())
		if err != nil {
			return err
		}
	}
	st.store.deliverAll()
	st.store.replicateAll()
	st.advance(rowsStaleFor)

	for _, pod := range st.pods {
		for _, limit := range st.limits {
			nowNs := pod.clock.Now().UnixNano()
			duration := limit.rate.Duration
			reference := referenceLevel(st.admitted[limit.fingerprint], nowNs, duration)
			stored := 0.0
			for key, r := range st.store.primary[limit.fingerprint] {
				stored += row{scaledSum: r.ScaledSum}.levelAt(key, nowNs, duration)
			}
			deleted, deletedRows := st.store.deletedLevel(limit.fingerprint, nowNs, duration)
			describe := fmt.Sprintf("rate %+v, reference %g, stored %g, deleted %g in %d rows", limit.rate, reference, stored, deleted, deletedRows)
			if math.Abs(stored+deleted-reference) > simTolerance(reference) {
				return fault.New("the stored rows differ from one machine: " + describe)
			}
			if deleted > negligibleUnits*float64(deletedRows)+simTolerance(reference) {
				return fault.New("cleanup deleted a row that had not faded: " + describe)
			}

			ctx := context.Background()
			pod.limiter.refreshRows(ctx, []Request{{Fingerprint: limit.fingerprint, Rate: limit.rate, Units: 0}})
			d, err := pod.limiter.entryOf(limit.fingerprint, limit.rate).check(nowNs, limit.rate, 0)
			if err != nil {
				return err
			}
			if math.Abs(d.levelBefore-stored) > simTolerance(reference)+deleted {
				return fault.New(fmt.Sprintf("the pod knows every row but decides on %g: %s", d.levelBefore, describe))
			}
			st.exactChecks++
		}
	}
	return nil
}

// advance moves every clock. A background read takes less than
// `readTimeout`, so the work started before runs first.
func (st *simState) advance(d time.Duration) {
	for _, pod := range st.pods {
		st.runBackground(pod)
	}
	for _, pod := range st.pods {
		pod.clock.Tick(d)
	}
	st.store.clock.Tick(d)
}

func (st *simState) runBackground(pod *simPod) {
	for len(pod.background) > 0 {
		work := pod.background[0]
		pod.background = pod.background[1:]
		work()
	}
}

type requestEvent struct{}

func (requestEvent) Name() string { return "request" }

func (requestEvent) Run(rng *rand.Rand, st *simState) error {
	total := 0.0
	for _, pod := range st.pods {
		total += pod.weight
	}
	pick := rng.Float64() * total
	pod := st.pods[len(st.pods)-1]
	for _, candidate := range st.pods {
		pick -= candidate.weight
		if pick < 0 {
			pod = candidate
			break
		}
	}
	limit := st.limits[rng.IntN(len(st.limits))]
	units := rng.Int64N(limit.rate.Limit + 2)
	if rng.IntN(4) == 0 {
		units = rng.Int64N(2)
	}
	return st.request(pod, limit, units)
}

// settleEvent is a quiet moment: every write lands and every pod reads
// again, and the stored rows must match the reference.
type settleEvent struct{}

func (settleEvent) Name() string { return "settle" }

func (settleEvent) Run(_ *rand.Rand, st *simState) error {
	st.store.reliable = true
	defer func() { st.store.reliable = false }()
	return st.settleAndCheck()
}

type flushEvent struct{}

func (flushEvent) Name() string { return "flush" }

func (flushEvent) Run(rng *rand.Rand, st *simState) error {
	// A failed write is part of the simulation, not a defect.
	_ = st.pods[rng.IntN(len(st.pods))].limiter.flush(context.Background())
	return nil
}

type deliverWriteEvent struct{}

func (deliverWriteEvent) Name() string { return "deliver write" }

func (deliverWriteEvent) Run(_ *rand.Rand, st *simState) error {
	st.store.deliverOne()
	return nil
}

type replicateEvent struct{}

func (replicateEvent) Name() string { return "replicate" }

func (replicateEvent) Run(_ *rand.Rand, st *simState) error {
	st.store.replicateSome()
	return nil
}

// backgroundEvent runs one background read of a random pod.
type backgroundEvent struct{}

func (backgroundEvent) Name() string { return "background" }

func (backgroundEvent) Run(rng *rand.Rand, st *simState) error {
	pod := st.pods[rng.IntN(len(st.pods))]
	if len(pod.background) > 0 {
		index := rng.IntN(len(pod.background))
		work := pod.background[index]
		pod.background = slices.Delete(pod.background, index, index+1)
		work()
	}
	return nil
}

type advanceTimeEvent struct{}

func (advanceTimeEvent) Name() string { return "advance time" }

func (advanceTimeEvent) Run(rng *rand.Rand, st *simState) error {
	// Mostly short steps, sometimes long ones, so periods end and rows fade.
	steps := []time.Duration{time.Millisecond, 50 * time.Millisecond, 500 * time.Millisecond, 2 * time.Second, 3 * time.Second}
	if rng.IntN(10) == 0 {
		steps = []time.Duration{10 * time.Second, 30 * time.Second, 5 * time.Minute}
	}
	st.advance(steps[rng.IntN(len(steps))])
	return nil
}

type cleanupEvent struct{}

func (cleanupEvent) Name() string { return "cleanup" }

func (cleanupEvent) Run(rng *rand.Rand, st *simState) error {
	// A failed read is part of the simulation, not a defect.
	_ = st.pods[rng.IntN(len(st.pods))].limiter.cleanUp(context.Background())
	return nil
}

// simStore is a [Store] with a primary and a lagging replica, and its own
// clock. Acknowledged writes land at once. Writes whose answer is lost wait
// in flight for up to a minute and land in random order, or never. Some
// writes, reads and deletes fail. Reads come from the replica, which copies
// rows and deletes in random order. Each seed picks its own rates.
type simStore struct {
	rng      *rand.Rand
	clock    *clock.TestClock
	primary  rowTable
	replica  rowTable
	inFlight []inFlightWrite

	// deleted holds the largest sum that cleanup deleted per row, so the
	// check can tell what cleanup removed from what a pod lost.
	deleted rowTable

	writeFailPercent     int
	answerLostPercent    int
	neverLandsPercent    int
	slowWritePercent     int
	readFailPercent      int
	replicateRowsPercent int

	// reliable turns off failures for the checks.
	reliable bool
}

type inFlightWrite struct {
	rows           []StoredRow
	deliverAfterNs int64
}

func newSimStore(rng *rand.Rand, clk *clock.TestClock) *simStore {
	pick := func(percents ...int) int { return percents[rng.IntN(len(percents))] }
	return &simStore{
		rng:                  rng,
		clock:                clk,
		primary:              make(rowTable),
		replica:              make(rowTable),
		deleted:              make(rowTable),
		writeFailPercent:     pick(0, 5, 10, 25, 50),
		answerLostPercent:    pick(0, 5, 10, 25),
		neverLandsPercent:    pick(0, 10, 50),
		slowWritePercent:     pick(0, 10, 30),
		readFailPercent:      pick(0, 5, 10, 25, 50),
		replicateRowsPercent: pick(10, 50, 90),
	}
}

func (s *simStore) NowNs(_ context.Context) (int64, error) {
	if !s.reliable && s.rng.IntN(100) < s.readFailPercent {
		return 0, fault.New("read failed")
	}
	return s.clock.Now().UnixNano(), nil
}

func (s *simStore) WriteRows(_ context.Context, rows []StoredRow) error {
	if s.reliable {
		s.primary.upsert(rows)
		return nil
	}
	roll := s.rng.IntN(100)
	if roll < s.writeFailPercent {
		return fault.New("write failed")
	}
	// A write whose answer is lost or late lands at some later time, or
	// never.
	if roll < s.writeFailPercent+s.answerLostPercent || s.rng.IntN(100) < s.slowWritePercent {
		if s.rng.IntN(100) >= s.neverLandsPercent {
			s.inFlight = append(s.inFlight, inFlightWrite{rows: slices.Clone(rows), deliverAfterNs: s.clock.Now().UnixNano() + s.rng.Int64N(time.Minute.Nanoseconds())})
		}
		return fault.New("write sent, but the answer was lost")
	}
	s.primary.upsert(rows)
	return nil
}

func (s *simStore) ReadRows(_ context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error) {
	if !s.reliable && s.rng.IntN(100) < s.readFailPercent {
		return nil, fault.New("read failed")
	}
	return s.replica.periods(fingerprint, podID, limit), nil
}

func (s *simStore) DeleteFadedRows(_ context.Context, beforeNs int64, limit int) (int64, error) {
	if !s.reliable && s.rng.IntN(100) < s.writeFailPercent {
		return 0, fault.New("delete failed")
	}
	deleted := s.primary.deleteFaded(beforeNs, limit)
	for _, r := range deleted {
		known := s.deleted[r.Fingerprint][rowKey{periodStartNs: r.PeriodStartNs, podID: r.PodID}]
		r.ScaledSum = max(r.ScaledSum, known.ScaledSum)
		s.deleted.set(r)
	}
	return int64(len(deleted)), nil
}

// deletedLevel returns the level that cleanup removed from a fingerprint and
// still is missing at `nowNs`: per deleted row, what its largest deleted sum
// holds beyond the row that a late write put back. It also returns the
// number of deleted rows.
func (s *simStore) deletedLevel(fingerprint [32]byte, nowNs int64, duration time.Duration) (float64, int) {
	level := 0.0
	for key, r := range s.deleted[fingerprint] {
		missing := max(0, r.ScaledSum-s.primary[fingerprint][key].ScaledSum)
		level += row{scaledSum: missing}.levelAt(key, nowNs, duration)
	}
	return level, len(s.deleted[fingerprint])
}

// deliverOne applies one random write whose wait is over.
func (s *simStore) deliverOne() {
	var ready []int
	for index, write := range s.inFlight {
		if write.deliverAfterNs <= s.clock.Now().UnixNano() {
			ready = append(ready, index)
		}
	}
	if len(ready) == 0 {
		return
	}
	s.deliver(ready[s.rng.IntN(len(ready))])
}

func (s *simStore) deliver(index int) {
	s.primary.upsert(s.inFlight[index].rows)
	s.inFlight = slices.Delete(s.inFlight, index, index+1)
}

func (s *simStore) deliverAll() {
	for len(s.inFlight) > 0 {
		s.deliver(s.rng.IntN(len(s.inFlight)))
	}
}

// replicateSome copies some primary rows to the replica and applies some
// deletes, so the replica can hold some rows newer than others.
func (s *simStore) replicateSome() {
	for _, fingerprint := range sortedFingerprints(s.replica) {
		for _, r := range s.replica.newestFirst(fingerprint) {
			_, kept := s.primary[fingerprint][rowKey{periodStartNs: r.PeriodStartNs, podID: r.PodID}]
			if !kept && s.rng.IntN(100) < s.replicateRowsPercent {
				delete(s.replica[fingerprint], rowKey{periodStartNs: r.PeriodStartNs, podID: r.PodID})
			}
		}
	}
	for _, fingerprint := range sortedFingerprints(s.primary) {
		for _, r := range s.primary.newestFirst(fingerprint) {
			if s.rng.IntN(100) < s.replicateRowsPercent {
				s.replica.set(r)
			}
		}
	}
}

func (s *simStore) replicateAll() {
	clear(s.replica)
	for _, fingerprint := range sortedFingerprints(s.primary) {
		for _, r := range s.primary.newestFirst(fingerprint) {
			s.replica.set(r)
		}
	}
}

func sortedFingerprints(table rowTable) [][32]byte {
	fingerprints := make([][32]byte, 0, len(table))
	for fingerprint := range table {
		fingerprints = append(fingerprints, fingerprint)
	}
	slices.SortFunc(fingerprints, func(left, right [32]byte) int {
		return cmp.Compare(string(left[:]), string(right[:]))
	})
	return fingerprints
}
