// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: ratelimit_list_by_key_ids.sql

package db

import (
	"context"
	"database/sql"
	"strings"
)

const listRatelimitsByKeyIDs = `-- name: ListRatelimitsByKeyIDs :many
SELECT
  id,
  key_id,
  name,
  ` + "`" + `limit` + "`" + `,
  duration,
  auto_apply
FROM ratelimits
WHERE key_id IN (/*SLICE:key_ids*/?)
ORDER BY key_id, id
`

type ListRatelimitsByKeyIDsRow struct {
	ID        string         `db:"id"`
	KeyID     sql.NullString `db:"key_id"`
	Name      string         `db:"name"`
	Limit     int32          `db:"limit"`
	Duration  int64          `db:"duration"`
	AutoApply bool           `db:"auto_apply"`
}

// ListRatelimitsByKeyIDs
//
//	SELECT
//	  id,
//	  key_id,
//	  name,
//	  `limit`,
//	  duration,
//	  auto_apply
//	FROM ratelimits
//	WHERE key_id IN (/*SLICE:key_ids*/?)
//	ORDER BY key_id, id
func (q *Queries) ListRatelimitsByKeyIDs(ctx context.Context, db DBTX, keyIds []sql.NullString) ([]ListRatelimitsByKeyIDsRow, error) {
	query := listRatelimitsByKeyIDs
	var queryParams []interface{}
	if len(keyIds) > 0 {
		for _, v := range keyIds {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:key_ids*/?", strings.Repeat(",?", len(keyIds))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:key_ids*/?", "NULL", 1)
	}
	rows, err := db.QueryContext(ctx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListRatelimitsByKeyIDsRow
	for rows.Next() {
		var i ListRatelimitsByKeyIDsRow
		if err := rows.Scan(
			&i.ID,
			&i.KeyID,
			&i.Name,
			&i.Limit,
			&i.Duration,
			&i.AutoApply,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
