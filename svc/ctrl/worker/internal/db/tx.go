package db

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/mysql"
)

func rootReplica(q Querier) (*mysql.Replica, error) {
	queries, ok := q.(*Queries)
	if !ok {
		return nil, fmt.Errorf("querier must be *db.Queries")
	}
	replica, ok := queries.db.(*mysql.Replica)
	if !ok {
		return nil, fmt.Errorf("transaction requires root querier")
	}
	return replica, nil
}

// Tx executes fn in a transaction.
func Tx(ctx context.Context, q Querier, fn func(context.Context, Querier) error) error {
	replica, err := rootReplica(q)
	if err != nil {
		return err
	}

	return mysql.Tx(ctx, replica, func(inner context.Context, tx mysql.DBTX) error {
		return fn(inner, &Queries{db: tx})
	})
}

// TxWithResult executes fn in a transaction and returns a typed result.
func TxWithResult[T any](ctx context.Context, q Querier, fn func(context.Context, Querier) (T, error)) (T, error) {
	replica, err := rootReplica(q)
	if err != nil {
		var zero T
		return zero, err
	}

	return mysql.TxWithResult(ctx, replica, func(inner context.Context, tx mysql.DBTX) (T, error) {
		return fn(inner, &Queries{db: tx})
	})
}

// TxRetry executes fn in a transaction with retry on transient errors.
func TxRetry(ctx context.Context, q Querier, fn func(context.Context, Querier) error) error {
	replica, err := rootReplica(q)
	if err != nil {
		return err
	}

	return mysql.TxRetry(ctx, replica, func(inner context.Context, tx mysql.DBTX) error {
		return fn(inner, &Queries{db: tx})
	})
}

// TxWithResultRetry executes fn in a transaction with retry on transient errors.
func TxWithResultRetry[T any](ctx context.Context, q Querier, fn func(context.Context, Querier) (T, error)) (T, error) {
	replica, err := rootReplica(q)
	if err != nil {
		var zero T
		return zero, err
	}

	return mysql.TxWithResultRetry(ctx, replica, func(inner context.Context, tx mysql.DBTX) (T, error) {
		return fn(inner, &Queries{db: tx})
	})
}

// WithRetryContext retries read/write operations on transient database errors.
func WithRetryContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return mysql.WithRetryContext(ctx, fn)
}
