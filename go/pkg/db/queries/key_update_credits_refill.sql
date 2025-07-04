-- name: UpdateKeyCreditsRefill :exec
UPDATE `keys` SET refill_amount = ?, refill_day = ? WHERE id = ?;
