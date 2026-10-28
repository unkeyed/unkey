package ratelimit

import (
	"context"
	"errors"

	"github.com/unkeyed/unkey/pkg/assert"
	shareddb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ratelimit/db"
)

// Store holds the rows that all pods share. Implementations must be safe for
// concurrent use: the flush, background reads, compaction and requests call
// them at the same time. The limiter never holds a shard lock while it calls
// them.
type Store interface {
	// NowNs returns the clock of the store in Unix nanoseconds. Cleanup uses
	// it to decide which buckets are closed, and New checks the pod clock
	// against it.
	NowNs(ctx context.Context) (int64, error)

	// WriteRows writes rows and keeps the larger `ConsumedNs` and
	// `ResetAtNs` of each, so a retried or reordered write never moves a row
	// back. It refuses rows whose `CompactAfterNs` has passed by `NowNs`,
	// without an error. The limiter sends at most `maxRowsPerWrite` rows per call.
	WriteRows(ctx context.Context, rows []StoredRow) error

	// ReadRows returns the rows of at most `limit` buckets of one fingerprint,
	// newest bucket first, from a replica. Per bucket it returns the row of
	// `podID` and one row with an empty pod ID that sums the rows of all
	// other pods: their consumed time and their largest reset time. Rows
	// with zero consumed time are left out. So a read grows with buckets,
	// not with pods.
	ReadRows(ctx context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error)

	// ListDueRows returns at most `limit` rows whose `CompactAfterNs` is
	// before `beforeNs`, ordered by `CompactAfterNs` and then `Fingerprint`,
	// starting after the row `after`. Pass a `CompactAfterNs` of -1 in
	// `after` to start at the first row.
	ListDueRows(ctx context.Context, beforeNs int64, after DueRow, limit int) ([]DueRow, error)

	// ReadClosedRows returns at most `limit` rows of one fingerprint from the
	// primary, oldest bucket first: the base row and the rows whose
	// `CompactAfterNs` is before `beforeNs`.
	ReadClosedRows(ctx context.Context, fingerprint [32]byte, beforeNs int64, limit int) ([]StoredRow, error)

	// ApplyCleanup deletes rows and inserts base rows in one transaction. A
	// delete matches its row only if `ConsumedNs` and `ResetAtNs` are
	// unchanged. If any delete matches no row, it changes nothing and
	// returns false.
	ApplyCleanup(ctx context.Context, deleted []StoredRow, inserted []StoredRow) (bool, error)
}

// DueRow is one row whose `compact_after_ns` has passed.
type DueRow struct {
	Fingerprint    [32]byte
	CompactAfterNs int64
}

// StoredRow is one row of the `ratelimit_state` table.
type StoredRow struct {
	Fingerprint    [32]byte
	BucketStartNs  int64
	PodID          string
	ConsumedNs     int64
	ResetAtNs      int64
	CompactAfterNs int64
}

// Primary is the connection that a [NewMySQLStore] writes to and cleans up
// on. A `*db.Replica` from `pkg/db` satisfies it and tags every query and
// transaction for Query Insights.
type Primary interface {
	db.DBTX
	Begin(ctx context.Context) (shareddb.DBTx, error)
}

type mysqlStore struct {
	primary Primary
	replica db.DBTX
}

// NewMySQLStore returns a [Store] that writes to the primary and reads from
// the replica. Both must be safe for concurrent use, as a `*db.Replica` is.
func NewMySQLStore(primary Primary, replica db.DBTX) Store {
	return &mysqlStore{primary: primary, replica: replica}
}

func (s *mysqlStore) NowNs(ctx context.Context) (int64, error) {
	nowNs, err := db.Query.Now(ctx, s.primary)
	if err != nil {
		return 0, fault.Wrap(err, fault.Internal("unable to read the MySQL clock"))
	}
	return nowNs, nil
}

func (s *mysqlStore) WriteRows(ctx context.Context, rows []StoredRow) error {
	params := make([]db.UpsertRowParams, len(rows))
	for i, r := range rows {
		params[i] = db.UpsertRowParams{
			Fingerprint:    r.Fingerprint[:],
			BucketStartNs:  r.BucketStartNs,
			PodID:          r.PodID,
			WriteUntilNs:   r.CompactAfterNs,
			ConsumedNs:     r.ConsumedNs,
			ResetAtNs:      r.ResetAtNs,
			CompactAfterNs: r.CompactAfterNs,
		}
	}
	err := db.BulkQuery.UpsertRow(ctx, s.primary, params)
	if err != nil {
		return fault.Wrap(err, fault.Internal("unable to write rate limit rows"))
	}
	return nil
}

