package ratelimit

import (
	"context"
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/pkg/assert"
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

	// ListDueRows returns at most `limit` rows whose `CompactAfterMs` is
	// before `beforeMs`, ordered by `CompactAfterMs` and then `Fingerprint`,
	// starting after the row `after`. Pass a `CompactAfterMs` of -1 in
	// `after` to start at the first row.
	ListDueRows(ctx context.Context, beforeMs int64, after DueRow, limit int) ([]DueRow, error)

	// ReadSessionsFromPrimary is ReadSessions without replica lag, so
	// cleanup sees every value that its deletes compare against.
	ReadSessionsFromPrimary(ctx context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error)

	// ApplyCleanup deletes and inserts sessions in one transaction. A delete
	// matches its row only if `ConsumedNs` is unchanged. If any delete
	// matches no row, it changes nothing and returns false.
	ApplyCleanup(ctx context.Context, deleted []StoredSession, inserted []StoredSession) (bool, error)
}

// DueRow is one row whose `compact_after_ms` has passed.
type DueRow struct {
	Fingerprint    [32]byte
	CompactAfterMs int64
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
	primary *sql.DB
	replica db.DBTX
}

// NewMySQLStore returns a [Store] that writes to the primary and reads from
// the replica. The replica must be safe for concurrent use, as a `*sql.DB`
// is.
func NewMySQLStore(primary *sql.DB, replica db.DBTX) Store {
	return &mysqlStore{primary: primary, replica: replica}
}

func (s *mysqlStore) WriteSessions(ctx context.Context, sessions []StoredSession) error {
	err := db.BulkQuery.UpsertSession(ctx, s.primary, upsertParams(sessions))
	if err != nil {
		return fault.Wrap(err, fault.Internal("unable to write rate limit sessions"))
	}
	return nil
}

func (s *mysqlStore) ReadSessions(ctx context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	return readSessions(ctx, s.replica, fingerprint, limit)
}

func (s *mysqlStore) ReadSessionsFromPrimary(ctx context.Context, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	return readSessions(ctx, s.primary, fingerprint, limit)
}

func (s *mysqlStore) ListDueRows(ctx context.Context, beforeMs int64, after DueRow, limit int) ([]DueRow, error) {
	rows, err := db.Query.ListRowsToCompact(ctx, s.primary, db.ListRowsToCompactParams{
		NowMs:               beforeMs,
		AfterCompactAfterMs: after.CompactAfterMs,
		AfterFingerprint:    after.Fingerprint[:],
		Limit:               int32(limit),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("unable to list rate limit rows to compact"))
	}
	due := make([]DueRow, len(rows))
	for i, row := range rows {
		err = assert.Equal(len(row.Fingerprint), 32, "fingerprint must have 32 bytes")
		if err != nil {
			return nil, err
		}
		due[i] = DueRow{Fingerprint: [32]byte(row.Fingerprint), CompactAfterMs: row.CompactAfterMs}
	}
	return due, nil
}

func (s *mysqlStore) ApplyCleanup(ctx context.Context, deleted []StoredSession, inserted []StoredSession) (bool, error) {
	tx, err := s.primary.BeginTx(ctx, nil)
	if err != nil {
		return false, fault.Wrap(err, fault.Internal("unable to begin the rate limit cleanup"))
	}
	applied, err := applyCleanup(ctx, tx, deleted, inserted)
	if err != nil || !applied {
		return false, errors.Join(err, tx.Rollback())
	}
	err = tx.Commit()
	if err != nil {
		return false, fault.Wrap(err, fault.Internal("unable to commit the rate limit cleanup"))
	}
	return true, nil
}

// applyCleanup runs the statements of ApplyCleanup in `tx`. It returns false
// as soon as a delete matches no row. The caller commits or rolls back.
func applyCleanup(ctx context.Context, tx db.DBTX, deleted []StoredSession, inserted []StoredSession) (bool, error) {
	for _, row := range deleted {
		affected, err := db.Query.DeleteSession(ctx, tx, db.DeleteSessionParams{
			Fingerprint: row.Fingerprint[:],
			SessionID:   row.SessionID,
			ConsumedNs:  row.ConsumedNs,
		})
		if err != nil {
			return false, fault.Wrap(err, fault.Internal("unable to delete a rate limit session"))
		}
		if affected == 0 {
			return false, nil
		}
	}
	if len(inserted) == 0 {
		return true, nil
	}
	err := db.BulkQuery.UpsertSession(ctx, tx, upsertParams(inserted))
	if err != nil {
		return false, fault.Wrap(err, fault.Internal("unable to insert merged rate limit sessions"))
	}
	return true, nil
}

func readSessions(ctx context.Context, conn db.DBTX, fingerprint [32]byte, limit int) ([]StoredSession, error) {
	rows, err := db.Query.ListSessions(ctx, conn, db.ListSessionsParams{
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

func upsertParams(sessions []StoredSession) []db.UpsertSessionParams {
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
	return params
}
