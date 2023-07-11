package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Decrement the `remaining` field and return the new value
// The returned value is the number of remaining verifications after the current one.
// This means the returned value can be negative, for example when the remaining is 0 and we call this function.
func (db *database) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (int64, error) {
	tx, err := db.write().BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("unable to start transaction: %w", err)
	}

	_, err = tx.Exec(`UPDATE unkey.keys SET remaining_requests = remaining_requests - 1 WHERE id = ?`, keyId)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return 0, fmt.Errorf("unable to roll back: %w", rollbackErr)
		}
		return 0, fmt.Errorf("unable to decrement: %w", err)
	}

	row := tx.QueryRow(`SELECT remaining_requests FROM unkey.keys WHERE id = ?`, keyId)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return 0, fmt.Errorf("unable to roll back: %w", rollbackErr)
		}
		return 0, fmt.Errorf("unable to query: %w", err)
	}
	var remainingAfter sql.NullInt64
	err = row.Scan(&remainingAfter)

	if err != nil {
		return 0, fmt.Errorf("unable to scan result: %w", err)
	}
	if !remainingAfter.Valid {
		return 0, fmt.Errorf("this key did not have a remaining config")
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("unable to commit transaction: %w", err)
	}

	return remainingAfter.Int64, err

}
