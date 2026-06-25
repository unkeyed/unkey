package db

import (
	"context"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
)

type database struct {
	primary *mysql.Replica
	*Queries
	bulk *BulkQueries
}

// New creates a control plane database from a single read-write MySQL DSN.
func New(dsn string) (*database, error) {
	primary, err := mysql.NewReplica(dsn, "rw")
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("cannot open control plane database"))
	}

	return &database{
		primary: primary,
		Queries: NewQueries(primary),
		bulk:    NewBulkQueries(primary),
	}, nil
}

// Conn returns the single read-write connection used by control plane queries.
func (d *database) Conn() *Replica {
	return d.primary
}

// RW returns the single read-write connection.
func (d *database) RW() *Replica {
	return d.primary
}

// RO returns the single read-write connection.
func (d *database) RO() *Replica {
	return d.primary
}

// Bulk returns generated bulk query methods bound to the control plane connection.
func (d *database) Bulk() *BulkQueries {
	return d.bulk
}

// Close releases the control plane database connection pool.
func (d *database) Close() error {
	if err := d.primary.Close(); err != nil {
		return fault.Wrap(err)
	}
	return nil
}

// Tx runs fn in a transaction on the provided control plane connection.
func Tx(ctx context.Context, db *Replica, fn func(context.Context, DBTX) error) error {
	return mysql.Tx(ctx, db, fn)
}

// TxWithResult runs fn in a transaction and returns its result.
func TxWithResult[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error) {
	return mysql.TxWithResult(ctx, db, fn)
}

// TxRetry runs fn in a transaction with retry handling for transient failures.
func TxRetry(ctx context.Context, db *Replica, fn func(context.Context, DBTX) error) error {
	return mysql.TxRetry(ctx, db, fn)
}

// TxWithResultRetry runs fn in a retried transaction and returns its result.
func TxWithResultRetry[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error) {
	return mysql.TxWithResultRetry(ctx, db, fn)
}

// WithRetryContext runs fn with the shared MySQL retry policy.
func WithRetryContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return mysql.WithRetryContext(ctx, fn)
}

// IsNotFound reports whether err represents a missing database row.
func IsNotFound(err error) bool {
	return mysql.IsNotFound(err)
}

// IsDuplicateKeyError reports whether err represents a MySQL duplicate-key failure.
func IsDuplicateKeyError(err error) bool {
	return mysql.IsDuplicateKeyError(err)
}

// IsTransientError reports whether err is retryable under the shared MySQL policy.
func IsTransientError(err error) bool {
	return mysql.IsTransientError(err)
}
