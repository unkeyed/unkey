package db

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func Tx[T any](ctx context.Context, db *Replica, fn func(ctx context.Context, tx DBTX) (T, error)) (T, error) {
	var t T

	tx, err := db.Begin(ctx)
	if err != nil {
		return t, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to create transaction"), fault.Public("Unable to start database transaction."),
		)
	}

	t, err = fn(ctx, tx)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			return t, fault.Wrap(rollbackErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to rollback transaction"), fault.Public("Unable to rollback database transaction."),
			)
		}
		return t, err
	}

	err = tx.Commit()
	if err != nil {
		return t, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to commit transaction"), fault.Public("Unable to commit database transaction."),
		)
	}

	return t, nil
}
