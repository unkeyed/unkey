package database

import (
	"context"
	"fmt"
)

func (db *database) CountKeys(ctx context.Context, keyAuthId string) (int, error) {

	const query = "SELECT count(*) FROM unkey.keys WHERE key_auth_id = ? "
	row := db.read().QueryRow(query, keyAuthId)

	count := 0
	err := row.Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("unable to count keys: %w", err)
	}
	return count, nil

}
