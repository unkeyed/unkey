package db

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Tx executes a function within a database transaction, providing automatic
// transaction lifecycle management with proper error handling and rollback
// behavior. Tx is a generic transaction wrapper that handles the common
// pattern of begin-execute-commit/rollback operations while preserving type
// safety for return values.
//
// The function begins a new transaction on the provided database replica,
// executes the given function with the transaction context, and automatically
// commits on success or rolls back on failure. All database errors are
// wrapped with appropriate fault codes for consistent error handling across
// the Unkey platform.
//
// Tx is shared across all Unkey services that require transactional operations,
// including the API service for atomic key operations, the admin service for
// workspace management, and the audit service for logging operations. Common
// usage scenarios include:
//   - Creating API keys with associated permissions atomically
//   - Updating workspace settings with audit trail creation
//   - Batch operations that must succeed or fail as a unit
//   - Complex queries that require consistency guarantees
//
// The ctx parameter provides cancellation and timeout control for the entire
// transaction. If the context is cancelled during execution, the transaction
// will be rolled back automatically. The db parameter must be a valid Replica
// instance, typically obtained from [Database.RW] for write operations.
//
// The fn parameter receives the transaction context and a [DBTX] interface
// that can be used with any of the generated query methods. The function
// should perform all required database operations and return the desired
// result along with any error that occurred.
//
// Returns the result of the executed function on successful commit, or an
// error if any step fails. Transaction begin failures return a wrapped error
// with code ServiceUnavailable. Rollback failures during error handling also
// return ServiceUnavailable errors, except for sql.ErrTxDone which indicates
// the transaction was already completed. Commit failures return
// ServiceUnavailable errors.
//
// Tx is safe for concurrent use and does not modify any global state.
// However, callers must ensure that the provided function fn does not
// perform operations that could deadlock with other concurrent transactions.
//
// The function handles several edge cases that could surprise developers:
//   - If fn returns an error, rollback is attempted even if the transaction
//     is already in a failed state, which may produce additional errors
//   - Context cancellation is automatically detected after fn execution,
//     causing an automatic rollback instead of commit
//   - Database connection issues during commit may leave the transaction
//     in an undefined state on the server side
//
// Avoid these common anti-patterns when using Tx:
//   - DO NOT perform long-running operations within fn that could timeout
//   - DO NOT nest calls to Tx, as this creates nested transactions
//   - DO NOT ignore the returned error from fn, even if you think it's safe
//   - DO NOT access the DBTX parameter outside of the fn callback
//
// Use context.WithTimeout to prevent indefinite blocking on database
// operations. For operations that may conflict, consider implementing
// retry logic with exponential backoff at the caller level.
//
// Example usage for atomic key creation with permissions:
//
//	result, err := db.Tx(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) (*models.Key, error) {
//		// Insert the new API key
//		key, err := db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
//			ID:        keyID,
//			KeyAuthID: keyAuthID,
//			Hash:      hashedKey,
//			// ... other parameters
//		})
//		if err != nil {
//			return nil, fmt.Errorf("failed to insert key: %w", err)
//		}
//
//		// Add permissions to the key
//		for _, permissionID := range permissionIDs {
//			err = db.Query.InsertKeyPermission(ctx, tx, db.InsertKeyPermissionParams{
//				KeyID:        key.ID,
//				PermissionID: permissionID,
//			})
//			if err != nil {
//				return nil, fmt.Errorf("failed to add permission: %w", err)
//			}
//		}
//
//		return &key, nil
//	})
//	if err != nil {
//		return fmt.Errorf("key creation transaction failed: %w", err)
//	}
//
// Example with automatic context cancellation handling:
//
//	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
//	defer cancel()
//
//	result, err := db.Tx(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) (*SomeResult, error) {
//		// Context cancellation is automatically handled by Tx
//		// No need to check ctx.Done() unless you want early termination
//
//		// Perform database operations...
//		return &SomeResult{}, nil
//	})
//	if err != nil {
//		// This could be a cancellation error or other transaction error
//		return fmt.Errorf("transaction failed: %w", err)
//	}
//
// See [Replica.Begin] for transaction initiation details and [DBTX] for
// available database operations within transactions. For read-only operations
// that don't require transactions, use the query methods directly on
// [Database.RO].
func Tx[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error) {
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
