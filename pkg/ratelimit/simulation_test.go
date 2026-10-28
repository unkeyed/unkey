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

// TestSimulation runs 1 to 8 pods against one shared store that delays,
// reorders, fails, loses and refuses writes, and serves stale or failed
// reads. Pods also run cleanup passes and compactions that merge and delete
// rows. Each seed picks its own failure rates. Pod clocks differ from the
// store clock by less than `maxClockSkew`, some pods get few requests, and
// half the seeds send everything to one limit. Runs last from about half an
// hour to a few hours, so buckets close and get merged.
//
// The reference is a single machine over the same buckets: the formula of
// `computeResetTimes` over the rows of the primary, where each pod's rows
// that it still writes are replaced by its values in memory. A write that
// MySQL refuses never reaches the primary, like the units of a crashed pod.
// Each pod computes its local reset_at from the rows it has seen. Decisions
// only use `max(reset_at, now)`, so every reset_at below means that. For
// every request we check:
//
//	local ≤ true                       a pod sees no units that do not exist
//	local ≥ own                        a pod never forgets units it still writes
//	allowed = local + units - now ≤ duration
//	true' ≤ true + units               a pod never charges more than the GCRA rule
//
// `own` is the reset_at of the rows that the pod still writes, and `true'`
// is the true reset_at after the request.
//
// If the pod knows every row, local = true, and the pod reports the true
// reset_at after the request. At the end every write is delivered and every
// pod reads again, so every pod must be exact.
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
		// One seed takes well under a second, so a minute is enough margin.
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
// first events. The fuzzer keeps inputs that reach new code and changes
// them further. Run it with
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
		sim.WithTicks[simState](3_000+rng.Int64N(12_000)),
		sim.WithState(func(rng *rand.Rand) *simState {
			state = newSimState(t, rng)
			return state
		}),
	)
	require.NoError(t, simulation.Run([]sim.Event[simState]{
		requestEvent{}, requestEvent{}, flushEvent{}, deliverWriteEvent{}, replicateEvent{}, backgroundEvent{}, advanceTimeEvent{}, settleEvent{}, cleanupEvent{},
	}))
	if len(simulation.Errors) > 0 {
		t.Fatalf("%d failed checks, the first: %v", len(simulation.Errors), simulation.Errors[0])
	}

	state.store.reliable = true
	for _, pod := range state.pods {
		require.NoError(t, state.settle())
		for _, limit := range state.limits {
			units := state.rng.Int64N(limit.rate.Limit + 1)
			exact, err := state.request(pod, limit, units)
			require.NoError(t, err)
			require.True(t, exact, "every write is visible, so the pod must know every row")
		}
	}
	require.Positive(t, state.exactDecisions)
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
	// background holds the background reads and compactions that have not
	// run yet.
	background []func()
	weight     float64
}

type simState struct {
	rng    *rand.Rand
	store  *simStore
	pods   []*simPod
	limits []simLimit

	exactDecisions int
}

