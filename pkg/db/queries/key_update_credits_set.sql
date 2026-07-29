-- name: UpdateKeyCreditsSet :execresult
UPDATE `keys`
SET
    remaining_requests = sqlc.narg('credits'),
    refill_amount = CASE
        WHEN CAST(sqlc.arg('clear_refill_amount') AS UNSIGNED) = 1 THEN NULL
        ELSE refill_amount
    END,
    refill_day = CASE
        WHEN CAST(sqlc.arg('clear_refill_day') AS UNSIGNED) = 1 THEN NULL
        ELSE refill_day
    END
WHERE id = ?
  AND deleted_at_m IS NULL;
