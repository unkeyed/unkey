package ratelimit

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
)

// rowTable is the `ratelimit_state` table in memory.
type rowTable map[[32]byte]map[rowKey]StoredRow

// memoryStore is a [Store] in memory for tests. Its clock is the test clock,
// and it keeps every write call.
type memoryStore struct {
	mu         sync.Mutex
	clock      clock.Clock
	rows       rowTable
	writeCalls [][]StoredRow
	failWrites bool
	readCalls  int
	failReads  bool
}

func newMemoryStore(clk clock.Clock) *memoryStore {
	return &memoryStore{clock: clk, rows: make(rowTable)}
}

func (s *memoryStore) NowMs(_ context.Context) (int64, error) {
	return s.clock.Now().UnixMilli(), nil
}

func (s *memoryStore) WriteRows(_ context.Context, rows []StoredRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeCalls = append(s.writeCalls, slices.Clone(rows))
	if s.failWrites {
		return fault.New("write failed")
	}
	s.rows.upsert(rows)
	return nil
}

func (s *memoryStore) ReadRows(_ context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readCalls++
	if s.failReads {
		return nil, fault.New("read failed")
	}
	return s.rows.periods(fingerprint, podID, limit), nil
}

func (s *memoryStore) DeleteFadedRows(_ context.Context, beforeMs int64, limit int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int64(len(s.rows.deleteFaded(beforeMs, limit))), nil
}

// upsert is `UpsertRow`: it keeps the larger sum and delete time.
func (table rowTable) upsert(rows []StoredRow) {
	for _, r := range rows {
		key := rowKey{periodStartMs: r.PeriodStartMs, podID: r.PodID}
		stored, ok := table[r.Fingerprint][key]
		if ok {
			r.ScaledSum = max(r.ScaledSum, stored.ScaledSum)
			r.DeleteAfterMs = max(r.DeleteAfterMs, stored.DeleteAfterMs)
		}
		table.set(r)
	}
}

// set stores a row as it is, as replication does.
func (table rowTable) set(r StoredRow) {
	if table[r.Fingerprint] == nil {
		table[r.Fingerprint] = make(map[rowKey]StoredRow)
	}
	table[r.Fingerprint][rowKey{periodStartMs: r.PeriodStartMs, podID: r.PodID}] = r
}

// newestFirst returns the rows of a fingerprint newest period first.
func (table rowTable) newestFirst(fingerprint [32]byte) []StoredRow {
	rows := make([]StoredRow, 0, len(table[fingerprint]))
	for _, r := range table[fingerprint] {
		rows = append(rows, r)
	}
	slices.SortFunc(rows, func(left, right StoredRow) int {
		return cmp.Or(cmp.Compare(right.PeriodStartMs, left.PeriodStartMs), cmp.Compare(left.PodID, right.PodID))
	})
	return rows
}

// periods is `ListPeriods` as `ReadRows` returns it: per period, newest
// first, the row of `podID` and one row that sums the other pods' rows.
func (table rowTable) periods(fingerprint [32]byte, podID string, limit int) []StoredRow {
	var rows []StoredRow
	newestFirst := table.newestFirst(fingerprint)
	for start, periodCount := 0, 0; start < len(newestFirst) && periodCount < limit; periodCount++ {
		end := start
		var own, peers StoredRow
		var deleteAfterMs int64
		for end < len(newestFirst) && newestFirst[end].PeriodStartMs == newestFirst[start].PeriodStartMs {
			r := newestFirst[end]
			end++
			deleteAfterMs = max(deleteAfterMs, r.DeleteAfterMs)
			if r.PodID == podID {
				own = r
				continue
			}
			peers.Fingerprint, peers.PeriodStartMs = r.Fingerprint, r.PeriodStartMs
			peers.ScaledSum += r.ScaledSum
		}
		if own.ScaledSum > 0 {
			own.DeleteAfterMs = deleteAfterMs
			rows = append(rows, own)
		}
		if peers.ScaledSum > 0 {
			peers.DeleteAfterMs = deleteAfterMs
			rows = append(rows, peers)
		}
		start = end
	}
	return rows
}

// deleteFaded is `DeleteFadedRows`. It returns the deleted rows, in key
// order so that runs repeat.
func (table rowTable) deleteFaded(beforeMs int64, limit int) []StoredRow {
	var faded []StoredRow
	for _, rows := range table {
		for _, r := range rows {
			if r.DeleteAfterMs < beforeMs {
				faded = append(faded, r)
			}
		}
	}
	slices.SortFunc(faded, func(left, right StoredRow) int {
		return cmp.Or(cmp.Compare(string(left.Fingerprint[:]), string(right.Fingerprint[:])), cmp.Compare(left.PeriodStartMs, right.PeriodStartMs), cmp.Compare(left.PodID, right.PodID))
	})
	faded = faded[:min(limit, len(faded))]
	for _, r := range faded {
		delete(table[r.Fingerprint], rowKey{periodStartMs: r.PeriodStartMs, podID: r.PodID})
		if len(table[r.Fingerprint]) == 0 {
			delete(table, r.Fingerprint)
		}
	}
	return faded
}

// seed stores rows as they are.
func (s *memoryStore) seed(rows []StoredRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range rows {
		s.rows.set(r)
	}
}

// storedRows returns all rows of a fingerprint, newest first.
func (s *memoryStore) storedRows(fingerprint [32]byte) []StoredRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rows.newestFirst(fingerprint)
}

func (s *memoryStore) setFailWrites(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failWrites = fail
}

func (s *memoryStore) writeCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.writeCalls)
}

func (s *memoryStore) setFailReads(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failReads = fail
}

func (s *memoryStore) readCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readCalls
}
