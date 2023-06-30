package database

import (
	"context"
	"fmt"
)

func (db *database) DeleteKey(ctx context.Context, keyId string) error {
	query := `DELETE FROM unkey.keys ` +
		`WHERE id = ?`

	_, err := db.write().Query(query, keyId)
	if err != nil {
		return fmt.Errorf("unable to delete key %s from db: %w", keyId, err)
	}

	return nil
}
