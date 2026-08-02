-- name: UpdateKeyCreditsIncrement :exec
UPDATE `keys`
SET remaining_requests = remaining_requests + sqlc.arg('credits')
WHERE id = ?;

-- LAST_INSERT_ID(expr) returns expr in this UPDATE's OK packet, so
-- sql.Result.LastInsertId reads the new balance without another query.
-- https://dev.mysql.com/doc/refman/8.4/en/information-functions.html#function_last-insert-id
-- name: UpdateKeyCreditsIncrementReturning :execresult
UPDATE `keys`
SET
    remaining_requests = LAST_INSERT_ID(remaining_requests + sqlc.arg('credits'))
WHERE id = ?
  AND deleted_at_m IS NULL
  AND remaining_requests IS NOT NULL
  AND remaining_requests <= 9223372036854775807 - sqlc.arg('credits');