func newSimState(t *testing.T, rng *rand.Rand) *simState {
	t.Helper()
	start := time.Unix(1_700_000_000, 0)
	state := &simState{
		rng:   rng,
		store: newSimStore(rng, clock.NewTestClock(start)),
	}
	// Large skews put a peer's buckets ahead of this pod's clock. Each pod
	// differs from the store clock by less than `maxClockSkew`.
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
	state.store.cleanUpOnAnotherPod = func() {
		// A failed read or transaction is part of the simulation.
		_ = state.pods[rng.IntN(len(state.pods))].limiter.cleanUp(context.Background())
	}
	durations := []time.Duration{time.Second, 2 * time.Second, 10 * time.Second, time.Minute, 10 * time.Minute, time.Hour}
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

// request sends one request and checks it against the reference. It reports
// whether the pod knew every row.
func (st *simState) request(pod *simPod, limit simLimit, units int64) (bool, error) {
	ctx := context.Background()
	request := Request{Fingerprint: limit.fingerprint, Rate: limit.rate, Units: units}
	nowNs := pod.clock.Now().UnixNano()
	// Refreshed and checked here so the check sees what the decision sees;
	// Allow then finds the rows fresh and changes nothing before deciding.
	pod.limiter.refreshRows(ctx, []Request{request})
	e := pod.limiter.entryOf(limit.fingerprint, limit.rate)
	_, err := e.check(nowNs, limit.rate, 0)
	if err != nil {
		return false, err
	}
	localNs := max(e.resetTimes.resetAtNs, nowNs)
	truthRows := st.truthRows(limit.fingerprint, nil)
	exact := sameRows(e.rows, summedPerBucket(truthRows, pod.limiter.podID))
	truthBeforeNs, err := resetAtOf(truthRows, nowNs)
	if err != nil {
		return false, err
	}
	ownNs, err := resetAtOf(st.truthRows(limit.fingerprint, pod), nowNs)
	if err != nil {
		return false, err
	}

	result, err := pod.limiter.Allow(ctx, request)
	if err != nil {
		return false, err
	}
	truthAfterNs, err := resetAtOf(st.truthRows(limit.fingerprint, nil), nowNs)
	if err != nil {
		return false, err
	}

	fits := func(resetAtNs int64) bool {
		return units <= limit.rate.Limit && max(resetAtNs, nowNs)+limit.rate.unitsToNs(units)-nowNs-limit.rate.Duration.Nanoseconds() <= limit.rate.roundingSlackNs()
	}
	// Zero units spend nothing, so they leave reset_at as it is.
	gcraResetAtNs := truthBeforeNs
	if units > 0 && units <= limit.rate.Limit {
		gcraResetAtNs = truthBeforeNs + limit.rate.unitsToNs(units)
	}
	describe := fmt.Sprintf("units %d, rate %+v, local %d, true %d → %d, own %d, now %d", units, limit.rate, localNs, truthBeforeNs, truthAfterNs, ownNs, nowNs)
	switch {
	case localNs > truthBeforeNs:
		return false, fault.New("the pod knows more than the reference: " + describe)
	case localNs < ownNs:
		return false, fault.New("the pod forgot its own units: " + describe)
	case result.Allowed != fits(localNs):
		return false, fault.New("the pod did not follow the GCRA rule on what it knows: " + describe)
	case result.Allowed && truthAfterNs > gcraResetAtNs:
		return false, fault.New("charged more than the GCRA rule: " + describe)
	case exact && localNs != truthBeforeNs:
		return false, fault.New("knows every row but reset_at differs: " + describe)
	case exact && result.ResetAt.UnixNano() != truthAfterNs:
		return false, fault.New(fmt.Sprintf("knows every row but reports reset_at %d: %s", result.ResetAt.UnixNano(), describe))
	}
	if exact {
		st.exactDecisions++
	}
	return exact, nil
}

// truthRows returns the rows of the reference: the primary's rows, where
// each pod's rows that it still writes are raised to its values in memory.
// If `only` is not nil, it returns only that pod's rows that it still writes.
func (st *simState) truthRows(fingerprint [32]byte, only *simPod) map[rowKey]row {
	rows := make(map[rowKey]row)
	if only == nil {
		for key, stored := range st.store.primary[fingerprint] {
			if stored.ConsumedNs > 0 {
				rows[key] = row{consumedNs: stored.ConsumedNs, resetAtNs: stored.ResetAtNs, own: false, committedNs: 0, committedResetAtNs: 0}
			}
		}
	}
	for _, pod := range st.pods {
		if only != nil && pod != only {
			continue
		}
		e, ok := pod.limiter.shards[shardIndex(fingerprint)].entries[fingerprint]
		if !ok {
			continue
		}
		for key, r := range e.rows {
			if !r.own || pod.clock.Now().UnixNano() > lastWriteAtNs(key.bucketStartNs, e.bucketWidthNs) {
				continue
			}
			known := rows[key]
			rows[key] = row{consumedNs: max(known.consumedNs, r.consumedNs), resetAtNs: max(known.resetAtNs, r.resetAtNs), own: false, committedNs: 0, committedResetAtNs: 0}
		}
	}
	return rows
}

// summedPerBucket returns the rows as `ReadRows` returns them to the pod
// `podID`: its own rows, and per bucket one row that sums the other pods'.
func summedPerBucket(rows map[rowKey]row, podID string) map[rowKey]row {
	summed := make(map[rowKey]row, len(rows))
	for key, r := range rows {
		if key.podID == podID {
			summed[key] = r
			continue
		}
		peersKey := rowKey{bucketStartNs: key.bucketStartNs, podID: ""}
		peers := summed[peersKey]
		peers.consumedNs += r.consumedNs
		peers.resetAtNs = max(peers.resetAtNs, r.resetAtNs)
		summed[peersKey] = peers
	}
	return summed
}

func resetAtOf(rows map[rowKey]row, nowNs int64) (int64, error) {
	times, err := computeResetTimes(rows, math.MaxInt64)
	return max(times.resetAtNs, nowNs), err
}

// sameRows reports whether a pod knows exactly the rows with consumed time
// of the reference.
func sameRows(known map[rowKey]row, truth map[rowKey]row) bool {
	count := 0
	for key, r := range known {
		if r.consumedNs == 0 {
			continue
		}
		count++
		reference, ok := truth[key]
		if !ok || reference.consumedNs != r.consumedNs || reference.resetAtNs != r.resetAtNs {
			return false
		}
	}
	return count == len(truth)
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

// settle makes every write visible and lets every pod's rows expire, so
// the next request on each pod reads every row. The store must be reliable.
func (st *simState) settle() error {
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
	return nil
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
	_, err := st.request(pod, limit, units)
	return err
}

// settleEvent is a quiet moment: every write lands and every pod reads
// again on its next request, so the next decisions can be exact.
type settleEvent struct{}

func (settleEvent) Name() string { return "settle" }

func (settleEvent) Run(_ *rand.Rand, st *simState) error {
	st.store.reliable = true
	defer func() { st.store.reliable = false }()
	return st.settle()
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

// backgroundEvent runs one background read or compaction of a random pod.
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
	// Mostly short steps, sometimes long ones, so buckets close.
	steps := []time.Duration{time.Millisecond, 50 * time.Millisecond, 500 * time.Millisecond, 2 * time.Second, 3 * time.Second}
	if rng.IntN(10) == 0 {
		steps = []time.Duration{10 * time.Second, 30 * time.Second}
	}
	st.advance(steps[rng.IntN(len(steps))])
	return nil
}

type cleanupEvent struct{}

func (cleanupEvent) Name() string { return "cleanup" }

func (cleanupEvent) Run(rng *rand.Rand, st *simState) error {
	// A failed read or transaction is part of the simulation, not a defect.
	_ = st.pods[rng.IntN(len(st.pods))].limiter.cleanUp(context.Background())
	return nil
}

// simStore is a [Store] with a primary and a lagging replica, and its own
// clock. Acknowledged writes land at once. Writes whose answer is lost wait
// in flight for up to a minute and land in random order, or MySQL refuses
// them. Some writes and cleanups fail,
// some fail after they were applied, and some reads fail. Some writes and
// other pods' cleanups land between a cleanup's read and its transaction.
// Reads come from the replica, except cleanup reads. A cleanup reaches the
// replica as one transaction. Each seed picks its own rates.
type simStore struct {
	rng      *rand.Rand
	clock    *clock.TestClock
	primary  rowTable
	replica  rowTable
	inFlight []inFlightWrite

	// cleanupsToReplicate are applied on the primary but not yet on the
	// replica, oldest first.
	cleanupsToReplicate []appliedCleanup

	// cleanUpOnAnotherPod runs a cleanup pass on a random pod. ApplyCleanup
	// sometimes calls it first, as if another pod's pass ran at the same
	// time and committed first.
	cleanUpOnAnotherPod func()
	insideOtherCleanup  bool

	writeFailPercent     int
	answerLostPercent    int
	slowWritePercent     int
	readFailPercent      int
	replicateRowsPercent int

	// reliable turns off failures for the final checks.
	reliable bool
}

type appliedCleanup struct {
	deleted  []StoredRow
	inserted []StoredRow
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
		writeFailPercent:     pick(0, 5, 10, 25, 50),
		answerLostPercent:    pick(0, 5, 10, 25),
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
	write := inFlightWrite{rows: slices.Clone(rows), deliverAfterNs: 0}
	if s.reliable {
		s.inFlight = append(s.inFlight, write)
		return nil
	}
	roll := s.rng.IntN(100)
	if roll < s.writeFailPercent {
		return fault.New("write failed")
	}
	// An acknowledged write has run. A write whose answer is lost or late
	// lands at some later time, or is refused then.
	if roll < s.writeFailPercent+s.answerLostPercent || s.rng.IntN(100) < s.slowWritePercent {
		write.deliverAfterNs = s.clock.Now().UnixNano() + s.rng.Int64N(time.Minute.Nanoseconds())
		s.inFlight = append(s.inFlight, write)
		return fault.New("write sent, but the answer was lost")
	}
	s.primary.upsert(rows, s.clock.Now().UnixNano())
	return nil
}

func (s *simStore) ReadRows(_ context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error) {
	if !s.reliable && s.rng.IntN(100) < s.readFailPercent {
		return nil, fault.New("read failed")
	}
	return s.replica.buckets(fingerprint, podID, limit), nil
}

func (s *simStore) ReadClosedRows(_ context.Context, fingerprint [32]byte, beforeNs int64, limit int) ([]StoredRow, error) {
	if !s.reliable && s.rng.IntN(100) < s.readFailPercent {
		return nil, fault.New("read failed")
	}
	return s.primary.closedRows(fingerprint, beforeNs, limit), nil
}

func (s *simStore) ListDueRows(_ context.Context, beforeNs int64, after DueRow, limit int) ([]DueRow, error) {
	if !s.reliable && s.rng.IntN(100) < s.readFailPercent {
		return nil, fault.New("read failed")
	}
	return s.primary.dueRows(beforeNs, after, limit), nil
}

func (s *simStore) ApplyCleanup(_ context.Context, deleted []StoredRow, inserted []StoredRow) (bool, error) {
	if !s.reliable {
		if s.rng.IntN(4) == 0 {
			s.deliverOne()
		}
		if !s.insideOtherCleanup && s.rng.IntN(4) == 0 {
			s.insideOtherCleanup = true
			s.cleanUpOnAnotherPod()
			s.insideOtherCleanup = false
		}
		roll := s.rng.IntN(100)
		if roll < s.writeFailPercent {
			return false, fault.New("cleanup failed")
		}
		if roll < s.writeFailPercent+s.answerLostPercent {
			s.applyCleanup(deleted, inserted)
			return false, fault.New("cleanup applied, but the answer was lost")
		}
	}
	return s.applyCleanup(deleted, inserted), nil
}

func (s *simStore) applyCleanup(deleted []StoredRow, inserted []StoredRow) bool {
	if !s.primary.applyCleanup(deleted, inserted) {
		return false
	}
	s.cleanupsToReplicate = append(s.cleanupsToReplicate, appliedCleanup{deleted: slices.Clone(deleted), inserted: slices.Clone(inserted)})
	return true
}

// deliverOne applies one random write whose wait is over, at the store
// time, so MySQL may refuse it.
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
	s.primary.upsert(s.inFlight[index].rows, s.clock.Now().UnixNano())
	s.inFlight = slices.Delete(s.inFlight, index, index+1)
}

func (s *simStore) deliverAll() {
	for len(s.inFlight) > 0 {
		s.deliver(s.rng.IntN(len(s.inFlight)))
	}
}

// replicateSome copies some primary rows to the replica, so the replica can
// hold some rows newer than others. Cleanups reach the replica in order,
// each in one step, and a base row only with its cleanup.
func (s *simStore) replicateSome() {
	for len(s.cleanupsToReplicate) > 0 && s.rng.IntN(100) < s.replicateRowsPercent {
		s.replicateCleanup()
	}
	waiting := make(map[[32]byte]bool)
	for _, applied := range s.cleanupsToReplicate {
		for _, r := range applied.inserted {
			waiting[r.Fingerprint] = true
		}
	}
	for _, fingerprint := range sortedFingerprints(s.primary) {
		for _, r := range s.primary.newestFirst(fingerprint) {
			if (r.BucketStartNs > 0 || !waiting[fingerprint]) && s.rng.IntN(100) < s.replicateRowsPercent {
				s.replica.set(r)
			}
		}
	}
}

// replicateCleanup applies the oldest cleanup to the replica. A deleted row
// that the replica never had is not there to delete.
func (s *simStore) replicateCleanup() {
	applied := s.cleanupsToReplicate[0]
	s.cleanupsToReplicate = s.cleanupsToReplicate[1:]
	for _, r := range applied.deleted {
		delete(s.replica[r.Fingerprint], rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID})
	}
	for _, r := range applied.inserted {
		s.replica.set(r)
	}
}

func (s *simStore) replicateAll() {
	for len(s.cleanupsToReplicate) > 0 {
		s.replicateCleanup()
	}
	for _, fingerprint := range sortedFingerprints(s.primary) {
		for _, r := range s.primary.newestFirst(fingerprint) {
			s.replica.set(r)
		}
	}
}

// set copies a row as it is, as replication does.
func (table rowTable) set(r StoredRow) {
	if table[r.Fingerprint] == nil {
		table[r.Fingerprint] = make(map[rowKey]StoredRow)
	}
	table[r.Fingerprint][rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}] = r
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
