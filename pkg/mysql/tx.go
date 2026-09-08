package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// TxWithResult executes fn within a database transaction and returns the result.
// It begins a transaction on db, executes fn with the transaction context,
// and commits on success or rolls back on failure.
func TxWithResult[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error) {
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
		return t, rollbackError(err, tx.Rollback())
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

func rollbackError(cause, rollbackErr error) error {
	if rollbackErr == nil || errors.Is(rollbackErr, sql.ErrTxDone) {
		return cause
	}
	return fault.Wrap(errors.Join(cause, rollbackErr),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("database failed to rollback transaction"), fault.Public("Unable to rollback database transaction."),
	)
}

// Tx executes fn within a database transaction without returning a result.
// It is a convenience wrapper around [TxWithResult] for operations that
// only need error handling.
func Tx(ctx context.Context, db *Replica, fn func(context.Context, DBTX) error) error {
	_, err := TxWithResult(ctx, db, func(inner context.Context, tx DBTX) (any, error) {
		return nil, fn(inner, tx)
	})
	return err
}
