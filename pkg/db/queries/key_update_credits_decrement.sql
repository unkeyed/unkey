-- name: UpdateKeyCreditsDecrement :exec
UPDATE `keys`
SET remaining_requests = CASE
    WHEN remaining_requests >= sqlc.arg('credits') THEN remaining_requests - sqlc.arg('credits')
    ELSE 0
END
WHERE id = ?;

-- LAST_INSERT_ID(expr) returns expr in this UPDATE's OK packet, so
-- sql.Result.LastInsertId reads the new balance without another query.
-- https://dev.mysql.com/doc/refman/8.4/en/information-functions.html#function_last-insert-id
-- name: UpdateKeyCreditsDecrementReturning :execresult
UPDATE `keys`
SET
    remaining_requests = LAST_INSERT_ID(CASE
        WHEN remaining_requests >= sqlc.arg('credits') THEN remaining_requests - sqlc.arg('credits')
        ELSE 0
    END)
WHERE id = ?
  AND deleted_at_m IS NULL
  AND remaining_requests IS NOT NULL;
