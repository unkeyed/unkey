package ratelimit

import (
	"bytes"
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/unkeyed/unkey/pkg/fault"
)

// memoryStore is a [Store] in memory for tests. It keeps every write call.
type memoryStore struct {
	mu         sync.Mutex
	rows       map[[32]byte]map[string]StoredSession
	writeCalls [][]StoredSession
	failWrites bool
	readCalls  int
	failReads  bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{rows: make(map[[32]byte]map[string]StoredSession)}
}

func (s *memoryStore) WriteSessions(_ context.Context, sessions []StoredSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeCalls = append(s.writeCalls, slices.Clone(sessions))
	if s.failWrites {
		return fault.New("write failed")
	}
	upsert(s.rows, sessions)
	return nil
}

func (s *memoryStore) ReadSessions(_ context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	s.mu.Lock()
	s.readCalls++
	fail := s.failReads
	s.mu.Unlock()
	if fail {
		return nil, fault.New("read failed")
	}
	sessions := s.sessions(fingerprint)
	return sessions[:min(limit, len(sessions))], nil
}

func (s *memoryStore) ReadSessionsFromPrimary(ctx context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	return s.ReadSessions(ctx, fingerprint, limit)
}

func (s *memoryStore) ListDueRows(_ context.Context, beforeMs int64, after DueRow, limit int) ([]DueRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return listDueRows(s.rows, beforeMs, after, limit), nil
}

func (s *memoryStore) ApplyCleanup(_ context.Context, deleted []StoredSession, inserted []StoredSession) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return applyCleanupTo(s.rows, deleted, inserted), nil
}

// listDueRows is `ListRowsToCompact` on a table in memory.
func listDueRows(table map[[32]byte]map[string]StoredSession, beforeMs int64, after DueRow, limit int) []DueRow {
	var due []DueRow
	for fingerprint, rows := range table {
		for _, row := range rows {
			isAfter := row.CompactAfterMs > after.CompactAfterMs ||
				(row.CompactAfterMs == after.CompactAfterMs && bytes.Compare(fingerprint[:], after.Fingerprint[:]) > 0)
			if row.CompactAfterMs < beforeMs && isAfter {
				due = append(due, DueRow{Fingerprint: fingerprint, CompactAfterMs: row.CompactAfterMs})
			}
		}
	}
	slices.SortFunc(due, func(left, right DueRow) int {
		return cmp.Or(cmp.Compare(left.CompactAfterMs, right.CompactAfterMs), bytes.Compare(left.Fingerprint[:], right.Fingerprint[:]))
	})
	return due[:min(limit, len(due))]
}

// applyCleanupTo is `ApplyCleanup` on a table in memory.
func applyCleanupTo(table map[[32]byte]map[string]StoredSession, deleted []StoredSession, inserted []StoredSession) bool {
	for _, row := range deleted {
		stored, ok := table[row.Fingerprint][row.SessionID]
		if !ok || stored.ConsumedNs != row.ConsumedNs {
			return false
		}
	}
	for _, row := range deleted {
		delete(table[row.Fingerprint], row.SessionID)
		if len(table[row.Fingerprint]) == 0 {
			delete(table, row.Fingerprint)
		}
	}
	upsert(table, inserted)
	return true
}

// sessions returns all sessions of a fingerprint, newest first.
func (s *memoryStore) sessions(fingerprint [32]byte) []StoredSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	var sessions []StoredSession
	for _, session := range s.rows[fingerprint] {
		sessions = append(sessions, session)
	}
	slices.SortFunc(sessions, func(left, right StoredSession) int {
		return cmp.Or(cmp.Compare(right.AnchorAtNs, left.AnchorAtNs), cmp.Compare(left.SessionID, right.SessionID))
	})
	return sessions
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
