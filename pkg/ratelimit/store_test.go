package ratelimit

import (
	"bytes"
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

func (s *memoryStore) NowNs(_ context.Context) (int64, error) {
	return s.clock.Now().UnixNano(), nil
}

func (s *memoryStore) WriteRows(_ context.Context, rows []StoredRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeCalls = append(s.writeCalls, slices.Clone(rows))
	if s.failWrites {
		return fault.New("write failed")
	}
	s.rows.upsert(rows, s.clock.Now().UnixNano())
	return nil
}

func (s *memoryStore) ReadRows(_ context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readCalls++
	if s.failReads {
		return nil, fault.New("read failed")
	}
	return s.rows.buckets(fingerprint, podID, limit), nil
}

func (s *memoryStore) ReadClosedRows(_ context.Context, fingerprint [32]byte, beforeNs int64, limit int) ([]StoredRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failReads {
		return nil, fault.New("read failed")
	}
	return s.rows.closedRows(fingerprint, beforeNs, limit), nil
}

func (s *memoryStore) ListDueRows(_ context.Context, beforeNs int64, after DueRow, limit int) ([]DueRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rows.dueRows(beforeNs, after, limit), nil
}

func (s *memoryStore) ApplyCleanup(_ context.Context, deleted []StoredRow, inserted []StoredRow) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rows.applyCleanup(deleted, inserted), nil
}

// upsert is `UpsertRow` at the store time `nowNs`, with its refusal of
// closed rows.
func (table rowTable) upsert(rows []StoredRow, nowNs int64) {
	for _, r := range rows {
		if r.CompactAfterNs < nowNs {
			r.ConsumedNs = 0
		}
		byKey, ok := table[r.Fingerprint]
		if !ok {
			byKey = make(map[rowKey]StoredRow)
			table[r.Fingerprint] = byKey
		}
		key := rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}
		stored, ok := byKey[key]
		if ok {
			if r.ConsumedNs > 0 {
				stored.ResetAtNs = max(stored.ResetAtNs, r.ResetAtNs)
			}
			stored.ConsumedNs = max(stored.ConsumedNs, r.ConsumedNs)
			r = stored
		}
		byKey[key] = r
	}
}

// newestFirst returns the rows of a fingerprint newest bucket first.
func (table rowTable) newestFirst(fingerprint [32]byte) []StoredRow {
	rows := make([]StoredRow, 0, len(table[fingerprint]))
	for _, r := range table[fingerprint] {
		rows = append(rows, r)
	}
	slices.SortFunc(rows, func(left, right StoredRow) int {
		return cmp.Or(cmp.Compare(right.BucketStartNs, left.BucketStartNs), cmp.Compare(left.PodID, right.PodID))
	})
	return rows
}

// buckets is `ListBuckets` as `ReadRows` returns it: per bucket, newest
// first, the row of `podID` and one row that sums the other pods' rows.
// Refused rows, with zero consumed time, do not count towards `limit`.
func (table rowTable) buckets(fingerprint [32]byte, podID string, limit int) []StoredRow {
	var rows []StoredRow
	newestFirst := slices.DeleteFunc(table.newestFirst(fingerprint), func(r StoredRow) bool { return r.ConsumedNs == 0 })
	for start, bucketCount := 0, 0; start < len(newestFirst) && bucketCount < limit; bucketCount++ {
		end := start
		var own, peers StoredRow
		var compactAfterNs int64
		for end < len(newestFirst) && newestFirst[end].BucketStartNs == newestFirst[start].BucketStartNs {
			r := newestFirst[end]
			end++
			compactAfterNs = max(compactAfterNs, r.CompactAfterNs)
			if r.PodID == podID {
				own = r
				continue
			}
			peers.Fingerprint, peers.BucketStartNs = r.Fingerprint, r.BucketStartNs
			peers.ConsumedNs += r.ConsumedNs
			peers.ResetAtNs = max(peers.ResetAtNs, r.ResetAtNs)
		}
		if own.ConsumedNs > 0 {
			own.CompactAfterNs = compactAfterNs
			rows = append(rows, own)
		}
		if peers.ConsumedNs > 0 {
			peers.CompactAfterNs = compactAfterNs
			rows = append(rows, peers)
		}
		start = end
	}
	return rows
}

// closedRows is `ListClosedRows`.
func (table rowTable) closedRows(fingerprint [32]byte, beforeNs int64, limit int) []StoredRow {
	var rows []StoredRow
	for _, r := range table[fingerprint] {
		if r.BucketStartNs == 0 || r.CompactAfterNs < beforeNs {
			rows = append(rows, r)
		}
	}
	slices.SortFunc(rows, func(left, right StoredRow) int {
		return cmp.Or(cmp.Compare(left.BucketStartNs, right.BucketStartNs), cmp.Compare(left.PodID, right.PodID))
	})
	return rows[:min(limit, len(rows))]
}

// dueRows is `ListRowsToCompact`.
func (table rowTable) dueRows(beforeNs int64, after DueRow, limit int) []DueRow {
	var due []DueRow
	for fingerprint, rows := range table {
		for _, r := range rows {
			isAfter := r.CompactAfterNs > after.CompactAfterNs ||
				(r.CompactAfterNs == after.CompactAfterNs && bytes.Compare(fingerprint[:], after.Fingerprint[:]) > 0)
			if r.CompactAfterNs < beforeNs && isAfter {
				due = append(due, DueRow{Fingerprint: fingerprint, CompactAfterNs: r.CompactAfterNs})
			}
		}
	}
	slices.SortFunc(due, func(left, right DueRow) int {
		return cmp.Or(cmp.Compare(left.CompactAfterNs, right.CompactAfterNs), bytes.Compare(left.Fingerprint[:], right.Fingerprint[:]))
	})
	return due[:min(limit, len(due))]
}

// applyCleanup is `ApplyCleanup`. An insert of an existing row fails like
// the duplicate key in MySQL, and rolls back.
func (table rowTable) applyCleanup(deleted []StoredRow, inserted []StoredRow) bool {
	remaining := make(map[[32]byte]map[rowKey]bool)
	for _, r := range deleted {
		key := rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}
		stored, ok := table[r.Fingerprint][key]
		if !ok || stored.ConsumedNs != r.ConsumedNs || stored.ResetAtNs != r.ResetAtNs || remaining[r.Fingerprint][key] {
			return false
		}
		if remaining[r.Fingerprint] == nil {
			remaining[r.Fingerprint] = make(map[rowKey]bool)
		}
		remaining[r.Fingerprint][key] = true
	}
	for _, r := range inserted {
		key := rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}
		if _, exists := table[r.Fingerprint][key]; exists && !remaining[r.Fingerprint][key] {
			return false
		}
	}
	for _, r := range deleted {
		delete(table[r.Fingerprint], rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID})
		if len(table[r.Fingerprint]) == 0 {
			delete(table, r.Fingerprint)
		}
	}
	for _, r := range inserted {
		if table[r.Fingerprint] == nil {
			table[r.Fingerprint] = make(map[rowKey]StoredRow)
		}
		table[r.Fingerprint][rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}] = r
	}
	return true
}

// seed stores rows as they are, as if pods wrote them while their buckets
// were open.
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
