package ratelimit

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
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
// reorders, fails and loses writes, and serves stale or failed reads. Each
// seed picks its own failure rates. Pod clocks differ, some pods get few
// requests, and half the seeds send everything to one limit.
//
// The ledger records every session that any pod wrote units into. From it
// we compute the true reset_at. Each pod computes its local reset_at from
// the sessions it has seen. For every request we check:
//
//	local ≤ true                       a pod sees no units that do not exist
//	max(local, now) ≥ max(own, now)    a pod never forgets its own units
//	allowed = max(local, now) + units - now ≤ duration
//	true' ≤ max(true, now) + units     a pod never charges more than the GCRA rule
//
// `own` is the reset_at of the sessions the pod created, and `true'` is the
// true reset_at after the request.
//
// If the pod has seen every session, local = true and the last line is an
// equality. At the end every write is delivered and every pod reads again,
// so every pod must be exact.
//
// Seeds 0 to 99 run on every run and catch regressions. The random seeds
// search new scenarios. Replay a failing seed with `-sim.seed=<seed>`, and
// add it to the fixed seeds once the bug is fixed.
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
	for _, seed := range seeds {
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
		sim.WithTicks[simState](2_000+rng.Int64N(8_000)),
		sim.WithState(func(rng *rand.Rand) *simState {
			state = newSimState(t, rng)
			return state
		}),
	)
	require.NoError(t, simulation.Run([]sim.Event[simState]{
		requestEvent{}, requestEvent{}, flushEvent{}, deliverWriteEvent{}, replicateEvent{}, backgroundReadEvent{}, advanceTimeEvent{}, settleEvent{},
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
			require.True(t, exact, "every write is visible, so the pod must know every session")
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
	// background holds the background reads that have not run yet.
	background []func()
	weight     float64
}

type simState struct {
	rng    *rand.Rand
	store  *simStore
	pods   []*simPod
	limits []simLimit
	ledger map[[32]byte]map[string]session
	// owner maps each session ID to the pod that created it.
	owner map[string]*simPod

	// elapsed stays below `compactAfter`, so no pod drops unwritten units.
	elapsed time.Duration

	exactDecisions int
}

func newSimState(t *testing.T, rng *rand.Rand) *simState {
	t.Helper()
	state := &simState{
		rng:    rng,
		store:  newSimStore(rng),
		ledger: make(map[[32]byte]map[string]session),
		owner:  make(map[string]*simPod),
	}
	// Large skews put a peer's anchors ahead of this pod's clock.
	skews := []time.Duration{0, 50 * time.Millisecond, 2 * time.Second, 10 * time.Second}
	maxSkew := skews[rng.IntN(len(skews))]
	start := time.Unix(1_700_000_000, 0)
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
	durations := []time.Duration{100 * time.Millisecond, time.Second, 10 * time.Second, time.Minute}
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

// request sends one request and checks it against the ledger. It reports
// whether the pod knew every session.
func (st *simState) request(pod *simPod, limit simLimit, units int64) (bool, error) {
	ctx := context.Background()
	request := Request{Fingerprint: limit.fingerprint, Rate: limit.rate, Units: units}
	// Refreshed here so the check sees what the decision sees; Allow then
	// finds the rows fresh.
	pod.limiter.refreshRows(ctx, []Request{request})
	e := pod.limiter.entryOf(limit.fingerprint)
	localNs := e.resetTimes.resetAtNs
	exact := sameSessions(allSessions(*e), st.ledger[limit.fingerprint])
	truthBeforeNs, err := st.truth(limit.fingerprint, nil)
	if err != nil {
		return false, err
	}
	ownNs, err := st.truth(limit.fingerprint, pod)
	if err != nil {
		return false, err
	}
	nowNs := pod.clock.Now().UnixNano()

	result, err := pod.limiter.Allow(ctx, request)
	if err != nil {
		return false, err
	}
	if result.Allowed && units > 0 {
		sessions, ok := st.ledger[limit.fingerprint]
		if !ok {
			sessions = make(map[string]session)
			st.ledger[limit.fingerprint] = sessions
		}
		sessions[e.liveID] = e.sessions[e.liveID]
		st.owner[e.liveID] = pod
	}
	truthAfterNs, err := st.truth(limit.fingerprint, nil)
	if err != nil {
		return false, err
	}

	fits := func(resetAtNs int64) bool {
		return units <= limit.rate.Limit && max(resetAtNs, nowNs)+limit.rate.unitsToNs(units)-nowNs <= limit.rate.Duration.Nanoseconds()
	}
	// Zero units spend nothing, so they leave reset_at as it is.
	gcraResetAtNs := truthBeforeNs
	if units > 0 && units <= limit.rate.Limit {
		gcraResetAtNs = max(truthBeforeNs, nowNs) + limit.rate.unitsToNs(units)
	}
	describe := fmt.Sprintf("units %d, rate %+v, local %d, true %d → %d, now %d", units, limit.rate, localNs, truthBeforeNs, truthAfterNs, nowNs)
	switch {
	case localNs > truthBeforeNs:
		return false, fault.New("the pod knows more than the ledger: " + describe)
	case max(localNs, nowNs) < max(ownNs, nowNs):
		return false, fault.New("the pod forgot its own units: " + describe)
	case result.Allowed != fits(localNs):
		return false, fault.New("the pod did not follow the GCRA rule on what it knows: " + describe)
	case !result.Allowed && fits(truthBeforeNs):
		return false, fault.New("denied a request that the true state allows: " + describe)
	case result.Allowed && truthAfterNs > gcraResetAtNs:
		return false, fault.New("charged more than the GCRA rule: " + describe)
	case exact && localNs != truthBeforeNs:
		return false, fault.New("knows every session but reset_at differs: " + describe)
	case exact && result.Allowed && truthAfterNs != gcraResetAtNs:
		return false, fault.New("knows every session but charged less than the GCRA rule: " + describe)
	}
	if exact {
		st.exactDecisions++
	}
	return exact, nil
}

// truth returns the reset_at of the ledger, or of the sessions that `owner`
// created if it is not nil.
func (st *simState) truth(fingerprint [32]byte, owner *simPod) (int64, error) {
	sessions := make(map[string]session)
	for _, s := range st.ledger[fingerprint] {
		if owner == nil || st.owner[s.id] == owner {
			sessions[s.id] = s
		}
	}
	result, err := computeResetTimes(sessions, "")
	return result.resetAtNs, err
}

// advance moves every pod clock. A background read takes less than
// `readTimeout`, so the reads started before run first.
func (st *simState) advance(d time.Duration) {
	for _, pod := range st.pods {
		st.runBackgroundReads(pod)
	}
	for _, pod := range st.pods {
		pod.clock.Tick(d)
	}
	st.elapsed += d
	st.store.elapsed = st.elapsed
}

func (st *simState) runBackgroundReads(pod *simPod) {
	for len(pod.background) > 0 {
		read := pod.background[0]
		pod.background = pod.background[1:]
		read()
	}
}

// settle makes every write visible and lets every pod's rows expire, so
// the next request on each pod reads every session. The store must be
// reliable.
func (st *simState) settle() error {
	for _, pod := range st.pods {
		st.runBackgroundReads(pod)
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

func sameSessions(known []session, ledger map[string]session) bool {
	if len(known) != len(ledger) {
		return false
	}
	for _, s := range known {
		if ledger[s.id] != s {
			return false
		}
	}
	return true
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
	if st.elapsed+rowsStaleFor >= compactAfter-5*time.Minute {
		return nil
	}
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

type backgroundReadEvent struct{}

func (backgroundReadEvent) Name() string { return "background read" }

func (backgroundReadEvent) Run(rng *rand.Rand, st *simState) error {
	pod := st.pods[rng.IntN(len(st.pods))]
	if len(pod.background) > 0 {
		index := rng.IntN(len(pod.background))
		read := pod.background[index]
		pod.background = slices.Delete(pod.background, index, index+1)
		read()
	}
	return nil
}

type advanceTimeEvent struct{}

func (advanceTimeEvent) Name() string { return "advance time" }

func (advanceTimeEvent) Run(rng *rand.Rand, st *simState) error {
	// Mostly short steps, sometimes long ones, so runs reach the end of a
	// session lifetime.
	steps := []time.Duration{time.Millisecond, 50 * time.Millisecond, 500 * time.Millisecond, 2 * time.Second, 3 * time.Second}
	if rng.IntN(10) == 0 {
		steps = []time.Duration{10 * time.Second, 30 * time.Second}
	}
	d := steps[rng.IntN(len(steps))]
	if st.elapsed+d < compactAfter-5*time.Minute {
		st.advance(d)
	}
	return nil
}

// simStore is a [Store] with a primary and a lagging replica. Writes wait in
// flight until delivered in random order, and some wait up to a minute. Some
// writes fail, some fail after they were applied, and some reads fail. Reads
// come from the replica. Each seed picks its own rates.
type simStore struct {
	rng      *rand.Rand
	primary  map[[32]byte]map[string]StoredSession
	replica  map[[32]byte]map[string]StoredSession
	inFlight []inFlightWrite

	// elapsed is the simulation time, for writes that wait.
	elapsed time.Duration

	writeFailPercent     int
	answerLostPercent    int
	slowWritePercent     int
	readFailPercent      int
	replicateRowsPercent int

	// reliable turns off failures for the final checks.
	reliable bool
}

type inFlightWrite struct {
	sessions     []StoredSession
	deliverAfter time.Duration
}

func newSimStore(rng *rand.Rand) *simStore {
	pick := func(percents ...int) int { return percents[rng.IntN(len(percents))] }
	return &simStore{
		rng:                  rng,
		primary:              make(map[[32]byte]map[string]StoredSession),
		replica:              make(map[[32]byte]map[string]StoredSession),
		writeFailPercent:     pick(0, 5, 10, 25, 50),
		answerLostPercent:    pick(0, 5, 10, 25),
		slowWritePercent:     pick(0, 10, 30),
		readFailPercent:      pick(0, 5, 10, 25, 50),
		replicateRowsPercent: pick(10, 50, 90),
	}
}

func (s *simStore) WriteSessions(_ context.Context, sessions []StoredSession) error {
	if s.reliable {
		s.inFlight = append(s.inFlight, inFlightWrite{sessions: slices.Clone(sessions), deliverAfter: 0})
		return nil
	}
	roll := s.rng.IntN(100)
	if roll < s.writeFailPercent {
		return fault.New("write failed")
	}
	write := inFlightWrite{sessions: slices.Clone(sessions), deliverAfter: 0}
	if s.rng.IntN(100) < s.slowWritePercent {
		write.deliverAfter = s.elapsed + time.Duration(s.rng.Int64N(int64(time.Minute)))
	}
	s.inFlight = append(s.inFlight, write)
	if roll < s.writeFailPercent+s.answerLostPercent {
		return fault.New("write applied, but the answer was lost")
	}
	return nil
}

func (s *simStore) ReadSessions(_ context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	if !s.reliable && s.rng.IntN(100) < s.readFailPercent {
		return nil, fault.New("read failed")
	}
	sessions := sortedSessions(s.replica[fingerprint])
	return sessions[:min(limit, len(sessions))], nil
}

// deliverOne applies one random write whose wait is over.
func (s *simStore) deliverOne() {
	var ready []int
	for index, write := range s.inFlight {
		if write.deliverAfter <= s.elapsed {
			ready = append(ready, index)
		}
	}
	if len(ready) == 0 {
		return
	}
	s.deliver(ready[s.rng.IntN(len(ready))])
}

func (s *simStore) deliver(index int) {
	upsert(s.primary, s.inFlight[index].sessions)
	s.inFlight = slices.Delete(s.inFlight, index, index+1)
}

func (s *simStore) deliverAll() {
	for len(s.inFlight) > 0 {
		s.deliver(s.rng.IntN(len(s.inFlight)))
	}
}

// replicateSome copies some primary rows to the replica, so the replica can
// hold some rows newer than others.
func (s *simStore) replicateSome() {
	for _, fingerprint := range sortedFingerprints(s.primary) {
		for _, row := range sortedSessions(s.primary[fingerprint]) {
			if s.rng.IntN(100) < s.replicateRowsPercent {
				upsert(s.replica, []StoredSession{row})
			}
		}
	}
}

func (s *simStore) replicateAll() {
	for _, rows := range s.primary {
		for _, row := range rows {
			upsert(s.replica, []StoredSession{row})
		}
	}
}

func upsert(table map[[32]byte]map[string]StoredSession, sessions []StoredSession) {
	for _, session := range sessions {
		rows, ok := table[session.Fingerprint]
		if !ok {
			rows = make(map[string]StoredSession)
			table[session.Fingerprint] = rows
		}
		stored, ok := rows[session.SessionID]
		if ok {
			stored.ConsumedNs = max(stored.ConsumedNs, session.ConsumedNs)
			session = stored
		}
		rows[session.SessionID] = session
	}
}

// sortedSessions returns rows newest first, like `ListSessions`. The order
// also keeps runs repeatable, because map order is random.
func sortedSessions(rows map[string]StoredSession) []StoredSession {
	sessions := make([]StoredSession, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, row)
	}
	slices.SortFunc(sessions, func(left, right StoredSession) int {
		return cmp.Or(cmp.Compare(right.AnchorAtNs, left.AnchorAtNs), cmp.Compare(left.SessionID, right.SessionID))
	})
	return sessions
}

func sortedFingerprints(table map[[32]byte]map[string]StoredSession) [][32]byte {
	fingerprints := make([][32]byte, 0, len(table))
	for fingerprint := range table {
		fingerprints = append(fingerprints, fingerprint)
	}
	slices.SortFunc(fingerprints, func(left, right [32]byte) int {
		return slices.Compare(left[:], right[:])
	})
	return fingerprints
}
