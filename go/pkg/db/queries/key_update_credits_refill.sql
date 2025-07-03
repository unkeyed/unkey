-- name: UpdateKeyCreditsRefill :exec
UPDATE `keys` SET refill_amount = ? AND refill_day = ? WHERE id = ?;
