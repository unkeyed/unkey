package ratelimit

import (
	"context"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ratelimit/db"
)

// Store holds the rows that all pods share. Implementations must be safe for
// concurrent use: the flush, background reads, cleanup and requests call
// them at the same time. The limiter never holds a shard lock while it calls
// them.
type Store interface {
	// NowNs returns the clock of the store in Unix nanoseconds. Cleanup
	// deletes by it and checks the pod clock against it.
	NowNs(ctx context.Context) (int64, error)

	// WriteRows writes rows and keeps the larger `ScaledSum` and
	// `DeleteAfterNs` of each, so a retried or reordered write never moves a
	// row back. The limiter sends at most `maxRowsPerWrite` rows per call.
	WriteRows(ctx context.Context, rows []StoredRow) error

	// ReadRows returns the rows of at most `limit` periods of one
	// fingerprint, newest period first, from a replica. Per period it
	// returns the row of `podID` and one row with an empty pod ID that sums
	// the rows of all other pods. Rows with a zero sum are left out. So a
	// read grows with periods, not with pods.
	ReadRows(ctx context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error)

	// DeleteFadedRows deletes at most `limit` rows whose `DeleteAfterNs` is
	// before `beforeNs`, of any fingerprint, and returns how many it
	// deleted.
	DeleteFadedRows(ctx context.Context, beforeNs int64, limit int) (int64, error)
}

// StoredRow is one row of the `ratelimit_state` table.
type StoredRow struct {
	Fingerprint   [32]byte
	PeriodStartNs int64
	PodID         string
	ScaledSum     float64
	DeleteAfterNs int64
}

type mysqlStore struct {
	primary db.DBTX
	replica db.DBTX
}

// NewMySQLStore returns a [Store] that writes and deletes on the primary and
// reads from the replica. Both must be safe for concurrent use, as a
// `*db.Replica` from `pkg/db` is.
func NewMySQLStore(primary db.DBTX, replica db.DBTX) Store {
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
			Fingerprint:   r.Fingerprint[:],
			PeriodStartNs: r.PeriodStartNs,
			PodID:         r.PodID,
			ScaledSum:     r.ScaledSum,
			DeleteAfterNs: r.DeleteAfterNs,
		}
	}
	err := db.BulkQuery.UpsertRow(ctx, s.primary, params)
	if err != nil {
		return fault.Wrap(err, fault.Internal("unable to write rate limit rows"))
	}
	return nil
}

func (s *mysqlStore) ReadRows(ctx context.Context, fingerprint [32]byte, podID string, limit int) ([]StoredRow, error) {
	periods, err := db.Query.ListPeriods(ctx, s.replica, db.ListPeriodsParams{
		PodID:       podID,
		Fingerprint: fingerprint[:],
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("unable to read rate limit rows"))
	}
	var stored []StoredRow
	for _, p := range periods {
		if p.OwnScaledSum > 0 {
			stored = append(stored, StoredRow{
				Fingerprint:   fingerprint,
				PeriodStartNs: p.PeriodStartNs,
				PodID:         podID,
				ScaledSum:     p.OwnScaledSum,
				DeleteAfterNs: p.DeleteAfterNs,
			})
		}
		if p.PeerScaledSum > 0 {
			stored = append(stored, StoredRow{
				Fingerprint:   fingerprint,
				PeriodStartNs: p.PeriodStartNs,
				PodID:         "",
				ScaledSum:     p.PeerScaledSum,
				DeleteAfterNs: p.DeleteAfterNs,
			})
		}
	}
	return stored, nil
}

func (s *mysqlStore) DeleteFadedRows(ctx context.Context, beforeNs int64, limit int) (int64, error) {
	deleted, err := db.Query.DeleteFadedRows(ctx, s.primary, db.DeleteFadedRowsParams{
		BeforeNs: beforeNs,
		Limit:    int32(limit),
	})
	if err != nil {
		return 0, fault.Wrap(err, fault.Internal("unable to delete faded rate limit rows"))
	}
	return deleted, nil
}
