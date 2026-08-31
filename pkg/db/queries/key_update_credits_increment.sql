-- name: UpdateKeyCreditsIncrement :exec
UPDATE `keys`
SET remaining_requests = remaining_requests + sqlc.arg('credits')
WHERE id = ?;

-- LAST_INSERT_ID(expr) returns expr in this UPDATE's OK packet, so
-- sql.Result.LastInsertId reads the new balance without another query.
-- https://dev.mysql.com/doc/refman/8.4/en/information-functions.html#function_last-insert-id
-- name: UpdateKeyCreditsIncrementReturning :execresult
-- transactional-batch-statement
UPDATE `keys`
SET
    remaining_requests = CASE
        WHEN deleted_at_m IS NOT NULL THEN
            IF(LAST_INSERT_ID(18446744073709551615) > 0, remaining_requests, remaining_requests)
        WHEN remaining_requests IS NULL THEN
            IF(LAST_INSERT_ID(18446744073709551614) > 0, remaining_requests, remaining_requests)
        WHEN remaining_requests > 9223372036854775807 - sqlc.arg('credits') THEN
            IF(LAST_INSERT_ID(18446744073709551613) > 0, remaining_requests, remaining_requests)
        ELSE LAST_INSERT_ID(remaining_requests + sqlc.arg('credits'))
    END
WHERE id = ?;
