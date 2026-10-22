package ratelimit

import (
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
	for _, session := range sessions {
		rows, ok := s.rows[session.Fingerprint]
		if !ok {
			rows = make(map[string]StoredSession)
			s.rows[session.Fingerprint] = rows
		}
		stored, ok := rows[session.SessionID]
		if ok {
			stored.ConsumedNs = max(stored.ConsumedNs, session.ConsumedNs)
			session = stored
		}
		rows[session.SessionID] = session
	}
	return nil
}

func (s *memoryStore) ReadSessions(_ context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	sessions := s.sessions(fingerprint)
	return sessions[:min(limit, len(sessions))], nil
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
