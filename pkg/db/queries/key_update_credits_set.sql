-- name: UpdateKeyCreditsSet :execresult
-- transactional-batch-statement
UPDATE `keys`
SET
    remaining_requests = CASE
        WHEN deleted_at_m IS NOT NULL THEN
            IF(LAST_INSERT_ID(18446744073709551615) > 0, remaining_requests, remaining_requests)
        WHEN sqlc.narg('credits') IS NULL THEN
            IF(LAST_INSERT_ID(0) = 0, NULL, NULL)
        ELSE LAST_INSERT_ID(sqlc.narg('credits'))
    END,
    refill_amount = CASE
        WHEN deleted_at_m IS NULL
          AND CAST(sqlc.arg('clear_refill_amount') AS UNSIGNED) = 1 THEN NULL
        ELSE refill_amount
    END,
    refill_day = CASE
        WHEN deleted_at_m IS NULL
          AND CAST(sqlc.arg('clear_refill_day') AS UNSIGNED) = 1 THEN NULL
        ELSE refill_day
    END
WHERE id = ?;
