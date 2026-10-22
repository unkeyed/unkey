package ratelimit

import (
	"context"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ratelimit/db"
)

// Store holds the sessions that all pods share. Implementations must be safe
// for concurrent use: the flush, background reads and requests call them at
// the same time. The limiter never holds a shard lock while it calls them.
type Store interface {
	// WriteSessions writes sessions and keeps the larger `ConsumedNs` of each,
	// so a retried or reordered write never moves a session back. The
	// limiter sends at most 10,000 sessions per call.
	WriteSessions(ctx context.Context, sessions []StoredSession) error

	// ReadSessions returns at most `limit` sessions of one fingerprint,
	// newest first.
	ReadSessions(ctx context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error)
}

// StoredSession is one row of the `ratelimit_state` table.
type StoredSession struct {
	Fingerprint    [32]byte
	SessionID      string
	AnchorAtNs     int64
	ConsumedNs     int64
	CompactAfterMs int64
}

type mysqlStore struct {
	primary db.DBTX
	replica db.DBTX
}

// NewMySQLStore returns a [Store] that writes to the primary and reads from
// the replica. Both must be safe for concurrent use, as a `*sql.DB` is.
func NewMySQLStore(primary, replica db.DBTX) Store {
	return &mysqlStore{primary: primary, replica: replica}
}

func (s *mysqlStore) WriteSessions(ctx context.Context, sessions []StoredSession) error {
	params := make([]db.UpsertSessionParams, len(sessions))
	for i, session := range sessions {
		params[i] = db.UpsertSessionParams{
			Fingerprint:    session.Fingerprint[:],
			SessionID:      session.SessionID,
			AnchorAtNs:     session.AnchorAtNs,
			ConsumedNs:     session.ConsumedNs,
			CompactAfterMs: session.CompactAfterMs,
		}
	}
	err := db.BulkQuery.UpsertSession(ctx, s.primary, params)
	if err != nil {
		return fault.Wrap(err, fault.Internal("unable to write rate limit sessions"))
	}
	return nil
}

func (s *mysqlStore) ReadSessions(ctx context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	rows, err := db.Query.ListSessions(ctx, s.replica, db.ListSessionsParams{
		Fingerprint: fingerprint[:],
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("unable to read rate limit sessions"))
	}
	sessions := make([]StoredSession, len(rows))
	for i, row := range rows {
		sessions[i] = StoredSession{
			Fingerprint:    fingerprint,
			SessionID:      row.SessionID,
			AnchorAtNs:     row.AnchorAtNs,
			ConsumedNs:     row.ConsumedNs,
			CompactAfterMs: row.CompactAfterMs,
		}
	}
	return sessions, nil
}