func (s *mysqlStore) ReadRows(ctx context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error) {
	buckets, err := db.Query.ListBuckets(ctx, s.replica, db.ListBucketsParams{
		PodID:       podID,
		Fingerprint: fingerprint[:],
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("unable to read rate limit rows"))
	}
	var stored []StoredRow
	for _, b := range buckets {
		if b.OwnConsumedNs > 0 {
			stored = append(stored, StoredRow{
				Fingerprint:    fingerprint,
				BucketStartNs:  b.BucketStartNs,
				PodID:          podID,
				ConsumedNs:     b.OwnConsumedNs,
				ResetAtNs:      b.OwnResetAtNs,
				CompactAfterNs: b.CompactAfterNs,
			})
		}
		if b.PeerConsumedNs > 0 {
			stored = append(stored, StoredRow{
				Fingerprint:    fingerprint,
				BucketStartNs:  b.BucketStartNs,
				PodID:          "",
				ConsumedNs:     b.PeerConsumedNs,
				ResetAtNs:      b.PeerResetAtNs,
				CompactAfterNs: b.CompactAfterNs,
			})
		}
	}
	return stored, nil
}

func (s *mysqlStore) ReadClosedRows(ctx context.Context, fingerprint [32]byte, beforeNs int64, limit int) ([]StoredRow, error) {
	rows, err := db.Query.ListClosedRows(ctx, s.primary, db.ListClosedRowsParams{
		Fingerprint: fingerprint[:],
		BeforeNs:    beforeNs,
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("unable to read closed rate limit rows"))
	}
	stored := make([]StoredRow, len(rows))
	for i, r := range rows {
		stored[i] = StoredRow{
			Fingerprint:    fingerprint,
			BucketStartNs:  r.BucketStartNs,
			PodID:          r.PodID,
			ConsumedNs:     r.ConsumedNs,
			ResetAtNs:      r.ResetAtNs,
			CompactAfterNs: r.CompactAfterNs,
		}
	}
	return stored, nil
}

func (s *mysqlStore) ListDueRows(ctx context.Context, beforeNs int64, after DueRow, limit int) ([]DueRow, error) {
	rows, err := db.Query.ListRowsToCompact(ctx, s.primary, db.ListRowsToCompactParams{
		BeforeNs:            beforeNs,
		AfterCompactAfterNs: after.CompactAfterNs,
		AfterFingerprint:    after.Fingerprint[:],
		Limit:               int32(limit),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("unable to list rate limit rows to compact"))
	}
	due := make([]DueRow, len(rows))
	for i, r := range rows {
		err = assert.Equal(len(r.Fingerprint), 32, "fingerprint must have 32 bytes")
		if err != nil {
			return nil, err
		}
		due[i] = DueRow{Fingerprint: [32]byte(r.Fingerprint), CompactAfterNs: r.CompactAfterNs}
	}
	return due, nil
}

func (s *mysqlStore) ApplyCleanup(ctx context.Context, deleted []StoredRow, inserted []StoredRow) (bool, error) {
	tx, err := s.primary.Begin(ctx)
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
func applyCleanup(ctx context.Context, tx db.DBTX, deleted []StoredRow, inserted []StoredRow) (bool, error) {
	for _, r := range deleted {
		affected, err := db.Query.DeleteRow(ctx, tx, db.DeleteRowParams{
			Fingerprint:   r.Fingerprint[:],
			BucketStartNs: r.BucketStartNs,
			PodID:         r.PodID,
			ConsumedNs:    r.ConsumedNs,
			ResetAtNs:     r.ResetAtNs,
		})
		if err != nil {
			return false, fault.Wrap(err, fault.Internal("unable to delete a rate limit row"))
		}
		if affected == 0 {
			return false, nil
		}
	}
	params := make([]db.InsertBaseRowParams, len(inserted))
	for i, r := range inserted {
		err := assert.Equal(r.BucketStartNs, 0, "cleanup only inserts base rows")
		if err != nil {
			return false, err
		}
		params[i] = db.InsertBaseRowParams{
			Fingerprint:    r.Fingerprint[:],
			ConsumedNs:     r.ConsumedNs,
			ResetAtNs:      r.ResetAtNs,
			CompactAfterNs: r.CompactAfterNs,
		}
	}
	err := db.BulkQuery.InsertBaseRows(ctx, tx, params)
	if err != nil {
		return false, fault.Wrap(err, fault.Internal("unable to insert a merged rate limit row"))
	}
	return true, nil
}
