package mysql

import (
	"context"
	"database/sql"
)

// ReadWithRetry retries one read-only query, including scanning its results.
// Transactions and unknown DBTX implementations run once. Pools must be *sql.DB
// or explicitly support read retries. The callback must not perform writes.
func ReadWithRetry[T any](ctx context.Context, db DBTX, fn func() (T, error)) (T, error) {
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	if _, transactional := db.(DBTx); transactional {
		return fn()
	}
	_, pool := db.(*sql.DB)
	if replica, ok := db.(interface{ SupportsReadRetry() bool }); ok {
		pool = replica.SupportsReadRetry()
	}
	if !pool {
		return fn()
	}
	return WithRetryContext(ctx, fn)
}
