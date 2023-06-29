package database

import (
	"context"
	"fmt"
)

func (db *Database) CountKeys(ctx context.Context, apiId string) (int, error) {

	const query = "SELECT count(*) FROM unkey.keys WHERE api_id = ? "

	row := db.read().QueryRow(query, apiId)

	count := 0
	err := row.Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("unable to count keys: %w", err)
	}
	return count, nil

}
